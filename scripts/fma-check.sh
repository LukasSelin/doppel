#!/usr/bin/env bash
#
# Fail if the compiler fused a multiply-add anywhere in this module's own code
# on arm64.
#
#   scripts/fma-check.sh
#
# Go may compile x*y + z into one FMA instruction, "possibly across
# statements", and does on arm64 where it does not on amd64. One expression
# then rounds once on an arm64 Mac and twice on amd64 Linux, and a score
# differs in its last bit. An explicit float64(x*y) rounds the product and
# forbids the fusion (see internal/detmath). This finds the expressions that
# still need one, without arm64 hardware: cross-compile, disassemble, and list
# every FMADD/FMSUB-family instruction inside a function of this module, with
# the source line it came from.
#
# internal/detmath is exempt. Its fused steps are math.FMA calls, written on
# purpose; math.FMA rounds once by definition, so they agree everywhere.
#
# It reads one compiler's output, so it sees what this Go version fuses and no
# more. A newer compiler that fuses more fails here first, which is the point.

set -euo pipefail

cd "$(dirname "$0")/.."

tmp=$(mktemp -d)
trap 'rm -rf -- "$tmp"' EXIT

GOOS=linux GOARCH=arm64 go build -o "$tmp/doppel" .
go tool objdump "$tmp/doppel" > "$tmp/objdump"

# An objdump instruction line is: file:line  address  encoding  mnemonic args.
# A TEXT line opens each function, and its symbol says whose code follows.
awk '
	/^TEXT / {
		sym = $2
		mine = sym ~ /^github\.com\/LukasSelin\/doppel\// && sym !~ /^github\.com\/LukasSelin\/doppel\/internal\/detmath\./
		next
	}
	mine && $4 ~ /^F(N)?M(ADD|SUB)[DS]$/ {
		print "  " $1 "  " $4 "  in " sym
		n++
	}
	END {
		if (n > 0) {
			print n " fused multiply-add(s) in doppel code on arm64; round the product with float64(x*y) (see internal/detmath)"
			exit 1
		}
		print "no fused multiply-add in doppel code on arm64"
	}
' "$tmp/objdump"
