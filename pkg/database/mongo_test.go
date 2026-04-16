package database

import "testing"

func TestMongoConfigBuildURLUsesExplicitURL(t *testing.T) {
	cfg := &MongoConfig{
		URL:      "mongodb://mongo:27017/qs?replicaSet=rs0&directConnection=true",
		Host:     "127.0.0.1:27017",
		Database: "ignored",
	}

	if got := cfg.BuildURL(); got != cfg.URL {
		t.Fatalf("BuildURL() = %q, want explicit url %q", got, cfg.URL)
	}
}

func TestMongoConfigBuildURLFromFields(t *testing.T) {
	cfg := &MongoConfig{
		Host:             "mongo:27017",
		Username:         "app_user",
		Password:         "s3cret",
		Database:         "qs",
		ReplicaSet:       "rs0",
		DirectConnection: true,
	}

	want := "mongodb://app_user:s3cret@mongo:27017/qs?directConnection=true&replicaSet=rs0"
	if got := cfg.BuildURL(); got != want {
		t.Fatalf("BuildURL() = %q, want %q", got, want)
	}
}

func TestMongoConfigBuildURLWithTLSFlags(t *testing.T) {
	cfg := &MongoConfig{
		Host:                     "mongo:27017",
		Database:                 "qs",
		UseSSL:                   true,
		SSLInsecureSkipVerify:    true,
		SSLAllowInvalidHostnames: true,
	}

	want := "mongodb://mongo:27017/qs?tls=true&tlsAllowInvalidHostnames=true&tlsInsecure=true"
	if got := cfg.BuildURL(); got != want {
		t.Fatalf("BuildURL() = %q, want %q", got, want)
	}
}
