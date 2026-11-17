# Text, shapes, pointer events, and timers (v2.8)

[English] · [Türkçe](README_TR.md)

Three small programs for the v2.8 additions to the
[Graphics](../../docs/GRAPHICS.md) Canvas.

## shapes_and_text

```bash
cd shapes_and_text
ahdcode run main.ahd
```

One page with `Canvas.text` in three alignments, a triangle and a computed
five-pointed star drawn with `Canvas.polygon`, an ellipse, and an arc. The page
is saved as `shapes.png` and `shapes.svg`, then drawn again inside itself with
`Canvas.image` and saved as `shapes-with-image.png`.

## bouncing_ball

```bash
cd bouncing_ball
ahdcode run main.ahd
```

A ball bounces off the four walls. `Canvas.every(16, frame)` redraws the
Canvas about sixty times a second, and each call is shown as one complete
frame, so clearing and redrawing does not flicker. Space pauses with
`Canvas.cancel` and resumes with a new timer, S saves `bouncing-ball.png`, and
Escape ends.

## free_drawing

```bash
cd free_drawing
ahdcode run main.ahd
```

Hold the mouse button and move to draw. `Canvas.onClick` starts a stroke,
`Canvas.onDrag` adds a segment for every reported position, and
`Canvas.onMove` ends the stroke. C clears the page, S saves `drawing.png`, and
Escape ends.

The Functions in these programs name the module bindings they use with
`uses [@canvas, ...]`, on the header line or on a line of its own.
