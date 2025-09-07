package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
)

// Organization represents an IoT customer's organization.
type Organization struct {
	ID   string `json:"id" validate:"required"`
	Name string `json:"name" validate:"required"`
}

// DeviceRegistration represents the payload for registering a new device.
type DeviceRegistration struct {
	OrganizationID string `json:"organization_id" validate:"required,uuid"`
	Name           string `json:"name" validate:"required"`
}

var (
	js         nats.JetStreamContext
	validate   *validator.Validate
	signingKey nkeys.KeyPair
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

	kp, err := nkeys.CreateUser()
	if err != nil {
		log.Fatal(err)
	}
	seed, _ := kp.Seed()
	log.Printf("Your private key seed is: %s", string(seed))

	validate = validator.New()

	// Setup HTTP router
	router := mux.NewRouter()
	router.HandleFunc("/organizations", createOrganizationHandler).Methods("POST")
	router.HandleFunc("/devices", registerDeviceHandler).Methods("POST")
	http.Handle("/", router)

	log.Println("Management Service listening on :8081...")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func createOrganizationHandler(w http.ResponseWriter, r *http.Request) {
	var org Organization
	if err := json.NewDecoder(r.Body).Decode(&org); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if err := validate.Struct(org); err != nil {
		http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
		return
	}

	// 1. Create a unique stream name and subjects for the organization.
	streamName := fmt.Sprintf("ORG_%s_STREAM", org.ID)
	streamSubjects := []string{fmt.Sprintf("data.%s.>", org.ID)}

	// 2. Provision the JetStream stream.
	log.Printf("Attempting to provision JetStream stream '%s'...", streamName)
	_, err := js.AddStream(&nats.StreamConfig{
		Name:     streamName,
		Subjects: streamSubjects,
		Storage:  nats.FileStorage,
	})

	if err != nil {
		log.Printf("Failed to create stream for organization '%s': %v", org.ID, err)
		http.Error(w, "Failed to create organization stream", http.StatusInternalServerError)
		return
	}

	// In a real-world scenario, you would also save the organization details to a database here.
	log.Printf("Successfully created organization '%s' and its NATS stream.", org.ID)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fmt.Sprintf("Organization '%s' created successfully.", org.ID)))
}

func registerDeviceHandler(w http.ResponseWriter, r *http.Request) {
	var reg DeviceRegistration
	if err := json.NewDecoder(r.Body).Decode(&reg); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if err := validate.Struct(reg); err != nil {
		http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
		return
	}

	deviceID := uuid.New().String()

	// Create a JWT for the device with specific publish permissions
	claims := jwt.NewUserClaims(deviceID)
	claims.Permissions.Pub.Allow.Add(fmt.Sprintf("data.%s.%s.>", reg.OrganizationID, deviceID))

	token, err := claims.Encode(signingKey)
	if err != nil {
		log.Printf("Failed to create JWT for device %s: %v", deviceID, err)
		http.Error(w, "Failed to register device", http.StatusInternalServerError)
		return
	}

	// In a real application, you would save device metadata to a database here
	log.Printf("Successfully registered device '%s' for organization '%s'", deviceID, reg.OrganizationID)

	// Return the device ID and the JWT token to the client
	response := map[string]string{
		"device_id": deviceID,
		"jwt_token": token,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}
