package source

import "sort"
import "unicode/utf8"

// File holds a source file and precomputed line offsets for diagnostics.
type File struct {
	Name        string
	Input       string
	lineOffsets []int // 0-based byte offsets of each line start
}

func NewFile(name string, input string) *File {
	f := &File{Name: name, Input: input}
	f.lineOffsets = []int{0}
	for i := 0; i < len(input); i++ {
		if input[i] == '\n' {
			f.lineOffsets = append(f.lineOffsets, i+1)
		}
	}
	return f
}

// LineCol returns 1-based line/column for a byte offset.
// Column is counted in runes (Unicode code points), not bytes.
func (f *File) LineCol(off int) (int, int) {
	if off < 0 {
		off = 0
	}
	if off > len(f.Input) {
		off = len(f.Input)
	}
	// lineOffsets is sorted.
	i := sort.Search(len(f.lineOffsets), func(i int) bool { return f.lineOffsets[i] > off }) - 1
	if i < 0 {
		i = 0
	}
	lineStart := f.lineOffsets[i]
	col := 1
	pos := lineStart
	for pos < off {
		_, sz := utf8.DecodeRuneInString(f.Input[pos:])
		if sz <= 0 {
			sz = 1
		}
		// If the offset points into a rune's bytes, keep the previous column.
		if pos+sz > off {
			break
		}
		col++
		pos += sz
	}
	return i + 1, col
}

type Span struct {
	File       *File
	Start, End int // byte offsets [start, end)
}

func (s Span) LocStart() (filename string, line int, col int) {
	if s.File == nil {
		return "", 0, 0
	}
	line, col = s.File.LineCol(s.Start)
	return s.File.Name, line, col
}
