package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/nfnt/resize"
)

var asciiChars = "@%#*+=-:. "

func pixelToASCII(val uint32) string {
	length := len(asciiChars)
	index := int((val * uint32(length-1)) / 65535)
	return string(asciiChars[index])
}

func drawImage(imgPath string, width, height int) (string, error) {
	imgFile, err := os.Open(imgPath)
	if err != nil {
		return "", err
	}
	defer imgFile.Close()
	img, _, err := image.Decode(imgFile)
	if err != nil {
		return "", err
	}
	img = resize.Resize(uint(width), uint(height), img, resize.Lanczos3)
	bounds := img.Bounds()
	var buffer bytes.Buffer
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if a < 6553 {
				buffer.WriteString(" ")
				continue
			}
			colorStr := fmt.Sprintf("#%02X%02X%02X", r>>8, g>>8, b>>8)
			asciiChar := pixelToASCII((r + g + b) / 3)
			coloredAscii := lipgloss.NewStyle().Background(lipgloss.Color(colorStr)).Render(asciiChar)
			buffer.WriteString(coloredAscii)
		}
		buffer.WriteString("\n")
	}
	return buffer.String(), nil
}
