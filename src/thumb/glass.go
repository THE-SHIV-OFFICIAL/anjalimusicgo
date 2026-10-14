/*
 * TgMusicBot - Telegram Music Bot
 *  Licensed under GNU GPL v3
 */

package thumb

import (
	_ "embed"
	"image"
	"image/color"
	"math"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// Output size and supersampling factor. Shapes are anti-aliased analytically
// (signed distance functions), so 1x is enough; raise S to 2 for extra smoothness.
const (
	W = 1280
	H = 720
	S = 1
)

//go:embed fonts/cc_regular.ttf
var fontRegularData []byte

//go:embed fonts/cc_bold.ttf
var fontBoldData []byte

var (
	fontOnce    sync.Once
	fontRegular *opentype.Font
	fontBold    *opentype.Font
	fontErr     error

	faceMu    sync.Mutex
	faceCache = map[string]font.Face{}
)

func loadFonts() error {
	fontOnce.Do(func() {
		fontRegular, fontErr = opentype.Parse(fontRegularData)
		if fontErr != nil {
			return
		}
		fontBold, fontErr = opentype.Parse(fontBoldData)
	})
	return fontErr
}

func face(bold bool, size float64) (font.Face, error) {
	if err := loadFonts(); err != nil {
		return nil, err
	}
	key := "r"
	f := fontRegular
	if bold {
		key, f = "b", fontBold
	}
	key += string(rune(int(size * 10)))

	faceMu.Lock()
	defer faceMu.Unlock()
	if fc, ok := faceCache[key]; ok {
		return fc, nil
	}
	fc, err := opentype.NewFace(f, &opentype.FaceOptions{Size: size * S, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		return nil, err
	}
	faceCache[key] = fc
	return fc, nil
}

func textWidth(fc font.Face, s string) float64 {
	return float64(font.MeasureString(fc, s)) / 64 / S
}

// fitText shortens s with "..." until it fits maxW design px.
func fitText(fc font.Face, s string, maxW float64) string {
	s = strings.TrimSpace(s)
	if s == "" {
		s = "Unknown"
	}
	if textWidth(fc, s) <= maxW {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && textWidth(fc, string(r)+"...") > maxW {
		r = r[:len(r)-1]
	}
	return strings.TrimRight(string(r), " ") + "..."
}

// drawText draws s with its top-left at (x, y) in design px, like PIL's default anchor.
func drawText(dst *image.RGBA, fc font.Face, s string, x, y float64, col color.NRGBA) {
	d := &font.Drawer{Dst: dst, Src: image.NewUniform(col), Face: fc}
	asc := fc.Metrics().Ascent
	d.Dot = fixed.Point26_6{
		X: fixed.Int26_6(x * S * 64),
		Y: fixed.Int26_6(y*S*64) + asc,
	}
	d.DrawString(s)
}

// Layout (design px, 1280x720).
var (
	cardBox = rect{220, 140, 1060, 580}
	artBox  = rect{260, 190, 600, 530}
)

var (
	white = color.NRGBA{255, 255, 255, 255}

	cardStops = []stop{
		{0.0, [3]float64{88, 160, 214}},
		{0.30, [3]float64{186, 128, 165}},
		{0.50, [3]float64{140, 214, 20}},
		{0.78, [3]float64{150, 205, 35}},
		{1.0, [3]float64{208, 122, 132}},
	}
	artStops = []stop{
		{0.0, [3]float64{100, 160, 215}},
		{0.5, [3]float64{130, 190, 120}},
		{1.0, [3]float64{150, 215, 30}},
	}
)

// item is one icon part: a shape and its colour.
type item struct {
	f   sdf
	col color.NRGBA
}

func tri(a, b, c [2]float64) sdf { return sdPoly([][2]float64{a, b, c}) }

// controlIcons returns every control-row icon (shuffle, repeat, prev, pause,
// next, heart, headphones).
func controlIcons() []item {
	green := color.NRGBA{60, 255, 160, 255}
	amber := color.NRGBA{255, 190, 40, 255}
	red := color.NRGBA{255, 60, 60, 255}
	var its []item
	add := func(f sdf, c color.NRGBA) { its = append(its, item{f, c}) }

	// shuffle
	for _, l := range [][4]float64{
		{629, 493, 639, 493}, {639, 493, 651, 507}, {651, 507, 655, 507},
		{629, 507, 639, 507}, {639, 507, 651, 493}, {651, 493, 655, 493},
	} {
		add(sdSegment(l[0], l[1], l[2], l[3], 1.5), green)
	}
	add(tri([2]float64{654, 488}, [2]float64{654, 498}, [2]float64{661, 493}), green)
	add(tri([2]float64{654, 502}, [2]float64{654, 512}, [2]float64{661, 507}), green)

	// repeat
	add(sdArc(698, 500, 9.5, 1.5, 40, 340), amber)
	add(tri([2]float64{701, 484}, [2]float64{701, 496}, [2]float64{711, 490}), amber)

	// previous / pause / next
	add(sdBox(737, 489, 741, 511, 0), white)
	add(tri([2]float64{755, 489}, [2]float64{755, 511}, [2]float64{742, 500}), white)
	add(tri([2]float64{769, 489}, [2]float64{769, 511}, [2]float64{756, 500}), white)
	add(sdBox(800, 489, 806, 511, 2), white)
	add(sdBox(810, 489, 816, 511, 2), white)
	add(sdBox(875, 489, 879, 511, 0), white)
	add(tri([2]float64{861, 489}, [2]float64{861, 511}, [2]float64{874, 500}), white)
	add(tri([2]float64{847, 489}, [2]float64{847, 511}, [2]float64{860, 500}), white)

	// heart
	add(sdPoly(heartPoly(920, 499, 0.95)), red)

	// headphones
	add(sdArc(976, 502, 12.4, 1.6, 180, 360), white)
	add(sdBox(960, 500, 968, 513, 3), white)
	add(sdBox(984, 500, 992, 513, 3), white)
	return its
}

var iconBounds = rect{620, 480, 1000, 520}

// adjustColor changes saturation and brightness of an opaque image in place.
func adjustColor(img *image.RGBA, sat, bright float64) {
	for i := 0; i+3 < len(img.Pix); i += 4 {
		r, g, b := float64(img.Pix[i]), float64(img.Pix[i+1]), float64(img.Pix[i+2])
		gray := 0.299*r + 0.587*g + 0.114*b
		r = (gray + (r-gray)*sat) * bright
		g = (gray + (g-gray)*sat) * bright
		b = (gray + (b-gray)*sat) * bright
		img.Pix[i] = uint8(math.Min(255, math.Max(0, r)))
		img.Pix[i+1] = uint8(math.Min(255, math.Max(0, g)))
		img.Pix[i+2] = uint8(math.Min(255, math.Max(0, b)))
	}
}

// upscale enlarges an opaque image to w x h (bilinear).
func upscale(src *image.RGBA, w, h int) *image.RGBA {
	return resize(src, src.Bounds(), w, h)
}

// Card is the data shown on the glass card.
type Card struct {
	Title    string
	Artist   string
	Duration string
}

// Render draws the glass player card (1280x720) for the given cover art.
func Render(art image.Image, c Card) (*image.RGBA, error) {
	fTitle, err := face(false, 33)
	if err != nil {
		return nil, err
	}
	fArtist, err := face(true, 23)
	if err != nil {
		return nil, err
	}
	fTime, err := face(true, 18)
	if err != nil {
		return nil, err
	}

	// background: blurred cover, boosted colour
	small := fitCover(art, W/2, H/2, 0.5, 0.5)
	frostSmall := image.NewRGBA(small.Bounds())
	copy(frostSmall.Pix, small.Pix)
	blurPlane(small.Pix, small.Bounds().Dx(), small.Bounds().Dy(), 4, 7)
	copy(frostSmall.Pix, small.Pix)
	blurPlane(frostSmall.Pix, frostSmall.Bounds().Dx(), frostSmall.Bounds().Dy(), 4, 7)

	base := upscale(small, W*S, H*S)
	adjustColor(base, 1.15, 0.92)
	frost := upscale(frostSmall, W*S, H*S)
	adjustColor(frost, 1.15, 0.92)
	canvas := base.Bounds()

	// frosted glass card
	cardShape := sdBox(cardBox.x0, cardBox.y0, cardBox.x1, cardBox.y1, 46)
	// darken the frost a little (black overlay alpha 95/255)
	for i := 0; i+3 < len(frost.Pix); i += 4 {
		k := 1 - 95.0/255
		frost.Pix[i] = uint8(float64(frost.Pix[i]) * k)
		frost.Pix[i+1] = uint8(float64(frost.Pix[i+1]) * k)
		frost.Pix[i+2] = uint8(float64(frost.Pix[i+2]) * k)
	}
	paintImage(base, frost, 0, 0, cardBox, cardShape)

	// glowing gradient border
	cardGrad := hGradient(cardBox.x0, cardBox.x1, cardStops)
	cardRing := inRing(cardShape, 5)
	gm := newMask(cardBox, 30, canvas)
	gm.add(cardRing)
	gm.blur(9)
	gm.paint(base, cardGrad, 0.9, nil)
	fill(base, rect{cardBox.x0 - 2, cardBox.y0 - 2, cardBox.x1 + 2, cardBox.y1 + 2}, cardRing, cardGrad, 1)

	// album art
	artShape := sdBox(artBox.x0, artBox.y0, artBox.x1, artBox.y1, 30)
	aw, ah := int(artBox.x1-artBox.x0)*S, int(artBox.y1-artBox.y0)*S
	pic := fitCover(art, aw, ah, 0.4, 0.5)
	paintImage(base, pic, int(artBox.x0)*S, int(artBox.y0)*S, artBox, artShape)

	shade := newMask(artBox, 20, canvas)
	shade.add(inRing(artShape, 14))
	shade.blur(7)
	shade.paint(base, solid(color.NRGBA{0, 0, 0, 255}), 0.55, artShape)

	artGrad := hGradient(artBox.x0, artBox.x1, artStops)
	artRing := inRing(artShape, 4)
	am := newMask(artBox, 20, canvas)
	am.add(artRing)
	am.blur(6)
	am.paint(base, artGrad, 0.8, nil)
	fill(base, rect{artBox.x0 - 2, artBox.y0 - 2, artBox.x1 + 2, artBox.y1 + 2}, artRing, artGrad, 1)

	// text
	drawText(base, fTitle, fitText(fTitle, strings.ToUpper(c.Title), 400), 640, 236, white)
	drawText(base, fArtist, fitText(fArtist, c.Artist, 400), 640, 299, color.NRGBA{222, 222, 222, 255})

	// progress bar
	fill(base, rect{640, 397, 1020, 403}, sdBox(640, 397, 1020, 403, 3), solid(color.NRGBA{205, 205, 205, 255}), 1)
	fill(base, rect{640, 397, 727, 403}, sdBox(640, 397, 727, 403, 3), solid(color.NRGBA{124, 210, 28, 255}), 1)
	knobShadow := newMask(rect{715, 391, 739, 415}, 10, canvas)
	knobShadow.add(sdCircle(727, 402, 9))
	knobShadow.blur(3)
	knobShadow.paint(base, solid(color.NRGBA{0, 0, 0, 255}), 90.0/255, nil)
	fill(base, rect{717, 390, 737, 410}, sdCircle(727, 400, 8), solid(white), 1)

	drawText(base, fTime, "00:00", 640, 425, white)
	dur := c.Duration
	if dur == "" {
		dur = "0:00"
	}
	drawText(base, fTime, dur, 1020-textWidth(fTime, dur), 425, white)

	// controls: soft white glow first, crisp icons on top
	icons := controlIcons()
	glow := newMask(iconBounds, 16, canvas)
	for _, it := range icons {
		glow.add(it.f)
	}
	glow.blur(5)
	glow.paint(base, solid(white), 0.55, nil)
	for _, it := range icons {
		fill(base, iconBounds, it.f, solid(it.col), 1)
	}

	// small badge, top right
	fill(base, rect{1178, 24, 1248, 94}, sdBox(1178, 24, 1248, 94, 16), solid(color.NRGBA{255, 255, 255, 215}), 1)
	fill(base, rect{1200, 46, 1226, 72}, inRing(sdCircle(1213, 59, 10), 3), solid(color.NRGBA{236, 140, 160, 255}), 1)

	return downscale(base), nil
}

// downscale halves the canvas with a 2x2 box filter.
func downscale(src *image.RGBA) *image.RGBA {
	if S == 1 {
		return src
	}
	dst := image.NewRGBA(image.Rect(0, 0, W, H))
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			var sum [3]int
			for j := 0; j < S; j++ {
				for i := 0; i < S; i++ {
					o := (y*S+j)*src.Stride + (x*S+i)*4
					sum[0] += int(src.Pix[o])
					sum[1] += int(src.Pix[o+1])
					sum[2] += int(src.Pix[o+2])
				}
			}
			o := y*dst.Stride + x*4
			dst.Pix[o] = uint8(sum[0] / (S * S))
			dst.Pix[o+1] = uint8(sum[1] / (S * S))
			dst.Pix[o+2] = uint8(sum[2] / (S * S))
			dst.Pix[o+3] = 255
		}
	}
	return dst
}
