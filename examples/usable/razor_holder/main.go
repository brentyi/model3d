package main

import (
	"io/ioutil"

	"github.com/unixpickle/model3d"
)

const (
	Width        = 2.5
	StickyHeight = 1.2
	HolderHeight = 0.3
	HolderLength = 1.0
	Thickness    = 0.2

	GapWidth = 0.75
)

func main() {
	solid := &model3d.SubtractedSolid{
		Positive: model3d.JoinedSolid{
			&model3d.RectSolid{
				MaxVal: model3d.Coord3D{X: Width, Y: HolderLength, Z: Thickness},
			},
			&model3d.RectSolid{
				MaxVal: model3d.Coord3D{X: Width, Y: Thickness, Z: StickyHeight},
			},
			&model3d.RectSolid{
				MinVal: model3d.Coord3D{X: 0, Y: HolderLength, Z: 0},
				MaxVal: model3d.Coord3D{X: Width, Y: HolderLength + Thickness,
					Z: Thickness + HolderHeight},
			},
		},
		Negative: &model3d.RectSolid{
			MinVal: model3d.Coord3D{X: (Width - GapWidth) / 2, Y: Thickness * 2},
			MaxVal: model3d.Coord3D{X: (Width + GapWidth) / 2, Y: Thickness + HolderLength,
				Z: StickyHeight},
		},
	}
	mesh := model3d.MarchingCubesSearch(solid, 0.01, 8)
	ioutil.WriteFile("razor_holder.stl", mesh.EncodeSTL(), 0755)
	model3d.SaveRandomGrid("rendering.png", model3d.MeshToCollider(mesh), 3, 3, 200, 200)
}
