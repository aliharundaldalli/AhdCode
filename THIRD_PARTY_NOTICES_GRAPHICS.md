# AhdCode — Third-party notices for the Graphics standard module

The AhdCode `Graphics` standard module draws its windows through the bundled
`ahdgraphics` helper (`libexec/ahdcode/ahdgraphics`). The helper is a separate
executable built from its own Go module, `cmd/ahdgraphics`, so none of the
libraries below is linked into the `ahdcode` CLI, into its runtime, or into any
program compiled with AhdCode; a compiled program only starts the installed
helper. Every license below is permissive.

The versions are frozen in `cmd/ahdgraphics/go.mod` and `go.sum` and change
only through a deliberate AhdCode commit. The helper is built without cgo; it
downloads nothing at run time and needs no package manager.

1. Ebitengine v2.10.2 (`github.com/hajimehoshi/ebiten/v2`)
   Project : https://github.com/hajimehoshi/ebiten
   License : Apache License 2.0
   Use     : opens the window and shows the image `ahdgraphics` draws. Its
             sprite, input, audio, text, and game-loop features are not used
             and are not reachable from AhdCode. Ebitengine's own `NOTICE.md`,
             which reproduces the licenses of the code it bundles (including
             GLFW under the zlib license), is included in the module license
             inventory beside its `LICENSE`.

2. purego v0.11.0 (`github.com/ebitengine/purego`)
   Project : https://github.com/ebitengine/purego
   License : Apache License 2.0
   Use     : calls the operating system's window and graphics libraries
             without cgo.

3. hideconsole v1.0.0 (`github.com/ebitengine/hideconsole`)
   Project : https://github.com/ebitengine/hideconsole
   License : Apache License 2.0

4. gomobile (`github.com/ebitengine/gomobile`, a module Ebitengine requires)
   Project : https://github.com/ebitengine/gomobile
   License : BSD 3-Clause (with the Go PATENTS grant)

5. Go extended libraries: `golang.org/x/image` v0.45.0 (the anti-aliased
   rasterizer behind the window image and PNG export, the image scaler, and
   the font reader), `golang.org/x/text` v0.41.0 (required by the font
   reader), `golang.org/x/sync`, and `golang.org/x/sys`
   Project : https://go.googlesource.com/image, /text, /sync, /sys
   License : BSD 3-Clause (with the Go PATENTS grant)

6. Go Regular font (`golang.org/x/image/font/gofont/goregular`), the typeface
   of `Canvas.text` since v2.8.0
   Project : https://go.dev/blog/go-fonts
   Authors : Bigelow & Holmes
   License : the Go fonts license, BSD 3-Clause, as distributed with
             `golang.org/x/image`
   Use     : embedded in `ahdgraphics`; text is drawn from its glyph
             outlines, in the window, in PNG files, and as paths in SVG files.

The exact module list linked into each platform's helper, with the complete
license and notice texts, is recorded in `licenses/modules.json` and
`licenses/modules/` of the distribution. The PNG and SVG writers and the
drawing model are AhdCode's own code. PNG and JPEG files drawn with
`Canvas.image` are decoded by the Go standard library.

The window shows the AhdCode name and icon from AhdCode's own
`cmd/ahdidentity` module, which is first-party code under AhdCode's MIT license.

On Linux the helper uses the X11 and OpenGL libraries of the user's own
desktop system at run time; none of them is redistributed.
