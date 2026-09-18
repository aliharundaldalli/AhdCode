# Graphics and Turtle

[English] · [Türkçe](README_TR.md)

Two small programs for the v1.6.0 [`Graphics`](../../../docs/GRAPHICS.md)
module: a Cartesian Canvas, its three shapes, and a Turtle pen.

## main.ahd

```bash
ahdcode run main.ahd
```

It opens an 800 × 600 window and draws:

- the `x` and `y` axes through the center, with a tick every 50 units;
- a half-transparent blue circle and a green rectangle (given by its
  lower-left corner) with a black dot at its center;
- a red five-pointed star: forward 220, turn right 144 degrees, five times;
- a spiral of eighteen squares, each a little larger and turned 10 degrees.

It saves the drawing as `graphics-demo.png` and `graphics-demo.svg` in the
current directory, prints a few lines, and waits until you close the window.

Expected output:

```text
Saved graphics-demo.png and graphics-demo.svg.
After five 144-degree turns the star turtle faces 0.0 degrees.
Close the window to finish.
Window closed.
```

The last line appears after you close the window. The star ends facing where
it started because its five turns add up to 720 degrees, twice around.

## turtle_polygon.ahd

```bash
ahdcode run turtle_polygon.ahd
```

A regular polygon from its exterior angle: walking once around any regular
polygon turns the Turtle through 360 degrees, so each of its `n` corners turns
`360 / n` degrees. The program prints the angles and draws the polygon.

```text
A 6-sided polygon turns 60.0 degrees at each corner.
Each interior angle is 120.0 degrees.
Total turning: 360.0 degrees.
```

Change `sides` at the top and run it again.

Neither program is a game: there is no animation loop, input, or sound, only
drawing commands that always produce the same picture.
