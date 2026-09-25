/**
 * Hora civil de Madrid para los campos «conocido en» de las pantallas RRHH.
 *
 * Los campos `datetime-local` no llevan zona: se interpretan siempre en
 * Europe/Madrid, nunca en la zona del navegador, y viajan en UTC.
 */
export const ZONA_MADRID = "Europe/Madrid";

const partesMadrid = new Intl.DateTimeFormat("en-GB", {
  timeZone: ZONA_MADRID, year: "numeric", month: "2-digit", day: "2-digit",
  hour: "2-digit", minute: "2-digit", hourCycle: "h23",
});

/** Devuelve `AAAA-MM-DDTHH:MM` del instante `ms` en hora de Madrid. */
export function localMadrid(ms) {
  const p = Object.fromEntries(partesMadrid.formatToParts(new Date(ms)).map(({ type, value }) => [type, value]));
  return `${p.year}-${p.month}-${p.day}T${p.hour}:${p.minute}`;
}

/**
 * Convierte una hora civil de Madrid (`AAAA-MM-DDTHH:MM`) en su instante UTC en
 * milisegundos. Una hora inexistente (salto de marzo) devuelve null; en la hora
 * repetida de octubre se toma la primera aparición, la más antigua.
 */
export function instanteDesdeHoraMadrid(local) {
  if (!/^\d{4}-\d\d-\d\dT\d\d:\d\d$/.test(String(local ?? ""))) return null;
  const base = Date.parse(`${local}:00Z`);
  if (!Number.isFinite(base)) return null;
  const candidatos = [2, 1, 0].map((horas) => base - horas * 3600000).filter((ms) => localMadrid(ms) === local);
  return candidatos.length ? candidatos[0] : null;
}
