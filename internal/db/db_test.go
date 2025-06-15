package db

import (
	"os"
	"testing"
)

func TestGetMaxPoolSizeDefault(t *testing.T) {
	os.Unsetenv("MONGO_MAX_POOL_SIZE")
	expected := uint64(100)
	if got := getMaxPoolSize(); got != expected {
		t.Errorf("expected %d, got %d", expected, got)
	}
}

func TestGetMaxPoolSizeFromEnv(t *testing.T) {
	os.Setenv("MONGO_MAX_POOL_SIZE", "50")
	defer os.Unsetenv("MONGO_MAX_POOL_SIZE")
	expected := uint64(50)
	if got := getMaxPoolSize(); got != expected {
		t.Errorf("expected %d, got %d", expected, got)
	}
}

func TestGetMaxPoolSizeInvalid(t *testing.T) {
	os.Setenv("MONGO_MAX_POOL_SIZE", "invalid")
	defer os.Unsetenv("MONGO_MAX_POOL_SIZE")
	expected := uint64(100)
	if got := getMaxPoolSize(); got != expected {
		t.Errorf("expected %d, got %d", expected, got)
	}
}
