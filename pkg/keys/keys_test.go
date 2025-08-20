package keys

import (
	"testing"
)

func TestUserQueueKey(t *testing.T) {
	expected := "user:queue:tenant1:123:session456"
	result := UserQueueKey("tenant1", 123, "session456")

	if result != expected {
		t.Errorf("UserQueueKey() = %v, want %v", result, expected)
	}
}

func TestQueueSeqKey(t *testing.T) {
	expected := "queue:seq:tenant1:123"
	result := QueueSeqKey("tenant1", 123)

	if result != expected {
		t.Errorf("QueueSeqKey() = %v, want %v", result, expected)
	}
}

func TestReleaseSeqKey(t *testing.T) {
	expected := "release:seq:tenant1:123"
	result := ReleaseSeqKey("tenant1", 123)

	if result != expected {
		t.Errorf("ReleaseSeqKey() = %v, want %v", result, expected)
	}
}

func TestActiveUsersKey(t *testing.T) {
	expected := "active:users:tenant1:123"
	result := ActiveUsersKey("tenant1", 123)

	if result != expected {
		t.Errorf("ActiveUsersKey() = %v, want %v", result, expected)
	}
}

func TestIPThrottleKey(t *testing.T) {
	expected := "throttle:ip:tenant1:123:iphash789"
	result := IPThrottleKey("tenant1", 123, "iphash789")

	if result != expected {
		t.Errorf("IPThrottleKey() = %v, want %v", result, expected)
	}
}

func TestUserDedupeKey(t *testing.T) {
	expected := "dedupe:user:tenant1:123"
	result := UserDedupeKey("tenant1", 123)

	if result != expected {
		t.Errorf("UserDedupeKey() = %v, want %v", result, expected)
	}
}

func TestMetricsKey(t *testing.T) {
	expected := "metrics:tenant1:123:enter_total"
	result := MetricsKey("tenant1", 123, "enter_total")

	if result != expected {
		t.Errorf("MetricsKey() = %v, want %v", result, expected)
	}
}
