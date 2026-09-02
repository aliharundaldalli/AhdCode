# Dil sunucusu (Language server)

[English](LSP.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [CLI](CLI_TR.md) · [Dil Turu](LANGUAGE_TOUR_TR.md)

```bash
ahdcode lsp
```

AhdCode dil sunucusunu başlatır. Standart
[Language Server Protocol](https://microsoft.github.io/language-server-protocol/)'ü
**yalnızca stdin/stdout üzerinden** konuşur — TCP portu, HTTP uç noktası,
tarayıcı, arka plan servisi (daemon) veya soket modu yoktur. `stdout`
yalnızca protokol çerçeveleri taşır; `ahdcode lsp` asla bir sürüm başlığı,
log satırı veya başka bir insan-okunabilir metin yazmaz. LSP konuşan
herhangi bir editör onu bir alt süreç (child process) olarak başlatıp
konuşabilir — [`editors/vscode`](../editors/vscode) içindeki VS Code
eklentisi olası istemcilerden yalnızca biridir, özel bir durum değildir.

## Neyi derleyici karar verir, LSP değil

Dil sunucusunun kendine ait bir ayrıştırıcısı (parser), tür denetleyicisi
veya sembol kataloğu yoktur. Her tanılama, hover, completion öğesi, rename
hedefi, semantic token, inlay hint ve quick fix, `ahdcode build` ve
`ahdcode run`'ın kullandığı aynı sözcüksel çözümleyici, ayrıştırıcı ve
anlamsal analizciden — `internal/analysis` ve `internal/lsp` katmanları
aracılığıyla — doğrudan gelir. Standart modül üyeleri asla elle listelenmez:
bunlar derleyicinin `bring`'i çözümlemek için kullandığı
`StandardModuleInterfaces()`'ten gelir.

## Yetenekler (v0.2.2)

Pratik günlük AhdCode LSP özellik seti v0.2.2 ile **tamamlanmıştır**.
`initialize` yanıtı yalnızca aşağıdakileri duyurur (artımlı senkronizasyon,
aralık biçimlendirme, semantic-token delta, call/type hierarchy, debugger
yok).

- **Belge senkronizasyonu** — tam (full) belge senkronizasyonu; artımlı
  derleyici yok.
- **Tanılamalar**, **Hover**, **Tanıma git**, **Belge sembolleri**,
  **Signature help**, **Referans bulma** (geçerli **derleme grafiği** ile
  sınırlı; workspace genelinde indeks değil).
- **Completion** — modül adları, `from ... bring` dışa aktarımları,
  namespace/Class üyeleri (**erişim-farkında Confidential üyeler**: yalnızca
  derleyici erişilebilir dediğinde önerilir), kapsamdaki yereller,
  otomatik import ve ölçülü anahtar kelimeler.
- **Rename** — Definition/References ile aynı semantik kimlik; derleme
  grafiği kapsamında.
- **Semantic Tokens** — derleyici/AST olgularından; UTF-16 doğru konumlar.
- **Inlay Hints** — çıkarılan türler ve ölçülü parametre-adı ipuçları.
- **Code Actions** — yalnızca yapılandırılmış tanılama kodlarına bağlı
  quick fix'ler: `SEM006` (eksik `Local`), `PAR009` (geçersiz `for`
  bağlaması `Local`), export-bulunamadı import tanılamaları.
- **Belge biçimlendirme** — mevcut biçimlendirici kütüphanesi bellek-içi;
  diske yazmaz, shell-out yok.
- **Workspace Symbols** — workspace kökleri ve giriş dizininde isteğe bağlı
  tarama; kalıcı indeks yok.
- **Folding Range** ve **Selection Range** — AST destekli.

Sunucu **kaydedilmemiş editör metnini** analiz eder; açık belgeyi yalnızca
derlemek için diske geri yazmaz.

## Otomatik import ve modül keşfi

Kullanıcı modülleri `initialize`'dan gelen workspace kökleri ile giriş
belgesinin dizinindeki kardeş `.ahd` dosyalarının sınırlı taramasıyla
bulunur. Sabit sembol kataloğu, arka plan izleyici veya kalıcı veritabanı
yoktur. Aynı ad iki modülde varsa completion ayrı girişler gösterir.

## Bilerek uygulanmayanlar

Artımlı ayrıştırıcı/derleyici, kalıcı workspace indeksi, semantic-token
delta, aralık biçimlendirme, call/type hierarchy, code lens, debugger/DAP,
AI completion ve üretken code action'lar kapsam dışıdır.

## Konum kodlaması

AhdCode kaynak konumları Unicode kod noktalarıyla bir-tabanlı satır/sütundur.
LSP konumları sıfır-tabanlı satır/UTF-16 kod birimleridir. Sunucu her
istekte gerçek kaynak metnini kullanarak dönüştürür; emoji gibi BMP-dışı
karakterlerden sonra doğru editör konumunu verir.
