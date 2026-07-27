import { describe, expect, it } from 'vitest';
import { matchesQuery } from './search';

describe('matchesQuery', () => {
  it('keeps every row when the query is blank', () => {
    expect(matchesQuery('', 'anything')).toBe(true);
    expect(matchesQuery('   ', 'anything')).toBe(true);
  });

  it('matches a substring in any field', () => {
    expect(matchesQuery('ber', 'web-01', 'Berlin')).toBe(true);
    expect(matchesQuery('oslo', 'web-01', 'Berlin')).toBe(false);
  });

  // Two terms have to narrow. Matching either would grow the list as an operator types.
  it('requires every term', () => {
    expect(matchesQuery('web berlin', 'web-01', 'Berlin')).toBe(true);
    expect(matchesQuery('web oslo', 'web-01', 'Berlin')).toBe(false);
  });

  it('ignores case and empty fields', () => {
    expect(matchesQuery('WEB', 'web-01', null, undefined, '')).toBe(true);
  });

  // Fields are joined before matching, so a term must not straddle two of them.
  it('does not match across a field boundary', () => {
    expect(matchesQuery('01berlin', 'web-01', 'Berlin')).toBe(false);
  });
});
