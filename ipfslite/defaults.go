package ipfslite

import (
	"net"

	"github.com/pkg/errors"
)

var noAnnounce = []string{
	"/ip4/10.0.0.0/ipcidr/8",
	"/ip4/100.64.0.0/ipcidr/10",
	"/ip4/169.254.0.0/ipcidr/16",
	"/ip4/172.16.0.0/ipcidr/12",
	"/ip4/192.0.0.0/ipcidr/24",
	"/ip4/192.0.0.0/ipcidr/29",
	"/ip4/192.0.0.8/ipcidr/32",
	"/ip4/192.0.0.170/ipcidr/32",
	"/ip4/192.0.0.171/ipcidr/32",
	"/ip4/192.0.2.0/ipcidr/24",
	"/ip4/192.168.0.0/ipcidr/16",
	"/ip4/198.18.0.0/ipcidr/15",
	"/ip4/198.51.100.0/ipcidr/24",
	"/ip4/203.0.113.0/ipcidr/24",
	"/ip4/240.0.0.0/ipcidr/4",
}

var addrFilters = []string{
	"10.0.0.0/8",
	"100.64.0.0/10",
	"169.254.0.0/16",
	"172.16.0.0/12",
	"192.0.0.0/24",
	"192.0.0.0/29",
	"192.0.0.8/32",
	"192.0.0.170/32",
	"192.0.0.171/32",
	"192.0.2.0/24",
	"192.168.0.0/16",
	"198.18.0.0/15",
	"198.51.100.0/24",
	"203.0.113.0/24",
	"240.0.0.0/4",
}

var addrFilterIPs []*net.IPNet

func init() {
	addrFilterIPs = make([]*net.IPNet, len(addrFilters))
	for i, cidr := range addrFilters {
		net, err := stringToIPNet(cidr)
		if err != nil {
			panic(errors.Wrap(err, "error getting stringToIPnet"))
		}
		addrFilterIPs[i] = net
	}
}

func stringToIPNet(str string) (*net.IPNet, error) {
	_, n, err := net.ParseCIDR(str)
	return n, err
}
