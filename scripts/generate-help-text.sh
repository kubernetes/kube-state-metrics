#!/usr/bin/env bash

echo "$ kube-state-metrics -h" > help.txt
./output/kube-state-metrics -h 2>> help.txt
exit 0
