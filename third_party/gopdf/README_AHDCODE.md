# AhdCode gopdf fork

This directory is an MIT-licensed source fork of
`github.com/SalvioniDigitalSolutions/gopdf` at
`v0.0.0-20260819123034-adfb3bf86a60`. Its `go.mod` retains the upstream module
identity so imports remain unchanged.

The only AhdCode change is support for a simple embedded CFF program's custom
code-to-glyph encoding. Tectonic writes that mapping into its Type-1C math
fonts. Using it lets the renderer consume the PDF's own glyph outlines without
calling `SystemFonts()` or reading host font directories.

See `glyph_cff.go`, `glyph_font.go`, and `simple_cff_encoding_test.go`.
