# pce-osi
CLI tool for working with Pextra-specific [OSI images](https://github.com/opencontainers/image-spec/blob/main/spec.md).

Pextra-specific extensions to the OSI image specification are documented in the [PEXTRA_OSI_EXTENSIONS.md](./PEXTRA_OSI_EXTENSIONS.md) file.

# Development
This CLI is written in Go. To build the tool, you need to have [Go installed](https://go.dev/doc/install).

## Tests
### Integration Tests
The integration tests require a Linux-based system with specific tools installed. Tests are skipped if the required tools are not available. Some tests require root privileges to run.

To run the integration tests, use the `integration` build tag:
```bash
go test -v -tags=integration -cover ./...
```
This will run all tests in the package, including integration tests, and display a code coverage report.

To get a more detailed report of the code coverage, run:
```bash
go test -v -tags=integration -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```
This will generate an HTML report of the code coverage and open it in your default web browser.

# License
This repository is licensed under the [Apache License 2.0](./LICENSE).
