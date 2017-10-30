package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"git.yale.edu/spinup/reaper/common"
	"git.yale.edu/spinup/reaper/reaper"
	"git.yale.edu/spinup/reaper/search"
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

type BySchedule []string

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

// Start fires up the batching routine loop which will do a search for each step;
// Notify, Decommission, Destroy, and then execute those steps.
func Start(ctx context.Context, config common.Config) error {
	interval, err := time.ParseDuration(config.Interval)
	if err != nil {
		log.Errorf("Couldn't parse interval duration %s. %+v", interval, err)
		return err
	}
	ticker := time.NewTicker(interval)

	// sort notifier schedule
	sort.Sort(BySchedule(config.Notify.Age))

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
				go runNotifier(config, &wg)

				wg.Add(1)
				go runDecommissioner(config, &wg)

				wg.Add(1)
				go runDestroyer(config, &wg)

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

// startHTTPServer registers the api endpoints and starts the webserver listening
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

// runNotifier runs the routine to search for resources with renewed_at dates
// within a given range configured for notification.
func runNotifier(config common.Config, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Infoln("Launching Notifier...")

	finder, err := search.NewFinder(&config)
	if err != nil {
		log.Errorln("Couldn't configure a new finder", err)
		return
	}

	// config.Notify.Age is pre-sorted from smaller duration to larger duration.
	// The duration is subtracted from 'now', so smaller numbers are more recent than
	// larger numbers. Therefore a resource with config.Notify.Age[n] is a newer resource
	// one with config.Notify.Age[n+1].
	for i, newer := range config.Notify.Age {
		var older string
		if len(config.Notify.Age) > i+1 {
			older = config.Notify.Age[i+1]
		} else {
			older = config.Decommission.Age
		}

		lte := fmt.Sprintf("now-%s", newer)
		gt := fmt.Sprintf("now-%s", older)
		log.Debugf("%s >= renewed_at > %s", lte, gt)

		resources, err := finder.DoDateRangeQuery(search.DateRangeQuery{
			Index:  "resources",
			Field:  "yale:renewed_at",
			Format: "YYYY/MM/dd HH:mm:ss",
			Lte:    lte,
			Gt:     gt,
		})

		if err != nil {
			log.Errorln("Failed to execute date range query", err)
			return
		}

		log.Debugf("Got %d resource(s) from notify query for age %s", len(resources), newer)

		for _, r := range resources {
			log.Infof("Notifying for resource ID %s on %s age", r.ID, newer)
		}
	}
}

// runDecommissioner runs the routine to search for resources with renewed_at dates
// within the decommission age and the destroy age
func runDecommissioner(config common.Config, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Infoln("Launching Decommissioner...")

	finder, err := search.NewFinder(&config)
	if err != nil {
		log.Errorln("Couldn't configure a new finder", err)
		return
	}

	newer := config.Decommission.Age
	older := config.Destroy.Age

	lte := fmt.Sprintf("now-%s", newer)
	gt := fmt.Sprintf("now-%s", older)
	log.Debugf("%s >= renewed_at > %s", lte, gt)

	resources, err := finder.DoDateRangeQuery(search.DateRangeQuery{
		Index:  "resources",
		Field:  "yale:renewed_at",
		Format: "YYYY/MM/dd HH:mm:ss",
		Lte:    lte,
		Gt:     gt,
	})

	if err != nil {
		log.Errorln("Failed to execute date range query", err)
		return
	}

	log.Debugf("Got %d resource(s) from decommission query for age %s", len(resources), newer)

	for _, r := range resources {
		if r.Status == "created" {
			log.Infof("Decommissioning resource ID %s on %s age", r.ID, newer)
		} else {
			log.Infof("%s is not in created state (%s). Not proceeding with decom", r.ID, r.Status)
		}
	}
}

// runDestroyer runs the routine to search for resources with renewed_at dates
// beyond the destroy age
func runDestroyer(config common.Config, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Infoln("Launching Destroyer...")
	finder, err := search.NewFinder(&config)
	if err != nil {
		log.Errorln("Couldn't configure a new finder", err)
		return
	}

	newer := config.Destroy.Age

	lte := fmt.Sprintf("now-%s", newer)
	log.Debugf("%s >= renewed_at", lte)

	resources, err := finder.DoDateRangeQuery(search.DateRangeQuery{
		Index:  "resources",
		Field:  "yale:renewed_at",
		Format: "YYYY/MM/dd HH:mm:ss",
		Lte:    lte,
	})

	if err != nil {
		log.Errorln("Failed to execute date range query", err)
		return
	}

	log.Debugf("Got %d resource(s) from destruction query for age %s", len(resources), newer)

	for _, r := range resources {
		if r.Status == "decom" {
			log.Infof("Destroying resource ID %s on %s age", r.ID, newer)
		} else {
			log.Infof("%s is not in decom state (%s). Not proceeding with destruction", r.ID, r.Status)
		}
	}
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

// Len is required to satisfy sort.Interface
func (s BySchedule) Len() int {
	return len(s)
}

// Swap is required to satisfy sort.Interface
func (s BySchedule) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less is required to satisfy sort.Interface
func (s BySchedule) Less(i, j int) bool {
	di, _ := parseDuration(s[i])
	dj, _ := parseDuration(s[j])
	return di < dj
}
