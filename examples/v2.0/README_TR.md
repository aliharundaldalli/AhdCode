# Masaüstü uygulamaları (v2.0)

[English](README.md) · [Türkçe]

v2.0 için üç program: eksiksiz bir [GUI](../../docs/GUI_TR.md) uygulaması,
3B bir [Plot](../../docs/PLOT_TR.md#surface) Surface'i ve
[paketlenecek](../../docs/PACKAGING_TR.md) küçük bir uygulama.

## ledger_app

```bash
cd ledger_app
ahdcode run main.ahd
```

SQLite'ta tutulan bir müşteri defteri. Bir müşteri yazın, Select'te Debt
(borç) veya Payment (ödeme) seçin, bir tutar ve TextArea'ya isteğe bağlı bir
not yazın ve Add entry'ye basın — bu düğme yalnızca müşteri ve pozitif bir
tutar doluyken `onChange` sayesinde etkindir. TableView girdileri güncel
bakiyeyle listeler; "Show every customer" işareti kaldırıldığında ListBox
müşteriye göre süzer. Bir satırı seçip onaydan sonra silin ve görünen
satırları kaydetme iletişim kutusuyla CSV'ye aktarın. Pencere yeniden
boyutlandırılabilir: tablo ve liste onunla büyür. Escape pencereyi kapatır.

`ahdcode-ledger.db` veritabanı çalışma klasöründe oluşturulur. Üretilmiş bir
dosyadır; örneğin parçası değildir.

## surface_plot

```bash
cd surface_plot
ahdcode run main.ahd
```

41 × 41'lik bir ızgarada z = sin(x) · cos(y) çizer, `surface.png` ve
`surface-wireframe.png` dosyalarını kaydeder ve Surface'i Plot
görüntüleyicisinde açar: döndürmek için sürükleyin, kaydırmak için
Shift+sürükleyin, yakınlaştırmak için tekerleği kullanın, sıfırlamak için R
veya Reset View, aynı PNG'yi yeniden yazmak için Save.

## package_hello

```bash
cd package_hello
ahdcode package main.ahd --name "Hello AhdCode"
```

En küçük masaüstü uygulaması: selam veren tek bir pencere. Komut macOS'ta
`dist/Hello AhdCode.app`'i (Windows ve Linux'ta bir klasör ve bir arşiv)
yapar; bu uygulama yalnızca programı ve `ahdgui` yardımcısını içerir ve
AhdCode kurulu olmadan çalışır. Defter de aynı şekilde paketlenir:

```bash
ahdcode package ../ledger_app/main.ahd --name Ledger
```
