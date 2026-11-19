package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"ahdcode/internal/migrate"
)

const migrateUsage = `usage:
  ahdcode migrate new <name> [--dir <directory>]
  ahdcode migrate status --db <file> [--dir <directory>] [--driver sqlite]
  ahdcode migrate up --db <file> [--dir <directory>] [--driver sqlite]
  ahdcode migrate down --db <file> [--dir <directory>] [--steps <n>] [--driver sqlite]
  ahdcode migrate --help

The default directory is migrations. The default driver is sqlite.
status reads the database and does not create a missing file.
up applies pending migrations. If a later migration fails, earlier ones stay applied.
down rolls back the latest applied migration (or --steps of them) and only
when that migration still has the down script recorded at apply time.
A failed up is not rolled back by deleting application data.
BEGIN, COMMIT, ROLLBACK, ATTACH, and writes to ahd_schema_migrations are rejected,
including a trigger body. A trigger that cannot be compiled under that check is rejected too.
`

var migrateFlagAllowed = map[string]map[string]bool{
	"new":    {"--dir": true},
	"status": {"--dir": true, "--db": true, "--driver": true},
	"up":     {"--dir": true, "--db": true, "--driver": true},
	"down":   {"--dir": true, "--db": true, "--driver": true, "--steps": true},
}

func runMigrate(arguments []string, output, errorOutput io.Writer) int {
	if len(arguments) == 0 {
		fmt.Fprint(errorOutput, migrateUsage)
		return 2
	}
	if arguments[0] == "--help" || arguments[0] == "-h" {
		fmt.Fprint(output, migrateUsage)
		return 0
	}
	command := arguments[0]
	switch command {
	case "new", "status", "up", "down":
	default:
		fmt.Fprintf(errorOutput, "ahdcode migrate: unknown subcommand %q\n%s", command, migrateUsage)
		return 2
	}
	for _, argument := range arguments[1:] {
		if argument == "--help" || argument == "-h" {
			fmt.Fprint(output, migrateUsage)
			return 0
		}
	}
	options := migrate.Options{Directory: "migrations", Driver: "sqlite"}
	var name string
	steps := 1
	seen := map[string]bool{}
	for index := 1; index < len(arguments); index++ {
		argument := arguments[index]
		switch argument {
		case "--dir", "--db", "--driver", "--steps":
			if index+1 >= len(arguments) || arguments[index+1] == "" || strings.HasPrefix(arguments[index+1], "--") {
				fmt.Fprintf(errorOutput, "ahdcode migrate %s: %s requires a value\n", command, argument)
				return 2
			}
			if !migrateFlagAllowed[command][argument] {
				fmt.Fprintf(errorOutput, "ahdcode migrate %s: %s is not valid with %s\n", command, argument, command)
				return 2
			}
			if seen[argument] {
				fmt.Fprintf(errorOutput, "ahdcode migrate %s: %s was given more than once\n", command, argument)
				return 2
			}
			seen[argument] = true
			index++
			value := arguments[index]
			switch argument {
			case "--dir":
				options.Directory = value
			case "--db":
				options.Database = value
			case "--driver":
				options.Driver = value
			case "--steps":
				parsed, err := strconv.Atoi(value)
				if err != nil || parsed < 1 || strconv.Itoa(parsed) != value {
					fmt.Fprintf(errorOutput, "ahdcode migrate down: --steps must be a positive integer, not %q\n", value)
					return 2
				}
				steps = parsed
			}
		default:
			if strings.HasPrefix(argument, "-") {
				fmt.Fprintf(errorOutput, "ahdcode migrate %s: unknown flag %q\n", command, argument)
				return 2
			}
			if command != "new" || name != "" {
				fmt.Fprintf(errorOutput, "ahdcode migrate %s: unexpected argument %q\n%s", command, argument, migrateUsage)
				return 2
			}
			name = argument
		}
	}
	if command == "new" {
		if name == "" {
			fmt.Fprintln(errorOutput, "ahdcode migrate new: a migration name is required, as in: ahdcode migrate new create_notes")
			return 2
		}
		file, err := migrate.Create(options.Directory, name)
		if err != nil {
			fmt.Fprintf(errorOutput, "ahdcode migrate new: %s\n", err.Error())
			return 1
		}
		fmt.Fprintf(output, "Created %s\n", file.UpPath)
		fmt.Fprintf(output, "Created %s\n", file.DownPath)
		return 0
	}
	if options.Database == "" {
		fmt.Fprintf(errorOutput, "ahdcode migrate %s: --db is required, as in: ahdcode migrate %s --db app.db\n", command, command)
		return 2
	}
	switch command {
	case "status":
		status, err := migrate.Inspect(options)
		if err != nil {
			fmt.Fprintf(errorOutput, "ahdcode migrate status: %s\n", err.Error())
			return 1
		}
		if !status.DatabaseExists {
			fmt.Fprintln(output, "Database file does not exist. Nothing is applied.")
		}
		if len(status.Applied) == 0 && status.DatabaseExists {
			fmt.Fprintln(output, "No migrations are applied.")
		}
		for _, row := range status.Applied {
			fmt.Fprintf(output, "applied\t%04d_%s\n", row.Version, row.Name)
		}
		for _, file := range status.Pending {
			fmt.Fprintf(output, "pending\t%s\n", file.Label)
		}
		if len(status.Pending) == 0 && (status.DatabaseExists || len(status.Applied) > 0) {
			fmt.Fprintln(output, "No pending migrations.")
		}
		return 0
	case "up":
		applied, err := migrate.Up(options)
		for _, result := range applied {
			fmt.Fprintf(output, "Applied %s\n", result.Label)
		}
		if err != nil {
			fmt.Fprintf(errorOutput, "ahdcode migrate up: %s\n", err.Error())
			return 1
		}
		if len(applied) == 0 {
			fmt.Fprintln(output, "No pending migrations.")
			return 0
		}
		return 0
	case "down":
		rolled, err := migrate.Down(options, steps)
		for _, result := range rolled {
			fmt.Fprintf(output, "Rolled back %s\n", result.Label)
		}
		if err != nil {
			fmt.Fprintf(errorOutput, "ahdcode migrate down: %s\n", err.Error())
			if len(rolled) > 0 {
				fmt.Fprintf(errorOutput, "ahdcode migrate down: %d migration(s) were rolled back before this failure\n", len(rolled))
			}
			return 1
		}
		return 0
	default:
		fmt.Fprintf(errorOutput, "ahdcode migrate: unknown subcommand %q\n%s", command, migrateUsage)
		return 2
	}
}
