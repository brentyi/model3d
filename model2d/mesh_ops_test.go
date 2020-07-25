package model2d

import (
	"math"
	"math/rand"
	"testing"
)

func TestMeshRepairNormals(t *testing.T) {
	mesh := MarchingSquaresSearch(&Circle{Radius: 0.3}, 0.01, 8)

	mesh1, numRepairs := mesh.RepairNormals(1e-8)
	if numRepairs > 0 {
		t.Errorf("expected 0 repairs but got: %d", numRepairs)
	}
	if !meshesEqual(mesh, mesh1) {
		t.Error("meshes are not equal")
	}

	flipped := NewMesh()
	expectedFlipped := 0
	mesh.Iterate(func(s *Segment) {
		if rand.Intn(2) == 0 {
			flipped.Add(s)
		} else {
			s1 := *s
			s1[0], s1[1] = s1[1], s1[0]
			flipped.Add(&s1)
			expectedFlipped++
		}
	})
	mesh1, numRepairs = flipped.RepairNormals(1e-8)
	if numRepairs != expectedFlipped {
		t.Errorf("expected %d repairs but got %d", expectedFlipped, numRepairs)
	}
	if !meshesEqual(mesh, mesh1) {
		t.Error("meshes are not equal")
	}
}

func TestMeshSubdivide(t *testing.T) {
	mesh := MarchingSquaresSearch(&Circle{Radius: 0.9}, 0.01, 8)
	mesh1 := mesh.Subdivide(2)

	if !mesh1.Manifold() {
		t.Fatal("non-manifold subdivided mesh")
	}
	if _, n := mesh1.RepairNormals(1e-8); n != 0 {
		t.Error("incorrect number of normals")
	}

	sdf := MeshToSDF(mesh)
	sdf1 := MeshToSDF(mesh1)
	for i := 0; i < 1000; i++ {
		c := NewCoordRandNorm()
		s1 := sdf.SDF(c)
		s2 := sdf1.SDF(c)
		if math.Abs(s1-s2) > 0.01 {
			t.Errorf("bad SDF at %v: expected %f but got %f", c, s1, s2)
		}
	}
}

func TestMeshDecimate(t *testing.T) {
	t.Run("Manifold", func(t *testing.T) {
		mesh := MarchingSquaresSearch(&Circle{Radius: 0.9}, 0.01, 8)
		mesh = mesh.Decimate(20)

		if n := len(mesh.VertexSlice()); n != 20 {
			t.Errorf("mesh had unexpected vertex count: %d", n)
		}

		if !mesh.Manifold() {
			t.Error("mesh is non-manifold")
		} else if _, n := mesh.RepairNormals(1e-5); n != 0 {
			t.Errorf("mesh had %d bad normals", n)
		}
	})
	t.Run("Correct", func(t *testing.T) {
		mesh := NewMesh()
		mesh.Add(&Segment{XY(0, 0), XY(1, 0)})
		mesh.Add(&Segment{XY(1, 0), XY(1, 1)})
		mesh.Add(&Segment{XY(1, 1), XY(0, 1)})
		mesh.Add(&Segment{XY(0, 1), XY(0, 0)})
		extraMesh := NewMesh()
		mesh.Iterate(func(s *Segment) {
			extraMesh.Add(&Segment{s[0], s.Mid()})
			extraMesh.Add(&Segment{s.Mid(), s[1]})
		})
		reduced := extraMesh.Decimate(4)
		if !meshesEqual(mesh, reduced) {
			t.Error("got unexpected reduced mesh")
		}
	})
	t.Run("NonManifold", func(t *testing.T) {
		mesh := NewMesh()
		mesh.Add(&Segment{XY(0, 0), XY(1, 0)})
		mesh.Add(&Segment{XY(1, 0), XY(2, 1)})
		mesh.Add(&Segment{XY(2, 1), XY(3, 0)})
		mesh.Add(&Segment{XY(3, 0), XY(4, 0)})
		extraMesh := NewMesh()
		mesh.Iterate(func(s *Segment) {
			extraMesh.Add(&Segment{s[0], s.Mid()})
			extraMesh.Add(&Segment{s.Mid(), s[1]})
		})
		reduced := extraMesh.Decimate(5)
		if !meshesEqual(mesh, reduced) {
			t.Error("got unexpected reduced mesh")
		}
		reduced2 := extraMesh.Decimate(0)
		lineMesh := NewMeshSegments([]*Segment{{X(0), X(4)}})
		if !meshesEqual(lineMesh, reduced2) {
			t.Error("got unexpected maximally-reduced mesh")
		}
	})
}

func meshesEqual(m1, m2 *Mesh) bool {
	seg1 := meshSegmentValues(m1)
	seg2 := meshSegmentValues(m2)
	if len(seg1) != len(seg2) {
		return false
	}
	for s, c := range seg1 {
		if seg2[s] != c {
			return false
		}
	}
	return true
}

func meshSegmentValues(m *Mesh) map[Segment]int {
	res := map[Segment]int{}
	m.Iterate(func(s *Segment) {
		res[*s]++
	})
	return res
}
