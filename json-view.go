package main

import (
	"encoding/json"
	"strings"

	"github.com/gdamore/tcell"
)

type jsonView struct {
	// The location that the log will be displayed in (inclusive of endpoints)
	frame rect

	entry logEntry
}

func (jv *jsonView) setEntry(screen tcell.Screen, entry logEntry) {
	jv.entry = entry
	jv.draw(screen)
}

func (jv *jsonView) resize(screen tcell.Screen, newFrame rect) {
	jv.frame = newFrame
	jv.draw(screen)
}

func (jv *jsonView) draw(screen tcell.Screen) {
	drawFrame(screen, jv.frame, "json")

	contentFrame := jv.frame.inset()
	for x := contentFrame.x0; x <= contentFrame.x1; x++ {
		for y := contentFrame.y0; y <= contentFrame.y1; y++ {
			screen.SetContent(x, y, ' ', nil, tcell.StyleDefault)
		}
	}

	b, err := json.MarshalIndent(jv.entry.rest, "", "  ")
	if err != nil {
		b = []byte("[error] Failed to marshal json entry!")
	}

	lines := strings.Split(string(b), "\n")

	for i, line := range lines {
		if contentFrame.y0+i > contentFrame.y1 {
			break
		}

		for j, char := range line {
			if (contentFrame.x0 + j) > contentFrame.x1 {
				break
			}
			screen.SetContent(contentFrame.x0+j, contentFrame.y0+i, char, nil, tcell.StyleDefault)
		}
	}
}
