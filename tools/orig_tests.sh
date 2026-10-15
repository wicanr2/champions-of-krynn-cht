#!/usr/bin/env bash
# 需要原版素材的測試。tools/ci.sh 不跑這些（它們預設 skip），這支補上。
#
#   DOSGOLEM_RICH2_DIR=~/cht/rich2/workplace \
#   DOSGOLEM_FD2_DIR=~/cht/fd2/org_game/炎龍騎士團/FLAME2 \
#   tools/orig_tests.sh
#
# 每一組的原版目錄各自**唯讀**掛到容器的 /orig（一次只掛一組）。
# 沒設環境變數或目錄不存在的那一組會明講略過——**略過不等於通過**。
#
#   DOSGOLEM_RICH2_DIR   大富翁2：底下要有 ida/RUN_full.EXE 與 orig/大宇大富翁2/DOS/game/RICH2
#   DOSGOLEM_FD2_DIR     炎龍騎士團 2：FD2.EXE 所在的遊戲目錄
#
# 本專案不含任何原版檔案。
set -uo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

fail=0
step() { printf '\n\033[1m=== %s\033[0m\n' "$1"; }

# run <名稱> <原版目錄> <go test 參數…>：其餘環境變數由呼叫端先 export。
run() {
  local name="$1" dir="$2"
  shift 2
  step "$name"
  if [[ -z "$dir" ]]; then
    echo "  略過：沒設原版目錄。⚠ 略過不等於通過"
    return
  fi
  if [[ ! -d "$dir" ]]; then
    echo "  ✗ 原版目錄不存在：$dir"
    fail=1
    return
  fi
  local log="workplace/orig-$name.log"
  mkdir -p workplace
  # 這些測試不能加 -short：大富翁2 的長流程在 -short 下會跳過。
  if DOSGOLEM_ORIG="$dir" DOSGOLEM_TIMEOUT="${DOSGOLEM_TIMEOUT:-60m}" \
    tools/go.sh test -v "$@" >"$log" 2>&1; then
    echo "  ✓（$(grep -cE -- '^--- PASS' "$log") 個頂層測試通過；紀錄：$log）"
  else
    echo "  ✗ 沒過（紀錄：$log）"
    grep -E -- '^\s*--- FAIL|^FAIL|^panic:|_test\.go:[0-9]+:' "$log" | head -40 | sed 's/^/      /'
    fail=1
  fi
  local skips
  skips=$(grep -cE -- '^\s*--- SKIP' "$log")
  [[ $skips -gt 0 ]] && echo "  略過 $skips 個（多半是缺另一組原版）"
}

DOSGOLEM_TEST_EXE=/orig/ida/RUN_full.EXE \
DOSGOLEM_TEST_ROOT="/orig/orig/大宇大富翁2/DOS/game/RICH2" \
  run rich2 "${DOSGOLEM_RICH2_DIR:-}" ./oracle/ ./apps/rich2/

DOSGOLEM_FD2_EXE=/orig/FD2.EXE DOSGOLEM_FD2_ROOT=/orig \
  run fd2 "${DOSGOLEM_FD2_DIR:-}" ./internal/machine/

printf '\n'
if [[ $fail -eq 0 ]]; then
  echo "有跑的都通過（略過的見上面）。"
else
  echo "有項目沒過，見上面的 ✗。"
fi
exit $fail
