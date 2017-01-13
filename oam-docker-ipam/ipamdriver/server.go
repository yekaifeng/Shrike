package ipamdriver

import (
	"encoding/json"
	"errors"
	"crypto/rand"
	"path/filepath"
	"net"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/ipam"

	"oam-docker-ipam/db"
	dc "oam-docker-ipam/dhcp4client"
	"github.com/d2g/dhcp4"
)

const (
	network_key_prefix = "/talkingdata/containers"
)

type Config struct {
	Ipnet string
	Mask  string
}

func StartServer() {
	d := &MyIPAMHandler{}
	h := ipam.NewHandler(d)
	h.ServeUnix("root", "talkingdata")
}

func ReleaseIP(ip_net, ip string) error {
	var err error

	m, err := net.ParseMAC("08-00-27-00-A8-E8") //bogus mac addr
	if err != nil {
		log.Printf("MAC Error:%v\n", err)
	}
	c, err := dc.NewInetSock(dc.SetLocalAddr(net.UDPAddr{IP: net.IPv4(0, 0, 0, 0), Port: 68}),
		dc.SetRemoteAddr(net.UDPAddr{IP: net.ParseIP(dc.GetDHCPAddr()), Port: 67}))
	if err != nil {
		log.Error("Client Conection Generation:" + err.Error())
	}

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
	acknowledgementpacket.SetCIAddr(net.ParseIP(ip))

	acknowledgementpacket.AddOption(dhcp4.OptionDHCPMessageType, []byte{byte(dhcp4.Release)})
	acknowledgementpacket.AddOption(dhcp4.OptionServerIdentifier, []byte(ip_net))


	err = exampleClient.Release(acknowledgementpacket)
	if err != nil {
		networkError, ok := err.(*net.OpError)
		if ok && networkError.Timeout() {
			log.Info("Release lease Failed! Because it didn't find the DHCP server very Strange")
			log.Errorf("Error" + err.Error())
		}
		log.Fatalf("Error:%v\n", err)
	} else {
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


	if len(ip) != 0 {
		dc.SetRequestedIP(ip)
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
		log.Printf("IP Received YIAddr:%v\n", acknowledgementpacket.YIAddr().String())
		log.Printf("IP Received CIAddr:%v\n", acknowledgementpacket.CIAddr().String())
		log.Printf("IP Received GIAddr:%v\n", acknowledgementpacket.GIAddr().String())
		log.Printf("IP Received Options:%v\n", acknowledgementpacket.Options())
		acknowledgementOptions := acknowledgementpacket.ParseOptions()
                mask := string(acknowledgementOptions[dhcp4.OptionSubnetMask][:])
		return acknowledgementpacket.YIAddr().String(), mask, nil
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
