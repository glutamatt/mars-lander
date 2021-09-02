package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
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
	lander   Lander
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
	steps := 50
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
	cursor Point
	target Point
	tick   time.Ticker
}

type PathFinder struct {
	found bool
	done  map[Point]struct{}
	next  []Point
}

var brain Brain

func (b *Brain) Load(g *Game) {
	b.tick = *time.NewTicker(time.Second)
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

	g.world.lander.pos = Point{x: 6500, y: 2000}
	g.world.lander.angleDeg = 0
	g.world.lander.power = 4
}

func (b *Brain) DevPath(g *Game) {

	g.lander = g.emptyLayer()
	landerPos := g.world.lander.pos

	mid := Point{x: landerPos.x + (b.target.x-landerPos.x)/2, y: landerPos.y + (b.target.y-landerPos.y)/2}
	pathFinder := PathFinder{
		done: make(map[Point]struct{}),
	}
	pathFinder.next = append(pathFinder.next, mid)
	pathFinder.done[mid] = struct{}{}

	for !pathFinder.found && len(pathFinder.next) > 0 && len(pathFinder.done) < 100000 {
		controlePoint := pathFinder.next[0]
		pathFinder.next = pathFinder.next[1:]

		stepMeters := 50.0
		for _, f := range []func(*Point){
			func(p *Point) { p.x += stepMeters },
			func(p *Point) { p.y += stepMeters },
			func(p *Point) { p.x -= stepMeters },
			func(p *Point) { p.y -= stepMeters },
		} {
			next := controlePoint
			f(&next)
			if _, isDone := pathFinder.done[next]; !isDone {
				pathFinder.next = append(pathFinder.next, next)
				pathFinder.done[next] = struct{}{}
			}
		}

		path := Bezier(landerPos, controlePoint, b.target)
		isCrash := false
		for _, p := range path {
			if isCrash {
				break
			}
			if p.x < 0 || p.y < 0 || p.x >= worldWidth || p.y > +worldHeight {
				isCrash = true
				break
			}
			for _, s := range g.world.surfaces {
				if s.a.y == s.b.y {
					continue
				}
				if d := s.Distance(p); d < 100 {
					isCrash = true
					break
				}
			}
		}

		if !isCrash {
			pathFinder.found = true
			{
				//firstTarget = path[1]
				//deriv = Point{2 * (controlePoint.x - landerPos.x), 2 * (controlePoint.y - landerPos.y)}
				//normale = deriv.Normale()
				//scaleDebuf := .1
				//g.Line(Surface{a: landerPos, b: Point{landerPos.x + deriv.x*scaleDebuf, landerPos.y + deriv.y*scaleDebuf}}, g.lander)
			}

			g.White(controlePoint, g.lander)
			g.White(landerPos, g.lander)
			for _, p := range path {
				g.White(p, g.lander)
			}
		}
	}
}

func (p Point) Normale() Point {
	div := math.Sqrt(p.x*p.x + p.y*p.y)
	return Point{p.x / div, p.y / div}
}

func (p Point) Angle(b Point) float64 {
	return math.Atan2(b.y, b.y) - math.Atan2(p.y, p.y)
}

func (g *Game) Update() error {

	select {
	case <-brain.tick.C:
		{
			x, y := ebiten.CursorPosition()
			curPos := g.GameToWorld(Point{float64(x), float64(y)})

			if curPos != brain.cursor {
				brain.cursor = curPos
				//g.world.lander.pos = curPos
				g.world.lander.angleDeg = math.Max(-90, math.Min(90, g.world.lander.angleDeg+math.Max(-15, math.Min(15, (g.world.lander.pos.x-curPos.x)*15/1000))))
				fmt.Printf("current angle: %.0f\n", g.world.lander.angleDeg)
				if rand.Float32() > .7 {
					g.world.lander.power = 3
				} else {
					g.world.lander.power = 4
				}
			}
		}
		brain.DevPath(g)
		//brain.DevPilote(g, target)
	default:

	}

	g.world.lander.Simulate(time.Second / time.Duration(ebiten.MaxTPS()))
	g.White(g.world.lander.pos, g.lander)

	return nil
}

func (b *Brain) DevPilote(g *Game, target Point) {
	type solution struct {
		power int
		angle int
	}

	var best solution
	minDiff := math.MaxFloat64

	for _, power := range []int{0, 1, 2, 3, 4} {
		for _, diffAngle := range []int{-15, -12, -10, -8, -5, -2, 0, 2, 5, 8, 10, 12, 15} {

			trying := solution{power: power, angle: diffAngle}

			try := g.world.lander
			try.angleDeg += float64(diffAngle)
			try.angleDeg = math.Max(-90, math.Min(90, try.angleDeg))
			try.power = power
			try.Simulate(2 * time.Second)

			diff := math.Abs(try.pos.x-target.x) + 100*math.Abs(try.pos.y-target.y)

			//fmt.Printf("trying %#v => diff %.2f\n", trying, diff)

			if diff < minDiff {
				best = trying
				minDiff = diff
			}
		}
	}

	//xNorm := pathNorm.x - landerNorm.x
	//yNorm := pathNorm.y - landerNorm.y

	//powerTarget := math.Max(0, math.Min(4, 4.0*derivPath.y+4))
	//angleTarget := math.Max(-15, math.Min(15, 15.0*xNorm))
	g.world.lander.power = best.power
	g.world.lander.angleDeg = math.Max(-90, math.Min(90, g.world.lander.angleDeg+float64(best.angle)))
	//g.world.lander.angleDeg = math.Max(-90, math.Min(90, g.world.lander.angleDeg+angleTarget))

	//fmt.Printf("y: derivPath %.1f => power %.2f\n", derivPath.y, powerTarget)

	fmt.Printf("best: %v\n", best)
	//fmt.Printf("landerNorm: %v\n", landerNorm)
	//fmt.Printf("diffNorm: %v %v\n", xNorm, yNorm)

	//\nlanderNorm %v target angle %.0f power: %.0f\n", pathNorm, landerNorm, angleTarget, powerTarget)
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
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

type Lander struct {
	pos      Point
	speed    Point
	angleDeg float64
	power    int
}

var gravity = Point{x: 0, y: -3.711}

func (l *Lander) Simulate(t time.Duration) {

	angle := l.angleDeg*math.Pi/180 + math.Pi/2

	power := Point{
		x: float64(l.power) * math.Cos(angle),
		y: float64(l.power) * math.Sin(angle),
	}

	l.speed = Point{
		l.speed.x + power.x + gravity.x,
		l.speed.y + power.y + gravity.y,
	}

	sec := t.Seconds()
	l.pos.x += l.speed.x * sec
	l.pos.y += l.speed.y * sec
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
