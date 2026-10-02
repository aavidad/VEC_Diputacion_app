/** Transporte local de Provisión, con contratos separados de Bolsa. */
const error = (codigo) => Object.assign(new Error(codigo), { codigo });
const MAXIMO_RESPUESTA = 4 * 1024 * 1024;
export function crearClienteConcursos({ fetchImpl = globalThis.fetch } = {}) {
  async function pedir(ruta, opciones = {}) {
    const respuesta = await fetchImpl(ruta, { credentials: "omit", redirect: "error", referrerPolicy: "no-referrer", cache: "no-store", ...opciones });
    if (!respuesta.ok) throw error([400, 422].includes(respuesta.status) ? "reglas_invalidas" : "simulacion_fallida");
    const texto = await respuesta.text();
    if (new TextEncoder().encode(texto).length > MAXIMO_RESPUESTA) throw error("respuesta_invalida");
    try { return JSON.parse(texto); } catch { throw error("respuesta_invalida"); }
  }
  return Object.freeze({
    async ejemplos({ signal } = {}) {
      const datos = await pedir("/api/provision/v1/configuracion-local", { signal });
      if (!Array.isArray(datos.ejemplos) || !datos.ejemplos.length || datos.ejemplos.length > 30
        || datos.ejemplos.some((e) => typeof e.referencia !== "string" || e.configuracion?.schema_version !== "provision.v1" || !e.entrada)) throw error("respuesta_invalida");
      return datos.ejemplos;
    },
    async simular(solicitud, { signal } = {}) {
      const datos = await pedir("/api/provision/v1/simulaciones", { method: "POST", signal,
        headers: { "Content-Type": "application/json", Accept: "application/json" }, body: JSON.stringify(solicitud) });
      return comprobarResultadoConcursos(datos, solicitud);
    },
  });
}

export function comprobarResultadoConcursos(datos, solicitud) {
  if (!datos || datos.schema_version !== "provision.simulacion.v1" || datos.alcance !== "simulacion"
    || datos.resultado?.convocatoria_ref !== solicitud.configuracion.convocatoria_ref) throw error("respuesta_invalida");
  const r = datos.resultado;
  const puntos = (v) => typeof v === "string" && /^(0|[1-9][0-9]{0,18})$/u.test(v);
  if (r.version_motor !== "provision.v1" || r.version_reglas !== solicitud.configuracion.version
    || ![r.huella_reglas, r.huella_entrada, r.huella_resultado].every((h) => /^[a-f0-9]{64}$/u.test(h ?? ""))
    || !Array.isArray(r.desglose) || !Array.isArray(r.incidencias)
    || r.desglose.some((d) => typeof d.familia !== "string" || !Array.isArray(d.detalles) || ![d.bruto, d.maximo, d.resultado].every(puntos))
    || (r.estado !== "simulacion_local_sin_efectos" || typeof r.completo !== "boolean" || (r.completo ? ![r.bruto, r.maximo_total, r.total].every(puntos) : r.total !== null))) throw error("respuesta_invalida");
  return datos;
}
