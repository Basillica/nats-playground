package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
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

// NotificationTrigger represents a condition that will trigger a notification.
type NotificationTrigger struct {
	OrganizationID string `json:"organization_id"`
	DeviceID       string `json:"device_id"`
	SensorName     string `json:"sensor_name"`
	Condition      string `json:"condition"`
}

var (
	js  nats.JetStreamContext
	rdb *redis.Client
	ctx = context.Background()
)

const (
	streamName   = "ORG_orgA_STREAM" // Example stream to consume from
	consumerName = "ANALYSIS_ENGINE"
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

	// Connect to Redis
	rdb = redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer rdb.Close()
	log.Println("Connected to Redis")

	// Create a durable pull consumer for the stream
	sub, err := js.PullSubscribe("", consumerName, nats.BindStream(streamName))
	if err != nil {
		log.Fatalf("Failed to subscribe to stream: %v", err)
	}

	log.Printf("Analysis Engine started, listening for messages on stream '%s'...", streamName)

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
	var dp DataPoint
	if err := json.Unmarshal(msg.Data, &dp); err != nil {
		log.Printf("Failed to unmarshal data: %v", err)
		msg.Nak() // Negative acknowledgment (re-queue)
		return
	}

	// Check for temperature threshold breach
	// This threshold would ideally be fetched from the database
	if dp.SensorName == "temp1" && dp.Value > 80.0 {
		log.Printf("Threshold breach detected! Device: %s, Value: %.2f", dp.DeviceID, dp.Value)

		// Key for tracking consecutive breaches
		redisKey := fmt.Sprintf("breach:count:%s:%s", dp.DeviceID, dp.SensorName)

		// Increment the breach count in Redis
		breachCount, err := rdb.Incr(ctx, redisKey).Result()
		if err != nil {
			log.Printf("Failed to increment Redis key: %v", err)
		}

		// Set a TTL (e.g., 5 minutes) to reset the count if the condition subsides
		rdb.Expire(ctx, redisKey, 5*time.Minute)

		// If threshold exceeded for 3 consecutive points, publish a notification
		if breachCount >= 3 {
			triggerPayload := NotificationTrigger{
				OrganizationID: dp.OrganizationID,
				DeviceID:       dp.DeviceID,
				SensorName:     dp.SensorName,
				Condition:      "Consecutive Temp Threshold Breach",
			}
			payload, _ := json.Marshal(triggerPayload)

			// Publish to the notification subject
			notificationSubject := fmt.Sprintf("notifications.%s.%s", dp.OrganizationID, dp.DeviceID)
			if _, err := js.Publish(notificationSubject, payload); err != nil {
				log.Printf("Failed to publish notification: %v", err)
			} else {
				log.Printf("Notification triggered for subject: %s", notificationSubject)
				// Reset the breach count in Redis after triggering the notification
				rdb.Del(ctx, redisKey)
			}
		}
	} else if dp.SensorName == "temp1" && dp.Value <= 80.0 {
		// If the condition is no longer met, clear the counter
		redisKey := fmt.Sprintf("breach:count:%s:%s", dp.DeviceID, dp.SensorName)
		rdb.Del(ctx, redisKey)
	}

	// Acknowledge the message to JetStream, indicating it was processed successfully
	msg.Ack()
}
