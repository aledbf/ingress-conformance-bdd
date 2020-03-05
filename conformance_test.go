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
	"os"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	"github.com/aledbf/ingress-conformance-bdd/test/conformance/defaultbackend"
	"github.com/aledbf/ingress-conformance-bdd/test/generated"
	"github.com/aledbf/ingress-conformance-bdd/test/utils"
)

func TestMain(m *testing.M) {
	var (
		godogFormat        string
		godogTags          string
		godogStopOnFailure bool
		godogNoColors      bool
		godogPaths         string
		godogOutput        string
	)

	flag.StringVar(&godogFormat, "format", "pretty", "Sets godog format to use")
	flag.StringVar(&godogTags, "tags", "", "Tags for conformance test")
	flag.BoolVar(&godogStopOnFailure, "stop-on-failure ", false, "Stop when failure is found")
	flag.BoolVar(&godogNoColors, "no-colors", false, "Disable colors in godog output")
	flag.StringVar(&godogPaths, "paths", "./features", "")
	flag.StringVar(&godogOutput, "output-file", "", "Output file for test")
	flag.Parse()

	kubeClient, err := setupSuite()
	if err != nil {
		klog.Fatal(err)
	}

	output := os.Stdout

	if godogOutput != "" {
		file, err := os.Create(godogOutput)
		if err != nil {
			klog.Fatal(err)
		}

		defer file.Close()
		output = file
	}

	utils.AddFileSource(utils.BindataFileSource{
		Asset:      generated.Asset,
		AssetNames: generated.AssetNames,
	})

	exitCode := godog.RunWithOptions("conformance", func(s *godog.Suite) {
		defaultbackend.FeatureContext(s, kubeClient)
		//withhost.FeatureContext(s, kubeClient)
		//withouthost.FeatureContext(s, kubeClient)
		//tls.FeatureContext(s, kubeClient)
	}, godog.Options{
		Format:        godogFormat,
		Paths:         strings.Split(godogPaths, ","),
		Tags:          godogTags,
		StopOnFailure: godogStopOnFailure,
		NoColors:      godogNoColors,
		Output:        output,
	})

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
		klog.Infof("kube-apiserver version: %s", serverVersion.GitVersion)
	}

	return c, nil
}
