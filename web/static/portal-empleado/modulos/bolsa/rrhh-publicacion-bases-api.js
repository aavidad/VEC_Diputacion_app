/**
 * Cliente RRHH de la publicación oficial de las bases de Bolsa.
 *
 * GET lista las versiones que el perfil puede declarar; POST registra la
 * declaración. Quién declara y el motivo los fija el servidor con la identidad
 * mTLS: el navegador solo envía los datos del anuncio. Sin cookies ni
 * almacenamiento web; la ruta no lleva parámetros.
 */
export const RUTA_PUBLICACIONES_OFICIALES = "/api/vec/bolsa/convocatorias/publicaciones-oficiales";
export const MEDIOS_PUBLICACION = Object.freeze(["bop_granada", "tablon_edictos_electronico", "boe", "boja"]);
export const MEDIO_SIN_BOLETIN = "tablon_edictos_electronico";

const MAXIMO = 512 * 1024;
const MAXIMO_ERROR = 8 * 1024;
const MAXIMO_ELEMENTOS = 1000;
const PLAZO_MS = 15_000;
const HUELLA = /^[a-f0-9]{64}$/u;
const CONVOCATORIA = /^cv1_[a-f0-9]{64}_v[1-9][0-9]{0,8}$/u;
const PUBLICACION = /^publicacion_bases:[a-f0-9]{64}$/u;
const ANUNCIO = /^[A-Za-z0-9][A-Za-z0-9./_-]{0,79}$/u;
const FECHA = /^\d{4}-\d\d-\d\d$/u;
const INSTANTE = /^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d{1,9})?(?:Z|[+-]\d\d:\d\d)$/u;
const CLAVE = /^[A-Za-z0-9:_-]{8,128}$/u;
const CODIGO = /^[a-z_]{1,64}$/u;

const referencia = (valor) => typeof valor === "string" && valor.length >= 1 && valor.length <= 512
  && !/[\s\u0000-\u001f\u007f]/u.test(valor);
const referenciaOpcional = (valor) => valor === null || valor === undefined || referencia(valor);
const texto = (valor, maximo) => typeof valor === "string" && valor.trim() !== "" && valor.length <= maximo;
const instante = (valor) => typeof valor === "string" && INSTANTE.test(valor) && Number.isFinite(Date.parse(valor));

/** Fecha civil AAAA-MM-DD existente en el calendario. */
export function fechaCivilValida(valor) {
  if (typeof valor !== "string" || !FECHA.test(valor)) return false;
  const fecha = new Date(`${valor}T00:00:00Z`);
  return Number.isFinite(fecha.getTime()) && fecha.toISOString().slice(0, 10) === valor;
}

/** Enlace opcional al anuncio: null o https:// de 12 a 512 caracteres. */
export function direccionValida(valor) {
  if (valor === null) return true;
  if (typeof valor !== "string" || valor.length < 12 || valor.length > 512 || !valor.startsWith("https://")
    || /[\s\u0000-\u001f\u007f]/u.test(valor)) return false;
  try { return new URL(valor).protocol === "https:"; } catch { return false; }
}

const boletinValido = (valor) => Number.isSafeInteger(valor) && valor >= 1 && valor <= 99999;

function validarPublicacion(p) {
  if (p === null) return null;
  if (!p || typeof p !== "object" || Array.isArray(p) || !PUBLICACION.test(p.publicacion_ref ?? "")
    || !MEDIOS_PUBLICACION.includes(p.medio)
    || !(p.boletin_numero === null || boletinValido(p.boletin_numero))
    || typeof p.anuncio_ref !== "string" || !ANUNCIO.test(p.anuncio_ref)
    || !fechaCivilValida(p.fecha_publicacion) || !direccionValida(p.direccion ?? null) || !instante(p.registrada_en)) {
    throw new TypeError("publicación oficial incompatible");
  }
  return Object.freeze({ publicacion_ref: p.publicacion_ref, medio: p.medio, boletin_numero: p.boletin_numero,
    anuncio_ref: p.anuncio_ref, fecha_publicacion: p.fecha_publicacion, direccion: p.direccion ?? null,
    registrada_en: p.registrada_en });
}

/** Valida la lista del GET y devuelve una copia congelada. */
export function validarListaPublicaciones(sobre) {
  const data = sobre?.data;
  if (!data || typeof data !== "object" || !Array.isArray(data.items) || data.items.length > MAXIMO_ELEMENTOS
    || typeof data.completa !== "boolean") throw new TypeError("lista de publicaciones incompatible");
  const items = data.items.map((item) => {
    if (!item || typeof item !== "object" || Array.isArray(item) || !CONVOCATORIA.test(item.convocatoria_ref ?? "")
      || !Number.isSafeInteger(item.secuencia) || item.secuencia < 1
      || !texto(item.identificador_publico, 512) || !texto(item.titulo, 2048)
      || typeof item.huella_version_sha256 !== "string" || !HUELLA.test(item.huella_version_sha256)
      || !(item.plazo_inscripcion_abre_en === null || instante(item.plazo_inscripcion_abre_en))) {
      throw new TypeError("convocatoria de publicación incompatible");
    }
    return Object.freeze({ convocatoria_ref: item.convocatoria_ref, secuencia: item.secuencia,
      identificador_publico: item.identificador_publico, titulo: item.titulo,
      huella_version_sha256: item.huella_version_sha256, plazo_inscripcion_abre_en: item.plazo_inscripcion_abre_en,
      publicacion: validarPublicacion(item.publicacion ?? null) });
  });
  return Object.freeze({ items: Object.freeze(items), completa: data.completa });
}

/** Valida el resguardo del POST (201 nuevo o 200 repetición del mismo envío). */
export function validarResguardoPublicacion(sobre, enviado) {
  const r = sobre?.data;
  if (!r || typeof r !== "object" || Array.isArray(r) || !PUBLICACION.test(r.publicacion_ref ?? "")
    || !referencia(r.recibo_ref) || !referenciaOpcional(r.evento_ref) || !referenciaOpcional(r.auditoria_ref)
    || r.convocatoria_ref !== enviado.convocatoria_ref || r.medio !== enviado.medio
    || r.boletin_numero !== enviado.boletin_numero || r.anuncio_ref !== enviado.anuncio_ref
    || r.fecha_publicacion !== enviado.fecha_publicacion || (r.direccion ?? null) !== enviado.direccion
    || (r.sustituye_ref ?? null) !== enviado.sustituye_ref
    || !Number.isSafeInteger(r.secuencia) || r.secuencia < 1 || !instante(r.registrada_en)
    || typeof r.reutilizada !== "boolean") {
    throw new TypeError("resguardo de publicación incompatible");
  }
  return Object.freeze({ ...r, direccion: r.direccion ?? null, sustituye_ref: r.sustituye_ref ?? null });
}

/**
 * Comprueba el cuerpo antes de enviarlo: exactamente las claves del contrato.
 * Devuelve la lista de campos con error (vacía si es válido).
 */
export function erroresCuerpoPublicacion(cuerpo, hoy) {
  const errores = [];
  if (!MEDIOS_PUBLICACION.includes(cuerpo.medio)) errores.push("medio");
  if (cuerpo.medio === MEDIO_SIN_BOLETIN ? cuerpo.boletin_numero !== null : !boletinValido(cuerpo.boletin_numero)) {
    errores.push("boletin_numero");
  }
  if (typeof cuerpo.anuncio_ref !== "string" || !ANUNCIO.test(cuerpo.anuncio_ref)) errores.push("anuncio_ref");
  if (!fechaCivilValida(cuerpo.fecha_publicacion) || (fechaCivilValida(hoy) && cuerpo.fecha_publicacion > hoy)) {
    errores.push("fecha_publicacion");
  }
  if (!direccionValida(cuerpo.direccion)) errores.push("direccion");
  return errores;
}

/** Clave de idempotencia nueva para un intento de envío. */
export function nuevaClavePublicacion(criptografia = globalThis.crypto) {
  if (typeof criptografia?.randomUUID !== "function") throw new TypeError("generador criptográfico no disponible");
  return `publicacion-bases:${criptografia.randomUUID()}`;
}

async function leerJSONAcotado(respuesta, maximo) {
  if (!/^application\/json(?:\s*;|\s*$)/iu.test(respuesta.headers?.get?.("content-type") || "")) {
    throw new TypeError("respuesta de publicación incompatible");
  }
  const longitud = respuesta.headers?.get?.("content-length");
  if (longitud && (!/^\d+$/u.test(longitud) || Number(longitud) > maximo)) throw new TypeError("respuesta excesiva");
  const lector = respuesta.body?.getReader?.();
  if (!lector) throw new TypeError("respuesta de publicación incompatible");
  const partes = [];
  let total = 0;
  try {
    for (;;) {
      const { done, value } = await lector.read();
      if (done) break;
      if (!(value instanceof Uint8Array) || (total += value.byteLength) > maximo) throw new TypeError("respuesta excesiva");
      partes.push(value);
    }
  } catch (error) {
    await lector.cancel().catch(() => {});
    throw error;
  } finally { try { lector.releaseLock(); } catch { /* ya liberado */ } }
  const bytes = new Uint8Array(total);
  let posicion = 0;
  for (const parte of partes) { bytes.set(parte, posicion); posicion += parte.byteLength; }
  return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes));
}

/** Error con el estado HTTP y el código gobernado del servidor (nunca se muestra tal cual). */
export class ErrorPublicacionBases extends Error {
  constructor(estado, codigo = "") {
    super(codigo || `estado_${estado}`);
    this.name = "ErrorPublicacionBases";
    this.estado = estado;
    this.codigo = codigo;
  }
}

async function codigoError(respuesta) {
  try {
    const codigo = (await leerJSONAcotado(respuesta, MAXIMO_ERROR))?.error?.codigo;
    return typeof codigo === "string" && CODIGO.test(codigo) ? codigo : "";
  } catch { return ""; }
}

export function crearClientePublicacionBases({ fetchImpl = globalThis.fetch } = {}) {
  if (typeof fetchImpl !== "function") throw new TypeError("cliente de publicación de bases no disponible");
  async function pedir(opciones, signal) {
    const controlador = new AbortController();
    const abortar = () => controlador.abort();
    if (signal?.aborted) abortar();
    else signal?.addEventListener?.("abort", abortar, { once: true });
    const temporizador = setTimeout(abortar, PLAZO_MS);
    try {
      // Mismo origen: el certificado TLS cliente identifica a quien declara.
      // JS no añade credenciales ni cookies y el servidor vuelve a autorizar.
      const respuesta = await fetchImpl(RUTA_PUBLICACIONES_OFICIALES, { ...opciones, signal: controlador.signal,
        credentials: "same-origin", mode: "same-origin", cache: "no-store",
        redirect: "error", referrerPolicy: "no-referrer" });
      if (!respuesta || respuesta.redirected || !Number.isInteger(respuesta.status)) throw new TypeError("respuesta incompatible");
      if (!respuesta.ok) throw new ErrorPublicacionBases(respuesta.status, await codigoError(respuesta));
      return { estado: respuesta.status, cuerpo: await leerJSONAcotado(respuesta, MAXIMO) };
    } finally {
      clearTimeout(temporizador);
      signal?.removeEventListener?.("abort", abortar);
    }
  }
  return Object.freeze({
    async listar({ signal } = {}) {
      const { estado, cuerpo } = await pedir({ method: "GET", headers: { Accept: "application/json" } }, signal);
      if (estado !== 200) throw new TypeError("estado de publicaciones incompatible");
      return validarListaPublicaciones(cuerpo);
    },
    async declarar(cuerpo, { claveIdempotencia, signal } = {}) {
      if (typeof claveIdempotencia !== "string" || !CLAVE.test(claveIdempotencia)) throw new TypeError("clave de envío no válida");
      const enviado = Object.freeze({
        convocatoria_ref: cuerpo?.convocatoria_ref, huella_version_sha256: cuerpo?.huella_version_sha256,
        medio: cuerpo?.medio, boletin_numero: cuerpo?.boletin_numero, anuncio_ref: cuerpo?.anuncio_ref,
        fecha_publicacion: cuerpo?.fecha_publicacion, direccion: cuerpo?.direccion ?? null,
        sustituye_ref: cuerpo?.sustituye_ref ?? null,
      });
      if (!CONVOCATORIA.test(enviado.convocatoria_ref ?? "") || !HUELLA.test(enviado.huella_version_sha256 ?? "")
        || !(enviado.sustituye_ref === null || PUBLICACION.test(enviado.sustituye_ref))
        || erroresCuerpoPublicacion(enviado).length > 0) throw new TypeError("declaración de publicación no válida");
      const { estado, cuerpo: respuesta } = await pedir({ method: "POST", body: JSON.stringify(enviado),
        headers: { "Content-Type": "application/json", Accept: "application/json", "Idempotency-Key": claveIdempotencia } }, signal);
      if (estado !== 200 && estado !== 201) throw new TypeError("estado de publicación incompatible");
      return validarResguardoPublicacion(respuesta, enviado);
    },
  });
}

/**
 * Sondeo de disponibilidad para el menú: «disponible» (200), «no_habilitada»
 * (503: la entrada se muestra con el estado «no disponible») u «oculta»
 * (403, 404 o cualquier otra respuesta: no se ofrece).
 */
export async function sondearPublicacionBases(cliente, { signal } = {}) {
  try {
    await cliente.listar({ signal });
    return "disponible";
  } catch (error) {
    return error?.estado === 503 ? "no_habilitada" : "oculta";
  }
}
