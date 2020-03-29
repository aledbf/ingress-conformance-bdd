/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/cucumber/godog"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	"github.com/aledbf/ingress-conformance-bdd/test/conformance/defaultbackend"
	"github.com/aledbf/ingress-conformance-bdd/test/conformance/withouthost"
	"github.com/aledbf/ingress-conformance-bdd/test/utils"
)

var (
	exitCode int
)

var (
	godogFormat        string
	godogTags          string
	godogStopOnFailure bool
	godogNoColors      bool
	godogOutput        string

	manifests string
)

func TestMain(m *testing.M) {
	// register flags from klog
	klog.InitFlags(nil)

	flag.StringVar(&godogFormat, "format", "pretty", "Sets godog format to use")
	flag.StringVar(&godogTags, "tags", "", "Tags for conformance test")
	flag.BoolVar(&godogStopOnFailure, "stop-on-failure ", false, "Stop when failure is found")
	flag.BoolVar(&godogNoColors, "no-colors", false, "Disable colors in godog output")
	flag.StringVar(&manifests, "manifests", "./manifests",
		"Directory where manifests for test applications or scenerarios are located")
	flag.StringVar(&godogOutput, "output-file", "", "Output file for test")
	flag.StringVar(&utils.IngressClassValue, "ingress-class", "conformance",
		"Sets the value of the annotation kubernetes.io/ingress.class in Ingress definitions")

	flag.Parse()

	manifestsPath, err := filepath.Abs(manifests)
	if err != nil {
		log.Fatal(err)
	}

	if !utils.IsDir(manifestsPath) {
		log.Fatalf("The specified value in the flag --manifests-directory (%v) is not a directory", manifests)
	}

	utils.ManifestPath = manifestsPath

	utils.KubeClient, err = setupSuite()
	if err != nil {
		log.Fatal(err)
	}

	if err := utils.CleanupNamespaces(utils.KubeClient); err != nil {
		log.Fatalf("error deleting temporal namespaces: %v", err)
	}

	if code := m.Run(); code > exitCode {
		exitCode = code
	}

	os.Exit(exitCode)
}

func setupSuite() (*clientset.Clientset, error) {
	c, err := utils.LoadClientset()
	if err != nil {
		return nil, fmt.Errorf("error loading client: %v", err)
	}

	dc := c.DiscoveryClient

	serverVersion, serverErr := dc.ServerVersion()
	if serverErr != nil {
		return nil, fmt.Errorf("unexpected server error retrieving version: %v", serverErr)
	}

	if serverVersion != nil {
		log.Printf("kube-apiserver version: %s", serverVersion.GitVersion)
	}

	return c, nil
}

var (
	features = map[string]func(*godog.Suite){
		"features/default_backend.feature": defaultbackend.FeatureContext,
		"features/without_host.feature":    withouthost.FeatureContext,
	}
)

func TestSuite(t *testing.T) {
	for feature, featureContext := range features {
		//TODO: refactor to remove the defer
		func() {
			output := os.Stdout
			if godogFormat == "cucumber" {
				rf := fmt.Sprintf("%v-report.json", feature)
				file, err := os.Create(rf)
				if err != nil {
					t.Fatalf("Error creating report file %v: %v", rf, err)
				}

				output = file
				defer func() {
					_ = file.Sync()
					_ = file.Close()
				}()
			}

			exitCode += godog.RunWithOptions("conformance", func(s *godog.Suite) {
				featureContext(s)
			}, godog.Options{
				Format:        godogFormat,
				Paths:         []string{feature},
				Tags:          godogTags,
				StopOnFailure: godogStopOnFailure,
				NoColors:      godogNoColors,
				Output:        output,
				Concurrency:   1, // do not run tests concurrently
			})

			if exitCode != 0 {
				t.Fatalf("Error encountered running the test suite")
			}
		}()
	}
}
