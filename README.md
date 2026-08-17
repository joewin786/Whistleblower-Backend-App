# 🛡️ Whistleblower REST API (Backend)

A high-performance RESTful API backend built with **Go (Golang)** for the Whistleblower System. This API provides authentication (JWT & Google OAuth), report & evidence management, AI-assisted report analysis via Google Gemini, real-time messaging (WebSocket & Pusher), push notifications (Firebase Cloud Messaging), and SMTP email integration.

---

## 🚀 Key Features

- 🔐 **Authentication & Security**:
  - User Registration & Login (JWT Access & Refresh Tokens).
  - Google OAuth2 Single Sign-On.
  - Multi-Factor Password Reset & Password Change via Email OTP.
  - Role-Based Access Control (RBAC) for Admin & SuperAdmin roles.
- 📋 **Whistleblower Report & Evidence Management**:
  - Anonymous and Authenticated report submissions.
  - File and image evidence upload management.
  - Report status tracking, admin assignment, and review workflow.
- 🤖 **AI Integration (Google Gemini)**:
  - Automatic risk assessment and summary generation for reports.
  - AI Assistant / Chat Agent to guide reporters.
- 💬 **Real-time Communication**:
  - Live messaging between reporters and administrators via WebSocket & Pusher.
- 🔔 **Notification System**:
  - Push Notifications via Firebase Cloud Messaging (FCM).
  - Email Notifications via SMTP (Gomail).
- 📊 **Analytics & Reporting**:
  - Report statistics and aggregated metrics for the Admin Dashboard.
- ⚙️ **System Configuration & Feedback**:
  - Category management, dynamic workflows, and system settings.
  - User feedback submission & management.

---

## 🛠️ Tech Stack

- **Programming Language**: Go 1.24+
- **HTTP Router**: [Chi v5](https://github.com/go-chi/chi/v5)
- **Database & ORM**: PostgreSQL / SQLite with [GORM](https://gorm.io/)
- **Authentication**: JWT (`golang-jwt/jwt/v5`), Google API Client
- **Real-Time**: Gorilla WebSocket, Pusher HTTP Go SDK
- **AI Model**: Google Gemini API SDK
- **Push Notification**: Firebase Admin SDK (`firebase.google.com/go/v4`)
- **Email Service**: Gomail (`gopkg.in/mail.v2`)
- **Container & Deployment**: Docker & Kubernetes

---

## 📂 Project Structure

```text
wb-api/
├── main.go                       # Main Go application entry point
├── go.mod / go.sum               # Go dependency management
├── REST API.yaml                 # OpenAPI / Swagger REST API Specification
├── Dockerfile                    # Multi-stage Docker build configuration
├── k8s/                          # Kubernetes deployment manifests
├── internal/
│   ├── admin/                    # Admin, role, category & workflow logic
│   ├── ai/                       # Google Gemini AI report analysis
│   ├── analytics/                # Analytics handlers & metric calculations
│   ├── auth/                     # Auth services (JWT, Google OAuth, Admin RBAC)
│   ├── chatagent/                # AI Chatbot agent handler
│   ├── database/                 # DB Connection (PostgreSQL/SQLite) & Migrations
│   ├── evidence/                 # Upload & evidence file handlers
│   ├── feedback/                 # User feedback management
│   ├── messages/                 # Internal messaging feature
│   ├── models/                   # GORM Database models
│   ├── notifications/            # Firebase FCM integration & token management
│   ├── reports/                  # Report CRUD & handling workflow logic
│   ├── reviews/                  # Report review and assessment logic
│   ├── utils/                    # Utility helpers (JSON responses, hashing, JWT)
│   └── websocket/                # WebSocket connection hub & handler
├── routes/
│   └── router.go                 # REST API router registration & middleware
├── scripts/                      # Helper scripts / migrations
└── uploads/                      # Uploaded evidence file storage directory
```

---

## 📋 System Prerequisites

Ensure you have the following installed:

- **Go**: version `1.24` or later.
- **PostgreSQL**: version `14+` (or SQLite for local testing).
- **Firebase Service Account**: `firebase-services-account.json` file for FCM notifications.
- **Google Gemini API Key**: from Google AI Studio.

---

## ⚙️ Environment Setup (`.env`)

Create a `.env` file in the root of `wb-api/` (copy from `.env.example`):

```env
# Server Port
PORT=8080

# Database Configuration
DB_DRIVER=postgres
DB_SOURCE=postgresql://postgres:password@localhost:5432/whistleblower?sslmode=disable

# Authentication & Keys
GOOGLE_CLIENT_ID=your-google-client-id.apps.googleusercontent.com
JWT_SECRET=your-super-secret-jwt-key

# Email SMTP Setup (Gomail)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASS=your-app-password

# Google Gemini AI Integration
GEMINI_API_KEY=your-gemini-api-key
GEMINI_API_KEY_CHAT=your-gemini-chat-api-key

# Firebase FCM Configuration
FIREBASE_SERVICE_ACCOUNT_PATH=firebase-services-account.json
```

---

## 🏃 Running the Application

### 1. Local Development

```bash
# 1. Navigate to backend directory
cd wb-api

# 2. Download dependencies
go mod download

# 3. Run application
go run main.go
```

The server will start at `http://localhost:8080`.

### 2. Docker Setup

```bash
# Build Docker Image
docker build -t wb-api:latest .

# Run Docker Container
docker run -d -p 8080:8080 --env-file .env --name wb-api-app wb-api:latest
```

---

## 🌐 REST API Endpoints Overview

Full OpenAPI/Swagger documentation is available in [`REST API.yaml`](file:///c:/Users/JOEWIN/Project/Whistleblower/wb-api/REST%20API.yaml).

| Method | Endpoint | Description | Access |
| :--- | :--- | :--- | :--- |
| **POST** | `/auth/register` | Register a new user | Public |
| **POST** | `/auth/login` | User login (returns JWT) | Public |
| **POST** | `/auth/google` | Authenticate via Google OAuth | Public |
| **GET** | `/auth/me` | Fetch authenticated user profile | User |
| **POST** | `/admin/auth/login` | Admin & SuperAdmin login | Public |
| **GET** | `/reports` | Fetch reports list | Admin / User |
| **POST** | `/reports` | Submit a new whistleblower report | Anonymous / User |
| **POST** | `/reports/{id}/evidence` | Upload evidence attachment | Reporter / Admin |
| **GET** | `/analytics/overview` | Fetch analytics summary metrics | Admin |
| **WS** | `/ws` | WebSocket endpoint for real-time messaging | Authenticated |

---

## 📄 License & Copyright

Copyright © 2026 PTFIC Whistleblower Team.
