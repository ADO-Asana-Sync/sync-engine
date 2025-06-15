package main

import (
	"os"
	"testing"
	"time"
)

// Returns 5 minutes duration when SLEEP_TIME environment variable is not set
func TestGetSleepTimeDefault(t *testing.T) {
	expected := 5 * time.Minute
	actual := getSleepTime()
	if actual != expected {
		t.Errorf("Expected sleep time: %v, but got: %v", expected, actual)
	}
}

// Returns duration parsed from SLEEP_TIME environment variable when it is set and valid
func TestGetSleepTimeWithValidSleepTime(t *testing.T) {
	// Set the SLEEP_TIME environment variable
	os.Setenv("SLEEP_TIME", "10s")
	defer os.Unsetenv("SLEEP_TIME")

	expected := 10 * time.Second
	actual := getSleepTime()
	if actual != expected {
		t.Errorf("Expected sleep time: %v, but got: %v", expected, actual)
	}
}

// Returns 5m when SLEEP_TIME environment variable is set but invalid
func TestGetSleepTimeInvalid(t *testing.T) {
	os.Setenv("SLEEP_TIME", "invalid")
	defer os.Unsetenv("SLEEP_TIME")
	expected := 5 * time.Minute
	actual := getSleepTime()
	if actual != expected {
		t.Errorf("Expected sleep time: %v, but got: %v", expected, actual)
	}
}

// Returns 10 minutes duration when SLEEP_TIME environment variable is set to "10m"
func TestGetSleepTimeWithValidSleepTimeMinutes(t *testing.T) {
	// Set the SLEEP_TIME environment variable
	os.Setenv("SLEEP_TIME", "10m")
	defer os.Unsetenv("SLEEP_TIME")

	expected := 10 * time.Minute
	actual := getSleepTime()
	if actual != expected {
		t.Errorf("Expected sleep time: %v, but got: %v", expected, actual)
	}
}

// Returns 1 hour duration when SLEEP_TIME environment variable is set to "1h"
func TestGetSleepTimeWithValidSleepTimeHours(t *testing.T) {
	// Set the SLEEP_TIME environment variable
	os.Setenv("SLEEP_TIME", "1h")
	defer os.Unsetenv("SLEEP_TIME")

	expected := 1 * time.Hour
	actual := getSleepTime()
	if actual != expected {
		t.Errorf("Expected sleep time: %v, but got: %v", expected, actual)
	}
}

func TestGetCacheTTLDefault(t *testing.T) {
	os.Unsetenv("PROPERTY_CACHE_TTL")
	expected := 24 * time.Hour
	if got := getCacheTTL(); got != expected {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

func TestGetCacheTTLWithEnv(t *testing.T) {
	os.Setenv("PROPERTY_CACHE_TTL", "12h")
	defer os.Unsetenv("PROPERTY_CACHE_TTL")
	expected := 12 * time.Hour
	if got := getCacheTTL(); got != expected {
		t.Errorf("expected %v, got %v", expected, got)
	}
}
