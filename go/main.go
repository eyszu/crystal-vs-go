package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

const lcgA uint64 = 6364136223846793005
const lcgC uint64 = 1442695040888963407

// ---------------------------------------------------------------- compute

func fib(n int) int {
	if n < 2 {
		return n
	}
	return fib(n-1) + fib(n-2)
}

func benchFib() int64 { return int64(fib(38)) }

func benchCollatz() int64 {
	const limit = 2_000_000
	var bestI, bestLen int64
	for i := int64(1); i < limit; i++ {
		n := i
		steps := int64(0)
		for n != 1 {
			if n&1 == 0 {
				n >>= 1
			} else {
				n = 3*n + 1
			}
			steps++
		}
		if steps > bestLen {
			bestLen = steps
			bestI = i
		}
	}
	return bestI*1000 + bestLen
}

func benchSieve() int64 {
	const n = 20_000_000
	sieve := make([]bool, n+1)
	for i := range sieve {
		sieve[i] = true
	}
	sieve[0], sieve[1] = false, false
	for i := 2; i*i <= n; i++ {
		if sieve[i] {
			for j := i * i; j <= n; j += i {
				sieve[j] = false
			}
		}
	}
	var count int64
	for _, p := range sieve {
		if p {
			count++
		}
	}
	return count
}

func benchMandelbrot() int64 {
	const size = 1000
	const maxIter = 100
	var inside int64
	for py := 0; py < size; py++ {
		ci := 2.0*float64(py)/size - 1.0
		for px := 0; px < size; px++ {
			cr := 3.0*float64(px)/size - 2.0
			zr, zi := 0.0, 0.0
			i := 0
			for ; i < maxIter && zr*zr+zi*zi <= 4.0; i++ {
				zr, zi = zr*zr-zi*zi+cr, 2.0*zr*zi+ci
			}
			if i == maxIter {
				inside++
			}
		}
	}
	return inside
}

func advance(dt float64, x, y, z, vx, vy, vz, m []float64) {
	n := len(x)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			dx := x[i] - x[j]
			dy := y[i] - y[j]
			dz := z[i] - z[j]
			d2 := dx*dx + dy*dy + dz*dz
			mag := dt / (d2 * math.Sqrt(d2))
			vx[i] -= dx * m[j] * mag
			vy[i] -= dy * m[j] * mag
			vz[i] -= dz * m[j] * mag
			vx[j] += dx * m[i] * mag
			vy[j] += dy * m[i] * mag
			vz[j] += dz * m[i] * mag
		}
	}
	for i := 0; i < n; i++ {
		x[i] += dt * vx[i]
		y[i] += dt * vy[i]
		z[i] += dt * vz[i]
	}
}

func energy(x, y, z, vx, vy, vz, m []float64) float64 {
	e := 0.0
	n := len(x)
	for i := 0; i < n; i++ {
		e += 0.5 * m[i] * (vx[i]*vx[i] + vy[i]*vy[i] + vz[i]*vz[i])
		for j := i + 1; j < n; j++ {
			dx := x[i] - x[j]
			dy := y[i] - y[j]
			dz := z[i] - z[j]
			e -= m[i] * m[j] / math.Sqrt(dx*dx+dy*dy+dz*dz)
		}
	}
	return e
}

func benchNbody() int64 {
	const pi = 3.141592653589793
	const solarMass = 4 * pi * pi
	const dpy = 365.24
	x := []float64{0, 4.84143144246472090e+00, 8.34336671824457987e+00, 1.28943695621391310e+01, 1.53796971148509165e+01}
	y := []float64{0, -1.16032004402742839e+00, 4.12479856412430479e+00, -1.51111514016986312e+01, -2.59193146099879641e+01}
	z := []float64{0, -1.03622044471123109e-01, -4.03523417114321381e-01, -2.23307578892655734e-01, 1.79258772950371181e-01}
	vx := []float64{0, 1.66007664274403694e-03 * dpy, -2.76742510726862411e-03 * dpy, 2.96460137564761618e-03 * dpy, 2.68067772490389322e-03 * dpy}
	vy := []float64{0, 7.69901118419740425e-03 * dpy, 4.99852801234917238e-03 * dpy, 2.37847173959480950e-03 * dpy, 1.62824170038242295e-03 * dpy}
	vz := []float64{0, -6.90460016972063023e-05 * dpy, 2.30417297573763929e-05 * dpy, -2.96589568540237556e-05 * dpy, -9.51592254519715870e-05 * dpy}
	m := []float64{solarMass, 9.54791938424326609e-04 * solarMass, 2.85885980666130812e-04 * solarMass, 4.36624404335156298e-05 * solarMass, 5.15138902046611451e-05 * solarMass}

	px, py, pz := 0.0, 0.0, 0.0
	for i := range m {
		px += vx[i] * m[i]
		py += vy[i] * m[i]
		pz += vz[i] * m[i]
	}
	vx[0] = -px / solarMass
	vy[0] = -py / solarMass
	vz[0] = -pz / solarMass

	for s := 0; s < 3_000_000; s++ {
		advance(0.01, x, y, z, vx, vy, vz, m)
	}
	return int64(math.Round(energy(x, y, z, vx, vy, vz, m) * 1e9))
}

func benchMatmul() int64 {
	const n = 400
	a := make([]float64, n*n)
	b := make([]float64, n*n)
	c := make([]float64, n*n)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			a[i*n+j] = float64((i + j) % 10)
			b[i*n+j] = float64((i * j) % 10)
		}
	}
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			s := 0.0
			for k := 0; k < n; k++ {
				s += a[i*n+k] * b[k*n+j]
			}
			c[i*n+j] = s
		}
	}
	total := 0.0
	for _, v := range c {
		total += v
	}
	return int64(total)
}

func queens(n int, all, cols, d1, d2 int64) int64 {
	if cols == all {
		return 1
	}
	var count int64
	avail := all &^ (cols | d1 | d2)
	for avail != 0 {
		bit := avail & -avail
		avail ^= bit
		count += queens(n, all, cols|bit, ((d1|bit)<<1)&all, (d2|bit)>>1)
	}
	return count
}

func benchNqueens() int64 {
	const n = 14
	return queens(n, (int64(1)<<n)-1, 0, 0, 0)
}

func benchLCS() int64 {
	const n = 8000
	a := make([]byte, n)
	b := make([]byte, n)
	x := uint64(12345)
	for i := range a {
		x = x*lcgA + lcgC
		a[i] = 'A' + byte((x>>33)&3)
	}
	for i := range b {
		x = x*lcgA + lcgC
		b[i] = 'A' + byte((x>>33)&3)
	}
	prev := make([]int32, n+1)
	cur := make([]int32, n+1)
	for i := 1; i <= n; i++ {
		ai := a[i-1]
		for j := 1; j <= n; j++ {
			if ai == b[j-1] {
				cur[j] = prev[j-1] + 1
			} else if prev[j] >= cur[j-1] {
				cur[j] = prev[j]
			} else {
				cur[j] = cur[j-1]
			}
		}
		prev, cur = cur, prev
	}
	return int64(prev[n])
}

// ---------------------------------------------------------------- data structures

type Node struct{ l, r *Node }

func bottomUp(d int) *Node {
	if d <= 0 {
		return &Node{}
	}
	return &Node{bottomUp(d - 1), bottomUp(d - 1)}
}

func (n *Node) check() int64 {
	if n.l == nil {
		return 1
	}
	return 1 + n.l.check() + n.r.check()
}

func benchBinaryTrees() int64 {
	const maxDepth = 17
	total := bottomUp(maxDepth + 1).check()
	long := bottomUp(maxDepth)
	for d := 4; d <= maxDepth; d += 4 {
		iters := 1 << (maxDepth - d + 4)
		var c int64
		for i := 0; i < iters; i++ {
			c += bottomUp(d).check()
		}
		total += c
	}
	total += long.check()
	return total
}

func benchSort() int64 {
	const n = 2_000_000
	a := make([]int64, n)
	x := uint64(42)
	for i := range a {
		x = x*lcgA + lcgC
		a[i] = int64(x >> 33)
	}
	slices.Sort(a)
	s := a[0] + a[n/2] + a[n-1]
	for i := 0; i < n; i += 10007 {
		s += a[i]
	}
	return s
}

func benchHashInt() int64 {
	m := make(map[int64]int64)
	x := uint64(7)
	for i := int64(0); i < 1_000_000; i++ {
		x = x*lcgA + lcgC
		m[int64(x>>33)%2_000_000] = i
	}
	var sum int64
	for i := int64(0); i < 2_000_000; i++ {
		if v, ok := m[i]; ok {
			sum += v
		}
	}
	for i := int64(0); i < 2_000_000; i += 2 {
		delete(m, i)
	}
	return sum*7 + int64(len(m))
}

func benchHashString() int64 {
	m := make(map[string]int64)
	x := uint64(99)
	for i := 0; i < 500_000; i++ {
		x = x*lcgA + lcgC
		id := (x >> 33) % 50_000
		m["word"+strconv.FormatUint(id, 10)]++
	}
	var maxc int64
	for _, c := range m {
		if c > maxc {
			maxc = c
		}
	}
	return int64(len(m))*1000 + maxc
}

// ---------------------------------------------------------------- strings / regex

func benchStringBuilder() int64 {
	var sb strings.Builder
	for i := 0; i < 2_000_000; i++ {
		sb.WriteString("item")
		sb.WriteString(strconv.Itoa(i))
		sb.WriteByte(',')
	}
	s := sb.String()
	var count7 int64
	for i := 0; i < len(s); i++ {
		if s[i] == '7' {
			count7++
		}
	}
	var pieces, pieceLen int64
	for _, p := range strings.Split(s, ",") {
		if len(p) > 0 {
			pieces++
			pieceLen += int64(len(p))
		}
	}
	return int64(len(s)) + count7*3 + pieces*7 + pieceLen
}

var logText string
var reEmail = regexp.MustCompile(`[a-z]+[0-9]+@[a-z]+[0-9]+\.example\.com`)
var reStatus = regexp.MustCompile(`status=([0-9]{3})`)

func getLogText() string {
	if logText == "" {
		statuses := [4]int{200, 404, 500, 301}
		var sb strings.Builder
		for i := 0; i < 200_000; i++ {
			st := strconv.Itoa(statuses[i%4])
			if i%10 == 0 {
				sb.WriteString("noise line " + strconv.Itoa(i) + " status=" + st + "\n")
			} else {
				sb.WriteString("user" + strconv.Itoa(i) + "@host" + strconv.Itoa(i%100) + ".example.com visited /path/" + strconv.Itoa(i) + " status=" + st + "\n")
			}
		}
		logText = sb.String()
	}
	return logText
}

func benchRegex() int64 {
	text := getLogText()
	emails := int64(len(reEmail.FindAllStringIndex(text, -1)))
	var statusSum int64
	for _, m := range reStatus.FindAllStringSubmatch(text, -1) {
		v, _ := strconv.Atoi(m[1])
		statusSum += int64(v)
	}
	return emails*1_000_003 + statusSum
}

// ---------------------------------------------------------------- JSON

type Address struct {
	City string `json:"city"`
	Zip  string `json:"zip"`
}

type Person struct {
	ID      int64    `json:"id"`
	Name    string   `json:"name"`
	Email   string   `json:"email"`
	Active  bool     `json:"active"`
	Score   float64  `json:"score"`
	Tags    []string `json:"tags"`
	Address Address  `json:"address"`
}

var jsonBytes []byte
var people []Person

func getJSON() []byte {
	if jsonBytes == nil {
		var sb strings.Builder
		sb.WriteByte('[')
		for i := 0; i < 50_000; i++ {
			if i > 0 {
				sb.WriteByte(',')
			}
			cents := (i * 37 % 10000) | 1
			active := "true"
			if i%3 == 0 {
				active = "false"
			}
			is := strconv.Itoa(i)
			sb.WriteString(`{"id":` + is + `,"name":"user` + is + `","email":"user` + is + `@example.com","active":` + active)
			sb.WriteString(fmt.Sprintf(`,"score":%d.%02d`, cents/100, cents%100))
			sb.WriteString(`,"tags":["t` + strconv.Itoa(i%7) + `","t` + strconv.Itoa(i%11) + `","t` + strconv.Itoa(i%13) + `"]`)
			sb.WriteString(`,"address":{"city":"City` + strconv.Itoa(i%100) + `","zip":"` + strconv.Itoa(10000+i%90000) + `"}}`)
		}
		sb.WriteByte(']')
		jsonBytes = []byte(sb.String())
	}
	return jsonBytes
}

func getPeople() []Person {
	if people == nil {
		if err := json.Unmarshal(getJSON(), &people); err != nil {
			panic(err)
		}
	}
	return people
}

func benchJSONDynamic() int64 {
	var v []any
	if err := json.Unmarshal(getJSON(), &v); err != nil {
		panic(err)
	}
	var sumID, active, tags, city int64
	score := 0.0
	for _, e := range v {
		o := e.(map[string]any)
		sumID += int64(o["id"].(float64))
		if o["active"].(bool) {
			active++
		}
		tags += int64(len(o["tags"].([]any)))
		score += o["score"].(float64)
		city += int64(len(o["address"].(map[string]any)["city"].(string)))
	}
	return sumID + active*3 + tags*7 + city*11 + int64(math.Round(score*100))
}

func benchJSONTyped() int64 {
	var ps []Person
	if err := json.Unmarshal(getJSON(), &ps); err != nil {
		panic(err)
	}
	var sumID, active, tags, city int64
	score := 0.0
	for i := range ps {
		p := &ps[i]
		sumID += p.ID
		if p.Active {
			active++
		}
		tags += int64(len(p.Tags))
		score += p.Score
		city += int64(len(p.Address.City))
	}
	return sumID + active*3 + tags*7 + city*11 + int64(math.Round(score*100))
}

func benchJSONSerialize() int64 {
	b, err := json.Marshal(getPeople())
	if err != nil {
		panic(err)
	}
	return int64(len(b))
}

// ---------------------------------------------------------------- crypto

var shaBuf []byte

func benchSHA256() int64 {
	if shaBuf == nil {
		shaBuf = make([]byte, 32*1024*1024)
		x := uint64(2024)
		for i := range shaBuf {
			x = x*lcgA + lcgC
			shaBuf[i] = byte(x >> 56)
		}
	}
	var acc int64
	for r := 0; r < 4; r++ {
		shaBuf[0] = byte(r)
		d := sha256.Sum256(shaBuf)
		acc = (acc*31 + int64(d[0])*65536 + int64(d[1])*256 + int64(d[2])) % 1_000_000_007
	}
	return acc
}

// ---------------------------------------------------------------- concurrency

func benchChanPingPong() int64 {
	a := make(chan int64)
	b := make(chan int64)
	go func() {
		for v := range a {
			b <- v + 1
		}
		close(b)
	}()
	var s int64
	for i := int64(0); i < 300_000; i++ {
		a <- i
		s += <-b
	}
	close(a)
	return s
}

func benchSpawnSum() int64 {
	res := make(chan int64, 1000)
	var sum int64
	for batch := 0; batch < 100; batch++ {
		for k := 0; k < 1000; k++ {
			i := int64(batch*1000 + k)
			go func() { res <- i * 2 }()
		}
		for k := 0; k < 1000; k++ {
			sum += <-res
		}
	}
	return sum
}

// ---------------------------------------------------------------- driver

func run(name string) int64 {
	switch name {
	case "fib":
		return benchFib()
	case "collatz":
		return benchCollatz()
	case "sieve":
		return benchSieve()
	case "mandelbrot":
		return benchMandelbrot()
	case "nbody":
		return benchNbody()
	case "matmul":
		return benchMatmul()
	case "nqueens":
		return benchNqueens()
	case "lcs":
		return benchLCS()
	case "binarytrees":
		return benchBinaryTrees()
	case "sort":
		return benchSort()
	case "hash_int":
		return benchHashInt()
	case "hash_string":
		return benchHashString()
	case "string_builder":
		return benchStringBuilder()
	case "regex":
		return benchRegex()
	case "json_dynamic":
		return benchJSONDynamic()
	case "json_typed":
		return benchJSONTyped()
	case "json_serialize":
		return benchJSONSerialize()
	case "sha256":
		return benchSHA256()
	case "chan_pingpong":
		return benchChanPingPong()
	case "spawn_sum":
		return benchSpawnSum()
	}
	fmt.Fprintln(os.Stderr, "unknown benchmark:", name)
	os.Exit(2)
	return 0
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: bench <name> <reps>")
		os.Exit(2)
	}
	name := os.Args[1]
	reps, _ := strconv.Atoi(os.Args[2])

	expected := run(name) // warm-up run, also fixes the expected checksum
	stable := true
	times := make([]string, 0, reps)
	for i := 0; i < reps; i++ {
		t0 := time.Now()
		r := run(name)
		el := float64(time.Since(t0).Microseconds()) / 1000.0
		if r != expected {
			stable = false
		}
		times = append(times, strconv.FormatFloat(el, 'f', 3, 64))
	}
	fmt.Printf(`{"bench":"%s","reps":%d,"times_ms":[%s],"checksum":%d,"stable":%t}`+"\n",
		name, reps, strings.Join(times, ","), expected, stable)
}
