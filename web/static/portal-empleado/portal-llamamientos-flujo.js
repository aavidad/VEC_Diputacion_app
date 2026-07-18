import { validarPropuestaLlamamientoPresentacion } from "./portal-llamamientos-contrato.js?v=20260718-llamamientos-v1";

/**
 * Decide entre la demostración local y el adaptador real. La rama sintética no
 * conoce el cliente HTTP; la confirmación real nunca autoriza el paso siguiente.
 */
export async function resolverSolicitudPropuestaLlamamiento({
  modoPresentacion,
  necesidadId,
  capacidad,
  obtenerPresentacion,
  cliente,
}) {
  if (modoPresentacion === true) {
    try {
      if (typeof obtenerPresentacion !== "function") throw new Error("Adaptador de presentación no disponible.");
      const propuesta = validarPropuestaLlamamientoPresentacion(obtenerPresentacion(necesidadId));
      return { ok: true, sintetica: true, avanzar: true, propuesta };
    } catch (error) {
      return { ok: false, mensaje: error instanceof Error ? error.message : "No se pudo cargar la presentación." };
    }
  }
  if (typeof cliente?.solicitar !== "function") {
    return { ok: false, mensaje: "El servicio de propuestas no está disponible." };
  }
  const resultado = await cliente.solicitar({ necesidadId, capacidad });
  if (!resultado.ok) return resultado;
  return {
    ...resultado,
    sintetica: false,
    avanzar: false,
    mensaje: "Confirmación recibida. Detalle no disponible; la configuración permanece bloqueada.",
  };
}
