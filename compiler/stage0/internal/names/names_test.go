package names

import (
	"reflect"
	"testing"
)

func TestGroupQualifiedTestsByModule(t *testing.T) {
	in := []string{
		"a::test_one",
		"a::test_two",
		"pkg.dep::a.b::test_dep_one",
		"pkg.dep::a.b::test_dep_two",
		"tests::test_it",
		"test_root",
	}
	got := GroupQualifiedTestsByModule(in)
	want := []TestModuleGroup{
		{Key: "", Tests: []string{"test_root"}},
		{Key: "a", Tests: []string{"a::test_one", "a::test_two"}},
		{Key: "pkg.dep::a.b", Tests: []string{"pkg.dep::a.b::test_dep_one", "pkg.dep::a.b::test_dep_two"}},
		{Key: "tests", Tests: []string{"tests::test_it"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected groups:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestGroupQualifiedTestsByModule_Empty(t *testing.T) {
	if got := GroupQualifiedTestsByModule(nil); got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
}
