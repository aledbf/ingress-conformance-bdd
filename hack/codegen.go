package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"html/template"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/aledbf/ingress-conformance-bdd/test/utils"
	"github.com/cucumber/gherkin-go/v11"
	"github.com/cucumber/messages-go/v10"
	"github.com/iancoleman/orderedmap"
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
	flag.StringVar(&templatePath, "template", "hack/codegen.template", "template file to generate go source file")

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
	codeTmpl, err := template.New("template").Funcs(templateHelperFuncs).Parse(templatePath)
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
	feature, err := parseFeature(path)
	if err != nil {
		return fmt.Errorf("parsing feature file: %w", err)
	}

	// 6. generate package name to use
	packageName := generatePackage(path)
	// 7. check if go source file exists
	goFile := filepath.Join(conformance, packageName, "feature.go")
	isGoFileOk := utils.Exists(goFile)

	// TODO: replace map
	data := map[string]interface{}{
		"package":     packageName,
		"featureFile": path,
		"features":    feature,
	}

	// 8. Extract functions from go source code
	if isGoFileOk {
		goFunctions, err := extractFuncs(goFile)
		if err != nil {
			return fmt.Errorf("extracting go functions: %w", err)
		}

		data["goFile"] = goFile
		data["goDefinitions"] = goFunctions
	}

	prettyJSON, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		return err
	}

	fmt.Printf("%s\n", string(prettyJSON))

	// 9. check if feature file is in sync with go code
	// 10. if update is set
	// 10.1 check signatures are ok
	// 10.2 add missing methods
	if update {

	}

	return nil
}

type codeTemplate struct {
	Package   string
	Functions []Function
}

var templateHelperFuncs = template.FuncMap{
	"backticked": func(s string) string {
		return "`" + s + "`"
	},
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
