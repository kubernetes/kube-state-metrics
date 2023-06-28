/*
Copyright 2023 The Kubernetes Authors All rights reserved.
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

package discovery

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GroupVersionKindPlural embeds schema.GroupVersionKind, in addition to the plural representation of the Kind.
type GroupVersionKindPlural struct {
	schema.GroupVersionKind
	Plural string
}

func (g GroupVersionKindPlural) String() string {
	return fmt.Sprintf("%s/%s, Kind=%s, Plural=%s", g.Group, g.Version, g.Kind, g.Plural)
}

// KindPlural entails the Kind and its plural representation.
type KindPlural struct {
	Kind   string
	Plural string
}
