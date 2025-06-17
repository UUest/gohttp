package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  false,
		},
		{
			name:     "long password within bcrypt limit",
			password: strings.Repeat("a", 72),
			wantErr:  false,
		},
		{
			name:     "password exceeding bcrypt limit",
			password: strings.Repeat("a", 100),
			wantErr:  true,
		},
		{
			name:     "password with special characters",
			password: "p@ssw0rd!#$%",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if hash == "" {
					t.Error("HashPassword() returned empty hash")
				}
				if hash == tt.password {
					t.Error("HashPassword() returned unhashed password")
				}
				// Verify the hash is different each time
				hash2, err := HashPassword(tt.password)
				if err != nil {
					t.Errorf("HashPassword() second call error = %v", err)
				}
				if hash == hash2 {
					t.Error("HashPassword() should return different hashes for same password")
				}
			}
		})
	}
}

func TestCheckPasswordHash(t *testing.T) {
	// Create a known hash for testing
	password := "testpassword123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to create hash for testing: %v", err)
	}

	tests := []struct {
		name     string
		hash     string
		password string
		wantErr  bool
	}{
		{
			name:     "correct password",
			hash:     hash,
			password: password,
			wantErr:  false,
		},
		{
			name:     "incorrect password",
			hash:     hash,
			password: "wrongpassword",
			wantErr:  true,
		},
		{
			name:     "empty password",
			hash:     hash,
			password: "",
			wantErr:  true,
		},
		{
			name:     "empty hash",
			hash:     "",
			password: password,
			wantErr:  true,
		},
		{
			name:     "invalid hash format",
			hash:     "invalid-hash",
			password: password,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckPasswordHash(tt.hash, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckPasswordHash() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMakeJWT(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "test-secret-key"

	tests := []struct {
		name        string
		userID      uuid.UUID
		tokenSecret string
		expiresIn   time.Duration
		wantErr     bool
	}{
		{
			name:        "valid token creation",
			userID:      userID,
			tokenSecret: tokenSecret,
			expiresIn:   time.Hour,
			wantErr:     false,
		},
		{
			name:        "zero expiration",
			userID:      userID,
			tokenSecret: tokenSecret,
			expiresIn:   0,
			wantErr:     false,
		},
		{
			name:        "negative expiration",
			userID:      userID,
			tokenSecret: tokenSecret,
			expiresIn:   -time.Hour,
			wantErr:     false,
		},
		{
			name:        "empty secret",
			userID:      userID,
			tokenSecret: "",
			expiresIn:   time.Hour,
			wantErr:     false,
		},
		{
			name:        "nil UUID",
			userID:      uuid.Nil,
			tokenSecret: tokenSecret,
			expiresIn:   time.Hour,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := MakeJWT(tt.userID, tt.tokenSecret, tt.expiresIn)
			if (err != nil) != tt.wantErr {
				t.Errorf("MakeJWT() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if token == "" {
					t.Error("MakeJWT() returned empty token")
				}
				// Verify token has proper JWT structure (header.payload.signature)
				parts := strings.Split(token, ".")
				if len(parts) != 3 {
					t.Errorf("MakeJWT() returned invalid JWT format, got %d parts, want 3", len(parts))
				}
			}
		})
	}
}

func TestValidateJWT(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "test-secret-key"

	// Create a valid token for testing
	validToken, err := MakeJWT(userID, tokenSecret, time.Hour)
	if err != nil {
		t.Fatalf("Failed to create valid token for testing: %v", err)
	}

	// Create an expired token
	expiredToken, err := MakeJWT(userID, tokenSecret, -time.Hour)
	if err != nil {
		t.Fatalf("Failed to create expired token for testing: %v", err)
	}

	// Create a token with different secret
	differentSecretToken, err := MakeJWT(userID, "different-secret", time.Hour)
	if err != nil {
		t.Fatalf("Failed to create token with different secret for testing: %v", err)
	}

	tests := []struct {
		name        string
		tokenString string
		tokenSecret string
		wantUserID  uuid.UUID
		wantErr     bool
	}{
		{
			name:        "valid token",
			tokenString: validToken,
			tokenSecret: tokenSecret,
			wantUserID:  userID,
			wantErr:     false,
		},
		{
			name:        "expired token",
			tokenString: expiredToken,
			tokenSecret: tokenSecret,
			wantUserID:  uuid.Nil,
			wantErr:     true,
		},
		{
			name:        "wrong secret",
			tokenString: differentSecretToken,
			tokenSecret: tokenSecret,
			wantUserID:  uuid.Nil,
			wantErr:     true,
		},
		{
			name:        "invalid token format",
			tokenString: "invalid.token.format",
			tokenSecret: tokenSecret,
			wantUserID:  uuid.Nil,
			wantErr:     true,
		},
		{
			name:        "empty token",
			tokenString: "",
			tokenSecret: tokenSecret,
			wantUserID:  uuid.Nil,
			wantErr:     true,
		},
		{
			name:        "malformed JWT",
			tokenString: "not-a-jwt-token",
			tokenSecret: tokenSecret,
			wantUserID:  uuid.Nil,
			wantErr:     true,
		},
		{
			name:        "empty secret",
			tokenString: validToken,
			tokenSecret: "",
			wantUserID:  uuid.Nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUserID, err := ValidateJWT(tt.tokenString, tt.tokenSecret)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJWT() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotUserID != tt.wantUserID {
				t.Errorf("ValidateJWT() gotUserID = %v, want %v", gotUserID, tt.wantUserID)
			}
		})
	}
}

func TestJWTRoundtrip(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "test-secret-key"
	expiresIn := time.Hour

	// Create a token
	token, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT() error = %v", err)
	}

	// Validate the token
	gotUserID, err := ValidateJWT(token, tokenSecret)
	if err != nil {
		t.Fatalf("ValidateJWT() error = %v", err)
	}

	if gotUserID != userID {
		t.Errorf("JWT roundtrip failed: got userID %v, want %v", gotUserID, userID)
	}
}

func TestJWTClaims(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "test-secret-key"
	expiresIn := time.Hour

	token, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT() error = %v", err)
	}

	// Parse the token to check claims
	parsedToken, err := jwt.ParseWithClaims(token, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	claims, ok := parsedToken.Claims.(*jwt.RegisteredClaims)
	if !ok {
		t.Fatal("Failed to get claims from token")
	}

	// Check issuer
	if claims.Issuer != "chirpy" {
		t.Errorf("Expected issuer 'chirpy', got '%s'", claims.Issuer)
	}

	// Check subject (userID)
	if claims.Subject != userID.String() {
		t.Errorf("Expected subject '%s', got '%s'", userID.String(), claims.Subject)
	}

	// Check that IssuedAt is recent
	if claims.IssuedAt == nil {
		t.Error("IssuedAt claim is nil")
	} else {
		issuedAt := claims.IssuedAt.Time
		if time.Since(issuedAt) > time.Minute {
			t.Error("IssuedAt claim is not recent")
		}
	}

	// Check that ExpiresAt is set correctly
	if claims.ExpiresAt == nil {
		t.Error("ExpiresAt claim is nil")
	} else {
		expectedExpiry := claims.IssuedAt.Time.Add(expiresIn)
		actualExpiry := claims.ExpiresAt.Time
		if !actualExpiry.Equal(expectedExpiry) {
			t.Errorf("Expected expiry %v, got %v", expectedExpiry, actualExpiry)
		}
	}
}

func BenchmarkHashPassword(b *testing.B) {
	password := "benchmarkpassword123"
	for i := 0; i < b.N; i++ {
		_, err := HashPassword(password)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCheckPasswordHash(b *testing.B) {
	password := "benchmarkpassword123"
	hash, err := HashPassword(password)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := CheckPasswordHash(hash, password)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMakeJWT(b *testing.B) {
	userID := uuid.New()
	tokenSecret := "benchmark-secret"
	expiresIn := time.Hour

	for i := 0; i < b.N; i++ {
		_, err := MakeJWT(userID, tokenSecret, expiresIn)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValidateJWT(b *testing.B) {
	userID := uuid.New()
	tokenSecret := "benchmark-secret"

	token, err := MakeJWT(userID, tokenSecret, time.Hour)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ValidateJWT(token, tokenSecret)
		if err != nil {
			b.Fatal(err)
		}
	}
}
