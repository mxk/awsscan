awsscan
=======

[![GoDoc](https://godoc.org/github.com/mxk/awsscan?status.svg)](https://godoc.org/github.com/mxk/awsscan)

A tool for mapping all resources in an AWS account.

Adding scan service support
---------------------------

1. Build svcgen: `cd aws/scan/svcgen && go install -tags codegen && cd -`
2. Generate service template: `svcgen aws/scan/svc <service-name> [...]`
   * Service names match [AWS SDK v2 service directory
   names](https://github.com/aws/aws-sdk-go-v2/tree/master/service).
3. Open the template file `aws/scan/svc/<service-name>.go` and remove
   unnecessary API calls.
4. Implement non-root API calls. See other services for examples. Most patterns
   are handled by `Ctx.Split`, `Ctx.Group`, and `Ctx.CopyInput`.
5. Format code and run unit tests: `go fmt ./... && go test ./...`
6. Build and run a scan: `go install && awsscan -services <service-name>`
   * Ideally, this should be done with an account that contains resources for
     the new service, but a scan that returns nothing is still useful to verify
     that the root calls do not return unexpected errors.
