import sys
base = '/home/anr2/cht/golden_box/champions_of_krynn/workplace/tr-jako'
for b in sys.argv[1:]:
    inp = [l.rstrip('\n').split('\t') for l in open(f'{base}/batches/{b}.tsv', encoding='utf-8')][1:]
    raw = open(f'{base}/done-ko/{b}.tsv', encoding='utf-8').read()
    lines = raw.split('\n')
    assert lines[-1] == ''
    out = [l.split('\t') for l in lines[:-1]]
    assert out[0] == ['key', 'translation', 'note'], out[0]
    out = out[1:]
    errs = 0
    if len(out) != len(inp): print(b, 'COUNT', len(out), len(inp)); errs += 1
    for i, (o, r) in enumerate(zip(out, inp), 1):
        if len(o) != 3: print(b, i, 'COLS', o); errs += 1; continue
        if o[0] != r[0]: print(b, i, 'KEY'); errs += 1
        t = o[1]
        if not t or t != t.strip() or '\r' in t: print(b, i, 'WS', repr(t)); errs += 1
        if len(t) > len(r[1]): print(b, i, 'LEN', len(t), len(r[1]), t, '|', r[1]); errs += 1
    print(b, 'rows', len(out), 'errs', errs)
