# require(...)

[English](REQUIRE.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [CLI](CLI_TR.md) · [Modüller](MODULES_TR.md)

`require("Path/To/File.ahd")` (v0.14), başka bir yerel `.ahd` kaynak
dosyasını **derleme zamanında** bu programa birleştirir. Bu bir çalışma
zamanı include'u, dinamik yükleme, paket importu ya da metinsel bir
önişlemci değildir: derleyici bunu çözer, hedef dosyayı ayrıştırır ve
bildirimlerini, semantik analiz hiç çalışmadan önce aynı programa katlar.
Derlenmiş bir program kendi `.ahd` kaynaklarını asla tekrar okumaz —
zaten üretilmiş bir `ahdcode build` ikili dosyasını orijinal kaynak
ağacını taşımak veya silmek etkilemez.

```ahd
require("Components/Navbar.ahd")

Navbar()
```

## `bring` ile `require` arasındaki fark

- `bring HTTP`, *standart bir AhdCode modülü kullan* anlamına gelir.
- `require("Pages/Home.ahd")`, *bu yerel kaynak dosyasını uygulamaya
  birleştir* anlamına gelir.

`bring` değişmedi: hâlâ standart modülleri (ve öncekiyle aynı şekilde, modül
adıyla getirilen komşu `.ahd` dosyalarını) çözer, her biri kendi dışa
aktarılan arayüzüyle kendi derleme birimi olarak analiz edilir.
`require(...)` aşağıda anlatılan farklı, daha düz bir mekanizmadır — ikisi
birbirinin yerine geçmez ve hiçbiri diğerine dönüştürülerek yeniden
tasarlanmıyor.

## Yalnızca literal yollar

Argüman, tek, düz, derleme-zamanı bir dize literali olmalıdır — enterpolasyon
yok, birleştirme yok, değişken yok:

```ahd
require("Pages/Home.ahd")              // geçerli

path: String := "Pages/Home.ahd"
require(path)                          // reddedilir: PAR014, literal değil

require("Pages/" + name)               // reddedilir: PAR014, literal değil
```

## Yalnızca modül kökü

`require(...)`, tıpkı `bring` gibi bir modül-kökü ifadesidir. Bir `Function`
gövdesi, bir döngü, bir koşul, bir `attempt` bloğu veya başka herhangi bir
iç içe kapsam içinde reddedilir (`PAR005`) — ifade yine de ayrıştırılır, bu
yüzden dosyanın geri kalanı normal şekilde kurtarılır, ama reddedilen bir
`require`'dan hiçbir şey birleştirilmez.

## Yalnızca `.ahd`

Yalnızca AhdCode kaynak dosyaları require edilebilir. `require("data.json")`
veya `require("public/app.css")` reddedilir (`SEM048`); statik varlıklar
ayrı bir konudur — bkz. [Server.static](HTTP_TR.md#statik-dosyalar).

## Uygulama-köküne-göreli çözümleme

Her `require(...)` yolu **uygulama köküne** — giriş `.ahd` dosyasını
içeren dizine — göre çözümlenir, `require(...)`'ı yazan dosyaya göre değil.
Şunu düşünün:

```
/project/app.ahd
/project/Components/Nav.ahd
/project/Shared/Theme.ahd
```

`app.ahd`, `require("Components/Nav.ahd")` içerdiğinde ve
`Components/Nav.ahd`'nin kendisi `require("Shared/Theme.ahd")` içerdiğinde,
ikinci yol yine de `/project/Shared/Theme.ahd`'ye çözümlenir,
`/project/Components/Shared/Theme.ahd`'ye DEĞİL. Bir require yolu, require
zinciri ne kadar derin olursa olsun, yazıldığı her yerde aynı dosya anlamına
gelir.

## Yol güvenliği

Mutlak yollar reddedilir (`SEM048`). Uygulama kökünden kaçan bir yol — düz
`../` geçişi yoluyla ya da kanonik hedefi kök dışına düşen bir sembolik bağ
yoluyla — aynı şekilde reddedilir, `require("../secret.ahd")` dahil.
Kanonik hedefi kök içinde kalan bir sembolik bağ normal şekilde izlenir; bu
özel bir durum değil, sıradan dosya sistemi davranışıdır.

## Tekilleştirme ve kanonik kimlik

Aynı dosyanın iki farklı yazımı — `require("Shared/A.ahd")` ve
`require("Shared/./A.ahd")`, aynı dosyadan veya farklı dosyalardan — tek bir
kanonik kimliğe çözümlenir ve tam olarak bir kez derlenir. Belirli bir
kaynak ağacı için birleştirme sırası deterministiktir: dosya sistemi dizin
numaralandırma sırasını değil, derleyicinin bulduğu açık require kenarlarını
izler.

## Döngüler

Bir require döngüsü (`A.ahd` → `B.ahd` → `C.ahd` → `A.ahd`), zinciri
adlandıran bir derleme hatasıdır (`SEM047`); asla takılmaz ve asla yığın
taşmasına kadar özyinelemez.

## Eksik dosyalar

`require("Pages/Missing.ahd")`, isteyen dosyayı, yazıldığı gibi literal
yolu ve derleyicinin bulmayı beklediği çözümlenmiş yolu adlandırarak temiz
bir şekilde başarısız olur (`SEM046`) — asla çıplak bir Go hatası değil.

## Paylaşılan bildirimler, dosyaya özgü `bring`

Her require edilen dosyanın üst düzey bildirimleri — `Function`, `Class`,
modül sabitleri — **tek bir paylaşılan uygulama ad alanına** katılır.
`Components/Card.ahd`'de bildirilen bir fonksiyon, hiçbir paket veya ad
alanı sözdizimi olmadan başka herhangi bir require edilen dosyadan niteliksiz
olarak çağrılır:

```ahd
// Components/Card.ahd
Card: Function := (title: String) -> HTMLNode {
    ...
}

// app.ahd
require("Components/Card.ahd")
Card("Hello")
```

İki require edilen dosya arasında yinelenen bir üst düzey bildirim adı, tek
bir dosyanın fırlatacağı aynı `SEM002` yineleme hatasıdır.

**`bring` bu paylaşımı izlemez.** Require edilen bir dosya, kullandığı
standart modülleri kendisi bildirmelidir:

```ahd
// Components/Card.ahd
bring HTML
from HTML bring HTMLNode
```

`app.ahd`'nin `bring HTTP` yapması, `Components/Card.ahd`'yi `HTML.*` veya
`HTMLNode` kullanmak için kendi `bring HTML`'ini bildirmekten muaf tutmaz —
ve tersi de aynı derecede doğrudur. Yalnızca başka bir dosyanın `bring`'i
üzerinden erişilebilen bir ada başvuran bir dosya, ihtiyaç duyduğu modülü
adlandırarak `SEM049` ile başarısız olur.

## Çevrimdışı, deterministik, paket yöneticisi yok

`require(...)` asla ağa dokunmaz, asla bir kayıt defterine başvurmaz ve bir
programın kendi `require(...)` ifadelerinin adlandırdığı yolların ötesinde
aday aramak için dosya sistemini asla taramaz. `ahd.toml` yoktur, manifesto
yoktur, semantik sürüm yoktur, kilit dosyası yoktur, uzak/Git/URL bağımlılığı
yoktur ve paket önbelleği yoktur — `require(...)` yerel uygulama kaynak
birleştirmesidir, başka bir şey değil.

## `ahdcode dev` ve `require(...)`

`ahdcode dev`, giriş dosyasını, çözümlenmiş `require(...)` grafiğini ve en
son derleme denemesinin adlandırdığı ama henüz bulamadığı herhangi bir
`require(...)` hedefini izler; böylece eksik bir require edilen dosyayı
oluşturmak, onu require eden dosyaya dokunmadan otomatik olarak yeniden
derler. Bkz. [CLI rehberi](CLI_TR.md#dev-izleme-kapsamı).
