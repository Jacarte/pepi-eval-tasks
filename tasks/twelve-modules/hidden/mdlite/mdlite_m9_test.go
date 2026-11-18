package mdlite_test

import (
	"testing"

	"twelvemods/mdlite"
)

type m9Case struct{ in, want string }

func m9Run(t *testing.T, cases []m9Case) {
	t.Helper()
	for _, c := range cases {
		if got := mdlite.ToHTML(c.in); got != c.want {
			t.Errorf("ToHTML(%q)\n got: %q\nwant: %q", c.in, got, c.want)
		}
	}
}

func TestM9Paragraphs(t *testing.T) {
	m9Run(t, []m9Case{
		{"", ""},
		{"\n\n  \n", ""},
		{"Hello", "<p>Hello</p>\n"},
		{"Hello\nworld", "<p>Hello\nworld</p>\n"},
		{"  a  \n   b", "<p>a\nb</p>\n"},
		{"a\n\nb", "<p>a</p>\n<p>b</p>\n"},
		{"a\n   \nb\n", "<p>a</p>\n<p>b</p>\n"},
		{"a\r\nb\r\n", "<p>a\nb</p>\n"},
		{"a\tb", "<p>a\tb</p>\n"},
		{"a  b", "<p>a  b</p>\n"},
	})
}

func TestM9Headings(t *testing.T) {
	m9Run(t, []m9Case{
		{"# Title", "<h1>Title</h1>\n"},
		{"###### six", "<h6>six</h6>\n"},
		{"####### seven", "<p>####### seven</p>\n"},
		{"#nospace", "<p>#nospace</p>\n"},
		{"#", "<h1></h1>\n"},
		{"### ", "<h3></h3>\n"},
		{"##   spaced   ", "<h2>spaced</h2>\n"},
		{"# a #", "<h1>a #</h1>\n"},
		{" # indented", "<p># indented</p>\n"},
		{"para\n# head\nmore", "<p>para</p>\n<h1>head</h1>\n<p>more</p>\n"},
		{"# *Hi* `x` & y", "<h1><em>Hi</em> <code>x</code> &amp; y</h1>\n"},
	})
}

func TestM9ThematicBreak(t *testing.T) {
	m9Run(t, []m9Case{
		{"---", "<hr>\n"},
		{"-----   ", "<hr>\n"},
		{"--", "<p>--</p>\n"},
		{"a\n---\nb", "<p>a</p>\n<hr>\n<p>b</p>\n"},
		{"***", "<p>***</p>\n"},
		{"- - -", "<ul>\n<li>- -</li>\n</ul>\n"},
		{"  ---", "<p>---</p>\n"},
	})
}

func TestM9FencedCode(t *testing.T) {
	m9Run(t, []m9Case{
		{"```\ncode <b>\n  x\n```", "<pre><code>code &lt;b&gt;\n  x\n</code></pre>\n"},
		{"```go extra\nfunc() {}\n```", "<pre><code class=\"language-go\">func() {}\n</code></pre>\n"},
		{"```   rust   \nx\n```", "<pre><code class=\"language-rust\">x\n</code></pre>\n"},
		{"```\n```", "<pre><code></code></pre>\n"},
		{"```\na\n\nb", "<pre><code>a\n\nb\n</code></pre>\n"},
		{"````\n```\n````", "<pre><code>```\n</code></pre>\n"},
		{"```\n# not heading\n- not list\n> not quote\n*x*\n```  \nafter", "<pre><code># not heading\n- not list\n&gt; not quote\n*x*\n</code></pre>\n<p>after</p>\n"},
		{"```a\"b\nx\n```", "<pre><code class=\"language-a&quot;b\">x\n</code></pre>\n"},
		{"```\n``` x\n```", "<pre><code>``` x\n</code></pre>\n"},
		{" ```\nx", "<p>```\nx</p>\n"},
		{"text\n```\ncode\n```\n- item", "<p>text</p>\n<pre><code>code\n</code></pre>\n<ul>\n<li>item</li>\n</ul>\n"},
		{"```\n  \n```", "<pre><code>  \n</code></pre>\n"},
	})
}

func TestM9Blockquotes(t *testing.T) {
	m9Run(t, []m9Case{
		{"> quote\n> more", "<blockquote>\n<p>quote\nmore</p>\n</blockquote>\n"},
		{"> a\n>\n> b", "<blockquote>\n<p>a</p>\n<p>b</p>\n</blockquote>\n"},
		{"> # H\n> - x", "<blockquote>\n<h1>H</h1>\n<ul>\n<li>x</li>\n</ul>\n</blockquote>\n"},
		{"> > nested", "<blockquote>\n<blockquote>\n<p>nested</p>\n</blockquote>\n</blockquote>\n"},
		{"> a\nb", "<blockquote>\n<p>a</p>\n</blockquote>\n<p>b</p>\n"},
		{">no space", "<blockquote>\n<p>no space</p>\n</blockquote>\n"},
		{">  two spaces", "<blockquote>\n<p>two spaces</p>\n</blockquote>\n"},
		{"> ```\n> code\nafter", "<blockquote>\n<pre><code>code\n</code></pre>\n</blockquote>\n<p>after</p>\n"},
		{">", "<blockquote>\n</blockquote>\n"},
		{"> a\n\n> b", "<blockquote>\n<p>a</p>\n</blockquote>\n<blockquote>\n<p>b</p>\n</blockquote>\n"},
		{" > a", "<p>&gt; a</p>\n"},
	})
}

func TestM9Lists(t *testing.T) {
	m9Run(t, []m9Case{
		{"- a\n- b", "<ul>\n<li>a</li>\n<li>b</li>\n</ul>\n"},
		{"* a\n+ b\n- c", "<ul>\n<li>a</li>\n<li>b</li>\n<li>c</li>\n</ul>\n"},
		{"1. one\n2. two", "<ol>\n<li>one</li>\n<li>two</li>\n</ol>\n"},
		{"3. three\n9. nine", "<ol start=\"3\">\n<li>three</li>\n<li>nine</li>\n</ol>\n"},
		{"007. x", "<ol start=\"7\">\n<li>x</li>\n</ol>\n"},
		{"0. zero", "<ol start=\"0\">\n<li>zero</li>\n</ol>\n"},
		{"1. a\n1. b\n1. c", "<ol>\n<li>a</li>\n<li>b</li>\n<li>c</li>\n</ol>\n"},
		{"- a\n  - b\n  - c\n- d", "<ul>\n<li>a\n<ul>\n<li>b</li>\n<li>c</li>\n</ul>\n</li>\n<li>d</li>\n</ul>\n"},
		{"- a\n  - b\n    - c", "<ul>\n<li>a\n<ul>\n<li>b\n<ul>\n<li>c</li>\n</ul>\n</li>\n</ul>\n</li>\n</ul>\n"},
		{"- a\n  - b\n    - c\n- d", "<ul>\n<li>a\n<ul>\n<li>b\n<ul>\n<li>c</li>\n</ul>\n</li>\n</ul>\n</li>\n<li>d</li>\n</ul>\n"},
		{"1. a\n   - b\n2. c", "<ol>\n<li>a\n<ul>\n<li>b</li>\n</ul>\n</li>\n<li>c</li>\n</ol>\n"},
		{"- a\n1. b", "<ul>\n<li>a</li>\n</ul>\n<ol>\n<li>b</li>\n</ol>\n"},
		{"- a\n  - b\n  1. c", "<ul>\n<li>a\n<ul>\n<li>b</li>\n</ul>\n<ol>\n<li>c</li>\n</ol>\n</li>\n</ul>\n"},
		{"- a\n    - b\n  - c", "<ul>\n<li>a\n<ul>\n<li>b</li>\n</ul>\n<ul>\n<li>c</li>\n</ul>\n</li>\n</ul>\n"},
		{"  - a\n- b", "<ul>\n<li>a</li>\n</ul>\n<ul>\n<li>b</li>\n</ul>\n"},
		{"- a\n\n- b", "<ul>\n<li>a</li>\n</ul>\n<ul>\n<li>b</li>\n</ul>\n"},
		{"- a\nb", "<ul>\n<li>a</li>\n</ul>\n<p>b</p>\n"},
		{"text\n- item", "<p>text</p>\n<ul>\n<li>item</li>\n</ul>\n"},
		{"-x\n1.x\n-\n1) y", "<p>-x\n1.x\n-\n1) y</p>\n"},
		{"-   ", "<p>-</p>\n"},
		{"- **bold** and `code`", "<ul>\n<li><strong>bold</strong> and <code>code</code></li>\n</ul>\n"},
		{"- a  \n-   b", "<ul>\n<li>a</li>\n<li>b</li>\n</ul>\n"},
		{"1234567890. x", "<p>1234567890. x</p>\n"},
		{"123456789. x", "<ol start=\"123456789\">\n<li>x</li>\n</ol>\n"},
		{"- a\n# h", "<ul>\n<li>a</li>\n</ul>\n<h1>h</h1>\n"},
		{"- <x> & y", "<ul>\n<li>&lt;x&gt; &amp; y</li>\n</ul>\n"},
	})
}

func TestM9Emphasis(t *testing.T) {
	m9Run(t, []m9Case{
		{"*em* and **strong**", "<p><em>em</em> and <strong>strong</strong></p>\n"},
		{"*a **b** c*", "<p><em>a <strong>b</strong> c</em></p>\n"},
		{"**a *b* c**", "<p><strong>a <em>b</em> c</strong></p>\n"},
		{"*a **b*", "<p><em>a **b</em></p>\n"},
		{"**a*", "<p>**a*</p>\n"},
		{"*a**", "<p>*a**</p>\n"},
		{"***a***", "<p>***a***</p>\n"},
		{"a*b*c", "<p>a<em>b</em>c</p>\n"},
		{"foo**bar**baz", "<p>foo<strong>bar</strong>baz</p>\n"},
		{"x * a * y", "<p>x * a * y</p>\n"},
		{"x * a*", "<p>x * a*</p>\n"},
		{"*a\nb*", "<p><em>a\nb</em></p>\n"},
		{"**a** **b", "<p><strong>a</strong> **b</p>\n"},
		{"*unclosed", "<p>*unclosed</p>\n"},
		{"a*b", "<p>a*b</p>\n"},
		{"*\tx*", "<p>*\tx*</p>\n"},
		{"*a *b* c*", "<p><em>a <em>b</em> c</em></p>\n"},
		{"*a* *b*", "<p><em>a</em> <em>b</em></p>\n"},
	})
}

func TestM9EscapesAndCode(t *testing.T) {
	m9Run(t, []m9Case{
		{"a & b < c > d \"q\" 'single'", "<p>a &amp; b &lt; c &gt; d &quot;q&quot; 'single'</p>\n"},
		{"\\*not em\\*", "<p>*not em*</p>\n"},
		{"\\**a*", "<p>*<em>a</em></p>\n"},
		{"a \\q b \\\\ c", "<p>a \\q b \\ c</p>\n"},
		{"end\\", "<p>end\\</p>\n"},
		{"\\<b\\>", "<p>&lt;b&gt;</p>\n"},
		{"\\# not heading", "<p># not heading</p>\n"},
		{"`code`", "<p><code>code</code></p>\n"},
		{"`a<b>&`", "<p><code>a&lt;b&gt;&amp;</code></p>\n"},
		{"`` a ` b ``", "<p><code>a ` b</code></p>\n"},
		{"` `", "<p><code> </code></p>\n"},
		{"`  x  `", "<p><code> x </code></p>\n"},
		{"`*not em*`", "<p><code>*not em*</code></p>\n"},
		{"`unclosed", "<p>`unclosed</p>\n"},
		{"``a`", "<p>``a`</p>\n"},
		{"`a\nb`", "<p><code>a b</code></p>\n"},
		{"`\\*`", "<p><code>\\*</code></p>\n"},
		{"\\`not code`", "<p>`not code`</p>\n"},
		{"`a``b`", "<p><code>a``b</code></p>\n"},
	})
}

func TestM9Links(t *testing.T) {
	m9Run(t, []m9Case{
		{"[link](http://x.com/?a=1&b=2)", "<p><a href=\"http://x.com/?a=1&amp;b=2\">link</a></p>\n"},
		{"see [**b**](u) now", "<p>see <a href=\"u\"><strong>b</strong></a> now</p>\n"},
		{"[a](b c)", "<p>[a](b c)</p>\n"},
		{"[a] (b)", "<p>[a] (b)</p>\n"},
		{"[a [b](c)", "<p>[a <a href=\"c\">b</a></p>\n"},
		{"[a\\]b](u)", "<p><a href=\"u\">a]b</a></p>\n"},
		{"[](u)", "<p><a href=\"u\"></a></p>\n"},
		{"[x]()", "<p><a href=\"\">x</a></p>\n"},
		{"[`a]`](u)", "<p>[<code>a]</code>](u)</p>\n"},
		{"*[a*](u)", "<p>*<a href=\"u\">a*</a></p>\n"},
		{"[a](u\"v)", "<p><a href=\"u&quot;v\">a</a></p>\n"},
		{"[a](u\\*)", "<p><a href=\"u\\*\">a</a></p>\n"},
		{"[a](u", "<p>[a](u</p>\n"},
		{"[a](b(c))", "<p>[a](b(c))</p>\n"},
		{"[*a*](x) [b](y)", "<p><a href=\"x\"><em>a</em></a> <a href=\"y\">b</a></p>\n"},
	})
}

func TestM9Document(t *testing.T) {
	in := "# Release notes\n" +
		"\n" +
		"Version **2.0** adds `x < y`\n" +
		"and fixes *many* bugs.\n" +
		"\n" +
		"## Changes\n" +
		"1. Faster [parser](https://e.org/p?q=1&r=2)\n" +
		"   - uses less memory\n" +
		"   - *safer*\n" +
		"2. New API\n" +
		"---\n" +
		"> Note:\n" +
		"> ```sh\n" +
		"> make <all>\n" +
		"> ```\n" +
		"> done\n" +
		"```\n" +
		"raw *text*\n" +
		"```\n"
	want := "<h1>Release notes</h1>\n" +
		"<p>Version <strong>2.0</strong> adds <code>x &lt; y</code>\n" +
		"and fixes <em>many</em> bugs.</p>\n" +
		"<h2>Changes</h2>\n" +
		"<ol>\n" +
		"<li>Faster <a href=\"https://e.org/p?q=1&amp;r=2\">parser</a>\n" +
		"<ul>\n" +
		"<li>uses less memory</li>\n" +
		"<li><em>safer</em></li>\n" +
		"</ul>\n" +
		"</li>\n" +
		"<li>New API</li>\n" +
		"</ol>\n" +
		"<hr>\n" +
		"<blockquote>\n" +
		"<p>Note:</p>\n" +
		"<pre><code class=\"language-sh\">make &lt;all&gt;\n" +
		"</code></pre>\n" +
		"<p>done</p>\n" +
		"</blockquote>\n" +
		"<pre><code>raw *text*\n" +
		"</code></pre>\n"
	m9Run(t, []m9Case{{in, want}})
}
