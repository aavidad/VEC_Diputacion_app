/** Transporte exclusivo de la herramienta local de simulación. */
const error = (codigo) => Object.assign(new Error(codigo), { codigo });
const MAXIMO_RESPUESTA = 4 * 1024 * 1024;
export function comprobarSimulacion(datos) {
  if (!datos || datos.alcance !== "simulacion" || !["vec.bolsa.simulacion_experiencia.v1", "vec.bolsa.simulacion_meritos.v1"].includes(datos.esquema)
    || !/^[a-f0-9]{64}$/u.test(datos.huella_resultado_sha256 ?? "") || !datos.resultado) throw error("respuesta_invalida");
  const r = datos.resultado;
  if (r.estado === "bloqueado") {
    if (Object.hasOwn(r, "total") || !Array.isArray(r.bloqueos ?? r.incidencias)) throw error("respuesta_invalida");
  } else if (r.estado !== "completado" || !/^(0|[1-9][0-9]*)$/u.test(r.total ?? "")) throw error("respuesta_invalida");
  return datos;
}
export function crearClienteBaremo({ fetchImpl = globalThis.fetch } = {}) {
  async function pedir(ruta, opciones = {}) {
    const respuesta = await fetchImpl(ruta, { credentials: "omit", redirect: "error", referrerPolicy: "no-referrer", cache: "no-store", ...opciones });
    if (!respuesta.ok) throw error(respuesta.status === 400 || respuesta.status === 422 ? "reglas_invalidas" : "simulacion_fallida");
    const texto = await respuesta.text();
    if (new TextEncoder().encode(texto).length > MAXIMO_RESPUESTA) throw error("respuesta_invalida");
    try { return JSON.parse(texto); } catch { throw error("respuesta_invalida"); }
  }
  return Object.freeze({
    async ejemplos({ signal } = {}) {
      const datos = await pedir("/ejemplos", { signal });
      if (!Array.isArray(datos.ejemplos) || !datos.ejemplos.length || datos.ejemplos.length > 30) throw error("respuesta_invalida");
      return datos.ejemplos;
    },
    async simular(solicitud, { signal } = {}) {
      const datos = comprobarSimulacion(await pedir("/simular", { method: "POST", signal,
        headers: { "Content-Type": "application/json", Accept: "application/json" }, body: JSON.stringify(solicitud) }));
      if (datos.esquema !== `vec.bolsa.simulacion_${solicitud.modo}.v1`
        || datos.convocatoria_ref !== (solicitud.reglas.identidad?.convocatoria_ref ?? solicitud.reglas.convocatoria_ref)) throw error("respuesta_invalida");
      return datos;
    },
  });
}
