package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/nats-io/nats.go"
)

// DataPoint represents a single sensor reading from a device.
type DataPoint struct {
	OrganizationID string  `json:"organization_id"`
	DeviceID       string  `json:"device_id"`
	SensorName     string  `json:"sensor_name"`
	Value          float64 `json:"value"`
	Timestamp      int64   `json:"timestamp"`
}

var (
	js           nats.JetStreamContext
	influxClient influxdb2.Client
)

const (
	streamName   = "ORG_orgA_STREAM"
	consumerName = "DATA_PERSISTENCE"
	influxOrg    = "your-influxdb-org"
	influxBucket = "iot-data-bucket"
)

func main() {
	// Connect to NATS
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	js, err = nc.JetStream()
	if err != nil {
		log.Fatalf("Failed to get JetStream context: %v", err)
	}

	// Connect to InfluxDB with a retry loop
	influxURL := os.Getenv("INFLUXDB_URL")
	influxToken := os.Getenv("INFLUXDB_TOKEN")

	// Retry loop for InfluxDB connection
	for i := 0; i < 10; i++ {
		influxClient = influxdb2.NewClient(influxURL, influxToken)
		_, err := influxClient.Health(context.Background())
		if err == nil {
			log.Println("Connected to InfluxDB after retries")
			break
		}

		log.Printf("Attempt %d failed to connect to InfluxDB: %v. Retrying...", i+1, err)
		time.Sleep(5 * time.Second)
	}

	if influxClient == nil {
		log.Fatal("Failed to connect to InfluxDB after multiple retries. Exiting.")
	}
	defer influxClient.Close()

	// ... (Rest of the main function remains the same)
	// Create a durable pull consumer for this stream
	js.AddStream(&nats.StreamConfig{
		Name:     streamName,
		Subjects: []string{"data.>"},
		Storage:  nats.FileStorage,
	})

	sub, err := js.PullSubscribe("", consumerName, nats.BindStream(streamName))
	if err != nil {
		log.Fatalf("Failed to subscribe to stream: %v", err)
	}

	log.Printf("Data Persistence Service started, listening for all data...")

	for {
		// Fetch new messages from the stream
		msgs, err := sub.Fetch(100, nats.MaxWait(1*time.Second))
		if err != nil {
			if err == nats.ErrTimeout {
				continue
			}
			log.Printf("Error fetching messages: %v", err)
			continue
		}

		for _, msg := range msgs {
			processMessage(msg)
		}
	}
}

func processMessage(msg *nats.Msg) {
	// ... (Rest of the processMessage function remains the same)
	var dp DataPoint
	if err := json.Unmarshal(msg.Data, &dp); err != nil {
		log.Printf("Failed to unmarshal data: %v", err)
		msg.Nak()
		return
	}

	writeAPI := influxClient.WriteAPI(influxOrg, influxBucket)

	point := influxdb2.NewPointWithMeasurement(dp.SensorName).
		AddTag("organization_id", dp.OrganizationID).
		AddTag("device_id", dp.DeviceID).
		AddTag("sensor_name", dp.SensorName).
		AddField("value", dp.Value).
		SetTime(time.Unix(dp.Timestamp, 0))

	writeAPI.WritePoint(point)

	log.Printf("Persisted data point from subject: %s", msg.Subject)

	msg.Ack()
}
