#!/bin/bash

# Ensure that we are compiled.

cd "`dirname $0`"

mv pdata/bzwikipedia.dat pdata/bzwikipedia.dat.old

exec sh StartWiki.sh
