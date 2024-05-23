package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
)

type wrappedHandleFunc func(w http.ResponseWriter, req *http.Request) error

//var debugOneThingAtATime sync.Mutex

func wrapHandleFunc(whf wrappedHandleFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		//debugOneThingAtATime.Lock()
		//defer debugOneThingAtATime.Unlock()
		if err := whf(w, req); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, err.Error())
		}
		// is it a good idea to do this here?
		// that way no one can forget to do it
		req.Body.Close()
	}
}

// fixme should be removed when deployed due to spam
// but useful for devving to not miss anything
func printUnhandled(w http.ResponseWriter, req *http.Request) error {
	dump, err := httputil.DumpRequest(req, true)
	if err != nil {
		log.Printf("failed to dump request: %s", err)
		return errors.New("bad request")
	}
	log.Printf("Unhandled Request:\n%s\nEND Unhandled Request", strings.TrimSpace(string(dump)))

	return errors.New("Whatever you requested is unhandled right now")
}

type Config struct {
	Motd string
}

var (
	userdir = flag.String("datadir", "data", "where the userdata is stored")
)

func main() {
	log.SetFlags(log.Lshortfile)
	flag.Parse()

	config, workingpath := loadConfig(*userdir)
	log.Printf("config: %q", config)

	egg := newEggstore(config.Motd, workingpath)

	mux := http.NewServeMux()
	// Handle /ei/ posts
	mux.HandleFunc("POST /ei/{subpath...}", wrapHandleFunc(egg.handlepath_ei))
	mux.HandleFunc("POST /ei", wrapHandleFunc(egg.handlepath_ei)) //is this a thing? we certainly need a handler or it disappears into the void
	// Analytics
	mux.HandleFunc("POST /ei_data/{subpath...}", wrapHandleFunc(egg.handlepath_eidata))
	// Redirect a pure call to "/" to the Landing Page
	mux.HandleFunc("/{$}", redirect)
	// Landing Pages
	mux.HandleFunc("/stat", egg.index)
	mux.HandleFunc("GET /favicon.ico", getico)
	mux.HandleFunc("GET /privacy", redirect) // the in app privacy button takes us here, forward it
	// Catch-all the rest
	mux.HandleFunc("/", wrapHandleFunc(printUnhandled))

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, os.Kill)

	srv := &http.Server{Handler: mux, Addr: ":80"}
	log.Printf("Starting HTTP server: %v", srv.Addr)
	go func() {
		err := srv.ListenAndServe()
		log.Print(err)
		// if the server crashed we exit
		stop <- os.Kill
	}()

	quitperiodicalrunner := make(chan struct{})
	go func() {
		// we save our state every minute to the files
		interval := time.NewTicker(1 * time.Minute)
		defer interval.Stop()
		for {
			select {
			case <-interval.C:
				egg.SaveData()
			case <-quitperiodicalrunner:
				return
			}
		}
	}()

	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	srv.Shutdown(ctx)
	quitperiodicalrunner <- struct{}{}
	egg.Shutdown()
}

func loadConfig(userdir string) (Config, string) {
	fullpath, err := filepath.Abs(userdir)
	if err != nil {
		log.Panic(err)
	}
	_, err = os.Stat(fullpath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Panicf("directory doesnt exist: %s", fullpath)
		}
		// other error
		log.Panic(err)
	}

	joined := filepath.Join(fullpath, "settings.json")
	serversettingsfile, err := os.Open(joined)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Making a new config file for you in %s", fullpath)
			serversettingsfile, err = os.Create(joined)
			if err != nil {
				log.Panic(err)
			}
			emptyconfig := Config{
				Motd: "WELCOME TO reEgg-go!!\nA custom server!!",
			}
			encoder := json.NewEncoder(serversettingsfile)
			encoder.SetIndent("", "\t")
			err = encoder.Encode(&emptyconfig)
			if err != nil {
				log.Panic(err)
			}
			serversettingsfile.Sync()
			serversettingsfile.Seek(0, io.SeekStart)
		} else {
			log.Panic(err)
		}
	}
	decoder := json.NewDecoder(serversettingsfile)
	config := Config{}
	err = decoder.Decode(&config)
	if err != nil {
		log.Panic(err)
	}
	serversettingsfile.Close()
	return config, fullpath
}
