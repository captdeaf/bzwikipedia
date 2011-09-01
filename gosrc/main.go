// main.go

package main

import (
	"bufio"
	"bytes"
	"compress/bzip2"
	"confparse"
	"exec"
	"fmt"
	"http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"template"
	"time"
)

// current db name, if extant.
var curdbname string

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
}

func basename(fp string) string {
	return filepath.Base(fp)
}

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

var versionrx = regexp.MustCompile("^version:([0-9]+)")
var dbnamerx = regexp.MustCompile("^dbname:(.*\\.xml\\.bz2)")

func needUpdate(recent string) bool {
	fin, err := os.Open(conf["dat_file"])
	var matches []string
	var cacheddbname string

	if err == nil {
		breader := bufio.NewReader(fin)
		line, err := breader.ReadString('\n')
		if err != nil {
			fmt.Println("Title file has invalid version.")
			return true
		}

		matches = versionrx.FindStringSubmatch(line)
		if matches == nil {
			fmt.Println("Title file has invalid version.")
			return true
		}

		line, err = breader.ReadString('\n')
		if err != nil {
			fmt.Println("Title file has invalid dbname.")
			return true
		}

		matches = dbnamerx.FindStringSubmatch(line)
		if matches == nil {
			fmt.Println("Title file has invalid format.")
			return true
		}

		cacheddbname = matches[1]
		if basename(cacheddbname) == basename(recent) {
			fmt.Println(recent, "matches cached database. No preparation needed.")
			return false
		}
	} else {
		fmt.Println("Title File doesn't exist.")
	}
	return true
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

func splitBz2File(recent string) {
	// Be user friendly: Alert the user and wait a few seconds."
	fmt.Println("I will be using bzip2recover to split", recent, "into many smaller files.")
	time.Sleep(3000000000)

	// Move the recent db over to the data dir since bzip2recover extracts
	// to the same directory the db exists in, and we don't want to pollute
	// drop_dir with the rec*.xml.bz2 files.
	newpath := filepath.Join(conf["data_dir"], basename(recent))
	os.Rename(recent, newpath)

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

	switch err {
	default:
		fmt.Println("err is:", err)
		panic("Unable to run bzip2recover? err is ")
	case os.ENOENT:
		//Maybe this should fmt and exit.
		//Does defer handle exit right?
		panic("bzip2recover not found. Giving up.")
	}
	bz2recover.Wait(0)
}

type SegmentedBzReader struct {
	index int
	bfin  *bufio.Reader
	cfin  *os.File
}

//
// This will sequentially read .bz2 files starting from a given index.
//
func NewBzReader(index int) *SegmentedBzReader {
	sbz := new(SegmentedBzReader)
	sbz.index = index
	sbz.bfin = nil
	sbz.cfin = nil

	sbz.OpenNext()
	return sbz
}

//
// Open rec<index>dbname.xml.bz2 for reading
//
func (sbz *SegmentedBzReader) OpenNext() {
	if sbz.cfin != nil {
		sbz.cfin.Close()
		sbz.cfin = nil
		sbz.bfin = nil
	}
	fn := fmt.Sprintf("%v/rec%05d%v", conf["data_dir"], sbz.index, curdbname)
	cfin, err := os.Open(fn)
	if err != nil {
		sbz.cfin = nil
		sbz.bfin = nil
	} else {
		sbz.cfin = cfin
		sbz.bfin = bufio.NewReader(bzip2.NewReader(cfin))
	}
}

func (sbz *SegmentedBzReader) ReadString() (string, os.Error) {
	if sbz.bfin == nil {
		return "", os.EOF
	}
	str, err := sbz.bfin.ReadString('\n')
	if err == nil {
		return str, nil
	}
	if err != os.EOF {
		fmt.Printf("Index %d: Non-EOF error!\n", sbz.index)
		fmt.Printf("str: '%v' err: '%v'\n", str, err)
		panic("Unrecoverable error")
	}

	sbz.index += 1
	sbz.OpenNext()

	// Last file?
	if err != nil || sbz.cfin == nil {
		return str, nil
	}

	nstr, nerr := sbz.bfin.ReadString('\n')

	str = fmt.Sprintf("%v%v", str, nstr)

	return str, nerr
}

func (sbz *SegmentedBzReader) Close() {
	sbz.cfin.Close()
	sbz.cfin = nil
	sbz.bfin = nil
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

	// Plop version and dbname into bzwikipedia.dat
	fmt.Fprintf(dfout, "version:1\n")
	fmt.Fprintf(dfout, "dbname:%v\n", curdbname)

	// Now read through all the bzip files looking for <title> bits.
	bzr := NewBzReader(1)

	// We print a notice every 100 chunks, just 'cuz it's more user friendly
	// to show _something_ going on.
	nextprint := 100

	// Total # of records, so that we know how big to make the title_map
	// when loading.
	record_count := 0

	titlerx := regexp.MustCompile("^ *<title>(.*)</title>")

	for {
		curindex := bzr.index
		if curindex >= nextprint {
			nextprint = curindex + 100
			fmt.Println("Reading chunk", curindex)
		}
		str, err := bzr.ReadString()
		if err == os.EOF {
			break
		}
		if err != nil {
			fmt.Printf("Error while reading chunk %v: %v\n", bzr.index, err)
			panic("Unrecoverable error.")
		}

		// This accounts for both "" and is a quick optimization.
		if len(str) < 10 {
			continue
		}

		var idx int

		for idx = 0; idx < len(str) && str[idx] == ' '; idx++ {
		}

		if (idx < len(str)) && (str[idx] == '<') && (str[idx+1] == 't') &&
			(str[idx+2] == 'i') && (str[idx+3] == 't') {
			matches := titlerx.FindStringSubmatch(str)
			if matches != nil {
				record_count++
				fmt.Fprintf(fout, "%v -- %d\n", matches[1], curindex)
			}
		}
	}
	fmt.Fprintf(dfout, "rcount:%v\n", record_count)

	return title_file_new, dat_file_new
}

////// Title file format:
// title -- startsegment

////// bzwikipedia.dat file format:
// version:1
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

	if !needUpdate(recent) {
		fmt.Println("Cache update not required.")
		return
	}

	// Clean out old files if we need 'em to be.
	cleanOldCache()

	// Turn the big old .xml.bz2 into a bunch of smaller .xml.bz2s
	splitBz2File(recent)

	curdbname = basename(recent)

	// Generate a new title file and dat file
	newtitlefile, newdatfile := generateNewTitleFile()

	// Rename them to the actual title and dat file
	os.Rename(newtitlefile, conf["title_file"])
	os.Rename(newdatfile, conf["dat_file"])

	// We have now completed pre-processing! Yay!
}

type TitleData struct {
	Title string
	Start int
}

var title_map map[string]TitleData
var record_count int

func loadTitleFile() bool {
	// Open the dat file.
	dfin, derr := os.Open(conf["dat_file"])
	if derr != nil {
		fmt.Println(derr)
		return false
	}
	defer dfin.Close()

	bdfin := bufio.NewReader(dfin)

	kvrx := regexp.MustCompile("^([a-z]+):(.*)\\n$")

	var str string

	if str, derr = bdfin.ReadString('\n'); derr != nil {
		return false
	}
	matches := kvrx.FindStringSubmatch(str)

	if matches == nil || matches[1] != "version" {
		return false
	}

	if str, derr = bdfin.ReadString('\n'); derr != nil {
		return false
	}
	matches = kvrx.FindStringSubmatch(str)

	if matches == nil || matches[1] != "dbname" {
		return false
	}
	curdbname = matches[2]

	if str, derr = bdfin.ReadString('\n'); derr != nil {
		return false
	}
	matches = kvrx.FindStringSubmatch(str)

	if matches == nil || matches[1] != "rcount" {
		return false
	}
	record_count, derr = strconv.Atoi(matches[2])
	if derr != nil {
		fmt.Println(derr)
		return false
	}

	fmt.Printf("DB '%s': Loading %d records.\n",
		curdbname, record_count)

	// This is one WHOPPING big map!
	title_map = make(map[string]TitleData, record_count+1)

	// Read in the titles.
	fin, err := os.Open(conf["title_file"])
	if err != nil {
		fmt.Println(err)
		return false
	}
	defer fin.Close()

	bfin := bufio.NewReader(fin)
	// recordrx := regexp.MustCompile("^(.*) -- ([0-9]+)\n$")

	for i := 0; i < record_count; i++ {
		str, err := bfin.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			return false
		}
		// matches = recordrx.FindStringSubmatch(str)
		res := strings.SplitN(str, " -- ", 2)
		// Since Atoi requires a chomp()'d string, and that's too slow:
		// start, err := strconv.Atoi(res[1])
		// if err != nil { fmt.Println(err); return false }
		start := 0
		for x := 0; x < len(res[1]) && res[1][x] >= 48; x++ {
			start *= 10
			start += int(res[1][x] - '0')
		}

		if i%100000 == 0 {
			fmt.Printf("Loaded %d (%d%% complete)\n", i, (i*100)/record_count)
		}
		td := TitleData{
			Title: res[0],
			Start: start,
		}
		title_map[td.Title] = td
	}

	return true
}

var wholetextrx = regexp.MustCompile("<text[^>]*>(.*)</text>")
var starttextrx = regexp.MustCompile("<text[^>]*>(.*)")
var endtextrx = regexp.MustCompile("(.*)</text>")

func readTitle(td TitleData) string {
	var str string
	var err os.Error

	toFind := fmt.Sprintf("<title>%s</title>", td.Title)

	// Start looking for the title.
	bzr := NewBzReader(td.Start)

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

var SearchTemplate *template.Template

func searchHandle(w http.ResponseWriter, req *http.Request) {
	// "/search/"
	pagetitle := getTitle(req.URL.Path[8:])
	if len(pagetitle) < 4 {
		fmt.Fprintf(w, "Search phrase too small for now.")
		return
	}

	ltitle := strings.ToLower(pagetitle)

	allResults := []string{}
	results := allResults[:]

	// Search all keys
	for key, _ := range title_map {
		if strings.Contains(strings.ToLower(key), ltitle) {
			results = append(results, key)
		}
	}

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

	td, ok := title_map[pagetitle]

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
		if conf[key] == "" {
			fmt.Printf("Unknown config key: '%v'\n", key)
		} else {
			conf[key] = value
		}
	}
}

func main() {
	fmt.Println("Switching dir to", dirname(os.Args[0]))
	os.Chdir(dirname(os.Args[0]))

	parseConfig("bzwikipedia.conf")

	// Load the templates first.
	SearchTemplate = template.MustParseFile(conf["search_template"], nil)
	WikiTemplate = template.MustParseFile(conf["wiki_template"], nil)

	// Check for any new databases, including initial startup, and
	// perform pre-processing.
	performUpdates()

	// Create the title_map variable
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
