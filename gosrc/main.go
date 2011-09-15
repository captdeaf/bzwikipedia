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
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"template"
	"time"
	"unicode"
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
	"listen":                 ":2012",
	"drop_dir":               "drop",
	"data_dir":               "pdata",
	"title_file":             "pdata/titlecache.dat",
	"dat_file":               "pdata/bzwikipedia.dat",
	"web_dir":                "web",
	"wiki_template":          "web/wiki.html",
	"search_template":        "web/searchresults.html",
	"cache_type":             "mmap",
	"cache_ignore_redirects": "true",
	"cache_ignore_rx":        "^(File|Category|Wikipedia|MediaWiki|Templates|Portal):",
	"search_routines":        "4",
	"search_ignore_rx":       "",
	"search_max_results":     "100",
	"recents_file":           "pdata/recent.dat",
	"recents_count":          "30",
}

func basename(fp string) string {
	return filepath.Base(fp)
}

var searchRoutines = 4
var searchMaxResults = 100
var ignoreSearchRx *regexp.Regexp

type searchRange struct{ Start, End int64 }

var recentCount int
var recentPages []string

var searchRanges []searchRange

const TITLE_DELIM = '\n'
const RECORD_DELIM = '\x02'

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
		executable = "/usr/bin/bzip2recover"
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

type searchlist []string

func (sl searchlist) Len() int {
	return len(sl)
}
func (sl searchlist) Less(a, b int) bool {
	x := len(sl[a]) - len(sl[b])
	if x == 0 {
		return sl[a] < sl[b]
	}
	if x > 0 {
		return false
	}
	return true
}
func (sl searchlist) Swap(a, b int) {
	sl[a], sl[b] = sl[b], sl[a]
}
func (sl searchlist) Sort() {
	sort.Sort(sl)
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

	ignoreRedirects := conf["cache_ignore_redirects"] == "true"
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
	// We are using <TITLE_DELIM>titlename<RECORD_DELIM>record, and it is sorted,
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
			nextprint = nextprint + 100
			fmt.Println("Reading chunk", curindex)
		}
		bstr, err := bzr.ReadBytes()
		if err == os.EOF {
			break
		}
		if err != nil {
			fmt.Printf("Error while reading chunk %v: %v\n", bzr.Index, err)
			panic("Unrecoverable error.")
		}

		// This accounts for both "" and is a quick optimization.
		if len(bstr) < 10 {
			continue
		}

		idx := bytes.Index(bstr, []byte("<title>"))

		if idx >= 0 {
			if td != nil {
				titleslice = append(titleslice, *td)
				td = nil
			}
			eidx := bytes.Index(bstr, []byte("</title>"))
			if eidx < 0 {
				fmt.Printf("eidx is less than 0 for </title>?\n")
				fmt.Printf("Index %d:%d\n", curindex, bzr.Index)
				fmt.Printf("String is: '%s'\n", bstr)
				panic("Can't find </title> tag - broken bz2?")
			}
			title := string(bstr[idx+7 : eidx])
			if ignoreRx == nil || !ignoreRx.MatchString(title) {
				td = &TitleData{Title: title, Start: curindex}
			}
		} else if ignoreRedirects && bytes.Index(bstr, []byte("<redirect")) >= 0 {
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
		fmt.Fprintf(fout, "%c%s%c%d", TITLE_DELIM, i.Title, RECORD_DELIM, i.Start)
	}

	fmt.Fprintf(dfout, "rcount:%v\n", len(titleslice))

	return title_file_new, dat_file_new
}

////// Title file format: Version 2
// <TITLE_DELIM>title<RECORD_DELIM>startsegment

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
	// Let's celebrate by restarting to clear out memory.
	panic(RestartSignal("Performing a full restart for efficiency."))
}

// Now we load the title cache file. We read it in as one huge lump.
// <TITLE_DELIM>title<RECORD_DELIM>tartsegment

var record_count int
var title_blob []byte
var title_size int64

func loadTitleFile() bool {
	var derr os.Error
	dat, derr = confparse.ParseFile(conf["dat_file"])
	if derr != nil {
		fmt.Println(derr)
		return false
	}

	curdbname = dat["dbname"]
	record_count, derr = strconv.Atoi(dat["rcount"])
	if derr != nil {
		fmt.Println(derr)
		return false
	}

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
		addr, errno := syscall.Mmap(
			fin.Fd(),
			0,
			int(title_size),
			syscall.PROT_READ,
			syscall.MAP_PRIVATE)
		if errno == 0 {
			title_blob = addr
			fmt.Printf("Successfully mmaped!\n")
		} else {
			fmt.Printf("Unable to mmap! error: '%v'\n", os.Errno(errno))
			dommap = false
		}
	}
	if !dommap {
		// Default: Load into memory.
		fmt.Printf("Loading titlecache.dat into Memory . . .\n")
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

// Compare a needle to an entry in the haystack, but do not create
// a new string just for it.
func caseCompare(needle, haystack []byte, hptr int64) int {
	var i int64
	l := int64(len(needle))
	for i = 0; i < l; i++ {
		if needle[i] != haystack[hptr+i] {
			if needle[i] > haystack[hptr+i] {
				return 1
			} else {
				return -1
			}
		}
	}
	if haystack[hptr+i] == RECORD_DELIM {
		return 0
	}
	return -1
}

// Binary search within a blob of unequal length strings.
func findTitleData(name string) (TitleData, bool) {
	// We limit to 100, just in case.
	searchesLeft := 100
	needle := []byte(name)

	min := int64(0)
	max := int64(title_size)

	minlen := int64(len(needle))

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

		// Go backwards to look for the TITLE_DELIM that signifies start of
		// record.
	record:
		for {
			if cur <= min {
				if cur <= min {
					// We may be very close, but searching the wrong way. Search forward,
					// now.
					cur = origcur
					for {
						if cur >= max {
							break search
						}
						if title_blob[cur] == TITLE_DELIM {
							break record
						}
						cur += 1
					}
				}
			}
			if title_blob[cur] == TITLE_DELIM {
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
			if title_blob[recordEnd] == RECORD_DELIM {
				break
			}
			recordEnd += 1
		}

		// Now we look for the <RECORD_DELIM>###(<TITLE_DELIM>|end) for the index.
		numStart := recordEnd + 1
		numEnd := numStart + 1
		for {
			if numEnd >= title_size {
				numEnd = title_size
				break
			}
			if title_blob[numEnd] == TITLE_DELIM {
				break
			}
			numEnd += 1
		}

		// Now compare
		ret := caseCompare(needle, title_blob, recordStart)

		// Did we find it? Did we?
		if ret == 0 {
			// We have the title.
			td := TitleData{}
			td.Title = string(title_blob[recordStart:recordEnd])
			td.Start, _ = strconv.Atoi(string(title_blob[numStart:numEnd]))
			return td, true
		}

		// Nope, let's divide and conquer.
		if ret > 0 {
			min = cur
		} else if ret < 0 {
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

	toFindb := []byte(toFind)
	for {
		bstr, berr := bzr.ReadBytes()
		if berr != nil {
			return ""
		}
		if bytes.Index(bstr, toFindb) >= 0 {
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
	Phrase                            string
	Results                           string
	ResultCount, StartingAt, EndingAt int
	PageNum, PageCount                int
}

func getTitleFromPos(haystack []byte, pos int) string {
	var i, end int
	for i = pos; i > 0 && haystack[i] != TITLE_DELIM; i -= 1 {
	}
	for end = i; end < len(haystack) && haystack[end] != RECORD_DELIM; end++ {
	}
	return string(haystack[i+1 : end])
}

// How we do searches:
//
// caseInsensitiveFinds() searches through haystack, which ideally is already
// properly bounded.
//
// First, it turns needle into both an upper case and lower case copy,
// so it can use both for quick reference. It discards all non-alphanumeric
// runes from needle so that "cslewis" will match "C. S. Lewis"
//
// Searching in the haystack also ignores non-alphanumeric runes.
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
		urunes = []int{}[:]

		i := 0
		for j := 0; j < len(tmp); {
			rune, cnt := utf8.DecodeRune(tmp[j:])
			j += cnt
			// Strip out all spaces.
			if !unicode.IsSpace(rune) {
				urunes = append(urunes, rune)
				i += 1
			}
		}
	}

	var lrunes []int
	if true {
		tmp := bytes.ToLower(needle)
		lrunes = []int{}[:]

		i := 0
		for j := 0; j < len(tmp); {
			rune, cnt := utf8.DecodeRune(tmp[j:])
			j += cnt
			// Strip out all spaces.
			if !unicode.IsSpace(rune) {
				lrunes = append(lrunes, rune)
				i += 1
			}
		}
	}

	if len(lrunes) < 1 {
		return
	}

	lc := lrunes[0]
	uc := urunes[0]
	n = len(lrunes)

	maxlen := len(haystack)

nextrecord:
	for i := 0; (i + n) < maxlen; {
		r, cnt := utf8.DecodeRune(haystack[i:])
		i += cnt

		// Check the first rune against either the lower or upper case needle
		// rune.
		if r == lc || r == uc {
			x := i
			var s int

			// If r is 0-9, then it could be we're looking at a record number in
			// the haystack. Make sure this doesn't happen.

			if r >= '0' && r <= '9' {
				// Skip over the next digits
				ptr := i
				for ; ptr < maxlen && haystack[ptr] >= '0' && haystack[ptr] <= '9'; ptr++ {
				}

				// If it ends at a TITLE_DELIM, then this is not a match.
				if ptr >= maxlen || haystack[ptr] == TITLE_DELIM {
					i = ptr
					continue nextrecord
				}
			}

			// Check the rest.
			for s = 1; s < n; s++ {
				// Skip over all non-alphanumerics.
				var r, cnt int
				for {
					if haystack[x] == RECORD_DELIM || haystack[x] == TITLE_DELIM {
						break
					}
					r, cnt = utf8.DecodeRune(haystack[x:])
					x += cnt
					if unicode.IsLetter(r) || unicode.IsDigit(r) {
						break
					}
				}
				if !(r == urunes[s] || r == lrunes[s]) {
					break
				}
			}
			if s >= n {
				cur := getTitleFromPos(haystack, i)
				if ignoreSearchRx == nil || !ignoreSearchRx.MatchString(cur) {
					results = append(results, cur)
				}
				for {
					if i > maxlen || haystack[i] == TITLE_DELIM {
						break
					}
					i += 1
				}
			}
		}
	}
}

func markRecent(uri string) {
	for _, i := range recentPages {
		if i == uri {
			return
		}
	}
	recentPages = append(recentPages, uri)
	if recentCount > 1 && len(recentPages) > recentCount {
		l := len(recentPages)
		recentPages = recentPages[l-recentCount : l]
	}
	// Put it all in the file.
	dfout, derr := os.OpenFile(conf["recents_file"], os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if derr != nil {
		fmt.Printf("Unable to create '%v': %v", conf["recents_file"], derr)
		return
	}
	defer dfout.Close()

	for _, i := range recentPages {
		fmt.Fprintf(dfout, "%v\n", i)
	}
}

type templateInfo struct {
	tpl   *template.Template
	mtime int64
	err   string
}

var templateCache = map[string]*templateInfo{}

func loadTemplate(tname string) *templateInfo {
	cached := templateCache[tname]

	if cached == nil {
		cached = &templateInfo{tpl: nil, mtime: 0, err: "Not loaded"}
	}

	// Check mtime.
	fin, err := os.Open(tname)
	if err != nil {
		cached.err = fmt.Sprintf("Unable to open '%s': '%v'", tname, err)
		return cached
	}
	defer fin.Close()
	stat, err := fin.Stat()
	if err != nil {
		cached.err = fmt.Sprintf("Unable to stat '%s': '%v'", tname, err)
		return cached
	}

	if stat.Mtime_ns <= cached.mtime {
		return cached
	}

	cached.tpl, err = template.ParseFile(tname)
	if err == nil {
		cached.err = ""
	} else {
		cached.err = fmt.Sprintf("Error while parsing '%s': '%v'", tname, err)
	}

	templateCache[tname] = cached
	return cached
}

func renderTemplate(tname string, data interface{}) (string, int) {
	cache := loadTemplate(tname)

	if cache.tpl == nil || cache.err != "" {
		return fmt.Sprintf("Error while loading template: %v", cache.err),
			http.StatusInternalServerError
	}

	buff := bytes.NewBuffer(nil)

	err := cache.tpl.Execute(buff, data)

	if err == nil {
		return buff.String(), http.StatusOK
	}
	return fmt.Sprintf("Error while executing template: %v", cache.err),
		http.StatusInternalServerError
}

func searchHandle(w http.ResponseWriter, req *http.Request) {
	// "/search/"
	pagetitle := getTitle(req.URL.Path[8:])
	startingAt := 0

	startPage := req.FormValue("p")
	if startPage != "" {
		pagenum, perr := strconv.Atoi(startPage)
		if perr == nil {
			pagenum = pagenum - 1
			// Bound pagenum.
			if pagenum < 0 {
				pagenum = 0
			}
			if pagenum > 1000 {
				pagenum = 1000
			}
			startingAt = pagenum * searchMaxResults
		}
	}

	go markRecent(req.URL.Path)

	// A watchdog for the goroutines.
	watchdog := make(chan []string)

	// Start all goroutine for searching.
	for i := 0; i < searchRoutines; i++ {
		go func(s, e int64, w chan []string) {
			caseInsensitiveFinds(title_blob[s:e], []byte(pagetitle), w)
		}(searchRanges[i].Start, searchRanges[i].End, watchdog)
	}

	// First results
	allresults := <-watchdog

	for i := 1; i < searchRoutines; i++ {
		additionalresults := <-watchdog
		allresults = append(allresults, additionalresults...)
	}

	// Sort results.
	//sort.Strings(allresults)
	searchlist(allresults).Sort()

	// Take the first searchMaxResults
	p := SearchPage{
		Phrase:      pagetitle,
		StartingAt:  startingAt + 1,
		ResultCount: len(allresults),
		PageNum:     (startingAt / searchMaxResults) + 1,
		PageCount:   (len(allresults) + (searchMaxResults - 1)) / searchMaxResults,
	}

	var results []string

	maxResultsLeft := len(allresults) - startingAt
	numResults := maxResultsLeft

	if maxResultsLeft > searchMaxResults {
		numResults = searchMaxResults
	} else if maxResultsLeft > 0 {
		numResults = maxResultsLeft
	} else {
		numResults = 0
	}

	if numResults > 0 {
		results = allresults[startingAt : startingAt+numResults]
	}
	p.EndingAt = startingAt + numResults

	p.Results = strings.Join(results, "|")

	page, status := renderTemplate(conf["search_template"], &p)
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(page)))

	w.WriteHeader(status)
	w.Write([]byte(page))
}

type WikiPage struct {
	Title string
	Body  string
}

func pageHandle(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	// "/wiki/"
	pagetitle := getTitle(req.URL.Path[6:])

	go markRecent(req.URL.Path)

	td, ok := findTitleData(pagetitle)

	if ok {
		p := WikiPage{Title: pagetitle, Body: readTitle(td)}
		page, status := renderTemplate(conf["wiki_template"], &p)
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(page)))

		w.WriteHeader(status)
		w.Write([]byte(page))
	} else {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "No such Wiki Page")
	}

}

func recentHandle(w http.ResponseWriter, req *http.Request) {
	// "/recent"
	x := strings.Join(recentPages, "\n")
	fmt.Fprintf(w, "%v\n", x)
}

// Prepare the globals needed for fast searching.
//
// type searchRange struct { Start, End int }
// var searchRanges []searchRange
//
// What this does is pre-split the db ('haystack') into approximately equal
// portions, bounded by TITLE_DELIM characters and the beginning and end of
// the titlecache file.
//
// A setup with a single searchRoutine would have Start = 1 and End = title_size
func prepSearchRoutines() {
	if conf["search_ignore_rx"] != "" {
		ignoreSearchRx = regexp.MustCompile(conf["search_ignore_rx"])
	} else {
		ignoreSearchRx = nil
	}

	if conf["search_routines"] != "" {
		var err os.Error
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

	if conf["search_max_results"] != "" {
		var err os.Error
		searchMaxResults, err = strconv.Atoi(conf["search_max_results"])
		if err != nil {
			fmt.Println("search_max_results: Unable to parse '%v' as integer: '%v'.\n",
				conf["search_max_results"], err)
			fmt.Println("search_max_results: Using default value.\n")
			searchMaxResults = 100
		} else if searchRoutines < 1 || searchRoutines > 64 {
			fmt.Println("search_max_results: Number '%v' Out of range (1-64).\n", searchRoutines)
		}
	}

	if searchRoutines > 1 {
		mult := title_size / int64(searchRoutines)
		searchRanges = make([]searchRange, searchRoutines)
		ptr := int64(0)
		for i := 0; i < searchRoutines; i++ {
			// Start at the end of the last one.
			searchRanges[i].Start = ptr
			ptr = mult * int64((i + 1))
			if ptr >= title_size {
				ptr = title_size
			} else {
				for {
					if title_blob[ptr] == TITLE_DELIM {
						break
					}
					if ptr < searchRanges[i].Start {
						fmt.Printf("Something is wrong with your titleCache.dat!\n")
						panic("Invalid titleCache.dat")
					}
					ptr--
				}
			}
			searchRanges[i].End = ptr
		}
	} else {
		searchRoutines = 1
		searchRanges = make([]searchRange, searchRoutines)
		searchRanges[0].Start = 0
		searchRanges[0].End = title_size
	}
}

// Load the recent_file, if it exists, and prepare for /recents
func prepRecents() {
	if conf["recent_count"] != "" {
		var err os.Error
		recentCount, err = strconv.Atoi(conf["recent_count"])
		if err != nil {
			fmt.Println("recent_count: Unable to parse '%v' as integer: '%v'.\n",
				conf["recent_count"], err)
			fmt.Println("recent_count: Using default value.\n")
			recentCount = 30
		} else if recentCount < 1 || recentCount > 1000 {
			fmt.Println("recent_count: Number '%v' Out of range (1-1000).\n", recentCount)
		}
	}

	// Load in the file.
	fin, err := os.Open(conf["recents_file"])
	if err != nil {
		fmt.Println(err)
		return
	}
	defer fin.Close()

	// Find out how big it is.
	stat, err := fin.Stat()
	if err != nil {
		fmt.Printf("Error while slurping in recents cache: '%v'\n", err)
	}
	recent_size := stat.Size

	recent_blob := make([]byte, recent_size, recent_size)

	nread, err := fin.Read(recent_blob)

	if err != nil && err != os.EOF {
		fmt.Printf("Error while slurping in recents cache: '%v'\n", err)
		return
	}
	if int64(nread) != recent_size || err != nil {
		fmt.Printf("Unable to read entire recents, only read %d/%d\n",
			nread, stat.Size)
		return
	}

	recentPages = append(recentPages, strings.Split(string(recent_blob), "\n")...)
}

func parseConfig(confname string) {
	fromfile, err := confparse.ParseFile(confname)
	if err != nil {
		fmt.Printf("Unable to read config file '%s'\n", confname)
		return
	}

	fmt.Printf("Read config file '%s'\n", confname)

	for key, value := range fromfile {
		if _, ok := conf[key]; !ok {
			fmt.Printf("Unknown config key: '%v'\n", key)
		} else {
			conf[key] = value
		}
	}
}

type GracefulError string
type RestartSignal string

func main() {
	// Defer this first to ensure cleanup gets done properly
	// 
	// Any error of type GracefulError is handled with an exit(1)
	// rather than by handing the user a backtrace.
	//
	// An error of type RestartSignal is technically not an error
	// but is the cleanest way to ensure no defers are skipped.
	defer func() {
		problem := recover()
		switch problem.(type) {
		case GracefulError:
			fmt.Println(problem)
			os.Exit(1)
		case RestartSignal:
			fmt.Println(problem)
			// Probably requires closing any fds still open
			// Will investigate later
			os.Exec(os.Args[0], os.Args, os.Envs)
			// If we're still here something went wrong.
			fmt.Printl("Couldn't restart. You'll have to restart manually.")
			os.Exit(0)
		default:
			panic(problem)
		}
	}()

	fmt.Println("Switching dir to", dirname(os.Args[0]))
	os.Chdir(dirname(os.Args[0]))

	parseConfig("bzwikipedia.conf")

	// Check for any new databases, including initial startup, and
	// perform pre-processing.
	performUpdates()

	// Load in the title cache
	if !loadTitleFile() {
		fmt.Println("Unable to read Title cache file: Invalid format?")
		return
	}
	prepSearchRoutines()
	prepRecents()

	fmt.Println("Loaded! Starting webserver . . .")

	// /wiki/... are pages.
	http.HandleFunc("/wiki/", pageHandle)
	// /search/ look for given text
	http.HandleFunc("/search/", searchHandle)
	// /recent, a list of recent searches
	http.HandleFunc("/recent", recentHandle)

	// Everything else is served from the web dir.
	http.Handle("/", http.FileServer(http.Dir(conf["web_dir"])))

	fmt.Printf("Forcing Go to use %d max threads.\n", searchRoutines)
	runtime.GOMAXPROCS(searchRoutines)

	fmt.Println("Starting Web server on port", conf["listen"])

	err := http.ListenAndServe(conf["listen"], nil)
	if err != nil {
		fmt.Println("Fatal error:", err.String())
	}
}
