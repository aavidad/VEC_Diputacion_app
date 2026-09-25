/**
 * Petición RRHH p.4: histórico de cambios del expediente. Misma ruta, método y
 * autorización que el detalle RRHH; solo cambia la representación pedida.
 */
import { RUTAS_CONSULTA_RRHH } from "./cliente-http-consultas-rrhh.js";

export const ACCEPT_CAMBIOS_EXPEDIENTE = "application/vnd.vec.contratacion-temporal.cambios-expediente+json";
export const ESQUEMA_CAMBIOS_EXPEDIENTE = "vec.contratacion_temporal.rrhh.cambios_expediente.v1";
const MAXIMO_RESPUESTA = 256 * 1024;
const MAXIMO_CAMBIOS = 500;
const REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const RUTA = /^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*|\[[0-9]{1,4}\])*$/u;
const ORIGEN = /^[a-z][a-z0-9_]{1,63}$/u;
const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/u;

function exactos(valor, campos) {
  return valor !== null && typeof valor === "object" && !Array.isArray(valor)
    && Object.keys(valor).length === campos.length && campos.every((c) => Object.hasOwn(valor, c));
}

function valor(v) {
  return v === null || (typeof v === "string" && v.length <= 200 && !/[\u0000\n\r]/u.test(v));
}

export function validarCambiosExpediente(cuerpo, expedienteRef) {
  const datos = cuerpo?.data;
  if (!exactos(cuerpo, ["data"]) || !exactos(datos, ["esquema", "expediente_ref", "version_expediente", "cambios", "recortado"])
    || typeof datos.recortado !== "boolean"
    || datos.esquema !== ESQUEMA_CAMBIOS_EXPEDIENTE || datos.expediente_ref !== expedienteRef
    || !Number.isSafeInteger(datos.version_expediente) || datos.version_expediente < 1
    || !Array.isArray(datos.cambios) || datos.cambios.length > MAXIMO_CAMBIOS) {
    throw new TypeError("histórico de cambios no válido");
  }
  let anterior = 0;
  for (const c of datos.cambios) {
    if (!exactos(c, ["version_expediente", "registrada_en", "origen_version", "ruta", "valor_anterior", "valor_nuevo"])
      || !Number.isSafeInteger(c.version_expediente) || c.version_expediente < 2
      || c.version_expediente > datos.version_expediente || c.version_expediente < anterior
      || typeof c.registrada_en !== "string" || !INSTANTE.test(c.registrada_en)
      || typeof c.origen_version !== "string" || !ORIGEN.test(c.origen_version)
      || typeof c.ruta !== "string" || c.ruta.length > 400 || !RUTA.test(c.ruta)
      || !valor(c.valor_anterior) || !valor(c.valor_nuevo)
      || (c.valor_anterior === null && c.valor_nuevo === null)) {
      throw new TypeError("cambio de expediente no válido");
    }
    anterior = c.version_expediente;
  }
  return Object.freeze({ ...structuredClone(datos), cambios: Object.freeze(structuredClone(datos.cambios)) });
}

export function crearClienteHTTPCambiosExpediente({ fetchImpl = globalThis.fetch } = {}) {
  return Object.freeze({
    async consultarCambios(solicitud, { signal } = {}) {
      if (!exactos(solicitud, ["expediente_ref", "version_observada"])
        || typeof solicitud.expediente_ref !== "string" || !REFERENCIA.test(solicitud.expediente_ref)
        || !Number.isSafeInteger(solicitud.version_observada) || solicitud.version_observada < 0) {
        throw new TypeError("solicitud de cambios no válida");
      }
      if (typeof fetchImpl !== "function") throw new Error("servicio_no_disponible");
      const respuesta = await fetchImpl(RUTAS_CONSULTA_RRHH.detalleRRHH, {
        method: "POST",
        headers: { "Content-Type": "application/json", Accept: ACCEPT_CAMBIOS_EXPEDIENTE },
        body: JSON.stringify({ expediente_ref: solicitud.expediente_ref, version_observada: solicitud.version_observada }),
        signal, mode: "same-origin", credentials: "same-origin",
        cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
      });
      if (respuesta.redirected || respuesta.status !== 200) {
        void respuesta.body?.cancel?.().catch(() => {});
        throw new Error("consulta_no_disponible");
      }
      if (!/^application\/json(?:;\s*charset=utf-8)?$/iu.test(respuesta.headers.get("Content-Type") ?? "")) {
        throw new TypeError("histórico de cambios no válido");
      }
      const texto = await respuesta.text();
      if (texto.length > MAXIMO_RESPUESTA) throw new TypeError("histórico de cambios no válido");
      return validarCambiosExpediente(JSON.parse(texto), solicitud.expediente_ref);
    },
  });
}
