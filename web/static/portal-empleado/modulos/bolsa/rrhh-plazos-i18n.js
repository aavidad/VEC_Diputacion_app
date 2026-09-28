/** Textos de la configuración versionada de ofertas de Bolsa. Se incorpora al catálogo común del portal. */
export const MENSAJES_RRHH_PLAZOS_ES = Object.freeze({
  rrhh_plazos_titulo: "Política de ofertas",
  rrhh_plazos_subtitulo: "Plazo, adjudicación y cobertura de la bolsa seleccionada",
  rrhh_plazos_cargando: "Cargando política…",
  rrhh_plazos_vacio: "Aún no hay una política configurada para esta bolsa.",
  rrhh_plazos_error_carga: "No se pudo consultar la política. Reinténtelo.",
  rrhh_plazos_denegado: "Su sesión no permite consultar la política de esta bolsa.",
  rrhh_plazos_reintentar: "Reintentar",
  rrhh_plazos_revisar: "Revisar versión vigente",
  rrhh_plazos_ejemplo: "Regla de ejemplo",
  rrhh_plazos_borrador: "Borrador",
  rrhh_plazos_activa: "Activa",
  rrhh_plazos_version: "Versión {version}",
  rrhh_plazos_plazo_titulo: "Plazo de disposición",
  rrhh_plazos_cantidad: "Cantidad",
  rrhh_plazos_unidad: "Unidad",
  rrhh_plazos_computo: "Inicio del cómputo",
  rrhh_plazos_zona: "Zona horaria",
  rrhh_plazos_municipio_sede: "Municipio de la sede (código INE)",
  rrhh_plazos_orden_titulo: "Orden de adjudicación",
  rrhh_plazos_criterio: "Criterio de prelación",
  rrhh_plazos_elegibilidad: "Personas elegibles",
  rrhh_plazos_no_cubierta_titulo: "Oferta no cubierta",
  rrhh_plazos_accion: "Acción",
  rrhh_plazos_condicion: "Condición",
  rrhh_plazos_motivo: "Motivo del cambio",
  rrhh_plazos_guardar: "Guardar nueva versión",
  rrhh_plazos_guardando: "Guardando…",
  rrhh_plazos_reintentar_guardado: "Reintentar el mismo guardado",
  rrhh_plazos_guardada: "Versión {version} guardada. Recibo {recibo}.",
  rrhh_plazos_error_guardado: "No se pudo guardar. Consulte la política antes de repetir la operación.",
  rrhh_plazos_conflicto: "La política cambió. Se conserva el formulario; consulte la versión vigente antes de guardar.",
  rrhh_plazos_validacion: "Revise los campos señalados por el formulario.",
  rrhh_plazos_sin_permiso: "Su sesión no permite modificar esta política.",
  rrhh_plazos_sin_edicion: "La edición no está disponible para esta sesión.",
  rrhh_plazos_comprobando_edicion: "Comprobando permiso de edición…",
  rrhh_plazos_error_capacidad: "No se pudo comprobar el permiso de edición. La política sigue en solo lectura.",
  rrhh_plazos_desconocido: "Valor no disponible",
  rrhh_plazos_calendario_fuente: "Calendario",
  rrhh_plazos_calendario_municipio: "Hábil del municipio INE {municipio}",
  rrhh_plazos_calendario_pendiente: "Pendiente de municipio",
  rrhh_plazos_orden_vigente: "Orden vigente de la bolsa",
  rrhh_plazos_disposicion_en_plazo: "Disposición presentada dentro del plazo",
  rrhh_plazos_llamamiento_directo: "Pasar a llamamiento directo",
  rrhh_plazos_sin_elegibles: "Ninguna disposición elegible",
  rrhh_plazos_dias_naturales: "Días naturales",
  rrhh_plazos_dias_habiles: "Días hábiles",
  rrhh_plazos_horas_naturales: "Horas naturales",
  rrhh_plazos_administrativo: "Cómputo administrativo",
  rrhh_plazos_continuo_utc: "Desde la apertura expresa de la oferta, en tiempo continuo UTC",
  rrhh_plazos_calendario_continuo_utc: "Tiempo continuo UTC, sin excluir festivos",
  rrhh_plazos_ayuda: "Ayuda sobre la política de ofertas",
  rrhh_plazos_ayuda_contenido: "La regla de ejemplo propone 48 horas naturales desde la apertura expresa de cada oferta. Cada oferta conserva apertura y vencimiento exactos en UTC; la hora local se interpreta en Europe/Madrid, también al cambiar el horario de verano. Es un plazo operativo provisional: el correo no acredita notificación ni inicia un plazo legal. Consulte la versión de la bolsa antes de publicar; el servidor vuelve a comprobar el permiso y genera un recibo.",
  rrhh_plazos_confirmacion: "La resolución requiere confirmación de RRHH",
  rrhh_plazos_segunda_validacion: "Segunda confirmación",
  rrhh_plazos_dos_personas: "Exigida a otra persona autorizada",
});

export const MENSAJES_RRHH_PLAZOS_EN = Object.freeze({
  rrhh_plazos_segunda_validacion: "Second confirmation",
  rrhh_plazos_dos_personas: "Required from another authorized person",
});

export function crearTraductorRRHHPlazos(catalogo = MENSAJES_RRHH_PLAZOS_ES) {
  return (clave, variables = {}) => {
    if (!Object.hasOwn(catalogo, clave) || typeof catalogo[clave] !== "string") {
      throw new TypeError(`clave i18n de política de ofertas desconocida: ${clave}`);
    }
    return catalogo[clave].replace(/\{([a-z_]+)\}/gu, (_texto, nombre) => String(variables[nombre] ?? ""));
  };
}
