package ecs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewBitmask(t *testing.T) {
	t.Parallel()

	b := NewBitmask()
	require.Empty(t, b.bits)
}

func TestBitmask_SetAndHas(t *testing.T) {
	t.Parallel()

	b := NewBitmask()

	tests := []struct {
		bit uint
	}{
		{0},
		{1},
		{63},
		{64},
		{127},
		{128},
		{1000},
	}

	for _, tt := range tests {
		exists := b.Has(tt.bit)
		require.False(t, exists)

		b.Set(tt.bit)
		exists = b.Has(tt.bit)
		require.True(t, exists)
	}
}

func TestBitmask_Unset(t *testing.T) {
	t.Parallel()

	b := NewBitmask()
	b.Set(10)
	b.Set(100)

	require.True(t, b.Has(10))
	require.True(t, b.Has(100))

	b.Unset(10)
	require.False(t, b.Has(10))
	require.True(t, b.Has(100))

	b.Unset(100)
	require.False(t, b.Has(100))
	require.False(t, b.Has(10))

	// Unset non-existent
	b.Unset(999) // Should not panic
}

func TestBitmask_HasAll(t *testing.T) {
	t.Parallel()

	// Scenario: b1 is superset of b2
	b1 := NewBitmask()
	b1.Set(1)
	b1.Set(100)

	b2 := NewBitmask()
	b2.Set(1)

	require.True(t, b1.HasAll(b2))

	b2.Set(100)
	require.True(t, b1.HasAll(b2))

	b2.Set(2) // b2 now has (1, 100, 2). b1 is missing 2.
	require.False(t, b1.HasAll(b2))

	// Larger range
	b3 := NewBitmask()
	b3.Set(1000)
	b1.Set(1000) // b1 has 1000 now
	require.True(t, b1.HasAll(b3))

	b3.Set(1001)
	require.False(t, b1.HasAll(b3))
}

func TestBitmask_Equals(t *testing.T) {
	t.Parallel()

	b1 := NewBitmask()
	b1.Set(1)
	b1.Set(100)

	b2 := NewBitmask()
	b2.Set(1)
	b2.Set(100)

	require.True(t, b1.Equals(b2))

	b2.Set(2)
	require.False(t, b1.Equals(b2))
}
