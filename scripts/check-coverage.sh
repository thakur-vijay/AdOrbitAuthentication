#!/usr/bin/env bash
set -euo pipefail

profile="${1:-coverage.out}"
if [[ ! -f "$profile" ]]; then
	echo "coverage profile not found: $profile" >&2
	exit 2
fi

coverage_for() {
	local path_prefix="$1"
	awk -v prefix="$path_prefix" '
		NR == 1 { next }
		{
			path = $1
			sub(/:[0-9].*$/, "", path)
			if (index(path, prefix) == 1) {
				# go test -coverprofile across multiple packages repeats the same
				# source block once per test binary. Merge those copies so a block
				# covered by any package is counted as covered exactly once.
				block = $1
				if (!(block in block_statements)) {
					block_statements[block] = $2
				}
				if ($3 > 0) block_covered[block] = 1
			}
		}
		END {
			for (block in block_statements) {
				statements += block_statements[block]
				if (block in block_covered) covered += block_statements[block]
			}
			if (statements == 0) {
				print "0.0"
			} else {
				printf "%.1f\n", 100 * covered / statements
			}
		}
	' "$profile"
}

failed=0

check_floor() {
	local label="$1"
	local path_prefix="$2"
	local floor="$3"
	local actual
	actual="$(coverage_for "$path_prefix")"
	printf '%-18s %5s%% (minimum %s%%)\n' "$label" "$actual" "$floor"
	if ! awk -v actual="$actual" -v floor="$floor" 'BEGIN { exit !(actual + 0 >= floor + 0) }'; then
		failed=1
	fi
}

check_floor "overall" "github.com/adorbit/authentication/" 40
check_floor "security" "github.com/adorbit/authentication/internal/security/" 90
check_floor "JWT and refresh" "github.com/adorbit/authentication/internal/jwt/" 65
check_floor "authentication" "github.com/adorbit/authentication/internal/auth/" 40
check_floor "email OTP" "github.com/adorbit/authentication/internal/emailOtp/" 45
check_floor "legacy sessions" "github.com/adorbit/authentication/internal/sessions/" 60
check_floor "RabbitMQ client" "github.com/adorbit/authentication/internal/messaging/" 45

if ((failed)); then
	echo "coverage is below one or more required CI floors" >&2
	exit 1
fi
