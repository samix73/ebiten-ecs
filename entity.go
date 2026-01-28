package ecs

import (
	"fmt"
	"log/slog"
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
	var componentData = make(map[ComponentID]any, len(components))
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

func (em *EntityManager) HasComponent(entityID EntityID, componentID ComponentID) bool {
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

func (em *EntityManager) RemoveComponent(entityID EntityID, componentID ComponentID) error {
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

func (em *EntityManager) Query(queryMask Bitmask) []EntityID {
	entities := make([]EntityID, 0)

	for i := range em.archetypes {
		if em.archetypes[i].MatchesQuery(queryMask) {
			entities = append(entities, em.archetypes[i].Entities()...)
		}
	}

	return entities
}

func (em *EntityManager) AddComponent(entityID EntityID, componentID ComponentID) (any, error) {
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

	return component, nil
}

func AddComponent[C any](em *EntityManager, entityID EntityID) (*C, error) {
	componentType := reflect.TypeFor[C]()
	componentID, ok := getComponentID(componentType)
	if !ok {
		return nil, fmt.Errorf("ecs.AddComponent: component %s not registered", componentType.Name())
	}

	component, err := em.AddComponent(entityID, componentID)
	if err != nil {
		slog.Error("ecs.AddComponent: failed to add component",
			slog.String("type", componentType.Name()),
			slog.Uint64("entityID", entityID),
			slog.Any("error", err),
		)

		return nil, fmt.Errorf("ecs.AddComponent: failed to add component: %w", err)
	}

	return component.(*C), nil
}

func (em *EntityManager) Teardown() {
	em.archetypes = nil
	em.entityArchetype = nil
	em.componentPools = nil
}

func RemoveComponent[C any](em *EntityManager, entityID EntityID) error {
	componentType := reflect.TypeFor[C]()
	componentID, ok := getComponentID(componentType)
	if !ok {
		return fmt.Errorf("ecs.RemoveComponent: component %s not registered", componentType.Name())
	}

	if err := em.RemoveComponent(entityID, componentID); err != nil {
		return fmt.Errorf("ecs.RemoveComponent: %w", err)
	}

	return nil
}

func Query[C any](em *EntityManager) []EntityID {
	queryMask, ok := getComponentsBitmask([]reflect.Type{reflect.TypeFor[C]()})
	if !ok {
		return []EntityID{}
	}

	return em.Query(queryMask)
}

func Query2[C1, C2 any](em *EntityManager) []EntityID {
	queryMask, ok := getComponentsBitmask([]reflect.Type{
		reflect.TypeFor[C1](),
		reflect.TypeFor[C2](),
	})
	if !ok {
		return []EntityID{}
	}

	return em.Query(queryMask)
}

func Query3[C1, C2, C3 any](em *EntityManager) []EntityID {
	queryMask, ok := getComponentsBitmask([]reflect.Type{
		reflect.TypeFor[C1](),
		reflect.TypeFor[C2](),
		reflect.TypeFor[C3](),
	})
	if !ok {
		return []EntityID{}
	}

	return em.Query(queryMask)
}

func HasComponent[C any](em *EntityManager, entityID EntityID) bool {
	componentType := reflect.TypeFor[C]()
	componentID, ok := getComponentID(componentType)
	if !ok {
		return false
	}

	return em.HasComponent(entityID, componentID)
}

func GetComponent[C any](em *EntityManager, entityID EntityID) (*C, bool) {
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

func MustGetComponent[C any](em *EntityManager, entityID EntityID) *C {
	component, exists := GetComponent[C](em, entityID)
	if !exists {
		panic(fmt.Sprintf("Entity %d does not have component of type %s", entityID, reflect.TypeFor[C]().Name()))
	}

	return component
}

func evaluateFilter[C any](em *EntityManager, entityID EntityID, filter Filter[C]) bool {
	if filter == nil {
		return true
	}

	component, ok := GetComponent[C](em, entityID)
	if !ok {
		return false
	}

	return filter(component)
}

func QueryWith[C any](em *EntityManager, filter Filter[C]) []EntityID {
	if filter == nil {
		return Query[C](em)
	}

	entities := Query[C](em)
	for i, entityID := range entities {
		if !evaluateFilter(em, entityID, filter) {
			entities = append(entities[:i], entities[i+1:]...)
		}
	}

	return entities
}

func QueryWith2[C1, C2 any](em *EntityManager, filter1 Filter[C1], filter2 Filter[C2]) []EntityID {
	if filter1 == nil && filter2 == nil {
		return Query2[C1, C2](em)
	}

	entities := Query2[C1, C2](em)
	for i, entityID := range entities {
		if !evaluateFilter(em, entityID, filter1) || !evaluateFilter(em, entityID, filter2) {
			entities = append(entities[:i], entities[i+1:]...)
		}
	}

	return entities
}

func QueryWith3[C1, C2, C3 any](em *EntityManager, filter1 Filter[C1], filter2 Filter[C2], filter3 Filter[C3]) []EntityID {
	if filter1 == nil && filter2 == nil && filter3 == nil {
		return Query3[C1, C2, C3](em)
	}

	entities := Query3[C1, C2, C3](em)
	for i, entityID := range entities {
		if !evaluateFilter(em, entityID, filter1) || !evaluateFilter(em, entityID, filter2) || !evaluateFilter(em, entityID, filter3) {
			entities = append(entities[:i], entities[i+1:]...)
		}
	}

	return entities
}
