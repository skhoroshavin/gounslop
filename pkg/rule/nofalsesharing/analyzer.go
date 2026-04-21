package nofalsesharing

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"maps"
	"slices"
	"strings"
	"sync"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"

	"github.com/skhoroshavin/gounslop/pkg/core/boundary"
	"github.com/skhoroshavin/gounslop/pkg/core/module"
)

func NewAnalyzer(modCache *module.Cache, cfg boundary.SharedRules) *analysis.Analyzer {
	fsCache := &cache{}

	return &analysis.Analyzer{
		Name: "nofalsesharing",
		Doc:  "detect exported symbols in shared packages that are not used by 2+ entities",
		Run: func(pass *analysis.Pass) (any, error) {
			return run(pass, modCache, fsCache, cfg)
		},
	}
}

func run(pass *analysis.Pass, modCache *module.Cache, fsCache *cache, cfg boundary.SharedRules) (any, error) {
	if !cfg.HasSharedSelectors {
		return nil, nil
	}

	info, err := modCache.Discover(pass)
	if err != nil {
		return nil, fmt.Errorf("nofalsesharing: %w", err)
	}

	diagnostics, err := fsCache.load(info, cfg)
	if err != nil {
		return nil, err
	}

	packageDiagnostics, ok := diagnostics[pass.Pkg.Path()]
	if !ok || len(packageDiagnostics) == 0 {
		return nil, nil
	}

	walker := &declarationWalker{files: pass.Files, info: pass.TypesInfo, pkg: pass.Pkg}
	locations := walker.collectSymbolLocations()
	for _, key := range slices.Sorted(maps.Keys(packageDiagnostics)) {
		if loc, ok := locations[key]; ok {
			pass.Reportf(loc.pos, "%s", packageDiagnostics[key])
		}
	}

	return nil, nil
}

type cache struct {
	cache sync.Map
}

func (c *cache) load(info module.Info, cfg boundary.SharedRules) (map[string]map[string]string, error) {
	key := cacheKey{
		moduleDir: info.Dir,
		configKey: cfg.CacheKey,
	}

	entryValue, _ := c.cache.LoadOrStore(key, &cacheEntry{})
	entry := entryValue.(*cacheEntry)
	entry.once.Do(func() {
		entry.diagnostics, entry.err = analyzeFalseSharing(info, cfg)
	})

	return entry.diagnostics, entry.err
}

type cacheKey struct {
	moduleDir string
	configKey string
}

type cacheEntry struct {
	once        sync.Once
	diagnostics map[string]map[string]string
	err         error
}

func analyzeFalseSharing(info module.Info, cfg boundary.SharedRules) (map[string]map[string]string, error) {
	packagesByPath, err := loadModulePackages(info.Dir)
	if err != nil {
		return nil, err
	}

	sharedPkgs := collectSharedPackages(packagesByPath, info, cfg.Selectors)
	if len(sharedPkgs) == 0 {
		return nil, nil
	}

	exposures := collectExposures(packagesByPath, sharedPkgs, info)

	usageMap := make(symbolUsageMap)
	for _, symbols := range sharedPkgs {
		for key, usage := range symbols {
			usageMap[key] = usage
		}
	}

	a := &usageAnalyzer{
		sharedPkgs: sharedPkgs,
		exposures:  exposures,
		usageMap:   usageMap,
	}
	analyzeUsageAcrossPackages(packagesByPath, info, a)
	return sharedPkgs.buildDiagnostics(), nil
}

func loadModulePackages(moduleDir string) (map[string]*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedImports |
			packages.NeedCompiledGoFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo,
		Dir: moduleDir,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("nofalsesharing: loading packages for shared-package analysis: %w", err)
	}

	packagesByPath := make(map[string]*packages.Package)
	for _, pkg := range pkgs {
		if pkg == nil || pkg.PkgPath == "" {
			continue
		}

		packagesByPath[pkg.PkgPath] = pkg
	}

	return packagesByPath, nil
}

func collectSharedPackages(
	packagesByPath map[string]*packages.Package,
	info module.Info,
	selectors []boundary.Selector,
) sharedPackageMap {
	sharedPkgs := make(sharedPackageMap)
	for pkgPath := range packagesByPath {
		relPath, ownership := module.ClassifyPath(pkgPath, info)
		if ownership != module.CurrentModule {
			continue
		}

		_, found := boundary.FindSharedSelector(selectors, relPath)
		if !found {
			continue
		}

		symbols := collectSharedPackageSymbols(packagesByPath[pkgPath])
		if len(symbols) == 0 {
			continue
		}

		sharedPkgs[pkgPath] = symbols
	}

	return sharedPkgs
}

func collectSharedPackageSymbols(pkg *packages.Package) map[string]*symbolUsage {
	walker := &declarationWalker{files: pkg.Syntax, info: pkg.TypesInfo, pkg: pkg.Types}
	locations := walker.collectSymbolLocations()
	if len(locations) == 0 {
		return nil
	}

	symbols := make(map[string]*symbolUsage, len(locations))
	for key, loc := range locations {
		symbols[key] = &symbolUsage{
			displayName: loc.displayName,
			consumers:   make(map[string]struct{}),
		}
	}

	return symbols
}

// collectExposures performs Pass 1: for each non-shared package, find all exported symbols
// that reference shared types — these are "exposure points" that indirectly pull in shared types.
// The result maps: package path → symbol key → shared type keys it exposes.
func collectExposures(
	packagesByPath map[string]*packages.Package,
	sharedPkgs sharedPackageMap,
	info module.Info,
) typeExposureMap {
	exposures := make(typeExposureMap)

	for pkgPath, pkg := range packagesByPath {
		if _, isShared := sharedPkgs[pkgPath]; isShared {
			continue
		}
		_, ownership := module.ClassifyPath(pkgPath, info)
		if ownership != module.CurrentModule {
			continue
		}
		if pkg == nil || pkg.TypesInfo == nil {
			continue
		}
		collectPackageExposures(pkg, sharedPkgs, exposures)
	}

	return exposures
}

func collectPackageExposures(pkg *packages.Package, sharedPkgs sharedPackageMap, exposures typeExposureMap) {
	walker := &declarationWalker{files: pkg.Syntax, info: pkg.TypesInfo, pkg: pkg.Types}
	walker.walk(func(_ ast.Node, ownedObjects []types.Object) {
		for _, obj := range ownedObjects {
			if !obj.Exported() {
				continue
			}
			sharedTypes := findExposedSharedTypes(obj, sharedPkgs)
			if len(sharedTypes) > 0 {
				id := identifySymbol(obj)
				if id.isValid() {
					if exposures[id.pkgPath] == nil {
						exposures[id.pkgPath] = make(map[string][]string)
					}
					exposures[id.pkgPath][id.key] = sharedTypes
				}
			}
		}
	})
}

func findExposedSharedTypes(obj types.Object, sharedPkgs sharedPackageMap) []string {
	switch typedObj := obj.(type) {
	case *types.TypeName:
		if named, ok := typedObj.Type().(*types.Named); ok {
			return findSharedTypesInType(named.Underlying(), sharedPkgs)
		}
	case *types.Var:
		if !typedObj.IsField() {
			return findSharedTypesInType(typedObj.Type(), sharedPkgs)
		}
	case *types.Const:
		return findSharedTypesInType(typedObj.Type(), sharedPkgs)
	case *types.Func:
		return findSharedTypesInType(typedObj.Type(), sharedPkgs)
	}
	return nil
}

// findSharedTypesInType recursively walks a Go type and returns the keys of any
// shared-package symbols it references (directly or through pointer/slice/struct/etc.).
func findSharedTypesInType(t types.Type, sharedPkgs sharedPackageMap) []string {
	s := &sharedTypeSearcher{sharedPkgs: sharedPkgs, seen: make(map[types.Type]struct{})}
	s.search(t)
	return s.found
}

type sharedTypeSearcher struct {
	sharedPkgs sharedPackageMap
	seen       map[types.Type]struct{}
	found      []string
}

func (s *sharedTypeSearcher) searchElement(t types.Type) bool {
	switch typed := t.(type) {
	case *types.Pointer:
		s.search(typed.Elem())
	case *types.Slice:
		s.search(typed.Elem())
	case *types.Array:
		s.search(typed.Elem())
	case *types.Chan:
		s.search(typed.Elem())
	case *types.Map:
		s.search(typed.Key())
		s.search(typed.Elem())
	case *types.TypeParam:
		s.search(typed.Constraint())
	default:
		return false
	}
	return true
}

func (s *sharedTypeSearcher) search(t types.Type) {
	if t == nil {
		return
	}
	if _, ok := s.seen[t]; ok {
		return
	}
	s.seen[t] = struct{}{}

	if s.searchElement(t) {
		return
	}

	switch typed := t.(type) {
	case *types.Named:
		s.searchNamed(typed)
	case *types.Signature:
		s.searchSignature(typed)
	case *types.Struct:
		s.searchStruct(typed)
	case *types.Interface:
		s.searchInterface(typed)
	}
}

func (s *sharedTypeSearcher) searchNamed(t *types.Named) {
	if t.Obj() != nil && t.Obj().Pkg() != nil {
		pkgPath := t.Obj().Pkg().Path()
		if sharedPkg, ok := s.sharedPkgs[pkgPath]; ok {
			typeKey := simpleSymbolID(pkgPath, "type", t.Obj().Name()).key
			if _, exists := sharedPkg[typeKey]; exists {
				s.found = append(s.found, typeKey)
			}
		}
	}
	s.search(t.Underlying())
}

func (s *sharedTypeSearcher) searchSignature(sig *types.Signature) {
	for i := 0; i < sig.Params().Len(); i++ {
		s.search(sig.Params().At(i).Type())
	}
	for i := 0; i < sig.Results().Len(); i++ {
		s.search(sig.Results().At(i).Type())
	}
}

func (s *sharedTypeSearcher) searchStruct(strct *types.Struct) {
	for i := 0; i < strct.NumFields(); i++ {
		if field := strct.Field(i); field.Exported() {
			s.search(field.Type())
		}
	}
}

func (s *sharedTypeSearcher) searchInterface(iface *types.Interface) {
	for i := 0; i < iface.NumMethods(); i++ {
		if method := iface.Method(i); method.Exported() {
			s.search(method.Type())
		}
	}
}

func analyzeUsageAcrossPackages(
	packagesByPath map[string]*packages.Package,
	info module.Info,
	a *usageAnalyzer,
) {
	for pkgPath, pkg := range packagesByPath {
		consumerRelPath, ownership := module.ClassifyPath(pkgPath, info)
		if ownership != module.CurrentModule {
			continue
		}
		hasNonTest := false
		for _, filePath := range pkg.CompiledGoFiles {
			if !strings.HasSuffix(filePath, "_test.go") {
				hasNonTest = true
				break
			}
		}
		if !hasNonTest {
			continue
		}

		// Use "." for the root package so it counts as a distinct consumer name.
		consumer := consumerRelPath
		if consumer == "" {
			consumer = "."
		}

		a.analyzePackage(pkgPath, pkg, consumer)
	}
}

func walkReferencedObjects(node ast.Node, info *types.Info, visit func(types.Object)) {
	var stack []ast.Node
	ast.Inspect(node, func(n ast.Node) bool {
		if n == nil {
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			return false
		}
		if sel, ok := n.(*ast.SelectorExpr); ok {
			visitSelectorObject(sel, info, visit)
		} else if ident, ok := n.(*ast.Ident); ok {
			visitIdentObject(ident, stack, info, visit)
		}
		stack = append(stack, n)
		return true
	})
}

func visitSelectorObject(sel *ast.SelectorExpr, info *types.Info, visit func(types.Object)) {
	if selection, ok := info.Selections[sel]; ok {
		visit(selection.Obj())
	} else if obj := info.Uses[sel.Sel]; obj != nil {
		visit(obj)
	}
}

func visitIdentObject(ident *ast.Ident, stack []ast.Node, info *types.Info, visit func(types.Object)) {
	if len(stack) > 0 {
		if sel, ok := stack[len(stack)-1].(*ast.SelectorExpr); ok && sel.Sel == ident {
			return
		}
	}
	if obj := info.Uses[ident]; obj != nil {
		visit(obj)
	}
}

// declarationWalker bundles the AST and type information needed for walking declarations.
type declarationWalker struct {
	files []*ast.File
	info  *types.Info
	pkg   *types.Package
}

func (w *declarationWalker) collectSymbolLocations() map[string]symbolLocation {
	locations := make(map[string]symbolLocation)
	if w.info == nil || w.pkg == nil {
		return locations
	}

	w.walk(func(_ ast.Node, ownedObjects []types.Object) {
		for _, obj := range ownedObjects {
			id := identifySymbol(obj)
			locations[id.key] = symbolLocation{
				displayName: id.displayName,
				pos:         obj.Pos(),
			}
		}
	})

	return locations
}

func (w *declarationWalker) walk(visit func(ast.Node, []types.Object)) {
	for _, file := range w.files {
		for _, decl := range file.Decls {
			switch typedDecl := decl.(type) {
			case *ast.FuncDecl:
				obj := w.info.Defs[typedDecl.Name]
				var owned []types.Object
				if w.isDefinedInPackage(obj) {
					owned = []types.Object{obj}
				}
				visit(typedDecl, owned)
			case *ast.GenDecl:
				for _, spec := range typedDecl.Specs {
					visit(spec, w.ownedSpecObjects(spec))
				}
			}
		}
	}
}

func (w *declarationWalker) ownedSpecObjects(spec ast.Spec) []types.Object {
	var owned []types.Object
	switch typedSpec := spec.(type) {
	case *ast.TypeSpec:
		obj := w.info.Defs[typedSpec.Name]
		if w.isDefinedInPackage(obj) {
			owned = append(owned, obj)
		}
	case *ast.ValueSpec:
		for _, name := range typedSpec.Names {
			obj := w.info.Defs[name]
			if w.isDefinedInPackage(obj) {
				owned = append(owned, obj)
			}
		}
	}
	return owned
}

func (w *declarationWalker) isDefinedInPackage(obj types.Object) bool {
	if obj == nil {
		return false
	}
	id := identifySymbol(obj)
	return id.isValid() && id.pkgPath == w.pkg.Path()
}

type usageContext struct {
	pkgPath      string
	consumer     string
	declaredKeys map[string]struct{}
	sharedPkgs   sharedPackageMap
	exposures    typeExposureMap
	usageMap     symbolUsageMap
}

func (ctx *usageContext) recordUsage(obj types.Object) {
	id := identifySymbol(obj)
	if !id.isValid() {
		return
	}

	ctx.creditIndirectUsage(id)
	ctx.creditDirectUsage(id)
}

// creditIndirectUsage: if obj is an exposure point (a non-shared symbol that exposes a shared type),
// then whoever uses obj is also using those shared types indirectly.
func (ctx *usageContext) creditIndirectUsage(id symbolID) {
	if sharedTypes, ok := ctx.exposures[id.pkgPath][id.key]; ok {
		for _, sharedTypeKey := range sharedTypes {
			if usage := ctx.usageMap[sharedTypeKey]; usage != nil {
				usage.consumers[ctx.consumer] = struct{}{}
			}
		}
	}
}

// creditDirectUsage: if obj is itself a shared symbol, record the current package as a consumer.
func (ctx *usageContext) creditDirectUsage(id symbolID) {
	sharedPkg, ok := ctx.sharedPkgs[id.pkgPath]
	if !ok {
		return
	}

	usage, ok := sharedPkg[id.key]
	if !ok {
		return
	}

	if id.pkgPath == ctx.pkgPath {
		// Usage within the shared package itself - only count if not the defining declaration.
		if _, isDeclared := ctx.declaredKeys[id.key]; isDeclared {
			return
		}
		usage.hasInternalConsumer = true
		return
	}

	usage.consumers[ctx.consumer] = struct{}{}
}

type usageAnalyzer struct {
	sharedPkgs sharedPackageMap
	exposures  typeExposureMap
	usageMap   symbolUsageMap
}

func (a *usageAnalyzer) analyzePackage(pkgPath string, pkg *packages.Package, consumer string) {
	if pkg == nil || pkg.TypesInfo == nil {
		return
	}

	ctx := &usageContext{
		pkgPath:    pkgPath,
		consumer:   consumer,
		sharedPkgs: a.sharedPkgs,
		exposures:  a.exposures,
		usageMap:   a.usageMap,
	}

	walker := &declarationWalker{files: pkg.Syntax, info: pkg.TypesInfo, pkg: pkg.Types}
	walker.walk(func(node ast.Node, ownedObjects []types.Object) {
		declaredKeys := make(map[string]struct{}, len(ownedObjects))
		for _, obj := range ownedObjects {
			id := identifySymbol(obj)
			if id.isValid() && id.pkgPath == walker.pkg.Path() {
				declaredKeys[id.key] = struct{}{}
			}
		}
		ctx.declaredKeys = declaredKeys
		walkReferencedObjects(node, pkg.TypesInfo, func(obj types.Object) {
			ctx.recordUsage(obj)
		})
	})
}

func identifySymbol(obj types.Object) symbolID {
	if !isExportedSymbol(obj) {
		return symbolID{}
	}

	switch typedObject := obj.(type) {
	case *types.Func:
		return identifyFunc(typedObject)
	case *types.TypeName:
		return simpleSymbolID(typedObject.Pkg().Path(), "type", typedObject.Name())
	case *types.Var:
		return identifyVar(typedObject)
	case *types.Const:
		return simpleSymbolID(typedObject.Pkg().Path(), "const", typedObject.Name())
	default:
		return symbolID{}
	}
}

func isExportedSymbol(obj types.Object) bool {
	return obj != nil && obj.Pkg() != nil && obj.Exported()
}

func identifyVar(v *types.Var) symbolID {
	if v.IsField() {
		return symbolID{}
	}
	return simpleSymbolID(v.Pkg().Path(), "var", v.Name())
}

func identifyFunc(fn *types.Func) symbolID {
	signature := fn.Type().(*types.Signature)
	if signature.Recv() == nil {
		pkgPath := fn.Pkg().Path()
		return symbolID{
			pkgPath:     pkgPath,
			key:         symbolKey(pkgPath, "func", "", fn.Name()),
			displayName: fn.Name(),
		}
	}
	return identifyMethod(fn, signature)
}

func identifyMethod(fn *types.Func, sig *types.Signature) symbolID {
	receiver := sig.Recv().Type()
	if pointer, ok := receiver.(*types.Pointer); ok {
		receiver = pointer.Elem()
	}
	named, ok := receiver.(*types.Named)
	if !ok {
		return symbolID{}
	}
	namedObj := named.Obj()
	if namedObj == nil || namedObj.Pkg() == nil {
		return symbolID{}
	}
	pkgPath := namedObj.Pkg().Path()
	receiverName := namedObj.Name()
	return symbolID{
		pkgPath:     pkgPath,
		key:         symbolKey(pkgPath, "method", receiverName, fn.Name()),
		displayName: receiverName + "." + fn.Name(),
	}
}

func simpleSymbolID(pkgPath, kind, name string) symbolID {
	return symbolID{
		pkgPath:     pkgPath,
		key:         symbolKey(pkgPath, kind, "", name),
		displayName: name,
	}
}

type symbolID struct {
	pkgPath     string
	key         string
	displayName string
}

func (id symbolID) isValid() bool { return id.key != "" }

func symbolKey(pkgPath, kind, receiver, name string) string {
	return pkgPath + "\x00" + kind + "\x00" + receiver + "\x00" + name
}

// symbolLocation records where a symbol is declared, for positioning diagnostics.
type symbolLocation struct {
	displayName string
	pos         token.Pos
}

// sharedPackageMap maps package path to the usage trackers for its exported symbols.
type sharedPackageMap map[string]map[string]*symbolUsage

func (m sharedPackageMap) buildDiagnostics() map[string]map[string]string {
	diagnostics := make(map[string]map[string]string)
	for _, pkgPath := range slices.Sorted(maps.Keys(m)) {
		symbols := m[pkgPath]
		for _, key := range slices.Sorted(maps.Keys(symbols)) {
			usage := symbols[key]
			if msg := usage.diagnosticMessage(); msg != "" {
				if diagnostics[pkgPath] == nil {
					diagnostics[pkgPath] = make(map[string]string)
				}
				diagnostics[pkgPath][key] = msg
			}
		}
	}
	return diagnostics
}

// typeExposureMap maps package path → symbol key → shared type keys the symbol exposes.
// A symbol "exposes" a shared type if its signature or underlying type references it.
type typeExposureMap map[string]map[string][]string

// symbolUsageMap is a flat lookup from symbol key to its usage tracker.
type symbolUsageMap map[string]*symbolUsage

// symbolUsage tracks how many distinct consumers use a shared symbol.
type symbolUsage struct {
	displayName         string
	consumers           map[string]struct{}
	hasInternalConsumer bool
}

func (u *symbolUsage) diagnosticMessage() string {
	consumerCount := len(u.consumers)
	if u.hasInternalConsumer {
		consumerCount++
	}
	if consumerCount >= 2 {
		return ""
	}
	return u.displayName + " " + u.summary() + " -> Must be used by 2+ entities"
}

func (u *symbolUsage) summary() string {
	switch {
	case len(u.consumers) == 0 && !u.hasInternalConsumer:
		return "not used by any entity"
	case len(u.consumers) == 0:
		return "only used by internal declaration in shared package"
	default:
		return "only used by: " + slices.Sorted(maps.Keys(u.consumers))[0]
	}
}
