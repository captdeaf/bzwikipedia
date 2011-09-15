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
  depth int
  refCount int
  refNames map[string] int
  refs []string
}

type token struct {
  IsToken bool
  Val string
}

var entityFinds = regexp.MustCompile("<|>|&")

func unparseEntities(input string) string {
  return entityFinds.ReplaceAllStringFunc(input, func(what string) string {
    switch what {
    case "&": return "&amp;";
    case ">": return "&gt;";
    case "<": return "&lt;";
    }
    return what
  })
}

var entityReplace = regexp.MustCompile("&(#?[a-z0-9]+);")

func parseEntities(input string) string {
  return entityReplace.ReplaceAllStringFunc(input, func(what string) string {
    switch what {
    case "&lt;": return "<";
    case "&gt;": return ">";
    case "&amp;": return "&";
    case "&#93;": return "]";
    case "&#92;": return "\\";
    case "&#91;": return "[";
    case "&quot;": return "\"";
    }
    // TODO: Handle "all" escape codes. But as this is doubly encoded (for
    // XML) and UTF-8, that might not be necessary. Also, Wikipedia is
    // _really_ inconsistent about their escaping.
    return what
  })
}

var wikitokens = regexp.MustCompile("\\n|\\n\\*|\\n#|\\{\\{|\\}\\}|\\[\\[|\\]\\]|\\[|\\]|'''''|'''|''|=====|====|===|==|<source[^>]*>|</source>|<ref[^>]*>|</ref>|<code[^>]*>|</code>")

func tokenize(input []byte) []token {
  // Find the location of all known tokens.
  allIndexes := wikitokens.FindAllIndex(input, -1)

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
      allTokens[j] = token {
        IsToken: false,
        Val: string(input[lastIndex:allIndexes[i][0]]),
      }
      j++
    }
    // This token
    allTokens[j] = token {
      IsToken: true,
      Val: string(input[allIndexes[i][0]:allIndexes[i][1]]),
    }
    j++
    lastIndex = allIndexes[i][1]
  }
  if lastIndex < len(input) {
    allTokens[j] = token {
      IsToken: false,
      Val: string(input[lastIndex:len(input)]),
    }
  }

  return allTokens
}

// {{ ... }}
func parseTemplate(input []byte, tokens []token, i int, mi *markupInfo) (string, int) {
  // fmt.Printf("Entering {{...\n")
  // defer fmt.Printf("Leaving }}\n")
  _, eidx := parseGeneral(input, tokens, i + 1, []string{"}}"}, mi)
  return "--template--", eidx
}

func parseExternalLink(input []byte, tokens []token, i int, mi *markupInfo) (string, int) {
  // fmt.Printf("Entering [...\n")
  // defer fmt.Printf("Leaving ]\n")
  body, eidx := parseGeneral(input, tokens, i + 1, []string{"]"}, mi)
  args := strings.SplitN(body, " ", 2)
  var title string
  page := args[0]
  if len(args) > 1 {
    title = args[1]
  } else {
    title = page
  }
  link := fmt.Sprintf("<a href=\"%s\">%s</a>", page, title)
  return link, eidx
}

// [[ ... ]]
func parseInternalLink(input []byte, tokens []token, i int, mi *markupInfo) (string, int) {
  // fmt.Printf("Entering [[...\n")
  // defer fmt.Printf("Leaving ]]\n")
  body, eidx := parseGeneral(input, tokens, i + 1, []string{"]]"}, mi)
  args := strings.SplitN(body, "|", 2)
  var title string
  page := args[0]
  if len(args) > 1 {
    title = args[1]
  } else {
    title = page
  }
  link := fmt.Sprintf("<a href=\"/wiki/%s\">%s</a>", page, title)
  return link, eidx
}

// [[ ... ]]
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
  body, eidx := parseGeneral(input, tokens, i + 1, []string{"</ref>"}, mi)
  mi.refCount++
  link := fmt.Sprintf("<a href=\"#ref%d\">[%d]</a>", mi.refCount, mi.refCount)
  mi.refs = append(mi.refs, fmt.Sprintf("<a tag=\"#ref%d\"></a>%s", body))
  return link, eidx
}

// == foo ==
func parseHeader(input []byte, tokens []token, i int, mi *markupInfo) (string, int) {
  start := i
  x := len(tokens[start].Val)
  // fmt.Printf("Entering %s...\n", tokens[start].Val)
  // defer fmt.Printf("Leaving %s\n", tokens[start].Val)

  body, eidx := parseGeneral(input, tokens, i + 1, []string{tokens[start].Val}, mi)

  return fmt.Sprintf("<h%d>%s</h%d>", x, body, x), eidx
}

// ''''' ... '''''
func parseMarkup(input []byte, tokens []token, i int, mi *markupInfo) (string, int) {
  start := i
  x := len(tokens[start].Val)
  // fmt.Printf("Entering %s...\n", tokens[start].Val)
  // defer fmt.Printf("Leaving %s\n", tokens[start].Val)

  body, eidx := parseGeneral(input, tokens, i + 1, []string{tokens[start].Val}, mi)

  switch (x) {
  case 2:
    return fmt.Sprintf("<i>%s</i>", body), eidx
  case 3:
    return fmt.Sprintf("<b>%s</b>", body), eidx
  case 5:
    return fmt.Sprintf("<b><i>%s</i></b>", body), eidx
  }
  return fmt.Sprintf("<span style=\"color:yellow;\">%s</span>", body), eidx
}

// ''''' ... '''''
func parseCode(input []byte, tokens []token, i int, mi *markupInfo, end string) (string, int) {
  // fmt.Printf("Entering %s...\n", tokens[start].Val)
  // defer fmt.Printf("Leaving %s\n", tokens[start].Val)
  results := []string{}

  i += 1
  for {
    if i >= len(tokens) { break }
    if tokens[i].Val == end { break }
    if tokens[i].IsToken && len(tokens[i].Val) > 5 && tokens[i].Val[0:5] == "<code" {
      results = append(results, tokens[i].Val)
      s, eidx := parseCode(input, tokens, i, mi, "</code>")
      results = append(results, s)
      results = append(results, "</code>")
      i = eidx
    } else if tokens[i].IsToken && len(tokens[i].Val) > 7 && tokens[i].Val[0:7] == "<source" {
      results = append(results, tokens[i].Val)
      s, eidx := parseCode(input, tokens, i, mi, "</source>")
      results = append(results, s)
      results = append(results, "</source>")
      i = eidx
    }
    results = append(results, tokens[i].Val)
    i++
  }

  if (end == "</code>") {
    return fmt.Sprintf("<tt>%s</tt>", unparseEntities(strings.Join(results, ""))), i
  }
  return fmt.Sprintf("<pre>%s</pre>", unparseEntities(strings.Join(results, ""))), i
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
  for ;; {
    if i >= len(tokens) { break }
    if tokens[i].IsToken {
      for j := 0; j < len(endtokens); j++ {
        if tokens[i+j].IsToken && tokens[i+j].Val == endtokens[j] {
          return strings.Join(results, ""), i
        }
      }
      switch {
      case tokens[i].Val == "\n":
        if listType != "" {
          results = append(results, fmt.Sprintf("</%s>", listType))
          listType = ""
        }
        results = append(results, "<br />")
      case tokens[i].Val == "\n*":
        if listType != "ul" {
          results = append(results, "<ul>")
          listType = "ul"
        }
        results = append(results, "<li>")
      case tokens[i].Val == "\n#":
        if listType != "ol" {
          results = append(results, "<ol>")
          listType = "ol"
        }
        results = append(results, "<li>")
      case tokens[i].Val == "{{":
        body, eidx := parseTemplate(input, tokens, i, mi)
        results = append(results, body)
        i = eidx
      case tokens[i].Val == "[[":
        body, eidx := parseInternalLink(input, tokens, i, mi)
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
      default:
        if endtokens != nil {
          fmt.Printf("Don't know what to do with token \"%s\". endtokens[0] is \"%s\"\n", tokens[i].Val, endtokens[0])
          fmt.Printf("  Start is: \"%s\"\n", tokens[start].Val)
        } else {
          fmt.Printf("Don't know what to do with token '%s'. No endtokens\n", tokens[i].Val)
        }
      }
    } else {
      results = append(results, unparseEntities(string(tokens[i].Val)))
    }
    i += 1
  }
  return strings.Join(results, ""), i
}

func Wiki2HTML(input string) (string, []string) {
  // Screwy wikipedia doesn't know its own entities?
  // I got &amp;#93; that was supposed to be a closing ] to a [-tag!
  input  = parseEntities(parseEntities(input))
  binput := []byte(input)
  tokens := tokenize(binput)
  mi := markupInfo {
    depth: 0,
  }
  res, _ := parseGeneral(binput, tokens, 0, nil, &mi)
  return res, mi.refs
}
