# Benchmarks

Benchmarked with `wrk` and `traefik/whoami`'s `/bench` endpoint

## Remote benchmark

- Direct connection

  ```shell
  root@yusing-pc:~# wrk -t 10 -c 200 -d 10s -H "Host: bench.6uo.me" --latency http://10.0.100.3:8003/bench
  Running 10s test @ http://10.0.100.3:8003/bench
    10 threads and 200 connections
    Thread Stats   Avg      Stdev     Max   +/- Stdev
      Latency    94.75ms  199.92ms   1.68s    91.27%
      Req/Sec     4.24k     1.79k   18.79k    72.13%
    Latency Distribution
      50%    1.14ms
      75%  120.23ms
      90%  245.63ms
      99%    1.03s
    423444 requests in 10.10s, 50.88MB read
    Socket errors: connect 0, read 0, write 0, timeout 29
  Requests/sec:  41926.32
  Transfer/sec:      5.04MB
  ```

- With reverse proxy

  ```shell
  root@yusing-pc:~# wrk -t 10 -c 200 -d 10s -H "Host: bench.6uo.me" --latency http://10.0.1.7/bench
  Running 10s test @ http://10.0.1.7/bench
    10 threads and 200 connections
    Thread Stats   Avg      Stdev     Max   +/- Stdev
      Latency    79.35ms  169.79ms   1.69s    92.55%
      Req/Sec     4.27k     1.90k   19.61k    75.81%
    Latency Distribution
      50%    1.12ms
      75%  105.66ms
      90%  200.22ms
      99%  814.59ms
    409836 requests in 10.10s, 49.25MB read
    Socket errors: connect 0, read 0, write 0, timeout 18
  Requests/sec:  40581.61
  Transfer/sec:      4.88MB
  ```

## Local benchmark (client running wrk and `go-proxy` server are under same proxmox host but different LXCs)

- Direct connection

  ```shell
  root@http-benchmark-client:~# wrk -t 10 -c 200 -d 10s --latency http://10.0.100.1/bench
  Running 10s test @ http://10.0.100.1/bench
    10 threads and 200 connections
    Thread Stats   Avg      Stdev     Max   +/- Stdev
      Latency   434.08us  539.35us   8.76ms   85.28%
      Req/Sec    67.71k     6.31k   87.21k    71.20%
    Latency Distribution
      50%  153.00us
      75%  646.00us
      90%    1.18ms
      99%    2.38ms
    6739591 requests in 10.01s, 809.85MB read
  Requests/sec: 673608.15
  Transfer/sec:     80.94MB
  ```

- With `go-proxy` reverse proxy

  ```shell
  root@http-benchmark-client:~# wrk -t 10 -c 200 -d 10s -H "Host: bench.6uo.me" --latency http://10.0.1.7/bench
  Running 10s test @ http://10.0.1.7/bench
    10 threads and 200 connections
    Thread Stats   Avg      Stdev     Max   +/- Stdev
      Latency     1.23ms    0.96ms  11.43ms   72.09%
      Req/Sec    17.48k     1.76k   21.48k    70.20%
    Latency Distribution
      50%    0.98ms
      75%    1.76ms
      90%    2.54ms
      99%    4.24ms
    1739079 requests in 10.01s, 208.97MB read
  Requests/sec: 173779.44
  Transfer/sec:     20.88MB
  ```

- With `traefik-v3`

  ```shell
  root@traefik-benchmark:~# wrk -t10 -c200 -d10s -H "Host: benchmark.whoami" --latency http://127.0.0.1:8000/bench
  Running 10s test @ http://127.0.0.1:8000/bench
    10 threads and 200 connections
    Thread Stats   Avg      Stdev     Max   +/- Stdev
      Latency     2.81ms   10.36ms 180.26ms   98.57%
      Req/Sec    11.35k     1.74k   13.76k    85.54%
    Latency Distribution
      50%    1.59ms
      75%    2.27ms
      90%    3.17ms
      99%   37.91ms
    1125723 requests in 10.01s, 109.50MB read
  Requests/sec: 112499.59
  Transfer/sec:     10.94MB
  ```