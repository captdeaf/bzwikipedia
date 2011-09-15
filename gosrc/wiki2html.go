// wiki2html.go
//
// Uses: For converting from Wikimedia-style markup to HTML.
//
// The only function of note in here that you should use is:
//
// Wiki2HTML(input string) (string template, []string references)
//
// It doesn't currently support templates, but it will!

package wiki2html

import (
	"fmt"
	"regexp"
	"strings"
)

type markupInfo struct {
	depth    int
	refCount int
	refNames map[string]int
	refs     []string
	inCode   bool
}

type token struct {
	IsToken bool
	Val     string
}

type nsHandler interface {
	Handle(namespace, page, title string) string
}

type nsIgnorable bool

const nsIgnore = nsIgnorable(false)

func (n nsIgnorable) Handle(namespace, page, title string) string {
	return ""
}

type nsPrefix string

func (n nsPrefix) Handle(namespace, page, title string) string {
	link := fmt.Sprintf("<a class=\"external\" href=\"%s%s\">%s</a>", n, page, title)
	return link
}

type nsFunction func(namespace, page, title string) string

func (n nsFunction) Handle(namespace, page, title string) string {
	return n(namespace, page, title)
}

func nolinkHandler(namespace, page, title string) string {
	notlink := fmt.Sprintf("<span style=\"border-bottom:1px dotted\">%s</span>", title)
	return notlink
}

var nsNoLink = nsFunction(nolinkHandler)

// string -> handler mapping. Strings must be lowercase.
// Due to data volume probably deserves its own file :/
var nsMap = map[string]nsHandler{
	"de":           nsIgnore,
	"fr":           nsIgnore,
	"it":           nsIgnore,
	"pl":           nsIgnore,
	"es":           nsIgnore,
	"ja":           nsIgnore,
	"ru":           nsIgnore,
	"nl":           nsIgnore,
	"pt":           nsIgnore,
	"sv":           nsIgnore,
	"zh":           nsIgnore,
	"ca":           nsIgnore,
	"uk":           nsIgnore,
	"no":           nsIgnore,
	"fi":           nsIgnore,
	"vi":           nsIgnore,
	"cs":           nsIgnore,
	"hu":           nsIgnore,
	"ko":           nsIgnore,
	"tr":           nsIgnore,
	"id":           nsIgnore,
	"ro":           nsIgnore,
	"fa":           nsIgnore,
	"ar":           nsIgnore,
	"da":           nsIgnore,
	"eo":           nsIgnore,
	"sr":           nsIgnore,
	"lt":           nsIgnore,
	"sk":           nsIgnore,
	"he":           nsIgnore,
	"ms":           nsIgnore,
	"bg":           nsIgnore,
	"sl":           nsIgnore,
	"vo":           nsIgnore,
	"eu":           nsIgnore,
	"war":          nsIgnore,
	"hr":           nsIgnore,
	"hi":           nsIgnore,
	"et":           nsIgnore,
	"az":           nsIgnore,
	"kk":           nsIgnore,
	"gl":           nsIgnore,
	"simple":       nsIgnore,
	"nn":           nsIgnore,
	"new":          nsIgnore,
	"th":           nsIgnore,
	"el":           nsIgnore,
	"roa-rup":      nsIgnore,
	"la":           nsIgnore,
	"tl":           nsIgnore,
	"ht":           nsIgnore,
	"ka":           nsIgnore,
	"mk":           nsIgnore,
	"te":           nsIgnore,
	"sh":           nsIgnore,
	"pms":          nsIgnore,
	"ceb":          nsIgnore,
	"be-x-old":     nsIgnore,
	"br":           nsIgnore,
	"ta":           nsIgnore,
	"jv":           nsIgnore,
	"lv":           nsIgnore,
	"mr":           nsIgnore,
	"sq":           nsIgnore,
	"cy":           nsIgnore,
	"lb":           nsIgnore,
	"be":           nsIgnore,
	"is":           nsIgnore,
	"bs":           nsIgnore,
	"oc":           nsIgnore,
	"yo":           nsIgnore,
	"an":           nsIgnore,
	"bpy":          nsIgnore,
	"mg":           nsIgnore,
	"bn":           nsIgnore,
	"io":           nsIgnore,
	"sw":           nsIgnore,
	"fy":           nsIgnore,
	"lmo":          nsIgnore,
	"gu":           nsIgnore,
	"ml":           nsIgnore,
	"pnb":          nsIgnore,
	"af":           nsIgnore,
	"nds":          nsIgnore,
	"scn":          nsIgnore,
	"ur":           nsIgnore,
	"qu":           nsIgnore,
	"ku":           nsIgnore,
	"zh-yue":       nsIgnore,
	"ne":           nsIgnore,
	"hy":           nsIgnore,
	"ast":          nsIgnore,
	"su":           nsIgnore,
	"nap":          nsIgnore,
	"diq":          nsIgnore,
	"ga":           nsIgnore,
	"cv":           nsIgnore,
	"bat-smg":      nsIgnore,
	"tt":           nsIgnore,
	"wa":           nsIgnore,
	"am":           nsIgnore,
	"kn":           nsIgnore,
	"als":          nsIgnore,
	"tg":           nsIgnore,
	"zh-min-nan":   nsIgnore,
	"bug":          nsIgnore,
	"vec":          nsIgnore,
	"roa-tara":     nsIgnore,
	"yi":           nsIgnore,
	"gd":           nsIgnore,
	"arz":          nsIgnore,
	"os":           nsIgnore,
	"ia":           nsIgnore,
	"sah":          nsIgnore,
	"uz":           nsIgnore,
	"pam":          nsIgnore,
	"my":           nsIgnore,
	"sco":          nsIgnore,
	"hsb":          nsIgnore,
	"mi":           nsIgnore,
	"li":           nsIgnore,
	"nah":          nsIgnore,
	"mn":           nsIgnore,
	"co":           nsIgnore,
	"gan":          nsIgnore,
	"glk":          nsIgnore,
	"ba":           nsIgnore,
	"si":           nsIgnore,
	"sa":           nsIgnore,
	"hif":          nsIgnore,
	"bcl":          nsIgnore,
	"fo":           nsIgnore,
	"mrj":          nsIgnore,
	"fiu-vro":      nsIgnore,
	"bar":          nsIgnore,
	"ckb":          nsIgnore,
	"nds-nl":       nsIgnore,
	"vls":          nsIgnore,
	"tk":           nsIgnore,
	"gv":           nsIgnore,
	"ilo":          nsIgnore,
	"se":           nsIgnore,
	"map-bms":      nsIgnore,
	"dv":           nsIgnore,
	"nrm":          nsIgnore,
	"pag":          nsIgnore,
	"rm":           nsIgnore,
	"mzn":          nsIgnore,
	"bo":           nsIgnore,
	"udm":          nsIgnore,
	"ps":           nsIgnore,
	"pa":           nsIgnore,
	"fur":          nsIgnore,
	"km":           nsIgnore,
	"wuu":          nsIgnore,
	"mt":           nsIgnore,
	"csb":          nsIgnore,
	"ug":           nsIgnore,
	"lij":          nsIgnore,
	"koi":          nsIgnore,
	"pi":           nsIgnore,
	"ang":          nsIgnore,
	"bh":           nsIgnore,
	"kv":           nsIgnore,
	"sc":           nsIgnore,
	"rue":          nsIgnore,
	"lad":          nsIgnore,
	"nov":          nsIgnore,
	"zh-classical": nsIgnore,
	"mhr":          nsIgnore,
	"ksh":          nsIgnore,
	"cbk-zam":      nsIgnore,
	"hak":          nsIgnore,
	"so":           nsIgnore,
	"kw":           nsIgnore,
	"frp":          nsIgnore,
	"nv":           nsIgnore,
	"szl":          nsIgnore,
	"ext":          nsIgnore,
	"stq":          nsIgnore,
	"ie":           nsIgnore,
	"xal":          nsIgnore,
	"rw":           nsIgnore,
	"haw":          nsIgnore,
	"ln":           nsIgnore,
	"pdc":          nsIgnore,
	"ky":           nsIgnore,
	"pcd":          nsIgnore,
	"pfl":          nsIgnore,
	"krc":          nsIgnore,
	"to":           nsIgnore,
	"or":           nsIgnore,
	"crh":          nsIgnore,
	"ace":          nsIgnore,
	"eml":          nsIgnore,
	"myv":          nsIgnore,
	"gn":           nsIgnore,
	"frr":          nsIgnore,
	"ay":           nsIgnore,
	"arc":          nsIgnore,
	"ce":           nsIgnore,
	"kl":           nsIgnore,
	"pap":          nsIgnore,
	"bjn":          nsIgnore,
	"lbe":          nsIgnore,
	"jbo":          nsIgnore,
	"wo":           nsIgnore,
	"tpi":          nsIgnore,
	"mdf":          nsIgnore,
	"av":           nsIgnore,
	"kab":          nsIgnore,
	"gag":          nsIgnore,
	"ty":           nsIgnore,
	"zea":          nsIgnore,
	"srn":          nsIgnore,
	"dsb":          nsIgnore,
	"lo":           nsIgnore,
	"xmf":          nsIgnore,
	"ab":           nsIgnore,
	"ig":           nsIgnore,
	"na":           nsIgnore,
	"as":           nsIgnore,
	"tet":          nsIgnore,
	"kg":           nsIgnore,
	"mwl":          nsIgnore,
	"ltg":          nsIgnore,
	"kaa":          nsIgnore,
	"rmy":          nsIgnore,
	"cu":           nsIgnore,
	"kbd":          nsIgnore,
	"sm":           nsIgnore,
	"mo":           nsIgnore,
	"sd":           nsIgnore,
	"bm":           nsIgnore,
	"bi":           nsIgnore,
	"ik":           nsIgnore,
	"ss":           nsIgnore,
	"iu":           nsIgnore,
	"pih":          nsIgnore,
	"ks":           nsIgnore,
	"pnt":          nsIgnore,
	"za":           nsIgnore,
	"chr":          nsIgnore,
	"cdo":          nsIgnore,
	"ee":           nsIgnore,
	"got":          nsIgnore,
	"ha":           nsIgnore,
	"ti":           nsIgnore,
	"bxr":          nsIgnore,
	"sn":           nsIgnore,
	"om":           nsIgnore,
	"zu":           nsIgnore,
	"ve":           nsIgnore,
	"ts":           nsIgnore,
	"rn":           nsIgnore,
	"sg":           nsIgnore,
	"cr":           nsIgnore,
	"dz":           nsIgnore,
	"tum":          nsIgnore,
	"lg":           nsIgnore,
	"ch":           nsIgnore,
	"fj":           nsIgnore,
	"ny":           nsIgnore,
	"ff":           nsIgnore,
	"xh":           nsIgnore,
	"st":           nsIgnore,
	"tn":           nsIgnore,
	"chy":          nsIgnore,
	"ki":           nsIgnore,
	"ak":           nsIgnore,
	"tw":           nsIgnore,
	"ng":           nsIgnore,
	"ii":           nsIgnore,
	"cho":          nsIgnore,
	"mh":           nsIgnore,
	"aa":           nsIgnore,
	"kj":           nsIgnore,
	"ho":           nsIgnore,
	"mus":          nsIgnore,
	"kr":           nsIgnore,
	"hz":           nsIgnore,
	"wiktionary":   nsPrefix("http://en.wiktionary.org/wiki/"),
	"wikiquote":    nsPrefix("http://en.wikiquote.org/wiki/"),
	"file":         nsNoLink,
	"category":     nsNoLink,
	"mediawiki":    nsNoLink,
	"templates":    nsNoLink,
	"portal":       nsNoLink,
	"special":      nsNoLink,
	"talk":         nsNoLink,
	"wikipedia":    nsNoLink,
}

var entityFinds = regexp.MustCompile("<|>|&")

func unparseEntities(input string) string {
	return entityFinds.ReplaceAllStringFunc(input, func(what string) string {
		switch what {
		case "&":
			return "&amp;"
		case ">":
			return "&gt;"
		case "<":
			return "&lt;"
		}
		return what
	})
}

var matchuri = regexp.MustCompile("(http|https|ftp)://[^ \\t\\n]*(\\.[^ \\t\\n\\.]*)*")

func parsePlainText(input string) string {
	input = unparseEntities(input)

	return matchuri.ReplaceAllStringFunc(input, func(what string) string {
		return fmt.Sprintf("<a href=\"%s\">%s</a>", what, what)
	})
}

var entityReplace = regexp.MustCompile("&(#?[a-z0-9]+);")

func parseEntities(input string) string {
	return entityReplace.ReplaceAllStringFunc(input, func(what string) string {
		switch what {
		case "&lt;":
			return "<"
		case "&gt;":
			return ">"
		case "&amp;":
			return "&"
		case "&#93;":
			return "]"
		case "&#92;":
			return "\\"
		case "&#91;":
			return "["
		case "&quot;":
			return "\""
		}
		// TODO: Handle "all" escape codes. But as this is doubly encoded (for
		// XML) and UTF-8, that might not be necessary. Also, Wikipedia is
		// _really_ inconsistent about their escaping.
		return what
	})
}

var allTokens = []string{
  "\\n\\*|\\n#|\\n",            // Lists
  "\\{\\{|\\}\\}",              // Templates
  "\\[|\\]",                    // Internal and external links.
  "'''''|'''|''",               // Bold+italic
  "=====|====|===|==",          // Headings
  "<source[^>]*>|</source>",    // Source code
  "<ref[^>]*>|</ref>",          // References
  "<code[^>]*>|</code>",        // Code examples
  "<nowiki>|</nowiki>",         // Nowiki: Stuff inside is _not_ evaluated.
  "<table[^>]*>|<tr[^>]*>|<td[^>]*>", // Tables
  "</table>|</tr>|</td>",             // Tables
  "<pre>|</pre>|<tt>|</tt>",    // raw HTML
  "<span[^>]*>|<br[^>]*>",      // raw HTML
}

var tokenizer = regexp.MustCompile(strings.Join(allTokens,"|"))

func tokenize(input []byte) []token {
	// Find the location of all known tokens.
	allIndexes := tokenizer.FindAllIndex(input, -1)

	count := 0
	lastIndex := 0

	for i := 0; i < len(allIndexes); i++ {
		// Any leading text?
		if allIndexes[i][0] > lastIndex {
			count++
		}
		// This token
		count++
		lastIndex = allIndexes[i][1]
	}
	// Any trailing text?
	if lastIndex < len(input) {
		count++
	}

	allTokens := make([]token, count, count)

	j := 0
	lastIndex = 0
	for i := 0; i < len(allIndexes); i++ {
		if allIndexes[i][0] > lastIndex {
			allTokens[j] = token{
				IsToken: false,
				Val:     string(input[lastIndex:allIndexes[i][0]]),
			}
			j++
		}
		// This token
		allTokens[j] = token{
			IsToken: true,
			Val:     string(input[allIndexes[i][0]:allIndexes[i][1]]),
		}
		j++
		lastIndex = allIndexes[i][1]
	}
	if lastIndex < len(input) {
		allTokens[j] = token{
			IsToken: false,
			Val:     string(input[lastIndex:len(input)]),
		}
	}

	return allTokens
}

func renderTemplate(tname string, namedArgs map[string]string, argv []string) string {
	lname := strings.ToLower(tname)
	content := strings.Join(argv, " ")
	switch lname {
	case "as of":
		return fmt.Sprintf(
			"%s %s",
			tname, content)
	case "see also":
		return fmt.Sprintf(
			"(%s: <i><a href=\"/wiki/%s\">%s</a></i>)",
			tname, content, content)
	case "cquote":
		return fmt.Sprintf(
			"<blockquote>%s</blockquote>",
			content)
	case "sic":
		return "<span class=\"sic\">[<a href=\"/wiki/Sic\">Sic</a>]</span>"
	case "refbegin":
		return "<ol>"
	case "refend":
		return "</ol>"
	case "citation":
		return fmt.Sprintf(
			"\"%s\" by %s, %s",
			namedArgs["title"], namedArgs["last1"], namedArgs["first1"])
	}
	return "FOO"
}

var namedArg = regexp.MustCompile("^ *([a-zA-Z0-9]+) *= *(.*) *$")
// {{ ... }}
func parseTemplate(input []byte, tokens []token, i int, mi *markupInfo) (string, int) {
	// fmt.Printf("Entering {{...\n")
	// defer fmt.Printf("Leaving }}\n")
	body, eidx := parseGeneral(input, tokens, i+1, []string{"}}"}, mi)
	args := strings.Split(body, "|")
	tname := strings.TrimSpace(args[0])
	namedArgs := map[string]string{}
	positionalArgs := []string{}

	for i := 1; i < len(args); i++ {
		arg := args[i]
		if strings.Contains(arg, "=") {
			matches := namedArg.FindStringSubmatch(arg)
			if matches != nil {
				namedArgs[strings.ToLower(matches[1])] = matches[2]
				continue
			}
		}
		positionalArgs = append(positionalArgs, arg)
	}

	result := renderTemplate(tname, namedArgs, positionalArgs)
	if result == "FOO" {
		return fmt.Sprintf("{{%s}}", body), eidx
	}
	return result, eidx
}

func parseNowiki(input []byte, tokens []token, i int, mi *markupInfo) (string, int) {
        result := []string{}
        for i = i + 1; i < len(tokens) && tokens[i].Val != "</nowiki>"; i++ {
          result = append(result, tokens[i].Val)
        }
	return strings.Join(result, ""), i
}

func parseExternalLink(input []byte, tokens []token, i int, mi *markupInfo) (string, int) {
	// fmt.Printf("Entering [...\n")
	// defer fmt.Printf("Leaving ]\n")
	// We only recurse if it looks like we're followed by an http.
	if len(tokens) > (i + 1) {
		if len(tokens[i+1].Val) < 7 || tokens[i+1].Val[0:7] != "http://" {
			return "[", i
		}
	}
	body, eidx := parseGeneral(input, tokens, i+1, []string{"]"}, mi)
	args := strings.SplitN(body, " ", 2)
	var title string
	page := args[0]
	if len(args) > 1 {
		title = args[1]
	} else {
		title = page
	}
	link := fmt.Sprintf("<a class=\"external\" href=\"%s\">%s</a>", page, title)
	return link, eidx
}

// [[ ... ]]
func parseInternalLink(input []byte, tokens []token, i int, mi *markupInfo) (string, int) {
	// fmt.Printf("Entering [[...\n")
	// defer fmt.Printf("Leaving ]]\n")

	// Internal link won't have any markup inside of it. At least, it better not!
	if len(tokens) < (i+2) || tokens[i+2].Val != "]" || tokens[i+3].Val != "]" {
		return "[[", i
	}

	body, eidx := parseGeneral(input, tokens, i+1, []string{"]", "]"}, mi)
	args := strings.SplitN(body, "|", 2)
	var title string
	page := args[0]
	if len(args) > 1 {
		title = args[1]
	} else {
		title = page
	}

	// Right now this will pretty much guarantee an unusable link with
	// language wiki links.
	// Not really a problem as they would be unusable without namespace
	// handling anyway, and will certainly lead to pages outside of
	// what we have. Still better to print a broken link than a gap.
	leadingColon := false
	if page[0] == ':' {
		leadingColon = true
		page = page[1:]
	}

	var namespace string
	if strings.Contains(page, ":") {
		subargs := strings.SplitN(page, ":", 2)
		namespace = subargs[0]
		newPage := subargs[1]
		namespace = strings.ToLower(namespace)
		handler := nsMap[namespace]

		if handler != nil && !(leadingColon && handler == nsIgnore) {
			instead := handler.Handle(namespace, newPage, title)
			return instead, eidx
		}
	}

	link := fmt.Sprintf("<a class=\"internal\" href=\"/wiki/%s\">%s</a>", page, title)
	return link, eidx
}

// <ref> ... </ref>
func parseReference(input []byte, tokens []token, i int, mi *markupInfo) (string, int) {
	start := i
	// fmt.Printf("Entering %s...\n", tokens[start].Val)
	// defer fmt.Printf("Leaving </ref>\n")

	// Now we need to find out if we are <ref>...</ref>, <ref name="...">..</ref>
	// or <ref name="..." />
	ref := tokens[start].Val

	// Check if we're a /. For now, it's an empty link.
	if strings.Index(ref, "/") >= 0 {
		return "", i
	}

	// Parse the reference body. 
	body, eidx := parseGeneral(input, tokens, i+1, []string{"</ref>"}, mi)
	mi.refCount++
	link := fmt.Sprintf("<a href=\"#ref%d\">[%d]</a>", mi.refCount, mi.refCount)
	mi.refs = append(mi.refs, fmt.Sprintf("<a name=\"ref%d\"></a>%s", mi.refCount, body))
	return link, eidx
}

// == foo ==
func parseHeader(input []byte, tokens []token, i int, mi *markupInfo) (string, int) {
	start := i
	x := len(tokens[start].Val)
	// fmt.Printf("Entering %s...\n", tokens[start].Val)
	// defer fmt.Printf("Leaving %s\n", tokens[start].Val)

	if len(tokens) < (i+2) || tokens[i+2].Val != tokens[start].Val {
		return tokens[start].Val, i
	}

	body, eidx := parseGeneral(input, tokens, i+1, []string{tokens[start].Val}, mi)

	return fmt.Sprintf("<h%d>%s</h%d>", x, body, x), eidx
}

// <pre>...</pre>, <tt>...</tt>, etc
func parseHtml(input []byte, tokens []token, i int, mi *markupInfo, end string) (string, int) {
	return fmt.Sprintf("%s", tokens[i].Val), i
}

// ''''' ... '''''
func parseMarkup(input []byte, tokens []token, i int, mi *markupInfo) (string, int) {
	start := i
	x := len(tokens[start].Val)
	// fmt.Printf("Entering %s...\n", tokens[start].Val)
	// defer fmt.Printf("Leaving %s\n", tokens[start].Val)

	if len(tokens) < (i+2) || tokens[i+2].Val != tokens[start].Val {
		return tokens[start].Val, i
	}

	body, eidx := parseGeneral(input, tokens, i+1, []string{tokens[start].Val}, mi)

	switch x {
	case 2:
		return fmt.Sprintf("<i>%s</i>", body), eidx
	case 3:
		return fmt.Sprintf("<b>%s</b>", body), eidx
	case 5:
		return fmt.Sprintf("<b><i>%s</i></b>", body), eidx
	}
	return fmt.Sprintf("<span style=\"color:yellow;\">%s</span>", body), eidx
}

// ...
func parseCode(input []byte, tokens []token, i int, mi *markupInfo, end string) (string, int) {
	// fmt.Printf("Entering %s...\n", tokens[i].Val)
	// defer fmt.Printf("Leaving %s\n", end)

	oldinCode := mi.inCode
	mi.inCode = true
	defer func() { mi.inCode = oldinCode }()

	body, eidx := parseGeneral(input, tokens, i+1, []string{end}, mi)

	if end == "</code>" {
		return fmt.Sprintf("<tt>%s</tt>", body), eidx
	}
	return fmt.Sprintf("<pre>%s</pre>", body), eidx
}

func doesListContinue(tokens []token, ltype string, i int) bool {
	for {
		i++
		if i >= len(tokens) {
			return false
		}
		if tokens[i].IsToken {
			switch tokens[i].Val {
			case ltype:
				return true
			case "\n#":
				return false
			case "\n*":
				return false
			case "\n":
				return false
			}
		}
	}
	return false
}

// Token parsers return the string value of their contents, and the next index
// to look at.
//
// parseGeneral is the overall one, and should be called by all the rest
// to recurse. "endtokens" is what ends the tokens for the caller.
// If endtokens is nil, then parseGeneral parses all the tokens.
func parseGeneral(input []byte, tokens []token, start int, endtokens []string, mi *markupInfo) (string, int) {
	mi.depth++
	defer func() {
		mi.depth--
	}()
	listType := ""
	i := start
	results := []string{}
	for {
		if i >= len(tokens) {
			break
		}
		if tokens[i].IsToken {
			if len(endtokens) > 0 && (i+len(endtokens)) <= len(tokens) {
				doret := true
				var j int
				for j = 0; j < len(endtokens); j++ {
					if tokens[i+j].Val != endtokens[j] {
						doret = false
						break
					}
				}
				if doret {
					return strings.Join(results, ""), i + len(endtokens) - 1
				}
			}
			switch {
			case tokens[i].Val == "\n":
				if listType != "" {
					results = append(results, fmt.Sprintf("</%s>", listType))
					listType = ""
				}
				if (i+1) < len(tokens) && tokens[i+1].IsToken && tokens[i+1].Val == "\n" {
					if mi.inCode {
						results = append(results, "\n\n")
					} else {
						results = append(results, "\n<br />\n<br />")
					}
                                        // Skip over successive newlines.
					for i++; i < len(tokens) && tokens[i].Val == "\n"; i++ {}
                                        i--
				} else {
					results = append(results, "\n")
				}
			case tokens[i].Val == "\n*":
				if !mi.inCode && listType != "ul" && doesListContinue(tokens, "\n*", i) {
					results = append(results, "<ul>")
					listType = "ul"
				}
				if listType != "" {
					results = append(results, "<li>")
				} else {
					results = append(results, tokens[i].Val)
				}
			case tokens[i].Val == "\n#":
				if !mi.inCode && listType != "ol" && doesListContinue(tokens, "\n#", i) {
					results = append(results, "<ol>")
					listType = "ol"
				}
				if listType != "" {
					results = append(results, "<li>")
				} else {
					results = append(results, tokens[i].Val)
				}
			case tokens[i].Val == "{{":
				body, eidx := parseTemplate(input, tokens, i, mi)
				results = append(results, body)
				i = eidx
			case len(tokens) > (i+1) && tokens[i].Val == "[" && tokens[i+1].Val == "[":
				body, eidx := parseInternalLink(input, tokens, i+1, mi)
				results = append(results, body)
				i = eidx
			case tokens[i].Val == "[":
				body, eidx := parseExternalLink(input, tokens, i, mi)
				results = append(results, body)
				i = eidx
			case tokens[i].Val[0] == '\'':
				body, eidx := parseMarkup(input, tokens, i, mi)
				results = append(results, body)
				i = eidx
			case tokens[i].Val[0] == '=':
				body, eidx := parseHeader(input, tokens, i, mi)
				results = append(results, body)
				i = eidx
			case len(tokens[i].Val) > 5 && tokens[i].Val[0:5] == "<code":
				body, eidx := parseCode(input, tokens, i, mi, "</code>")
				results = append(results, body)
				i = eidx
			case len(tokens[i].Val) > 7 && tokens[i].Val[0:7] == "<source":
				body, eidx := parseCode(input, tokens, i, mi, "</source>")
				results = append(results, body)
				i = eidx
			case len(tokens[i].Val) > 4 && tokens[i].Val[0:4] == "<ref":
				body, eidx := parseReference(input, tokens, i, mi)
				results = append(results, body)
				i = eidx
			case len(tokens[i].Val) > 7 && tokens[i].Val[0:7] == "<nowiki":
				body, eidx := parseNowiki(input, tokens, i, mi)
				results = append(results, body)
				i = eidx
			// The last case for html tags: <.*>, including pre, /pre, etc.
                        case len(tokens[i].Val) > 1 && tokens[i].Val[0:1] == "<":
				body, eidx := parseHtml(input, tokens, i, mi, "</pre>")
				results = append(results, body)
				i = eidx
			case tokens[i].Val == "]":
				// This happens a lot. No biggie.
				results = append(results, "]")
			default:
				if endtokens != nil {
					fmt.Printf("Don't know what to do with token \"%s\". endtokens is \"%v\"\n", tokens[i].Val, endtokens)
					fmt.Printf("Tokens[i].Val is: %s\n", tokens[i].Val)
					fmt.Printf("Tokens[i-1].Val is: %s\n", tokens[i-1].Val)
					fmt.Printf("Tokens[i-2].Val is: %s\n", tokens[i-2].Val)
					fmt.Printf("Tokens[i-3].Val is: %s\n", tokens[i-3].Val)
					fmt.Printf("Tokens[i-4].Val is: %s\n", tokens[i-4].Val)
					fmt.Printf("  Start is: \"%s\"\n", tokens[start].Val)
					fmt.Printf("  Opener was: %s\n", tokens[start-1].Val)
					fmt.Printf("  Pre-Opener was: %s\n", tokens[start-2].Val)
				} else {
					fmt.Printf("Don't know what to do with token '%s'. No endtokens\n", tokens[i].Val)
				}
				results = append(results, unparseEntities(tokens[i].Val))
			}
		} else {
			results = append(results, parsePlainText(string(tokens[i].Val)))
		}
		i += 1
	}
	return strings.Join(results, ""), i
}

func Wiki2HTML(input string) (string, []string) {
	// Screwy wikipedia doesn't know its own entities?
	// I got &amp;#93; that was supposed to be a closing ] to a [-tag!
	input = parseEntities(parseEntities(input))
	binput := []byte(input)
	tokens := tokenize(binput)
	mi := markupInfo{
		depth: 0,
	}
	res, _ := parseGeneral(binput, tokens, 0, nil, &mi)
	return res, mi.refs
}
