package ecs

import (
	"fmt"
	"iter"
	"reflect"
	"slices"
)

type EntityID = ID

type EntityManager struct {
	entities                  map[EntityID]struct{}
	componentContainers       map[reflect.Type]*ComponentContainer
	entityComponentSignatures map[EntityID]map[reflect.Type]struct{}
}

func NewEntityManager() *EntityManager {
	return &EntityManager{
		entities:                  make(map[EntityID]struct{}),
		componentContainers:       make(map[reflect.Type]*ComponentContainer),
		entityComponentSignatures: make(map[EntityID]map[reflect.Type]struct{}),
	}
}

func (em *EntityManager) NewEntity() EntityID {
	id := NextID()
	em.entities[id] = struct{}{}
	em.entityComponentSignatures[id] = make(map[reflect.Type]struct{})

	return id
}

func (em *EntityManager) HasComponent(entityID EntityID, componentType reflect.Type) bool {
	if _, exists := em.entities[entityID]; !exists {
		return false
	}

	if _, exists := em.entityComponentSignatures[entityID][componentType]; !exists {
		return false
	}

	return true
}

func (em *EntityManager) Remove(entityID EntityID) {
	if _, exists := em.entities[entityID]; !exists {
		return
	}

	for componentType := range em.entityComponentSignatures[entityID] {
		if container, exists := em.componentContainers[componentType]; exists {
			container.Remove(entityID)
		}
	}

	delete(em.entityComponentSignatures, entityID)
	delete(em.entities, entityID)
}

func (em *EntityManager) RemoveComponent(entityID EntityID, componentType reflect.Type) {
	if _, exists := em.entities[entityID]; !exists {
		return
	}

	if _, exists := em.entityComponentSignatures[entityID][componentType]; !exists {
		return
	}

	container, exists := em.componentContainers[componentType]
	if !exists {
		return
	}

	container.Remove(entityID)
	delete(em.entityComponentSignatures[entityID], componentType)
}

// Query returns a sequence of EntityIDs that match the specified component types.
func (em *EntityManager) Query(componentTypes ...reflect.Type) iter.Seq[EntityID] {
	zeroIter := func(yield func(EntityID) bool) {}

	if len(componentTypes) == 0 {
		return zeroIter
	}

	// If only one component type is specified, return entities with that component
	if len(componentTypes) == 1 {
		componentContainer, exists := em.componentContainers[componentTypes[0]]
		if !exists {
			return zeroIter
		}

		return componentContainer.Entities()
	}

	// Pre-check: if any component type doesn't exist, return empty iterator
	containers := make([]*ComponentContainer, len(componentTypes))
	for i, componentType := range componentTypes {
		container, exists := em.componentContainers[componentType]
		if !exists {
			return zeroIter
		}
		containers[i] = container
	}

	// Find the container with the smallest number of entities to start with
	// This reduces the number of entities we need to check
	smallestIdx := 0
	smallestCount := containers[0].Count()

	// Check if the smallest container is empty
	if smallestCount == 0 {
		return zeroIter
	}

	for i := 1; i < len(containers); i++ {
		count := containers[i].Count()
		if count == 0 {
			// If any container has zero entities, we can return immediately
			return zeroIter
		} else if count < smallestCount {
			smallestCount = count
			smallestIdx = i
		}
	}

	// Start with the smallest set and filter iteratively
	smallestContainer := containers[smallestIdx]
	otherContainers := make([]*ComponentContainer, 0, len(containers)-1)
	for i, container := range containers {
		if i != smallestIdx {
			otherContainers = append(otherContainers, container)
		}
	}

	return func(yield func(EntityID) bool) {
		for entityID := range smallestContainer.Entities() {
			// Check if this entity exists in all other containers
			hasAllComponents := true
			for _, container := range otherContainers {
				if _, exists := container.Get(entityID); !exists {
					hasAllComponents = false
					break
				}
			}

			if hasAllComponents {
				if !yield(entityID) {
					break
				}
			}
		}
	}
}

func (em *EntityManager) Teardown() {
	for _, container := range em.componentContainers {
		container.Teardown()
	}

	em.entities = nil
	em.entityComponentSignatures = nil
	em.componentContainers = nil
}

func AddComponent[C any](em *EntityManager, entityID EntityID) *C {
	if _, exists := em.entities[entityID]; !exists {
		return nil
	}

	var zero C
	// Check if the component type is already registered for this entity
	componentType := reflect.TypeOf(zero)
	if _, exists := em.entityComponentSignatures[entityID][componentType]; exists {
		return MustGetComponent[C](em, entityID)
	}

	container, exists := em.componentContainers[componentType]
	if !exists {
		container = NewComponentContainer(func() any {
			var c C
			return &c
		})
		em.componentContainers[componentType] = container
	}

	component := container.Add(entityID)
	em.entityComponentSignatures[entityID][componentType] = struct{}{}

	return component.(*C)
}

func RemoveComponent[C any](em *EntityManager, entityID EntityID) {
	var zero C
	em.RemoveComponent(entityID, reflect.TypeOf(zero))
}

func Query[C any](em *EntityManager) iter.Seq[EntityID] {
	var zero C
	return em.Query(reflect.TypeOf(zero))
}

func Query2[C1, C2 any](em *EntityManager) iter.Seq[EntityID] {
	var zero1 C1
	var zero2 C2
	return em.Query(reflect.TypeOf(zero1), reflect.TypeOf(zero2))
}

func Query3[C1, C2, C3 any](em *EntityManager) iter.Seq[EntityID] {
	var zero1 C1
	var zero2 C2
	var zero3 C3
	return em.Query(reflect.TypeOf(zero1), reflect.TypeOf(zero2), reflect.TypeOf(zero3))
}

func HasComponent[C any](em *EntityManager, entityID EntityID) bool {
	var zero C
	return em.HasComponent(entityID, reflect.TypeOf(zero))
}

func GetComponent[C any](em *EntityManager, entityID EntityID) (*C, bool) {
	var zero C
	componentType := reflect.TypeOf(zero)

	if _, exists := em.entities[entityID]; !exists {
		return nil, false
	}

	if _, exists := em.entityComponentSignatures[entityID][componentType]; !exists {
		return nil, false
	}

	container, exists := em.componentContainers[componentType]
	if !exists {
		return nil, false
	}

	component, exists := container.Get(entityID)
	if !exists {
		return nil, false
	}

	return component.(*C), true
}

func MustGetComponent[C any](em *EntityManager, entityID EntityID) *C {
	component, exists := GetComponent[C](em, entityID)
	if !exists {
		var zero C
		panic(fmt.Sprintf("Entity %d does not have component of type %s", entityID, reflect.TypeOf(zero).Name()))
	}

	return component
}

func First(iterator iter.Seq[EntityID]) (EntityID, bool) {
	for item := range iterator {
		return item, true
	}

	return UndefinedID, false
}

func Count(it iter.Seq[EntityID]) int {
	return len(slices.Collect(it))
}
