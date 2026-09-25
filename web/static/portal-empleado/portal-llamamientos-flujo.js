/** Una confirmación HTTP no autoriza el paso siguiente sin un detalle real. */
export async function resolverSolicitudPropuestaLlamamiento({
  necesidadId,
  capacidad,
  cliente,
}) {
  if (typeof cliente?.solicitar !== "function") {
    return { ok: false, mensaje: "El servicio de propuestas no está disponible." };
  }
  const resultado = await cliente.solicitar({ necesidadId, capacidad });
  if (!resultado.ok) return resultado;
  return {
    ...resultado,
    avanzar: false,
    mensaje: "Confirmación recibida. Detalle no disponible; la configuración permanece bloqueada.",
  };
}
