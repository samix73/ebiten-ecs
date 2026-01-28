package ecs

import (
	"fmt"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
)

// SystemID is a type alias for the unique identifier of a system.
type SystemID = uint64

const UndefinedSystemID SystemID = 0

// System is the interface that all systems must implement.
// Systems are responsible for updating and processing entities that have specific components.
// Each system is associated with a unique ID and a priority level.
type System interface {
	ID() SystemID
	Priority() int
	Update() error
	Teardown()

	baseSystem() *BaseSystem
}

// DrawableSystem is an optional interface that systems can implement if they need to perform drawing operations.
type DrawableSystem interface {
	System
	Draw(screen *ebiten.Image)
}

// BaseSystem provides a foundational implementation of the System interface.
// It includes common fields and methods that can be reused by concrete system implementations.
type BaseSystem struct {
	id            SystemID
	priority      int
	entityManager *EntityManager
	game          *Game
}

// NewBaseSystem creates a new BaseSystem with the given ID and priority.
func NewBaseSystem(priority int) *BaseSystem {
	return &BaseSystem{
		priority: priority,
	}
}

// ID returns the unique identifier of the system.
func (s *BaseSystem) ID() SystemID {
	return s.id
}

func (s *BaseSystem) setID(id SystemID) {
	s.id = id
}

// Priority returns the priority level of the system.
// Systems with lower priority values are executed before those with higher values.
func (s *BaseSystem) Priority() int {
	return s.priority
}

// EntityManager returns the EntityManager associated with the system.
func (s *BaseSystem) EntityManager() *EntityManager {
	return s.entityManager
}

// Game returns the Game instance associated with the system.
func (s *BaseSystem) Game() *Game {
	return s.game
}

// baseSystem returns the BaseSystem instance.
// This is used to enforce embedding of BaseSystem in concrete system implementations.
func (s *BaseSystem) baseSystem() *BaseSystem {
	return s
}

func (s *BaseSystem) canUpdate() bool {
	return s.entityManager != nil && s.game != nil && s.ID() != UndefinedSystemID
}

// SystemManager manages a collection of systems within the framework.
// It is responsible for adding, removing, updating, and drawing systems.
// The SystemManager ensures that systems are executed in order of their priority.
type SystemManager struct {
	nextID        SystemID
	systems       []System
	entityManager *EntityManager
	game          *Game
}

// NewSystemManager creates a new SystemManager with the provided EntityManager and Game instance.
func NewSystemManager(entityManager *EntityManager, game *Game) *SystemManager {
	return &SystemManager{
		nextID:        1,
		systems:       make([]System, 0),
		entityManager: entityManager,
		game:          game,
	}
}

func (sm *SystemManager) sortSystems() {
	slices.SortStableFunc(sm.systems, func(a, b System) int {
		if a.Priority() < b.Priority() {
			return -1
		}

		if a.Priority() > b.Priority() {
			return 1
		}

		return 0
	})
}

// Add adds one or more systems to the SystemManager.
// It ensures that each system has access to the EntityManager and Game instance.
// After adding, it sorts the systems based on their priority.
func (sm *SystemManager) Add(systems ...System) {
	if len(systems) == 0 {
		return
	}

	for _, system := range systems {
		system.baseSystem().setID(sm.nextID)
		sm.nextID++
		if system.baseSystem().EntityManager() == nil {
			system.baseSystem().entityManager = sm.entityManager
		}

		if system.baseSystem().Game() == nil {
			system.baseSystem().game = sm.game
		}
	}

	sm.systems = append(sm.systems, systems...)

	sm.sortSystems()
}

// Remove removes a system from the SystemManager by its ID.
// If the system implements the Teardowner interface, its Teardown method is called before removal.
func (sm *SystemManager) Remove(systemID SystemID) {
	indexToDelete, exists := slices.BinarySearchFunc(sm.systems, systemID, func(s System, id SystemID) int {
		if s.ID() < id {
			return -1
		}

		if s.ID() > id {
			return 1
		}

		return 0
	})

	if !exists {
		return
	}

	systemToDelete := sm.systems[indexToDelete]
	sm.systems[indexToDelete] = sm.systems[len(sm.systems)-1]
	sm.systems = sm.systems[:len(sm.systems)-1]

	systemToDelete.Teardown()
}

// Update updates all systems managed by the SystemManager.
// It calls the Update method of each system in order of their priority.
// If any system returns an error during its update, the process is halted and the error is returned.
func (sm *SystemManager) Update() error {
	for _, system := range sm.systems {
		if !system.baseSystem().canUpdate() {
			continue
		}

		if err := system.Update(); err != nil {
			return fmt.Errorf("ecs.SystemManager.Update error updating system %d: %w", system.ID(), err)
		}
	}

	return nil
}

// Draw calls the Draw method of all systems that implement the DrawableSystem interface.
func (sm *SystemManager) Draw(screen *ebiten.Image) {
	for _, system := range sm.systems {
		if system, ok := system.(DrawableSystem); ok {
			system.Draw(screen)
		}
	}
}

// Teardown calls the Teardown method of all systems that implement the Teardowner interface.
func (sm *SystemManager) Teardown() {
	for _, system := range sm.systems {
		system.Teardown()
	}

	sm.systems = nil
}
