#!/bin/sh
if ! test "$PM"
then	test -f ../../pm && PM=../../pm || PM=pm
fi
failed=0
for h in *.html
do	printf '%s\r' "$h"
	if ! "$PM" -dC config.toml "$h" | diff all.expected -
	then	failed=$(($failed+1))
		printf 'FAIL: %s\n' "$h"
	fi
done
printf '\n'
exit "$failed"
