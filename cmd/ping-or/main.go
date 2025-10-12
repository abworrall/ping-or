package main

// For raspberry pi: `GOOS=linux GOARCH=arm go build github.com/abworrall/ping-orcmd/ping-or/*go`

// If you run this and see `panic: socket: permission denied`
// then try `# sudo sysctl -w net.ipv4.ping_group_range="0 2147483647"`

import(
	"flag"
	"io"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/prometheus-community/pro-bing"
)

var(
	fPingTarget string
	fPingFrequency time.Duration
	fPingFailThreshold time.Duration
	fAction string
	fActionWaitTime time.Duration
	fVerbose = true
)

var restartNetworkScript = `
echo == restarting network kinda thing ==
id
`

func init() {
	flag.StringVar(&fPingTarget, "dest", "8.8.8.8", "IP address that is going to to get pinged")
	flag.DurationVar(&fPingFrequency, "freq", 5*time.Second, "duration to wait between pings")
	flag.DurationVar(&fPingFailThreshold, "threshold", time.Minute, "how long pings can fail before we act")
	flag.StringVar(&fAction, "cmd", "ls -l", "command to run when pings have failed for too long")
	flag.DurationVar(&fActionWaitTime, "wait", time.Minute, "after we act, how long to wait before resuming pinging")
	flag.BoolVar(&fVerbose, "v", false, "verbose - log successful pings")
	flag.Parse()
}

func main() {
	log.Printf("Starting to ping %s every %s. If it fails for >%s, will exec [%s]\n", fPingTarget, fPingFrequency, fPingFailThreshold, fAction)
	pingLoop()
}

func pingLoop() {
	lastOK := time.Now()
	for {
		if ok, _ := ping(); ok {
			lastOK = time.Now()
		} else {
			if time.Since(lastOK) > fPingFailThreshold {
				log.Printf("ping target has been down for %s\n", time.Since(lastOK))
				executeAction()
				lastOK = time.Now()
			}
		}
		time.Sleep(fPingFrequency)
	}
}

func executeAction() {	
	log.Printf("*** Gonna do action %s, then wait for %s\n", fAction, fActionWaitTime)

	output := ""

	switch fAction {
	case "%restart-network%": output = runScript(restartNetworkScript)
	default:                  output = runCommand(fAction)
	}

	log.Printf(">>>>>> action output\n%s\n", output)
	log.Printf("<<<<<<\n")
	
	time.Sleep(fActionWaitTime)
}

// runScript pipes the script contents into a bash shell for execution, and
// returns the output
func runScript(script string) string {
	cmd := exec.Command("bash")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}

	io.WriteString(stdin, script)
	stdin.Close()

	out, err := cmd.CombinedOutput()
	if err != nil {
		panic(err)
	}

	return string(out)
}

func runCommand(cmdString string) string {
	args := strings.Fields(cmdString)

	output, err := exec.Command(args[0], args[1:]...).Output()

	if err != nil {
		switch err.(type) {
		case *exec.Error:
			log.Println("exec failed:", err)
			//case *exec.ExitError:
			// log.Println("command exit rc =", e.ExitCode())
		default:
			panic(err)
		}
	}

	return string(output)
}

func ping() (pingOK bool, pingRtt time.Duration) {
	pingOK = true

	pinger, err := probing.NewPinger(fPingTarget)
	if err != nil {
		panic(err)
	}

	pinger.Count = 1
	pinger.Timeout = time.Second
	pinger.OnRecv = func(pkt *probing.Packet) {
		pingRtt = pkt.Rtt
	}
	pinger.OnFinish = func(stats *probing.Statistics) {
		if stats.PacketsRecv == 0 {
			pingOK = false
		}
	}
	
	err = pinger.Run()
	if err != nil {
		panic(err)
	}

	if fVerbose || !pingOK {
		log.Printf("ping %s (%s): OK=%v, rtt=%s\n", pinger.Addr(), pinger.IPAddr(), pingOK, pingRtt)
	}
	return
}
