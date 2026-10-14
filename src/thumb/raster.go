/*
 * TgMusicBot - Telegram Music Bot
 *  Licensed under GNU GPL v3
 */

package thumb

// Tiny software rasteriser used by the card renderer. Everything is drawn at
// S x the design size from signed distance functions (SDF), so edges are
// anti-aliased, and the finished card is box-filtered down to the output size.

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

// sdf returns the signed distance (design px, negative = inside) from a point.
type sdf func(x, y float64) float64

type rect struct{ x0, y0, x1, y1 float64 }

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// ---------------------------------------------------------------- shapes
func sdBox(x0, y0, x1, y1, r float64) sdf {
	cx, cy := (x0+x1)/2, (y0+y1)/2
	hx, hy := (x1-x0)/2-r, (y1-y0)/2-r
	return func(x, y float64) float64 {
		qx := math.Abs(x-cx) - hx
		qy := math.Abs(y-cy) - hy
		return math.Hypot(math.Max(qx, 0), math.Max(qy, 0)) + math.Min(math.Max(qx, qy), 0) - r
	}
}

func sdCircle(cx, cy, r float64) sdf {
	return func(x, y float64) float64 { return math.Hypot(x-cx, y-cy) - r }
}

// inRing turns a filled shape into an inner outline of the given width.
func inRing(f sdf, width float64) sdf {
	return func(x, y float64) float64 {
		d := f(x, y)
		return math.Max(d, -d-width)
	}
}

func sdSegment(ax, ay, bx, by, halfW float64) sdf {
	return func(x, y float64) float64 {
		pax, pay := x-ax, y-ay
		bax, bay := bx-ax, by-ay
		h := clamp01((pax*bax + pay*bay) / (bax*bax + bay*bay))
		return math.Hypot(pax-bax*h, pay-bay*h) - halfW
	}
}

func sdPoly(pts [][2]float64) sdf {
	n := len(pts)
	return func(px, py float64) float64 {
		dx, dy := px-pts[0][0], py-pts[0][1]
		d := dx*dx + dy*dy
		s := 1.0
		for i, j := 0, n-1; i < n; j, i = i, i+1 {
			ex, ey := pts[j][0]-pts[i][0], pts[j][1]-pts[i][1]
			wx, wy := px-pts[i][0], py-pts[i][1]
			t := clamp01((wx*ex + wy*ey) / (ex*ex + ey*ey))
			bx, by := wx-ex*t, wy-ey*t
			d = math.Min(d, bx*bx+by*by)
			c1 := py >= pts[i][1]
			c2 := py < pts[j][1]
			c3 := ex*wy > ey*wx
			if (c1 && c2 && c3) || (!c1 && !c2 && !c3) {
				s = -s
			}
		}
		return s * math.Sqrt(d)
	}
}

// sdArc is a stroked circular arc. Angles are in degrees, measured clockwise
// from the +x axis (screen coordinates), sweeping from a0 up to a1.
func sdArc(cx, cy, R, halfW, a0, a1 float64) sdf {
	span := a1 - a0
	for span <= 0 {
		span += 360
	}
	r0, r1 := a0*math.Pi/180, a1*math.Pi/180
	e0x, e0y := cx+R*math.Cos(r0), cy+R*math.Sin(r0)
	e1x, e1y := cx+R*math.Cos(r1), cy+R*math.Sin(r1)
	return func(x, y float64) float64 {
		dx, dy := x-cx, y-cy
		ang := math.Atan2(dy, dx) * 180 / math.Pi
		rel := math.Mod(ang-a0+720, 360)
		if rel <= span {
			return math.Abs(math.Hypot(dx, dy)-R) - halfW
		}
		return math.Min(math.Hypot(x-e0x, y-e0y), math.Hypot(x-e1x, y-e1y)) - halfW
	}
}

func heartPoly(cx, cy, k float64) [][2]float64 {
	pts := make([][2]float64, 0, 60)
	for i := 0; i < 60; i++ {
		t := float64(i*6) * math.Pi / 180
		x := 16 * math.Pow(math.Sin(t), 3)
		y := -(13*math.Cos(t) - 5*math.Cos(2*t) - 2*math.Cos(3*t) - math.Cos(4*t))
		pts = append(pts, [2]float64{cx + x*k, cy + y*k})
	}
	return pts
}

// ---------------------------------------------------------------- colours
type colorAt func(x, y float64) color.NRGBA

func solid(c color.NRGBA) colorAt { return func(x, y float64) color.NRGBA { return c } }

type stop struct {
	pos float64
	c   [3]float64
}

// hGradient is a left-to-right gradient over [x0, x1].
func hGradient(x0, x1 float64, stops []stop) colorAt {
	return func(x, y float64) color.NRGBA {
		t := clamp01((x - x0) / (x1 - x0))
		for i := 0; i < len(stops)-1; i++ {
			a, b := stops[i], stops[i+1]
			if t >= a.pos && t <= b.pos {
				k := (t - a.pos) / math.Max(1e-9, b.pos-a.pos)
				return color.NRGBA{
					R: uint8(a.c[0] + (b.c[0]-a.c[0])*k),
					G: uint8(a.c[1] + (b.c[1]-a.c[1])*k),
					B: uint8(a.c[2] + (b.c[2]-a.c[2])*k),
					A: 255,
				}
			}
		}
		last := stops[len(stops)-1].c
		return color.NRGBA{R: uint8(last[0]), G: uint8(last[1]), B: uint8(last[2]), A: 255}
	}
}

// ---------------------------------------------------------------- drawing
func blend(pix []uint8, i int, c color.NRGBA, a float64) {
	// pix holds premultiplied RGBA, c is straight colour, a is 0..1 opacity.
	a *= float64(c.A) / 255
	if a <= 0 {
		return
	}
	inv := 1 - a
	pix[i] = uint8(float64(c.R)*a + float64(pix[i])*inv + 0.5)
	pix[i+1] = uint8(float64(c.G)*a + float64(pix[i+1])*inv + 0.5)
	pix[i+2] = uint8(float64(c.B)*a + float64(pix[i+2])*inv + 0.5)
	pix[i+3] = uint8(255*a + float64(pix[i+3])*inv + 0.5)
}

func pixBounds(b rect, bounds image.Rectangle) image.Rectangle {
	r := image.Rect(
		int(math.Floor(b.x0*S))-1, int(math.Floor(b.y0*S))-1,
		int(math.Ceil(b.x1*S))+1, int(math.Ceil(b.y1*S))+1,
	)
	return r.Intersect(bounds)
}

func coverage(d float64) float64 { return clamp01(0.5 - d*S) }

// fill paints the shape f (restricted to box) onto dst.
func fill(dst *image.RGBA, box rect, f sdf, col colorAt, opacity float64) {
	r := pixBounds(box, dst.Bounds())
	for py := r.Min.Y; py < r.Max.Y; py++ {
		y := (float64(py) + 0.5) / S
		row := py * dst.Stride
		for px := r.Min.X; px < r.Max.X; px++ {
			cov := coverage(f((float64(px)+0.5)/S, y))
			if cov <= 0 {
				continue
			}
			x := (float64(px) + 0.5) / S
			blend(dst.Pix, row+px*4, col(x, y), cov*opacity)
		}
	}
}

// paintImage copies src (positioned at pixel offset ox,oy) onto dst through the
// shape f. Only the pixels inside box (design px) are visited.
func paintImage(dst, src *image.RGBA, ox, oy int, box rect, f sdf) {
	area := pixBounds(box, dst.Bounds())
	sb := src.Bounds()
	for sy := sb.Min.Y; sy < sb.Max.Y; sy++ {
		py := oy + sy
		if py < area.Min.Y || py >= area.Max.Y {
			continue
		}
		y := (float64(py) + 0.5) / S
		for sx := sb.Min.X; sx < sb.Max.X; sx++ {
			px := ox + sx
			if px < area.Min.X || px >= area.Max.X {
				continue
			}
			cov := coverage(f((float64(px)+0.5)/S, y))
			if cov <= 0 {
				continue
			}
			si := sy*src.Stride + sx*4
			c := color.NRGBA{R: src.Pix[si], G: src.Pix[si+1], B: src.Pix[si+2], A: 255}
			blend(dst.Pix, py*dst.Stride+px*4, c, cov)
		}
	}
}

// ---------------------------------------------------------------- masks
// mask is an 8-bit coverage map over a sub-rectangle of the canvas (pixel coords).
type mask struct {
	r   image.Rectangle
	pix []uint8
}

func newMask(box rect, margin float64, canvas image.Rectangle) *mask {
	b := rect{box.x0 - margin, box.y0 - margin, box.x1 + margin, box.y1 + margin}
	r := pixBounds(b, canvas)
	return &mask{r: r, pix: make([]uint8, r.Dx()*r.Dy())}
}

// add unions the shape into the mask (a + b - ab).
func (m *mask) add(f sdf) {
	w := m.r.Dx()
	for py := m.r.Min.Y; py < m.r.Max.Y; py++ {
		y := (float64(py) + 0.5) / S
		for px := m.r.Min.X; px < m.r.Max.X; px++ {
			cov := coverage(f((float64(px)+0.5)/S, y))
			if cov <= 0 {
				continue
			}
			i := (py-m.r.Min.Y)*w + (px - m.r.Min.X)
			old := float64(m.pix[i]) / 255
			m.pix[i] = uint8((old + cov - old*cov) * 255)
		}
	}
}

func (m *mask) blur(sigma float64) {
	blurPlane(m.pix, m.r.Dx(), m.r.Dy(), 1, sigma*S)
}

// paint draws colour through the mask. clip (optional) multiplies by another shape.
func (m *mask) paint(dst *image.RGBA, col colorAt, scale float64, clip sdf) {
	w := m.r.Dx()
	for py := m.r.Min.Y; py < m.r.Max.Y; py++ {
		y := (float64(py) + 0.5) / S
		for px := m.r.Min.X; px < m.r.Max.X; px++ {
			v := m.pix[(py-m.r.Min.Y)*w+(px-m.r.Min.X)]
			if v == 0 {
				continue
			}
			a := float64(v) / 255 * scale
			x := (float64(px) + 0.5) / S
			if clip != nil {
				a *= coverage(clip(x, y))
			}
			if a <= 0 {
				continue
			}
			blend(dst.Pix, py*dst.Stride+px*4, col(x, y), clamp01(a))
		}
	}
}

// ---------------------------------------------------------------- blur
func boxSizes(sigma float64, n int) []int {
	wIdeal := math.Sqrt(12*sigma*sigma/float64(n) + 1)
	wl := int(math.Floor(wIdeal))
	if wl%2 == 0 {
		wl--
	}
	if wl < 1 {
		wl = 1
	}
	wu := wl + 2
	mIdeal := (12*sigma*sigma - float64(n*wl*wl) - 4*float64(n*wl) - 3*float64(n)) / (-4*float64(wl) - 4)
	m := int(math.Round(mIdeal))
	out := make([]int, n)
	for i := range out {
		if i < m {
			out[i] = wl
		} else {
			out[i] = wu
		}
	}
	return out
}

// blurLine box-blurs one line of n samples (edge pixels are repeated).
func blurLine(src, dst []uint8, start, step, n, r int) {
	if r < 1 || n < 2*r+2 {
		for i := 0; i < n; i++ {
			dst[start+i*step] = src[start+i*step]
		}
		return
	}
	iarr := 1 / float64(r+r+1)
	fv := int(src[start])
	lv := int(src[start+(n-1)*step])
	val := (r + 1) * fv
	for j := 0; j < r; j++ {
		val += int(src[start+j*step])
	}
	ti, li, ri := 0, 0, r
	for j := 0; j <= r; j++ {
		val += int(src[start+ri*step]) - fv
		dst[start+ti*step] = uint8(float64(val)*iarr + 0.5)
		ri++
		ti++
	}
	for j := r + 1; j < n-r; j++ {
		val += int(src[start+ri*step]) - int(src[start+li*step])
		dst[start+ti*step] = uint8(float64(val)*iarr + 0.5)
		ri++
		li++
		ti++
	}
	for j := n - r; j < n; j++ {
		val += lv - int(src[start+li*step])
		dst[start+ti*step] = uint8(float64(val)*iarr + 0.5)
		li++
		ti++
	}
}

// blurPlane applies an approximate Gaussian blur to interleaved 8-bit data.
func blurPlane(pix []uint8, w, h, ch int, sigma float64) {
	if sigma < 0.5 || w < 4 || h < 4 {
		return
	}
	tmp := make([]uint8, len(pix))
	for _, size := range boxSizes(sigma, 3) {
		r := (size - 1) / 2
		for y := 0; y < h; y++ {
			for c := 0; c < ch; c++ {
				blurLine(pix, tmp, y*w*ch+c, ch, w, r)
			}
		}
		for x := 0; x < w; x++ {
			for c := 0; c < ch; c++ {
				blurLine(tmp, pix, x*ch+c, w*ch, h, r)
			}
		}
	}
}

// ---------------------------------------------------------------- resize
func toRGBA(src image.Image) *image.RGBA {
	if r, ok := src.(*image.RGBA); ok && r.Rect.Min == (image.Point{}) {
		return r
	}
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return dst
}

// resize scales the region sr of src to dw x dh with bilinear sampling
// (2x2 taps per pixel when shrinking by more than 1.5x, to avoid aliasing).
func resize(srcImg image.Image, sr image.Rectangle, dw, dh int) *image.RGBA {
	src := toRGBA(srcImg)
	sr = sr.Sub(srcImg.Bounds().Min).Intersect(src.Bounds())
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	sw, sh := float64(sr.Dx()), float64(sr.Dy())

	taps := 1
	if sw/float64(dw) > 1.5 || sh/float64(dh) > 1.5 {
		taps = 2
	}

	type axis struct {
		i0, i1 int
		w      int // weight of i1, 0..256
	}
	build := func(n, dn, off, sn int, ssize float64) [][]axis {
		out := make([][]axis, dn)
		for d := 0; d < dn; d++ {
			out[d] = make([]axis, taps)
			for t := 0; t < taps; t++ {
				f := float64(off) + (float64(d)+(float64(t)+0.5)/float64(taps))*ssize/float64(dn) - 0.5
				i0 := int(math.Floor(f))
				w := int((f - float64(i0)) * 256)
				i1 := i0 + 1
				i0 = min(max(i0, off), off+sn-1)
				i1 = min(max(i1, off), off+sn-1)
				out[d][t] = axis{i0, i1, w}
			}
		}
		return out
	}
	xs := build(sr.Dx(), dw, sr.Min.X, sr.Dx(), sw)
	ys := build(sr.Dy(), dh, sr.Min.Y, sr.Dy(), sh)

	for y := 0; y < dh; y++ {
		for x := 0; x < dw; x++ {
			var r, g, b int
			for _, ay := range ys[y] {
				row0 := ay.i0 * src.Stride
				row1 := ay.i1 * src.Stride
				for _, ax := range xs[x] {
					p00, p10 := row0+ax.i0*4, row0+ax.i1*4
					p01, p11 := row1+ax.i0*4, row1+ax.i1*4
					for c := 0; c < 3; c++ {
						top := int(src.Pix[p00+c])*(256-ax.w) + int(src.Pix[p10+c])*ax.w
						bot := int(src.Pix[p01+c])*(256-ax.w) + int(src.Pix[p11+c])*ax.w
						v := (top*(256-ay.w) + bot*ay.w) >> 16
						switch c {
						case 0:
							r += v
						case 1:
							g += v
						default:
							b += v
						}
					}
				}
			}
			n := taps * taps
			o := y*dst.Stride + x*4
			dst.Pix[o] = uint8(r / n)
			dst.Pix[o+1] = uint8(g / n)
			dst.Pix[o+2] = uint8(b / n)
			dst.Pix[o+3] = 255
		}
	}
	return dst
}

// fitCover scales and crops src so it fully covers dw x dh (like ImageOps.fit).
// (cx, cy) is the crop centre in 0..1.
func fitCover(src image.Image, dw, dh int, cx, cy float64) *image.RGBA {
	b := src.Bounds()
	sw, sh := float64(b.Dx()), float64(b.Dy())
	scale := math.Max(float64(dw)/sw, float64(dh)/sh)
	cw, ch := float64(dw)/scale, float64(dh)/scale
	x0 := (sw - cw) * cx
	y0 := (sh - ch) * cy
	sr := image.Rect(
		b.Min.X+int(math.Round(x0)), b.Min.Y+int(math.Round(y0)),
		b.Min.X+int(math.Round(x0+cw)), b.Min.Y+int(math.Round(y0+ch)),
	).Intersect(b)
	if sr.Dx() < 1 || sr.Dy() < 1 {
		sr = b
	}
	return resize(src, sr, dw, dh)
}
