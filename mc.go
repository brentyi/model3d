package model3d

import (
	"math"
	"sort"

	"github.com/unixpickle/essentials"
)

// MarchingCubes turns a Solid into a surface mesh using a
// corrected marching cubes algorithm.
func MarchingCubes(s Solid, delta float64) *Mesh {
	table := mcLookupTable()

	spacer := newSquareSpacer(s, delta)
	cache := newSolidCache(s, spacer)

	mesh := NewMesh()

	spacer.IterateSquares(func(x, y, z int) {
		var intersections mcIntersections
		mask := mcIntersections(1)
		for i := 0; i < 2; i++ {
			for j := 0; j < 2; j++ {
				for k := 0; k < 2; k++ {
					x1 := x + k
					y1 := y + j
					z1 := z + i
					if cache.CornerValue(x1, y1, z1) {
						if x1 == 0 || x1 == len(spacer.Xs)-1 ||
							y1 == 0 || y1 == len(spacer.Ys)-1 ||
							z1 == 0 || z1 == len(spacer.Zs)-1 {
							panic("solid is true outside of bounds")
						}
						intersections |= mask
					}
					mask <<= 1
				}
			}
		}

		triangles := table[intersections]
		if len(triangles) > 0 {
			min := spacer.CornerCoord(x, y, z)
			max := spacer.CornerCoord(x+1, y+1, z+1)
			corners := mcCornerCoordinates(min, max)
			for _, t := range triangles {
				mesh.Add(t.Triangle(corners))
			}
		}
	})

	return mesh
}

// MarchingCubesSearch is like MarchingCubes, but applies
// an additional search step to move the vertices along
// the edges of each cube.
//
// The tightness of the triangulation will double for
// every iteration.
func MarchingCubesSearch(s Solid, delta float64, iters int) *Mesh {
	mesh := MarchingCubes(s, delta)

	if iters == 0 {
		return mesh
	}

	min := s.Min().Array()
	return mesh.MapCoords(func(c Coord3D) Coord3D {
		arr := c.Array()

		// Figure out which axis the containing edge spans.
		axis := -1
		var falsePoint, truePoint float64
		for i := 0; i < 3; i++ {
			modulus := math.Abs(math.Mod(arr[i]-min[i], delta))
			if modulus > delta/4 && modulus < 3*delta/4 {
				axis = i
				falsePoint = arr[i] - modulus
				truePoint = falsePoint + delta
				break
			}
		}
		if axis == -1 {
			panic("vertex not on edge")
		}
		if mesh.Find(c)[0].Normal().Array()[axis] > 0 {
			truePoint, falsePoint = falsePoint, truePoint
		}

		for i := 0; i < iters; i++ {
			midPoint := (falsePoint + truePoint) / 2
			arr[axis] = midPoint
			if s.Contains(NewCoord3DArray(arr)) {
				truePoint = midPoint
			} else {
				falsePoint = midPoint
			}
		}

		arr[axis] = (falsePoint + truePoint) / 2
		return NewCoord3DArray(arr)
	})
}

// mcCorner is a corner index on a cube used for marching
// cubes.
//
// Ordered as:
//
//     (0, 0, 0), (1, 0, 0), (0, 1, 0), (1, 1, 0),
//     (0, 0, 1), (1, 0, 1), (0, 1, 1), (1, 1, 1)
//
// Here is a visualization of the cube indices:
//
//         6 + -----------------------+ 7
//          /|                       /|
//         / |                      / |
//        /  |                     /  |
//     4 +------------------------+ 5 |
//       |   |                    |   |
//       |   |                    |   |
//       |   |                    |   |
//       |   | 2                  |   | 3
//       |   +--------------------|---+
//       |  /                     |  /
//       | /                      | /
//       |/                       |/
//       +------------------------+
//      0                           1
//
type mcCorner uint8

// mcCornerCoordinates gets the coordinates of all eight
// corners for a cube.
func mcCornerCoordinates(min, max Coord3D) [8]Coord3D {
	return [8]Coord3D{
		min,
		{X: max.X, Y: min.Y, Z: min.Z},
		{X: min.X, Y: max.Y, Z: min.Z},
		{X: max.X, Y: max.Y, Z: min.Z},

		{X: min.X, Y: min.Y, Z: max.Z},
		{X: max.X, Y: min.Y, Z: max.Z},
		{X: min.X, Y: max.Y, Z: max.Z},
		max,
	}
}

// mcRotation represents a cube rotation for marching
// cubes.
//
// For corner c, rotation[c] is the new corner at that
// location.
type mcRotation [8]mcCorner

// allMcRotations gets all 24 possible rotations for a
// cube in marching cubes.
func allMcRotations() []mcRotation {
	// Create a generating basis.
	zRotation := mcRotation{2, 0, 3, 1, 6, 4, 7, 5}
	xRotation := mcRotation{2, 3, 6, 7, 0, 1, 4, 5}

	queue := []mcRotation{{0, 1, 2, 3, 4, 5, 6, 7}}
	resMap := map[mcRotation]bool{queue[0]: true}
	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		resMap[next] = true
		for _, op := range []mcRotation{zRotation, xRotation} {
			rotated := op.Compose(next)
			if !resMap[rotated] {
				resMap[rotated] = true
				queue = append(queue, rotated)
			}
		}
	}

	var result []mcRotation
	for rotation := range resMap {
		result = append(result, rotation)
	}

	// Make the rotation order deterministic and fairly
	// sensible.
	sort.Slice(result, func(i, j int) bool {
		r1 := result[i]
		r2 := result[j]
		for k := range r1 {
			if r1[k] < r2[k] {
				return true
			} else if r1[k] > r2[k] {
				return false
			}
		}
		return false
	})

	return result
}

// Compose combines two rotations.
func (m mcRotation) Compose(m1 mcRotation) mcRotation {
	var res mcRotation
	for i := range res {
		res[i] = m[m1[i]]
	}
	return res
}

// ApplyCorner applies the rotation to a corner.
func (m mcRotation) ApplyCorner(c mcCorner) mcCorner {
	return m[c]
}

// ApplyTriangle applies the rotation to a triangle.
func (m mcRotation) ApplyTriangle(t mcTriangle) mcTriangle {
	var res mcTriangle
	for i, c := range t {
		res[i] = m.ApplyCorner(c)
	}
	return res
}

// ApplyIntersections applies the rotation to an
// mcIntersections.
func (m mcRotation) ApplyIntersections(i mcIntersections) mcIntersections {
	var res mcIntersections
	for c := mcCorner(0); c < 8; c++ {
		if i.Inside(c) {
			res |= 1 << m.ApplyCorner(c)
		}
	}
	return res
}

// mcTriangle is a triangle constructed out of midpoints
// of edges of a cube.
// There are 6 corners because each pair of two represents
// an edge.
//
// The triangle is ordered in counter-clockwise order when
// looked upon from the outside.
type mcTriangle [6]mcCorner

// Triangle creates a real triangle out of the mcTriangle,
// given the corner coordinates.
func (m mcTriangle) Triangle(corners [8]Coord3D) *Triangle {
	return &Triangle{
		corners[m[0]].Mid(corners[m[1]]),
		corners[m[2]].Mid(corners[m[3]]),
		corners[m[4]].Mid(corners[m[5]]),
	}
}

// mcIntersections represents which corners on a cube are
// inside of a solid.
// Each corner is a bit, and 1 means inside.
type mcIntersections uint8

// newMcIntersections creates an mcIntersections using the
// corners that are inside the solid.
func newMcIntersections(trueCorners ...mcCorner) mcIntersections {
	if len(trueCorners) > 8 {
		panic("expected at most 8 corners")
	}
	var res mcIntersections
	for _, c := range trueCorners {
		res |= (1 << c)
	}
	return res
}

// Inside checks if a corner c is true.
func (m mcIntersections) Inside(c mcCorner) bool {
	return (m & (1 << c)) != 0
}

// mcLookupTable creates a full lookup table that maps
// each mcIntersection to a set of triangles.
func mcLookupTable() [256][]mcTriangle {
	rotations := allMcRotations()
	result := map[mcIntersections][]mcTriangle{}

	for baseInts, baseTris := range baseTriangleTable {
		for _, rot := range rotations {
			newInts := rot.ApplyIntersections(baseInts)
			if _, ok := result[newInts]; !ok {
				newTris := []mcTriangle{}
				for _, t := range baseTris {
					newTris = append(newTris, rot.ApplyTriangle(t))
				}
				result[newInts] = newTris
			}
		}
	}

	var resultArray [256][]mcTriangle
	for key, value := range result {
		resultArray[key] = value
	}
	return resultArray
}

// baseTriangleTable encodes the marching cubes lookup
// table (up to rotations) from:
// "A survey of the marching cubes algorithm" (2006).
// https://cg.informatik.uni-freiburg.de/intern/seminar/surfaceReconstruction_survey%20of%20marching%20cubes.pdf
var baseTriangleTable = map[mcIntersections][]mcTriangle{
	// Case 0-5
	newMcIntersections(): []mcTriangle{},
	newMcIntersections(0): []mcTriangle{
		{0, 1, 0, 2, 0, 4},
	},
	newMcIntersections(0, 1): []mcTriangle{
		{0, 4, 1, 5, 0, 2},
		{1, 5, 1, 3, 0, 2},
	},
	newMcIntersections(0, 5): []mcTriangle{
		{0, 1, 0, 2, 0, 4},
		{5, 7, 1, 5, 4, 5},
	},
	newMcIntersections(0, 7): []mcTriangle{
		{0, 1, 0, 2, 0, 4},
		{6, 7, 3, 7, 5, 7},
	},
	newMcIntersections(1, 2, 3): []mcTriangle{
		{0, 1, 1, 5, 0, 2},
		{0, 2, 1, 5, 2, 6},
		{2, 6, 1, 5, 3, 7},
	},

	// Case 6-11
	newMcIntersections(0, 1, 7): []mcTriangle{
		// Case 2.
		{0, 4, 1, 5, 0, 2},
		{1, 5, 1, 3, 0, 2},
		// End of case 4
		{6, 7, 3, 7, 5, 7},
	},
	newMcIntersections(1, 4, 7): []mcTriangle{
		{4, 6, 4, 5, 0, 4},
		{1, 5, 1, 3, 0, 1},
		// End of case 4.
		{6, 7, 3, 7, 5, 7},
	},
	newMcIntersections(0, 1, 2, 3): []mcTriangle{
		{0, 4, 1, 5, 3, 7},
		{0, 4, 3, 7, 2, 6},
	},
	newMcIntersections(0, 2, 3, 6): []mcTriangle{
		{0, 1, 4, 6, 0, 4},
		{0, 1, 6, 7, 4, 6},
		{0, 1, 1, 3, 6, 7},
		{1, 3, 3, 7, 6, 7},
	},
	newMcIntersections(1, 2, 5, 6): []mcTriangle{
		{0, 2, 2, 3, 6, 7},
		{0, 2, 6, 7, 4, 6},
		{0, 1, 4, 5, 5, 7},
		{5, 7, 1, 3, 0, 1},
	},
	newMcIntersections(0, 2, 3, 7): []mcTriangle{
		{0, 4, 0, 1, 2, 6},
		{0, 1, 5, 7, 2, 6},
		{2, 6, 5, 7, 6, 7},
		{0, 1, 1, 3, 5, 7},
	},

	// Case 12-17
	newMcIntersections(1, 2, 3, 4): []mcTriangle{
		{0, 1, 1, 5, 0, 2},
		{0, 2, 1, 5, 2, 6},
		{2, 6, 1, 5, 3, 7},
		{4, 5, 0, 4, 4, 6},
	},
	newMcIntersections(1, 2, 4, 7): []mcTriangle{
		{0, 1, 1, 5, 1, 3},
		{0, 2, 2, 3, 2, 6},
		{4, 5, 0, 4, 4, 6},
		{5, 7, 6, 7, 3, 7},
	},
	newMcIntersections(1, 2, 3, 6): []mcTriangle{
		{0, 2, 0, 1, 4, 6},
		{0, 1, 3, 7, 4, 6},
		{0, 1, 1, 5, 3, 7},
		{4, 6, 3, 7, 6, 7},
	},
	newMcIntersections(0, 2, 3, 5, 6): []mcTriangle{
		// Case 9
		{0, 1, 4, 6, 0, 4},
		{0, 1, 6, 7, 4, 6},
		{0, 1, 1, 3, 6, 7},
		{1, 3, 3, 7, 6, 7},
		// End of case 3
		{5, 7, 1, 5, 4, 5},
	},
	newMcIntersections(2, 3, 4, 5, 6): []mcTriangle{
		{5, 7, 1, 5, 0, 4},
		{0, 4, 6, 7, 5, 7},
		{0, 2, 6, 7, 0, 4},
		{0, 2, 3, 7, 6, 7},
		{0, 2, 1, 3, 3, 7},
	},
	newMcIntersections(0, 4, 5, 6, 7): []mcTriangle{
		{1, 5, 0, 1, 0, 2},
		{0, 2, 2, 6, 1, 5},
		{1, 5, 2, 6, 3, 7},
	},

	// Case 18-22
	newMcIntersections(1, 2, 3, 4, 5, 6): []mcTriangle{
		// Inverse of case 4.
		{0, 2, 0, 1, 0, 4},
		{3, 7, 6, 7, 5, 7},
	},
	newMcIntersections(1, 2, 3, 4, 6, 7): []mcTriangle{
		{0, 2, 4, 5, 0, 4},
		{0, 2, 5, 7, 4, 5},
		{0, 2, 1, 5, 5, 7},
		{0, 1, 1, 5, 0, 2},
	},
	newMcIntersections(2, 3, 4, 5, 6, 7): []mcTriangle{
		// Inverse of case 2.
		{1, 5, 0, 4, 0, 2},
		{1, 3, 1, 5, 0, 2},
	},
	newMcIntersections(1, 2, 3, 4, 5, 6, 7): []mcTriangle{
		// Inverse of case 1.
		{0, 2, 0, 1, 0, 4},
	},
	newMcIntersections(0, 1, 2, 3, 4, 5, 6, 7): []mcTriangle{},
}

type squareSpacer struct {
	Xs []float64
	Ys []float64
	Zs []float64
}

func newSquareSpacer(s Solid, delta float64) *squareSpacer {
	var xs, ys, zs []float64
	min := s.Min()
	max := s.Max()
	for x := min.X - delta; x <= max.X+delta; x += delta {
		xs = append(xs, x)
	}
	for y := min.Y - delta; y <= max.Y+delta; y += delta {
		ys = append(ys, y)
	}
	for z := min.Z - delta; z <= max.Z+delta; z += delta {
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

func (s *squareSpacer) IndexToCorner(idx int) (int, int, int) {
	x := idx % len(s.Xs)
	idx /= len(s.Xs)
	y := idx % len(s.Ys)
	z := idx / len(s.Ys)
	return x, y, z
}

type solidCache struct {
	spacer *squareSpacer
	solid  Solid

	startZ  int
	strideZ int
	cachedZ int
	values  []bool
}

func newSolidCache(solid Solid, spacer *squareSpacer) *solidCache {
	cachedZ := essentials.MinInt(len(spacer.Zs), 10)
	strideZ := len(spacer.Xs) * len(spacer.Ys)
	cache := &solidCache{
		spacer:  spacer,
		solid:   solid,
		strideZ: strideZ,
		cachedZ: cachedZ,
		values:  make([]bool, strideZ*cachedZ),
	}
	cache.fillTailValues(cachedZ)
	return cache
}

func (s *solidCache) NumInteriorCorners(x, y, z int) int {
	var res int
	for k := z; k < z+2; k++ {
		for j := y; j < y+2; j++ {
			for i := x; i < x+2; i++ {
				if s.CornerValue(i, j, k) {
					res++
				}
			}
		}
	}
	return res
}

func (s *solidCache) CornerValue(x, y, z int) bool {
	if z >= s.startZ && z < s.startZ+s.cachedZ {
		return s.values[s.spacer.CornerIndex(x, y, z-s.startZ)]
	}

	newStart := essentials.MinInt(essentials.MaxInt(0, z-s.cachedZ/2),
		len(s.spacer.Zs)-s.cachedZ)
	shift := newStart - s.startZ
	s.startZ = newStart
	if shift < 0 || shift >= s.cachedZ {
		// Start the cache all over again.
		s.fillTailValues(s.cachedZ)
	} else {
		copy(s.values, s.values[shift*s.strideZ:])
		s.fillTailValues(shift)
	}

	return s.CornerValue(x, y, z)
}

func (s *solidCache) fillTailValues(numTail int) {
	idx := (s.cachedZ - numTail) * s.strideZ
	for _, z := range s.spacer.Zs[s.startZ+s.cachedZ-numTail:][:numTail] {
		for _, y := range s.spacer.Ys {
			for _, x := range s.spacer.Xs {
				s.values[idx] = s.solid.Contains(Coord3D{X: x, Y: y, Z: z})
				idx++
			}
		}
	}
}
