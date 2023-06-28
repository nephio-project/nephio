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

const configurationTemplateSource = `mcc: '208'          # Mobile Country Code value
mnc: '93'           # Mobile Network Code value (2 or 3 digits)
nci: '0x000000010'  # NR Cell Identity (36-bit)
idLength: 32        # NR gNB ID length in bits [22...32]
tac: 1              # Tracking Area Code
# List of supported S-NSSAIs by this gNB
slices:
  - sst: 0x1
	  sd: 0x010203
# Indicates whether or not SCTP stream number errors should be ignored.
ignoreStreamIds: true

linkIp: 0.0.0.0   # gNB's local IP address for Radio Link Simulation (Usually same with local IP)
# gNB's local IP address for N2 Interface (Usually same with local IP)
ngapIp: {{ .N2 }}
gtpIp: {{ .N3 }}    # gNB's local IP address for N3 Interface (Usually same with local IP)

# List of AMF address information
amfConfigs:
{{- range $amf := .AMF }}
  - address: {{ $amf }}
	  port: 38412
{{- end }}
`

var (
	configurationTemplate = template.Must(template.New("UERANSIMConfiguration").Parse(configurationTemplateSource))
)

type configurationTemplateValues struct {
	N2  string
	N3  string
	AMF []string
}

func renderConfigurationTemplate(values configurationTemplateValues) (string, error) {
	var buffer bytes.Buffer
	if err := configurationTemplate.Execute(&buffer, values); err == nil {
		return buffer.String(), nil
	} else {
		return "", err
	}
}
