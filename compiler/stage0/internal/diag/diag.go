package diag

import (
	"fmt"
	"io"
	"sort"
)

type Item struct {
	Filename string
	Line     int
	Col      int
	Msg      string
}

type Bag struct {
	Items []Item
}

func (b *Bag) Add(filename string, line int, col int, msg string) {
	b.Items = append(b.Items, Item{Filename: filename, Line: line, Col: col, Msg: msg})
}

func (b *Bag) AddAt(loc Loc, msg string) {
	b.Add(loc.Filename, loc.Line, loc.Col, msg)
}

type Loc struct {
	Filename string
	Line     int
	Col      int
}

func Print(w io.Writer, b *Bag) {
	if b == nil || len(b.Items) == 0 {
		return
	}
	items := make([]Item, 0, len(b.Items))
	items = append(items, b.Items...)
	sort.Slice(items, func(i, j int) bool {
		if items[i].Filename != items[j].Filename {
			return items[i].Filename < items[j].Filename
		}
		if items[i].Line != items[j].Line {
			return items[i].Line < items[j].Line
		}
		return items[i].Col < items[j].Col
	})
	for _, it := range items {
		fmt.Fprintf(w, "%s:%d:%d: error: %s\n", it.Filename, it.Line, it.Col, it.Msg)
	}
}
