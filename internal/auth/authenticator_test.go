package auth

import (
	"testing"
	"time"
	"github.com/google/uuid"
)

func TestCheckPasswordHash(t *testing.T) {
	// First, we need to create some hashed passwords for testing
	password1 := "correctPassword123!"
	password2 := "anotherPassword456!"
	hash1, _ := HashPassword(password1)
	hash2, _ := HashPassword(password2)

	tests := []struct {
		name     string
		password string
		hash     string
		wantErr  bool
	}{
		{
			name:     "Correct password",
			password: password1,
			hash:     hash1,
			wantErr:  false,
		},
		{
			name:     "Incorrect password",
			password: "wrongPassword",
			hash:     hash1,
			wantErr:  true,
		},
		{
			name:     "Password doesn't match different hash",
			password: password1,
			hash:     hash2,
			wantErr:  true,
		},
		{
			name:     "Empty password",
			password: "",
			hash:     hash1,
			wantErr:  true,
		},
		{
			name:     "Invalid hash",
			password: password1,
			hash:     "invalidhash",
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
func TestJWT(t *testing.T) {
	// First, we need to create some hashed passwords for testing
	user1 := uuid.New()
	user2 := uuid.New()
	user3 := uuid.New()
	secretToken1 := "correctPassword123!"
	secretToken2 := "anotherPassword456!"
	secretToken3 := "iperPassword"
	exp1, _ := time.ParseDuration("10h")
	exp2, _ := time.ParseDuration("10h")
	exp3, _ := time.ParseDuration("1µs")
	jwt1, _ := MakeJWT(user1, secretToken1, exp1)
	jwt2, _ := MakeJWT(user2, secretToken2, exp2)
	jwt3, _ := MakeJWT(user3, secretToken3, exp3)


	tests := []struct {
		name     string
		secret string
		token     string
		wantErr  bool
	}{
		{
			name:     "Correct password",
			secret: secretToken1,
			token:     jwt1,
			wantErr:  false,
		},
		{
			name:     "Incorrect password",
			secret: "wrongPassword",
			token:     jwt1,
			wantErr:  true,
		},
		{
			name:     "Password doesn't match different hash",
			secret: secretToken1,
			token:     jwt2,
			wantErr:  true,
		},
		{
			name:     "Empty password",
			secret: "",
			token:     jwt1,
			wantErr:  true,
		},
		{
			name:     "expired token",
			secret: secretToken3,
			token:     jwt3,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateJWT(tt.token, tt.secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckPasswordHash() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}