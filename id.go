package ecs

import (
	"sync/atomic"
)

var nextID = atomic.Uint64{}

func NextID() ID {
	return ID(nextID.Add(1))
}

type ID uint64

const UndefinedID ID = 0
