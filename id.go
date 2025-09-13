package ecs

import (
	"sync/atomic"
)

var nextID = atomic.Uint64{}

// NextID generates and returns a new unique ID.
func NextID() ID {
	return ID(nextID.Add(1))
}

// ID represents a unique identifier for entities and systems within the ECS framework.
type ID uint64

// UndefinedID is a constant representing an undefined or invalid ID.
const UndefinedID ID = 0
