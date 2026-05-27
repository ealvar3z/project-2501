#!/bin/sh
if test -z "$PM"
then	test -f ../../pm && PM=../../pm || PM=pm
fi

sed -E -e 's/^#([^ ])/\1/' ../../bonus/config.toml >tmp_config.toml
if ! $PM -Ctmp_config.toml /dev/null | diff /dev/null -
then	rm tmp_config.toml
	exit 1
fi
rm tmp_config.toml

$PM -Ctest.toml test.test >/dev/null || exit 1

if ! $PM -Ctest.toml -r 'cmd.custom.b()' 2>&1|diff test3.expected -
then	echo "Custom command test failed"
	exit 1
fi

if ! $PM -Ctest.toml test2.test2 | diff test2.expected -
then	exit 1
fi

$PM -r 'quit()'
