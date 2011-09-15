bzwikipedia:

  Serve wikimedia (Wikipedia, Wiktionary, Wikinews, etc) format websites from
  xml.bz2 compressed files.

  This is intended for people to run on their own laptops, taking few
  resources (once the initial title caching is done), so they can have
  access to wikipedia.

Features:

  * Serves wipedia pages/articles using limited resources: 7.2GB on disk
    and 10-20MB RAM (up to 100MB burst, with search).

  * Fast wiki page access. "search" is fast for the resources given.

  * Advanced title search: Ignoring punctuation, spaces and case.

  * Quick and easy setup.

  * Optionally ignores redirect articles. (Default: ignores redirects)

  * Optionally ignores certain pages. (Default: Ignores metadata pages)

Initial setup:

  Things should work out of the box on anything that has a Go compiler and
  bzip2recover.

1) Download the pages-articles .xml.bz2 file from:

  http://en.wikipedia.org/wiki/Wikipedia:Database_download#English-language_Wikipedia

2) Drop the .xml.bz2 you just downloaded into the drop/ directory.

   If there is only one .xml.bz2 file, then bzwikipedia will use that. If
   there is more than one, then bzwikipedia will use the one with the most
   recent timestamp in the filename
   (e.g: enwiki-20110803-pages-articles.xml.bz2)

3) Optionally: Edit bzwikipedia.conf to fiddle with your own settings.

4) Start the server:

  Linux: Run "StartWikiServer.sh"

  It will perform initial setup on its own. This can take up to a few hours
  the first time and any time you drop a new .xml.bz2 file into the drop/
  directory.

  NOTE: Unfortunately, when it parses the .xml.bz2 file, it can chew up
  close to a GB of RAM. This is one time only, and I'm considering a process
  to let people download pre-generated titlecache.dat and bzwikipedia.dat
  files.

To access:

Go to http://localhost:2012

How to UPDATE:

  Simply kill the server, drop an updated pages-articles .xml.bz2 file with a
  newer timestamp in its filename (e.g: enwiki-20110803-pages-articles will
  replace enwiki-20110403-pages-articles) into the drop/ directory and start
  the server again.

  Alternately, if you aren't using timestamps in the filenames, run
  ForceUpdate.sh
