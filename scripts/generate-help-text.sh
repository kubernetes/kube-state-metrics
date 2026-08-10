#!/usr/bin/env bash

echo "$ kube-state-metrics -h" > help.txt
./kube-state-metrics -h >> help.txt
exit 0
