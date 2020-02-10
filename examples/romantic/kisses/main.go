package main

import (
	"github.com/unixpickle/model3d"
	"github.com/unixpickle/model3d/model2d"
)

const (
	PrintSize      = 4
	PrintThickness = 0.3
	TextThickness  = 0.1
)

func main() {
	img := model2d.MustReadBitmap("text.png", nil).FlipY()
	bmpSolid := model2d.BitmapToSolid(img)
	bmpSolid = model2d.ScaleSolid(bmpSolid, PrintSize/float64(img.Width))
	fullSolid := model3d.JoinedSolid{
		&TextSolid{
			Text: bmpSolid,
		},
		HersheyKissSolid{Center: model3d.Coord3D{X: 0.8, Y: PrintSize - 0.9,
			Z: PrintThickness}},
		HersheyKissSolid{Center: model3d.Coord3D{X: PrintSize / 2, Y: PrintSize - 0.7,
			Z: PrintThickness}},
		HersheyKissSolid{Center: model3d.Coord3D{X: PrintSize - 0.8, Y: PrintSize - 0.9,
			Z: PrintThickness}},
	}
	m := model3d.SolidToMesh(fullSolid, 0.01, 0, -1, 10)
	m = m.FlattenBase(0)
	m.SaveGroupedSTL("kiss.stl")
	model3d.SaveRandomGrid("rendering.png", model3d.MeshToCollider(m), 3, 3, 300, 300)
}

type TextSolid struct {
	Text model2d.Solid
}

func (t *TextSolid) Min() model3d.Coord3D {
	return model3d.Coord3D{X: 0, Y: 0, Z: 0}
}

func (t *TextSolid) Max() model3d.Coord3D {
	return model3d.Coord3D{X: PrintSize, Y: PrintSize, Z: TextThickness + PrintThickness}
}

func (t *TextSolid) Contains(c model3d.Coord3D) bool {
	if !model3d.InSolidBounds(t, c) {
		return false
	}
	if !t.Text.Contains(c.Coord2D()) {
		return c.Z < PrintThickness
	} else {
		return true
	}
}