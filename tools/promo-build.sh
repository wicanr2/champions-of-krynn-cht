#!/usr/bin/env bash
# 推廣影片組裝（規格 016）：依剪輯表把 workplace/promo/seg/<段>/ 的畫格與音訊剪成片段，加字卡與片尾，接成
# workplace/promo/cok-promo.mp4（1920×1200、30 fps、H.264＋AAC）。前置：tools/promo-rec.sh 錄好各段。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"; P="$ROOT/workplace/promo"; FONTS=/usr/share/fonts/opentype/noto
# 段 開始秒 長度秒 字卡（-＝無） 背景曲（-＝無）。原版開場劇情沒有音樂（規格 014 §1），這幾段以新編配樂的
# 渲染檔（workplace/music/out/<曲>.wav）墊底，同一首曲子跨片段接續播放。
CUTS='title	0.0	10.8	-	-
open	1.3	3.7	劇情全文繁體中文	sanctuary
open	7.0	2.5	大地圖與地名	sanctuary
open	20.5	2.8	人物對話	sanctuary
open	23.4	3.1	手札內文自動顯示	sanctuary
open	32.5	3.0	選單中文化	-
throtl	9.6	9.0	3D 牆面 HD 重繪	-
throtl	23.7	2.2	自動地圖	-
throtl	26.8	4.0	軟體世界攻略對照	-
battle	0.0	12.0	戰鬥畫面 HD 重繪	-
langs	0.0	6.9	繁中・简中・日本語・한국어・English	outpost'
for s in $(printf '%s\n' "$CUTS" | cut -f1 | sort -u); do test -f "$P/seg/$s/audio.s16le" || { echo "缺 $P/seg/$s" >&2; exit 1; }; done
test -f "$FONTS/NotoSansCJK-Bold.ttc"; test -f "$ROOT/workplace/music/out/finale.wav"
rm -rf "$P/build"; mkdir -p "$P/build"; printf '%s\n' "$CUTS" > "$P/build/cuts.tsv"
DR=(docker run --rm --network none -u "$(id -u):$(id -g)" --memory 2g --cpus 4 --pids-limit 512 --log-opt max-size=10m --log-opt max-file=3)
# 字卡（透明 PNG）與片尾
timeout 300s "${DR[@]}" -i -v "$FONTS:/fonts:ro" -v "$P:/p" --entrypoint python cok-img:r1 -I - <<'PY'
from PIL import Image, ImageDraw, ImageFont
B = lambda n: ImageFont.truetype('/fonts/NotoSansCJK-Bold.ttc', n, index=3)  # TC
R = lambda n: ImageFont.truetype('/fonts/NotoSansCJK-Regular.ttc', n, index=3)
for i, line in enumerate(open('/p/build/cuts.tsv', encoding='utf-8').read().splitlines()):
    cap = line.split('\t')[3].strip('-')
    im = Image.new('RGBA', (1920, 1200), (0, 0, 0, 0))
    if cap:
        d = ImageDraw.Draw(im); f = B(52)
        w = d.textlength(cap, font=f)
        d.rounded_rectangle((40, 1030, 40 + w + 64, 1130), 18, fill=(0, 0, 0, 190))
        d.text((72, 1050), cap, font=f, fill=(255, 230, 150, 255))
    im.save(f'/p/build/cap{i:02d}.png')
im = Image.new('RGB', (1920, 1200), (8, 10, 24)); d = ImageDraw.Draw(im)
def center(y, s, f, c):
    d.text(((1920 - d.textlength(s, font=f)) / 2, y), s, font=f, fill=c)
center(330, '克萊恩英豪', B(150), (255, 214, 120))
center(530, 'Champions of Krynn　繁體中文版', B(64), (235, 235, 235))
center(700, '原版 DOS 程式即時覆繪・HD 重繪・原版 AdLib 音效與新編配樂', R(42), (190, 200, 220))
center(770, '繁中・简中・日本語・한국어・English', R(42), (190, 200, 220))
center(1060, '原作 © 1990 TSR, Inc. / Strategic Simulations, Inc.　非官方玩家中文化', R(30), (130, 140, 160))
im.save('/p/build/end.png')
PY
# 逐段剪輯：畫格 2 倍最近鄰放大、字卡 0.3 秒後淡入、頭尾 0.25 秒淡入淡出
i=0; : > "$P/build/list.txt"; declare -A BED
while IFS=$'\t' read -r seg start dur cap bed; do
  n=$(printf '%02d' "$i"); first=$(awk -v s="$start" 'BEGIN{printf "%d", s*30+0.5}'); frames=$(awk -v d="$dur" 'BEGIN{printf "%d", d*30+0.5}')
  bedin=(); amix="[1:a]anull"
  if [[ "$bed" != - ]]; then
    off="${BED[$bed]:-0}"; bedin=(-ss "$off" -t "$dur" -i "/m/$bed.wav")
    amix="[3:a]aresample=48000,aformat=channel_layouts=stereo,volume=0.8[b];[1:a][b]amix=inputs=2:duration=first:normalize=0"
    BED[$bed]=$(awk -v a="$off" -v d="$dur" 'BEGIN{printf "%.3f", a+d}')
  fi
  fo=$(awk -v d="$dur" 'BEGIN{printf "%.2f", d-0.25}'); ce=$(awk -v d="$dur" 'BEGIN{printf "%.2f", d-0.2}')
  timeout 600s "${DR[@]}" -v "$P:/p" -v "$ROOT/workplace/music/out:/m:ro" cok-ffmpeg:r1 ffmpeg -hide_banner -loglevel error -y \
    -framerate 30 -start_number "$first" -i "/p/seg/$seg/frames/%06d.png" \
    -f s16le -ar 48000 -ac 2 -ss "$start" -t "$dur" -i "/p/seg/$seg/audio.s16le" -loop 1 -framerate 30 -i "/p/build/cap$n.png" \
    "${bedin[@]}" -filter_complex "[0:v]scale=1920:1200:flags=neighbor,format=rgba[g];[2:v]fade=t=in:st=0.3:d=0.3:alpha=1[c];[g][c]overlay=0:0:enable='between(t,0.3,$ce)':shortest=1,fade=t=in:st=0:d=0.25,fade=t=out:st=$fo:d=0.25,format=yuv420p[v];$amix[m];[m]afade=t=in:st=0:d=0.15,afade=t=out:st=$fo:d=0.25[a]" \
    -map "[v]" -map "[a]" -frames:v "$frames" -c:v libx264 -preset slow -crf 18 -r 30 -c:a aac -b:a 192k -ar 48000 "/p/build/c$n.mp4"
  echo "file 'c$n.mp4'" >> "$P/build/list.txt"; i=$((i+1))
done <<< "$CUTS"
timeout 300s "${DR[@]}" -v "$P:/p" -v "$ROOT/workplace/music/out:/m:ro" cok-ffmpeg:r1 ffmpeg -hide_banner -loglevel error -y \
  -loop 1 -framerate 30 -i /p/build/end.png -ss 0 -t 6 -i /m/finale.wav \
  -filter_complex "[0:v]fade=t=in:st=0:d=0.6,fade=t=out:st=5.2:d=0.8,format=yuv420p[v];[1:a]aresample=48000,afade=t=in:st=0:d=0.4,afade=t=out:st=4.5:d=1.5[a]" \
  -map "[v]" -map "[a]" -t 6 -c:v libx264 -preset slow -crf 18 -r 30 -c:a aac -b:a 192k -ar 48000 -ac 2 /p/build/end.mp4
echo "file 'end.mp4'" >> "$P/build/list.txt"
timeout 300s "${DR[@]}" -v "$P:/p" cok-ffmpeg:r1 ffmpeg -hide_banner -loglevel error -y -f concat -safe 0 -i /p/build/list.txt \
  -c copy -movflags +faststart /p/cok-promo.mp4
timeout 60s "${DR[@]}" -v "$P:/p:ro" cok-ffmpeg:r1 ffprobe -v error -show_entries format=duration:stream=codec_name,width,height,r_frame_rate,duration -of compact /p/cok-promo.mp4
