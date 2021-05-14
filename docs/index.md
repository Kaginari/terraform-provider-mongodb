
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
  ssl = true
  replica_set = "replica-set" #optional
  
}
```

## Example Usage with ssl

```hcl
# Configure the MongoDB Provider
provider "mongodb" {

  insecure_skip_verify = true  # default false (set to true to ignore hostname verification) 
  # -> specify either
  cert_path = pathexpand("path/to/certificate")

  # -> or the following ( you can use this if you are using a custom key and cert)
  
  ca_material   = file(pathexpand("path/to/certificate/ca.pem")) # this can be omitted
  
  cert_material = file(pathexpand("path/to/certificate/cert.pem"))
  key_material  = file(pathexpand("path/to/certificate/key.pem"))

  
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
  ssl = true
  # -> specify either
  cert_path = pathexpand("~/.mongodb")

  # -> or the following
  ca_material   = file(pathexpand("~/.mongodb/ca.pem")) # this can be omitted
  cert_material = file(pathexpand("~/.mongodb/cert.pem"))
  key_material  = file(pathexpand("~/.mongodb/key.pem"))
  
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

* `cert_path` - (Optional) Path to a directory with certificate information for connecting to the Docker host via TLS. It is expected that the 3 files {ca, cert, key}.pem are present in the path. If the path is blank, the MONGODB_CERT_PATH will also be checked.

* `ca_material`, `cert_material`, `key_material`, - (Optional) Content of ca.pem, cert.pem, and key.pem files for TLS authentication. Cannot be used together with cert_path. If ca_material is omitted the client does not check the servers certificate chain and host name.

* `username ` - (Optional) Specifies a username with which to authenticate to the MongoDB database. It must be
  provided, but it can also be sourced from the `MONGO_USR`
  environment variable.
* `password  ` - (Optional) Specifies a password with which to authenticate to the MongoDB database. It must be
  provided, but it can also be sourced from the `MONGO_PWD`
  environment variable.
* `auth_database   ` - (Required) Specifies the authentication database where the specified `username` has been created.
* `ssl   ` - (Optional) `default = false `set it to true to connect to a deployment using TLS/SSL with SCRAM authentication.
  
