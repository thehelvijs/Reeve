// Package sshinstall pushes the agent onto a LAN host over SSH: the server
// connects out, copies the matching agent binary and the install script, and
// runs the installer under sudo. The pull path (curl the server-hosted
// install.sh) still exists; this is the same installer, driven from the server.
package sshinstall

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// Timeouts: dialTimeout covers the TCP+SSH handshake, runTimeout the whole
// install so a wedged host cannot pin a request handler open.
const (
	dialTimeout = 10 * time.Second
	runTimeout  = 5 * time.Minute
)

// maxOutput caps captured remote output.
const maxOutput = 256 << 10

// DefaultPort is the SSH port assumed when none is given.
const DefaultPort = 22

// Target is one host to install on. Credentials are used for this operation
// only; nothing here is persisted.
type Target struct {
	Address      string
	Port         int
	User         string
	Password     string
	PrivateKey   string
	Passphrase   string
	SudoPassword string
	// Fingerprint is the SHA256 host-key fingerprint the admin confirmed. An
	// empty value is refused: without it this would trust any host that answers.
	Fingerprint string
}

// Scripts carries the installer and uninstaller the server ships.
type Scripts struct {
	Install   []byte
	Uninstall []byte
}

// BinaryProvider returns the agent build for a `uname -m` value, along with its
// hex sha256 so the remote installer can check what landed on disk.
type BinaryProvider func(unameArch string) (data []byte, sha256Hex string, err error)

func (t Target) addr() string {
	port := t.Port
	if port == 0 {
		port = DefaultPort
	}
	return net.JoinHostPort(t.Address, strconv.Itoa(port))
}

// Validate reports whether the target is complete enough to attempt.
func (t Target) Validate() error {
	if strings.TrimSpace(t.Address) == "" {
		return errors.New("host address is required")
	}
	if t.Port < 0 || t.Port > 65535 {
		return errors.New("ssh port must be between 1 and 65535")
	}
	if strings.TrimSpace(t.User) == "" {
		return errors.New("ssh username is required")
	}
	if t.Password == "" && strings.TrimSpace(t.PrivateKey) == "" {
		return errors.New("a password or a private key is required")
	}
	if strings.TrimSpace(t.Fingerprint) == "" {
		return errors.New("confirm the host key fingerprint first")
	}
	return nil
}

// Probe returns the host's public key fingerprint and type without
// authenticating, so an admin can confirm it before credentials are sent.
func Probe(ctx context.Context, address string, port int) (fingerprint, keyType string, err error) {
	t := Target{Address: address, Port: port}
	var captured ssh.PublicKey
	cfg := &ssh.ClientConfig{
		User:            "reeve-probe",
		Auth:            nil,
		Timeout:         dialTimeout,
		HostKeyCallback: func(_ string, _ net.Addr, key ssh.PublicKey) error { captured = key; return nil },
	}
	conn, dialErr := dial(ctx, t.addr(), cfg)
	if conn != nil {
		conn.Close()
	}
	// The handshake exchanges host keys before authentication, so a rejected
	// login still tells us the key. Only a failure before that point is fatal.
	if captured == nil {
		if dialErr != nil {
			return "", "", dialErr
		}
		return "", "", errors.New("host presented no key")
	}
	return ssh.FingerprintSHA256(captured), captured.Type(), nil
}

// session bundles a live connection with a running transcript of what was run.
type session struct {
	client *ssh.Client
	log    strings.Builder
}

// connect opens an authenticated session to the target.
func connect(ctx context.Context, t Target) (*session, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}
	cfg, err := clientConfig(t)
	if err != nil {
		return nil, err
	}
	client, err := dial(ctx, t.addr(), cfg)
	if err != nil {
		return nil, err
	}
	return &session{client: client}, nil
}

func (s *session) close() { s.client.Close() }

// run executes one command, appending it and its output to the transcript.
func (s *session) run(label, cmd string, stdin []byte) (string, error) {
	out, err := runCommand(s.client, cmd, stdin)
	s.log.WriteString("$ " + label + "\n")
	s.log.WriteString(out)
	if !strings.HasSuffix(out, "\n") && out != "" {
		s.log.WriteString("\n")
	}
	return out, err
}

// stage creates a private temporary directory on the host and uploads files
// into it, returning the directory and a cleanup function.
func (s *session) stage(files map[string][]byte) (string, func(), error) {
	dirOut, err := s.run("mktemp -d", "mktemp -d /tmp/reeve-install.XXXXXX", nil)
	if err != nil {
		return "", func() {}, fmt.Errorf("create staging directory: %w", err)
	}
	dir := strings.TrimSpace(dirOut)
	if !safePath.MatchString(dir) {
		return "", func() {}, fmt.Errorf("host returned an unusable staging path %q", dir)
	}
	cleanup := func() { runCommand(s.client, "rm -rf "+dir, nil) }
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if len(files[name]) == 0 {
			continue
		}
		if _, err := s.run("upload "+name, "cat > "+dir+"/"+name, files[name]); err != nil {
			cleanup()
			return "", func() {}, fmt.Errorf("upload %s: %w", name, err)
		}
	}
	return dir, cleanup, nil
}

// Install copies the agent and the installer to the host and runs it, returning
// the combined remote output.
func Install(ctx context.Context, t Target, provider BinaryProvider, scripts Scripts, env map[string]string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()

	s, err := connect(ctx, t)
	if err != nil {
		return "", err
	}
	defer s.close()
	log := &s.log
	run := s.run

	unameOut, err := run("uname -m", "uname -m", nil)
	if err != nil {
		return log.String(), fmt.Errorf("detect architecture: %w", err)
	}
	arch := strings.TrimSpace(unameOut)
	binary, sum, err := provider(arch)
	if err != nil {
		return log.String(), err
	}

	dir, cleanup, err := s.stage(map[string][]byte{
		"agent":        binary,
		"install.sh":   scripts.Install,
		"uninstall.sh": scripts.Uninstall,
	})
	if err != nil {
		return log.String(), err
	}
	defer cleanup()

	full := map[string]string{
		"REEVE_AGENT_BINARY":     dir + "/agent",
		"REEVE_AGENT_SHA256":     sum,
		"REEVE_UNINSTALL_SCRIPT": dir + "/uninstall.sh",
	}
	for k, v := range env {
		full[k] = v
	}
	assigns, err := envAssignments(full)
	if err != nil {
		return log.String(), err
	}
	cmd, stdin := privileged(t, "env"+assigns+" bash "+dir+"/install.sh")
	if _, err := run("install", cmd, stdin); err != nil {
		return log.String(), fmt.Errorf("run installer: %w (output above)", err)
	}
	return log.String(), nil
}

// Uninstall removes the agent from a host: the installed uninstaller if it is
// there, otherwise the copy the server ships. Removing the host from the catalog
// stays a separate, explicit action.
func Uninstall(ctx context.Context, t Target, scripts Scripts) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()

	s, err := connect(ctx, t)
	if err != nil {
		return "", err
	}
	defer s.close()

	dir, cleanup, err := s.stage(map[string][]byte{"uninstall.sh": scripts.Uninstall})
	if err != nil {
		return s.log.String(), err
	}
	defer cleanup()

	cmd, stdin := privileged(t, "bash "+dir+"/uninstall.sh")
	if _, err := s.run("uninstall", cmd, stdin); err != nil {
		return s.log.String(), fmt.Errorf("run uninstaller: %w (output above)", err)
	}
	return s.log.String(), nil
}

// safePath guards the shell interpolation of the remote staging directory.
var safePath = regexp.MustCompile(`^/[A-Za-z0-9._/-]+$`)

// envAssignments renders the installer environment. Values are single-quoted and
// rejected if they contain a quote or control character, so nothing an admin
// types can break out of the command.
func envAssignments(env map[string]string) (string, error) {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	// Deterministic order keeps the command (and its tests) stable.
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		v := env[k]
		if strings.ContainsAny(v, "'\n\r\x00") {
			return "", fmt.Errorf("value for %s contains a quote or newline", k)
		}
		b.WriteString(" " + k + "='" + v + "'")
	}
	return b.String(), nil
}

// privileged wraps a command so it runs as root: directly when the SSH user is
// root, otherwise through sudo, reading the password from stdin when one was
// supplied and using -n when it was not.
func privileged(t Target, script string) (string, []byte) {
	if t.User == "root" {
		return script, nil
	}
	if t.SudoPassword != "" {
		return "sudo -S -p '' " + script, []byte(t.SudoPassword + "\n")
	}
	return "sudo -n " + script, nil
}

// clientConfig builds the SSH client config, pinning the confirmed host key.
func clientConfig(t Target) (*ssh.ClientConfig, error) {
	var methods []ssh.AuthMethod
	if strings.TrimSpace(t.PrivateKey) != "" {
		var signer ssh.Signer
		var err error
		if t.Passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(t.PrivateKey), []byte(t.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(t.PrivateKey))
		}
		if err != nil {
			return nil, fmt.Errorf("private key: %w", err)
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}
	if t.Password != "" {
		methods = append(methods, ssh.Password(t.Password))
	}
	want := strings.TrimSpace(t.Fingerprint)
	return &ssh.ClientConfig{
		User:    t.User,
		Auth:    methods,
		Timeout: dialTimeout,
		HostKeyCallback: func(_ string, _ net.Addr, key ssh.PublicKey) error {
			got := ssh.FingerprintSHA256(key)
			if got != want {
				return fmt.Errorf("host key fingerprint %s does not match the confirmed %s", got, want)
			}
			return nil
		},
	}, nil
}

// dial connects with the context bounding the handshake.
func dial(ctx context.Context, addr string, cfg *ssh.ClientConfig) (*ssh.Client, error) {
	conn, err := dialTCP(ctx, addr)
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", addr, err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, cfg)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("ssh handshake with %s: %w", addr, err)
	}
	conn.SetDeadline(time.Time{})
	return ssh.NewClient(c, chans, reqs), nil
}

// dialTCP connects to addr, retrying through mDNS when DNS could not answer for
// a .local name. The original error is what surfaces if the retry also fails.
func dialTCP(ctx context.Context, addr string) (net.Conn, error) {
	d := net.Dialer{Timeout: dialTimeout}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err == nil {
		return conn, nil
	}
	var dnsErr *net.DNSError
	if !errors.As(err, &dnsErr) {
		return nil, err
	}
	host, port, splitErr := net.SplitHostPort(addr)
	if splitErr != nil || !strings.HasSuffix(strings.ToLower(host), ".local") {
		return nil, err
	}
	ip, mdnsErr := lookupMDNS(ctx, host)
	if mdnsErr != nil {
		return nil, err
	}
	conn, retryErr := d.DialContext(ctx, "tcp", net.JoinHostPort(ip, port))
	if retryErr != nil {
		return nil, retryErr
	}
	return conn, nil
}

// runCommand runs one command, returning its combined output.
func runCommand(client *ssh.Client, cmd string, stdin []byte) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	var out bytes.Buffer
	session.Stdout = &limitedWriter{w: &out, remaining: maxOutput}
	session.Stderr = &limitedWriter{w: &out, remaining: maxOutput}
	if stdin != nil {
		session.Stdin = bytes.NewReader(stdin)
	}
	err = session.Run(cmd)
	return out.String(), err
}

// limitedWriter drops output past a cap instead of buffering whatever a remote
// host decides to print.
type limitedWriter struct {
	w         *bytes.Buffer
	remaining int
}

func (l *limitedWriter) Write(p []byte) (int, error) {
	if l.remaining <= 0 {
		return len(p), nil
	}
	if len(p) > l.remaining {
		p = p[:l.remaining]
	}
	n, err := l.w.Write(p)
	l.remaining -= n
	return len(p), err
}
