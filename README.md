await ![version v1.0.0](https://img.shields.io/badge/version-v1.0.0-brightgreen.svg) ![License MIT](https://img.shields.io/badge/license-MIT-blue.svg)
=============

Utility to await before launching a command.

It allows you to:
* Wait for other services to be available using TCP, HTTP(S), unix before starting the main process.

For example, it can delay the starting of a python application until the database is running and listening on the TCP port.

Based on dockerize by jwilder with the 
* Focused on only waiting (Stripped out template, log file tailing support
* Added support for looking for strings in HTTP response body to determine service readiness
* Added support for retry backoff
* M:ain process is EXEC-ed so that it takes over the PID of the await command. This is useful in Docker containers, to allow docker to handle signals.
* Updated to latest Golang and Alpine
* No Shell required
  * Small self-contained scratch image available (treksler/await:scratch) 
  * Useful for running go bianries compiled with CGO_ENABLED=0
* Binaries compessed with UPX 
  * treksler/await:scratch image is only 1.58MB
  * treksler/await:alpine is only 7.14MB

## Installation

Download the latest version in your container:

* [linux/amd64](https://github.com/treksler/await/releases/download/v1.0.0/await-linux-amd64-v1.0.0.tar.gz)
* [alpine/amd64](https://github.com/treksler/await/releases/download/v1.0.0/await-alpine-linux-amd64-v1.0.0.tar.gz)
* [darwin/amd64](https://github.com/treksler/await/releases/download/v1.0.0/await-darwin-amd64-v1.0.0.tar.gz)


### Docker Base Image

The `treksler/await` image is a base image based on `alpine linux`.  `await` is installed in the `$PATH` and can be used directly.

```
FROM treksler/await
...
ENTRYPOINT await ...
```

### Ubuntu Images

``` Dockerfile
RUN apt-get update && apt-get install -y wget

ENV AWAIT_VERSION v1.0.0
RUN wget https://github.com/treksler/await/releases/download/$AWAIT_VERSION/await-linux-amd64-$AWAIT_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf await-linux-amd64-$AWAIT_VERSION.tar.gz \
    && rm await-linux-amd64-$AWAIT_VERSION.tar.gz
```


### For Alpine Images:

``` Dockerfile
RUN apk add --no-cache openssl

ENV AWAIT_VERSION v1.0.0
RUN wget https://github.com/treksler/await/releases/download/$AWAIT_VERSION/await-alpine-linux-amd64-$AWAIT_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf await-alpine-linux-amd64-$AWAIT_VERSION.tar.gz \
    && rm await-alpine-linux-amd64-$AWAIT_VERSION.tar.gz
```

## Usage

await works by wrapping the call to your application using the `ENTRYPOINT` or `CMD` directives.

This would run `nginx` only after awaiting for the `web` host to respond on `tcp 8000`:

``` Dockerfile
CMD await -url tcp://web:8000 nginx
```

### Command-line Options

Http headers can be specified for http/https protocols.

```
$ await -url http://web:80 -http-header "Authorization:Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ=="
```

## Waiting for other dependencies

It is common when using tools like [Docker Compose](https://docs.docker.com/compose/) to depend on services in other linked containers, however oftentimes relying on [links](https://docs.docker.com/compose/compose-file/#links) is not enough - whilst the container itself may have _started_, the _service(s)_ within it may not yet be ready - resulting in shell script hacks to work around race conditions.

Wait gives you the ability to await for services on a specified protocol (`file`, `tcp`, `tcp4`, `tcp6`, `http`, `https` and `unix`) before starting your application:

```
$ await -url tcp://db:5432 -url http://web:80 -url file:///tmp/generated-file
```

### Timeout

You can optionally specify how long to await for the services to become available by using the `-timeout #` argument (Default: 10 seconds).  If the timeout is reached and the service is still not available, the process exits with status code 1.

```
$ await -url tcp://db:5432 -url http://web:80 -timeout 10s
```

See [this issue](https://github.com/docker/compose/issues/374#issuecomment-126312313) for a deeper discussion, and why support isn't and won't be available in the Docker ecosystem itself.

## License

MIT


[go.string.Split]: https://golang.org/pkg/strings/#Split
[go.string.Replace]: https://golang.org/pkg/strings/#Replace
[go.url.Parse]: https://golang.org/pkg/net/url/#Parse
[go.url.URL]: https://golang.org/pkg/net/url/#URL
