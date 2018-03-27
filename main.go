package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
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
	usr, _ := user.Current()
	dir := usr.HomeDir
	fp := filepath.Join(dir, ".credentials", "dnscreds")
	jf, err := os.Open(fp)
	if err != nil {
		log.Fatalf("Failure to access credentials: %v", err)
	}
	b, _ := ioutil.ReadAll(jf)
	err = json.Unmarshal(b, &ds)
	if err != nil {
		log.Fatalf("Unable to unmarshal creds: %v", err)
	}

	return ds
}

func restartNetwork() (ok bool, err error) {
	// Reload the daemon because otherwise we can't restart network
	daemonReload := exec.Command("systemctl", "daemon-reload")
	_, err = daemonReload.CombinedOutput()
	if err != nil {
		log.Printf("Could not reload daemon. %v \n Continuing", err)
	}
	// Restart the Pi's network
	netRestart := exec.Command("systemctl", "restart", "network.service")
	_, err = netRestart.CombinedOutput()
	if err != nil {
		log.Printf("Failed to restart network: %v", err)
		return false, nil
	}
	return true, nil
}

func updateDNS(u, p string) (err error) {
	req, err := http.NewRequest("GET", "https://domains.google.com/nic/update?hostname=home.gluecode.net", nil)
	if err != nil {
		log.Printf("unable to create request: %v", err)
		return err
	}
	req.SetBasicAuth(u, p)

	c := gobundledhttp.NewClient()
	c.Timeout = 15 * time.Second

	resp, err := c.Do(req)
	if err != nil {
		log.Printf("failed to update Google DNS")
		return err
	}
	result, _ := ioutil.ReadAll(resp.Body) // Don't actually care, if it succeeds
	resp.Body.Close()
	log.Printf("Updated DNS: %s", result)

	return
}

func main() {
	// setup signal catching
	sigs := make(chan os.Signal, 1)
	// catch all signals since not explicitly listing
	signal.Notify(sigs)
	//signal.Notify(sigs,syscall.SIGQUIT)
	// method invoked upon seeing signal
	go func() {
		s := <-sigs
		log.Printf("RECEIVED SIGNAL: %s", s)
		os.Exit(1)
	}()

	// Get creds and set up timers
	creds := getCreds()
	tick := time.Tick(3 * time.Minute)
	// Update DNS immediately
	_ = updateDNS(creds.Username, creds.Password)

	// Forever loop waiting for ticker
	for {
		select {
		case <-tick:
			// Every 3 minutes, attempt to update DNS. Restart network
			err := updateDNS(creds.Username, creds.Password)
			if err != nil {
				ok, e := restartNetwork()
				if e != nil {
					log.Printf("catastrophic error: %v", e)
				}
				if ok {
					updateDNS(creds.Username, creds.Password)
				}
			}
		default:
			time.Sleep(20 * time.Second)
		}
	}
}
