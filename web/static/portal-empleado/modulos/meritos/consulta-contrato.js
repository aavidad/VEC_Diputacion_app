/** Contrato de lectura minimizada. No admite el Hecho completo del dominio. */
const ESTADOS = new Set(["declarado", "pendiente", "rechazado", "acreditado"]);
const TIPOS = new Set(["titulacion", "curso_asistencia", "curso_superacion", "experiencia", "idioma", "otro"]);
const HASH = /^[a-f0-9]{64}$/u;
const encoder = new TextEncoder();

export function referenciaConsultaValida(valor) {
  return typeof valor === "string" && /^[A-Za-z0-9:._-]{1,256}$/u.test(valor);
}

function cerrado(valor, necesarias, opcionales = []) {
  return valor !== null && typeof valor === "object" && !Array.isArray(valor)
    && necesarias.every((clave) => Object.hasOwn(valor, clave))
    && Object.keys(valor).every((clave) => necesarias.includes(clave) || opcionales.includes(clave));
}
function positivo(valor) { return Number.isSafeInteger(valor) && valor > 0 && valor <= 2147483647; }
function fecha(valor) {
  if (typeof valor !== "string" || !/^\d{4}-\d{2}-\d{2}$/u.test(valor)) return false;
  const date = new Date(`${valor}T12:00:00Z`);
  return Number.isFinite(date.getTime()) && date.toISOString().slice(0, 10) === valor;
}
function instante(valor) {
  if (typeof valor !== "string" || valor.length > 40) return false;
  const partes = /^(\d{4}-\d{2}-\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.\d{1,9})?(Z|[+-]\d{2}:\d{2})$/u.exec(valor);
  return partes !== null && fecha(partes[1]) && Number(partes[2]) < 24
    && Number(partes[3]) < 60 && Number(partes[4]) < 60 && Number.isFinite(Date.parse(valor));
}
function texto(valor) {
  return typeof valor === "string" && valor.trim() === valor && valor.length > 0
    && encoder.encode(valor).byteLength <= 512 && !/[\p{Cc}]/u.test(valor);
}
function validarHecho(h) {
  if (!cerrado(h, ["referencia", "version", "tipo", "concepto_ref", "denominacion", "procedencia", "vigencia", "estado", "evidencias"], ["horas", "revision"])
    || !referenciaConsultaValida(h.referencia) || !positivo(h.version) || !TIPOS.has(h.tipo)
    || !referenciaConsultaValida(h.concepto_ref) || !texto(h.denominacion) || !ESTADOS.has(h.estado)) return false;
  if (Object.hasOwn(h, "horas") && (!Number.isSafeInteger(h.horas) || h.horas < 0 || h.horas > 2147483647 || !["curso_asistencia", "curso_superacion"].includes(h.tipo))) return false;
  const p = h.procedencia;
  if (!cerrado(p, ["fuente_ref", "version", "hecho_origen_ref", "capturada_en"])
    || ![p.fuente_ref, p.version, p.hecho_origen_ref].every(referenciaConsultaValida) || !instante(p.capturada_en)) return false;
  const v = h.vigencia;
  if (!cerrado(v, ["desde"], ["hasta"]) || !fecha(v.desde)
    || (Object.hasOwn(v, "hasta") && (!fecha(v.hasta) || v.hasta < v.desde))) return false;
  if (!Array.isArray(h.evidencias) || h.evidencias.length > 32) return false;
  const vistos = new Set();
  for (const e of h.evidencias) {
    if (!cerrado(e, ["id", "version"]) || !referenciaConsultaValida(e.id) || !positivo(e.version)) return false;
    const clave = `${e.id}/${e.version}`;
    if (vistos.has(clave)) return false;
    vistos.add(clave);
  }
  if (["acreditado", "rechazado"].includes(h.estado) && !h.revision) return false;
  if (h.estado === "acreditado" && h.evidencias.length === 0) return false;
  if (Object.hasOwn(h, "revision")) {
    const r = h.revision;
    if (!cerrado(r, ["referencia", "motivo_ref", "fecha"]) || ![r.referencia, r.motivo_ref].every(referenciaConsultaValida) || !instante(r.fecha)) return false;
  }
  return true;
}

function congelar(valor) {
  if (Array.isArray(valor)) return Object.freeze(valor.map(congelar));
  if (valor !== null && typeof valor === "object") return Object.freeze(Object.fromEntries(Object.entries(valor).map(([clave, dato]) => [clave, congelar(dato)])));
  return valor;
}

/** Verifica correspondencia con la selección y copia antes de congelar. */
export function validarResultadoConsulta(datos, hechoRef) {
  if (!referenciaConsultaValida(hechoRef) || !cerrado(datos, ["codigo", "hecho_actual", "recibo_consulta"])
    || !["obtenida", "no_encontrada"].includes(datos.codigo)) throw new TypeError("meritos.consulta.respuesta_invalida");
  const encontrada = datos.codigo === "obtenida";
  if (encontrada ? !validarHecho(datos.hecho_actual) || datos.hecho_actual.referencia !== hechoRef : datos.hecho_actual !== null) throw new TypeError("meritos.consulta.respuesta_invalida");
  const r = datos.recibo_consulta;
  if (!cerrado(r, ["referencia", "hecho_ref", "version_consultada", "decision_ref", "consumo_huella_sha256", "auditoria_ref", "correlacion_ref", "consultada_en"])
    || ![r.referencia, r.hecho_ref, r.decision_ref, r.auditoria_ref, r.correlacion_ref].every(referenciaConsultaValida)
    || r.hecho_ref !== hechoRef || !HASH.test(r.consumo_huella_sha256)
    || !instante(r.consultada_en) || r.version_consultada !== (encontrada ? datos.hecho_actual.version : 0)) throw new TypeError("meritos.consulta.respuesta_invalida");
  return congelar(datos);
}
