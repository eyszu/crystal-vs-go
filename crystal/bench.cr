require "json"
require "digest/sha256"

LCG_A = 6364136223846793005_u64
LCG_C = 1442695040888963407_u64

# ---------------------------------------------------------------- compute

def fib(n : Int64) : Int64
  n < 2 ? n : fib(n - 1) + fib(n - 2)
end

def bench_fib : Int64
  fib(38_i64)
end

def bench_collatz : Int64
  limit = 2_000_000_i64
  best_i = 0_i64
  best_len = 0_i64
  i = 1_i64
  while i < limit
    n = i
    steps = 0_i64
    while n != 1
      if n & 1 == 0
        n >>= 1
      else
        n = n * 3 + 1
      end
      steps += 1
    end
    if steps > best_len
      best_len = steps
      best_i = i
    end
    i += 1
  end
  best_i * 1000 + best_len
end

def bench_sieve : Int64
  n = 20_000_000
  sieve = Array(Bool).new(n + 1, true)
  sieve[0] = false
  sieve[1] = false
  i = 2
  while i * i <= n
    if sieve[i]
      j = i * i
      while j <= n
        sieve[j] = false
        j += i
      end
    end
    i += 1
  end
  count = 0_i64
  sieve.each { |p| count += 1 if p }
  count
end

def bench_mandelbrot : Int64
  size = 1000
  max_iter = 100
  inside = 0_i64
  size.times do |py|
    ci = 2.0 * py / size - 1.0
    size.times do |px|
      cr = 3.0 * px / size - 2.0
      zr = 0.0
      zi = 0.0
      i = 0
      while i < max_iter && zr * zr + zi * zi <= 4.0
        zr, zi = zr * zr - zi * zi + cr, 2.0 * zr * zi + ci
        i += 1
      end
      inside += 1 if i == max_iter
    end
  end
  inside
end

def advance(dt : Float64, x : Array(Float64), y : Array(Float64), z : Array(Float64),
            vx : Array(Float64), vy : Array(Float64), vz : Array(Float64), m : Array(Float64))
  n = x.size
  i = 0
  while i < n
    j = i + 1
    while j < n
      dx = x[i] - x[j]
      dy = y[i] - y[j]
      dz = z[i] - z[j]
      d2 = dx * dx + dy * dy + dz * dz
      mag = dt / (d2 * Math.sqrt(d2))
      vx[i] -= dx * m[j] * mag
      vy[i] -= dy * m[j] * mag
      vz[i] -= dz * m[j] * mag
      vx[j] += dx * m[i] * mag
      vy[j] += dy * m[i] * mag
      vz[j] += dz * m[i] * mag
      j += 1
    end
    i += 1
  end
  i = 0
  while i < n
    x[i] += dt * vx[i]
    y[i] += dt * vy[i]
    z[i] += dt * vz[i]
    i += 1
  end
end

def energy(x : Array(Float64), y : Array(Float64), z : Array(Float64),
           vx : Array(Float64), vy : Array(Float64), vz : Array(Float64), m : Array(Float64)) : Float64
  e = 0.0
  n = x.size
  i = 0
  while i < n
    e += 0.5 * m[i] * (vx[i] * vx[i] + vy[i] * vy[i] + vz[i] * vz[i])
    j = i + 1
    while j < n
      dx = x[i] - x[j]
      dy = y[i] - y[j]
      dz = z[i] - z[j]
      e -= m[i] * m[j] / Math.sqrt(dx * dx + dy * dy + dz * dz)
      j += 1
    end
    i += 1
  end
  e
end

def bench_nbody : Int64
  pi = 3.141592653589793
  solar_mass = 4 * pi * pi
  dpy = 365.24
  x = [0.0, 4.84143144246472090e+00, 8.34336671824457987e+00, 1.28943695621391310e+01, 1.53796971148509165e+01]
  y = [0.0, -1.16032004402742839e+00, 4.12479856412430479e+00, -1.51111514016986312e+01, -2.59193146099879641e+01]
  z = [0.0, -1.03622044471123109e-01, -4.03523417114321381e-01, -2.23307578892655734e-01, 1.79258772950371181e-01]
  vx = [0.0, 1.66007664274403694e-03 * dpy, -2.76742510726862411e-03 * dpy, 2.96460137564761618e-03 * dpy, 2.68067772490389322e-03 * dpy]
  vy = [0.0, 7.69901118419740425e-03 * dpy, 4.99852801234917238e-03 * dpy, 2.37847173959480950e-03 * dpy, 1.62824170038242295e-03 * dpy]
  vz = [0.0, -6.90460016972063023e-05 * dpy, 2.30417297573763929e-05 * dpy, -2.96589568540237556e-05 * dpy, -9.51592254519715870e-05 * dpy]
  m = [solar_mass, 9.54791938424326609e-04 * solar_mass, 2.85885980666130812e-04 * solar_mass, 4.36624404335156298e-05 * solar_mass, 5.15138902046611451e-05 * solar_mass]

  px = 0.0
  py = 0.0
  pz = 0.0
  m.size.times do |i|
    px += vx[i] * m[i]
    py += vy[i] * m[i]
    pz += vz[i] * m[i]
  end
  vx[0] = -px / solar_mass
  vy[0] = -py / solar_mass
  vz[0] = -pz / solar_mass

  3_000_000.times { advance(0.01, x, y, z, vx, vy, vz, m) }
  (energy(x, y, z, vx, vy, vz, m) * 1e9).round_away.to_i64
end

def bench_matmul : Int64
  n = 400
  a = Array(Float64).new(n * n, 0.0)
  b = Array(Float64).new(n * n, 0.0)
  c = Array(Float64).new(n * n, 0.0)
  n.times do |i|
    n.times do |j|
      a[i * n + j] = ((i + j) % 10).to_f
      b[i * n + j] = ((i * j) % 10).to_f
    end
  end
  n.times do |i|
    n.times do |j|
      s = 0.0
      n.times do |k|
        s += a[i * n + k] * b[k * n + j]
      end
      c[i * n + j] = s
    end
  end
  total = 0.0
  c.each { |v| total += v }
  total.to_i64
end

def queens(n : Int32, all : Int64, cols : Int64, d1 : Int64, d2 : Int64) : Int64
  return 1_i64 if cols == all
  count = 0_i64
  avail = all & ~(cols | d1 | d2)
  while avail != 0
    bit = avail & -avail
    avail ^= bit
    count += queens(n, all, cols | bit, ((d1 | bit) << 1) & all, (d2 | bit) >> 1)
  end
  count
end

def bench_nqueens : Int64
  n = 14
  queens(n, (1_i64 << n) - 1, 0_i64, 0_i64, 0_i64)
end

def bench_lcs : Int64
  n = 8000
  a = Bytes.new(n)
  b = Bytes.new(n)
  x = 12345_u64
  n.times do |i|
    x = x &* LCG_A &+ LCG_C
    a[i] = 65_u8 + ((x >> 33) & 3).to_u8
  end
  n.times do |i|
    x = x &* LCG_A &+ LCG_C
    b[i] = 65_u8 + ((x >> 33) & 3).to_u8
  end
  prev = Array(Int32).new(n + 1, 0)
  cur = Array(Int32).new(n + 1, 0)
  (1..n).each do |i|
    ai = a[i - 1]
    (1..n).each do |j|
      if ai == b[j - 1]
        cur[j] = prev[j - 1] + 1
      elsif prev[j] >= cur[j - 1]
        cur[j] = prev[j]
      else
        cur[j] = cur[j - 1]
      end
    end
    prev, cur = cur, prev
  end
  prev[n].to_i64
end

# ---------------------------------------------------------------- data structures

class Node
  def initialize(@l : Node? = nil, @r : Node? = nil)
  end

  def check : Int64
    if (l = @l) && (r = @r)
      1_i64 + l.check + r.check
    else
      1_i64
    end
  end
end

def bottom_up(d : Int32) : Node
  if d <= 0
    Node.new
  else
    Node.new(bottom_up(d - 1), bottom_up(d - 1))
  end
end

def bench_binarytrees : Int64
  max_depth = 17
  total = bottom_up(max_depth + 1).check
  long = bottom_up(max_depth)
  d = 4
  while d <= max_depth
    iters = 1 << (max_depth - d + 4)
    c = 0_i64
    iters.times { c += bottom_up(d).check }
    total += c
    d += 4
  end
  total += long.check
  total
end

def bench_sort : Int64
  n = 2_000_000
  a = Array(Int64).new(n, 0_i64)
  x = 42_u64
  n.times do |i|
    x = x &* LCG_A &+ LCG_C
    a[i] = (x >> 33).to_i64
  end
  a.sort!
  s = a[0] + a[n // 2] + a[n - 1]
  i = 0
  while i < n
    s += a[i]
    i += 10007
  end
  s
end

def bench_hash_int : Int64
  m = Hash(Int64, Int64).new
  x = 7_u64
  1_000_000_i64.times do |i|
    x = x &* LCG_A &+ LCG_C
    m[(x >> 33).to_i64 % 2_000_000] = i
  end
  sum = 0_i64
  2_000_000_i64.times do |i|
    if v = m[i]?
      sum += v
    end
  end
  i = 0_i64
  while i < 2_000_000
    m.delete(i)
    i += 2
  end
  sum * 7 + m.size
end

def bench_hash_string : Int64
  m = Hash(String, Int64).new
  x = 99_u64
  500_000.times do
    x = x &* LCG_A &+ LCG_C
    id = (x >> 33) % 50_000
    key = "word#{id}"
    m[key] = m.fetch(key, 0_i64) + 1
  end
  maxc = 0_i64
  m.each_value { |c| maxc = c if c > maxc }
  m.size.to_i64 * 1000 + maxc
end

# ---------------------------------------------------------------- strings / regex

def bench_string_builder : Int64
  s = String.build do |io|
    2_000_000.times do |i|
      io << "item" << i << ','
    end
  end
  count7 = 0_i64
  s.each_byte { |byte| count7 += 1 if byte == 55_u8 }
  pieces = 0_i64
  piece_len = 0_i64
  s.split(',').each do |p|
    if p.bytesize > 0
      pieces += 1
      piece_len += p.bytesize
    end
  end
  s.bytesize.to_i64 + count7 * 3 + pieces * 7 + piece_len
end

RE_EMAIL  = /[a-z]+[0-9]+@[a-z]+[0-9]+\.example\.com/
RE_STATUS = /status=([0-9]{3})/

class Address
  include JSON::Serializable
  property city : String
  property zip : String
end

class Person
  include JSON::Serializable
  property id : Int64
  property name : String
  property email : String
  property active : Bool
  property score : Float64
  property tags : Array(String)
  property address : Address
end

module Data
  @@log_text : String? = nil
  @@json_text : String? = nil
  @@people : Array(Person)? = nil
  @@sha_buf : Bytes? = nil

  def self.log_text : String
    @@log_text ||= begin
      statuses = {200, 404, 500, 301}
      String.build do |io|
        200_000.times do |i|
          st = statuses[i % 4]
          if i % 10 == 0
            io << "noise line " << i << " status=" << st << '\n'
          else
            io << "user" << i << "@host" << (i % 100) << ".example.com visited /path/" << i << " status=" << st << '\n'
          end
        end
      end
    end
  end

  def self.json_text : String
    @@json_text ||= String.build do |io|
      io << '['
      50_000.times do |i|
        io << ',' if i > 0
        cents = (i * 37 % 10000) | 1
        active = i % 3 == 0 ? "false" : "true"
        io << %({"id":) << i << %(,"name":"user) << i << %(","email":"user) << i << %(@example.com","active":) << active
        io << %(,"score":) << (cents // 100) << '.' << (cents % 100).to_s.rjust(2, '0')
        io << %(,"tags":["t) << (i % 7) << %(","t) << (i % 11) << %(","t) << (i % 13) << %("])
        io << %(,"address":{"city":"City) << (i % 100) << %(","zip":") << (10000 + i % 90000) << %("}})
      end
      io << ']'
    end
  end

  def self.people : Array(Person)
    @@people ||= Array(Person).from_json(json_text)
  end

  def self.sha_buf : Bytes
    @@sha_buf ||= begin
      buf = Bytes.new(32 * 1024 * 1024)
      x = 2024_u64
      buf.size.times do |i|
        x = x &* LCG_A &+ LCG_C
        buf[i] = (x >> 56).to_u8
      end
      buf
    end
  end
end

def bench_regex : Int64
  text = Data.log_text
  emails = text.scan(RE_EMAIL).size.to_i64
  status_sum = 0_i64
  text.scan(RE_STATUS) { |m| status_sum += m[1].to_i }
  emails * 1_000_003 + status_sum
end

# ---------------------------------------------------------------- JSON

def bench_json_dynamic : Int64
  data = JSON.parse(Data.json_text).as_a
  sum_id = 0_i64
  active = 0_i64
  tags = 0_i64
  city = 0_i64
  score = 0.0
  data.each do |e|
    o = e.as_h
    sum_id += o["id"].as_i64
    active += 1 if o["active"].as_bool
    tags += o["tags"].as_a.size
    score += o["score"].as_f
    city += o["address"].as_h["city"].as_s.size
  end
  sum_id + active * 3 + tags * 7 + city * 11 + (score * 100).round_away.to_i64
end

def bench_json_typed : Int64
  ps = Array(Person).from_json(Data.json_text)
  sum_id = 0_i64
  active = 0_i64
  tags = 0_i64
  city = 0_i64
  score = 0.0
  ps.each do |p|
    sum_id += p.id
    active += 1 if p.active
    tags += p.tags.size
    score += p.score
    city += p.address.city.size
  end
  sum_id + active * 3 + tags * 7 + city * 11 + (score * 100).round_away.to_i64
end

def bench_json_serialize : Int64
  Data.people.to_json.bytesize.to_i64
end

# ---------------------------------------------------------------- crypto

def bench_sha256 : Int64
  buf = Data.sha_buf
  acc = 0_i64
  4.times do |r|
    buf[0] = r.to_u8
    d = Digest::SHA256.digest(buf)
    acc = (acc * 31 + d[0].to_i64 * 65536 + d[1].to_i64 * 256 + d[2].to_i64) % 1_000_000_007
  end
  acc
end

# ---------------------------------------------------------------- concurrency

def bench_chan_pingpong : Int64
  a = Channel(Int64).new
  b = Channel(Int64).new
  spawn do
    while v = a.receive?
      b.send(v + 1)
    end
    b.close
  end
  s = 0_i64
  300_000_i64.times do |i|
    a.send(i)
    s += b.receive
  end
  a.close
  s
end

def bench_spawn_sum : Int64
  res = Channel(Int64).new(1000)
  sum = 0_i64
  100.times do |batch|
    1000.times do |k|
      i = (batch * 1000 + k).to_i64
      spawn { res.send(i * 2) }
    end
    1000.times { sum += res.receive }
  end
  sum
end

# ---------------------------------------------------------------- driver

def run(name : String) : Int64
  case name
  when "fib"            then bench_fib
  when "collatz"        then bench_collatz
  when "sieve"          then bench_sieve
  when "mandelbrot"     then bench_mandelbrot
  when "nbody"          then bench_nbody
  when "matmul"         then bench_matmul
  when "nqueens"        then bench_nqueens
  when "lcs"            then bench_lcs
  when "binarytrees"    then bench_binarytrees
  when "sort"           then bench_sort
  when "hash_int"       then bench_hash_int
  when "hash_string"    then bench_hash_string
  when "string_builder" then bench_string_builder
  when "regex"          then bench_regex
  when "json_dynamic"   then bench_json_dynamic
  when "json_typed"     then bench_json_typed
  when "json_serialize" then bench_json_serialize
  when "sha256"         then bench_sha256
  when "chan_pingpong"  then bench_chan_pingpong
  when "spawn_sum"      then bench_spawn_sum
  else
    STDERR.puts "unknown benchmark: #{name}"
    exit 2
  end
end

if ARGV.size < 2
  STDERR.puts "usage: bench <name> <reps>"
  exit 2
end

name = ARGV[0]
reps = ARGV[1].to_i

expected = run(name) # warm-up run, also fixes the expected checksum
stable = true
times = [] of String
reps.times do
  t0 = Time.monotonic
  r = run(name)
  el = (Time.monotonic - t0).total_milliseconds
  stable = false if r != expected
  times << el.round(3).to_s
end
puts %({"bench":"#{name}","reps":#{reps},"times_ms":[#{times.join(",")}],"checksum":#{expected},"stable":#{stable}})
