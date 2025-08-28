package ecs

import (
	"iter"
	"sync"
)

type Component interface {
	Reset()
}

type ComponentContainer struct {
	pool sync.Pool

	components []any
	entityIDs  []EntityID

	componentLookupMap map[EntityID]int
}

func NewComponentContainer(newFn func() any) *ComponentContainer {
	return &ComponentContainer{
		pool: sync.Pool{New: func() any { return newFn() }},

		components:         make([]any, 0, 1024),
		entityIDs:          make([]EntityID, 0, 1024),
		componentLookupMap: make(map[EntityID]int),
	}
}

func (c *ComponentContainer) Add(entityID EntityID) any {
	if _, ok := c.componentLookupMap[entityID]; ok {
		return nil
	}

	component := c.pool.Get()

	if initable, ok := component.(interface{ Init() }); ok {
		initable.Init()
	}

	c.components = append(c.components, component)
	c.entityIDs = append(c.entityIDs, entityID)
	c.componentLookupMap[entityID] = len(c.components) - 1

	return component
}

func (c *ComponentContainer) Remove(entityID EntityID) {
	indexToRemove, ok := c.componentLookupMap[entityID]
	if !ok {
		return
	}

	componentToRemove := c.components[indexToRemove]

	lastIndex := len(c.entityIDs) - 1
	if lastIndex != indexToRemove {
		c.components[indexToRemove] = c.components[lastIndex]
		c.entityIDs[indexToRemove] = c.entityIDs[lastIndex]

		c.componentLookupMap[c.entityIDs[indexToRemove]] = indexToRemove
	}

	c.components = c.components[:lastIndex]
	c.entityIDs = c.entityIDs[:lastIndex]

	delete(c.componentLookupMap, entityID)

	if typedComponent, ok := componentToRemove.(Component); ok {
		typedComponent.Reset()
	}
	c.pool.Put(componentToRemove)
}

func (c *ComponentContainer) All() iter.Seq2[EntityID, any] {
	return func(yield func(EntityID, any) bool) {
		for i, entityID := range c.entityIDs {
			if !yield(entityID, c.components[i]) {
				break
			}
		}
	}
}

func (c *ComponentContainer) Get(entityID EntityID) (any, bool) {
	index, ok := c.componentLookupMap[entityID]
	if !ok {
		return nil, false
	}

	return c.components[index], true
}

func (c *ComponentContainer) Count() int {
	return len(c.components)
}

func (c *ComponentContainer) Entities() iter.Seq[EntityID] {
	return func(yield func(EntityID) bool) {
		for _, entityID := range c.entityIDs {
			if !yield(entityID) {
				break
			}
		}
	}
}

func (c *ComponentContainer) Components() iter.Seq[any] {
	return func(yield func(any) bool) {
		for _, component := range c.components {
			if !yield(component) {
				break
			}
		}
	}
}

func (c *ComponentContainer) Teardown() {
	c.components = nil
	c.entityIDs = nil
	c.componentLookupMap = nil
	c.pool = sync.Pool{}
}
