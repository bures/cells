/*
 * Copyright (c) 2019. Abstrium SAS <team (at) pydio.com>
 * This file is part of Pydio Cells.
 *
 * Pydio Cells is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * Pydio Cells is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with Pydio Cells.  If not, see <http://www.gnu.org/licenses/>.
 *
 * The latest code can be found at <https://pydio.com>.
 */

// Package snapshot provides fast in-memory or on-file implementations of endpoint
// for storing snapshots
package snapshot

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pydio/cells/common/log"
	"go.uber.org/zap"

	"github.com/pborman/uuid"

	"github.com/etcd-io/bbolt"
	"github.com/golang/protobuf/proto"
	"github.com/micro/go-micro/errors"

	"github.com/pydio/cells/common/proto/tree"
	"github.com/pydio/cells/common/sync/model"
)

var (
	bucketName        = []byte("snapshot")
	captureBucketName = []byte("capture")
)

type BoltSnapshot struct {
	db         *bbolt.DB
	name       string
	empty      bool
	folderPath string

	createsSession     map[string]*tree.Node
	createsSessionLock *sync.Mutex

	autoBatchChan  chan *tree.Node
	autoBatchClose chan struct{}
}

func NewBoltSnapshot(folderPath, name string) (*BoltSnapshot, error) {
	s := &BoltSnapshot{name: name}
	options := bbolt.DefaultOptions
	options.Timeout = 5 * time.Second
	s.folderPath = folderPath
	p := filepath.Join(s.folderPath, "snapshot-"+name)
	if _, err := os.Stat(p); err != nil {
		s.empty = true
	}
	db, err := bbolt.Open(p, 0644, options)
	if err != nil {
		return nil, err
	}
	s.db = db

	s.autoBatchChan = make(chan *tree.Node)
	s.autoBatchClose = make(chan struct{}, 1)

	go s.startAutoBatching()

	return s, nil
}

func (s *BoltSnapshot) sortByKey(data map[string]*tree.Node) (output []*tree.Node) {
	var kk []string
	for k, _ := range data {
		kk = append(kk, k)
	}
	sort.Strings(kk)
	for _, k := range kk {
		output = append(output, data[k])
	}
	return
}

func (s *BoltSnapshot) startAutoBatching() {
	defer func() {
		close(s.autoBatchChan)
	}()
	creates := make(map[string]*tree.Node, 500)
	nextTime := 1 * time.Hour
	flush := func() {
		if len(creates) == 0 {
			return
		}
		log.Logger(context.Background()).Debug("Flushing AutoBatcher")
		s.db.Update(func(tx *bbolt.Tx) error {
			b := tx.Bucket(bucketName)
			if b == nil {
				return fmt.Errorf("cannot find root bucket")
			}
			for _, node := range s.sortByKey(creates) {
				b.Put([]byte(node.Path), s.marshal(node))
			}
			return nil
		})
		creates = make(map[string]*tree.Node, 500)
	}
	for {
		select {
		case node := <-s.autoBatchChan:
			if queued, ok := creates[node.GetPath()]; ok {
				if node.GetMTime() > queued.MTime {
					creates[node.GetPath()] = node
				}
			} else {
				creates[node.GetPath()] = node
			}
			nextTime = 300 * time.Millisecond
			if len(creates) == 500 {
				flush()
			}
		case <-time.After(nextTime):
			// Autoflush after 300ms idle
			flush()
			nextTime = 1 * time.Hour
		case <-s.autoBatchClose:
			flush()
			log.Logger(context.Background()).Debug("Closing AutoBatcher")
			return
		}
	}
}

func (s *BoltSnapshot) StartSession(ctx context.Context, rootNode *tree.Node, silent bool) (*tree.IndexationSession, error) {
	s.createsSession = make(map[string]*tree.Node)
	s.createsSessionLock = &sync.Mutex{}
	return &tree.IndexationSession{
		Uuid: uuid.New(),
	}, nil
}

func (s *BoltSnapshot) FlushSession(ctx context.Context, sessionUuid string) error {
	if len(s.createsSession) == 0 {
		return nil
	}
	log.Logger(ctx).Info("Flushing BoltDB creates", zap.Int("creates", len(s.createsSession)))
	e := s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketName)
		if b == nil {
			return fmt.Errorf("cannot find root bucket")
		}
		s.createsSessionLock.Lock()
		for _, node := range s.sortByKey(s.createsSession) {
			b.Put([]byte(node.Path), s.marshal(node))
		}
		s.createsSessionLock.Unlock()
		return nil
	})
	s.createsSession = make(map[string]*tree.Node)
	return e
}

func (s *BoltSnapshot) FinishSession(ctx context.Context, sessionUuid string) error {
	s.FlushSession(ctx, sessionUuid)
	return nil
}

func (s *BoltSnapshot) CreateNode(ctx context.Context, node *tree.Node, updateIfExists bool) (err error) {
	if s.createsSession != nil {
		return s.db.View(func(tx *bbolt.Tx) error {
			b := tx.Bucket(bucketName)
			if b == nil {
				return fmt.Errorf("cannot find root bucket")
			}
			s.createsSessionLock.Lock()
			defer s.createsSessionLock.Unlock()
			// Create parents if necessary
			dir := strings.Trim(path.Dir(node.Path), "/")
			if dir != "" && dir != "." {
				parts := strings.Split(strings.Trim(path.Dir(node.Path), "/"), "/")
				for i := 0; i < len(parts); i++ {
					pKey := strings.Join(parts[:i+1], "/")
					if _, ok := s.createsSession[pKey]; ok {
						continue
					}
					if ex := b.Get([]byte(pKey)); ex != nil {
						continue
					}
					s.createsSession[pKey] = &tree.Node{Path: pKey, Type: tree.NodeType_COLLECTION, Etag: "-1"}
				}
			}
			s.createsSession[node.Path] = node
			return nil
		})
	} else {
		return s.db.View(func(tx *bbolt.Tx) error {
			b := tx.Bucket(bucketName)
			if b == nil {
				return fmt.Errorf("cannot find root bucket")
			}
			// Create parents if necessary
			dir := strings.Trim(path.Dir(node.Path), "/")
			if dir != "" && dir != "." {
				parts := strings.Split(strings.Trim(path.Dir(node.Path), "/"), "/")
				for i := 0; i < len(parts); i++ {
					pKey := strings.Join(parts[:i+1], "/")
					if ex := b.Get([]byte(pKey)); ex == nil {
						//b.Put([]byte(pKey), s.marshal(&tree.Node{Path: pKey, Type: tree.NodeType_COLLECTION, Etag: "-1"}))
						s.autoBatchChan <- &tree.Node{Path: pKey, Type: tree.NodeType_COLLECTION, Etag: "-1"}
					}
				}
			}
			s.autoBatchChan <- node
			return nil
			//return b.Put([]byte(node.Path), s.marshal(node))
		})

		/*
			return s.db.Update(func(tx *bbolt.Tx) error {
				b := tx.Bucket(bucketName)
				if b == nil {
					return fmt.Errorf("cannot find root bucket")
				}
				// Create parents if necessary
				dir := strings.Trim(path.Dir(node.Path), "/")
				if dir != "" && dir != "." {
					parts := strings.Split(strings.Trim(path.Dir(node.Path), "/"), "/")
					for i := 0; i < len(parts); i++ {
						pKey := strings.Join(parts[:i+1], "/")
						if ex := b.Get([]byte(pKey)); ex == nil {
							b.Put([]byte(pKey), s.marshal(&tree.Node{Path: pKey, Type: tree.NodeType_COLLECTION, Etag: "-1"}))
						}
					}
				}
				return b.Put([]byte(node.Path), s.marshal(node))
			})
		*/
	}
}

func (s *BoltSnapshot) DeleteNode(ctx context.Context, path string) (err error) {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketName)
		if b == nil {
			return fmt.Errorf("cannot find root bucket")
		}
		if d := b.Get([]byte(path)); d != nil {
			b.Delete([]byte(path))
			// Delete children
			c := b.Cursor()
			prefix := []byte(strings.TrimRight(path, "/") + "/")
			var children [][]byte
			for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
				children = append(children, k)
			}
			for _, k := range children {
				b.Delete(k)
			}
		}
		return nil
	})
}

func (s *BoltSnapshot) MoveNode(ctx context.Context, oldPath string, newPath string) (err error) {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketName)
		if b == nil {
			return fmt.Errorf("cannot find root bucket")
		}
		var moves [][]byte
		if d := b.Get([]byte(oldPath)); d != nil {
			moves = append(moves, []byte(oldPath))
			// Delete children
			c := b.Cursor()
			prefix := []byte(oldPath + "/")
			for k, _ := c.Seek(prefix); k != nil && (string(k) == oldPath || bytes.HasPrefix(k, prefix)); k, _ = c.Next() {
				moves = append(moves, k)
			}
		}
		for _, m := range moves {
			renamed := path.Join(newPath, strings.TrimPrefix(string(m), oldPath))
			if node, e := s.unmarshal(b.Get(m)); e == nil {
				node.Path = renamed
				b.Delete(m)
				b.Put([]byte(renamed), s.marshal(node))
			}
		}
		return nil
	})
}

func (s *BoltSnapshot) IsEmpty() bool {
	return s.empty
}

func (s *BoltSnapshot) Close(delete ...bool) {
	s.db.Close()
	if len(delete) > 0 && delete[0] && s.folderPath != "" {
		os.RemoveAll(s.folderPath)
	}
	close(s.autoBatchClose)
}

func (s *BoltSnapshot) Capture(ctx context.Context, source model.PathSyncSource, paths ...string) error {
	// Capture in temporary bucket
	e := s.db.Update(func(tx *bbolt.Tx) error {
		var capture *bbolt.Bucket
		var e error
		if b := tx.Bucket(captureBucketName); b != nil {
			if e = tx.DeleteBucket(captureBucketName); e != nil {
				return e
			}
		}
		if capture, e = tx.CreateBucket(captureBucketName); e != nil {
			return e
		}
		if len(paths) == 0 {
			return source.Walk(func(path string, node *tree.Node, err error) {
				capture.Put([]byte(path), s.marshal(node))
			}, "/", true)
		} else {
			for _, p := range paths {
				e := source.Walk(func(path string, node *tree.Node, err error) {
					if err != nil {
						log.Logger(ctx).Error("BoltDB:Capture: ignoring path, error is not nil", zap.String("path", path), zap.Error(err))
						return
					}
					capture.Put([]byte(path), s.marshal(node))
				}, p, true)
				if e != nil {
					return e
				}
			}
			return nil
		}
	})
	if e != nil {
		return e
	}
	// Now copy all to original bucket
	if e = s.db.Update(func(tx *bbolt.Tx) error {
		var clear *bbolt.Bucket
		var e error
		if b := tx.Bucket(bucketName); b != nil {
			if e = tx.DeleteBucket(bucketName); e != nil {
				return e
			}
		}
		if clear, e = tx.CreateBucket(bucketName); e != nil {
			return e
		}
		if captured := tx.Bucket(captureBucketName); captured != nil {
			if e := captured.ForEach(func(k, v []byte) error {
				return clear.Put(k, v)
			}); e != nil {
				return e
			}
		}
		return tx.DeleteBucket(captureBucketName)
	}); e == nil {
		s.empty = false
	}
	return e
}

func (s *BoltSnapshot) LoadNode(ctx context.Context, path string, extendedStats ...bool) (node *tree.Node, err error) {
	if path == "/" {
		// Return fake Root
		node = &tree.Node{Path: "/"}
	} else {
		err = s.db.View(func(tx *bbolt.Tx) error {
			if b := tx.Bucket(bucketName); b != nil {
				value := b.Get([]byte(path))
				if value != nil {
					if node, err = s.unmarshal(value); err != nil {
						return err
					}
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		} else if node == nil {
			err = errors.NotFound("not.found", "node not found in snapshot %s", path)
		}
	}
	if len(extendedStats) > 0 && extendedStats[0] {
		var size, folders, files int64
		s.Walk(func(path string, node *tree.Node, err error) {
			if node.IsLeaf() {
				size += node.Size
				files += 1
			} else {
				folders += 1
			}
		}, path, true)
		node.SetMeta("RecursiveChildrenSize", size)
		node.SetMeta("RecursiveChildrenFiles", files)
		node.SetMeta("RecursiveChildrenFolders", folders)
	}
	return
}

func (s *BoltSnapshot) GetEndpointInfo() model.EndpointInfo {
	return model.EndpointInfo{
		URI: "snapshot://" + s.name,
		RequiresNormalization: false,
		RequiresFoldersRescan: false,
	}
}

func (s *BoltSnapshot) Walk(walknFc model.WalkNodesFunc, root string, recursive bool) (err error) {
	root = strings.Trim(root, "/") + "/"
	err = s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketName)
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			key := string(k)
			if root != "/" && !strings.HasPrefix(key, root) {
				return nil
			}
			if !recursive && strings.Contains(strings.TrimPrefix(key, root), "/") {
				return nil
			}
			if node, e := s.unmarshal(v); e == nil {
				walknFc(key, node, nil)
			}
			return nil
		})
	})
	return err
}

func (s *BoltSnapshot) Watch(recursivePath string) (*model.WatchObject, error) {
	return nil, fmt.Errorf("not.implemented")
}

func (s *BoltSnapshot) ComputeChecksum(node *tree.Node) error {
	return fmt.Errorf("not.implemented")
}

func (s *BoltSnapshot) marshal(node *tree.Node) []byte {
	store := node.Clone()
	store.MetaStore = nil
	data, _ := proto.Marshal(node)
	return data
}

func (s *BoltSnapshot) unmarshal(value []byte) (*tree.Node, error) {
	var n tree.Node
	if e := proto.Unmarshal(value, &n); e != nil {
		return nil, e
	} else {
		return &n, nil
	}
}
