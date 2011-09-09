// main.go

package main

import (
	"bytes"
	"bzreader"
	"confparse"
	"exec"
	"fmt"
	"http"
	"os"
	"path/filepath"
        "reflect"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
        "syscall"
	"template"
	"time"
        "unsafe"
        "utf8"
)

// current db name, if extant.
var curdbname string

// Current cache version.
var current_cache_version = 3

// Current bzwikipedia.dat info
var dat map[string]string

// global config variable
var conf = map[string]string{
	"listen":          ":2012",
	"drop_dir":        "drop",
	"data_dir":        "pdata",
	"title_file":      "pdata/titlecache.dat",
	"dat_file":        "pdata/bzwikipedia.dat",
	"web_dir":         "web",
	"wiki_template":   "web/wiki.html",
	"search_template": "web/searchresults.html",
        "cache_type":      "mmap",
        "cache_ignore_redirects": "true",
        "cache_ignore_rx": "^(File|Category|Wikipedia|MediaWiki|Templates|Portal):",
        "search_routines": "4",
        "search_ignore_rx": "",
}

func basename(fp string) string {
	return filepath.Base(fp)
}

var searchRoutines = 4
var ignoreSearchRx *regexp.Regexp

//
// Go provides a filepath.Base but not a filepath.Dirname ?!
// Given foo/bar/baz, return foo/bar
//
var dirnamerx = regexp.MustCompile("^(.*)/")

func dirname(fp string) string {
	matches := dirnamerx.FindStringSubmatch(filepath.ToSlash(fp))
	if matches == nil {
		return "."
	}

	nfp := matches[1]
	if nfp == "" {
		nfp = "/"
	}
	return filepath.FromSlash(nfp)
}

//
// Convert enwiki-20110405-pages-articles.xml into the integer 20110405
//
var timestamprx = regexp.MustCompile("(20[0-9][0-9])([0-9][0-9])[^0-9]*([0-9][0-9])")

func fileTimestamp(fp string) int {
	matches := timestamprx.FindStringSubmatch(basename(fp))
	if matches == nil {
		return 0
	}
	tyear, _ := strconv.Atoi(matches[1])
	tmonth, _ := strconv.Atoi(matches[2])
	tday, _ := strconv.Atoi(matches[3])
	return tyear*10000 + tmonth*100 + tday
}

//
// Check data_dir for the newest (using filename YYYYMMDD timestamp)
// *.xml.bz2 file that exists, and return it.
//
func getRecentDb() string {
	dbs, _ := filepath.Glob(filepath.Join(conf["drop_dir"], "*.xml.bz2"))
	recent := ""
	recentTimestamp := -1
	for _, fp := range dbs {
		// In the event of a non-timestamped filename.
		if recent == "" {
			recent = fp
		}
		ts := fileTimestamp(fp)
		if ts > recentTimestamp {
			recentTimestamp = ts
			recent = fp
		}
	}
	return recent
}

//
// dosplit, docache := needUpdate()
// If dosplit is true, then call bzip2recover.
// If docache is true, then the title cache file needs to
// be regenerated.
//
func needUpdate(recent string) (bool, bool) {
	olddat, err := confparse.ParseFile(conf["dat_file"])
	version := 0

	if err == nil {
                version, err = strconv.Atoi(olddat["version"])

		if err != nil {
			fmt.Println("Dat file has invalid format.")
			return true, true
		}

		if basename(olddat["dbname"]) == basename(recent) {
			// The .bz2 records exist, but we may need to
			// regenerate the title cache file.
			if version < current_cache_version {
                                fmt.Printf("Version of the title cache file is %d.\n", version)
                                fmt.Printf("Wiping cache and replacing with version %d. This will take a while.\n", current_cache_version)
                                time.Sleep(5000000000)
				return false, true
			}
                        cic := conf["cache_ignore_redirects"] == "true"
                        cid := olddat["cache_ignore_redirects"] == "true"
                        if cic != cid {
                                fmt.Println("cache_ignore_redirects value has changed.")
                                fmt.Println("Wiping cache and regenerating.\n")
                                time.Sleep(5000000000)
				return false, true
                        }

                        if conf["cache_ignore_rx"] != olddat["cache_ignore_rx"] {
                                fmt.Println("cache_ignore_rx value has changed.")
                                fmt.Println("Wiping cache and regenerating.\n")
                                time.Sleep(5000000000)
				return false, true
                        }
			return false, false
		}
	} else {
		fmt.Printf("Unable to open %v: %v\n", conf["dat_file"], err)
	}
	return true, true
}

//
// Clear out any old rec*.xml.bz2 or titlecache.txt files
//
func cleanOldCache() {
	recs, _ := filepath.Glob(filepath.Join(conf["data_dir"], "rec*.xml.bz2"))
	tfs, _ := filepath.Glob(conf["title_file"])
	dfs, _ := filepath.Glob(conf["dat_file"])

	// If any old record or title cache files exist, give the user an opportunity
	// to ctrl-c to cancel this.

	if len(recs) > 0 || len(tfs) > 0 || len(dfs) > 0 {
		fmt.Println("Old record and/or title cache file exist. Removing in 5 seconds ...")
		time.Sleep(5000000000)
	}

	if len(recs) > 0 {
		fmt.Println("Removing old record files . . .")
		for _, fp := range recs {
			os.Remove(fp)
		}
	}

	if len(tfs) > 0 {
		fmt.Println("Removing old title file . . .")
		for _, fp := range tfs {
			os.Remove(fp)
		}
	}

	if len(dfs) > 0 {
		fmt.Println("Removing old dat file . . .")
		for _, fp := range dfs {
			os.Remove(fp)
		}
	}
}

//
// Copy the big database into the data/ dir, bzip2recover to split it into
// rec#####dbname.bz2, and move the big database back to the drop dir.
//
func splitBz2File(recent string) {
	// Be user friendly: Alert the user and wait a few seconds."
	fmt.Println("I will be using bzip2recover to split", recent, "into many smaller files.")
	time.Sleep(3000000000)

	// Move the recent db over to the data dir since bzip2recover extracts
	// to the same directory the db exists in, and we don't want to pollute
	// drop_dir with the rec*.xml.bz2 files.
	newpath := filepath.Join(conf["data_dir"], basename(recent))
	err := os.Rename(recent, newpath)

	if err != nil {
		if e, ok := err.(*os.LinkError); ok && e.Error == os.EXDEV {
			panic(GracefulError("Your source file must be on the same partition as your target dir. Sorry."))
		} else {
			panic(fmt.Sprintf("rename: %T %#v\n", err, err))
		}
	}

	// Make sure that we move it _back_ to drop dir, no matter what happens.
	defer os.Rename(newpath, recent)

	args := []string{"bzip2recover", newpath}

	executable, patherr := exec.LookPath("bzip2recover")
	if patherr != nil {
		fmt.Println("bzip2recover not found anywhere in your path, making wild guess")
		executable = "/usr/bin/bz2recover"
	}

	environ := os.ProcAttr{
		Dir:   ".",
		Env:   os.Environ(),
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	}

	bz2recover, err := os.StartProcess(executable, args, &environ)

	if err != nil {
		switch t := err.(type) {
		case *os.PathError:
			if err.(*os.PathError).Error == os.ENOENT {
				panic(GracefulError("bzip2recover not found. Giving up."))
			} else {
				fmt.Printf("err is: %T: %#v\n", err, err)
				panic("Unable to run bzip2recover? err is ")
			}
		default:
			fmt.Printf("err is: %T: %#v\n", err, err)
			panic("Unable to run bzip2recover? err is ")
		}
	}
	bz2recover.Wait(0)
}

type TitleData struct {
	Title string
	Start int
}

type tdlist []TitleData

func (tds tdlist) Len() int {
	return len(tds)
}
func (tds tdlist) Less(a, b int) bool {
	return tds[a].Title < tds[b].Title
}
func (tds tdlist) Swap(a, b int) {
	tds[a], tds[b] = tds[b], tds[a]
}
func (tds tdlist) Sort() {
	sort.Sort(tds)
}

//
// Generate the new title cache file.
//
func generateNewTitleFile() (string, string) {
	// Create pdata/bzwikipedia.dat.
	dat_file_new := fmt.Sprintf("%v.new", conf["dat_file"])
	dfout, derr := os.OpenFile(dat_file_new, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if derr != nil {
		fmt.Printf("Unable to create '%v': %v", dat_file_new, derr)
		return "", ""
	}
	defer dfout.Close()

	// Create pdata/titlecache.dat.
	title_file_new := fmt.Sprintf("%v.new", conf["title_file"])
	fout, err := os.OpenFile(title_file_new, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		fmt.Printf("Unable to create '%v': %v", title_file_new, derr)
		return "", ""
	}
	defer fout.Close()

        ignoreRedirects := conf["cache_ignore_redirects"] == "true";
        var ignoreRx *regexp.Regexp = nil

        irx := conf["cache_ignore_rx"]

        if irx != "" {
          ignoreRx = regexp.MustCompile(irx)
        }

	// Plop version and dbname into bzwikipedia.dat
	fmt.Fprintf(dfout, "version:%d\n", current_cache_version)
	fmt.Fprintf(dfout, "dbname:%v\n", curdbname)
        fmt.Fprintf(dfout, "cache_ignore_rx:%v\n", irx)
        if ignoreRedirects {
          fmt.Fprintf(dfout, "cache_ignore_redirects:true\n")
        } else {
          fmt.Fprintf(dfout, "cache_ignore_redirects:false\n")
        }

	// Now read through all the bzip files looking for <title> bits.
	bzr := bzreader.NewBzReader(conf["data_dir"], curdbname, 1)

	// We print a notice every 1000 chunks, just 'cuz it's more user friendly
	// to show _something_ going on.
	nextprint := 0

	// For title cache version 3:
	//
	// We are using \ntitlename\x02record, and it is sorted,
	// case sensitively, for binary searching.
	//
	// We are optionally discarding redirects and other titles.
        // Discarding redirects adds a small amount of complexity since we have
        // <title>, then a few lines later <redirect may or may not exist. So
        // we don't add <title> to the array until either A: We see another
        // <title> without seeing <redirect, or we reach end of file.
	//

	var titleslice tdlist
	var td *TitleData
	for {
		curindex := bzr.Index
		if curindex >= nextprint {
			nextprint = curindex + 1000
			fmt.Println("Reading chunk", curindex)
		}
		str, err := bzr.ReadString()
		if err == os.EOF {
			break
		}
		if err != nil {
			fmt.Printf("Error while reading chunk %v: %v\n", bzr.Index, err)
			panic("Unrecoverable error.")
		}

		// This accounts for both "" and is a quick optimization.
		if len(str) < 10 {
			continue
		}

		idx := strings.Index(str, "<title>")

		if idx >= 0 {
			if td != nil {
				titleslice = append(titleslice, *td)
                                td = nil
			}
			eidx := strings.Index(str, "</title>")
			if eidx < 0 {
				fmt.Printf("eidx is less than 0 for </title>?\n")
				fmt.Printf("Index %d:%d\n", curindex, bzr.Index)
				fmt.Printf("String is: '%v'\n", str)
				panic("Can't find </title> tag - broken bz2?")
			}
			title := str[idx+8 : eidx]
                        if ignoreRx == nil || !ignoreRx.MatchString(title) {
                          td = &TitleData{Title: title, Start: curindex}
                        }
		} else if ignoreRedirects && strings.Contains(str, "<redirect") {
			if td != nil {
				// Discarding redirect.
				td = nil
			}
		}
	}
	if td != nil {
		titleslice = append(titleslice, *td)
	}
	// Now sort titleslice.
	titleslice.Sort()

	for _, i := range titleslice {
		fmt.Fprintf(fout, "\n%s\x02%d", i.Title, i.Start)
	}

	fmt.Fprintf(dfout, "rcount:%v\n", len(titleslice))

	// We are now done with our in-memory list of Title data.
	// Let's aggressively GC.
	runtime.GC()

	return title_file_new, dat_file_new
}

////// Title file format: Version 2
// \ntitle\x02startsegment

////// bzwikipedia.dat file format:
// version:2
// dbname:enwiki-20110405-pages-articles.xml.bz2
// rcount:12345
// (rcount being record count.)

//
// Check if any updates to the cached files are needed, and perform
// them if necessary.
//
func performUpdates() {
	fmt.Printf("Checking for new .xml.bz2 files in '%v/'.", conf["drop_dir"])
	recent := getRecentDb()
	if recent == "" {
		fmt.Println("No available database exists in '%v/'.", conf["drop_dir"])
	}
	fmt.Println("Latest DB:", recent)

	dosplit, docache := needUpdate(recent)

	if !docache {
		fmt.Println("Cache update not required.")
		return
	}

	if dosplit {
		// Clean out old files if we need 'em to be.
		cleanOldCache()

		// Turn the big old .xml.bz2 into a bunch of smaller .xml.bz2s
		splitBz2File(recent)
	}

	curdbname = basename(recent)

	// Generate a new title file and dat file
	newtitlefile, newdatfile := generateNewTitleFile()

	// Rename them to the actual title and dat file
	os.Rename(newtitlefile, conf["title_file"])
	os.Rename(newdatfile, conf["dat_file"])

	// We have now completed pre-processing! Yay!
}

// Now we load the title cache file. We read it in as one huge lump.
// \ntitle\x02startsegment

var record_count int
var title_blob []byte
var title_size int64

func loadTitleFile() bool {
        var derr os.Error
	dat, derr = confparse.ParseFile(conf["dat_file"])
	if derr != nil { fmt.Println(derr); return false }

	curdbname = dat["dbname"]
	record_count, derr = strconv.Atoi(dat["rcount"])
	if derr != nil { fmt.Println(derr); return false }

	fmt.Printf("DB '%s': Contains %d records.\n", curdbname, record_count)

        //
        // Read in the massive title blob.
        //
        fin, err := os.Open(conf["title_file"])
        if err != nil {
                fmt.Println(err)
                return false
        }
        defer fin.Close()

	// Find out how big it is.
	stat, err := fin.Stat()
	if err != nil {
		fmt.Printf("Error while slurping in title cache: '%v'\n", err)
		return false
	}
	title_size = stat.Size


        // How should we approach this? We have a few options:
        //  mmap: Use disk. Less memory, but slower access.
        //  ram: Read into RAM. A lot more memory, but faster access.
        dommap := conf["cache_type"] == "mmap"

        if dommap {
          // Try to mmap.
          addr, _, errno := syscall.Syscall6(syscall.SYS_MMAP,
                                             0,
                                             uintptr(title_size),
                                             uintptr(1),
                                             uintptr(2),
                                             uintptr(fin.Fd()),
                                             0)
          if errno == 0 {
            dh := (*reflect.SliceHeader)(unsafe.Pointer(&title_blob))
            dh.Data = addr
            dh.Len = int(title_size) // Hmmm.. truncating here feels like trouble.
            dh.Cap = dh.Len
            fmt.Printf("Successfully mmaped!\n")
          } else {
            fmt.Printf("Unable to mmap! error: '%v'\n", os.Errno(errno))
          }
        }
        if !dommap {
          // Default: Load into memory.
          title_blob = make([]byte, title_size, title_size)

          nread, err := fin.Read(title_blob)

          if err != nil && err != os.EOF {
                  fmt.Printf("Error while slurping in title cache: '%v'\n", err)
                  return false
          }
          if int64(nread) != title_size || err != nil {
                  fmt.Printf("Unable to read entire file, only read %d/%d\n",
                          nread, stat.Size)
                  return false
          }
        }
	return true
}

// var title_blob []byte
// var title_size int64

// Binary search within a blob of unequal length strings.
func findTitleData(name string) (TitleData, bool) {
	// We limit to 100, just in case.
	searchesLeft := 100

	min := int64(-1)
	max := int64(title_size)

	minlen := int64(len(name))

search:
	for {
		// Find the halfway point.
		if searchesLeft <= 0 {
			break search
		}
		searchesLeft -= 1
		cur := int64(((max - min) / 2) + min)
		origcur := cur
		if (title_size - cur) < minlen {
			break search
		}

		// Go backwards to look for the \n that signifies start of
		// record.
	record:
		for {
			if cur <= min {
				if cur <= min {
					// We may be very close, but searching the wrong way. Search forward,
					// now.
					cur = origcur
					for {
						if cur > max {
							break search
						}
						if title_blob[cur] == '\n' {
							break record
						}
						cur += 1
					}
				}
			}
			if title_blob[cur] == '\n' {
				break record
			}
			cur -= 1
		}

		if (max - cur) < minlen {
			break search
		}

		recordStart := cur + 1
		recordEnd := recordStart + 1
		for {
			if title_blob[recordEnd] == '\x02' {
				break
			}
			recordEnd += 1
		}

		td := TitleData{}

		// We have the title.
		td.Title = string(title_blob[recordStart:recordEnd])

		// Now we look for the \x02###(\n|end) for the index.
		recordStart = recordEnd + 1
		recordEnd = recordStart + 1
		for {
			if recordEnd >= title_size {
				recordEnd = title_size
				break
			}
			if title_blob[recordEnd] == '\n' {
				break
			}
			recordEnd += 1
		}
		num := string(title_blob[recordStart:recordEnd])

		td.Start, _ = strconv.Atoi(num)

		// Did we find it? Did we?
		if td.Title == name {
			return td, true
		}

		// Nope, let's divide and conquer.
		if td.Title < name {
			min = cur
		} else if td.Title > name {
			max = cur
		}
	}
	return TitleData{}, false
}

var wholetextrx = regexp.MustCompile("<text[^>]*>(.*)</text>")
var starttextrx = regexp.MustCompile("<text[^>]*>(.*)")
var endtextrx = regexp.MustCompile("(.*)</text>")

func readTitle(td TitleData) string {
	var str string
	var err os.Error

	toFind := fmt.Sprintf("<title>%s</title>", td.Title)

	// Start looking for the title.
	bzr := bzreader.NewBzReader(conf["data_dir"], curdbname, td.Start)

	for {
		str, err = bzr.ReadString()
		if err != nil {
			return ""
		}
		if strings.Contains(str, toFind) {
			break
		}
	}

	toFind = "<text"
	for {
		str, err = bzr.ReadString()
		if err != nil {
			return ""
		}
		if strings.Contains(str, toFind) {
			break
		}
	}

	// We found <text> in string. Capture everything after it.
	// It may contain </text>
	matches := wholetextrx.FindStringSubmatch(str)
	if matches != nil {
		return matches[1]
	}

	// Otherwise, it just has <text>
	buffer := bytes.NewBufferString("")

	matches = starttextrx.FindStringSubmatch(str)
	if matches != nil {
		fmt.Fprint(buffer, matches[1])
	}

	toFind = "</text>"
	for {
		str, err = bzr.ReadString()
		if err != nil {
			return ""
		}
		if strings.Contains(str, toFind) {
			break
		}
		fmt.Fprint(buffer, str)
	}

	matches = endtextrx.FindStringSubmatch(str)
	if matches != nil {
		fmt.Fprint(buffer, matches[1])
	}

	return string(buffer.Bytes())
}

func getTitle(str string) string {
	// Turn foo_bar into foo bar. Strip leading and trailing spaces.
	str = strings.TrimSpace(str)
	str = strings.Replace(str, "_", " ", -1)
	return str
}

type SearchPage struct {
	Phrase  string
	Results string
}

func getTitleFromPos(haystack []byte, pos int) string {
  var i, end int
  for i = pos; i > 0 && haystack[i] != '\n' ; i -= 1 {}
  for end = i; end < len(haystack) && haystack[end] != '\x02'; end++ {}
  return string(haystack[i+1:end])
}

func caseInsensitiveFinds(haystack, needle []byte, watchdog chan []string) {
  results := []string{}[:]

  defer func() {
    watchdog <- results
  }()

  n := len(needle)
  if n == 0 {
    return
  }

  var urunes []int
  if true {
    tmp := bytes.ToUpper(needle)
    tmpc := utf8.RuneCount(tmp)
    urunes = make([]int, tmpc)

    for i, j := 0, 0; i < tmpc && j < len(tmp); i++ {
      rune, cnt := utf8.DecodeRune(tmp[j:])
      j += cnt
      urunes[i] = rune
    }
  }

  var lrunes []int
  if true {
    tmp := bytes.ToLower(needle)
    tmpc := utf8.RuneCount(tmp)
    lrunes = make([]int, tmpc)

    for i, j := 0, 0; i < tmpc && j < len(tmp); i++ {
      rune, cnt := utf8.DecodeRune(tmp[j:])
      j += cnt
      lrunes[i] = rune
    }
  }

  lc := lrunes[0]
  uc := urunes[0]
  n  = len(lrunes)

  maxlen := len(haystack)

  for i := 0; (i + n) < maxlen; {
    r, cnt := utf8.DecodeRune(haystack[i:])
    i += cnt

    // Check the first rune against either the lower or upper case needle
    // rune.
    if r == lc || r == uc {
      x := i
      var s int

      // Check the rest.
      for s = 1; s < n; s++ {
        r, cnt := utf8.DecodeRune(haystack[x:])
        x += cnt
        if !(r == urunes[s] || r == lrunes[s]) { break }
      }
      if s >= n {
        cur := getTitleFromPos(haystack, i)
        if (ignoreSearchRx == nil || !ignoreSearchRx.MatchString(cur)) {
          results = append(results, cur)
        }
        for {
          if i > maxlen || haystack[i] == '\n' { break }
          i += 1
        }
      }
    }
  }
}

var SearchTemplate *template.Template

func searchHandle(w http.ResponseWriter, req *http.Request) {
  // "/search/"
  pagetitle := getTitle(req.URL.Path[8:])
  if len(pagetitle) < 4 {
          fmt.Fprintf(w, "Search phrase too small for now.")
          return
  }

  // Search all keys
  watchdog := make(chan []string)

  // Start a goroutine for searching.
  go caseInsensitiveFinds(title_blob, []byte(pagetitle), watchdog)

  results := <- watchdog

  newtpl, terr := template.ParseFile(conf["search_template"], nil)
  if terr != nil {
          fmt.Println("Error in template:", terr)
  } else {
          SearchTemplate = newtpl
  }

  p := SearchPage{Phrase: pagetitle, Results: strings.Join(results, "|")}
  err := SearchTemplate.Execute(w, &p)

  if err != nil {
          http.Error(w, err.String(), http.StatusInternalServerError)
  }
}

type WikiPage struct {
	Title string
	Body  string
}

var WikiTemplate *template.Template

func pageHandle(w http.ResponseWriter, req *http.Request) {
	// "/wiki/"
	pagetitle := getTitle(req.URL.Path[6:])

	newtpl, terr := template.ParseFile(conf["wiki_template"], nil)
	if terr != nil {
		fmt.Println("Error in template:", terr)
	} else {
		WikiTemplate = newtpl
	}

	td, ok := findTitleData(pagetitle)

	if ok {
		p := WikiPage{Title: pagetitle, Body: readTitle(td)}
		err := WikiTemplate.Execute(w, &p)

		if err != nil {
			fmt.Printf("Error with WikiTemplate.Execute: '%v'\n", err)
		}
	} else {
		http.Error(w, "No such Wiki Page", http.StatusNotFound)
	}

}

func parseConfig(confname string) {
	fromfile, err := confparse.ParseFile(confname)
	if err != nil {
		fmt.Printf("Unable to read config file '%s'\n", confname)
		return
	}

	fmt.Printf("Read config file '%s'\n", confname)

	for key, value := range fromfile {
		if _, ok := conf[key] ; !ok {
			fmt.Printf("Unknown config key: '%v'\n", key)
		} else {
			conf[key] = value
		}
	}

        // Set globals for speed.
        if conf["search_ignore_rx"] != "" {
          ignoreSearchRx = regexp.MustCompile(conf["search_ignore_rx"])
        } else {
          ignoreSearchRx = nil
        }

        if conf["search_routines"] != "" {
          searchRoutines, err = strconv.Atoi(conf["search_routines"])
          if err != nil {
            fmt.Println("search_routines: Unable to parse '%v' as integer: '%v'.\n",
                        conf["search_routines"], err)
            fmt.Println("search_routines: Using default value.\n")
            searchRoutines = 4
          } else if searchRoutines < 1 || searchRoutines > 64 {
            fmt.Println("search_routines: Number '%v' Out of range (1-64).\n", searchRoutines)
          }
        }
}

type GracefulError string

func main() {
	// Defer this first to ensure cleanup gets done properly
	// 
	// Any error of type GracefulError is handled with an exit(1)
	// rather than by handing the user a backtrace.
	defer func() {
		problem := recover()
		switch problem.(type) {
		case GracefulError:
			fmt.Println(problem)
			os.Exit(1)
		default:
			panic(problem)
		}
	}()

	fmt.Println("Switching dir to", dirname(os.Args[0]))
	os.Chdir(dirname(os.Args[0]))

	parseConfig("bzwikipedia.conf")

	// Load the templates first.
	SearchTemplate = template.MustParseFile(conf["search_template"], nil)
	WikiTemplate = template.MustParseFile(conf["wiki_template"], nil)

	// Check for any new databases, including initial startup, and
	// perform pre-processing.
	performUpdates()

	// Load in the title cache
	if !loadTitleFile() {
		fmt.Println("Unable to read Title cache file: Invalid format?")
		return
	}

	fmt.Println("Loaded! Preparing templates ...")

	fmt.Println("Starting Web server on port", conf["listen"])

	// /wiki/... are pages.
	http.HandleFunc("/wiki/", pageHandle)
	// /search/ look for given text
	http.HandleFunc("/search/", searchHandle)

	// Everything else is served from the web dir.
	http.Handle("/", http.FileServer(http.Dir(conf["web_dir"])))

	err := http.ListenAndServe(conf["listen"], nil)
	if err != nil {
		fmt.Println("Fatal error:", err.String())
	}
}
