bzwikipedia:

  Serve wikimedia (Wikipedia, Wiktionary, Wikinews, etc) format websites from
  xml.bz2 compressed files.

  Currently, values are hard coded, but I'll be adding config file support
  eventually-ish.
 
  Things should work out of the box on anything that has a Go compiler. 6g
  is the currently configured compiler, to change it, edit gosrc/Makefile

  WARNING: Still under development. It's ugly, and there's just raw text dumps
  of the wiki/ data, but it works!

Initial setup:

1) Download the pages-articles .xml.bz2 file from:

  http://en.wikipedia.org/wiki/Wikipedia:Database_download#English-language_Wikipedia

2) Drop the .xml.bz2 you just downloaded into the drop/ directory.

3) Start the server:

  OS X: Double click "StartWikiServer.command"
  Linux: Run "StartWikiServer.rb"
  Windows: However you run ruby stuff.

  It will perform initial setup on its own. This can take up to a few hours
  the first time and any time you drop a new .xml.bz2 file into the drop/
  directory

To access:

Go to http://localhost:2012

How to UPDATE:

  Simply kill the server, drop an updated pages-articles .xml.bz2 file into
  the drop/ directory and start the server again.
