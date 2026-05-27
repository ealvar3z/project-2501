#!/bin/sh

if ! test "$PM"
then	test -f ../../pm && PM=../../pm || PM=pm
fi

./run -a | {
	IFS= read -r port
	addr="http://localhost:$port"
	# Now we have to close stdin, but if I exec here then ./run dies.
	</dev/null | {
		failed=0
		for h in *.html *.http
		do	case $h in
			cookie.css.http|headers.http|module*.http) continue;;
			esac
			printf '%s\r' "$h"
			if ! "$PM" -C config.toml "$addr/$h" | diff all.expected -
			then	failed=$(($failed+1))
				printf 'FAIL: %s\n' "$h"
			fi
		done
		printf '\n'
		$PM -C config.toml -d "$addr/stop" >/dev/null
		exit "$failed"
	}
}
