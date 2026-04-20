package analyzer

import (
	"go/ast"
	"go/token"
	"os"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// BuildSwapFix creates a SuggestedFix that swaps two declarations in place.
func BuildSwapFix(fset *token.FileSet, file *ast.File, src []byte, a, b ast.Decl) *analysis.SuggestedFix {
	aStart, aEnd := DeclRange(fset, src, a)
	bStart, bEnd := DeclRange(fset, src, b)

	// Ensure a comes before b
	if aStart > bStart {
		aStart, aEnd, bStart, bEnd = bStart, bEnd, aStart, aEnd
	}

	// Sanity: ranges must not overlap
	if aEnd > bStart {
		return nil
	}

	aText := make([]byte, aEnd-aStart)
	copy(aText, src[aStart:aEnd])
	bText := make([]byte, bEnd-bStart)
	copy(bText, src[bStart:bEnd])

	tf := fset.File(file.Pos())

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

// BuildReorderFix creates a SuggestedFix that reorders a sequence of declarations.
// currentDecls is the list of ast.Decl entries in current order.
// newOrder is a permutation: newOrder[i] is the index into currentDecls for position i.
func BuildReorderFix(fset *token.FileSet, file *ast.File, src []byte,
	currentDecls []ast.Decl, newOrder []int) *analysis.SuggestedFix {

	if len(currentDecls) != len(newOrder) {
		return nil
	}

	// Find the range of positions that changed
	firstChanged := -1
	lastChanged := -1
	for i, srcIdx := range newOrder {
		if srcIdx != i {
			if firstChanged == -1 {
				firstChanged = i
			}
			lastChanged = i
		}
	}

	if firstChanged == -1 {
		return nil
	}

	// Get byte ranges for each declaration (just the decl, no gaps)
	type declInfo struct {
		start, end int
	}
	infos := make([]declInfo, len(currentDecls))
	for i, d := range currentDecls {
		s, e := DeclRange(fset, src, d)
		infos[i] = declInfo{s, e}
	}

	// Compute the full range including gaps between declarations
	rangeStart := infos[firstChanged].start
	rangeEnd := infos[lastChanged].end

	// Collect gaps between consecutive declarations (blank lines, etc.)
	gaps := make([][]byte, len(currentDecls))
	for i := firstChanged; i < lastChanged; i++ {
		gapStart := infos[i].end
		gapEnd := infos[i+1].start
		if gapStart < gapEnd {
			gaps[i] = src[gapStart:gapEnd]
		}
	}

	// Build the replacement text: reordered declarations with preserved gaps
	var b strings.Builder
	for i := firstChanged; i <= lastChanged; i++ {
		srcIdx := newOrder[i]
		di := infos[srcIdx]
		b.Write(src[di.start:di.end])
		// Use the original gap at this position (preserves spacing pattern)
		if i < lastChanged {
			if gap := gaps[i]; gap != nil {
				b.Write(gap)
			} else {
				b.WriteByte('\n') // default: single blank line separator
			}
		}
	}

	tf := fset.File(file.Pos())

	return &analysis.SuggestedFix{
		Message: "Reorder declarations",
		TextEdits: []analysis.TextEdit{{
			Pos:     tf.Pos(rangeStart),
			End:     tf.Pos(rangeEnd),
			NewText: []byte(b.String()),
		}},
	}
}

// BuildMoveFix creates a SuggestedFix that moves a declaration to a target position.
// The declaration is removed from its current position and inserted before the target position.
func BuildMoveFix(fset *token.FileSet, file *ast.File, src []byte, toMove ast.Decl, insertBeforeOffset int) *analysis.SuggestedFix {
	moveStart, moveEnd := DeclRange(fset, src, toMove)
	moveText := make([]byte, moveEnd-moveStart)
	copy(moveText, src[moveStart:moveEnd])

	tf := fset.File(file.Pos())

	if moveStart < insertBeforeOffset {
		// Moving forward: delete first, then insert
		return &analysis.SuggestedFix{
			Message: "Reorder declarations",
			TextEdits: []analysis.TextEdit{
				{
					Pos:     tf.Pos(moveStart),
					End:     tf.Pos(moveEnd),
					NewText: nil,
				},
				{
					Pos:     tf.Pos(insertBeforeOffset),
					End:     tf.Pos(insertBeforeOffset),
					NewText: moveText,
				},
			},
		}
	}

	// Moving backward: insert first, then delete
	return &analysis.SuggestedFix{
		Message: "Reorder declarations",
		TextEdits: []analysis.TextEdit{
			{
				Pos:     tf.Pos(insertBeforeOffset),
				End:     tf.Pos(insertBeforeOffset),
				NewText: moveText,
			},
			{
				Pos:     tf.Pos(moveStart),
				End:     tf.Pos(moveEnd),
				NewText: nil,
			},
		},
	}
}

// ReadFileSource reads the source bytes for the file containing the given AST file.
func ReadFileSource(fset *token.FileSet, file *ast.File) ([]byte, error) {
	filename := fset.Position(file.Pos()).Filename
	return os.ReadFile(filename)
}

// DeclRange returns the byte offset range [start, end) for a declaration,
// including its doc comment and everything up to the end of the last line (including newline).
func DeclRange(fset *token.FileSet, src []byte, d ast.Decl) (start, end int) {
	startPos := d.Pos()
	switch n := d.(type) {
	case *ast.FuncDecl:
		if n.Doc != nil {
			startPos = n.Doc.Pos()
		}
	case *ast.GenDecl:
		if n.Doc != nil {
			startPos = n.Doc.Pos()
		}
	}

	start = fset.Position(startPos).Offset
	end = fset.Position(d.End()).Offset

	// Include everything up to and including the next newline
	// (captures inline comments like // want "...")
	for end < len(src) && src[end] != '\n' {
		end++
	}
	if end < len(src) && src[end] == '\n' {
		end++
	}

	return start, end
}
