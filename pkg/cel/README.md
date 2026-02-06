# CEL Extensions

CEL extension library for kube-state-metrics.

## Package Structure

* `pkg/cel/` - Type definitions
* `pkg/cel/library/` - CEL library implementation

## Types

### WithLabels Type

`WithLabels(value, labels)` wraps a metric value with additional labels.

```cel
WithLabels(100.0, {})
WithLabels(42, {'severity': 'high'})
WithLabels(double(value) * 10.0, {'multiplied': 'true'})
```

Fields:

* `Val` - metric value (converted to float64 by extractor)
* `AdditionalLabels` - labels to add to metric

## Usage

```go
import (
    "github.com/google/cel-go/cel"
    "k8s.io/kube-state-metrics/v2/pkg/cel/library"
)

env, err := cel.NewEnv(
    library.KSM(),
    cel.Variable("value", cel.DynType),
)
```

## Extending

Add new functions to `ksmLibraryDecls` in [library/library.go](library/library.go).

New types go in this package with ref.Val implementation, then register in library's `Types()` method.
