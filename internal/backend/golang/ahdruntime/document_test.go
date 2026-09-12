package ahdruntime

import (
	"strings"
	"testing"
)

func TestPageLayoutValidation(t *testing.T) {
	layout, problem := AhdPageLayoutText("Latex.document", "Letter", "Letter", false, 2.54, nil, nil, nil, nil)
	if problem != "" || layout.Configured {
		t.Fatalf("default Letter layout must not need explicit geometry: %+v %q", layout, problem)
	}
	layout, problem = AhdPageLayoutText("PDF", "A4", "A4", false, 2.54, nil, nil, nil, nil)
	if problem != "" || layout.Configured {
		t.Fatalf("default A4 PDF layout must not need explicit geometry: %+v %q", layout, problem)
	}
	layout, problem = AhdPageLayoutText("Latex.document", "Letter", "A3", true, 2.0, nil, nil, []string{"top", "left"}, []float64{3, 2.5})
	if problem != "" || !layout.Configured {
		t.Fatalf("A3 landscape layout: %+v %q", layout, problem)
	}
	if got := layout.Geometry(); got != "landscape,paperwidth=29.7cm,paperheight=42.0cm,top=3.0cm,right=2.0cm,bottom=2.0cm,left=2.5cm" {
		t.Fatalf("geometry = %q", got)
	}
	layout, problem = AhdPageLayoutText("PDF", "A4", "Custom", false, 0.5, []string{"width", "height"}, []float64{10, 6}, nil, nil)
	if problem != "" || layout.Geometry() != "paperwidth=10.0cm,paperheight=6.0cm,top=0.5cm,right=0.5cm,bottom=0.5cm,left=0.5cm" {
		t.Fatalf("custom layout %q %q", layout.Geometry(), problem)
	}
	for _, paper := range []string{"A3", "A4", "A5", "Letter", "Legal"} {
		if _, problem := AhdPageLayoutText("test", "A4", paper, false, 2.54, nil, nil, nil, nil); problem != "" {
			t.Fatalf("paper %s rejected: %s", paper, problem)
		}
	}
	cases := []struct {
		paper                  string
		sizeKeys, marginKeys   []string
		sizeValues, marginVals []float64
		problem                string
	}{
		{"a4", nil, nil, nil, nil, `test paper must be A3, A4, A5, Letter, Legal, or Custom; received "a4"`},
		{"Tabloid", nil, nil, nil, nil, `test paper must be A3, A4, A5, Letter, Legal, or Custom; received "Tabloid"`},
		{"Custom", nil, nil, nil, nil, `test paper "Custom" requires pageSize with both width and height`},
		{"Custom", []string{"width"}, nil, []float64{10}, nil, `test paper "Custom" requires pageSize with both width and height`},
		{"Custom", []string{"width", "height"}, nil, []float64{0, 6}, nil, "test pageSize width must be greater than 0 and at most 1000 centimeters"},
		{"Custom", []string{"width", "height"}, nil, []float64{10, -1}, nil, "test pageSize height must be greater than 0 and at most 1000 centimeters"},
		{"Custom", []string{"depth"}, nil, []float64{10}, nil, `test pageSize supports only width and height; received "depth"`},
		{"A4", []string{"width", "height"}, nil, []float64{10, 6}, nil, `test pageSize is only used with paper "Custom"; paper "A4" already sets the size`},
		{"A4", nil, []string{"top"}, nil, []float64{0}, "test margins top must be greater than 0 and at most 1000 centimeters"},
		{"A4", nil, []string{"bottom"}, nil, []float64{-2}, "test margins bottom must be greater than 0 and at most 1000 centimeters"},
		{"A4", nil, []string{"inner"}, nil, []float64{2}, `test margins supports only top, right, bottom, and left; received "inner"`},
		{"A5", nil, []string{"left", "right"}, nil, []float64{8, 7}, "test margins leave no room for content on a 14.8 by 21.0 cm page"},
	}
	for _, test := range cases {
		_, problem := AhdPageLayoutText("test", "A4", test.paper, false, 2.54, test.sizeKeys, test.sizeValues, test.marginKeys, test.marginVals)
		if problem != test.problem {
			t.Fatalf("layout %+v\n got %q\nwant %q", test, problem, test.problem)
		}
	}
}

func TestDocumentURLEncodingIsSafeAndExact(t *testing.T) {
	cases := []struct{ url, want string }{
		{"https://ahdcode.org/verify?id=AHD-42&lang=tr#top", `https://ahdcode.org/verify?id=AHD-42\&lang=tr\#top`},
		{"https://ahdcode.org/p%C5%9F?q=a%20b", `https://ahdcode.org/p\%C5\%9F?q=a\%20b`},
		{"https://ahdcode.org/Ayşe Yılmaz", "https://ahdcode.org/Ay%C5%9Fe%20Y%C4%B1lmaz"},
		{"https://ahdcode.org/~user/a_b$c", `https://ahdcode.org/\~user/a\_b\$c`},
		{`https://ahdcode.org/{x}\y^z|"<>` + "`", "https://ahdcode.org/%7Bx%7D%5Cy%5Ez%7C%22%3C%3E%60"},
		{"http://localhost:8080/", "http://localhost:8080/"},
		{"mailto:info@ahdcode.org?subject=Hi%20there", `mailto:info@ahdcode.org?subject=Hi\%20there`},
		{"HTTPS://AHDCODE.ORG", "HTTPS://AHDCODE.ORG"},
	}
	for _, test := range cases {
		got, problem := AhdDocumentURL("test", test.url)
		if problem != "" || got != test.want {
			t.Fatalf("URL %q = %q (%s), want %q", test.url, got, problem, test.want)
		}
		for _, forbidden := range []string{"{", "}", " ", "\\href", "\n"} {
			if strings.Contains(strings.NewReplacer(`\#`, "", `\%`, "", `\&`, "", `\_`, "", `\$`, "", `\~`, "").Replace(got), forbidden) {
				t.Fatalf("encoded URL %q still carries %q", got, forbidden)
			}
		}
	}
	rejected := []struct{ url, problem string }{
		{"", "test url must not be empty"},
		{"javascript:alert(1)", `test url must start with https://, http://, or mailto:; received "javascript:alert(1)"`},
		{"file:///etc/passwd", `test url must start with https://, http://, or mailto:; received "file:///etc/passwd"`},
		{"ftp://ahdcode.org", `test url must start with https://, http://, or mailto:; received "ftp://ahdcode.org"`},
		{"https://", `test url must name a host after "https://"`},
		{"mailto:", "test url must name an address after mailto:"},
		{"https://ahdcode.org/\x00x", "test url must not contain control characters"},
		{"https://ahdcode.org/}\\input{/etc/passwd}", ""},
	}
	for _, test := range rejected {
		got, problem := AhdDocumentURL("test", test.url)
		if problem != test.problem {
			t.Fatalf("URL %q problem %q, want %q", test.url, problem, test.problem)
		}
		if test.problem == "" && got != "https://ahdcode.org/%7D%5Cinput%7B/etc/passwd%7D" {
			t.Fatalf("an injection attempt was not neutralized: %q", got)
		}
	}
}

func TestLatexNavigationFragments(t *testing.T) {
	link, problem := AhdLatexLinkText("Latex.link", "Doğrula & aç", "https://ahdcode.org/v?a=1&b=2")
	if problem != "" || link != `\href{https://ahdcode.org/v?a=1\&b=2}{Doğrula \& aç}` {
		t.Fatalf("link = %q %q", link, problem)
	}
	if _, problem := AhdLatexLinkText("Latex.link", "", "https://ahdcode.org"); problem != "Latex.link text must not be empty" {
		t.Fatalf("empty link text problem %q", problem)
	}
	bookmark, problem := AhdLatexBookmarkText("Latex.bookmark", "Özet & 100% #1", 2)
	if problem != "" || bookmark != "%\n% AHDCODE_BOOKMARK\n\\ahdbookmark{1}{Özet \\& 100\\% \\#1}%\n" {
		t.Fatalf("bookmark = %q %q", bookmark, problem)
	}
	for _, level := range []int64{0, 5, -1} {
		if _, problem := AhdLatexBookmarkText("Latex.bookmark", "x", level); !strings.HasPrefix(problem, "Latex.bookmark level must be between 1 and 4") {
			t.Fatalf("level %d problem %q", level, problem)
		}
	}
	if _, problem := AhdLatexBookmarkText("Latex.bookmark", "  ", 1); problem != "Latex.bookmark title must not be empty" {
		t.Fatalf("blank title problem %q", problem)
	}
}

func TestLatexRunningAndPlacementFragments(t *testing.T) {
	header := AhdLatexRunningText("head", "L", "C", "R")
	if header != "%\n% AHDCODE_PAGESTYLE\n\\fancyhead[L]{%\nL}%\n\\fancyhead[C]{%\nC}%\n\\fancyhead[R]{%\nR}%\n" {
		t.Fatalf("header = %q", header)
	}
	if footer := AhdLatexRunningText("foot", "", "Page "+AhdLatexPageNumberText()+" / "+AhdLatexPageCountText(), ""); !strings.Contains(footer, "\\fancyfoot[C]{%\nPage \\thepage{} / %\n% AHDCODE_LASTPAGE\n\\pageref*{LastPage}}%\n") {
		t.Fatalf("footer = %q", footer)
	}
	anchors := map[string]string{"north west": "[tl]", "north": "[t]", "north east": "[tr]", "west": "[l]", "center": "",
		"east": "[r]", "south west": "[bl]", "south": "[b]", "south east": "[br]"}
	for anchor, box := range anchors {
		text, problem := AhdLatexPlaceText("X", 2.5, 3, anchor)
		if problem != "" || text != "%\n\\AddToHookNext{shipout/foreground}{{\\setlength{\\unitlength}{1cm}\\put(2.5,-3.0){\\makebox(0,0)"+box+"{%\nX}}}}%\n" {
			t.Fatalf("place %s = %q %q", anchor, text, problem)
		}
	}
	for _, test := range []struct {
		x, y    float64
		anchor  string
		problem string
	}{
		{1, 1, "top left", `Latex.place anchor must be north west, north, north east, west, center, east, south west, south, or south east; received "top left"`},
		{1, 1, "North", `Latex.place anchor must be north west, north, north east, west, center, east, south west, south, or south east; received "North"`},
		{-1, 1, "center", "Latex.place x must be between 0 and 1000 centimeters; received -1.0"},
		{1, 2000, "center", "Latex.place y must be between 0 and 1000 centimeters; received 2000.0"},
	} {
		if _, problem := AhdLatexPlaceText("X", test.x, test.y, test.anchor); problem != test.problem {
			t.Fatalf("place problem %q, want %q", problem, test.problem)
		}
	}
}

func TestLatexFeaturePreambleFollowsMarkers(t *testing.T) {
	if got := AhdLatexFeaturePreamble("plain body", "\\section{x}"); got != "" {
		t.Fatalf("a document without v1.3.0 fragments gained preamble lines: %q", got)
	}
	bookmark, _ := AhdLatexBookmarkText("Latex.bookmark", "x", 1)
	image, _ := AhdLatexImageText("a.png", nil, nil, []string{"opacity"}, []float64{0.5}, false, "", "")
	preamble := AhdLatexFeaturePreamble(AhdLatexRunningText("foot", "", AhdLatexPageCountText(), ""), bookmark+image)
	order := []string{"\\usepackage{fancyhdr}", "\\let\\ps@plain\\ps@fancy", "\\usepackage{lastpage}", "\\newcounter{ahdbookmark}", "\\newcommand{\\ahdimagetrim}", "\\newcommand{\\ahdimageopacity}"}
	position := -1
	for _, want := range order {
		index := strings.Index(preamble, want)
		if index <= position {
			t.Fatalf("preamble lacks %q in order:\n%s", want, preamble)
		}
		position = index
	}
	// A marker quoted inside ordinary text does not count.
	if got := AhdLatexFeaturePreamble("text % AHDCODE_PAGESTYLE"); got != "" {
		t.Fatalf("an inline marker loaded packages: %q", got)
	}
}

func TestLatexImageTransformsAndSVG(t *testing.T) {
	marker, staged := ahdLatexAssetMarker("logo.png", ".png")
	plain, problem := AhdLatexImageText("logo.png", []string{"width"}, []float64{6}, nil, nil, false, "", "")
	if problem != "" || plain != marker+"\\includegraphics[width=6.0cm]{"+staged+"}\n" {
		t.Fatalf("an untransformed PNG changed from v1.2.0: %q", plain)
	}
	figure, _ := AhdLatexImageText("logo.png", nil, nil, nil, nil, true, "Logo & mark", "fig:logo")
	if figure != marker+"\\begin{figure}[!ht]\n\\centering\n\\includegraphics{"+staged+"}\n\\caption{Logo \\& mark}\n\\label{fig:logo}\n\\end{figure}\n" {
		t.Fatalf("an untransformed figure changed from v1.2.0: %q", figure)
	}
	rotated, _ := AhdLatexImageText("logo.png", nil, nil, []string{"rotation"}, []float64{90}, false, "", "")
	if rotated != marker+"\\rotatebox{90.0}{\\includegraphics{"+staged+"}}\n" || strings.Contains(rotated, "AHDCODE_TIKZ") {
		t.Fatalf("rotation alone needs no PGF: %q", rotated)
	}
	full, _ := AhdLatexImageText("photo.jpg", []string{"width", "height"}, []float64{6, 4},
		[]string{"rotation", "opacity", "trimLeft", "trimTop", "trimRight", "trimBottom"}, []float64{-15, 0.4, 0.5, 0.25, 0.5, 0.75}, false, "", "")
	jpgMarker, jpgStaged := ahdLatexAssetMarker("photo.jpg", ".jpg")
	want := jpgMarker + "%\n% AHDCODE_TIKZ\n% AHDCODE_IMAGE\n\\ahdimageopacity{0.4}{\\rotatebox{-15.0}{\\ahdimagetrim{0.5}{0.25}{0.5}{0.75}{\\includegraphics[width=6.0cm,height=4.0cm]{" + jpgStaged + "}}}}\n"
	if full != want {
		t.Fatalf("full transform\n got %q\nwant %q", full, want)
	}
	svgMarker, svgStaged := ahdLatexAssetMarker("logo.svg", ".svg")
	svg, _ := AhdLatexImageText("logo.svg", []string{"height"}, []float64{0.8}, nil, nil, false, "", "")
	if svg != svgMarker+"%\n% AHDCODE_TIKZ\n\\resizebox{!}{0.8cm}{\\input{"+strings.TrimSuffix(svgStaged, ".svg")+"-svg.tex}}\n" {
		t.Fatalf("SVG fragment = %q", svg)
	}
	natural, _ := AhdLatexImageText("Logo.SVG", nil, nil, nil, nil, false, "", "")
	if !strings.Contains(natural, "\\mbox{\\input{") || !strings.Contains(natural, ".svg\n") {
		t.Fatalf("natural-size SVG fragment = %q", natural)
	}
	for _, test := range []struct {
		path       string
		sizeKeys   []string
		sizeValues []float64
		keys       []string
		values     []float64
		problem    string
	}{
		{"a.gif", nil, nil, nil, nil, "Latex image supports PNG, PDF, JPEG, and SVG assets"},
		{"", nil, nil, nil, nil, "Latex image path must not be empty"},
		{"a.png", []string{"depth"}, []float64{1}, nil, nil, "Latex image size supports only width and height"},
		{"a.png", []string{"width"}, []float64{0}, nil, nil, "Latex image dimensions must be positive"},
		{"a.png", nil, nil, []string{"scale"}, []float64{2}, `Latex image transform supports only rotation, opacity, trimLeft, trimTop, trimRight, and trimBottom; received "scale"`},
		{"a.png", nil, nil, []string{"opacity"}, []float64{1.5}, "Latex image transform opacity must be between 0.0 and 1.0; received 1.5"},
		{"a.png", nil, nil, []string{"opacity"}, []float64{-0.1}, "Latex image transform opacity must be between 0.0 and 1.0; received -0.1"},
		{"a.png", nil, nil, []string{"rotation"}, []float64{720}, "Latex image transform rotation must be between -360 and 360 degrees; received 720.0"},
		{"a.png", nil, nil, []string{"trimTop"}, []float64{-1}, "Latex image transform trimTop must be between 0 and 1000 centimeters; received -1.0"},
		{"a.png", []string{"width"}, []float64{4}, []string{"trimLeft", "trimRight"}, []float64{2, 2}, "Latex image transform trimLeft and trimRight remove the whole 4.0 cm width"},
		{"a.svg", []string{"height"}, []float64{2}, []string{"trimTop", "trimBottom"}, []float64{1.5, 0.5}, "Latex image transform trimTop and trimBottom remove the whole 2.0 cm height"},
	} {
		if _, problem := AhdLatexImageText(test.path, test.sizeKeys, test.sizeValues, test.keys, test.values, false, "", ""); problem != test.problem {
			t.Fatalf("image %+v problem\n got %q\nwant %q", test, problem, test.problem)
		}
	}
}

func TestLatexDocumentProfessionalOptions(t *testing.T) {
	empty := AhdBuildPair([]string{}, []string{})
	for _, kind := range []string{"Article", "Report", "Beamer"} {
		old := AhdLatexDocumentFull("Body", "Title", "Author", "", kind, 2.54, "", "", empty, "Default", false)
		text, problem := AhdLatexDocumentText(AhdLatexDocumentOptions{Body: "Body", Title: "Title", Author: "Author", Type: kind, Margin: 2.54, Theme: "Default", Paper: "Letter"})
		if problem != "" || text != old {
			t.Fatalf("%s: default v1.3.0 options changed the document (%s)", kind, problem)
		}
		if strings.Contains(old, "paperwidth") || strings.Contains(old, "pdfsubject") || strings.Contains(old, "fancyhdr") {
			t.Fatalf("%s: a v1.2.0-shaped call gained v1.3.0 preamble lines", kind)
		}
	}
	configured, problem := AhdLatexDocumentText(AhdLatexDocumentOptions{
		Body: AhdLatexRunningText("foot", "", AhdLatexPageCountText(), ""), Title: "Rapor & Özet", Author: "Ayşe", Type: "Report",
		Margin: 2.54, Theme: "Default", Paper: "A4", MarginKeys: []string{"top"}, MarginValues: []float64{3},
		Subject: "Numerical results", Keywords: []string{"report", "Türkçe"}, Creator: "AhdCode",
	})
	if problem != "" {
		t.Fatal(problem)
	}
	for _, want := range []string{
		"\\geometry{paperwidth=21.0cm,paperheight=29.7cm,top=3.0cm,right=2.54cm,bottom=2.54cm,left=2.54cm}\n\\hypersetup{hidelinks}\n",
		"\\hypersetup{pdftitle={Rapor \\& Özet},pdfauthor={Ayşe},pdfsubject={Numerical results},pdfkeywords={report, Türkçe},pdfcreator={AhdCode}}\n",
		"\\usepackage{fancyhdr}\n", "\\usepackage{lastpage}\n",
	} {
		if !strings.Contains(configured, want) {
			t.Fatalf("configured document lacks %q:\n%s", want, configured)
		}
	}
	header := AhdLatexRunningText("head", "x", "", "")
	for _, test := range []struct {
		options AhdLatexDocumentOptions
		problem string
	}{
		{AhdLatexDocumentOptions{Body: "b", Type: "Beamer", Margin: 2.54, Theme: "Default", Paper: "A4"}, "Latex.document paper, pageSize, and margins require an Article or Report document"},
		{AhdLatexDocumentOptions{Body: header, Type: "Beamer", Margin: 2.54, Theme: "Default", Paper: "Letter"}, "Latex.document header and footer require an Article or Report document"},
		{AhdLatexDocumentOptions{Body: "b", Type: "Article", Margin: 2.54, Theme: "Default", Paper: "Custom"}, `Latex.document paper "Custom" requires pageSize with both width and height`},
		{AhdLatexDocumentOptions{Body: "b", Type: "Article", Margin: 2.54, Theme: "Default", Paper: "Letter", Keywords: []string{"a,b"}}, "Latex.document keywords must be non-empty and must not contain commas"},
		{AhdLatexDocumentOptions{Body: "b", Type: "Article", Margin: 0, Theme: "Default", Paper: "A4"}, "Latex.document margin must be positive"},
	} {
		if _, problem := AhdLatexDocumentText(test.options); problem != test.problem {
			t.Fatalf("document problem\n got %q\nwant %q", problem, test.problem)
		}
	}
	// The native entry point raises ValueError with the same message.
	requireLatexValueError(t, "Latex.document paper must be A3, A4, A5, Letter, Legal, or Custom; received \"B5\"", func() {
		AhdLatexDocumentComplete("b", "", "", "", "Article", 2.54, "", "", nil, "Default", false, "B5", nil, nil, "", nil, "")
	})
}
