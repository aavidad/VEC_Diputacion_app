/** Fecha civil de Madrid para filtros y valores de formularios; no presenta texto. */
export function hoyCivilCronos(zona = "Europe/Madrid", ahora = new Date()) {
  const partes = new Intl.DateTimeFormat(undefined, {
    calendar: "iso8601", numberingSystem: "latn",
    timeZone: zona, year: "numeric", month: "2-digit", day: "2-digit",
  }).formatToParts(ahora);
  const valores = Object.fromEntries(partes.map(({ type, value }) => [type, value]));
  return `${valores.year}-${valores.month}-${valores.day}`;
}
