package main

import (
	"fmt"
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

func (g *Grid) RenderRegions(w io.Writer) error {
	img := image.NewPaletted(g.Bounds(), palette.Plan9)
	mats := make(map[Material]color.Color)
	mats[Rock] = color.Black
	mats[Carved] = color.White
	fmt.Printf("regions: %s\n", len(g.regions))
	for y := 0; y < g.Size.Y; y++ {
		for x := 0; x < g.Size.X; x++ {
			fmt.Printf("region(%s)\n", y*g.Size.X+x)
			img.Set(x, y, palette.Plan9[g.RegionAt(Pt(x, y))%256])
		}
	}
	err := png.Encode(w, img)
	return err
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

type RoomParams struct {
	Min, Max Point
}

var Dir = struct {
	Up    Point
	Right Point
	Down  Point
	Left  Point
}{Pt(0, -1), Pt(1, 0), Pt(0, 1), Pt(-1, 0)}

var Dirs = []Point{Dir.Up, Dir.Right, Dir.Down, Dir.Left}

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

	joinSomeRegions(grid)

	writeImage(grid, "maze.png")
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

		unmade := make([]Point, 0)

		for _, d := range Dirs {
			if canCarve(grid, cell, d) {
				unmade = append(unmade, d)
			}
		}

		if len(unmade) > 0 {
			dir := unmade[rand.Intn(len(unmade))]
			grid.SetMaterial(cell.Add(dir), Carved)
			grid.SetRegion(cell.Add(dir), region)
			grid.SetMaterial(cell.Add(dir).Add(dir), Carved)
			grid.SetRegion(cell.Add(dir).Add(dir), region)
			cells = append(cells, cell.Add(dir).Add(dir))
		} else {
			cells = cells[1:]
		}
	}
}

func canCarve(g *Grid, from Point, to Point) bool {
	beyond := from.Add(to).Add(to).Add(to)
	next := from.Add(to).Add(to)

	return beyond.In(g.Bounds()) && g.At(next) == Rock
}

func joinSomeRegions(g *Grid) {
}

func writeImage(grid *Grid, file string) {
	w, err := os.Create(file)
	if err != nil {
		log.Fatalf("Can not create file '%s': %s\n", file, err)
	}

	//err = grid.RenderMaterials(w)
	err = grid.RenderRegions(w)
	if err != nil {
		log.Fatalf("Can not write image to '%s': %s\n", file, err)
	}
}
