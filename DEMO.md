# Adding Tracing to your code with go-log

[go-log](https://github.com/ipfs/go-log) is the logging library used by go-ipfs, it currently uses a modified version of [go-logging](https://github.com/whyrusleeping/go-logging) to implement the standard printf-style log output. In addition to standard logging output, go-log has the capability to produce [event logs](https://en.wikipedia.org/wiki/Log_file#Event_logs), which in turn can be used to generate tracing data via the [OpenTracing API](https://github.com/opentracing/opentracing-go). 

Here we will show how to instrument some example code with the `go-log`, package such that it can produce useful tracing data. If you are unfamilair with OpenTracing check out [these notes](https://github.com/ipfs/notes/issues/277), otherwise lets get started!

```go
package main

import (
        "context"
        "errors"
        "fmt"
        "time"

        logging "github.com/ipfs/go-log"
        opentracing "github.com/opentracing/opentracing-go"
        config "github.com/uber/jaeger-client-go/config"

        breakfast "github.com/frrist/breakfast"
)


var log = logging.Logger("breakfast")
```

In the above code snippet we create a logger called `log` with a service name `breakfast`. We have also import the OpenTracing go client package, and a tracing system packaged called [Jaeger](https://github.com/jaegertracing/jaeger).

Next, lets setup our `ServeBreakfast` method.

```go
func ServeBreakfast() error {
        //Context used for the request
        ctx, cancel := context.WithCancel(context.Background())
        defer cancel()

        // Create a span called rootSpan.
        // This span will be the parent of all other spans created
        // during the exection of methods called inside ServeBreakfast
        rootSpan := opentracing.StartSpan("ServeHotCakes")
        defer rootSpan.Finish()

        // Create a new ctx that holds a reference to rootSpan's SpanContext
        ctx = opentracing.ContextWithSpan(ctx, rootSpan)

        //Make some pancakes to process
        cakes := breakfast.MakePancakes(3)

        if err := FlipPancakes(ctx, cakes); err != nil {
                return err
        }
        ready := SyrupPancakes(ctx, cakes)
        EatPancakes(ready)
        return nil
}
```

In the above snippet we are making calls to 3 different methods, `FlipPancakes` which may return an error, `SyrupPancakes` which returns a channel, and `EatPancakes` which consumes a channel. This is the only place that we will interact with OpenTracing directly, the rest is handled by `go-log`.

Now let's take a look at how `FlipPancakes` logs events.

```go
func FlipPancakes(ctx context.Context, cakes []breakfast.Pancake) (err error) {
        // Create an EventInProgress - eip - named FlipPancakes
        eip := log.EventBegin(ctx, "FlipPancakes")
        defer func() {
                if err != nil {
                        eip.SetError(err)
                }
                eip.Done()
        }()
		
  		// Flip all the pancakes
        for p := range cakes {
                if err := cakes[p].Flip(); err != nil {
                        return err
                }
        }

        // Let the pancakes cook
        time.Sleep(1 * time.Second)

  		// Make sure they aren't burnt
        for p := range cakes {
                if cakes[p].IsBurnt() {
                        return errors.New("Burnt Pancake")
                }
        }

        return nil
}
```

Two things to notice here, we have a named return - `err` - and we have deferred completing the `eip` until `FlipPancakes` returns. This allows us to call `eip.SetError(err)` just once, instead of everywhere an error is returned - this prevents the code from getting cluttered.

But what if we have a function that spins off a go-routine and writes the result to a channel instead of plainly returning, such as `SyrupPancakes`?

```go
func SyrupPancakes(ctx context.Context, cakes []breakfast.Pancake) <-chan breakfast.Pancake {
        // Create a new ctx that holds a reference to a log event in progress
        ctx = log.EventBeginInContext(ctx, "SyrupPancakes")
  
        // The channel perfectly syruped pancakes will be added to
        out := make(chan breakfast.Pancake)
        go func() {
                // If there is an event in the context, defer compltion of it
                // until we have handled all pancakes.
                defer logging.MaybeFinishEvent(ctx)
                defer close(out)

                // Where soggy pancakes go..
                var mistakes []breakfast.Pancake
                for p := range cakes {
                        if err := cakes[p].Syrup(); err != nil {
                                fmt.Errorf("Ohh no, soggy pancakes!")
                                mistakes = append(mistakes, cakes[p])
                                continue
                        }
                        select {
                        // Send off our perfect pancakes
                        case out <- cakes[p]:
                        case <-ctx.Done():
                                return
                        }
                        if len(mistakes) == 0 {
                                return
                        } else {
                                // fix your pancakes...
                        }
                }
        }()
        return out
}
```

What we have done here is nearly identical to what was done in `ServerBreakfast` with the `opentracing.ContextWithSpan()` call.  But instead, we associate a *LogEvent* with a context.

When the go-routine completes, `MaybeFinishEvent` will complete the LogEvent we associated with the context earlier. This is a good way to add event logging when the event spans multiple methods.
