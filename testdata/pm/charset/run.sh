#!/bin/sh

if test -z "$PM"
then	test -f ../../pm && PM=../../pm || PM=pm
fi

if ! $PM -Iiso-2022-jp ./x | diff x.expected -
then	exit 1
fi
