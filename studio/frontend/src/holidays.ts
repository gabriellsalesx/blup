// National holidays shown on the calendar, by the Studio's language:
// Brazil (pt), the United States (en) and Spain (es). Worked out here, no
// service needed: fixed days, Easter's (Carnival, Good Friday, Corpus
// Christi) and the US's "nth weekday of the month".

const iso = (y: number, m: number, d: number) => `${y}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}`;

// Easter Sunday (the Gregorian computus: Meeus/Jones/Butcher).
function easter(y: number): Date {
  const a = y % 19, b = Math.floor(y / 100), c = y % 100, d = Math.floor(b / 4), e = b % 4;
  const f = Math.floor((b + 8) / 25), g = Math.floor((b - f + 1) / 3), h = (19 * a + b - d - g + 15) % 30;
  const i = Math.floor(c / 4), k = c % 4, l = (32 + 2 * e + 2 * i - h - k) % 7, m = Math.floor((a + 11 * h + 22 * l) / 451);
  const month = Math.floor((h + l - 7 * m + 114) / 31), day = ((h + l - 7 * m + 114) % 31) + 1;
  return new Date(y, month - 1, day);
}
const fromEaster = (y: number, days: number) => {
  const d = easter(y);
  d.setDate(d.getDate() + days);
  return iso(d.getFullYear(), d.getMonth() + 1, d.getDate());
};
// The nth weekday (0 Sunday) of a month; n = -1 is the last one.
function nth(y: number, m: number, weekday: number, n: number): string {
  if (n > 0) {
    const first = new Date(y, m - 1, 1).getDay();
    return iso(y, m, 1 + ((weekday - first + 7) % 7) + (n - 1) * 7);
  }
  const last = new Date(y, m, 0);
  return iso(y, m, last.getDate() - ((last.getDay() - weekday + 7) % 7));
}

// holidays(year, lang): date (2006-01-02) -> name.
export function holidays(y: number, lang: string): Record<string, string> {
  if (lang === "pt") return {
    [iso(y, 1, 1)]: "Confraternização Universal",
    [fromEaster(y, -48)]: "Carnaval",
    [fromEaster(y, -47)]: "Carnaval",
    [fromEaster(y, -2)]: "Sexta-feira Santa",
    [iso(y, 4, 21)]: "Tiradentes",
    [iso(y, 5, 1)]: "Dia do Trabalho",
    [fromEaster(y, 60)]: "Corpus Christi",
    [iso(y, 9, 7)]: "Independência do Brasil",
    [iso(y, 10, 12)]: "Nossa Senhora Aparecida",
    [iso(y, 11, 2)]: "Finados",
    [iso(y, 11, 15)]: "Proclamação da República",
    [iso(y, 11, 20)]: "Consciência Negra",
    [iso(y, 12, 25)]: "Natal",
  };
  if (lang === "es") return {
    [iso(y, 1, 1)]: "Año Nuevo",
    [iso(y, 1, 6)]: "Epifanía del Señor",
    [fromEaster(y, -2)]: "Viernes Santo",
    [iso(y, 5, 1)]: "Día del Trabajador",
    [iso(y, 8, 15)]: "Asunción de la Virgen",
    [iso(y, 10, 12)]: "Fiesta Nacional de España",
    [iso(y, 11, 1)]: "Todos los Santos",
    [iso(y, 12, 6)]: "Día de la Constitución",
    [iso(y, 12, 8)]: "Inmaculada Concepción",
    [iso(y, 12, 25)]: "Navidad",
  };
  return {
    [iso(y, 1, 1)]: "New Year's Day",
    [nth(y, 1, 1, 3)]: "Martin Luther King Jr. Day",
    [nth(y, 2, 1, 3)]: "Presidents' Day",
    [nth(y, 5, 1, -1)]: "Memorial Day",
    [iso(y, 6, 19)]: "Juneteenth",
    [iso(y, 7, 4)]: "Independence Day",
    [nth(y, 9, 1, 1)]: "Labor Day",
    [nth(y, 10, 1, 2)]: "Columbus Day",
    [iso(y, 11, 11)]: "Veterans Day",
    [nth(y, 11, 4, 4)]: "Thanksgiving",
    [iso(y, 12, 25)]: "Christmas Day",
  };
}
