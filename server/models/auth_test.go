package models

import (
	"strings"
	"testing"
)

func TestRegisterRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		req     RegisterRequest
		wantErr bool
	}{
		{
			name: "valid",
			req: RegisterRequest{
				Username: "alice",
				Password: "1234",
			},
		},
		{
			name: "valid username with symbols",
			req: RegisterRequest{
				Username: "alice.smith-1",
				Password: "abc12345",
			},
		},
		{
			name: "empty username",
			req: RegisterRequest{
				Username: "   ",
				Password: "1234",
			},
			wantErr: true,
		},
		{
			name: "username too short",
			req: RegisterRequest{
				Username: "ab",
				Password: "1234",
			},
			wantErr: true,
		},
		{
			name: "username has spaces",
			req: RegisterRequest{
				Username: "alice smith",
				Password: "1234",
			},
			wantErr: true,
		},
		{
			name: "password too short",
			req: RegisterRequest{
				Username: "alice",
				Password: "123",
			},
			wantErr: true,
		},
		{
			name: "password may be non numeric",
			req: RegisterRequest{
				Username: "alice",
				Password: "12a4",
			},
		},
		{
			name: "password too long",
			req: RegisterRequest{
				Username: "alice",
				Password: strings.Repeat("a", 73),
			},
			wantErr: true,
		},
		{
			name: "legacy pin field still validates",
			req: RegisterRequest{
				Username:  "alice",
				LegacyPIN: "1234",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr && err == nil {
				t.Fatalf("expected validation error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no validation error, got %v", err)
			}
		})
	}
}

func TestAuthResponseValidate(t *testing.T) {
	resp := AuthResponse{}
	if err := resp.Validate(); err == nil {
		t.Fatalf("expected error for missing token")
	}

	resp.Token = "token"
	if err := resp.Validate(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
