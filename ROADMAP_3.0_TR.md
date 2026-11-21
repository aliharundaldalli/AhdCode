# AhdCode 2.7 → 3.0 yol haritası

Hazırlanma tarihi: 17 Kasım 2026. Güncelleme: 18 Kasım 2026. v2.8.0 kaynak
sürüm olarak yayımlandı. En güncel kurulabilir sürüm v2.7.0'dır.

3.0'ın hedefi tek cümle: **aynı AhdCode GUI programından Android ve iOS uygulaması çıkarmak.**
Oraya varmadan önce üç şey yerine oturmalı: Graphics ve GUI'nin temel eksikleri,
Web tarafının günlük kullanım pürüzleri ve dilin kendisinin dondurulması.

Bu belgedeki "doğrulandı" notu, maddenin 17 Kasım'da v2.7.0 üzerinde program
çalıştırılarak ya da kaynak okunarak görüldüğünü belirtir. "Öneri" notu benim
tasarım teklifimdir; kabul edip etmemek size ait.

## 1. Bugünkü durum

Sağlam olan kısımlar:

- Çekirdek dil, standart kütüphane, biçimlendirici, dil sunucusu ve 2097 Go testi.
- Konsol programları, dosyalar, SQLite, GUI formları, tablo, iletişim kutuları,
  HTTP sunucusu ve Web çatısı: "sıfırdan programlama" serisi için yazdığım 26
  örnek programın tamamı v2.7.0'da çalışıyor (GUI örnekleri tıklama ve tuş
  olaylarıyla, web örnekleri gerçek isteklerle sınandı).
- Masaüstü paketleme (`ahdcode package`) macOS, Windows ve Linux için hazır.

Eksik ya da pürüzlü olan kısımlar aşağıda sürümlere dağıtıldı.

## 2. Bulgular

### 2.1 Dil ve tanılama

| # | Bulgu | Durum |
|---|---|---|
| D1 | Ayrılmış bir sözcük ad olarak kullanılınca (örneğin `not: Int := 70`) derleyici 12–30 satırlık zincirleme `PAR001/PAR002/PAR003` hatası veriyor. Hiçbir satır "`not` ayrılmış bir sözcüktür" demiyor. | doğrulandı |
| D2 | Tek bir yazım hatası birçok ardışık hataya yol açıyor; ilk gerçek hata kalabalıkta kayboluyor. | doğrulandı |
| D3 | Sunucu başlatan bir programda `site.start()` öncesindeki `write(...)` çıktısı, çıktı bir dosyaya ya da boruya yönlendirildiğinde hiç görünmüyor. `ahdcode init web` starter'ının yazdığı adres satırı da bu yüzden günlüklerde yok. | doğrulandı |
| D4 | `ahdcode format` tek seferde tek dosya kabul ediyor; bir klasörü ya da birden çok dosyayı biçimlendirmek için döngü yazmak gerekiyor. | doğrulandı |
| D5 | Belirtim `AHDCODE_LANGUAGE_SPEC_v0.1.md` adını taşıyor ve başlığı "v1.0.0" diyordu; v2.1–v2.7 modülleri belirtimde hiç geçmiyordu, `uses [@ad]` ile `Global` ilişkisi (§9.3) ve niteleyicili çıkarımlı bildirim (§8.1) yazılı değildi. İlk taslaktaki "null olabilen türler, lambda yakalamaları, `uses` ve Protocol Methods belirtimin dışında dağınık" tespiti yanlıştı: bunlar §7.2, §54, §15 ve §47'de zaten var. | v2.8'de düzeltildi (§91–§94, §9.3, §8.1, §3.3); dosya adı bilinçli olarak korunuyor |
| D8 | `ahdcode format`, `uses` listesi ya da `{` ayrı satırda yazılmış bir Function'ı (belirtimin §15'teki kendi örneğinin biçimi) geçersiz kaynağa çevirip dosyanın üzerine yazıyordu; belirteçler kayboluyordu. v2.7.0'da yeniden üretildi. | v2.8'de düzeltildi; biçimlendirici artık içerik denetiminden geçmeyen düzeni yazmıyor (`FMT001`) |
| D9 | `ad: Constant := değer` (türsüz, niteleyicili bildirim) modül kökünde dahili `IR002` hatası veriyordu. | v2.8.0'da düzeltildi: tür çıkarımı, sabit ifade denetimi, bağımlılık sırası; döngü ve erken kullanım `SEM010` |
| D10 | `{` işaretini alt satıra yazmak (`if x` / `{`) blok başına 4–5 zincirleme hata üretiyordu. | v2.8'de düzeltildi: blok başına tek hata ve ipucu |
| D7 | GUI belgeleri, öğrenci rehberi ve örnekler geri çağırmalarda her bileşen için bir `ad: Global Tür` satırı yazıyor; beş bileşene dokunan bir işleyici beş satır bildirimle başlıyor. Dil bunu gerektirmiyor: v2.3'ten beri `uses [@a, @b]` modül kökündeki adlandırılmış Function'larda da geçerli (v2.7.0'da GUI geri çağırmasıyla denendi). Eksik olan dil özelliği değil, belgelerin ve örneklerin bunu göstermesi. | doğrulandı; ilk taslaktaki "dil kararı gerekir" tespiti yanlıştı |

### 2.2 Graphics

`Canvas` yalnızca çizgi, daire ve dikdörtgen çiziyor. Doğrulanan eksikler:

| # | Eksik | Neden temel |
|---|---|---|
| G1 | Metin (`canvas.text`) | Eksen etiketi, puan, başlık, oyun yazısı yazılamıyor. |
| G2 | Çokgen, yay, elips | Üçgen bile üç ayrı çizgiyle çiziliyor ve içi doldurulamıyor. |
| G3 | Görsel yükleme (`canvas.image`) | PNG gösterilemiyor. |
| G4 | Zamanlayıcı | Hareket eden hiçbir şey yapılamıyor; tek girdi tıklama ve tuş. |
| G5 | Fare hareketi ve sürükleme | Serbest çizim programı yazılamıyor. |

### 2.3 GUI

| # | Eksik | Durum |
|---|---|---|
| U1 | Metin kutularında seçim, kopyala/yapıştır ve geri al yok (`cmd/ahdgui` kaynağında pano ya da seçim kodu bulunmuyor). Her kullanıcının ilk denediği şey. | doğrulandı |
| U2 | Görsel bileşeni yok; pencereye logo ya da fotoğraf konamıyor. | doğrulandı (belge) |
| U3 | Label tek satır; satır kaydırma, yazı boyutu ve kalınlık yok. Başlık ile gövde metni aynı görünüyor. | doğrulandı (belge) |
| U4 | Görünürlük (`setVisible`) yok; koşula göre alan gizlenemiyor. | doğrulandı (belge) |
| U5 | Yerleşim yalnızca sütun ve satır; hizalama, ızgara ve kaydırılabilir kapsayıcı yok. Sığmayan içerik kırpılıyor. | doğrulandı (belge) |
| U6 | Uzun süren geri çağırma pencereyi donduruyor; HTTP isteği ya da büyük sorgu sırasında pencere yanıt vermiyor. Zamanlayıcı da yok. | doğrulandı (belge) |
| U7 | Menü çubuğu, sekme, radyo düğmesi, ilerleme çubuğu, tarih seçici yok. | doğrulandı (belge) |
| U8 | Graphics çizimi bir GUI penceresinin içine yerleştirilemiyor; çizim ve form iki ayrı pencere. | doğrulandı (belge) |
| U9 | Değiştirici tuşlu kısayol (Cmd/Ctrl+S) yok. | doğrulandı (belge) |
| U10 | Windows iletişim kutuları derleniyor ama gerçek bir Windows makinesinde denenmedi. | doğrulandı (belge) |
| U11 | Bileşenleri AhdCode kendi çizdiği için ekran okuyucu desteği yok. | öneri: 3.0 sonrasına bilinçli erteleme |

### 2.4 Web

| # | Bulgu | Durum |
|---|---|---|
| W1 | Üretilen `<html>` etiketinde `lang` yok ve `Web.page` bunu ayarlamanın yolunu sunmuyor. | doğrulandı |
| W2 | Starter yerleşimi her sayfayı dikeyde ortalıyor; içerik sayfası (form, liste) ekranın ortasında yüzüyor. Örnek sitede bunu CSS ile düzeltmem gerekti. | doğrulandı |
| W3 | Tarayıcı canlı yenilemesi yok: `ahdcode dev` sunucuyu yeniden başlatıyor, sayfayı elle yenilemek gerekiyor. | doğrulandı (belge) |
| W4 | Oturumlar bellekte; sunucu her yeniden başladığında herkes çıkış yapmış oluyor. `dev` döngüsünde her kayıtta oluyor. | doğrulandı (belge) |
| W5 | Veritabanı şeması için göç (migration) düzeni yok; her uygulama `CREATE TABLE IF NOT EXISTS` yazıyor. | doğrulandı (belge) |
| W6 | Özel 404 ve 500 sayfası tanımlama yolu belgede görünmüyor. | belgeye bakıldı, kaynakta doğrulanmalı |
| W7 | Yerel HTTPS yok (bilinçli erteleme). Mobil tarayıcıda kamera, konum ve kurulabilir web uygulaması için HTTPS şart. | doğrulandı (belge) |

### 2.5 Mobil için mimari engeller

| # | Engel | Açıklama |
|---|---|---|
| M1 | Yardımcı süreç mimarisi | GUI (`ahdgui`), Graphics (`ahdgraphics`) ve SQLite (`ahdsqlite`) ayrı süreçler olarak çalışıyor ve programla standart girdi/çıktı üzerinden konuşuyor. iOS bir uygulamanın başka süreç başlatmasına izin vermez; Android'de de uygulama içi yardımcı süreç pratik değil. Çizici ve SQLite motoru programın içine bağlanmalı. |
| M2 | Sabit boyutlu pencere | Yerleşim, verilen genişlik ve yüksekliğe göre yapılıyor; sığmayan kırpılıyor. Telefon ekranı için ekrana uyan yerleşim gerekir. |
| M3 | Girdi | Dokunma, kaydırma hareketi ve ekran klavyesi yok. |
| M4 | Araç zinciri | Bu Mac'te Xcode 26.6 kurulu; Android SDK/NDK ve `gomobile` kurulu değil. Paketli Go araç zincirinin mobil hedefleri derleyebildiği doğrulanmalı. |

Elverişli taraflar: GUI yardımcısı Ebitengine üzerine kurulu ve Ebitengine Android
ile iOS'u destekliyor; SQLite motoru `wasm2go` ile saf Go'ya çevrilmiş durumda,
yani CGO gerektirmeden programın içine bağlanabilir.

## 3. Sürüm planı

Her sürüm bir tema taşır, yeni sözdizimi eklemez (2.11 hariç hiçbir sürüm dile
dokunmaz) ve her v2.7 programı çalışmaya devam eder.

### v2.8 — Graphics'in Tamamlanması ve İlk Hata Deneyimi

Küçük, hızlı ve dile yeni başlayan herkese doğrudan dokunan sürüm. Video serisinin 14–16.
bölümleri bu sürümle zenginleşir.

Graphics:
- `canvas.text(x, y, text, color, size)`: paketli Go Regular yazı tipiyle; PNG ve SVG çıktısında da aynı.
- `canvas.polygon(points, stroke, fill, width)`, `canvas.ellipse(...)`, `canvas.arc(...)`.
- `canvas.image(path, x, y, width, height)`: PNG ve JPEG.
- `canvas.every(milliseconds, handler)` ve `canvas.cancel(...)`: geri çağırma modeline uyan zamanlayıcı. Kare döngüsü ve sprite sistemi kapsam dışında kalır; hareket, zamanlayıcının `clear` + yeniden çizim yapmasıyla elde edilir.
- `canvas.onMove(handler)` ve `canvas.onDrag(handler)`.
- Zamanlayıcıyla yeniden çizim titremesin diye `canvas.batch(handler)` ya da eşdeğeri tek karede toplu çizim.

Tanılama (D1, D2):
- Ayrılmış sözcük ad olarak kullanıldığında tek, kısa bir hata: `'not' is a reserved keyword in AhdCode. Choose a different identifier, such as 'score'.`
- Ayrıştırıcıda hata toparlama: bir satırdaki ilk hatadan sonra aynı deyime ait ardışık hataların bastırılması.
- Mevcut tanılama kodları değişmez. Tanılama metinleri İngilizce kalır (bkz. §6).

Araçlar (D3, D4):
- `Server.start` öncesinde standart çıktının boşaltılması ve sonrasında her satırın yazıldığı anda teslim edilmesi (`Window.wait`, `Canvas.wait` ve `Scheduler.run` bunu zaten yapıyordu).
- `ahdcode format <klasör>` ve çoklu dosya.

Kabul ölçütü: serideki 14–16. bölüm örnekleri metin ve çokgenle yeniden yazılır; bir "top sektirme" ve bir "serbest çizim" örneği `examples/v2.8` altında çalışır; `not: Int := 70` tek satırlık hata verir.

### v2.9 — GUI'nin Kusursuzlaşması

Amaç: bir öğrencinin ya da küçük bir işletmenin yazdığı form uygulaması
"oyuncak" hissi vermesin.

Metin girişi (U1): seçim (fare ve Shift+ok), tümünü seç, kes/kopyala/yapıştır,
geri al/yinele, çift tıkla sözcük seçimi; `GUI.clipboard()` ve `GUI.setClipboard(text)`.

Görünüm (U2, U3, U4):
- `container.image(path, width, height)`.
- Label için satır kaydırma ve `setSize("small" | "normal" | "large" | "title")`, `setBold(true)`. Serbest yazı tipi seçimi kapsam dışında kalır.
- Her bileşen ve Container için `setVisible` / `isVisible`.

Yerleşim (U5): `container.grid(columns)`, Row ve Column için hizalama
(`start`, `center`, `end`, `stretch`), `container.scroll()` kaydırılabilir
kapsayıcı, bileşen için en küçük ve tercih edilen genişlik. Bu kurallar 3.0'da
telefon ekranına uyan yerleşimin de temeli.

Tepkisellik (U6):
- `window.every(milliseconds, handler)` zamanlayıcı.
- `GUI.task(work, done)`: `work` arka planda çalışır ve bir değer döndürür, `done` sonucu ana akışta alır. Kullanıcı kodunda iş parçacığı, kilit ya da paylaşılan değişken bulunmaz; "geri çağırmalar tek tek çalışır" kuralı korunur.

Bileşenler (U7, U8, U9): menü çubuğu ve kısayollar (`Cmd/Ctrl+S`), sekmeler,
radyo grubu, ilerleme çubuğu, tarih seçici, `container.canvas(width, height)` ile
pencere içinde Graphics çizimi, TableView'da başlığa tıklayarak sıralama.

Platform (U10): Windows 11'de iletişim kutuları ve yüksek çözünürlük canlı
testi; Linux'ta Wayland oturumu testi.

Kabul ölçütü: `examples/v2.0/ledger_app` menü, kısayol, sıralama ve arka plan
dışa aktarmayla yeniden yazılır ve üç platformda elle denenir; bir
ekran-görüntüsü karşılaştırma testi her bileşeni kapsar.

### v2.10 — Web'in Kusursuzlaşması

Yayımlanan kapsam için [v2.10.0 sürüm notlarına](docs/RELEASE_2.10.0_TR.md) bakın.
Bu bölüm ilk planı kaydeder. Yerel HTTPS ve PWA bu kaynak sürümünden sonraya ertelendi.

Geliştirme deneyimi:
- `ahdcode dev` tarayıcıyı kendisi yeniler (W3): yalnızca geliştirme kipinde sayfaya eklenen küçük bir yeniden bağlanma betiği.
- Kalıcı oturum deposu (W4): `Web.sessions(..., store: "file" | "sqlite")`; `dev` yeniden başlatmasında giriş korunur.
- Özel 404 ve 500 sayfaları (W6), geliştirme kipinde kaynak satırını gösteren hata sayfası.

Starter'lar (W1, W2):
- `Web.page` için `lang` parametresi; starter'lar `<html lang="en">` üretir ve uygulama kendi dilini tek satırla seçer. Starter metinleri İngilizce kalır.
- İçerik sayfaları için yukarıdan başlayan yerleşim sınıfı; ortalanmış karşılama yalnızca ana sayfada.
- Yeni starter: **Site** (ana sayfa, hakkımda, iletişim formu, ziyaretçi defteri). Video serisinin son beş bölümünün çıkış noktası.

Veri: numaralı SQL dosyalarıyla `ahdcode migrate` (W5); sayfalama yardımcısı.

Mobil tarayıcıya hazırlık (W7): starter'larda duyarlı (responsive) düzenin telefon
genişliğinde denetlenmesi; `manifest.webmanifest` ve simge üreten
`Web.installable(...)`; `ahdcode dev --https` için tek seferlik yerel sertifika.
Bu, 3.0'dan önce "telefona kurulabilen web uygulaması" verir.

Yayın: `ahdcode deploy check` (ortam sözleşmesi, `APP_ENV`, TLS'in ters vekilde
sonlandığı) ve Caddy + systemd için tek sayfalık rehber (İngilizce ve Türkçe belge).

Kabul ölçütü: ahdcode.org ve bir starter projesi telefonda Lighthouse
erişilebilirlik ve en iyi uygulama denetimlerinden geçer; starter'ların tamamı
masaüstü ve telefon genişliğinde tarayıcıda gözle sınanır.

### v2.11 — Web Platformu ve PWA

Amaç, AhdCode ile kurulabilir web uygulamaları geliştirmek, sınamak ve
dağıtmak. Geliştirme, yayımlanan v2.10.0 kaynağına dayanır. Derleyici sürümü
son sürüm hazırlığına kadar v2.10.0 olarak kalır.

1. **Faz 0 — Kararlılık.** HTTP/WebSocket geri çağırma kabulü ile sunucu
   kapanışını ve SQLite oturum sahipliğini eşgüdümle. Bağlama hatasından
   sonra sunucu tanıtıcısının tekrar kullanılabildiğini ve WebSocket saat
   testinin sabit bir başlangıç noktasına dayandığını doğrula. Faz 5'e
   geçiş için bağımsız QA kararı gerekir. [Faz 0 raporu](docs/qa/v2.11/phase-0.md)
2. **Faz 5 — İsteğe bağlı yerel HTTPS.** Go'nun TLS dinleyicisini ve
   yerel sertifika yönetimi seçeneklerini incele. Sertifikanın DNS/IP
   adları, tarayıcı güveni, anahtar erişimi, süre sonu ve yenileme birlikte
   ele alınmalı. Mevcut `.test` yönlendiricisi, doğrudan loopback erişimi,
   `wss://`, canlı yenileme ve ters vekilde TLS sonlandırma ayrı kabul
   senaryolarıdır. `ahdcode dev --https` arayüzü tasarım adayıdır. [Go HTTP](https://pkg.go.dev/net/http#Server.ServeTLS)
3. **Faz 6 — PWA.** Manifest, simgeler, service worker, çevrimdışı sayfa
   ve sürümlü varlık önbelleği için açık Web dosyaları üretmeyi değerlendir.
   Service worker güvenli bağlama ve doğru köken/kapsama bağlıdır. HTTP
   üzerindeki `localhost` ve loopback istisnalarını özel `.test` adları
   için varsayma. Oturumlu HTML, özel API, yönetim sayfaları ve POST
   yanıtları varsayılan olarak yalnız ağdan alınır. Gerçek masaüstü ve
   mobil tarayıcılarda kurulum, güncelleme ve çevrimdışı dönüş sınanır.
   [Güvenli bağlamlar](https://w3c.github.io/webappsec-secure-contexts/)
   · [Service Worker](https://w3c.github.io/ServiceWorker/)
4. **Faz 6.5 — Web varlıklarının dağıtımı.** Faz 6'nın dosya adları ve
   kapsamı kesinleşince derlenen .ahd uygulaması, statik dosya, manifest, worker ve simgeler
   için açık bir dağıtım listesi tasarla. `ahdcode package` ile ayrı Web
   dağıtımı seçeneklerini karşılaştır. Taşıma, çalışma dizini, platform
   yolları ve gizli dosyaların dışarıda kalması kabul ölçütleridir.
5. **Faz 7 — Son QA ve sürüm hazırlığı.** Onaylı fazlar tamamlandıktan sonra
   tarayıcı ve platform matrisini, önceki Web davranışlarını ve kaynak
   dağıtımını doğrula. Yayın için açık kullanıcı onayı gerekir.

Web uygulamalarında .ahd, Web.UI ve HTMLNode kullanılır. PWA ve gelecekteki
paketleme çalışmaları derlenen uygulama ve statik varlıkları temel alır.

#### Dil belirtimi ve süreç içi çalışma zamanı

Aşağıdaki mobil önkoşulları için ayrı bir kapsam ve sürüm kararı gerekir.

Dil dondurma (D5, D7):
- Tek, güncel **AhdCode Dil Belirtimi 3.0 (taslak)**: belirtim v2.8'de güncellendi (§91–§94); 3.0 için kalan iş, sürüm sırasına göre eklenmiş bölümlerin (§47–§94) konu sırasına göre yeniden düzenlenmesi ve dosyanın yeniden adlandırılması; İngilizce ve Türkçe.
- Uyumluluk test takımı: belirtimin her bölümü için kabul ve ret programları; `ahdcode` her sürümde bu takımdan geçer.
- Kararlılık sözü: 3.x boyunca geçerli her program derlenmeye devam eder; kaldırılacak her şey bir küçük sürüm boyunca uyarı verir.
- D7 (karar: 17 Kasım 2026, bkz. §7): yeni sözdizimi yok. GUI, Graphics ve Web belgeleri, öğrenci rehberi, `examples/` ve starter'lar geri çağırmalarda `uses [@...]` biçimine geçirilir; `ad: Global Tür` geçerli kalır ve belgelerde "eski biçim" olarak anlatılır. Dil sunucusunun eksik bağımlılık için sunduğu hızlı düzeltme `uses` listesini önerir. Belirtim 3.0, iki biçimin aynı takma adı kurduğunu tek bölümde tanımlar.
- Ayrılmış sözcük listesinin son hâli. 3.0'da yeni ayrılmış sözcük eklenirse eski programlar kırılır; mobil için gereken adlar modül ve tür adı olarak tasarlanır.

Süreç içi çalışma zamanı (M1), mobilin ön koşulu:
- GUI ve Graphics çizicisinin ve SQLite motorunun programın içine bağlanabildiği ikinci bir çalışma kipi. Masaüstünde yardımcı süreç kipi varsayılan kalır; `ahdcode package --inprocess` deneysel olarak tek dosyalık uygulama üretir.
- İki kipin aynı sonucu verdiğini doğrulayan ortak test takımı (mevcut `AHDCODE_GUI_HEADLESS_EVENTS` senaryoları her iki kipte koşar).

Araçlar: VS Code eklentisinin Marketplace'e çıkması; `ahdcode run` için derleme önbelleği (bugün her çalıştırma yaklaşık 1–2,5 saniye).

Kabul ölçütü: `ledger_app` tek bir çalıştırılabilir dosya olarak paketlenir ve yardımcı süreç olmadan çalışır.

### v3.0-beta — Android önizlemesi

- `ahdcode package app.ahd --target android-arm64` → imzasız `.apk`. Yalnızca süreç içi kip.
- Dokunma olayları mevcut geri çağırmalara eşlenir (dokun = tıkla); kaydırma hareketi kaydırılabilir bileşenlere; ekran klavyesi metin kutusuna odaklanınca açılır (M3).
- Yerleşim (M2): pencere telefonda ekranı doldurur; `GUI.window` genişlik ve yüksekliği masaüstü için öneri olur; 2.9'un hizalama ve kaydırma kuralları ekrana uydurur; güvenli alan boşlukları otomatik.
- `ahdcode doctor mobile`: Android SDK/NDK ve Xcode varlığını, sürümlerini ve eksik adımı söyler. SDK'lar pakete gömülmez; kullanıcı kurar.
- Uygulama yaşam döngüsü: arka plana alınınca ve dönünce durum korunur; `window.onPause` / `onResume`.
- Dosya yolları: `File` ve `SQLite` göreli yolları uygulamanın kendi veri klasörüne çözülür.

Kabul ölçütü: serideki "Not Hesapla", "Yapılacaklar" ve "Öğrenci Defteri"
programları kaynak değişmeden gerçek bir Android telefonda çalışır.

### v3.0 — Mobil Çıktı

- Android: imzalı `.apk` ve `.aab` (`--keystore`), uygulama adı, simge, sürüm numarası.
- iOS: `ahdcode package --target ios` bir Xcode projesi ve `.ipa` üretir; imzalama Xcode hesabıyla.
- Mobilde desteklenen modüller listesi açıkça yayımlanır: çekirdek dil, GUI, Graphics, SQLite, File, HTTP istemcisi, JSON, Time, Math ve ilgili saf modüller. Latex, PDF, Plot görüntüleyicisi ve Process mobilde derleme zamanı hatasıyla reddedilir.
- `ahdcode init mobile`: tek ekranlı ve çok ekranlı iki starter.
- Belgeler: `docs/MOBILE_TR.md`, öğrenci rehberinde yeni bölüm, mağazaya yükleme rehberi.
- Dil Belirtimi 3.0'ın kesinleşmesi.

Kabul ölçütü: aynı kaynak dosya macOS, Windows, Linux, Android ve iOS'ta
çalışır; bir örnek uygulama TestFlight ve Play dahili test kanalına yüklenir.

## 4. Sıra ve bağımlılıklar

```text
v2.8  Graphics + ilk hata deneyimi      (kaynak sürüm; kurulum paketleri v2.7.0'da kalır)
v2.9  GUI                               (kaynak sürüm)
v2.10 Web                               (kaynak sürüm)
v2.11 Web platformu + PWA              (kaynak sürüm planı)
v3.0-beta Android                       (yalnızca geliştirici ve mobil test paketleri)
v3.0  Android + iOS                     (tam masaüstü ve mobil dağıtım burada döner)
```

Video serisi bu sürümleri beklemez: 30 bölümün tamamı v2.7.0 ile çekilebilir.
v2.8 çıktığında 14–16. bölümler, v2.10 çıktığında 26–30. bölümler için kısa birer
"yenilikler" eki yeterli olur.

## 5. Sizden beklenen kararlar

1. **Mobil yolu.** Önerim yukarıdaki yol: GUI'nin kendi çizicisini telefona taşımak. Alternatif, Web uygulamasını bir WebView kabuğuna sarmak; daha hızlı çıkar ama GUI programları telefona taşınmaz ve "tek kaynak, her platform" sözü verilemez. 2.10'daki kurulabilir web uygulaması adımı bu alternatifin faydasının büyük kısmını zaten sağlar.
2. **Graphics sınırı.** Belge bugün "animasyon döngüsü planlanmıyor" diyor. Zamanlayıcı (G4) bu sınırı genişletir; kare döngüsü ve sprite yine kapsam dışında kalır. Onayınız gerekir.
3. **GUI geri çağırmalarındaki `Global` satırları (D7).** Karara bağlandı, bkz. §7.
4. **Sürüm temposu.** v2.0.0–v2.7.0 arası sekiz sürüm 19–29 Eylül 2026 arasında, on bir günde çıktı. Bu planda her sürüm, üç platformda elle deneme ve örneklerin yeniden yazımı dahil, öncekilerden büyük.

## 6. Kesinleşmiş tasarım kararı: yerelleştirme yok

17 Kasım 2026'da karara bağlandı. AhdCode uluslararası bir programlama dilidir;
belirli bir ülkenin öğrencileri onun tanımlı birincil kitlesi değildir.

- Dil, derleyici tanılamaları, uyarılar, öneriler ve geliştiriciye dönük bütün hata metinleri İngilizcedir.
- Kapsam dışı: `AHDCODE_LANG=tr`, `--lang tr`, çevrilmiş tanılama metinleri, Türkçe anahtar sözcük takma adları ve tanılama yerelleştirme altyapısı.
- İngilizce ve Türkçe belgeler yan yana yaşamaya devam eder. Belge çevirisi derleyicinin yerelleştirilmesi anlamına gelmez.
- Bu konu, açıkça istenmedikçe yeniden açılmaz.

## 7. Kesinleşmiş tasarım kararı: D7, `uses` ile çözülür

17 Kasım 2026'da karara bağlandı. İlke: daha az tekrar, aynı açıklık, aynı
güvenlik; yeni sözdizimi yalnızca gerçekten gerekliyse.

İnceleme sonucu (v2.7.0 üzerinde program çalıştırılarak ve
`internal/semantic/callable.go`, `lambda_capture.go` okunarak):

1. `ad: Global Tür` bildirimi ile `uses [@ad]` aynı işi yapar. İkisi de derleyicide aynı işlevle (`installGlobalAlias`) modül bağlamasına canlı bir takma ad kurar.
2. Tür güvenliği aynıdır: `@count` üzerinden `count = "metin"` `SEM004`, `Constant` bir bağlamaya atama `SEM009`, var olmayan ad `SEM044` verir. Değiştirilebilirlik aynıdır: atama ve `+=` modül bağlamasını değiştirir. Yaşam süresi aynıdır: kopya ya da anlık görüntü alınmaz, sonraki değişiklikler görülür.
3. Tek fark yazımdadır: `Global` satırı türü yeniden yazar ve derleyici bunu bağlamanın türüyle karşılaştırır; `uses` listesi tür taşımaz, türü bağlamadan alır. `#ad` ise farklı bir şeydir (iç içe Function için değerle yakalama) ve bu kararın konusu değildir.
4. `uses`, modül kökündeki adlandırılmış Function'larda ve sınıf metotlarında v2.3'ten beri çalışır; listenin imzadan sonraki satıra yazılması da geçerlidir. Genişletilecek bir şey yoktur.
5. Eski programlar etkilenmez: hiçbir kural değişmez, `Global` bildirimi geçerli kalır.

Bu geçiş dil değişikliği gerektirmez. Belge, örnek, starter ve hızlı
düzeltme güncellemeleri ayrı kapsam kararı bekler.

## 8. Kesinleşmiş dağıtım politikası: v3.0'a kadar kaynak sürüm

17 Kasım 2026'da karara bağlandı. v2.8.0, 18 Kasım 2026'da bu kuralla
yayımlandı. v2.9.0, v2.10.0 ve v2.11.0 da kaynak sürüm olarak yayımlanır.

- Bu dört sürümde DMG, PKG, Windows kurulum dosyası, Linux dağıtım arşivi,
  Apple notarization ve imzalama yoktur.
- v2.7.0 kurulum paketleri GitHub Release üzerinde olduğu gibi kalır.
  v2.7.0 yeniden etiketlenmez ve içeriği değiştirilmez.
- Yeni sürümün yayını kaynak kod, git etiketi ve GitHub Release'dir.
  Odaklı testler, regresyon testleri ve platform cross-build kontrolleri sürer.
  Yayına özel büyük paket derlemesi ve yüklemesi yapılmaz.
- ahdcode.org iki sürümü ayrı yazar: en güncel kaynak sürüm
  (`latest_source_version`) ve en güncel kurulabilir sürüm (`latest_version`).
  v2.8.0 kaynağı yayımlanınca birincisi v2.8.0 olur, ikincisi v2.7.0 kalır.
  Kaynak ayarı boşsa site ikisini aynı sürüm sayar.
- v3.0-beta yalnızca gerekli geliştirici ve mobil test paketlerini üretebilir.
  Tam masaüstü ve mobil dağıtım v3.0'a bırakılır.
- `tooling/distribution/build.py`, sürüm numarası 2.8–2.11 iken ya da 3.0 beta
  iken kurulum paketi üretmeyi reddeder. v2.7.0 ve v3.0 bu kapının dışındadır.
