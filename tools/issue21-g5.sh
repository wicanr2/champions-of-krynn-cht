#!/usr/bin/env bash
# 原封重播 G5 的七組情境，僅改輸出位置與 Docker 資源／標籤。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if [[ "${1:-}" != --verify-only ]]; then
bash "$ROOT/tools/issue21-hd-img.sh" python -B -c '
from pathlib import Path
p=Path("/src/workplace/probe/issue21-g5-runner");p.mkdir(exist_ok=True)
s=Path("/src/tools/g5-receipt.sh").read_text()
s=s.replace("ROOT=\"$(cd \"$(dirname \"${BASH_SOURCE[0]}\")/..\" && pwd)\"", "ROOT=\""+str(Path("/src"))+"\"")
s=s.replace("$ROOT/workplace/probe/g5-receipt", "$ROOT/workplace/probe/issue21-g5-receipt")
# ROOT 在主機用實際儲存庫路徑，其他命令、步數與輸入維持原樣。
s=s.replace("ROOT=\"/src\"", "ROOT=\""+__import__("sys").argv[1]+"\"")
(p/"g5.sh").write_text(s)
' "$ROOT"
docker() {
 local args=("$@")
 if [[ "${args[0]:-}" == run ]]; then args=(run --label cok.issue=21 "${args[@]:1}"); fi
 command docker "${args[@]}"
}
export -f docker
export DOSGOLEM_CPUS=2 DOSGOLEM_MEM=2g DOSGOLEM_TIMEOUT=15m
timeout 35m bash "$ROOT/workplace/probe/issue21-g5-runner/g5.sh"
fi
bash "$ROOT/tools/issue21-hd-img.sh" python -B -c '
from pathlib import Path
import json
p=Path("/src/workplace/probe/issue21-g5-receipt")
want=dict(demo="70356ecc82bf",guide="b950f535a3b1",journal="f7cc762077e9",map="b43868123abc",menu="124364912e3b",quiz="de66a48ec636",town="fe7e63826edf")
rows=[]
for name,expected in want.items():
 r=json.loads((p/name/"result.json").read_text());s=r["shots"][0];got=s["overlay_sha256"][:12];rows.append(dict(scene=name,expected=expected,actual=got,pass_=got==expected))
out=Path("/src/workplace/hd/out/issue21-g5.json");out.write_text(json.dumps(dict(all_pass=all(r["pass_"] for r in rows),items=rows),indent=2)+"\n")
print(rows);assert all(r["pass_"] for r in rows)
'
