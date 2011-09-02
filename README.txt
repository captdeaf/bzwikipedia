bzwikipedia:

  Serve wikimedia (Wikipedia, Wiktionary, Wikinews, etc) format websites from
  xml.bz2 compressed files.

  Things should work out of the box on anything that has a Go compiler.

  WARNING: Still under development. It's ugly, and there's just raw text dumps
  of the wiki/ data, but it works!

Initial setup:

1) Download the pages-articles .xml.bz2 file from:

  http://en.wikipedia.org/wiki/Wikipedia:Database_download#English-language_Wikipedia

2) Drop the .xml.bz2 you just downloaded into the drop/ directory.

   If there is only one .xml.bz2 file, then bzwikipedia will use that. If
   there is more than one, then bzwikipedia will use the one with the most
   recent timestamp in the filename
   (e.g: enwiki-20110803-pages-articles.xml.bz2)

3) Start the server:

  Linux: Run "StartWikiServer.sh"

  It will perform initial setup on its own. This can take up to a few hours
  the first time and any time you drop a new .xml.bz2 file into the drop/
  directory

To access:

Go to http://localhost:2012

How to UPDATE:

  Simply kill the server, drop an updated pages-articles .xml.bz2 file with a
  newer timestamp in its filename (e.g: enwiki-20110803-pages-articles will
  replace enwiki-20110403-pages-articles) into the drop/ directory and start
  the server again.

  Alternately, if you aren't using timestamps in the filenames, run
  ForceUpdate.sh
