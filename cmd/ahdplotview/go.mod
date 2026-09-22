module ahdplotview

go 1.26.4

require (
	ahdcode v0.0.0
	ahdidentity v0.0.0
	github.com/hajimehoshi/ebiten/v2 v2.10.2
	golang.org/x/image v0.45.0
)

require (
	github.com/SalvioniDigitalSolutions/gopdf v0.0.0-20260819123034-adfb3bf86a60 // indirect
	github.com/ebitengine/gomobile v0.0.0-20260820040257-d11f821a26a6 // indirect
	github.com/ebitengine/hideconsole v1.0.0 // indirect
	github.com/ebitengine/purego v0.11.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
)

// ahdidentity is AhdCode's own shared window identity (name and icon), kept
// beside this module and never fetched.
replace ahdidentity => ../ahdidentity

replace ahdcode => ../..

replace github.com/SalvioniDigitalSolutions/gopdf => ../../third_party/gopdf
