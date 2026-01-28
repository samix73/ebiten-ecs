package ecs

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"

	"github.com/BurntSushi/toml"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/samix73/game/game/assets"
)

var _ ebiten.Game = (*Game)(nil)

type GameConfig struct {
	Title                     string
	ScreenWidth, ScreenHeight int
	Fullscreen                bool
}

type Game struct {
	cfg         *GameConfig
	activeWorld *World
	timeScale   float64
}

func NewGame(cfg *GameConfig) *Game {
	return &Game{
		cfg:       cfg,
		timeScale: 1.0,
	}
}

func (g *Game) TimeScale() float64 {
	return g.timeScale
}

func (g *Game) SetTimeScale(scale float64) {
	g.timeScale = math.Max(scale, 0)
}

func (g *Game) Config() GameConfig {
	return *g.cfg
}

func (g *Game) SetActiveWorld(world *World) error {
	if g.activeWorld != nil {
		g.activeWorld.Teardown()
	}

	g.activeWorld = world

	return nil
}

func (g *Game) loadSystems(systemManager *SystemManager, systemCfgs []SystemConfig) error {
	for _, systemCfg := range systemCfgs {
		systemCtor, ok := GetSystem(systemCfg.Name)
		if !ok {
			return fmt.Errorf("ecs.Game.loadSystems: system %s not found", systemCfg.Name)
		}

		systemManager.Add(systemCtor(systemCfg.Priority))
	}

	return nil
}

// loadEntities loads entities from entity configs and world metadata.
// Components from world metadata overwrite components from entity files.
func (g *Game) loadEntities(em *EntityManager, entityCfgs []EntityConfig, worldMD toml.MetaData) error {
	for _, entityCfg := range entityCfgs {
		if entityCfg.Name == "" {
			return errors.New("ecs.Game.loadEntities: entity name is empty")
		}

		componentsByName := make(map[string]any)

		for name, args := range entityCfg.Components {
			if name == "" {
				return errors.New("ecs.Game.loadEntities: component name is empty")
			}

			component, ok := NewComponent(em, name)
			if !ok {
				return fmt.Errorf("ecs.Game.loadEntities: component %s not found", name)
			}

			if err := worldMD.PrimitiveDecode(args, component); err != nil {
				return fmt.Errorf("ecs.Game.loadEntities: world PrimitiveDecode %w", err)
			}

			componentsByName[name] = component
		}

		entityData, err := assets.GetEntity(entityCfg.Name)
		if err != nil {
			return fmt.Errorf("ecs.Game.loadEntities: %w", err)
		}

		var protoEntity EntityComponentsConfig
		md, err := toml.NewDecoder(bytes.NewReader(entityData)).Decode(&protoEntity)
		if err != nil {
			return fmt.Errorf("ecs.Game.loadEntities: %w", err)
		}

		for componentName, args := range protoEntity {
			if _, ok := componentsByName[componentName]; ok { // skip if component is already loaded from world config
				continue
			}

			component, ok := NewComponent(em, componentName)
			if !ok {
				return fmt.Errorf("ecs.Game.loadEntities: component %s not found", componentName)
			}

			if err := md.PrimitiveDecode(args, component); err != nil {
				return fmt.Errorf("ecs.Game.loadEntities: PrimitiveDecode %w", err)
			}

			componentsByName[componentName] = component
		}

		componentsData := slices.Collect(maps.Values(componentsByName))

		if _, err := em.NewEntity(componentsData...); err != nil {
			return fmt.Errorf("ecs.Game.loadEntities: %w", err)
		}
	}

	return nil
}

func (g *Game) LoadWorld(path string) (*World, error) {
	data, err := assets.GetWorld(path)
	if err != nil {
		return nil, fmt.Errorf("ecs.Game.LoadWorld: %w", err)
	}

	var worldConfig WorldConfig
	md, err := toml.NewDecoder(bytes.NewReader(data)).Decode(&worldConfig)
	if err != nil {
		return nil, fmt.Errorf("ecs.Game.LoadWorld: %w", err)
	}

	em := NewEntityManager()
	sm := NewSystemManager(em, g)

	if err := g.loadSystems(sm, worldConfig.Systems); err != nil {
		return nil, fmt.Errorf("ecs.Game.LoadWorld: %w", err)
	}

	if err := g.loadEntities(em, worldConfig.Entities, md); err != nil {
		return nil, fmt.Errorf("ecs.Game.LoadWorld: %w", err)
	}

	return &World{
		cfg: worldConfig,

		systemManager: sm,
		entityManager: em,
	}, nil
}

func (g *Game) DeltaTime() float64 {
	return 1.0 / float64(ebiten.TPS()) * g.TimeScale()
}

func (g *Game) Start() error {
	ebiten.SetWindowSize(g.cfg.ScreenWidth, g.cfg.ScreenHeight)
	ebiten.SetFullscreen(g.cfg.Fullscreen)
	ebiten.SetWindowTitle(g.cfg.Title)

	if err := ebiten.RunGameWithOptions(g, nil); err != nil {
		return fmt.Errorf("ecs.Game.Start ebiten.RunGameWithOptions error: %w", err)
	}

	return nil
}

func (g *Game) Layout(outsideWidth int, outsideHeight int) (screenWidth int, screenHeight int) {
	return g.cfg.ScreenWidth, g.cfg.ScreenHeight
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.activeWorld == nil {
		return
	}

	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("FPS: %.2f", ebiten.ActualFPS()), 16, 32)

	g.activeWorld.Draw(screen)
}

func (g *Game) Update() error {
	if g.activeWorld == nil {
		return nil
	}

	if err := g.activeWorld.Update(); err != nil {
		return fmt.Errorf("ecs.Game.Update activeWorld.Update error: %w", err)
	}

	return nil
}
