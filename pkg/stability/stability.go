/*
Copyright 2019 The Kubernetes Authors All rights reserved.

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

package stability

import (
	basemetrics "k8s.io/component-base/metrics"
)

// Stability represents the API guarantees for a given defined metric.
type Stability basemetrics.StabilityLevel

const (
	// ALPHA metrics have no stability guarantees, as such, labels may
	// be arbitrarily added/removed and the metric may be deleted at any time.
	ALPHA Stability = Stability(basemetrics.ALPHA)
	// STABLE metrics can be changed in few cases when alerts won't be
	// broken.
	// The deprecation policy outlined in the below or check
	// definition of basemetrics.BETA.
	//
	// Allowed changes:
	// - Additional labels are allowed
	// - Bug fixes to the implementation (e.g. change specific label values or edge cases on nil value)
	// - Description change
	// - Deprecation. Stable metric can be deprecated in minor release: Lifecycle is: STABLE (2.n)-> DEPRECATED (2.n+1)-> REMOVED (2.n+2). A removal is a breaking change and will be announced prominently in the CHANGELOG.
	//
	// Not allowed changes:
	// - Renaming, dropping of labels
	// - Changing metric name, metric type or metric meaning
	STABLE Stability = Stability(basemetrics.BETA)
)
