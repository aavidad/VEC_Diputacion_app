/**
 * Cliente del marcaje propio. La frontera acreditada resuelve identidad,
 * empleado, instante y canal. La clave permanece en el comando entre reintentos.
 */
export const RUTA_MARCAJE_PROPIO_CRONOS = "/api/interna/cronos/marcajes/propio";

function errorMarcaje(codigo) {
  const error = new Error(codigo);
  error.code = codigo;
  return error;
}
function claveOperacionEstable(valor) {
  if (typeof valor !== "string" || !/^[a-z0-9][a-z0-9_-]{7,127}$/i.test(valor)) throw errorMarcaje("cronos_clave_operacion_invalida");
  return valor;
}
function reciboValido(cuerpo) {
  if (!cuerpo || Object.keys(cuerpo).length !== 1 || !cuerpo.recibo || typeof cuerpo.recibo !== "object") return false;
  const r = cuerpo.recibo;
  return Object.keys(r).length === 4 &&
    typeof r.referencia === "string" && /^recibo:cronos:[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/.test(r.referencia) &&
    typeof r.marcaje_original_ref === "string" && r.marcaje_original_ref.startsWith("marcaje:cronos:") &&
    typeof r.instante_utc === "string" && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d{1,6})?(Z|\+00:00)$/.test(r.instante_utc) &&
    Number.isFinite(Date.parse(r.instante_utc)) &&
    typeof r.replay === "boolean";
}

export function crearEjecutorHTTPMarcajesCronos({ fetchImpl = globalThis.fetch, ruta = RUTA_MARCAJE_PROPIO_CRONOS } = {}) {
  if (typeof fetchImpl !== "function" || ruta !== RUTA_MARCAJE_PROPIO_CRONOS) throw errorMarcaje("cronos_transporte_no_disponible");
  return async (comando) => {
    if (!comando || comando.tipo !== "registrar_fichaje" || !["entrada","salida","inicio_pausa","fin_pausa"].includes(comando.movimiento)) throw errorMarcaje("cronos_comando_invalido");
    const claveOperacion = claveOperacionEstable(comando.clave_operacion);
    const respuesta = await fetchImpl(ruta, {
      method: "POST", mode: "same-origin", credentials: "omit", redirect: "error", cache: "no-store",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ movimiento: comando.movimiento, clave_operacion: claveOperacion }),
    });
    const cuerpo = await respuesta.json().catch(() => null);
    if (!respuesta.ok) throw errorMarcaje(respuesta.status === 409 ? "cronos_clave_operacion_conflicto" : "cronos_marcaje_no_disponible");
    if (!reciboValido(cuerpo) || cuerpo.recibo.marcaje_original_ref !== "marcaje:cronos:" + claveOperacion) throw errorMarcaje("cronos_recibo_no_confiable");
    return cuerpo;
  };
}
