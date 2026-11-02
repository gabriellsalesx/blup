// What a typed field saves (FieldInput): undefined when nothing changed or
// what was typed can't be kept. A number column keeps a finite number.
export function typedValue(kind: string, typed: string, before: string): string | number | undefined {
  if (kind === "number" && typed !== "") {
    const n = Number(typed);
    return Number.isFinite(n) && String(n) !== before ? n : undefined;
  }
  return typed !== before ? typed : undefined;
}
