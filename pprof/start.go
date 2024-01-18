package pprof

import (
	"log"
	"net/http"
	_ "net/http/pprof"
)

// Start pprof, open http://{{addr}}/debug/pprof/ in web browser for view pprof
func Start(addr string) {
	go func() {
		log.Fatalln(http.ListenAndServe(addr, nil)) // for pprof
	}()
}
