/*
Copyright 2026 The Nephio Authors.

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

package cel

import (
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/kptdev/krm-functions-sdk/go/fn"
	kptfilelibv1 "github.com/nephio-project/nephio/krm-functions/lib/kptfile/v1"
	"sigs.k8s.io/yaml"
)

func EvaluateCondition(expression string, rl *fn.ResourceList) (bool, error) {
	if expression == "" {
		return true, nil
	}

	env, err := cel.NewEnv(
		cel.Variable("items", cel.ListType(cel.DynType)),
	)
	if err != nil {
		return false, fmt.Errorf("failed to create CEL env: %w", err)
	}

	ast, issues := env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return false, fmt.Errorf("failed to compile CEL expression: %w", issues.Err())
	}

	prg, err := env.Program(ast)
	if err != nil {
		return false, fmt.Errorf("failed to create CEL program: %w", err)
	}

	// Convert each item to a map[string]interface{} for CEL evaluation
	resources := make([]interface{}, 0, len(rl.Items))
	for _, item := range rl.Items {
		var m map[string]interface{}
		if err := yaml.Unmarshal([]byte(item.String()), &m); err != nil {
			return false, fmt.Errorf("failed to unmarshal resource for CEL: %w", err)
		}
		resources = append(resources, m)
	}

	vars := map[string]interface{}{
		"items": resources,
	}

	out, _, err := prg.Eval(vars)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate CEL expression: %w", err)
	}

	boolVal, ok := out.Value().(bool)
	if !ok {
		return false, fmt.Errorf("CEL expression did not evaluate to a boolean")
	}

	return boolVal, nil
}

func EvaluateConditionForImage(rl *fn.ResourceList, image string) (bool, error) {
	kptfileObject := rl.Items.GetRootKptfile()
	if kptfileObject == nil {
		return true, nil // No Kptfile, no conditions to skip
	}

	kptf := &kptfilelibv1.KptFile{Kptfile: kptfileObject}
	condition := kptf.GetFunctionCondition(image)
	if condition == "" {
		return true, nil
	}

	return EvaluateCondition(condition, rl)
}
