import { traducirPortal } from "./portal-i18n.js?v=20260926-i18n-v1";
/** Una confirmación HTTP no autoriza el paso siguiente sin un detalle real. */
export async function resolverSolicitudPropuestaLlamamiento({
  necesidadId,
  capacidad,
  cliente,
}) {
  if (typeof cliente?.solicitar !== "function") {
    return { ok: false, mensaje: traducirPortal("txt_el_servicio_de_propuestas_no_esta_disponible") };
  }
  const resultado = await cliente.solicitar({ necesidadId, capacidad });
  if (!resultado.ok) return resultado;
  return {
    ...resultado,
    avanzar: false,
    mensaje: traducirPortal("txt_confirmacion_recibida_detalle_no_disponible_la_c"),
  };
}
