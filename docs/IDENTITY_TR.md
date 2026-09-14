# Identity

`Identity.id()` herkese açık bir tanımlayıcı üretir. Bu bir sır değildir.

```
bring Identity

value: String := Identity.id()
```

Değer, 16 kriptografik rastgele baytın dolgusuz URL-güvenli Base64
karşılığıdır: `A-Za-z0-9-_` kümesinden 22 karakter.

Bir `public_id` sütunu gibi herkese açık kayıt kimlikleri için kullanın.
Parola, oturum sırrı veya CSRF jetonu için kullanmayın. Bunlar
`Security.token()` ve `Security.passwordHash()` işidir.

Çözümleme API'si yoktur. Tanımlayıcı opak bir dizgedir.

Başka bir sistem UUID beklediğinde, örneğin bir PostgreSQL `uuid` sütunu ya da
harici bir API, bunun yerine [`UUID`](UUID_TR.md) kullanın. `Identity.id()`
v1.4.0'da değişmedi ve ikisi arasında dönüşüm yoktur.

AhdCode Web başlangıç uygulamaları veritabanı için özel sayısal bir `id` ve
URL'ler için ayrı bir `public_id` saklar. Rotalar sayısal kimliği göstermez.
