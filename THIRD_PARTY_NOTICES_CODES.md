# AhdCode — Third-party notice for the QR and Barcode standard modules

The QR and Barcode modules, and the `Latex.qr`, `Latex.barcode`,
`PDFDocument.qr`, and `PDFDocument.barcode` helpers built on them, encode
symbols with `github.com/boombuler/barcode` v1.1.0: a pure-Go encoder with no
CGO, no external process, and no network access. Only its `qr`, `code128`,
`ean`, and `utils` packages are used. It is Copyright © 2014 Florian
Sundermann and is distributed under the MIT License; the upstream license is
available at <https://github.com/boombuler/barcode/blob/v1.1.0/LICENSE>.

The encoder is embedded into AhdCode itself (see
`internal/backend/golang/ahdruntime/codesvendor`) and copied verbatim into a
generated program's build workspace as `vendor/` when that program uses QR or
barcode encoding, so the program builds with `go build -mod=vendor` and never
fetches the dependency over the network. Its LICENSE file travels with the
vendored source and remains present in that `vendor/` tree.

AhdCode renders PNG and SVG files and Latex/PDF vector drawings from the
encoder's logical symbol itself. No `qrencode`, `zbar`, ImageMagick,
Ghostscript, Inkscape, browser, Python, or Node process is used or required.
AhdCode does not decode symbols; its own QA decodes them with an independent
reader that is not part of any AhdCode binary or release payload.
