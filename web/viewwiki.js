
var citeCount = 0
var DataFormatters = {
  As: function(content) {
    var m = content.match(/\d+/);
    return "As of " + m[0];
  },
  infobox: function(content) {
    return "[[INFOBOX]]";
  },
  cite: function(content) {
    citeCount += 1
    return "[[CITE:" + citeCount + "]]";
  },
  cquote: function(content) {
    return "<blockquote>" + content + "</blockquote>"
  },
  Sic: function() {
    return '<i>[<a href="/wiki/Sic">Sic</a>]</i>'
  }
};

$(document).ready(function() {
  var input = $('#inbox').text()

  // Use the InstaView converter to cover _most_ of the formatting stuff.
  input = InstaView.convert(input)

  // InstaView doesn't handle everything, though, so code here handles
  // the rest. (Or at least, what I know of what's left)

  // Handle {{...}}:
  function replace_curlies(regexp, str, cb) {
    if (!str) { return ""; }
    return str.replace(regexp, function(full, contents) {
      var m = contents.match(/^\s*(\w+)\s*(?:\|\s*)?(.*)$/)
      if (m && DataFormatters[m[1]]) {
        return DataFormatters[m[1]](cb(m[2]));
      }
      return full;
    });
  }

  input = replace_curlies(/\{\{([^{}]+)\}\}/g, input, function(s) { return s });
  input = replace_curlies(/\{\{(([^{}]+|\{\{[^{}]+\}\})+)\}\}/g,
                          input, function(s) {
                            return replace_curlies(/\{\{([^{}]+)\}\}/g, s,
                              function(y) { return y; });
                          });

  $('#outbox').html(input)
});
