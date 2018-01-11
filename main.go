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

func main() {
	fmt.Printf("Start Jaeger by running: \n\n")
	fmt.Printf(`docker run -e COLLECTOR_ZIPKIN_HTTP_PORT=9411 -p5775:5775/udp -p6831:6831/udp -p6832:6832/udp \
	-p5778:5778 -p16686:16686 -p14268:14268 -p9411:9411 jaegertracing/all-in-one:latest`)
	fmt.Printf("\n\nThen visit 'localhost:16686' in your web-browser\n\n")

	time.Sleep(5 * time.Second)
	fmt.Printf("Making Breakfast!\n")

	tracer, err := InitTracer()
	opentracing.SetGlobalTracer(tracer)

	if err != nil {
		fmt.Printf("Tracer Init Error %s\n", err)
		return
	}
	for {
		if err := ServeBreakfast(); err != nil {
			fmt.Printf("Breakfast is ruined! %s\n", err)
		} else {
			fmt.Printf("Breakfast Success!\n")
		}
	}
}

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

	//Lets make some pancakes
	cakes := breakfast.MakePancakes(3)

	// If an error occurs, tag the span and log the error
	if err := FlipPancakes(ctx, cakes); err != nil {
		return err
	}
	ready := SyrupPancakes(ctx, cakes)
	EatPancakes(ready)
	return nil
}
func FlipPancakes(ctx context.Context, cakes []breakfast.Pancake) (err error) {
	// Create an EventInProgress - eip - named FlipPancakes
	eip := log.EventBegin(ctx, "FlipPancakes")
	defer func() {
		if err != nil {
			eip.SetError(err)
		}
		eip.Done()
	}()

	for p := range cakes {
		if err := cakes[p].Flip(); err != nil {
			return err
		}
	}

	// Let the pancakes cook
	time.Sleep(1 * time.Second)

	for p := range cakes {
		if cakes[p].IsBurnt() {
			return errors.New("Burnt Pancake")
		}
	}

	return nil
}
func SyrupPancakes(ctx context.Context, cakes []breakfast.Pancake) <-chan breakfast.Pancake {
	// Create a new ctx that holds a reference to a log event in progress
	ctx = log.EventBeginInContext(ctx, "PancakeReady")
	return syrupPancakes(ctx, cakes)
}

func syrupPancakes(ctx context.Context, cakes []breakfast.Pancake) <-chan breakfast.Pancake {
	// The channel perfectly syruped pancakes will be written to
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

func EatPancakes(<-chan breakfast.Pancake) {
	return
}

//Initalize a Jaeger tracer and set it as the global tracer in opentracing api
func InitTracer() (opentracing.Tracer, error) {
	tracerCfg := &config.Configuration{
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans: true,
		},
	}
	//we are ignoring the closer for now
	tracer, _, err := tracerCfg.New("Breakfast")
	if err != nil {
		return nil, err
	}
	return tracer, nil
}
