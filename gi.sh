#!/usr/bin/env bash
set -eu

for file in $(find . -name "*.go"); do
  echo ${file%/*}
done | \
  sed "s|\.|github.com/Nerdmaster/sendmail|g" | \
  sort | uniq | \
  xargs -l1 go install
