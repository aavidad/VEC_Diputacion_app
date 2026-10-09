import { registroB2, referenciaB2, validarConsultaB2, validarSolicitudPlanB2, validarSolicitudConfirmacionB2, validarReciboB2 } from "./contrato-incorporacion-personal-b2.js";
import { validarSignal, ejecutarAbortable, cancelarRespuesta, validarTipoJSON, longitudDeclarada, serializarAcotado, construirCabeceras } from "./cliente-http-transporte.js?v=20261009-ct-bolsa-cohorte-v6";

export const RUTA_PLAN_B2 = "/api/vec/contratacion-temporal/incorporacion-personal-b2/plan/v1";
export const RUTA_CONFIRMAR_B2 = "/api/vec/contratacion-temporal/incorporacion-personal-b2/confirmar/v1";
const ERRORES = new Map([[400, ["peticion_no_valida"]], [405, ["metodo_no_permitido"]], [422, ["contenido_no_valido"]], [401, ["autenticacion_requerida"]],
  [403, ["acceso_denegado"]], [404, ["recurso_no_encontrado"]], [409, ["conflicto", "preparacion_pendiente", "version_en_conflicto", "clave_idempotencia_reutilizada"]],
  [503, ["servicio_no_disponible"]]]);
export class ErrorIncorporacionPersonalB2 extends Error {
  constructor(codigo, datos = {}) { super(codigo); this.name = "ErrorIncorporacionPersonalB2"; this.codigo = codigo; Object.assign(this, datos); }
}
const error = (codigo, datos) => new ErrorIncorporacionPersonalB2(codigo, datos);


// JSON de Go conserva escapes HTML. La forma y los vínculos se validan después,
// sin exigir que JSON.stringify reproduzca esos escapes del transporte.
async function leerJSONB2(respuesta, signal) {
  validarTipoJSON(respuesta, error);
  const declarada = longitudDeclarada(respuesta, 256 * 1024, error);
  if (!respuesta.body?.getReader) throw error("respuesta_no_incremental");
  const lector = respuesta.body.getReader();
  const fragmentos = []; let total = 0;
  try {
    while (true) {
      const parte = await ejecutarAbortable(() => lector.read(), signal, () => lector.cancel(), null, error);
      if (!parte || typeof parte.done !== "boolean") throw error("respuesta_incompatible");
      if (parte.done) break;
      if (!(parte.value instanceof Uint8Array) || !parte.value.length) throw error("respuesta_incompatible");
      total += parte.value.length; fragmentos.push(parte.value);
      if (total > 256 * 1024 || fragmentos.length > 256) throw error("respuesta_excesiva");
    }
    if (declarada !== null && declarada !== total) throw error("respuesta_incompatible");
    const bytes = new Uint8Array(total); let posicion = 0;
    for (const f of fragmentos) { bytes.set(f, posicion); posicion += f.length; }
    return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes));
  } catch (e) { await cancelarRespuesta(respuesta, lector); throw e; }
  finally { lector.releaseLock?.(); }
}

export function crearClienteIncorporacionPersonalB2HTTP({ fetchImpl = globalThis.fetch, HeadersImpl = globalThis.Headers } = {}) {
  if (typeof fetchImpl !== "function" || typeof HeadersImpl !== "function") throw error("cliente_no_disponible");
  async function pedir(ruta, entrada, opciones, validar) {
    const o = opciones === undefined ? {} : registroB2(opciones, Object.hasOwn(opciones, "signal") ? ["signal"] : []);
    const signal = validarSignal(o.signal, error);
    const conCuerpo = entrada !== undefined;
    const body = conCuerpo ? serializarAcotado(entrada, 8192, error) : undefined;
    let respuesta;
    try {
      respuesta = await ejecutarAbortable(() => fetchImpl(ruta, {
        method: conCuerpo ? "POST" : "GET", headers: construirCabeceras(HeadersImpl, undefined, conCuerpo),
        ...(conCuerpo ? { body } : {}), signal, credentials: "same-origin", mode: "same-origin",
        cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
      }), signal, null, cancelarRespuesta, error);
      if (!respuesta || respuesta.redirected || ![200, ...ERRORES.keys()].includes(respuesta.status)) throw error("respuesta_incompatible");
      const json = await leerJSONB2(respuesta, signal);
      if (respuesta.status >= 400) {
        const d = registroB2(registroB2(json, ["error"]).error, ["codigo", "clave_i18n", "correlacion_ref"]);
        if (!ERRORES.get(respuesta.status)?.includes(d.codigo)
          || ![ `api.contratacion_temporal.incorporacion_personal_b2.error.${d.codigo}`, `api.vec.ruta_exacta.error.${d.codigo}` ].includes(d.clave_i18n)
          || typeof d.correlacion_ref !== "string" || !/^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u.test(d.correlacion_ref)) throw error("respuesta_error_no_valida", { estado: respuesta.status });
        throw error(d.codigo, { estado: respuesta.status, envelopeValido: true });
      }
      return validar(registroB2(json, ["data"]).data);
    } catch (e) {
      if (respuesta) await cancelarRespuesta(respuesta);
      if (e instanceof ErrorIncorporacionPersonalB2) throw e;
      throw error(signal?.aborted ? "operacion_abortada" : "respuesta_no_verificada");
    }
  }
  return Object.freeze({
    consultar(expedienteRef, opciones) {
      if (!referenciaB2(expedienteRef)) throw error("contenido_no_valido");
      return pedir(`${RUTA_PLAN_B2}?expediente_ref=${encodeURIComponent(expedienteRef)}`, undefined, opciones, (v) => validarConsultaB2(v, expedienteRef));
    },
    preparar(solicitud, opciones) {
      const entrada = validarSolicitudPlanB2(solicitud);
      return pedir(RUTA_PLAN_B2, entrada, opciones, (v) => {
        const c = validarConsultaB2(v, entrada.expediente_ref);
        if (!c.plan || JSON.stringify(c.plan.intencion) !== JSON.stringify(entrada)) throw error("respuesta_no_verificada");
        return c;
      });
    },
    confirmar(solicitud, opciones) {
      const entrada = validarSolicitudConfirmacionB2(solicitud);
      return pedir(RUTA_CONFIRMAR_B2, entrada, opciones, (v) => validarReciboB2(v, entrada));
    },
  });
}
