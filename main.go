// bmc-inventory prints the inventory bmclib reads from a BMC as JSON on stdout.
// go.mod replaces bmclib with a fork carrying unreleased changes, so they can
// be tested against real hardware.
//
//	BMC_PASSWORD=... bmc-inventory -host 10.0.0.5 -user root | jq .gpus
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/bombsimon/logrusr/v2"
	"github.com/sirupsen/logrus"

	bmclib "github.com/bmc-toolbox/bmclib/v2"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	host := flag.String("host", "", "BMC address (required)")
	user := flag.String("user", "", "BMC username (required)")
	password := flag.String("password", "", "BMC password; prefer the BMC_PASSWORD environment variable so it stays out of shell history")
	port := flag.Int("port", 443, "BMC Redfish port")
	provider := flag.String("provider", "gofish", "bmclib provider to use (gofish, dell, supermicro, lenovo, openbmc, asrockrack), or \"all\" to try every compatible one, IPMI included")
	basicAuth := flag.Bool("basic-auth", false, "use HTTP basic auth instead of a Redfish session")
	timeout := flag.Duration("timeout", 3*time.Minute, "overall timeout")
	debug := flag.Bool("debug", false, "log bmclib debug output to stderr")
	flag.Parse()

	if *password == "" {
		*password = os.Getenv("BMC_PASSWORD")
	}
	if *host == "" || *user == "" || *password == "" {
		flag.Usage()
		return fmt.Errorf("-host, -user and a password (-password or BMC_PASSWORD) are required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	l := logrus.New()
	l.Out = os.Stderr
	l.Level = logrus.WarnLevel
	if *debug {
		l.Level = logrus.DebugLevel
	}

	cl := bmclib.NewClient(*host, *user, *password,
		bmclib.WithLogger(logrusr.New(l)),
		bmclib.WithRedfishPort(strconv.Itoa(*port)),
		bmclib.WithRedfishUseBasicAuth(*basicAuth),
	)
	if *provider != "all" {
		cl.Registry.Drivers = cl.Registry.For(*provider)
		if len(cl.Registry.Drivers) == 0 {
			return fmt.Errorf("unknown provider %q", *provider)
		}
	}
	cl.Registry.Drivers = cl.Registry.FilterForCompatible(ctx)
	if len(cl.Registry.Drivers) == 0 {
		return fmt.Errorf("no bmclib provider is compatible with %s (wrong address, port or credentials?)", *host)
	}

	if err := cl.Open(ctx); err != nil {
		return fmt.Errorf("bmc login: %w", err)
	}
	defer func() { _ = cl.Close(context.Background()) }()

	device, err := cl.Inventory(ctx)
	if err != nil {
		return fmt.Errorf("inventory: %w", err)
	}

	out, err := json.MarshalIndent(device, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))

	md := cl.GetMetadata()
	fmt.Fprintf(os.Stderr, "provider=%s cpus=%d gpus=%d nics=%d drives=%d dimms=%d\n",
		md.SuccessfulProvider, len(device.CPUs), len(device.GPUs), len(device.NICs), len(device.Drives), len(device.Memory))
	return nil
}
