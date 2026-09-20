# Öğrenci Performans Gezgini (v2.2)

[English](README.md) · [Türkçe]

> Bu örnek AhdCode v2.2.0 sürümüyle birlikte gelir.

Bir öğrencinin notlarını toplayan ve aynı on beş sayıyı yedi ayrı biçimde
gösteren tek bir masaüstü uygulaması. v2.2'nin Plot eklerinin karşısında
geliştirildiği program budur: pasta grafiği, ısı haritası ve adlandırılmış
Surface eksenleri, önceki grafiklerin yanıtlayamadığı soruları yanıtlar.

```bash
cd student_performance
ahdcode run main.ahd
```

## Ne yapar

[GUI](../../../docs/GUI_TR.md) veriyi toplar: bir öğrenci adı ve 5 × 3'lük
bir not ızgarası — Analiz, Cebir, Geometri, İstatistik ve Programlama, üç
akademik yıl boyunca. Her alan yazıldıkça denetlenir: bir not 0 ile 100
arasında bir sayı olmalıdır ve ad ile on beş notun tamamı geçerli olana
kadar grafik düğmeleri devre dışı kalır. Özet `TableView` ve yıllık
ortalamalar her tuş vuruşunda güncellenir.

[Numeric](../../../docs/NUMERIC_TR.md) ızgarayı bir `Matrix` olarak tutar,
[Plot](../../../docs/PLOT_TR.md) ise çizer:

| Düğme | Grafik | Yanıtladığı soru |
|---|---|---|
| Seçili grafiği aç | çizgi, dağılım ya da çubuk | bir ders üç yılda nasıl değişti? |
| Histogram aç | histogram | on beş notun dağılımı nasıl? |
| Kutu grafiği aç | kutu | yayılım nedir — medyan, çeyrekler, uçlar? |
| Yıllık hata çubuğu aç | hata çubuğu | her yıl ne kadar tutarlıydı? |
| **Seçili yılın pasta grafiği** | **pasta** | **bir yıl derslere göre nasıl dağıldı?** |
| **Performans ısı haritası** | **ısı haritası** | **güçlü ve zayıf hücreler bir bakışta nerede?** |
| 3B performans yüzeyi aç | yüzey | tüm ızgara bir şekil olarak nasıl görünüyor? |

Arayüz iki dillidir: dil seçimi her etiketi, düğmeyi ve tablo başlığını
yeniden yazar; grafikler de seçilen dilde başlıklanır.

## v2.2 buraya ne ekliyor

**`Plot.pie`** bir yılın beş dersini dilimler olarak gösterir. Yanındaki
Select yılı seçer. Pasta grafiğinin ekseni yoktur, bu yüzden program ona
`xLabel` ya da `yLabel` koymaz — varsayılan olarak açık olan göstergesi
dersleri adlandırır:

```ahd
chart: Local Chart := Plot.pie(courseNames(), yearGrades).title(pTitle)
```

**`Plot.heatmap`** on beş notun tamamını ders ve yıl olarak birden gösterir.
`Matrix`, y etiketi başına bir satır ve x etiketi başına bir sütundur — beş
ders aşağı, üç yıl yana — ve renk ölçeği sayıları okumanın yerini alır:

```ahd
chart: Local Chart := Plot.heatmap(yearNames(), courseNames(), Numeric.matrix(gradeMatrix()))
chart = chart.title(hTitle).xLabel(xLabelText).yLabel(yLabelText)
```

**`Surface.xCategories` ve `Surface.yCategories`** yüzeyin koordinatlarını
adlandırır. v2.2'den önce aynı yüzey `1..3` ve `1..5` üzerine çizilir ve
başlığının sayıların ne anlama geldiğini açıklaması gerekirdi; artık eksenler
"1. Yıl" ve "Geometri" der. Geometri değişmez — etiketler yalnızca
gösterileni değiştirir:

```ahd
surface: Local Surface := Plot.surface(x, y, Numeric.matrix(matrixRows))
surface = surface.xCategories(yearNames()).yCategories(courseNames())
surface = surface.title(sTitle).xLabel(xLabelText).yLabel(yLabelText).zLabel(zLabelText)
```

`xLabel` ve `yLabel` eksenleri adlandırmaya devam eder; `xCategories` ve
`yCategories` ise eksen üzerindeki noktaları adlandırır.

## Deneyin

Bir ad ve şu ızgarayı yazın:

| Ders | 1. Yıl | 2. Yıl | 3. Yıl |
|---|---|---|---|
| Analiz | 72 | 78 | 84 |
| Cebir | 68 | 75 | 81 |
| Geometri | 80 | 82 | 88 |
| İstatistik | 65 | 74 | 83 |
| Programlama | 85 | 91 | 95 |

Sonra ısı haritasını ve yüzeyi yan yana açın. Her grafik kendi
[Plot görüntüleyici](../../../docs/PLOT_TR.md) penceresinde açılır:
sürükleyin, yakınlaştırın, döndürün ve kaydedin; uygulama arkada
kullanılabilir kalır.

## Notlar

Bir alanı boş bırakmak ya da negatif bir sayı, 100'den büyük bir sayı veya
sayı olmayan bir şey yazmak durum satırında bildirilir ve düzeltilene kadar
grafik düğmelerini devre dışı bırakır. Bir görüntüleyicinin Save'ini
kullanmadıkça diske hiçbir şey yazılmaz.
