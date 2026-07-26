import { describe, expect, it } from 'vitest';
import { isSensitiveField, orderedSecretFields } from './secretFields';

describe('orderedSecretFields', () => {
  // The server marshals a map, so the wire order is alphabetical and puts the
  // password above the username it belongs to.
  it('puts the identity before the secret', () => {
    const got = orderedSecretFields({ password: 'pw', username: 'root', sudo_password: 'sp' });
    expect(got.map(([k]) => k)).toEqual(['username', 'password', 'sudo_password']);
  });

  it('keeps unknown fields after the known ones, alphabetically', () => {
    const got = orderedSecretFields({ zeta: '1', alpha: '2', username: 'root' });
    expect(got.map(([k]) => k)).toEqual(['username', 'alpha', 'zeta']);
  });

  it('loses no field', () => {
    const secret = { username: 'root', private_key: 'k', passphrase: 'p', note: 'n' };
    expect(orderedSecretFields(secret)).toHaveLength(Object.keys(secret).length);
  });
});

describe('isSensitiveField', () => {
  it('masks secrets and not identities', () => {
    expect(isSensitiveField('password')).toBe(true);
    expect(isSensitiveField('private_key')).toBe(true);
    expect(isSensitiveField('sudo_password')).toBe(true);
    expect(isSensitiveField('username')).toBe(false);
  });
});
