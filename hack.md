# Hack items

* build/make process
* kube-dns program


## build/make process
Added below files for go binary building and docker iamges building
* wise2c_dns_build.sh -- build go binary
* kube-dns-amd64-dockerfile -- build docker image

## kube-dns program

Added below files

* pkg/dns/dns_wise2c.go

Changed files

* pkg/dns/dns.go -- add wise2c hack codes into process of handling service adding/updating/removing

