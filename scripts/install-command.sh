#!/usr/bin/env bash
set -euo pipefail

SCRIPT_PATH="${BASH_SOURCE[0]:-$0}"
PROJECT_DIR="$(cd "$(dirname "$SCRIPT_PATH")/.." && pwd)"
BINARY_NAME="radio"

# Go kontrolü
if ! command -v go >/dev/null 2>&1; then
  echo "Hata: 'go' bulunamadı. Lütfen Go'yu yükleyin: https://go.dev/dl/"
  exit 1
fi

# ffplay kontrolü
if ! command -v ffplay >/dev/null 2>&1; then
  echo "Uyarı: 'ffplay' bulunamadı. Oynatma için ffmpeg gereklidir."
fi

echo "Binary derleniyor..."
(cd "$PROJECT_DIR" && go build -o "$BINARY_NAME" ./cmd/radio)
echo "✅ Derleme tamamlandı: $PROJECT_DIR/$BINARY_NAME"

install_binary() {
  local target_dir="$1"
  mkdir -p "$target_dir"
  # Eski sembolik link veya dosya varsa sil (kırık link cp'yi bozar)
  rm -f "$target_dir/$BINARY_NAME"
  cp "$PROJECT_DIR/$BINARY_NAME" "$target_dir/$BINARY_NAME"
  chmod +x "$target_dir/$BINARY_NAME"
  echo "✅ Komut kuruldu: $target_dir/$BINARY_NAME"
}

if [ -w "/usr/local/bin" ]; then
  install_binary "/usr/local/bin"
else
  INSTALL_DIR="$HOME/.local/bin"
  install_binary "$INSTALL_DIR"

  # PATH kontrolü ve otomatik ekleme
  if [[ ":${PATH}:" != *":$INSTALL_DIR:"* ]]; then
    added=0
    for rc_file in "$HOME/.zshrc" "$HOME/.bashrc" "$HOME/.bash_profile"; do
      if [ -f "$rc_file" ] && ! grep -q "$INSTALL_DIR" "$rc_file" 2>/dev/null; then
        echo "export PATH=\"$INSTALL_DIR:\$PATH\"" >> "$rc_file"
        echo "✅ PATH ayarı $rc_file dosyasına eklendi."
        added=1
      fi
    done

    if [ "$added" -eq 0 ] && [ ! -f "$HOME/.zshrc" ] && [ ! -f "$HOME/.bashrc" ]; then
      echo "export PATH=\"$INSTALL_DIR:\$PATH\"" >> "$HOME/.profile"
      echo "✅ PATH ayarı $HOME/.profile dosyasına eklendi."
    fi

    echo
    echo "⚠️  PATH güncellendi. Lütfen terminali kapatıp açın veya şunu çalıştırın:"
    echo "  source ~/.zshrc   # zsh kullanıyorsanız"
    echo "  source ~/.bashrc  # bash kullanıyorsanız"
  fi
fi
