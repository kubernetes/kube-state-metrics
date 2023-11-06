To run these tests, you need to provide two CLI flags:

- `--ksm-http-metrics-url`: url to access the kube-state-metrics service
- `--ksm-telemetry-url`: url to access the kube-state-metrics telemetry endpoint

Example:

```
go test -v ./tests/e2e \
   --ksm-http-metrics-url=http://localhost:8080/ \
   --ksm-telemetry-url=http://localhost:8081/
```
