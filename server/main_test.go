package main

import "testing"

func TestParseCSVEnv(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://a.example.com, https://b.example.com")
	got := parseCSVEnv("CORS_ALLOWED_ORIGINS", "*")
	if len(got) != 2 || got[0] != "https://a.example.com" || got[1] != "https://b.example.com" {
		t.Fatalf("unexpected parseCSVEnv output: %#v", got)
	}
}

func TestParseCSVEnvDefaults(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "   ")
	got := parseCSVEnv("CORS_ALLOWED_ORIGINS", "*")
	if len(got) != 1 || got[0] != "*" {
		t.Fatalf("unexpected default output: %#v", got)
	}
}
