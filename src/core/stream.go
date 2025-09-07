package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func main() {

	url := os.Getenv("NATS_URL")
	if url == "" {
		url = nats.DefaultURL
	}

	nc, _ := nats.Connect(url)

	defer nc.Drain()

	js, _ := jetstream.New(nc)

	cfg := jetstream.StreamConfig{
		Name:     "EVENTS",
		Subjects: []string{"events.>"},
	}

	cfg.Storage = jetstream.FileStorage

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, _ := js.CreateStream(ctx, cfg)
	fmt.Println("created the stream")

	js.Publish(ctx, "events.page_loaded", nil)
	js.Publish(ctx, "events.mouse_clicked", nil)
	js.Publish(ctx, "events.mouse_clicked", nil)
	js.Publish(ctx, "events.page_loaded", nil)
	js.Publish(ctx, "events.mouse_clicked", nil)
	js.Publish(ctx, "events.input_focused", nil)
	fmt.Println(ctx, "published 6 messages")

	js.PublishAsync("events.input_changed", nil)
	js.PublishAsync("events.input_blurred", nil)
	js.PublishAsync("events.key_pressed", nil)
	js.PublishAsync("events.input_focused", nil)
	js.PublishAsync("events.input_changed", nil)
	js.PublishAsync("events.input_blurred", nil)

	select {
	case <-js.PublishAsyncComplete():
		fmt.Println("published 6 messages")
	case <-time.After(time.Second):
		log.Fatal("publish took too long")
	}

	printStreamState(ctx, stream)

	cfg.MaxMsgs = 10
	js.UpdateStream(ctx, cfg)
	fmt.Println("set max messages to 10")

	printStreamState(ctx, stream)

	cfg.MaxBytes = 300
	js.UpdateStream(ctx, cfg)
	fmt.Println("set max bytes to 300")

	printStreamState(ctx, stream)

	cfg.MaxAge = time.Second
	js.UpdateStream(ctx, cfg)
	fmt.Println("set max age to one second")

	printStreamState(ctx, stream)

	fmt.Println("sleeping one second...")
	time.Sleep(time.Second)

	printStreamState(ctx, stream)
}

func printStreamState(ctx context.Context, stream jetstream.Stream) {
	info, _ := stream.Info(ctx)
	b, _ := json.MarshalIndent(info.State, "", " ")
	fmt.Println("inspecting stream info")
	fmt.Println(string(b))
}
