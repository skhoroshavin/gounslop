package boundarycontrol

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strings"
	"sync"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
)

func reportFalseSharingDiagnostic(pass *analysis.Pass, moduleCtx moduleContext, cfg compiledConfig) error {
	if !cfg.hasSharedSelectors {
		return nil
	}

	diagnostics, err := loadFalseSharingDiagnostics(moduleCtx, cfg)
	if err != nil {
		return err
	}

	packageDiagnostics, ok := diagnostics[pass.Pkg.Path()]
	if !ok || len(packageDiagnostics) == 0 {
		return nil
	}

	definitions := collectPackageSymbolDefinitions(pass.Files, pass.TypesInfo, pass.Pkg)
	for _, key := range sortedMapKeys(packageDiagnostics) {
		if def, ok := definitions[key]; ok {
			pass.Reportf(def.pos, "%s", packageDiagnostics[key])
		}
	}

	return nil
}

func loadFalseSharingDiagnostics(moduleCtx moduleContext, cfg compiledConfig) (map[string]map[string]string, error) {
	cacheKey := falseSharingCacheKey{
		moduleDir: moduleCtx.dir,
		configKey: cfg.cacheKey,
	}

	entryValue, _ := falseSharingCache.LoadOrStore(cacheKey, &falseSharingCacheEntry{})
	entry := entryValue.(*falseSharingCacheEntry)
	entry.once.Do(func() {
		entry.diagnostics, entry.err = analyzeFalseSharing(moduleCtx, cfg)
	})

	return entry.diagnostics, entry.err
}

var falseSharingCache sync.Map

func analyzeFalseSharing(moduleCtx moduleContext, cfg compiledConfig) (map[string]map[string]string, error) {
	packagesByPath, err := loadModulePackages(moduleCtx.dir)
	if err != nil {
		return nil, err
	}

	sharedPackages := collectSharedPackages(packagesByPath, moduleCtx, cfg.policies)
	if len(sharedPackages) == 0 {
		return nil, nil
	}

	countSharedSymbolConsumers(packagesByPath, sharedPackages, moduleCtx)
	return falseSharingDiagnostics(sharedPackages), nil
}

type falseSharingCacheKey struct {
	moduleDir string
	configKey string
}

type falseSharingCacheEntry struct {
	once        sync.Once
	diagnostics map[string]map[string]string
	err         error
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
		return nil, fmt.Errorf("boundarycontrol: loading packages for shared-package analysis: %w", err)
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
	moduleCtx moduleContext,
	policies []compiledPolicy,
) map[string]*sharedPackageEntry {
	sharedPackages := make(map[string]*sharedPackageEntry)
	for pkgPath := range packagesByPath {
		relPath, ownership := classifyImportPath(pkgPath, moduleCtx)
		if ownership != importOwnershipCurrentModule {
			continue
		}

		owner, found := resolveOwner(policies, relPath)
		if !found || !owner.shared {
			continue
		}

		symbols := collectSharedPackageSymbols(packagesByPath[pkgPath])
		if len(symbols) == 0 {
			continue
		}

		sharedPackages[pkgPath] = &sharedPackageEntry{
			symbols: symbols,
		}
	}

	return sharedPackages
}

func collectSharedPackageSymbols(pkg *packages.Package) map[string]*sharedSymbolEntry {
	definitions := collectPackageSymbolDefinitions(pkg.Syntax, pkg.TypesInfo, pkg.Types)
	if len(definitions) == 0 {
		return nil
	}

	symbols := make(map[string]*sharedSymbolEntry, len(definitions))
	for key, definition := range definitions {
		symbols[key] = &sharedSymbolEntry{
			displayName:       definition.displayName,
			externalConsumers: make(map[string]struct{}),
		}
	}

	return symbols
}

func countSharedSymbolConsumers(
	packagesByPath map[string]*packages.Package,
	sharedPackages map[string]*sharedPackageEntry,
	moduleCtx moduleContext,
) {
	for pkgPath, pkg := range packagesByPath {
		consumerRelPath, ownership := classifyImportPath(pkgPath, moduleCtx)
		if ownership != importOwnershipCurrentModule || !hasNonTestFiles(pkg.CompiledGoFiles) {
			continue
		}

		consumer := consumerRelPath
		if consumer == "" {
			consumer = "."
		}

		countPackageSymbolConsumers(pkgPath, pkg, sharedPackages, consumer)
	}
}

func countPackageSymbolConsumers(
	packagePath string,
	pkg *packages.Package,
	sharedPackages map[string]*sharedPackageEntry,
	consumer string,
) {
	if pkg == nil || pkg.TypesInfo == nil {
		return
	}

	forEachDeclaration(pkg.Syntax, pkg.TypesInfo, func(node ast.Node, ownedObjects []types.Object) {
		ownerKeys := ownerKeySet(ownedObjects, pkg.Types)
		countSymbolConsumersInNode(node, ownerKeys, packagePath, consumer, pkg.TypesInfo, sharedPackages)
	}, pkg.Types)
}

func countSymbolConsumersInNode(
	node ast.Node,
	ownerKeys map[string]struct{},
	packagePath string,
	consumer string,
	info *types.Info,
	sharedPackages map[string]*sharedPackageEntry,
) {
	walkReferencedObjects(node, info, func(obj types.Object) {
		si := falseSharingSymbolDetails(obj)
		if !si.ok {
			return
		}

		sharedPackage, ok := sharedPackages[si.pkgPath]
		if !ok {
			return
		}

		symbol, ok := sharedPackage.symbols[si.key]
		if !ok {
			return
		}

		if si.pkgPath == packagePath {
			if _, ok := ownerKeys[si.key]; ok {
				return
			}

			symbol.hasInternalConsumer = true
			return
		}

		symbol.externalConsumers[consumer] = struct{}{}
	})
}

func walkReferencedObjects(node ast.Node, info *types.Info, visit func(types.Object)) {
	if node == nil || info == nil {
		return
	}

	stack := make([]ast.Node, 0, 8)
	ast.Inspect(node, func(current ast.Node) bool {
		if current == nil {
			stack = stack[:len(stack)-1]
			return false
		}

		var parent ast.Node
		if len(stack) > 0 {
			parent = stack[len(stack)-1]
		}

		switch typedNode := current.(type) {
		case *ast.SelectorExpr:
			if selection, ok := info.Selections[typedNode]; ok {
				visit(selection.Obj())
			} else if obj := info.Uses[typedNode.Sel]; obj != nil {
				visit(obj)
			}
		case *ast.Ident:
			selector, ok := parent.(*ast.SelectorExpr)
			if ok && selector.Sel == typedNode {
				break
			}

			if obj := info.Uses[typedNode]; obj != nil {
				visit(obj)
			}
		}

		stack = append(stack, current)
		return true
	})
}

func falseSharingDiagnostics(sharedPackages map[string]*sharedPackageEntry) map[string]map[string]string {
	diagnostics := make(map[string]map[string]string)
	for _, pkgPath := range sortedMapKeys(sharedPackages) {
		entry := sharedPackages[pkgPath]
		for _, key := range sortedMapKeys(entry.symbols) {
			symbol := entry.symbols[key]
			consumerCount := len(symbol.externalConsumers)
			if symbol.hasInternalConsumer {
				consumerCount++
			}

			if consumerCount >= 2 {
				continue
			}

			var reason string
			switch {
			case len(symbol.externalConsumers) == 0 && !symbol.hasInternalConsumer:
				reason = "not used by any entity"
			case len(symbol.externalConsumers) == 0:
				reason = "only used by internal declaration in shared package"
			default:
				consumers := sortedMapKeys(symbol.externalConsumers)
				reason = "only used by: " + consumers[0]
			}

			if diagnostics[pkgPath] == nil {
				diagnostics[pkgPath] = make(map[string]string)
			}
			diagnostics[pkgPath][key] = symbol.displayName + " " + reason + " -> Must be used by 2+ entities"
		}
	}

	return diagnostics
}

type sharedPackageEntry struct {
	symbols map[string]*sharedSymbolEntry
}

type sharedSymbolEntry struct {
	displayName         string
	externalConsumers   map[string]struct{}
	hasInternalConsumer bool
}

func collectPackageSymbolDefinitions(
	files []*ast.File,
	info *types.Info,
	pkg *types.Package,
) map[string]symbolDefinition {
	definitions := make(map[string]symbolDefinition)
	if info == nil || pkg == nil {
		return definitions
	}

	forEachDeclaration(files, info, func(_ ast.Node, ownedObjects []types.Object) {
		for _, obj := range ownedObjects {
			si := falseSharingSymbolDetails(obj)
			definitions[si.key] = symbolDefinition{
				displayName: si.displayName,
				pos:         obj.Pos(),
			}
		}
	}, pkg)

	return definitions
}

func forEachDeclaration(files []*ast.File, info *types.Info, visit func(ast.Node, []types.Object), pkg *types.Package) {
	if info == nil || pkg == nil {
		return
	}

	for _, file := range files {
		for _, decl := range file.Decls {
			switch typedDecl := decl.(type) {
			case *ast.FuncDecl:
				obj := info.Defs[typedDecl.Name]
				var owned []types.Object
				if isOwnedInPackage(obj, pkg) {
					owned = []types.Object{obj}
				}
				visit(typedDecl, owned)
			case *ast.GenDecl:
				for _, spec := range typedDecl.Specs {
					var owned []types.Object
					switch typedSpec := spec.(type) {
					case *ast.TypeSpec:
						obj := info.Defs[typedSpec.Name]
						if isOwnedInPackage(obj, pkg) {
							owned = append(owned, obj)
						}
					case *ast.ValueSpec:
						for _, name := range typedSpec.Names {
							obj := info.Defs[name]
							if isOwnedInPackage(obj, pkg) {
								owned = append(owned, obj)
							}
						}
					}
					visit(spec, owned)
				}
			}
		}
	}
}

func isOwnedInPackage(obj types.Object, pkg *types.Package) bool {
	if obj == nil {
		return false
	}
	si := falseSharingSymbolDetails(obj)
	return si.ok && si.pkgPath == pkg.Path()
}

func ownerKeySet(objects []types.Object, pkg *types.Package) map[string]struct{} {
	keys := make(map[string]struct{}, len(objects))
	for _, obj := range objects {
		si := falseSharingSymbolDetails(obj)
		if si.ok && si.pkgPath == pkg.Path() {
			keys[si.key] = struct{}{}
		}
	}
	return keys
}

func falseSharingSymbolDetails(obj types.Object) symbolInfo {
	if obj == nil || obj.Pkg() == nil || !obj.Exported() {
		return symbolInfo{}
	}

	switch typedObject := obj.(type) {
	case *types.Func:
		return funcSymbolDetails(typedObject)
	case *types.TypeName:
		return typeNameSymbolDetails(typedObject)
	case *types.Var:
		return varSymbolDetails(typedObject)
	case *types.Const:
		return constSymbolDetails(typedObject)
	default:
		return symbolInfo{}
	}
}

func funcSymbolDetails(fn *types.Func) symbolInfo {
	signature, ok := fn.Type().(*types.Signature)
	if !ok {
		return symbolInfo{}
	}

	if signature.Recv() == nil {
		pkgPath := fn.Pkg().Path()
		return symbolInfo{
			pkgPath:     pkgPath,
			key:         symbolKey(pkgPath, "func", "", fn.Name()),
			displayName: fn.Name(),
			ok:          true,
		}
	}

	return methodSymbolDetails(fn, signature)
}

func methodSymbolDetails(fn *types.Func, sig *types.Signature) symbolInfo {
	receiver := sig.Recv().Type()
	if pointer, ok := receiver.(*types.Pointer); ok {
		receiver = pointer.Elem()
	}

	named, ok := receiver.(*types.Named)
	if !ok {
		return symbolInfo{}
	}

	namedObj := named.Obj()
	if namedObj == nil || namedObj.Pkg() == nil {
		return symbolInfo{}
	}

	pkgPath := namedObj.Pkg().Path()
	receiverName := namedObj.Name()
	return symbolInfo{
		pkgPath:     pkgPath,
		key:         symbolKey(pkgPath, "method", receiverName, fn.Name()),
		displayName: receiverName + "." + fn.Name(),
		ok:          true,
	}
}

func typeNameSymbolDetails(tn *types.TypeName) symbolInfo {
	pkgPath := tn.Pkg().Path()
	return symbolInfo{
		pkgPath:     pkgPath,
		key:         symbolKey(pkgPath, "type", "", tn.Name()),
		displayName: tn.Name(),
		ok:          true,
	}
}

func varSymbolDetails(v *types.Var) symbolInfo {
	if v.IsField() {
		return symbolInfo{}
	}

	pkgPath := v.Pkg().Path()
	return symbolInfo{
		pkgPath:     pkgPath,
		key:         symbolKey(pkgPath, "var", "", v.Name()),
		displayName: v.Name(),
		ok:          true,
	}
}

func constSymbolDetails(c *types.Const) symbolInfo {
	pkgPath := c.Pkg().Path()
	return symbolInfo{
		pkgPath:     pkgPath,
		key:         symbolKey(pkgPath, "const", "", c.Name()),
		displayName: c.Name(),
		ok:          true,
	}
}

func symbolKey(pkgPath, kind, receiver, name string) string {
	return strings.Join([]string{pkgPath, kind, receiver, name}, "\x00")
}

type symbolInfo struct {
	pkgPath     string
	key         string
	displayName string
	ok          bool
}

func sortedMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

type symbolDefinition struct {
	displayName string
	pos         token.Pos
}

func hasNonTestFiles(filePaths []string) bool {
	for _, filePath := range filePaths {
		if strings.HasSuffix(filePath, "_test.go") {
			continue
		}

		return true
	}

	return false
}
