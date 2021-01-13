// hitbox
package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	_ "net/http/pprof"
	"runtime"

	"time"

	"github.com/veandco/go-sdl2/sdl"
)

type HitBox struct {
	VList      [4]sdl.Point
	AxisList   [2]sdl.Point
	angle      float64
	x, y, w, h int32
}

func RotatePoint(p, c sdl.Point, angle float64) sdl.Point {
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
	return sdl.Point{X: newX, Y: newY}

}

func NewHitBox(x, y, W, H int32) *HitBox {
	box := &HitBox{
		VList: [4]sdl.Point{
			sdl.Point{X: x, Y: y},
			sdl.Point{X: x + W, Y: y},
			sdl.Point{X: x + W, Y: y + H},
			sdl.Point{X: x, Y: y + H},
		},
		w: W,
		h: H,
		x: x,
		y: y,
	}
	box.calAxis()
	return box
}

func (box HitBox) String() string {
	s := ""
	for i, v := range box.VList[:] {
		s += fmt.Sprintf("V%d %v\n", i, v)
	}
	return s
}

func (box *HitBox) Draw(red *sdl.Renderer) {
	red.DrawLines(append(box.VList[:], box.VList[0]))

}
func (box *HitBox) RestoreToNormalPostion() {
	box.VList = [4]sdl.Point{
		sdl.Point{X: box.x, Y: box.y},
		sdl.Point{X: box.x + box.w, Y: box.y},
		sdl.Point{X: box.x + box.w, Y: box.y + box.h},
		sdl.Point{X: box.x, Y: box.y + box.h},
	}

}
func (box *HitBox) Move(x, y int32) {
	box.x = x
	box.y = y
	box.RotateAroundCenter(box.angle)
}

func (box *HitBox) RotateAroundCenter(angle float64) {
	defer box.calAxis()
	defer func() {
		box.angle = angle
	}()
	box.RestoreToNormalPostion()
	centralpoint := sdl.Point{
		X: (box.VList[2].X-box.VList[0].X)/2 + box.VList[0].X,
		Y: (box.VList[2].Y-box.VList[0].Y)/2 + box.VList[0].Y,
	}
	for i, p := range box.VList[:] {
		box.VList[i] = RotatePoint(p, centralpoint, angle)
	}

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
	lowestY := box.VList[0].Y
	n := 0
	for i := 1; i < 4; i++ {
		if lowestY > box.VList[i].Y {
			n = i
			lowestY = box.VList[i].Y
		}
	}
	return n
}

func (box *HitBox) calAxis() {
	top := box.getTopPoint()
	box.AxisList[0].X = box.VList[top].X - box.VList[box.leftPoint(top)].X
	box.AxisList[0].Y = box.VList[top].Y - box.VList[box.leftPoint(top)].Y
	box.AxisList[1].X = box.VList[top].X - box.VList[box.rightPoint(top)].X
	box.AxisList[1].Y = box.VList[top].Y - box.VList[box.rightPoint(top)].Y
	// log.Printf("axis are %+v", box.AxisList)
}

func (box *HitBox) GetMinMaxProjectVals(axis sdl.Point) (min, max float64) {
	for i, p := range box.VList {
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

func (box *HitBox) Collide(B *HitBox) bool {
	for _, axis := range box.AxisList {
		ownmin, ownmax := box.GetMinMaxProjectVals(axis)
		othermin, othermax := B.GetMinMaxProjectVals(axis)
		if isOverlap(ownmin, ownmax, othermin, othermax) == false {
			return false
		}
	}
	for _, axis := range B.AxisList {
		ownmin, ownmax := box.GetMinMaxProjectVals(axis)
		othermin, othermax := B.GetMinMaxProjectVals(axis)
		if isOverlap(ownmin, ownmax, othermin, othermax) == false {
			return false
		}
	}
	return true
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

func getPorjectVal(p, axis sdl.Point) float64 {
	t := float64((p.X*axis.X + p.Y*axis.Y)) / (math.Pow(float64(axis.X), 2) + math.Pow(float64(axis.Y), 2))
	projx := t * float64(axis.X)
	projy := t * float64(axis.Y)
	return projx*float64(axis.X) + projy*float64(axis.Y)
}

func run() {
	var ww int32 = 1200
	var wh int32 = 1200
	var window *sdl.Window
	var err error
	var render *sdl.Renderer
	ticker := time.NewTicker(10 * time.Millisecond)
	window, err = sdl.CreateWindow("test", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		ww, wh, sdl.WINDOW_SHOWN)
	if err != nil {
		log.Fatal(err)
	}

	render, err = sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		log.Fatal(err)
	}

	defer window.Destroy()
	box1 := NewHitBox(100, 100, 100, 50)
	box1.RotateAroundCenter(30.0)
	box2 := NewHitBox(150, 100, 100, 50)
	for {
		select {
		case <-ticker.C:
			for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
				switch event.(type) {
				case *sdl.QuitEvent:
					println("Quit")
					return
				case *sdl.MouseMotionEvent:
					evt := event.(*sdl.MouseMotionEvent)
					box1.Move(evt.X, evt.Y)
					if box1.Collide(box2) {
						log.Printf("collisoned")
					}
				}
			}
			sdl.Do(func() {
				render.SetDrawColor(0, 0, 0, 1)
				render.Clear()
				render.SetDrawColor(255, 0, 0, 1)
				box1.Draw(render)
				render.SetDrawColor(255, 255, 0, 1)
				box2.Draw(render)

				render.Present()
			})

		}
	}

}

func main() {
	if false {
		runtime.SetBlockProfileRate(1000000000)
		go func() {
			log.Println(http.ListenAndServe("0.0.0.0:6060", nil))
		}()
	}
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	sdl.Main(run)

}
