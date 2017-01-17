package ipamdriver

import (
	"encoding/json"
	"errors"
	"crypto/rand"
	"path/filepath"
	"net"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/ipam"
	"golang.org/x/net/context"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"oam-docker-ipam/db"
	dc "oam-docker-ipam/dhcp4client"
	"github.com/d2g/dhcp4"
)

const (
	network_key_prefix = "/talkingdata/containers"
)

var imap = make(map[string]string)

type Config struct {
	Ipnet string
	Mask  string
}

func StartServer() {
	loadIpMap()
	d := &MyIPAMHandler{}
	h := ipam.NewHandler(d)
	h.ServeUnix("root", "talkingdata")
}

func ReleaseIP(ip_net, ip string) error {
	var err error

	macaddr := getMacAddr(ip) //Get the mac address from ip
	m, err := net.ParseMAC(macaddr)
	if err != nil {
		log.Errorf("MAC Error:%v\n", err)
	}
	c, err := dc.NewInetSock(dc.SetLocalAddr(net.UDPAddr{IP: net.IPv4(0, 0, 0, 0), Port: 68}),
		dc.SetRemoteAddr(net.UDPAddr{IP: net.ParseIP(dc.GetDHCPAddr()), Port: 67}))
	if err != nil {
		log.Error("Client Conection Generation:" + err.Error())
	}
        log.Debugf("DHCP Addr: %s", dc.GetDHCPAddr())

	exampleClient, err := dc.New(dc.HardwareAddr(m), dc.Connection(c))
	if err != nil {
		log.Fatalf("Error:%v\n", err)
	}

	//create ack packet
	messageid := make([]byte, 4)
	if _, err := rand.Read(messageid); err != nil {
		panic(err)
	}

	acknowledgementpacket := dhcp4.NewPacket(dhcp4.BootRequest)
	acknowledgementpacket.SetCHAddr(m)
	acknowledgementpacket.SetXId(messageid)
	acknowledgementpacket.SetYIAddr(net.ParseIP(ip))

	acknowledgementpacket.AddOption(dhcp4.OptionDHCPMessageType, []byte{byte(dhcp4.Release)})
	acknowledgementpacket.AddOption(dhcp4.OptionServerIdentifier, net.ParseIP(dc.GetDHCPAddr()))


	err = exampleClient.Release(acknowledgementpacket)
	if err != nil {
		networkError, ok := err.(*net.OpError)
		if ok && networkError.Timeout() {
			log.Info("Release lease Failed! Because it didn't find the DHCP server very Strange")
			log.Errorf("Error" + err.Error())
		}
		log.Fatalf("Error:%v\n", err)
	} else {
		delete(imap, ip)
		log.Debugf(imap)
		log.Info("Relase lease successfully!\n")
	}
	exampleClient.Close()

	return nil
}

func AllocateIP(ip_net, ip string, macaddr string) (string, string, error) {
	var err error

	m, err := net.ParseMAC(macaddr)
	if err != nil {
		log.Printf("MAC Error:%v\n", err)
	}
	//Create a connection to use
	//We need to set the connection ports to 1068 and 1067 so we don't need root access
	//c, err := NewInetSock(SetLocalAddr(net.UDPAddr{IP: net.IPv4(0, 0, 0, 0), Port: 68}), SetRemoteAddr(net.UDPAddr{IP: net.IPv4bcast, Port: 67}))
	c, err := dc.NewInetSock(dc.SetLocalAddr(net.UDPAddr{IP: net.IPv4(0, 0, 0, 0), Port: 68}),
		dc.SetRemoteAddr(net.UDPAddr{IP: net.ParseIP(dc.GetDHCPAddr()), Port: 67}))
	if err != nil {
		log.Error("Client Conection Generation:" + err.Error())
	}
        log.Debugf("DHCP Addr: %s", dc.GetDHCPAddr())

	if len(ip) != 0 {
		dc.SetRequestedIP(ip)
		log.Debugf("Requested IP:%s", ip)
	}
	exampleClient, err := dc.New(dc.HardwareAddr(m), dc.Connection(c))
	if err != nil {
		log.Fatalf("Error:%v\n", err)
	}

	success, acknowledgementpacket, err := exampleClient.Request()

	log.Infof("Success:%v\n", success)
	log.Infof("Packet:%v\n", acknowledgementpacket)

	if err != nil {
		networkError, ok := err.(*net.OpError)
		if ok && networkError.Timeout() {
			log.Error("Can not find a DHCP Server ...")
		}
		log.Fatalf("Error:%v\n", err)
	}

	exampleClient.Close()
	if !success {
		log.Error("We didn't sucessfully get a DHCP Lease?")
	} else {
		log.Debugf("IP Received YIAddr:%v\n", acknowledgementpacket.YIAddr().String())
		log.Debugf("IP Received CIAddr:%v\n", acknowledgementpacket.CIAddr().String())
		log.Debugf("IP Received GIAddr:%v\n", acknowledgementpacket.GIAddr().String())
		log.Debugf("IP Received Options:%v\n", acknowledgementpacket.Options())
		acknowledgementOptions := acknowledgementpacket.ParseOptions()
                var mask = net.IPMask(acknowledgementOptions[dhcp4.OptionSubnetMask])
                masknum,_ := mask.Size()
		yiaddr := acknowledgementpacket.YIAddr().String()
		imap[yiaddr] = macaddr
		log.Debugf(imap)
		return yiaddr, strconv.Itoa(masknum), nil
	}

        return "", "", errors.New("Can not allocate ip")
}



func checkIPAssigned(ip_net, ip string) bool {
	if exist := db.IsKeyExist(filepath.Join(network_key_prefix, ip_net, "assigned", ip)); exist {
		return true
	}
	return false
}


func GetConfig(ip_net string) (*Config, error) {
	config, err := db.GetKey(filepath.Join(network_key_prefix, ip_net, "config"))
	if err == nil {
		log.Debugf("GetConfig %s from network %s", config, ip_net)
	}
	conf := &Config{}
	json.Unmarshal([]byte(config), conf)
	return conf, err
}

func getMacAddr(ip string) string {
	if mac,found := imap[ip];found {
		log.Debugf("Found IP:MAC %s:%s",ip,mac)
		return mac
	}else {
		log.Debug("IP not found")
		return ""
	}
}

func ListContainers(socketurl string) ([]types.Container, error) {
	var c *client.Client
	var err error
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	c,err = client.NewClient(socketurl, "", nil, defaultHeaders)
	if err != nil {
		log.Fatalf("Create Docker Client error", err)
		return nil, err
	}

	// List containers
	opts := types.ContainerListOptions{}
	ctx, cancel := context.WithTimeout(context.Background(), 20000*time.Millisecond)
	defer cancel()
	containers, err := c.ContainerList(ctx, opts)
	if err != nil {
		log.Fatal("List Container error", err)
		return nil, err
	}
	return containers, err
}

func loadIpMap() {
	containers, _ := ListContainers("unix:///var/run/docker.sock")
	for _,container := range containers {
		networks := container.NetworkSettings.Networks
		for _,n := range networks {
			imap[n.IPAddress] = n.MacAddress
			log.Debugf("Found IP: MacAddress %s:%s", n.IPAddress,n.MacAddress)
		}
	}
	log.Debugf(imap)
}