# NATS Stats

This process collects varz, connz, routez, and subsz data from the NATS server and saves them to a file in easily parsible JSON.

## Usage

```bash
./natsstats [-V] [-create] <config.json>
```

The `-V` option will enable verbosity for debugging purposes.

A configuration file is required, but there is a convenient way to create a configuration file using the `-config` parameter:

```bash
./natsstats -create myconfig.json
```

This will create a default configuration file with all the available options.  Interrupting the process (ctrl-c) will gracefully shutdown and close the output file.

## Configuration

The configuration file format looks like this:

```text
{
  "url": "http://localhost:8222",
  "getvarz": true,
  "getconnz": false,
  "getsubsz": false,
  "getroutez": false,
  "interval": "1s",
  "outfile": "results.json",
  "prettyprint": false,
  "verbose": false
}
```

* `url`: The full url to for the monitoring endpoint of a NATS server.
* `getvars`: If true, will get the `varz` NATS server data.
* `getconnz`: If true, will get the `connz` NATS server data.
* `getsubsz`: If true, will get the `subsz` NATS server data.
* `getroutez`: If true, will get the `routez` NATS server data.
* `interval`: The poll interval, parsable as a go duration string.  For this application, it is a unsigned sequence of decimal numbers, each with optional fraction and a unit suffix, such as "300ms", "-1.5h" or "2h45m". Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".  Default is "1s".
* `outfile`:  An json output file to be generated.  Default is "results.json".  This will replace any existing output file.
* `prettyprint`:  By default, the output file will have a single JSON object per line for ease of use parsing in other applications.  Setting prettyprint to true will format the JSON to the common readable format.
* `verbose`: Enable verbosity

_Note, at least one of the `getvarz`, `getconnz`, `getsubsz`, or `getroutez` field must be set to true._

## Output

The output is as follows:

### stdout

```bash
./natsstats config.json
Monitor Url: http://localhost:8222
Poll ivl:    1s
Get varz:    true
Get subsz:   false
Get connz:   false
Get routez:  false
Output File: results.json
PrettyPrint: false
============================
```

### Output File

By default, the output file is one JSON object per line, containing the time the statistic was generated, a hostname, and the NATS server statistics that were gathered as specfied by the configuration.

```text
{"time":"2018-09-21T15:50:11.167395478-06:00","index":1,"hostname":"ColinsInstance","varz":{"server_id":"tNNOkwWcrJRI3U6CY27SyO","version":"1.3.1","proto":1,"go":"go1.10.3","host":"0.0.0.0","addr":"0.0.0.0","max_connections":65536,"ping_interval":120000000000,"ping_max":2,"http_host":"0.0.0.0","http_port":8222,"https_port":0,"auth_timeout":1,"max_control_line":4096,"cluster":{},"tls_timeout":0.5,"port":4222,"max_payload":1048576,"start":"2018-09-20T19:17:00.58637925-06:00","now":"2018-09-21T15:50:11.159961848-06:00","uptime":"8h47m25s","mem":9379840,"cores":8,"cpu":0,"connections":0,"total_connections":0,"routes":0,"remotes":0,"in_msgs":0,"out_msgs":0,"in_bytes":0,"out_bytes":0,"slow_consumers":0,"max_pending":268435456,"write_deadline":2000000000,"subscriptions":0,"http_req_stats":{"/":0,"/connz":108,"/routez":108,"/subsz":108,"/varz":173},"config_load_time":"2018-09-20T19:17:00.58637925-06:00"}}
{"time":"2018-09-21T15:50:12.177328831-06:00","index":2,"hostname":"ColinsInstance","varz":{"server_id":"tNNOkwWcrJRI3U6CY27SyO","version":"1.3.1","proto":1,"go":"go1.10.3","host":"0.0.0.0","addr":"0.0.0.0","max_connections":65536,"ping_interval":120000000000,"ping_max":2,"http_host":"0.0.0.0","http_port":8222,"https_port":0,"auth_timeout":1,"max_control_line":4096,"cluster":{},"tls_timeout":0.5,"port":4222,"max_payload":1048576,"start":"2018-09-20T19:17:00.58637925-06:00","now":"2018-09-21T15:50:12.170559501-06:00","uptime":"8h47m26s","mem":9555968,"cores":8,"cpu":0,"connections":0,"total_connections":0,"routes":0,"remotes":0,"in_msgs":0,"out_msgs":0,"in_bytes":0,"out_bytes":0,"slow_consumers":0,"max_pending":268435456,"write_deadline":2000000000,"subscriptions":0,"http_req_stats":{"/":0,"/connz":108,"/routez":108,"/subsz":108,"/varz":174},"config_load_time":"2018-09-20T19:17:00.58637925-06:00"}}
{"time":"2018-09-21T15:50:13.186055652-06:00","index":3,"hostname":"ColinsInstance","varz":{"server_id":"tNNOkwWcrJRI3U6CY27SyO","version":"1.3.1","proto":1,"go":"go1.10.3","host":"0.0.0.0","addr":"0.0.0.0","max_connections":65536,"ping_interval":120000000000,"ping_max":2,"http_host":"0.0.0.0","http_port":8222,"https_port":0,"auth_timeout":1,"max_control_line":4096,"cluster":{},"tls_timeout":0.5,"port":4222,"max_payload":1048576,"start":"2018-09-20T19:17:00.58637925-06:00","now":"2018-09-21T15:50:13.17935352-06:00","uptime":"8h47m27s","mem":9707520,"cores":8,"cpu":0,"connections":0,"total_connections":0,"routes":0,"remotes":0,"in_msgs":0,"out_msgs":0,"in_bytes":0,"out_bytes":0,"slow_consumers":0,"max_pending":268435456,"write_deadline":2000000000,"subscriptions":0,"http_req_stats":{"/":0,"/connz":108,"/routez":108,"/subsz":108,"/varz":175},"config_load_time":"2018-09-20T19:17:00.58637925-06:00"}}
```

This is somewhat custom with a newline between each JSON object.  If you need a fully parsable standard JSON file, enabling PrettyPrint will print the JSON in a fully parsable, and easy to read, form.