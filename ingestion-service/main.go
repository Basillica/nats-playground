package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/nats-io/nats.go"
)

// DataPoint represents a single sensor reading from a device.
type DataPoint struct {
	OrganizationID string  `json:"organization_id" validate:"required"`
	DeviceID       string  `json:"device_id" validate:"required"`
	SensorName     string  `json:"sensor_name" validate:"required"`
	Value          float64 `json:"value" validate:"required"`
	Timestamp      int64   `json:"timestamp" validate:"required"`
}

var (
	nc       *nats.Conn
	js       nats.JetStreamContext
	validate *validator.Validate
)

func main() {
	// 1. Connect to NATS
	var err error
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}

	nc, err = nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	js, err = nc.JetStream()
	if err != nil {
		log.Fatalf("Failed to get JetStream context: %v", err)
	}

	validate = validator.New()

	// 2. Set up HTTP server to receive data
	http.HandleFunc("/ingest", ingestHandler)
	log.Println("Ingestion Service listening on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// ingestHandler processes incoming data from devices.
func ingestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is supported", http.StatusMethodNotAllowed)
		return
	}

	var dp DataPoint
	if err := json.NewDecoder(r.Body).Decode(&dp); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Validate the incoming data
	if err := validate.Struct(dp); err != nil {
		http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
		return
	}

	// 3. Construct the NATS subject using the organization and device ID
	subject := fmt.Sprintf("data.%s.%s.%s", dp.OrganizationID, dp.DeviceID, dp.SensorName)

	payload, _ := json.Marshal(dp)

	// 4. Publish the message to JetStream with At-Least-Once delivery
	// The stream must be created in advance (by the Management Service)
	if _, err := js.Publish(subject, payload); err != nil {
		log.Printf("Failed to publish message to NATS: %v", err)
		http.Error(w, "Failed to ingest data", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully ingested data for subject: %s", subject)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Data ingested successfully"))
}
