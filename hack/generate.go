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
	"log"
	"os"
	"path"
	"regexp"
	"strings"
	"unicode"

	"github.com/aledbf/ingress-conformance-bdd/test/utils"
	"github.com/cucumber/gherkin-go/v11"
	"github.com/cucumber/messages-go/v10"
)

// Function holds the definition of a function in a go file or godog step
type Function struct {
	// Name
	Name string
	// Expr Regexp to use in godog Step definition
	Expr string
	// Args function arguments using a k,v pair.
	// k = name of the argument
	// v = type of the argument
	Args map[string]string
}

type generateOptions struct {
	Update          bool
	Features        []string
	ConformancePath string
}

func defaultOptions() *generateOptions {
	return &generateOptions{
		Update:          false,
		ConformancePath: "test/conformance",
	}
}

func main() {
	o := defaultOptions()
	flag.BoolVar(&o.Update, "update", o.Update, "update files in place in case of missing steps or method definitions")
	flag.StringVar(&o.ConformancePath, "conformance-path", o.ConformancePath, "path to conformance test package location")

	flag.Parse()

	o.Features = flag.CommandLine.Args()
	if len(o.Features) == 0 {
		fmt.Println("Usage: generate [-update=false] [-conformance-path=test/conformance] [features]")
		fmt.Println()
		fmt.Println("Example: generate features/default_backend.feature")
		flag.CommandLine.Usage()
		os.Exit(1)
	}

	// TODO: if features is a directory, iterate and search for files with extension .feature

	for _, featurePath := range o.Features {
		feature, err := parseFeature(featurePath)
		if err != nil {
			log.Fatal(err)
		}

		// TODO: search for go file for the feature
		// create if does not exists?
		functions, err := extractFuncs(path.Join(o.ConformancePath, "/defaultbackend/feature.go"))
		if err != nil {
			log.Fatal(err)
		}

		data := map[string]interface{}{
			"file":          featurePath,
			"feature":       feature,
			"goDefinitions": functions,
		}

		prettyJSON, err := json.MarshalIndent(data, "", " ")
		if err != nil {
			log.Fatal("Failed to generate json", err)
		}

		fmt.Printf("%s\n", string(prettyJSON))
	}
}

func parseFeature(path string) ([]Function, error) {
	if exists := utils.Exists(path); !exists {
		return nil, fmt.Errorf("file %v does not exists", path)
	}

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

// some snippet formatting regexps
var snippetExprCleanup = regexp.MustCompile("([\\/\\[\\]\\(\\)\\\\^\\$\\.\\|\\?\\*\\+\\'])")
var snippetExprQuoted = regexp.MustCompile("(\\W|^)\"(?:[^\"]*)\"(\\W|$)")
var snippetMethodName = regexp.MustCompile("[^a-zA-Z\\_\\ ]")
var snippetNumbers = regexp.MustCompile("(\\d+)")

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

// extractFuncs
func extractFuncs(path string) ([]Function, error) {
	if !strings.HasSuffix(path, ".go") {
		return nil, fmt.Errorf("only files with go extensions are valid")
	}

	funcs := []Function{}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var printErr error
	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}

		args := map[string]string{}
		for _, p := range fn.Type.Params.List {
			var typeNameBuf bytes.Buffer
			err := printer.Fprint(&typeNameBuf, fset, p.Type)
			if err != nil {
				printErr = err
				return false
			}

			args[p.Names[0].String()] = typeNameBuf.String()
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

const (
	numberGroup = "(\\d+)"
	stringGroup = "\"([^\"]*)\""
)

// from https://github.com/cucumber/godog/blob/4da503aab2d0b71d380fbe8c48a6af9f729b6f5a/undefined_snippets_gen.go#L41
func parseStepArgs(exp string, argument *messages.PickleStepArgument) map[string]string {
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

	stepArgs := map[string]string{}
	for i, v := range args {
		k := fmt.Sprintf("arg%d, ", i+1)
		stepArgs[k] = v
	}

	return stepArgs
}
