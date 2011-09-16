#!/bin/bash

# Ensure that we are compiled.

type -p gomake && gomake -C gosrc

if [ ! -f gosrc/bzwikipedia ] ; then
  echo "Apparently unable to compile bzwikipedia."
  exit
fi

# On windows ln == cp
[ `uname -s|sed 's/\(.....\).*/\1/'` = MINGW ] && rm bzwikipedia

[ -f bzwikipedia ] || ln -s gosrc/bzwikipedia bzwikipedia

./bzwikipedia --conf bzwikipedia.conf
