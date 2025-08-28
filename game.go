package ecs

import (
	"fmt"
	"math"
	"reflect"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var _ ebiten.Game = (*Game)(nil)

type GameConfig struct {
	Title                     string
	ScreenWidth, ScreenHeight int
	Fullscreen                bool
}

type Game struct {
	cfg         *GameConfig
	activeWorld World
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

func (g *Game) RestartActiveWorld() error {
	typ := reflect.TypeOf(g.activeWorld).Elem()
	newWorld := reflect.New(typ).Interface().(World)

	if err := g.SetActiveWorld(newWorld); err != nil {
		return fmt.Errorf("ecs.Game.RestartActiveWorld g.SetActiveWorld error: %w", err)
	}

	return nil
}

func (g *Game) SetActiveWorld(world World) error {
	if g.activeWorld != nil {
		g.activeWorld.Teardown()
	}

	if err := world.Init(g); err != nil {
		return fmt.Errorf("ecs.Game.SetActiveWorld world.Init error: %w", err)
	}

	g.activeWorld = world

	return nil
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
