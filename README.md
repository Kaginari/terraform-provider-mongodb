# Terraform Provider Mongodb

This repository is a Algolia Mongodb for [Terraform](https://www.terraform.io).

### Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 0.13
- [Go](https://golang.org/doc/install) >= 1.15

### Installation

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the `make install` command:

````bash
git clone https://github.com/Kaginari/terraform-provider-mongodb
cd terraform-provider-mongodb
make install
````

### To test locally 

**1: lunch mongo instance**
````bash
cd docker
docker-compose up -d
````
**2: Build the provider**

follow the [Installation](#Installation)

**3: Use the provider**

````bash
cd mongodb
make apply
````