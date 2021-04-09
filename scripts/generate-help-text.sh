#!/usr/bin/env bash

echo "$ kube-state-metrics -h" > help.txt
./kube-state-metrics -h 2>> help.txt
echo "$ kube-state-metrics -h" > docs/help.txt
./kube-state-metrics -h 2>> docs/help.txt
exit 0
