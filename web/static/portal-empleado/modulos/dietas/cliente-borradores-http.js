import { sinBordes, textoRef } from "./texto-dietas.js?v=20260925-d5d6-v1";
const RUTA_COMISIONES = "/api/vec/dietas/comisiones";
const MAXIMO_CUERPO_SOLICITUD_BYTES = 16 * 1024;
const MAXIMO_RESPUESTA_BYTES = 128 * 1024;
const MAXIMO_FRAGMENTOS = 256;
const codificador = new TextEncoder();
const CODIGOS_POR_ESTADO = new Map([[400, new Set(["peticion_invalida"])], [401, new Set(["autenticacion_requerida"])], [403, new Set(["acceso_denegado"])], [404, new Set(["no_encontrada"])], [405, new Set(["metodo_no_permitido"])], [409, new Set(["relacion_ambigua", "conflicto_idempotencia", "conflicto_version", "estado_incompatible", "documento_no_disponible"])], [415, new Set(["tipo_no_admitido"])], [422, new Set(["relacion_no_valida", "documento_invalido"])], [503, new Set(["relacion_no_disponible", "resultado_incierto", "no_disponible"])]]);

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
const REFERENCIA_JUSTIFICANTE = /^[A-Za-z][A-Za-z0-9:_-]{2,127}$/u;
const HUELLA_JUSTIFICANTE = /^[a-f0-9]{64}$/u;
const CLAVES_OTRO_GASTO = ["tipo", "tipo_gasto", "catalogo_version", "fecha", "concepto", "importe_centimos", "justificante_ref", "justificante_sha256"];
// D5: al guardar, toda línea lleva tipo del catálogo, fecha y justificante
// por referencia y huella; el servidor coteja además catálogo y fechas.
function validarOtros(otros) {
  if (!Array.isArray(otros) || otros.length > 32) throw new TypeError("líneas de otros gastos no válidas");
  return Object.freeze(otros.map((linea) => {
    if (!registro(linea) || Object.keys(linea).length !== CLAVES_OTRO_GASTO.length || Object.keys(linea).some((clave) => !CLAVES_OTRO_GASTO.includes(clave)) ||
        !["otro_medio", "otro_gasto"].includes(linea.tipo) || typeof linea.tipo_gasto !== "string" || !/^[a-z][a-z_]{1,40}$/u.test(linea.tipo_gasto) ||
        typeof linea.catalogo_version !== "string" || !/^provisional:[a-z0-9:-]{8,120}$/u.test(linea.catalogo_version) || !fechaCivil(linea.fecha) ||
        !textoVisible(linea.concepto, 500) || linea.concepto.length < 3 || linea.concepto !== linea.concepto.trim() ||
        !Number.isSafeInteger(linea.importe_centimos) || linea.importe_centimos <= 0 || linea.importe_centimos > 100000000 ||
        !REFERENCIA_JUSTIFICANTE.test(linea.justificante_ref) || !HUELLA_JUSTIFICANTE.test(linea.justificante_sha256))
      throw new TypeError("línea de otros gastos no válida");
    return Object.freeze({ ...linea });
  }));
}
function validarRutas(rutas, vehiculoPropio) {
  if (!Array.isArray(rutas) || (vehiculoPropio ? rutas.length < 1 || rutas.length > 8 : rutas.length !== 0))
    throw new TypeError("rutas de vehículo propio no válidas");
  return Object.freeze(rutas.map((ruta) => {
    const codigos = ruta?.codigos_ruta;
    if (!registro(ruta) || Object.keys(ruta).some((clave) => !["codigos_ruta", "ajuste_kilometros", "motivo_ajuste"].includes(clave)) ||
        !Array.isArray(codigos) || codigos.length < 2 || codigos.length > 12 ||
        !codigos.every((codigo) => typeof codigo === "string" && /^[A-Za-z0-9:_-]{1,64}$/u.test(codigo)) ||
        new Set(codigos).size !== codigos.length ||
        !/^-?(?:0|[1-9]\d{0,3})\.\d{4}$/u.test(ruta.ajuste_kilometros) ||
        Math.abs(Number(ruta.ajuste_kilometros)) > 1000 || ruta.ajuste_kilometros === "-0.0000" ||
        (ruta.ajuste_kilometros === "0.0000"
          ? ruta.motivo_ajuste !== ""
          : !textoVisible(ruta.motivo_ajuste, 500) || ruta.motivo_ajuste.length < 3))
      throw new TypeError("ruta de vehículo propio no válida");
    return Object.freeze({ ...ruta, codigos_ruta: Object.freeze([...codigos]) });
  }));
}
function validarMutacion(entrada, completa = false) {
  const campos = completa
    ? ["clave_idempotencia", "version_esperada", "relacion_ref", "fecha_inicio", "fecha_fin", "hora_inicio", "hora_fin", "motivo", "codigos_ruta", "vehiculo_propio", "rutas", "otros", "tramos_aceptados", "version_tarifa_aceptada"]
    : ["clave_idempotencia", "version_esperada", "relacion_ref"];
  if (!registro(entrada) || Object.keys(entrada).some((clave) => !campos.includes(clave)) ||
      !/^[A-Za-z0-9_-]{16,128}$/u.test(entrada.clave_idempotencia || "") ||
      !Number.isSafeInteger(entrada.version_esperada) || entrada.version_esperada < 1 ||
      !referencia(entrada.relacion_ref, "rel_")) throw new TypeError("operación de comisión no válida");
  if (!completa) return Object.freeze({ ...entrada });
  const { clave_idempotencia, version_esperada, relacion_ref, vehiculo_propio, rutas, otros,
    tramos_aceptados, version_tarifa_aceptada, ...cabecera } = entrada;
  validarSolicitud({ clave_idempotencia, relacion_ref, ...cabecera });
  if (typeof vehiculo_propio !== "boolean") throw new TypeError("vehículo propio no confirmado");
  if (!Array.isArray(tramos_aceptados) || tramos_aceptados.length > 62 ||
      !tramos_aceptados.every((indice, posicion) => Number.isSafeInteger(indice) && indice >= 0 && indice < 62 &&
        (posicion === 0 || indice > tramos_aceptados[posicion - 1])) ||
      !/^provisional:[a-z0-9:-]{8,120}$/u.test(version_tarifa_aceptada))
    throw new TypeError("aceptación de tramos de Dietas no válida");
  return Object.freeze({ ...entrada, rutas: validarRutas(rutas, vehiculo_propio),
    tramos_aceptados: Object.freeze([...tramos_aceptados]),
    ...(otros === undefined ? {} : { otros: validarOtros(otros) }) });
}
function validarRecibo(recibo) { if (!registro(recibo) || Object.keys(recibo).length !== 4 || !referencia(recibo.referencia, "rcd_") || !Number.isSafeInteger(recibo.version) || recibo.version < 1 || typeof recibo.registrado_en !== "string" || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$/u.test(recibo.registrado_en) || !Number.isFinite(Date.parse(recibo.registrado_en)) || typeof recibo.repeticion !== "boolean") throw new TypeError("recibo de Dietas incompatible"); return Object.freeze({ ...recibo }); }
function validarCalculo(calculo,codigos,rutasDeclaradas,vehiculo,documento) {
  if (!registro(calculo) || !/^provisional:[a-z0-9:-]{8,120}$/u.test(calculo.version_tarifa) ||
      !/^\d{1,5}\.\d{4}$/u.test(calculo.kilometros) || !/^0\.\d{4}$/u.test(calculo.eur_por_km) ||
      !Number.isSafeInteger(calculo.importe_kilometraje_centimos) || calculo.importe_kilometraje_centimos < 0 ||
      !Array.isArray(calculo.tramos_ruta) || !Array.isArray(calculo.opciones_dieta) || calculo.opciones_dieta.length !== 3)
    throw new TypeError("cálculo de comisión incompatible");
  if (documento) {
    if ((calculo.vehiculo_propio === true) !== vehiculo || calculo.tramos_ruta.length !== 0 ||
        (calculo.rutas !== undefined && !Array.isArray(calculo.rutas)) || (calculo.rutas || []).length !== rutasDeclaradas.length ||
        (vehiculo ? calculo.procedencia !== "osrm_interno" || calculo.motor !== "OSRM" :
          calculo.procedencia !== "sin_vehiculo_propio" || calculo.motor !== "no_aplica" || calculo.kilometros !== "0.0000" || calculo.importe_kilometraje_centimos !== 0))
      throw new TypeError("cálculo de documento incompatible");
    (calculo.rutas || []).forEach((ruta, indice) => {
      const declarada = rutasDeclaradas[indice];
      if (!registro(ruta) || !Array.isArray(ruta.codigos_ruta) ||
          JSON.stringify(ruta.codigos_ruta) !== JSON.stringify(declarada.codigos_ruta) ||
          ruta.ajuste_kilometros !== declarada.ajuste_kilometros || ruta.motivo_ajuste !== declarada.motivo_ajuste ||
          !Array.isArray(ruta.tramos_ruta) || ruta.tramos_ruta.length !== declarada.codigos_ruta.length - 1 ||
          !/^\d{1,5}\.\d{4}$/u.test(ruta.kilometros_finales) || !Number.isSafeInteger(ruta.importe_centimos))
        throw new TypeError("ruta calculada incompatible");
    });
  } else {
    if (calculo.procedencia !== "osrm_interno" || calculo.motor !== "OSRM" || typeof calculo.version_grafo !== "string" ||
        calculo.tramos_ruta.length !== codigos.length - 1)
      throw new TypeError("cálculo de comisión incompatible");
    calculo.tramos_ruta.forEach((tramo,i)=>{ if (tramo.origen_codigo!==codigos[i] || tramo.destino_codigo!==codigos[i+1] || !/^\d{1,5}\.\d{4}$/u.test(tramo.kilometros)) throw new TypeError("tramo de comisión incompatible"); });
  }
  calculo.opciones_dieta.forEach((opcion,i)=>{ if (opcion.grupo!==i+1 || !registro(opcion.calculo) || !Array.isArray(opcion.calculo.tramos) || !Number.isSafeInteger(opcion.calculo.total_maximo_orientativo_centimos)) throw new TypeError("tramos de dieta incompatibles"); });
  return Object.freeze({ ...calculo, tramos_ruta:Object.freeze(calculo.tramos_ruta.map((tramo)=>Object.freeze({...tramo}))), opciones_dieta:Object.freeze(calculo.opciones_dieta.map((opcion)=>Object.freeze({...opcion,calculo:Object.freeze({...opcion.calculo,tramos:Object.freeze(opcion.calculo.tramos.map((tramo)=>Object.freeze({...tramo})))})}))) });
}
// Tras enviarla, la comisión recorre el circuito de revisión; la persona
// titular sigue viendo su documento en cualquiera de esos estados.
const ESTADOS_COMISION_PROPIA = Object.freeze(["borrador", "eliminado", "enviado_pendiente_revision", "pendiente_autorizacion", "pendiente_liquidacion", "pendiente_fiscalizacion", "fiscalizada", "devuelta"]);
const ETAPAS_DEVOLUCION = Object.freeze(["revision", "autorizacion", "liquidacion", "fiscalizacion"]);
// Devolución vigente: solo en un documento devuelto o en corrección, con la
// etapa que lo devolvió, su motivo, la versión devuelta y la fecha.
function validarDevolucion(devolucion, comision) {
  if (!registro(devolucion) || Object.keys(devolucion).length !== 4 || !ETAPAS_DEVOLUCION.includes(devolucion.etapa) ||
      !textoVisible(devolucion.motivo, 600) || devolucion.motivo.length < 3 || !sinBordes(devolucion.motivo) ||
      !Number.isSafeInteger(devolucion.version) || devolucion.version < 3 || !Number.isSafeInteger(comision.version) || devolucion.version > comision.version ||
      typeof devolucion.devuelta_en !== "string" || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$/u.test(devolucion.devuelta_en) ||
      !["devuelta", "borrador"].includes(comision.estado))
    throw new TypeError("devolución de Dietas incompatible");
  return Object.freeze({ ...devolucion });
}
function validarComision(comision) {
  const campos = ["referencia", "version", "numero_documento", "fecha_apertura", "estado", "fecha_inicio", "fecha_fin", "motivo", "codigos_ruta", "relacion_ref", "centro_ref", "unidad_ref", "calculo", "documento", "vehiculo_propio", "rutas", "devolucion"];
  if (!registro(comision) || Object.keys(comision).some((clave) => !campos.includes(clave)) || !referencia(comision.referencia, "dco_") || (comision.version !== undefined && (!Number.isSafeInteger(comision.version) || comision.version < 1)) || !ESTADOS_COMISION_PROPIA.includes(comision.estado) || !fechaCivil(comision.fecha_inicio) || !fechaCivil(comision.fecha_fin) || comision.fecha_fin < comision.fecha_inicio || !textoVisible(comision.motivo, 600) || !referencia(comision.relacion_ref, "rel_")) throw new TypeError("comisión de Dietas incompatible");
  if ((comision.numero_documento !== undefined && !/^VEC-D-\d{4}-\d{6}$/u.test(comision.numero_documento)) ||
      (comision.fecha_apertura !== undefined && !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$/u.test(comision.fecha_apertura)))
    throw new TypeError("identificador de comisión incompatible");
  const codigos = comision.codigos_ruta === undefined ? [] : comision.codigos_ruta;
  if (!Array.isArray(codigos) || codigos.length > 16 || !codigos.every((codigo) => typeof codigo === "string" && /^[A-Za-z0-9:_-]{1,64}$/u.test(codigo)) || new Set(codigos).size !== codigos.length) throw new TypeError("comisión de Dietas incompatible");
  if (comision.vehiculo_propio !== undefined && typeof comision.vehiculo_propio !== "boolean") throw new TypeError("comisión de Dietas incompatible");
  // Centro y unidad llegan con el documento v2 como referencias opacas de
  // Personal: mismo validador y límites que la asignación (160 y 256).
  if ((comision.centro_ref !== undefined && !textoRef(comision.centro_ref, 160)) ||
      (comision.unidad_ref !== undefined && !textoRef(comision.unidad_ref, 256))) throw new TypeError("comisión de Dietas incompatible");
  const devolucion = comision.devolucion === undefined ? undefined : validarDevolucion(comision.devolucion, comision);
  if (comision.rutas !== undefined && comision.rutas !== null) validarRutas(comision.rutas, comision.vehiculo_propio === true);
  if (comision.documento !== undefined && comision.documento !== null) {
    const documento = comision.documento;
    if (!registro(documento) || !Array.isArray(documento.lineas) || documento.lineas.length > 256 ||
        typeof documento.vehiculo_propio !== "boolean" || documento.vehiculo_propio !== comision.vehiculo_propio ||
        ![1, 2, 3].includes(documento.grupo_dieta) ||
        !/^provisional:[a-z0-9:-]{8,120}$/u.test(documento.version_tarifa_aceptada) ||
        !Array.isArray(documento.tramos_aceptados) || documento.tramos_aceptados.length > 62 ||
        !documento.tramos_aceptados.every((indice, posicion) => Number.isSafeInteger(indice) && indice >= 0 && indice < 62 &&
          (posicion === 0 || indice > documento.tramos_aceptados[posicion - 1])) ||
        ["manutencion_centimos", "alojamiento_tope_centimos", "kilometraje_centimos", "otros_centimos", "total_orientativo_centimos"]
          .some((clave) => !Number.isSafeInteger(documento[clave]) || documento[clave] < 0) ||
        documento.total_orientativo_centimos !== documento.manutencion_centimos + documento.alojamiento_tope_centimos +
          documento.kilometraje_centimos + documento.otros_centimos)
      throw new TypeError("documento de comisión incompatible");
    const tramosGrupo = comision.calculo?.opciones_dieta?.[documento.grupo_dieta - 1]?.calculo?.tramos;
    if (!Array.isArray(tramosGrupo) ||
        (tramosGrupo.length === 0 ? documento.tramos_aceptados.length !== 0 :
          documento.tramos_aceptados.length !== 1 && documento.tramos_aceptados.length !== tramosGrupo.length) ||
        documento.tramos_aceptados.some((indice, posicion) => indice >= tramosGrupo.length ||
          (documento.tramos_aceptados.length === tramosGrupo.length && indice !== posicion)))
      throw new TypeError("aceptación de comisión incompatible");
    let totalDietas = 0;
    let totalKM = 0;
    let totalOtros = 0;
    let dietas = 0;
    let kilometrajes = 0;
    for (const linea of documento.lineas) {
      if (!registro(linea) || !["dieta", "kilometraje", "otro_medio", "otro_gasto"].includes(linea.tipo) ||
          !Number.isSafeInteger(linea.importe_centimos) || linea.importe_centimos < 0)
        throw new TypeError("línea de comisión incompatible");
      if (linea.tipo === "kilometraje") {
        kilometrajes += 1;
        if (!Number.isSafeInteger(linea.ruta_indice) || linea.ruta_indice !== kilometrajes ||
            !/^\d{1,5}\.\d{4}$/u.test(linea.kilometros)) throw new TypeError("kilometraje incompatible");
        totalKM += linea.importe_centimos;
      } else if (linea.tipo === "otro_medio" || linea.tipo === "otro_gasto") {
        // Las líneas anteriores a D5 no tienen tipo ni fecha y su justificante
        // es opcional; las D5 lo llevan siempre.
        const catalogada = linea.tipo_gasto !== undefined || linea.catalogo_version !== undefined;
        if (!textoVisible(linea.concepto, 500) ||
            (catalogada && (typeof linea.tipo_gasto !== "string" || !/^[a-z][a-z_]{1,40}$/u.test(linea.tipo_gasto) ||
              typeof linea.catalogo_version !== "string" || !fechaCivil(linea.fecha) ||
              !REFERENCIA_JUSTIFICANTE.test(linea.justificante_ref || "") || !HUELLA_JUSTIFICANTE.test(linea.justificante_sha256 || ""))) ||
            !((linea.justificante_ref === "" && linea.justificante_sha256 === "") ||
              (REFERENCIA_JUSTIFICANTE.test(linea.justificante_ref || "") &&
                HUELLA_JUSTIFICANTE.test(linea.justificante_sha256 || ""))))
          throw new TypeError("otro gasto incompatible");
        totalOtros += linea.importe_centimos;
      } else {
        const indice = documento.tramos_aceptados[dietas];
        const tramo = tramosGrupo[indice];
        if (linea.grupo !== documento.grupo_dieta || !fechaCivil(linea.fecha) ||
            linea.indice_tramo !== indice || tramo?.fecha !== linea.fecha ||
            tramo?.importe_centimos !== linea.importe_centimos || tramo?.tipo !== linea.concepto)
          throw new TypeError("dieta incompatible");
        totalDietas += linea.importe_centimos; dietas += 1;
      }
    }
    if (totalDietas !== documento.manutencion_centimos + documento.alojamiento_tope_centimos ||
        totalKM !== documento.kilometraje_centimos || totalOtros !== documento.otros_centimos ||
        dietas !== documento.tramos_aceptados.length || kilometrajes !== (comision.rutas?.length || 0) ||
        documento.version_tarifa_aceptada !== comision.calculo?.version_tarifa)
      throw new TypeError("total de comisión incompatible");
  }
  return Object.freeze({ ...comision, codigos_ruta: Object.freeze([...codigos]), ...(devolucion ? { devolucion } : {}), ...(comision.calculo ? {calculo:validarCalculo(comision.calculo,codigos,comision.rutas || [],comision.vehiculo_propio === true,Boolean(comision.documento))} : {}) });
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
async function ejecutar(fetchImpl, ruta, opciones, estadosCorrectos, signal, escritura = false, validarResultado) {
  let respuesta; try { respuesta = await fetchImpl(ruta, { ...opciones, credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal }); } catch { throw fallo(signal?.aborted ? "operacion_abortada" : "red_no_disponible", 0, escritura && !signal?.aborted); }
  if (!respuesta || respuesta.redirected === true) { await cancelarRespuesta(respuesta); throw fallo("respuesta_rechazada", respuesta?.status || 0, escritura); }
  const estado = respuesta.status || 0; let cuerpo;
  try { cuerpo = await leerJSONAcotado(respuesta, signal); } catch (causa) {
    if (escritura && (estado === 0 || estado >= 500 || (estado >= 200 && estado < 300)) &&
        !(causa instanceof ErrorClienteBorradoresDietas && causa.codigo === "operacion_abortada"))
      throw fallo(causa instanceof ErrorClienteBorradoresDietas ? causa.codigo : "respuesta_incompatible", estado, true);
    throw causa;
  }
  if (!estadosCorrectos.includes(estado) || respuesta.ok !== true) {
    const codigo = codigoError(cuerpo, estado);
    throw fallo(codigo, estado, escritura && (estado === 0 || estado >= 500 || (estado >= 200 && estado < 300)));
  }
  if (validarResultado) {
    try { return validarResultado(cuerpo); }
    catch { throw fallo("respuesta_incompatible", estado, escritura); }
  }
  return cuerpo;
}
export function crearClienteBorradoresDietasHTTP({ fetchImpl = globalThis.fetch } = {}) {
  if (typeof fetchImpl !== "function") throw new TypeError("cliente de borradores de Dietas no disponible");
  async function mutar(metodo, referenciaComision, entrada, opciones, completa = false, sufijo = "") {
    if (!referencia(referenciaComision, "dco_")) throw new TypeError("referencia de Dietas no válida");
    const solicitud = validarMutacion(entrada, completa);
    const signal = validarOpciones(opciones);
    const cuerpo = JSON.stringify(solicitud);
    if (codificador.encode(cuerpo).byteLength > MAXIMO_CUERPO_SOLICITUD_BYTES)
      throw new TypeError("operación de Dietas demasiado grande");
    const resultado = await ejecutar(fetchImpl, `${RUTA_COMISIONES}/${encodeURIComponent(referenciaComision)}${sufijo}`,
      { method: metodo, headers: { "Content-Type": "application/json; charset=utf-8", Accept: "application/json" }, body: cuerpo },
      [200, 201], signal, true, validarItem);
    if (resultado.comision.referencia !== referenciaComision || resultado.recibo.version < solicitud.version_esperada + 1 ||
        (metodo === "PUT" && resultado.comision.estado !== "borrador") ||
        (metodo === "DELETE" && resultado.comision.estado !== "eliminado") ||
        (sufijo === "/enviar" && resultado.comision.estado !== "enviado_pendiente_revision"))
      throw fallo("respuesta_incompatible", 200, true);
    return resultado;
  }
  return Object.freeze({
    async crear(entrada, opciones = {}) { const solicitud = validarSolicitud(entrada); const signal = validarOpciones(opciones); const cuerpo = JSON.stringify(solicitud); if (codificador.encode(cuerpo).byteLength > MAXIMO_CUERPO_SOLICITUD_BYTES) throw new TypeError("solicitud de borrador de Dietas demasiado grande"); return ejecutar(fetchImpl, RUTA_COMISIONES, { method: "POST", headers: { "Content-Type": "application/json; charset=utf-8", Accept: "application/json" }, body: cuerpo }, [200, 201], signal, true, validarItem); },
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
    editar(referenciaComision, entrada, opciones = {}) { return mutar("PUT", referenciaComision, entrada, opciones, true); },
    eliminar(referenciaComision, entrada, opciones = {}) { return mutar("DELETE", referenciaComision, entrada, opciones); },
    enviar(referenciaComision, entrada, opciones = {}) { return mutar("POST", referenciaComision, entrada, opciones, false, "/enviar"); },
  });
}
