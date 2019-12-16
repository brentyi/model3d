package main

import (
	"log"
	"math"
	"os"

	"github.com/unixpickle/model3d/toolbox3d"

	"github.com/unixpickle/model3d"
)

const (
	ScrewRadius  = 0.25
	ScrewGrooves = 0.05
	ScrewSlack   = 0.04
	ScrewLength  = 0.5

	StandRadius = 3.5
	TopRadius   = 3.5

	PoleThickness = 0.4
	FootRadius    = 0.8
	LegLength     = 6.0
)

func main() {
	if _, err := os.Stat("stand.stl"); os.IsNotExist(err) {
		log.Println("Creating stand...")
		mesh := model3d.SolidToMesh(StandSolid(), 0.01, 0, -1, 10)
		mesh.SaveGroupedSTL("stand.stl")
		model3d.SaveRandomGrid("stand.png", model3d.MeshToCollider(mesh), 3, 3, 300, 300)
	}

	if _, err := os.Stat("leg.stl"); os.IsNotExist(err) {
		log.Println("Creating leg...")
		mesh := model3d.SolidToMesh(LegSolid(), 0.01, 0, -1, 10)
		mesh.SaveGroupedSTL("leg.stl")
		model3d.SaveRandomGrid("leg.png", model3d.MeshToCollider(mesh), 3, 3, 300, 300)
	}

	if _, err := os.Stat("top.stl"); os.IsNotExist(err) {
		log.Println("Creating top...")
		mesh := model3d.SolidToMesh(TopSolid(), 0.01, 0, -1, 10)
		log.Println("Eliminating co-planar...")
		mesh = mesh.EliminateCoplanar(1e-8)
		mesh.SaveGroupedSTL("top.stl")
		model3d.SaveRandomGrid("top.png", model3d.MeshToCollider(mesh), 3, 3, 300, 300)
	}
}

func StandSolid() model3d.Solid {
	topCenter := model3d.Coord3D{Z: StandRadius}
	var corners [3]model3d.Coord3D
	for i := range corners {
		corners[i] = model3d.Coord3D{
			X: StandRadius * math.Cos(float64(i)*math.Pi*2/3),
			Y: StandRadius * math.Sin(float64(i)*math.Pi*2/3),
		}
	}
	return model3d.JoinedSolid{
		&model3d.SubtractedSolid{
			Positive: model3d.JoinedSolid{
				&model3d.SphereSolid{
					Center: topCenter,
					Radius: FootRadius,
				},
				&model3d.CylinderSolid{
					P1:     topCenter,
					P2:     corners[0],
					Radius: PoleThickness,
				},
				&model3d.CylinderSolid{
					P1:     topCenter,
					P2:     corners[1],
					Radius: PoleThickness,
				},
				&model3d.CylinderSolid{
					P1:     topCenter,
					P2:     corners[2],
					Radius: PoleThickness,
				},
				&model3d.SphereSolid{
					Center: corners[0],
					Radius: FootRadius,
				},
				&model3d.SphereSolid{
					Center: corners[1],
					Radius: FootRadius,
				},
				&model3d.SphereSolid{
					Center: corners[2],
					Radius: FootRadius,
				},
			},
			Negative: model3d.JoinedSolid{
				&model3d.RectSolid{
					MinVal: model3d.Coord3D{X: -StandRadius * 2, Y: -StandRadius * 2, Z: StandRadius},
					MaxVal: model3d.Coord3D{X: StandRadius * 2, Y: StandRadius * 2, Z: StandRadius * 2},
				},
				&model3d.RectSolid{
					MinVal: model3d.Coord3D{X: -StandRadius * 2, Y: -StandRadius * 2, Z: -StandRadius},
					MaxVal: model3d.Coord3D{X: StandRadius * 2, Y: StandRadius * 2, Z: 0},
				},
			},
		},
		&toolbox3d.ScrewSolid{
			P1:         topCenter,
			P2:         topCenter.Add(model3d.Coord3D{Z: ScrewLength}),
			GrooveSize: ScrewGrooves,
			Radius:     ScrewRadius - ScrewSlack,
		},
	}
}

func LegSolid() model3d.Solid {
	return &model3d.SubtractedSolid{
		Positive: model3d.JoinedSolid{
			&model3d.CylinderSolid{
				P2:     model3d.Coord3D{Z: LegLength},
				Radius: PoleThickness,
			},
			&toolbox3d.Ramp{
				Solid: &model3d.CylinderSolid{
					P2:     model3d.Coord3D{Z: FootRadius},
					Radius: FootRadius,
				},
				P1: model3d.Coord3D{Z: FootRadius},
			},
			&toolbox3d.Ramp{
				Solid: &model3d.CylinderSolid{
					P1:     model3d.Coord3D{Z: LegLength - FootRadius},
					P2:     model3d.Coord3D{Z: LegLength},
					Radius: FootRadius,
				},
				P1: model3d.Coord3D{Z: LegLength - FootRadius},
				P2: model3d.Coord3D{Z: LegLength},
			},
			&toolbox3d.ScrewSolid{
				P1:         model3d.Coord3D{Z: LegLength},
				P2:         model3d.Coord3D{Z: LegLength + ScrewLength},
				Radius:     ScrewRadius - ScrewSlack,
				GrooveSize: ScrewGrooves,
			},
		},
		Negative: &toolbox3d.ScrewSolid{
			P2:         model3d.Coord3D{Z: ScrewLength + ScrewRadius + ScrewSlack},
			Radius:     ScrewRadius,
			GrooveSize: ScrewGrooves,
			Pointed:    true,
		},
	}
}

func TopSolid() model3d.Solid {
	return &model3d.SubtractedSolid{
		Positive: &model3d.CylinderSolid{
			P2:     model3d.Coord3D{Z: ScrewLength},
			Radius: TopRadius,
		},
		Negative: &toolbox3d.ScrewSolid{
			P2:         model3d.Coord3D{Z: ScrewLength},
			Radius:     ScrewRadius,
			GrooveSize: ScrewGrooves,
		},
	}
}
