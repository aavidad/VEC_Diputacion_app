/**
 * Transporte de la carga de bolsas desde CONVOCA. El fichero viaja en base64
 * dentro de un JSON cerrado; la identidad la pone la frontera mTLS del
 * servidor, nunca este cliente (sin cabeceras de autorización ni almacenamiento).
 */
export const RUTA_VISTA_PREVIA = "/api/vec/bolsa/cargas-convoca/vista-previa";
export const RUTA_CONFIRMAR = "/api/vec/bolsa/cargas-convoca";
export const MAXIMO_FICHERO = 1024 * 1024;
export const PLAZO_MS = 60000;
const MAXIMO_RESPUESTA = 12 * 1024 * 1024;
const ESQUEMA_VISTA = "vec.bolsa.rrhh.carga_convoca.vista_previa.v1";
const ESQUEMA_RECIBO = "vec.bolsa.rrhh.carga_convoca.recibo.v1";
const HUELLA = /^[a-f0-9]{64}$/u;
const CODIGO = /^[a-z][a-z0-9_]{1,63}$/u;
const EXTENSION = /\.(xls|xlsx)$/iu;

/** Error de la carga con el estado HTTP y el código estable del servidor. */
export class ErrorCargaConvoca extends Error {
  constructor(estado, codigo) {
    super(codigo);
    this.estado = estado;
    this.codigo = codigo;
  }
}

/** Comprueba el fichero antes de leerlo: nombre, extensión y tamaño. */
export function comprobarFichero(fichero) {
  if (!fichero || typeof fichero.name !== "string" || typeof fichero.size !== "number") return "faltaFichero";
  if (!EXTENSION.test(fichero.name)) return "ficheroTipo";
  if (fichero.size <= 0 || fichero.size > MAXIMO_FICHERO) return "ficheroGrande";
  return "";
}

/** Bytes a base64 por trozos (sin desbordar la pila con ficheros grandes). */
export function bytesABase64(bytes) {
  let binario = "";
  for (let i = 0; i < bytes.length; i += 0x8000) {
    binario += String.fromCharCode(...bytes.subarray(i, i + 0x8000));
  }
  return btoa(binario);
}

const entero = (v) => Number.isSafeInteger(v) && v >= 0;
const texto = (v, max = 1024) => typeof v === "string" && v.length <= max;

function validarFila(f) {
  return f && typeof f === "object" && entero(f.numero) && (f.estado === "aceptada" || f.estado === "rechazada")
    && (f.posicion === undefined || entero(f.posicion))
    && ["documento", "primer_apellido", "segundo_apellido", "nombre", "experiencia", "formacion", "total"]
      .every((c) => f[c] === undefined || texto(f[c], 512))
    && Array.isArray(f.errores) && f.errores.every((e) => texto(e?.campo, 128) && CODIGO.test(e?.codigo ?? ""))
    && Array.isArray(f.avisos) && f.avisos.every((a) => CODIGO.test(a));
}

export function validarVistaPrevia(sobre) {
  const d = sobre?.data;
  if (d?.esquema !== ESQUEMA_VISTA || !HUELLA.test(d.huella_sha256 ?? "") || !texto(d.nombre_fichero, 255)
    || ![d.filas_leidas, d.aceptadas, d.rechazadas, d.con_avisos].every(entero)
    || !(d.bloqueo === "" || CODIGO.test(d.bloqueo ?? "")) || !Array.isArray(d.filas) || !d.filas.every(validarFila)) {
    throw new TypeError("vista previa incompatible");
  }
  return Object.freeze({ ...d, filas: Object.freeze(d.filas.map((f) => Object.freeze({ ...f }))) });
}

export function validarRecibo(sobre) {
  const d = sobre?.data;
  if (d?.esquema !== ESQUEMA_RECIBO || !texto(d.bolsa_ref, 512) || !entero(d.version_bolsa) || !HUELLA.test(d.huella_sha256 ?? "")
    || typeof d.reutilizada !== "boolean" || !entero(d.filas_cargadas) || !entero(d.filas_excluidas)
    || !Array.isArray(d.pendientes_revision) || !d.pendientes_revision.every((p) => entero(p?.fila) && CODIGO.test(p?.motivo ?? ""))
    || !Array.isArray(d.sustituye_a) || !texto(d.auditoria_ref, 128) || !Number.isFinite(Date.parse(d.confirmada_en))) {
    throw new TypeError("recibo de carga incompatible");
  }
  return Object.freeze({ ...d });
}

async function leerJSON(respuesta) {
  if (!/^application\/json(?:;|$)/iu.test(respuesta.headers?.get?.("content-type") || "")) return null;
  const longitud = Number(respuesta.headers.get("content-length") || 0);
  if (longitud > MAXIMO_RESPUESTA) throw new TypeError("respuesta excesiva");
  const contenido = await respuesta.text();
  if (contenido.length > MAXIMO_RESPUESTA) throw new TypeError("respuesta excesiva");
  return JSON.parse(contenido);
}

export function crearClienteCargaConvoca({ fetchImpl = globalThis.fetch, plazoMs = PLAZO_MS } = {}) {
  if (typeof fetchImpl !== "function" || !Number.isSafeInteger(plazoMs) || plazoMs < 1) {
    throw new TypeError("cliente de carga no disponible");
  }
  async function enviar(ruta, cuerpo, signal) {
    const controlador = new AbortController();
    const cancelar = () => controlador.abort();
    signal?.addEventListener("abort", cancelar, { once: true });
    const temporizador = setTimeout(cancelar, plazoMs);
    try {
      if (signal?.aborted) controlador.abort();
      const respuesta = await fetchImpl(ruta, {
        method: "POST", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error",
        referrerPolicy: "no-referrer", headers: { "Content-Type": "application/json", Accept: "application/json" },
        body: JSON.stringify(cuerpo), signal: controlador.signal,
      });
      const json = await leerJSON(respuesta).catch(() => null);
      if (!respuesta.ok) {
        const codigo = json?.error?.codigo;
        throw new ErrorCargaConvoca(respuesta.status, CODIGO.test(codigo ?? "") ? codigo : "servicio_no_disponible");
      }
      return json;
    } finally {
      clearTimeout(temporizador);
      signal?.removeEventListener("abort", cancelar);
    }
  }
  return Object.freeze({
    async previsualizar({ nombre, base64, signal }) {
      return validarVistaPrevia(await enviar(RUTA_VISTA_PREVIA, { nombre_fichero: nombre, contenido_base64: base64 }, signal));
    },
    async confirmar({ nombre, base64, categoria, excluir, signal }) {
      const cuerpo = { nombre_fichero: nombre, contenido_base64: base64, categoria };
      if (excluir) cuerpo.excluir_filas_con_errores = true;
      return validarRecibo(await enviar(RUTA_CONFIRMAR, cuerpo, signal));
    },
  });
}
