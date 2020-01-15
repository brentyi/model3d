package main

import (
	"github.com/unixpickle/model3d"
	"github.com/unixpickle/model3d/toolbox3d"
)

func CrankBoltMesh() *model3d.Mesh {
	solid := CrankBoltSolid()
	mesh := model3d.SolidToMesh(solid, 0.005, 0, -1, 5)
	return mesh
}

func CrankBoltSolid() model3d.Solid {
	return model3d.StackedSolid{
		&model3d.CylinderSolid{
			P2:     model3d.Coord3D{Z: CrankBoltThickness},
			Radius: CrankBoltRadius,
		},
		&model3d.CylinderSolid{
			P2:     model3d.Coord3D{Z: SpineThickness + PoleExtraLength},
			Radius: PoleRadius,
		},
		&toolbox3d.ScrewSolid{
			P2:         model3d.Coord3D{Z: GearThickness},
			Radius:     ScrewRadius - ScrewSlack,
			GrooveSize: ScrewGrooveSize,
		},
	}
}