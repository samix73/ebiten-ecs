# ebiten-ecs

A lightweight, generic, allocation–friendly Entity Component System (ECS) built for games using [Ebiten](https://ebitengine.org). It provides:

- Entity + component storage with pooling ([`ecs.ComponentContainer`](component.go))
- Generic helpers for adding and querying components ([`ecs.AddComponent`](entity.go), [`ecs.Query`](entity.go), [`ecs.Query2`](entity.go), [`ecs.GetComponent`](entity.go))
- Cache‑friendly multi-component querying
- Systems with priorities and optional rendering phase ([`ecs.System`](system.go), [`ecs.RendererSystem`](system.go))
- Worlds to scope game states/scenes ([`ecs.World`](world.go), [`ecs.BaseWorld`](world.go))
- A thin wrapper over Ebiten’s game loop ([`ecs.Game`](game.go), [`ecs.GameConfig`](game.go))
- Simple ID generation ([`ecs.NextID`](id.go))

## Installation

```bash
go get github.com/samix73/ebiten-ecs
```

## Quick Start

Below is a minimal runnable example showing:
- Defining components
- Creating a system (with update + draw)
- Creating a world embedding [`ecs.BaseWorld`](world.go)
- Bootstrapping a game with [`ecs.Game`](game.go)

```go
package main

import (
    "log"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/samix73/ebiten-ecs/ecs"
)

// --- Components ---

type Transform struct {
    X, Y float64
}

// Initialize the Transform component
func (t *Transform) Init()  {t.X, t.Y = 0, 0}
// Reset the Transform component
func (t *Transform) Reset() { t.X, t.Y = 0, 0 }

// --- Systems ---

type MovementSystem struct {
    *ecs.BaseSystem
}

func NewMovementSystem(priority int, em *ecs.EntityManager, g *ecs.Game) *MovementSystem {
    return &MovementSystem{
        BaseSystem: ecs.NewBaseSystem(ecs.NextID(), priority, em, g),
    }
}

func (s *MovementSystem) Update() error {
    // Move every Transform slightly
    for entityID := range ecs.Query[Transform](s.EntityManager()) {
        if tr, ok := ecs.GetComponent[Transform](s.EntityManager(), entityID); ok {
            tr.X += 60 * s.Game().DeltaTime()
        }
    }

    return nil
}

// Optional render phase (implements ecs.RendererSystem)
func (s *MovementSystem) Draw(screen *ebiten.Image) {
    // Could draw debug info here (omitted for brevity)
}

func (s *MovementSystem) Teardown() {}

// --- World ---

type DemoWorld struct {
    *ecs.BaseWorld
}

func (w *DemoWorld) Init(g *ecs.Game) error {
    em := ecs.NewEntityManager()
    sm := ecs.NewSystemManager(em)

    w.BaseWorld = ecs.NewBaseWorld(em, sm, g)

    // Systems
    sm.Add(NewMovementSystem(0, em, g))

    // Entities
    player := em.NewEntity()
    ecs.AddComponent[Transform](em, player)

    return nil
}

// --- main ---

func main() {
    game := ecs.NewGame(&ecs.GameConfig{
        Title:        "ECS Demo",
        ScreenWidth:  800,
        ScreenHeight: 600,
        Fullscreen:   false,
    })

    if err := game.SetActiveWorld(&DemoWorld{}); err != nil {
        log.Fatal(err)
    }

    if err := game.Start(); err != nil {
        log.Fatal(err)
    }
}
```

## Core Concepts

- Entities: Opaque IDs (`EntityID` = [`ecs.ID`](id.go)) created via [`ecs.EntityManager.NewEntity`](entity.go).
- Components: Plain structs with optional `Init()` + `Reset()` (for pooling). Added via [`ecs.AddComponent`](entity.go).
- Queries: Use generics for compile-time type safety ([`ecs.Query`](entity.go), [`ecs.Query2`](entity.go), [`ecs.Query3`](entity.go)).
- Systems: Provide behavior; ordered by `Priority()` (lower first). Rendering systems also implement `Draw`.
- Worlds: Aggregate an entity + system set; switchable via [`ecs.Game.SetActiveWorld`](game.go).

## Query Examples

```go
for e := range ecs.Query[Transform](em) { /* ... */ }
for e := range ecs.Query2[Transform, AnotherComponent](em) { /* ... */ }
tr, ok := ecs.GetComponent[Transform](em, e)
```

## Performance

See benchmarks in [entity_test.go](entity_test.go) exercising queries vs direct component access.

## License

MIT – see [LICENSE](LICENSE).
