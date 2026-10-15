# 197 80386 實模式子集

狀態：**READY**（CPU 語意為強推論，見 §6）
日期：2026-10-04
前置：[`012-cpu-386-subset.md`](012-cpu-386-subset.md)（0x66 前綴的起點）、
[`002-cpu-8086.md`](002-cpu-8086.md)
後續：[`200-eob3-oracle-adapter.md`](200-eob3-oracle-adapter.md)（第一個要求 32 位元暫存器的實模式程式）

---

## 1. 問題

《魔眼殺機三》的 `AESOP.EXE` 是實模式 MZ，但整支程式用 80386 指令：

- 開機先做 CPU 偵測：`pushfd` 改 EFLAGS.AC（bit 18），再 `popfd`／`pushfd` 讀回，
  AC 立得起來才是 486，立不起來是 386，連 EFLAGS.ID 都試。
- 內文大量使用 32 位元暫存器、`0x66` 運算元大小前綴、`0x67` 位址大小前綴與 SIB、
  `0F 8x` 近跳躍、`movzx／movsx`、`setcc`、`shld／shrd`。
- 沒支援的 `0F xx` 在 8086 模型裡是 `POP CS`，會**安靜地**把 CS 換掉然後跑到別處，
  症狀是離奇的跳躍，離真正缺的指令很遠。

先前的子集（`012`）只涵蓋 FD2 用到的幾種形狀。本規格把它擴到「**實模式下 32 位元運算
與定址的一般形式**」，並規定沒涵蓋的一律失敗即關閉。

## 2. 支援範圍

只在 `Model80386` 啟用。`Model8086`（預設）完全不變：`0F` 仍是 `POP CS`，
`60`–`6F` 仍是 `70`–`7F` 的別名，前綴 `64`、`65`、`66`、`67` 不被辨識。
SingleStepTests 8088 語料全綠是這條的閘門。

### 2.1 前綴與暫存器

| 項目 | 規則 |
|---|---|
| 暫存器 | `R[i]` 是低 16 位元，`Hi[i]` 是高 16 位元。16 位元寫入不動高半 |
| `FS`、`GS` | 另存 `FSeg`、`GSeg`，不在 `Seg[4]`；`64`／`65` 前綴可覆寫 |
| `0x66` | 把「運算元大小敏感」的 opcode 改成 32 位元（`sizeSensitive` 表） |
| `0x67` | ModRM 改走 ModRM32（含 SIB）；`A0`–`A3` 的位移變 32 位元 |
| 狀態檔與快照 | `Hi`、`FSeg`、`GSeg` 都進 `SaveState` 與記憶體內快照、回呼框 |

`sizeSensitive` 之外的 opcode（位元組運算、短位移跳躍的位移、I/O 位元組、旗標運算、
BCD、浮點跳脫）不看運算元大小，帶 `0x66` 照 16 位元路徑跑。

### 2.2 指令清單

主 opcode，帶 `0x66` 時的 32 位元形式：

- ALU 八種的 form 1、3、5（`ADD OR ADC SBB AND SUB XOR CMP`，含 `r/m32, r32`、
  `r32, r/m32`、`EAX, imm32`），`81／83` 立即值群組。
- `INC／DEC／PUSH／POP` 暫存器（`40`–`5F`）、`PUSH imm32`（`68`）、`PUSH imm8`（`6A`，
  符號延伸）、`PUSHAD／POPAD`（`60／61`，推的是進入時的 ESP，POPAD 丟掉 ESP 那格）。
- `IMUL r32, r/m32, imm`（`69／6B`）、`TEST`（`85／A9`）、`XCHG`（`87／91`–`97`）、
  `MOV`（`89／8B／B8`–`BF／C7`／moffs `A1／A3`）、`MOV r/m32, Sreg`（`8C`，暫存器目的地零延伸、
  記憶體目的地只寫 16 位元）、`LEA`（`8D`）、`POP r/m32`（`8F`，先彈再解位址）。
- `CWDE／CDQ`（`98／99`）、`PUSHFD／POPFD`（`9C／9D`，見 §5）。
- 字串指令 `MOVS／CMPS／STOS／LODS／SCAS` 的 dword 形式（`A5／A7／AB／AD／AF`，含 `REP`）。
- 位移群組 `C1／D1／D3`（`ROL ROR RCL RCR SHL SHR SAL SAR`）、`F7` 群組（`TEST NOT NEG MUL IMUL DIV IDIV`）、
  `FF` 群組中的 `INC／DEC／PUSH`。

`0F` 雙位元組：

- `0F 80`–`8F` Jcc rel16／rel32、`0F 90`–`9F` SETcc。
- `0F A0／A8` PUSH FS／GS，`0F A1／A9` POP FS／GS。
- `0F A3／AB／B3／BB／BA` BT、BTS、BTR、BTC（暫存器與 imm8 形式）。
- `0F A4／A5／AC／AD` SHLD、SHRD。16 位元形式計數大於 16 時 Intel 標為未定義，**失敗即關閉**。
- `0F BC／BD` BSF、BSR。`0F AF` IMUL r, r/m。
- `0F B2／B4／B5` LSS、LFS、LGS。
- `0F B6／B7／BE／BF` MOVZX、MOVSX。

**不在清單上的 `0F xx` 與 `0x66` 組合一律回錯誤**：

- `0F xx`：`80386 子集沒有這一道`。
- 在 `sizeSensitive` 內但沒實作的 `66 xx`：`80386 子集沒有這一道`。
  包含 `66` 加短條件跳躍（`70`–`7F`）與 `66 FF /2、/3、/4、/5`（32 位元的 call／jmp）。
- 不記成「未實作」之後繼續跑，因為繼續跑的結果不可信。

## 3. 實模式的 64 KB 段限

實模式的段限是 64 KB，有效位址超過 FFFFh 在真機是 #GP。

- 0x67 的 ModRM32 算出的有效位址大於 FFFFh：**失敗即關閉**，錯誤帶位址。
  `LEA` 例外（它只算數，不存取，可以得到 32 位元的結果）。
- `A0`–`A3` 的 moffs32 超過 FFFFh、`0F 8x` 的 rel32 目標超過 FFFFh、
  BT 系指令位址調整後超過 FFFFh：同樣失敗即關閉。
- `0x67` 下靠隱含位址暫存器運作的 opcode（`INS／OUTS`、字串指令、`XLAT`、`LOOPcc／JCXZ`）
  會改用 ESI／EDI／EBX／ECX，**一律未實作**，帶 `0x67` 就失敗。
- 這些全都是**回報錯誤，不模擬 #GP**：沒有看過任何一個要靠 #GP 才能正確執行的實模式
  遊戲程式，模擬例外只會讓錯誤晚一步出現。

## 4. 32 位元運算與旗標

實作在 `internal/cpu/wide.go`。旗標語意照 Intel 軟體開發手冊，並與 DOSBox-X 的
lazy flags（`src/cpu/flags.cpp`、`src/cpu/instructions.h`）逐項交叉核對。
**兩處刻意不照 DOSBox-X**：

| 項目 | 本實作 | DOSBox-X（讀原始碼） |
|---|---|---|
| 32 位元結果的 PF | 只看結果的低位元組（Intel） | `PARITY32` 把四個位元組的奇偶 XOR 起來（`flags.cpp` 433 至 434 行），與真機不同 |
| `SHR r/m32, 1` 的 OF | 原運算元的最高位 | `lf_var1d > 0x80000000`（`flags.cpp` 407 行），原運算元恰為 `0x80000000` 時少了 1 |
| 16 位元 SHLD 的 OF | 結果與原 dst 的符號位不同（與 Bochs 相同） | `DSHLW` 把 `lf_var1d` 設成 `(dst << 16) \| src`，OF 用 `lf_resw ^ lf_var1w`，也就是跟 src 比（`instructions.h` 984 至 992 行、`flags.cpp` 738 行），與 Intel 不符 |

手冊標為「未定義」的旗標取確定值，兩次跑同一個狀態結果一致：

- 多位移位的 OF 沿用與計數 1 同一個算式（OF 依 DOSBox-X 的算式），移位的 AF 清 0。
- `MUL／IMUL` 不動 SF／ZF／AF／PF，`DIV／IDIV` 不動任何旗標。
- 程式去讀這些旗標屬於程式自己的行為，沒有硬體語料可驗。

其餘：

- 32 位元移位與旋轉的計數遮成 5 位元（遮罩 31）。**8 位元與 16 位元的移位仍走 8086 的核心
  （`shiftRotate`），計數不遮罩**，CL 大於等於 32 時與 386 不同（已知限制，見 §7）。
- `RCL／RCR` 的 32 位元計數 1 到 31 以位元序列的參考實作逐位元驗證。
- `MUL／IMUL／DIV／IDIV` 以 `math/big` 驗證。除法溢位與除以零引發 #DE
  （`divideFault`），推回的位址是**指令起點**（386 的行為，8086 推的是下一道）。
- `PUSH ESP` 推減之前的值（286 以上，DOSBox-X 同）。
- `PUSHFD／POPFD` 見 §5。

## 5. EFLAGS 與 CPU 偵測

`PUSHFD` 推出 16 位元旗標零延伸成 32 位元，`POPFD` 只收低半、高半**丟掉**。

結果是：程式把 AC（bit 18）立起來再 `popfd`，下一次 `pushfd` 讀回來是 0，
偵測得到「AC 立不起來」，也就是 **386**。這是刻意的選擇：

- 不模擬 AC 的話，486 的程式路徑（`CPUID`、`BSWAP`、`CMPXCHG`）都要實作；
  386 就夠了。
- 真正的 386 也是這樣回答。

測試：`TestEFLAGSACDetectionYields386`、`TestPushfdPopfd`。

## 6. 驗收與證據等級

| 項目 | 證據 | 等級 |
|---|---|---|
| 8086 行為不變 | SingleStepTests 8088 語料（`testdata/8088`）全綠；`TestPrefixesAreAliasesBeforeModel386` | 已證實 |
| 32 位元 ALU、移位、乘除的結果 | 與獨立參考實作（位元序列、`math/big`）逐一比對 | 已證實（對 Intel 手冊語意） |
| 旗標細節與 DOSBox-X 的差異 | 讀 `src/cpu/instructions.h`、`src/cpu/lazyflags.h`、`src/cpu/core_normal/prefix_66.h`、`prefix_66_0f.h`、`prefix_0f.h` 交叉對照；**沒有**在 DOSBox-X 上跑同一份測資比對 | 強推論 |
| 對真機的行為 | 沒有 80386 實機或硬體語料可比 | 未知 |
| 失敗即關閉 | `TestUnknown0FFailsClosed`、`TestUnsupported66FailsClosed`、`TestUnsupported67FailsClosed`、`TestModRM32LimitAndLEA` | 已證實 |
| AESOP 實跑 | 開機、CINE、主選單、`CHGEN` 路徑都在這組子集內跑通（見 `200` 與 `docs/re/eob3-dosgolem-oracle.md`） | 已證實（只對這條路徑） |

新增的單元測試在 `internal/cpu/ops386_wide_test.go`。

## 7. 已知限制

- 保護模式、描述表、分頁、`CR0`、`LGDT` 等一概沒有。
- `66` 加 `CALL／JMP`（`FF /2..5`、`E8／E9／EA`）的 32 位元 EIP 形式沒有實作，因為實模式 EIP
  要維持在 16 位元。需要時**加一條規格再做**。
- 8 位元與 16 位元的移位與旋轉（`D0／D1／D2／D3／C0／C1` 的非 `66` 形式）在 `Model80386` 下
  仍用 8086 的 `shiftRotate`，計數不遮成 5 位元，CL 大於等於 32 時結果與 386 不同。
  AESOP 目前走過的路徑沒有觸發（沒有逐一核對所有 8／16 位元移位的計數來源，所以不寫成已證實）。
  要修的時候讓 `Model80386` 先遮罩 31 再呼叫 `shiftRotate`，SingleStepTests 的 8088 語料
  只跑 `Model8086`，不受影響。
- `0F` 之下只做 §2.2 的清單。沒碰到的指令（例如 `BSWAP`、`CMPXCHG`、`XADD`、`CPUID`）
  遇到就失敗，這是設計。
