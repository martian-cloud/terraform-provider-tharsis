# Terraform Tharsis Provider

This generic template project will define various standard files that should be included whenever creating a new project.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.17

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Using the provider

To use this provider you must add the provider definition to your configuration. You can use either a static token or service account for authentication.

To use static tokens you can define your provider block as follows:

```hcl
provider "tharsis" {
    host = "https://tharsis.example.com"
    static_token = "my-static-token"
}
```

For a service account it would look like this:

```hcl
provider "tharsis" {
    host = "https://tharsis.example.com"
    service_account_name = "my-service_account-name"
    service_account_token = "my-service-account-token"
}
```

Alternatively, you can provide these values by environment variables.

| Environment Variable            | Definition                                                |
| ------------------------------- | --------------------------------------------------------- |
| `THARSIS_ENDPOINT`              | The host for Tharsis.                                     |
| `THARSIS_STATIC_TOKEN`          | The static token to use with the provider.                |
| `THARSIS_SERVICE_ACCOUNT_PATH`  | The service account's full path to use with the provider. |
| `THARSIS_SERVICE_ACCOUNT_TOKEN` | The service account token to use with the provider.       |

The provider block values take precedence over environment variables. It is recommended to use configuration values to define the provider over environment variables, especially if you are defining the provider more than once.

In the case both static and service account credentials are passed, service account credentials take precedence.

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `go generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

_Note:_ Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```

## Security

If you've discovered a security vulnerability in the Terraform Tharsis Provider, please create a new issue in this project and ask for a preferred security contact so we can setup a private means of communication (the issue should NOT include any information related to the security vulnerability).

## Statement of support

Please submit any bugs or feature requests for Tharsis.  Of course, MR's are even better.  :)

## License

Terraform Tharsis Provider is distributed under [Mozilla Public License v2.0](https://www.mozilla.org/en-US/MPL/2.0/).
