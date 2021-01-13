// Copyright 2020 Hu Jun. All rights reserved.
// This project is licensed under the terms of the MIT license.

/*Package hitbox implement collision detection for 2D game,
it detects collision between two rectangles, supports rotate and flip;
it is based on separating axis theorem (SAT).

Usage: create one or multiple HitBox to match the shape of your sprite,
call HitBox.Collide to test collision with another HitBox.
all exposed methods of HitBox is thread safe;
Cordinate system: X increases going right, Y increases going down*/
package hitbox

import (
	"math"
	"sync"
)

// Flip modes
const (
	FlipVertical = iota
	FlipHorizontal
)

// Point is the cordinate of a point
type Point struct {
	X, Y int32
}

// HitBox is represents a rectangle for collision detection
type HitBox struct {
	vList      [4]Point
	axisList   [2]Point
	angle      float64
	x, y, w, h int32
	mux        *sync.RWMutex
}

func rotatePoint(p, c Point, angle float64) Point {
	if c.X > p.X {
		angle += 180
	}
	tanval := float64(c.Y-p.Y) / float64(p.X-c.X)
	curRad := math.Atan(tanval)
	curAngle := (curRad * 180) / math.Pi
	newAngle := curAngle + angle
	longline := math.Sqrt(math.Pow(float64(c.X-p.X), 2) + math.Pow(float64(c.Y-p.Y), 2))
	//convert angle back rad
	newRad := (newAngle / 180) * math.Pi
	//get opposite line length of new triangle
	newOpLine := longline * math.Sin(newRad)
	newY := c.Y - int32(newOpLine)
	newAdjLine := longline * math.Cos(newRad)
	newX := c.X + int32(newAdjLine)
	return Point{X: newX, Y: newY}

}

// NewHitBox create a new hit box at cordinate (x,y), width W and height H
func NewHitBox(x, y, W, H int32) *HitBox {
	box := &HitBox{
		vList: [4]Point{
			Point{X: x, Y: y},
			Point{X: x + W, Y: y},
			Point{X: x + W, Y: y + H},
			Point{X: x, Y: y + H},
		},
		w:   W,
		h:   H,
		x:   x,
		y:   y,
		mux: new(sync.RWMutex),
	}
	box.calAxis()
	return box
}

// GetPoints return current cordinates of 4 points
func (box *HitBox) GetPoints() [4]Point {
	box.mux.RLock()
	defer box.mux.RUnlock()
	return box.vList
}

func (box *HitBox) restoreToNormalPostion() {
	box.vList = [4]Point{
		Point{X: box.x, Y: box.y},
		Point{X: box.x + box.w, Y: box.y},
		Point{X: box.x + box.w, Y: box.y + box.h},
		Point{X: box.x, Y: box.y + box.h},
	}

}

// Move move the hitbox to a differect cordinate
func (box *HitBox) Move(x, y int32) {
	box.mux.Lock()
	box.x = x
	box.y = y
	box.mux.Unlock()
	box.RotateAroundCenter(box.angle)
}

// Rotate rotate box around the specified point for the specified angle (degree), counter-clock wise
func (box *HitBox) Rotate(angle float64, center Point) {
	defer box.calAxis()
	defer func() {
		box.angle = angle
	}()
	box.mux.Lock()
	defer box.mux.Unlock()
	for i, p := range box.vList[:] {
		box.vList[i] = rotatePoint(p, center, angle)
	}
}

// RotateAroundCenter rotate box around its center for the specified angle (degree), counter-clock wise
func (box *HitBox) RotateAroundCenter(angle float64) {
	box.mux.Lock()
	box.restoreToNormalPostion()
	box.mux.Unlock()
	centralpoint := box.Center()
	box.Rotate(angle, centralpoint)
}
func (box *HitBox) leftPoint(i int) int {
	switch i {
	case 0:
		return 3
	default:
		return i - 1
	}
}
func (box *HitBox) rightPoint(i int) int {
	switch i {
	case 3:
		return 0
	default:
		return i + 1
	}
}

func (box *HitBox) getTopPoint() int {
	lowestY := box.vList[0].Y
	n := 0
	for i := 1; i < 4; i++ {
		if lowestY > box.vList[i].Y {
			n = i
			lowestY = box.vList[i].Y
		}
	}
	return n
}

func (box *HitBox) calAxis() {
	top := box.getTopPoint()
	box.axisList[0].X = box.vList[top].X - box.vList[box.leftPoint(top)].X
	box.axisList[0].Y = box.vList[top].Y - box.vList[box.leftPoint(top)].Y
	box.axisList[1].X = box.vList[top].X - box.vList[box.rightPoint(top)].X
	box.axisList[1].Y = box.vList[top].Y - box.vList[box.rightPoint(top)].Y
}

func (box *HitBox) getMinMaxProjectVals(axis Point) (min, max float64) {
	for i, p := range box.vList {
		if i == 0 {
			min = getPorjectVal(p, axis)
			max = min
			continue
		}
		v := getPorjectVal(p, axis)
		if v < min {
			min = v

		}
		if v > max {
			max = v
		}
	}
	return
}

// Collide return true if box collides with B, false otherwise
func (box *HitBox) Collide(B *HitBox) bool {
	box.mux.RLock()
	defer box.mux.RUnlock()
	B.mux.RLock()
	defer B.mux.RUnlock()
	for _, axis := range box.axisList {
		ownmin, ownmax := box.getMinMaxProjectVals(axis)
		othermin, othermax := B.getMinMaxProjectVals(axis)
		if isOverlap(ownmin, ownmax, othermin, othermax) == false {
			return false
		}
	}
	for _, axis := range B.axisList {
		ownmin, ownmax := box.getMinMaxProjectVals(axis)
		othermin, othermax := B.getMinMaxProjectVals(axis)
		if isOverlap(ownmin, ownmax, othermin, othermax) == false {
			return false
		}
	}
	return true
}

// Center returns box's central point
func (box *HitBox) Center() Point {
	box.mux.RLock()
	defer box.mux.RUnlock()
	p := Point{
		X: (box.vList[2].X-box.vList[0].X)/2 + box.vList[0].X,
		Y: (box.vList[2].Y-box.vList[0].Y)/2 + box.vList[0].Y,
	}
	return p
}

// Flip flip the box around point p, either vertically or horizantally
func (box *HitBox) Flip(mode int, c Point) {
	box.mux.Lock()
	defer box.mux.Unlock()
	if mode == FlipHorizontal {
		for i, p := range box.vList {
			if p.X == c.X {
				continue
			}
			if p.X < c.X {
				box.vList[i].X = c.X + (c.X - p.X)
			} else {
				box.vList[i].X = c.X - (p.X - c.X)
			}
		}
	} else {
		for i, p := range box.vList {
			if p.Y == c.Y {
				continue
			}
			if p.Y < c.Y {
				box.vList[i].Y = c.Y + (c.Y - p.Y)
			} else {
				box.vList[i].Y = c.Y - (p.Y - c.Y)
			}
		}
	}

}

// FlipAroundCenter flip the box around its center, either vertically or horizantally
func (box *HitBox) FlipAroundCenter(mode int) {
	c := box.Center()
	box.Flip(mode, c)

}

func isOverlap(min1, max1, min2, max2 float64) bool {
	if min1 < min2 {
		if max1 < min2 {
			return false
		}
		return true
	}
	if min2 < min1 {
		if max2 < min1 {
			return false
		}
		return true
	}
	return true

}

func getPorjectVal(p, axis Point) float64 {
	t := float64((p.X*axis.X + p.Y*axis.Y)) / (math.Pow(float64(axis.X), 2) + math.Pow(float64(axis.Y), 2))
	projx := t * float64(axis.X)
	projy := t * float64(axis.Y)
	return projx*float64(axis.X) + projy*float64(axis.Y)
}
