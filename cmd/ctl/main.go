/*
 * Copyright (c) 2018. Abstrium SAS <team (at) pydio.com>
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

// Package ctl provides the Pydio Cells Control client for performing grpc requests on running services.
package main

import (
	"github.com/pydio/cells/cmd/ctl/cmd"
	"github.com/pydio/cells/common"

	_ "github.com/pydio/cells/discovery/nats"
)

func main() {
	common.PackageType = "cli"
	common.PackageLabel = "Pydio Cells Control"
	cmd.Execute()
}
