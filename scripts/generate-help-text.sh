#!/usr/bin/env bash

GOOS=$(go env GOOS)
GOARCH=$(go env GOARCH)

# goreleaser appends an arch-variant suffix: GOAMD64 for amd64, GOARM for arm
case "$GOARCH" in
  amd64) VARIANT="_$(go env GOAMD64)" ;;
  arm)   VARIANT="_$(go env GOARM)" ;;
  arm64) VARIANT="_$(go env GOARM64)" ;;
  *)     VARIANT="" ;;
esac

echo "$ kube-state-metrics -h" > help.txt
./dist/"kube-state-metrics_${GOOS}_${GOARCH}${VARIANT}"/kube-state-metrics -h >> help.txt
exit 0
