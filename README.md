# 🐦 Chirpy - Social Media API

**Chirpy** is a Twitter-like social media REST API built with **Go** and **PostgreSQL**. It provides a complete backend for a social media platform with user authentication, post creation (chirps), and premium subscription features.

## ✨ Features

- **User Management**: Registration, login, and profile updates
- **Authentication**: JWT-based authentication with refresh tokens
- **Chirps (Posts)**: Create, read, and delete social media posts
- **Premium Features**: Chirpy Red subscription integration via Polka webhooks
- **Content Moderation**: Automatic profanity filtering
- **RESTful API**: Clean HTTP endpoints following REST conventions
- **Type-Safe Database**: SQLC-generated Go code for PostgreSQL operations
- **Security**: Password hashing with bcrypt, secure token management

## 🏗️ Project Structure

```
gohttp/
├── assets/
│   └── logo.png             # Chirpy logo asset
├── internal/
│   ├── auth/                # Authentication utilities
│   │   ├── auth.go         # JWT, bcrypt, token handling
│   │   └── auth_test.go    # Authentication tests
│   └── database/           # SQLC-generated database code
├── sql/
│   ├── queries/            # SQL queries for SQLC
│   │   ├── users.sql      # User operations
│   │   ├── chirps.sql     # Chirp operations
│   │   └── tokens.sql     # Token management
│   └── schema/            # Database migrations
│       ├── 001_users.sql
│       ├── 002_chirps.sql
│       ├── 003_passwords.sql
│       ├── 004_refresh_tokens.sql
│       └── 005_chirpy_red.sql
├── main.go                # HTTP server setup and routing
├── api.go                 # API handlers and business logic
├── index.html            # Welcome page
├── sqlc.yaml             # SQLC configuration
└── go.mod                # Go module definition
```

## 🚀 Getting Started

### Prerequisites

- Go 1.24.4 or higher
- PostgreSQL database
- [SQLC](https://sqlc.dev/) for code generation
- [Goose](https://github.com/pressly/goose) for database migrations (optional)

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/UUest/gohttp.git
   cd gohttp
   ```

2. **Install dependencies**
   ```bash
   go mod tidy
   ```

3. **Set up environment variables**
   
   Create a `.env` file in the project root:
   ```env
   DB_URL=postgres://username:password@localhost:5432/chirpy?sslmode=disable
   JWT_SECRET=your-super-secret-jwt-key
   PLATFORM=dev
   POLKA_KEY=your-polka-webhook-key
   ```

4. **Set up the database**
   - Create a PostgreSQL database named `chirpy`
   - Run migrations from `sql/schema/` directory

5. **Generate database code**
   ```bash
   sqlc generate
   ```

6. **Build and run**
   ```bash
   go build -o chirpy
   ./chirpy
   ```

The server will start on `http://localhost:8080`

## 📖 API Reference

### Authentication

#### Register User
```http
POST /api/users
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepassword"
}
```

#### Login
```http
POST /api/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepassword"
}
```

#### Refresh Token
```http
POST /api/refresh
Authorization: Bearer <refresh_token>
```

#### Revoke Token
```http
POST /api/revoke
Authorization: Bearer <refresh_token>
```

### User Management

#### Update User
```http
PUT /api/users
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "email": "newemail@example.com",
  "password": "newpassword"
}
```

### Chirps (Posts)

#### Create Chirp
```http
POST /api/chirps
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "body": "This is my first chirp!"
}
```

#### Get All Chirps
```http
GET /api/chirps
```

Optional query parameters:
- `author_id`: Filter by user ID
- `sort`: Sort order (`asc` or `desc`)

#### Get Single Chirp
```http
GET /api/chirps/{chirpID}
```

#### Delete Chirp
```http
DELETE /api/chirps/{chirpID}
Authorization: Bearer <access_token>
```

### Admin & Monitoring

#### Health Check
```http
GET /api/healthz
```

#### Metrics (Admin)
```http
GET /admin/metrics
```

#### Reset Users (Admin)
```http
POST /admin/reset
```

### Webhooks

#### Polka Webhook (Premium Upgrades)
```http
POST /api/polka/webhooks
Authorization: ApiKey <polka_key>
Content-Type: application/json

{
  "event": "user.upgraded",
  "data": {
    "user_id": "user-uuid-here"
  }
}
```

## 🛠️ Development

### Database Migrations

Using [Goose](https://github.com/pressly/goose):

```bash
# Install goose
go install github.com/pressly/goose/v3/cmd/goose@latest

# Run migrations
goose -dir sql/schema postgres "$DB_URL" up

# Check migration status
goose -dir sql/schema postgres "$DB_URL" status
```

### Code Generation

```bash
# Regenerate database code after schema changes
sqlc generate
```

### Testing

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...
```

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `DB_URL` | PostgreSQL connection string | Yes |
| `JWT_SECRET` | Secret key for JWT signing | Yes |
| `PLATFORM` | Platform identifier (dev/prod) | Yes |
| `POLKA_KEY` | API key for Polka webhooks | Yes |

## 🗄️ Database Schema

- **users**: User accounts with email authentication
- **chirps**: Social media posts with content and timestamps  
- **refresh_tokens**: Secure refresh token storage
- **user_passwords**: Hashed password storage
- **chirpy_red**: Premium subscription tracking

## 🔒 Security Features

- **Password Hashing**: bcrypt with salt for secure password storage
- **JWT Authentication**: Stateless authentication with access/refresh tokens
- **Content Filtering**: Automatic profanity detection and replacement
- **API Key Protection**: Webhook endpoints protected with API keys
- **Request Validation**: Input sanitization and validation

## 🎯 Features & Roadmap

### Current Features ✅
- [x] User registration and authentication
- [x] JWT-based session management
- [x] CRUD operations for chirps
- [x] Content moderation (profanity filtering)
- [x] Premium subscription integration
- [x] RESTful API design
- [x] PostgreSQL database with migrations
- [x] Static file serving

### Planned Enhancements 🚀
- [ ] **Rate Limiting**: Prevent API abuse
- [ ] **Email Verification**: Verify user email addresses
- [ ] **Follow System**: User following/followers
- [ ] **Like System**: Like/unlike chirps
- [ ] **Media Upload**: Image and video support
- [ ] **Real-time Updates**: WebSocket support
- [ ] **Search**: Full-text search for chirps
- [ ] **Hashtags**: Tag support and trending topics
- [ ] **Direct Messages**: Private messaging system
- [ ] **Admin Dashboard**: Web-based admin interface

### Technical Improvements 🔧
- [ ] **Caching**: Redis integration for performance
- [ ] **Logging**: Structured logging with levels
- [ ] **Metrics**: Prometheus metrics collection
- [ ] **Docker**: Containerization support
- [ ] **CI/CD**: GitHub Actions workflow
- [ ] **API Documentation**: OpenAPI/Swagger specs
- [ ] **Load Testing**: Performance benchmarks

## 🧪 Testing

Example API calls using curl:

```bash
# Register a new user
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'

# Login
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'

# Create a chirp (replace TOKEN with your JWT)
curl -X POST http://localhost:8080/api/chirps \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer TOKEN" \
  -d '{"body":"Hello, Chirpy world!"}'

# Get all chirps
curl http://localhost:8080/api/chirps
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with [Go](https://golang.org/) standard library HTTP server
- Database operations powered by [SQLC](https://sqlc.dev/)
- JWT authentication using [golang-jwt/jwt](https://github.com/golang-jwt/jwt)
- Password hashing with [golang.org/x/crypto/bcrypt](https://pkg.go.dev/golang.org/x/crypto/bcrypt)
- PostgreSQL driver by [lib/pq](https://github.com/lib/pq)
- Environment management with [godotenv](https://github.com/joho/godotenv)