# AhdCode v0.1 examples

[Back to project README](../../README.md)

These programs are small, working introductions to the v0.1 language.

```bash
ahdcode run examples/v0.1/01_hello.ahd
```

The input examples can be run interactively:

```bash
ahdcode run examples/v0.1/02_input.ahd
ahdcode run examples/v0.1/14_grade_app.ahd
```

| Example | Topic |
|---|---|
| `01_hello.ahd` | declarations, interpolation, output |
| `02_input.ahd` | `take`, `int`, terminal input |
| `03_grade_average.ahd` | Lists and Fundamentals reductions |
| `04_loops.ahd` | `while`, post-check `until`, `for`, `between` |
| `05_functions.ahd` | Functions, defaults, named calls, callbacks |
| `06_list_api.ahd` | List mutation, map/filter, deterministic shuffle |
| `07_string_api.ahd` | immutable String operations |
| `08_pair.ahd` | insertion-ordered Pair workflow |
| `09_class.ahd` | structure attributes and methods |
| `10_errors.ahd` | `attempt`, `except`, `ultimately`, `toss` |
| `11_modules.ahd` | direct import from `Greeting.ahd` |
| `12_math.ahd` | explicit Math module and seeding |
| `13_null_safety.ahd` | flow-sensitive null refinement |
| `14_grade_app.ahd` | compact interactive CLI application |
| `15_time.ahd` | Time module: DateTime, Duration, Calendar, monotonic |
| `16_latex.ahd` | Latex module: module alias, helpers, PDF, LatexError |

`Greeting.ahd` is the sibling module used by `11_modules.ahd`.
