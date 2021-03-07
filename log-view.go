package main

import (
	"fmt"
	"math"
	"time"

	"github.com/gdamore/tcell"
)

type logView struct {
	// The location that the log will be displayed in (inclusive of endpoints)
	frame rect

	// The list of log entries
	file []logEntry

	// The lines that are matched by the search query
	starredLinesList []int
	starredLines     map[int]struct{}

	// The display mode for starred/unstarred lines.
	//  - If true, all entries are displayed and starred entries are annotated with a star
	//  - If false, only starred entries are displayed and no annotation is added at all
	// displayUnstarred bool

	// The offset is the amount that the top of the window is below the start of the logfile, and
	// the cursor is the location inside of the offset
	offset, cursor int
}

func (l *logView) setStarredLines(starred []int) {
	curEntry := l.cursor + l.offset

	if l.starredLinesList != nil {
		curEntry = l.starredLinesList[curEntry]
	}

	l.starredLines = make(map[int]struct{})

	var newEntry int

	if starred == nil {
		newEntry = curEntry
	} else {
		for i, line := range starred {
			if line <= curEntry {
				newEntry = i
			}

			l.starredLines[i] = struct{}{}
		}
	}

	l.starredLinesList = starred

	height := l.frame.inset().height()
	newOrigin := newEntry - height/2
	if newOrigin < 0 {
		newOrigin = 0
	} else if newOrigin+height-1 >= l.bufferLength() {
		newOrigin = l.bufferLength() - height
	}

	newCursor := newEntry - newOrigin

	l.cursor = newCursor
	l.offset = newOrigin
}

func (l *logView) nextMatch(screen tcell.Screen) {
	// pos := l.getEntry()
	// for j := pos + 1; j < len(l.file); j++ {
	// 	if _, ok := l.starredLines[j]; ok {
	// 		l.scroll(screen, j-pos)
	// 		return
	// 	}
	// }
}

func (l *logView) prevMatch(screen tcell.Screen) {
	// pos := l.getEntry()
	// for j := pos - 1; j >= 0; j-- {
	// 	if _, ok := l.starredLines[j]; ok {
	// 		l.scroll(screen, j-pos)
	// 		return
	// 	}
	// }
}

func (l *logView) bufferLength() int {
	if len(l.starredLinesList) > 0 {
		return len(l.starredLinesList)
	}

	return len(l.file)
}

func (l *logView) messageAt(ind int) logEntry {
	if len(l.starredLinesList) > 0 {
		return l.file[l.starredLinesList[ind]]
	}

	return l.file[ind]
}

func (l *logView) currentMessage() logEntry {
	return l.messageAt(l.cursor + l.offset)
}

func (l *logView) scroll(screen tcell.Screen, amnt int) {
	defer l.renderPercent(screen)

	contentFrame := l.frame.inset()
	height := contentFrame.y1 - contentFrame.y0 + 1
	fileLength := l.bufferLength()

	newIndex := l.offset + l.cursor + amnt

	if newIndex < 0 {
		newIndex = 0
	}
	if newIndex >= fileLength {
		newIndex = fileLength - 1
	}
	if l.offset > newIndex {
		l.offset = newIndex
		l.cursor = 0
		l.draw(screen)
	} else if l.offset+height-1 < newIndex {
		l.offset = newIndex - height + 1
		l.cursor = height - 1
		l.draw(screen)
	} else {
		oldCursor := l.cursor
		l.cursor = newIndex - l.offset
		l.renderLogMessage(screen, oldCursor, oldCursor+l.offset)
		l.renderLogMessage(screen, l.cursor, l.cursor+l.offset)
	}
}

func (l *logView) resize(screen tcell.Screen, newFrame rect) {
	frameHeight := newFrame.inset().height()

	if frameHeight != 0 {
		prevIndex := l.cursor + l.offset
		cursorFrac := float64(l.cursor*frameHeight) / float64(l.frame.inset().height())

		l.cursor = int(math.Round(cursorFrac))
		l.offset = prevIndex - l.cursor

		if l.offset+l.frame.inset().height() > l.bufferLength() {
			l.offset = l.bufferLength() - l.frame.inset().height() - 1
		}
		if l.offset < 0 {
			l.offset = 0
		}

		l.cursor = prevIndex - l.offset
	}

	l.frame = newFrame
	l.draw(screen)
}

func (l *logView) draw(screen tcell.Screen) {
	drawFrame(screen, l.frame, "log")

	contentFrame := l.frame.inset()

	for y := contentFrame.y0; y <= contentFrame.y1; y++ {
		l.renderLogMessage(screen, y-contentFrame.y0, y-contentFrame.y0+l.offset)
	}

	// Redraw the "n matches" section at the bottom of the frame
	numMatches := len(l.starredLines)
	matchStr := fmt.Sprintf(" %d match", numMatches)

	if numMatches != 1 {
		matchStr += "es "
	} else {
		matchStr += " "
	}

	for i, r := range matchStr {
		screen.SetContent(l.frame.x0+2+i, l.frame.y1, r, nil, tcell.StyleDefault)
	}

	l.renderPercent(screen)
}

// Redraw the percentage displayed at the bottom of the frame
func (l *logView) renderPercent(screen tcell.Screen) {
	pct := ((l.cursor + l.offset) * 100) / l.bufferLength()
	pctStr := []rune(fmt.Sprintf(" %02d%% ", pct))

	for i, r := range pctStr {
		screen.SetContent(l.frame.x1-2-len(pctStr)+i, l.frame.y1, r, nil, tcell.StyleDefault)
	}
}

func (l *logView) renderLogMessage(screen tcell.Screen, loc int, ind int) {
	var textList []styledRune
	style := tcell.StyleDefault.Reverse(loc == l.cursor)

	if ind < l.bufferLength() {
		line := l.messageAt(ind)

		var indicator = "  "
		if _, ok := l.starredLines[ind]; ok {
			indicator = "* "
		}

		textList = append(textList, styleString(indicator, style.Reverse(false).Bold(true))...)
		textList = append(textList, styleString(line.timestamp.Format(time.StampMilli)+" ", style.Foreground(tcell.GetColor("#4C566A")))...)
		textList = append(textList, getColoredLvlCode(line.level, style)...)
		textList = append(textList, styledRune{' ', style.Foreground(tcell.GetColor("#4C566A"))})
		textList = append(textList, styleString(line.message, style)...)
	}

	// Render the text to the screen and empty all of the characters occurring after the finish of
	// line
	contentFrame := l.frame.inset()
	var j int
	for j = 0; j < len(textList) && j <= (contentFrame.x1-contentFrame.x0); j++ {
		screen.SetContent(contentFrame.x0+j, contentFrame.y0+loc, textList[j].r, nil, textList[j].style)
	}

	for j <= (contentFrame.x1 - contentFrame.x0) {
		screen.SetContent(contentFrame.x0+j, contentFrame.y0+loc, ' ', nil, style)
		j++
	}
}
