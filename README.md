Kataribe
========

Nginx/Apache/Varnishncsa Log Profiler

## Prerequisites

### Apache

Add %D to [LogFormat](http://httpd.apache.org/docs/current/mod/mod_log_config.html#logformat).
```
LogFormat "%h %l %u %t \"%r\" %>s %b \"%{Referer}i\" \"%{User-agent}i\" %D" with_time
CustomLog logs/access_log with_time
```

### Nginx

Add $request\_time to [log\_format](http://nginx.org/en/docs/http/ngx_http_log_module.html#log_format).
```
log_format with_time '$remote_addr - $remote_user [$time_local] '
                     '"$request" $status $body_bytes_sent '
                     '"$http_referer" "$http_user_agent" $request_time';
access_log /var/log/nginx/access.log with_time;
```

### H2O

Add format to [access-log directive](https://h2o.examp1e.net/configure/access_log_directives.html)
```
access-log:
  path: /var/log/h2o/access.log
  format: "%h %l %u %t \"%r\" %s %b \"%{Referer}i\" \"%{User-agent}i\" %{duration}x"
```

### Varnishncsa

Add %D to [varnishncsa -F option](https://www.varnish-cache.org/docs/trunk/reference/varnishncsa.html).
```
varnishncsa -a -w $logfile -D -P $pidfile -F '%h %l %u %t "%r" %s %b "%{Referer}i" "%{User-agent}i" %D'
```

### Rack

Add Rack::CommonLogger to config.ru.
```
logger = Logger.new("/tmp/app.log")
use Rack::CommonLogger, logger
```

## Usage

- Download [release file](https://github.com/matsuu/kataribe/releases)
- Generate kataribe.toml
```
# kataribe -generate
```
- Edit kataribe.toml
- Pass access log to kataribe by stdin
```
# cat /var/log/nginx/access.log | ./kataribe [-f kataribe.toml]
```

## Example

```
Sort By Count
Count   Total      Mean    Stddev    Min    P50    P90    P95    P99    Max    2xx   3xx  4xx  5xx  Request
17238   0.000  0.000000  0.000000  0.000  0.000  0.000  0.000  0.000  0.000  17238     0    0    0  GET /stylesheets/*
 5746   0.000  0.000000  0.000000  0.000  0.000  0.000  0.000  0.000  0.000   5746     0    0    0  GET /images/*
 5198  12.449  0.002395  0.002292  0.001  0.002  0.003  0.005  0.009  0.069   5198     0    0    0  GET / HTTP/1.1
 2873  22.753  0.007920  0.007529  0.004  0.006  0.011  0.015  0.035  0.193      0  2873    0    0  POST /login HTTP/1.1
  548   2.851  0.005203  0.004015  0.003  0.004  0.007  0.009  0.021  0.066    548     0    0    0  GET /mypage HTTP/1.1
    1   0.303  0.303000  0.000000  0.303  0.303  0.303  0.303  0.303  0.303      0     0    0    1  GET /report HTTP/1.1

Sort By Total
Count   Total      Mean    Stddev    Min    P50    P90    P95    P99    Max    2xx   3xx  4xx  5xx  Request
 2873  22.753  0.007920  0.007529  0.004  0.006  0.011  0.015  0.035  0.193      0  2873    0    0  POST /login HTTP/1.1
 5198  12.449  0.002395  0.002292  0.001  0.002  0.003  0.005  0.009  0.069   5198     0    0    0  GET / HTTP/1.1
  548   2.851  0.005203  0.004015  0.003  0.004  0.007  0.009  0.021  0.066    548     0    0    0  GET /mypage HTTP/1.1
    1   0.303  0.303000  0.000000  0.303  0.303  0.303  0.303  0.303  0.303      0     0    0    1  GET /report HTTP/1.1
17238   0.000  0.000000  0.000000  0.000  0.000  0.000  0.000  0.000  0.000  17238     0    0    0  GET /stylesheets/*
 5746   0.000  0.000000  0.000000  0.000  0.000  0.000  0.000  0.000  0.000   5746     0    0    0  GET /images/*

Sort By Mean
Count   Total      Mean    Stddev    Min    P50    P90    P95    P99    Max    2xx   3xx  4xx  5xx  Request
    1   0.303  0.303000  0.000000  0.303  0.303  0.303  0.303  0.303  0.303      0     0    0    1  GET /report HTTP/1.1
 2873  22.753  0.007920  0.007529  0.004  0.006  0.011  0.015  0.035  0.193      0  2873    0    0  POST /login HTTP/1.1
  548   2.851  0.005203  0.004015  0.003  0.004  0.007  0.009  0.021  0.066    548     0    0    0  GET /mypage HTTP/1.1
 5198  12.449  0.002395  0.002292  0.001  0.002  0.003  0.005  0.009  0.069   5198     0    0    0  GET / HTTP/1.1
17238   0.000  0.000000  0.000000  0.000  0.000  0.000  0.000  0.000  0.000  17238     0    0    0  GET /stylesheets/*
 5746   0.000  0.000000  0.000000  0.000  0.000  0.000  0.000  0.000  0.000   5746     0    0    0  GET /images/*

Sort By Standard Deviation
Count   Total      Mean    Stddev    Min    P50    P90    P95    P99    Max    2xx   3xx  4xx  5xx  Request
 2873  22.753  0.007920  0.007529  0.004  0.006  0.011  0.015  0.035  0.193      0  2873    0    0  POST /login HTTP/1.1
  548   2.851  0.005203  0.004015  0.003  0.004  0.007  0.009  0.021  0.066    548     0    0    0  GET /mypage HTTP/1.1
 5198  12.449  0.002395  0.002292  0.001  0.002  0.003  0.005  0.009  0.069   5198     0    0    0  GET / HTTP/1.1
    1   0.303  0.303000  0.000000  0.303  0.303  0.303  0.303  0.303  0.303      0     0    0    1  GET /report HTTP/1.1
17238   0.000  0.000000  0.000000  0.000  0.000  0.000  0.000  0.000  0.000  17238     0    0    0  GET /stylesheets/*
 5746   0.000  0.000000  0.000000  0.000  0.000  0.000  0.000  0.000  0.000   5746     0    0    0  GET /images/*

Sort By Maximum(100 Percentile)
Count   Total      Mean    Stddev    Min    P50    P90    P95    P99    Max    2xx   3xx  4xx  5xx  Request
    1   0.303  0.303000  0.000000  0.303  0.303  0.303  0.303  0.303  0.303      0     0    0    1  GET /report HTTP/1.1
 2873  22.753  0.007920  0.007529  0.004  0.006  0.011  0.015  0.035  0.193      0  2873    0    0  POST /login HTTP/1.1
 5198  12.449  0.002395  0.002292  0.001  0.002  0.003  0.005  0.009  0.069   5198     0    0    0  GET / HTTP/1.1
  548   2.851  0.005203  0.004015  0.003  0.004  0.007  0.009  0.021  0.066    548     0    0    0  GET /mypage HTTP/1.1
17238   0.000  0.000000  0.000000  0.000  0.000  0.000  0.000  0.000  0.000  17238     0    0    0  GET /stylesheets/*
 5746   0.000  0.000000  0.000000  0.000  0.000  0.000  0.000  0.000  0.000   5746     0    0    0  GET /images/*

TOP 37 Slow Requests
 1  0.303  GET /report HTTP/1.1
 2  0.193  POST /login HTTP/1.1
 3  0.149  POST /login HTTP/1.1
 4  0.108  POST /login HTTP/1.1
 5  0.105  POST /login HTTP/1.1
 6  0.101  POST /login HTTP/1.1
 7  0.084  POST /login HTTP/1.1
 8  0.080  POST /login HTTP/1.1
 9  0.080  POST /login HTTP/1.1
10  0.069  GET / HTTP/1.1
11  0.066  GET /mypage HTTP/1.1
12  0.063  POST /login HTTP/1.1
13  0.063  POST /login HTTP/1.1
14  0.057  POST /login HTTP/1.1
15  0.056  POST /login HTTP/1.1
16  0.056  GET / HTTP/1.1
17  0.054  POST /login HTTP/1.1
18  0.054  POST /login HTTP/1.1
19  0.048  POST /login HTTP/1.1
20  0.046  GET /mypage HTTP/1.1
21  0.045  POST /login HTTP/1.1
22  0.045  POST /login HTTP/1.1
23  0.044  POST /login HTTP/1.1
24  0.042  POST /login HTTP/1.1
25  0.041  GET / HTTP/1.1
26  0.040  POST /login HTTP/1.1
27  0.039  GET / HTTP/1.1
28  0.038  POST /login HTTP/1.1
29  0.038  GET / HTTP/1.1
30  0.037  POST /login HTTP/1.1
31  0.037  POST /login HTTP/1.1
32  0.036  GET / HTTP/1.1
33  0.036  POST /login HTTP/1.1
34  0.036  GET / HTTP/1.1
35  0.036  POST /login HTTP/1.1
36  0.035  POST /login HTTP/1.1
37  0.035  POST /login HTTP/1.1
```

## License

Apache-2.0

## Author

[matsuu](https://github.com/matsuu)

