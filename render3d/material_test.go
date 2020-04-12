package render3d

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/unixpickle/model3d"
)

func TestLambertMaterialSampling(t *testing.T) {
	testMaterialSampling(t, &LambertMaterial{
		DiffuseColor: Color{X: 1, Y: 0.9, Z: 0.5},
	})
}

func TestPhongMaterialSampling(t *testing.T) {
	for _, alpha := range []float64{0, 0.5, 2} {
		t.Run(fmt.Sprintf("Alpha%.1f", alpha), func(t *testing.T) {
			testMaterialSampling(t, &PhongMaterial{
				Alpha:         alpha,
				SpecularColor: Color{X: 1, Y: 0.9, Z: 0.5},
			})
		})
	}
	t.Run("Diffuse", func(t *testing.T) {
		testMaterialSampling(t, &PhongMaterial{
			Alpha:         2,
			SpecularColor: Color{X: 1, Y: 0.9, Z: 0.5},
			DiffuseColor:  Color{X: 0.3, Y: 0.2, Z: 0.5},
		})
	})
}

func TestHGMaterialBSDF(t *testing.T) {
	for _, g := range []float64{-0.9, -0.5, 0, 0.5, 0.9} {
		t.Run(fmt.Sprintf("G%.1f", g), func(t *testing.T) {
			mat := &HGMaterial{
				G:            g,
				ScatterColor: Color{X: 1, Y: 1.0, Z: 1.0},
			}
			var sum float64
			var count float64
			source := model3d.NewCoord3DRandUnit()
			for i := 0; i < 1000000; i++ {
				dest := model3d.NewCoord3DRandUnit()
				sum += mat.BSDF(model3d.Coord3D{}, source, dest).X
				count++
			}
			expectation := sum / count
			if math.Abs(expectation-1) > 1e-2 {
				t.Errorf("unexpected mean BSDF: %f", expectation)
			}
		})
	}
}

func TestHGMaterialSampling(t *testing.T) {
	for _, g := range []float64{-0.5, 0, 0.5} {
		t.Run(fmt.Sprintf("G%.1f", g), func(t *testing.T) {
			testMaterialSampling(t, &HGMaterial{
				G:            g,
				ScatterColor: Color{X: 1, Y: 0.9, Z: 0.5},
			})
		})
	}
}

func testMaterialSampling(t *testing.T, m Material) {
	sourceColorFunc := func(source model3d.Coord3D) Color {
		return Color{
			X: source.X + 2*source.Y*source.Y + 3*source.Z*source.Z*source.Z,
			Y: source.Z - source.X + source.Y,
			Z: 1,
		}
	}

	normal := model3d.NewCoord3DRandUnit()
	dest := model3d.NewCoord3DRandUnit()
	for dest.Dot(normal) < 0.1 {
		dest = model3d.NewCoord3DRandUnit()
	}

	var actual Color
	for i := 0; i < 4000000; i++ {
		source := model3d.NewCoord3DRandUnit()
		reflection := m.BSDF(normal, source, dest)
		color := reflection.Mul(sourceColorFunc(source))
		actual = actual.Add(color)
	}

	var expected Color
	gen := rand.New(rand.NewSource(rand.Int63()))
	for i := 0; i < 4000000; i++ {
		source := m.SampleSource(gen, normal, dest)
		weight := 1 / m.SourceDensity(normal, source, dest)
		reflection := m.BSDF(normal, source, dest)
		color := reflection.Mul(sourceColorFunc(source)).Scale(weight)
		expected = expected.Add(color)
	}

	if actual.Sub(expected).Norm() > actual.Norm()*0.01 {
		t.Errorf("expected %f but got %f", expected, actual)
	}
}
