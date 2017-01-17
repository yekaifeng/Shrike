package main

import (
	"oam-docker-ipam/command"
	"os"

	"github.com/codegangsta/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "oam-docker-ipam"
	app.Version = "2.0.0"
	app.Author = "chao.ma,kenneth.ye"
	app.Usage = "TalkingData network plugin with remote IPAM by DHCP"
	app.Flags = []cli.Flag{
	cli.StringFlag{Name: "dhcp-server", Value: "0.0.0.0", Usage: "remote ip address of dhcp server. [$DHCP_SERVER]"},
		cli.StringFlag{Name: "listen-addr", Value: "0.0.0.0", Usage: "local ip address for dhcp client. [$LISTEN_ADDR]"},
		cli.BoolFlag{Name: "debug", Usage: "debug mode [$DEBUG]"},
	}
	app.Commands = []cli.Command{
		command.NewServerCommand(),
	}
	app.Run(os.Args)
}
