package main

const CONFIG_TOML = `# Top Ranking Group By Request
ranking_count = 20

# Top Slow Requests
slow_count = 37

# Show Standard Deviation column
show_stddev = true

# Show HTTP Status Code columns
show_status_code = true

# Show HTTP Response Bytes columns
show_bytes = true

# Percentiles
percentiles = [ 50.0, 90.0, 95.0, 99.0 ]

# for Nginx($request_time)
scale = 0
effective_digit = 3

# for Apache(%D) and Varnishncsa(%D)
#scale = -6
#effective_digit = 6

# for Rack(Rack::CommonLogger)
#scale = 0
#effective_digit = 4


# combined + duration
# Nginx example: '$remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" $request_time'
# Apache example: "%h %l %u %t \"%r\" %>s %b \"%{Referer}i\" \"%{User-Agent}i\" %D"
# H2O example: "%h %l %u %t \"%r\" %s %b \"%{Referer}i\" \"%{User-agent}i\" %{duration}x"
# Varnishncsa example: '%h %l %u %t "%r" %s %b "%{Referer}i" "%{User-agent}i" %D'
log_format = '^([^ ]+) ([^ ]+) ([^ ]+) \[([^\]]+)\] "((?:\\"|[^"])*)" (\d+) (\d+|-) "((?:\\"|[^"])*)" "((?:\\"|[^"])*)" ([0-9.]+)$'

request_index = 5
status_index = 6
bytes_index = 7
duration_index = 10

# The default is ltsv.
# If you choose json, please comment in.
# parser = "json"

# Rack example: use Rack::CommonLogger, Logger.new("/tmp/app.log")
#log_format = '^([^ ]+) ([^ ]+) ([^ ]+) \[([^\]]+)\] "((?:\\"|[^"])*)" (\d+) (\d+|-) ([0-9.]+)$'
#request_index = 5
#status_index = 6
#bytes_index = 7
#duration_index = 8

# You can aggregate requests by regular expression
# For overview of regexp syntax: https://golang.org/pkg/regexp/syntax/
[[bundle]]
regexp = '^(GET|HEAD) /memo/[0-9]+\b'
name = "get memo"

[[bundle]]
regexp = '^(GET|HEAD) /stylesheets/'
name = "stylesheets"

[[bundle]]
regexp = '^(GET|HEAD) /images/'
name = "images"

# You can replace the part of urls which matched to your regular expressions.
# For overview of regexp syntax: https://golang.org/pkg/regexp/syntax/
#[[replace]]
#regexp = '/[0-9]+/'
#replace = '/<num>/'
#
#[[replace]]
#regexp = '/[0-9]+\s'
#replace = '/<num> '
#
#[[replace]]
#regexp = '=[0-9]+&'
#replace = '=<num>&'
#
#[[replace]]
#regexp = '=[0-9]+\s'
#replace = '=<num> '
`
