#!/usr/bin/env python3
"""Builds report.html from results.json and the benchmark sources."""
import html, json, math, os

HERE = os.path.dirname(os.path.abspath(__file__))
D = json.load(open(os.path.join(HERE, "results.json")))
M, R = D["meta"], D["results"]
TIE = 1.10  # differences under 10% are treated as a tie (shared 2-vCPU VM, run-to-run sd is 5-15%)

rows = []
for bid, v in R.items():
    cr, go, p1 = v["crystal"], v["go"], v["go_p1"]
    ratio = go["median_ms"] / cr["median_ms"]  # >1: Crystal faster
    winner = "crystal" if ratio > TIE else ("go" if ratio < 1 / TIE else "tie")
    rows.append(dict(id=bid, cat=v["category"], title=v["title"], desc=v["desc"], cr=cr, go=go, p1=p1, ratio=ratio, winner=winner))

gm = math.exp(sum(math.log(r["ratio"]) for r in rows) / len(rows))
nc = sum(r["winner"] == "crystal" for r in rows)
ng = sum(r["winner"] == "go" for r in rows)
nt = sum(r["winner"] == "tie" for r in rows)
best_c = sorted(rows, key=lambda r: -r["ratio"])[:3]
best_g = sorted(rows, key=lambda r: r["ratio"])[:3]

def fmt_ms(x):
    return f"{x:,.0f}" if x >= 100 else f"{x:,.1f}"

def fmt_x(r):
    return f"{r:.1f}×" if r >= 1 else f"{1 / r:.1f}×"

# ---- ratio chart (log scale, parity in the middle) ------------------------------------
LO, HI = -2.0, 4.0  # log2: Go 4x faster ... Crystal 16x faster
def pos(r):
    return (math.log2(r) - LO) / (HI - LO) * 100

center = pos(1.0)
chart = []
ticks = [(0.25, "4× Go"), (0.5, "2×"), (1, "equal"), (2, "2×"), (4, "4×"), (8, "8×"), (16, "16× Crystal")]
axis = "".join(f'<span class="tick" style="left:{pos(t):.2f}%">{html.escape(l)}</span>' for t, l in ticks)
grid = "".join(f'<i class="gl{" gl0" if t == 1 else ""}" style="left:{pos(t):.2f}%"></i>' for t, _ in ticks)
last_cat = None
for r in rows:
    if r["cat"] != last_cat:
        chart.append(f'<div class="cat">{html.escape(r["cat"])}</div>')
        last_cat = r["cat"]
    p = pos(max(min(r["ratio"], 2 ** HI), 2 ** LO))
    left, width = (center, p - center) if p >= center else (p, center - p)
    cls = {"crystal": "bar-c", "go": "bar-g", "tie": "bar-t"}[r["winner"]]
    who = {"crystal": "Crystal", "go": "Go", "tie": "tie"}[r["winner"]]
    lab = "within noise" if r["winner"] == "tie" else f'{fmt_x(r["ratio"])} {who}'
    # put the value label on the outer side of the bar, inside the plot
    if p >= center:
        lstyle = f"left:{min(p + 1, 78):.2f}%"
    else:
        lstyle = f"left:{min(p, center) - 1:.2f}%;transform:translateX(-100%)"
    tip = f'{r["title"]}: Crystal {fmt_ms(r["cr"]["median_ms"])} ms, Go {fmt_ms(r["go"]["median_ms"])} ms'
    chart.append(
        f'<div class="row" title="{html.escape(tip)}"><div class="rl">{html.escape(r["title"])}</div>'
        f'<div class="track">{grid}<b class="{cls}" style="left:{left:.2f}%;width:{max(width, 0.4):.2f}%"></b>'
        f'<span class="val" style="{lstyle}">{html.escape(lab)}</span></div></div>'
    )

# ---- results table ---------------------------------------------------------------------
trs = []
last_cat = None
for r in rows:
    if r["cat"] != last_cat:
        trs.append(f'<tr class="catrow"><th colspan="8" scope="colgroup">{html.escape(r["cat"])}</th></tr>')
        last_cat = r["cat"]
    cw = "win" if r["winner"] == "crystal" else ""
    gw = "win" if r["winner"] == "go" else ""
    badge = {"crystal": "Crystal", "go": "Go", "tie": "Tie"}[r["winner"]]
    trs.append(
        f'<tr><th scope="row"><span class="t">{html.escape(r["title"])}</span><span class="d">{html.escape(r["desc"])}</span></th>'
        f'<td class="n {cw}">{fmt_ms(r["cr"]["median_ms"])}</td><td class="n {gw}">{fmt_ms(r["go"]["median_ms"])}</td>'
        f'<td class="n dim">{fmt_ms(r["p1"]["median_ms"])}</td>'
        f'<td class="n"><span class="pill p-{r["winner"]}">{badge}</span> {"" if r["winner"] == "tie" else fmt_x(r["ratio"])}</td>'
        f'<td class="n">{r["cr"]["peak_rss_mb"]:.0f}</td><td class="n">{r["go"]["peak_rss_mb"]:.0f}</td>'
        f'<td class="n dim">±{r["cr"]["stdev_ms"] / r["cr"]["median_ms"] * 100:.0f}% / ±{r["go"]["stdev_ms"] / r["go"]["median_ms"] * 100:.0f}%</td></tr>'
    )

# ---- source listings ------------------------------------------------------------------
def src(path):
    return html.escape(open(os.path.join(HERE, path)).read())

def hl(r):
    return f'<li><strong>{html.escape(r["title"])}</strong> {fmt_x(r["ratio"])} faster in {"Crystal" if r["ratio"] > 1 else "Go"} ({fmt_ms(r["cr"]["median_ms"])} ms vs {fmt_ms(r["go"]["median_ms"])} ms)</li>'

TEMPLATE = open(os.path.join(HERE, "report_template.html")).read()
out = (TEMPLATE
       .replace("{{GM}}", f"{gm:.2f}")
       .replace("{{NC}}", str(nc)).replace("{{NG}}", str(ng)).replace("{{NT}}", str(nt)).replace("{{N}}", str(len(rows)))
       .replace("{{BEST_C}}", "".join(hl(r) for r in best_c))
       .replace("{{BEST_G}}", "".join(hl(r) for r in best_g))
       .replace("{{AXIS}}", axis).replace("{{CHART}}", "".join(chart)).replace("{{TABLE}}", "".join(trs))
       .replace("{{CPU}}", html.escape(M["cpu"])).replace("{{CORES}}", str(M["cores"])).replace("{{MEM}}", f'{M["mem_mb"] / 1024:.0f}')
       .replace("{{OS}}", html.escape(M["os"])).replace("{{GOV}}", html.escape(M["go_version"].replace("go version ", "")))
       .replace("{{CRV}}", html.escape(M["crystal_version"].split("LLVM")[0].strip())).replace("{{REPS}}", str(M["reps"]))
       .replace("{{DATE}}", M["date"].split(" ")[0])
       .replace("{{SRC_GO}}", src("go/main.go")).replace("{{SRC_CR}}", src("crystal/bench.cr")).replace("{{SRC_RUN}}", src("run_bench.py")))
open(os.path.join(HERE, "report.html"), "w").write(out)
print("report.html", len(out), "bytes; geomean", round(gm, 3), "wins", nc, ng, nt)
