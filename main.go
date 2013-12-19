package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/dbrower/noids/server"
	"github.com/gorilla/pat"
	flag "github.com/ogier/pflag"
)

type Reopener interface {
	Reopen()
}

type loginfo struct {
	name string
	f    *os.File
}

func NewReopener(filename string) *loginfo {
	return &loginfo{name: filename}
}

func (li *loginfo) Reopen() {
	if li.name == "" {
		return
	}
	if li.f != nil {
		log.Println("Reopening Log files")
	}
	newf, err := os.OpenFile(li.name, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(newf)
	if li.f != nil {
		li.f.Close()
	}
	li.f = newf
}

func signalHandler(sig <-chan os.Signal, logw Reopener) {
	for s := range sig {
		log.Println("Got", s)
		switch s {
		case syscall.SIGUSR1:
			logw.Reopen()
		}
	}
}

func main() {
	var port string
	var storageDir string
	var logfilename string
	var logw Reopener

	flag.StringVarP(&port, "port", "p", "8080", "port to run on")
	flag.StringVarP(&logfilename, "log", "l", "", "name of log file")
	flag.StringVarP(&storageDir, "storage", "s", "", "directory to save noid information")

	flag.Parse()

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	logw = NewReopener(logfilename)
	logw.Reopen()
	log.Println("-----Starting Server")

	if storageDir != "" {
		server.StartSaver(server.NewJsonFileSaver(storageDir))
	}
	sig := make(chan os.Signal, 5)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGUSR1, syscall.SIGUSR2)
	go signalHandler(sig, logw)

	r := pat.New()
	r.Get("/pools/{poolname}", server.PoolShowHandler)
	r.Put("/pools/{poolname}/open", server.PoolOpenHandler)
	r.Put("/pools/{poolname}/close", server.PoolCloseHandler)
	r.Post("/pools/{poolname}/mint", server.MintHandler)
	r.Post("/pools/{poolname}/advancePast", server.AdvancePastHandler)
	// r.Get("/stats", StatsHandler)
	r.Get("/pools", server.PoolsHandler)
	r.Post("/pools", server.NewPoolHandler)

	http.Handle("/", r)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
