package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"syscall"
	"time"

	"github.com/tydavis/gobundledhttp"
)

type dnscreds struct {
	Username string `json:"Username"`
	Password string `json:"Password"`
}

func getCreds() dnscreds {
	var ds dnscreds
	// Expand the home directory to get the proper file path
	usr, e := user.Current()
	if e != nil {
		log.Fatalf("unable to get current user: %v", e)
	}
	dir := usr.HomeDir
	fp := filepath.Join(dir, ".credentials", "dnscreds")
	jf, err := os.Open(fp)
	if err != nil {
		log.Fatalf("failure to access credentials: %v", err)
	}
	b, e := ioutil.ReadAll(jf)
	if e != nil {
		log.Fatalf("unable to read credentials file: %v", e)
	}
	err = json.Unmarshal(b, &ds)
	if err != nil {
		log.Fatalf("unable to unmarshal creds: %v", err)
	}

	return ds
}

func restartNetwork() (ok bool, err error) {
	// Reload the daemon because otherwise we can't restart network
	daemonReload := exec.Command("sudo", "systemctl", "daemon-reload")
	_, err = daemonReload.CombinedOutput()
	if err != nil {
		log.Printf("Could not reload daemon. %v \n Continuing", err)
	}
	// Restart the Pi's network
	netRestart := exec.Command("sudo", "systemctl", "restart", "networking.service")
	_, err = netRestart.CombinedOutput()
	if err != nil {
		log.Printf("Failed to restart network: %v", err)
		return false, nil
	}
	return true, nil
}

func updateDNS(c *http.Client, req *http.Request) (err error) {
	resp, err := c.Do(req)
	if err != nil {
		log.Printf("failed to update Google DNS")
		return err
	}
	result, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		log.Fatalln("unable to read request response body: %v", err)
	}
	log.Printf("Updated DNS: %s", result)

	return
}

func main() {
	// setup signal catching
	sigs := make(chan os.Signal, 1)
	// Explicitly catch these signals
	signal.Notify(sigs, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM)
	go func() {
		s := <-sigs
		switch s {
		case syscall.SIGQUIT:
			log.Printf("RECEIVED QUIT: %s", s)
			os.Exit(0)
		case syscall.SIGTERM:
			log.Printf("RECEIVED TERM: %s", s)
			os.Exit(0)
		case os.Interrupt:
			log.Printf("RECEIVED INTERRUPT: %s", s)
			os.Exit(0)
		default:
			log.Printf("RECEIVED SIGNAL: %s", s)
			os.Exit(1)
		}
	}()

	// Mandatory profiling endpoint on an alternate (never directly exposed) port.
	// Put this on an alternate goroutine to avoid locking/blocking operations
	// where possible.
	go func() {
		// Need it to bind to all interfaces, not just localhost
		hp := fmt.Sprintf("127.0.0.1:%d", 6060)
		log.Printf("starting debug endpoint at %s ", hp)
		// Log server output to avoid silently dying / failing creating endpoint.
		log.Println(http.ListenAndServe(hp, nil))
	}() // End debug endpoint.

	// Get creds and set up timers
	creds := getCreds()
	tick := time.Tick(3 * time.Minute)

	// Build Request
	req, err := http.NewRequest("GET", "https://domains.google.com/nic/update?hostname=home.gluecode.net", nil)
	if err != nil {
		log.Fatalf("unable to create request: %v", err)
	}
	req.SetBasicAuth(creds.Username, creds.Password)

	c := gobundledhttp.NewClient()

	// Update DNS immediately
	err = updateDNS(c, req)
	if err != nil {
		log.Fatalf("catastrophic error: %v", err)
	}

	// Forever loop waiting for ticker
	for {
		select {
		case <-tick:
			// Every 3 minutes, attempt to update DNS. Restart network
			err := updateDNS(c, req)
			if err != nil {
				ok, e := restartNetwork()
				if e != nil {
					log.Printf("catastrophic error: %v", e)
				}
				if ok {
					err := updateDNS(c, req)
					if err != nil {
						log.Fatalf("failed to update after network restart: %v", err)
					}
				}
			}
		default:
			time.Sleep(20 * time.Second)
		}
	}
}
