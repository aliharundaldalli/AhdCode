# Graphics ve Turtle

[English](README.md) · [Türkçe]

v1.6.0 [`Graphics`](../../../docs/GRAPHICS_TR.md) modülü için iki küçük
program: Kartezyen bir Canvas, üç şekli ve bir Turtle kalemi.

## main.ahd

```bash
ahdcode run main.ahd
```

800 × 600'lük bir pencere açar ve şunları çizer:

- merkezden geçen `x` ve `y` eksenleri, her 50 birimde bir çentikle;
- yarı saydam mavi bir çember ve merkezinde siyah bir nokta olan, sol alt
  köşesiyle verilmiş yeşil bir dikdörtgen;
- kırmızı beş köşeli bir yıldız: beş kez 220 ileri git, 144 derece sağa dön;
- her biri biraz daha büyük ve 10 derece döndürülmüş on sekiz kareden oluşan
  bir spiral.

Çizimi geçerli dizine `graphics-demo.png` ve `graphics-demo.svg` olarak
kaydeder, birkaç satır yazar ve siz pencereyi kapatana kadar bekler.

Beklenen çıktı:

```text
Saved graphics-demo.png and graphics-demo.svg.
After five 144-degree turns the star turtle faces 0.0 degrees.
Close the window to finish.
Window closed.
```

Son satır pencereyi kapattıktan sonra görünür. Yıldız başladığı yöne bakarak
biter, çünkü beş dönüşünün toplamı 720 derecedir, yani iki tam tur.

## turtle_polygon.ahd

```bash
ahdcode run turtle_polygon.ahd
```

Dış açısından düzgün bir çokgen: herhangi bir düzgün çokgenin çevresini bir kez
dolaşmak Turtle'ı toplam 360 derece döndürür; bu yüzden `n` köşesinin her biri
`360 / n` derece döner. Program açıları yazar ve çokgeni çizer.

```text
A 6-sided polygon turns 60.0 degrees at each corner.
Each interior angle is 120.0 degrees.
Total turning: 360.0 degrees.
```

En üstteki `sides` değerini değiştirip yeniden çalıştırın.

İki program da oyun değildir: animasyon döngüsü, girdi ya da ses yoktur;
yalnızca her zaman aynı resmi üreten çizim komutları vardır.
