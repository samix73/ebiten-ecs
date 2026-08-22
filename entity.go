package ecs

import (
	"fmt"
	"reflect"
	"sync"
)

type EntityID = uint64

type EntityManager struct {
	nextID          EntityID
	archetypes      []*Archetype
	entityArchetype map[EntityID]*Archetype
	componentPools  map[reflect.Type]*sync.Pool
}

func NewEntityManager() *EntityManager {
	return &EntityManager{
		nextID:          1,
		archetypes:      make([]*Archetype, 0),
		entityArchetype: make(map[EntityID]*Archetype),
		componentPools:  make(map[reflect.Type]*sync.Pool),
	}
}

func (em *EntityManager) NewEntity(components ...any) (EntityID, error) {
	id := em.nextID
	em.nextID++

	var signature Bitmask
	componentData := make(map[ComponentID]any, len(components))
	for _, component := range components {
		componentID, ok := getComponentID(reflect.TypeOf(component))
		if !ok {
			return 0, fmt.Errorf("ecs.EntityManager.NewEntity: component %T not registered, call RegisterComponent first", component)
		}
		signature.Set(componentID)
		componentData[componentID] = component
	}

	archetype := em.getOrCreateArchetype(signature)
	if err := archetype.AddEntity(id, componentData); err != nil {
		return 0, fmt.Errorf("ecs.EntityManager.NewEntity: failed to add entity %w", err)
	}
	em.entityArchetype[id] = archetype

	return id, nil
}

func (em *EntityManager) getOrCreateArchetype(signature Bitmask) *Archetype {
	for _, archetype := range em.archetypes {
		if archetype.SignatureMatches(signature) {
			return archetype
		}
	}

	archetype := NewArchetype(signature)
	em.archetypes = append(em.archetypes, archetype)

	return archetype
}

func (em *EntityManager) HasComponent[C any](entityID EntityID) bool {
	componentType := reflect.TypeFor[C]()
	componentID, ok := getComponentID(componentType)
	if !ok {
		return false
	}

	archetype, exists := em.entityArchetype[entityID]
	if !exists {
		return false
	}

	return archetype.HasComponent(entityID, componentID)
}

func (em *EntityManager) Remove(entityID EntityID) error {
	archetype, exists := em.entityArchetype[entityID]
	if !exists {
		return fmt.Errorf("ecs.EntityManager.Remove: entity %d does not exist", entityID)
	}

	componentData, err := archetype.RemoveEntity(entityID)
	if err != nil {
		return fmt.Errorf("ecs.EntityManager.Remove: %w", err)
	}

	// Return components to pools
	for componentID, component := range componentData {
		pool, ok := getComponentPool(componentID)
		if !ok {
			return fmt.Errorf("ecs.EntityManager.Remove: component %d not registered", componentID)
		}
		pool.Put(component)
	}

	delete(em.entityArchetype, entityID)

	return nil
}

func (em *EntityManager) RemoveComponent[C any](entityID EntityID) error {
	componentType := reflect.TypeFor[C]()
	componentID, ok := getComponentID(componentType)
	if !ok {
		return fmt.Errorf("ecs.RemoveComponent: component %s not registered", componentType.Name())
	}

	archetype, exists := em.entityArchetype[entityID]
	if !exists {
		return fmt.Errorf("ecs.EntityManager.RemoveComponent: entity %d does not exist", entityID)
	}

	if !archetype.HasComponent(entityID, componentID) {
		return fmt.Errorf("ecs.EntityManager.RemoveComponent: entity %d does not have component %d", entityID, componentID)
	}

	componentData, err := archetype.RemoveEntity(entityID)
	if err != nil {
		return fmt.Errorf("ecs.EntityManager.RemoveComponent: %w", err)
	}

	// Get the component to return to pool
	removedComponent := componentData[componentID]

	// Remove the specified component type
	delete(componentData, componentID)

	// Return removed component to pool
	if resettable, ok := removedComponent.(Component); ok {
		resettable.Reset()
	}

	pool, ok := getComponentPool(componentID)
	if !ok {
		return fmt.Errorf("ecs.EntityManager.RemoveComponent: component %d not registered", componentID)
	}
	pool.Put(removedComponent)

	// Calculate new signature
	var newSignature Bitmask
	for componentID := range componentData {
		newSignature.Set(componentID)
	}

	// Move entity to new archetype
	newArchetype := em.getOrCreateArchetype(newSignature)
	if err := newArchetype.AddEntity(entityID, componentData); err != nil {
		return fmt.Errorf("ecs.EntityManager.RemoveComponent: %w", err)
	}
	em.entityArchetype[entityID] = newArchetype

	return nil
}

func (em *EntityManager) AddComponent[C any](entityID EntityID) (*C, error) {
	componentType := reflect.TypeFor[C]()
	componentID, ok := getComponentID(componentType)
	if !ok {
		return nil, fmt.Errorf("ecs.AddComponent: component %s not registered", componentType.Name())
	}

	archetype, exists := em.entityArchetype[entityID]
	if !exists {
		return nil, fmt.Errorf("entity %d does not exist", entityID)
	}

	if archetype.HasComponent(entityID, componentID) {
		return nil, fmt.Errorf("ecs.EntityManager.AddComponent: entity %d already has component %d", entityID, componentID)
	}

	// Get current component data
	componentData, err := archetype.RemoveEntity(entityID)
	if err != nil {
		return nil, fmt.Errorf("ecs.EntityManager.AddComponent: failed to remove entity %d: %w", entityID, err)
	}

	pool, ok := getComponentPool(componentID)
	if !ok {
		return nil, fmt.Errorf("ecs.EntityManager.AddComponent: component %d not registered", componentID)
	}

	component := pool.Get()

	if resettable, ok := any(component).(Component); ok {
		resettable.Init()
	}

	// Add new component
	componentData[componentID] = component

	// Calculate new signature
	var newSignature Bitmask
	for componentID := range componentData {
		newSignature.Set(componentID)
	}

	// Move entity to new archetype
	newArchetype := em.getOrCreateArchetype(newSignature)
	if err := newArchetype.AddEntity(entityID, componentData); err != nil {
		return nil, fmt.Errorf("ecs.EntityManager.AddComponent: %w", err)
	}
	em.entityArchetype[entityID] = newArchetype

	return component.(*C), nil
}

func (em *EntityManager) query(queryMask Bitmask) []EntityID {
	entities := make([]EntityID, 0)

	for i := range em.archetypes {
		if em.archetypes[i].MatchesQuery(queryMask) {
			entities = append(entities, em.archetypes[i].Entities()...)
		}
	}

	return entities
}

func (em *EntityManager) Query[C any]() []EntityID {
	queryMask, ok := getComponentsBitmask([]reflect.Type{reflect.TypeFor[C]()})
	if !ok {
		return []EntityID{}
	}

	return em.query(queryMask)
}

func (em *EntityManager) Query2[C1, C2 any]() []EntityID {
	queryMask, ok := getComponentsBitmask([]reflect.Type{
		reflect.TypeFor[C1](),
		reflect.TypeFor[C2](),
	})
	if !ok {
		return []EntityID{}
	}

	return em.query(queryMask)
}

func (em *EntityManager) Query3[C1, C2, C3 any]() []EntityID {
	queryMask, ok := getComponentsBitmask([]reflect.Type{
		reflect.TypeFor[C1](),
		reflect.TypeFor[C2](),
		reflect.TypeFor[C3](),
	})
	if !ok {
		return []EntityID{}
	}

	return em.query(queryMask)
}

func (em *EntityManager) Teardown() {
	em.archetypes = nil
	em.entityArchetype = nil
	em.componentPools = nil
}

func (em *EntityManager) GetComponent[C any](entityID EntityID) (*C, bool) {
	archetype, exists := em.entityArchetype[entityID]
	if !exists {
		return nil, false
	}

	componentType := reflect.TypeFor[C]()
	componentID, ok := getComponentID(componentType)
	if !ok {
		return nil, false
	}

	component, exists := archetype.GetComponent(entityID, componentID)
	if !exists {
		return nil, false
	}

	return component.(*C), true
}

func (em *EntityManager) MustGetComponent[C any](entityID EntityID) *C {
	component, exists := em.GetComponent[C](entityID)
	if !exists {
		panic(fmt.Sprintf("Entity %d does not have component of type %s", entityID, reflect.TypeFor[C]().Name()))
	}

	return component
}

func (em *EntityManager) evaluateFilter[C any](entityID EntityID, filter Filter[C]) bool {
	if filter == nil {
		return true
	}

	component, ok := em.GetComponent[C](entityID)
	if !ok {
		return false
	}

	return filter(component)
}

func (em *EntityManager) QueryWith[C any](filter Filter[C]) []EntityID {
	if filter == nil {
		return em.Query[C]()
	}

	entities := em.Query[C]()
	for i, entityID := range entities {
		if !em.evaluateFilter(entityID, filter) {
			entities = append(entities[:i], entities[i+1:]...)
		}
	}

	return entities
}

func (em *EntityManager) QueryWith2[C1, C2 any](filter1 Filter[C1], filter2 Filter[C2]) []EntityID {
	if filter1 == nil && filter2 == nil {
		return em.Query2[C1, C2]()
	}

	entities := em.Query2[C1, C2]()
	for i, entityID := range entities {
		if !em.evaluateFilter(entityID, filter1) || !em.evaluateFilter(entityID, filter2) {
			entities = append(entities[:i], entities[i+1:]...)
		}
	}

	return entities
}

func (em *EntityManager) QueryWith3[C1, C2, C3 any](filter1 Filter[C1], filter2 Filter[C2], filter3 Filter[C3]) []EntityID {
	if filter1 == nil && filter2 == nil && filter3 == nil {
		return em.Query3[C1, C2, C3]()
	}

	entities := em.Query3[C1, C2, C3]()
	for i, entityID := range entities {
		if !em.evaluateFilter(entityID, filter1) || !em.evaluateFilter(entityID, filter2) || !em.evaluateFilter(entityID, filter3) {
			entities = append(entities[:i], entities[i+1:]...)
		}
	}

	return entities
}
