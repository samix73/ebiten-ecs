package ecs

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type World interface {
	Update() error
	Draw(screen *ebiten.Image)
	Teardown()
	Init(g *Game) error

	baseWorld() *BaseWorld // Force embedding BaseWorld
}

type BaseWorld struct {
	systemManager *SystemManager
}

func (bw *BaseWorld) baseWorld() *BaseWorld {
	return bw
}

func NewBaseWorld(systemManager *SystemManager) *BaseWorld {
	return &BaseWorld{
		systemManager: systemManager,
	}
}

func (w *BaseWorld) Update() error {
	if err := w.SystemManager().Update(); err != nil {
		return err
	}
	return nil
}

func (w *BaseWorld) Draw(screen *ebiten.Image) {
	w.SystemManager().Draw(screen)
}

func (w *BaseWorld) SystemManager() *SystemManager {
	return w.systemManager
}

func (m *BaseWorld) Teardown() {
	m.SystemManager().Teardown()
}
