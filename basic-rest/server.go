package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// Start reading this from main(), then return here

type Product struct {
	Name  string  `json:"name"` // Struct Tags are used in Json Marshalling to change key names.
	Price float64 `json:"price"`
}

type Products []Product

type ProductHandler struct {
	sync.Mutex // implements Mutual Exclusion >> sync.Mutex has two methods: Lock and Unlock
	// To make sure that only one go-routine can access Products at a time to avoid conflicts.

	// Directly adding structs here will spread them.
	// ie: sync.Mutex's properties will be added directly in ProductHandler (see ph.Unlock() and ph.Lock() used below)

	products Products
}

// Implements Handler Interface (see below)
func (ph *ProductHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		ph.get(w, r)
	case "POST":
		ph.post(w, r)
	case "PUT", "PATCH":
		ph.put(w, r)
	case "DELETE":
		ph.delete(w, r)
	default:
		respondWithJSON(w, http.StatusMethodNotAllowed, "invalid method") // http has built in status code variables. StatusMethodNotAllowed is 405
	}
}

func (ph *ProductHandler) get(w http.ResponseWriter, r *http.Request) {
	defer ph.Unlock() // "defer" statement defers the execution of a function until the surrounding function returns
	// ie: ph.Unlock() will get executed after all functions in this scope (ph.Lock() and respondWithJSON()) have returned.
	ph.Lock()

	id, err := idFromUrl(r)
	if err != nil {
		// if no ID return all products in response
		respondWithJSON(w, http.StatusOK, ph.products)
		return // still need to return after sending response.
	}
	if id >= len(ph.products) || id < 0 {
		respondWithError(w, http.StatusNotFound, "not found")
		return
	}
	respondWithJSON(w, http.StatusOK, ph.products[id])

}
func (ph *ProductHandler) post(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()                // It is a good practise to close the Body once you are done. Google whY??
	body, err := ioutil.ReadAll(r.Body) // Reads all from a Reader (r.Body implements Reader interface)
	// body here is a []byte

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error()) // err.Error() returns the message string
		return                                                           // always return after responding.
	}
	contentType := r.Header.Get("content-type")
	if contentType != "application/json" {
		respondWithError(w, http.StatusUnsupportedMediaType, "content type 'application/json' required")
		return
	}
	var product Product
	err = json.Unmarshal(body, &product) // converts a []byte json string to a Product
	/* json.Unmarshal(data []byte, v interface{}) error
	Parses JSON-encoded data and stores the result in the value pointed to by v (pointer)
	*/
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer ph.Unlock() // defered functions go in a STACK. execution is Last-In-First-Out
	ph.Lock()
	ph.products = append(ph.products, product)
	respondWithJSON(w, http.StatusCreated, product)
}

func (ph *ProductHandler) put(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	id, err := idFromUrl(r)
	if err != nil {
		respondWithError(w, http.StatusNotFound, err.Error())
		return
	}
	body, err := ioutil.ReadAll(r.Body) // Reads all from a Reader (r.Body implements Reader interface)
	// body here is a []byte

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error()) // err.Error() returns the message string
		return                                                           // always return after responding.
	}
	contentType := r.Header.Get("content-type")
	if contentType != "application/json" {
		respondWithError(w, http.StatusUnsupportedMediaType, "content type 'application/json' required")
		return
	}
	var product Product
	err = json.Unmarshal(body, &product) // Takes cares of type validation also (to some extent)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer ph.Unlock()
	ph.Lock()

	// length might change if not locked.
	if id >= len(ph.products) || id < 0 {
		respondWithError(w, http.StatusNotFound, "not found")
		return
	}

	if product.Name != "" {
		ph.products[id].Name = product.Name
	}
	if product.Price != 0.0 {
		ph.products[id].Price = product.Price
	}
	respondWithJSON(w, http.StatusCreated, ph.products[id])
}
func (ph *ProductHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := idFromUrl(r)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "not found")
		return
	}

	defer ph.Unlock()
	ph.Lock()

	if id >= len(ph.products) || id < 0 {
		respondWithError(w, http.StatusNotFound, "not found")
		return
	}

	if id < len(ph.products)-1 { // if not the last product then switch with the last product.
		ph.products[len(ph.products)-1], ph.products[id] = ph.products[id], ph.products[len(ph.products)-1] // Trick to switch last and ith elemetn // BASIC GO PATTERN
	}

	ph.products = ph.products[:len(ph.products)-1] // from 0 upto (not including) last index
	respondWithJSON(w, http.StatusNoContent, "")   // It is convention to not return anything for DELETE request
}

func respondWithJSON(w http.ResponseWriter, code int, data interface{}) { // empty interface is like Any in JS, try avoid using this.
	response, err := json.Marshal(data) // returns a []byte (json in string)
	if err != nil {
		fmt.Println("Error:", err)
	}
	w.Header().Add("content-type", "application/json")
	w.WriteHeader(code) // writes status code (eg. 200 OK)
	w.Write(response)
	// As soon as w.Write() is executed, the Server will send the response
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	respondWithJSON(w, code, map[string]string{"error": msg})
}

// IDs here are simply indexes for simpliciy
func idFromUrl(r *http.Request) (int, error) {
	parts := strings.Split(r.URL.String(), "/") // r.URL.String() returns a string representation of the URL
	if len(parts) != 3 {
		return -1, errors.New("not found") // errors.New returns an error with given message (obviously)
	}
	id, err := strconv.Atoi(parts[len(parts)-1]) // strconv.Atoi parses a string to in int
	if err != nil {
		return -1, errors.New("not found")
	}
	return id, nil

}

func newProductHandler() *ProductHandler { // Kinda like a constructor // BASIC GO PATTERN
	return &ProductHandler{
		products: Products{
			Product{"Shoes", 25.00},
			Product{"Webcam", 50.00},
			Product{"Mic", 20.00},
		},
	}
}

func main() {
	port := ":8080"
	ph := newProductHandler()

	/* http.Handle(pattern string, handler Handler) takes a Handler interface:
	type Handler interface {
		ServeHTTP(ResponseWriter, *Request)
	} */

	http.Handle("/products", ph)  // won't match "/products/" so be careful
	http.Handle("/products/", ph) // "/products" and "/products/" are handled differently

	/* http.HandleFunc(pattern string, handler func(ResponseWriter, *Request)) */
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello World") // Fprintf takes a Writer and outputs to it.
	})

	// Routes must be defined before starting server
	launchErr := http.ListenAndServe(port, nil) // starts a server > listens at port
	// second arg is a handler. nil will result in DefaultServeMux to be used.
	// http.Handle and http.HandleFunc to add handlers to DefaultServeMux

	log.Fatal(launchErr) // Will log incase something goes wrong
	// log.Fatal is like fmt.Print followed by a call to os.Exit(1)

}
