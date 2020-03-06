package main

import (
	"log"
	"math"

	"github.com/unixpickle/model3d"
)

const (
	BodyLength = 1.1
	BodyRadius = 0.3
	BodyToNeck = 0.2

	NeckTheta  = 0.4 * math.Pi
	NeckLength = 0.6
	NeckRadius = 0.23

	HeadRadius = 0.23

	NubRadius  = 0.1
	NubXOffset = -0.3

	SnoutLargeRadius = 0.17
	SnoutSmallRadius = 0.12
	SnoutInset       = 0.05
	SnoutZOffset     = -0.08
	SnoutLength      = 0.29

	EarYOffset   = 0.12
	EarZOffset   = 0.12
	EarTheta     = 0.1 * math.Pi
	EarWidth     = 0.24
	EarHeight    = 0.4
	EarThickness = 0.06

	LegInset               = 0.15
	LegRadius              = 0.07
	LegMinZ                = -0.6
	HindLegX               = -0.2
	HindLegMuscleWidth     = 0.5
	HindLegMuscleHeight    = 0.6
	HindLegMuscleThickness = 0.4
	HindLegMuscleZ         = -0.02
	HindLegMuscleX         = -0.1
)

func main() {
	log.Println("creating body solid...")
	model := SmoothJoin(0.1, MakeBody(), MakeHeadNeck(), MakeLegs(), MakeHindLegMuscles(),
		MakeSnout(), MakeNub(), MakeEars())
	log.Println("creating mesh...")
	mesh := model3d.SolidToMesh(model, 0.01, 0, -1, 5)
	log.Println("saving...")
	mesh.SaveGroupedSTL("corgi.stl")
	log.Println("rendering...")
	model3d.SaveRandomGrid("rendering.png", model3d.MeshToCollider(mesh), 3, 3, 300, 300)
}

func MakeBody() model3d.Solid {
	return model3d.JoinedSolid{
		&model3d.CylinderSolid{
			P2:     model3d.Coord3D{X: BodyLength},
			Radius: BodyRadius,
		},
		&model3d.SphereSolid{
			Radius: BodyRadius,
		},
		&model3d.SphereSolid{
			Center: model3d.Coord3D{X: BodyLength},
			Radius: BodyRadius,
		},
	}
}

func MakeHeadNeck() model3d.Solid {
	return model3d.JoinedSolid{
		&model3d.CylinderSolid{
			P1: model3d.Coord3D{X: BodyLength},
			P2: model3d.Coord3D{X: BodyLength + NeckLength*math.Cos(NeckTheta),
				Z: NeckLength * math.Sin(NeckTheta)},
			Radius: NeckRadius,
		},
		&model3d.SphereSolid{
			Center: model3d.Coord3D{X: BodyLength + NeckLength*math.Cos(NeckTheta),
				Z: NeckLength * math.Sin(NeckTheta)},
			Radius: HeadRadius,
		},
	}
}

func MakeHindLegMuscles() model3d.Solid {
	return model3d.JoinedSolid{
		HindLegMuscleSolid{
			Center: model3d.Coord3D{X: HindLegMuscleX, Y: -BodyRadius + LegInset, Z: HindLegMuscleZ},
		},
		HindLegMuscleSolid{
			Center: model3d.Coord3D{X: HindLegMuscleX, Y: BodyRadius - LegInset, Z: HindLegMuscleZ},
		},
	}
}

func MakeLegs() model3d.Solid {
	var res model3d.JoinedSolid
	y1 := -BodyRadius + LegInset
	bottomZ := LegMinZ
	for _, x := range []float64{HindLegX, BodyLength} {
		for _, y := range []float64{y1, -y1} {
			res = append(res, &model3d.CylinderSolid{
				P1:     model3d.Coord3D{X: x, Y: y},
				P2:     model3d.Coord3D{X: x, Y: y, Z: bottomZ},
				Radius: LegRadius,
			})
		}
	}
	return res
}

func MakeSnout() model3d.Solid {
	origin := model3d.Coord3D{
		X: BodyLength + NeckLength*math.Cos(NeckTheta) + HeadRadius - SnoutInset,
		Z: NeckLength*math.Sin(NeckTheta) + SnoutZOffset,
	}
	return &SnoutSolid{
		P1: origin,
		P2: origin.Add(model3d.Coord3D{X: SnoutLength * math.Sin(NeckTheta),
			Z: -SnoutLength * math.Cos(NeckTheta)}),
	}
}

func MakeNub() model3d.Solid {
	return &model3d.SphereSolid{
		Center: model3d.Coord3D{X: NubXOffset, Z: BodyRadius - NubRadius},
		Radius: NubRadius,
	}
}

func MakeEars() model3d.Solid {
	origin := model3d.Coord3D{
		X: BodyLength + NeckLength*math.Cos(NeckTheta),
		Z: NeckLength*math.Sin(NeckTheta) + EarZOffset,
	}
	var res model3d.JoinedSolid
	for _, y := range []float64{-EarYOffset, EarYOffset} {
		yDiff := EarHeight * math.Sin(EarTheta)
		if y < 0 {
			yDiff *= -1
		}
		res = append(res, &EarSolid{
			Base: origin.Add(model3d.Coord3D{Y: y}),
			Tip:  origin.Add(model3d.Coord3D{Y: y + yDiff, Z: EarHeight * math.Cos(EarTheta)}),
		})
	}
	return res
}

type HindLegMuscleSolid struct {
	Center model3d.Coord3D
}

func (h HindLegMuscleSolid) Min() model3d.Coord3D {
	return h.Center.Add(model3d.Coord3D{X: -HindLegMuscleWidth / 2, Y: -HindLegMuscleThickness / 2,
		Z: -HindLegMuscleHeight / 2})
}

func (h HindLegMuscleSolid) Max() model3d.Coord3D {
	return h.Center.Add(model3d.Coord3D{X: HindLegMuscleWidth / 2, Y: HindLegMuscleThickness / 2,
		Z: HindLegMuscleHeight / 2})
}

func (h HindLegMuscleSolid) Contains(c model3d.Coord3D) bool {
	if !model3d.InSolidBounds(h, c) {
		return false
	}
	c = c.Sub(h.Center)
	muscleScale := model3d.Coord3D{X: 2 / HindLegMuscleWidth, Y: 2 / HindLegMuscleThickness,
		Z: 2 / HindLegMuscleHeight}
	return c.Mul(muscleScale).Norm() < 1
}

type SnoutSolid struct {
	P1 model3d.Coord3D
	P2 model3d.Coord3D
}

func (s *SnoutSolid) Min() model3d.Coord3D {
	return s.boundingCylinder().Min()
}

func (s *SnoutSolid) Max() model3d.Coord3D {
	return s.boundingCylinder().Max()
}

func (s *SnoutSolid) Contains(c model3d.Coord3D) bool {
	cyl := s.boundingCylinder()
	if !cyl.Contains(c) {
		return false
	}

	c = c.Sub(s.P1)

	diff := s.P2.Sub(s.P1)
	frac := diff.Dot(c) / diff.Dot(diff)
	if frac < 0 || frac > 1 {
		return false
	}

	b1 := model3d.Coord3D{Y: 1}
	b2 := diff.Cross(b1).Normalize()
	c2 := model3d.Coord2D{X: b1.Dot(c), Y: b2.Dot(c)}

	// Smooth tip, and make it "wide".
	c2.X /= math.Pow(1-frac, 0.3) * SnoutLargeRadius
	c2.Y /= math.Pow(1-frac, 0.3) * SnoutSmallRadius

	return c2.Norm() < 1
}

func (s *SnoutSolid) boundingCylinder() *model3d.CylinderSolid {
	return &model3d.CylinderSolid{
		P1:     s.P1,
		P2:     s.P2,
		Radius: SnoutLargeRadius,
	}
}

type EarSolid struct {
	Base model3d.Coord3D
	Tip  model3d.Coord3D
}

func (e *EarSolid) Min() model3d.Coord3D {
	return e.boundingCylinder().Min()
}

func (e *EarSolid) Max() model3d.Coord3D {
	return e.boundingCylinder().Max()
}

func (e *EarSolid) Contains(c model3d.Coord3D) bool {
	cyl := e.boundingCylinder()
	if !cyl.Contains(c) {
		return false
	}

	c = c.Sub(e.Base)

	diff := e.Tip.Sub(e.Base)
	frac := diff.Dot(c) / diff.Dot(diff)
	if frac < 0 || frac > 1 {
		return false
	}

	// Curved tip
	frac = math.Pow(1-frac, 0.3)

	xAxis := model3d.Coord3D{Y: 1}.ProjectOut(diff).Normalize()
	yAxis := diff.Cross(xAxis).Normalize()
	return math.Abs(xAxis.Dot(c)) < frac*EarWidth/2 &&
		math.Abs(yAxis.Dot(c)) < EarThickness/2
}

func (e *EarSolid) boundingCylinder() *model3d.CylinderSolid {
	return &model3d.CylinderSolid{
		P1:     e.Base,
		P2:     e.Tip,
		Radius: EarWidth / 2,
	}
}