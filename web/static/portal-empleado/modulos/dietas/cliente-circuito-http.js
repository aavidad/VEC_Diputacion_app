const RUTA_CIRCUITO = "/api/vec/dietas/comisiones/circuito";
const MAXIMO_CUERPO_BYTES = 4 * 1024;
const MAXIMO_RESPUESTA_BYTES = 64 * 1024;
const MAXIMO_FRAGMENTOS = 128;
const codificador = new TextEncoder();

export class ErrorClienteCircuitoDietas extends Error {
  constructor(codigo, estado = 0, resultadoIndeterminado = false) {
    super(`cliente de circuito de Dietas: ${codigo}`);
    this.name = "ErrorClienteCircuitoDietas";
    this.codigo = codigo;
    this.estado = estado;
    this.resultadoIndeterminado = resultadoIndeterminado;
    Object.freeze(this);
  }
}

const fallo = (codigo, estado = 0, resultadoIndeterminado = false) =>
  new ErrorClienteCircuitoDietas(codigo, estado, resultadoIndeterminado);
const registro = (valor) => valor !== null && typeof valor === "object" && !Array.isArray(valor)
  && (Object.getPrototypeOf(valor) === Object.prototype || Object.getPrototypeOf(valor) === null);
const referencia = (valor, prefijo = "dco_") => typeof valor === "string"
  && new RegExp(`^${prefijo}[A-Za-z0-9_-]{22,128}$`, "u").test(valor);
const texto = (valor, minimo, maximo) => typeof valor === "string" && valor.trim() === valor
  && valor.length >= minimo && codificador.encode(valor).byteLength <= maximo
  && !/[\x00-\x1f\x7f]/u.test(valor);
const etapaValida = (valor) => ["revision", "autorizacion", "liquidacion", "fiscalizacion"].includes(valor);
const fechaCivil = (valor) => {
  if (typeof valor !== "string" || !/^\d{4}-\d{2}-\d{2}$/u.test(valor)) return false;
  const [ano, mes, dia] = valor.split("-").map(Number);
  const fecha = new Date(Date.UTC(ano, mes - 1, dia));
  return fecha.getUTCFullYear() === ano && fecha.getUTCMonth() === mes - 1 && fecha.getUTCDate() === dia;
};
function validarSignal(signal) {
  if (signal === undefined) return undefined;
  if (!signal || typeof signal !== "object" || typeof signal.aborted !== "boolean"
    || typeof signal.addEventListener !== "function" || typeof signal.removeEventListener !== "function") throw fallo("signal_no_valida");
  if (signal.aborted) throw fallo("operacion_abortada");
  return signal;
}
function validarOpciones(opciones = {}) {
  if (!registro(opciones) || Object.keys(opciones).some((clave) => clave !== "signal")) throw fallo("opciones_no_validas");
  return validarSignal(opciones.signal);
}
function validarConsulta(consulta) {
  if (!registro(consulta) || Object.keys(consulta).some((clave) => !["etapa", "fecha_desde", "fecha_hasta", "limit", "cursor"].includes(clave))
    || !etapaValida(consulta.etapa)) throw new TypeError("consulta del circuito de Dietas no válida");
  const { etapa, fecha_desde, fecha_hasta, limit = 20, cursor } = consulta;
  if ((fecha_desde !== undefined && !fechaCivil(fecha_desde)) || (fecha_hasta !== undefined && !fechaCivil(fecha_hasta))
    || (fecha_desde && fecha_hasta && fecha_desde > fecha_hasta) || !Number.isSafeInteger(limit) || limit < 1 || limit > 50
    || (cursor !== undefined && !referencia(cursor))) throw new TypeError("consulta del circuito de Dietas no válida");
  return Object.freeze({ etapa, ...(fecha_desde ? { fecha_desde } : {}), ...(fecha_hasta ? { fecha_hasta } : {}), limit, ...(cursor ? { cursor } : {}) });
}
function validarDecision(entrada) {
  const campos = ["etapa", "decision", "motivo", "clave_idempotencia", "version_esperada"];
  if (!registro(entrada) || Object.keys(entrada).some((clave) => !campos.includes(clave)) || !etapaValida(entrada.etapa)
    || !["aprobar", "devolver"].includes(entrada.decision) || typeof entrada.clave_idempotencia !== "string"
    || !/^[A-Za-z0-9:_-]{16,128}$/u.test(entrada.clave_idempotencia) || !Number.isSafeInteger(entrada.version_esperada)
    || entrada.version_esperada < 1 || (entrada.motivo !== undefined && !texto(entrada.motivo, 0, 600))
    || (entrada.decision === "devolver" && !texto(entrada.motivo, 3, 600))) throw new TypeError("decisión del circuito de Dietas no válida");
  return Object.freeze({ etapa: entrada.etapa, decision: entrada.decision, motivo: entrada.motivo ?? "", clave_idempotencia: entrada.clave_idempotencia, version_esperada: entrada.version_esperada });
}
function validarComision(comision, referenciaEsperada) {
  if (!registro(comision) || Object.keys(comision).some((clave) => !["referencia", "estado", "version"].includes(clave))
    || !referencia(comision.referencia) || (referenciaEsperada && comision.referencia !== referenciaEsperada)
    || typeof comision.estado !== "string" || !["enviado_pendiente_revision", "pendiente_autorizacion", "pendiente_liquidacion", "pendiente_fiscalizacion", "fiscalizada", "devuelta"].includes(comision.estado)
    || !Number.isSafeInteger(comision.version) || comision.version < 1) throw new TypeError("comisión de circuito incompatible");
  return Object.freeze({ ...comision });
}
function validarItem(item) {
  if (!registro(item) || Object.keys(item).some((clave) => !["referencia", "estado", "version", "fecha_inicio", "fecha_fin"].includes(clave))
    || !referencia(item.referencia) || typeof item.estado !== "string" || !Number.isSafeInteger(item.version) || item.version < 1
    || !fechaCivil(item.fecha_inicio) || !fechaCivil(item.fecha_fin) || item.fecha_fin < item.fecha_inicio) throw new TypeError("fila del circuito incompatible");
  return Object.freeze({ ...item });
}
function validarPagina(pagina) {
  if (!registro(pagina) || Object.keys(pagina).some((clave) => clave !== "items" && clave !== "siguiente_cursor")
    || !Array.isArray(pagina.items) || pagina.items.length > 50 || (pagina.siguiente_cursor !== undefined && !referencia(pagina.siguiente_cursor))) throw new TypeError("página del circuito incompatible");
  return Object.freeze({ items: Object.freeze(pagina.items.map(validarItem)), ...(pagina.siguiente_cursor ? { siguiente_cursor: pagina.siguiente_cursor } : {}) });
}
function validarResultado(resultado, referenciaEsperada, versionEsperada) {
  if (!registro(resultado) || Object.keys(resultado).some((clave) => clave !== "comision" && clave !== "recibo")) throw new TypeError("resultado del circuito incompatible");
  const comision = validarComision(resultado.comision, referenciaEsperada);
  const recibo = resultado.recibo;
  if (!registro(recibo) || Object.keys(recibo).some((clave) => !["referencia", "version", "registrado_en", "repeticion"].includes(clave))
    || !referencia(recibo.referencia, "rcd_") || !Number.isSafeInteger(recibo.version) || recibo.version < versionEsperada
    || typeof recibo.registrado_en !== "string" || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$/u.test(recibo.registrado_en)
    || !Number.isFinite(Date.parse(recibo.registrado_en)) || typeof recibo.repeticion !== "boolean") throw new TypeError("recibo del circuito incompatible");
  return Object.freeze({ comision, recibo: Object.freeze({ ...recibo }) });
}
async function cancelar(respuesta, lector) { try { await (lector?.cancel?.() ?? respuesta?.body?.cancel?.()); } catch {} }
async function leerJSON(respuesta, signal) {
  const estado = respuesta?.status || 0;
  const longitud = respuesta?.headers?.get?.("content-length");
  if (longitud !== null && longitud !== undefined && (!/^\d+$/u.test(longitud) || Number(longitud) > MAXIMO_RESPUESTA_BYTES)) { await cancelar(respuesta); throw fallo("respuesta_excesiva", estado); }
  if (!/^application\/json(?:;\s*charset=utf-8)?$/iu.test(respuesta?.headers?.get?.("content-type") || "") || respuesta?.headers?.get?.("content-encoding") || !respuesta?.body?.getReader) { await cancelar(respuesta); throw fallo("respuesta_incompatible", estado); }
  const lector = respuesta.body.getReader(); let total = 0; const partes = [];
  const abortar = () => { Promise.resolve(lector.cancel()).catch(() => {}); };
  signal?.addEventListener("abort", abortar, { once: true });
  try {
    while (true) {
      if (signal?.aborted) throw fallo("operacion_abortada", estado);
      const parte = await lector.read();
      if (parte.done) break;
      if (!(parte.value instanceof Uint8Array) || parte.value.byteLength === 0 || ++total > MAXIMO_FRAGMENTOS || partes.reduce((n, valor) => n + valor.byteLength, 0) + parte.value.byteLength > MAXIMO_RESPUESTA_BYTES) throw fallo("respuesta_excesiva", estado);
      partes.push(parte.value);
    }
    const bytes = new Uint8Array(partes.reduce((n, parte) => n + parte.byteLength, 0)); let posicion = 0;
    for (const parte of partes) { bytes.set(parte, posicion); posicion += parte.byteLength; }
    if (longitud !== null && longitud !== undefined && Number(longitud) !== bytes.byteLength) throw fallo("respuesta_incompatible", estado);
    try { return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes)); } catch { throw fallo("json_no_valido", estado); }
  } finally { signal?.removeEventListener("abort", abortar); try { lector.releaseLock?.(); } catch {} }
}
function codigoError(cuerpo, estado) {
  const codigo = typeof cuerpo?.error === "string" && cuerpo.error.startsWith("dietas.error.") ? cuerpo.error.slice("dietas.error.".length) : "";
  return ({ 400: ["peticion_invalida"], 403: ["acceso_denegado"], 404: ["no_encontrada"], 409: ["conflicto_estado"], 503: ["resultado_incierto", "no_disponible"] }[estado] || []).includes(codigo) ? codigo : "respuesta_rechazada";
}
async function ejecutar(fetchImpl, ruta, opciones, estados, signal, escritura, validar) {
  let respuesta;
  try { respuesta = await fetchImpl(ruta, { ...opciones, credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal }); }
  catch { throw fallo(signal?.aborted ? "operacion_abortada" : "red_no_disponible", 0, escritura && !signal?.aborted); }
  if (!respuesta || respuesta.redirected) { await cancelar(respuesta); throw fallo("respuesta_rechazada", respuesta?.status || 0, escritura); }
  const estado = respuesta.status || 0; let cuerpo;
  try { cuerpo = await leerJSON(respuesta, signal); } catch (error) {
    if (escritura && !signal?.aborted && (estado >= 500 || (estado >= 200 && estado < 300))) throw fallo(error.codigo || "respuesta_incompatible", estado, true);
    throw error;
  }
  if (!estados.includes(estado) || respuesta.ok !== true) throw fallo(codigoError(cuerpo, estado), estado, escritura && estado >= 500);
  try { return validar(cuerpo); } catch { throw fallo("respuesta_incompatible", estado, escritura); }
}

/** Puerto HTTP de las bandejas D6/D8; no acepta identidad ni unidad del navegador. */
export function crearClienteCircuitoDietasHTTP({ fetchImpl = globalThis.fetch } = {}) {
  if (typeof fetchImpl !== "function") throw new TypeError("cliente del circuito de Dietas no disponible");
  return Object.freeze({
    listar: async (consulta, opciones = {}) => {
      const valores = validarConsulta(consulta); const parametros = new URLSearchParams({ etapa: valores.etapa, limit: String(valores.limit) });
      for (const clave of ["fecha_desde", "fecha_hasta", "cursor"]) if (valores[clave]) parametros.set(clave, valores[clave]);
      return ejecutar(fetchImpl, `${RUTA_CIRCUITO}?${parametros}`, { method: "GET", headers: { Accept: "application/json" } }, [200], validarOpciones(opciones), false, validarPagina);
    },
    decidir: async (referenciaComision, entrada, opciones = {}) => {
      if (!referencia(referenciaComision)) throw new TypeError("referencia de comisión no válida");
      const decision = validarDecision(entrada); const cuerpo = JSON.stringify(decision);
      if (codificador.encode(cuerpo).byteLength > MAXIMO_CUERPO_BYTES) throw new TypeError("decisión del circuito demasiado grande");
      return ejecutar(fetchImpl, `${RUTA_CIRCUITO}/${encodeURIComponent(referenciaComision)}/decisiones`, { method: "POST", headers: { "Content-Type": "application/json; charset=utf-8", Accept: "application/json" }, body: cuerpo }, [200, 201], validarOpciones(opciones), true, (resultado) => validarResultado(resultado, referenciaComision, decision.version_esperada));
    },
  });
}
