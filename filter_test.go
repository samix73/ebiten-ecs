package ecs_test

import (
	"testing"

	ecs "github.com/samix73/ebiten-ecs"
	"github.com/stretchr/testify/assert"
	"golang.org/x/image/math/f64"
)

func TestFilterHelpers(t *testing.T) {
	// Test Where filter
	zoomFilter := ecs.Where(func(c *CameraComponent) bool {
		return c.Zoom > 1.0
	})

	camera1 := &CameraComponent{Zoom: 1.5}
	camera2 := &CameraComponent{Zoom: 0.5}

	assert.True(t, zoomFilter(camera1))
	assert.False(t, zoomFilter(camera2))

	// Test And filter
	highZoom := ecs.Where(func(c *CameraComponent) bool { return c.Zoom > 1.0 })
	veryHighZoom := ecs.Where(func(c *CameraComponent) bool { return c.Zoom < 3.0 })
	combinedFilter := ecs.And(highZoom, veryHighZoom)

	camera3 := &CameraComponent{Zoom: 2.0} // Between 1.0 and 3.0
	camera4 := &CameraComponent{Zoom: 4.0} // Above 3.0

	assert.True(t, combinedFilter(camera3))
	assert.False(t, combinedFilter(camera4))

	// Test Or filter
	lowZoom := ecs.Where(func(c *CameraComponent) bool { return c.Zoom < 0.5 })
	extremeZoom := ecs.Or(highZoom, lowZoom)

	camera5 := &CameraComponent{Zoom: 0.3} // Low zoom
	camera6 := &CameraComponent{Zoom: 0.8} // Medium zoom

	assert.True(t, extremeZoom(camera5)) // Low zoom passes
	assert.False(t, extremeZoom(camera6)) // Medium zoom fails both conditions

	// Test Not filter
	notHighZoom := ecs.Not(highZoom)

	assert.False(t, notHighZoom(camera1)) // High zoom, so Not returns false
	assert.True(t, notHighZoom(camera2))  // Low zoom, so Not returns true
}

func TestSpatialFilters(t *testing.T) {
	// Test WithinBoundsCheck helper function
	pos1 := f64.Vec2{5, 5}   // Inside bounds
	pos2 := f64.Vec2{15, 5}  // Outside bounds

	assert.True(t, ecs.WithinBoundsCheck(pos1, 0, 0, 10, 10))
	assert.False(t, ecs.WithinBoundsCheck(pos2, 0, 0, 10, 10))

	// Test WithinRadiusCheck helper function  
	pos3 := f64.Vec2{3, 4}  // Distance = 5, exactly on boundary
	pos4 := f64.Vec2{4, 5}  // Distance > 5, outside

	assert.True(t, ecs.WithinRadiusCheck(pos3, 0, 0, 5))
	assert.False(t, ecs.WithinRadiusCheck(pos4, 0, 0, 5))
}

func TestQueryWith(t *testing.T) {
	em := ecs.NewEntityManager()

	// Create test entities with different zoom levels
	entity1 := em.NewEntity()
	camera1 := ecs.AddComponent[CameraComponent](em, entity1)
	camera1.Zoom = 2.0

	entity2 := em.NewEntity()
	camera2 := ecs.AddComponent[CameraComponent](em, entity2)
	camera2.Zoom = 0.5

	entity3 := em.NewEntity()
	camera3 := ecs.AddComponent[CameraComponent](em, entity3)
	camera3.Zoom = 1.5

	// Test QueryWith with high zoom filter
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

	// Test QueryWith with no filters (should return all)
	var allEntities []ecs.EntityID
	for entityID := range ecs.QueryWith[CameraComponent](em) {
		allEntities = append(allEntities, entityID)
	}

	assert.Len(t, allEntities, 3)
}

func TestQueryWith2(t *testing.T) {
	em := ecs.NewEntityManager()

	// Create entities with both Transform and Camera components
	entity1 := em.NewEntity()
	ecs.AddComponent[TransformComponent](em, entity1)
	camera1 := ecs.AddComponent[CameraComponent](em, entity1)
	camera1.Zoom = 2.0

	entity2 := em.NewEntity()
	ecs.AddComponent[TransformComponent](em, entity2)
	camera2 := ecs.AddComponent[CameraComponent](em, entity2)
	camera2.Zoom = 0.5

	// Create entity with only Transform (should not appear in Query2)
	entity3 := em.NewEntity()
	ecs.AddComponent[TransformComponent](em, entity3)

	// Test QueryWith2 with camera filter
	highZoomFilter := ecs.Where(func(c *CameraComponent) bool {
		return c.Zoom > 1.0
	})

	var filteredEntities []ecs.EntityID
	for entityID := range ecs.QueryWith2[CameraComponent, TransformComponent](em, highZoomFilter) {
		filteredEntities = append(filteredEntities, entityID)
	}

	assert.Len(t, filteredEntities, 1)
	assert.Contains(t, filteredEntities, entity1)
	assert.NotContains(t, filteredEntities, entity2) // Low zoom
	assert.NotContains(t, filteredEntities, entity3) // No camera component
}

func TestQueryWith3(t *testing.T) {
	em := ecs.NewEntityManager()

	// Create a third component type for testing
	type VelocityComponent struct {
		X, Y float64
	}

	// Create entities with all three components
	entity1 := em.NewEntity()
	ecs.AddComponent[TransformComponent](em, entity1)
	camera1 := ecs.AddComponent[CameraComponent](em, entity1)
	camera1.Zoom = 2.0
	ecs.AddComponent[VelocityComponent](em, entity1)

	entity2 := em.NewEntity()
	ecs.AddComponent[TransformComponent](em, entity2)
	camera2 := ecs.AddComponent[CameraComponent](em, entity2)
	camera2.Zoom = 0.5
	ecs.AddComponent[VelocityComponent](em, entity2)

	// Test QueryWith3 with camera filter
	highZoomFilter := ecs.Where(func(c *CameraComponent) bool {
		return c.Zoom > 1.0
	})

	var filteredEntities []ecs.EntityID
	for entityID := range ecs.QueryWith3[CameraComponent, TransformComponent, VelocityComponent](em, highZoomFilter) {
		filteredEntities = append(filteredEntities, entityID)
	}

	assert.Len(t, filteredEntities, 1)
	assert.Contains(t, filteredEntities, entity1)
	assert.NotContains(t, filteredEntities, entity2) // Low zoom
}

func TestComplexFilterCombinations(t *testing.T) {
	em := ecs.NewEntityManager()

	// Create entities with various zoom levels
	testCases := []float64{0.2, 0.8, 1.2, 2.5, 4.0}
	entities := make([]ecs.EntityID, len(testCases))

	for i, zoom := range testCases {
		entity := em.NewEntity()
		camera := ecs.AddComponent[CameraComponent](em, entity)
		camera.Zoom = zoom
		entities[i] = entity
	}

	// Create complex filter: zoom < 0.5 OR zoom > 2.0 (extreme values)
	lowZoom := ecs.Where(func(c *CameraComponent) bool { return c.Zoom < 0.5 })
	highZoom := ecs.Where(func(c *CameraComponent) bool { return c.Zoom > 2.0 })
	extremeZoom := ecs.Or(lowZoom, highZoom)

	var extremeEntities []ecs.EntityID
	for entityID := range ecs.QueryWith(em, extremeZoom) {
		extremeEntities = append(extremeEntities, entityID)
	}

	// Should match entities with zoom 0.2, 2.5, and 4.0
	assert.Len(t, extremeEntities, 3)
	assert.Contains(t, extremeEntities, entities[0]) // 0.2
	assert.Contains(t, extremeEntities, entities[3]) // 2.5
	assert.Contains(t, extremeEntities, entities[4]) // 4.0
}

func TestSpatialFilterIntegration(t *testing.T) {
	em := ecs.NewEntityManager()

	// Create entities at different positions
	positions := []f64.Vec2{
		{2, 3},   // Inside 5x5 bounds
		{8, 2},   // Outside 5x5 bounds
		{1, 1},   // Inside 5x5 bounds
		{10, 10}, // Far outside
	}

	entities := make([]ecs.EntityID, len(positions))
	for i, pos := range positions {
		entity := em.NewEntity()
		transform := ecs.AddComponent[TransformComponent](em, entity)
		transform.Position = pos
		entities[i] = entity
	}

	// Test spatial filtering using the spatial helpers
	boundsFilter := ecs.Where(func(t *TransformComponent) bool {
		return ecs.WithinBoundsCheck(t.Position, 0, 0, 5, 5)
	})

	var entitiesInBounds []ecs.EntityID
	for entityID := range ecs.QueryWith(em, boundsFilter) {
		entitiesInBounds = append(entitiesInBounds, entityID)
	}

	// Should match entities at positions (2,3) and (1,1)
	assert.Len(t, entitiesInBounds, 2)
	assert.Contains(t, entitiesInBounds, entities[0]) // (2,3)
	assert.Contains(t, entitiesInBounds, entities[2]) // (1,1)
}