# File ve Path modülleri

[English](FILESYSTEM.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [Hatalar](ERRORS_TR.md)

Modülleri açıkça içe aktarın:

```ahd
bring Path
bring File
from File bring FileError
```

## Path

`Path`, ana bilgisayar işletim sistemine duyarlı (host-operating-system-
aware), saf yol işlemleri gerçekleştirir:

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

Metin UTF-8'dir. `File.list`, doğrudan girdi adlarını kararlı, artan
sözlüksel (ascending lexical) sırada döndürür; özyinelemeli (recursive)
değildir. Göreli yollar, süreç çalışma dizinini (process working directory)
kullanır. Bir REPL'de bu, `ahdcode`'un başlatıldığı dizindir.

```ahd
File.createDir("notes")
File.writeText("notes/today.txt", "first")
File.append("notes/today.txt", " second")
write(File.readText("notes/today.txt"))
write(File.list("notes"))
```

`File.exists`, eksik bir yol için `false` döndürür. Diğer File işlemlerinin
hataları, `IOError` ve `Error`'dan türeyen `FileError`'ı fırlatır:

```ahd
attempt {
    File.readText("missing.txt")
}
except FileError as error {
    write(error.message)
}
```

File işlemleri, hiçbir zaman ana bilgisayar (host) hata nesnesini
göstermez. v0.1.7, kasıtlı olarak özyinelemeli listeleme, ikili (binary)
G/Ç, izinler API'si, dizin gezinme (directory walking) veya geniş bir OS
modülüne sahip değildir.
