# secret-volume [![Travis](https://img.shields.io/travis/negz/secret-volume.svg?maxAge=300)](https://travis-ci.org/negz/secret-volume/) [![Codecov](https://img.shields.io/codecov/c/github/negz/secret-volume.svg?maxAge=3600)](https://codecov.io/gh/negz/secret-volume/) [![Godoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/negz/secret-volume)

`secret-volume` is a small daemon intended to manage sets of files containing secrets like database passwords on behalf of containerised services.

Container orchestration platforms like [Helios](http://github.com/spotify/helios) can call the `secret-volume` API to request secrets be procured and stored in a 'secret volume', then request said volume be mounted into the container of the service that must consume the secrets. Currently it supports producing secrets by querying [Talos](https://github.com/spotify/talos). Secret files are stored in-memory using either `tmpfs` volumes or an [Afero](https://github.com/spf13/afero) `MemMapFs`.

# Installing
`secret-volume` is in early development and should not yet be used in production. To build a binary with debug loggin:
```bash
$ go get github.com/negz/secret-volume
$ go build --tags debug src/github.com/negz/secret-volume/secretvolume.go 
$ ./secretvolume --help
```

