package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"git.yale.edu/spinup/reaper/common"
	"git.yale.edu/spinup/reaper/reaper"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// Version is the main version number
const Version = reaper.Version

// VersionPrerelease is a prerelease marker
const VersionPrerelease = reaper.VersionPrerelease

var (
	configFileName = flag.String("config", "config/config.json", "Configuration file.")
	version        = flag.Bool("version", false, "Display version information and exit.")
	globalWg       sync.WaitGroup
)

func main() {
	flag.Parse()
	if *version {
		vers()
	}

	log.Infof("Reaper version %s%s", Version, VersionPrerelease)

	configFile, err := os.Open(*configFileName)
	if err != nil {
		log.Fatalln("Unable to open config file", err)
	}

	r := bufio.NewReader(configFile)
	config, err := common.ReadConfig(r)
	if err != nil {
		log.Fatalf("Unable to read configuration from %s.  %+v", *configFileName, err)
	}

	// Set the loglevel, info if it's unset
	switch config.LogLevel {
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}

	log.Debugf("Loaded Config: %+v", config)

	// Setup context to allow goroutines to be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = Start(ctx, config)
	if err != nil {
		cancel()
		log.Fatalln("Couldn't initialize schedule routines", err)
	}

	srv := startHTTPServer(cancel)

	// Waitgroup waits for all goroutines to exit
	globalWg.Wait()

	log.Info("Stopping HTTP server")
	if err := srv.Shutdown(context.TODO()); err != nil {
		log.Fatalln("Failed to shutdown HTTP server cleanly", err)
	}
	log.Info("Done. exiting")
}

func vers() {
	fmt.Printf("Reaper Version: %s%s\n", Version, VersionPrerelease)
	os.Exit(0)
}

// parseDuration parses durations of days, weeks and months (in the most simplistic way)
// since time.ParseDuration only supports up to hours https://github.com/golang/go/issues/11473
// If there's a parsing error, return 0 and the error.  Originally, this returned MAXINT64 and the
// error, but time.ParseDuration(foo) returns 0 on error and I wanted to stay consistent.
func parseDuration(d string) (time.Duration, error) {
	switch {
	case strings.HasSuffix(d, "d"):
		t := strings.TrimSuffix(d, "d")
		num, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			return time.Duration(0), err
		}
		return time.Duration(num*24) * time.Hour, nil
	case strings.HasSuffix(d, "w"):
		t := strings.TrimSuffix(d, "w")
		num, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			return time.Duration(0), err
		}
		return time.Duration(num*7*24) * time.Hour, nil
	case strings.HasSuffix(d, "mo"):
		t := strings.TrimSuffix(d, "mo")
		num, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			return time.Duration(0), err
		}
		return time.Duration(num*30*24) * time.Hour, nil
	default:
		return time.ParseDuration(d)
	}
}

// Start fires up the batching routine loop which will do a search for each step;
// Notify, Decommission, Destroy, and then execute those steps.
func Start(ctx context.Context, config common.Config) error {
	interval, err := time.ParseDuration(config.Interval)
	if err != nil {
		log.Errorf("Couldn't parse interval duration %s. %+v", interval, err)
		return err
	}
	ticker := time.NewTicker(interval)

	globalWg.Add(1)
	// launch a goroutine to run our batch on a schedule
	go func() {
		defer globalWg.Done()
		log.Infof("Initializing the batching routine loop.")
		for {
			select {
			case <-ticker.C:
				var wg sync.WaitGroup
				log.Infoln("Batch routine running...")

				wg.Add(1)
				go func() {
					defer wg.Done()
					log.Infoln("Launching Notifier...")
				}()

				wg.Add(1)
				go func() {
					defer wg.Done()
					log.Infoln("Launching Decommissioner...")
				}()

				wg.Add(1)
				go func() {
					defer wg.Done()
					log.Infoln("Launching Destroyer...")
				}()

				log.Infoln("Batch routine sleeping...")
				wg.Wait()
			case <-ctx.Done():
				log.Infoln("Shutdown the batch routine")
				return
			}
		}
	}()

	return nil
}

func startHTTPServer(cancel func()) *http.Server {
	router := mux.NewRouter()
	api := router.PathPrefix("/v1").Subrouter()
	api.HandleFunc("/reaper/ping", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte{})
			return
		}
		log.Debug("Got ping request, responding pong.")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	})

	api.HandleFunc("/reaper/shutdown", func(w http.ResponseWriter, r *http.Request) {
		log.Infoln("Received shutdown request, cancelling goroutines.")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
		cancel()
		return
	})

	srv := &http.Server{
		Handler:      handlers.LoggingHandler(os.Stdout, router),
		Addr:         ":8080",
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  30 * time.Second,
	}

	go func() {
		log.Infof("Starting listener on :8080")
		if err := srv.ListenAndServe(); err != nil {
			log.Infof("Httpserver: ListenAndServe() error: %s", err)
			log.Fatal(err)
		}
	}()

	return srv
}
