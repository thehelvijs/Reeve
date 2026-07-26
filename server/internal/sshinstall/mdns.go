package sshinstall

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

// A .local name is answered by multicast on the LAN, never by a DNS server, and
// Go's resolver in a static binary never consults NSS (so no avahi, no mdns4
// lookup). Without this, "Install over SSH" cannot reach a host by the name its
// owner knows it by. The query needs the container on the host network:
// multicast out of a bridge network stops at the bridge.
// A responder must not repeat a record within a second, so a probe followed by
// an install would get silence on the second query; retryDelay outruns that.
const (
	mdnsGroup      = "224.0.0.251:5353"
	mdnsTimeout    = 4 * time.Second
	mdnsRetryDelay = 1200 * time.Millisecond
)

// lookupMDNS returns the first IPv4 address the LAN claims for host. Listening
// on the group address rather than an ephemeral port is what makes this work
// next to an avahi-daemon already bound to 5353, and it is where responders send
// the answer.
func lookupMDNS(ctx context.Context, host string) (string, error) {
	query, err := mdnsQuery(host)
	if err != nil {
		return "", err
	}
	group, err := net.ResolveUDPAddr("udp4", mdnsGroup)
	if err != nil {
		return "", err
	}
	conn, err := net.ListenMulticastUDP("udp4", nil, group)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	deadline := time.Now().Add(mdnsTimeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	buf := make([]byte, 1500)
	for time.Now().Before(deadline) {
		if _, err := conn.WriteTo(query, group); err != nil {
			return "", err
		}
		until := time.Now().Add(mdnsRetryDelay)
		if until.After(deadline) {
			until = deadline
		}
		conn.SetReadDeadline(until)
		for {
			n, _, err := conn.ReadFrom(buf)
			if err != nil {
				break
			}
			// The group carries every host's chatter; anything else is not a miss.
			if ip, ok := mdnsAnswer(buf[:n], host); ok {
				return ip, nil
			}
		}
	}
	return "", fmt.Errorf("no mDNS answer for %s", host)
}

// mdnsQuery builds a one-shot A query. mDNS ignores the transaction ID.
func mdnsQuery(host string) ([]byte, error) {
	name, err := dnsmessage.NewName(strings.TrimSuffix(host, ".") + ".")
	if err != nil {
		return nil, err
	}
	b := dnsmessage.NewBuilder(nil, dnsmessage.Header{})
	if err := b.StartQuestions(); err != nil {
		return nil, err
	}
	q := dnsmessage.Question{Name: name, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET}
	if err := b.Question(q); err != nil {
		return nil, err
	}
	return b.Finish()
}

// mdnsAnswer picks the A record for host out of a response. Replies for other
// names arrive constantly on this group, so a miss is normal, not an error.
func mdnsAnswer(msg []byte, host string) (string, bool) {
	want := strings.ToLower(strings.TrimSuffix(host, ".")) + "."
	var p dnsmessage.Parser
	if _, err := p.Start(msg); err != nil {
		return "", false
	}
	if err := p.SkipAllQuestions(); err != nil {
		return "", false
	}
	for {
		h, err := p.AnswerHeader()
		if errors.Is(err, dnsmessage.ErrSectionDone) || err != nil {
			return "", false
		}
		if h.Type != dnsmessage.TypeA || !strings.EqualFold(h.Name.String(), want) {
			if err := p.SkipAnswer(); err != nil {
				return "", false
			}
			continue
		}
		a, err := p.AResource()
		if err != nil {
			return "", false
		}
		return net.IP(a.A[:]).String(), true
	}
}
