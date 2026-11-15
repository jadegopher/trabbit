# Starting tRabbit

Run `trabbit` to open the login form, or supply a RabbitMQ management HTTP(S)
endpoint to connect automatically:

```sh
trabbit --url 'http://guest:guest@localhost:15672'
trabbit --url 'http://localhost:15672' --user guest --pass guest
```

Use the management endpoint, typically port 15672, rather than an AMQP URL.
Percent-encode special characters in credentials embedded in the URL. Separate
`--user` and `--pass` flags override their corresponding embedded values; an
explicit empty `--pass=''` clears the embedded password. A username is required;
the password may be empty.

Failed connections leave the supplied values in the login form for correction
and retry. Run `trabbit --help` for all flags.

Logs are written to `debug.log` in the current working directory, replacing the
previous run. Normal runs log warnings and errors. Set the minimum severity with
`--log-level` (default: `warn`):

```sh
trabbit --log-level debug
trabbit --log-level error --url 'http://localhost:15672' --user guest --pass guest
```

Accepted levels are `debug`, `info`, `warn` and `error`. Debug adds HTTP method,
resource path, status and duration, plus screen and dialog lifecycle events.
Canceled requests log at debug level. Request bodies, authorization headers and
query values are omitted.
