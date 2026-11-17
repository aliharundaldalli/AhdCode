# Graphics standard module

[English] · [Türkçe](GRAPHICS_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [Plot](PLOT.md) · [Math](MATH.md)

`Graphics` is the compiler-registered `builtin:Graphics` module added in
AhdCode v1.6.0. It opens a window with a 2D Canvas, draws lines, circles,
rectangles, ellipses, polygons, arcs, text, and images on it in mathematical
(Cartesian) coordinates, moves a Turtle pen across it, and saves the drawing as
PNG or SVG.

Graphics is for visual programming and 2D drawing: coordinates, angles,
geometry, transformations, fractals, and turtle programs in a mathematics or
programming lesson. **It is not a game engine and not a GUI toolkit.** It has no
sprites, scenes, collision, physics, input polling, sound, widgets, or 3D, and
none of them is planned for it. Since v1.8.0 a Canvas can
report clicks and key presses to a callback, so a Turtle can be steered with
the arrow keys; see [Clicks and key presses](#clicks-and-key-presses). Since
v2.8.0 it also draws text, polygons, ellipses, arcs, and images, reports
pointer movement and dragging, and runs timers, which is enough for a moving
picture or a drawing made with the mouse; see [Text](#text),
[Images](#images), [Pointer movement and dragging](#pointer-movement-and-dragging),
and [Timers and animation](#timers-and-animation). For windows with buttons
and text fields, use [GUI](GUI.md).

## Public surface

```text
bring Graphics
from Graphics bring (Canvas, Turtle, GraphicsError)

Graphics.open(
    width: Int := 800
    height: Int := 600
    title: String := "AhdCode Graphics"
    background: String := "white"
)                                                                    -> Canvas

canvas.clear(color: String := "white")                               -> Nothing
canvas.line(
    x1: Real, y1: Real, x2: Real, y2: Real
    color: String := "black"
    width: Real := 1.0
)                                                                    -> Nothing
canvas.circle(
    x: Real, y: Real, radius: Real
    stroke: String := "black"
    fill: String? := null
    width: Real := 1.0
)                                                                    -> Nothing
canvas.rectangle(
    x: Real, y: Real, width: Real, height: Real
    stroke: String := "black"
    fill: String? := null
    lineWidth: Real := 1.0
)                                                                    -> Nothing
canvas.ellipse(
    x: Real, y: Real, radiusX: Real, radiusY: Real
    stroke: String := "black"
    fill: String? := null
    width: Real := 1.0
)                                                                    -> Nothing
canvas.polygon(
    points: List<Real>            // x0, y0, x1, y1, ...; List<Int> too
    stroke: String := "black"
    fill: String? := null
    width: Real := 1.0
)                                                                    -> Nothing
canvas.arc(
    x: Real, y: Real, radius: Real, start: Real, end: Real
    color: String := "black"
    width: Real := 1.0
)                                                                    -> Nothing
canvas.text(
    x: Real, y: Real, text: String
    color: String := "black"
    size: Real := 16.0
    align: String := "left"
)                                                                    -> Nothing
canvas.image(
    path: String, x: Real, y: Real
    width: Real := 0.0
    height: Real := 0.0
)                                                                    -> Nothing
canvas.batch(handler: () -> Nothing)                                 -> Nothing
canvas.save(path: String)                                            -> Nothing
canvas.wait()                                                        -> Nothing
canvas.onClick(handler: (x: Real, y: Real) -> Nothing)               -> Nothing
canvas.onKey(handler: (key: String) -> Nothing)                      -> Nothing
canvas.onMove(handler: (x: Real, y: Real) -> Nothing)                -> Nothing
canvas.onDrag(handler: (x: Real, y: Real) -> Nothing)                -> Nothing
canvas.every(milliseconds: Int, handler: () -> Nothing)              -> Int
canvas.cancel(timer: Int)                                            -> Nothing
canvas.close()                                                       -> Nothing
canvas.isOpen()                                                      -> Bool
canvas.turtle()                                                      -> Turtle

turtle.forward(distance: Real)                                       -> Nothing
turtle.backward(distance: Real)                                      -> Nothing
turtle.left(degrees: Real)                                           -> Nothing
turtle.right(degrees: Real)                                          -> Nothing
turtle.moveTo(x: Real, y: Real)                                      -> Nothing
turtle.setHeading(degrees: Real)                                     -> Nothing
turtle.penUp()                                                       -> Nothing
turtle.penDown()                                                     -> Nothing
turtle.setColor(color: String)                                       -> Nothing
turtle.setWidth(width: Real)                                         -> Nothing
turtle.home()                                                        -> Nothing
turtle.x()                                                           -> Real
turtle.y()                                                           -> Real
turtle.heading()                                                     -> Real

GraphicsError  (derived from Error)
```

`Canvas` and `Turtle` have no constructor: a Canvas comes from
`Graphics.open`, and a Turtle from `canvas.turtle()`. There is no
`bring Turtle`; Turtle belongs to Graphics.

## A first drawing

```ahd
bring Graphics

canvas := Graphics.open(400, 300)
canvas.line(-200, 0, 200, 0)
canvas.line(0, -150, 0, 150)
canvas.circle(x: 0, y: 0, radius: 80, stroke: "blue", fill: "#ffff0080")

turtle := canvas.turtle()
turtle.setColor("red")
side := 0
while side < 4 {
    turtle.forward(100)
    turtle.left(90)
    side = side + 1
}

canvas.save("first.png")
canvas.wait()
```

The program draws two axes, a half-transparent yellow circle, and a red square
whose lower-left corner is the origin, saves the picture, and keeps the window
open until you close it.

## Calling Graphics

Every call is entirely positional or entirely named, like every call in
AhdCode. Arguments with a default may be left out; with named arguments any of
them may be left out, in any order.

```ahd
bring Graphics

canvas := Graphics.open(800, 600)
labelled := Graphics.open(
    width: 640
    height: 480
    title: "Shapes"
    background: "black"
)
canvas.line(0, 0, 100, 50, "red", 2)
canvas.line(x1: 0, y1: 0, x2: 100, y2: 50, width: 3)
canvas.circle(x: 0, y: 0, radius: 40, fill: "yellow")
```

A call that mixes the two styles is a compile error. Integer arguments are
accepted wherever a `Real` is expected.

## Coordinates

Every Canvas uses mathematical coordinates:

- the origin `(0, 0)` is the **center** of the Canvas;
- `+x` points right and `-x` left;
- `+y` points **up** and `-y` down;
- one unit is one pixel of the window.

An 800 × 600 Canvas shows `x` from -400 to +400 and `y` from -300 to +300.
Drawing that reaches outside the Canvas is clipped; it is not an error.

`canvas.rectangle(x, y, width, height)` takes its **lower-left** corner, the
natural corner in these coordinates, and grows right and up. It is never read
as a top-left corner.

## Colors

Every color argument is one of:

- one of nine names, written exactly like this (lower case):
  `black`, `white`, `red`, `green`, `blue`, `yellow`, `cyan`, `magenta`, `gray`;
- `#RRGGBB`, six hexadecimal digits (`#1e90ff`);
- `#RRGGBBAA`, eight digits whose last pair is the opacity, from `00`
  (transparent) to `ff` (opaque): `#ff000080` is half-transparent red.

| Name      | Value     |
| --------- | --------- |
| `black`   | `#000000` |
| `white`   | `#ffffff` |
| `red`     | `#ff0000` |
| `green`   | `#008000` |
| `blue`    | `#0000ff` |
| `yellow`  | `#ffff00` |
| `cyan`    | `#00ffff` |
| `magenta` | `#ff00ff` |
| `gray`    | `#808080` |

Any other text, such as `"Red"`, `"purple"`, or `"#fff"`, raises
`GraphicsError`; a color is never silently replaced by black.

## Drawing

`canvas.line(x1, y1, x2, y2)` draws a straight line with round ends. `width`
must be greater than 0. A line whose two ends are the same point is a round
dot as wide as the line.

`canvas.circle(x, y, radius)` draws a circle around `(x, y)`. `stroke` is the
outline color and `width` its thickness, which must be greater than 0. `fill`
is the inside color; `null`, the default, leaves the inside empty. `radius`
must be 0 or greater, and a circle of radius 0 draws nothing.

`canvas.rectangle(x, y, width, height)` draws a rectangle from its lower-left
corner. `width` and `height` must be 0 or greater, and a rectangle with a zero
side draws nothing. `stroke`, `fill`, and `lineWidth` work like the circle's
`stroke`, `fill`, and `width`.

`canvas.ellipse(x, y, radiusX, radiusY)` (since v2.8.0) draws an ellipse
around `(x, y)` with a horizontal and a vertical radius. Both must be 0 or
greater, and an ellipse with a zero radius draws nothing. `stroke`, `fill`, and
`width` work like the circle's.

`canvas.polygon(points)` (since v2.8.0) draws a closed shape through its
corners. `points` is one flat List of coordinates, `[x0, y0, x1, y1, ...]`, of
`Real` or of `Int`: a triangle is six numbers. The last corner is joined back
to the first. A polygon needs between 3 and 2000 corners, and an odd count of
numbers raises `GraphicsError`. `stroke`, `fill`, and `width` work like the
circle's. Where a polygon crosses itself, as a five-pointed star does, every
part the outline encloses is filled.

`canvas.arc(x, y, radius, start, end)` (since v2.8.0) draws part of a circle's
outline around `(x, y)`. `start` and `end` are angles in degrees, measured
counterclockwise from the positive x axis, as in mathematics and as a Turtle's
heading: 0 is right, 90 is up. The arc runs counterclockwise when `end` is
greater than `start` and clockwise when it is smaller, and never more than one
full turn. An arc has round ends and no fill. `radius` must be 0 or greater.

When a shape has both, the fill is drawn first and the outline over it. Later
drawings cover earlier ones.

```ahd
bring Graphics

canvas := Graphics.open(400, 300)
canvas.text(0, 100, "Triangle", "blue", 28, "center")
canvas.polygon([-100, -80, 100, -80, 0, 60], "black", "yellow", 2)
canvas.ellipse(0, -110, 120, 15, "gray")
canvas.arc(0, -80, 40, 0, 180, "red", 3)
canvas.save("triangle.png")
canvas.wait()
```

`canvas.clear(color)` removes every drawing and makes `color` the new
background. It does not close the window, and it does not move or change any
Turtle.

## Text

`canvas.text(x, y, text)` (since v2.8.0) writes one line of text. `(x, y)` is
on the baseline, the line the letters stand on, so letters rise above `y` and
only the tails of letters such as `g` and `p` reach below it.

- `align` places the text around `x`: `"left"` (the default) starts it there,
  `"center"` centers it there, and `"right"` ends it there. Any other value
  raises `GraphicsError`.
- `size` is the height of the font in Canvas units; it must be greater than 0
  and at most 2000.
- `color` is any Graphics color.

The text is a single line of at most 1000 characters: a line break raises
`GraphicsError`, so write a second line with a second call, lower by about
1.2 times the size. An empty text draws nothing.

Every Canvas uses the same bundled font (Go Regular), so a drawing looks the
same on every computer. It covers Latin letters, the Turkish letters among
them, and Greek and Cyrillic; a character it lacks is drawn as an empty box.
In an SVG file the text is saved as its outline, so the file needs no font to
display exactly, and it carries the original text as a label.

## Images

`canvas.image(path, x, y)` (since v2.8.0) draws a PNG or JPEG file with its
lower-left corner at `(x, y)`, like a rectangle. A relative path is relative to
the folder the program runs in.

- With no `width` and `height`, the image is drawn at its own size, one pixel
  per Canvas unit.
- With one of them, the other follows so the image keeps its proportions.
- With both, the image is stretched to exactly that size.

A file is read once, the first time it is drawn; drawing it again, for example
in every frame of an animation, costs nothing more. An image file may be at
most 64 MB, 8192 pixels on a side, and 32 million pixels in all. A missing file, a file that is not a PNG
or JPEG image, or a negative size raises `GraphicsError`. A PNG with
transparency keeps it. In an SVG file the image is stored inside the file.

## Saving

`canvas.save(path)` writes the Canvas to a file. The format comes from the
extension, whatever its case:

- `.png` — an image exactly `width` × `height` pixels, with no window frame;
- `.svg` — a vector drawing whose lines, circles, and rectangles stay separate
  `line`, `circle`, and `rect` elements, in the same coordinates.

A relative path is relative to the program's working directory. Any other
extension, such as `.jpg` or `.pdf`, raises `GraphicsError`. Both files come
from the same list of drawings the window shows, so the window, the PNG, and
the SVG agree. The SVG contains no script and no external reference.

Save before the window is closed: a closed Canvas can no longer be saved.

## Turtle

A Turtle is a pen you steer. `canvas.turtle()` puts a new one at the origin
facing right (heading 0), with its pen down, drawing black lines of width 1.
Every Turtle keeps its own position, heading, pen, color, and width, and draws
only through its Canvas's `line`.

Headings are degrees, measured like angles in mathematics:

| Heading | Direction |
| ------- | --------- |
| 0       | right (+x) |
| 90      | up (+y) |
| 180     | left (-x) |
| 270     | down (-y) |

- `left(a)` turns counter-clockwise: the heading grows by `a`.
- `right(a)` turns clockwise: the heading shrinks by `a`.
- `setHeading(a)` faces an absolute direction without moving.
- `heading()` always reports a value from 0 up to (not including) 360:
  after `left(450)` it is 90, and `right(180)` from 90 gives 270.

`forward(d)` moves `d` units along the heading: the new position is
`x + d·cos(heading)`, `y + d·sin(heading)`. `backward(d)` moves the opposite
way. Negative distances and negative angles are allowed: `forward(-20)` is
`backward(20)`, and `left(-90)` is `right(90)`.

`moveTo(x, y)` goes straight to a point. `home()` goes to `(0, 0)` and faces
heading 0; it does not clear the Canvas and does not change the pen, color, or
width.

With the pen **down**, every movement (`forward`, `backward`, `moveTo`,
`home`) draws a line from where the Turtle was to where it arrives. With the
pen **up** (`penUp()`), the Turtle only moves. `penDown()` starts drawing
again. `setColor` and `setWidth` apply to the lines drawn after them; the
width must be greater than 0.

Turtle geometry does not depend on the window, the frame rate, or the time:
the same commands always give the same position. The four axis headings are
exact, so a square closes exactly at `(0, 0)`; other angles use ordinary
`Real` arithmetic.

```ahd
bring Graphics

canvas := Graphics.open(500, 500)
turtle := canvas.turtle()
sides := 5
exteriorAngle := 360.0 / sides
count := 0
while count < sides {
    turtle.forward(150)
    turtle.left(exteriorAngle)
    count = count + 1
}
canvas.save("pentagon.svg")
canvas.close()
```

The exterior angles of any regular polygon add up to 360 degrees, so the
Turtle ends where it started, facing the way it began, up to the rounding of
`Real` arithmetic.

There is no visible turtle icon; the Turtle is the pen.

## The window

`Graphics.open` opens a real window and returns once it is ready; you never
write an event loop. `width` and `height` must each be between 1 and 4096,
the title at most 256 characters.

- `canvas.wait()` blocks until the user closes that window, then returns. The
  rest of the program carries on afterwards.
- `canvas.close()` closes the window from the program. Closing a Canvas that is
  already closed does nothing.
- `canvas.isOpen()` is `true` while the Canvas can be drawn on, and `false`
  after `close()`, after `wait()` returns, or after the user closed the window.

On a closed Canvas, drawing (`clear`, `line`, `circle`, `rectangle`, the other
shapes, `text`, `image`, `save`, and a Turtle move with its pen down) and
`every` and `batch` raise `GraphicsError`. A closed Canvas is
never reopened by itself; open a new one instead. After `wait()` returns, or
after the user closed the window, a Turtle whose pen is up may still move and
turn. `canvas.close()` releases the Canvas and every Turtle it gave out: after
it, any call on those Turtles, `x()`, `y()`, and `heading()` included, raises
`GraphicsError`.

Each `Graphics.open` is its own window, so several can be open at once;
closing one does not affect the others. When the program ends, every window it
left open closes. Keep a window on screen with `canvas.wait()`.

## Clicks and key presses

Since v1.8.0, a Canvas reports two kinds of events to a
callback:

```text
canvas.onClick(handler: (x: Real, y: Real) -> Nothing)
canvas.onKey(handler: (key: String) -> Nothing)
```

- `onClick` receives the point that was clicked in the Canvas's own Cartesian
  coordinates: `(0, 0)` is the center, `+x` right, `+y` up. Only a left-button
  press inside the Canvas counts.
- `onKey` receives one normalized key name per key press while the Canvas
  window is focused: `ArrowUp`, `ArrowDown`, `ArrowLeft`, `ArrowRight`,
  `Enter`, `Escape`, `Space`, `Tab`, `Backspace`, `Delete`, `Home`, `End`,
  `PageUp`, `PageDown`, the letters `A`–`Z`, and the digits `0`–`9` (the same
  names as [GUI](GUI.md#key-names)). Holding a key down does not repeat it, and
  there is no key-up event.

The callbacks run inside `canvas.wait()`, one at a time and in order, on the
program's own path of execution; a callback may draw, move a Turtle, clear, or
save. Registering `onClick` or `onKey` again replaces the previous callback. A
long callback delays the handling of the next event. If a callback raises an
error, the Canvas is closed and the error propagates unchanged out of
`wait()`. Events still waiting when the window closes are discarded. A program
that registers no callback waits exactly as before.

```ahd
bring Graphics
from Graphics bring (Canvas, Turtle)

canvas: Canvas := Graphics.open(400, 400)
pen: Turtle := canvas.turtle()

step: Function := (key: String) -> Nothing {
    pen: Global Turtle
    state key {
        condition "ArrowUp" {
            pen.setHeading(90)
        }
        condition "ArrowDown" {
            pen.setHeading(270)
        }
        condition "ArrowLeft" {
            pen.setHeading(180)
        }
        condition "ArrowRight" {
            pen.setHeading(0)
        }
        condition default {
            return
        }
    }
    pen.forward(20)
}

canvas.onKey(step)
canvas.onClick(lambda [@pen] (x: Real, y: Real) -> pen.moveTo(x, y))
canvas.wait()
```

Each arrow key press draws exactly one 20-unit line; no terminal input is
read. See [`examples/v1.8/turtle_events`](../examples/v1.8/README.md). There
is still no input polling (`mouseX`, `isKeyDown`), wheel, or double-click, and
no `Turtle.speed`.

## Pointer movement and dragging

Since v2.8.0, a Canvas also reports where the pointer goes:

```text
canvas.onMove(handler: (x: Real, y: Real) -> Nothing)
canvas.onDrag(handler: (x: Real, y: Real) -> Nothing)
```

- `onMove` receives the pointer's position when it moves over the Canvas with
  no button held.
- `onDrag` receives it when the pointer moves after a left-button press that
  began inside the Canvas. The drag keeps reporting after the pointer leaves,
  so a position may lie outside. A press that began outside does not become a
  drag when the pointer enters, and the press itself is a click, not a drag.

Both use the Canvas's Cartesian coordinates and follow every rule of
[Clicks and key presses](#clicks-and-key-presses): they run inside
`canvas.wait()`, one at a time, and registering again replaces the callback.
A position is reported at most once per tick, and the window ticks 60 times a
second. The press that begins a
drag is a click, so `onClick` is where a stroke starts. If a move callback is
slower than the pointer, the positions it has not reached yet are replaced by
the newest one; the points of a drag are never dropped.

```ahd
bring Graphics
from Graphics bring (Canvas)

canvas: Canvas := Graphics.open(600, 400)
lastX: Real := 0.0
lastY: Real := 0.0

start: Function := (x: Real, y: Real) -> Nothing uses [@lastX, @lastY] {
    lastX = x
    lastY = y
}

draw: Function := (x: Real, y: Real) -> Nothing uses [@canvas, @lastX, @lastY] {
    canvas.line(lastX, lastY, x, y, "blue", 3)
    lastX = x
    lastY = y
}

canvas.onClick(start)
canvas.onDrag(draw)
canvas.wait()
```

See [`examples/v2.8/free_drawing`](../examples/v2.8/README.md).

## Timers and animation

Since v2.8.0, a Canvas can call a Function again and again:

```text
canvas.every(milliseconds: Int, handler: () -> Nothing) -> Int
canvas.cancel(timer: Int)
canvas.batch(handler: () -> Nothing)
```

`canvas.every(milliseconds, handler)` starts a timer and returns its number.
While the program is in `canvas.wait()`, `handler` runs once every
`milliseconds`; 16 is about sixty times a second. The interval must be between
1 millisecond and one day. Timer callbacks run like every other callback: one
at a time, in order with clicks and keys, on the program's own path of
execution, so they need no locking. A callback that takes longer than its
interval simply runs less often; missed turns are not made up.
`canvas.cancel(timer)` stops a timer, also from inside its own callback, and
cancelling a timer that is not running does nothing. A Canvas runs at most 64
timers at a time, and closing it stops them all.

Each timer callback is one frame: everything it draws appears together when it
returns. That is what makes the usual animation, clear and draw again, free of
flicker:

```ahd
bring Graphics
from Graphics bring (Canvas)

canvas: Canvas := Graphics.open(400, 300)
x: Real := -180.0
speed: Real := 4.0

frame: Function := () -> Nothing uses [@canvas, @x, @speed] {
    x += speed
    if x > 180 or x < -180 {
        speed = -speed
    }
    canvas.clear("white")
    canvas.circle(x, 0, 20, "black", "red")
}

canvas.every(16, frame)
canvas.wait()
```

`canvas.batch(handler)` gives any other code the same guarantee: the window
keeps showing its last complete picture while `handler` runs and shows the
result at once when it returns. Use it for a redraw made of many shapes in a
click or key callback. Batches may be nested, and an error inside one still
releases the picture before it propagates.

A cleared Canvas forgets its earlier drawings, so an animation that clears in
every frame can run for as long as the window is open. See
[`examples/v2.8/bouncing_ball`](../examples/v2.8/README.md).

## Errors

Mistakes the compiler can see, such as a missing argument, a `String` where a
`Real` belongs, `null` for a non-nullable parameter, or a member that does not
exist, are compile errors. Problems only the running program can see raise
`GraphicsError`:

- a Canvas size outside 1..4096;
- an unknown color;
- a line width, circle stroke width, or Turtle width that is not greater than 0;
- a negative radius, width, or height;
- a polygon with fewer than 3 or more than 2000 corners, or an odd count of
  numbers;
- a text with a line break, more than 1000 characters, a size outside
  0..2000, or an unknown `align`;
- an image file that is missing, too large, or not a PNG or JPEG image;
- a timer interval outside 1 millisecond..one day, or a 65th timer;
- a save path with an unsupported extension, or a file that cannot be written;
- drawing on a closed Canvas, or using a Turtle whose Canvas was closed with
  `close()`;
- the window helper missing, failing to start, or stopping unexpectedly.

```ahd
bring Graphics
from Graphics bring (GraphicsError)

canvas := Graphics.open(200, 200)
attempt {
    canvas.circle(x: 0, y: 0, radius: 30, stroke: "orange")
}
except GraphicsError as error {
    write(error.message)
}
canvas.close()
```

## How it runs

Each open Canvas is drawn by a small bundled program, `ahdgraphics`, that
AhdCode starts for it and talks to through its standard input and output.
AhdCode itself validates every argument and computes all Turtle geometry;
`ahdgraphics` only keeps the list of drawings, shows it in the window, and
writes PNG and SVG files. It runs no shell command, writes no file except
through `save`, reads none except the images the program draws, and uses no
network.

`ahdcode run`, native builds, and the REPL all use the same implementation. A
native program built with `ahdcode build` records where the installed
`ahdgraphics` is; to run it on another computer, install AhdCode there too.
If the helper cannot be found, `Graphics.open` raises `GraphicsError`.

## Platform notes

- **macOS**: windows open like any application window. On a high-density
  display the window is drawn at the display's resolution, so lines stay sharp;
  saved PNG files are always one pixel per unit. macOS decides which app is in
  front: when another app is active, or when Stage Manager is on, a new Canvas
  window may open behind it or in the Stage Manager strip. Click its icon in the
  Dock or its thumbnail to bring it forward.
- **Application identity** (since v1.9.0): a Canvas window shows
  the AhdCode name and icon — on macOS **AhdCode** in the menu bar and the
  AhdCode icon in the Dock, on Windows and Linux the window icon where the
  system shows one. Drawing, events, and saved files are unchanged.
- **Windows**: needs Windows 10 or newer.
- **Linux**: needs a desktop session with an X11 display (XWayland counts) and
  the system's OpenGL libraries. Without a display, `Graphics.open` raises
  `GraphicsError`.

`AHDCODE_GRAPHICS_HEADLESS=1` opens every Canvas without a window, for
automated tests and servers without a display: drawing and `save` work as
usual, and `wait()` returns at once because there is no window to close. A
headless Canvas has no clock, so its timers do not run.

## Not in Graphics

Graphics deliberately has no sprites, scenes, collision, physics, input
polling (`mouseX`, `isKeyDown`), wheel or double-click events, audio, 3D,
shaders, widgets, fonts to choose from, multi-line or rotated text, image
rotation or cropping, and no `Turtle.speed`. Its input is the click, key,
move, and drag callbacks above, and its only clock is `canvas.every`.
