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
Message queue protocols, gRPC, protobuf, FlatBuffers, RESTful Web API's, WebSockets, and so on.

However, in some situations--especially with small projects--, any approach
you choose may look like completely oversized, not to mention the additional
package dependencies that you'd have to introduce.

Luckily, creating simple network communication with the standard `net` package
is not as difficult as it may seem.


## Simplification #1: connections are io streams

The `net.Conn` interface implements the `io.Reader`, `io.Writer`, and `io.Closer`
interfaces. Hence you can use a TCP connection like any `io` stream.

I know what you think -- "Ok, so I can send strings or byte slices over a
TCP connection. That's nice but what about complex data types? Structs and such?"


## Simplification #2: Go knows how to encode complex types efficiently

When it comes to encoding structured data for sending over the net, JSON comes
readily to mind. But wait - Go's standard `encoding/gob` package provides
a way of serializing and deserializing Go data types without the need for
adding string tags to structs, dealing with JSON/Go incompatibilites, or waiting
for json.Unmarshal to laboriously parse text into binary data.

Gob encoders and decoders work directly on `io` streams - and this fits just
nicely into our simplification #1 - connections are `io` streams.

Let's put this all together in a small sample app.



## The code

The two basic objects we need for TCP networking are:

* Information about the node(s) that we want to send data to, and
* information about the socket on which we will be listening for incoming
	connections.

For brevity, we assume that we have only one other process to communicate with.
}
*/

// ## Imports and globals
package main

import (
	"bufio"
	"log"
	"net"
	"strconv"

	"github.com/pkg/errors"

	"encoding/gob"
	"flag"
)

// A struct with a mix of fields, used for the GOB example.
type complexData struct {
	N int
	S string
	M map[string]int
}

const (
	// Port is the port number to listen to.
	Port = 61000
)

/* Using an outgoing connection is a snap. A `net.Conn` satisfies the io.Reader
and `io.Writer` interfaces, so we can treat a TCP connection just like any other
`Reader` or `Writer`.
*/

// Open connects to a TCP Address.
// It returns a TCP connection wrapped into a buffered reader.
// To close the connection, call the reader's
// Close method.
func Open(addr *net.TCPAddr) (*bufio.ReadWriter, error) {
	// Dial the remote process. The local address (second argument) can
	// be nil, as the local port is chosen on the fly.
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return nil, errors.Wrap(err, "Dialing "+addr.String()+" failed")
	}
	return bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)), nil
}

/* Preparing for incoming data is a bit more involved. According to our ad-hoc
protocol, we receive the name of a request terminated by `\n`, followed by data.
The nature of the data depends on the respective request. To handle this, we
create an `Endpoint` object with the following properties:

* It allows to register one or more handler functions, where each can handle a particular request.
* It dispatches incoming requests to the associated handler based on the request name.

*/

// HandleFunc is a function that handles incoming data.
type HandleFunc func(net.Conn)

// Endpoint provides an endpoint to other processess
// that they can send data to.
type Endpoint struct {
	addr     net.Addr
	listener net.Listener
	handler  map[string]HandleFunc
}

// NewEndpoint creates a new endpoint. Too keep things simple,
// the endpoint listens on a fixed port number.
func NewEndpoint() (*Endpoint, error) {
	// Make a net.Addr from the local port number.
	hostAddrs, err := net.LookupHost("localhost")
	if err != nil {
		return nil, errors.Wrap(err, "Cannot determine this machine's IP addresses")
	}
	a, err := net.ResolveIPAddr("ip", net.JoinHostPort(hostAddrs[0], strconv.Itoa(Port)))
	if err != nil {
		return nil, errors.Wrap(err, "Error resolving IP address for new Endpoint")
	}
	// Create a new Endpoint with an address and an empty list of handler funcs.
	return &Endpoint{
		addr:    a,
		handler: map[string]HandleFunc{},
	}, nil
}

// AddHandleFunc adds a new function for handling incoming data.
func (e *Endpoint) AddHandleFunc(name string, f HandleFunc) {
	e.handler[name] = f
}

// Listen starts listening on the endpoint port. At least one
// handler function must have been added through AddHandleFunc() before.
func (e *Endpoint) Listen() error {
	var err error
	e.listener, err = net.Listen("tcp", e.addr.String())
	if err != nil {
		return errors.Wrap(err, "Unable to listen on "+e.addr.String()+"\n")
	}
	log.Println("Listening on", e.listener.Addr().String())
	for {
		conn, err := e.listener.Accept()
		if err != nil {
			log.Println("Failed accepting a connection request:", err)
			continue
		}
		go e.dispatch(conn)
	}
}

// dispatch reads the connection up to the first newline.
// Based on this string, it calls the appropriate HandlerFunc.
func (e *Endpoint) dispatch(conn net.Conn) {
	// Close the connection when the handler func finishes
	// or when an error occurs.
	defer func() { _ = conn.Close() }()

	// Wrap the connection into a buffered reader for easier reading.
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	// Read the request.
	request, err := rw.ReadString('\n')
	if err != nil {
		log.Println("Error reading command. Got: '"+request+"'\n", err)
	}
	log.Println("Received request '" + request + "'")
	// Call the appropriate handler function.
	e.handler[request](conn)
}

/* Now let's create two handler functions. The easiest case is where our
ad-hoc protocol only sends string data.

The second handler receives and processes a struct that was send as GOB data.
*/

// handleStrings handles the "STRING" request.
func handleStrings(conn net.Conn) {
	// Receive a string.
	r := bufio.NewReader(conn)
	s, err := r.ReadString('\n')
	if err != nil {
		log.Println("Cannot create Reader from connection.\n", err)
	}
	log.Println("Received STRING message:", s)
	w := bufio.NewWriter(conn)
	_, err = w.WriteString("Thank you.\n")
	if err != nil {
		log.Println("Cannot write to connection.\n", err)
	}
}

// handleGob handles the "GOB" request. It decodes the received GOB data
// into a struct.
func handleGob(conn net.Conn) {
	var data complexData
	r := bufio.NewReader(conn)
	// Create a decoder that decodes directly into a struct variable.
	dec := gob.NewDecoder(r)
	err := dec.Decode(&data)
	if err != nil {
		log.Println("Error decoding GOB data:", err)
		return
	}
	log.Printf("Received GOB data:\n%#v\n", data)

}

/* With all this in place, we can now set up client and server functions.

The client function connects to the server and sends STRING and GOB requests.

The server starts listening for requests and triggers the appropriate handlers.
*/

// client is called if the app is called with -connect=`ip addr`.
func client(ip string) error {
	// Some test data.
	testStruct := complexData{
		N: 23,
		S: "string data",
		M: map[string]int{"one": 1, "two": 2, "three": 3},
	}

	// Create a TCPAddr from the ip string passed as -connect flag.
	addr, err := net.ResolveTCPAddr("tcp", ip+":"+strconv.Itoa(Port))
	if err != nil {
		return errors.Wrap(err, "Client: Cannot resolve address "+ip+"\n")
	}

	// Open a connection to the server.
	rw, err := Open(addr)
	if err != nil {
		return errors.Wrap(err, "Client: Failed to open connection to "+addr.String())
	}

	// Send a STRING request.
	// Send the request name.
	// Send the data.
	log.Println("Sending the string request.")
	n, err := rw.WriteString("STRING\n")
	if err != nil {
		return errors.Wrap(err, "Could not send the STRING request ("+strconv.Itoa(n)+" bytes written)")
	}
	n, err = rw.WriteString("Additional data.\n")
	if err != nil {
		return errors.Wrap(err, "Could not send additional STRING data ("+strconv.Itoa(n)+" bytes written)")
	}

	// Read the reply.
	response, err := rw.ReadString('\n')
	if err != nil {
		return errors.Wrap(err, "Client: Failed to read the reply: '"+response+"'")
	}

	log.Println("STRING request: got a response:", response)

	// Send a GOB request.
	// Create an encoder that directly transmits to `rw`.
	// Send the request name.
	// Send the GOB.
	log.Println("Sending a struct as GOB.")
	enc := gob.NewEncoder(rw)
	n, err = rw.WriteString("GOB\n")
	if err != nil {
		return errors.Wrap(err, "Could not write GOB data ("+strconv.Itoa(n)+" bytes written)")
	}
	return enc.Encode(testStruct)
}

// server listens for incoming requests and dispatches them to
// registered handler functions.
func server() error {
	endpoint, err := NewEndpoint()
	if err != nil {
		return errors.Wrap(err, "Cannot create endpoint")
	}

	// Add the handle funcs.
	endpoint.AddHandleFunc("STRING", handleStrings)
	endpoint.AddHandleFunc("GOB", handleGob)

	// Start listening.
	return endpoint.Listen()
}

// main
func main() {
	connect := flag.String("connect", "", "IP address of process to join. If empty, go into listen mode.")
	flag.Parse()

	// If the connect flag is set, go into client mode.
	if *connect != "" {
		err := client(*connect)
		if err != nil {
			log.Println("Error:", err)
		}
		log.Println("Client done.")
		return
	}

	// Else go into server mode.
	err := server()
	if err != nil {
		log.Println("Error:", err)
	}

	log.Println("Server done.")
}

func init() {
	log.SetFlags(log.Lshortfile)
}
