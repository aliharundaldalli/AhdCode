package packaging

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

// Application icons. A packaged application's icon is one PNG: the one given
// with --icon, or the official AhdCode icon. The macOS bundle gets it as an
// .icns file of PNG images, and a Windows executable as an icon resource of
// PNG images linked into it; both are written here with the standard library
// only, deterministically.

//go:embed ahdcode-icon.png
var defaultIcon []byte

// Icon bounds.
const (
	maxIconBytes = 8 << 20
	maxIconSide  = 4096
	minIconSide  = 16
)

// DefaultIconPNG is the official AhdCode icon.
func DefaultIconPNG() []byte { return defaultIcon }

// loadIcon reads and checks an icon PNG: a PNG image, square, at least 16
// pixels on a side.
func loadIcon(path string) ([]byte, image.Image, error) {
	data := defaultIcon
	if path != "" {
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			return nil, nil, fmt.Errorf("the icon %s cannot be read", path)
		}
		if info.Size() > maxIconBytes {
			return nil, nil, fmt.Errorf("the icon %s is larger than 8 MB", path)
		}
		if data, err = os.ReadFile(path); err != nil {
			return nil, nil, fmt.Errorf("the icon %s cannot be read", path)
		}
	}
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, nil, errors.New("the icon must be a PNG image")
	}
	if config.Width != config.Height || config.Width < minIconSide || config.Width > maxIconSide {
		return nil, nil, fmt.Errorf("the icon must be square, %d to %d pixels on a side; it is %dx%d", minIconSide, maxIconSide, config.Width, config.Height)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, nil, errors.New("the icon must be a PNG image")
	}
	return data, img, nil
}

// resize scales a square image to size pixels by bilinear sampling of the
// straight (non-premultiplied) colors.
func resize(source image.Image, size int) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, size, size))
	bounds := source.Bounds()
	scale := float64(bounds.Dx()) / float64(size)
	at := func(x, y int) color.NRGBA {
		x = min(max(x, 0), bounds.Dx()-1)
		y = min(max(y, 0), bounds.Dy()-1)
		return color.NRGBAModel.Convert(source.At(bounds.Min.X+x, bounds.Min.Y+y)).(color.NRGBA)
	}
	for y := range size {
		for x := range size {
			if scale > 1 {
				// Shrinking averages the whole source block.
				x0, y0 := int(float64(x)*scale), int(float64(y)*scale)
				x1, y1 := max(int(float64(x+1)*scale), x0+1), max(int(float64(y+1)*scale), y0+1)
				var r, g, b, a, n float64
				for sy := y0; sy < y1; sy++ {
					for sx := x0; sx < x1; sx++ {
						c := at(sx, sy)
						alpha := float64(c.A)
						r, g, b, a, n = r+float64(c.R)*alpha, g+float64(c.G)*alpha, b+float64(c.B)*alpha, a+alpha, n+1
					}
				}
				if a > 0 {
					out.SetNRGBA(x, y, color.NRGBA{uint8(math.Round(r / a)), uint8(math.Round(g / a)), uint8(math.Round(b / a)), uint8(math.Round(a / n))})
				}
				continue
			}
			fx, fy := (float64(x)+0.5)*scale-0.5, (float64(y)+0.5)*scale-0.5
			x0, y0 := int(math.Floor(fx)), int(math.Floor(fy))
			tx, ty := fx-float64(x0), fy-float64(y0)
			var channel [4]float64
			for _, corner := range []struct {
				dx, dy int
				w      float64
			}{{0, 0, (1 - tx) * (1 - ty)}, {1, 0, tx * (1 - ty)}, {0, 1, (1 - tx) * ty}, {1, 1, tx * ty}} {
				c := at(x0+corner.dx, y0+corner.dy)
				channel[0] += float64(c.R) * corner.w
				channel[1] += float64(c.G) * corner.w
				channel[2] += float64(c.B) * corner.w
				channel[3] += float64(c.A) * corner.w
			}
			out.SetNRGBA(x, y, color.NRGBA{uint8(math.Round(channel[0])), uint8(math.Round(channel[1])), uint8(math.Round(channel[2])), uint8(math.Round(channel[3]))})
		}
	}
	return out
}

func encodePNG(img image.Image) []byte {
	var buffer bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(&buffer, img); err != nil {
		panic("encoding an in-memory PNG cannot fail")
	}
	return buffer.Bytes()
}

// iconSizes renders the icon at each size.
func iconSizes(source image.Image, sizes []int) map[int][]byte {
	images := map[int][]byte{}
	for _, size := range sizes {
		images[size] = encodePNG(resize(source, size))
	}
	return images
}

// makeICNS writes a macOS .icns file of PNG images: 16 to 1024 pixels.
func makeICNS(source image.Image) []byte {
	entries := []struct {
		kind string
		size int
	}{{"icp4", 16}, {"icp5", 32}, {"icp6", 64}, {"ic07", 128}, {"ic08", 256}, {"ic09", 512}, {"ic10", 1024}}
	var sizes []int
	for _, entry := range entries {
		sizes = append(sizes, entry.size)
	}
	images := iconSizes(source, sizes)
	var body bytes.Buffer
	for _, entry := range entries {
		data := images[entry.size]
		body.WriteString(entry.kind)
		_ = binary.Write(&body, binary.BigEndian, uint32(8+len(data)))
		body.Write(data)
	}
	var file bytes.Buffer
	file.WriteString("icns")
	_ = binary.Write(&file, binary.BigEndian, uint32(8+body.Len()))
	file.Write(body.Bytes())
	return file.Bytes()
}

// windowsIconSizes are the icon images linked into a Windows executable.
var windowsIconSizes = []int{16, 24, 32, 48, 64, 128, 256}

// makeWindowsResource writes a COFF object (.syso) holding the icon as an
// RT_GROUP_ICON resource (id 1) and its RT_ICON images (ids 1..n), which the
// Go linker links into a Windows executable. machine is 0x8664 (x64) or
// 0xAA64 (ARM64).
func makeWindowsResource(source image.Image, machine uint16) []byte {
	images := iconSizes(source, windowsIconSizes)
	// The group icon directory names each image.
	var group bytes.Buffer
	_ = binary.Write(&group, binary.LittleEndian, [3]uint16{0, 1, uint16(len(windowsIconSizes))})
	for index, size := range windowsIconSizes {
		side := uint8(size)
		if size >= 256 {
			side = 0
		}
		_ = binary.Write(&group, binary.LittleEndian, struct {
			Width, Height, Colors, Reserved uint8
			Planes, BitCount                uint16
			Bytes                           uint32
			ID                              uint16
		}{side, side, 0, 0, 1, 32, uint32(len(images[size])), uint16(index + 1)})
	}

	// Resource data blobs, in order: the icon images, then the group.
	type leaf struct {
		kind, id uint16
		data     []byte
	}
	var leaves []leaf
	for index, size := range windowsIconSizes {
		leaves = append(leaves, leaf{3, uint16(index + 1), images[size]})
	}
	leaves = append(leaves, leaf{14, 1, group.Bytes()})

	// The resource directory: type -> id -> language (neutral) -> data.
	const (
		directorySize = 16
		entrySize     = 8
		dataEntrySize = 16
		languageID    = 0
	)
	types := []uint16{3, 14}
	byType := map[uint16][]leaf{}
	for _, item := range leaves {
		byType[item.kind] = append(byType[item.kind], item)
	}
	// Layout: root directory, type directories, id directories, data
	// entries, then the data.
	offset := directorySize + len(types)*entrySize
	typeOffsets := map[uint16]int{}
	for _, kind := range types {
		typeOffsets[kind] = offset
		offset += directorySize + len(byType[kind])*entrySize
	}
	idOffsets := map[[2]uint16]int{}
	for _, kind := range types {
		for _, item := range byType[kind] {
			idOffsets[[2]uint16{kind, item.id}] = offset
			offset += directorySize + entrySize
		}
	}
	dataEntryOffsets := map[[2]uint16]int{}
	for _, kind := range types {
		for _, item := range byType[kind] {
			dataEntryOffsets[[2]uint16{kind, item.id}] = offset
			offset += dataEntrySize
		}
	}
	dataOffsets := map[[2]uint16]int{}
	for _, kind := range types {
		for _, item := range byType[kind] {
			offset = (offset + 7) &^ 7
			dataOffsets[[2]uint16{kind, item.id}] = offset
			offset += len(item.data)
		}
	}
	section := make([]byte, (offset+7)&^7)
	put16 := func(at int, value uint16) { binary.LittleEndian.PutUint16(section[at:], value) }
	put32 := func(at int, value uint32) { binary.LittleEndian.PutUint32(section[at:], value) }
	directory := func(at, count int) { put16(at+14, uint16(count)) } // all entries are ids
	const subdirectory = 0x80000000
	directory(0, len(types))
	var relocations []uint32
	for index, kind := range types {
		entry := directorySize + index*entrySize
		put32(entry, uint32(kind))
		put32(entry+4, subdirectory|uint32(typeOffsets[kind]))
		directory(typeOffsets[kind], len(byType[kind]))
		for itemIndex, item := range byType[kind] {
			key := [2]uint16{kind, item.id}
			idEntry := typeOffsets[kind] + directorySize + itemIndex*entrySize
			put32(idEntry, uint32(item.id))
			put32(idEntry+4, subdirectory|uint32(idOffsets[key]))
			directory(idOffsets[key], 1)
			put32(idOffsets[key]+directorySize, languageID)
			put32(idOffsets[key]+directorySize+4, uint32(dataEntryOffsets[key]))
			// The data entry's address is image-relative: the linker fixes
			// it through a relocation against the section.
			put32(dataEntryOffsets[key], uint32(dataOffsets[key]))
			put32(dataEntryOffsets[key]+4, uint32(len(item.data)))
			relocations = append(relocations, uint32(dataEntryOffsets[key]))
			copy(section[dataOffsets[key]:], item.data)
		}
	}

	// The COFF object: header, one .rsrc section, its relocations, and a
	// symbol table naming the section.
	relocationType := uint16(3) // IMAGE_REL_AMD64_ADDR32NB
	if machine == 0xAA64 {
		relocationType = 2 // IMAGE_REL_ARM64_ADDR32NB
	}
	const headerSize, sectionHeaderSize, relocationSize, symbolSize = 20, 40, 10, 18
	dataStart := headerSize + sectionHeaderSize
	relocationStart := dataStart + len(section)
	symbolStart := relocationStart + len(relocations)*relocationSize
	var object bytes.Buffer
	_ = binary.Write(&object, binary.LittleEndian, struct {
		Machine            uint16
		Sections           uint16
		TimeDateStamp      uint32
		SymbolTable        uint32
		Symbols            uint32
		OptionalHeaderSize uint16
		Characteristics    uint16
	}{machine, 1, 0, uint32(symbolStart), 1, 0, 0})
	var name [8]byte
	copy(name[:], ".rsrc")
	_ = binary.Write(&object, binary.LittleEndian, struct {
		Name            [8]byte
		VirtualSize     uint32
		VirtualAddress  uint32
		RawSize         uint32
		RawData         uint32
		Relocations     uint32
		LineNumbers     uint32
		RelocationCount uint16
		LineNumberCount uint16
		Characteristics uint32
	}{name, 0, 0, uint32(len(section)), uint32(dataStart), uint32(relocationStart), 0, uint16(len(relocations)), 0,
		0x40000040}) // initialized data, readable
	object.Write(section)
	for _, at := range relocations {
		_ = binary.Write(&object, binary.LittleEndian, struct {
			Address uint32
			Symbol  uint32
			Type    uint16
		}{at, 0, relocationType})
	}
	_ = binary.Write(&object, binary.LittleEndian, struct {
		Name         [8]byte
		Value        uint32
		Section      int16
		Type         uint16
		StorageClass uint8
		AuxSymbols   uint8
	}{name, 0, 1, 0, 3, 0}) // IMAGE_SYM_CLASS_STATIC
	_ = binary.Write(&object, binary.LittleEndian, uint32(4)) // an empty string table
	return object.Bytes()
}

// macOSShape draws a full-square icon as macOS shows application icons: a
// rounded square, 824 of 1024 units, with corners of radius 185 units, on a
// transparent margin. It is used for the default AhdCode icon; an icon given
// with --icon is used as it is.
func macOSShape(source image.Image) *image.NRGBA {
	const size = 1024
	out := image.NewNRGBA(image.Rect(0, 0, size, size))
	body, radius := 824, 185.0
	inset := (size - body) / 2
	scaled := resize(source, body)
	for y := range body {
		for x := range body {
			px, py, side := float64(x)+0.5, float64(y)+0.5, float64(body)
			cx := math.Max(radius-px, math.Max(px-(side-radius), 0))
			cy := math.Max(radius-py, math.Max(py-(side-radius), 0))
			coverage := 1.0
			if cx > 0 && cy > 0 {
				coverage = math.Max(0, math.Min(1, radius-math.Sqrt(cx*cx+cy*cy)+0.5))
			}
			if coverage <= 0 {
				continue
			}
			c := scaled.NRGBAAt(x, y)
			c.A = uint8(math.Round(float64(c.A) * coverage))
			out.SetNRGBA(inset+x, inset+y, c)
		}
	}
	return out
}
