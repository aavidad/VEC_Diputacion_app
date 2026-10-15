export const RUTA_REGLAS_SITUACION = "/api/vec/bolsa/politica-transiciones";
const ESQUEMA = "vec.bolsa.rrhh.politica-transiciones.v1";
const SITUACIONES = new Set(["disponible", "no_disponible", "trabajando", "pendiente_incorporacion", "renuncia", "excluido", "disponible_desde", "en_revision"]);
const HUELLA = /^[a-f0-9]{64}$/u;
const MAXIMO = 64 * 1024;
const FECHA_RFC3339 = /^(\d{4})-(\d{2})-(\d{2})T([01]\d|2[0-3]):([0-5]\d):([0-5]\d)(?:\.\d+)?(Z|[+-](?:[01]\d|2[0-3]):[0-5]\d)$/u;

function fechaPublicacionValida(valor) {
  if (typeof valor !== "string") return false;
  const partes = FECHA_RFC3339.exec(valor);
  if (!partes || !Number.isFinite(Date.parse(valor))) return false;
  const anio = Number(partes[1]), mes = Number(partes[2]), dia = Number(partes[3]);
  const bisiesto = anio % 4 === 0 && (anio % 100 !== 0 || anio % 400 === 0);
  const dias = [31, bisiesto ? 29 : 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];
  return mes >= 1 && mes <= 12 && dia >= 1 && dia <= dias[mes - 1];
}

export function parValido(par) {
  if (typeof par !== "string" || par.length > 64) return false;
  const partes = par.split(">");
  return partes.length === 2 && partes[0] !== partes[1] && partes.every((parte) => SITUACIONES.has(parte));
}
const listaValida = (lista) => Array.isArray(lista) && lista.length <= 64
  && lista.every(parValido) && new Set(lista).size === lista.length;
const subconjunto = (lista, limite) => lista.every((par) => limite.includes(par));

export function validarPoliticaTransiciones(sobre, { publicacion = false } = {}) {
  const p = sobre?.data;
  const l = p?.limites;
  if (p?.esquema !== ESQUEMA || !Number.isSafeInteger(p.version) || p.version < 1
    || !listaValida(p.transiciones_guardadas) || !listaValida(p.transiciones_efectivas)
    || !listaValida(p.transiciones_obligatorias) || !l || typeof l !== "object" || Array.isArray(l)
    || typeof l.catalogo_ref !== "string" || !l.catalogo_ref.trim() || l.catalogo_ref.length > 512
    || !Number.isSafeInteger(l.catalogo_version) || l.catalogo_version < 1
    || typeof l.catalogo_sha256 !== "string" || !HUELLA.test(l.catalogo_sha256)
    || !listaValida(l.transiciones) || typeof p.puede_publicar !== "boolean"
    || typeof p.catalogo_activado !== "boolean"
    || !subconjunto(p.transiciones_obligatorias, l.transiciones)
    || (publicacion && (!subconjunto(p.transiciones_guardadas, l.transiciones)
      || !subconjunto(p.transiciones_obligatorias, p.transiciones_guardadas)))
    || (publicacion && (typeof p.recibo_ref !== "string" || !p.recibo_ref.trim() || p.recibo_ref.length > 512
      || typeof p.reutilizada !== "boolean" || !fechaPublicacionValida(p.publicada_en)))
    || (!publicacion && p.publicada_en !== undefined && p.publicada_en !== "" && !fechaPublicacionValida(p.publicada_en))
    || (!publicacion && (p.recibo_ref !== undefined || (p.reutilizada !== undefined && p.reutilizada !== false)))) {
    throw new TypeError("política de situaciones incompatible");
  }
  return Object.freeze({ ...p, limites: Object.freeze({ ...l, transiciones: Object.freeze([...l.transiciones]) }),
    transiciones_guardadas: Object.freeze([...p.transiciones_guardadas]),
    transiciones_efectivas: Object.freeze([...p.transiciones_efectivas]),
    transiciones_obligatorias: Object.freeze([...p.transiciones_obligatorias]) });
}

export function validarPublicacionTransiciones(cuerpo) {
  if (!cuerpo || Object.keys(cuerpo).sort().join(",") !== "clave_idempotencia,motivo,transiciones,version_esperada"
    || !Number.isSafeInteger(cuerpo.version_esperada) || cuerpo.version_esperada < 1
    || !listaValida(cuerpo.transiciones)
    || typeof cuerpo.motivo !== "string" || cuerpo.motivo !== cuerpo.motivo.trim()
    || !cuerpo.motivo || new TextEncoder().encode(cuerpo.motivo).length > 1000
    || typeof cuerpo.clave_idempotencia !== "string"
    || !/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/iu.test(cuerpo.clave_idempotencia)) {
    throw new TypeError("publicación de situaciones incompatible");
  }
  return cuerpo;
}

async function leerJSON(respuesta) {
  if (!/^application\/json(?:;|$)/iu.test(respuesta.headers?.get?.("content-type") || "")) throw new TypeError("respuesta incompatible");
  const longitud = respuesta.headers?.get?.("content-length");
  if (longitud && (!/^\d+$/u.test(longitud) || Number(longitud) > MAXIMO)) throw new TypeError("respuesta excesiva");
  const lector = respuesta.body?.getReader?.();
  if (!lector) throw new TypeError("respuesta incompatible");
  const partes = []; let total = 0;
  try {
    for (;;) {
      const { done, value } = await lector.read();
      if (done) break;
      total += value.byteLength;
      if (total > MAXIMO) throw new TypeError("respuesta excesiva");
      partes.push(value);
    }
  } finally { try { lector.releaseLock(); } catch {} }
  const bytes = new Uint8Array(total); let posicion = 0;
  for (const parte of partes) { bytes.set(parte, posicion); posicion += parte.byteLength; }
  return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes));
}

export function crearClienteReglasSituacionBolsa({ fetchImpl = globalThis.fetch } = {}) {
  if (typeof fetchImpl !== "function") throw new TypeError("cliente no disponible");
  async function solicitar(method, { signal, cuerpo } = {}) {
    const controlador = new AbortController();
    const abortar = () => controlador.abort();
    if (signal?.aborted) abortar(); else signal?.addEventListener?.("abort", abortar, { once: true });
    const temporizador = setTimeout(abortar, 15_000);
    try {
      const respuesta = await fetchImpl(RUTA_REGLAS_SITUACION, { method, signal: controlador.signal,
        credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
        headers: method === "GET" ? { Accept: "application/json" } : { Accept: "application/json", "Content-Type": "application/json" },
        ...(method === "POST" ? { body: JSON.stringify(validarPublicacionTransiciones(cuerpo)) } : {}) });
      if (!respuesta || respuesta.redirected || !Number.isInteger(respuesta.status)) throw new TypeError("respuesta incompatible");
      if (!respuesta.ok) throw Object.assign(new Error("operación rechazada"), { estado: respuesta.status });
      if (method === "GET" ? respuesta.status !== 200 : ![200, 201].includes(respuesta.status)) throw new TypeError("estado incompatible");
      const politica = validarPoliticaTransiciones(await leerJSON(respuesta), { publicacion: method === "POST" });
      if (method === "POST" && politica.reutilizada !== (respuesta.status === 200)) throw new TypeError("recibo incompatible");
      if (method === "POST" && (politica.transiciones_guardadas.length !== cuerpo.transiciones.length
        || !cuerpo.transiciones.every((par) => politica.transiciones_guardadas.includes(par)))) throw new TypeError("recibo ajeno a la publicación");
      return politica;
    } finally { clearTimeout(temporizador); signal?.removeEventListener?.("abort", abortar); }
  }
  return Object.freeze({ consultar: ({ signal } = {}) => solicitar("GET", { signal }),
    publicar: (cuerpo, { signal } = {}) => solicitar("POST", { signal, cuerpo }) });
}
