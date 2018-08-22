/*
 * Copyright 2018 The microkube authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	log "github.com/sirupsen/logrus"
	"net"
)

// CalculateIPRanges takes the pod and service range as strings and calculates the required networks
// for Microkube from it
func CalculateIPRanges(podRange, serviceRange string) (pod, service, cluster *net.IPNet,
	bind, firstSVC net.IP, errRet error) {
	// Parse commandline arguments
	podRangeIP, podRangeNet, err := net.ParseCIDR(podRange)
	if err != nil {
		log.WithFields(log.Fields{
			"range": podRange,
		}).WithError(err).Warn("Couldn't parse pod CIDR range")
		return nil, nil, nil, nil, nil, err
	}
	serviceRangeIP, serviceRangeNet, err := net.ParseCIDR(serviceRange)
	if err != nil {
		log.WithFields(log.Fields{
			"range": podRange,
		}).WithError(err).Warn("Couldn't parse service CIDR range")
		return nil, nil, nil, nil, nil, err
	}

	// Find address to bind to
	bindAddr := FindBindAddress()

	// To combine pod and service range to form the cluster range, find first diverging bit
	baseOffset := 0
	serviceBelowPod := false
	for idx, octet := range serviceRangeNet.IP {
		if podRangeNet.IP[idx] != octet {
			// This octet diverges -> find bit
			baseOffset = idx * 8
			for mask := byte(0x80); mask > 0; mask /= 2 {
				baseOffset++
				if (podRangeNet.IP[idx] & mask) != (octet & mask) {
					// Found it
					serviceBelowPod = octet < podRangeNet.IP[idx]
					break
				}
			}
			baseOffset--
		}
	}
	clusterIPRange := &net.IPNet{
		IP: podRangeIP,
	}
	if serviceBelowPod {
		clusterIPRange.IP = serviceRangeIP
	}
	clusterIPRange.Mask = net.CIDRMask(baseOffset, 32)
	log.WithFields(log.Fields{
		"podRange":     podRangeNet.String(),
		"serviceRange": serviceRangeNet.String(),
		"clusterRange": clusterIPRange.String(),
		"hostIP":       bindAddr,
	}).Info("IP ranges calculated")

	return podRangeNet, serviceRangeNet, clusterIPRange, bindAddr, serviceRangeIP, nil
}

// FindBindAddress tries to find a private IPv4 address from some local interface that can be used to bind services to it
func FindBindAddress() net.IP {
	ifaces, err := net.Interfaces()
	if err != nil {
		log.WithError(err).Fatal("Couldn't read interface list")
	}

	var candidates []net.IP
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

	if len(candidates) == 0 {
		log.WithError(err).Fatal("No non-loopback IPv4 addresses found")
	}

	return findBindAddress(candidates)
}

// findBindAddress tries to find a private IPv4 address from a list of addresses provided
func findBindAddress(candidates []net.IP) net.IP {
	_, privateA, _ := net.ParseCIDR("10.0.0.0/8")
	_, privateB, _ := net.ParseCIDR("172.16.0.0/12")
	_, privateC, _ := net.ParseCIDR("192.168.0.0/16")
	log.WithFields(log.Fields{
		"candidates": candidates,
		"app":        "microkube",
		"component":  "findIP",
	}).Debug("Beginning cadidate selection")
	for _, item := range candidates {
		if privateA.Contains(item) || privateB.Contains(item) || privateC.Contains(item) {
			return item
		}
	}
	log.WithFields(log.Fields{
		"candidates": candidates,
		"app":        "microkube",
		"component":  "findIP",
	}).Info("Didn't find interface with local IPv4, falling back to a public one")
	return candidates[0]
}
