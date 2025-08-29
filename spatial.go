package ecs

import "golang.org/x/image/math/f64"

// WithinBounds creates a spatial bounds filter
// Users should adapt this for their specific transform component types
// Example usage:
//   boundsFilter := ecs.Where(func(t *MyTransform) bool { 
//     return ecs.WithinBoundsCheck(t.Position, 0, 0, 100, 100) 
//   })
func WithinBoundsCheck(position f64.Vec2, minX, minY, maxX, maxY float64) bool {
	return position[0] >= minX &&
		position[0] <= maxX &&
		position[1] >= minY &&
		position[1] <= maxY
}

// WithinRadius creates a spatial radius filter
// Users should adapt this for their specific transform component types  
// Example usage:
//   radiusFilter := ecs.Where(func(t *MyTransform) bool { 
//     return ecs.WithinRadiusCheck(t.Position, 50, 50, 25) 
//   })
func WithinRadiusCheck(position f64.Vec2, centerX, centerY, radius float64) bool {
	dx := position[0] - centerX
	dy := position[1] - centerY
	distanceSquared := dx*dx + dy*dy
	return distanceSquared <= radius*radius
}