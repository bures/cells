<img src="https://github.com/pydio/cells/wiki/images/PydioCellsColor.png" width="400" />

[Homepage](https://pydio.com/) | [Dev Guide](https://pydio.com/en/docs/developer-guide) | [GitHub-Repository](https://github.com/pydio/cells) |
[Issue-Tracker](https://github.com/pydio/cells/issues)

[![License Badge](https://img.shields.io/badge/License-AGPL%203%2B-blue.svg)](LICENSE)
[![GoDoc](https://godoc.org/github.com/pydio/cells?status.svg)](https://godoc.org/github.com/pydio/cells)
[![Build Status](https://travis-ci.org/pydio/cells.svg?branch=master)](https://travis-ci.org/pydio/cells)
[![Go Report Card](https://goreportcard.com/badge/github.com/pydio/cells?rand=3)](https://goreportcard.com/report/github.com/pydio/cells)

Pydio Cells is the nextgen file sharing platform for organizations. It is a full rewrite of the Pydio project using the Go language following a micro-service architecture.

<p align="center"> 
  <img src="https://github.com/pydio/cells-dist/raw/master/resources/v1.4.0/homepage.png" width="600" style="border: 3px solid #e0e0e0; border-radius: 5px;"/>
</p>

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for **development** and testing purposes. See the [Deployment section below](#deployment)  for notes on how to deploy the project on a live system.

### Prerequisites

The following elements are required to compile and run pydio on your machine

- Go language v1.10 or higher (tested with 1.10.x, 1.11.x, 1.12.x), with a [correctly configured](https://golang.org/doc/install#testing) Go toolchain,
- MySQL database 5.6 or higher (or MariaDB equivalent). The new mysql 8 authentication method is supported starting at Cells 1.4.1.

_Note: We have developped and tested Pydio Cells on MacOS, Ubuntu, Debian and CentOS. Windows version might still have unknown glitches and is not yet supported._

### Installing

Assuming that your system meets the above prerequisites, building the **Pydio Cells** backend from the source code is quite straight forward:

```sh
# Retrieve the code
go get -u github.com/pydio/cells
# From this line on, we assume you are in Pydio Cells' code roots directory
cd $GOPATH/src/github.com/pydio/cells
# Build your binary
make dev
```

To have the environment running, you must also:

- Create a database in your chosen DB server,
- Run the Pydio Cells installer that will guide you through the necessary steps: you might refer to the [official documentation](https://pydio.com/en/docs/cells/v1/install-pydio-cells) for additional information.

```sh
./cells install
```

#### Note on the third party libraries

We still currently manage third party dependencies via the [vendor mechanism](https://github.com/kardianos/govendor): shortly said, we pick up and maintain specific versions of the sources for each dependency we use by copying them in the `vendor/` subfolder. The binary is built using these codes.

When you clone the `github.com/pydio/cells` repository, you then also have an embedded local copy of  all the sources for you to investigate. Yet, you should not try to directly modify code that have been _vendored_.

Please also not that we had to fork a few libraries before integrating them as dependencies, most important being dex and minio. If you need to modify this part of the code, please get in touch with us.

## Running the tests

To run the tests, simply do

```sh
go test -v ./...
```

Please read the [CONTRIBUTING.md](CONTRIBUTING.md) document if you wish to add more tests or contribute to the code.

## Deployment

Binaries are currently provided for [Linux and MacOSX distributions](https://pydio.com/en/download). To deploy them on a live system, please see the [Installation Guide](https://pydio.com/en/docs/cells/v1/installation-guides) instructions.

## Built With

Pydio Cells uses many open source golang libraries. Most important ones are listed below, please see [DEPENDENCIES](DEPENDENCIES) for an exhaustive list of other libs and their licenses.

- [Micro](https://github.com/micro/micro) - Micro-service framework
- [Minio](https://github.com/minio/minio) - Objects server implementing s3 protocol

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct, and the process for submitting pull requests to us. You ca find a comprehensive [Developer Guide](https://pydio.com/en/docs/developer-guide) on our web site. Our online docs are open source as well, feel free to improve them by contributing!

## Versioning

We use [SemVer](http://semver.org/) for versioning. For the versions available, see the [tags on this repository](https://github.com/pydio/cells/tags).

## Authors

See the list of [contributors](https://github.com/pydio/cells/graphs/contributors) who participated in this project. Pydio Cells is also a continuation of the Pydio project and many contributions were ported from [pydio-core](https://github.com/pydio/pydio-core) to the code that can be found under `frontend/front-srv/assets`.

## License

This project is licensed under the AGPLv3 License - see the [LICENSE](LICENSE) file for details.
