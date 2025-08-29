package ecs_test

import (
	"slices"
	"testing"

	ecs "github.com/samix73/ebiten-ecs"
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

func TestFilteredQueries(t *testing.T) {
	em := ecs.NewEntityManager()

	// Create entities with different zoom levels
	entity1 := em.NewEntity()
	camera1 := ecs.AddComponent[CameraComponent](em, entity1)
	camera1.Zoom = 2.0

	entity2 := em.NewEntity()
	camera2 := ecs.AddComponent[CameraComponent](em, entity2)
	camera2.Zoom = 0.5

	entity3 := em.NewEntity()
	camera3 := ecs.AddComponent[CameraComponent](em, entity3)
	camera3.Zoom = 1.5

	// Test basic filtering
	highZoomFilter := ecs.Where(func(c *CameraComponent) bool {
		return c.Zoom > 1.0
	})

	var highZoomEntities []ecs.EntityID
	for entityID := range ecs.QueryWith(em, highZoomFilter) {
		highZoomEntities = append(highZoomEntities, entityID)
	}

	assert.Len(t, highZoomEntities, 2)
	assert.Contains(t, highZoomEntities, entity1)
	assert.Contains(t, highZoomEntities, entity3)
	assert.NotContains(t, highZoomEntities, entity2)

	// Test that QueryWith without filters returns all entities
	var allCameras []ecs.EntityID
	for entityID := range ecs.QueryWith[CameraComponent](em) {
		allCameras = append(allCameras, entityID)
	}
	assert.Len(t, allCameras, 3)

	// Test complex filter combinations
	lowZoom := ecs.Where(func(c *CameraComponent) bool { return c.Zoom < 0.8 })
	extremeZoom := ecs.Or(highZoomFilter, lowZoom)

	var extremeEntities []ecs.EntityID
	for entityID := range ecs.QueryWith(em, extremeZoom) {
		extremeEntities = append(extremeEntities, entityID)
	}

	assert.Len(t, extremeEntities, 3) // All entities have either high or low zoom
}

func TestSpatialFiltering(t *testing.T) {
	em := ecs.NewEntityManager()

	// Create entities at different positions
	entity1 := em.NewEntity()
	transform1 := ecs.AddComponent[TransformComponent](em, entity1)
	transform1.Position = f64.Vec2{2, 3}

	entity2 := em.NewEntity()
	transform2 := ecs.AddComponent[TransformComponent](em, entity2)
	transform2.Position = f64.Vec2{8, 2}

	entity3 := em.NewEntity()
	transform3 := ecs.AddComponent[TransformComponent](em, entity3)
	transform3.Position = f64.Vec2{1, 1}

	// Test spatial filtering using the helpers
	boundsFilter := ecs.Where(func(t *TransformComponent) bool {
		return ecs.WithinBoundsCheck(t.Position, 0, 0, 5, 5)
	})

	var entitiesInBounds []ecs.EntityID
	for entityID := range ecs.QueryWith(em, boundsFilter) {
		entitiesInBounds = append(entitiesInBounds, entityID)
	}

	assert.Len(t, entitiesInBounds, 2) // entities 1 and 3 are within bounds
	assert.Contains(t, entitiesInBounds, entity1)
	assert.Contains(t, entitiesInBounds, entity3)
	assert.NotContains(t, entitiesInBounds, entity2)

	// Test radius filtering
	radiusFilter := ecs.Where(func(t *TransformComponent) bool {
		return ecs.WithinRadiusCheck(t.Position, 0, 0, 4)
	})

	var entitiesInRadius []ecs.EntityID
	for entityID := range ecs.QueryWith(em, radiusFilter) {
		entitiesInRadius = append(entitiesInRadius, entityID)
	}

	assert.Len(t, entitiesInRadius, 1) // Only entity3 at (1,1) is within radius 4 of origin
	assert.Contains(t, entitiesInRadius, entity3)
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

func BenchmarkFilteredQueries(b *testing.B) {
	em := ecs.NewEntityManager()

	// Create entities with varying zoom levels
	for i := 0; i < 1_000_000; i++ {
		entity := em.NewEntity()
		camera := ecs.AddComponent[CameraComponent](em, entity)
		// Create a distribution: 50% high zoom (>1.0), 50% low zoom (<=1.0)
		if i%2 == 0 {
			camera.Zoom = 2.0
		} else {
			camera.Zoom = 0.5
		}
	}

	// Benchmark different filtering scenarios
	highZoomFilter := ecs.Where(func(c *CameraComponent) bool {
		return c.Zoom > 1.0
	})

	b.Run("QueryWith Filter", func(b *testing.B) {
		for b.Loop() {
			count := 0
			for entityID := range ecs.QueryWith(em, highZoomFilter) {
				count++
				_ = entityID
			}
			if count != 500_000 {
				b.Fatalf("Expected 500k entities, got %d", count)
			}
		}
	})

	b.Run("QueryWith No Filter", func(b *testing.B) {
		for b.Loop() {
			count := 0
			for entityID := range ecs.QueryWith[CameraComponent](em) {
				count++
				_ = entityID
			}
			if count != 1_000_000 {
				b.Fatalf("Expected 1M entities, got %d", count)
			}
		}
	})

	b.Run("Regular Query (baseline)", func(b *testing.B) {
		for b.Loop() {
			count := 0
			for entityID := range ecs.Query[CameraComponent](em) {
				count++
				_ = entityID
			}
			if count != 1_000_000 {
				b.Fatalf("Expected 1M entities, got %d", count)
			}
		}
	})

	// Complex filter benchmark
	complexFilter := ecs.And(
		ecs.Where(func(c *CameraComponent) bool { return c.Zoom > 0.1 }),
		ecs.Where(func(c *CameraComponent) bool { return c.Zoom < 10.0 }),
	)

	b.Run("QueryWith Complex Filter", func(b *testing.B) {
		for b.Loop() {
			count := 0
			for entityID := range ecs.QueryWith(em, complexFilter) {
				count++
				_ = entityID
			}
			if count != 1_000_000 {
				b.Fatalf("Expected 1M entities, got %d", count)
			}
		}
	})
}
