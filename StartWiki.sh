#!/bin/bash

# Ensure that we are compiled.

type -p gomake && gomake -C gosrc

if [ ! -f gosrc/bzwikipedia ] ; then
  echo "Apparently unable to compile bzwikipedia."
  exit
fi

[ -f bzwikipedia ] || ln -s gosrc/bzwikipedia bzwikipedia

./bzwikipedia
