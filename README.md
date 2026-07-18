[![Go CI](https://github.com/samix73/ebiten-ecs/actions/workflows/ci.yml/badge.svg)](https://github.com/samix73/ebiten-ecs/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/samix73/ebiten-ecs.svg)](https://pkg.go.dev/github.com/samix73/ebiten-ecs)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/samix73/ebiten-ecs)

# ebiten-ecs

A lightweight, generic, allocation–friendly Entity Component System (ECS) built for games using [Ebiten](https://ebitengine.org). It provides:

- Entity + component storage with pooling ([`ecs.EntityManager`](entity.go))
- EntityManager methods for adding and querying components ([`em.AddComponent`](entity.go), [`em.Query`](entity.go), [`em.Query2`](entity.go), [`em.GetComponent`](entity.go))
- Cache‑friendly multi-component querying
- Flexible filtering system with `QueryWith` functions ([`filter.go`](filter.go), [`spatial.go`](spatial.go))
- Worlds to scope game states/scenes ([`ecs.World`](world.go), [`ecs.BaseWorld`](world.go))
- A thin wrapper over Ebiten’s game loop ([`ecs.Game`](game.go), [`ecs.GameConfig`](game.go))

## Installation

```bash
go get github.com/samix73/ebiten-ecs/v2
```

## Integration

### 1. Registering Systems

Systems must be registered in the `init()` function of their package using `ecs.RegisterSystem`.

```go
package systems

import (
	"log/slog"

	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/samix73/ebiten-ecs/v2"
)

// Ensure PauseSystem implements ecs.System
var _ ecs.System = (*PauseSystem)(nil)

func init() {
	ecs.RegisterSystem(NewPauseSystem)
}

type PauseSystem struct {
	*ecs.BaseSystem

	paused            bool
	originalTimeScale float64
}

func NewPauseSystem(priority int) *PauseSystem {
	return &PauseSystem{
		BaseSystem: ecs.NewBaseSystem(priority),

		paused: false,
	}
}

// Update checks for input and toggles the game's time scale.
func (p *PauseSystem) Update() error {
	// Example input check
	if !inpututil.IsKeyJustPressed(ebiten.KeyP) {
		return nil
	}

	game := p.Game()

	if p.paused {
		game.SetTimeScale(p.originalTimeScale)
	} else {
		p.originalTimeScale = game.TimeScale()
		game.SetTimeScale(0)
	}

	p.paused = !p.paused

	slog.Info("Paused", "paused", p.paused)

	return nil
}

func (p *PauseSystem) Teardown() {
}
```

### 2. Registering Components

Components are registered similarly using `ecs.RegisterComponent`.

```go
package components

import (
	"github.com/jakecoffman/cp"
	"github.com/samix73/ebiten-ecs/v2"
)

func init() {
	ecs.RegisterComponent[Transform]()
}

// Transform represents the position and rotation of an entity in 2D space.
type Transform struct {
	Position cp.Vector
	Rotation float64
}

func (t *Transform) SetPosition(x, y float64) {
	t.Position.X = x
	t.Position.Y = y
}

func (t *Transform) Translate(x, y float64) {
	t.Position.X += x
	t.Position.Y += y
}

func (t *Transform) Reset() {
	t.Position.X = 0
	t.Position.Y = 0
	t.Rotation = 0
}
```

### 3. World Configuration

Define your world and its initial state in a TOML file (e.g., `main_world.toml`).

```toml
name = "main_world"

[[systems]]
name = "RestartSystem"
priority = 0

[[systems]]
name = "PauseSystem"
priority = 100

[[entities]]
path = "entities/ActiveCamera.toml"
```

### 4. Loading and Running a World

Use `ecs.Game` to load the world configuration and start the game loop.

```go
package main

import (
	"log/slog"
	"os"

	"github.com/samix73/ebiten-ecs/v2"
)

func main() {
	g := ecs.NewGame(&ecs.GameConfig{
		Title:        "Game",
		ScreenWidth:  1280,
		ScreenHeight: 960,
		Fullscreen:   false,
	})

	// Load the world from the configuration file
	world, err := g.LoadWorld("main_world.toml")
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	g.SetActiveWorld(world)

	if err := g.Start(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
```

## Core Concepts

- Entities: Opaque IDs (`EntityID` = `uint64`) created via [`ecs.EntityManager.NewEntity`](entity.go).
- Components: Plain data types with optional `Init()` + `Reset()` (for pooling). Added via [`em.AddComponent`](entity.go).
- Queries: Use generics for compile-time type safety ([`em.Query`](entity.go), [`em.Query2`](entity.go), [`em.Query3`](entity.go)).
- Systems: Provide behavior; ordered by `Priority()` (lower first). Rendering systems also implement `Draw`.
- Worlds: Aggregate an entity + system set; switchable via [`ecs.Game.SetActiveWorld`](game.go).

## Query Examples

```go
for _, e := range em.Query[Transform]() { /* ... */ }
for _, e := range em.Query2[Transform, AnotherComponent]() { /* ... */ }
tr, ok := em.GetComponent[Transform](e)
```

## Filtering

The ECS supports flexible filtering of query results using `QueryWith` functions. You can filter on specific component types within a query:

```go
// Define a filter function
highZoomFilter := func(c *CameraComponent) bool {
    return c.Zoom > 1.0
}

// 1. Single Component Query with Filter
// Query entities with CameraComponent where Zoom > 1.0
for _, entityID := range em.QueryWith(highZoomFilter) {
    camera := em.MustGetComponent[CameraComponent](entityID)
    // Process high-zoom cameras
}

// 2. Multi-Component Query with Filter
// Query entities with CameraComponent and Transform, filtering ONLY on CameraComponent
// Pass 'nil' for the second filter to skip filtering on Transform
for _, entityID := range em.QueryWith2(highZoomFilter, nil) {
    camera := em.MustGetComponent[CameraComponent](entityID)
    transform := em.MustGetComponent[Transform](entityID)
    // Process...
}

// 3. Multi-Component Query with Multiple Filters
boundsFilter := func(t *Transform) bool {
    return t.X >= 0 && t.X <= 100 && t.Y >= 0 && t.Y <= 100
}

// Filter on BOTH components (Logical AND between component filters is implicit in QueryWith2)
for _, entityID := range em.QueryWith2(highZoomFilter, boundsFilter) {
    // Process entities where Camera.Zoom > 1.0 AND Transform is within bounds
}

// 4. Advanced Filter Composition
// Use ecs.And, ecs.Or, ecs.Not to create complex filters for a single component

// Create a compound filter for CameraComponent
complexCameraFilter := ecs.And(
    highZoomFilter,
    ecs.Not(func(c *CameraComponent) bool { return c.FOV > 90 }),
)

for _, entityID := range em.QueryWith(complexCameraFilter) {
    // Process cameras with Zoom > 1.0 AND FOV <= 90
}
```

### Filter Functions
- **`em.QueryWith(filter)`**: Query entities with 1 component type and apply filter.
- **`em.QueryWith2(filter1, filter2)`**: Query entities with 2 component types. Pass `nil` for any filter to skip it.
- **`em.QueryWith3(filter1, filter2, filter3)`**: Query entities with 3 component types.
- **`ecs.And(filters...)`**: Combines filters for the *same component type* with logical AND.
- **`ecs.Or(filters...)`**: Combines filters for the *same component type* with logical OR.
- **`ecs.Not(filter)`**: Negates a filter.

### Performance
Filtering applies the filter function during the query iteration. It is efficient but obviously slower than a raw `Query` if the filter logic is heavy. `QueryWith` functions iterate over the archetypes that match the component composition first, then apply the filter functions to the candidates.

## Performance

See benchmarks in [entity_test.go](entity_test.go) exercising queries vs direct component access.

## License

MIT – see [LICENSE](LICENSE).
