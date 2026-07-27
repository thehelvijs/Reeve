package collect

import (
	"regexp"
	"strings"
)

// secretArgNames are argument names whose value is a credential, matched as a
// substring of the name so --db-password and --api-token both hit.
var secretArgNames = []string{
	"password", "passwd", "passphrase", "secret", "token", "apikey",
	"api-key", "api_key", "credential", "auth", "access-key", "access_key",
	"private-key", "private_key", "bearer",
}

// authSchemes stand between an Authorization argument and the credential, so the
// slot after the name holds the scheme and the one after that holds the secret.
var authSchemes = map[string]bool{"bearer": true, "basic": true, "token": true}

// userArgNames carry a user:password pair, where only the half after the colon
// is secret.
var userArgNames = map[string]bool{"-u": true, "--user": true, "--username": true}

// mysqlClients read the password from a value attached to -p. No other common
// program does, and plenty use -pf or -print, so the mask needs the program name
// to tell "-phunter2" from an ordinary short flag.
var mysqlClients = map[string]bool{
	"mysql": true, "mysqldump": true, "mysqladmin": true, "mysqlshow": true,
	"mariadb": true, "mariadb-dump": true, "mariadb-admin": true,
}

const redactedValue = "REDACTED"

// urlUserinfo matches the password half of scheme://user:password@host.
var urlUserinfo = regexp.MustCompile(`(://[^:/@\s]+:)[^@/\s]+@`)

// redactSecrets masks credential values in a command line, so a password passed
// in argv never leaves the host. It keys on the argument that names the secret
// rather than on the value, because a credential has no shape worth guessing. A
// command with nothing to mask is returned byte for byte.
func redactSecrets(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return command
	}
	mysql := mysqlClients[basename(fields[0])]
	changed := false
	for i := 0; i < len(fields); i++ {
		if masked, ok := maskInline(fields[i], mysql); ok {
			fields[i] = masked
			changed = true
			continue
		}
		value := i + 1
		if userArgNames[strings.ToLower(fields[i])] && value < len(fields) {
			if user, _, ok := strings.Cut(fields[value], ":"); ok {
				fields[value] = user + ":" + redactedValue
				changed = true
				i = value
			}
			continue
		}
		if !namesSecret(fields[i]) {
			continue
		}
		// An Authorization argument names the scheme first and the credential
		// after it, so the slot to mask is one further along.
		if value < len(fields) && authSchemes[strings.ToLower(fields[value])] {
			value++
		}
		if value >= len(fields) || strings.HasPrefix(fields[value], "-") {
			continue
		}
		fields[value] = redactedValue
		changed = true
		i = value
	}
	if !changed {
		return command
	}
	return strings.Join(fields, " ")
}

// maskInline masks a value carried inside the argument itself: --token=x, a
// mysql client's -px, or a URL with userinfo.
func maskInline(f string, mysql bool) (string, bool) {
	if name, value, ok := strings.Cut(f, "="); ok && value != "" && namesSecret(name) {
		return name + "=" + redactedValue, true
	}
	if mysql && strings.HasPrefix(f, "-p") && !strings.HasPrefix(f, "--") && len(f) > 2 {
		return "-p" + redactedValue, true
	}
	if masked := urlUserinfo.ReplaceAllString(f, "${1}"+redactedValue+"@"); masked != f {
		return masked, true
	}
	return f, false
}

// basename is the program name without its directory.
func basename(path string) string {
	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		return path[i+1:]
	}
	return path
}

// namesSecret reports whether an argument name introduces a credential.
func namesSecret(arg string) bool {
	name := strings.ToLower(strings.TrimLeft(arg, "-"))
	for _, s := range secretArgNames {
		if strings.Contains(name, s) {
			return true
		}
	}
	return false
}
