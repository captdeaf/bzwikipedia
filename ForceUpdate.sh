#!/bin/bash

# Ensure that we are compiled.

cd "`dirname $0`"

echo "About to remove pdata/bzwikipedia.dat in 10 seconds"
echo "Press Ctrl+c to cancel."

sleep 10

mv pdata/bzwikipedia.dat pdata/bzwikipedia.dat.old

exec sh StartWiki.sh
