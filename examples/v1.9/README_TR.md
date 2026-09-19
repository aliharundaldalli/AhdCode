# GUI renkleri ve etkileşimli Plot görüntüleyicisi (v1.9)

[English](README.md) · [Türkçe]

v1.9 için iki küçük program: [GUI](../../docs/GUI_TR.md) renkleri ve
etkinlik durumu ile AhdCode'un kendi
[Plot](../../docs/PLOT_TR.md#show-gösterme) görüntüleyicisi.

## gui_colors

```bash
cd gui_colors
ahdcode run main.ahd
```

Açık renkler kullanan bir sipariş formu: renkli pencere ve başlık, renkli bir
TextInput, renkli bir Checkbox yazısı ve yeşil bir Kaydet düğmesi. Kaydet
devre dışı başlar; Check'e basın, Kaydet yalnızca bir müşteri adı yazıldığında
ve koşullar kabul edildiğinde etkinleşir. Kaydetmek bir satır yazar ve
Kaydet'i yeniden devre dışı bırakır. Escape pencereyi kapatır. Renkler
Graphics yazımlarını kullanır: dokuz küçük harfli ad, `#RRGGBB` veya
`#RRGGBBAA`.

Bu sıradan küçük bir formdur, bir stil vitrini değildir: tema, yazı tipi veya
özel bileşen yoktur.

## interactive_plot

```bash
cd interactive_plot
ahdcode run main.ahd
```

Başlığı, eksen etiketleri ve lejantı olan bir çizgi ve bir scatter serisi,
AhdCode'un kendi görüntüleyicisinde gösterilir:

- imlecin çevresinde yakınlaştırmak için kaydırın (veya trackpad kullanın),
- kaydırmak için sol tuşla sürükleyin,
- görünümü sola veya sağa çeyrek tur döndürmek için Q veya E'ye basın,
- sıfırlamak için R'ye basın ve
- kapatmak için Escape'e basın.

Program, görüntüleyici açılır açılmaz devam eder ve ardından `result.png`'yi
kaydeder — görüntüleyicide ne yaptığınızdan bağımsız olarak grafiğin tam
tanımlandığı hâli: görüntüleyici yalnızca görünümü değiştirir. `result.png`
deponun parçası değildir.
