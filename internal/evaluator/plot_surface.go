package evaluator

import (
	"strings"

	"ahdcode/internal/backend/golang/ahdruntime"
	"ahdcode/internal/ir"
)

// Plot.surface and the Surface members (v2.0), through the same runtime
// validation and viewer a compiled program uses.

const plotSurfaceClassID = ir.ClassID("builtin:Plot::class::Surface")

func plotSurfaceField(name string) ir.FieldID {
	return ir.FieldID(string(plotSurfaceClassID) + "::field::" + name)
}

// plotSurface is the working shape of one Surface value.
type plotSurface struct {
	x, y                          []float64
	z                             [][]float64
	title, xLabel, yLabel, zLabel string
	width, height                 int64
	wireframe                     bool
}

func (session *Session) surfaceOf(value any) plotSurface {
	instance := session.requireInstance(value)
	field := func(name string) any { return instance.Fields[plotSurfaceField(name)] }
	return plotSurface{
		x: plotRealsFromField(field("x")), y: plotRealsFromField(field("y")), z: plotRealGridFromField(field("z")),
		title: field("title").(string), xLabel: field("xLabel").(string), yLabel: field("yLabel").(string),
		zLabel: field("zLabel").(string), width: field("width").(int64), height: field("height").(int64),
		wireframe: field("wireframe").(bool),
	}
}

func plotSurfaceValue(s plotSurface) *Instance {
	return &Instance{Class: plotSurfaceClassID, Fields: map[ir.FieldID]any{
		plotSurfaceField("x"): plotRealsToField(s.x), plotSurfaceField("y"): plotRealsToField(s.y),
		plotSurfaceField("z"): plotRealGridToField(s.z), plotSurfaceField("title"): s.title,
		plotSurfaceField("xLabel"): s.xLabel, plotSurfaceField("yLabel"): s.yLabel, plotSurfaceField("zLabel"): s.zLabel,
		plotSurfaceField("width"): s.width, plotSurfaceField("height"): s.height, plotSurfaceField("wireframe"): s.wireframe,
	}}
}

// plotSurfaceBuiltin is Plot.surface(x, y, z).
func (session *Session) plotSurfaceBuiltin(arguments []any) any {
	x, y := session.plotNumbers(arguments[0]), session.plotNumbers(arguments[1])
	z := session.matrixRows(arguments[2])
	session.plotCheck(ahdruntime.AhdPlotSurfaceProblem(x, y, z))
	return plotSurfaceValue(plotSurface{x: x, y: y, z: z, xLabel: "x", yLabel: "y", zLabel: "z",
		width: plotDefaultWidth, height: plotDefaultHeight})
}

func (session *Session) plotSurfaceOperation(name string, receiver any, arguments []any) any {
	s := session.surfaceOf(receiver)
	text := func() string { return arguments[0].(string) }
	temp := func() string {
		dir, err := plotTempDir()
		if err != nil {
			session.raise("PlotError", "creating temporary directory: "+err.Error())
		}
		return dir
	}
	switch strings.TrimPrefix(name, "Surface.") {
	case "title":
		s.title = text()
	case "xLabel":
		s.xLabel = text()
	case "yLabel":
		s.yLabel = text()
	case "zLabel":
		s.zLabel = text()
	case "size":
		width, height := arguments[0].(int64), arguments[1].(int64)
		session.plotCheck(ahdruntime.AhdPlotSurfaceSizeProblem(width, height))
		s.width, s.height = width, height
	case "wireframe":
		s.wireframe = arguments[0].(bool)
	case "save":
		spec := ahdruntime.AhdPlotSurfaceData(s.x, s.y, s.z, s.title, s.xLabel, s.yLabel, s.zLabel, s.width, s.height, s.wireframe)
		session.plotCheck(ahdruntime.AhdPlotSurfaceSaveSpec(spec, session.sessionPath(text()), temp()))
		return Nothing
	case "show":
		spec := ahdruntime.AhdPlotSurfaceData(s.x, s.y, s.z, s.title, s.xLabel, s.yLabel, s.zLabel, s.width, s.height, s.wireframe)
		session.plotCheck(ahdruntime.AhdPlotSurfaceShowSpec(spec, temp()))
		return Nothing
	default:
		session.raise("Error", "unsupported Plot operation "+name)
	}
	return plotSurfaceValue(s)
}
