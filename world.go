package ecs

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// World is the interface that all game worlds must implement.
// A world is responsible for managing the game state, including entities and systems.
// Each world must provide methods for updating, drawing, and tearing down the world.
// Additionally, a world must be initialized with a reference to the Game instance.
type World interface {
	Update() error
	Draw(screen *ebiten.Image)
	Teardown()
	Init(g *Game) error

	baseWorld() *BaseWorld // Force embedding BaseWorld
}

// BaseWorld provides a foundational implementation of the World interface.
type BaseWorld struct {
	systemManager *SystemManager
	entityManager *EntityManager
}

// baseWorld returns the BaseWorld instance.
// This is used to enforce embedding of BaseWorld in concrete world implementations.
func (bw *BaseWorld) baseWorld() *BaseWorld {
	return bw
}

// NewBaseWorld creates a new BaseWorld with the given EntityManager and SystemManager.
func NewBaseWorld(entityManager *EntityManager, systemManager *SystemManager) *BaseWorld {
	return &BaseWorld{
		entityManager: entityManager,
		systemManager: systemManager,
	}
}

// Update updates the world by updating all systems managed by the SystemManager.
// If any system returns an error during its update, the process is halted and the error is returned.
func (w *BaseWorld) Update() error {
	if err := w.SystemManager().Update(); err != nil {
		return err
	}

	return nil
}

// Draw draws the world by calling the Draw method of all systems that implement the DrawableSystem interface.
func (w *BaseWorld) Draw(screen *ebiten.Image) {
	w.SystemManager().Draw(screen)
}

// SystemManager returns the SystemManager associated with the world.
func (w *BaseWorld) SystemManager() *SystemManager {
	return w.systemManager
}

// EntityManager returns the EntityManager associated with the world.
func (w *BaseWorld) EntityManager() *EntityManager {
	return w.entityManager
}

// Teardown tears down the world by calling the Teardown method of all systems that implement the Teardowner interface
// and then tearing down the EntityManager.
func (m *BaseWorld) Teardown() {
	m.SystemManager().Teardown()
	m.EntityManager().Teardown()
}
