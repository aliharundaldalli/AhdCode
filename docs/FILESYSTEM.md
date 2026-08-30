# File and Path modules

[Back to README](../README.md) · [Modules](MODULES.md) · [Errors](ERRORS.md)

Import the modules explicitly:

```ahd
bring Path
bring File
from File bring FileError
```

## Path

`Path` performs pure, host-operating-system-aware path operations:

```text
Path.join(parts: List<String>) -> String
Path.ext(path: String)         -> String
Path.base(path: String)        -> String
Path.dir(path: String)         -> String
```

```ahd
filePath := Path.join(["reports", "result.txt"])
write(Path.ext(filePath))
write(Path.base(filePath))
write(Path.dir(filePath))
```

## File

```text
File.exists(path: String)                     -> Bool
File.readText(path: String)                   -> String
File.writeText(path: String, content: String) -> Nothing
File.append(path: String, content: String)    -> Nothing
File.delete(path: String)                     -> Nothing
File.createDir(path: String)                  -> Nothing
File.list(path: String)                       -> List<String>
```

Text is UTF-8. `File.list` returns immediate entry names in stable ascending
lexical order; it is not recursive. Relative paths use the process working
directory. In a REPL that is the directory from which `ahdcode` was launched.

```ahd
File.createDir("notes")
File.writeText("notes/today.txt", "first")
File.append("notes/today.txt", " second")
write(File.readText("notes/today.txt"))
write(File.list("notes"))
```

`File.exists` returns `false` for a missing path. Failures of the other File
operations raise `FileError`, which derives from `IOError` and `Error`:

```ahd
attempt {
    File.readText("missing.txt")
}
except FileError as error {
    write(error.message)
}
```

File operations never expose host error objects. v0.1.7 deliberately has no
recursive listing, binary I/O, permissions API, directory walking, or broad OS
module.
