package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	worldWidth  = 7000
	worldHeight = 3000
)

type Surface struct {
	a, b Point
}

type World struct {
	surfaces []Surface
}

type Point struct{ x, y float64 }

func (p Point) IsInWorld() bool {
	return p.x >= 0 && p.y >= 0 && int(p.x) < worldWidth && int(p.y) < worldHeight
}

func (a Point) Lerp(b Point, step float64) Point {
	return Point{
		x: lerp(a.x, b.x, step),
		y: lerp(a.y, b.y, step),
	}
}

func lerp(start, end, ratio float64) float64 {
	return (end-start)*ratio + start
	//return start*(1-ratio) + ratio*end
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

type Game struct {
	world         World
	surfaces      []byte
	lander        []byte
	width, height int
	scale         int
}

func (g *Game) Draw(screen *ebiten.Image) {
	all := g.emptyLayer()
	copy(all, g.surfaces)
	for i, p := range g.lander {
		if p > 0 {
			all[i] = p
		}
	}
	screen.ReplacePixels(all)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.width, g.height
}

func (g *Game) White(p Point, layer []byte) {
	if p.IsInWorld() {
		p = g.WorldToGame(p)
		i := int(p.y)*g.width + int(p.x)
		layer[4*i] = 0xff
		layer[4*i+1] = 0xff
		layer[4*i+2] = 0xff
		layer[4*i+3] = 0xff
	}
}

func (g *Game) WorldToGame(p Point) Point {
	return Point{p.x / float64(g.scale), float64(g.height) - p.y/float64(g.scale)}
}
func (g *Game) GameToWorld(p Point) Point {
	return Point{p.x * float64(g.scale), p.y*float64(-g.scale) + float64(worldHeight)}
}

func (g *Game) Line(s Surface, layer []byte) {
	steps := float64(20)
	for i := float64(0); i < steps; i++ {
		g.White(Point{lerp(s.a.x, s.b.x, i/steps), lerp(s.a.y, s.b.y, i/steps)}, layer)
	}
}

func (g *Game) Update() error {

	g.lander = g.emptyLayer()

	{
		flatSurface := g.world.surfaces[5]
		target := Point{
			(flatSurface.a.x + flatSurface.b.x) / 2,
			flatSurface.a.y,
		}
		startPoint := Point{x: 6500, y: 2000}
		x, y := ebiten.CursorPosition()
		controlePoint := g.GameToWorld(Point{float64(x), float64(y)})

		for _, p := range Bezier(
			startPoint,
			controlePoint,
			target,
		) {
			g.White(p, g.lander)
		}
	}

	return nil
}

func (g *Game) emptyLayer() []byte {
	return make([]byte, 4*g.height*g.width)
}

func NewGame(scale int) *Game {
	g := &Game{
		scale:  scale,
		width:  worldWidth / scale,
		height: worldHeight / scale,
		world:  World{},
	}

	var prev Point
	surfaces := strings.Split(surfacesInput, "\n")
	for i, s := range surfaces {
		var p Point
		fmt.Sscanf(s, "%f %f", &p.x, &p.y)
		if i == 0 {
			prev = p
			continue
		}
		g.world.surfaces = append(g.world.surfaces, Surface{a: prev, b: p})
		prev = p
	}

	g.surfaces = g.emptyLayer()
	g.lander = g.emptyLayer()

	for _, s := range g.world.surfaces {
		g.Line(s, g.surfaces)
	}

	return g
}

func main() {
	scale := 5
	g := NewGame(scale)

	fmt.Printf("surfaces: %#v\n", g.world.surfaces)
	ebiten.SetWindowSize(g.width, g.height)
	ebiten.SetWindowTitle("Bezier learning with Mars Lander")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

const surfacesInput = `0 1800
300 1200
1000 1550
2000 1200
2500 1650
3700 220
4700 220
4750 1000
4700 1650
4000 1700
3700 1600
3750 1900
4000 2100
4900 2050
5100 1000
5500 500
6200 800
6999 600`
