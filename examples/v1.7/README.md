# Standard-library completion (v1.7)

[English] · [Türkçe](README_TR.md)

Three small programs for the v1.7.0 additions to
[Math](../../docs/MATH.md), [Numeric](../../docs/NUMERIC.md),
[Time](../../docs/TIME.md), [Data](../../docs/DATA.md), and
[Statistics](../../docs/STATISTICS.md). Each prints the same output compiled
(`ahdcode build`) and with `ahdcode run`. bcrypt is deliberately not shown:
it exists only for compatibility with old password databases, and new
programs should use `Security.passwordHash` (Argon2id).

## scientific_basics.ahd

```bash
ahdcode run scientific_basics.ahd
```

A leaning ladder solved with `sqrt`, `atan2`, and `degrees`; a fraction
reduced with `gcd` and a sum over a common denominator with `lcm`; a plane
normal with `Vector.cross`; and a 90-degree rotation applied with
`Matrix.matvec`.

```text
height 4.0 m, angle 53.13 degrees
hypot check 5.0
asin(0.5) is 30.0 degrees
84/126 = 2/3
1/6 + 1/4 = 5/12
2^10 has log2 10.0
u x v = [0.0, 0.0, 1.0]
|(3, 4, 12)| = 13.0
rotated (1, 0) is (0.0, 1.0)
rotated point length 1.0 (a rotation keeps lengths)
```

## time_iso.ahd

```bash
ahdcode run time_iso.ahd
```

Reads a departure with `Time.parseISO`, adds a flight `Duration`, and writes
the arrival with `toISO()` in the original offset and in UTC. The last lines
show that only the strict ISO form is accepted.

```text
departs  2026-09-18T22:45:00.000+03:00
arrives  2026-09-19T03:15:00.000+03:00 (Istanbul clock)
arrives  2026-09-19T00:15:00.000Z (UTC)
same moment after a round trip: true
reminder 2026-09-18T20:45:00.000+03:00
minutes between reminder and departure: 120
a late reminder would be -7200000 ms, abs 7200000 ms
ok       2026-09-18T22:45:00.000Z
ok       2026-09-18T22:45:00.500-05:30
rejected 2026-09-18 22:45
rejected 18/09/2026
rejected Europe/Istanbul
```

## data_join.ahd

```bash
ahdcode run data_join.ahd
```

Stacks a late student onto a roster with `Table.concat`, joins study records
with `Table.innerJoin(other, leftKey, rightKey)` (the record for student 9 has
no roster row, so the inner join leaves it out), converts the cells explicitly
to Int, and fits `score = slope * hours + intercept` with
`Statistics.linearRegression`.

```text
id,name,hours,score
1,Ada,2,58
2,Alan,5,81
3,Grace,3,66
4,Linus,6,90

correlation 0.9995
score = 7.9 * hours + 42.15
covariance 19.75
```
