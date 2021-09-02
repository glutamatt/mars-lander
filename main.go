package main

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	worldWidth  = 7000
	worldHeight = 3000
)

type Surface struct {
	a, b Point
}

func (s Surface) Distance(p Point) float64 {
	px := s.b.x - s.a.x
	py := s.b.y - s.a.y

	norm := px*px + py*py
	u := ((p.x-s.a.x)*px + (p.y-s.a.y)*py) / norm
	if u > 1 {
		u = 1
	} else {
		if u < 0 {
			u = 0
		}
	}

	return p.Distance(Point{
		x: s.a.x + u*px,
		y: s.a.y + u*py,
	})
}

type World struct {
	surfaces []Surface
}

type Point struct{ x, y float64 }

func (p Point) IsInWorld() bool {
	return p.x >= 0 && p.y >= 0 && int(p.x) < worldWidth && int(p.y) < worldHeight
}

func (p Point) Distance(t Point) float64 {
	x, y := p.x-t.x, p.y-t.y
	return math.Sqrt(x*x + y*y)
}

func (a Point) Lerp(b Point, step float64) Point {
	return Point{
		x: lerp(a.x, b.x, step),
		y: lerp(a.y, b.y, step),
	}
}

func lerp(start, end, ratio float64) float64 {
	return (end-start)*ratio + start
}

type Path []Point

func (p Path) Distance() (d float64) {
	for i := 1; i < len(p); i++ {
		d += p[i-1].Distance(p[i])
	}
	return d
}

func Bezier(points ...Point) Path {
	result := []Point{}
	process := func(p []Point, step float64) []Point {
		res := make([]Point, len(p)-1)
		for i := 0; i < len(res); i++ {
			res[i] = p[i].Lerp(p[i+1], step)
		}
		return res
	}
	steps := 40
	for i := 0; i < steps; i++ {
		step := 1.0 / float64(steps) * float64(i)
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
		if i := int(p.y)*g.width + int(p.x); i > 0 && (4*i+3) < len(layer) {
			layer[4*i] = 0xff
			layer[4*i+1] = 0xff
			layer[4*i+2] = 0xff
			layer[4*i+3] = 0xff
		}
	}
}

func (g *Game) WorldToGame(p Point) Point {
	return Point{p.x / float64(g.scale), float64(g.height) - p.y/float64(g.scale)}
}
func (g *Game) GameToWorld(p Point) Point {
	return Point{p.x * float64(g.scale), p.y*float64(-g.scale) + float64(worldHeight)}
}

func (g *Game) Line(s Surface, layer []byte) {
	steps := float64(50)
	for i := float64(0); i < steps; i++ {
		g.White(Point{lerp(s.a.x, s.b.x, i/steps), lerp(s.a.y, s.b.y, i/steps)}, layer)
	}
}

type Brain struct {
	current Point
	target  Point
	tick    time.Ticker
	found   bool
	done    map[Point]struct{}
	next    []Point
}

var brain Brain

func (b *Brain) Load(g *Game) {
	b.tick = *time.NewTicker(time.Second / 4)
	var flatSurface Surface
	for _, s := range g.world.surfaces {
		if s.a.y == s.b.y {
			flatSurface = s
			break
		}
	}
	b.target = Point{
		(flatSurface.a.x + flatSurface.b.x) / 2,
		flatSurface.a.y,
	}

	b.current = Point{x: 6500, y: 2000}
	mid := Point{x: b.current.x + (b.target.x-b.current.x)/2, y: b.current.y + (b.target.y-b.current.y)/2}
	b.done = make(map[Point]struct{})
	b.done[mid] = struct{}{}
	b.next = append(b.next, mid)
}

func (b *Brain) DevPath(g *Game) {

	if b.found || len(b.next) == 0 {
		return
	}

	select {
	case <-b.tick.C:
	default:
		return
	}

	//x, y := ebiten.CursorPosition()
	//g.GameToWorld(Point{float64(x), float64(y)})
	controlePoint := b.next[0]
	g.White(controlePoint, g.lander)
	b.next = b.next[1:]

	{
		//draw derivative vector at t0
		//deriv := Point{2 * (controlePoint.x - startPoint.x), 2 * (controlePoint.y - startPoint.y)}
		//g.Line(Surface{a: startPoint, b: Point{x: startPoint.x + deriv.x, y: startPoint.y + deriv.y}}, g.lander)
	}

	stepMeters := 500.0
	for _, f := range []func(*Point){
		func(p *Point) { p.x += stepMeters },
		func(p *Point) { p.y += stepMeters },
		func(p *Point) { p.x -= stepMeters },
		func(p *Point) { p.y -= stepMeters },
	} {
		next := controlePoint
		f(&next)
		if _, isDone := b.done[next]; !isDone {
			b.next = append(b.next, next)
			b.done[next] = struct{}{}
		}
	}

	path := Bezier(b.current, controlePoint, b.target)
	fmt.Printf("distance: %.2f\n", path.Distance())
	for i, p := range path {
		if p.x < 0 || p.y < 0 || p.x >= worldWidth || p.y > +worldHeight {
			fmt.Printf("#%d point is out of world\n", i)
			return
		}
		for ii, s := range g.world.surfaces {
			if d := s.Distance(p); d < 100 {
				fmt.Printf("surface %d too close to #%d point (%.0f meters)\n", ii, i, d)
				return
			}
		}
		g.White(p, g.lander)
	}

	b.found = true
}

func (g *Game) Update() error {
	brain.DevPath(g)
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
	brain.Load(g)
	fmt.Printf("surfaces: %#v\n", g.world.surfaces)
	ebiten.SetWindowSize(g.width, g.height)
	ebiten.SetWindowTitle("Bezier learning with Mars Lander")
	ebiten.SetMaxTPS(10)
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
