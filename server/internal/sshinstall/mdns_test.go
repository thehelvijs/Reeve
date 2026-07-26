package sshinstall

import (
	"context"
	"os"
	"testing"

	"golang.org/x/net/dns/dnsmessage"
)

// response builds an mDNS reply the way a responder does: the question echoed
// back, then answers, with the names compressed against each other.
func response(t *testing.T, names []string, ips [][4]byte) []byte {
	t.Helper()
	b := dnsmessage.NewBuilder(nil, dnsmessage.Header{Response: true, Authoritative: true})
	b.EnableCompression()
	if err := b.StartQuestions(); err != nil {
		t.Fatal(err)
	}
	if err := b.Question(dnsmessage.Question{
		Name:  dnsmessage.MustNewName(names[0]),
		Type:  dnsmessage.TypeA,
		Class: dnsmessage.ClassINET,
	}); err != nil {
		t.Fatal(err)
	}
	if err := b.StartAnswers(); err != nil {
		t.Fatal(err)
	}
	for i, n := range names {
		h := dnsmessage.ResourceHeader{Name: dnsmessage.MustNewName(n), Class: dnsmessage.ClassINET, TTL: 120}
		if err := b.AResource(h, dnsmessage.AResource{A: ips[i]}); err != nil {
			t.Fatal(err)
		}
	}
	msg, err := b.Finish()
	if err != nil {
		t.Fatal(err)
	}
	return msg
}

func TestMDNSQueryAsksForA(t *testing.T) {
	q, err := mdnsQuery("box.local")
	if err != nil {
		t.Fatal(err)
	}
	var p dnsmessage.Parser
	if _, err := p.Start(q); err != nil {
		t.Fatal(err)
	}
	question, err := p.Question()
	if err != nil {
		t.Fatal(err)
	}
	if question.Name.String() != "box.local." {
		t.Fatalf("name = %q", question.Name.String())
	}
	if question.Type != dnsmessage.TypeA {
		t.Fatalf("type = %v", question.Type)
	}
	if question.Class != dnsmessage.ClassINET {
		t.Fatalf("class = %v", question.Class)
	}
}

func TestMDNSAnswerPicksTheRequestedName(t *testing.T) {
	msg := response(t,
		[]string{"other.local.", "box.local."},
		[][4]byte{{10, 0, 0, 1}, {192, 168, 1, 100}},
	)
	ip, ok := mdnsAnswer(msg, "box.local")
	if !ok || ip != "192.168.1.100" {
		t.Fatalf("mdnsAnswer = %q, %v", ip, ok)
	}
}

func TestMDNSAnswerIgnoresOtherHosts(t *testing.T) {
	msg := response(t, []string{"printer.local."}, [][4]byte{{10, 0, 0, 5}})
	if ip, ok := mdnsAnswer(msg, "box.local"); ok {
		t.Fatalf("matched an unrelated name: %q", ip)
	}
}

func TestMDNSAnswerRejectsGarbage(t *testing.T) {
	if _, ok := mdnsAnswer([]byte{0x00, 0x01, 0x02}, "box.local"); ok {
		t.Fatal("parsed a truncated message")
	}
}

// Live check against a real responder on the LAN, off by default:
// REEVE_MDNS_HOST=some-box.local go test ./server/internal/sshinstall -run Live
func TestMDNSLive(t *testing.T) {
	host := os.Getenv("REEVE_MDNS_HOST")
	if host == "" {
		t.Skip("set REEVE_MDNS_HOST to a .local name on this LAN")
	}
	ip, err := lookupMDNS(context.Background(), host)
	if err != nil {
		t.Fatalf("lookupMDNS(%s): %v", host, err)
	}
	t.Logf("%s -> %s", host, ip)
}
