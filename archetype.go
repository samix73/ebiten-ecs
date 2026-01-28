package ecs

import (
	"fmt"
	"reflect"
)

// Component is an interface for components.
type Component interface {
	Reset()
	Init()
}

// Archetype represents a group of entities with the same component signature.
type Archetype struct {
	signature    Bitmask
	entities     []EntityID
	components   map[ComponentID]reflect.Value
	entityLookup map[EntityID]int
}

// NewArchetype creates a new archetype with the given component signature.
func NewArchetype(signature Bitmask) *Archetype {
	return &Archetype{
		signature:    signature,
		entities:     make([]EntityID, 0, 64),
		components:   make(map[ComponentID]reflect.Value),
		entityLookup: make(map[EntityID]int),
	}
}

// AddEntity adds an entity to the archetype.
func (a *Archetype) AddEntity(entityID EntityID, components map[ComponentID]any) error {
	if _, exists := a.entityLookup[entityID]; exists {
		return fmt.Errorf("ecs.Archetype.AddEntity: entity %d already exists in archetype", entityID)
	}

	index := len(a.entities)
	a.entities = append(a.entities, entityID)
	a.entityLookup[entityID] = index

	for componentID, component := range components {
		componentValue := reflect.ValueOf(component)
		if componentValue.Kind() != reflect.Pointer {
			return fmt.Errorf("ecs.Archetype.AddEntity: component %T must be a pointer", component)
		}

		if !a.signature.Has(componentID) {
			return fmt.Errorf("ecs.Archetype.AddEntity: component %T not found in archetype", component)
		}

		if _, ok := a.components[componentID]; !ok {
			a.components[componentID] = reflect.MakeSlice(reflect.SliceOf(componentValue.Type()), 0, 64)
		}
		a.components[componentID] = reflect.Append(a.components[componentID], componentValue)
	}

	return nil
}

// RemoveEntity removes an entity from the archetype.
func (a *Archetype) RemoveEntity(entityID EntityID) (map[ComponentID]any, error) {
	if _, exists := a.entityLookup[entityID]; !exists {
		return nil, fmt.Errorf("ecs.Archetype.RemoveEntity: entity %d not found in archetype", entityID)
	}

	index := a.entityLookup[entityID]
	delete(a.entityLookup, entityID)

	lastIndex := len(a.entities) - 1
	if index != lastIndex {
		lastEntityID := a.entities[lastIndex]
		a.entities[index] = lastEntityID
		a.entityLookup[lastEntityID] = index
	}

	a.entities = a.entities[:lastIndex]
	components := make(map[ComponentID]any, len(a.components))
	for componentID := range a.components {
		components[componentID] = a.components[componentID].Index(index).Interface()
		if index != lastIndex {
			targetVal := a.components[componentID].Index(index)
			lastVal := a.components[componentID].Index(lastIndex)
			targetVal.Set(lastVal)
		}

		zeroValue := reflect.Zero(a.components[componentID].Type().Elem())
		a.components[componentID].Index(lastIndex).Set(zeroValue)

		a.components[componentID] = a.components[componentID].Slice(0, lastIndex)
	}

	return components, nil
}

// Entities returns a slice of entity IDs in the archetype.
func (a *Archetype) Entities() []EntityID {
	return a.entities
}

// HasComponent returns true if the entity has the specified component.
func (a *Archetype) HasComponent(entityID EntityID, componentID ComponentID) bool {
	index, exists := a.entityLookup[entityID]
	if !exists {
		return false
	}

	if !a.signature.Has(componentID) {
		return false
	}

	return a.components[componentID].Len() > index
}

func (a *Archetype) GetComponent(entityID EntityID, componentID ComponentID) (any, bool) {
	if !a.HasComponent(entityID, componentID) {
		return nil, false
	}

	index := a.entityLookup[entityID]
	return a.components[componentID].Index(index).Interface(), true
}

// MatchesQuery returns true if the archetype matches the query mask.
func (a *Archetype) MatchesQuery(queryMask Bitmask) bool {
	return a.signature.HasAll(queryMask)
}

// SignatureMatches returns true if the archetype's signature matches the given signature.
func (a *Archetype) SignatureMatches(signature Bitmask) bool {
	return a.signature.Equals(signature)
}
