package main

import (
	"bytes"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"text/template"
	"time"

	"github.com/odwrtw/transmission"
	"github.com/rs/xlog"
)

var (
	transmissionURL = flag.String("transmission-url", "http://localhost:9091/transmission/rpc", "The URL of the transmission RPC client")
	removeStalled   = flag.Bool("remove-stalled", false, "Remove stalled torrents")
	removeFinished  = flag.Bool("remove-finished", false, "Remove finished torrents")
	ignoreTemplate  = flag.String("ignore-template", "", "A text/template that is passed a torrent `t` and if evaluates to `true` will ignore the torrent")
	cycles          = flag.Int("cycles", 5, "How many cycles a torrent should maintain stalled or finished cycle before being removed?")
	timeout         = flag.Duration("timeout", 30*time.Second, "How long to wait between cycles")

	tconn    *transmission.Client
	tstrikes = make(map[int]map[string]int)
	it       *template.Template
)

func main() {
	var err error

	flag.Parse()
	if *ignoreTemplate != "" {
		it = template.Must(template.New("ignore-template").Parse(*ignoreTemplate))
	}
	tconn, err = transmission.New(transmission.Config{
		Address:    *transmissionURL,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	})
	if err != nil {
		xlog.Fatalf("error connection to transmission: %s", err)
	}
	signalC := make(chan os.Signal, 1)
	signal.Notify(signalC, os.Interrupt)
	timeoutTicker := time.NewTicker(*timeout)
	for {
		select {
		case <-signalC:
			timeoutTicker.Stop()
			return
		case <-timeoutTicker.C:
			xlog.Info("running a cycle")
			cycle()
		}
	}
}

func cycle() {
	tseen := make(map[int]bool)
	ts, err := tconn.GetTorrents()
	if err != nil {
		xlog.Errorf("error getting the torrents: %s", err)
	}

	// search and remove finished and stalled torrents
	for _, t := range ts {
		// should this torrent be ignored?
		if ignore(t) {
			continue
		}
		// mark the torrent as seen
		tseen[t.ID] = true
		// make sure we have it in the strikes map
		if tstrikes[t.ID] == nil {
			tstrikes[t.ID] = make(map[string]int)
		}
		// is it finished?
		if t.IsFinished {
			tstrikes[t.ID]["finished"]++
		}
		// is it stalled?
		if t.IsStalled {
			tstrikes[t.ID]["stalled"]++
		}
		// has this torrent been marked as stalled for *cycles?
		if *removeStalled && tstrikes[t.ID]["stalled"] >= *cycles {
			xlog.Infof("The torrent %s has been stalled for %d cycles and will be removed", t.Name, tstrikes[t.ID]["stalled"])
			tconn.RemoveTorrents([]*transmission.Torrent{t}, true)
		}
		// has this torrent been marked as finished for *cycles?
		if *removeFinished && tstrikes[t.ID]["finished"] >= *cycles {
			xlog.Infof("The torrent %s has been finished for %d cycles and will be removed", t.Name, tstrikes[t.ID]["finished"])
			tconn.RemoveTorrents([]*transmission.Torrent{t}, false)
		}
	}

	// Garbage-Collect the tstrikes map
	for id := range tstrikes {
		if !tseen[id] {
			delete(tstrikes, id)
		}
	}
}

func ignore(t *transmission.Torrent) bool {
	if it == nil {
		return false
	}
	var buf bytes.Buffer
	if err := it.Execute(&buf, t); err != nil {
		xlog.Errorf("error executing the ignore template: %s", err)
		return true
	}
	if buf.String() == "true" {
		xlog.Infof("The torrent %s has been ignored", t.Name)
		return true
	}
	return false
}
