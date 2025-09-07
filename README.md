# Scalable IoT Platform with Go and NATS

This project is a multi-service, scalable IoT platform built with Go, leveraging NATS JetStream for a robust messaging backbone. The platform is designed to handle high-volume data ingestion, real-time analysis, notifications, and long-term storage, with a focus on organization-level scalability and independent service management.

[Architectural diagram](architecture.jpg)

## Architecture Overview

The platform consists of several decoupled microservices orchestrated with Docker Compose:

* NATS Server: The central nervous system for all messaging, using JetStream for durable streams.

* Management Service: A REST API for creating organizations and registering devices. It provisions NATS resources and issues secure JWT credentials.

* Ingestion Service: A lightweight gateway that receives data from devices and publishes it to the appropriate NATS subject.

* Analysis Engine: Subscribes to NATS streams, performs real-time analysis (e.g., threshold checks), and triggers alerts.

* Notification Service: Listens for alerts from the Analysis Engine and dispatches notifications (e.g., email).

* Data Persistence Service: Consumes all incoming data and stores it in a time-series database (InfluxDB) for historical analysis.

* Databases:
    * PostgreSQL: For structured metadata (organizations, devices, users).
    * Redis: For transient state management (e.g., consecutive breach counts).
    * InfluxDB: For high-volume, time-series data storage.

* Observability Stack:
    * Prometheus: For collecting system and service metrics.
    * Grafana: For visualizing metrics and creating dashboards.
    * Loki & Promtail: For centralized log aggregation and querying.

## Getting Started

### Prerequisites
* Docker and Docker Compose installed.
* A recent version of Go (for running individual services outside of Docker, if desired).

### Setup
1. Clone the repository.
2. Navigate to the project root directory.
3. Run the following command to start all services:

```bash
docker-compose up --build
```

This command will build the Docker images for all Go services and launch the entire platform, including the databases and the observability stack.

## API Endpoints & Testing
All services are running in Docker, so you'll interact with them via localhost and their exposed ports.

1. Management Service (http://localhost:8081)
This service is used to bootstrap the platform by creating a new organization and registering devices.

    * Create an Organization
    This endpoint provisions a dedicated NATS JetStream stream for the organization.
        * Method: POST
        * Endpoint: /organizations
        * Example Request:

            ```bash
            curl -X POST http://localhost:8081/organizations -H "Content-Type: application/json" -d '{"id": "orgA", "name": "A Corp"}'
            ```

    * Register a Device
    This endpoint creates a new device UUID and issues a secure JWT token for it.

        * Method: POST
        * Endpoint: /devices
        * Example Request:

        ```bash
        curl -X POST http://localhost:8081/devices -H "Content-Type: application/json" -d '{"organization_id": "orgA", "name": "Temp Sensor 1"}'
        ```

2. Ingestion Service (http://localhost:8080)
This service is the entry point for device data. It publishes incoming payloads to the correct NATS JetStream subject.

    * Ingest a Data Point
Simulate a device sending a data payload. The payload must include a valid organization_id that was created via the Management Service.

        * Method: POST
        * Endpoint: /ingest
        * Example Request:

        ```bash
        curl -X POST http://localhost:8080/ingest -H "Content-Type: application/json" -d '{
            "organization_id": "orgA",
            "device_id": "machine-123",
            "sensor_name": "temp1",
            "value": 85.0,
            "timestamp": 1672531200
        }'
        ```

## End-to-End Testing
To test the entire data pipeline, follow these steps:

1. Start all services with `docker-compose up --build`.
2. Create an organization using the Management Service endpoint:
```bash
curl -X POST http://localhost:8081/organizations -H "Content-Type: application/json" -d '{"id": "orgA", "name": "A Corp"}'
```
3. Simulate consecutive data points from a device with a temperature value above the hardcoded threshold (80.0) to trigger an alert.
Run the `curl` command for data ingestion three times to meet the Analysis Engine's condition for a persistent breach.
```bash
curl -X POST http://localhost:8080/ingest -H "Content-Type: application/json" -d '{"organization_id": "orgA", "device_id": "machine-123", "sensor_name": "temp1", "value": 90.0, "timestamp": 1}'
```

4. Observe the logs from the running Docker containers. You will see:

* `ingestion-service`: Logs confirming data publication.
* `analysis-engine`: Logs showing a threshold breach and, on the third message, the alert being published to the notifications subject.
* `notification-service`: Logs confirming the alert was received and a "fake email" was sent.
* `data-persistence-service`: Logs confirming the data point was successfully written to InfluxDB.

## Observability
You can monitor the health and performance of the system using the provided dashboards.

* NATS Monitoring: `http://localhost:8222`
* Prometheus UI: `http://localhost:9090`
* Grafana Dashboards: `http://localhost:3000` (Login with `admin/password`). You can connect to the Prometheus and Loki data sources to build custom dashboards.