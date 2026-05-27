#!/bin/sh
if test -z "$PM"
then	test -f ../../pm && PM=../../pm || PM=pm
fi
failed=0
for h in *.html
do	printf '%s\r' "$h"
	expected=${h%.html}.expected
	if test -f "$expected"
	then	if ! "$PM" -C config.toml "$h" | diff "$expected" -
		then	failed=$(($failed+1))
			printf 'FAIL: %s\n' "$h"
		fi
	else	printf 'WARNING: expected file not found for %s\n' "$h"
	fi
done
printf '\n'
exit "$failed"
