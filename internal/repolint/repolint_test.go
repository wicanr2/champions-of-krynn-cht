// Package repolint 放「整個儲存庫的一致性」檢查：編譯與 vet 都會過、
// 只有跑起來或有人去讀才會發現的錯。
//
// 這些檢查都是合併多條分支時實際踩到的（WORKLOG 2026-10-15）。
package repolint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// root 是儲存庫根目錄（本檔在 internal/repolint）。
const root = "../.."

// 同一個執行檔裡兩處 `flag.String("scratch", ...)` 編得過、vet 也過，
// 一執行就 panic「flag redefined」。三條分支各自加了 `-scratch` 時就是這樣。
func TestCommandFlagsAreUnique(t *testing.T) {
	defs := map[string]map[string][]string{} // 目錄 → 旗標名 → 位置
	for _, top := range []string{"cmd", "apps"} {
		err := filepath.WalkDir(filepath.Join(root, top), func(path string, e os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if e.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return err
			}
			if f.Name.Name != "main" {
				return nil
			}
			ast.Inspect(f, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "flag" {
					return true
				}
				// flag.String("名字", ...)；flag.Var(&v, "名字", ...) 的名字在第二個參數。
				arg := 0
				switch sel.Sel.Name {
				case "Var", "TextVar":
					arg = 1
				case "Bool", "Int", "Int64", "Uint", "Uint64", "String", "Float64", "Duration", "Func", "BoolFunc":
				default:
					return true
				}
				if len(call.Args) <= arg {
					return true
				}
				lit, ok := call.Args[arg].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				name, err := strconv.Unquote(lit.Value)
				if err != nil {
					return true
				}
				dir := filepath.Dir(path)
				if defs[dir] == nil {
					defs[dir] = map[string][]string{}
				}
				defs[dir][name] = append(defs[dir][name], fset.Position(lit.Pos()).String())
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(defs) == 0 {
		t.Fatal("一個執行檔都沒找到，路徑可能錯了")
	}
	for dir, names := range defs {
		for name, at := range names {
			if len(at) > 1 {
				t.Errorf("%s：旗標 -%s 定義了 %d 次：%s", dir, name, len(at), strings.Join(at, "、"))
			}
		}
	}
}

// 每份規格都要在 `docs/spec/000-index.md` 裡出現。
//
// 引用規格要連號碼帶檔名，而同一個號碼底下有好幾份；索引漏列的那一份，
// 讀者就只能靠 grep 才知道它存在。合併時發現 17 份是原分支就漏登的。
func TestSpecIndexListsEverySpec(t *testing.T) {
	dir := filepath.Join(root, "docs", "spec")
	idx, err := os.ReadFile(filepath.Join(dir, "000-index.md"))
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var missing []string
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".md") || n == "000-index.md" {
			continue
		}
		if !strings.Contains(string(idx), n) {
			missing = append(missing, n)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("索引沒列到 %d 份規格：%s", len(missing), strings.Join(missing, "、"))
	}
}
