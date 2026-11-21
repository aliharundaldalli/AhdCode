# AhdCode v2.11'e geçiş

[English](UPGRADE_2.11.md)

Proje sahibi, genel kullanıma açık `Templates` modülünün v2.11'de
kaldırılmasını onayladı. Bu, bilinçli bir geriye dönük uyumluluk kırılmasıdır.
v2.10 API'sini kullanan uygulamaları yükseltmeden önce `Templates`
aktarımlarını ve `TemplateSet`, `TemplateData`, `TemplateError` kullanımlarını
değiştirin.

Sayfa, yerleşim ve bileşenleri `.ahd` dosyalarında yazın. `Web.UI` ile
`HTMLNode` değerleri oluşturan fonksiyonlara türü belli değerler aktarın.
Yerleşimden `Web.page` ile yanıt döndürün. Özel 404/500 işleyicilerini
mevcut Web veya HTTP API'leriyle kaydedin. Değişken metinleri, oluşturma
sırasında kaçırılmaları için metin düğümleriyle veya metin alan yardımcılarla
aktarın. Yeni üretilen Basic uygulaması bu yapıyı gösterir.
[Web uygulama yapısı](WEB_TR.md).

Derlenen uygulamayı ve genel statik varlıklarını birlikte geliştirin ve
paketleyin. Gelecekteki PWA ve paketleme çalışmaları bu yapıyı temel alır.

Önceki API, [v2.10.0 sürüm notlarında](RELEASE_2.10.0_TR.md) yer alır.
