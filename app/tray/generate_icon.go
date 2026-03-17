// +build ignore

package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

func main() {
	// Generate active icon (colored)
	generateActiveIcon()
	
	// Generate inactive icon (gray)
	generateInactiveIcon()
}

func generateActiveIcon() {
	// Create a 64x64 icon
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))

	// Fill with a gradient blue color (representing Nonelane)
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			// Create a circular icon
			cx, cy := x-32, y-32
			if cx*cx+cy*cy < 30*30 {
				// Inside circle - blue gradient
				img.Set(x, y, color.RGBA{
					R: uint8(66 + x/2),
					G: uint8(133 + y/4),
					B: 245,
					A: 255,
				})
			} else {
				// Outside circle - transparent
				img.Set(x, y, color.RGBA{A: 0})
			}
		}
	}

	// Save to file
	file, err := os.Create("icon-active.png")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		panic(err)
	}

	println("✓ Active icon generated: icon-active.png")
}

func generateInactiveIcon() {
	// Create a 64x64 icon
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))

	// Fill with gray color (inactive state)
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			// Create a circular icon
			cx, cy := x-32, y-32
			if cx*cx+cy*cy < 30*30 {
				// Inside circle - gray gradient
				gray := uint8(128 + x/4)
				img.Set(x, y, color.RGBA{
					R: gray,
					G: gray,
					B: gray,
					A: 255,
				})
			} else {
				// Outside circle - transparent
				img.Set(x, y, color.RGBA{A: 0})
			}
		}
	}

	// Save to file
	file, err := os.Create("icon-inactive.png")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		panic(err)
	}

	println("✓ Inactive icon generated: icon-inactive.png")
}