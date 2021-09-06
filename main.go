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

func (s Surface) Dot(v Surface) float64 {
	return Point{s.b.x - s.a.x, s.b.y - s.a.y}.Dot(Point{v.b.x - v.a.x, v.b.y - v.a.y})
}

func (p Point) Dot(t Point) float64 {
	return (p.x)*(t.x) + (p.y)*(t.y)
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
}

func (b *Brain) DevCommand(g *Game, targetPoint Point) (int, float64) {

	type Command struct {
		power int
		angle float64
	}

	//bestCommand
	var best Command
	bestSimilarity := -1.0

	//	scale := 0.1
	//	//target := Point{
	//	//	x: lerp(g.world.lander.pos.x, targetPoint.x, scale),
	//	//	y: lerp(g.world.lander.pos.y, targetPoint.y, scale),
	//	//}
	//	target := Surface{targetPoint, Point{
	//		x: lerp(g.world.lander.pos.x, targetPoint.x, scale),
	//		y: lerp(g.world.lander.pos.y, targetPoint.y, scale),
	//	}}

	//g.Line(target, g.lander)

	vTarget := Point{
		x: targetPoint.x - g.world.lander.pos.x,
		y: targetPoint.x - g.world.lander.pos.y,
	}.Normale()

	g.Line(Surface{Point{vTarget.x + worldWidth/2, vTarget.y + worldHeight/2}, Point{worldWidth / 2, worldHeight / 2}}, g.lander)

	for _, p := range []int{0, 1, 2, 3, 4} {
		for _, a := range []float64{0, 30, 60, 90, -30, -60, -90, 5, 10, 15, -5, -10, -15} {
			simul := g.world.lander

			for i := 0; i < 10; i++ {
				simul.Command(a, p)
				simul.Physics(time.Second / 5)
			}

			normaleSpeed := simul.speed.Normale()
			similarity := vTarget.Dot(normaleSpeed)

			//fmt.Printf("similarity %.3f\n", similarity)

			//fmt.Printf("dist: %.0f for %v\n", dist, Command{p, a})
			if similarity > bestSimilarity {
				g.Line(Surface{Point{normaleSpeed.x + worldWidth/2, normaleSpeed.y + worldHeight/2}, Point{worldWidth / 2, worldHeight / 2}}, g.lander)
				//	println("BETTER")
				//g.Line(Surface{simul.pos, target}, g.lander)
				best = Command{p, a}
				bestSimilarity = similarity
			}
		}
	}

	return best.power, best.angle
}

func (b *Brain) DevPath(g *Game) Point {

	g.lander = g.emptyLayer()
	landerPos := g.world.lander.pos

	mid := Point{x: lerp(landerPos.x, b.target.x, .5), y: lerp(landerPos.y, b.target.y, .5)}
	g.White(mid, g.lander)
	pathFinder := PathFinder{done: make(map[Point]struct{})}
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

		//g.White(controlePoint, g.lander)
		path := Bezier(landerPos, controlePoint, b.target)
		isCrash := false
		for _, p := range path {
			if isCrash {
				break
			}
			if p.x < 0 || p.y < 0 || p.x >= worldWidth || p.y > worldHeight || p.y < b.target.y {
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
			g.Line(Surface{a: landerPos, b: controlePoint}, g.lander)
			g.White(controlePoint, g.lander)
			g.White(landerPos, g.lander)
			for _, p := range path {
				g.White(p, g.lander)
			}
			return controlePoint
		}
	}

	return mid
}

func (p Point) Normale() Point {
	div := math.Sqrt(p.x*p.x+p.y*p.y) / 1000
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
				g.world.lander.pos = curPos

			}
		}

		targetPoint := brain.DevPath(g)
		power, angle := brain.DevCommand(g, targetPoint)
		fmt.Printf("SET COMMAND power: %d, angle: %f\n", power, angle)
		g.world.lander.Command(angle, power)
		fmt.Printf("power %d ; angle %.1f ; speed %v\n", g.world.lander.power, g.world.lander.angleDeg, g.world.lander.speed)
	default:

	}

	g.world.lander.Physics(time.Second / time.Duration(ebiten.MaxTPS()))
	g.White(g.world.lander.pos, g.lander)

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

func (l *Lander) Command(angle float64, power int) {
	deltaAngle := math.Max(-15, math.Min(15, angle-l.angleDeg))
	l.angleDeg = math.Max(-90, math.Min(90, l.angleDeg+deltaAngle))

	if power > l.power && l.power < 4 {
		l.power++
	}
	if power < l.power && l.power > 0 {
		l.power--
	}
}

var gravity = Point{x: 0, y: -3.711}

func (l *Lander) Physics(t time.Duration) {

	angle := l.angleDeg*math.Pi/180 + math.Pi/2

	power := Point{
		x: float64(l.power) * math.Cos(angle),
		y: float64(l.power) * math.Sin(angle),
	}

	sec := t.Seconds()

	l.speed = Point{
		l.speed.x + (power.x+gravity.x)*sec,
		l.speed.y + (power.y+gravity.y)*sec,
	}

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
