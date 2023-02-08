# GitSec POC v1 Backend

## Table of Contents

- [Introduction](#Introduction)
- [Features](#Features)
- [Usage](#Usage)
- [Configuration](#Configuration)
- [Makefile commands](#Makefile-commands)
- [Todo](#Todo)
- [Contributing](#Contributing)
- [License](#License)

## Introduction
Gitsec POC v1 Backend is a implementation of a Git server that allows clients to interact with Git repositories over HTTP.
It is designed to be used as a proof of concept and does not include features such as authentication, SSL/TLS, Git hooks, or repository management.

## Features
The Gitsec PoC v1 Backend supports the Git HTTP protocol, which allows clients to fetch and push to repositories over HTTP.
This protocol is used by Git clients to communicate with the server and exchange data.

The following Git commands are supported in this version:

* `git clone` - used to clone a repository from the server
* `git fetch` - used to retrieve new data from a remote repository
* `git push` - used to push data to a remote repository
* `git pull` - used to fetch and merge data from a remote repository into the local repository

## Usage
To use Gitsec POC v1 Backend, you will need to have Go and Make installed on your system.
You can then clone the repository and build the server using the following commands:
```shell
$ git clone https://github.com/uddugteam/gitsec-backend.git
$ cd gitsec-backend
$ make build
```

This will build an executable file called gitsec-backend in the project root directory.
You can then start the server using the following command:
```shell
$ ./gitsec-backend serve
```

By default, the server will listen on http://localhost:8080 and serve Git repositories from the `.repos` directory.
You can change these default values by setting the `HTTP_PORT` and `GIT_PATH` environment variables, respectively.

For all next command you should replace repo.git with the name of your repository.

To add the server as a remote origin to a local repository, you can use the git remote add command:
```shell
$ git remote add origin http://localhost:8080/repos/repo.git
```

And push commits from local repository to remote one:
```shell
$ git push
```

You can also clone the repository using the git clone command:
```shell
$ git clone http://localhost:8080/repos/repo.git
```

## Configuration
The following environment variables can be used to configure the server:

* `HTTP_PORT`: The port number on which the server will listen for HTTP requests. Default is `8080`
* `GIT_PATH`: The directory where the Git repositories are stored. Default is `.repos`

## Makefile commands
* `make build`: Builds the `gitsec-backend` executable
* `make run`: Runs the server in development mode with race detection enabled
* `make test`: Runs the tests for the project
* `make test-coverage`: Runs the tests for the project and generates a coverage report
* `make lint`: Runs the linter to check

## Todo
- [x] Add support for IPFS storage
- [x] Add performance optimisation for IPFS storage
- [ ] Add support for onchain registry
- [ ] Add repository management features
- [ ] Add support for SSH protocol
- [ ] Add authentication
- [ ] Add SSL/TLS support
- [ ] Add Git hooks support

## Contributing
To contribute to the Gitsec POC v1 backend, fork the repository and create a pull request with your changes.
Make sure to include thorough testing and documentation for your changes.

## License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.