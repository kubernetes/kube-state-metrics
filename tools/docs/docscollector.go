/*
Copyright 2021 The Kubernetes Authors All rights reserved.
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

package docs

import (
	"log"
	"os"
	"text/template"

	"k8s.io/kube-state-metrics/v2/internal/store"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

//docsTemplateText is template text for Docs
const docsTemplateText = `| Metric name | Metric Type | Description | Unit (where applicable) | Labels/tags | Status |
| ----------- | ----------- | ----------- | ----------------------- | ----------- | ------ |
{{range.}}| {{.Name}} | {{.Type}} | {{.Help}} |
{{end}}
`

//Create creates md files automatically
func Create(file string) {
	var docsMetaData []generator.FamilyGenerator
	docsMetaData = store.GetFamily()

	mdfile, err := os.Create("./docs/dynamic-docs/" + file + ".md")
	if err != nil {
		log.Fatalf("Error Creating Markdown Files : %v", err)
	}
	t := template.Must(template.New("tmpl").Parse(docsTemplateText))
	t.Execute(mdfile, docsMetaData)

}
