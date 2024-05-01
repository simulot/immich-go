package asciimage

/*

Notes:
	the package nfnt is archived since long

Credit
	Andrew Albers https://github.com/Zebbeni
		- QuarterBlock rendering
		- AvgColor
*/

import (
	"image"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/nfnt/resize"
)

const (
	charRatio = 0.5
)

// Utf8Renderer
// Render the image i as an ascii box of height x width chars box
//

func Utf8Renderer(input image.Image, height, width int) (string, error) {
	imgW, imgH := float32(input.Bounds().Dx()), float32(input.Bounds().Dy())
	fitHeight := float32(width) * (imgH / imgW) * float32(charRatio)
	fitWidth := (float32(height) * (imgW / imgH)) / float32(charRatio)
	if fitHeight > float32(height) {
		width = int(fitWidth)
	} else {
		height = int(fitHeight)
	}

	sizedImage := resize.Resize(uint(width)*2, uint(height)*2, input, resize.NearestNeighbor)

	// sizedImage := image.NewRGBA(image.Rect(0, 0, width*2, height*2))
	// draw.NearestNeighbor.Scale(sizedImage, sizedImage.Rect, input, input.Bounds(), draw.Over, nil)

	rendered := strings.Builder{}
	for y := 0; y < height*2; y += 2 {
		for x := 0; x < width*2; x += 2 {
			// r1 r2
			// r3 r4
			r1, _ := colorful.MakeColor(sizedImage.At(x, y))
			r2, _ := colorful.MakeColor(sizedImage.At(x+1, y))
			r3, _ := colorful.MakeColor(sizedImage.At(x, y+1))
			r4, _ := colorful.MakeColor(sizedImage.At(x+1, y+1))

			// pick the block, fg and bg color with the lowest total difference
			// convert the colors to ansi, render the block and add it at row[x]
			r, fg, bg := getBlock(quarterBlockFunctions, r1, r2, r3, r4)

			pFg, _ := colorful.MakeColor(fg)
			pBg, _ := colorful.MakeColor(bg)

			style := lipgloss.NewStyle().Foreground(lipgloss.Color(pFg.Hex())).Background(lipgloss.Color(pBg.Hex()))
			rendered.WriteString(style.Render(string(r)))
		}
		rendered.WriteRune('\n')
	}
	return rendered.String(), nil
}

// find the best block character and foreground and background colors to match
// a set of 4 pixels. return
func getBlock(fns map[rune]blockFunctions, r1, r2, r3, r4 colorful.Color) (r rune, fg, bg colorful.Color) {
	minDist := 100.0
	for bRune, bFunc := range fns {
		f, b, dist := bFunc(r1, r2, r3, r4)
		if dist < minDist {
			minDist = dist
			r, fg, bg = bRune, f, b
		}
	}
	return
}

// Evaluate block foreground and background colors and return the error made
type blockFunctions func(r1, r2, r3, r4 colorful.Color) (colorful.Color, colorful.Color, float64)

var quarterBlockFunctions = map[rune]blockFunctions{
	'▀': calcTop,
	'▐': calcRight,
	'▞': calcDiagonal,
	'▖': calcBotLeft,
	'▘': calcTopLeft,
	'▝': calcTopRight,
	'▗': calcBotRight,
}

func calcTop(r1, r2, r3, r4 colorful.Color) (colorful.Color, colorful.Color, float64) {
	if r1.R == 0 && r1.G == 0 && r1.B == 0 && (r3.R != 0 || r3.G != 0 || r3.B != 0) {
		r1.R = r1.G
	}
	fg, fDist := avgCol(r1, r2)
	bg, bDist := avgCol(r3, r4)
	return fg, bg, fDist + bDist
}

func calcRight(r1, r2, r3, r4 colorful.Color) (colorful.Color, colorful.Color, float64) {
	fg, fDist := avgCol(r2, r4)
	bg, bDist := avgCol(r1, r3)
	return fg, bg, fDist + bDist
}

func calcDiagonal(r1, r2, r3, r4 colorful.Color) (colorful.Color, colorful.Color, float64) {
	fg, fDist := avgCol(r2, r3)
	bg, bDist := avgCol(r1, r4)
	return fg, bg, fDist + bDist
}

func calcBotLeft(r1, r2, r3, r4 colorful.Color) (colorful.Color, colorful.Color, float64) {
	fg, fDist := avgCol(r3)
	bg, bDist := avgCol(r1, r2, r4)
	return fg, bg, fDist + bDist
}

func calcTopLeft(r1, r2, r3, r4 colorful.Color) (colorful.Color, colorful.Color, float64) {
	fg, fDist := avgCol(r1)
	bg, bDist := avgCol(r2, r3, r4)
	return fg, bg, fDist + bDist
}

func calcTopRight(r1, r2, r3, r4 colorful.Color) (colorful.Color, colorful.Color, float64) {
	fg, fDist := avgCol(r2)
	bg, bDist := avgCol(r1, r3, r4)
	return fg, bg, fDist + bDist
}

func calcBotRight(r1, r2, r3, r4 colorful.Color) (colorful.Color, colorful.Color, float64) {
	fg, fDist := avgCol(r4)
	bg, bDist := avgCol(r1, r2, r3)
	return fg, bg, fDist + bDist
}

func avgCol(colors ...colorful.Color) (colorful.Color, float64) {
	rSum, gSum, bSum := 0.0, 0.0, 0.0
	for _, col := range colors {
		rSum += col.R
		gSum += col.G
		bSum += col.B
	}
	count := float64(len(colors))
	avg := colorful.Color{R: rSum / count, G: gSum / count, B: bSum / count}

	// compute sum of squares
	totalDist := 0.0
	for _, col := range colors {
		totalDist += math.Pow(col.DistanceCIEDE2000(avg), 2)
	}
	return avg, totalDist
}
