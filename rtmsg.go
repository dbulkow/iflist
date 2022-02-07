// MIT License

// Copyright (c) 2022 David Bulkow

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"
)

type Rtmsg struct {
	Family   byte
	DestLen  byte
	SrcLen   byte
	TOS      byte
	Table    byte
	Protocol byte
	Scope    byte
	Type     byte
	Flags    int
}

var FamilyStrings = map[byte]string{
	syscall.AF_UNSPEC: "AF_UNSPEC",
	syscall.AF_INET:   "AF_INET",
	syscall.AF_INET6:  "AF_INET6",
}

var TableStrings = map[byte]string{
	syscall.RT_CLASS_UNSPEC:  "unspec",
	syscall.RT_CLASS_DEFAULT: "default",
	syscall.RT_CLASS_LOCAL:   "local",
	syscall.RT_CLASS_MAIN:    "main",
}

var ProtoStrings = map[byte]string{
	syscall.RTPROT_UNSPEC:   "RTPROT_UNSPEC",
	syscall.RTPROT_REDIRECT: "RTPROT_REDIRECT",
	syscall.RTPROT_KERNEL:   "RTPROT_KERNEL",
	syscall.RTPROT_BOOT:     "RTPROT_BOOT",
	syscall.RTPROT_STATIC:   "RTPROT_STATIC",
}

var TypeStrings = map[byte]string{
	syscall.RTN_UNSPEC:      "RTN_UNSPEC",
	syscall.RTN_UNICAST:     "RTN_UNICAST",
	syscall.RTN_LOCAL:       "RTN_LOCAL",
	syscall.RTN_BROADCAST:   "RTN_BROADCAST",
	syscall.RTN_ANYCAST:     "RTN_ANYCAST",
	syscall.RTN_MULTICAST:   "RTN_MULTICAST",
	syscall.RTN_BLACKHOLE:   "RTN_BLACKHOLE",
	syscall.RTN_UNREACHABLE: "RTN_UNREACHABLE",
	syscall.RTN_PROHIBIT:    "RTN_PROHIBIT",
	syscall.RTN_THROW:       "RTN_THROW",
	syscall.RTN_NAT:         "RTN_NAT",
	syscall.RTN_XRESOLVE:    "RTN_XRESOLVE",
}

var ScopeStrings = map[byte]string{
	syscall.RT_SCOPE_UNIVERSE: "RT_SCOPE_UNIVERSE",
	syscall.RT_SCOPE_SITE:     "RT_SCOPE_SITE",
	syscall.RT_SCOPE_LINK:     "RT_SCOPE_LINK",
	syscall.RT_SCOPE_HOST:     "RT_SCOPE_HOST",
	syscall.RT_SCOPE_NOWHERE:  "RT_SCOPE_NOWHERE",
}

var ifla = []string{
	syscall.RTA_UNSPEC:    "RTA_UNSPEC",
	syscall.RTA_DST:       "RTA_DST",
	syscall.RTA_SRC:       "RTA_SRC",
	syscall.RTA_IIF:       "RTA_IIF",
	syscall.RTA_OIF:       "RTA_OIF",
	syscall.RTA_GATEWAY:   "RTA_GATEWAY",
	syscall.RTA_PRIORITY:  "RTA_PRIORITY",
	syscall.RTA_PREFSRC:   "RTA_PREFSRC",
	syscall.RTA_METRICS:   "RTA_METRICS",
	syscall.RTA_MULTIPATH: "RTA_MULTIPATH",
	syscall.RTA_FLOW:      "RTA_FLOW",
	syscall.RTA_CACHEINFO: "RTA_CACHEINFO",
	syscall.RTA_TABLE:     "RTA_TABLE",
	20:                    "unknown attribute",
}

func getint(buf []byte) int {
	return (int(buf[3]) << 24) | (int(buf[2]) << 16) | (int(buf[1]) << 8) | int(buf[0])
}

func UnmarshalRtmsg(buf []byte) *Rtmsg {
	if len(buf) < 8 {
		fmt.Println("buffer too short for rtmsg")
		return nil
	}
	r := new(Rtmsg)
	r.Family = buf[0]
	r.DestLen = buf[1]
	r.SrcLen = buf[2]
	r.TOS = buf[3]
	r.Table = buf[4]
	r.Protocol = buf[5]
	r.Scope = buf[6]
	r.Type = buf[7]

	if len(buf[8:]) < 4 {
		fmt.Println("buffer too short for rtmsg.Flags")
		return nil
	}
	r.Flags = getint(buf[8:])

	return r
}

func (r *Rtmsg) FlagStr() string {
	str := ""
	switch {
	case (r.Flags & syscall.RTM_F_CLONED) == syscall.RTM_F_CLONED:
		str = str + "RTM_F_CLONED"
	case (r.Flags & syscall.RTM_F_EQUALIZE) == syscall.RTM_F_EQUALIZE:
		str = str + "RTM_F_EQUALIZE"
	case (r.Flags & syscall.RTM_F_NOTIFY) == syscall.RTM_F_NOTIFY:
		str = str + "RTM_F_NOTIFY"
	case (r.Flags & syscall.RTM_F_PREFIX) == syscall.RTM_F_PREFIX:
		str = str + "RTM_F_PREFIX"
	}
	return str
}

type Route struct {
	Dest            string
	Source          string
	PreferredSource string
	Gateway         string
	InputInterface  int
	OutputInterface int
	Priority        int
	Metrics         int
	Table           string
	CacheInfo       []byte
}

func ReadRoutes() ([]Route, error) {
	tab, err := syscall.NetlinkRIB(syscall.RTM_GETROUTE, syscall.AF_UNSPEC)
	if err != nil {
		return nil, err
	}
	msgs, err := syscall.ParseNetlinkMessage(tab)
	if err != nil {
		return nil, err
	}
	routes := make([]Route, 0)
loop:
	for _, m := range msgs {
		switch m.Header.Type {
		case syscall.NLMSG_DONE:
			break loop
		case syscall.RTM_NEWROUTE:
			rtmsg := UnmarshalRtmsg(m.Data)
			// we don't have an ipv6 environment, so ignore anything not ipv4
			if rtmsg.Family != syscall.AF_INET {
				continue
			}
			attr, err := syscall.ParseNetlinkRouteAttr(&m)
			if err != nil {
				return nil, os.NewSyscallError("ParseNetlinkRouteAttr", err)
			}
			rt := Route{}
			for _, a := range attr {
				switch a.Attr.Type {
				case syscall.RTA_DST:
					rt.Dest = net.IPv4(a.Value[0], a.Value[1], a.Value[2], a.Value[3]).String()
				case syscall.RTA_SRC:
					rt.Source = net.IPv4(a.Value[0], a.Value[1], a.Value[2], a.Value[3]).String()
				case syscall.RTA_PREFSRC:
					rt.PreferredSource = net.IPv4(a.Value[0], a.Value[1], a.Value[2], a.Value[3]).String()
				case syscall.RTA_GATEWAY:
					rt.Gateway = net.IPv4(a.Value[0], a.Value[1], a.Value[2], a.Value[3]).String()
				case syscall.RTA_IIF:
					rt.InputInterface = getint(a.Value)
				case syscall.RTA_OIF:
					rt.OutputInterface = getint(a.Value)
				case syscall.RTA_PRIORITY:
					rt.Priority = getint(a.Value)
				case syscall.RTA_METRICS:
					rt.Metrics = getint(a.Value)
				case syscall.RTA_TABLE:
					rt.Table = TableStrings[byte(getint(a.Value))]
				case syscall.RTA_CACHEINFO:
					rt.CacheInfo = a.Value
				default:
					fmt.Printf("%-15s %v\n", ifla[a.Attr.Type], a.Value)
				}
			}
			routes = append(routes, rt)
		default:
			fmt.Printf("unknown type %x\n", m.Header.Type)
		}
	}

	return routes, nil
}

func DefaultRoute(routes []Route) (int, error) {
	for _, rt := range routes {
		if rt.Table == "main" && rt.Dest == "" && rt.Gateway != "" {
			return rt.OutputInterface, nil
		}
	}
	return 0, errors.New("no default route")
}
