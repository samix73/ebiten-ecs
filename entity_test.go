package ecs_test

import (
	"slices"
	"testing"

	"github.com/samix73/game/ecs"
	"github.com/stretchr/testify/assert"
	"golang.org/x/image/math/f64"
)

type TransformComponent struct {
	Position f64.Vec2
	Rotation float64
}

func (t *TransformComponent) Init() {
	t.Position = f64.Vec2{0, 0}
	t.Rotation = 0
}

func (t *TransformComponent) Reset() {
	t.Position = f64.Vec2{0, 0}
	t.Rotation = 0
}

type CameraComponent struct {
	Zoom float64
}

func (c *CameraComponent) Init() {
	c.Zoom = 1.0
}

func (c *CameraComponent) Reset() {
	c.Zoom = 1.0
}

func NewPlayerEntity(tb testing.TB, em *ecs.EntityManager) ecs.EntityID {
	tb.Helper()

	entityID := em.NewEntity()

	transform := ecs.AddComponent[TransformComponent](em, entityID)
	assert.NotNil(tb, transform)

	return entityID
}

func NewCameraEntity(tb testing.TB, em *ecs.EntityManager) ecs.EntityID {
	tb.Helper()

	entityID := em.NewEntity()

	transform := ecs.AddComponent[TransformComponent](em, entityID)
	if _, ok := tb.(*testing.B); !ok {
		assert.NotNil(tb, transform)
	}
	camera := ecs.AddComponent[CameraComponent](em, entityID)
	if _, ok := tb.(*testing.B); !ok {
		assert.NotNil(tb, camera)
	}

	return entityID
}

func NewEmptyEntity(tb testing.TB, em *ecs.EntityManager) ecs.EntityID {
	tb.Helper()

	return em.NewEntity()
}

func TestEntityCreation(t *testing.T) {
	em := ecs.NewEntityManager()

	player := NewPlayerEntity(t, em)
	assert.NotEqual(t, player, ecs.UndefinedID)
	camera := NewCameraEntity(t, em)
	assert.NotEqual(t, camera, ecs.UndefinedID)
	empty := NewEmptyEntity(t, em)
	assert.NotEqual(t, empty, ecs.UndefinedID)
}

func BenchmarkQueryEntities(b *testing.B) {
	em := ecs.NewEntityManager()

	// Create a set of entities with Transform components
	for range 500_000 {
		NewPlayerEntity(b, em)
	}

	for range 500_000 {
		NewCameraEntity(b, em)
	}

	for range 1000 {
		NewEmptyEntity(b, em)
	}

	b.Run("Query Only", func(b *testing.B) {
		for b.Loop() {
			for entityID := range ecs.Query[TransformComponent](em) {
				_ = entityID // Just consume the entityID
			}
		}
	})

	b.Run("Query2 Only", func(b *testing.B) {
		for b.Loop() {
			for entityID := range ecs.Query2[TransformComponent, CameraComponent](em) {
				_ = entityID // Just consume the entityID
			}
		}
	})

	b.Run("GetComponent Only", func(b *testing.B) {
		// Pre-collect entity IDs
		entityIDs := slices.Collect(ecs.Query[TransformComponent](em))

		b.ResetTimer()
		for b.Loop() {
			for _, entityID := range entityIDs {
				if _, ok := ecs.GetComponent[TransformComponent](em, entityID); !ok {
					b.Fatalf("Expected component for entity %d", entityID)
				}
			}
		}
	})

	b.Run("Query + GetComponent", func(b *testing.B) {
		for b.Loop() {
			for entityID := range ecs.Query[TransformComponent](em) {
				if _, ok := ecs.GetComponent[TransformComponent](em, entityID); !ok {
					b.Fatalf("Expected component for entity %d", entityID)
				}
			}
		}
	})

	b.Run("Query2 + GetComponent", func(b *testing.B) {
		for b.Loop() {
			for entityID := range ecs.Query2[TransformComponent, CameraComponent](em) {
				if _, ok := ecs.GetComponent[TransformComponent](em, entityID); !ok {
					b.Fatalf("Expected component for entity %d", entityID)
				}

				if _, ok := ecs.GetComponent[CameraComponent](em, entityID); !ok {
					b.Fatalf("Expected component for entity %d", entityID)
				}
			}
		}
	})
}
