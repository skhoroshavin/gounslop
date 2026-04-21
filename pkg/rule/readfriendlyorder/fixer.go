package readfriendlyorder

import (
	"go/ast"
	"go/token"
	"os"
	"strings"

	"golang.org/x/tools/go/analysis"
)

func (ctx *fixContext) buildSwapFix(a, b ast.Decl) *analysis.SuggestedFix {
	aStart, aEnd := ctx.declRange(a)
	bStart, bEnd := ctx.declRange(b)

	if aStart > bStart {
		aStart, aEnd, bStart, bEnd = bStart, bEnd, aStart, aEnd
	}

	if aEnd > bStart {
		return nil
	}

	aText := make([]byte, aEnd-aStart)
	copy(aText, ctx.src[aStart:aEnd])
	bText := make([]byte, bEnd-bStart)
	copy(bText, ctx.src[bStart:bEnd])

	tf := ctx.fset.File(ctx.file.Pos())

	return &analysis.SuggestedFix{
		Message: "Reorder declarations",
		TextEdits: []analysis.TextEdit{
			{
				Pos:     tf.Pos(aStart),
				End:     tf.Pos(aEnd),
				NewText: bText,
			},
			{
				Pos:     tf.Pos(bStart),
				End:     tf.Pos(bEnd),
				NewText: aText,
			},
		},
	}
}

func (ctx *fixContext) computeReorderFix(currentDecls []ast.Decl, newOrder []int) *analysis.SuggestedFix {
	if len(currentDecls) != len(newOrder) {
		return nil
	}

	changed := findChangedRange(newOrder)
	if changed.first < 0 {
		return nil
	}

	infos := ctx.buildDeclInfos(currentDecls)
	rangeStart := infos[changed.first].start
	rangeEnd := infos[changed.last].end

	reordered := reorderTextBuilder{
		src:      ctx.src,
		infos:    infos,
		newOrder: newOrder,
		gaps:     extractGaps(ctx.src, infos, changed),
		changed:  changed,
	}.buildReorderedText()

	tf := ctx.fset.File(ctx.file.Pos())
	return &analysis.SuggestedFix{
		Message: "Reorder declarations",
		TextEdits: []analysis.TextEdit{{
			Pos:     tf.Pos(rangeStart),
			End:     tf.Pos(rangeEnd),
			NewText: []byte(reordered),
		}},
	}
}

func findChangedRange(newOrder []int) changedRange {
	changed := changedRange{first: -1, last: -1}
	for i, srcIdx := range newOrder {
		if srcIdx != i {
			if changed.first == -1 {
				changed.first = i
			}
			changed.last = i
		}
	}
	return changed
}

func (ctx *fixContext) buildDeclInfos(decls []ast.Decl) []declInfo {
	infos := make([]declInfo, len(decls))
	for i, d := range decls {
		s, e := ctx.declRange(d)
		infos[i] = declInfo{s, e}
	}
	return infos
}

func extractGaps(src []byte, infos []declInfo, changed changedRange) [][]byte {
	gaps := make([][]byte, len(infos))
	for i := changed.first; i < changed.last; i++ {
		gapStart := infos[i].end
		gapEnd := infos[i+1].start
		if gapStart < gapEnd {
			gaps[i] = src[gapStart:gapEnd]
		}
	}
	return gaps
}

func (builder reorderTextBuilder) buildReorderedText() string {
	var b strings.Builder
	for i := builder.changed.first; i <= builder.changed.last; i++ {
		srcIdx := builder.newOrder[i]
		di := builder.infos[srcIdx]
		b.Write(builder.src[di.start:di.end])
		if i < builder.changed.last {
			if gap := builder.gaps[i]; gap != nil {
				b.Write(gap)
			} else {
				b.WriteByte('\n')
			}
		}
	}
	return b.String()
}

func (ctx *fixContext) buildMoveFix(toMove ast.Decl, insertBeforeOffset int) *analysis.SuggestedFix {
	moveStart, moveEnd := ctx.declRange(toMove)
	moveText := make([]byte, moveEnd-moveStart)
	copy(moveText, ctx.src[moveStart:moveEnd])

	tf := ctx.fset.File(ctx.file.Pos())

	edits := []analysis.TextEdit{
		{Pos: tf.Pos(moveStart), End: tf.Pos(moveEnd), NewText: nil},
		{Pos: tf.Pos(insertBeforeOffset), End: tf.Pos(insertBeforeOffset), NewText: moveText},
	}
	if moveStart < insertBeforeOffset {
		edits[0], edits[1] = edits[1], edits[0]
	}

	return &analysis.SuggestedFix{
		Message:   "Reorder declarations",
		TextEdits: edits,
	}
}

func readFileSource(fset *token.FileSet, file *ast.File) ([]byte, error) {
	filename := fset.Position(file.Pos()).Filename
	return os.ReadFile(filename)
}

func (ctx *fixContext) declRange(d ast.Decl) (start, end int) {
	start = ctx.fset.Position(declStartPos(d)).Offset
	end = extendDeclEnd(ctx.src, ctx.fset.Position(d.End()).Offset)
	return start, end
}

func declStartPos(d ast.Decl) token.Pos {
	switch n := d.(type) {
	case *ast.FuncDecl:
		return docStartPos(n.Doc, d.Pos())
	case *ast.GenDecl:
		return docStartPos(n.Doc, d.Pos())
	default:
		return d.Pos()
	}
}

func docStartPos(doc *ast.CommentGroup, fallback token.Pos) token.Pos {
	if doc == nil {
		return fallback
	}
	return doc.Pos()
}

func extendDeclEnd(src []byte, end int) int {
	for end < len(src) && src[end] != '\n' {
		end++
	}
	if end < len(src) && src[end] == '\n' {
		end++
	}
	return end
}

func newFixContext(fset *token.FileSet, file *ast.File, src []byte) *fixContext {
	return &fixContext{fset: fset, file: file, src: src}
}

type fixContext struct {
	fset *token.FileSet
	file *ast.File
	src  []byte
}

type reorderTextBuilder struct {
	src      []byte
	infos    []declInfo
	newOrder []int
	gaps     [][]byte
	changed  changedRange
}

type declInfo struct {
	start, end int
}

type changedRange struct {
	first int
	last  int
}
