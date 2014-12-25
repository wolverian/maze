package main

import (
	"image"
	"image/color"
	"image/color/palette"
	"image/png"
	"io"
	"log"
	"math/rand"
	"os"
)

const IMG_SIZE = 61
const ROOM_TRIES = 10

var ROOM_PARAMS = RoomParams{
	Min: Pt(5, 5),
	Max: Pt(15, 15),
}

type Material int
type Region int

const (
	Rock = iota
	Carved
)

type Grid struct {
	g        []Material
	Size     Point
	regions  []Region
	regCount Region
}

func (g *Grid) Regions() []Region {
	regs := make([]Region, 0)

	var i Region

	for i = 0; i < g.regCount; i++ {
		regs = append(regs, i)
	}

	return regs
}

func (g *Grid) NewRegion() Region {
	g.regCount++
	return g.regCount
}

func (g *Grid) Bounds() image.Rectangle {
	return image.Rect(0, 0, g.Size.X, g.Size.Y)
}

func (g *Grid) At(p Point) Material {
	return g.g[p.Y*g.Size.X+p.X]
}

func (g *Grid) RegionAt(p Point) Region {
	return g.regions[p.Y*g.Size.X+p.X]
}

func (g *Grid) SetMaterial(p Point, m Material) {
	g.g[p.Y*g.Size.X+p.X] = m
}

func (g *Grid) SetRegion(p Point, r Region) {
	g.regions[p.Y*g.Size.X+p.X] = r
}

func (g *Grid) RenderMaterials(w io.Writer) error {
	img := image.NewPaletted(g.Bounds(), palette.Plan9)
	cols := make(map[Material]color.Color)
	cols[Rock] = color.Black
	cols[Carved] = color.White
	for y := 0; y < g.Size.Y; y++ {
		for x := 0; x < g.Size.X; x++ {
			img.Set(x, y, cols[g.At(Pt(x, y))])
		}
	}
	err := png.Encode(w, img)
	return err
}

func (g *Grid) RenderRegions(img *image.Paletted) {
	mats := make(map[Material]color.Color)
	mats[Rock] = color.Black
	mats[Carved] = color.White
	for y := 0; y < g.Size.Y; y++ {
		for x := 0; x < g.Size.X; x++ {
			img.Set(x, y, palette.Plan9[g.RegionAt(Pt(x, y))%256])
		}
	}
}

type Point struct{ image.Point }

func Pt(x, y int) Point {
	pt := image.Pt(x, y)
	return Point{pt}
}

func (p Point) Add(o Point) Point {
	pt := p.Point.Add(o.Point)
	return Point{pt}
}

func (p Point) AddDir(d direction) Point {
	return p.Add(*d.Point)
}

func (p Point) Mul(i int) Point {
	pt := p.Point.Mul(i)
	return Point{pt}
}

type RoomParams struct {
	Min, Max Point
}

type direction struct {
	*Point
}

func (d direction) Reverse() direction {
	pt := d.Point.Mul(-1)
	return direction{&pt}
}

func D(x int, y int) direction {
	pt := Pt(x, y)
	return direction{&pt}
}

var Dir = struct{ Up, Right, Down, Left direction }{D(0, -1), D(1, 0), D(0, 1), D(-1, 0)}

var Dirs = []direction{Dir.Up, Dir.Right, Dir.Down, Dir.Left}

func main() {
	build()
}

func build() {
	grid := newGrid(Pt(IMG_SIZE, IMG_SIZE))

	rooms := createRooms(grid.Bounds(), ROOM_PARAMS)

	for _, r := range rooms {
		region := grid.NewRegion()
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				grid.SetMaterial(Pt(x, y), Carved)
				grid.SetRegion(Pt(x, y), region)
			}
		}
	}

	growMaze(grid)

	//joinSomeRegions(grid)
	conns := findConnectors(grid)

	writeImageAnnotated(grid, conns, "maze.png")
}

func newGrid(size Point) *Grid {
	return &Grid{
		make([]Material, size.X*size.Y),
		size,
		make([]Region, size.X*size.Y),
		0,
	}
}

func createRooms(clip image.Rectangle, rp RoomParams) []image.Rectangle {
	rooms := make([]image.Rectangle, 1)

TryingRooms:
	for i := 0; i < ROOM_TRIES; i++ {
		y := rand.Intn(clip.Max.X/2)*2 + 1
		x := rand.Intn(clip.Max.Y/2)*2 + 1
		height := rand.Intn(rp.Max.Y/2)*2 + rp.Min.Y
		width := rand.Intn(rp.Max.X/2)*2 + rp.Min.X
		room := image.Rect(x, y, x+width, y+height)

		if !room.In(clip) {
			continue TryingRooms
		}

		for _, old := range rooms {
			if room.Overlaps(old) {
				continue TryingRooms
			}
		}

		rooms = append(rooms, room)
	}

	return rooms
}

func growMaze(grid *Grid) {
	bounds := grid.Bounds()
	region := grid.NewRegion()

	for y := bounds.Min.Y + 1; y < bounds.Max.Y; y += 2 {
		for x := bounds.Min.X + 1; x < bounds.Max.X; x += 2 {
			grow(grid, Pt(x, y), region)
		}
	}
}

func grow(grid *Grid, from Point, region Region) {
	cells := make([]Point, 0)
	cells = append(cells, from)

	i := 0
	for len(cells) > 0 {
		i++

		cell := cells[rand.Intn(len(cells))] //cells[len(cells)-1]

		unmade := make([]direction, 0)

		for _, d := range Dirs {
			if canCarve(grid, cell, d) {
				unmade = append(unmade, d)
			}
		}

		if len(unmade) > 0 {
			dir := unmade[rand.Intn(len(unmade))]
			grid.SetMaterial(cell.AddDir(dir), Carved)
			grid.SetRegion(cell.AddDir(dir), region)
			grid.SetMaterial(cell.AddDir(dir).AddDir(dir), Carved)
			grid.SetRegion(cell.AddDir(dir).AddDir(dir), region)
			cells = append(cells, cell.AddDir(dir).AddDir(dir))
		} else {
			cells = cells[1:]
		}
	}
}

func canCarve(g *Grid, from Point, dir direction) bool {
	beyond := from.AddDir(dir).AddDir(dir).AddDir(dir)
	next := from.AddDir(dir).AddDir(dir)

	return beyond.In(g.Bounds()) && g.At(next) == Rock
}

func joinSomeRegions(g *Grid) {
	for {
		regions := g.Regions()
		connectors := findConnectors(g)
		mr := regions[rand.Intn(len(regions))]
		mcs := make([]connector, 0)

		for _, c := range connectors {
			if c.a.region == mr || c.b.region == mr {
				mcs = append(mcs, c)
			}
		}

		break
	}
}

type conn struct {
	dir    direction
	region Region
}

type connector struct {
	a, b conn
	loc  Point
}

func findConnectors(g *Grid) []connector {
	bounds := g.Bounds()
	conns := make([]connector, 0)

	for y := bounds.Min.Y + 2; y < bounds.Max.Y-2; y += 1 {
		for x := bounds.Min.X + 2; x < bounds.Max.X-2; x += 1 {
			here := Pt(x, y)
			mat := g.At(here)
			if mat != Rock {
				continue
			}
			for _, dir := range Dirs {
				theOtherWay := dir.Reverse()
				a := here.AddDir(dir)
				b := here.AddDir(theOtherWay)
				ra := g.RegionAt(a)
				rb := g.RegionAt(b)

				if g.At(a) == Rock || g.At(b) == Rock {
					continue
				}

				if ra != rb {
					conns = append(conns, connector{
						a:   conn{dir: dir, region: ra},
						b:   conn{dir: theOtherWay, region: rb},
						loc: here,
					})
				}
			}
		}
	}

	return conns
}

func writeImageAnnotated(g *Grid, conns []connector, file string) {
	w, err := os.Create(file)
	defer w.Close()
	if err != nil {
		log.Fatalf("Can not create file '%s': %s\n", file, err)
	}

	//err = g.RenderMaterials(w)
	img := image.NewPaletted(g.Bounds(), palette.Plan9)
	g.RenderRegions(img)
	renderConnectors(img, conns)
	err = png.Encode(w, img)
	if err != nil {
		log.Fatalf("Can not write image to '%s': %s\n", file, err)
	}
}

func renderConnectors(img *image.Paletted, conns []connector) {
	for _, c := range conns {
		img.Set(c.loc.X, c.loc.Y, palette.Plan9[200])
	}
}
