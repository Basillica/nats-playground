package main

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/nats-io/nats.go"
)

// NotificationTrigger represents a condition that will trigger a notification.
type NotificationTrigger struct {
	OrganizationID string `json:"organization_id"`
	DeviceID       string `json:"device_id"`
	SensorName     string `json:"sensor_name"`
	Condition      string `json:"condition"`
}

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

	// Use a standard NATS subscription with a queue group for load balancing.
	// This ensures that if you have multiple instances of the Notification Service,
	// only one of them processes each message.
	_, err = nc.QueueSubscribe("notifications.>", "notification-group", func(msg *nats.Msg) {
		processNotification(msg)
	})
	if err != nil {
		log.Fatalf("Failed to subscribe to notifications: %v", err)
	}

	log.Println("Notification Service is listening for alerts...")

	// Keep the service running
	select {}
}

func processNotification(msg *nats.Msg) {
	var trigger NotificationTrigger
	if err := json.Unmarshal(msg.Data, &trigger); err != nil {
		log.Printf("Failed to unmarshal notification payload: %v", err)
		return
	}

	log.Printf("🔔 ALERT: Condition met for Organization: %s, Device: %s, Sensor: %s",
		trigger.OrganizationID, trigger.DeviceID, trigger.SensorName)
	log.Printf("Condition: %s", trigger.Condition)

	// In a real-world scenario, you would dispatch the notification here:
	// - Send an email (using a library like 'sendgrid' or 'gomail')
	// - Send an SMS (using a service like Twilio)
	// - Push to a webhook (e.g., Slack or a custom API)
	sendFakeEmail(trigger)
}

func sendFakeEmail(trigger NotificationTrigger) {
	log.Printf("📧 Sending fake email to organization %s...", trigger.OrganizationID)
	// Simulate the email sending process
	time.Sleep(1 * time.Second)
	log.Printf("Email sent successfully!")
}
