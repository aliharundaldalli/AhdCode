# AhdCode v2.11'e geçiş

[English](UPGRADE_2.11.md)

Genel kullanıma açık `Templates` modülü AhdCode v2.11.0 ile kaldırılmıştır.
v2.10 `Templates` API'sini kullanan uygulamaları yükseltmeden önce `Templates`
aktarımlarını ve `TemplateSet`, `TemplateData`, `TemplateError` kullanımlarını
aşağıdaki yapıya taşıyın.

Sayfa, yerleşim ve bileşenleri `.ahd` dosyalarında yazın. `Web.UI` ile
`HTMLNode` değerleri oluşturan fonksiyonlara türü belli değerler aktarın.
Yerleşimden `Web.page` ile yanıt döndürün. Özel 404/500 işleyicilerini
mevcut Web veya HTTP API'leriyle kaydedin. Değişken metinleri, oluşturma
sırasında kaçırılmaları için metin düğümleriyle veya metin alan yardımcılarla
aktarın. Yeni üretilen Basic uygulaması bu yapıyı gösterir.
[Web uygulama yapısı](WEB_TR.md).

Derlenen uygulamayı genel statik varlıkları ve isteğe bağlı PWA dosyalarıyla
birlikte `ahdcode package app.ahd --web` (ve PWA yapılandırılmışsa `--pwa`)
komutuyla paketleyip dağıtabilirsiniz.

Önceki API, [v2.10.0 sürüm notlarında](RELEASE_2.10.0_TR.md) yer alır.
