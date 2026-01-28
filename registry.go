package ecs

import (
	"fmt"
	"log/slog"
	"reflect"
	"sync"
)

type ComponentID = uint

type (
	SystemCtor[S System] func(priority int) S
)

var (
	systemsRegistry                  = make(map[string]SystemCtor[System])
	componentsNameLookup             = make(map[string]any)
	componentTypeLookup              = make(map[reflect.Type]ComponentID)
	componentsPools                  = make(map[ComponentID]*sync.Pool)
	nextComponentBit     ComponentID = 1
)

func getName[S any]() string {
	t := reflect.TypeFor[S]()
	if t.Kind() == reflect.Pointer {
		return t.Elem().Name()
	}

	return t.Name()
}

// RegisterSystem registers a system constructor in the ECS registry
// to allow for dynamic system creation.
func RegisterSystem[S System](systemCtor SystemCtor[S]) error {
	name := getName[S]()
	if _, ok := systemsRegistry[name]; ok {
		return fmt.Errorf("ecs.RegisterSystem: system %s already registered", name)
	}

	systemsRegistry[name] = func(priority int) System {
		return systemCtor(priority)
	}

	slog.Debug("ecs.RegisterSystem: registered system", slog.String("name", name))
	return nil
}

func RegisterComponent[T any]() error {
	name := getName[T]()
	if _, ok := componentsNameLookup[name]; ok {
		return fmt.Errorf("ecs.RegisterComponent: component %s already registered", name)
	}

	componentsNameLookup[name] = *new(T)

	// Assign a unique bit position for bitmask filtering
	componentType := reflect.TypeFor[T]()
	if componentType.Kind() == reflect.Ptr {
		componentType = componentType.Elem()
	}

	if _, exists := componentTypeLookup[componentType]; exists {
		return fmt.Errorf("ecs.RegisterComponent: component %s already registered", name)
	}

	componentTypeLookup[componentType] = nextComponentBit
	componentsPools[nextComponentBit] = &sync.Pool{New: func() any { return reflect.New(componentType).Interface() }}
	nextComponentBit++

	slog.Debug("ecs.RegisterComponent: registered component", slog.String("name", name))
	return nil
}

func NewComponent(em *EntityManager, name string) (any, bool) {
	comp, ok := componentsNameLookup[name]
	if !ok {
		return nil, false
	}

	componentID, ok := getComponentID(reflect.TypeOf(comp))
	if !ok {
		return nil, false
	}

	componentPool, ok := getComponentPool(componentID)
	if !ok {
		return nil, false
	}

	return componentPool.Get(), true
}

// GetSystem retrieves a system constructor from the ECS registry by name.
func GetSystem(name string) (SystemCtor[System], bool) {
	ctor, ok := systemsRegistry[name]
	if !ok {
		return nil, false
	}

	return ctor, true
}

func getComponentPool(componentID ComponentID) (*sync.Pool, bool) {
	pool, exists := componentsPools[componentID]
	return pool, exists
}

// getComponentsBitmask computes a bitmask from component types for fast signature matching.
func getComponentsBitmask(componentTypes []reflect.Type) (Bitmask, bool) {
	var mask Bitmask
	for _, ct := range componentTypes {
		bitPos, ok := getComponentID(ct)
		if !ok {
			return Bitmask{}, false
		}

		mask.Set(bitPos)
	}

	return mask, true
}

// getComponentID returns the bitmask position for a component type.
func getComponentID(componentType reflect.Type) (ComponentID, bool) {
	if componentType.Kind() == reflect.Pointer {
		componentType = componentType.Elem()
	}

	bitPos, exists := componentTypeLookup[componentType]
	return bitPos, exists
}

func ComponentsBitMask(componentTypes []reflect.Type) (Bitmask, bool) {
	return getComponentsBitmask(componentTypes)
}

func GetComponentID[T any]() (ComponentID, bool) {
	return getComponentID(reflect.TypeFor[T]())
}
