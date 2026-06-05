# Radio Terminal

Terminalden Türkiye ve dünya radyo istasyonlarını dinlemek için Go ile yazılmış
TUI ve web arayüzlü FM radyo oynatıcı.

Varsayılan deneyim tam ekran Bubble Tea TUI'dir. Sol panelde istasyonlar,
sağ panelde komut çıktıları, altta komut girişi ve canlı durum çubuğu bulunur.
Web arayüzü aynı oynatıcı ve servisleri `127.0.0.1:8765` üzerinden kullanır.

## Gereksinimler

| Araç | Sürüm | Not |
|------|------|-----|
| Go | 1.25+ | `go.mod` Go 1.25 hedefler |
| ffplay | ffmpeg paketiyle gelir | Canlı yayın oynatma için gerekir |
| ffmpeg | ffmpeg paketiyle gelir | `kaydet` komutu ile MP3 kayıt için gerekir |

`online-ara` ve `online-ekle` komutları RadioBrowser API'sine internet erişimi
gerektirir. Linux masaüstü bildirimi için `notify-send`, macOS için
`osascript` kullanılır.

## Kurulum ve Çalıştırma

```bash
go build -o radio ./cmd/radio
./radio
```

Derlemeden çalıştırmak için:

```bash
go run ./cmd/radio
```

Testler:

```bash
go test ./...
```

Go build cache dizini yazılabilir değilse:

```bash
GOCACHE=/tmp/go-build go test ./...
```

İlk etkileşimli çalıştırmada uygulama dil seçimi ister ve sonucu
`~/.radio-shell/settings.json` içine yazar. Desteklenen dil kodları:
`en`, `tr`, `de`, `fr`, `it`.

## Web Arayüzü

```bash
./radio --web              # arka planda web sunucusu başlatır ve tarayıcı açar
./radio --web --foreground # web sunucusunu mevcut terminalde çalıştırır
./radio --kill             # arka plandaki web sunucusunu durdurur
```

Web arayüzü `http://127.0.0.1:8765` adresinde çalışır. Arka plan süreci
`~/.radio-shell/web.pid` dosyasıyla takip edilir.

## TUI Kısayolları

| Tuş | İşlem |
|-----|------|
| `Enter` | Yazılı komutu çalıştırır; giriş boşsa seçili istasyonu çalar |
| `Tab` | Komut, bayrak, istasyon, ülke, tür, tema veya dil önerisini tamamlar |
| `Yukarı` / `Aşağı` | Giriş boşken istasyon seçimini taşır; yazarken geçmişi/önerileri gezer |
| `Ctrl+N` / `Ctrl+P` | İstasyon seçimini ileri/geri taşır |
| `Ctrl+F` | Seçili istasyonu favorilere ekler veya çıkarır |
| `Ctrl+S` | Oynatmayı durdurur |
| `Ctrl+L` | Komut çıktısını temizler |
| `Ctrl+R` | İstasyon listesini yeniler |
| `PgUp` / `PgDown` | Komut çıktısını kaydırır |
| `Ctrl+C` / `Esc` | Uygulamadan çıkar |

## Komutlar

### Listeleme ve Arama

| Komut | Açıklama |
|-------|----------|
| `listele [-n sayı] [--hepsi]` | İstasyonları listeler. Varsayılan ilk 50 sonuçtur |
| `turkiye` | Türkiye istasyonlarını listeler |
| `ulkeler` | Mevcut ülkeleri ve istasyon sayılarını gösterir |
| `ulke -i <ülke>` | Belirli ülkedeki istasyonları listeler |
| `turler` | Mevcut türleri gösterir |
| `tur -i <tür>` | Belirli türdeki istasyonları listeler |
| `ara -s <sorgu>` | Ad, ülke veya tür içinde arar |
| `online-ara [-s sorgu] [-u ülke] [-t tür] [-l limit]` | RadioBrowser üzerinden çevrimiçi arama yapar |

### Oynatma

| Komut | Açıklama |
|-------|----------|
| `cal <id\|no>` veya `cal -i <id>` | İstasyonu ID ya da son listedeki sıra numarasıyla çalar |
| `son` | Son çalınan istasyonu başlatır |
| `dur` / `durdur` | Oynatmayı durdurur |
| `durum` | Geçerli istasyon, şarkı, ses ve kayıt durumunu gösterir |
| `ses [0-100]` veya `ses -s <0-100>` | Ses seviyesini değiştirir |
| `sessiz [ac\|kapat]` / `mute` | Sesi kapatır veya açar |
| `sonraki` / `ileri` | Son listede bir sonraki istasyona geçer |
| `onceki` / `geri` | Son listede bir önceki istasyona geçer |
| `karistir [-u ülke] [-t tür]` / `rastgele` | Filtrelenmiş rastgele istasyon çalar |
| `uyku -d <dakika>` / `uyku iptal` | Uyku zamanlayıcısı kurar veya iptal eder |
| `gecmis` | Son alınan şarkı metadata geçmişini gösterir |

### Kayıt

| Komut | Açıklama |
|-------|----------|
| `kaydet` | Çalan yayını `~/.radio-shell/recordings/` altına 128 kbps MP3 olarak kaydetmeye başlar |
| `kayitdur` | Aktif kaydı durdurur |

### Yönetim

| Komut | Açıklama |
|-------|----------|
| `favori [id]` | Verilen veya çalan istasyonu favorilere ekler/çıkarır |
| `favoriler` | Favori istasyonları listeler |
| `tema [ad]` | Temayı gösterir veya değiştirir |
| `kontrol [id]` | Bir istasyonu veya tüm istasyonları HTTP HEAD/GET ile kontrol eder |
| `ekle --id <id> --isim <ad> --url <url> [--ulke <ülke>] [--tur <tür>]` | Özel istasyon ekler veya aynı ID'yi günceller |
| `duzenle --id <id> [--isim ...] [--url ...] [--ulke ...] [--tur ...]` | Özel istasyonu düzenler |
| `sil --id <id>` | Özel istasyonu siler |
| `iceaktar -d <playlist.m3u> [-u ülke] [-t tür] [-p prefix]` | M3U playlist içindeki HTTP yayınlarını özel istasyonlara ekler |
| `bildirim [ac\|kapat]` | Masaüstü bildirimlerini açar veya kapatır |
| `online-ekle -n <no>` | Son `online-ara` sonucundan istasyon ekler |
| `dil -i <kod>` / `lang -i <kod>` | Uygulama dilini değiştirir |
| `istatistik` | Dinleme istatistiklerini gösterir |
| `sistem` | OS, Go sürümü, CPU ve bellek bilgilerini gösterir |
| `web` | TUI içinden web arayüzünü başlatır |
| `temizle` / `clear` | Komut çıktısını temizler |
| `help` / `?` | Yardım menüsünü gösterir |
| `exit` / `q` / `quit` | Uygulamadan çıkar |

Mevcut terminal temaları: `default`, `hacker`, `ocean`, `sunset`,
`midnight`, `sakura`, `winamp-classic`, `besiktas`.

## Proje Yapısı

```text
cmd/radio/          İnce binary giriş noktası
internal/app/       Bayraklar, servis kurulumu, TUI/web yaşam döngüsü
internal/config/    Varsayılan ffplay ayarları ve dosya yolları
internal/models/    RadioStation ve UserSettings veri modelleri
internal/tui/       Bubble Tea tam ekran terminal arayüzü
internal/shell/     Komut kayıtları, komut işleyicileri ve tamamlayıcılar
internal/player/    ffplay oynatma, ffmpeg kayıt ve metadata izleme
internal/services/  İstasyon, ayar, istatistik, bildirim, sistem, RadioBrowser
internal/ui/        Terminal çıktı yardımcıları ve temalar
internal/web/       Gin tabanlı web API ve gömülü statik arayüz
```

## Kalıcı Durum

Tüm kullanıcı durumu `~/.radio-shell/` altında tutulur:

| Dosya/dizin | İçerik |
|-------------|--------|
| `favorites.json` | Favori istasyon ID'leri |
| `custom-stations.json` | Kullanıcının eklediği istasyonlar |
| `settings.json` | Ses, sessiz durumu, son istasyon, bildirim ve dil |
| `stats.json` | En az 30 saniyelik dinleme oturumları |
| `theme` | Seçili terminal teması |
| `recordings/` | MP3 kayıtlar |
| `web.pid` | Arka plan web sunucusu PID bilgisi |

Yerleşik istasyonlar `internal/services/stations.json` dosyasından Go embed ile
binary içine gömülür.

## Geliştirme Notları

- Yeni bağımlılık eklerken `go get <modül>` veya `go mod tidy` kullanın ve
  `go.mod` ile `go.sum` değişikliklerini birlikte commit edin.
- Yeni komut eklerken `internal/shell/commands.go` içinde `RegisterAllCommands`
  üzerinden kaydedin, açıklama anahtarını `internal/services/localization.go`
  içine ekleyin ve gerekiyorsa `shell.FlagSuggestions` ile TUI tamamlama
  mantığını güncelleyin.
- Komut çıktıları `internal/ui` yardımcıları üzerinden yazılmalıdır. TUI komut
  çıktısını bu katmanı yakalayarak sağ panelde gösterir.
- Kullanıcıya görünen komut adları ve varsayılan metinler Türkçedir; destekli
  çeviriler `internal/services/localization.go` içinde tutulur.

## Lisans

MIT License.
