# Crystal vs Go benchmark

20 matched programs written in both languages, covering math, sorting, hash maps, strings, regex, JSON, SHA-256 and concurrency. Every test returns a checksum, and the runner confirms both languages produce the same value before the timing counts.

## Results

Run on a shared 2-core Xeon 2.1 GHz VM, Ubuntu 24.04, Go 1.24.7, Crystal 1.11.2 (`--release`). Time is the median of 7 runs after one warm-up.

Overall: Crystal is about **1.35x faster** (geometric mean). Crystal won 9 tests, Go won 5, and 6 were ties (within 10%).

| Test | Crystal ms | Go ms | Result |
|---|---|---|---|
| Regex (log scan) | 63 | 846 | Crystal 13.4x |
| Channel ping-pong | 29 | 136 | Crystal 4.6x |
| JSON parse (typed) | 75 | 188 | Crystal 2.5x |
| Spawn 100k tasks | 28 | 54 | Crystal 1.9x |
| String build and split | 128 | 220 | Crystal 1.7x |
| JSON parse (dynamic) | 109 | 181 | Crystal 1.7x |
| Binary trees (GC stress) | 571 | 803 | Crystal 1.4x |
| Hash map, int keys | 221 | 271 | Crystal 1.2x |
| Fibonacci (recursive) | 218 | 272 | Crystal 1.3x |
| Sort 2M ints | 264 | 232 | Go 1.1x |
| Sieve, 20M | 122 | 101 | Go 1.2x |
| Hash map, string keys | 98 | 77 | Go 1.3x |
| JSON serialize | 49 | 32 | Go 1.5x |
| N-body simulation | 294 | 190 | Go 1.6x |
| Collatz, Mandelbrot, matmul, N-Queens, LCS, SHA-256 | | | Tie |

Plain loops and math are close. The big gaps are in libraries: regex, JSON, strings, memory allocation and task switching.

Caveats: Crystal 1.11.2 is an older release, the machine is small and shared, and the code is ordinary idiomatic code with no hand-tuning. Compile time, startup time, multi-core scaling and I/O were not measured. The full table with memory use is in `report.html`.

## Files

```
go/main.go           Go benchmarks
crystal/bench.cr     Crystal benchmarks
run_bench.py         runs everything, writes results.json
build_report.py      builds report.html from results.json
report_template.html page template used by build_report.py
results.json         raw results from the run above
report.html          the finished report
```

## Run it locally

Needs Linux, Go 1.24 or newer, Crystal, Python 3 and GNU `time` (used to measure peak memory).

Install on Ubuntu or Debian:

```sh
sudo apt install crystal time libssl-dev libyaml-dev libgmp-dev libpcre2-dev libevent-dev libgc-dev
```

Go comes from https://go.dev/dl/ or your package manager. On macOS, `brew install crystal go gnu-time`, then change `/usr/bin/time` in `run_bench.py` to the path of `gtime`.

Build both programs from this folder:

```sh
mkdir -p out
(cd go && go build -o ../out/bench_go .)
(cd crystal && crystal build --release --no-debug -o ../out/bench_cr bench.cr)
```

Run the full suite (takes about 3 minutes) and rebuild the report:

```sh
python3 run_bench.py        # set REPS=11 for more runs per test
python3 build_report.py     # writes report.html
```

Run a single test, for example regex with 5 timed runs:

```sh
./out/bench_cr regex 5
./out/bench_go regex 5
```

Test names: `fib collatz sieve mandelbrot nbody matmul nqueens lcs binarytrees sort hash_int hash_string string_builder regex json_dynamic json_typed json_serialize sha256 chan_pingpong spawn_sum`

Close other heavy programs while it runs. Results shift with CPU model, core count and Crystal version, so expect different absolute numbers on your machine.
