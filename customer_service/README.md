# Customer Service Microservice

Microservice untuk mengelola Customer Service Hub dengan fitur AI-to-Human handover. Sistem ini memungkinkan AI menangani pertanyaan user, dan jika AI tidak bisa jawab atau confidence rendah, case akan di-escalate ke CS agent.

## 📋 Table of Contents

- [Architecture](#architecture)
- [Features](#features)
- [Message Classification](#message-classification)
- [Pubsub Communication](#pubsub-communication)
- [API Endpoints](#api-endpoints)
- [Integration Guide](#integration-guide)
- [Deployment](#deployment)

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Main Application                          │
│  - pubsub/updateStatus.go (webhook handler)                 │
│  - services/groqService.go (AI)                             │
└──────────────────┬──────────────────────────────────────────┘
                   │ Pubsub Events
                   ↓
┌─────────────────────────────────────────────────────────────┐
│              Customer Service Microservice                   │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │   Pubsub     │→ │   Services   │→ │ Controllers  │     │
│  │  Handlers    │  │   (Business  │  │  (API REST)  │     │
│  │              │  │    Logic)    │  │              │     │
│  └──────────────┘  └──────┬───────┘  └──────────────┘     │
│                           ↓                                  │
│                    ┌──────────────┐                         │
│                    │ Repositories │                         │
│                    │  (Database)  │                         │
│                    └──────────────┘                         │
└─────────────────────────────────────────────────────────────┘
```

### Components

| Component | Purpose | Location |
|-----------|---------|----------|
| **DTOs** | Request/Response structures, Events | `dtos/` |
| **Pubsub Handlers** | Event publishers & subscribers | `pubsub/` |
| **Services** | Business logic layer | `services/` |
| **Controllers** | REST API endpoints | `controllers/` |
| **Repositories** | Database access (shared) | `../repositories/customerServiceCases/` |

---

## ✨ Features

### Core Features
- ✅ **AI-to-Human Handover** - AI escalate ke CS jika tidak bisa jawab
- ✅ **Message Classification** - Distinguish initial question, gangguan, inquiry, payment, campaign
- ✅ **Case Management** - Create, claim, escalate, resolve, close cases
- ✅ **Real-time Communication** - Pubsub-based event system
- ✅ **SLA Tracking** - 2-hour response time SLA
- ✅ **Billing Integration** - Charge Rp 500 per case when closed
- ✅ **Audit Trail** - Track all assignments and escalations
- ✅ **Dashboard API** - REST API for CS dashboard

### Billing Model
- **AI Session**: Rp 380 per session (AI handles full conversation)
- **CS Conversation**: Rp 500 per case (human CS handles)
- **Billing Trigger**: When CS closes case (not per message!)

---

## 🎯 Message Classification

### CRITICAL: Membedakan Tipe Pesan

System menggunakan **MessageClassifier** untuk distinguish message types:

#### 1. Initial Question (Greeting)
**Keywords**: halo, hello, hai, hi, pagi, siang, sore, malam, permisi, assalamualaikum

**Classification**: `initial_question`
**Confidence**: 0.9
**Action**: Trigger AI conversation

**Example**:
```
User: "Halo, mau tanya PDAM"
→ Classification: initial_question
→ Action: AI responds
```

#### 2. Gangguan (Complaint/Outage)
**Keywords**: air mati, tidak ada air, air tidak mengalir, air macet, gangguan, rusak, bocor, pipa bocor, meter rusak, air keruh, air bau, tekanan rendah, air kecil

**Classification**: `gangguan`
**Confidence**: 0.95
**Priority**: High (2)
**Severity**: high
**Action**: Create case with higher priority

**Example**:
```
User: "Air saya mati sejak 2 hari yang lalu"
→ Classification: gangguan
→ Confidence: 0.95
→ Severity: high
→ Action: Escalate to CS if AI can't handle
```

#### 3. Inquiry (Tagihan/Info)
**Keywords**: tagihan, cek tagihan, berapa tagihan, info tagihan, nomor pelanggan, meter, cek meter, baca meter, biaya, tarif, harga air

**Classification**: `inquiry`
**Confidence**: 0.85
**Action**: AI handles with PPOB inquiry function

**Example**:
```
User: "Mau cek tagihan PDAM"
→ Classification: inquiry
→ Action: AI triggers PPOB inquiry function
```

#### 4. Payment
**Keywords**: bayar, pembayaran, virtual account, va, bank, transfer

**Classification**: `payment`
**IsPaymentFlow**: true
**Action**: Route to payment service

**Example**:
```
User: "Mau bayar tagihan via BCA"
→ Classification: payment
→ IsPaymentFlow: true
→ Action: Show bank selection, create VA
```

#### 5. Campaign
**Keywords**: promo, diskon, penawaran, campaign

**Classification**: varies
**IsCampaign**: true
**Action**: Skip CS case creation (just log)

**Example**:
```
User: "Info promo PDAM bulan ini"
→ IsCampaign: true
→ Action: AI responds, no case created
```

### Classification Flow

```go
// In pubsub/handlers.go
classifier := GetMessageClassifier()
classification, _ := classifier.Classify(messageContent, clientId, userPhone)

if classification.HasActiveCSCase {
    // Route to active CS case
    routeMessageToCase(classification)
} else if classification.IsEscalation {
    // AI will ask user permission to escalate
    aiAskEscalationConfirmation()
} else if classification.IsCampaign {
    // Just log, don't create case
    logCampaignMessage()
} else if classification.IsPaymentFlow {
    // Route to payment service
    routeToPaymentService()
} else {
    // AI handles normally
    triggerAIConversation()
}
```

---

## 📡 Pubsub Communication

### Pubsub Topics

#### CS Case Events
| Topic | Event | When Published |
|-------|-------|----------------|
| `cs.case.created` | CSCaseCreatedEvent | New case created |
| `cs.case.claimed` | CSCaseClaimedEvent | CS agent claims case |
| `cs.case.escalated` | CSCaseEscalatedEvent | Case escalated to senior CS |
| `cs.case.resolved` | CSCaseResolvedEvent | Case marked as resolved |
| `cs.case.closed` | CSCaseClosedEvent | Case closed (triggers billing) |
| `cs.case.message.sent` | CSCaseMessageSentEvent | CS sends message |
| `cs.case.message.received` | CSCaseMessageReceivedEvent | User replies to CS case |
| `cs.case.sla.breached` | CSCaseSLABreachedEvent | SLA deadline exceeded |

#### AI Escalation Events
| Topic | Event | When Published |
|-------|-------|----------------|
| `ai.escalation.request` | AIEscalationRequestEvent | AI wants to escalate |
| `ai.escalation.confirmed` | AIEscalationConfirmedEvent | User confirms "Ya" |
| `ai.escalation.rejected` | AIEscalationRejectedEvent | User rejects "Tidak" |

#### Message Routing Events
| Topic | Event | When Published |
|-------|-------|----------------|
| `user.message.received` | - | Raw message from user |
| `user.message.classified` | MessageClassification | After classification |

### Publishing Example

```go
// In main application (pubsub/updateStatus.go)
import (
    csDtos "gitlab.com/bot3342545/il-dashboard/customer_service/dtos"
    csPubsub "gitlab.com/bot3342545/il-dashboard/customer_service/pubsub"
)

// Publish AI escalation confirmed event
event := &csDtos.AIEscalationConfirmedEvent{
    SessionID:   sessionID,
    ClientId:    clientId,
    UserPhone:   userPhone,
    UserName:    userName,
    UserMessage: userMessage,
    Category:    "gangguan",
    ConfirmedAt: time.Now(),
}

eventJSON, _ := json.Marshal(event)
gocom.PubSub().Publish(csDtos.TopicAIEscalationConfirmed, string(eventJSON))
```

### Subscribing Example

```go
// In CS microservice (pubsub/handlers.go)
func (h *CSPubSubHandlerImpl) Init() {
    // Subscribe to AI escalation confirmed
    pubsub.Get().Subscribe(
        dtos.TopicAIEscalationConfirmed,
        h.handleAIEscalationConfirmed(),
    )
}

func (h *CSPubSubHandlerImpl) handleAIEscalationConfirmed() pubsub.PubSubEventHandler {
    return func(topic, msg string) {
        var event dtos.AIEscalationConfirmedEvent
        json.Unmarshal([]byte(msg), &event)

        // Create CS case
        caseData, _ := h.caseService.CreateCase(
            event.ClientId,
            event.UserPhone,
            event.UserName,
            event.SessionID,
            event.Category,
            event.UserMessage,
        )

        // Publish case created event
        // ...
    }
}
```

---

## 🔌 API Endpoints

### Base URL
```
/api/v1/cs
```

### Authentication
All endpoints require:
- `Authorization: Bearer <token>` header
- User must have CS role: `cs_agent`, `cs_supervisor`, `cs_manager`, or `admin`

### Endpoints

#### 1. Get Cases List
```http
GET /api/v1/cs/cases?status=new&severity=high&page=1&limit=20
```

**Query Parameters**:
- `status` (optional): new, assigned, resolved, escalated, closed
- `severity` (optional): low, normal, high, urgent
- `category` (optional): initial_question, gangguan, inquiry_payment, campaign
- `assigned_to` (optional): user_id
- `sla_breached` (optional): true/false
- `page` (default: 1)
- `limit` (default: 20)
- `sort` (optional): created_at, severity, sla_deadline
- `sort_order` (optional): asc, desc

**Response**:
```json
{
  "status": "success",
  "message": "Cases retrieved successfully",
  "data": {
    "cases": [
      {
        "id": "01HZQ...",
        "case_number": "CS-20251203-0001",
        "client_id": "CLIENT_ABC",
        "user_phone": "6281234567890",
        "user_name": "Ahmad Wijaya",
        "category": "gangguan",
        "subject": "Air mati sejak 2 hari yang lalu",
        "status": "new",
        "severity": "high",
        "priority": 2,
        "sla_deadline": "2025-12-03T12:00:00Z",
        "sla_breached": false,
        "wait_time_seconds": 3600,
        "wait_time_formatted": "1h 0m ago",
        "total_messages": 5,
        "created_at": "2025-12-03T10:00:00Z"
      }
    ],
    "pagination": {
      "total": 45,
      "page": 1,
      "limit": 20,
      "total_pages": 3
    },
    "stats": {
      "new_count": 12,
      "assigned_count": 18,
      "resolved_count": 10,
      "closed_count": 5,
      "sla_breached_count": 3
    }
  }
}
```

#### 2. Get Case Details
```http
GET /api/v1/cs/cases/:id
```

**Response**: Full case details with conversation history, notes, and assignments.

#### 3. Claim Case
```http
POST /api/v1/cs/cases/:id/claim
Content-Type: application/json

{
  "severity": "high",
  "priority": 2,
  "notes": "Urgent case, water outage"
}
```

#### 4. Send Message
```http
POST /api/v1/cs/cases/:id/messages
Content-Type: application/json

{
  "message_content": "Halo Pak Ahmad, bisa info alamat lengkapnya?",
  "message_type": "text"
}
```

#### 5. Add Internal Note
```http
POST /api/v1/cs/cases/:id/notes
Content-Type: application/json

{
  "note_type": "internal",
  "note_content": "Sudah cek di sistem, ada gangguan di area tersebut",
  "is_internal": true
}
```

#### 6. Resolve Case
```http
POST /api/v1/cs/cases/:id/resolve
Content-Type: application/json

{
  "resolution_notes": "Issue resolved. Water is running. Caused by pipe maintenance."
}
```

#### 7. Close Case (Triggers Billing)
```http
POST /api/v1/cs/cases/:id/close
Content-Type: application/json

{
  "close_notes": "User confirmed satisfied with resolution"
}
```

**Note**: Closing case triggers billing (Rp 500 charged to client).

#### 8. Escalate Case
```http
POST /api/v1/cs/cases/:id/escalate
Content-Type: application/json

{
  "escalate_to": "USER_MANAGER_001",
  "escalation_reason": "Complex technical issue requiring senior expertise",
  "severity": "urgent",
  "priority": 1
}
```

#### 9. Get Statistics
```http
GET /api/v1/cs/statistics?period=today
```

---

## 🔗 Integration Guide

### 1. Integrate dengan Main Application

#### Step 1: Import Customer Service Module

```go
// In main.go or router setup
import (
    csControllers "gitlab.com/bot3342545/il-dashboard/customer_service/csControllers"
    csPubsub "gitlab.com/bot3342545/il-dashboard/customer_service/pubsub"
    csServices "gitlab.com/bot3342545/il-dashboard/customer_service/services"
)

func main() {
    // Initialize CS pubsub handlers
    csSvc := csServices.GetCustomerServiceSvc()
    csHandler := csPubsub.GetCSPubSubHandler(csSvc)
    csHandler.Init()

    // Register CS API routes
    csController := csControllers.GetCustomerServiceController()
    router := gin.Default()
    apiV1 := router.Group("/api/v1")
    csController.RegisterRoutes(apiV1)

    // Start server
    router.Run(":8080")
}
```

#### Step 2: Update Webhook Handler (pubsub/updateStatus.go)

```go
// Add to handleActionWaWebHook function
import (
    csDtos "gitlab.com/bot3342545/il-dashboard/customer_service/dtos"
    csPubsub "gitlab.com/bot3342545/il-dashboard/customer_service/pubsub"
)

// Classify message
classifier := csPubsub.GetMessageClassifier()
classification, _ := classifier.Classify(messageContent, clientId, userPhone)

// Publish classified message event
classificationJSON, _ := json.Marshal(classification)
gocom.PubSub().Publish(csDtos.TopicUserMessageClassified, string(classificationJSON))

// If user confirms escalation
if userResponse == "ya" || userResponse == "yes" || userResponse == "lanjut" {
    event := &csDtos.AIEscalationConfirmedEvent{
        SessionID:   sessionID,
        ClientId:    clientId,
        UserPhone:   message.From,
        UserName:    contactName,
        UserMessage: previousMessage,
        Category:    "gangguan", // or from classification
        ConfirmedAt: time.Now(),
    }
    eventJSON, _ := json.Marshal(event)
    gocom.PubSub().Publish(csDtos.TopicAIEscalationConfirmed, string(eventJSON))
}
```

#### Step 3: Update AI Service (services/groqService.go)

```go
// Add to AI conversation logic
import csDtos "gitlab.com/bot3342545/il-dashboard/customer_service/dtos"

func (s *GroqServiceImpl) ChatWithKnowledgeAndFunctions(...) {
    // Check AI confidence
    if confidence < 0.6 || answerNotFound {
        // Set escalation pending flag
        escalationKey := fmt.Sprintf("cs_escalation_pending:%s:%s", clientId, userPhone)
        gocom.KeyVal().Set(escalationKey, "pending", 10*time.Minute)

        // Publish AI escalation request
        event := &csDtos.AIEscalationRequestEvent{
            SessionID:    sessionID,
            ClientId:     clientId,
            UserPhone:    userPhone,
            UserMessage:  userMessage,
            AIResponse:   "Maaf, pertanyaan ini perlu bantuan CS. Lanjutkan?",
            Reason:       "knowledge_gap",
            AIConfidence: confidence,
            RequestedAt:  time.Now(),
        }
        eventJSON, _ := json.Marshal(event)
        gocom.PubSub().Publish(csDtos.TopicAIEscalationRequest, string(eventJSON))

        return "Maaf, pertanyaan Anda memerlukan bantuan Customer Service kami. Apakah Anda ingin saya hubungkan dengan CS? (Balas: Ya/Tidak)", nil
    }
}
```

### 2. Database Migration

Jalankan migration untuk create tables:
```bash
mysql -u username -p database_name < migrations/008_customer_service_hub.sql
```

### 3. Configure Pricing

```sql
-- Add CS conversation pricing
INSERT INTO pricing (id, client_id, service_type, category, cost, currency, is_active)
VALUES (ULID(), NULL, 'whatsapp', 'cs_conversation', 500.00, 'IDR', TRUE);

-- Add AI session pricing
INSERT INTO pricing (id, client_id, service_type, category, cost, currency, is_active)
VALUES (ULID(), NULL, 'whatsapp', 'ai_session', 380.00, 'IDR', TRUE);
```

---

## 🚀 Deployment

### Option 1: Monolith (Single Application) - CURRENT SETUP

Keep everything in one application. CS module runs as part of main application.

**Pros**:
- ✅ Simple deployment
- ✅ Shared database (no network latency)
- ✅ Single codebase
- ✅ Easier debugging

**Cons**:
- ❌ Harder to scale CS module independently
- ❌ Single point of failure

### Option 2: Microservice (Separate Application) - NANTI KALAU MAU PISAH

Deploy CS module as separate microservice.

**Pros**:
- ✅ Independent scaling
- ✅ Independent deployment
- ✅ Technology flexibility
- ✅ Fault isolation

**Cons**:
- ❌ More complex deployment
- ❌ Network latency
- ❌ Need API Gateway / Service Mesh

---

## 🔪 Cara Pisahin Jadi Microservice (Nanti)

### Step-by-Step Separation Guide

Ketika lu mau pisahin jadi microservice terpisah, ikutin langkah ini:

#### Step 1: Buat File Main Terpisah

Buat file baru untuk CS service yang terpisah:

```bash
mkdir -p cmd/cs-service
```

```go
// cmd/cs-service/main.go
package main

import (
    "github.com/ariandi/gocom"
    "github.com/ariandi/gocom/logger"
    "github.com/gin-gonic/gin"

    // Import CS modules
    csControllers "gitlab.com/bot3342545/il-dashboard/customer_service/csControllers"
    csPubsub "gitlab.com/bot3342545/il-dashboard/customer_service/pubsub"
    csServices "gitlab.com/bot3342545/il-dashboard/customer_service/services"

    // Import shared repositories (tetap butuh akses DB)
    _ "gitlab.com/bot3342545/il-dashboard/repositories/customerServiceCases"
)

func main() {
    logger.Infof("🚀 Starting Customer Service Microservice...")

    // Initialize gocom (DB, Redis, Pubsub)
    gocom.Init()

    // Initialize CS pubsub handlers
    logger.Infof("📡 Initializing pubsub handlers...")
    csSvc := csServices.GetCustomerServiceSvc()
    csHandler := csPubsub.GetCSPubSubHandler(csSvc)
    csHandler.Init()

    // Setup Gin router
    router := gin.Default()

    // CORS middleware (if needed)
    router.Use(func(c *gin.Context) {
        c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
        c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }
        c.Next()
    })

    // Health check
    router.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status":  "ok",
            "service": "customer-service",
        })
    })

    // Register CS API routes
    apiV1 := router.Group("/api/v1")
    csController := csControllers.GetCustomerServiceController()
    csController.RegisterRoutes(apiV1)

    logger.Infof("✅ Customer Service Microservice ready on :8081")
    logger.Infof("📋 Endpoints:")
    logger.Infof("   - Health: http://localhost:8081/health")
    logger.Infof("   - API: http://localhost:8081/api/v1/cs/*")

    // Start server on different port (8081)
    router.Run(":8081")
}
```

#### Step 2: Pisahin Routes di Main App

Di main application, **remove CS routes** karena sudah dihandle di CS microservice:

```go
// main.go (main application)
func main() {
    // ... existing setup ...

    // ❌ REMOVE THIS - Pindah ke CS microservice
    // csController := csControllers.GetCustomerServiceController()
    // csController.RegisterRoutes(apiV1)

    // Keep other routes
    router.Run(":8080")
}
```

**TAPI TETAP KEEP** pubsub integration di webhook handler karena main app perlu publish events!

#### Step 3: Build Separate Binary

Build CS service sebagai binary terpisah:

```bash
# Build CS service
go build -o bin/cs-service cmd/cs-service/main.go

# Build main app
go build -o bin/main-app cmd/main/main.go

# Run both
./bin/main-app &          # Port 8080
./bin/cs-service &        # Port 8081
```

#### Step 4: Docker Setup (Optional)

Buat Dockerfile terpisah untuk CS service:

```dockerfile
# Dockerfile.cs-service
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy all source (karena butuh shared modules)
COPY . .

# Download dependencies
RUN go mod download

# Build CS service binary
RUN CGO_ENABLED=0 GOOS=linux go build -o cs-service cmd/cs-service/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

# Copy binary
COPY --from=builder /app/cs-service .

# Copy config files (if any)
COPY --from=builder /app/config.properties .

EXPOSE 8081

CMD ["./cs-service"]
```

#### Step 5: Docker Compose

Jalanin both services dengan Docker Compose:

```yaml
# docker-compose.yml
version: '3.8'

services:
  # Main Application
  main-app:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: main-app
    ports:
      - "8080:8080"
    environment:
      - DATABASE_HOST=mysql
      - DATABASE_PORT=3306
      - DATABASE_USER=root
      - DATABASE_PASSWORD=root
      - DATABASE_NAME=il_dashboard
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - PUBSUB_HOST=redis
      - PUBSUB_PORT=6379
    depends_on:
      - mysql
      - redis
    networks:
      - app-network
    restart: unless-stopped

  # Customer Service Microservice
  cs-service:
    build:
      context: .
      dockerfile: Dockerfile.cs-service
    container_name: cs-service
    ports:
      - "8081:8081"
    environment:
      - DATABASE_HOST=mysql
      - DATABASE_PORT=3306
      - DATABASE_USER=root
      - DATABASE_PASSWORD=root
      - DATABASE_NAME=il_dashboard
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - PUBSUB_HOST=redis
      - PUBSUB_PORT=6379
    depends_on:
      - mysql
      - redis
    networks:
      - app-network
    restart: unless-stopped

  # MySQL Database (SHARED)
  mysql:
    image: mysql:8
    container_name: mysql
    ports:
      - "3306:3306"
    environment:
      - MYSQL_ROOT_PASSWORD=root
      - MYSQL_DATABASE=il_dashboard
    volumes:
      - mysql-data:/var/lib/mysql
      - ./migrations:/docker-entrypoint-initdb.d
    networks:
      - app-network
    restart: unless-stopped

  # Redis (SHARED - untuk pubsub)
  redis:
    image: redis:7-alpine
    container_name: redis
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    networks:
      - app-network
    restart: unless-stopped

volumes:
  mysql-data:
  redis-data:

networks:
  app-network:
    driver: bridge
```

**Jalanin**:
```bash
# Start all services
docker-compose up -d

# Check logs
docker-compose logs -f main-app
docker-compose logs -f cs-service

# Stop all
docker-compose down
```

#### Step 6: Nginx Reverse Proxy (Optional)

Kalau mau single endpoint, pake Nginx:

```nginx
# nginx.conf
upstream main_app {
    server main-app:8080;
}

upstream cs_service {
    server cs-service:8081;
}

server {
    listen 80;

    # Route /api/v1/cs/* ke CS microservice
    location /api/v1/cs {
        proxy_pass http://cs_service;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # Route semua request lain ke main app
    location / {
        proxy_pass http://main_app;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

Add ke docker-compose.yml:
```yaml
  nginx:
    image: nginx:alpine
    container_name: nginx
    ports:
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
    depends_on:
      - main-app
      - cs-service
    networks:
      - app-network
```

#### Step 7: Update Environment Variables

Buat `.env` file untuk manage configs:

```bash
# .env
# Database (SHARED)
DATABASE_HOST=localhost
DATABASE_PORT=3306
DATABASE_USER=root
DATABASE_PASSWORD=root
DATABASE_NAME=il_dashboard

# Redis (SHARED untuk Pubsub)
REDIS_HOST=localhost
REDIS_PORT=6379

# Service Ports
MAIN_APP_PORT=8080
CS_SERVICE_PORT=8081
```

#### Step 8: Test Communication

Test pubsub communication antar services:

```bash
# Terminal 1: Run main app
go run cmd/main/main.go

# Terminal 2: Run CS service
go run cmd/cs-service/main.go

# Terminal 3: Test webhook (main app receives, publishes event)
curl -X POST http://localhost:8080/api/v1/whatsapp/webhook \
  -H "Content-Type: application/json" \
  -d '{ "entry": [...] }'

# Terminal 4: Check CS service logs
# Should see: "[CSPubSubHandler] Received AI escalation confirmation: ..."
```

#### Step 9: Kubernetes Deployment (Advanced - Optional)

Kalau deploy ke production pake K8s:

```yaml
# k8s/cs-service.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cs-service
  namespace: production
spec:
  replicas: 3  # Scale sesuai kebutuhan
  selector:
    matchLabels:
      app: cs-service
  template:
    metadata:
      labels:
        app: cs-service
    spec:
      containers:
      - name: cs-service
        image: your-registry/cs-service:latest
        ports:
        - containerPort: 8081
          name: http
        env:
        - name: DATABASE_HOST
          valueFrom:
            configMapKeyRef:
              name: app-config
              key: database_host
        - name: REDIS_HOST
          valueFrom:
            configMapKeyRef:
              name: app-config
              key: redis_host
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8081
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8081
          initialDelaySeconds: 10
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: cs-service
  namespace: production
spec:
  selector:
    app: cs-service
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8081
  type: ClusterIP  # Internal only, expose via Ingress
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cs-service-ingress
  namespace: production
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  rules:
  - host: api.yourdomain.com
    http:
      paths:
      - path: /api/v1/cs
        pathType: Prefix
        backend:
          service:
            name: cs-service
            port:
              number: 80
```

Deploy:
```bash
# Apply configs
kubectl apply -f k8s/cs-service.yaml

# Check status
kubectl get pods -n production | grep cs-service
kubectl logs -f deployment/cs-service -n production

# Scale up/down
kubectl scale deployment cs-service --replicas=5 -n production
```

---

### 🎯 Key Points untuk Separasi

#### Yang HARUS SHARED:
1. **Database** - Both services akses DB yang sama (customer_service_cases, etc)
2. **Redis/Pubsub** - Communication channel antar services
3. **Repositories** - Shared code untuk DB access

#### Yang TERPISAH:
1. **API Endpoints** - CS endpoints hanya di CS service
2. **Binary/Container** - Separate deployment
3. **Port** - Main app: 8080, CS service: 8081
4. **Scaling** - Bisa scale CS service independently

#### Communication Flow After Separation:
```
User WhatsApp Message
    ↓
Main App (8080) - Webhook Handler
    ↓
Publish: ai.escalation.confirmed (via Redis Pubsub)
    ↓
CS Service (8081) - Pubsub Subscriber
    ↓
Create CS Case
    ↓
CS Dashboard calls CS Service API
```

---

## 📊 Monitoring & Observability

### Key Metrics to Track

1. **Case Metrics**
   - Total cases created per day
   - Average first response time
   - Average resolution time
   - SLA compliance rate
   - Cases by category (gangguan, inquiry, etc.)

2. **CS Performance**
   - Cases per CS agent
   - Messages per case
   - Average session duration

3. **Classification Accuracy**
   - Message classification confidence
   - Escalation rate
   - False positive rate

### Logging

All operations logged with structured logging:
```
[CustomerServiceSvc CreateCase] Case created successfully: 01HZQ... (Case #CS-20251203-0001)
[CSPubSubHandler] Message classification - Type: gangguan, Confidence: 0.95, IsEscalation: true
[CustomerServiceSvc CloseCase] Case CS-20251203-0001 billed: Rp 500.00, Reference: BILL_123
```

---

## 🔒 Security

### Authentication & Authorization

- All API endpoints require authentication token
- Role-based access control (RBAC) - Actual Implementation:
  - `STAFF` (CS Agents): Can view, claim, message, resolve, close, escalate cases for their client
  - `OWNER`: Can manage all cases for their own client (based on `authInfo.ClientId`)
  - `ADMIN`: Full access to all cases across all clients

**Note:** All CS management operations (claim, send messages, resolve, close, escalate) are available to STAFF, OWNER, and ADMIN roles.

### Data Protection

- Sensitive data (phone numbers, user info) handled securely
- Audit trail for all actions
- Billing records immutable

---

## 📝 Notes & Best Practices

### Message Classification
- **CRITICAL**: Always classify messages before routing
- Check for active CS case first before triggering AI
- Campaign messages should NOT create CS cases
- Payment flow messages should route to payment service

### Billing
- Billing only triggered when CS closes case
- Individual messages are FREE (not charged per message)
- One charge per case: Rp 500
- Prevent double billing with billing_reference check

### SLA Management
- Default SLA: 2 hours from case creation
- SLA breach alerts sent to managers
- Auto-escalation can be configured for severe breaches

### Pubsub Best Practices
- Always publish events after state changes
- Use structured events with proper typing
- Handle event failures gracefully
- Monitor pubsub queue lengths

---

## 🆘 Troubleshooting

### Issue: Messages not routed to CS case

**Solution**: Check message classification
```go
classification := classifier.Classify(message, clientId, userPhone)
log.Infof("Classification: %s, HasActiveCSCase: %v",
    classification.Classification, classification.HasActiveCSCase)
```

### Issue: Billing not triggered

**Solution**: Check case status and billing_reference
```sql
SELECT id, case_number, status, billing_cost, billing_reference
FROM customer_service_cases
WHERE status = 'closed' AND billing_cost = 0;
```

### Issue: Pubsub events not received

**Solution**: Check pubsub subscription
```go
// Verify subscription registered
pubsub.Get().Subscribe(dtos.TopicAIEscalationConfirmed, handler)
```

---

## 📚 References

- [Summary 08: Customer Service Hub](../summary/08-CUSTOMER-SERVICE-HUB.md)
- [Summary 09: CS Hub Flows](../summary/09-CS-HUB-FLOWS.md)
- [Summary 10: Implementation Checklist](../summary/10-CS-HUB-IMPLEMENTATION-CHECKLIST.md)
- [Migration 008: Database Schema](../migrations/008_customer_service_hub.sql)

---

## 📧 Support

For questions or issues:
- Check logs: `[CustomerServiceSvc]`, `[CSPubSubHandler]`, `[MessageClassifier]`
- Review pubsub events published/received
- Verify database records in `customer_service_cases` table

---

**Version**: 1.0
**Last Updated**: 2025-12-03
**Status**: Ready for Integration & Testing
