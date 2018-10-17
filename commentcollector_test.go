package cpdocs2impl_test

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/tzneal/cpdocs2impl"
	"golang.org/x/tools/go/packages"
)

func TestGolden(t *testing.T) {
	files, err := ioutil.ReadDir("testdata")
	if err != nil {
		t.Fatalf("error reading testdata package: %s", err)
	}

	cfg := &packages.Config{Mode: packages.LoadAllSyntax}
	pkgs, err := packages.Load(cfg, "github.com/tzneal/cpdocs2impl/testdata")
	if err != nil {
		t.Fatalf("error loading package: %s", err)
	}
	c := cpdocs2impl.NewCommentCollector(false, false)
	rewritten := map[string]string{}
	c.RewriteFn = func(filename string, fset *token.FileSet, file *ast.File) {
		buf := bytes.Buffer{}
		if err := format.Node(&buf, fset, file); err != nil {
			t.Fatalf("error formatting code: %s", err)
		}
		idx := strings.Index(filename, "testdata") + len("testdata") + 1
		filename = filename[idx:]
		rewritten[filename] = buf.String()
	}
	c.Process(pkgs)

	for _, fi := range files {
		if !strings.HasSuffix(fi.Name(), ".go") {
			continue
		}
		t.Run(fi.Name(), func(t *testing.T) {
			exp := readGolden(fi.Name())
			got := rewritten[fi.Name()]
			errorLineDiff(t, exp, got)
		})
	}
}

func errorLineDiff(t *testing.T, exp, got string) {
	expLines := strings.Split(strings.TrimSpace(exp), "\n")
	gotLines := strings.Split(strings.TrimSpace(got), "\n")
	nLines := len(expLines)
	if len(gotLines) < nLines {
		t.Errorf("line count differed, expected %d, got %d", nLines, len(gotLines))
		nLines = len(gotLines)
	} else if len(gotLines) > nLines {
		t.Errorf("line count differed, expected %d, got %d", nLines, len(gotLines))
	}
	for i := 0; i < nLines; i++ {
		exp := strings.TrimSpace(expLines[i])
		got := strings.TrimSpace(gotLines[i])
		if exp != got {
			t.Errorf("expected\n%s\ngot\n%s", exp, got)
		}
	}
}
func readGolden(baseName string) string {
	// foo.go + lden = foo.golden
	buf, err := ioutil.ReadFile("testdata/" + baseName + "lden")
	if err != nil {
		return ""
	}
	return string(buf)
}
