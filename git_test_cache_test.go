package devflow

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTestCache_SaveAndValidate(t *testing.T) {
	// Skip if not in a git repository
	if _, err := RunCommandSilent("git", "rev-parse", "HEAD"); err != nil {
		t.Skip("Not in a git repository")
	}

	cache := NewTestCache()
	testMsg := "✅ vet ok, ✅ tests ok"

	// Clean up any existing cache
	cache.InvalidateCache()

	// Initially cache should be invalid
	if cache.IsCacheValid() {
		t.Error("Cache should be invalid before saving")
	}

	// Save cache with message
	if err := cache.SaveCache(testMsg); err != nil {
		t.Fatalf("Failed to save cache: %v", err)
	}

	// Now cache should be valid
	if !cache.IsCacheValid() {
		t.Error("Cache should be valid after saving")
	}

	// Cached message should match
	if got := cache.GetCachedMessage(); got != testMsg {
		t.Errorf("GetCachedMessage() = %q, want %q", got, testMsg)
	}

	// Cleanup
	cache.InvalidateCache()
}

func TestTestCache_InvalidateCache(t *testing.T) {
	if _, err := RunCommandSilent("git", "rev-parse", "HEAD"); err != nil {
		t.Skip("Not in a git repository")
	}

	cache := NewTestCache()

	// Save cache
	if err := cache.SaveCache("test message"); err != nil {
		t.Fatalf("Failed to save cache: %v", err)
	}

	// Verify it's valid
	if !cache.IsCacheValid() {
		t.Error("Cache should be valid after saving")
	}

	// Invalidate
	cache.InvalidateCache()

	// Should be invalid now
	if cache.IsCacheValid() {
		t.Error("Cache should be invalid after invalidation")
	}
}

func TestTestCache_CacheKey(t *testing.T) {
	cache := NewTestCache()

	key, err := cache.getCacheKey()
	if err != nil {
		t.Fatalf("Failed to get cache key: %v", err)
	}

	if len(key) != 16 {
		t.Errorf("Cache key should be 16 characters, got %d: %s", len(key), key)
	}

	// Key should be consistent
	key2, _ := cache.getCacheKey()
	if key != key2 {
		t.Error("Cache key should be consistent across calls")
	}
}

func TestTestCache_GitState(t *testing.T) {
	if _, err := RunCommandSilent("git", "rev-parse", "HEAD"); err != nil {
		t.Skip("Not in a git repository")
	}

	cache := NewTestCache()

	state, err := cache.getGitState()
	if err != nil {
		t.Fatalf("Failed to get git state: %v", err)
	}

	// State should be in format "commitHash:diffHash"
	if len(state) < 10 {
		t.Errorf("Git state seems too short: %s", state)
	}

	// State should contain a colon separator
	if !containsColon(state) {
		t.Errorf("Git state should contain colon separator: %s", state)
	}

	// State should be consistent when code hasn't changed
	state2, _ := cache.getGitState()
	if state != state2 {
		t.Error("Git state should be consistent when code hasn't changed")
	}
}

func TestTestCache_CacheDirectory(t *testing.T) {
	cache := NewTestCache()

	expectedDir := filepath.Join(os.TempDir(), "gotest-cache")
	if cache.cacheDir != expectedDir {
		t.Errorf("Cache dir should be %s, got %s", expectedDir, cache.cacheDir)
	}
}

func containsColon(s string) bool {
	for _, c := range s {
		if c == ':' {
			return true
		}
	}
	return false
}
