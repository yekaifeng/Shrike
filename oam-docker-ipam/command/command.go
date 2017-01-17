package command

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"

	"oam-docker-ipam/dhcp4client"
	"oam-docker-ipam/ipamdriver"

)

var (
	debug bool
)

func initialize_log() {
	log.SetOutput(os.Stderr)
	if debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}

func NewServerCommand() cli.Command {
	return cli.Command{
		Name:   "server",
		Usage:  "start the TalkingData IPAM plugin",
		Action: startServerAction,
	}
}

func startServerAction(c *cli.Context) {
	debug = c.GlobalBool("debug")
	dhcp4client.SetDHCPAddr(c.GlobalString("dhcp-server"))
	dhcp4client.SetListenAddr(c.GlobalString("listen-addr"))
	initialize_log()
	ipamdriver.StartServer()
}


