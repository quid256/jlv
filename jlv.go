package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/gdamore/tcell"
)

var logRatio = 0.5

const (
	runeTL = '╭'
	runeTR = '╮'
	runeBL = '╰'
	runeBR = '╯'
	runeH  = '─'
	runeV  = '│'
)

type logViewer struct {
	screen tcell.Screen

	logPane  rect
	jsonPane rect

	offset int
	cursor int

	file jsonLogFile

	displayFiltered bool

	isEditMode        bool
	editCursor        int
	query             string
	invalidQuery      bool
	validQuery        bool
	program           *vm.Program
	matchingLines     map[int]struct{}
	matchingLinesList []int
}

func (l *logViewer) init(filename string) {
	l.file = parseJSONLogFile(filename)
	l.matchingLines = make(map[int]struct{})
	l.matchingLinesList = nil
}

func (l *logViewer) handleKey(key *tcell.EventKey) (shouldQuit bool) {
	if l.isEditMode {
		switch key.Key() {
		case tcell.KeyRune:
			l.validQuery = false
			l.invalidQuery = false
			r := key.Rune()
			l.query = l.query[:l.editCursor] + string(r) + l.query[l.editCursor:]

			_, height := l.screen.Size()
			l.editCursor++
			l.screen.ShowCursor(l.editCursor+2, height-1)

			l.redrawQuery()
			l.screen.Show()
		case tcell.KeyLeft:
			_, height := l.screen.Size()
			l.editCursor--
			if l.editCursor < 0 {
				l.editCursor = 0
			}
			l.screen.ShowCursor(l.editCursor+2, height-1)
			l.screen.Show()
		case tcell.KeyRight:
			_, height := l.screen.Size()
			l.editCursor++
			if l.editCursor > len(l.query) {
				l.editCursor = len(l.query)
			}
			l.screen.ShowCursor(l.editCursor+2, height-1)
			l.screen.Show()
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			l.validQuery = false
			l.invalidQuery = false
			if l.editCursor > 0 {
				l.query = l.query[:l.editCursor-1] + l.query[l.editCursor:]

				_, height := l.screen.Size()
				l.editCursor--
				l.screen.ShowCursor(l.editCursor+2, height-1)

				l.redrawQuery()
				l.screen.Show()
			}
		case tcell.KeyCtrlU:
			l.validQuery = false
			l.invalidQuery = false
			l.query = ""

			_, height := l.screen.Size()
			l.editCursor = 0
			l.screen.ShowCursor(2, height-1)

			l.redrawQuery()
			l.screen.Show()

		case tcell.KeyEnter:
			l.editMode(false)
			prog, err := expr.Compile(l.query, expr.AsBool())
			if l.query == "" {
				l.invalidQuery = false
				l.validQuery = false
				l.program = nil
			} else if err == nil {
				l.invalidQuery = false
				l.validQuery = true
				l.program = prog
			} else {
				l.invalidQuery = true
				l.validQuery = false
				l.program = nil
			}
			l.executeProgram()
			l.redrawQuery()
			l.screen.Show()

		case tcell.KeyEsc:
			l.editMode(false)
			l.screen.Show()
		}
	} else {
		switch key.Key() {
		case tcell.KeyRune:
			r := key.Rune()
			switch r {
			case 'j':
				l.scroll(+1)
				l.redrawJSONPane()
			case 'k':
				l.scroll(-1)
				l.redrawJSONPane()
			case 'J':
				l.scroll(l.logPane.y1 - l.logPane.y0 + 1)
				l.redrawJSONPane()
			case 'K':
				l.scroll(-(l.logPane.y1 - l.logPane.y0 + 1))
				l.redrawJSONPane()
			case 'h':
				logRatio -= 0.05
				if logRatio < 0.2 {
					logRatio = 0.2
				}
				width, height := l.screen.Size()
				l.resize(width, height)
			case 'l':
				logRatio += 0.05
				if logRatio > 0.8 {
					logRatio = 0.8
				}
				width, height := l.screen.Size()
				l.resize(width, height)
			case 'g':
				l.scroll(-len(l.file))
				l.redrawJSONPane()
			case 'G':
				l.scroll(len(l.file))
				l.redrawJSONPane()
			case '/', ':':
				l.editMode(true)
			case 'n':
				l.nextMatch()
				l.redrawJSONPane()
			case 'N':
				l.prevMatch()
				l.redrawJSONPane()
			case 'q':
				return true
			}
			l.screen.Show()
		case tcell.KeyDown:
			l.scroll(+1)
			l.redrawJSONPane()
			l.screen.Show()
		case tcell.KeyUp:
			l.scroll(-1)
			l.redrawJSONPane()
			l.screen.Show()
		case tcell.KeyCtrlC:
			return true
		}
	}
	return false
}

func (l *logViewer) nextMatch() {
	pos := l.cursor + l.offset
	for j := pos + 1; j < len(l.file); j++ {
		if _, ok := l.matchingLines[j]; ok {
			l.scroll(j - pos)
			return
		}
	}
}

func (l *logViewer) prevMatch() {
	pos := l.cursor + l.offset
	for j := pos - 1; j >= 0; j-- {
		if _, ok := l.matchingLines[j]; ok {
			l.scroll(j - pos)
			return
		}
	}
}

func (l *logViewer) executeProgram() {

	defer l.redrawLogPane()

	l.matchingLines = make(map[int]struct{})
	l.matchingLinesList = nil

	if l.program == nil {
		return
	}

	for i, line := range l.file {
		result, err := expr.Run(l.program, line.rest)
		if err == nil && result.(bool) {
			l.matchingLines[i] = struct{}{}
			l.matchingLinesList = append(l.matchingLinesList, i)
		}
	}
}

func (l *logViewer) editMode(is bool) {
	if is {
		l.isEditMode = true
		_, height := l.screen.Size()
		l.screen.ShowCursor(2+len(l.query), height-1)
		l.editCursor = len(l.query)
	} else {
		l.isEditMode = false
		l.editCursor = 0
		l.screen.HideCursor()
	}
}

func (l *logViewer) resize(width, height int) {
	l.screen.Clear()

	dividerX := int(math.Ceil(logRatio*float64(width))) - 1

	logPane := rect{0, 0, dividerX, height - 2}
	jsonPane := rect{dividerX + 1, 0, width - 1, height - 2}

	drawFrame(l.screen, logPane, "log")
	drawFrame(l.screen, jsonPane, "json")

	l.logPane = logPane.inset()
	l.redrawLogPane()

	l.jsonPane = jsonPane.inset()
	l.redrawJSONPane()

	l.redrawQuery()

	// TODO would Show work here? idk
	l.screen.Sync()
}

func (l *logViewer) redrawQuery() {
	width, height := l.screen.Size()

	fullQueryString := []rune("? " + l.query)

	var j int

	for j = 0; j < len(fullQueryString) && j < width; j++ {
		style := tcell.StyleDefault
		r := fullQueryString[j]

		if j == 0 {
			if l.invalidQuery {
				style = style.Bold(true).Foreground(tcell.GetColor("#BF616A"))
				r = 'X'
			} else if l.validQuery {
				style = style.Bold(true).Foreground(tcell.GetColor("#A3BE8C"))
				r = '✓'
			} else {
				style = style.Bold(true).Foreground(tcell.GetColor("#8FBCBB"))
				r = '?'
			}
		}
		l.screen.SetContent(j, height-1, r, nil, style)
	}

	for j < width {
		l.screen.SetContent(j, height-1, ' ', nil, tcell.StyleDefault)
		j++
	}
}

func (l *logViewer) redrawJSONPane() {
	for x := l.jsonPane.x0; x <= l.jsonPane.x1; x++ {
		for y := l.jsonPane.y0; y <= l.jsonPane.y1; y++ {
			l.screen.SetContent(x, y, ' ', nil, tcell.StyleDefault)
		}
	}

	p := l.offset + l.cursor
	b, err := json.MarshalIndent(l.file[p].rest, "", "  ")
	if err != nil {

	}

	lines := strings.Split(string(b), "\n")

	for i, line := range lines {
		y := l.jsonPane.y0 + i
		if y > l.jsonPane.y1 {
			break
		}

		for j, char := range line {
			if (l.jsonPane.x0 + j) > l.jsonPane.x1 {
				break
			}
			l.screen.SetContent(l.jsonPane.x0+j, y, char, nil, tcell.StyleDefault)
		}
	}
}

func (l *logViewer) scroll(amnt int) {
	oy := l.offset
	cy := l.cursor
	height := l.logPane.y1 - l.logPane.y0 + 1

	defer l.redrawLogPercent()

	newCy := cy + amnt
	if newCy < 0 {
		if oy+newCy < 0 {
			l.offset = 0
		} else {
			l.offset = oy + newCy
		}
		l.cursor = 0
		for i := 0; i < height; i++ {
			l.renderLogMessage(i, i+l.offset)
		}
		return
	}

	b := len(l.file)

	if newCy >= height {
		maxOrigin := b - height
		if maxOrigin < 0 {
			maxOrigin = 0
		}

		if oy+newCy-(height-1) >= maxOrigin {
			l.offset = maxOrigin
		} else {
			l.offset = oy + newCy - (height - 1)
		}
		l.cursor = height - 1
		for i := 0; i < height; i++ {
			l.renderLogMessage(i, i+l.offset)
		}
		return
	}

	oldCursor := l.cursor
	l.cursor = newCy

	l.renderLogMessage(oldCursor, oldCursor+l.offset)
	l.renderLogMessage(l.cursor, l.cursor+l.offset)
}

func (l *logViewer) redrawLogPane() {
	height := l.logPane.y1 - l.logPane.y0 + 1

	if height != 0 {
		prevIndex := l.cursor + l.offset
		cursorFrac := float64(l.cursor*(l.logPane.y1-l.logPane.y0+1)) / float64(height)

		l.cursor = int(math.Floor(cursorFrac))
		l.offset = prevIndex - l.cursor

		if l.offset+(l.logPane.y1-l.logPane.y0) >= len(l.file) {
			l.offset = (len(l.file) + l.logPane.y0 - l.logPane.y1) - 1
		}
		l.cursor = prevIndex - l.offset
	}

	for i := 0; i < height; i++ {
		l.renderLogMessage(i, i+l.offset)
	}
	l.redrawLogPercent()
	l.redrawLogMatches()
}

func (l *logViewer) redrawLogPercent() {
	pct := ((l.cursor + l.offset) * 100) / len(l.file)
	pctStr := []rune(fmt.Sprintf(" %02d%% ", pct))

	for i, r := range pctStr {
		l.screen.SetContent(l.logPane.x1-2-len(pctStr)+i, l.logPane.y1+1, r, nil, tcell.StyleDefault)
	}
}

func (l *logViewer) redrawLogMatches() {
	numMatches := len(l.matchingLines)
	pctStr := fmt.Sprintf(" %d match", numMatches)

	if numMatches != 1 {
		pctStr += "es "
	} else {
		pctStr += " "
	}

	for i, r := range pctStr {
		l.screen.SetContent(l.logPane.x0+2+i, l.logPane.y1+1, r, nil, tcell.StyleDefault)
	}
}

func (l *logViewer) renderLogMessage(loc int, ind int) {
	line := l.file[ind]

	var j int

	y := l.logPane.y0 + loc

	style := tcell.StyleDefault.Reverse(loc == l.cursor)

	var textList []styledRune

	var indicator = "  "
	if _, ok := l.matchingLines[ind]; ok {
		indicator = "* "
	}

	textList = append(textList, styleString(indicator, style.Reverse(false).Bold(true))...)
	textList = append(textList, styleString(line.timestamp.Format(time.Stamp)+" ", style.Foreground(tcell.GetColor("#4C566A")))...)
	textList = append(textList, getColoredLvlCode(line.level, style)...)
	textList = append(textList, styledRune{' ', style.Foreground(tcell.GetColor("#4C566A"))})
	textList = append(textList, styleString(line.message, style)...)

	for j = 0; j < len(textList) && j <= (l.logPane.x1-l.logPane.x0); j++ {
		l.screen.SetContent(l.logPane.x0+j, y, textList[j].r, nil, textList[j].style)
	}

	for j <= (l.logPane.x1 - l.logPane.x0) {
		l.screen.SetContent(l.logPane.x0+j, y, ' ', nil, style)
		j++
	}
}

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Must provide exactly one argument (the log file), provided %d", len(os.Args))
		return
	}
	if _, err := os.Stat(os.Args[1]); err != nil {
		fmt.Printf("Unable to locate file %s", os.Args[1])
		return
	}

	screen, err := tcell.NewScreen()
	checkErr(err, "Unable to construct screen: %v")

	err = screen.Init()
	checkErr(err, "Unable to initialize screen: %v")
	defer screen.Fini()

	viewer := &logViewer{screen: screen}
	viewer.init(os.Args[1])

	for {
		ev := screen.PollEvent()
		if ev == nil {
			// Screen has been finalized, need to exit event loop
			break
		}

		switch event := ev.(type) {
		case *tcell.EventError:
			checkErr(event, "t-cell event loop error: %v")
		case *tcell.EventInterrupt:
			screen.Sync()
		case *tcell.EventKey:
			if event.Key() == tcell.KeyCtrlC {
				return
			}
			shouldQuit := viewer.handleKey(event)
			if shouldQuit {
				return
			}
		case *tcell.EventResize:
			viewer.resize(event.Size())
		}
	}
}
