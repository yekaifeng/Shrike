package dhcp4client

import (
	"log"
	"net"
	"testing"
)

/*
 * Example Client
 */
func Test_ExampleClient(test *testing.T) {
	var err error

	m, err := net.ParseMAC("08-00-27-00-A8-E8")
	if err != nil {
		log.Printf("MAC Error:%v\n", err)
	}

	//Create a connection to use
	//We need to set the connection ports to 1068 and 1067 so we don't need root access
	//c, err := NewInetSock(SetLocalAddr(net.UDPAddr{IP: net.IPv4(0, 0, 0, 0), Port: 68}), SetRemoteAddr(net.UDPAddr{IP: net.IPv4bcast, Port: 67}))
	c, err := NewInetSock(SetLocalAddr(net.UDPAddr{IP: net.IPv4(0, 0, 0, 0), Port: 68}), SetRemoteAddr(net.UDPAddr{IP: net.IPv4(10,100,144,1), Port: 67}))
	if err != nil {
		test.Error("Client Conection Generation:" + err.Error())
	}

	exampleClient, err := New(HardwareAddr(m), Connection(c))
	if err != nil {
		test.Fatalf("Error:%v\n", err)
	}

	success, acknowledgementpacket, err := exampleClient.Request()

	test.Logf("Success:%v\n", success)
	test.Logf("Packet:%v\n", acknowledgementpacket)

	if err != nil {
		networkError, ok := err.(*net.OpError)
		if ok && networkError.Timeout() {
			test.Log("Test Skipping as it didn't find a DHCP Server")
			test.SkipNow()
		}
		test.Fatalf("Error:%v\n", err)
	}

	if !success {
		test.Error("We didn't sucessfully get a DHCP Lease?")
	} else {
		log.Printf("IP Received YIAddr:%v\n", acknowledgementpacket.YIAddr().String())
		log.Printf("IP Received CIAddr:%v\n", acknowledgementpacket.CIAddr().String())
		log.Printf("IP Received GIAddr:%v\n", acknowledgementpacket.GIAddr().String())
		log.Printf("IP Received Options:%v\n", acknowledgementpacket.Options())
	}

	test.Log("Start Release Lease")
	err = exampleClient.Release(acknowledgementpacket)
	if err != nil {
		networkError, ok := err.(*net.OpError)
		if ok && networkError.Timeout() {
			test.Log("Release lease Failed! Because it didn't find the DHCP server very Strange")
			test.Errorf("Error" + err.Error())
		}
		test.Fatalf("Error:%v\n", err)
	} else {
		test.Log("Relase lease successfully!\n")
	}
	/*
	test.Log("Start Renewing Lease")
	success, acknowledgementpacket, err = exampleClient.Renew(acknowledgementpacket)
	if err != nil {
		networkError, ok := err.(*net.OpError)
		if ok && networkError.Timeout() {
			test.Log("Renewal Failed! Because it didn't find the DHCP server very Strange")
			test.Errorf("Error" + err.Error())
		}
		test.Fatalf("Error:%v\n", err)
	}

	if !success {
		test.Error("We didn't sucessfully Renew a DHCP Lease?")
	} else {
		log.Printf("IP Received:%v\n", acknowledgementpacket.YIAddr().String())
	}
	*/


}
