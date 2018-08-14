package cmd

import (
	"net"
	"os"
	log "github.com/sirupsen/logrus"
)

func FindBindAddress() (string) {
	ifaces, err := net.Interfaces()
	if err != nil {
		log.WithError(err).Fatal("Couldn't read interface list")
		os.Exit(-1)
	}
	candidates := []net.IP{}
	_, loopback, _ := net.ParseCIDR("127.0.0.1/8")
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			log.WithError(err).Warn("Couldn't read interface address")
			continue
		}
		for _, addr := range addrs {
			str := addr.String()
			ip, _, err := net.ParseCIDR(str)
			if err == nil && ip != nil && ip.To4() != nil && !loopback.Contains(ip) {
				candidates = append(candidates, ip)
			}
		}
	}

	_, privateA, _ := net.ParseCIDR("10.0.0.0/24")
	_, privateB, _ := net.ParseCIDR("172.16.0.0/20")
	_, privateC, _ := net.ParseCIDR("192.168.0.0/16")
	if len(candidates) == 0 {
		if err != nil {
			log.WithError(err).Fatal("No non-loopback IPv4 addresses found")
			os.Exit(-1)
		}
	}
	log.WithFields(log.Fields{
		"candidates": candidates,
		"app": "microkube",
		"component": "findIP",
	}).Debug("Beginning cadidate selection")
	for _, item := range candidates {
		if privateA.Contains(item) ||  privateB.Contains(item) ||  privateC.Contains(item) {
			return item.String()
		}
	}
	log.WithFields(log.Fields{
		"candidates": candidates,
		"app": "microkube",
		"component": "findIP",
	}).Info("Didn't find interface with local IPv4, falling back to a public one")
	return candidates[0].String()
}
