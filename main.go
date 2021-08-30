package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// World represents the game state.
type World struct {
	area   []bool
	width  int
	height int
}

// NewWorld creates a new world.
func NewWorld(width, height int, maxInitLiveCells int) *World {
	w := &World{
		area:   make([]bool, width*height),
		width:  width,
		height: height,
	}
	return w
}

func (w *World) Clear() {
	w.area = make([]bool, w.width*w.height)
}

func (w *World) Line(x1, y1, x2, y2 float64) {
	for step := float64(0); step < 1; step += .01 {
		w.White(int(lerp(x1, x2, step)), int(lerp(y1, y2, step)))
	}
}

func lerp(start, end, ratio float64) float64 {
	return (end-start)*ratio + start
}

type Point struct{ x, y float64 }

func (a Point) Lerp(b Point, step float64) Point {
	return Point{
		x: lerp(a.x, b.x, step),
		y: lerp(a.y, b.y, step),
	}
}

func Bezier(points ...Point) []Point {
	result := []Point{}
	process := func(p []Point, step float64) []Point {
		res := make([]Point, len(p)-1)
		for i := 0; i < len(res); i++ {
			res[i] = p[i].Lerp(p[i+1], step)
		}
		return res
	}
	for step := float64(0); step < 1; step += .01 {
		p := process(points, step)
		for len(p) > 1 {
			p = process(p, step)
		}
		result = append(result, p[0])
	}
	return result
}

func (w *World) Bezier(points ...Point) {
	for i := len(points) - 1; i > 0; i-- {
		w.Line(points[i].x, points[i].y, points[i-1].x, points[i-1].y)
	}
	for _, p := range Bezier(points...) {
		w.White(int(p.x), int(p.y))
	}
}

func (w *World) White(x, y int) {
	if i := y*w.width + x; i < len(w.area) && i >= 0 {
		w.area[y*w.width+x] = true
	}

}

// Update game state by one tick.
func (w *World) Update() {
	x, y := ebiten.CursorPosition()
	w.Clear()
	w.Bezier(
		Point{0, float64(w.height / 2)},
		Point{float64(w.width) / 2, 0},
		Point{float64(w.width) / 2, float64(w.height / 2)},
		Point{float64(x), float64(y)},
		Point{0, float64(w.height)},
	)

}

// Draw paints current game state.
func (w *World) Draw(pix []byte) {
	for i, v := range w.area {
		if v {
			pix[4*i] = 0xff
			pix[4*i+1] = 0xff
			pix[4*i+2] = 0xff
			pix[4*i+3] = 0xff
		} else {
			pix[4*i] = 0
			pix[4*i+1] = 0
			pix[4*i+2] = 0
			pix[4*i+3] = 0
		}
	}
}

const (
	screenWidth  = 320
	screenHeight = 240
)

type Game struct {
	world  *World
	pixels []byte
}

func (g *Game) Update() error {
	g.world.Update()
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.pixels == nil {
		g.pixels = make([]byte, screenWidth*screenHeight*4)
	}
	g.world.Draw(g.pixels)
	screen.ReplacePixels(g.pixels)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	g := &Game{
		world: NewWorld(screenWidth, screenHeight, int((screenWidth*screenHeight)/10)),
	}
	ebiten.SetWindowSize(screenWidth*2, screenHeight*2)
	ebiten.SetWindowTitle("Bezize learning")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
