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
				AccountID: "alice",
				Password:  "1234",
			},
		},
		{
			name: "valid username with symbols",
			req: RegisterRequest{
				AccountID: "alice.smith-1",
				Password:  "abc12345",
			},
		},
		{
			name: "empty username",
			req: RegisterRequest{
				AccountID: "   ",
				Password:  "1234",
			},
			wantErr: true,
		},
		{
			name: "username too short",
			req: RegisterRequest{
				AccountID: "ab",
				Password:  "1234",
			},
			wantErr: true,
		},
		{
			name: "username has spaces",
			req: RegisterRequest{
				AccountID: "alice smith",
				Password:  "1234",
			},
			wantErr: true,
		},
		{
			name: "password too short",
			req: RegisterRequest{
				AccountID: "alice",
				Password:  "123",
			},
			wantErr: true,
		},
		{
			name: "password may be non numeric",
			req: RegisterRequest{
				AccountID: "alice",
				Password:  "12a4",
			},
		},
		{
			name: "password too long",
			req: RegisterRequest{
				AccountID: "alice",
				Password:  strings.Repeat("a", 73),
			},
			wantErr: true,
		},
		{
			name: "legacy pin field still validates",
			req: RegisterRequest{
				AccountID: "alice",
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

func TestUpdatePasswordRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdatePasswordRequest
		wantErr bool
	}{
		{
			name: "valid",
			req: UpdatePasswordRequest{
				CurrentPassword: "secret123",
				NewPassword:     "newsecret123",
			},
		},
		{
			name: "missing current password",
			req: UpdatePasswordRequest{
				NewPassword: "newsecret123",
			},
			wantErr: true,
		},
		{
			name: "new password too short",
			req: UpdatePasswordRequest{
				CurrentPassword: "secret123",
				NewPassword:     "123",
			},
			wantErr: true,
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
