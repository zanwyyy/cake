# 📌 Project Overview

This project is a **Transaction Service** built in Go, following a clean architecture style with clear separation of concerns.  
It provides APIs (HTTP/gRPC) to handle money transfers between users, stores data in PostgreSQL, and publishes events to Kafka (or Google Pub/Sub).

---

## 🗂 Project Structure

### 1. `cmd/`
Contains application entrypoints (main commands).
- `consumer.go` → Define kafka consumer and command. 
- `grpc_server.go` → Define the gRPC server.  
- `http_server.go` → Define the HTTP server for REST APIs.  
- `root.go` → Root command/configuration (e.g., CLI setup).  
- `server.go` → Define server command. 


---

### 2. `config/`
- `config.go` → Loads environment variables and configurations (database, Kafka broker, Pub/Sub, etc.).  


---

### 3. `internal/`
Core application logic and domain code.  
- `api/transfer.go` → HTTP/gRPC handlers for transfer APIs.  
- `model/transaction.go` → Domain models (`Transaction`, `User`, etc.).  
- `repo/`  
  - `kafka.go` → Kafka repository .  
  - `pubsub.go` → Google Pub/Sub repository .  
  - `transfer.go` → PostgreSQL repository (persist transactions).  
  - `transfer_test.go` → Unit tests for the transfer repository.  
- `service/transfer.go` → Business logic: validate balance, execute transfers, and publish events.  


---


---

### 5. Root files
- `main.go` → Application entrypoint (often calls `cmd/server.go`).  
- `init.sql` → Database initialization script (creates `users` and `transactions` tables).  
- `docker-compose.yml` → Local development setup (Postgres + Kafka + App).  
- `demo-app.dockerfile` → Dockerfile for building the app image.  
- `k8s/` → Kubernetes manifests (Deployment, Service, Job for DB/Kafka init).  
- `README.md` → Project documentation.  

---

## 🔄 Workflow

1. A client sends a **transfer request** via HTTP/gRPC.  
2. The **service layer** validates the request and checks user balances in PostgreSQL.  
3. If valid → inserts the transaction into DB.  
4. The service then **publishes an event to Kafka** (`transactions` topic).  
5. A **Kafka consumer** subscribes to that topic, receives the event, and **forwards it to Google Pub/Sub**.  
6. A **Pub/Sub consumer** subscribes to the Pub/Sub topic, processes the message, and sends an **ack** to confirm successful handling.  



# 🚀 Getting Started

## 1. Prerequisites

Make sure you have these installed on your system:

* [Go 1.21+](https://go.dev/dl/)
* [Docker](https://docs.docker.com/get-docker/)
* [Docker Compose](https://docs.docker.com/compose/install/)
* [kubectl](https://kubernetes.io/docs/tasks/tools/)
* [Minikube](https://minikube.sigs.k8s.io/docs/start/) (if running on K8s)
* [Pub/Sub Emulator](https://cloud.google.com/pubsub/docs/emulator) (for local Pub/Sub testing)

---

## 2. Run with Docker Compose (Local Dev)

This setup runs:

* PostgreSQL
* Kafka + Zookeeper
* Google Pub/Sub emulator
* The Transaction Service

```bash
# Clone repo
git clone https://github.com/your-org/demo-app.git
cd demo-app

# Start services
docker-compose up --build
```

After startup:

* HTTP server → `http://localhost:8080`
* gRPC server → `localhost:9090`
* Kafka → `localhost:9092`
* Pub/Sub emulator → `localhost:8085`

---

## 3. Run with Minikube (Kubernetes)

If you prefer to run on K8s:

```bash
# Start Minikube
minikube start

# Build image inside Minikube
eval $(minikube docker-env)
docker build -t demo-app -f demo-app.dockerfile .

# Apply manifests
kubectl apply -f k8s/

# Get exposed service URL
minikube service app-server --url
```

📌 Reference: [Minikube Docs](https://minikube.sigs.k8s.io/docs/start/)

---

## 4. Pub/Sub Emulator (Local Setup)

Start the emulator manually if not running via Docker:

```bash
 gcloud beta emulators pubsub env-init
 export PUBSUB_EMULATOR_HOST=localhost:8085
export PUBSUB_PROJECT_ID=demo-project
gcloud beta emulators pubsub start --project=demo-project    

// clone the python repo in ref, cd to samples, snippests
pip install -r requirements.txt
 python3 publisher.py demo-project create transactions (create topic)
 python3 subscriber.py demo-project create transactions sub-transactions (create pull sub)
```


📌 Reference: [Pub/Sub Emulator Docs](https://cloud.google.com/pubsub/docs/emulator)

---

## 5. API Endpoints

### 1️⃣ Get Transactions of a User

Fetch all transactions for a specific user by ID.

**Request:**

```bash
curl --location 'http://127.0.0.1:<PORT>/v1/transfer/1/transactions'
```

**Response:**

```json
{
    "number": "2",
  {
    "id": 1,
    "from_user": "1",
    "to_user": "2",
    "amount": 200
  },
  {
    "id": 2,
    "from_user": "1",
    "to_user": "3",
    "amount": 100
  }
}
```

---

### 2️⃣ Send Money (Transfer)

Create a new transaction between two users.

**Request:**

```bash
curl --location 'http://127.0.0.1:<PORT>/v1/transfer/send' \
--header 'Content-Type: application/json' \
--data '{
  "from": "1",
  "to": "2",
  "amount": 36
}'
```

**Response (Success):**

```json
{
    "success": true,
    "errorMessage": ""
}
```

**Response (Insufficient Balance):**

```json
{
    "code": 2,
    "message": "insufficient balance",
    "details": []
}
```

---


✅ Now you have the project running locally with **Postgres + Kafka + Pub/Sub integration**.
