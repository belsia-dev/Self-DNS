package resolver

import (
	"bufio"
	"net"
	"os"
	"os/exec"
	goruntime "runtime"
	"strings"
)

func (r *Resolver) RefreshSystemDNS() {
	s := discoverSystemDNS()
	r.mu.Lock()
	r.systemDNS = s
	r.mu.Unlock()
}

func (r *Resolver) RefreshNetworkDNS() {
	n := discoverNetworkDNS()
	r.mu.Lock()
	r.networkDNS = n
	r.mu.Unlock()
}

func (r *Resolver) SystemDNS() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]string(nil), r.systemDNS...)
}

func (r *Resolver) NetworkDNS() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]string(nil), r.networkDNS...)
}

func discoverSystemDNS() []string {
	if goruntime.GOOS == "windows" {
		return []string{"1.1.1.1:53", "8.8.8.8:53"}
	}
	return parseResolvConf("/etc/resolv.conf", true)
}

func discoverNetworkDNS() []string {
	switch goruntime.GOOS {
	case "darwin":
		return macNetworkDNS()
	case "linux":
		return linuxNetworkDNS()
	case "windows":
		return windowsNetworkDNS()
	}
	return nil
}

func macNetworkDNS() []string {
	var result []string
	for _, iface := range []string{"en0", "en1", "en2", "en3", "en4"} {
		out, err := exec.Command("ipconfig", "getoption", iface, "domain_name_server").Output()
		if err != nil {
			continue
		}
		for _, raw := range strings.Fields(string(out)) {
			ip := strings.Trim(raw, "{},")
			if net.ParseIP(ip) == nil || isLoopbackIP(ip) {
				continue
			}
			addr := net.JoinHostPort(ip, "53")
			if !strContains(result, addr) {
				result = append(result, addr)
			}
		}
	}
	if len(result) > 0 {
		return result
	}
	return macScutilDNS()
}

func macScutilDNS() []string {
	out, err := exec.Command("scutil", "--dns").Output()
	if err != nil {
		return nil
	}
	var result []string
	inScoped := false
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "for scoped queries") {
			inScoped = true
		}
		if inScoped && strings.HasPrefix(line, "nameserver[") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			ip := strings.TrimSpace(parts[1])
			if net.ParseIP(ip) == nil || isLoopbackIP(ip) {
				continue
			}
			addr := net.JoinHostPort(ip, "53")
			if !strContains(result, addr) {
				result = append(result, addr)
			}
		}
		if inScoped && line == "" {
			if len(result) > 0 {
				break
			}
		}
	}
	return result
}

func linuxNetworkDNS() []string {
	paths := []string{
		"/run/systemd/resolve/resolv.conf",
		"/run/resolvconf/resolv.conf",
		"/var/run/resolvconf/resolv.conf",
	}
	for _, p := range paths {
		servers := parseResolvConf(p, false)
		if len(servers) > 0 {
			return servers
		}
	}
	if f, err := os.Open("/var/lib/dhcp/dhclient.leases"); err == nil {
		defer f.Close()
		var result []string
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if !strings.HasPrefix(line, "option domain-name-servers") {
				continue
			}
			parts := strings.SplitN(line, " ", 3)
			if len(parts) < 3 {
				continue
			}
			for _, raw := range strings.Split(strings.TrimSuffix(parts[2], ";"), ",") {
				ip := strings.TrimSpace(raw)
				if net.ParseIP(ip) == nil || isLoopbackIP(ip) {
					continue
				}
				addr := net.JoinHostPort(ip, "53")
				if !strContains(result, addr) {
					result = append(result, addr)
				}
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return nil
}

func windowsNetworkDNS() []string {
	out, err := exec.Command("powershell", "-Command",
		"Get-DnsClientServerAddress -AddressFamily IPv4 | Select-Object -ExpandProperty ServerAddresses").Output()
	if err != nil {
		return nil
	}
	var result []string
	for _, line := range strings.Split(string(out), "\n") {
		ip := strings.TrimSpace(line)
		if net.ParseIP(ip) == nil || isLoopbackIP(ip) {
			continue
		}
		addr := net.JoinHostPort(ip, "53")
		if !strContains(result, addr) {
			result = append(result, addr)
		}
	}
	return result
}

func parseResolvConf(path string, skipLoopback bool) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var servers []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "nameserver ") {
			continue
		}
		addr := strings.TrimSpace(strings.TrimPrefix(line, "nameserver "))
		if skipLoopback && (addr == "127.0.0.1" || addr == "::1" || addr == "127.0.0.53") {
			continue
		}
		if !skipLoopback && isLoopbackIP(addr) {
			continue
		}
		servers = append(servers, net.JoinHostPort(addr, "53"))
	}
	return servers
}

func discoverUnixDNS() []string {
	return parseResolvConf("/etc/resolv.conf", true)
}

func isLoopbackIP(addr string) bool {
	ip := net.ParseIP(addr)
	return ip != nil && ip.IsLoopback()
}

func strContains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
