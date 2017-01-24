/*
<!--
Copyright (c) 2016 Christoph Berger. Some rights reserved.
Use of this text is governed by a Creative Commons Attribution Non-Commercial
Share-Alike License that can be found in the LICENSE.txt file.

The source code contained in this file may import third-party source code
whose licenses are provided in the respective license files.
-->

<!--
NOTE: The comments in this file are NOT godoc compliant. This is not an oversight.

Comments and code in this file are used for describing and explaining a particular topic to the reader. While this file is a syntactically valid Go source file, its main purpose is to get converted into a blog article. The comments were created for learning and not for code documentation.
-->

+++
title = "TCP/IP Networking"
description = "How to communicate at TCP/IP level in Go"
author = "Christoph Berger"
email = "chris@appliedgo.net"
date = "2017-01-24"
publishdate = "2017-01-24"
draft = "true"
domains = ["Distributed Computing"]
tags = ["network", "tcp"]
categories = ["Tutorial"]
+++

Connecting two processes at TCP/IP level might seem scary at first, but in Go
it is easier as one might think.

<!--more-->

While preparing another blog post, I realized that the networking part of the
code was quickly becoming larger than the part of the code that was meant to
illustrate the topic of the post. Furthermore, networking in Go at TCP/IP
level feels a little underrepresented in the Go blog space (but I might be
wrong).

## Who needs sending things at TCP/IP level?

Granted, many, if not most, scenarios, undoubtedly do better with a higher-level
network protocol that hides all the technical details beneath a fancy API.
And there are already plenty to choose from, depending on the needs:
Message queue protocols, gRPC, protobuf, FlatBuffers, RESTful Web API's,
WebSockets, and so on.

However, in some situations--especially with small projects--, any approach
you choose may look like completely oversized, not to mention the additional
package dependencies that you'd have to introduce.

Luckily, creating simple network communication with
[the standard `net` package](https://golang.org/pkg/net/)
is not as difficult as it may seem.


## Simplification #1: connections are io streams

The `net.Conn` interface implements [the `io.Reader`, `io.Writer`, and `io.Closer`
interfaces](https://golang.org/pkg/io/). Hence you can use a TCP connection
like any `io` stream.

I know what you think -- "Ok, so I can send strings or byte slices over a
TCP connection. That's nice but what about complex data types? Structs and such?"


## Simplification #2: Go knows how to encode complex types efficiently

*(Fun fact: About every other time I review this text on the rendered Web page,
I misread the title as "God knows...".
So if this just happened to you, you are not alone :-)*

When it comes to encoding structured data for sending over the net, JSON comes
readily to mind. But wait - Go's standard `encoding/gob` package provides
a way of serializing and deserializing Go data types without the need for
adding string tags to structs, dealing with JSON/Go incompatibilites, or waiting
for json.Unmarshal to laboriously parse text into binary data.

Gob encoders and decoders work directly on `io` streams - and this fits just
nicely into our simplification #1 - connections are `io` streams.

Let's put this all together in a small sample app.

## Goal

The sample app shall do two things:

1. Send and receive a simple message as a string
2. Send and receive a `struct` via GOB

The first part--sending simple strings--shall demonstrate how easy it is
to send data over a TCP/IP network without any higher-level protocols.

The second part goes a step further and sends a complete struct over the
network, with strings, slices, maps, and even a recursive pointer to the
struct itself:

HYPE[Sending a struct as GOB](gob.html)

The `gob` package makes this as easy as pie.

## Basic ingredients for sending string data over TCP

### On the sending side

Sending strings requires three simple steps.

1. Open a connection to the receiving process
2. Write the string
3. Close the connection

The `net` package provides a couple of methods for this.

`ResolveTCPAddr()` takes a string representing a TCP address (like, for example,
`localhost:80`, `127.0.0.1:80`, or `[::1]:80`, which all represent port #80 on
the local machine) and returns a `net.TCPAddr` (or an error if the string
cannot be resolved to a valid TCP address).

`DialTCP()` takes a `net.TCPAddr` and connects to this address. It returns
the open connection as a `net.TCPConn` object (or an error if the connection
attempt fails).

If we don't need much fine-grained control over the Dial settings, we can use
`net.Dial()` instead. This function takes an address string directly and
returns a general `net.Conn` object. This is sufficient for our test case;
however, if you need functionality that is only available on TCP connections,
you have to use the "TCP" variants (`DialTCP`, `TCPConn`, `TCPAddr`, etc).

After successful dialing, we can treat the new connection like any other
input/output stream, as mentioned above. We can even wrap the connection into
a `bufio.ReadWriter` and benefit from the various `ReadWriter` methods like
`ReadString()`, `ReadBytes`, `WriteString`, etc.

** Remember that buffered Writers need to call `Flush()` after writing,
so that all data is forwarded to the underlying network connection.**

Finally, each connection object has a `Close()` method to conclude the
communication.


### Fine tuning

A couple of tuning options are also available. Some examples:

The `Dialer` interface provides these options (among others):

* `DeadLine` and `Timeout` options for timing out an unsuccessful dial;
* a `KeepAlive` option for managing the life span of the connection

The `Conn` interface also has deadline settings; either for the connection as
a whole (`SetDeadLine()`) or specific to read or write calls (`SetReadDeadLine()`
and `SetWriteDeadLine()`).

Note that the deadlines are fixed points in (wallclock) time. Unlike timeouts,
they don't reset after a new activity. Each activity on the connection must
therefore set a new deadline.

The sample code below uses no deadlines, as it is simple enough so that we can
easily see when things get stuck. `Ctrl-C` is our manual "deadline trigger
tool".


### On the receiving side

The receiver has to follow these steps.

1. Start listening on a local port.
2. When a request comes in, spawn a goroutine to handle the request.
3. In the goroutine, read the data. Optionally, send a response.
4. Close the connection.

Listening requires a local port to listen to. Typically, the listening
application (a.k.a. "server") announces the port it listens to, or if it
provides a standard service, it uses the port associated with that service.
For example, Web servers usually listen on port 80 for HTTP requests and
on port 443 for HTTPS requests. SSH daemons listen on port 22 by default,
and a WHOIS server uses port 43.

The core parts of the `net` package for implementing the server side are:

`net.Listen()` creates a new listener on a given local network address. If
only a port ist passed, as in ":61000", then the listener listens on
all available network interfaces. This is quite handy, as a computer usually
has at least two active interfaces, the loopback interface and at least one
real network card.

A listener's `Accept()` method waits until a connection request comes in.
Then it accepts the request and returns the new connection to the caller.
`Accept()` is typically called within a loop to be able to serve multiple
connnections simultaneously. Each connection can be handled by a goroutine,
as we will see in the code.


## The code

Instead of just pushing a few bytes around, I wanted the code to demonstrate
something more useful. I want to be able to send different commands with
different data payload to the server. The server shall identify each
command and decode the command's data.

So the client in the code below sends two test commands: "STRING" and "GOB".
Each are terminated by a newline.

The STRING command includes one line of string
data, which can be handled by simple read and write methods from `bufio`.

The GOB command comes with a `struct` that contains a couple of fields,
including a slice, a map, and a even a pointer to itself. As you can see when
running the code, the `gob` package moves all this through our network
connection without any fuss.




*/

// ## Imports and globals
package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"encoding/gob"
	"flag"
)

// A struct with a mix of fields, used for the GOB example.
type complexData struct {
	N int
	S string
	M map[string]int
	P []byte
	C *complexData
}

const (
	// Port is the port number that the server listens to.
	Port = ":61000"
)

/*
## Outcoing connections

Using an outgoing connection is a snap. A `net.Conn` satisfies the io.Reader
and `io.Writer` interfaces, so we can treat a TCP connection just like any other
`Reader` or `Writer`.
*/

// Open connects to a TCP Address.
// It returns a TCP connection armed with a timeout and wrapped into a
// buffered ReadWriter.
func Open(addr string) (*bufio.ReadWriter, error) {
	// Dial the remote process.
	// Note that the local port is chosen on the fly. If the local port
	// must be a specific one, use DialTCP() instead.
	log.Println("Dial " + addr)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, errors.Wrap(err, "Dialing "+addr+" failed")
	}
	return bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)), nil
}

/*
## Incoming connections

Preparing for incoming data is a bit more involved. According to our ad-hoc
protocol, we receive the name of a command terminated by `\n`, followed by data.
The nature of the data depends on the respective command. To handle this, we
create an `Endpoint` object with the following properties:

* It allows to register one or more handler functions, where each can handle a
  particular command.
* It dispatches incoming commands to the associated handler based on the commands
  name.

*/

// HandleFunc is a function that handles an incoming command.
// It receives the open connection wrapped in a `ReadWriter` interface.
type HandleFunc func(*bufio.ReadWriter)

// Endpoint provides an endpoint to other processess
// that they can send data to.
type Endpoint struct {
	listener net.Listener
	handler  map[string]HandleFunc
}

// NewEndpoint creates a new endpoint. Too keep things simple,
// the endpoint listens on a fixed port number.
func NewEndpoint() *Endpoint {
	// Create a new Endpoint with an empty list of handler funcs.
	return &Endpoint{
		handler: map[string]HandleFunc{},
	}
}

// AddHandleFunc adds a new function for handling incoming data.
func (e *Endpoint) AddHandleFunc(name string, f HandleFunc) {
	e.handler[name] = f
}

// Listen starts listening on the endpoint port on all interfaces.
// At least one handler function must have been added
// through AddHandleFunc() before.
func (e *Endpoint) Listen() error {
	var err error
	e.listener, err = net.Listen("tcp", Port)
	if err != nil {
		return errors.Wrap(err, "Unable to listen on "+e.listener.Addr().String()+"\n")
	}
	log.Println("Listen on", e.listener.Addr().String())
	for {
		log.Println("Accept a connection request.")
		conn, err := e.listener.Accept()
		if err != nil {
			log.Println("Failed accepting a connection request:", err)
			continue
		}
		log.Println("Handle incoming messages.")
		go e.handleMessages(conn)
	}
}

// handleMessages reads the connection up to the first newline.
// Based on this string, it calls the appropriate HandleFunc.
func (e *Endpoint) handleMessages(conn net.Conn) {
	// Wrap the connection into a buffered reader for easier reading.
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	defer conn.Close()

	// Read from the connection until EOF. Expect a command name as the
	// next input. Call the handler that is registered for this command.
	for {
		log.Print("Receive command '")
		cmd, err := rw.ReadString('\n')
		switch {
		case err == io.EOF:
			log.Println("Reached EOF - close this connection.")
			return
		case err != nil:
			log.Println("\nError reading command. Got: '"+cmd+"'\n", err)
			return
		}
		// Trim the request string - ReadString does not strip any newlines.
		cmd = strings.Trim(cmd, "\n ")
		log.Println(cmd + "'")

		// Fetch the appropriate handler function from the 'handler' map and call it.
		handleCommand, ok := e.handler[cmd]
		if !ok {
			log.Println("Command '" + cmd + "' is not registered.")
			return
		}
		handleCommand(rw)
	}
}

/* Now let's create two handler functions. The easiest case is where our
ad-hoc protocol only sends string data.

The second handler receives and processes a struct that was send as GOB data.
*/

// handleStrings handles the "STRING" request.
func handleStrings(rw *bufio.ReadWriter) {
	// Receive a string.
	log.Print("Receive STRING message:")
	s, err := rw.ReadString('\n')
	if err != nil {
		log.Println("Cannot read from connection.\n", err)
	}
	s = strings.Trim(s, "\n ")
	log.Println(s)
	_, err = rw.WriteString("Thank you.\n")
	if err != nil {
		log.Println("Cannot write to connection.\n", err)
	}
	err = rw.Flush()
	if err != nil {
		log.Println("Flush failed.", err)
	}
}

// handleGob handles the "GOB" request. It decodes the received GOB data
// into a struct.
func handleGob(rw *bufio.ReadWriter) {
	log.Print("Receive GOB data:")
	var data complexData
	// Create a decoder that decodes directly into a struct variable.
	dec := gob.NewDecoder(rw)
	err := dec.Decode(&data)
	if err != nil {
		log.Println("Error decoding GOB data:", err)
		return
	}
	// Print the complexData struct and the nested one, too, to prove
	// that both travelled across the wire.
	log.Printf("Outer complexData struct: \n%#v\n", data)
	log.Printf("Inner complexData struct: \n%#v\n", data.C)
}

/*
## The client and server functions

With all this in place, we can now set up client and server functions.

The client function connects to the server and sends STRING and GOB requests.

The server starts listening for requests and triggers the appropriate handlers.
*/

// client is called if the app is called with -connect=`ip addr`.
func client(ip string) error {
	// Some test data. Note how GOB even handles maps, slices, and
	// recursive data structures without problems.
	testStruct := complexData{
		N: 23,
		S: "string data",
		M: map[string]int{"one": 1, "two": 2, "three": 3},
		P: []byte("abc"),
		C: &complexData{
			N: 256,
			S: "Recursive structs? Piece of cake!",
			M: map[string]int{"01": 1, "10": 2, "11": 3},
		},
	}

	// Open a connection to the server.
	rw, err := Open(ip + Port)
	if err != nil {
		return errors.Wrap(err, "Client: Failed to open connection to "+ip+Port)
	}

	// Send a STRING request.
	// Send the request name.
	// Send the data.
	log.Println("Send the string request.")
	n, err := rw.WriteString("STRING\n")
	if err != nil {
		return errors.Wrap(err, "Could not send the STRING request ("+strconv.Itoa(n)+" bytes written)")
	}
	n, err = rw.WriteString("Additional data.\n")
	if err != nil {
		return errors.Wrap(err, "Could not send additional STRING data ("+strconv.Itoa(n)+" bytes written)")
	}
	log.Println("Flush the buffer.")
	err = rw.Flush()
	if err != nil {
		return errors.Wrap(err, "Flush failed.")
	}

	// Read the reply.
	log.Println("Read the reply.")
	response, err := rw.ReadString('\n')
	if err != nil {
		return errors.Wrap(err, "Client: Failed to read the reply: '"+response+"'")
	}

	log.Println("STRING request: got a response:", response)

	// Send a GOB request.
	// Create an encoder that directly transmits to `rw`.
	// Send the request name.
	// Send the GOB.
	log.Println("Send a struct as GOB:")
	log.Printf("Outer complexData struct: \n%#v\n", testStruct)
	log.Printf("Inner complexData struct: \n%#v\n", testStruct.C)
	enc := gob.NewEncoder(rw)
	n, err = rw.WriteString("GOB\n")
	if err != nil {
		return errors.Wrap(err, "Could not write GOB data ("+strconv.Itoa(n)+" bytes written)")
	}
	err = enc.Encode(testStruct)
	if err != nil {
		return errors.Wrapf(err, "Encode failed for struct: %#v", testStruct)
	}
	err = rw.Flush()
	if err != nil {
		return errors.Wrap(err, "Flush failed.")
	}
	return nil
}

// server listens for incoming requests and dispatches them to
// registered handler functions.
func server() error {
	endpoint := NewEndpoint()

	// Add the handle funcs.
	endpoint.AddHandleFunc("STRING", handleStrings)
	endpoint.AddHandleFunc("GOB", handleGob)

	// Start listening.
	return endpoint.Listen()
}

/*
## Main

Main starts either a client or a server, depending on whether the `connect`
flag is set. Without the flag, the process starts as a server, listening
for incoming requests. With the flag the process starts as a client and connects
to the host specified by the flag value.

Try "localhost" or "127.0.0.1" when running both processes on the same machine.

*/

// main
func main() {
	connect := flag.String("connect", "", "IP address of process to join. If empty, go into listen mode.")
	flag.Parse()

	// If the connect flag is set, go into client mode.
	if *connect != "" {
		err := client(*connect)
		if err != nil {
			log.Println("Error:", errors.WithStack(err))
		}
		log.Println("Client done.")
		return
	}

	// Else go into server mode.
	err := server()
	if err != nil {
		log.Println("Error:", errors.WithStack(err))
	}

	log.Println("Server done.")
}

// The Lshortfile flag includes file name and line number in log messages.
func init() {
	log.SetFlags(log.Lshortfile)
}

/*
## How to get and run the codes

Step 1: `go get` the code. Note the `-d` flag that prevents auto-installing
the binary into `$GOPATH/bin`.

    go get -d github.com/appliedgo/networking

Step 2: `cd` to the source code directory.

    cd $GOPATH/github.com/appliedgo/networking

Step 3. Run the server.

    go run networking.go

Step 4. Open another shell, `cd` to the source code (see Step 2), and
run the client.

    go run networking.go -connect localhost



Happy coding!

*/
