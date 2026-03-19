# Dependencies Guide

This guide covers dependency management for Agones development.

## Dependencies

This project uses the [go modules](https://github.com/golang/go/wiki/Modules) as its manager. You can see the list of dependencies [here](https://github.com/agones-dev/agones/blob/main/go.mod).

### Vendoring

Agones uses [module vendoring](https://tip.golang.org/cmd/go/#hdr-Modules_and_vendoring) to reliably produce versioned builds with consistent behavior.

Adding a new dependency to Agones:

*  `go mod tidy` This will import your new deps into the go.mod file and trim out any removed dependencies.
*  `go mod vendor` Pulls module code into the vendor directory.

Sometimes the code added to vendor may not include a subdirectory that houses code being used but not as an import
(protos passed as args to scripts is a good example). In this case you can go into the module cache and copy what you need to the path in vendor.

Here is an example for getting third_party from grpc-ecosystem/grpc-gateway v1.5.1 into vendor:

*  AGONES_PATH=/wherever/your/agones/path/is
*  cp -R $GOPATH/pkg/mod/github.com/grpc-ecosystem/grpc-gateway@v1.5.1/third_party $AGONES_PATH/vendor/github.com/grpc-ecosystem/grpc-gateway/

Note the version in the pathname. Go may eliminate the need to do this in future versions.

We also use vendor to hold code patches while waiting for the project to release the fixes in their own code. An example is in [k8s.io/apimachinery](https://github.com/agones-dev/agones/issues/414) where a fix will be released later this year, but we updated our own vendored version in order to fix the issue sooner.

## Next Steps

- See [Building and Testing Guide](building-testing.md) for information on building with dependencies
- See [Make Reference](make-reference.md) for dependency-related make targets