package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"net/url"
)

//Error handling
func handleErr(err error){
	if err!=nil{
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
}


type Server interface{
	// Address returns the address of the server
	Address() string
	// HealthCheck returns true if the server is healthy
	HealthCheck() bool
	// ServeHTTP forwards the request to the server
	Serve( rw http.ResponseWriter, req *http.Request)
}

type simpleServer struct{
	address string
	proxy *httputil.ReverseProxy
}

func(s *simpleServer) Address() string{ return s.address }
func(s *simpleServer) HealthCheck() bool{ return true }
func(s *simpleServer) Serve(rw http.ResponseWriter, req *http.Request){
	s.proxy.ServeHTTP(rw, req)
}

func newSimpleServer(address string) *simpleServer{
	serverUrl, err := url.Parse(address)
	handleErr(err)
	return &simpleServer{
		address: address,
		proxy: httputil.NewSingleHostReverseProxy(serverUrl),
	}
}

type LoadBalancer struct{
	port string
	roundRobinCounter int
	servers []Server
}

func NewLoadBalancer(port string, servers []Server) *LoadBalancer{
	return &LoadBalancer{
		port: port,
		roundRobinCounter: 0,
		servers: servers,
	}
}

func (lb * LoadBalancer) getNextAvailableServer() Server{
	server := lb.servers[lb.roundRobinCounter%len(lb.servers)]
	for !server.HealthCheck(){
		lb.roundRobinCounter++
		server = lb.servers[lb.roundRobinCounter%len(lb.servers)]
	}
	lb.roundRobinCounter++
	return server
}

func (lb *LoadBalancer) serveProxy(rw http.ResponseWriter, req *http.Request){
	targetServer := lb.getNextAvailableServer()
	fmt.Printf("Forwarding request to server: %s\n", targetServer.Address())
	targetServer.Serve(rw, req)
}


func main(){
	servers := []Server{
		newSimpleServer("https://www.google.com"),
		newSimpleServer("https://www.bing.com"),
		newSimpleServer("https://www.yahoo.com"),
	}

	lb := NewLoadBalancer("8080", servers)

	handleRedirect := func(rw http.ResponseWriter, req *http.Request){
		lb.serveProxy(rw, req)
	}

	http.HandleFunc("/", handleRedirect)
	fmt.Printf("serving requests at 'http://localhost:%s'\n", lb.port)
	http.ListenAndServe(":"+lb.port, nil)
}
