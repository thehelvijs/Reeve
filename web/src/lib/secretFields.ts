// Field order for a revealed secret. The server sends a map, which marshals in
// alphabetical order and puts the password above the user it belongs to; these
// are read top to bottom while typing a login, so the identity comes first.
const ORDER = ['username', 'user', 'host', 'port', 'password', 'passphrase', 'private_key', 'token', 'sudo_password'];

// secretsSensitive are masked until the reader asks to see them.
const SENSITIVE = new Set(['password', 'passphrase', 'private_key', 'token', 'sudo_password', 'secret']);

export function orderedSecretFields(secret: Record<string, string>): [string, string][] {
  const rank = (k: string) => {
    const i = ORDER.indexOf(k);
    if (i === -1) {
      return ORDER.length;
    }
    return i;
  };
  return Object.entries(secret).sort((a, b) => {
    const d = rank(a[0]) - rank(b[0]);
    if (d !== 0) {
      return d;
    }
    return a[0].localeCompare(b[0]);
  });
}

export function isSensitiveField(name: string): boolean {
  return SENSITIVE.has(name);
}
