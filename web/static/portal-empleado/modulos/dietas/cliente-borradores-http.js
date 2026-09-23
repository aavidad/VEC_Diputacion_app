const RUTA_COMISIONES = "/api/vec/dietas/comisiones";
const MAXIMO_CUERPO_SOLICITUD_BYTES = 16 * 1024;
const MAXIMO_RESPUESTA_BYTES = 128 * 1024;
const MAXIMO_FRAGMENTOS = 256;
const codificador = new TextEncoder();
const CODIGOS_POR_ESTADO = new Map([[400, new Set(["peticion_invalida"])], [401, new Set(["autenticacion_requerida"])], [403, new Set(["acceso_denegado"])], [404, new Set(["no_encontrada"])], [405, new Set(["metodo_no_permitido"])], [409, new Set(["relacion_ambigua", "conflicto_idempotencia"])], [415, new Set(["tipo_no_admitido"])], [422, new Set(["relacion_no_valida"])], [503, new Set(["relacion_no_disponible", "resultado_incierto", "no_disponible"])]]);

export class ErrorClienteBorradoresDietas extends Error {
  constructor(codigo, estado = 0, resultadoIndeterminado = false) { super(`cliente de borradores de Dietas: ${codigo}`); this.name = "ErrorClienteBorradoresDietas"; this.codigo = codigo; this.estado = estado; this.resultadoIndeterminado = resultadoIndeterminado; Object.freeze(this); }
}
function fallo(codigo, estado = 0, resultadoIndeterminado = false) { return new ErrorClienteBorradoresDietas(codigo, estado, resultadoIndeterminado); }
function registro(valor) { return valor !== null && typeof valor === "object" && !Array.isArray(valor) && (Object.getPrototypeOf(valor) === Object.prototype || Object.getPrototypeOf(valor) === null); }
function textoVisible(valor, maximoBytes) { return typeof valor === "string" && valor.length > 0 && codificador.encode(valor).byteLength <= maximoBytes && !/[\x00-\x1F\x7F]/u.test(valor); }
function fechaCivil(valor) { if (typeof valor !== "string" || !/^\d{4}-\d{2}-\d{2}$/u.test(valor)) return false; const [ano, mes, dia] = valor.split("-").map(Number); const fecha = new Date(Date.UTC(ano, mes - 1, dia)); return fecha.getUTCFullYear() === ano && fecha.getUTCMonth() === mes - 1 && fecha.getUTCDate() === dia; }
function referencia(valor, prefijo) { return typeof valor === "string" && new RegExp(`^${prefijo}[A-Za-z0-9_-]{22,128}$`, "u").test(valor); }
function validarSignal(signal) { if (signal === undefined) return undefined; if (!signal || typeof signal !== "object" || typeof signal.aborted !== "boolean" || typeof signal.addEventListener !== "function" || typeof signal.removeEventListener !== "function") throw fallo("signal_no_valida"); if (signal.aborted) throw fallo("operacion_abortada"); return signal; }
function validarOpciones(opciones = {}) { if (!registro(opciones) || Object.keys(opciones).some((clave) => clave !== "signal")) throw fallo("opciones_no_validas"); return validarSignal(opciones.signal); }
function validarOpcionesConsulta(opciones = {}) {
  if (!registro(opciones) || Object.keys(opciones).some((clave) => clave !== "signal" && clave !== "relacion_ref")) throw fallo("opciones_no_validas");
  if (opciones.relacion_ref !== undefined && !referencia(opciones.relacion_ref, "rel_")) throw new TypeError("relación no válida");
  return Object.freeze({ signal: validarSignal(opciones.signal), relacion_ref: opciones.relacion_ref });
}
function validarSolicitud(entrada) {
  const campos = ["clave_idempotencia", "fecha_inicio", "fecha_fin", "hora_inicio", "hora_fin", "motivo", "codigos_ruta", "relacion_ref"];
  if (!registro(entrada) || Object.keys(entrada).some((clave) => !campos.includes(clave)) || typeof entrada.clave_idempotencia !== "string" || !/^[A-Za-z0-9_-]{16,128}$/u.test(entrada.clave_idempotencia) || !fechaCivil(entrada.fecha_inicio) || !fechaCivil(entrada.fecha_fin) || entrada.fecha_fin < entrada.fecha_inicio || !textoVisible(entrada.motivo, 600)) throw new TypeError("solicitud de borrador de Dietas no válida");
  if (entrada.codigos_ruta !== undefined && (!Array.isArray(entrada.codigos_ruta) || entrada.codigos_ruta.length > 16 || !entrada.codigos_ruta.every((codigo) => typeof codigo === "string" && /^[A-Za-z0-9:_-]{1,64}$/u.test(codigo)) || new Set(entrada.codigos_ruta).size !== entrada.codigos_ruta.length)) throw new TypeError("códigos de ruta no válidos");
  for (const hora of [entrada.hora_inicio,entrada.hora_fin]) if (hora !== undefined && (typeof hora !== "string" || !/^([01]\d|2[0-3]):[0-5]\d$/u.test(hora))) throw new TypeError("hora de comisión no válida");
  if (entrada.relacion_ref !== undefined && !referencia(entrada.relacion_ref, "rel_")) throw new TypeError("relación no válida");
  return Object.freeze({ ...entrada, ...(entrada.codigos_ruta ? { codigos_ruta: Object.freeze([...entrada.codigos_ruta]) } : {}) });
}
function validarRecibo(recibo) { if (!registro(recibo) || Object.keys(recibo).length !== 4 || !referencia(recibo.referencia, "rcd_") || !Number.isSafeInteger(recibo.version) || recibo.version < 1 || typeof recibo.registrado_en !== "string" || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$/u.test(recibo.registrado_en) || !Number.isFinite(Date.parse(recibo.registrado_en)) || typeof recibo.repeticion !== "boolean") throw new TypeError("recibo de Dietas incompatible"); return Object.freeze({ ...recibo }); }
function validarCalculo(calculo,codigos) {
  if (!registro(calculo) || calculo.procedencia !== "osrm_interno" || calculo.motor !== "OSRM" || typeof calculo.version_grafo !== "string" || !/^provisional:[a-z0-9:-]{8,120}$/u.test(calculo.version_tarifa) || !/^\d{1,5}\.\d{4}$/u.test(calculo.kilometros) || !/^0\.\d{4}$/u.test(calculo.eur_por_km) || !Number.isSafeInteger(calculo.importe_kilometraje_centimos) || calculo.importe_kilometraje_centimos<0 || !Array.isArray(calculo.tramos_ruta) || calculo.tramos_ruta.length!==codigos.length-1 || !Array.isArray(calculo.opciones_dieta) || calculo.opciones_dieta.length!==3) throw new TypeError("cálculo de comisión incompatible");
  calculo.tramos_ruta.forEach((tramo,i)=>{ if (tramo.origen_codigo!==codigos[i] || tramo.destino_codigo!==codigos[i+1] || !/^\d{1,5}\.\d{4}$/u.test(tramo.kilometros)) throw new TypeError("tramo de comisión incompatible"); });
  calculo.opciones_dieta.forEach((opcion,i)=>{ if (opcion.grupo!==i+1 || !registro(opcion.calculo) || !Array.isArray(opcion.calculo.tramos) || !Number.isSafeInteger(opcion.calculo.total_maximo_orientativo_centimos)) throw new TypeError("tramos de dieta incompatibles"); });
  return Object.freeze({ ...calculo, tramos_ruta:Object.freeze(calculo.tramos_ruta.map((tramo)=>Object.freeze({...tramo}))), opciones_dieta:Object.freeze(calculo.opciones_dieta.map((opcion)=>Object.freeze({...opcion,calculo:Object.freeze({...opcion.calculo,tramos:Object.freeze(opcion.calculo.tramos.map((tramo)=>Object.freeze({...tramo})))})}))) });
}
function validarComision(comision) {
  const campos = ["referencia", "estado", "fecha_inicio", "fecha_fin", "motivo", "codigos_ruta", "relacion_ref", "calculo"];
  if (!registro(comision) || Object.keys(comision).some((clave) => !campos.includes(clave)) || !referencia(comision.referencia, "dco_") || comision.estado !== "borrador" || !fechaCivil(comision.fecha_inicio) || !fechaCivil(comision.fecha_fin) || comision.fecha_fin < comision.fecha_inicio || !textoVisible(comision.motivo, 600) || !referencia(comision.relacion_ref, "rel_")) throw new TypeError("comisión de Dietas incompatible");
  const codigos = comision.codigos_ruta === undefined ? [] : comision.codigos_ruta;
  if (!Array.isArray(codigos) || codigos.length > 16 || !codigos.every((codigo) => typeof codigo === "string" && /^[A-Za-z0-9:_-]{1,64}$/u.test(codigo)) || new Set(codigos).size !== codigos.length) throw new TypeError("comisión de Dietas incompatible");
  return Object.freeze({ ...comision, codigos_ruta: Object.freeze([...codigos]), ...(comision.calculo ? {calculo:validarCalculo(comision.calculo,codigos)} : {}) });
}
function validarItem(valor) { if (!registro(valor) || Object.keys(valor).length !== 2 || !Object.hasOwn(valor, "comision") || !Object.hasOwn(valor, "recibo")) throw new TypeError("resultado de Dietas incompatible"); return Object.freeze({ comision: validarComision(valor.comision), recibo: validarRecibo(valor.recibo) }); }
function validarPagina(valor) { if (!registro(valor) || Object.keys(valor).some((clave) => clave !== "items" && clave !== "siguiente_cursor") || !Array.isArray(valor.items) || valor.items.length > 50 || (valor.siguiente_cursor !== undefined && !textoVisible(valor.siguiente_cursor, 400))) throw new TypeError("página de Dietas incompatible"); return Object.freeze({ items: Object.freeze(valor.items.map(validarItem)), ...(valor.siguiente_cursor ? { siguiente_cursor: valor.siguiente_cursor } : {}) }); }
async function cancelarRespuesta(respuesta, lector) { try { await (lector?.cancel?.("respuesta descartada") ?? respuesta?.body?.cancel?.("respuesta descartada")); } catch {} }
async function leerJSONAcotado(respuesta, signal) {
  const estado = respuesta?.status || 0; const longitud = respuesta?.headers?.get?.("content-length");
  if (longitud !== null && longitud !== undefined && (!/^(?:0|[1-9][0-9]*)$/u.test(longitud) || Number(longitud) > MAXIMO_RESPUESTA_BYTES)) { await cancelarRespuesta(respuesta); throw fallo("respuesta_excesiva", estado); }
  const tipo = respuesta?.headers?.get?.("content-type");
  if (typeof tipo !== "string" || !/^application\/json;\s*charset=utf-8$/iu.test(tipo) || respuesta?.headers?.get?.("content-encoding")) { await cancelarRespuesta(respuesta); throw fallo("tipo_respuesta_no_valido", estado); }
  if (!respuesta?.body || typeof respuesta.body.getReader !== "function") { await cancelarRespuesta(respuesta); throw fallo("respuesta_no_incremental", estado); }
  const lector = respuesta.body.getReader(); if (!lector || typeof lector.read !== "function" || typeof lector.cancel !== "function") { await cancelarRespuesta(respuesta); throw fallo("respuesta_no_incremental", estado); }
  let cancelada = false; const abortar = () => { cancelada = true; Promise.resolve(lector.cancel("operación cancelada")).catch(() => {}); }; signal?.addEventListener("abort", abortar, { once: true });
  const fragmentos = []; let total = 0;
  try { while (true) { if (signal?.aborted || cancelada) throw fallo("operacion_abortada", estado); const fragmento = await lector.read(); if (!fragmento || typeof fragmento.done !== "boolean" || (!fragmento.done && (!(fragmento.value instanceof Uint8Array) || fragmento.value.byteLength === 0))) throw fallo("respuesta_incompatible", estado); if (fragmento.done) break; total += fragmento.value.byteLength; if (total > MAXIMO_RESPUESTA_BYTES || fragmentos.length >= MAXIMO_FRAGMENTOS) throw fallo("respuesta_excesiva", estado); fragmentos.push(fragmento.value); }
    if (longitud !== null && longitud !== undefined && total !== Number(longitud)) throw fallo("respuesta_incompatible", estado); const bytes = new Uint8Array(total); let posicion = 0; for (const fragmento of fragmentos) { bytes.set(fragmento, posicion); posicion += fragmento.byteLength; } try { return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes)); } catch { throw fallo("json_no_valido", estado); }
  } catch (causa) { await cancelarRespuesta(respuesta, lector); throw causa; } finally { signal?.removeEventListener("abort", abortar); try { lector.releaseLock?.(); } catch {} }
}
function codigoError(cuerpo, estado) { const codigo = typeof cuerpo?.error === "string" && cuerpo.error.startsWith("dietas.error.") ? cuerpo.error.slice("dietas.error.".length) : null; return CODIGOS_POR_ESTADO.get(estado)?.has(codigo) ? codigo : "respuesta_rechazada"; }
async function ejecutar(fetchImpl, ruta, opciones, estadosCorrectos, signal, escritura = false) {
  let respuesta; try { respuesta = await fetchImpl(ruta, { ...opciones, credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal }); } catch { throw fallo(signal?.aborted ? "operacion_abortada" : "red_no_disponible", 0, escritura && !signal?.aborted); }
  if (!respuesta || respuesta.redirected === true) { await cancelarRespuesta(respuesta); throw fallo("respuesta_rechazada", respuesta?.status || 0, escritura); }
  const estado = respuesta.status || 0; let cuerpo;
  try { cuerpo = await leerJSONAcotado(respuesta, signal); } catch (causa) {
    if (causa instanceof ErrorClienteBorradoresDietas && escritura && estado === 503) throw fallo(causa.codigo, estado, true);
    throw causa;
  }
  if (!estadosCorrectos.includes(estado) || respuesta.ok !== true) { const codigo = codigoError(cuerpo, estado); throw fallo(codigo, estado, escritura && (estado === 0 || estado === 503)); }
  return cuerpo;
}
export function crearClienteBorradoresDietasHTTP({ fetchImpl = globalThis.fetch } = {}) {
  if (typeof fetchImpl !== "function") throw new TypeError("cliente de borradores de Dietas no disponible");
  return Object.freeze({
    async crear(entrada, opciones = {}) { const solicitud = validarSolicitud(entrada); const signal = validarOpciones(opciones); const cuerpo = JSON.stringify(solicitud); if (codificador.encode(cuerpo).byteLength > MAXIMO_CUERPO_SOLICITUD_BYTES) throw new TypeError("solicitud de borrador de Dietas demasiado grande"); return validarItem(await ejecutar(fetchImpl, RUTA_COMISIONES, { method: "POST", headers: { "Content-Type": "application/json; charset=utf-8", Accept: "application/json" }, body: cuerpo }, [200, 201], signal, true)); },
    async listar(consulta = {}, opciones = {}) {
      if (!registro(consulta) || Object.keys(consulta).some((clave) => !["limit", "cursor", "relacion_ref"].includes(clave))) throw new TypeError("consulta de borradores de Dietas no válida");
      const { limit = 20, cursor, relacion_ref } = consulta;
      if (!Number.isSafeInteger(limit) || limit < 1 || limit > 50 || (cursor !== undefined && !textoVisible(cursor, 400)) || (relacion_ref !== undefined && !referencia(relacion_ref, "rel_"))) throw new TypeError("consulta de borradores de Dietas no válida");
      const signal = validarOpciones(opciones);
      const parametros = new URLSearchParams({ limit: String(limit) });
      if (cursor) parametros.set("cursor", cursor);
      if (relacion_ref) parametros.set("relacion_ref", relacion_ref);
      return validarPagina(await ejecutar(fetchImpl, `${RUTA_COMISIONES}?${parametros}`, { method: "GET", headers: { Accept: "application/json" } }, [200], signal));
    },
    async obtener(referenciaComision, opciones = {}) {
      if (!referencia(referenciaComision, "dco_")) throw new TypeError("referencia de Dietas no válida");
      const { signal, relacion_ref } = validarOpcionesConsulta(opciones);
      const parametros = new URLSearchParams();
      if (relacion_ref) parametros.set("relacion_ref", relacion_ref);
      const consulta = parametros.size ? `?${parametros}` : "";
      return validarItem(await ejecutar(fetchImpl, `${RUTA_COMISIONES}/${encodeURIComponent(referenciaComision)}${consulta}`, { method: "GET", headers: { Accept: "application/json" } }, [200], signal));
    },
  });
}
