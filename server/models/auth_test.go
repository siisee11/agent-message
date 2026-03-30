package models

import "testing"

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
				PIN:      "1234",
			},
		},
		{
			name: "valid username with symbols",
			req: RegisterRequest{
				Username: "alice.smith-1",
				PIN:      "123456",
			},
		},
		{
			name: "empty username",
			req: RegisterRequest{
				Username: "   ",
				PIN:      "1234",
			},
			wantErr: true,
		},
		{
			name: "username too short",
			req: RegisterRequest{
				Username: "ab",
				PIN:      "1234",
			},
			wantErr: true,
		},
		{
			name: "username has spaces",
			req: RegisterRequest{
				Username: "alice smith",
				PIN:      "1234",
			},
			wantErr: true,
		},
		{
			name: "pin too short",
			req: RegisterRequest{
				Username: "alice",
				PIN:      "123",
			},
			wantErr: true,
		},
		{
			name: "pin non numeric",
			req: RegisterRequest{
				Username: "alice",
				PIN:      "12a4",
			},
			wantErr: true,
		},
		{
			name: "pin too long",
			req: RegisterRequest{
				Username: "alice",
				PIN:      "1234567",
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
