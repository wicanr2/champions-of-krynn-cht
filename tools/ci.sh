#!/usr/bin/env bash
# 本機 CI：**收工前這一支要全綠。**
#
#   tools/ci.sh
#   DOSGOLEM_CPUS=2 tools/ci.sh    # 這台機器上還有別人在跑時
#
# 全部跑在 docker 裡（走 tools/go.sh），不裝任何東西到系統環境。
#
# 五步，每一步各自回報，最後一行才是結論。**跳過不等於通過**——
# 缺語料的那一步會明講自己沒驗到什麼。
set -uo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

fail=0
# 只驗**追蹤中**的套件與檔案：workplace/ 底下常有未追蹤的實驗程式
# （同一個目錄好幾個 main），`./...` 會把它們一起編而失敗。
mapfile -t GOFILES < <(git ls-files '*.go')
mapfile -t PKGS < <(printf '%s\n' "${GOFILES[@]}" | xargs -n1 dirname | sort -u | sed 's|^|./|')
step() { printf '\n\033[1m=== %s\033[0m\n' "$1"; }
bad() { echo "  ✗ $1"; fail=1; }

# ---- 1. 格式 -------------------------------------------------------------
# 整個 repo 都要乾淨。十條分支合併的時候順手格式化過一輪，所以這裡不必
# 再只挑「本分支動過的檔」——那個做法會在合併進 master 之後變成「零個檔案」，
# 於是這一步永遠是綠的而且什麼都沒檢查。
# ---- 0. 合併殘留 ---------------------------------------------------------
# gofmt 只看 Go 檔；Markdown、腳本裡留著衝突標記不會讓任何一步失敗。
step "合併衝突標記（追蹤中的所有檔案）"
markers=$(git grep -nIE '^(<<<<<<<|>>>>>>>) ' -- . ':!*.patch' 2>/dev/null)
if [[ -n "$markers" ]]; then
  bad "還有衝突標記："
  echo "$markers" | head -20 | sed 's/^/      /'
else
  echo "  ✓"
fi

step "gofmt（整個 repo）"
out=$(DOSGOLEM_GO_CMD=gofmt tools/go.sh -l "${GOFILES[@]}" 2>&1)
if [[ -n "$out" ]]; then
  bad "沒有格式化："
  echo "$out" | sed 's/^/      /'
else
  echo "  ✓"
fi

# ---- 2. 靜態檢查 ---------------------------------------------------------
step "go vet"
tools/go.sh vet "${PKGS[@]}" && echo "  ✓" || bad "go vet 有問題"

# ---- 3. 單元測試（不含語料）----------------------------------------------
# **語料留給下一步**：`./...` 已經含 internal/cpu，不加 `-short` 的話同一份
# 727 MB 語料會在這一輪跑兩次（實測序列 182–287 秒一次），一輪 CI 平白多花
# 三分多鐘做同一件事。
# Ebiten 的套件在 init 就開 GLFW，沒有 DISPLAY 會 panic：一律包在 xvfb-run 裡
# （映像見 tools/docker/go-ebiten.Dockerfile）。
#
# 用 -v 跑、完整紀錄寫到 workplace/ci-test.log，這裡只印失敗與略過的統計。
# **略過也要講出來**：需要原版素材的測試沒設環境變數就 skip，
# 不印的話「全部通過」看起來像連原版對拍都驗過了。
step "go test（不含 CPU 語料）"
log=workplace/ci-test.log
mkdir -p workplace
if DOSGOLEM_GO_CMD=xvfb-run tools/go.sh -a go test -short -v "${PKGS[@]}" >"$log" 2>&1; then
  echo "  ✓（$(grep -c -- '^--- PASS' "$log") 個頂層測試通過）"
else
  bad "測試沒過（完整紀錄：$log）"
  grep -E -- '^\s*--- FAIL|^FAIL|^panic:|_test\.go:[0-9]+:' "$log" | head -60 | sed 's/^/      /'
fi
skips=$(grep -cE -- '^\s*--- SKIP' "$log")
if [[ $skips -gt 0 ]]; then
  echo "  略過 $skips 個（⚠ 跳過不等於通過）。原因前幾名："
  # -v 模式下 t.Skip 的訊息印在 `--- SKIP` 那一行的前一行。
  grep -B1 -E -- '^\s*--- SKIP' "$log" | grep -E -- '_test\.go:[0-9]+: ' \
    | sed -E 's/^ +[^:]+:[0-9]+: //' | sort | uniq -c | sort -rn | head -8 | sed 's/^/      /'
  echo "      原版素材的環境變數見 README「怎麼跑」；tools/go.sh 會依前綴傳進容器。"
fi

# ---- 4. CPU 語料 ---------------------------------------------------------
# 判準是**全部通過**（`docs/spec/002` §5）：CPU 的錯不會報錯，只會讓上層
# 在幾百萬道指令之後畫錯一個像素。
#
# 路徑要與 `internal/cpu/singlestep_test.go` 的 testDir 一致
# （`testdata/8088/`，底下直接是 `00.json.gz`…）。看錯一層的話這一步會
# 一直「跳過」，而跳過在輸出裡不長得像失敗——CPU 就這樣一路沒被驗過。
step "CPU 語料（SingleStepTests）"
corpus=(testdata/8088/*.json.gz)
if [[ -e "${corpus[0]}" ]]; then
  echo "  語料 ${#corpus[@]} 檔"
  if [[ ${#corpus[@]} -lt 300 ]]; then
    bad "語料只有 ${#corpus[@]} 檔（完整的一份是 323）——先跑 tools/fetch_cputests.sh"
  fi
  tools/go.sh test ./internal/cpu -run TestSingleStep && echo "  ✓" || bad "語料沒全綠"
else
  echo "  跳過：testdata/8088/ 底下沒有語料。用 tools/fetch_cputests.sh 抓。"
  echo "  ⚠ **跳過不等於通過**：CPU 的驗收判準是「全部通過」，這一輪沒有驗到 CPU。"
fi

printf '\n'
if [[ $fail -eq 0 ]]; then
  echo "全部通過。"
else
  echo "有項目沒過，見上面的 ✗。"
fi
exit $fail
