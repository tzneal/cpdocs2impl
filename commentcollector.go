package cpdocs2impl

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"log"
	"os"
	"sort"

	"golang.org/x/tools/go/packages"
)

type CommentCollector struct {
	// If set, RewriteFn will be called for files that need to be rewritten (used for unit test)
	RewriteFn       func(filename string, fset *token.FileSet, file *ast.File)
	ifcComments     map[string][]*ast.Comment
	implements      map[string]string
	processImports  bool
	replaceComments bool
}

func NewCommentCollector(processImports bool, replaceComments bool) *CommentCollector {
	c := &CommentCollector{
		ifcComments:     map[string][]*ast.Comment{},
		implements:      map[string]string{},
		processImports:  processImports,
		replaceComments: replaceComments,
	}
	c.RewriteFn = c.rewrite
	return c
}

func (c *CommentCollector) Process(pkgs []*packages.Package) {
	// walk the packages that the user specified
	for _, pkg := range pkgs {
		for _, syn := range pkg.Syntax {
			ast.Inspect(syn, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.GenDecl:
					for _, s := range n.Specs {
						switch s := s.(type) {
						case *ast.ValueSpec:
							// looking for var _ IfcType = (*ConcreteType)(nil) declarations
							c.visitValueSpec(pkg, s)
						case *ast.TypeSpec:
							// look for struct/interface type declarations
							c.visitTypeSpec(pkg, s)
						}
					}
				}
				return true
			})
		}

		// if requested, check for direct imports as well. This can
		// be used to discover stdlib interfaces like io.Writer if desired.
		if c.processImports {
			for _, imp := range pkg.Imports {
				for _, syn := range imp.Syntax {
					ast.Inspect(syn, func(n ast.Node) bool {
						switch n := n.(type) {
						case *ast.TypeSpec:
							c.visitTypeSpec(imp, n)
						}
						return true
					})
				}
			}
		}

		// Finally walk the packages we directly loaded
		for i, syn := range pkg.Syntax {
			updated := false
			ast.Inspect(syn, func(n ast.Node) bool {
				// possibly updating function level comments
				switch n := n.(type) {
				case *ast.FuncDecl:
					if c.visitFunc(pkg, n, syn) {
						updated = true
					}
				}
				return true
			})
			// and rewriting the source if required
			if updated {
				c.RewriteFn(pkg.CompiledGoFiles[i], pkg.Fset, syn)
			}
		}
	}
}
func (c *CommentCollector) visitValueSpec(pkg *packages.Package, vs *ast.ValueSpec) {
	if len(vs.Names) != 1 || len(vs.Values) != 1 {
		return
	}
	// 	looking for var _ IfcType = (*ConcreteType)(nil)
	isAnon := vs.Names[0].Name == "_"
	if !isAnon {
		return
	}
	cx, ok := vs.Values[0].(*ast.CallExpr)
	if !ok {
		return
	}
	if len(cx.Args) == 1 {
		nilArg, ok := cx.Args[0].(*ast.Ident)
		if !ok || nilArg.Name != "nil" {
			return
		}
	}

	px, ok := cx.Fun.(*ast.ParenExpr)
	if !ok {
		return
	}
	sx, ok := px.X.(*ast.StarExpr)
	if !ok {
		return
	}
	concreteType, ok := sx.X.(*ast.Ident)
	if !ok {
		return
	}

	ifc, _, _ := isInterface(pkg.TypesInfo.Types[vs.Type].Type)
	if ifc == nil {
		return
	}
	ifcType := pkg.TypesInfo.Types[vs.Type].Type
	concreteIdent := fmt.Sprintf("%s.%s", pkg.ID, concreteType.Name)
	c.implements[concreteIdent] = fmt.Sprintf("%s", ifcType)
}

func (c *CommentCollector) visitTypeSpec(pkg *packages.Package, ts *ast.TypeSpec) {
	switch t := ts.Type.(type) {
	case *ast.InterfaceType:
		for _, f := range t.Methods.List {
			if len(f.Names) != 1 {
				continue
			}
			// record method comments
			comment := f.Doc.Text()
			if comment == "" {
				continue
			}
			ident := fmt.Sprintf("%s.%s/%s", pkg.ID, ts.Name.Name, f.Names[0].Name)
			c.ifcComments[ident] = f.Doc.List
		}
	}

}

func (c *CommentCollector) visitFunc(pkg *packages.Package, f *ast.FuncDecl, file *ast.File) bool {
	// we only care about methods with receivers
	if f.Recv == nil {
		return false
	}
	recv := f.Recv.List[0]
	typ := pkg.TypesInfo.Types[recv.Type].Type
	if pt, ok := typ.(*types.Pointer); ok {
		typ = pt.Elem()
	}
	// should probably always be true, but might as well check
	nt, ok := typ.(*types.Named)
	if !ok {
		return false
	}

	concreteIdent := fmt.Sprintf("%s.%s", pkg.ID, nt.Obj().Name())
	ifcName, ok := c.implements[concreteIdent]
	if !ok {
		return false
	}
	ifcIdent := fmt.Sprintf("%s/%s", ifcName, f.Name.Name)
	comment, ok := c.ifcComments[ifcIdent]
	if !ok {
		return false
	}

	// this function already has a comment
	if f.Doc != nil && len(f.Doc.List) > 0 {
		// if we're not replacing, we're done
		if !c.replaceComments {
			return false
		}

		// otherwise we need to go nuke this doc's comments from the file
		for j, v := range file.Comments {
			if v == f.Doc {
				file.Comments[j] = nil
			}
		}
		comments := []*ast.CommentGroup{}
		for _, v := range file.Comments {
			if v != nil {
				comments = append(comments, v)
			}
		}
		file.Comments = comments

	}

	// add our comments to the file
	cg := &ast.CommentGroup{}
	for i, c := range comment {
		text := c.Text
		/* on the first line of the comment, we prepend a "\n", this fixes a case where
		   missing blank lines, such as in:
		   }
		   func (b *Blah) BlahBlah()....

		   cause us to put the comment on the same line as the preceding brace:
		   }// BlahBlah is an interface comment
		   func (b *Blah) BlahBlah()....
		*/
		if i == 0 {
			text = "\n" + text
		}
		cg.List = append(cg.List, &ast.Comment{
			Slash: f.Pos() - 1,
			Text:  text,
		})
	}
	// append to the file's comments, which we need to sort later
	file.Comments = append(file.Comments, cg)
	return true
}

func (c *CommentCollector) rewrite(filename string, fset *token.FileSet, file *ast.File) {
	f, err := os.Create(filename)
	if err != nil {
		log.Fatalf("unable to open output: %s", err)
	}
	defer f.Close()
	sort.Slice(file.Comments, func(a, b int) bool {
		return file.Comments[a].Pos() < file.Comments[b].Pos()
	})
	if err := format.Node(f, fset, file); err != nil {
		log.Fatalf("error formatting code: %s", err)
	}
}

func isInterface(t types.Type) (*types.Interface, string, bool) {
	var name string
	if nt, ok := t.(*types.Named); ok {
		name = nt.Obj().Name()
		t = nt.Underlying()
	}
	ifc, ok := t.(*types.Interface)
	return ifc, name, ok
}
