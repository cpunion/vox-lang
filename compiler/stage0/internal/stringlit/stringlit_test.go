package stringlit

import "testing"

func TestDecodeRegular(t *testing.T) {
	got, err := Decode(`"a\nb\t\"c\"\\"`)
	if err != nil {
		t.Fatalf("decode regular: %v", err)
	}
	if got != "a\nb\t\"c\"\\" {
		t.Fatalf("unexpected decoded regular: %q", got)
	}
}

func TestDecodeTripleUnindent(t *testing.T) {
	got, err := Decode("\"\"\"\n    line1\n      line2\n    line3\n\"\"\"")
	if err != nil {
		t.Fatalf("decode triple: %v", err)
	}
	if got != "line1\n  line2\nline3" {
		t.Fatalf("unexpected decoded triple: %q", got)
	}
}

func TestDecodeTripleEscapes(t *testing.T) {
	got, err := Decode("\"\"\"\n    a\\n\\t\\\"x\\\"\n\"\"\"")
	if err != nil {
		t.Fatalf("decode triple escapes: %v", err)
	}
	if got != "a\n\t\"x\"" {
		t.Fatalf("unexpected decoded triple escapes: %q", got)
	}
}

func TestDecodeTripleRejectsTabIndent(t *testing.T) {
	_, err := Decode("\"\"\"\n\tbad\n\"\"\"")
	if err == nil {
		t.Fatalf("expected tab-indent decode error")
	}
}
