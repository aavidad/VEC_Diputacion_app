import { normalizarConfiguracion, normalizarPropuesta, normalizarOpciones, solicitudPropuesta, solicitudControl } from "./contratos.js?v=20261001-cs09-copias-ux-v2";

const BASE = "/api/admin/copias/v1";
const REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9:_-]{0,159}$/u;
const HUELLA = /^[a-f0-9]{64}$/u;
const CAPACIDADES = ["consultar", "lanzar", "configurar_calendario", "configurar_retencion", "proponer", "revisar", "ejecutar"];

export class ErrorCopias extends Error {
  constructor(codigo = "respuesta_invalida") { super(codigo); this.codigo = codigo; }
}
export function referencia(valor) {
  if (typeof valor !== "string" || !REFERENCIA.test(valor)) throw new ErrorCopias();
  return valor;
}
export function objeto(valor) {
  if (!valor || typeof valor !== "object" || Array.isArray(valor)) throw new ErrorCopias();
  return valor;
}
export function entero(valor) {
  if (!Number.isSafeInteger(valor) || valor < 0) throw new ErrorCopias();
  return valor;
}
export function fecha(valor) {
  if (typeof valor !== "string" || !/^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d+)?Z$/u.test(valor) || !Number.isFinite(Date.parse(valor))) throw new ErrorCopias();
  return valor;
}
export function codigo(valor) {
  if (typeof valor !== "string" || !/^[a-z][a-z0-9_]{0,79}$/u.test(valor)) throw new ErrorCopias();
  return valor;
}
function confirmarRecibo(valor, operacion) {
  const recibo = normalizarRecibo(valor); if (recibo.operacion_ref !== operacion) throw new ErrorCopias(); return recibo;
}
function opcional(datos, clave, validar) {
  return datos[clave] === undefined ? {} : { [clave]: validar(datos[clave]) };
}
export function normalizarCopia(valor) {
  const c = objeto(valor), compatibilidad = objeto(c.compatibilidad);
  return Object.freeze({
    copia_ref: referencia(c.copia_ref), version: entero(c.version), tipo: codigo(c.tipo), estado: codigo(c.estado),
    iniciada_en: fecha(c.iniciada_en), compatibilidad: Object.freeze({ estado: codigo(compatibilidad.estado), ...(typeof compatibilidad.clave_i18n === "string" && /^[a-z][a-z0-9_.-]{0,179}$/u.test(compatibilidad.clave_i18n) ? { clave_i18n: compatibilidad.clave_i18n } : {}) }),
    ...opcional(c, "finalizada_en", fecha), ...opcional(c, "tamano_bytes", entero),
    ...opcional(c, "release_ref", referencia), ...opcional(c, "retencion_hasta", fecha),
    ...opcional(c, "huella_sha256", v => { if (!HUELLA.test(v)) throw new ErrorCopias(); return v; }),
    componentes: c.componentes === undefined ? undefined : lista(c.componentes).map(v => {
      const d = objeto(v); return Object.freeze({ componente: codigo(d.componente), estado: codigo(d.estado) });
    }),
  });
}
export function lista(valor) {
  if (!Array.isArray(valor) || valor.length > 500) throw new ErrorCopias();
  return valor;
}
export function normalizarRecibo(valor) {
  const r = objeto(valor);
  return Object.freeze({ operacion_ref: referencia(r.operacion_ref), recibo_ref: referencia(r.recibo_ref), recurso_ref: referencia(r.recurso_ref),
    version: entero(r.version), estado: codigo(r.estado), registrado_en: fecha(r.registrado_en) });
}
export function crearClienteCopias({ fetchImpl = globalThis.fetch } = {}) {
  if (typeof fetchImpl !== "function") throw new ErrorCopias("no_disponible");
  async function pedir(ruta, { signal, cuerpo, method = cuerpo === undefined ? "GET" : "POST" } = {}) {
    let respuesta;
    try {
      respuesta = await fetchImpl(BASE + ruta, { method, signal, credentials: "same-origin", cache: "no-store",
        redirect: "error", referrerPolicy: "no-referrer", headers: { Accept: "application/json", ...(cuerpo === undefined ? {} : { "Content-Type": "application/json" }) },
        ...(cuerpo === undefined ? {} : { body: JSON.stringify(cuerpo) }) });
    } catch (error) { if (error?.name === "AbortError") throw error; throw new ErrorCopias("no_disponible"); }
    if (!respuesta.ok) {
      throw new ErrorCopias(respuesta.status === 401 || respuesta.status === 403 ? "denegado"
        : respuesta.status === 409 ? "conflicto" : respuesta.status === 400 ? "solicitud_invalida" : "no_disponible");
    }
    if (!/^application\/json(?:;|$)/iu.test(respuesta.headers?.get?.("Content-Type") ?? "")) throw new ErrorCopias();
    const declarado = Number(respuesta.headers.get("Content-Length"));
    if (declarado > 1024 * 1024) throw new ErrorCopias();
    const texto = await respuesta.text();
    if (texto.length > 1024 * 1024) throw new ErrorCopias();
    try { return objeto(JSON.parse(texto)); } catch { throw new ErrorCopias(); }
  }
  return Object.freeze({
    async capacidades(opciones) {
      const d = objeto((await pedir("/capacidades", opciones)).capacidades);
      return Object.freeze(Object.fromEntries(CAPACIDADES.map(clave => [clave, d[clave] === true])));
    },
    async listar({ signal, cursor } = {}) {
      const d = await pedir(cursor === undefined ? "" : "?cursor=" + encodeURIComponent(referencia(cursor)), { signal });
      return Object.freeze({ copias: Object.freeze(lista(d.copias).map(normalizarCopia)), version: entero(d.version), ...opcional(d, "cursor_siguiente", referencia) });
    },
    async detalle(ref, opciones) { return normalizarCopia((await pedir("/" + encodeURIComponent(referencia(ref)), opciones)).copia); },
    async configuracion({ signal } = {}) {
      return normalizarConfiguracion((await pedir("/calendario", { signal })).configuracion);
    },
    async retencion({ signal } = {}) {
      return normalizarConfiguracion((await pedir("/retencion", { signal })).configuracion);
    },
    async guardarConfiguracion(ambito, solicitud, { signal } = {}) {
      if (!["calendario", "retencion"].includes(ambito)) throw new ErrorCopias();
      const configuracion = normalizarConfiguracion({ version: solicitud.version_esperada, politica: solicitud.politica });
      return confirmarRecibo((await pedir("/" + ambito, { signal, cuerpo: { operacion_ref: referencia(solicitud.operacion_ref),
        version_esperada: configuracion.version, politica: configuracion.politica } })).recibo, solicitud.operacion_ref);
    },
    async propuestas({ signal } = {}) {
      return Object.freeze(lista((await pedir("/propuestas", { signal })).propuestas).map(normalizarPropuesta));
    },
    async opcionesRestauracion({ signal } = {}) { return normalizarOpciones(await pedir("/opciones-restauracion", { signal })); },
    async proponer(solicitud, { signal } = {}) {
      const cuerpo = solicitudPropuesta(solicitud), propuesta = normalizarPropuesta((await pedir("/propuestas", { signal, cuerpo })).propuesta);
      for (const k of ["conjunto_ref", "destino_ref", "motivo_ref", "ventana_ref"]) if (propuesta[k] !== cuerpo[k]) throw new ErrorCopias();
      for (const k of ["ventana_inicio", "ventana_fin", "caduca_en"]) if (Date.parse(propuesta[k]) !== Date.parse(cuerpo[k])) throw new ErrorCopias();
      return propuesta;
    },
    async revisar(ref, solicitud, { signal } = {}) {
      const cuerpo = solicitudControl(solicitud), propuesta = normalizarPropuesta((await pedir("/propuestas/" + encodeURIComponent(referencia(ref)) + "/revision", { signal, cuerpo })).propuesta);
      if (propuesta.propuesta_ref !== ref || propuesta.destino_ref !== cuerpo.destino_ref || propuesta.huella_sha256 !== cuerpo.propuesta_huella_sha256 || propuesta.version < cuerpo.version_esperada) throw new ErrorCopias();
      return propuesta;
    },
    async ejecutar(ref, solicitud, { signal } = {}) {
      return confirmarRecibo((await pedir("/propuestas/" + encodeURIComponent(referencia(ref)) + "/ejecucion", { signal, cuerpo: solicitudControl(solicitud) })).recibo, solicitud.operacion_ref);
    },
    async lanzar({ operacion_ref, version_esperada, tipo = "completa" }, { signal } = {}) {
      if (tipo !== "completa") throw new ErrorCopias("solicitud_invalida");
      return confirmarRecibo((await pedir("/lanzamientos", { signal, cuerpo: { operacion_ref: referencia(operacion_ref), version_esperada: entero(version_esperada), tipo } })).recibo, operacion_ref);
    },
  });
}
