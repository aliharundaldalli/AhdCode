# v0.17 rota grupları ve bekçiler

Bağlam duyarlı rotalar, bir `/admin` grubu ve iki sıralı bekçi. Veritabanı,
npm veya CDN yoktur.

```bash
cp .env.example .env
ahdcode dev app.ahd
```

| İstek | Sonuç |
| --- | --- |
| `GET /` | `200 home` |
| Oturumsuz `GET /admin` | `401` — `adminOnly` çalışmaz |
| `role=member` ile giriş, sonra `GET /admin` | `403` |
| `role=admin` ile giriş, sonra `GET /admin` | `200 admin dashboard` |

Bekçi ve işleyici, seçtikleri yolda `context.respond` çağırır. Gizli
sonlandırma ve genel ara katman zinciri yoktur.
