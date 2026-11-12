#!/usr/bin/env python3
"""Runs every benchmark for Crystal and Go, records timings + peak RSS, writes results.json."""
import json, os, platform, statistics, subprocess, sys, time

HERE = os.path.dirname(os.path.abspath(__file__))
OUT = os.path.join(HERE, "out")
REPS = int(os.environ.get("REPS", "7"))

BENCHES = [
    # (id, category, title, what it does)
    ("fib", "Compute", "Recursive Fibonacci", "fib(38): ~126M function calls, pure integer recursion"),
    ("collatz", "Compute", "Collatz chains", "Longest Collatz chain for every start below 2M (tight integer loop)"),
    ("sieve", "Compute", "Sieve of Eratosthenes", "Primes below 20M in a 20 MB boolean array"),
    ("mandelbrot", "Compute", "Mandelbrot", "1000x1000 grid, 100 iterations, float64 math"),
    ("nbody", "Compute", "N-body simulation", "5 bodies, 3M steps, float64 math + sqrt"),
    ("matmul", "Compute", "Matrix multiply", "400x400 float64, naive triple loop on flat arrays"),
    ("nqueens", "Compute", "N-Queens (n=14)", "Bitmask backtracking, 365,596 solutions"),
    ("lcs", "Compute", "Longest common subsequence", "Dynamic programming on two 8000-char strings (64M cells)"),
    ("binarytrees", "Data structures", "Binary trees", "Allocate/walk/discard ~17M tree nodes (GC stress)"),
    ("sort", "Data structures", "Sort 2M integers", "Standard-library sort on 2M pseudo-random Int64"),
    ("hash_int", "Data structures", "Hash map (int keys)", "1M inserts, 2M lookups, 1M deletes"),
    ("hash_string", "Data structures", "Hash map (string keys)", "500k word-count updates over 50k distinct string keys"),
    ("string_builder", "Strings & regex", "String building & split", "Build a ~25 MB string, scan every byte, split on commas"),
    ("regex", "Strings & regex", "Regular expressions", "Two regexes over a 15 MB, 200k-line log (match + capture)"),
    ("json_dynamic", "JSON", "JSON parse (dynamic)", "Parse 7 MB / 50k objects into generic values, then walk them"),
    ("json_typed", "JSON", "JSON parse (typed)", "Parse the same document straight into typed structs/classes"),
    ("json_serialize", "JSON", "JSON serialize", "Serialize 50k typed objects back to JSON text"),
    ("sha256", "Crypto", "SHA-256", "Hash a 32 MB buffer 4 times"),
    ("chan_pingpong", "Concurrency", "Channel ping-pong", "300k unbuffered round trips between two goroutines/fibers"),
    ("spawn_sum", "Concurrency", "Spawn 100k tasks", "Spawn 100k goroutines/fibers that each send a value over a channel"),
]

CONFIGS = [
    ("crystal", [os.path.join(OUT, "bench_cr")], {}),
    ("go", [os.path.join(OUT, "bench_go")], {}),
    ("go_p1", [os.path.join(OUT, "bench_go")], {"GOMAXPROCS": "1"}),
]


def run_one(cmd, extra_env):
    env = dict(os.environ)
    env.update(extra_env)
    # GNU time launches the benchmark from a tiny process, so its max-RSS figure is the
    # benchmark's own peak (os.wait4 from Python would inherit Python's ~14 MB footprint).
    rss_file = os.path.join(OUT, ".rss")
    t0 = time.monotonic()
    p = subprocess.run(["/usr/bin/time", "-f", "%M", "-o", rss_file] + cmd,
                       stdout=subprocess.PIPE, stderr=subprocess.PIPE, env=env)
    wall = (time.monotonic() - t0) * 1000
    if p.returncode != 0:
        return {"error": p.stderr.decode()[:300] or f"exit status {p.returncode}"}
    rss_kb = float(open(rss_file).read().strip().splitlines()[-1])
    out = p.stdout
    rec = json.loads(out.decode().strip().splitlines()[-1])
    t = rec["times_ms"]
    return {
        "median_ms": statistics.median(t),
        "min_ms": min(t),
        "max_ms": max(t),
        "stdev_ms": statistics.pstdev(t),
        "times_ms": t,
        "checksum": rec["checksum"],
        "stable": rec["stable"],
        "peak_rss_mb": rss_kb / 1024.0,
        "process_wall_ms": wall,
    }


def sh(cmd):
    try:
        return subprocess.check_output(cmd, shell=True, text=True, stderr=subprocess.STDOUT).strip()
    except Exception as e:
        return str(e)


def main():
    results = {}
    for bid, cat, title, desc in BENCHES:
        results[bid] = {"category": cat, "title": title, "desc": desc}
        for name, cmd, env in CONFIGS:
            r = run_one(cmd + [bid, str(REPS)], env)
            results[bid][name] = r
            if "error" in r:
                print(f"{bid:15s} {name:8s} ERROR {r['error']}", flush=True)
            else:
                print(f"{bid:15s} {name:8s} median {r['median_ms']:9.1f} ms  rss {r['peak_rss_mb']:7.1f} MB  sd {r['stdev_ms']:.1f}", flush=True)
        a, b = results[bid]["crystal"], results[bid]["go"]
        results[bid]["checksum_match"] = ("error" not in a and "error" not in b and a["checksum"] == b["checksum"] == results[bid]["go_p1"].get("checksum"))
    meta = {
        "date": time.strftime("%Y-%m-%d %H:%M UTC", time.gmtime()),
        "reps": REPS,
        "cpu": sh("lscpu | grep 'Model name' | sed 's/.*: *//'"),
        "cores": os.cpu_count(),
        "mem_mb": int(sh("free -m | awk '/Mem:/{print $2}'") or 0),
        "os": sh("grep PRETTY_NAME /etc/os-release | cut -d'\"' -f2"),
        "kernel": platform.release(),
        "go_version": sh("go version"),
        "crystal_version": sh("crystal --version | head -3 | tr '\\n' ' '"),
    }
    with open(os.path.join(HERE, "results.json"), "w") as f:
        json.dump({"meta": meta, "results": results}, f, indent=1)
    print("wrote results.json")


if __name__ == "__main__":
    main()
