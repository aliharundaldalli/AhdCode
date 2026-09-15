# v1.5.0 Terminal tanıtımı

[English](README.md) · [Türkçe]

[`Terminal`](../../../docs/TERMINAL_TR.md) standart modülünün her parçasını,
değişmeyen `write` ile yan yana gösteren küçük bir program.

## Çalıştırma

```bash
ahdcode run main.ahd
```

Sonra standart çıktıyı yönlendirerek ve bir kez de renksiz olarak yeniden
çalıştırın:

```bash
ahdcode run main.ahd > output.txt
NO_COLOR=1 ahdcode run main.ahd
```

## Neye bakmalı

| Satır | Gösterdiği |
| --- | --- |
| 1 | `write` eskisiyle tamamen aynı davranır |
| 2–4 | varsayılan ayırıcı, özel ayırıcı ve boş sonla `Terminal.emit` |
| 5 | `Terminal.error` standart hataya yazar; çıktı yönlendirildiğinde bile ekranda kalır |
| 6–8 | `isInteractive`, `width`/`height` ve `supportsColor` standart çıktıyı anlatır |
| 9 | `Terminal.style` terminalde sözcükleri renklendirir, başka yerde düz metin döndürür |
| 10 | Unicode aynen yazılır |
| 11 | `Terminal.pretty` bir `Pair<String, List<Int>>` değerini birkaç satıra yayar |

Yönlendirildiğinde 6–8. satırlar şöyledir:

```text
6. interactive: false
7. size: unknown, because standard output is not a terminal
8. color: false
```

ve `output.txt` hiçbir kaçış dizisi içermez.
