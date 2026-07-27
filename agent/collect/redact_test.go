package collect

import (
	"testing"
	"time"
)

func TestRedactSecretsMasksCredentialsInArgv(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"mysql attached password", "mysql -uroot -phunter2 mydb", "mysql -uroot -pREDACTED mydb"},
		{"mysqldump by full path", "/usr/bin/mysqldump -phunter2 mydb", "/usr/bin/mysqldump -pREDACTED mydb"},
		{"long flag with equals", "app --db-password=hunter2 --port=8080", "app --db-password=REDACTED --port=8080"},
		{"long flag separated", "app --token hunter2 --queue high", "app --token REDACTED --queue high"},
		{"api key underscore", "app --api_key=abc123", "app --api_key=REDACTED"},
		{"bearer header", "curl -H Authorization: Bearer abc123 https://api.local", "curl -H Authorization: Bearer REDACTED https://api.local"},
		{"basic header", "curl -H Authorization: Basic dXNlcjpwYXNz", "curl -H Authorization: Basic REDACTED"},
		{"curl userinfo pair", "curl -u admin:hunter2 https://api.local", "curl -u admin:REDACTED https://api.local"},
		{"url userinfo", "psql postgres://app:hunter2@db.local/app", "psql postgres://app:REDACTED@db.local/app"},
		{"secret flag last with no value", "app --password", "app --password"},
		{"secret flag followed by a flag", "app --password --verbose", "app --password --verbose"},
		{"case insensitive name", "app --PASSWORD=hunter2", "app --PASSWORD=REDACTED"},
	}
	for _, c := range cases {
		if got := redactSecrets(c.in); got != c.want {
			t.Errorf("%s:\n got  %q\n want %q", c.name, got, c.want)
		}
	}
}

func TestRedactSecretsLeavesOrdinaryCommandsAlone(t *testing.T) {
	// Anything with nothing to mask must come back byte for byte, spacing
	// included, so an untouched command line is never rewritten.
	cases := []string{
		"/usr/lib/plexmediaserver/Plex Media Server",
		"apt-get -qq -y update",
		"docker run -p 8080:80 nginx",
		"/usr/bin/python3 /opt/app/worker.py --queue=high",
		"test -x /usr/sbin/anacron || ( cd /  && run-parts --report /etc/cron.daily )",
		"chown -R app:app /srv/app",
		// Short flags starting with p are common; only a mysql client means
		// password by them.
		"/sbin/dhclient -d -q -pf /run/dhclient-enp3s0.pid -lf /var/lib/dhclient.lease",
		"find /srv -name '*.log' -print",
		"",
	}
	for _, in := range cases {
		if got := redactSecrets(in); got != in {
			t.Errorf("rewrote a clean command:\n got  %q\n want %q", got, in)
		}
	}
}

func TestParseCmdlineRedacts(t *testing.T) {
	got := ParseCmdline("mysql\x00--password=hunter2\x00mydb\x00")
	if want := "mysql --password=REDACTED mydb"; got != want {
		t.Errorf("cmdline = %q, want %q", got, want)
	}
}

func TestParseCrontabRedacts(t *testing.T) {
	jobs := ParseCrontab("0 3 * * * /usr/bin/backup --api-token=abc123 /srv\n", false)
	if len(jobs) != 1 {
		t.Fatalf("jobs = %d, want 1", len(jobs))
	}
	if want := "/usr/bin/backup --api-token=REDACTED /srv"; jobs[0].Name != want {
		t.Errorf("cron command = %q, want %q", jobs[0].Name, want)
	}
}

func TestScanLogErrorsRedacts(t *testing.T) {
	events := ScanLogErrors("app", []string{
		"error: connect failed for postgres://app:hunter2@db.local/app",
		"FATAL upload rejected: --api-token=abc123",
		"error: disk full on /srv",
	}, time.Unix(0, 0).UTC(), nil)
	want := []string{
		"error: connect failed for postgres://app:REDACTED@db.local/app",
		"FATAL upload rejected: --api-token=REDACTED",
		"error: disk full on /srv",
	}
	if len(events) != len(want) {
		t.Fatalf("events = %d, want %d", len(events), len(want))
	}
	for i := range want {
		if events[i].Message != want[i] {
			t.Errorf("message %d:\n got  %q\n want %q", i, events[i].Message, want[i])
		}
	}
}
