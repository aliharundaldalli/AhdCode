# QR standard module

[English] · [Türkçe](QR_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [Barcode](BARCODE.md) · [Latex](LATEX.md) · [PDF](PDF.md)

QR (v1.3.0) creates QR codes inside the program: no network service, no
external tool, and no browser. Import it explicitly:

```ahd
bring QR
from QR bring QRCode
from QR bring QRError
```

The canonical module identity is `builtin:QR`; a sibling `QR.ahd` cannot
shadow it. Every argument must be `NonNull`.

## Surface

```text
QR.create(value: String, level: String = "M") -> QRCode

QRCode.value()                                -> String
QRCode.level()                                -> String
QRCode.size()                                 -> Int
QRCode.matrix()                               -> List<List<Bool>>
QRCode.savePNG(path: String, pixels: Int = 512) -> Nothing
QRCode.saveSVG(path: String, size: Real = 5.0)  -> Nothing

QRCode
QRError
```

`QRCode` operations are positional-only, like `PDFDocument` and `Document`.

## Creating a code

`QR.create(value, level)` encodes any non-empty UTF-8 String — a URL, plain
text, Turkish letters, or emoji. The encoder picks the smallest symbol and the
most compact encoding for the value automatically.

`level` is the error-correction level, exactly `"L"`, `"M"`, `"Q"`, or `"H"`
(case-sensitive). A higher level lets a scanner recover more of a damaged or
covered symbol, and needs more modules for the same value:

| Level | Recovers about |
|---|---|
| `"L"` | 7% |
| `"M"` (default) | 15% |
| `"Q"` | 25% |
| `"H"` | 30% |

```ahd
site: QRCode := QR.create("https://ahdcode.org")
contact: QRCode := QR.create("Ayşe Yılmaz, +90 555 000 00 00", "Q")
```

`QR.create` encodes the value once to validate it, so every `QRCode` value can
be rendered. A value too long for the largest symbol at the requested level
raises `QRError`.

## Reading the symbol

A `QRCode` is immutable.

- `value()` and `level()` return what the code was created with.
- `size()` is the number of modules per side, from 21 for the smallest symbol
  to 177 for the largest, without the quiet zone.
- `matrix()` returns the modules row by row from the top, each row from the
  left, `true` for a dark module, without the quiet zone. Every call returns a
  fresh List, so changing it never changes the `QRCode`.

```ahd
matrix: List<List<Bool>> := site.matrix()
write(str(site.size()) + " modules per side, " + str(len(matrix)) + " rows")
```

## PNG output

`savePNG(path, pixels)` writes a black-on-white PNG of exactly `pixels` by
`pixels`, including the standard quiet zone of four light modules on every
side. Every module is a whole number of pixels, so module edges stay sharp;
pixels left over after the largest whole module size widen the quiet zone
evenly. `pixels` must be at least `size() + 8` and at most 10000.

```ahd
site.savePNG("site.png")
site.savePNG("site-print.png", 2048)
```

## SVG output

`saveSVG(path, size)` writes a square vector SVG `size` centimeters wide,
quiet zone included: a white square and one black path of module runs. `size`
must be greater than 0 and at most 1000.

```ahd
site.saveSVG("site.svg", 3.0)
```

Both save operations require the matching extension (`.png` or `.svg`), write
to a temporary file in the destination directory, verify the encoded bytes,
and then replace the destination in one step, so a failed save never leaves a
partial file.

## QR codes in documents

`Latex.qr(value, size, level)` and `PDFDocument.qr(value, size, level, align)`
draw the same symbol as vector graphics inside a PDF, from the same encoder, so
a value produces the same modules everywhere. They need only `bring Latex` or
`bring PDF`:

```ahd
body += L.place(L.qr("https://ahdcode.org/verify?id=42", 2.5, "Q"), 19.0, 26.0, "south east")
doc = doc.qr("https://ahdcode.org/verify?id=42", 2.5, "Q", "right")
```

See [Latex](LATEX.md#qr-codes-and-barcodes-v130) and
[PDF](PDF.md#qr-codes-and-barcodes).

## Errors

`QRError` covers an invalid level, an empty or oversized value, an invalid
`pixels` or `size`, a wrong extension, and write failures:

```ahd
attempt {
    QR.create("https://ahdcode.org", "X")
}
except QRError as error {
    write(error.message)
}
```

`Latex.qr` reports the same problems as `ValueError`, and `PDFDocument.qr` as
`PDFError`. Static argument count and type mistakes remain compiler
diagnostics.

## Implementation and license

The encoder is [`github.com/boombuler/barcode`](https://github.com/boombuler/barcode)
v1.1.0 (MIT License), vendored into the AhdCode source tree and into every
native program that uses QR codes; a program without QR codes carries none of
it. Nothing is downloaded at build or run time. See
[`THIRD_PARTY_NOTICES_CODES.md`](../THIRD_PARTY_NOTICES_CODES.md).

## Not in this version

Reading or decoding QR codes from images or a camera, colors, logos, rounded
or styled modules, Micro QR, structured append, and choosing the symbol
version, mask, or encoding mode by hand are not part of v1.3.0.
