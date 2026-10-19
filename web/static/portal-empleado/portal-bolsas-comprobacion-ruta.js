/**
 * Comprobación del ofrecimiento propuesta desde la selección de Contratación
 * temporal. El aviso «pendiente de comprobación» lleva a la ficha de la
 * persona en Bolsa con el contexto que publicó el servidor (16 campos) y la
 * versión esperada del acto. La URL solo propone: Bolsa vuelve a cotejar todo
 * al registrar. No contiene datos personales: solo referencias y versiones.
 */
export const CAMPOS_CONTEXTO_COMPROBACION = Object.freeze({
  ambito_ref: "ref", bolsa_ref: "ref", categoria_ref: "ref", causa_ofrecida: "texto",
  expediente_ref: "ref", huella_politica_ofertas_sha256: "huella", modalidad_ofrecida: "texto_obligatorio",
  necesidad_ref: "ref", orden_acta: "entero", organizacion_ref: "ref", participacion_ref: "ref",
  unidad_ref: "ref", version_bolsa: "entero", version_expediente: "entero", version_instantanea: "entero",
  version_politica_ofertas: "entero",
});
const REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9:._/-]{1,255}$/u;
const HUELLA = /^[0-9a-f]{64}$/u;
const MAXIMA_VERSION = 999_999_999;

function objetoPlano(valor) {
  return valor !== null && typeof valor === "object" && !Array.isArray(valor);
}

function camposExactos(objeto, campos) {
  const claves = Object.keys(objeto);
  return claves.length === campos.length && campos.every((campo) => Object.hasOwn(objeto, campo));
}

function campoValido(valor, tipo) {
  if (tipo === "entero") return Number.isSafeInteger(valor) && valor > 0;
  if (typeof valor !== "string") return false;
  if (tipo === "huella") return HUELLA.test(valor);
  if (tipo === "texto") return valor.length <= 256;
  if (tipo === "texto_obligatorio") return valor.length <= 256 && valor.trim() !== "";
  return REFERENCIA.test(valor);
}

/** Devuelve la propuesta congelada o null si no tiene la forma exacta. */
export function propuestaComprobacionValida(valor) {
  if (!objetoPlano(valor) || !camposExactos(valor, ["contexto", "version_esperada"])) return null;
  const { contexto, version_esperada: version } = valor;
  if (!objetoPlano(contexto) || !camposExactos(contexto, Object.keys(CAMPOS_CONTEXTO_COMPROBACION))
    || !Object.entries(CAMPOS_CONTEXTO_COMPROBACION).every(([campo, tipo]) => campoValido(contexto[campo], tipo))
    || !Number.isSafeInteger(version) || version < 0 || version > MAXIMA_VERSION) return null;
  const ordenado = Object.fromEntries(Object.keys(CAMPOS_CONTEXTO_COMPROBACION).map((campo) => [campo, contexto[campo]]));
  return Object.freeze({ contexto: Object.freeze(ordenado), version_esperada: version });
}

/** Propuesta a partir del 409 ya validado por el cliente de la selección. */
export function propuestaDesdeComprobacionPendiente(comprobacion) {
  if (!objetoPlano(comprobacion?.contexto) || !objetoPlano(comprobacion?.estado)) return null;
  return propuestaComprobacionValida({ contexto: comprobacion.contexto, version_esperada: comprobacion.estado.version_esperada });
}

export function codificarPropuestaComprobacion(propuesta) {
  const valida = propuestaComprobacionValida(propuesta);
  if (!valida) throw new TypeError("comprobación propuesta no válida");
  return JSON.stringify(valida);
}

/** Un valor ilegible se descarta: no impide abrir la bolsa. */
export function decodificarPropuestaComprobacion(texto) {
  if (typeof texto !== "string" || texto.length > 3_072) return null;
  try { return propuestaComprobacionValida(JSON.parse(texto)); } catch { return null; }
}
