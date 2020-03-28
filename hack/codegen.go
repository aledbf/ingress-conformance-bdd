package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"unicode"

	"github.com/aledbf/ingress-conformance-bdd/test/utils"
	"github.com/cucumber/gherkin-go/v11"
	"github.com/cucumber/messages-go/v10"
	"github.com/iancoleman/orderedmap"
	"k8s.io/apimachinery/pkg/util/sets"
)

// Function holds the definition of a function in a go file or godog step
type Function struct {
	// Name
	Name string
	// Expr Regexp to use in godog Step definition
	Expr string
	// Args function arguments
	// k = name of the argument
	// v = type of the argument
	Args *orderedmap.OrderedMap
}

func main() {
	var (
		verbose         bool
		update          bool
		features        []string
		conformancePath string
		templatePath    string
	)

	flag.BoolVar(&verbose, "verbose", false, "enable verbose output")
	flag.BoolVar(&update, "update", false, "update files in place in case of missing steps or method definitions")
	flag.StringVar(&conformancePath, "conformance-path", "test/conformance", "path to conformance test package location")
	flag.StringVar(&templatePath, "template", "hack/codegen.tmpl", "template file to generate go source file")

	flag.Parse()

	// 1. verify flags
	features = flag.CommandLine.Args()
	if len(features) == 0 {
		fmt.Println("Usage: codegen [-update=false] [-verbose=false] [-conformance-path=test/conformance] [-template=hack/generate.tmpl] [features]")
		fmt.Println()
		fmt.Println("Example: codegen features/default_backend.feature")
		flag.CommandLine.Usage()
		os.Exit(1)
	}

	// 2. verify template file
	data, err := ioutil.ReadFile(templatePath)
	if err != nil {
		log.Fatalf("unexpected error reading template %v: %v", templatePath, err)
	}

	codeTmpl, err := template.New("template").Funcs(templateHelperFuncs).Parse(string(data))
	if err != nil {
		log.Fatalf("Unexpected error parsing template: %v", err)
	}

	// 3. if features is a directory, iterate and search for files with extension .feature

	// 4. iterate feature files
	for _, path := range features {
		err := processFeature(path, conformancePath, update, codeTmpl)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func processFeature(path, conformance string, update bool, template *template.Template) error {
	// 5. parse feature file
	featureSteps, err := parseFeature(path)
	if err != nil {
		return fmt.Errorf("parsing feature file: %w", err)
	}

	// 6. generate package name to use
	packageName := generatePackage(path)
	// 7. check if go source file exists
	goFile := filepath.Join(conformance, packageName, "feature.go")
	isGoFileOk := utils.Exists(goFile)

	// TODO: replace map
	mapping := &Mapping{
		Package:      packageName,
		FeatureFile:  path,
		Features:     featureSteps,
		NewFunctions: featureSteps,
		GoFile:       goFile,
	}

	// 8. Extract functions from go source code
	if isGoFileOk {
		goFunctions, err := extractFuncs(goFile)
		if err != nil {
			return fmt.Errorf("extracting go functions: %w", err)
		}

		mapping.GoDefinitions = goFunctions
	}

	// 9. check if feature file is in sync with go code
	isInSync := false

	signatureChanges := []SignatureChange{}

	if isGoFileOk {
		inFeatures := sets.NewString()
		inGo := sets.NewString()

		for _, feature := range mapping.Features {
			inFeatures.Insert(feature.Name)
		}

		for _, gofunc := range mapping.GoDefinitions {
			inGo.Insert(gofunc.Name)
		}

		if newFunctions := inFeatures.Difference(inGo); newFunctions.Len() > 0 {
			log.Printf("Feature file %v contains %v new functions", mapping.FeatureFile, newFunctions.Len())

			isInSync = false

			var funcs []Function
			for _, f := range newFunctions.List() {
				for _, feature := range mapping.Features {
					if feature.Name == f {
						funcs = append(funcs, feature)
						break
					}
				}
			}

			mapping.NewFunctions = funcs
		}

		continueLoop := true
		for _, feature := range mapping.Features {
			for _, gofunc := range mapping.GoDefinitions {
				if feature.Name != gofunc.Name {
					continue
				}

				if !reflect.DeepEqual(feature.Args, gofunc.Args) {
					signatureChanges = append(signatureChanges, SignatureChange{
						Function: gofunc.Name,
						Have:     argsFromMap(gofunc.Args),
						Want:     argsFromMap(feature.Args),
					})

					continueLoop = false
					break
				}
			}

			if !continueLoop {
				break
			}
		}
	}

	// 10. check signatures are ok
	if len(signatureChanges) != 0 {
		var argBuf bytes.Buffer
		for _, sc := range signatureChanges {
			argBuf.WriteString(fmt.Sprintf(`
function %v
	have %v
	want %v
`, sc.Function, sc.Have, sc.Want))
		}

		return fmt.Errorf("source file %v has a different signature/s:\n %v", mapping.GoFile, argBuf.String())
	}

	// 11. if in sync, nothing to do
	if isInSync {
		return nil
	}

	if !isGoFileOk {
		log.Printf("Generating new go file %v...", mapping.GoFile)
		buf := bytes.NewBuffer(make([]byte, 0))
		err := template.Execute(buf, mapping)
		if err != nil {
			return err
		}

		log.Printf("#%v#\n", buf.String())
		// 10. if update is set
		if update {
			err := ioutil.WriteFile(mapping.GoFile, buf.Bytes(), 0644)
			if err != nil {
				return err
			}
		}

		return nil
	}

	log.Printf("Updating go file %v...", mapping.GoFile)
	log.Printf("%v\n", mapping.NewFunctions)

	// 10. if update is set
	if update {
	}

	return nil
}

type Mapping struct {
	Package string

	FeatureFile string
	Features    []Function

	GoFile        string
	GoDefinitions []Function

	NewFunctions []Function
}

// SignatureChange holds information about the definition of a go function
type SignatureChange struct {
	Function string
	Have     string
	Want     string
}

var templateHelperFuncs = template.FuncMap{
	"backticked": func(s string) string {
		return "`" + s + "`"
	},
	"unescape": func(s string) template.HTML {
		return template.HTML(s)
	},
	"argsFromMap": argsFromMap,
}

// parseFeature parses a godog feature file returning the unique
// steps definitions
func parseFeature(path string) ([]Function, error) {
	data, err := utils.Read(path)
	if err != nil {
		return nil, err
	}

	gd, err := gherkin.ParseGherkinDocument(bytes.NewReader(data), (&messages.Incrementing{}).NewId)
	if err != nil {
		return nil, err
	}

	scenarios := gherkin.Pickles(*gd, path, (&messages.Incrementing{}).NewId)

	def := []Function{}
	for _, s := range scenarios {
		def = parseSteps(s.Steps, def)
	}

	return def, nil
}

// extractFuncs reads a file containing go source code and returns
// the functions defined in the file.
func extractFuncs(filePath string) ([]Function, error) {
	if !strings.HasSuffix(filePath, ".go") {
		return nil, fmt.Errorf("only files with go extension are valid")
	}

	funcs := []Function{}

	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var printErr error
	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}

		args := orderedmap.New()
		for _, p := range fn.Type.Params.List {
			var typeNameBuf bytes.Buffer

			err := printer.Fprint(&typeNameBuf, fset, p.Type)
			if err != nil {
				printErr = err
				return false
			}

			args.Set(p.Names[0].String(), typeNameBuf.String())
		}

		// Go functions do not have an expression
		funcs = append(funcs, Function{Name: fn.Name.Name, Args: args})

		return true
	})

	if printErr != nil {
		return nil, printErr
	}

	return funcs, nil
}

// generatePackage returns the name of the
// package to use using the feature filename
func generatePackage(filePath string) string {
	base := path.Base(filePath)
	base = strings.ToLower(base)
	base = strings.ReplaceAll(base, "_", "")
	base = strings.ReplaceAll(base, ".feature", "")

	return base
}

func argsFromMap(args *orderedmap.OrderedMap) string {
	s := "("
	for _, k := range args.Keys() {
		v, ok := args.Get(k)
		if !ok {
			continue
		}

		s = s + fmt.Sprintf("%v, ", v)
	}

	if len(args.Keys()) > 0 {
		s = s[0 : len(s)-2]
	}

	s = s + ")"

	return s
}

//
// Code below this comment comes from github.com/cucumber/godog
// (code defined in private methods)

const (
	numberGroup = "(\\d+)"
	stringGroup = "\"([^\"]*)\""
)

// parseStepArgs extracts arguments from an expression defined in a step RegExp.
// This code was extracted from
// https://github.com/cucumber/godog/blob/4da503aab2d0b71d380fbe8c48a6af9f729b6f5a/undefined_snippets_gen.go#L41
func parseStepArgs(exp string, argument *messages.PickleStepArgument) *orderedmap.OrderedMap {
	var (
		args      []string
		pos       int
		breakLoop bool
	)

	for !breakLoop {
		part := exp[pos:]
		ipos := strings.Index(part, numberGroup)
		spos := strings.Index(part, stringGroup)

		switch {
		case spos == -1 && ipos == -1:
			breakLoop = true
		case spos == -1:
			pos += ipos + len(numberGroup)
			args = append(args, "int")
		case ipos == -1:
			pos += spos + len(stringGroup)
			args = append(args, "string")
		case ipos < spos:
			pos += ipos + len(numberGroup)
			args = append(args, "int")
		case spos < ipos:
			pos += spos + len(stringGroup)
			args = append(args, "string")
		}
	}

	if argument != nil {
		if argument.GetDocString() != nil {
			args = append(args, "*messages.PickleStepArgument_PickleDocString")
		}

		if argument.GetDataTable() != nil {
			args = append(args, "*messages.PickleStepArgument_PickleTable")
		}
	}

	stepArgs := orderedmap.New()
	for i, v := range args {
		k := fmt.Sprintf("arg%d, ", i+1)
		stepArgs.Set(k, v)
	}

	return stepArgs
}

// some snippet formatting regexps
var snippetExprCleanup = regexp.MustCompile("([\\/\\[\\]\\(\\)\\\\^\\$\\.\\|\\?\\*\\+\\'])")
var snippetExprQuoted = regexp.MustCompile("(\\W|^)\"(?:[^\"]*)\"(\\W|$)")
var snippetMethodName = regexp.MustCompile("[^a-zA-Z\\_\\ ]")
var snippetNumbers = regexp.MustCompile("(\\d+)")

// parseSteps converts a string step definition in a different one valid as a regular
// expression that can be used in a go Step definition. This original code is located in
// https://github.com/cucumber/godog/blob/4da503aab2d0b71d380fbe8c48a6af9f729b6f5a/fmt.go#L457
func parseSteps(steps []*messages.Pickle_PickleStep, funcDefs []Function) []Function {
	var index int

	for _, step := range steps {
		text := step.Text

		expr := snippetExprCleanup.ReplaceAllString(text, "\\$1")
		expr = snippetNumbers.ReplaceAllString(expr, "(\\d+)")
		expr = snippetExprQuoted.ReplaceAllString(expr, "$1\"([^\"]*)\"$2")
		expr = "^" + strings.TrimSpace(expr) + "$"

		name := snippetNumbers.ReplaceAllString(text, " ")
		name = snippetExprQuoted.ReplaceAllString(name, " ")
		name = strings.TrimSpace(snippetMethodName.ReplaceAllString(name, ""))

		var words []string
		for i, w := range strings.Split(name, " ") {
			switch {
			case i != 0:
				w = strings.Title(w)
			case len(w) > 0:
				w = string(unicode.ToLower(rune(w[0]))) + w[1:]
			}

			words = append(words, w)
		}

		name = strings.Join(words, "")
		if len(name) == 0 {
			index++
			name = fmt.Sprintf("StepDefinitioninition%d", index)
		}

		var found bool
		for _, f := range funcDefs {
			if f.Expr == expr {
				found = true
				break
			}
		}

		if !found {
			args := parseStepArgs(expr, step.Argument)
			funcDefs = append(funcDefs, Function{Name: name, Expr: expr, Args: args})
		}
	}

	return funcDefs
}
