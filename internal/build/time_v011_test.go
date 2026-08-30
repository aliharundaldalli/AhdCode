package build

import "testing"

func TestTimeUTCAndTimestampRunAsNativeExecutables(t *testing.T) {
	cases := []program{
		{
			name: "current UTC and timestamp clocks agree",
			sources: map[string]string{"main.ahd": timeImports + `current: DateTime := Time.utc()
write(current.offsetMinutes)
write(abs(Time.timestamp() - current.timestamp()) <= 2000)
`},
			expected: "0\ntrue\n",
		},
		{
			name: "UTC fixed offsets and timestamps preserve the instant",
			sources: map[string]string{"main.ahd": timeImports + `utc: DateTime := Time.dateTimeUTC(year: 1970, month: 1, day: 1)
east: DateTime := Time.dateTimeOffset(year: 1970, month: 1, day: 1, offsetMinutes: 180, hour: 3)
west: DateTime := utc.toOffset(-300)
write(utc.timestamp())
write(east.timestamp())
write(utc.sameMoment(east))
write(west.hour)
write(west.day)
write(west.offsetMinutes)
write(west.toUTC().sameMoment(utc))
write(Time.between(west, east).milliseconds)
`},
			expected: "0\n0\ntrue\n19\n31\n-300\ntrue\n0\n",
		},
		{
			name: "negative timestamps round trip through UTC",
			sources: map[string]string{"main.ahd": timeImports + `value: DateTime := Time.fromTimestamp(-1)
write(value.toString())
write(value.millisecond)
write(value.offsetMinutes)
write(value.timestamp())
`},
			expected: "1969-12-31 23:59:59\n999\n0\n-1\n",
		},
		{
			name: "invalid fixed offset is ValueError",
			sources: map[string]string{"main.ahd": timeImports + `Time.dateTimeOffset(year: 2026, month: 1, day: 1, offsetMinutes: 841)
`},
			exitCode: 1, errorClass: "ValueError",
		},
		{
			name: "fixed offset inclusive boundaries are valid",
			sources: map[string]string{"main.ahd": timeImports + `write(Time.dateTimeOffset(year: 2026, month: 1, day: 1, offsetMinutes: -840).offsetMinutes)
write(Time.dateTimeOffset(year: 2026, month: 1, day: 1, offsetMinutes: 840).offsetMinutes)
`},
			expected: "-840\n840\n",
		},
		{
			name: "unrepresentable timestamp is ValueError",
			sources: map[string]string{"main.ahd": timeImports + `Time.fromTimestamp(9223372036854775807)
`},
			exitCode: 1, errorClass: "ValueError",
		},
	}
	runProgramCases(t, cases)
}
