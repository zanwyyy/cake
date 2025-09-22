# 📌 Project Overview

This project is a **Transaction Service** built in Go, following a clean architecture style with clear separation of concerns.  
It provides HTTP APIs that are converted to gRPC calls using **gRPC-Gateway** to handle money transfers between users with JWT authentication, stores data in PostgreSQL, and publishes events to Google Pub/Sub.

---

## 🗂 Project Structure

### 1. `cmd/`
Contains application entrypoints (main commands).
- `consumer.go` → Define Pub/Sub consumer and command. 
- `grpc_server.go` → Define the gRPC server (internal service communication).  
- `http_server.go` → Define the HTTP server with gRPC-Gateway (user-facing APIs).  
- `root.go` → Root command/configuration (e.g., CLI setup).  
- `server.go` → Define server command that runs both HTTP gateway and gRPC server. 

---

### 2. `config/`
- `config.go` → Loads environment variables and configurations (database, Pub/Sub, JWT secret, etc.).  

---

### 3. `internal/`
Core application logic and domain code.  
- `api/`
  - `auth.go` → gRPC handlers for login/logout services.
  - `transfer.go` → gRPC handlers for transfer services.  
- `model/`
  - `transaction.go` → Domain models (`Transaction`, etc.).
  - `user.go` → User domain models and authentication structures.
- `repo/`  
  - `pubsub.go` → Google Pub/Sub repository for event publishing.
  - `redis.go` → Redis repository for caching and session management.
  - `transfer.go` → PostgreSQL repository (persist transactions).  
  - `transfer_test.go` → Unit tests for the transfer repository.  
- `service/`
  - `auth.go` → Authentication service: login validation, JWT generation/validation.
  - `transfer.go` → Business logic: validate balance, execute transfers, and publish events.
- `utils/`
  - `jwt.go` → JWT utility functions for token generation and validation.
  - `snowflake.go` → Snowflake ID generator utility.  

### 4. `pkg/`
Shared packages and protocol buffers.
- `interceptor/`
  - `auth.go` → gRPC authentication interceptor.
- `pb/`
  - `transfer_grpc.pb.go` → Generated gRPC server code.
  - `transfer.pb.go` → Generated protobuf message structures.
  - `transfer.pb.gw.go` → Generated gRPC gateway code.
- `probuf/`
  - `transfer.proto` → Protocol buffer definitions.

---

### 5. `k8s/`
Kubernetes deployment manifests.
- `consumer_pubsub.yaml` → Pub/Sub consumer deployment.
- `db.yaml` → PostgreSQL database deployment.
- `redis.yaml` → Redis deployment.
- `server.yaml` → Main application server deployment.

---

### 6. Root files
- `main.go` → Application entrypoint (often calls `cmd/server.go`).  
- `init.sql` → Database initialization script (creates `users` and `transactions` tables).  
- `docker-compose.yml` → Local development setup (Postgres + Pub/Sub Emulator + App).  
- `demo-app.dockerfile` → Dockerfile for building the app image.  
- `k8s/` → Kubernetes manifests (Deployment, Service, Job for DB/Pub/Sub init).  
- `README.md` → Project documentation.  

---

## 🔄 Workflow

1. A client sends an **HTTP request** for login with credentials to the **gRPC-Gateway**.
2. The **gRPC-Gateway** converts the HTTP request to a **gRPC call** and forwards it to the gRPC server.
3. The **gRPC server** processes authentication and returns a **JWT token** via gRPC response.
4. The **gRPC-Gateway** converts the gRPC response back to **HTTP response** with the JWT token.
5. For transfer requests, the client sends **HTTP requests with JWT token** to the **gRPC-Gateway**.
6. The **gRPC interceptor** validates the JWT token from the converted gRPC call.
7. The **gRPC service** validates the request and checks user balances in PostgreSQL.  
8. If valid → inserts the transaction into DB.  
9. The gRPC service then **publishes an event directly to Google Pub/Sub** (`transactions` topic).  
10. A **Pub/Sub consumer** subscribes to the Pub/Sub topic, processes the message, and sends an **ack** to confirm successful handling.  

---

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
* Redis
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

* HTTP Gateway (user-facing) → `http://localhost:8080`
* gRPC server (internal) → `localhost:9090`
* Redis → `localhost:6379`
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

# Clone the python repo in ref, cd to samples, snippets
pip install -r requirements.txt
python3 publisher.py demo-project create transactions # (create topic)
python3 subscriber.py demo-project create transactions sub-transactions # (create pull sub)
```

📌 Reference: [Pub/Sub Emulator Docs](https://cloud.google.com/pubsub/docs/emulator)

---

## 5. API Endpoints

**Note:** All API calls are made via HTTP, which are automatically converted to gRPC calls by the gRPC-Gateway.

### 🔐 Authentication Endpoints

#### 1️⃣ Login

Authenticate user and get JWT token. **(HTTP → gRPC-Gateway → gRPC Service)**

**Request:**

```bash
curl --location 'http://127.0.0.1:<PORT>/v1/auth/login' \
--header 'Content-Type: application/json' \
--data '{
  "username": 2,
  "password": "password456"
}'
```

**Response (Success):**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
}
```

**Response (Invalid Credentials):**

```json
{
  "success": false,
  "error_message": "invalid username or password"
}
```

---

#### 2️⃣ Logout

Invalidate JWT token (add to Redis blacklist). **(HTTP → gRPC-Gateway → gRPC Service)**

**Request:**

```bash
curl --location --request POST 'http://127.0.0.1:<PORT>/v1/auth/logout' \
--header 'Authorization: Bearer <JWT_TOKEN>'
```

**Response:**

```json
{
  "success": true,
}
```

---

### 💰 Transfer Endpoints (Requires Authentication)

**Note:** All requests go through HTTP → gRPC-Gateway → gRPC Service with JWT validation in gRPC interceptors.

#### 3️⃣ Get User Balance

Get the current balance of the authenticated user.

**Request:**

```bash
curl --location 'http://127.0.0.1:<PORT>/v1/transfer/<USER_ID>/balance' \
--header 'Authorization: Bearer <JWT_TOKEN>'
```

**Response:**

```json
{
  "user_id": "2",
  "balance": 1500.50,
}
```

---

#### 4️⃣ Get Transactions of a User

Fetch all transactions for a specific user by ID.

**Request:**

```bash
curl --location 'http://127.0.0.1:<PORT>/v1/transfer/<USER_ID>/transactions' \
--header 'Authorization: Bearer <JWT_TOKEN>'
```

**Response:**

```json
{
  "number": "2",
  "transactions": [
    {
      "id": 1,
      "from_user": "1",
      "to_user": "2",
      "amount": 200,
      "created_at": "2024-01-01T10:00:00Z"
    },
    {
      "id": 2,
      "from_user": "1",
      "to_user": "3",
      "amount": 100,
      "created_at": "2024-01-01T11:00:00Z"
    }
  ]
}
```

---

#### 5️⃣ Send Money (Transfer)

Create a new transaction between users.

**Request:**

```bash
curl --location 'http://127.0.0.1:<PORT>/v1/transfer/send' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer <JWT_TOKEN>' \
--data '{
  "from": 2,
  "to": 3,
  "amount": 12
}'
```

**Response (Success):**

```json
{
  "success": true,
  "transaction_id": "123",
  "message": "transfer completed successfully"
}
```

**Response (Insufficient Balance):**

```json
{
  "success": false,
  "error_message": "insufficient balance"
}
```

**Response (Unauthorized):**

```json
{
  "code": 401,
  "message": "unauthorized: invalid or expired token",
  "details": []
}
```

---

## 🏗️ Architecture Overview

```
[HTTP Client] 
    ↓ HTTP Request with JWT
[gRPC-Gateway :8080]
    ↓ Convert HTTP to gRPC  
[gRPC Interceptor] → JWT Validation
    ↓ Authenticated gRPC Call
[gRPC Service :9090] 
    ↓ Business Logic
[PostgreSQL] ← Store Transaction
[Redis] ← Session Management  
[Google Pub/Sub] ← Publish Events
```


---



✅ Now you have the project running with **PostgreSQL + Redis + Google Pub/Sub + JWT Authentication** for secure money transfers with session management.