# Service standart modülü

[English](SERVICE.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [Disk](DISK_TR.md) · [Process](PROCESS_TR.md) · [Hatalar](ERRORS_TR.md)

`Service` (v2.7.0), bir **Linux systemd** biriminin durumunu bildirir — bir
servisin etkin mi, çalışıyor mu, başarısız mı ya da etkinleştirilmiş mi
olduğunu. **Salt okunurdur**: hiçbir şeyi asla başlatmaz, durdurmaz, yeniden
başlatmaz, etkinleştirmez, devre dışı bırakmaz ya da yeniden yüklemez. Açıkça içe
aktarın:

```ahd
bring Service
from Service bring (ServiceInfo, ServiceError)
```

Kanonik modül kimliği `builtin:Service`'tir; bir kardeş `Service.ahd` onu
gölgeleyemez. (v2.7.0'dan itibaren `Service` bir standart modül adıdır: yerel bir
`Service.ahd` artık `bring Service`'in yüklediği şey değildir — böyle bir dosyayı
yeniden adlandırın. Web projelerinin `require` ile yüklenen `Services/`
klasörleri bundan etkilenmez.)

## Yüzey

```text
Service.status(name: String) -> ServiceInfo

ServiceInfo.name()        -> String   // systemd'nin çözdüğü birim, örn. "nginx.service"
ServiceInfo.activeState() -> String   // normalleştirilmiş: aşağıya bakın
ServiceInfo.subState()    -> String   // systemd'nin alt düzey durumu, örn. "running", "exited", "dead"
ServiceInfo.running()     -> Bool     // subState() == "running"
ServiceInfo.enabled()     -> Bool     // birim açılışta başlayacak şekilde etkinleştirilmiş

ServiceError  (Error'dan türer)
```

`ServiceInfo` yalnızca `Service.status` tarafından üretilen opak bir değerdir.

```ahd
unit: ServiceInfo := Service.status("nginx.service")
write(unit.name() + " is " + unit.activeState() + " (" + unit.subState() + ")")
```

(Üyenin adı `state()` değil `activeState()`'tir: `state`, AhdCode'da ayrılmış bir
sözcüktür. Ad, systemd'nin kendi `ActiveState` özelliğiyle eşleşir.)

## Durumlar

`activeState()` sabit bir sözcük dağarcığına normalleştirilir:

| `activeState()` | systemd ActiveState |
|---|---|
| `active` | `active`, `reloading`, `refreshing` |
| `inactive` | `inactive` |
| `failed` | `failed` |
| `activating` | `activating` |
| `deactivating` | `deactivating` |
| `unknown` | diğer her şey |

`subState()`, systemd'nin kendi alt düzey durumudur (`running`, `exited`,
`dead`, `failed`, `start`, `stop-sigterm`, …); yalnızca düz küçük harfli metin
olduğunda bildirilir, aksi halde `unknown` olur. `running()` yalnızca `running`
için true'dur: işini bitirmiş etkin bir tek seferlik birim (`active`/`exited`)
çalışıyor sayılmaz.

`enabled()`, systemd'nin `UnitFileState` değeri `enabled` ya da
`enabled-runtime` olduğunda true'dur. `disabled`, `masked` ve `static` olan ya da
bir soket veya başka bir birim tarafından başlatılan birimler için `false`'tur
(örneğin `systemd-journald.service` ya da `ssh.socket`'in başlattığı sistemlerde
`ssh.service`) — bunlar yine de etkin ve çalışıyor olabilir.

Soneki olmayan bir birim adını systemd çözer (`nginx` → `nginx.service`);
`name()` çözülmüş adı bildirir.

## Nasıl çalışır

`Service.status`, mutlak `systemctl` programını (`/usr/bin/systemctl` ya da
`/bin/systemctl`) [`Process.run`](PROCESS_TR.md) ile aynı kabuksuz mekanizmayla
çalıştırır: tek bir program, sabit bir argüman listesi, 10 saniyelik bir zaman
aşımı ve sınırlı çıktı. Yalnızca makinenin okuyabileceği özellikleri ister:

```text
systemctl show --no-pager --property=Id,LoadState,ActiveState,SubState,UnitFileState -- <name>
```

İnsanlara yönelik, renkli ve yerelleştirilmiş olabilen `systemctl status`
çıktısını asla ayrıştırmaz. Ad, `--`'den sonra tek bir düz argümandır ve önce
doğrulanır: yalnızca ASCII harfler, rakamlar ve `:_.@-` karakterlerine izin
verilir; en fazla 255 karakter olabilir, `-` ya da `.` ile başlayamaz. Boşluklar,
`;`, `$`, tırnaklar, eğik çizgiler ve denetim karakterleri herhangi bir süreç
çalışmadan önce reddedilir; bu yüzden hiçbir ad bir komuta ya da bir seçeneğe
dönüşemez.

## Hatalar

Her hata, iletisi `service "<ad>" status failed: <neden>` olan bir `ServiceError`
fırlatır:

```text
service "ghost.service" status failed: service not found
service "nginx" status failed: systemd is not available on this system
service "nginx" status failed: permission denied
service "nginx" status failed: the inspection timed out after 10 seconds
service "a;b" status failed: the service name contains ';'; only letters, digits, and ':_.@-' are allowed
service "nginx" status failed: service inspection is supported only on Linux with systemd; this system is darwin
```

Ham `systemctl` hata metni asla iletinin parçası değildir.

## Platform desteği

| Platform | Davranış |
|---|---|
| systemd'li Linux | desteklenir |
| systemd'siz Linux (konteynerler, başka init sistemleri) | `ServiceError`: systemd kullanılamıyor |
| macOS, Windows, diğerleri | `ServiceError`: yalnızca systemd'li Linux'ta desteklenir |

AhdCode v2.7.0, diğer platformlarda servis durumunu `launchctl` ya da `sc.exe`
çıktısından tahmin etmeye bilinçli olarak çalışmaz.

## Tasarım gereği salt okunur

`Service.start`, `stop`, `restart`, `enable`, `disable` ya da `reload` yoktur. Bir
servisi değiştirmesi gereken bir işçi bunu, kendi izin listesiyle,
[`Process.run`](PROCESS_TR.md) üzerinden açıkça yapar — örneğin
`Process.run("/usr/bin/systemctl", ["restart", validatedUnit])`.

## Güvenlik: güvenilmeyen girdi için yalnızca işçi

`Service.status`'u asla güvenilmeyen bir web isteğinin doğrudan arkasında değil,
doğrulanmış, izin listesindeki birimleri inceleyen özel bir işçide tutun:

```text
web uygulaması
    -> kimliği doğrulanmış kontrol kanalı
        -> özel bir AhdCode işçisi
            -> doğrulanmış / izin listesindeki servis adı
                -> Service.status
```

```ahd
bring Service
from Service bring (ServiceInfo, ServiceError)

for unit in ["caddy.service", "ahdcode-org.service"] {
    attempt {
        info: Local ServiceInfo := Service.status(unit)
        if info.activeState() != "active" {
            write("ALERT: " + info.name() + " is " + info.activeState())
        }
    }
    except ServiceError as failure {
        write(failure.message)
    }
}
```

## Bu sürümde yok

Servisleri değiştirme, birimleri listeleme, günlükleri (journald) okuma,
zamanlayıcıların takvimleri ve systemd dışındaki servis yöneticileri
`Service`'in parçası değildir.
