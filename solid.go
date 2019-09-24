package model3d

import (
	"math"
)

// A Solid is a boolean function in 3D where a value of
// true indicates that a point is part of the solid, and
// false indicates that it is not.
type Solid interface {
	// Get the corners of a bounding rectangular volume.
	// Outside of this volume, Contains() must always
	// return false.
	Min() Coord3D
	Max() Coord3D

	Contains(p Coord3D) bool
}

// A SphereSolid is a Solid that yields values for a
// sphere.
type SphereSolid struct {
	Center Coord3D
	Radius float64
}

func (s *SphereSolid) Min() Coord3D {
	return Coord3D{X: s.Center.X - s.Radius, Y: s.Center.Y - s.Radius, Z: s.Center.Z - s.Radius}
}

func (s *SphereSolid) Max() Coord3D {
	return Coord3D{X: s.Center.X + s.Radius, Y: s.Center.Y + s.Radius, Z: s.Center.Z + s.Radius}
}

func (s *SphereSolid) Contains(p Coord3D) bool {
	return p.Dist(s.Center) < s.Radius
}

// A CylinderSolid is a Solid that yields values for a
// cylinder. The cylinder is defined as all the positions
// less than Radius distance from the line segment between
// P1 and P2.
type CylinderSolid struct {
	P1     Coord3D
	P2     Coord3D
	Radius float64
}

func (c *CylinderSolid) Min() Coord3D {
	return Coord3D{
		X: math.Min(c.P1.X, c.P2.X) - c.Radius,
		Y: math.Min(c.P1.Y, c.P2.Y) - c.Radius,
		Z: math.Min(c.P1.Z, c.P2.Z) - c.Radius,
	}
}

func (c *CylinderSolid) Max() Coord3D {
	return Coord3D{
		X: math.Max(c.P1.X, c.P2.X) + c.Radius,
		Y: math.Max(c.P1.Y, c.P2.Y) + c.Radius,
		Z: math.Max(c.P1.Z, c.P2.Z) + c.Radius,
	}
}

func (c *CylinderSolid) Contains(p Coord3D) bool {
	diff := c.P1.Add(c.P2.Scale(-1))
	direction := diff.Scale(1 / diff.Norm())
	frac := p.Add(c.P2.Scale(-1)).Dot(direction)
	if frac < 0 || frac > diff.Norm() {
		return false
	}
	projection := c.P2.Add(direction.Scale(frac))
	return projection.Dist(p) < c.Radius
}

// A JoinedSolid is a Solid composed of other solids.
type JoinedSolid []Solid

func (j JoinedSolid) Min() Coord3D {
	min := j[0].Min()
	for _, s := range j[1:] {
		min1 := s.Min()
		min.X = math.Min(min.X, min1.X)
		min.Y = math.Min(min.Y, min1.Y)
		min.Z = math.Min(min.Z, min1.Z)
	}
	return min
}

func (j JoinedSolid) Max() Coord3D {
	max := j[0].Max()
	for _, s := range j[1:] {
		max1 := s.Max()
		max.X = math.Max(max.X, max1.X)
		max.Y = math.Max(max.Y, max1.Y)
		max.Z = math.Max(max.Z, max1.Z)
	}
	return max
}

func (j JoinedSolid) Contains(c Coord3D) bool {
	for _, s := range j {
		if s.Contains(c) {
			return true
		}
	}
	return false
}

// SolidToMesh approximates the solid s as a triangle mesh
// by blurring the result of a RectScanner.
//
// The delta argument specifies the initial spacing
// between sampled cubes, and subdivisions indicates the
// maximum number of times these cubes can be cut in half.
//
// The blurFrac argument specifies how much each vertex is
// moved towards its neighbors, between 0 and 1.
// The blurIters argument specifies how many times the
// resulting mesh is blurred before being returned.
func SolidToMesh(s Solid, delta float64, subdivisions int, blurFrac float64, blurIters int) *Mesh {
	scanner := NewRectScanner(s, delta)
	for i := 0; i < subdivisions; i++ {
		scanner.Subdivide()
	}
	mesh := scanner.Mesh()
	for i := 0; i < blurIters; i++ {
		mesh = mesh.Blur(blurFrac)
	}
	return mesh
}

// A RectScanner maps out the edges of a solid using
// axis-aligned cubes.
type RectScanner struct {
	border map[*rectPiece]bool
	solid  Solid
}

// NewRectScanner creates a RectScanner by uniformly
// scanning the solid with a spacing of delta units.
func NewRectScanner(s Solid, delta float64) *RectScanner {
	spacer := newSquareSpacer(s, delta)
	cache := newSolidCache(s, spacer)

	pieces := map[int]*rectPiece{}
	res := &RectScanner{
		border: map[*rectPiece]bool{},
		solid:  s,
	}

	// First, create all border pieces so that we can
	// create all the empty and locked pieces next to them
	// without creating unneeded ones.
	spacer.IterateSquares(func(x, y, z int) {
		piece := &rectPiece{
			Min: spacer.CornerCoord(x, y, z),
			Max: spacer.CornerCoord(x+1, y+1, z+1),

			NumInteriorCorners: cache.NumInteriorCorners(x, y, z),
		}
		if piece.NumInteriorCorners != 0 && piece.NumInteriorCorners != 8 {
			piece.Neighbors = map[*rectPiece]bool{}
			pieces[spacer.SquareIndex(x, y, z)] = piece
			res.border[piece] = true
		} else if piece.NumInteriorCorners == 8 {
			if x == 0 || x == len(spacer.Xs)-2 || y == 0 || y == len(spacer.Ys)-2 ||
				z == 0 || z == len(spacer.Zs)-2 {
				panic("solid is true outside of bounds")
			}
		}
	})

	// Create all neighbors of the border pieces while
	// discarding pieces with no border neighbors.
	// This can save considerable amounts of memory.
	spacer.IterateSquares(func(x, y, z int) {
		var piece *rectPiece
		if p, ok := pieces[spacer.SquareIndex(x, y, z)]; ok {
			piece = p
		} else {
			piece = &rectPiece{
				Min: spacer.CornerCoord(x, y, z),
				Max: spacer.CornerCoord(x+1, y+1, z+1),

				NumInteriorCorners: cache.NumInteriorCorners(x, y, z),
			}
			if piece.NumInteriorCorners == 0 {
				piece.Deleted = true
			} else if piece.NumInteriorCorners == 8 {
				piece.Locked = true
			}
		}
		addNeighbor := func(x, y, z int) {
			if p1, ok := pieces[spacer.SquareIndex(x, y, z)]; ok {
				p1.Neighbors[piece] = true
			}
		}
		if x > 0 {
			addNeighbor(x-1, y, z)
		}
		if x+2 < len(spacer.Xs) {
			addNeighbor(x+1, y, z)
		}
		if y > 0 {
			addNeighbor(x, y-1, z)
		}
		if y+2 < len(spacer.Ys) {
			addNeighbor(x, y+1, z)
		}
		if z > 0 {
			addNeighbor(x, y, z-1)
		}
		if z+2 < len(spacer.Zs) {
			addNeighbor(x, y, z+1)
		}
	})

	return res
}

// Subdivide doubles the resolution along the border of
// the solid.
func (r *RectScanner) Subdivide() {
	pieces := make([]*rectPiece, 0, len(r.border))
	for p := range r.border {
		pieces = append(pieces, p)
	}
	for _, p := range pieces {
		r.splitBorder(p)
	}
}

// BorderRects calls f with every rectangle on the outside
// of the border.
//
// Each rectangle is passed in counter-clockwise order, so
// using the right-hand rule will yield normals facing the
// inside of the solid.
func (r *RectScanner) BorderRects(f func(points [4]Coord3D)) {
	for p := range r.border {
		// Left and right sides.
		if p.IsSideBorder(0, false) {
			p1, p2, p3 := p.Min, p.Min, p.Min
			p1.Y = p.Max.Y
			p2.Y = p.Max.Y
			p2.Z = p.Max.Z
			p3.Z = p.Max.Z
			f([4]Coord3D{p.Min, p1, p2, p3})
		}
		if p.IsSideBorder(0, true) {
			p1, p2, p3 := p.Max, p.Max, p.Max
			p1.Z = p.Min.Z
			p2.Z = p.Min.Z
			p2.Y = p.Min.Y
			p3.Y = p.Min.Y
			f([4]Coord3D{p.Max, p1, p2, p3})
		}

		// Top and bottom sides.
		if p.IsSideBorder(1, false) {
			p1, p2, p3 := p.Min, p.Min, p.Min
			p1.Z = p.Max.Z
			p2.Z = p.Max.Z
			p2.X = p.Max.X
			p3.X = p.Max.X
			f([4]Coord3D{p.Min, p1, p2, p3})
		}
		if p.IsSideBorder(1, true) {
			p1, p2, p3 := p.Max, p.Max, p.Max
			p1.X = p.Min.X
			p2.X = p.Min.X
			p2.Z = p.Min.Z
			p3.Z = p.Min.Z
			f([4]Coord3D{p.Max, p1, p2, p3})
		}

		// Front and back sides.
		if p.IsSideBorder(2, false) {
			p1, p2, p3 := p.Min, p.Min, p.Min
			p1.X = p.Max.X
			p2.X = p.Max.X
			p2.Y = p.Max.Y
			p3.Y = p.Max.Y
			f([4]Coord3D{p.Min, p1, p2, p3})
		}
		if p.IsSideBorder(2, true) {
			p1, p2, p3 := p.Max, p.Max, p.Max
			p1.Y = p.Min.Y
			p2.Y = p.Min.Y
			p2.X = p.Min.X
			p3.X = p.Min.X
			f([4]Coord3D{p.Max, p1, p2, p3})
		}
	}
}

// Mesh creates a mesh for the border.
func (r *RectScanner) Mesh() *Mesh {
	m := NewMesh()
	r.BorderRects(func(points [4]Coord3D) {
		m.Add(&Triangle{points[0], points[2], points[1]})
		m.Add(&Triangle{points[0], points[3], points[2]})
	})
	return m
}

func (r *RectScanner) splitBorder(rp *rectPiece) {
	delete(r.border, rp)
	for n := range rp.Neighbors {
		delete(n.Neighbors, rp)
	}

	var newPieces []*rectPiece

	mid := rp.Min.Mid(rp.Max)
	for xIdx := 0; xIdx < 2; xIdx++ {
		minX := rp.Min.X
		maxX := rp.Max.X
		if xIdx == 0 {
			maxX = mid.X
		} else {
			minX = mid.X
		}
		for yIdx := 0; yIdx < 2; yIdx++ {
			minY := rp.Min.Y
			maxY := rp.Max.Y
			if yIdx == 0 {
				maxY = mid.Y
			} else {
				minY = mid.Y
			}
			for zIdx := 0; zIdx < 2; zIdx++ {
				minZ := rp.Min.Z
				maxZ := rp.Max.Z
				if zIdx == 0 {
					maxZ = mid.Z
				} else {
					minZ = mid.Z
				}

				newPiece := &rectPiece{
					Min:       Coord3D{X: minX, Y: minY, Z: minZ},
					Max:       Coord3D{X: maxX, Y: maxY, Z: maxZ},
					Neighbors: map[*rectPiece]bool{},
				}
				newPiece.CountInteriorCorners(r.solid)
				newPiece.UpdateNeighbors(rp.Neighbors)
				rp.Neighbors[newPiece] = true
				newPieces = append(newPieces, newPiece)
			}
		}
	}

	for _, p := range newPieces {
		if p.NumInteriorCorners == 0 {
			if p.TouchingLocked() {
				r.border[p] = true
			} else {
				p.Neighbors = nil
				p.Deleted = true
			}
		} else if p.NumInteriorCorners == 8 {
			if p.TouchingDeleted() {
				r.border[p] = true
			} else {
				p.Neighbors = nil
				p.Locked = true
			}
		} else {
			r.border[p] = true
		}
	}
}

type rectPiece struct {
	Min Coord3D
	Max Coord3D

	// A set of adjacent pieces.
	//
	// May be nil for locked or deleted pieces.
	Neighbors map[*rectPiece]bool

	// The number of corners inside the solid.
	NumInteriorCorners int

	// If true, this piece is definitely inside the solid
	// and is not allowed to be on the border.
	// It will not be subdivided any more, and no pieces
	// touching it may be deleted.
	Locked bool

	// If true, this piece is definitely outside the
	// solid.
	// Therefore, no pieces touching it may be locked.
	Deleted bool
}

func (r *rectPiece) CheckNeighbor(r1 *rectPiece) bool {
	for i := 0; i < 3; i++ {
		i1 := (i + 1) % 3
		i2 := (i + 2) % 3
		if r.Min.array()[i1] >= r1.Max.array()[i1] ||
			r.Min.array()[i2] >= r1.Max.array()[i2] ||
			r.Max.array()[i1] <= r1.Min.array()[i1] ||
			r.Max.array()[i2] <= r1.Min.array()[i2] {
			continue
		}
		if r.Min.array()[i] == r1.Max.array()[i] {
			return true
		} else if r.Max.array()[i] == r1.Min.array()[i] {
			return true
		}
	}
	return false
}

func (r *rectPiece) CountInteriorCorners(s Solid) {
	for _, x := range []float64{r.Min.X, r.Max.X} {
		for _, y := range []float64{r.Min.Y, r.Max.Y} {
			for _, z := range []float64{r.Min.Z, r.Max.Z} {
				if s.Contains(Coord3D{X: x, Y: y, Z: z}) {
					r.NumInteriorCorners++
				}
			}
		}
	}
}

func (r *rectPiece) UpdateNeighbors(possible map[*rectPiece]bool) {
	for n := range possible {
		if n.CheckNeighbor(r) {
			if r.Neighbors != nil {
				r.Neighbors[n] = true
			}
			if n.Neighbors != nil {
				n.Neighbors[r] = true
			}
		}
	}
}

func (r *rectPiece) TouchingLocked() bool {
	for n := range r.Neighbors {
		if n.Locked {
			return true
		}
	}
	return false
}

func (r *rectPiece) TouchingDeleted() bool {
	for n := range r.Neighbors {
		if n.Deleted {
			return true
		}
	}
	return false
}

func (r *rectPiece) IsSideBorder(axis int, max bool) bool {
	for n := range r.Neighbors {
		if n.Deleted {
			continue
		}
		if max {
			if n.Min.array()[axis] == r.Max.array()[axis] {
				return false
			}
		} else {
			if n.Max.array()[axis] == r.Min.array()[axis] {
				return false
			}
		}
	}
	return true
}

type squareSpacer struct {
	Xs []float64
	Ys []float64
	Zs []float64
}

func newSquareSpacer(s Solid, delta float64) *squareSpacer {
	var xs, ys, zs []float64
	for x := s.Min().X - delta; x <= s.Max().X+delta; x += delta {
		xs = append(xs, x)
	}
	for y := s.Min().Y - delta; y <= s.Max().Y+delta; y += delta {
		ys = append(ys, y)
	}
	for z := s.Min().Z - delta; z <= s.Max().Z+delta; z += delta {
		zs = append(zs, z)
	}
	return &squareSpacer{Xs: xs, Ys: ys, Zs: zs}
}

func (s *squareSpacer) IterateSquares(f func(x, y, z int)) {
	for z := 0; z < len(s.Zs)-1; z++ {
		for y := 0; y < len(s.Ys)-1; y++ {
			for x := 0; x < len(s.Xs)-1; x++ {
				f(x, y, z)
			}
		}
	}
}

func (s *squareSpacer) NumSquares() int {
	return (len(s.Xs) - 1) * (len(s.Ys) - 1) * (len(s.Zs) - 1)
}

func (s *squareSpacer) SquareIndex(x, y, z int) int {
	return x + y*(len(s.Xs)-1) + z*(len(s.Xs)-1)*(len(s.Ys)-1)
}

func (s *squareSpacer) CornerCoord(x, y, z int) Coord3D {
	return Coord3D{X: s.Xs[x], Y: s.Ys[y], Z: s.Zs[z]}
}

func (s *squareSpacer) IterateCorners(f func(x, y, z int)) {
	for z := range s.Zs {
		for y := range s.Ys {
			for x := range s.Xs {
				f(x, y, z)
			}
		}
	}
}

func (s *squareSpacer) NumCorners() int {
	return len(s.Xs) * len(s.Ys) * len(s.Zs)
}

func (s *squareSpacer) CornerIndex(x, y, z int) int {
	return x + y*len(s.Xs) + z*len(s.Xs)*len(s.Ys)
}

type solidCache struct {
	spacer *squareSpacer
	values []bool
}

func newSolidCache(s Solid, spacer *squareSpacer) *solidCache {
	values := make([]bool, spacer.NumCorners())
	spacer.IterateCorners(func(x, y, z int) {
		values[spacer.CornerIndex(x, y, z)] = s.Contains(spacer.CornerCoord(x, y, z))
	})
	return &solidCache{spacer: spacer, values: values}
}

func (s *solidCache) NumInteriorCorners(x, y, z int) int {
	var res int
	for k := z; k < z+2; k++ {
		for j := y; j < y+2; j++ {
			for i := x; i < x+2; i++ {
				if s.values[s.spacer.CornerIndex(i, j, k)] {
					res++
				}
			}
		}
	}
	return res
}