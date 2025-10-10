# Chirpy REST API

A Twitter-like microblogging platform built with Go, featuring user authentication, chirp management, and premium subscription functionality.

## Features

- **User Management**: Registration, authentication, and profile updates
- **Chirp System**: Create, read, and delete short messages (max 140 characters)
- **JWT Authentication**: Secure token-based authentication with refresh tokens
- **Content Filtering**: Automatic censorship of inappropriate words
- **Premium Subscriptions**: Chirpy Red upgrade functionality via webhook integration
- **Admin Dashboard**: Metrics and management endpoints
- **Database Integration**: PostgreSQL with SQLC for type-safe queries

## Tech Stack

- **Language**: Go 1.22.4
- **Database**: PostgreSQL
- **Authentication**: JWT tokens with Argon2id password hashing
- **Query Builder**: SQLC for type-safe database operations
- **Environment**: Environment variable configuration

## Prerequisites

- Go 1.22.4 or later
- PostgreSQL database
- Environment variables configured (see Configuration section)

## Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd chirpy
```

2. Install dependencies:
```bash
go mod download
```

3. Set up your environment variables (see Configuration section)

4. Run database migrations:
```bash
# Using goose or your preferred migration tool
goose -dir sql/schema up
```

5. Start the server:
```bash
go run main.go
```

The server will start on port 8080.

## Configuration

Create a `.env` file in the root directory with the following variables:

```env
DB_URL=postgres://username:password@localhost/dbname?sslmode=disable
JWT_SECRET=your-jwt-secret-key
PLATFORM=dev
POLKA_KEY=your-payment-api-key
```

### Environment Variables

- `DB_URL`: PostgreSQL connection string
- `JWT_SECRET`: Secret key for JWT token signing
- `PLATFORM`: Environment setting (dev/prod)
- `POLKA_KEY`: API key for payment webhook authentication

## API Endpoints

### Health Check
- `GET /api/healthz` - Server health check

### User Management
- `POST /api/users` - Create a new user
- `PUT /api/users` - Update user profile (requires authentication)
- `POST /api/login` - User login
- `POST /api/refresh` - Refresh access token
- `POST /api/revoke` - Revoke refresh token

### Chirp Management
- `GET /api/chirps` - Get all chirps (optional `author_id` query parameter)
- `GET /api/chirps/{chirp_id}` - Get specific chirp by ID
- `POST /api/chirps` - Create a new chirp (requires authentication)
- `DELETE /api/chirps/{chirp_id}` - Delete chirp (requires authentication, owner only)

### Payment Integration
- `POST /api/polka/webhooks` - Payment webhook for Chirpy Red upgrades

### Admin Endpoints
- `GET /admin/metrics` - View server metrics
- `POST /admin/reset` - Reset metrics and clear all users (dev only)

### Static Files
- `GET /app/*` - Serve static files (with hit counter)

## API Usage Examples

### User Registration
```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password123"}'
```

### User Login
```bash
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password123"}'
```

### Create Chirp
```bash
curl -X POST http://localhost:8080/api/chirps \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{"body": "Hello, Chirpy!"}'
```

### Get All Chirps
```bash
curl http://localhost:8080/api/chirps
```

### Get Chirps by Author
```bash
curl "http://localhost:8080/api/chirps?author_id=USER_UUID"
```

## Data Models

### User
```json
{
  "id": "uuid",
  "created_at": "timestamp",
  "updated_at": "timestamp",
  "email": "string",
  "is_chirpy_red": "boolean"
}
```

### Chirp
```json
{
  "id": "uuid",
  "created_at": "timestamp",
  "updated_at": "timestamp",
  "user_id": "uuid",
  "body": "string",
  "valid": "boolean"
}
```

## Authentication

The API uses JWT (JSON Web Tokens) for authentication. Include the token in the Authorization header:

```
Authorization: Bearer YOUR_JWT_TOKEN
```

### Token Types
- **Access Token**: Short-lived (1 hour by default) for API access
- **Refresh Token**: Long-lived for obtaining new access tokens

## Content Filtering

The API automatically filters inappropriate content by replacing taboo words with asterisks:
- kerfuffle → ****
- sharbert → ****
- fornax → ****

## Chirpy Red Premium

Users can upgrade to Chirpy Red premium status through webhook integration. The upgrade is processed via the `/api/polka/webhooks` endpoint.

## Database Schema

### Users Table
- `id` (UUID, Primary Key)
- `created_at` (Timestamp)
- `updated_at` (Timestamp)
- `email` (Text, Unique)
- `hashed_password` (Text)
- `is_chirpy_red` (Boolean)

### Chirps Table
- `id` (UUID, Primary Key)
- `created_at` (Timestamp)
- `updated_at` (Timestamp)
- `body` (Text)
- `user_id` (UUID, Foreign Key)

### Refresh Tokens Table
- `token` (Text, Primary Key)
- `user_id` (UUID, Foreign Key)
- `expires_at` (Timestamp)
- `revoked_at` (Timestamp)
- `created_at` (Timestamp)
- `updated_at` (Timestamp)

## Development

### Running Tests
```bash
go test ./...
```

### Database Migrations
The project uses SQLC for database operations. To regenerate database code after schema changes:

```bash
sqlc generate
```

### Project Structure
```
chirpy/
├── main.go                 # Main application entry point
├── payments.go            # Payment webhook handling
├── response.go            # HTTP response utilities
├── internal/
│   ├── auth/              # Authentication utilities
│   └── database/          # Database models and queries
├── sql/
│   ├── queries/           # SQLC query files
│   └── schema/            # Database migration files
└── assets/                # Static assets
```

## Error Handling

The API returns consistent error responses in JSON format:

```json
{
  "error": "Error message description"
}
```

Common HTTP status codes:
- `200` - Success
- `201` - Created
- `400` - Bad Request
- `401` - Unauthorized
- `403` - Forbidden
- `404` - Not Found
- `500` - Internal Server Error

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License.