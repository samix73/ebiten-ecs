package ecs_test

import (
	"testing"

	"github.com/samix73/game/ecs"
	"github.com/stretchr/testify/assert"
)

func highZoomFilter(c *CameraComponent) bool {
	return c.Zoom < 1.0
}

func lowZoomFilter(c *CameraComponent) bool {
	return c.Zoom > 0.5
}

func TestAnd(t *testing.T) {
	camera1 := &CameraComponent{Zoom: 1.5}
	camera2 := &CameraComponent{Zoom: 0.4}
	camera3 := &CameraComponent{Zoom: 0.6}

	assert.False(t, highZoomFilter(camera1))
	assert.True(t, highZoomFilter(camera2))
	assert.True(t, highZoomFilter(camera3))

	assert.True(t, lowZoomFilter(camera1))
	assert.False(t, lowZoomFilter(camera2))
	assert.True(t, lowZoomFilter(camera3))

	assert.False(t, ecs.And(highZoomFilter, lowZoomFilter)(camera1))
	assert.False(t, ecs.And(highZoomFilter, lowZoomFilter)(camera2))
	assert.True(t, ecs.And(highZoomFilter, lowZoomFilter)(camera3))
}

func TestOr(t *testing.T) {
	camera1 := &CameraComponent{Zoom: 1.5}
	camera2 := &CameraComponent{Zoom: 0.4}
	camera3 := &CameraComponent{Zoom: 0.6}

	assert.False(t, highZoomFilter(camera1))
	assert.True(t, highZoomFilter(camera2))
	assert.True(t, highZoomFilter(camera3))

	assert.True(t, lowZoomFilter(camera1))
	assert.False(t, lowZoomFilter(camera2))
	assert.True(t, lowZoomFilter(camera3))

	assert.True(t, ecs.Or(highZoomFilter, lowZoomFilter)(camera1))
	assert.True(t, ecs.Or(highZoomFilter, lowZoomFilter)(camera2))
	assert.True(t, ecs.Or(highZoomFilter, lowZoomFilter)(camera3))
}

func TestNot(t *testing.T) {
	camera1 := &CameraComponent{Zoom: 1.5}
	camera2 := &CameraComponent{Zoom: 0.4}
	camera3 := &CameraComponent{Zoom: 0.6}

	assert.True(t, ecs.Not(highZoomFilter)(camera1))
	assert.False(t, ecs.Not(highZoomFilter)(camera2))
	assert.False(t, ecs.Not(highZoomFilter)(camera3))

	assert.False(t, ecs.Not(lowZoomFilter)(camera1))
	assert.True(t, ecs.Not(lowZoomFilter)(camera2))
	assert.False(t, ecs.Not(lowZoomFilter)(camera3))
}
