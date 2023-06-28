/*
Copyright 2023 Nephio.

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

package fn

import (
	"bytes"
	"text/template"
)

const nadTemplateSource = `[
 {
{{- $length := len .}}
{{- range $index, $nad := . }}
  "name": "{{ $nad.Name }}",
  "interface": "{{ $nad.Interface }}",
  "ips": ["{{ $nad.IPs }}"],
  "gateways": ["{{ $nad.Gateways }}"]
{{- if (isNotLast $index $length) }}
 },
 {
{{- end }}
{{- end }}
 }
]
`

var (
	nadTemplate = template.Must(template.New("UERANSIMNad").Funcs(fns).Parse(nadTemplateSource))
)

type nadTemplateValues struct {
	Name      string
	Interface string
	IPs       string
	Gateways  string
}

var fns = template.FuncMap{
	"isNotLast": func(index int, len int) bool {
		return index+1 != len
	},
}

func renderNadTemplate(values []nadTemplateValues) (string, error) {
	var buffer bytes.Buffer
	if err := nadTemplate.Execute(&buffer, values); err == nil {
		return buffer.String(), nil
	} else {
		return "", err
	}
}
