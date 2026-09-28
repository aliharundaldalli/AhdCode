# Process standart modülü

[English](PROCESS.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [File](FILESYSTEM_TR.md) · [Hatalar](ERRORS_TR.md)

`Process` (v2.5.0), harici bir programı açık bir argüman listesiyle çalıştırır,
bitmesini bekler ve çıkış kodunu, standart çıktısını ve standart hatasını
döndürür — sonlu bir zaman aşımıyla, sonlu bir çıktı bütçesiyle ve **asla bir
kabuk (shell) üzerinden değil**. Açıkça içe aktarın:

```ahd
bring Process
from Process bring (ProcessResult, ProcessError)
```

Kanonik modül kimliği `builtin:Process`'tir; bir kardeş `Process.ahd` onu
gölgeleyemez. (v2.5.0'dan itibaren `Process` bir standart modül adıdır: yerel
bir `Process.ahd` artık `bring Process`'in yüklediği şey değildir — böyle bir
dosyayı yeniden adlandırın.)

## Yüzey

```text
Process.run(command: String,
            args: List<String> = [],
            timeoutSeconds: Int = 30,
            maxOutputBytes: Int = 1048576) -> ProcessResult

ProcessResult.exitCode() -> Int
ProcessResult.stdout()   -> String
ProcessResult.stderr()   -> String

ProcessError  (Error'dan türer)
```

`ProcessResult` yalnızca `Process.run` tarafından üretilen opak bir değerdir.
Her AhdCode çağrısında olduğu gibi argümanlar ya tümüyle konumsal ya da
tümüyle adlıdır:

```ahd
result: ProcessResult := Process.run(
    command: "/usr/bin/systemctl",
    args: ["status", "example.service", "--no-pager"],
    timeoutSeconds: 15,
    maxOutputBytes: 1048576
)
write(str(result.exitCode()))
write(result.stdout())
write(result.stderr())
```

## Asla kabuk yok

`Process.run`, çalıştırılabilir dosyayı **doğrudan** başlatır ve `args`'ın her
öğesini ayrı bir argüman olarak verir. Modülün hiçbir yerinde `/bin/sh -c`,
`cmd.exe /C` ya da PowerShell yoktur ve bir komut satırını tek bir String
olarak kabul eden hiçbir işlev yoktur. Bir argümandaki kabuk sözdizimi sıradan
metindir:

```ahd
result := Process.run("/bin/echo", ["; touch SHOULD_NOT_EXIST", "$(whoami)"])
write(result.stdout())   // ; touch SHOULD_NOT_EXIST $(whoami)
```

Hiçbir şey çalıştırılmaz, genişletilmez, joker olarak açılmaz ya da
yönlendirilmez. `*`, `|`, `&&`, `;`, `$VAR`, `%VAR%`, ters tırnaklar ve
tırnaklar programa tam yazıldığı gibi ulaşır. Boşluk içeren bir argüman tek
argüman olarak kalır. Argüman listelerini veriden kurun; asla bir komut
satırını metin birleştirerek oluşturmayın.

**Windows**'ta `.bat` ve `.cmd` dosyaları `ProcessError` ile reddedilir:
Windows bunları argümanları yeniden ayrıştıran `cmd.exe` üzerinden çalıştırır.
Bunun yerine gerçek çalıştırılabilir dosyayı çalıştırın.

## Çalıştırılabilir dosyanın bulunması

- Mutlak bir yol (`/usr/bin/systemctl`) olduğu gibi kullanılır.
  **Ayrıcalıklı işçiler her zaman mutlak yol kullanmalıdır.**
- Ayırıcı içeren göreli bir yol (`./bin/tool`, `bin/tool`) çalışma dizinine
  göre çözülür.
- Yalın bir ad (`git`) `PATH` içinde aranır. Geçerli dizindeki bir program
  asla yalın adla bulunmaz.

Alt sürecin çalışma dizini, programın çalışma dizinidir (REPL'de oturum
dizini). Standart girdi boş aygıttır; girdi bekleyen bir alt süreç askıda
kalmak yerine dosya sonunu görür. Ortam değişkenleri olduğu gibi devralınır;
`Process` onları okumak, yazdırmak ya da değiştirmek için hiçbir yol sunmaz ve
hata iletileri onları asla içermez.

## Çıkış kodları ve hatalar

Çalışıp — herhangi bir durumla — çıkan bir program bir hata değil, bir
**sonuçtur**:

```ahd
check := Process.run("/usr/bin/systemctl", ["is-active", "--quiet", "example.service"])
if check.exitCode() == 0 {
    write("active")
} else {
    write("not active: " + str(check.exitCode()))
}
```

`exitCode()`, programın çıkış durumudur. Unix'te işlemediği bir sinyalle
öldürülen bir program `-1` bildirir.

`ProcessError` yalnızca döndürülecek sıradan bir sonuç olmadığında fırlatılır:

| Durum | İletideki neden |
|---|---|
| çalıştırılabilir dosya yok | `executable not found` |
| çalıştırılabilir değil / izin yok | `permission denied` |
| çalışma dizini yok | `executable or working directory not found` |
| boş komut, komutta ya da argümanda NUL baytı | `the command is empty`, `argument 2 contains a NUL byte` |
| zaman aşımı | `timed out after N seconds (timeoutSeconds); the process was terminated` |
| çıktı bütçesi aşıldı | `output limit exceeded: stdout and stderr together produced more than N bytes (maxOutputBytes); the process was terminated` |
| aralık dışı sınır | `timeoutSeconds must be between 1 and 3600; received 0` |
| Windows toplu iş dosyası | `batch files run through cmd.exe; Process.run never uses a shell` |

Her ileti `run "<komut>" failed: <neden>` biçimindedir. Argümanlar gizli
bilgi taşıyabileceği için argüman listesi iletilerde bilinçli olarak
tekrarlanmaz.

## Zaman aşımı

Her çalıştırmanın sonlu bir zaman aşımı vardır: `timeoutSeconds` varsayılan
olarak **30**'dur ve **1 ile 3600** arasında olmalıdır. Süre dolduğunda alt
süreç öldürülür, ardından beklenir (toplanır); geride zombi süreç kalmaz ve
`ProcessError` fırlatılır.

**Unix**'te alt süreç kendi süreç grubunda başlatılır ve grubun tamamı
öldürülür; böylece başlattığı yardımcılar da sonlandırılır (bilinçli olarak
başka bir oturuma ya da gruba geçen bir alt süreç bu güvencenin dışındadır).
**Windows**'ta yalnızca doğrudan alt süreç öldürülür; onun alt süreçlerinin
sonlandırılması garanti edilmez. Hayatta kalan bir alt süreç çıktı borularını
açık tutarsa `run`, onu beklemek yerine iki saniyelik bir süreden sonra
okumayı bırakır.

## Çıktı bütçesi

`stdout` ve `stderr` bellekte toplandığı için birlikte sınırlanır:
`maxOutputBytes` (varsayılan **1 MiB**, izin verilen **1 – 67 108 864**) iki
akışın toplamıdır. Daha fazlasını üreten bir program öldürülür ve
`ProcessError` fırlatılır. Çıktı **asla sessizce kesilmez** — ya tamamını ya da
bir hata alırsınız.

Çıktı UTF-8 metin olarak çözülür; geçerli UTF-8 olmayan bayt dizileri
`U+FFFD` ile değiştirilir. `Process`, çıktısı metin olan programlar içindir;
ikili çıktıyı programın kendi seçenekleriyle bir dosyaya yönlendirin.

## Güvenlik modeli: bir politika değil, bir yetenek

`Process` bir yetenektir. Hangi programların ya da argümanların güvenli
olduğuna bilinçli olarak **karar vermez** — dilin içinde genel bir izin listesi
yoktur — çünkü bunu yalnızca uygulama bilir. Bir barındırma kontrol paneli gibi
ayrıcalıklı her şey için önerilen mimari:

```text
Web uygulaması (ayrıcalıksız)
    -> kimliği doğrulanmış, dar bir kontrol kanalı (örneğin bir Unix soketi)
        -> özel bir AhdCode işçisi (ayrıcalıklı)
            -> izin listesindeki sistem programları, mutlak yollar, doğrulanmış argümanlar
```

- İstekleri işleyen web koduna `Process.run`'a doğrudan bir yol vermeyin.
- İşçide her isteği sabit bir çalıştırılabilir dosyaya (mutlak yolla) eşleyin
  ve argümanlarını doğrulanmış değerlerden kurun:

```ahd
bring Process
bring Regex
from Process bring ProcessResult
from Regex bring Pattern

restartService: Function := (unit: String) -> ProcessResult {
    // Accept only plain unit names such as "ata-panel.service".
    unitName: Local Pattern := Regex.compile(r'^[a-z0-9][a-z0-9-]*\.service$')
    if not unitName.matches(unit) {
        toss ValueError("refused unit name: " + unit)
    }
    // A fixed program, by absolute path; the unit is one literal argument.
    return Process.run(command: "/usr/bin/systemctl", args: ["restart", unit], timeoutSeconds: 60)
}

attempt {
    restartService("x.service; rm -rf /")
} except ValueError as error {
    write(error.message)
}
```

- İstekten gelen her argümanı doğrulayın; `-` ile başlayan bir argüman, kabuk
  olmasa bile hedef programın bir seçeneği olabilir.
- `timeoutSeconds` ve `maxOutputBytes` değerlerini işin izin verdiği kadar
  küçük tutun.

## Bu sürümde yok

Bir kabuk ya da komut satırı String API'si, süreçler arası boru hatları,
standart girdi, akış halinde çıktı, arka plan ya da bağımsız süreçler, çağrı
başına ortam değişikliği, bir çalışma dizini parametresi, süreç listeleme ya da
sinyal gönderme, servis yöneticisi (`systemd`) soyutlamaları, SSH ve
konteynerler `Process`'in parçası değildir.
