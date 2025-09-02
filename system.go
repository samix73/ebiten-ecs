package ecs

import (
	"fmt"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
)

type SystemID = ID

type System interface {
	ID() SystemID
	Priority() int
	Update() error
	Teardown()
	baseSystem() *BaseSystem
}

type DrawableSystem interface {
	System
	Draw(screen *ebiten.Image)
}

type BaseSystem struct {
	id            SystemID
	priority      int
	entityManager *EntityManager
	game          *Game
}

func NewBaseSystem(id SystemID, priority int) *BaseSystem {
	return &BaseSystem{
		id:       id,
		priority: priority,
	}
}

func (s *BaseSystem) ID() SystemID {
	return s.id
}

func (s *BaseSystem) Priority() int {
	return s.priority
}

func (s *BaseSystem) EntityManager() *EntityManager {
	return s.entityManager
}

func (s *BaseSystem) Game() *Game {
	return s.game
}

func (s *BaseSystem) baseSystem() *BaseSystem {
	return s
}

func (s *BaseSystem) canUpdate() bool {
	return s.entityManager != nil && s.game != nil
}

type SystemManager struct {
	systems       []System
	entityManager *EntityManager
	game          *Game
}

func NewSystemManager(entityManager *EntityManager, game *Game) *SystemManager {
	return &SystemManager{
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

func (sm *SystemManager) Add(systems ...System) {
	if len(systems) == 0 {
		return
	}

	for _, system := range systems {
		if system.baseSystem().entityManager == nil {
			system.baseSystem().entityManager = sm.entityManager
		}

		if system.baseSystem().game == nil {
			system.baseSystem().game = sm.game
		}
	}

	sm.systems = append(sm.systems, systems...)

	sm.sortSystems()
}

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

func (sm *SystemManager) Update() error {
	for _, system := range sm.systems {
		if !system.baseSystem().canUpdate() {
			continue
		}

		if err := system.Update(); err != nil {
			return fmt.Errorf("error updating system %d: %w", system.ID(), err)
		}
	}

	return nil
}

func (sm *SystemManager) Draw(screen *ebiten.Image) {
	for _, system := range sm.systems {
		if system, ok := system.(DrawableSystem); ok {
			system.Draw(screen)
		}
	}
}

func (sm *SystemManager) Teardown() {
	for _, system := range sm.systems {
		system.Teardown()
	}

	sm.systems = nil
}
