package main

import (
	"log"
	"math"

	"github.com/unixpickle/model3d"
	"github.com/unixpickle/model3d/toolbox3d"
)

const (
	Height        = 3.0
	InnerDiameter = 1.0
	OuterDiameter = 1.2

	LidHeight        = 0.4
	LidScrewHeight   = 0.2
	LidScrewDiameter = 1.1
	LidScrewSlack    = 0.03
	LidScrewGroove   = 0.05

	PreviewZ1    = 0.4
	PreviewZ2    = 2.5
	PreviewWidth = 0.1
)

func main() {
	lidSolid := &model3d.StackedSolid{
		&model3d.CylinderSolid{
			P2:     model3d.Coord3D{Z: LidHeight},
			Radius: OuterDiameter / 2,
		},
		&toolbox3d.ScrewSolid{
			P2:         model3d.Coord3D{Z: LidScrewHeight},
			Radius:     LidScrewDiameter/2 - LidScrewSlack,
			GrooveSize: LidScrewGroove,
		},
	}
	bodySolid := &model3d.SubtractedSolid{
		Positive: &model3d.CylinderSolid{
			P2:     model3d.Coord3D{Z: Height},
			Radius: OuterDiameter / 2,
		},
		Negative: model3d.JoinedSolid{
			&model3d.CylinderSolid{
				P1:     model3d.Coord3D{Z: OuterDiameter - InnerDiameter},
				P2:     model3d.Coord3D{Z: Height},
				Radius: InnerDiameter / 2,
			},
			&toolbox3d.ScrewSolid{
				P1:         model3d.Coord3D{Z: Height - (LidScrewHeight + LidScrewSlack)},
				P2:         model3d.Coord3D{Z: Height},
				Radius:     LidScrewDiameter / 2,
				GrooveSize: LidScrewGroove,
			},
			PreviewCutout{},
		},
	}

	smoother := &model3d.MeshSmoother{
		StepSize:           0.1,
		Iterations:         20,
		ConstraintDistance: 0.005,
		ConstraintWeight:   0.02,
	}

	log.Println("Creating lid mesh...")
	lidMesh := model3d.SolidToMesh(lidSolid, 0.005, 0, 0, 0)
	lidMesh = smoother.Smooth(lidMesh)
	log.Println("Saving lid mesh...")
	lidMesh.SaveGroupedSTL("qh_lid.stl")

	log.Println("Creating body mesh...")
	bodyMesh := model3d.SolidToMesh(bodySolid, 0.005, 0, 0, 0)
	bodyMesh = smoother.Smooth(bodyMesh)
	log.Println("Saving body mesh...")
	bodyMesh.SaveGroupedSTL("qh_body.stl")

	log.Println("Rendering...")
	model3d.SaveRandomGrid("rendering_lid.png", model3d.MeshToCollider(lidMesh), 3, 3, 300, 300)
	model3d.SaveRandomGrid("rendering_body.png", model3d.MeshToCollider(bodyMesh), 3, 3, 300, 300)
}

type PreviewCutout struct{}

func (p PreviewCutout) Min() model3d.Coord3D {
	return model3d.Coord3D{X: -OuterDiameter / 2, Y: -OuterDiameter / 2, Z: PreviewZ1}
}

func (p PreviewCutout) Max() model3d.Coord3D {
	return model3d.Coord3D{X: OuterDiameter / 2, Y: OuterDiameter / 2, Z: PreviewZ2}
}

func (p PreviewCutout) Contains(c model3d.Coord3D) bool {
	if !model3d.InSolidBounds(p, c) {
		return false
	}
	width := math.Min(PreviewWidth, math.Abs(c.Z-p.Max().Z))
	return math.Abs(c.X) < width
}
