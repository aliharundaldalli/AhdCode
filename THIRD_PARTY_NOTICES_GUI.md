# AhdCode — Third-party notices for the GUI module and the Plot viewer

The AhdCode `GUI` standard module draws its windows through the bundled
`ahdgui` helper (`libexec/ahdcode/ahdgui`), and `Chart.show`/`Figure.show` of
the Plot standard module open the bundled interactive viewer `ahdplotview`
(`libexec/ahdcode/ahdplotview`). Each helper is a separate executable built
from its own Go module, `cmd/ahdgui` and `cmd/ahdplotview`, so none of the
libraries below is linked into the `ahdcode` CLI, into its runtime, or into any
program compiled with AhdCode; a compiled program only starts an installed
helper. Every license below is permissive.

The versions are frozen in each module's `go.mod` and `go.sum` and change only
through a deliberate AhdCode commit. Both helpers use exactly the same modules
and versions; they are the Graphics helper's (see
`THIRD_PARTY_NOTICES_GRAPHICS.md`) plus `golang.org/x/text`. The helpers are
built without cgo; they download nothing at run time, need no package manager,
and download no fonts or icons. The AhdCode name and icon they show come from
AhdCode's own `cmd/ahdidentity` module (the icon is
`editors/vscode/images/ahdcode-icon.png`, embedded unchanged), which is covered
by AhdCode's MIT license and is not a third-party component.

1. Ebitengine v2.10.2 (`github.com/hajimehoshi/ebiten/v2`)
   Project : https://github.com/hajimehoshi/ebiten
   License : Apache License 2.0
   Use     : opens the window, reads the mouse and keyboard, and shows the
             image `ahdgui` draws or the chart `ahdplotview` shows. The
             widgets, layout, text editing, and viewer controls are AhdCode's
             own code; Ebitengine's sprite, audio, text, and game features
             are not used. Ebitengine's own `NOTICE.md`, which
             reproduces the licenses of the code it bundles (including GLFW
             under the zlib license), is included in the module license
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
   rasterizer and the OpenType font reader that draw the widgets and text),
   `golang.org/x/text` v0.41.0 (used by the font reader), `golang.org/x/sync`,
   and `golang.org/x/sys`
   Project : https://go.googlesource.com/image, /text, /sync, /sys
   License : BSD 3-Clause (with the Go PATENTS grant)

6. Go Regular font (`golang.org/x/image/font/gofont/goregular`, part of
   `golang.org/x/image` v0.45.0)
   Project : https://go.dev/blog/go-fonts
   License : BSD-style license of the Go project, reproduced below
   Use     : embedded in the helper and used for every piece of text in a GUI
             window, so text looks the same on every computer. No system font
             is read.

The exact module list linked into each platform's helper, with the complete
license and notice texts, is recorded in `licenses/modules.json` and
`licenses/modules/` of the distribution.

On Linux the helper uses the X11 and OpenGL libraries of the user's own
desktop system at run time; none of them is redistributed.

## Go fonts license

The following text accompanies the Go fonts in `golang.org/x/image`
(`font/gofont/ttfs/README`):

```text
These fonts were created by the Bigelow & Holmes foundry specifically for the
Go project. See https://blog.golang.org/go-fonts for details.

They are licensed under the same open source license as the rest of the Go
project's software:

Copyright (c) 2016 Bigelow & Holmes Inc.. All rights reserved.

Distribution of this font is governed by the following license. If you do not
agree to this license, including the disclaimer, do not distribute or modify
this font.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

	* Redistributions of source code must retain the above copyright notice,
	  this list of conditions and the following disclaimer.

	* Redistributions in binary form must reproduce the above copyright notice,
	  this list of conditions and the following disclaimer in the documentation
	  and/or other materials provided with the distribution.

	* Neither the name of Google Inc. nor the names of its contributors may be
	  used to endorse or promote products derived from this software without
	  specific prior written permission.

DISCLAIMER: THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO,
THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```
