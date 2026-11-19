# `ahdcode dev` altında tarayıcı yenilemesi

[English](README.md) · [Türkçe]

Bu sayfa sıradan bir `HTTP.html` yanıtıdır. Tarayıcıyı yenileyen `ahdcode dev`'dir.

```text
ahdcode dev app.ahd
```

Programın yazdığı adresi açın. Sekmeyi açık bırakın ve `app.ahd` içindeki
başlığı değiştirin. Dev günlüğü `→ Restarting...`, `✓ Running` ve `bring Web`
kullanan bir programda `✓ Browser will refresh` gösterdikten sonra sekme,
elle yenilemeden yeni başlığı gösterir. Bir Web programının ilk başlangıcı
başlıkta `Browser refresh:` bölümünü yazar ve ilk `✓ Running` satırından
sonra o onayı yazmaz.

Bir sözdizimi hatası kaydedin. Terminal tanılamayı tutar, tarayıcı son
sayfada kalır ve yenileme adresi aynı oturumu ve nesli bildirmeye devam
eder. Dosyayı düzeltince sekme yeniden yüklenir. `ahdcode dev` durdurulup
yeniden başlatılınca yeni sürecin oturumu farklıdır; açık bırakılan sekme
eski sayfada kalmaz.

Betik `/__ahdcode/dev/reload.js` adresindeki harici bir dosyadır; `script-src
'self'` izin veren bir `Content-Security-Policy` onu çalıştırabilir. `.env`
kancayı açamaz. `gzip`, `UTF-16` veya bir yönlendirme olarak işaretlenmiş
yanıt değiştirilmez.

`ahdcode run app.ahd` sayfayı sunar ve betiği enjekte etmez.
`GET /__ahdcode/dev/reload` o zaman sıradan bir 404'tür.

Bu örnek, AhdCode v2.10.0 ile gelen canlı yenilemeyi gösterir.

## Sınırlar

- Yenileme tam sayfa yüklemesidir. Form girdisi ve kaydırma konumu korunmaz.
- Statik dosyalar ve `HTTP.file` yanıtları izlemenin parçası değildir ve
  betikle yeniden yazılmaz.
- Kanca, yalnızca süreç `ahdcode dev` ile başlatıldıysa vardır.
- Yerel HTTPS değişmedi: `ahdcode dev` hâlâ `APP_PROTOCOL=https` değerini reddeder.
