# AhdCode — Third-party notices for the Plot standard module

The AhdCode `Plot` standard module renders charts using the Gonum plotting
library, compiled directly into the `ahdcode` binary. This file records what
is statically linked and under which terms. Every license below is
permissive; none imposes copyleft obligations on AhdCode or on programs
compiled with it.

1. Gonum plot
   Project : https://github.com/gonum/plot
   License : BSD 3-Clause ("Copyright (c) 2013, The Gonum Authors. All
             rights reserved.")

2. go-latex/latex (Gonum plot's LaTeX-style label typesetting)
   Project : https://codeberg.org/go-latex/latex
   License : BSD 3-Clause

3. sbinet/gg (2-D vector graphics context used by Gonum's raster/PNG backend)
   Project : https://git.sr.ht/~sbinet/gg
   License : MIT

4. go-fonts/liberation (bundled Liberation font faces used for chart text)
   Project : https://codeberg.org/go-fonts/liberation
   License : Font data under the SIL Open Font License 1.1; supporting Go
             source under BSD 3-Clause

5. go-pdf/fpdf (PDF output backend)
   Project : https://codeberg.org/go-pdf/fpdf
   License : MIT

6. ajstarks/svgo (SVG output backend)
   Project : https://github.com/ajstarks/svgo
   License : Creative Commons Attribution 4.0 International (CC BY 4.0)

7. golang/freetype (font rasterization, a transitive dependency of the PNG
   backend)
   Project : https://github.com/golang/freetype
   License : Dual-licensed under the FreeType License (FTL) or GPLv2+.
             AhdCode uses this dependency under the FreeType License (FTL),
             a permissive BSD-style license with an attribution clause. No
             GPL-licensed code is used or distributed.

8. golang.org/x/image (supporting image/font decoding, a Go extended-stdlib
   package)
   Project : https://pkg.go.dev/golang.org/x/image
   License : BSD 3-Clause (same terms as the Go project itself)

Full upstream license texts accompany each module under
`$GOPATH/pkg/mod/<module>@<version>/LICENSE` for a local build, and are
reproduced in each project's repository linked above.
