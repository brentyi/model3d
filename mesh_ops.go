package model3d

import "math"

// Blur creates a new mesh by moving every vertex closer
// to its connected vertices.
//
// The rate argument specifies how much the vertex should
// be moved, 0 being no movement and 1 being the most.
// If multiple rates are passed, then multiple iterations
// of the algorithm are performed in succession.
func (m *Mesh) Blur(rates ...float64) *Mesh {
	capacity := len(m.triangles) * 3
	if m.vertexToTriangle != nil {
		capacity = len(m.vertexToTriangle)
	}
	coordToIdx := make(map[Coord3D]int, capacity)
	coords := make([]Coord3D, 0, capacity)
	neighbors := make([][]int, 0, capacity)
	m.Iterate(func(t *Triangle) {
		var indices [3]int
		for i, c := range t {
			if idx, ok := coordToIdx[c]; !ok {
				indices[i] = len(coords)
				coordToIdx[c] = len(coords)
				coords = append(coords, c)
				neighbors = append(neighbors, []int{})
			} else {
				indices[i] = idx
			}
		}
		for _, idx1 := range indices {
			for _, idx2 := range indices {
				if idx1 == idx2 {
					continue
				}
				var found bool
				for _, n := range neighbors[idx1] {
					if n == idx2 {
						found = true
						break
					}
				}
				if !found {
					neighbors[idx1] = append(neighbors[idx1], idx2)
				}
			}
		}
	})

	for _, rate := range rates {
		newCoords := make([]Coord3D, len(coords))
		for i, c := range coords {
			neighborAvg := Coord3D{}
			for _, c1 := range neighbors[i] {
				neighborAvg = neighborAvg.Add(coords[c1])
			}
			neighborAvg = neighborAvg.Scale(1 / float64(len(neighbors[i])))
			newPoint := neighborAvg.Scale(rate).Add(c.Scale(1 - rate))
			newCoords[i] = newPoint
		}
		coords = newCoords
	}

	m1 := NewMesh()
	m.Iterate(func(t *Triangle) {
		t1 := *t
		for i, c := range t1 {
			t1[i] = coords[coordToIdx[c]]
		}
		m1.Add(&t1)
	})
	return m1
}

// Repair finds vertices that are close together and
// combines them into one.
//
// The epsilon argument controls how close points have to
// be. In particular, it sets the approximate maximum
// distance across all dimensions.
func (m *Mesh) Repair(epsilon float64) *Mesh {
	hashToClass := map[Coord3D]*equivalenceClass{}
	allClasses := map[*equivalenceClass]bool{}
	for c := range m.getVertexToTriangle() {
		hashes := make([]Coord3D, 0, 8)
		classes := make(map[*equivalenceClass]bool, 8)
		for i := 0.0; i <= 1.0; i += 1.0 {
			for j := 0.0; j <= 1.0; j += 1.0 {
				for k := 0.0; k <= 1.0; k += 1.0 {
					hash := Coord3D{
						X: math.Round(c.X/epsilon) + i,
						Y: math.Round(c.Y/epsilon) + j,
						Z: math.Round(c.Z/epsilon) + k,
					}
					hashes = append(hashes, hash)
					if class, ok := hashToClass[hash]; ok {
						classes[class] = true
					}
				}
			}
		}
		if len(classes) == 0 {
			class := &equivalenceClass{
				Elements:  []Coord3D{c},
				Hashes:    hashes,
				Canonical: c,
			}
			for _, hash := range hashes {
				hashToClass[hash] = class
			}
			allClasses[class] = true
			continue
		}
		newClass := &equivalenceClass{
			Elements:  []Coord3D{c},
			Hashes:    hashes,
			Canonical: c,
		}
		for class := range classes {
			delete(allClasses, class)
			newClass.Elements = append(newClass.Elements, class.Elements...)
			for _, hash := range class.Hashes {
				var found bool
				for _, hash1 := range newClass.Hashes {
					if hash1 == hash {
						found = true
						break
					}
				}
				if !found {
					newClass.Hashes = append(newClass.Hashes, hash)
				}
			}
		}
		for _, hash := range newClass.Hashes {
			hashToClass[hash] = newClass
		}
		allClasses[newClass] = true
	}

	coordToClass := map[Coord3D]*equivalenceClass{}
	for class := range allClasses {
		for _, c := range class.Elements {
			coordToClass[c] = class
		}
	}

	return m.MapCoords(func(c Coord3D) Coord3D {
		return coordToClass[c].Canonical
	})
}

// NeedsRepair checks if every edge touches exactly two
// triangles. If not, NeedsRepair returns true.
func (m *Mesh) NeedsRepair() bool {
	for t := range m.triangles {
		for i := 0; i < 3; i++ {
			p1 := t[i]
			p2 := t[(i+1)%3]
			if len(m.Find(p1, p2)) != 2 {
				return true
			}
		}
	}
	return false
}

// An equivalenceClass stores a set of points which share
// hashes. It is used for Repair to group vertices.
type equivalenceClass struct {
	Elements  []Coord3D
	Hashes    []Coord3D
	Canonical Coord3D
}

// EliminateEdges creates a new mesh by iteratively
// removing edges according to the function f.
//
// The f function takes the current new mesh and a line
// segment, and returns true if the segment should be
// removed.
func (m *Mesh) EliminateEdges(f func(tmp *Mesh, segment Segment) bool) *Mesh {
	result := NewMesh()
	remainingSegments := map[Segment]bool{}
	m.Iterate(func(t *Triangle) {
		t1 := *t
		result.Add(&t1)
		for _, seg := range t.Segments() {
			remainingSegments[seg] = true
		}
	})
	changed := true
	for changed && len(remainingSegments) > 0 {
		changed = false
		segments := make([]Segment, 0, len(remainingSegments))
		for segment := range remainingSegments {
			segments = append(segments, segment)
		}
		for i := 0; i < len(segments); i++ {
			segment := segments[i]
			if !remainingSegments[segment] {
				continue
			}
			if !canEliminateSegment(result, segment) || !f(result, segment) {
				continue
			}
			neighbors := neighborsForSegment(result, segment)
			eliminateSegment(result, segment, neighbors, remainingSegments, &segments)
			changed = true
		}
	}
	return result
}

// EliminateCoplanar eliminates line segments which are
// touching a collection of coplanar triangles.
//
// The epsilon argument controls how close two normals
// must be for the triangles to be considered coplanar.
func (m *Mesh) EliminateCoplanar(epsilon float64) *Mesh {
	return m.EliminateEdges(func(m *Mesh, s Segment) bool {
		isFirst := true
		var normal Coord3D
		for _, p := range s {
			for _, neighbor := range m.getVertexToTriangle()[p] {
				if isFirst {
					normal = neighbor.Normal()
					isFirst = false
				} else if math.Abs(neighbor.Normal().Dot(normal)-1) > epsilon {
					return false
				}
			}
		}
		return true
	})
}

func neighborsForSegment(m *Mesh, segment Segment) map[*Triangle]int {
	v2t := m.getVertexToTriangle()
	neighbors1 := v2t[segment[0]]
	neighbors2 := v2t[segment[1]]
	neighbors := make(map[*Triangle]int, len(neighbors1)+len(neighbors2))
	for _, neighbor := range neighbors1 {
		neighbors[neighbor]++
	}
	for _, neighbor := range neighbors2 {
		neighbors[neighbor]++
	}
	return neighbors
}

func canEliminateSegment(m *Mesh, seg Segment) bool {
	v2t := m.getVertexToTriangle()
	neighbors1 := v2t[seg[0]]
	neighbors2 := v2t[seg[1]]
	otherSegs := make([]Segment, 0, len(neighbors1)+len(neighbors2))
	for _, neighbors := range [][]*Triangle{neighbors1, neighbors2} {
		for _, t := range neighbors {
			p1, p2 := t[0], t[1]
			if p1 == seg[0] || p1 == seg[1] {
				p1, p2 = p2, t[2]
			} else if p2 == seg[0] || p2 == seg[1] {
				p2 = t[2]
			}
			if p2 == seg[0] || p2 == seg[1] {
				// This triangle contains the segment and
				// will definitely be eliminated.
				continue
			}
			otherSeg := NewSegment(p1, p2)
			for _, s := range otherSegs {
				if s == otherSeg {
					// Two triangles will become duplicates.
					return false
				}
			}
			otherSegs = append(otherSegs, otherSeg)

			t1 := Triangle{p1, p2, seg[0]}
			t2 := Triangle{p1, p2, seg[1]}
			if t1.Normal().Dot(t2.Normal()) < 0 {
				return false
			}
		}
	}
	return true
}

func eliminateSegment(m *Mesh, segment Segment, neighbors map[*Triangle]int,
	remaining map[Segment]bool, allSegments *[]Segment) {
	mp := segment.Mid()
	for neighbor, count := range neighbors {
		m.Remove(neighbor)
		for _, seg := range neighbor.Segments() {
			if seg[0] == segment[0] || seg[0] == segment[1] || seg[1] == segment[0] ||
				seg[1] == segment[1] {
				delete(remaining, seg)
			}
		}
		if count != 1 {
			continue
		}
		for i, p := range neighbor {
			if p == segment[0] || p == segment[1] {
				neighbor[i] = mp
				seg1 := NewSegment(mp, neighbor[(i+1)%3])
				seg2 := NewSegment(mp, neighbor[(i+2)%3])
				remaining[seg1] = true
				remaining[seg2] = true
				*allSegments = append(*allSegments, seg1, seg2)
			}
		}
		m.Add(neighbor)
	}
}