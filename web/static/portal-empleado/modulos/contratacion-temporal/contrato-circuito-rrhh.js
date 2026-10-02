/** Consulta minimizada del circuito; no concede permisos ni acredita firmas. */
const REFERENCIA = /^[a-zA-Z0-9][a-zA-Z0-9._:/#-]{2,159}$/u;
const CLAVE = /^[a-z][a-z0-9._-]{1,127}$/u;
const HUELLA = /^[a-f0-9]{64}$/u;
const TIPOS = new Set([
  "peticion_firmada", "autorizacion_rrhh", "credito_comprobado", "oferta_emitida",
  "adjudicacion", "informe_jefatura", "intervencion_favorable", "intervencion_reparo",
  "resolucion_firmada", "ginpix_registrado", "firma_interesado", "devolucion", "reinicio", "incorporacion",
]);
function exacto(valor, campos) {
  return valor !== null && typeof valor === "object" && !Array.isArray(valor)
    && Object.getPrototypeOf(valor) === Object.prototype
    && Object.keys(valor).length === campos.length && campos.every((campo) => Object.hasOwn(valor, campo));
}
function referencia(valor) { return typeof valor === "string" && REFERENCIA.test(valor); }
function clave(valor) { return typeof valor === "string" && CLAVE.test(valor); }
function version(valor) { return Number.isSafeInteger(valor) && valor > 0; }
function flujo(valor) {
  return exacto(valor, ["definicion_ref", "version", "huella_sha256"])
    && referencia(valor.definicion_ref) && version(valor.version)
    && typeof valor.huella_sha256 === "string" && HUELLA.test(valor.huella_sha256) && !/^0+$/u.test(valor.huella_sha256);
}
function fallo() { throw new TypeError("circuito_rrhh.contrato_no_valido"); }
export function validarConsultaCircuitoRRHH(valor) {
  if (!exacto(valor, ["expediente_ref", "version_observada"])
    || !referencia(valor.expediente_ref) || !version(valor.version_observada)) fallo();
  return Object.freeze({ ...valor });
}
export function validarCircuitoRRHH(datos, consulta) {
  validarConsultaCircuitoRRHH(consulta);
  if (!exacto(datos, ["flujo", "circuito", "version_expediente", "transiciones_permitidas"])
    || !flujo(datos.flujo) || !version(datos.version_expediente)
    || datos.version_expediente !== consulta.version_observada
    || !exacto(datos.circuito, ["definicion", "estado_actual", "hitos"])
    || !flujo(datos.circuito.definicion)
    || Object.keys(datos.flujo).some((clave) => datos.flujo[clave] !== datos.circuito.definicion[clave])
    || !clave(datos.circuito.estado_actual)
    || !Array.isArray(datos.circuito.hitos) || datos.circuito.hitos.length > 2000
    || !Array.isArray(datos.transiciones_permitidas) || datos.transiciones_permitidas.length > 128) fallo();
  const hitos = datos.circuito.hitos.map((hito, indice) => {
    if (!exacto(hito, ["secuencia", "clave", "tipo", "origen", "destino", "registrado_en", "recibo_ref"])
      || hito.secuencia !== indice + 1 || !clave(hito.clave) || !TIPOS.has(hito.tipo)
      || !clave(hito.origen) || !clave(hito.destino) || !referencia(hito.recibo_ref)
      || typeof hito.registrado_en !== "string"
      || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,6})?Z$/u.test(hito.registrado_en)
      || !Number.isFinite(Date.parse(hito.registrado_en))) fallo();
    return Object.freeze({ ...hito });
  });
  const transiciones = datos.transiciones_permitidas.map((paso) => {
    if (!exacto(paso, ["clave", "tipo", "perfil_clave", "requiere_documento", "requiere_firma"])
      || !clave(paso.clave) || !TIPOS.has(paso.tipo) || !clave(paso.perfil_clave)
      || typeof paso.requiere_documento !== "boolean" || typeof paso.requiere_firma !== "boolean") fallo();
    return Object.freeze({ ...paso });
  });
  if (new Set(transiciones.map(({ clave }) => clave)).size !== transiciones.length) fallo();
  return Object.freeze({
    flujo: Object.freeze({ ...datos.flujo }), version_expediente: datos.version_expediente,
    circuito: Object.freeze({ definicion: Object.freeze({ ...datos.circuito.definicion }),
      estado_actual: datos.circuito.estado_actual, hitos: Object.freeze(hitos) }),
    transiciones_permitidas: Object.freeze(transiciones),
  });
}
