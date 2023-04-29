
# MongoDB Provider

The MongoDB provider is used to interact with the resources supported by [MongoDB](https://www.mongodb.com/). The provider needs to be configured with the proper credentials before it can be used.

Use the navigation to the left to read about the available provider resources.

You may want to consider pinning the [provider version](https://www.terraform.io/docs/configuration/providers.html#provider-versions) to ensure you have a chance to review and prepare for changes.

## Example Usage

```hcl
# Configure the MongoDB Provider
provider "mongodb" {
  host = "127.0.0.1"
  port = "27017"
  username = "root"
  password = "root"
  auth_database = "admin"
  tls = true
  ca_file = "ca.pem"
  certificate_key_file = "cert_key.pem"
  replica_set = "replica-set" #optional
  retrywrites = false # default true
  direct = true // default false
  proxy = "socks5://myproxy:8080" // Optional
}
```

## Example Usage with tls

```hcl
# Configure the MongoDB Provider
provider "mongodb" {
  ca_file = "ca.pem"
  certificate_key_file = "cert_key.pem"
}
```

### Environment variables

You can also provide your credentials via the environment variables, MONGO_HOST, MONGO_PORT, MONGO_USR, and MONGO_PWD respectively:

```hcl
provider "mongodb" {
  auth_database = "admin"
}
```

Usage (prefix the export commands with a space to avoid the keys being recorded in OS history):

```shell
$  export MONGO_HOST="xxxx"
$  export MONGO_PORT="xxxx"
$  export MONGO_USR="xxxx"
$  export MONGO_PWD="xxxx"
$ terraform plan
```




## Certificate information :
Specify certificate information either with a directory or directly with the content of the files for connecting to the Mongodb host via TLS.

```hcl
provider "mongodb" {
  host = "127.0.0.1"
  port = "27017"
  username = "root"
  password = "root"
  auth_database = "admin"
  tls = true
  ca_file = "ca.pem"
  certificate_key_file = "cert_key.pem"
  }
```
## Argument Reference

In addition to [generic `provider`
arguments](https://www.terraform.io/docs/configuration/providers.html) (e.g.
`alias` and `version`), the following arguments are supported in the MongoDB
`provider` block:

* `host` - (Optional) This is the host your MongoDB Server. It must be
  provided, but it can also be sourced from the `MONGO_HOST`
  environment variable.
* `port` - (Optional) This is the port that your MongoDB Server uses. It must be
  provided, but it can also be sourced from the `MONGO_PORT`
  environment variable.

* `certificate_key_file` - (Optional) PEM file containing client private key and certificate  for connecting via TLS. I. If the path is blank, the MONGODB_CA_FILE will also be checked.

* `ca_file` - (Optional) CA certificate(s) file for connecting via TLS. If the path is blank, the MONGODB_CERT_KEY_FILE will also be checked.

* `username ` - (Optional) Specifies a username with which to authenticate to the MongoDB database. It must be
  provided, but it can also be sourced from the `MONGO_USR`
  environment variable.
* `password  ` - (Optional) Specifies a password with which to authenticate to the MongoDB database. It must be
  provided, but it can also be sourced from the `MONGO_PWD`
  environment variable.
* `auth_database   ` - (Required) Specifies the authentication database where the specified `username` has been created.
* `tls   ` - (Optional) `default = false `set it to true to connect to a deployment using TLS with SCRAM authentication.
* `retrywrites   ` - (Optional) `default = true `Retryable writes allow MongoDB drivers to automatically retry certain write operations a single time if they encounter network errors, or if they cannot find a healthy primary in the replica sets or sharded cluster.
* `direct   ` - (Optional) `default = false ` determine if a direct connection is needed..
* `proxy   ` - (Optional) `default = "" ` determine if connecting via a SOCKS5 proxy is needed, it can also be sourced from the `ALL_PROXY` or `all_proxy` environment variable.

