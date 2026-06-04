#!/usr/bin/env bash
set -euo pipefail

SCRIPT_PATH="${BASH_SOURCE[0]:-$0}"
PROJECT_DIR="$(cd "$(dirname "$SCRIPT_PATH")/.." && pwd)"
LAUNCHER="$PROJECT_DIR/radio.sh"

if [ ! -x "$LAUNCHER" ]; then
  echo "radio.sh bulunamadı veya çalıştırılabilir değil: $LAUNCHER"
  exit 1
fi

install_link() {
  local target_dir="$1"
  mkdir -p "$target_dir"
  ln -sfn "$LAUNCHER" "$target_dir/radio"
  echo "✅ Komut oluşturuldu: $target_dir/radio -> $LAUNCHER"
}

if [ -w "/usr/local/bin" ]; then
  install_link "/usr/local/bin"
else
  install_link "$HOME/.local/bin"
  
  # PATH kontrolü ve otomatik ekleme kısmı
  case ":${PATH}:" in
    *":$HOME/.local/bin:"*)
      # PATH zaten ayarlı, bir şey yapmaya gerek yok
      ;;
    *)
      ZSHRC="$HOME/.zshrc"
      # .zshrc dosyasında bu yol zaten ekli mi diye kontrol et
      if ! grep -q "$HOME/.local/bin" "$ZSHRC" 2>/dev/null; then
        echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$ZSHRC"
        echo "✅ PATH ayarı $ZSHRC dosyasına otomatik eklendi."
      fi
      
      echo
      echo "⚠️ PATH güncellendi ancak mevcut terminalin bunu algılaması gerekiyor."
      echo "Lütfen şu komutu çalıştırın veya terminali kapatıp açın:"
      echo '  source ~/.zshrc'
      ;;
  esac
fi
