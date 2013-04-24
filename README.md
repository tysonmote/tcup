tcup
====

tcup is an HTTPS server that forwards data to a UDP address. tcup can be
configured to expect a `X-Token` header for simple authentication.

    Usage: ./tcup [flags]
      -help=false: Print usage info
      -cert="cert.pem": Path to SSL certificate file
      -key="key.pem": Path to SSL key file
      -in="127.0.0.1:1984": TCP listening address
      -out="127.0.0.1:1984": UDP destination address
      -token="": Expected "X-Token" header for all requests
      -log=0: Stats logging interval. A value of 0 will cause no stats to be logged

tcup returns the following status codes:

* 200: Request successfully forwarded.
* 400: Empty request body.
* 401: Incorrect X-Token header given.
* 500: There was an error reading the request body or sending it to the
  destination UDP address. Check the response body for an error message.

How do I install it?
--------------------

Install [Go](http://golang.org) and then run:

    go install github.com/tysontate/tcup

Should I use it?
----------------

No.

