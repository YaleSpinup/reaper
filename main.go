package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sort"
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

	// AppConfig is the global applicatoin configuration
	AppConfig common.Config
)

// Notification template
const NotificationTemplate = `
<html>
<head>
<meta http-equiv="refresh" content="2;url=https://spinup.internal.yale.edu" />
<title>Success</title>
</head>
<body>
Success. Redirecting to the <a href="https://spinup.internal.yale.edu">spinup portal</a>.
</body>
</html>`

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
	AppConfig = config

	// Set the loglevel, info if it's unset
	switch AppConfig.LogLevel {
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

	err = Start(ctx)
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

	api.HandleFunc("/reaper/renew/{id:[A-Za-z0-9-]+}", RenewalHander)

	srv := &http.Server{
		Handler:      handlers.LoggingHandler(os.Stdout, router),
		Addr:         AppConfig.Listen,
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

// RenewalHander handles resource renewal
// - The request method is checked, it should be GET
// - Query parameter 'token' is retrieved from the request
// - The subject resource id is retrieved from the URL variable
// - Resource with the id 'id' is fetched from elasticsearch
// - Token is validated against the information pulled from the resource
// - If everything is good, the renewed_at tag is updated
func RenewalHander(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte{})
		return
	}

	// Look for the token in the query
	tokens, ok := r.URL.Query()["token"]
	if !ok || len(tokens) != 1 {
		log.Warnf("Token parameter is missing or of bad format for request: %s", r.URL)
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte{})
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte{})
		return
	}

	finder, err := search.NewFinder(&AppConfig)
	if err != nil {
		log.Errorln("Couldn't configure a new finder", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed connecting to elasticsearch"))
		return
	}

	resource, err := finder.DoGet("resources", "server", id)
	if err != nil {
		log.Errorf("Couldn't get the %s resource from elasticsearch, %s", id, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed getting details about resource"))
		return
	}

	log.Debugf("Got resource %+v back from elasticsearch", resource)

	renewalSecret := &RenewalSecret{RenewedAt: resource.RenewedAt, Secret: AppConfig.EncryptionSecret}
	if err = renewalSecret.ValidateRenewalToken(tokens[0]); err != nil {
		log.Warnf("Failed to validate token string %s, %s", tokens[0], err.Error())
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte{})
		return
	}

	log.Infof("Renewing %s", vars["id"])

	tagger := NewTagger(AppConfig.Tagging.Endpoint, AppConfig.Tagging.Token, id, resource.Org)
	if err = tagger.Tag(map[string]string{"yale:renewed_at": time.Now().Format("2006/01/02 15:04:05")}); err != nil {
		log.Errorf("Failed to renew resource %s, %s", id, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Unable to process renewal, please try again later."))
		return
	}

	w.WriteHeader(http.StatusOK)

	// TODO: parse this template with redirect URL
	w.Write([]byte(NotificationTemplate))
}

// Start fires up the batching routine loop which will do a search for each step;
// Notify, Decommission, Destroy, and then execute those steps.
func Start(ctx context.Context) error {
	interval, err := time.ParseDuration(AppConfig.Interval)
	if err != nil {
		log.Errorf("Couldn't parse interval duration %s. %+v", interval, err)
		return err
	}
	ticker := time.NewTicker(interval)

	// sort notifier schedule
	sort.Sort(BySchedule(AppConfig.Notify.Age))

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
				go runNotifier(&wg)

				wg.Add(1)
				go runDecommissioner(&wg)

				wg.Add(1)
				go runDestroyer(&wg)

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

// runNotifier runs the routine to search for resources with renewed_at dates within a given range configured for notification.
// If the age-based notification threshold is crossed and a notification hasn't been sent:
// - Update the notified_at tag on the instance
// - Bail on this resource and continue to the next if tagging fails
// - Notify with a Notifier
// - Rollback tag if the notification fails
func runNotifier(wg *sync.WaitGroup) {
	log.Infoln("Launching Notifier...")
	defer wg.Done()

	finder, err := search.NewFinder(&AppConfig)
	if err != nil {
		log.Errorln("Couldn't configure a new finder", err)
		return
	}

	ages := AppConfig.Notify.Age
	lte := fmt.Sprintf("now-%s", ages[0])
	log.Debugf("%s >= renewed_at", lte)

	// Query for anything older than the oldest age with the configured filters and status created
	termfilter := append(search.NewTermQueryList(AppConfig.Filter), search.TermQuery{Term: "status", Value: "created"})
	resources, err := finder.DoDateRangeQuery("resources", &search.DateRangeQuery{
		Field:      "yale:renewed_at",
		Format:     "YYYY/MM/dd HH:mm:ss",
		Lte:        lte,
		TermFilter: termfilter,
	})

	if err != nil {
		log.Errorln("Failed to execute date range query", err)
		return
	}

	// loop over the returned resources
	for _, r := range resources {
		log.Debugf("Checking returned resource: %+v", r)

		if r.Org == "" {
			log.Errorf("Cannot operate on a resource without an org.  ID: %s", r.ID)
			continue
		}

		// time of the last renewal
		renewedAt, err := time.Parse("2006/01/02 15:04:05", r.RenewedAt)
		if err != nil {
			log.Errorf("%s Couldn't parse renewed_at (%s) as a time value. %s", r.ID, r.RenewedAt, err.Error())
			continue
		}
		log.Infof("%s last renewed at %s", r.ID, renewedAt.String())

		decomAge, err := parseDuration(AppConfig.Decommission.Age)
		if err != nil {
			log.Errorf("%s Couldn't parse %s as a duration. %s", r.ID, AppConfig.Decommission.Age, err.Error())
			return
		}
		decomAt := renewedAt.Add(decomAge)

		renewalSecret := &RenewalSecret{
			RenewedAt: r.RenewedAt,
			Secret:    AppConfig.EncryptionSecret,
		}
		token, err := renewalSecret.GenerateRenewalToken()
		if err != nil {
			log.Errorf("Failed to generate renewal token, %s", err.Error())
			continue
		}
		renewalLink := fmt.Sprintf("%s/renew/%s?token=%s", AppConfig.BaseURL, r.ID, token)
		log.Debugf("Generated renewal link: %s", renewalLink)

		if r.NotifiedAt == "" {
			log.Infof("%s Notified At is not set, Notifying on age threshold %s", r.ID, ages[0])
			err := notify(r, renewalLink, decomAt, renewedAt)
			if err != nil {
				log.Errorf("Failed to notify. %s", err.Error())
				continue
			}
		} else {
			// time of the last notification
			notifiedAt, err := time.Parse("2006/01/02 15:04:05", r.NotifiedAt)
			if err != nil {
				log.Errorf("%s Couldn't parse notified_at (%s) as a time value. %s", r.ID, r.NotifiedAt, err.Error())
				continue
			}
			log.Infof("%s last notified at %s", r.ID, notifiedAt.String())

			// range over the ages and check if we've notified since the age threshold was crossed
			for _, age := range ages {
				log.Debugf("Checking for notification on age %s", age)

				ageDuration, err := parseDuration(age)
				if err != nil {
					log.Errorf("%s Couldn't parse %s as a duration. %s", r.ID, age, err.Error())
					// If we can't parse the age, stop trying and move on
					break
				}

				// time the age threshold was crossed
				ageThresholdAt := renewedAt.Add(ageDuration)
				log.Infof("%s crossed the %s age threshold at %s", r.ID, age, ageThresholdAt.String())

				if ageThresholdAt.Before(time.Now()) && notifiedAt.Before(ageThresholdAt) {
					log.Infof("%s notified (%s) before age threshold (%s) was crossed (%s). Notifying", r.ID, notifiedAt.String(), age, ageThresholdAt.String())

					err := notify(r, renewalLink, decomAt, renewedAt)
					if err != nil {
						log.Errorf("Failed to notify. %s", err.Error())
					}

					// stop notifying if we matched
					break
				}

				log.Debugf("%s has been notified (%s) since crossing the %s age threshold (%s)", r.ID, notifiedAt.String(), age, ageThresholdAt.String())
			}
		}
	}
}

func notify(r *search.Resource, renewalLink string, decomAt, renewedAt time.Time) error {
	tagger := NewTagger(AppConfig.Tagging.Endpoint, AppConfig.Tagging.Token, r.ID, r.Org)
	err := tagger.Tag(map[string]string{
		"yale:notified_at": time.Now().Format("2006/01/02 15:04:05"),
	})

	if err != nil {
		log.Errorf("Failed to update tag for %s, not notifying, %s", r.ID, err.Error())
		return err
	}

	err = NewNotifier(AppConfig.Notify.Endpoint, AppConfig.Notify.Token).Notify(map[string]string{
		"netid":      r.CreatedBy,
		"link":       renewalLink,
		"expire_on":  decomAt.Format("2006/01/02 15:04:05"),
		"renewed_at": renewedAt.Format("2006/01/02 15:04:05"),
		"fqdn":       r.FQDN,
	})

	if err != nil {
		log.Errorf("Failed to notify.  Rolling back notified_at tag to ''. %s", err.Error())

		// if the notification failed, try to roll back the tag
		err = tagger.Tag(map[string]string{
			"yale:notified_at": r.NotifiedAt,
		})

		if err != nil {
			log.Errorf("Failed to roll back notified_at tag for %s, %s", r.ID, err.Error())
		}
	}

	return err
}

// runDecommissioner runs the routine to search for resources with renewed_at dates
// within the decommission age and the destroy age
func runDecommissioner(wg *sync.WaitGroup) {
	log.Infoln("Launching Decommissioner...")
	defer wg.Done()

	finder, err := search.NewFinder(&AppConfig)
	if err != nil {
		log.Errorln("Couldn't configure a new finder", err)
		return
	}

	// Query for anything older than the decommission age with the configured filters and status created
	termfilter := append(search.NewTermQueryList(AppConfig.Filter), search.TermQuery{Term: "status", Value: "created"})
	resources, err := finder.DoDateRangeQuery("resources", &search.DateRangeQuery{
		Field:      "yale:renewed_at",
		Format:     "YYYY/MM/dd HH:mm:ss",
		Lte:        fmt.Sprintf("now-%s", AppConfig.Decommission.Age),
		TermFilter: termfilter,
	})

	if err != nil {
		log.Errorln("Failed to execute date range query", err)
		return
	}

	// loop over the returned resources
	for _, r := range resources {
		log.Debugf("Checking returned resource: %+v", r)

		if r.Org == "" {
			log.Errorf("Cannot operate on a resource without an org.  ID: %s", r.ID)
			continue
		}

		// time of the last renewal
		renewedAt, err := time.Parse("2006/01/02 15:04:05", r.RenewedAt)
		if err != nil {
			log.Errorf("%s Couldn't parse renewed_at (%s) as a time value. %s", r.ID, r.RenewedAt, err.Error())
			continue
		}
		log.Infof("%s last renewed at %s", r.ID, renewedAt.String())

		destroyAge, err := parseDuration(AppConfig.Destroy.Age)
		if err != nil {
			log.Errorf("%s Couldn't parse %s as a duration. %s", r.ID, AppConfig.Destroy.Age, err.Error())
			return
		}
		// Add the destroy age to the renewed_at date to get the destroy_at date
		destroyAt := renewedAt.Add(destroyAge)

		if destroyAt.Before(time.Now()) {
			log.Warnf("%s has crossed the destroy threshold but hasn't been decommissioned (Destruction scheduled: %s)", r.ID, destroyAt.String())
		}

		log.Infof("%s has crossed the decommision threshold. (Destruction scheduled: %s)", r.ID, destroyAt.String())

		err = NewDecommissioner(AppConfig.Decommission.Endpoint, AppConfig.Decommission.Token, r.ID, r.Org).SetStatus()
		if err != nil {
			log.Errorf("Unable to decommission %s, %s", r.ID, err.Error())
		}
	}
}

// runDestroyer runs the routine to search for resources with renewed_at dates
// beyond the destroy age
func runDestroyer(wg *sync.WaitGroup) {
	log.Infoln("Launching Destroyer...")
	defer wg.Done()

	finder, err := search.NewFinder(&AppConfig)
	if err != nil {
		log.Errorln("Couldn't configure a new finder", err)
		return
	}

	// Query for anything older than the destroy age with the configured filters and status decom
	termfilter := append(search.NewTermQueryList(AppConfig.Filter), search.TermQuery{Term: "status", Value: "decom"})
	resources, err := finder.DoDateRangeQuery("resources", &search.DateRangeQuery{
		Field:      "yale:renewed_at",
		Format:     "YYYY/MM/dd HH:mm:ss",
		Lte:        fmt.Sprintf("now-%s", AppConfig.Destroy.Age),
		TermFilter: termfilter,
	})

	if err != nil {
		log.Errorln("Failed to execute date range query", err)
		return
	}

	// loop over the returned resources
	for _, r := range resources {
		log.Debugf("Checking returned resource: %+v", r)

		if r.Org == "" {
			log.Errorf("Cannot operate on a resource without an org.  ID: %s", r.ID)
			continue
		}

		// time of the last renewal
		renewedAt, err := time.Parse("2006/01/02 15:04:05", r.RenewedAt)
		if err != nil {
			log.Errorf("%s Couldn't parse renewed_at (%s) as a time value. %s", r.ID, r.RenewedAt, err.Error())
			continue
		}

		log.Infof("%s last renewed at %s", r.ID, renewedAt.String())
		log.Infof("%s has crossed the destruction threshold.", r.ID)

		err = NewDestroyer(AppConfig.Decommission.Endpoint, AppConfig.Decommission.Token, r.ID, r.Org).Destroy()
		if err != nil {
			log.Errorf("Unable to destroy %s, %s", r.ID, err.Error())
		}
	}

}
