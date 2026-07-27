// The shared list filter behind every page's header search. Terms are matched
// independently, so "ops berlin" narrows to rows carrying both instead of
// widening to rows carrying either.
export function matchesQuery(query: string, ...fields: (string | null | undefined)[]): boolean {
  const terms = query.trim().toLowerCase().split(/\s+/).filter(Boolean);
  if (terms.length === 0) {
    return true;
  }
  const haystack = fields.filter(Boolean).join(' ').toLowerCase();
  return terms.every((term) => haystack.includes(term));
}
