package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"math/rand"
	"os"
	"path/filepath"

	"golang.org/x/image/vector"
)

// The palette, shared with The Forgotten Maze: one cool off-white for the
// player and the light, mint for whatever is the player's, violet for rare
// things, a muted rose for anything hostile, and slate for enemy hulls.
var (
	ink      = hex(0x0b0d12)
	text     = hex(0xe7e9ef)
	textDim  = hex(0xb4b9c6)
	accent   = hex(0x7dd3c0)
	pale     = hex(0xbfeee3)
	alt      = hex(0xa597ff)
	danger   = hex(0xe07a93)
	dangerLo = hex(0xa9566e)
	blush    = hex(0xf6d3dd)
	hull     = hex(0x4a4560)
	hullLo   = hex(0x332f45)
	hullHi   = hex(0x625c7d)
	surface  = hex(0x12161d)
)

func hex(v uint32) color.NRGBA {
	return color.NRGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 255}
}

func fade(c color.NRGBA, a float64) color.NRGBA {
	c.A = uint8(math.Round(float64(c.A) * a))
	return c
}

type pt struct{ x, y float64 }

// canvas is a picture built from flat, anti-aliased shapes.
type canvas struct {
	img *image.NRGBA
}

func newCanvas(w, h int) *canvas {
	return &canvas{img: image.NewNRGBA(image.Rect(0, 0, w, h))}
}

func (c *canvas) fill(col color.NRGBA, paths ...[]pt) {
	b := c.img.Bounds()
	r := vector.NewRasterizer(b.Dx(), b.Dy())
	for _, p := range paths {
		if len(p) == 0 {
			continue
		}
		r.MoveTo(float32(p[0].x), float32(p[0].y))
		for _, q := range p[1:] {
			r.LineTo(float32(q.x), float32(q.y))
		}
		r.ClosePath()
	}
	r.Draw(c.img, b, image.NewUniform(col), image.Point{})
}

func (c *canvas) poly(col color.NRGBA, p ...pt) { c.fill(col, p) }

func ellipsePath(cx, cy, rx, ry float64, reverse bool) []pt {
	n := int(math.Max(24, (rx+ry)*2))
	out := make([]pt, n)
	for i := range out {
		a := float64(i) / float64(n) * 2 * math.Pi
		if reverse {
			a = -a
		}
		out[i] = pt{cx + math.Cos(a)*rx, cy + math.Sin(a)*ry}
	}
	return out
}

func (c *canvas) circle(col color.NRGBA, cx, cy, r float64) {
	c.fill(col, ellipsePath(cx, cy, r, r, false))
}

func (c *canvas) ellipse(col color.NRGBA, cx, cy, rx, ry float64) {
	c.fill(col, ellipsePath(cx, cy, rx, ry, false))
}

// ring draws a circle outline of the given width, centred on radius r.
func (c *canvas) ring(col color.NRGBA, cx, cy, r, width float64) {
	c.fill(col, ellipsePath(cx, cy, r+width/2, r+width/2, false), ellipsePath(cx, cy, r-width/2, r-width/2, true))
}

// capsule is a line segment with round ends.
func (c *canvas) capsule(col color.NRGBA, x0, y0, x1, y1, width float64) {
	a := math.Atan2(y1-y0, x1-x0)
	h := width / 2
	var p []pt
	for i := 0; i <= 12; i++ {
		t := a + math.Pi/2 + math.Pi*float64(i)/12
		p = append(p, pt{x0 + math.Cos(t)*h, y0 + math.Sin(t)*h})
	}
	for i := 0; i <= 12; i++ {
		t := a - math.Pi/2 + math.Pi*float64(i)/12
		p = append(p, pt{x1 + math.Cos(t)*h, y1 + math.Sin(t)*h})
	}
	c.fill(col, p)
}

// mirror returns p reflected across the vertical line x = cx, in reverse
// order so it keeps the same winding.
func mirror(cx float64, p []pt) []pt {
	out := make([]pt, len(p))
	for i, q := range p {
		out[len(p)-1-i] = pt{2*cx - q.x, q.y}
	}
	return out
}

func (c *canvas) symmetric(col color.NRGBA, cx float64, p ...pt) {
	c.fill(col, p)
	c.fill(col, mirror(cx, p))
}

func (c *canvas) save(dir, name string) {
	f, err := os.Create(filepath.Join(dir, name+".png"))
	must(err)
	defer f.Close()
	must(png.Encode(f, c.img))
}

func buildArt(dir string) {
	ship(dir)
	flames(dir)
	bullets(dir)
	drone(dir)
	dart(dir)
	gunship(dir)
	mothership(dir)
	powerups(dir)
	shield(dir)
	spark(dir)
	explosions(dir)
	sky(dir)
}

// The player: a slim off-white arrow with swept wings, pointing up.
func ship(dir string) {
	c := newCanvas(45, 45)
	m := 22.5
	// wings
	c.symmetric(textDim, m, pt{m, 15}, pt{m + 18.5, 33}, pt{m + 18.5, 37}, pt{m + 5, 34}, pt{m, 34})
	// fuselage
	c.symmetric(text, m, pt{m, 3}, pt{m + 5, 18}, pt{m + 5.5, 38}, pt{m, 38})
	// a darker cut along each wing's trailing edge
	c.symmetric(hex(0x8b91a1), m, pt{m + 6, 33}, pt{m + 18.5, 35}, pt{m + 18.5, 37}, pt{m + 6, 35.5})
	// cockpit
	c.poly(accent, pt{m, 11}, pt{m + 2.2, 18}, pt{m, 23}, pt{m - 2.2, 18})
	// engine
	c.poly(hex(0x8b91a1), pt{m - 3.5, 38}, pt{m + 3.5, 38}, pt{m + 2.5, 41}, pt{m - 2.5, 41})
	c.save(dir, "ship")
}

// The engine flame, drawn under the ship. Two lengths make the flicker.
func flames(dir string) {
	for i, h := range []float64{11, 14} {
		c := newCanvas(15, int(h)+1)
		m := 7.5
		c.poly(fade(pale, 0.55), pt{m - 3, 0}, pt{m + 3, 0}, pt{m, h})
		c.poly(text, pt{m - 1.4, 0}, pt{m + 1.4, 0}, pt{m, h * 0.55})
		c.save(dir, []string{"flame1", "flame2"}[i])
	}
}

func bullets(dir string) {
	c := newCanvas(9, 21)
	c.capsule(accent, 4.5, 4, 4.5, 17, 3.2)
	c.capsule(pale, 4.5, 4.5, 4.5, 8, 1.4)
	c.save(dir, "bullet")

	e := newCanvas(15, 15)
	e.circle(danger, 7.5, 7.5, 4.6)
	e.circle(blush, 7.5, 7.5, 1.8)
	e.save(dir, "enemy-bullet")
}

// Drones drift down in a weave: a round slate body with a rose eye.
func drone(dir string) {
	c := newCanvas(33, 39)
	m := 16.5
	c.symmetric(hullLo, m, pt{m + 6, 22}, pt{m + 13, 30}, pt{m + 12, 35}, pt{m + 4, 27})
	c.circle(hull, m, 17, 12)
	// lower half in shadow
	c.fill(hullLo, append([]pt{{m - 12, 17}, {m + 12, 17}}, ellipseHalf(m, 17, 12, 12)...))
	c.capsule(danger, m-4, 18, m+4, 18, 3)
	c.capsule(hullHi, m-6, 8.5, m-2, 6.6, 1.6)
	c.save(dir, "drone")
}

// Darts zig-zag fast: a narrow chevron, drawn pointing down and turned to
// face where it's going.
func dart(dir string) {
	c := newCanvas(33, 33)
	m := 16.5
	c.symmetric(hull, m, pt{m, 31}, pt{m, 15}, pt{m + 5, 5}, pt{m + 13, 3})
	c.symmetric(hullLo, m, pt{m, 31}, pt{m, 22}, pt{m + 10, 7}, pt{m + 13, 3})
	c.poly(danger, pt{m, 13}, pt{m + 2.5, 19}, pt{m, 24}, pt{m - 2.5, 19})
	c.save(dir, "dart")
}

// Gunships hold back and fire aimed shots from the barrel under the hull.
func gunship(dir string) {
	c := newCanvas(78, 60)
	m := 39.0
	// side pods
	c.symmetric(hullLo, m, pt{m + 20, 14}, pt{m + 34, 18}, pt{m + 36, 38}, pt{m + 30, 46}, pt{m + 22, 40})
	c.symmetric(hullHi, m, pt{m + 31, 19}, pt{m + 34, 19}, pt{m + 35.5, 36}, pt{m + 32.5, 36})
	// body
	c.symmetric(hull, m, pt{m, 6}, pt{m + 14, 8}, pt{m + 22, 20}, pt{m + 18, 40}, pt{m + 8, 48}, pt{m, 48})
	c.symmetric(hullLo, m, pt{m, 34}, pt{m + 19.2, 34}, pt{m + 18, 40}, pt{m + 8, 48}, pt{m, 48})
	// barrel
	c.poly(hullLo, pt{m - 3, 44}, pt{m + 3, 44}, pt{m + 2.5, 57}, pt{m - 2.5, 57})
	// window
	c.capsule(danger, m-7, 22, m+7, 22, 4)
	c.save(dir, "gunship")
}

// The mothership: a wide saucer with a domed top, a rose core and a row of
// lights along the rim.
func mothership(dir string) {
	c := newCanvas(132, 102)
	m := 66.0
	// prongs
	c.symmetric(hullLo, m, pt{m + 30, 60}, pt{m + 40, 60}, pt{m + 30, 92}, pt{m + 26, 92})
	c.poly(hullLo, pt{m - 5, 64}, pt{m + 5, 64}, pt{m + 3, 98}, pt{m - 3, 98})
	// dome
	c.ellipse(hullHi, m, 44, 32, 30)
	// saucer
	c.ellipse(hull, m, 56, 64, 20)
	c.fill(hullLo, append(
		[]pt{{2, 56}, {130, 56}},
		ellipseHalf(m, 56, 64, 20)...,
	))
	// core
	c.circle(ink, m, 52, 13)
	c.ring(danger, m, 52, 10, 3)
	c.circle(danger, m, 52, 5)
	// rim lights
	for _, x := range []float64{-50, -36, -22, 22, 36, 50} {
		y := 56 + 20*math.Sqrt(1-(x/64)*(x/64)) - 7
		c.circle(fade(danger, 0.85), m+x, y, 2.2)
	}
	c.save(dir, "mothership")
}

// ellipseHalf is the lower half of an ellipse, from right to left.
func ellipseHalf(cx, cy, rx, ry float64) []pt {
	var p []pt
	for i := 0; i <= 40; i++ {
		a := math.Pi * float64(i) / 40
		p = append(p, pt{cx + math.Cos(a)*rx, cy + math.Sin(a)*ry})
	}
	return p
}

// Power-ups are small dark tokens with a coloured edge and a mark.
func powerups(dir string) {
	token := func(edge color.NRGBA) *canvas {
		c := newCanvas(39, 39)
		c.circle(surface, 19.5, 19.5, 16)
		c.ring(edge, 19.5, 19.5, 15.5, 2)
		return c
	}

	t := token(accent)
	for _, dx := range []float64{-6, 0, 6} {
		t.capsule(accent, 19.5+dx, 13, 19.5+dx, 26, 2.6)
	}
	t.save(dir, "power-triple")

	s := token(alt)
	s.poly(alt, pt{19.5, 11.5}, pt{26.5, 14.5}, pt{25.5, 22}, pt{19.5, 27.5}, pt{13.5, 22}, pt{12.5, 14.5})
	s.poly(surface, pt{19.5, 15}, pt{23.2, 16.6}, pt{22.6, 21}, pt{19.5, 24}, pt{19.5, 15})
	s.save(dir, "power-shield")

	l := token(danger)
	l.circle(danger, 16.2, 17, 3.7)
	l.circle(danger, 22.8, 17, 3.7)
	l.poly(danger, pt{12.7, 18.2}, pt{26.3, 18.2}, pt{19.5, 26.5})
	l.save(dir, "power-life")
}

func shield(dir string) {
	c := newCanvas(90, 90)
	c.circle(fade(alt, 0.08), 45, 45, 40)
	c.ring(alt, 45, 45, 40, 1.6)
	c.save(dir, "shield")
}

// The spark is white so the game can tint it.
func spark(dir string) {
	c := newCanvas(9, 9)
	c.circle(color.NRGBA{255, 255, 255, 255}, 4.5, 4.5, 2.2)
	c.save(dir, "spark")
}

// Explosions: a flash that opens into a ring and thins out, with a few
// fragments thrown clear.
func explosions(dir string) {
	rng := rand.New(rand.NewSource(7))
	type frag struct{ a, d, r float64 }
	frags := make([]frag, 7)
	for i := range frags {
		frags[i] = frag{float64(i)/7*2*math.Pi + rng.Float64()*0.6, 0.8 + rng.Float64()*0.35, 1.4 + rng.Float64()*1.2}
	}
	const m = 36.0
	type step struct{ disc, ring, width, alpha, spread float64 }
	steps := []step{
		{11, 0, 0, 1, 0},
		{15, 19, 2.5, 1, 0},
		{6, 23, 7, 1, 20},
		{0, 27, 4, 0.8, 26},
		{0, 31, 2, 0.5, 30},
		{0, 33.5, 1, 0.25, 33},
	}
	for i, s := range steps {
		c := newCanvas(72, 72)
		if s.disc > 0 {
			c.circle(fade(text, s.alpha), m, m, s.disc)
		}
		if s.ring > 0 {
			c.ring(fade(text, s.alpha), m, m, s.ring, s.width)
		}
		if s.spread > 0 {
			for _, f := range frags {
				d := math.Min(s.spread*f.d, 34-f.r)
				c.circle(fade(pale, s.alpha), m+math.Cos(f.a)*d, m+math.Sin(f.a)*d, f.r)
			}
		}
		c.save(dir, "explosion"+string(rune('1'+i)))
	}
}

// The sky is the flat ink of the menus, with two layers of stars that scroll
// at different speeds. Stars keep clear of the top and bottom edges so the
// layers tile without cutting one in half.
func sky(dir string) {
	bg := newCanvas(480, 720)
	draw.Draw(bg.img, bg.img.Bounds(), image.NewUniform(ink), image.Point{}, draw.Src)
	bg.save(dir, "background")

	rng := rand.New(rand.NewSource(1984))
	stars := func(name string, n int, r0, r1, a0, a1 float64, tints []color.NRGBA) {
		c := newCanvas(480, 720)
		for i := 0; i < n; i++ {
			r := r0 + rng.Float64()*(r1-r0)
			col := text
			if rng.Float64() < 0.18 {
				col = tints[rng.Intn(len(tints))]
			}
			c.circle(fade(col, a0+rng.Float64()*(a1-a0)), 3+rng.Float64()*474, 3+rng.Float64()*714, r)
		}
		c.save(dir, name)
	}
	stars("stars-far", 140, 0.55, 0.9, 0.18, 0.45, []color.NRGBA{alt, accent})
	stars("stars-near", 38, 0.9, 1.4, 0.5, 0.85, []color.NRGBA{pale, alt})
}
