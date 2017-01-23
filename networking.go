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
title = ""
description = ""
author = "Christoph Berger"
email = "chris@appliedgo.net"
date = "2016-00-00"
publishdate = "2016-00-00"
draft = "true"
domains = [""]
tags = ["", "", ""]
categories = ["Tutorial"]
+++

### Summary goes here

<!--more-->

## Intro goes here

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
	"io"
	"log"
	"net"
	"strconv"

	"github.com/pkg/errors"

	"flag"
)

var (
	RemoteNode *net.Addr
	Addr       *net.Addr
	Port       int
)

/* Using an outgoing connection is a snap. A `net.Conn` satisfies the io.Reader
and `io.Writer` interfaces, so we can treat a TCP connection just like any other
`Reader` or `Writer`.
*/

// Open connects to a node's TCP Address.
// It returns a TCP connection wrapped into a buffered reader.
// To close the connection, call the reader's
// Close method.
func (node *Node) Open() (io.Reader, error) {
	conn, err := net.DialTCP("tcp", Addr, node.addr)
	if err != nil {
		return nil, errors.Wrap(err, "Dialing "+node.addr.String()+" failed")
	}
	return bufio.NewReader(conn), nil
}

/* Preparing for incoming data takes a bit more effort. According to our ad-hoc protocol,
we receive the name of a request terminated by `\n`, followed by data. The nature of the data
depends on the respective request. To handle this, we create an `Endpoint` object
with the following properties:

* It allows to register one or more handler functions, where each can handle
a particular request.
* It dispatches incoming requests to the associated handler based on the request name.

*/

// HandleFunc is a function that handles incoming data.
type HandleFunc func(*net.Conn) error

// Endpoint provides an endpoint to other processess
// that they can send data to.
type Endpoint struct {
	addr     net.Addr
	listener *net.Listener
	handler  map[string]HandleFunc
}

// NewEndpoint creates a new endpoint. Too keep things simple,
// the endpoint listens on a fixed port number.
func NewEndpoint() *Endpoint {
	a, err := net.ResolveIPAddr("tcp", strconv.Itoa(Port))
	if err != nil {
		return &Endpoint{} // FIXME: 'New' functions don't return errors, but here we have one.
	}
	return &Endpoint{
		addr:    a,
		handler: map[string]HandleFunc{},
	}
}

// AddHandleFunc adds a new function for handling incoming data.
func (e *Endpoint) AddHandleFunc(name string, f HandleFunc) {
	e.handler[name] = f
}

// Listen starts listening on the endpoint port. At least one
// handler function must have been added through AddHandleFunc() before.
func (e *Endpoint) Listen() error {
	e.listener = net.Listen("tcp")
	for {
		conn, err := e.listener.Accept()
		if err != nil {
			log.Println("Failed accepting a connection request:", err)
			continue
		}
		go dispatch(conn)
	}
	return nil
}

// dispatch reads the connection up to the first newline.
// Based on this string, it calls the appropriate HandlerFunc.
func (e *Endpoint) dispatch(c *net.Conn) {
	defer conn.Close()
	// Wrap the connection into a buffered reader for easier reading.
	r := bufio.NewReader(conn)
	// Read the request.
	request, err := r.ReadString('\n')
	if err != nil {
		log.Println("Error reading command. Got: '"+request+"'\n", err)
	}
	// Call the appropriate handler function.
	e.handler[request](r)
}

/* Now let's create two handler functions. The easiest case is where our ad-hoc protocol only
sends string data.

The second handler receives and processes a struct that was send as GOB data.
*/

// main
func main() {
	join := flag.String("join", "", "IP address of process to join. If empty, go into listen mode.")
	endpoint := NewTCPEndpoint()
	nodes := Nodes{}

}
