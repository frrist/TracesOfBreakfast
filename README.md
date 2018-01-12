# Traces Of Breakfast

This project was built to demo how the `opentracing-go` and `go-log` packages can be used to instrament tracing on some code.

## Install

```shell
$ go get -u "github.com/frrist/TracesOfBreakfast"
```

## Usage

Build the package.

```shell
$ go build 
```

Run Jaeger.

```shell
$ docker run -e COLLECTOR_ZIPKIN_HTTP_PORT=9411 -p5775:5775/udp -p6831:6831/udp -p6832:6832/udp -p5778:5778 -p16686:16686 -p14268:14268 -p9411:9411 jaegertracing/all-in-one:latest
```

View Jaeger UI by navigating to `localhost:16686` in your browser. 

# Issues building?

If Jaeger is giving you a lot of errors try this:

Navigate to the `jaeger-client-go` package

```shell
$ cd $GOPATH/src/github.com/uber/jaeger-client-go/
```

Update its submodules

```shell
$ git submodule update --init --recursive
```

Build the package

```shell
$ make install
```

Remove its vendored opentrace version

```shell
$ rm -rf vendor/github.com/opentracing/
```

Try building `TracesOfBreakfast` again.
