// Databases of a vault (see internal/app/database.go): the kinds of
// column, the colors of the choices, the templates a new database starts
// from, and the small rules the table, the board and the record's page
// share.
import type { DBColumn, DBSchema, DBRecord } from "../bindings/github.com/gabriellsalesx/blup/internal/app/models";
import { t } from "./i18n";

export type ColumnType = "text" | "number" | "phone" | "email" | "url" | "date" | "select" | "multi" | "check" | "link";
export const TYPES: ColumnType[] = ["text", "number", "select", "multi", "date", "phone", "email", "url", "check", "link"];

// The choices' colors: a tint behind, the color itself for the text.
const OPTION_COLORS = ["gray", "red", "orange", "yellow", "green", "blue", "pink"] as const;
const HEX: Record<string, string> = { gray: "#94a3b8", red: "#ef4444", orange: "#f59e0b", yellow: "#eab308", green: "#22c55e", blue: "#3b82f6", pink: "#ec4899" };
export const optionColor = (c?: string) => HEX[c ?? "gray"] ?? HEX.gray;
export const nextColor = (c?: string) => OPTION_COLORS[((OPTION_COLORS as readonly string[]).indexOf(c ?? "gray") + 1) % OPTION_COLORS.length];

// A property's value, as a note's front matter (YAML) comes from the
// Studio: JSON. The bindings leave the Go side's value untyped; the window
// keeps it as this.
export type Json = string | number | boolean | null | Json[] | { [k: string]: Json };
export type JsonProps = Record<string, Json>;
// Whether a value (from a field, say) is JSON all the way down.
export function isJson(v: unknown): v is Json {
  if (v === null || typeof v === "string" || typeof v === "boolean") return true;
  if (typeof v === "number") return Number.isFinite(v);
  if (Array.isArray(v)) return v.every(isJson);
  if (typeof v === "object") return Object.getPrototypeOf(v) === Object.prototype && Object.values(v).every(isJson);
  return false;
}

// A property name from a column name: lower case, no accents, dashes.
export function keyOf(name: string, taken: string[]) {
  const base = name.normalize("NFD").replace(/[̀-ͯ]/g, "").toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "") || "campo";
  let key = base;
  for (let i = 2; taken.includes(key) || ["tags", "created", "updated", "title"].includes(key); i++) key = `${base}-${i}`;
  return key;
}

// A cell's value as text (for the search, the cards and sorting).
export function asText(v: unknown): string {
  if (v == null) return "";
  if (Array.isArray(v)) return v.map(asText).join(", ");
  // A ticked box is "true" (the cards draw it as a check icon).
  if (typeof v === "boolean") return v ? "true" : "";
  return String(v).replace(/^\[\[|\]\]$/g, "");
}

// Whether a record has what the search asks for: its name, any field or
// its text.
export function matches(r: DBRecord, q: string) {
  if (!q) return true;
  const hay = [r.name, r.text, ...Object.values(r.props ?? {}).map(asText)].join("\n").toLowerCase();
  return q.toLowerCase().split(/\s+/).every((w) => hay.includes(w));
}

export function compare(a: DBRecord, b: DBRecord, key: string, col?: DBColumn) {
  const va = key ? a.props?.[key] : a.name, vb = key ? b.props?.[key] : b.name;
  if (col?.type === "number") return (Number(va) || 0) - (Number(vb) || 0);
  if (col?.type === "select") {
    const order = (col.options ?? []).map((o) => o.value);
    return (order.indexOf(String(va)) + 1 || 999) - (order.indexOf(String(vb)) + 1 || 999);
  }
  return asText(va).localeCompare(asText(vb), undefined, { numeric: true });
}

// The templates: ready columns and views for common lists.
export type Template = { id: string; name: string; about: string; schema: DBSchema };
export function templates(): Template[] {
  const d = t.db;
  const col = (key: string, name: string, type: ColumnType, options: [string, string][] = []): DBColumn => ({ key, name, type, options: options.map(([value, color]) => ({ value, color })) });
  const views = (groupBy: string) => [
    { id: "table", name: d.viewTable, type: "table", groupBy: "", sort: "", filter: "", match: "" },
    ...(groupBy ? [{ id: "board", name: d.viewBoard, type: "board", groupBy, sort: "", filter: "", match: "" }] : []),
  ];
  const schema = (columns: DBColumn[], groupBy = ""): DBSchema => ({ version: 1, records: "", tag: "", columns, views: views(groupBy) });
  return [
    { id: "clients", name: d.tClients, about: d.tClientsAbout, schema: schema([
      col("empresa", d.cCompany, "text"),
      col("telefone", d.cPhone, "phone"),
      col("email", d.cEmail, "email"),
      col("temperatura", d.cTemperature, "select", [[d.oHot, "red"], [d.oWarm, "orange"], [d.oCold, "blue"]]),
      col("ultimo-contato", d.cLastContact, "date"),
    ], "temperatura") },
    { id: "tasks", name: d.tTasks, about: d.tTasksAbout, schema: schema([
      col("status", d.cStatus, "select", [[d.oTodo, "gray"], [d.oDoing, "blue"], [d.oDone, "green"]]),
      col("prioridade", d.cPriority, "select", [[d.oHigh, "red"], [d.oMedium, "orange"], [d.oLow, "gray"]]),
      col("prazo", d.cDue, "date"),
      col("responsavel", d.cOwner, "text"),
    ], "status") },
    { id: "reading", name: d.tReading, about: d.tReadingAbout, schema: schema([
      col("autor", d.cAuthor, "text"),
      col("status", d.cStatus, "select", [[d.oWant, "gray"], [d.oReading, "blue"], [d.oRead, "green"]]),
      col("nota", d.cRating, "number"),
      col("link", d.cLink, "url"),
    ], "status") },
    { id: "empty", name: d.tEmpty, about: d.tEmptyAbout, schema: schema([col("status", d.cStatus, "select", [[d.oTodo, "gray"], [d.oDone, "green"]])], "status") },
  ];
}
