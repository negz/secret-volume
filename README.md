# secret-volume [![Travis](https://img.shields.io/travis/negz/secret-volume.svg?maxAge=300)](https://travis-ci.org/negz/secret-volume/) [![Codecov](https://img.shields.io/codecov/c/github/negz/secret-volume.svg?maxAge=3600)](https://codecov.io/gh/negz/secret-volume/) [![Godoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/negz/secret-volume)

`secret-volume` is a small daemon intended to manage sets of files containing secrets like database passwords on behalf of containerised services.

Container orchestration platforms like [Helios] can call the `secret-volume` API to request secrets be procured and stored in a 'secret volume', then request said volume be mounted into the container of the service that must consume the secrets. Currently it supports producing secrets by querying [Talos]. Secret files are stored in-memory using either `tmpfs` volumes or an [Afero] `MemMapFs`.

# Running
By default `secret-volume` listens for HTTP connections on port 10002 on all interfaces with no secret providers enabled. Provide a [Talos] SRV record query (i.e. `_talos._https.example.org`) to enable the Talos provider.

All `secret-volume` flags can also be supplied as env vars per [Kingpin]. For example setting `SECRET_VOLUME_PARENT=/differentsecrets` be equivalent to `--parent=/differentsecrets`.

```
$ bin/secret-volume --help
usage: secret-volume [<flags>]

Manages sets of files containing secrets.

Flags:
  --help                 Show context-sensitive help (also try --help-long and --help-man).
  --talos-srv=TALOS-SRV  Enables Talos by providing an SRV record at which to find it.
  --addr=":10002"        Address at which to serve requests (host:port).
  --ns=NS                DNS server to use to lookup SRV records (host:port).
  --parent="/secrets"    Directory under which to mount secret volumes.
  --virtual              Use an in-memory filesystem and a no-op mounter.
  --close-after=1m       Wait this long at shutdown before closing HTTP connections.
  --kill-after=2m        Wait this long at shutdown before exiting.
```

# API
To request that `secret-volume` procure secrets from Talos and store them at `/secrets/awesomevolume` send an HTTP POST to `http://secretvolume:10002/` with the following JSON encoded body:
```json
{
  "ID": "awesomevolume",
  "Source": "Talos",
  "Tags": {"awesome": ["very"]},
  "KeyPair": {
    "Certificate": "-----BEGIN CERTIFICATE-----\nMIIG[...]D5/Hyg==\n-----END CERTIFICATE-----\n",
    "PrivateKey": "-----BEGIN RSA PRIVATE KEY-----\nMIIJ[...]KAQmg=\n-----END RSA PRIVATE KEY-----\n"
  }
}
```
* The `Source` informs `secret-volume` this is a volume of [Talos] secrets.
* The `KeyPair` is a PEM encoded certificate and private key used to authenticate to the secret source on behalf of the owner of the secrets (i.e. a host or Docker container). The `KeyPair` is discarded once secrets have been procured.
* The `Tags` are passed to [Talos] for use with the `unsafe_scopes` option. In this case the Talos URL would be `https://talos.example.org?awesome=very`.

A successful creation will result in a HTTP 200 status code and a JSON rendering of the created volume, sans `KeyPair`, i.e.:
```json
{
  "ID": "awesomevolume",
  "Source": "Talos",
  "Tags": {"awesome": ["very"]}
}
```

You can list extant volumes by sending an HTTP GET to `http://secretvolume:10002/`. The returned list will look like:
```json
[
  {
    "ID": "awesomevolume",
    "Source": "Talos",
    "Tags": {"awesome": ["very"]}
  },
  {"ID": "..."}
]
```
Note the `KeyPair` is omitted.

You may also query for an individual volume by sending an HTTP GET to `http://secretvolume:10002/<id>`, for example `http://secretvolume:10002/awesomevolume`. The JSON result will be identical to that returned when the volume was created. `secret-volume` will return an HTTP 404 status code if no such volume exists.

To destroy a volume send an HTTP DELETE to `http://secretvolume:10002/<id>`. The volume will be unmounted and its secrets will no longer be available.

# Building
`secret-volume` can be built as a native binary, or a Docker container. Builds for `GOOS=linux` (i.e. Docker) will default to storing secrets in `tmpfs`. Any other OS (i.e. `GOOS=darwin`) will only support using `MemMapFs`.

## From source
`secret-volume` uses [Glide] to manage vendor dependencies. Run the following from `$GOPATH/src/github.com/negz/secret-volume`:

```
$ glide install
[INFO]  Downloading dependencies. Please wait...
...
[INFO]  Replacing existing vendor dependencies
$ go install .
$ $GOPATH/bin/secret-volume --help
```

## Using Docker
Run the following from `github.com/negz/secret-volume` (a working Go environment should not be required):
```
$ docker build -q  --tag negz/secret-volume .
Sending build context to Docker daemon 30.14 MB
...
Successfully built somehash
$ docker run -d -p 10002 negz/secret-volume
6b0867a57c7596698a808589ee1d421d630facf1ea207eb81e84dfd9b4cc395d
messremb(negz@secret-volume)$ docker ps
CONTAINER ID        IMAGE                COMMAND                CREATED             STATUS              PORTS                      NAMES
6b0867a57c75        negz/secret-volume   "/dumb-init /go/bin/   3 seconds ago       Up 2 seconds        0.0.0.0:32775->10002/tcp   reverent_darwin
$ echo $DOCKER_HOST
tcp://192.168.99.100:2376
messremb(negz@secret-volume)$ curl http://192.168.99.100:32775
[]
$ docker logs 6b0867a57c75
{"level":"info","ts":1476240696.763892,"msg":"http request","method":"GET","url":"/","addr":"192.168.99.1:53060"}
```

## Debug Builds
By default `secret-volume` emits very few logs. Use the `debug` build tag to build a version with verbose debug logging enabled. Note that debug builds emit human-friendly logs while production builds emit JSON.

```
$ go run -tags debug src/github.com/negz/secret-volume/secretvolume.go
[D] 2016-10-12T02:56:35Z notlinux.go:13: Forcing in-memory filesystem and noop mounter due to non-Linux environment
[I] 2016-10-12T02:56:41Z handlers.go:154: http request method=GET url=/ addr=[::1]:53109
[D] 2016-10-12T02:56:41Z manager.go:267: listing volumes
```

# Testing
Use `go test` as usual to test. You may wish to use `-v -tags debug` to see debug logging. Integration tests are not run by default. Enable them with `-tags integration`. From `$GOPATH/src/github.com/negz/secret-volume`

```
$ go test -race -cover -tags integration $(glide novendor)
?       github.com/negz/secret-volume/api       [no test files]
?       github.com/negz/secret-volume/cmd       [no test files]
?       github.com/negz/secret-volume/fixtures  [no test files]
ok      github.com/negz/secret-volume/secrets   1.087s  coverage: 69.7% of statements
ok      github.com/negz/secret-volume/server    1.036s  coverage: 55.3% of statements
ok      github.com/negz/secret-volume/volume    1.029s  coverage: 69.2% of statements
ok      github.com/negz/secret-volume   1.076s  coverage: 0.0% of statements
```

[helios]: http://github.com/spotify/helios
[talos]: http://github.com/spotify/talos
[afero]: http://github.com/spf13/afero
[glide]: https://github.com/Masterminds/glide
[kingpin]: https://godoc.org/gopkg.in/alecthomas/kingpin.v2#Application.DefaultEnvars
