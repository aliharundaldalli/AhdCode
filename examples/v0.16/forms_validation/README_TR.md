# Formlar ve doğrulama — v0.16 adayı

```bash
ahdcode run examples/v0.16/forms_validation/app.ahd
```

http://127.0.0.1:8160/register adresini açın. Veritabanı veya `.env` gerekmez.
Kabukta `SERVER_PORT=8162` portu değiştirir. `.env.example` açıklama amaçlıdır.

- GET `/register`: bağlam, oturuma bağlı CSRF, semantik Web.UI formu.
- POST `/register`: açık CSRF denetimi (403), sıralı doğrulama, hatada yalnızca
  ad/e-posta eski girdisi (422), boş parola kontrolleri.
- Başarılı POST: genel flash, 303 yönlendirme, açık yanıt sonlandırma.
- GET `/profile`: flash tüketimi ve silmenin kaydı; yenilemede mesaj yoktur.

Bu örnek hesap oluşturmayı veya kimlik doğrulamayı değil form durumunu öğretir.
Parolalar yalnızca istek belleğinde denetlenir. OldInput'a yalnızca `name` ve
`email` girer; gönderilen değerler oturuma kopyalanmaz. Metin ve öznitelikler
Web.UI ile kaçırılır. Yerel loopback sunucusu HTTP için Secure olmayan çerez
kullanır; HTTPS uygulamasında Secure çerez seçin. Flash alınana kadar bekler.

İşleyici adlandırması v0.16 sözleşmesini izler: `Pages/Register.ahd` içinde
`register` (GET) ve `registerSubmit` (POST), `Pages/Profile.ahd` içinde
`profile` bulunur. `Page` zorunlu bir sonek değildir — yönlendiriciye bir
`Function` değeri verilir ve adı hiç okunmaz. Bkz.
[10.1 Adlandırma](../../../docs/WEB_TR.md#101-adlandırma).

[Tam iş akışı ve API](../../../docs/WEB_TR.md).
