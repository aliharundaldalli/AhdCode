# v0.18 Web starter'lar

`ahdcode init web` bulunulan dizine üç starter'dan birini yazar.

```bash
mkdir my-app
cd my-app
ahdcode init web
ahdcode dev app.ahd
```

- **Empty** — karşılama uygulaması. Veritabanı yok, giriş yok.
- **Basic** — karşılama uygulaması artı `Config/Mail.ahd`.
- **Admin** — Home, Login, Dashboard, bir yönetici, SQLite veya MySQL.

Üretilen kaynak uygulamaya aittir. Kimlik politikası sıradan koddur
(`signedIn`); bir çerçeve özelliği değildir.
