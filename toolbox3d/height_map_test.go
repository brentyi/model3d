package toolbox3d

import (
	"math"
	"math/rand"
	"testing"

	"github.com/unixpickle/model3d/model2d"
)

func TestHeigthMapInterp(t *testing.T) {
	hm := createRandomizedHeightMap()
	for i := 0; i < 1000; i++ {
		c1 := model2d.NewCoordRandBounds(
			hm.Min.Sub(model2d.XY(0.05, 0.05)),
			hm.Max.Add(model2d.XY(0.05, 0.05)),
		)
		c2 := c1.Add(model2d.NewCoordRandUniform().Scale(1e-6))
		h1 := math.Sqrt(hm.HeightSquaredAt(c1))
		h2 := math.Sqrt(hm.HeightSquaredAt(c2))
		if math.Abs(h1-h2) > 1e-4 {
			t.Errorf("going from %v to %v resulted in heights %f, %f", c1, c2, h1, h2)
		}
	}
}

func TestHeightMapAdd(t *testing.T) {
	h1 := createRandomizedHeightMap()
	h2 := h1.Copy()
	hAdd := createRandomizedHeightMap()

	h1.AddHeightMap(hAdd)
	hAdd.Min = hAdd.Min.Add(model2d.XY(1e-8, -1e-8))
	hAdd.Max = hAdd.Max.Add(model2d.XY(1e-8, -1e-8))
	h2.AddHeightMap(hAdd)

	for i, x := range h1.Data {
		a := h2.Data[i]
		if math.Abs(x-a) > 1e-4 {
			t.Fatalf("unexpected interpolation: got %f but expected %f", a, x)
		}
	}
}

func createRandomizedHeightMap() *HeightMap {
	result := NewHeightMap(model2d.XY(0.1, 0.2), model2d.XY(0.3, 0.7), 1000)
	for i := 0; i < rand.Intn(100)+10; i++ {
		center := model2d.NewCoordRandBounds(result.Min, result.Max)
		result.AddSphere(center, rand.Float64()*0.05)
	}
	return result
}