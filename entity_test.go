package ecs_test

import (
	"testing"

	"github.com/jakecoffman/cp"
	"github.com/samix73/game/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TransformComponent struct {
	Position cp.Vector
	Rotation float64
}

func (t *TransformComponent) Init() {
	t.Position = cp.Vector{X: 0, Y: 0}
	t.Rotation = 0
}

func (t *TransformComponent) Reset() {
	t.Position = cp.Vector{X: 0, Y: 0}
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

type VelocityComponent struct {
	X, Y float64
}

func (v *VelocityComponent) Init() {
	v.X = 0
	v.Y = 0
}

func (v *VelocityComponent) Reset() {
	v.X = 0
	v.Y = 0
}

type MassComponent struct {
	Value float64
}

func (m *MassComponent) Init() {
	m.Value = 1.0
}

func (m *MassComponent) Reset() {
	m.Value = 1.0
}

type HealthComponent struct {
	Current, Max int
}

func (h *HealthComponent) Init() {
	h.Current = 100
	h.Max = 100
}

func (h *HealthComponent) Reset() {
	h.Current = 100
	h.Max = 100
}

func NewPlayerEntity(tb testing.TB, em *ecs.EntityManager) ecs.EntityID {
	tb.Helper()

	entityID, err := em.NewEntity()
	require.NoError(tb, err)

	transform, err := ecs.AddComponent[TransformComponent](em, entityID)
	require.NotNil(tb, transform)
	require.NoError(tb, err)

	return entityID
}

func NewCameraEntity(tb testing.TB, em *ecs.EntityManager) ecs.EntityID {
	tb.Helper()

	entityID, err := em.NewEntity()
	require.NoError(tb, err)

	transform, err := ecs.AddComponent[TransformComponent](em, entityID)
	require.NotNil(tb, transform)
	require.NoError(tb, err)

	camera, err := ecs.AddComponent[CameraComponent](em, entityID)
	assert.NotNil(tb, camera)
	assert.NoError(tb, err)

	return entityID
}

func TestEntityCreation(t *testing.T) {
	err := ecs.RegisterComponent[TransformComponent]()
	require.NoError(t, err)
	err = ecs.RegisterComponent[CameraComponent]()
	require.NoError(t, err)

	em := ecs.NewEntityManager()

	player := NewPlayerEntity(t, em)
	assert.Equal(t, player, ecs.EntityID(1))
	camera := NewCameraEntity(t, em)
	assert.Equal(t, camera, ecs.EntityID(2))
	empty, err := em.NewEntity()
	require.NoError(t, err)
	assert.Equal(t, empty, ecs.EntityID(3))
}

func TestQuery(t *testing.T) {
	t.Parallel()

	err := ecs.RegisterComponent[TransformComponent]()
	require.NoError(t, err)
	err = ecs.RegisterComponent[CameraComponent]()
	require.NoError(t, err)
	err = ecs.RegisterComponent[VelocityComponent]()
	require.NoError(t, err)
	err = ecs.RegisterComponent[MassComponent]()
	require.NoError(t, err)
	err = ecs.RegisterComponent[HealthComponent]()
	require.NoError(t, err)
	err = ecs.RegisterComponent[int]()
	require.NoError(t, err)

	em := ecs.NewEntityManager()

	// Create 10 entities with ONLY Transform component
	entityIDs := make([]ecs.EntityID, 10)
	for i := range 10 {
		entityID, err := em.NewEntity()
		require.NoError(t, err)

		_, err = ecs.AddComponent[TransformComponent](em, entityID)
		require.NoError(t, err)

		if i%2 == 0 {
			_, err = ecs.AddComponent[CameraComponent](em, entityID)
			require.NoError(t, err)
			_, err = ecs.AddComponent[int](em, entityID)
			require.NoError(t, err)
		}
		if i%3 == 0 {
			_, err = ecs.AddComponent[VelocityComponent](em, entityID)
			require.NoError(t, err)
		}
		if i%5 == 0 {
			_, err = ecs.AddComponent[MassComponent](em, entityID)
			require.NoError(t, err)
		}
		if i%7 == 0 {
			_, err = ecs.AddComponent[HealthComponent](em, entityID)
			require.NoError(t, err)
		}

		entityIDs[i] = entityID
	}

	t.Run("Query", func(t *testing.T) {
		entities := ecs.Query[TransformComponent](em)
		assert.Len(t, entities, len(entityIDs))
	})

	t.Run("Query2", func(t *testing.T) {
		entities := ecs.Query2[TransformComponent, VelocityComponent](em)
		assert.Len(t, entities, 4)
	})

	t.Run("Query3", func(t *testing.T) {
		entities := ecs.Query3[TransformComponent, CameraComponent, int](em)
		assert.Len(t, entities, 5)
	})
}

func TestQuerySingleComponentFromMultiComponentEntity(t *testing.T) {
	err := ecs.RegisterComponent[TransformComponent]()
	require.NoError(t, err)
	err = ecs.RegisterComponent[CameraComponent]()
	require.NoError(t, err)

	em := ecs.NewEntityManager()

	// Create an entity with both Transform and Camera components
	entityID := NewCameraEntity(t, em)

	// Query for only Transform component (entity has both Transform and Camera)
	transformEntities := ecs.Query[TransformComponent](em)

	// Entity should be found when querying for just Transform, even though it also has Camera
	assert.Contains(t, transformEntities, entityID, "Entity with Transform+Camera should be found when querying for just Transform")
	assert.Equal(t, 1, len(transformEntities), "Should find exactly 1 entity with Transform component")

	// Query for both components
	bothComponents := ecs.Query2[TransformComponent, CameraComponent](em)
	assert.Contains(t, bothComponents, entityID, "Entity should be found when querying for both components")
	assert.Equal(t, 1, len(bothComponents), "Should find exactly 1 entity with both components")

	// Create an entity with only Transform
	playerID := NewPlayerEntity(t, em)

	// Query for Transform should now return both entities
	transformEntities = ecs.Query[TransformComponent](em)
	assert.Contains(t, transformEntities, entityID, "Camera entity should still be in Transform query")
	assert.Contains(t, transformEntities, playerID, "Player entity should be in Transform query")
	assert.Equal(t, 2, len(transformEntities), "Should find 2 entities with Transform component")

	// Query for both components should only return the camera entity
	bothComponents = ecs.Query2[TransformComponent, CameraComponent](em)
	assert.Contains(t, bothComponents, entityID, "Only camera entity has both components")
	assert.NotContains(t, bothComponents, playerID, "Player entity should not be in query for both components")
	assert.Equal(t, 1, len(bothComponents), "Should find exactly 1 entity with both components")
}

// there is a case where adding a component while iterating moves the entity to a different archetype
// and the iteration might miss it or double count it. Need to fix this. This test is trying to capture that.
//
// The edge case occurs because:
//  1. When we add a component to an entity, it's removed from its current archetype
//  2. The removal uses swap-and-pop: the last entity is moved to the removed entity's position
//  3. If we're iterating by index and the iterator reads the slice length dynamically,
//     the swapped entity at the current position may be skipped
func TestAddComponentWhileIterating(t *testing.T) {
	err := ecs.RegisterComponent[TransformComponent]()
	require.NoError(t, err)
	err = ecs.RegisterComponent[CameraComponent]()
	require.NoError(t, err)
	err = ecs.RegisterComponent[VelocityComponent]()
	require.NoError(t, err)
	err = ecs.RegisterComponent[MassComponent]()
	require.NoError(t, err)
	err = ecs.RegisterComponent[HealthComponent]()
	require.NoError(t, err)

	em := ecs.NewEntityManager()

	// Create 10 entities with ONLY Transform component
	entityIDs := make([]ecs.EntityID, 10)
	visitedEntities := make(map[ecs.EntityID]int)
	for i := range 10 {
		entityID, err := em.NewEntity()
		require.NoError(t, err)

		_, err = ecs.AddComponent[TransformComponent](em, entityID)
		require.NoError(t, err)

		if i%2 == 0 {
			_, err = ecs.AddComponent[CameraComponent](em, entityID)
			require.NoError(t, err)
			_, err = ecs.AddComponent[HealthComponent](em, entityID)
			require.NoError(t, err)
			_, err = ecs.AddComponent[VelocityComponent](em, entityID)
			require.NoError(t, err)
		}

		entityIDs[i] = entityID
		visitedEntities[entityID] = 0
	}

	// Track which entities we visit during iteration and how many times
	var visitOrder []ecs.EntityID

	for _, entity := range ecs.Query[TransformComponent](em) {
		visitedEntities[entity]++
		visitOrder = append(visitOrder, entity)

		if !ecs.HasComponent[CameraComponent](em, entity) {
			_, err := ecs.AddComponent[CameraComponent](em, entity)
			require.NoError(t, err)
		}

		if ecs.HasComponent[HealthComponent](em, entity) {
			err := ecs.RemoveComponent[HealthComponent](em, entity)
			require.NoError(t, err)
		}

		if ecs.HasComponent[VelocityComponent](em, entity) {
			err := ecs.RemoveComponent[VelocityComponent](em, entity)
			require.NoError(t, err)
		}
	}

	t.Logf("Visit order: %v", visitOrder)
	t.Logf("Visit counts: %v", visitedEntities)

	// Assert that all 10 entities were visited exactly once
	// The LAST entity (10) should be SKIPPED because it was swapped into position 0
	// which was already processed
	missedEntities := []ecs.EntityID{}
	doubleCountedEntities := []ecs.EntityID{}

	for _, entityID := range entityIDs {
		count := visitedEntities[entityID]
		if count == 0 {
			missedEntities = append(missedEntities, entityID)
		} else if count > 1 {
			doubleCountedEntities = append(doubleCountedEntities, entityID)
		}
	}

	t.Logf("Missed entities: %v", missedEntities)
	t.Logf("Double-counted entities: %v", doubleCountedEntities)

	// This assertion should FAIL
	assert.Empty(t, missedEntities, "No entities should be missed during iteration")
	assert.Empty(t, doubleCountedEntities, "No entities should be double-counted during iteration")
	assert.Equal(t, 10, len(visitedEntities), "Should have visited exactly 10 unique entities")
}

func TestModifyOtherEntityWhileIterating(t *testing.T) {
	err := ecs.RegisterComponent[TransformComponent]()
	require.NoError(t, err)

	em := ecs.NewEntityManager()

	// Create 5 entities
	entities := make([]ecs.EntityID, 5)
	for i := range 5 {
		e, err := em.NewEntity()
		require.NoError(t, err)
		_, err = ecs.AddComponent[TransformComponent](em, e)
		require.NoError(t, err)
		entities[i] = e
	}

	visited := make(map[ecs.EntityID]int)

	// Iterate
	// Expectation with 5 entities: [0, 1, 2, 3, 4]
	// Backwards iteration visits: 4, 3, 2, 1, 0
	for _, e := range ecs.Query[TransformComponent](em) {
		visited[e]++

		// When we visit the last entity (entities[4]), remove the first entity (entities[0])
		if e == entities[4] {
			// Only try to remove if it still has the component (avoid error on double-visit)
			if ecs.HasComponent[TransformComponent](em, entities[0]) {
				err := ecs.RemoveComponent[TransformComponent](em, entities[0])
				require.NoError(t, err)
			}
		}
	}

	// Assertions
	// Entity 4 should be visited exactly once. If it's visited twice, the test fails.
	assert.Equal(t, 1, visited[entities[4]], "Entity 4 should be visited exactly once")

	assert.Equal(t, 1, visited[entities[0]])
}

func TestModifyingComponentValue(t *testing.T) {
	err := ecs.RegisterComponent[TransformComponent]()
	require.NoError(t, err)

	em := ecs.NewEntityManager()

	// Create 5 entities
	entities := make([]ecs.EntityID, 5)
	for i := range 5 {
		e, err := em.NewEntity()
		require.NoError(t, err)
		transform, err := ecs.AddComponent[TransformComponent](em, e)
		require.NoError(t, err)

		transform.Position = cp.Vector{X: 24, Y: 54}

		entities[i] = e
	}

	for _, e := range ecs.Query[TransformComponent](em) {
		transform := ecs.MustGetComponent[TransformComponent](em, e)
		assert.Equal(t, 24, transform.Position.X)
		assert.Equal(t, 54, transform.Position.Y)
	}
}

func BenchmarkGetComponent(b *testing.B) {
	err := ecs.RegisterComponent[TransformComponent]()
	require.NoError(b, err)
	err = ecs.RegisterComponent[CameraComponent]()
	require.NoError(b, err)

	em := ecs.NewEntityManager()

	// Create a set of entities with Transform components
	for range 500_000 {
		NewPlayerEntity(b, em)
	}

	for range 500_000 {
		NewCameraEntity(b, em)
	}

	for range 1000 {
		_, err := em.NewEntity()
		require.NoError(b, err)
	}

	entityIDs := ecs.Query[TransformComponent](em)

	b.Run("GetComponent single entity", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			ecs.GetComponent[TransformComponent](em, entityIDs[0])
		}
	})

	b.Run("GetComponent all entities", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			for _, entityID := range entityIDs {
				ecs.GetComponent[TransformComponent](em, entityID)
			}
		}
	})
}

func BenchmarkQueryEntities(b *testing.B) {
	err := ecs.RegisterComponent[TransformComponent]()
	require.NoError(b, err)
	err = ecs.RegisterComponent[CameraComponent]()
	require.NoError(b, err)
	err = ecs.RegisterComponent[HealthComponent]()
	require.NoError(b, err)
	err = ecs.RegisterComponent[VelocityComponent]()
	require.NoError(b, err)
	err = ecs.RegisterComponent[int]()
	require.NoError(b, err)
	err = ecs.RegisterComponent[string]()
	require.NoError(b, err)

	em := ecs.NewEntityManager()

	// Create a set of entities with Transform components
	for i := range 1_000_000 {
		entityID, err := em.NewEntity()
		require.NoError(b, err)

		switch i % 6 {
		case 0:
			_, err = ecs.AddComponent[TransformComponent](em, entityID)
			require.NoError(b, err)
			_, err = ecs.AddComponent[int](em, entityID)
			require.NoError(b, err)
		case 1:
			_, err = ecs.AddComponent[TransformComponent](em, entityID)
			require.NoError(b, err)
			_, err = ecs.AddComponent[string](em, entityID)
			require.NoError(b, err)
		case 2:
			_, err = ecs.AddComponent[CameraComponent](em, entityID)
			require.NoError(b, err)
			_, err = ecs.AddComponent[HealthComponent](em, entityID)
			require.NoError(b, err)
		case 3:
			_, err = ecs.AddComponent[TransformComponent](em, entityID)
			require.NoError(b, err)
			_, err = ecs.AddComponent[int](em, entityID)
			require.NoError(b, err)
		case 4:
			_, err = ecs.AddComponent[VelocityComponent](em, entityID)
			require.NoError(b, err)
			_, err = ecs.AddComponent[string](em, entityID)
			require.NoError(b, err)
		case 5:
			_, err = ecs.AddComponent[TransformComponent](em, entityID)
			require.NoError(b, err)
			_, err = ecs.AddComponent[HealthComponent](em, entityID)
			require.NoError(b, err)
			_, err = ecs.AddComponent[CameraComponent](em, entityID)
			require.NoError(b, err)
		}
	}

	b.Run("Query", func(b *testing.B) {
		for b.Loop() {
			ecs.Query[TransformComponent](em)
		}
	})

	b.Run("Query2", func(b *testing.B) {
		for b.Loop() {
			ecs.Query2[TransformComponent, int](em)
		}
	})

	b.Run("Query3", func(b *testing.B) {
		for b.Loop() {
			ecs.Query3[TransformComponent, CameraComponent, HealthComponent](em)
		}
	})
}
