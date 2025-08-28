package utils

import (
	"fmt"
	"testing"
	"time"
)

func TestCacheAddGet(t *testing.T) {
	const interval = 5 * time.Second
	cases := []struct {
		key string
		val []byte
	}{
		{
			key: "https://pokeapi.co/api/v2/pokemon/1",
			val: []byte("bulbasaur"),
		},
		{
			key: "https://pokeapi.co/api/v2/pokemon/25",
			val: []byte("pikachu"),
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("Test case %v", i), func(t *testing.T) {
			cache := NewCache(interval)

			// Add to cache
			cache.Add(c.key, c.val)

			// Try to get it back
			val, ok := cache.Get(c.key)
			if !ok {
				t.Errorf("expected key %q to be in cache", c.key)
			}
			if string(val) != string(c.val) {
				t.Errorf("expected %q, got %q", c.val, val)
			}
		})
	}
}

func TestCacheExpiration(t *testing.T) {
	const ttl = 10 * time.Millisecond
	cache := NewCache(ttl)

	key := "https://pokeapi.co/api/v2/pokemon/150"
	val := []byte("mewtwo")

	// Add to cache
	cache.Add(key, val)

	// Immediately fetch → should exist
	if got, ok := cache.Get(key); !ok || string(got) != string(val) {
		t.Errorf("expected to find key %q immediately", key)
	}

	// Wait until after expiration
	time.Sleep(ttl + 5*time.Millisecond)

	// Should no longer exist
	if _, ok := cache.Get(key); ok {
		t.Errorf("expected key %q to expire", key)
	}
}
