package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gdamore/tcell"
)

func checkErr(err error, message string) {
	if err != nil {
		panic(fmt.Sprintf(message, err))
	}
}

type (
	logEntry struct {
		timestamp time.Time
		caller    string
		level     string
		message   string
		rest      map[string]interface{}
	}
)

func parseJSONLogFile(filename string) []logEntry {
	lf, err := os.Open(filename)
	if err != nil {
		panic("Unable to read specified file!")
	}
	defer lf.Close()

	dec := json.NewDecoder(lf)
	var f []logEntry

	for {
		var data map[string]interface{}
		err := dec.Decode(&data)
		if err != nil {
			break
		}

		var logTime time.Time

		timeString, ok := data["time"].(string)
		if !ok {
			timeMS := data["time"].(float64)
			logTime = time.Unix(0, int64(timeMS)*int64(time.Millisecond)/int64(time.Nanosecond))
		} else {
			logTime, _ = time.Parse(time.RFC3339, timeString)
		}

		caller := data["caller"].(string)
		level := data["level"].(string)
		message := data["message"].(string)

		f = append(f, logEntry{
			timestamp: logTime,
			caller:    caller,
			level:     level,
			message:   message,
			rest:      data,
		})
	}

	return f
}

type rect struct {
	x0, y0, x1, y1 int
}

func (r rect) inset() rect {
	return rect{
		r.x0 + 1, r.y0 + 1, r.x1 - 1, r.y1 - 1,
	}
}

func (r rect) height() int {
	return r.y1 - r.y0 + 1
}

func (r rect) dims() (int, int) {
	return (r.x1 - r.x0 + 1), (r.y1 - r.y0 + 1)
}

func drawFrame(s tcell.Screen, r rect, name string) {
	s.SetContent(r.x0, r.y0, runeTL, nil, tcell.StyleDefault)
	s.SetContent(r.x0, r.y1, runeBL, nil, tcell.StyleDefault)
	s.SetContent(r.x1, r.y0, runeTR, nil, tcell.StyleDefault)
	s.SetContent(r.x1, r.y1, runeBR, nil, tcell.StyleDefault)

	for x := r.x0 + 1; x < r.x1; x++ {
		s.SetContent(x, r.y0, runeH, nil, tcell.StyleDefault)
		s.SetContent(x, r.y1, runeH, nil, tcell.StyleDefault)
	}

	for y := r.y0 + 1; y < r.y1; y++ {
		s.SetContent(r.x0, y, runeV, nil, tcell.StyleDefault)
		s.SetContent(r.x1, y, runeV, nil, tcell.StyleDefault)
	}

	fullName := " " + name + " "

	for i := 0; i < len(fullName); i++ {
		s.SetContent(r.x0+i+2, r.y0, []rune(fullName)[i], nil, tcell.StyleDefault)
	}
}

type styledRune struct {
	r     rune
	style tcell.Style
}

func styleString(inp string, style tcell.Style) []styledRune {
	out := make([]styledRune, len([]rune(inp)))
	for i, r := range inp {
		out[i] = styledRune{r, style}
	}
	return out
}

func getColoredLvlCode(lvl string, style tcell.Style) []styledRune {
	var levelToColoredStr = map[string][]styledRune{
		"panic": styleString("PNC", style.Foreground(tcell.GetColor("#BF616A"))),
		"fatal": styleString("FTL", style.Foreground(tcell.GetColor("#BF616A"))),
		"error": styleString("ERR", style.Foreground(tcell.GetColor("#D08770"))),
		"warn":  styleString("WRN", style.Foreground(tcell.GetColor("#EBCB8B"))),
		"info":  styleString("INF", style.Foreground(tcell.GetColor("#A3BE8C"))),
		"debug": styleString("DBG", style.Foreground(tcell.GetColor("#81A1C1"))),
		"trace": styleString("TRC", style.Foreground(tcell.GetColor("#8FBCBB"))),
	}

	return levelToColoredStr[lvl]
}

// var writeFunc = make(chan string)

// func init() {
// 	go func() {
// 		f, err := os.Create("jlv-log.txt")
// 		checkErr(err, "Unable to create log file!")
// 		defer f.Close()

// 		for {
// 			s := <-writeFunc
// 			f.Write([]byte(s))
// 		}
// 	}()
// }

// func logToFile(s string) {
// 	writeFunc <- s
// }
