// The words added to the dictionary, as the Prefs binding types them: a Go
// nil slice arrives as null, which counts as no word.
export const wordCount = (words: readonly string[] | null | undefined): number => words?.length ?? 0;

// A word taken out of the list needs a fresh dictionary; one added is
// already in it (spell.ts addWord).
export const wordRemoved = (words: readonly string[] | null | undefined, before: number | undefined): boolean =>
  wordCount(words) < (before ?? 0);
