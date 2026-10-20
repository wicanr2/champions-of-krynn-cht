#!/usr/bin/env bash
# 發行包（規格 012）：Linux AppImage、Windows zip、macOS universal zip，各分原版與原版＋HD 兩包。
#
#   tools/package.sh <版本> [all|linux|windows|macos]       # 可散布 patch → dist-all/<版本>/patch/
#   COK_WITH_DATA=1 tools/package.sh <版本> all            # 本機完整版 → dist-all/<版本>/full-local/
#
# 版本格式 v.<主>.<次>.<修>-YYYYMMDD（retro-remake-release-versioning）。
# ⚠ full-local 含原版遊戲與倚天字型：只留被忽略的 dist-all/，絕不推 git、絕不上傳。
# 一律 Docker、--network none。dosgolem 以已推送的 dosgolem/cok 分支 HEAD git archive 出來建置，
# 版本與 dosgolem commit 以 -ldflags -X 注入。可散布包打包後解開做外洩掃描，命中即刪產物並失敗。
# 架構與輔助工具取自幽靈騎士 tools/package.sh（其規格 012）。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
VER="${1:?用法：tools/package.sh <版本> [all|linux|windows|macos]}"
TARGET="${2:-all}"
[[ "$VER" =~ ^v\.[0-9]+\.[0-9]+\.[0-9]+-[0-9]{8}$ ]] || { echo "[package] 版本須為 v.<主>.<次>.<修>-YYYYMMDD：$VER" >&2; exit 1; }

W="$ROOT/workplace"
DG="$W/dosgolem"
DG_BRANCH="dosgolem/cok"
MODCACHE="$DG/workplace/gomodcache"
GOCACHE_DIR="$W/pkg-gocache"
STAGE="$W/pkg-stage"
UNI="$W/unifont-src/unifont-17.0.05"
UNIFONT_TAR="$W/unifont-src/unifont-17.0.05.tar.gz"
UNIFONT_TAR_SHA="f287cffb26e22723aa36e6684869b0f3ff3bfb822c4b01008bd847911ec1b631"
THEME="$W/hd/theme"
ORIG_TREE="$W/input/dos/ckrynn"
ETEN_SRC="/home/anr2/cht/etan_font/ET353S/FILES"
# 規格 012 §3.3：沿用本機既有映像，以映像 ID 釘住（只使用，不刪除、不改標籤）。
GO_IMAGE="dosgolem-go-ebiten:1.24-r1"
PY_IMAGE="python:3.13-slim"
MAC_IMAGE="${COK_MAC_IMAGE:-psgolem-osxcross:1.24-15.5-r1}"  # go1.24.13、MacOSX15.5 SDK（共用映像，只使用；原 psychicwar-osxcross r2 已不在本機）
APPIMAGE_IMAGE="hr-appimage:runtime-recovery-r1"
declare -A IMAGE_ID=(
  [$GO_IMAGE]="sha256:8ed6bdee2bce1f32f804e888f942b25bbf62f2165a43344bb4587f3371de0581"
  [$MAC_IMAGE]="sha256:0a4be2d7ac9651c4f3ae22a2e504fbab5d72d90851686c7e8512c8b1dd622a7b"
  [$APPIMAGE_IMAGE]="sha256:e961c807e7810d5fca6b2e880e9ebed49207074e9871683de926f646c28d41fa"
)
MAC_MIN="11.0"
WITH_DATA="${COK_WITH_DATA:-}"
KIND=patch SUF=""
[[ "$WITH_DATA" == 1 ]] && KIND=full-local SUF="-with-data"
DIST="$ROOT/dist-all/$VER/$KIND"

die() { echo "[package] $*" >&2; exit 1; }
dr() {
  timeout 30m docker run --rm --network none --memory 6g --cpus 4 --pids-limit 512 \
    --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" -e HOME=/tmp "$@"
}

# --- 前置檢查 --------------------------------------------------------------------
[[ "$(git config user.email)" == "wicanr2@gmail.com" ]] || die "git 身分不是 wicanr2@gmail.com"
DC="$(git -C "$DG" rev-parse HEAD)"
[[ "$(git ls-remote origin "refs/heads/$DG_BRANCH" | cut -f1)" == "$DC" ]] || die "dosgolem $DC 不是已推送的 $DG_BRANCH HEAD"
[[ -z "$(git -C "$DG" status --porcelain --untracked-files=no)" ]] || die "dosgolem 有未提交的改動"
[[ "$(sha256sum "$UNIFONT_TAR" | cut -c1-64)" == "$UNIFONT_TAR_SHA" ]] || die "Unifont tarball 雜湊不符"
for f in COPYING OFL-1.1.txt; do [[ -f "$UNI/$f" ]] || die "缺 Unifont $f（先解開 $UNIFONT_TAR）"; done
[[ -d "$MODCACHE" ]] || die "缺模組快取 $MODCACHE"
# HD 主題（G8）還沒有時只出原版包
EDS=(base); [[ -f "$THEME/theme.json" ]] && EDS=(base hd)
echo "[package] 版別：${EDS[*]}"
[[ -f "$ORIG_TREE/START.EXE" ]] || die "缺原版 $ORIG_TREE（外洩掃描與完整版都要）"
for f in ASCFONT.15 SPCFONT.15 STDFONT.15; do [[ -f "$ETEN_SRC/$f" ]] || die "缺倚天 $f"; done
for i in "$GO_IMAGE" "$PY_IMAGE" "$MAC_IMAGE" "$APPIMAGE_IMAGE"; do
  id="$(docker image inspect -f '{{.Id}}' "$i" 2>/dev/null)" || die "缺映像 $i"
  [[ -z "${IMAGE_ID[$i]:-}" || "${IMAGE_ID[$i]}" == "$id" ]] || die "映像 $i 的 ID 變了：$id"
done

echo "[package] 版本 $VER（$KIND），dosgolem $DC"
rm -rf "$STAGE"
mkdir -p "$STAGE/src" "$STAGE/common/font" "$STAGE/common/text" "$STAGE/out" "$STAGE/leak/eten" "$GOCACHE_DIR" "$DIST"
git -C "$DG" archive "$DC" | tar -x -C "$STAGE/src"

LDFLAGS="-s -w -X main.version=$VER -X main.dosgolemCommit=$DC"
GOENV=(-e GOMODCACHE=/gomodcache -e GOCACHE=/gocache -e GOFLAGS=-mod=mod -e GOPROXY=off -e GOSUMDB=off -e GOTOOLCHAIN=local)
GOMOUNT=(-v "$STAGE/src:/src" -v "$MODCACHE:/gomodcache:ro" -v "$GOCACHE_DIR:/gocache" -v "$STAGE/out:/out")

# --- 共同內容：譯文、字型、授權文件 ------------------------------------------------
cp text/*.tsv "$STAGE/common/text/"
# 規格 015 §3.3：英文手札逐字稿只在本機（不進 Git），有就放進發行包
[[ -f "$W/journal/journal.en.tsv" ]] && cp "$W/journal/journal.en.tsv" "$STAGE/common/text/"
# 規格 013：玩家名音譯資料（CMU 發音詞典 BSD 式、譯音表 CC BY-SA 4.0，授權隨檔）
cp -r text/cmudict "$STAGE/common/text/"
cp text/LICENSE-translit-table.md "$STAGE/common/text/"
for d in text/translit-*/; do [[ -d "$d" ]] && cp -r "$d" "$STAGE/common/text/"; done  # 規格 015：各語言的玩家名音譯
CATALOGS=(ecl-text.zh-TW.tsv ui-text.zh-TW.tsv name.zh-TW.tsv composed-templates.tsv host-text.zh-TW.tsv journal.zh-TW.tsv
  walkthrough.zh-TW.tsv map-guide.zh-TW.tsv map-geo.tsv translit-chars.zh-TW.tsv)
dr -v "$ROOT/tools:/tools:ro" -v "$ROOT/text:/text:ro" -v "$UNI:/uni:ro" -v "$ETEN_SRC:/eten:ro" -v "$STAGE:/stage" \
  -w /text "$PY_IMAGE" sh -c "
  python -I /tools/eten_font.py ${CATALOGS[*]} --unifont /uni/font/precompiled/unifont_t-17.0.05.hex --out /stage/common/font/cok-unifont.golemfnt &&
  python -I /tools/eten_font.py ${CATALOGS[*]} --asc /eten/ASCFONT.15 --spc /eten/SPCFONT.15 --std /eten/STDFONT.15 --out /stage/leak/eten/cok-eten.golemfnt &&
  for s in 256 512; do python -I /tools/appicon.py /stage/common/font/cok-unifont.golemfnt /stage/icon-\$s.png \$s; done"
# 規格 015：繁中以外的語言各以自己的譯文字集從 Unifont 建字型 font/cok-<代碼>.golemfnt（cok-play 依代碼找）
for ui in text/ui-text.*.tsv; do
  code="${ui#text/ui-text.}"; code="${code%.tsv}"
  [[ "$code" == zh-TW || "$code" == en ]] && continue
  LANG_CATALOGS=($(cd text && ls ./*."$code".tsv))
  # 該語言有自己的玩家名音譯表時，音譯用字也要收（簡中 77 字只出現在這裡）
  [[ -f "text/translit-$code/translit-chars.zh-TW.tsv" ]] && LANG_CATALOGS+=("translit-$code/translit-chars.zh-TW.tsv")
  # 規格 015 §4.1：簡中用大陸字形、日文用日本字形，其餘（韓文）沿用台灣字源
  case "$code" in zh-CN) hex=unifont-17.0.05.hex ;; ja) hex=unifont_jp-17.0.05.hex ;; *) hex=unifont_t-17.0.05.hex ;; esac
  [[ -f "$UNI/font/precompiled/$hex" ]] || die "缺 Unifont $hex（從 $UNIFONT_TAR 解開）"
  dr -v "$ROOT/tools:/tools:ro" -v "$ROOT/text:/text:ro" -v "$UNI:/uni:ro" -v "$STAGE:/stage" -w /text "$PY_IMAGE" \
    python -I /tools/eten_font.py "${LANG_CATALOGS[@]}" --unifont "/uni/font/precompiled/$hex" \
    --out "/stage/common/font/cok-$code.golemfnt"
  echo "[package] 語言 $code 字型：${#LANG_CATALOGS[@]} 份表，$hex"
done
# 規格 015 §3.1：英文版字型由 Courier Prime Bold（OFL）產生，2 倍 16×16、3 倍 24×24
if [[ -f text/host-text.en.tsv ]]; then
  COURIER="$W/fonts-en/courierprime__CourierPrime-Bold.ttf"
  [[ -f "$COURIER" ]] || die "缺英文字型來源 $COURIER"
  docker image inspect cok-img:r1 >/dev/null 2>&1 || die "缺映像 cok-img:r1（tools/docker/img.Dockerfile，英文字型要 Pillow）"
  for c in 16 24; do
    n=cok-en; [[ $c == 24 ]] && n=cok-en-3x
    dr -v "$ROOT/tools:/tools:ro" -v "$W/fonts-en:/f:ro" -v "$STAGE:/stage" cok-img:r1 \
      python -I /tools/en_font.py --ttf /f/courierprime__CourierPrime-Bold.ttf --cell $c --out "/stage/common/font/$n.golemfnt"
  done
  cp "$W/fonts-en/courierprime__OFL.txt" "$STAGE/common/font/OFL-CourierPrime.txt"
fi
cp "$UNI/OFL-1.1.txt" "$STAGE/common/font/OFL-1.1.txt"
cp "$UNI/COPYING" "$STAGE/common/font/COPYING-unifont"
cp LICENSE "$STAGE/common/LICENSE"
cp "$STAGE/src/LICENSE" "$STAGE/common/LICENSE-dosgolem"
cp "$STAGE/src/audio/nukedopl/COPYING.LGPL" "$STAGE/common/COPYING.LGPL"
cp "$STAGE/src/audio/nukedopl/SOURCE.md" "$STAGE/common/nukedopl-SOURCE.md"
cp docs/release/讀我.txt "$STAGE/common/讀我.txt"
# 規格 014：遊戲中配樂（Codex 以原版標題曲為基礎的新編曲，OPL2 寫入序列；cok-play 即時合成）
MUSIC="$W/music/out"
if [[ -f "$MUSIC/music.tsv" ]]; then
  mkdir -p "$STAGE/common/music"
  cp "$MUSIC/music.tsv" "$STAGE/common/music/"
  while IFS=$'\t' read -r key _; do
    [[ "$key" == key ]] && continue
    cp "$MUSIC/$key.opl" "$STAGE/common/music/" || die "缺配樂 $key.opl"
  done < "$MUSIC/music.tsv"
fi

build_linux() {
  dr "${GOENV[@]}" "${GOMOUNT[@]}" -w /src "$GO_IMAGE" sh -c "
    go build -trimpath -ldflags '$LDFLAGS' -o /out/linux/cok-play ./cmd/cok-play &&
    go version -m /out/linux/cok-play > /out/modinfo-linux.txt &&
    cp \$(go env GOROOT)/LICENSE /out/go-LICENSE && go version > /out/go-version.txt"
}
build_linux
dr -v "$ROOT/tools:/tools:ro" -v "$MODCACHE:/gomodcache:ro" -v "$STAGE:/stage" "$PY_IMAGE" \
  python -I /tools/release/third_party.py /stage/out/modinfo-linux.txt /gomodcache /stage/common
cp "$STAGE/out/go-LICENSE" "$STAGE/common/licenses/go-LICENSE"
printf '另含 Go 標準程式庫與執行期（%s，BSD-3-Clause，licenses/go-LICENSE）。\n' "$(cut -d' ' -f3 "$STAGE/out/go-version.txt")" >> "$STAGE/common/THIRD-PARTY.md"
printf '%s\n' "" "另含玩家名音譯資料（規格 013）：" \
  "- CMU 發音詞典（text/cmudict/，BSD 式授權，見 text/cmudict/LICENSE）" \
  "- 英語人名譯音表（text/translit-table.tsv，改作自中文維基百科，CC BY-SA 4.0，見 text/LICENSE-translit-table.md）" >> "$STAGE/common/THIRD-PARTY.md"

# payload <目標目錄> <版別>：HD 版放主題；完整版另放原版、倚天字型與本機自用說明。
payload() {
  if [[ "$2" == hd ]]; then
    mkdir -p "$1/hd"
    cp -r "$THEME"/. "$1/hd/"  # 整個主題：戰鬥、人物、效果、牆面（walls/）與完整來源（exact.json）都在子檔
  fi
  [[ "$WITH_DATA" == 1 ]] || return 0
  mkdir -p "$1/original" "$1/font"
  cp -r "$ORIG_TREE"/. "$1/original/"
  cp "$STAGE/leak/eten/cok-eten.golemfnt" "$1/font/cok-eten.golemfnt"
  printf '%s\n' "本機自用完整版：內含原版遊戲《Champions of Krynn》，著作權屬原權利人。" \
    "內附倚天中文字型（購買授權，限本機使用），預設優先使用；刪除 font/cok-eten.golemfnt 即改用 GNU Unifont。" \
    "請勿散布、上傳或分享本檔案。" > "$1/本機自用-請勿散布.txt"
}
scan() {  # scan <解開目錄>：禁止來源為原版、倚天來源與本機倚天字型
  dr -v "$ROOT/tools:/tools:ro" -v "$1:/pkg:ro" -v "$ORIG_TREE:/leak0:ro" -v "$ETEN_SRC:/leak1:ro" -v "$STAGE/leak/eten:/leak2:ro" \
    "$PY_IMAGE" python -I /tools/release/leak_scan.py /pkg /leak0 /leak1 /leak2
}
# check <目錄> <版別>：可散布包做外洩掃描；完整版確認原版在包內。兩種都確認 HD 版有、原版包沒有 hd/theme.json。
check() {
  local hd
  hd="$(find "$1" -path '*hd/theme.json' | head -1)"
  if [[ "$2" == hd ]]; then
    [[ -n "$hd" ]] || die "HD 版缺 hd/theme.json"
    local d; d="$(dirname "$hd")"
    for f in $(cd "$THEME" && find . -type f | sort); do [[ -f "$d/$f" ]] || die "HD 版缺主題檔 $f"; done
  else [[ -z "$hd" ]] || die "原版包不得含 hd/"; fi
  if [[ "$WITH_DATA" == 1 ]]; then
    [[ -n "$(find "$1" -path '*original/START.EXE' | head -1)" ]] || die "完整版缺 original/START.EXE"
    [[ -n "$(find "$1" -path '*font/cok-eten.golemfnt' | head -1)" ]] || die "完整版缺倚天字型"
  else
    scan "$1"
  fi
}
name() { echo "COKCHT-$VER$([[ $1 == hd ]] && echo -hd)$SUF-$2"; }

pack_linux() {  # pack_linux <版別>
  local ed=$1 app="$STAGE/AppDir-$1" out
  out="$DIST/$(name "$ed" x86_64.AppImage)"
  mkdir -p "$app/usr/share/doc/cok-cht"
  cp -r "$STAGE/common/." "$app/"
  cp "$STAGE/out/linux/cok-play" "$app/cok-play"
  cp LICENSE "$app/usr/share/doc/cok-cht/LICENSE"
  cp "$STAGE/icon-256.png" "$app/cok-cht.png"
  cp "$STAGE/icon-256.png" "$app/.DirIcon"
  cat > "$app/AppRun" <<'SH'
#!/bin/sh
HERE="$(dirname "$(readlink -f "$0")")"
exec "$HERE/cok-play" "$@"
SH
  chmod +x "$app/AppRun" "$app/cok-play"
  cat > "$app/cok-cht.desktop" <<'DESK'
[Desktop Entry]
Type=Application
Name=克萊恩英豪（繁中）
Name[en]=Champions of Krynn CHT
Exec=cok-play
Icon=cok-cht
Categories=Game;RolePlaying;
Terminal=false
DESK
  payload "$app" "$ed"
  check "$app" "$ed"
  rm -f "$out"
  timeout 15m docker run --rm --network none --memory 2g --cpus 2 --pids-limit 64 \
    --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" -e HOME=/tmp \
    -v "$STAGE:/stage" -v "$DIST:/dist" --entrypoint sh "$APPIMAGE_IMAGE" -c "
      set -eu
      RS=\$(stat -c %s /opt/runtime-x86_64)
      mksquashfs /stage/AppDir-$ed /tmp/app.squashfs -root-owned -noappend -no-progress -comp zstd -Xcompression-level 19 >/dev/null
      cat /opt/runtime-x86_64 /tmp/app.squashfs > '/dist/$(basename "$out")'
      chmod +x '/dist/$(basename "$out")'
      rm -rf /stage/check-appimage-$ed
      unsquashfs -q -o \$RS -d /stage/check-appimage-$ed '/dist/$(basename "$out")' >/dev/null"
  check "$STAGE/check-appimage-$ed" "$ed" || { rm -f "$out"; die "AppImage 檢查失敗"; }
  echo "[package] $out"
}

build_windows() {
  dr "${GOENV[@]}" "${GOMOUNT[@]}" -e GOOS=windows -e GOARCH=amd64 -e CGO_ENABLED=0 -w /src "$GO_IMAGE" sh -c "
    go build -trimpath -ldflags '$LDFLAGS -H windowsgui' -o /out/win/COKCHT.exe ./cmd/cok-play &&
    go version -m /out/win/COKCHT.exe > /out/modinfo-windows.txt"
}
pack_windows() {
  local ed=$1 dir="$STAGE/win-$1/COKCHT" out
  out="$DIST/$(name "$ed" win64.zip)"
  mkdir -p "$dir"
  cp -r "$STAGE/common/." "$dir/"
  cp "$STAGE/out/win/COKCHT.exe" "$dir/"
  payload "$dir" "$ed"
  check "$dir" "$ed"
  rm -f "$out"
  dr -v "$ROOT/tools:/tools:ro" -v "$STAGE:/stage" -v "$DIST:/dist" "$PY_IMAGE" \
    python -I /tools/release/zipdir.py "/stage/win-$ed/COKCHT" "/dist/$(basename "$out")"
  echo "[package] $out"
}

build_macos() {
  mkdir -p "$STAGE/out/mac"
  dr "${GOENV[@]}" "${GOMOUNT[@]}" -e "MIN=$MAC_MIN" -w /src "$MAC_IMAGE" bash -c "
    set -euo pipefail
    eval \"\$(osxcross-conf)\"
    export CGO_ENABLED=1 GOOS=darwin GOPATH=/tmp/gopath MACOSX_DEPLOYMENT_TARGET=\$MIN
    for arch in arm64 amd64; do
      case \$arch in arm64) pre=arm64-apple-\$OSXCROSS_TARGET ;; amd64) pre=x86_64-apple-\$OSXCROSS_TARGET ;; esac
      env GOARCH=\$arch CC=\$pre-clang CXX=\$pre-clang++ CGO_CFLAGS=-mmacosx-version-min=\$MIN CGO_LDFLAGS=-mmacosx-version-min=\$MIN \
        go build -trimpath -ldflags '$LDFLAGS' -o /out/mac/cok-play-\$arch ./cmd/cok-play
    done
    x86_64-apple-\$OSXCROSS_TARGET-lipo -create /out/mac/cok-play-arm64 /out/mac/cok-play-amd64 -output /out/mac/cok-play
    x86_64-apple-\$OSXCROSS_TARGET-lipo -archs /out/mac/cok-play > /out/mac/archs.txt"
  local a; a="$(cat "$STAGE/out/mac/archs.txt")"
  [[ "$a" == *x86_64* && "$a" == *arm64* ]] || die "lipo 架構不齊：$a"
}
pack_macos() {
  local ed=$1 top="$STAGE/mac-$1/COKCHT" app out short
  app="$top/COKCHT.app"
  out="$DIST/$(name "$ed" macos.zip)"
  mkdir -p "$app/Contents/MacOS" "$app/Contents/Resources"
  cp "$STAGE/out/mac/cok-play" "$app/Contents/MacOS/cok-play"
  chmod +x "$app/Contents/MacOS/cok-play"
  cp -r "$STAGE/common/." "$app/Contents/Resources/"
  for f in 讀我.txt LICENSE; do cp "$STAGE/common/$f" "$top/$f"; done
  dr -i -v "$STAGE:/stage" -e "APP=/stage/mac-$ed/COKCHT/COKCHT.app" "$PY_IMAGE" python -I - <<'PY'
import os, struct
body = b""
for size, tag in ((256, b"ic08"), (512, b"ic09")):
    png = open(f"/stage/icon-{size}.png", "rb").read()
    body += tag + struct.pack(">I", len(png) + 8) + png
open(os.environ["APP"] + "/Contents/Resources/cok-cht.icns", "wb").write(b"icns" + struct.pack(">I", len(body) + 8) + body)
PY
  short="$(printf '%s' "$VER" | sed -n 's/^v\.\([0-9]*\.[0-9]*\.[0-9]*\)-.*/\1/p')"
  cat > "$app/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleDevelopmentRegion</key><string>zh_TW</string>
	<key>CFBundleDisplayName</key><string>克萊恩英豪（繁中）</string>
	<key>CFBundleExecutable</key><string>cok-play</string>
	<key>CFBundleIconFile</key><string>cok-cht</string>
	<key>CFBundleIdentifier</key><string>io.github.wicanr2.cokcht</string>
	<key>CFBundleInfoDictionaryVersion</key><string>6.0</string>
	<key>CFBundleName</key><string>COKCHT</string>
	<key>CFBundlePackageType</key><string>APPL</string>
	<key>CFBundleShortVersionString</key><string>$short</string>
	<key>CFBundleVersion</key><string>$VER</string>
	<key>LSApplicationCategoryType</key><string>public.app-category.role-playing-games</string>
	<key>LSMinimumSystemVersion</key><string>$MAC_MIN</string>
	<key>NSHighResolutionCapable</key><true/>
</dict>
</plist>
PLIST
  payload "$app/Contents/Resources" "$ed"
  [[ "$WITH_DATA" == 1 ]] && cp "$app/Contents/Resources/本機自用-請勿散布.txt" "$top/"
  check "$top" "$ed"
  rm -f "$out"
  dr -v "$ROOT/tools:/tools:ro" -v "$STAGE:/stage" -v "$DIST:/dist" "$PY_IMAGE" \
    python -I /tools/release/zipdir.py "/stage/mac-$ed/COKCHT" "/dist/$(basename "$out")"
  echo "[package] $out"
}

case "$TARGET" in
  linux) for e in "${EDS[@]}"; do pack_linux "$e"; done ;;
  windows) build_windows; for e in "${EDS[@]}"; do pack_windows "$e"; done ;;
  macos) build_macos; for e in "${EDS[@]}"; do pack_macos "$e"; done ;;
  all) for e in "${EDS[@]}"; do pack_linux "$e"; done; build_windows; for e in "${EDS[@]}"; do pack_windows "$e"; done
       build_macos; for e in "${EDS[@]}"; do pack_macos "$e"; done ;;
  *) die "目標須為 all、linux、windows 或 macos" ;;
esac

# SHA256SUMS.json：該版本 patch 與 full-local 的全部產物（retro-remake-dist-all-output）。
dr -i -v "$ROOT/dist-all/$VER:/d" -e "VER=$VER" -e "DC=$DC" "$PY_IMAGE" python -I - <<'PY'
import hashlib, json, os
out = {"version": os.environ["VER"], "dosgolem_commit": os.environ["DC"], "files": []}
for kind in ("patch", "full-local"):
    d = f"/d/{kind}"
    if not os.path.isdir(d):
        continue
    for n in sorted(os.listdir(d)):
        p = f"{d}/{n}"
        h = hashlib.sha256(open(p, "rb").read()).hexdigest()
        out["files"].append({"path": f"{kind}/{n}", "size": os.path.getsize(p), "sha256": h})
open("/d/SHA256SUMS.json", "w").write(json.dumps(out, ensure_ascii=False, indent=2) + "\n")
print(f"SHA256SUMS.json：{len(out['files'])} 個檔案")
PY
ls -l "$DIST"
