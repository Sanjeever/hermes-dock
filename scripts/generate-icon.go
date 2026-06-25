package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

const (
	iconSize = 1024
	scale    = 3
)

type point struct {
	x int
	y int
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	img := renderIcon(iconSize)
	appIconPath := filepath.Join(root, "build", "appicon.png")
	if err := writePNG(appIconPath, img); err != nil {
		panic(err)
	}
	icoImage := renderIcon(256)
	icoPath := filepath.Join(root, "build", "windows", "icon.ico")
	if err := writeICO(icoPath, icoImage); err != nil {
		panic(err)
	}
}

func renderIcon(size int) *image.RGBA {
	canvas := image.NewRGBA(image.Rect(0, 0, size*scale, size*scale))
	s := func(v int) int { return v * size * scale / iconSize }

	fillRoundedRect(canvas, s(40), s(40), s(984), s(984), s(172), rgba(18, 28, 24, 255))
	fillRoundedRect(canvas, s(76), s(76), s(948), s(948), s(142), rgba(42, 71, 59, 255))
	fillRoundedRect(canvas, s(98), s(98), s(926), s(926), s(124), rgba(13, 19, 17, 255))

	fillPolygon(canvas, []point{{s(512), s(206)}, {s(232), s(350)}, {s(288), s(418)}, {s(512), s(322)}}, rgba(61, 214, 198, 255))
	fillPolygon(canvas, []point{{s(512), s(206)}, {s(792), s(350)}, {s(736), s(418)}, {s(512), s(322)}}, rgba(255, 209, 90, 255))
	fillPolygon(canvas, []point{{s(512), s(282)}, {s(312), s(466)}, {s(712), s(466)}}, rgba(15, 37, 31, 255))

	fillRoundedRect(canvas, s(278), s(380), s(350), s(652), s(24), rgba(71, 215, 199, 255))
	fillRoundedRect(canvas, s(476), s(322), s(548), s(652), s(24), rgba(142, 230, 107, 255))
	fillRoundedRect(canvas, s(674), s(380), s(746), s(652), s(24), rgba(255, 209, 90, 255))

	fillRoundedRect(canvas, s(224), s(524), s(388), s(640), s(18), rgba(47, 185, 167, 255))
	fillRoundedRect(canvas, s(430), s(524), s(594), s(640), s(18), rgba(123, 216, 94, 255))
	fillRoundedRect(canvas, s(636), s(524), s(800), s(640), s(18), rgba(232, 185, 77, 255))
	fillRoundedRect(canvas, s(250), s(558), s(362), s(578), s(10), rgba(16, 37, 31, 190))
	fillRoundedRect(canvas, s(456), s(558), s(568), s(578), s(10), rgba(16, 37, 31, 190))
	fillRoundedRect(canvas, s(662), s(558), s(774), s(578), s(10), rgba(16, 37, 31, 190))

	fillRoundedRect(canvas, s(202), s(654), s(822), s(732), s(32), rgba(154, 240, 111, 255))
	fillRoundedRect(canvas, s(270), s(744), s(754), s(794), s(25), rgba(38, 73, 61, 255))
	fillCircle(canvas, s(512), s(769), s(16), rgba(154, 240, 111, 255))

	return downsample(canvas, size, size)
}

func writePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}

func writeICO(path string, img image.Image) error {
	var data bytes.Buffer
	if err := png.Encode(&data, img); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := binary.Write(file, binary.LittleEndian, uint16(0)); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, uint16(1)); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, uint16(1)); err != nil {
		return err
	}
	file.Write([]byte{0, 0, 0, 0})
	if err := binary.Write(file, binary.LittleEndian, uint16(1)); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, uint16(32)); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, uint32(data.Len())); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, uint32(22)); err != nil {
		return err
	}
	_, err = file.Write(data.Bytes())
	return err
}

func fillRoundedRect(img *image.RGBA, x0, y0, x1, y1, r int, c color.RGBA) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if insideRoundedRect(x, y, x0, y0, x1, y1, r) {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func insideRoundedRect(x, y, x0, y0, x1, y1, r int) bool {
	cx := clamp(x, x0+r, x1-r-1)
	cy := clamp(y, y0+r, y1-r-1)
	dx := x - cx
	dy := y - cy
	return dx*dx+dy*dy <= r*r
}

func fillCircle(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	r2 := r * r
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx := x - cx
			dy := y - cy
			if dx*dx+dy*dy <= r2 {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func fillPolygon(img *image.RGBA, points []point, c color.RGBA) {
	if len(points) < 3 {
		return
	}
	minY, maxY := points[0].y, points[0].y
	for _, p := range points[1:] {
		if p.y < minY {
			minY = p.y
		}
		if p.y > maxY {
			maxY = p.y
		}
	}
	for y := minY; y <= maxY; y++ {
		var nodes []int
		j := len(points) - 1
		for i := range points {
			pi := points[i]
			pj := points[j]
			if (pi.y < y && pj.y >= y) || (pj.y < y && pi.y >= y) {
				x := pi.x + (y-pi.y)*(pj.x-pi.x)/(pj.y-pi.y)
				nodes = append(nodes, x)
			}
			j = i
		}
		for i := 1; i < len(nodes); i++ {
			for j := i; j > 0 && nodes[j-1] > nodes[j]; j-- {
				nodes[j-1], nodes[j] = nodes[j], nodes[j-1]
			}
		}
		for i := 0; i+1 < len(nodes); i += 2 {
			for x := nodes[i]; x < nodes[i+1]; x++ {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func downsample(src *image.RGBA, width, height int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			var r, g, b, a uint32
			for sy := 0; sy < scale; sy++ {
				for sx := 0; sx < scale; sx++ {
					cr, cg, cb, ca := src.At(x*scale+sx, y*scale+sy).RGBA()
					r += cr
					g += cg
					b += cb
					a += ca
				}
			}
			div := uint32(scale * scale)
			dst.SetRGBA(x, y, color.RGBA{
				R: uint8(math.Round(float64(r/div) / 257)),
				G: uint8(math.Round(float64(g/div) / 257)),
				B: uint8(math.Round(float64(b/div) / 257)),
				A: uint8(math.Round(float64(a/div) / 257)),
			})
		}
	}
	return dst
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func rgba(r, g, b, a uint8) color.RGBA {
	return color.RGBA{R: r, G: g, B: b, A: a}
}
