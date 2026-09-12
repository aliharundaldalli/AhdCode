# Barcode standard module

[English] · [Türkçe](BARCODE_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [QR](QR.md) · [Latex](LATEX.md) · [PDF](PDF.md)

Barcode (v1.3.0) creates linear barcodes inside the program: no network
service, no external tool, and no font. Import it explicitly:

```ahd
bring Barcode
from Barcode bring BarcodeCode
from Barcode bring BarcodeError
```

The canonical module identity is `builtin:Barcode`; a sibling `Barcode.ahd`
cannot shadow it. Every argument must be `NonNull`.

## Surface

```text
Barcode.code128(value: String) -> BarcodeCode
Barcode.ean13(value: String)   -> BarcodeCode
Barcode.upca(value: String)    -> BarcodeCode

BarcodeCode.kind()    -> String
BarcodeCode.value()   -> String
BarcodeCode.pattern() -> List<Bool>
BarcodeCode.savePNG(path: String, width: Int = 800, height: Int = 240) -> Nothing
BarcodeCode.saveSVG(path: String, width: Real = 8.0, height: Real = 2.4) -> Nothing

BarcodeCode
BarcodeError
```

`BarcodeCode` operations are positional-only.

## Symbologies

| Function | `kind()` | Accepts | Quiet zones (modules) | Typical use |
|---|---|---|---|---|
| `Barcode.code128` | `"Code128"` | 1 to 80 ASCII characters | 10 left, 10 right | order numbers, SKUs, shipping |
| `Barcode.ean13` | `"EAN13"` | 12 digits, or 13 with the check digit | 11 left, 7 right | retail products |
| `Barcode.upca` | `"UPCA"` | 11 digits, or 12 with the check digit | 9 left, 9 right | retail products in North America |

```ahd
product: BarcodeCode := Barcode.ean13("590123412345")
grocery: BarcodeCode := Barcode.upca("03600029145")
order: BarcodeCode := Barcode.code128("AHD-2026-0042")
```

**Check digits.** EAN-13 and UPC-A end with a GS1 modulo-10 check digit. Give
the data digits alone and Barcode computes it; give the full number and
Barcode verifies it. A wrong check digit raises `BarcodeError` naming the
correct one — it is never silently replaced. `value()` always includes the
check digit:

```ahd
write(product.value())
```

prints `5901234123457`. A UPC-A symbol's bars are the EAN-13 bars of the same
number with a leading zero, which is why EAN-13 scanners read UPC-A.

**Code 128** encodes ASCII text, choosing the most compact code sets
automatically. A character outside ASCII raises `BarcodeError` with its
position; use a [QR code](QR.md) for Unicode text.

## Reading the symbol

A `BarcodeCode` is immutable. `kind()` names the symbology, `value()` is the
value the bars carry, and `pattern()` returns the modules from left to right,
`true` for a dark module, without the quiet zones. An EAN-13 or UPC-A pattern
is always 95 modules. Every call returns a fresh List.

## PNG output

`savePNG(path, width, height)` writes black bars on white in an image of
exactly `width` by `height` pixels, quiet zones included. Every module is a
whole number of pixels wide, so bar edges stay sharp; pixels left over after
the largest whole module width widen the quiet zones evenly. `width` must be
at least the module count including the quiet zones and at most 10000;
`height` must be from 1 to 10000.

```ahd
product.savePNG("product.png")
order.savePNG("order.png", 1200, 300)
```

## SVG output

`saveSVG(path, width, height)` writes a vector SVG `width` by `height`
centimeters, quiet zones included. Both must be greater than 0 and at most
1000.

```ahd
product.saveSVG("product.svg", 6.0, 2.0)
```

Both save operations require the matching extension (`.png` or `.svg`), write
to a temporary file in the destination directory, verify the encoded bytes,
and then replace the destination in one step, so a failed save never leaves a
partial file.

## Barcodes in documents

`Latex.barcode(kind, value, width, height)` and
`PDFDocument.barcode(kind, value, width, height, align)` draw the same bars as
vector graphics inside a PDF, from the same encoder. `kind` is `"Code128"`,
`"EAN13"`, or `"UPCA"`. They need only `bring Latex` or `bring PDF`:

```ahd
body += L.barcode("EAN13", "590123412345", 6.0, 2.0)
doc = doc.barcode("Code128", "AHD-2026-0042", 7.0, 1.5, "left")
```

See [Latex](LATEX.md#qr-codes-and-barcodes-v130) and
[PDF](PDF.md#qr-codes-and-barcodes).

## Errors

`BarcodeError` covers an empty, oversized, non-ASCII, or non-digit value, a
wrong digit count or check digit, an invalid size, a wrong extension, and write
failures:

```ahd
attempt {
    Barcode.ean13("5901234123450")
}
except BarcodeError as error {
    write(error.message)
}
```

`Latex.barcode` reports the same problems as `ValueError`, and
`PDFDocument.barcode` as `PDFError`. Static argument count and type mistakes
remain compiler diagnostics.

## Implementation and license

The encoders are [`github.com/boombuler/barcode`](https://github.com/boombuler/barcode)
v1.1.0 (MIT License), vendored into the AhdCode source tree and into every
native program that uses barcodes. Nothing is downloaded at build or run time.
See [`THIRD_PARTY_NOTICES_CODES.md`](../THIRD_PARTY_NOTICES_CODES.md).

## Not in this version

Scanning or decoding barcodes, the human-readable digits printed under the
bars, EAN-8, UPC-E, EAN-2/EAN-5 add-ons, ITF-14, Code 39, GS1-128 application
identifiers, 2D symbologies other than QR (Data Matrix, PDF417, Aztec), and
colors are not part of v1.3.0.
