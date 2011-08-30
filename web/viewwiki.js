
var citeCount = 0
var DataFormatters = {
  // As of (time)
  as: function(content) {
    var m = content.match(/\d+/);
    return "As of " + m[0];
  },
  // Cite. Should eventually add to end of page.
  cite: function(content) {
    citeCount += 1
    return "[[CITE:" + citeCount + "]]";
  },
  // A quotation.
  cquote: function(content) {
    return "<blockquote>" + content + "</blockquote>"
  },
  // Death date, birth date, age.
  death: function(content) {
    return "(Death: " + content + ")";
  },
  // Info box.
  infobox: function(content) {
    return "[[INFOBOX]]";
  },
  // Pronunciation guide.
  ipa: function(content) {
    x = content.split(/\|/,2)
    return "<i>" + x[1] + "</i>";
  },
  // Main article
  main: function(content) {
    return "(Main: <i><a href=\"/wiki/\"" + content + ">" +
           content + "</a></i>)";
  },
  // See Also
  see: function(content) {
    args = content.split(/\|/); // also|Foo|
    return "(See also: <i><a href=\"/wiki/\"" + args[1] + ">" +
           args[1] + "</a></i>)";
  },
  // Goof [sic]
  sic: function() {
    return '<i>[<a href="/wiki/Sic">Sic</a>]</i>'
  },
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
    if (str.match(/^Info/)) {
      alert("Matched info something: " + str)
    }
    return str.replace(regexp, function(full, contents) {
      var m = contents.match(/^\s*(\w+)\s*(?:\|\s*)?(.*)$/)
      if (m && DataFormatters[m[1].toLowerCase()]) {
        return DataFormatters[m[1].toLowerCase()](cb(m[2]));
      }
      return full;
    });
  }

  input = input.replace(/[\r\n]/g,'');
  input = replace_curlies(/\{\{((?:[^{}]+|\{\{[^{}]+\}\})*)\}\}/g,
                          input, function(s) {
                            return replace_curlies(/\{\{([^{}]+)\}\}/g, s,
                              function(y) { return y; });
                          });
  input = replace_curlies(/\{\{([^{}]+)\}\}/g, input, function(s) { return s });
  input = replace_curlies(/\{\{([^{}]+)\}\}/g, input, function(s) { return s });

  $('#outbox').html(input)
});
