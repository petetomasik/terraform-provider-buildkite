#!/bin/bash
set -ueo pipefail

ln -nsf . terraform-provider-buildkite && cd terraform-provider-buildkite

echo "--- :terraform: Running make docs"
make docs

echo "--- :git: Checking for changes"
git diff --exit-code docs &>/dev/null && true

docs_changes="$?"

if [ "${docs_changes:-0}" -ne 0 ] ; then
	echo "+++ :bk-status-failed: Documentation changes detected!!!"
	git status --short | tee -a git_diff_output
	printf '```git status --short\n%b\n```' "$(cat git_diff_output)" | buildkite-agent annotate --style info
fi

exit "$docs_changes"
