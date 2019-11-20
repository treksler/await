await ![version v1.0.0](https://img.shields.io/badge/version-v1.0.0-brightgreen.svg) ![License MIT](https://img.shields.io/badge/license-MIT-blue.svg)
=============

Utility to wait for dependencies before launching a command.

It is common when using tools like [Docker Compose](https://docs.docker.com/compose/) to depend on services in other linked containers, however oftentimes relying on [links](https://docs.docker.com/compose/compose-file/#links) is not enough - whilst the container itself may have _started_, the _service(s)_ within it may not yet be ready - resulting in shell script hacks to work around race conditions.

await gives you the ability to wait for services on a specified protocol (`file`, `tcp`, `tcp4`, `tcp6`, `http`, `https` and `unix`) before starting your application:

For example, it can delay the starting of a python application until the database is running and listening on the TCP port.

Based on dockerize by jwilder:
* Focused on only waiting (Stripped out template, log file tailing support)
* Added support for present/absent strings in HTTP response body to determine service readiness
* Added support for retry backoff
* Main process is EXEC-ed. 
  * It takes over the PID of the await command. 
  * Useful in Docker containers to maintain PID=1.
* Updated to latest Golang and Alpine
* No Shell required
  * Small self-contained scratch image available (treksler/await:scratch) 
  * Useful for running go bianries compiled with CGO_ENABLED=0
* Binaries compessed with UPX 
  * treksler/await:scratch image is only 1.58MB
  * treksler/await:alpine is only 7.14MB

See [this issue](https://github.com/docker/compose/issues/374#issuecomment-126312313) for a deeper discussion, and why support isn't and won't be available in the Docker ecosystem itself.

## Docker Base Images

### Alpine Base Image

The `treksler/await` image is a base image based on `alpine linux`.  `await` is installed in the `$PATH` and can be used directly.

```
FROM treksler/await
...
COPY app /app
...
ENTRYPOINT await 
CMD -url ... /app
```

### Scratch Base Image

The `treksler/await:scratch` image is a base image based on `scratch`.  `await` is installed in the `$PATH` and can be used directly.

```
FROM treksler/await:scratch
...
COPY app /app
...
ENTRYPOINT await 
CMD -url ... /app
```

## Installation

Download the latest version in your container:

* [linux/amd64](https://github.com/treksler/await/releases/download/v1.0.0/await-linux-amd64-v1.0.0.tar.gz)
* [alpine/amd64](https://github.com/treksler/await/releases/download/v1.0.0/await-alpine-linux-amd64-v1.0.0.tar.gz)
* [darwin/amd64](https://github.com/treksler/await/releases/download/v1.0.0/await-darwin-amd64-v1.0.0.tar.gz)

### Install in Ubuntu Images

``` Dockerfile
FROM ubuntu
RUN apt-get update && apt-get install -y wget

ENV AWAIT_VERSION v1.0.0
RUN wget https://github.com/treksler/await/releases/download/$AWAIT_VERSION/await-linux-amd64-$AWAIT_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf await-linux-amd64-$AWAIT_VERSION.tar.gz \
    && rm await-linux-amd64-$AWAIT_VERSION.tar.gz
```

### Install in Alpine Images

``` Dockerfile
FROM alpine
RUN apk add --no-cache openssl

ENV AWAIT_VERSION v1.0.0
RUN wget https://github.com/treksler/await/releases/download/$AWAIT_VERSION/await-alpine-linux-amd64-$AWAIT_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf await-alpine-linux-amd64-$AWAIT_VERSION.tar.gz \
    && rm await-alpine-linux-amd64-$AWAIT_VERSION.tar.gz
```

## Usage

await works by wrapping the call to your application using the `ENTRYPOINT` or `CMD` directives.

This would run `nginx` only after awaiting a response from `web` host on `tcp 8000`:

``` Dockerfile
CMD await -url tcp://web:8000 nginx
```

### Command-line Options

#### -http-header value
  
HTTP headers, colon separated. e.g "Accept-Encoding: gzip". Can be passed multiple times

```
$ await -url http://web:80 -http-header "Authorization:Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ=="
```

#### -retry-backoff

Double the retry time, with each iteration. (default: false)

#### -retry-backoff-max-interval duration

Maximum duration to wait before retrying, when retry backoff is enabled. (default 5m0s)

#### -retry-interval duration

Duration to wait before retrying (default 1s)

#### -text-absent value

Text required to be absent from HTTP response body. Can be passed multiple times. The command will not run until none of the forbidden text is present in all http responses

```
$ await -url tcp://db:5432 -url http://web:80 -text-present "not ready"
```

#### -text-present value

Text required to be present in HTTP response body. Can be passed multiple times. The command will not run until the required text is present in all http responses

```
$ await -url tcp://db:5432 -url http://web:80 -text-present "is ready"
```

#### -timeout duration

You can optionally specify how long to await for the services to become available by using the `-timeout #` argument (Default: 10 seconds).  If the timeout is reached and the service is still not available, the process exits with status code 1.

```
$ await -url tcp://db:5432 -url http://web:80 -timeout 10s
```
#### -url value

Host (tcp/tcp4/tcp6/http/https/unix/file) to await before this container starts. Can be passed multiple times. e.g. tcp://db:5432

#### -version
show version


### Examples

Wait for a database to become available on port 5432 and start nginx.
```
$ await -url tcp://db:5432 nginx
```

Wait up to 90s for a website to become available on port 38383. Look for text "ready" and make sure text "fail" is not present in response body. Retry after 5,10,20,40,80,80,etc. seconds, if needed. Start nginx when all conditions are met.
```
$ await --url http://localhost:38383 --text-present "ready" --text-absent "fail" --timeout 300s --retry-interval 5s --retry-backoff --retry-backoff-max-interval 80s nginx
```

Wait for a file to be generated, before starting nginx
```
$ await -url tcp://db:5432 -url http://web:80 -url file:///tmp/generated-file nginx
```


## License

MIT

[go.url.Parse]: https://golang.org/pkg/net/url/#Parse
[go.url.URL]: https://golang.org/pkg/net/url/#URL
