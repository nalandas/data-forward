# data-forwarder

## What is this?

A tool to collect the data locally in preparation for processing elsewhere!
The source of the data include: MySQL, csv, PostgreSQL and etc.

### Transport Problems

Few log transport mechanisms provide security, low latency, and reliability.

The lumberjack protocol used by this project exists to provide a network
protocol for transmission that is secure, low latency, low resource usage, and
reliable.

## Configuring

data-forwarder is configured with a json file you specify with the -config flag:

`data-forwarder -config yourstuff.json`

Here's a sample, with comments in-line to describe the settings. Comments are
invalid in JSON, but data-forwarder will strip them out for you if they're
the only thing on the line:

    {
      # The network section covers network configuration :)
      "network": {
        # A list of downstream servers listening for our messages.
        # data-forwarder will pick one at random and only switch if
        # the selected one appears to be dead or unresponsive
        "servers": [ "localhost:5043" ],

        # The path to your client ssl certificate (optional)
        "ssl certificate": "./data-forwarder.crt",
        # The path to your client ssl key (optional)
        "ssl key": "./data-forwarder.key",

        # The path to your trusted ssl CA file. This is used
        # to authenticate your downstream server.
        "ssl ca": "./data-forwarder.crt",

        # Network timeout in seconds. This is most important for
        # data-forwarder determining whether to stop waiting for an
        # acknowledgement from the downstream server. If an timeout is reached,
        # data-forwarder will assume the connection or server is bad and
        # will connect to a server chosen at random from the servers list.
        "timeout": 15
      },

      # The list of files configurations
      "files": [
        # An array of hashes. Each hash tells what paths to watch and
        # what fields to annotate on events from those paths.
        {
          "paths": [
            # single paths are fine
            "/var/log/messages",
            # globs are fine too, they will be periodically evaluated
            # to see if any new files match the wildcard.
            "/var/log/*.log"
          ],

          # A dictionary of fields to annotate on each event.
          "fields": { "type": "syslog" }
        }, {
          # A path of "-" means stdin.
          "paths": [ "-" ],
          "fields": { "type": "stdin" }
        }, {
          "paths": [
            "/var/log/apache/httpd-*.log"
          ],
          "fields": { "type": "apache" }
        }
      ]
    }

You can also read an entire directory of JSON configs by specifying a directory instead of a file with the `-config` option.

# IMPORTANT TLS/SSL CERTIFICATE NOTES

This program will reject SSL/TLS certificates which have a subject which does not match the `servers` value, for any given connection. For example, if you have `"servers": [ "foobar:12345" ]` then the 'foobar' server MUST use a certificate with subject or subject-alternative that includes `CN=foobar`. Wildcards are supported also for things like `CN=*.example.com`. If you use an IP address, such as `"servers": [ "1.2.3.4:12345" ]`, your ssl certificate MUST use an IP SAN with value "1.2.3.4". If you do not, the TLS handshake will FAIL and the lumberjack connection will close due to trust problems.

Creating a correct SSL/TLS infrastructure is outside the scope of this document. 

As a very poor example (largely due unpredictability in your system's defaults for openssl), you can try the following command as an example for creating a self-signed certificate/key pair for use with a server named "data.example.com":

```
openssl req -x509  -batch -nodes -newkey rsa:2048 -keyout lumberjack.key -out lumberjack.crt -subj /CN=data.example.com
```

The above example will create an SSL cert for the host 'data.example.com'. You cannot use `/CN=1.2.3.4` to create an SSL certificate for an IP address. In order to do a certificate with an IP address, you must create a certificate with an "IP Subject Alternative" or often called "IP SAN". Creating a certificate with an IP SAN is difficult and annoying, so I highly recommend you use hostnames only. If you have no DNS available to you, it is still often easier to set hostnames in /etc/hosts than it is to create a certificate with an IP SAN.

data-forwarder needs the `.crt` file, and data will need both `.key` and `.crt` files.

Again, creating a correct SSL/TLS certificate authority or generally doing certificate management is outside the scope of this document. 

If you see an error like this:

```
x509: cannot validate certificate for 1.2.3.4 because it doesn't contain any IP SANs
```

It means you are telling data-forwarder to connect to a host by IP address,
and therefore you must include an IP SAN in your certificate. Generating an SSL
certificate with an IP SAN is quite annoying, so I *HIGHLY* recommend you use
dns names and set the CN in your cert to your dns name.

### Goals

* Minimize resource usage where possible (CPU, memory, network).
* Secure transmission of data.
* Configurable event data.
* Easy to deploy with minimal moving parts.
* Simple inputs only:
  * Follows files and respects rename/truncation conditions.
  * Accepts `STDIN`, useful for things like `varnishlog | data-forwarder...`.

## Building it

1. Install [go](http://golang.org/doc/install)

2. Compile data-forwarder

Note: Do not use gccgo for this project. If you don't know what that means,
you're probably OK to ignore this.

        git clone git://github.com/esse-io/data-forwarder.git
        cd data-forwarder
        go build -o data-forwarder

gccgo note: Using gccgo is not recommended because it produces a binary with a
runtime dependency on libgo. With the normal go compiler, this dependency
doesn't exist and, as a result, makes it easier to deploy. You can check if you
are using gccgo by running `go version` and if it outputs something like `go
version xgcc`, you're probably not using gccgo, and I recommend you don't.
You can also check the resulting binary by doing `ldd ./data-forwarder` and
seeing if `libgo` appears in the output; if it appears, then you are using gccgo,
and I recommend you don't.

## Packaging it (optional)

You can make native packages of data-forwarder.

To do this, a recent version of Ruby is required. At least version 2.0.0 or
newer. If you are using your OS distribution's version of Ruby, especially on
Red Hat- or Debian-derived systems (Ubuntu, CentOS, etc), you will need to install
ruby and whatever the "ruby development" package is called for your system.
On Red Hat systems, you probably want `yum install ruby-devel`. On Debian systems,
you probably want `apt-get install ruby-dev`.

Prerequisite steps to prepare ruby to build your packages are:

```
gem install bundler
bundle install
```

The `bundle install` will install any Ruby library dependencies that are used
in building packages.

Now build an rpm:

        make rpm

Or:

        make deb

## Installing it (via packages only)

If you don't use rpm or deb make targets as above, you can skip this section.

Packages install to `/opt/data-forwarder`.

There are no run-time dependencies.

## Running it

Generally:

    data-forwarder -config data-forwarder.conf

See `data-forwarder -help` for all the flags. The `-config` option is required and data-forwrder will not run without it.

The config file is documented further up in this file.

And also note that data-forwarder runs quietly when all is a-ok. If you want informational feedback, use the `verbose` flag to enable log Emits to stdout.

Fatal errors are always sent to stderr regardless of the `-quiet` command-line option and process exits with a non-zero status.

### Key points

* You'll need an SSL CA to verify the server (host) with.
* You can specify custom fields for each set of paths in the config file. Any
  number of these may be specified. I use them to set fields like `type` and
  other custom attributes relevant to each log.

### Generating an ssl certificate

Logstash supports all certificates, including self-signed certificates. To generate a certificate, you can run the following command:

    $ openssl req -x509 -batch -nodes -newkey rsa:2048 -keyout data-forwarder.key -out data-forwarder.crt -days 365

This will generate a key at `data-forwarder.key` and the 1-year valid certificate at `data-forwarder.crt`. Both the server that is running data-forwarder as well as the data instances receiving logs will require these files on disk to verify the authenticity of messages. 

Recommended file locations:

- certificates: `/etc/pki/tls/certs/data-forwarder/`
- keys: `/etc/pki/tls/private/data-forwarder/`

## Use with data

In data, you'll want to use the [lumberjack](http://data.net/docs/latest/inputs/lumberjack) input, something like:

    input {
      lumberjack {
        # The port to listen on
        port => 12345

        # The paths to your ssl cert and key
        ssl_certificate => "path/to/ssl.crt"
        ssl_key => "path/to/ssl.key"

        # Set this to whatever you want.
        type => "somelogs"
      }
    }

## Implementation details

Below is valid as of 2012/09/19

### Minimize resource usage

* Sets small resource limits (memory, open files) on start up based on the
  number of files being watched.
* CPU: sleeps when there is nothing to do.
* Network/CPU: sleeps if there is a network failure.
* Network: uses zlib for compression.

### Secure transmission

* Uses OpenSSL to verify the server certificates (so you know who you
  are sending to).
* Uses OpenSSL to transport logs.

### Configurable event data

* The protocol supports sending a `string:string` map.

## License

See LICENSE file.

