/** Transporte del marcaje remoto. La identidad, el canal y el reloj pertenecen al servidor. */
export const RUTA_DISPONIBILIDAD_REMOTA = "/api/interna/cronos/marcajes/remoto/disponibilidad";
export const RUTA_MARCAJE_REMOTO = "/api/interna/cronos/marcajes/remoto";
export const RUTA_RECUPERACION_REMOTA = "/api/interna/cronos/marcajes/remoto/recibo";

const MOVIMIENTOS = new Set(["entrada", "salida", "inicio_pausa", "fin_pausa"]);
const CLAVE = /^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$/u;
const REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{0,159}$/u;
const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/u;
const MAXIMO_RESPUESTA = 8192;

export class ErrorClienteRemotoCronos extends Error {
  constructor(codigo, estado = 0, incierto = false) {
    super(codigo);
    this.name = "ErrorClienteRemotoCronos";
    this.codigo = codigo;
    this.estado = estado;
    this.incierto = incierto;
  }
}

function registro(valor) {
  return valor !== null && typeof valor === "object" && !Array.isArray(valor)
    && (Object.getPrototypeOf(valor) === Object.prototype || Object.getPrototypeOf(valor) === null);
}
function campos(valor, nombres) {
  return registro(valor) && Object.keys(valor).length === nombres.length
    && nombres.every((nombre) => Object.hasOwn(valor, nombre));
}
function instanteUTC(valor) {
  return typeof valor === "string" && INSTANTE.test(valor) && !Number.isNaN(Date.parse(valor));
}
function validarOpciones(opciones) {
  if (!registro(opciones) || Object.keys(opciones).some((clave) => clave !== "signal")
    || (opciones.signal !== undefined && !(opciones.signal instanceof AbortSignal))) {
    throw new TypeError("opciones del cliente remoto no válidas");
  }
  return opciones.signal;
}
export function validarDisponibilidadRemota(valor) {
  if (!registro(valor) || typeof valor.continuidad_confirmada !== "boolean") {
    throw new ErrorClienteRemotoCronos("continuidad_no_confirmada");
  }
  if (!Array.isArray(valor.movimientos_permitidos)) {
    throw new ErrorClienteRemotoCronos("secuencia_no_permitida");
  }
  if (valor.movimientos_permitidos.length > MOVIMIENTOS.size
    || valor.movimientos_permitidos.some((movimiento) => !MOVIMIENTOS.has(movimiento))
    || new Set(valor.movimientos_permitidos).size !== valor.movimientos_permitidos.length) {
    throw new ErrorClienteRemotoCronos("respuesta_incompatible");
  }
  const movimientos = valor.movimientos_permitidos;
  if (!registro(valor) || typeof valor.autorizado !== "boolean" || typeof valor.motivo !== "string"
    || (valor.autorizado && (!campos(valor, ["autorizado", "continuidad_confirmada", "movimientos_permitidos", "periodo", "motivo"])
      || !campos(valor.periodo, ["desde", "hasta"])
      || !instanteUTC(valor.periodo.desde) || !instanteUTC(valor.periodo.hasta)
      || Date.parse(valor.periodo.desde) >= Date.parse(valor.periodo.hasta)
      || valor.motivo !== (valor.continuidad_confirmada
        ? (movimientos.length ? "autorizado" : "secuencia_no_permitida") : "continuidad_no_confirmada")
      || (!valor.continuidad_confirmada && movimientos.length !== 0)))
    || (!valor.autorizado && (!campos(valor, ["autorizado", "continuidad_confirmada", "movimientos_permitidos", "motivo"])
      || valor.continuidad_confirmada || movimientos.length !== 0 || valor.motivo !== "teletrabajo_no_autorizado"))) {
    throw new ErrorClienteRemotoCronos("respuesta_incompatible");
  }
  return Object.freeze({ autorizado: valor.autorizado, continuidad_confirmada: valor.continuidad_confirmada,
    movimientos_permitidos: Object.freeze([...movimientos]),
    ...(valor.autorizado ? { periodo: Object.freeze({ ...valor.periodo }) } : {}), motivo: valor.motivo });
}
export function validarSolicitudMarcajeRemoto(valor) {
  if (!campos(valor, ["movimiento", "clave_operacion"]) || !MOVIMIENTOS.has(valor.movimiento)
    || typeof valor.clave_operacion !== "string" || !CLAVE.test(valor.clave_operacion)) {
    throw new TypeError("solicitud de marcaje remoto no válida");
  }
  return Object.freeze({ movimiento: valor.movimiento, clave_operacion: valor.clave_operacion });
}
export function validarReciboMarcajeRemoto(valor) {
  if (!campos(valor, ["recibo"]) || !campos(valor.recibo, ["referencia", "instante_utc", "marcaje_original_ref", "replay"])
    || typeof valor.recibo.referencia !== "string" || !REFERENCIA.test(valor.recibo.referencia)
    || typeof valor.recibo.marcaje_original_ref !== "string" || !REFERENCIA.test(valor.recibo.marcaje_original_ref)
    || !instanteUTC(valor.recibo.instante_utc) || typeof valor.recibo.replay !== "boolean") {
    throw new ErrorClienteRemotoCronos("respuesta_incompatible", 200, true);
  }
  return Object.freeze({ ...valor.recibo });
}
async function leerJSON(respuesta, signal, escritura) {
  const estado = respuesta?.status || 0;
  const tipo = respuesta?.headers?.get?.("content-type");
  const largo = respuesta?.headers?.get?.("content-length");
  if (typeof tipo !== "string" || !/^application\/json(?:;\s*charset=utf-8)?$/iu.test(tipo)
    || respuesta?.headers?.get?.("content-encoding")
    || (largo !== null && largo !== undefined && (!/^\d+$/u.test(largo) || Number(largo) > MAXIMO_RESPUESTA))) {
    throw new ErrorClienteRemotoCronos("respuesta_incompatible", estado, escritura);
  }
  if (!respuesta.body || typeof respuesta.body.getReader !== "function") {
    throw new ErrorClienteRemotoCronos("respuesta_incompatible", estado, escritura);
  }
  const lector = respuesta.body.getReader();
  const partes = []; let tamano = 0;
  const cancelar = () => { Promise.resolve(lector.cancel()).catch(() => {}); };
  signal?.addEventListener("abort", cancelar, { once: true });
  try {
    while (true) {
      if (signal?.aborted) throw new ErrorClienteRemotoCronos("operacion_abortada", estado, escritura);
      const paso = await lector.read();
      if (paso.done) break;
      if (!(paso.value instanceof Uint8Array) || paso.value.byteLength === 0) throw new Error();
      tamano += paso.value.byteLength;
      if (tamano > MAXIMO_RESPUESTA) throw new Error();
      partes.push(paso.value);
    }
    if (largo !== null && largo !== undefined && Number(largo) !== tamano) throw new Error();
    const bytes = new Uint8Array(tamano); let posicion = 0;
    for (const parte of partes) { bytes.set(parte, posicion); posicion += parte.byteLength; }
    return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes));
  } catch (error) {
    cancelar();
    if (error instanceof ErrorClienteRemotoCronos) throw error;
    throw new ErrorClienteRemotoCronos("respuesta_incompatible", estado, escritura);
  } finally {
    signal?.removeEventListener("abort", cancelar);
    try { lector.releaseLock(); } catch {}
  }
}
async function ejecutar(fetchImpl, ruta, metodo, body, signal, cabeceras = {}) {
  const escritura = metodo === "POST";
  let respuesta;
  try {
    respuesta = await fetchImpl(ruta, { method: metodo,
      headers: { Accept: "application/json", ...(escritura ? { "Content-Type": "application/json" } : {}), ...cabeceras },
      ...(escritura ? { body: JSON.stringify(body) } : {}), credentials: "same-origin", mode: "same-origin",
      cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal });
  } catch {
    throw new ErrorClienteRemotoCronos(signal?.aborted ? "operacion_abortada" : "red_no_disponible", 0, escritura);
  }
  if (signal?.aborted) throw new ErrorClienteRemotoCronos("operacion_abortada", 0, escritura);
  if (!respuesta || respuesta.redirected) {
    throw new ErrorClienteRemotoCronos("respuesta_incompatible", respuesta?.status || 0, escritura);
  }
  const estado = respuesta.status || 0;
  let datos;
  try { datos = await leerJSON(respuesta, signal, escritura && estado === 200); }
  catch (error) {
    if (estado === 401) throw new ErrorClienteRemotoCronos("autenticacion_requerida", estado);
    if (estado === 403) throw new ErrorClienteRemotoCronos("acceso_denegado", estado);
    if (estado === 503) throw new ErrorClienteRemotoCronos("servicio_no_disponible", estado, escritura);
    throw error;
  }
  if (estado === 200 && respuesta.ok) return datos;
  const codigo = campos(datos, ["error"]) && typeof datos.error === "string" ? datos.error : "";
  if (estado === 401) throw new ErrorClienteRemotoCronos("autenticacion_requerida", estado);
  if (estado === 403) throw new ErrorClienteRemotoCronos(codigo === "teletrabajo_no_autorizado" ? codigo : "acceso_denegado", estado);
  if (estado === 409) throw new ErrorClienteRemotoCronos(codigo === "secuencia_no_permitida" ? codigo : "conflicto", estado);
  if (estado === 404) throw new ErrorClienteRemotoCronos(codigo === "ausencia_confirmada" ? codigo : "respuesta_rechazada", estado);
  if (estado === 503) throw new ErrorClienteRemotoCronos(codigo === "continuidad_no_confirmada" ? codigo : "servicio_no_disponible", estado,
    escritura && codigo !== "continuidad_no_confirmada");
  throw new ErrorClienteRemotoCronos("respuesta_rechazada", estado, escritura && ![400, 422].includes(estado));
}

export function crearClienteRemotoCronosHTTP({ fetchImpl = globalThis.fetch } = {}) {
  if (typeof fetchImpl !== "function") throw new TypeError("fetch remoto no disponible");
  return Object.freeze({
    async disponibilidad(opciones = {}) {
      const signal = validarOpciones(opciones);
      return validarDisponibilidadRemota(await ejecutar(fetchImpl, RUTA_DISPONIBILIDAD_REMOTA, "GET", null, signal));
    },
    async registrar(solicitud, opciones = {}) {
      const entrada = validarSolicitudMarcajeRemoto(solicitud);
      const signal = validarOpciones(opciones);
      return validarReciboMarcajeRemoto(await ejecutar(fetchImpl, RUTA_MARCAJE_REMOTO, "POST", entrada, signal));
    },
    async recuperar(solicitud, opciones = {}) {
      const entrada = validarSolicitudMarcajeRemoto(solicitud);
      const signal = validarOpciones(opciones);
      return validarReciboMarcajeRemoto(await ejecutar(fetchImpl, RUTA_RECUPERACION_REMOTA, "GET", null, signal, {
        "X-Cronos-Clave-Operacion": entrada.clave_operacion,
        "X-Cronos-Movimiento": entrada.movimiento,
      }));
    },
  });
}
