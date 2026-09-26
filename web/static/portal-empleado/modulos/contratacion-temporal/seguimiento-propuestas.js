/**
 * Propuestas de nombramiento en el panel de seguimiento: tras una no
 * incorporación la siguiente persona recibe una propuesta nueva y la
 * anterior queda en la historia, sustituida por esa no incorporación. El
 * servidor decide cuál es la vigente; aquí solo se valida y se pinta, sin
 * referencias internas.
 */

const REF = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/u;
const MOTIVO = /^[a-z][a-z0-9_]{1,63}$/u;
const MAXIMO = 50;

function registro(valor, campos) {
  if (!valor || typeof valor !== "object" || Array.isArray(valor)) return false;
  const claves = Object.keys(valor);
  return claves.length === campos.length && campos.every((c) => Object.hasOwn(valor, c));
}

const instanteValido = (v) => typeof v === "string" && INSTANTE.test(v) && Number.isFinite(Date.parse(v));

/**
 * Valida la lista del estado: órdenes 1..n, versiones crecientes, solo la
 * última vigente y cada anterior con la no incorporación que la sustituyó.
 */
export function validarPropuestasSeguimiento(lista) {
  if (!Array.isArray(lista) || lista.length > MAXIMO) throw new TypeError("propuestas no válidas");
  lista.forEach((p, i) => {
    const ultima = i === lista.length - 1;
    if (!registro(p, ["orden", "version_resultante", "confirmada_en", "recibo_ref", "vigente", "sustitucion"])
      || p.orden !== i + 1 || !Number.isSafeInteger(p.version_resultante) || p.version_resultante < 1
      || (i > 0 && p.version_resultante <= lista[i - 1].version_resultante)
      || !instanteValido(p.confirmada_en) || !REF.test(p.recibo_ref) || p.vigente !== ultima
      || (p.sustitucion === null) !== ultima) throw new TypeError("propuestas no válidas");
    const s = p.sustitucion;
    if (s !== null && (!registro(s, ["no_incorporacion_recibo_ref", "motivo_clave", "registrada_en"])
      || !REF.test(s.no_incorporacion_recibo_ref) || !MOTIVO.test(s.motivo_clave) || !instanteValido(s.registrada_en))) {
      throw new TypeError("propuestas no válidas");
    }
  });
  return Object.freeze(lista.map((p) => Object.freeze({ ...p, sustitucion: p.sustitucion ? Object.freeze({ ...p.sustitucion }) : null })));
}

/**
 * La no incorporación que sigue en vigor: la de la persona de la propuesta
 * vigente. Si una propuesta posterior la sustituyó, es historia.
 */
export function noIncorporacionVigente(estado) {
  const n = estado?.no_incorporacion;
  if (!n) return null;
  const sustituida = (estado.propuestas ?? []).some((p) => p.sustitucion?.no_incorporacion_recibo_ref === n.recibo_ref);
  return sustituida ? null : n;
}

// Día civil en Madrid de un instante UTC, en el formato de las fechas civiles.
function diaCivil(instante) {
  const partes = new Intl.DateTimeFormat("en-CA", { timeZone: "Europe/Madrid", year: "numeric", month: "2-digit", day: "2-digit" })
    .formatToParts(new Date(instante));
  const v = (tipo) => partes.find((p) => p.type === tipo)?.value;
  return `${v("year")}-${v("month")}-${v("day")}`;
}

/** Filas del resumen: la propuesta vigente y, si las hay, las sustituidas. */
export function filasPropuestas(estado, opciones, t, fecha) {
  const lista = estado?.propuestas ?? [];
  if (lista.length === 0) return [];
  const filas = [];
  for (const p of lista) {
    if (p.vigente) {
      filas.push([t("propuesta_vigente"), t("propuesta_vigente_desde", { fecha: fecha(diaCivil(p.confirmada_en)) })]);
      continue;
    }
    const motivo = opciones?.no_incorporacion?.motivos?.find((m) => m.clave === p.sustitucion.motivo_clave);
    filas.push([t("propuesta_anterior", { orden: p.orden }), t("propuesta_sustituida", {
      desde: fecha(diaCivil(p.confirmada_en)), motivo: motivo ? motivo.etiqueta : p.sustitucion.motivo_clave,
      fecha: fecha(diaCivil(p.sustitucion.registrada_en)) })]);
  }
  return filas;
}
