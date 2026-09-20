const MENSAJES_ADMINISTRACION_ES = Object.freeze({
  accion_bloqueada: "Acción no disponible: requiere backend, autorización administrativa, auditoría y doble control.",
  ver_resumen: "Ver resumen", elemento: "Elemento", referencia: "Referencia", estado: "Estado", accion: "Acción",
  sin_resultados: "No hay resultados para el filtro local.", detalle: "Detalle de presentación", seleccionar: "Seleccione un elemento",
  tratamiento: "Tratamiento", consulta_local: "Solo consulta visual local", sin_valor: "—", solicitar_cambio: "Solicitar cambio",
  limite_roles: "No se asigna ni revoca ningún permiso.", limite_configuracion: "No se modifica ninguna configuración.",
  ia_titulo: "Configuración de IA local / RAG", ia_ayuda: "Desactivada por defecto. Estos campos solo ilustran una futura configuración local; no almacenan, envían ni prueban valores.",
  ia_endpoint: "Endpoint local", ia_modelo: "Modelo local", ia_indice: "Índice documental", ia_modelo_ejemplo: "Modelo pendiente de seleccionar", ia_indice_ejemplo: "Índice pendiente de aprobar",
  probar_conexion: "Probar conexión", activar_ia: "Activar IA local", prueba_bloqueada: "Prueba deshabilitada: no hay endpoint ni secretos configurados.", activar_bloqueada: "Activación deshabilitada: requiere evaluación, autorización, cifrado, auditoría y doble control.",
  ia_limite: "Pendiente: evaluación de impacto, fuentes documentales autorizadas, aislamiento local, control de acceso por documento, trazabilidad y revisión humana. La IA no decide permisos, requisitos ni efectos administrativos.",
  guardar_asignacion: "Guardar asignación", revocar_permiso: "Revocar permiso", guardar_catalogo: "Guardar catálogo", publicar_version: "Publicar versión", guardar_calendario: "Guardar calendario", publicar_calendario: "Publicar calendario", guardar_regla: "Guardar regla", publicar_regla: "Publicar regla", rotar_secreto: "Rotar secreto", activar_modulo: "Activar módulo", publicar_modulo: "Publicar módulo", guardar_conservacion: "Guardar conservación", revocar_acceso: "Revocar acceso",
  kpi_roles: "Roles de referencia", kpi_catalogos: "Catálogos visuales", kpi_conectores: "Conectores activos", kpi_ia: "IA local", ia_desactivada: "Desactivada", presentacion_sin_conexion: "Presentación sin conexión", responsable: "Responsable de referencia: {responsable} · {unidad}.",
  sobrelinea: "Administración · presentación RRHH", titulo: "Gobierno, configuración y controles", descripcion: "Vista demostrativa y segregada visualmente. No concede permisos ni configura sistemas.", pestanas: "Secciones de Administración", filtro: "Filtrar elementos visibles", aplicar_filtro: "Aplicar filtro", filtro_aplicado: "Filtro aplicado solo a ejemplos sintéticos visibles.", seccion_seleccionada: "Sección {etiqueta} seleccionada.",
  tab_resumen: "Resumen de gobierno", tab_roles: "Roles y permisos", tab_catalogos: "Catálogos", tab_calendarios: "Calendarios", tab_reglas: "Reglas", tab_conectores: "Conectores", tab_modulos: "Estado de módulos", tab_privacidad: "Privacidad y conservación", tab_ia: "IA local y RAG",
  estado_resumen: "Administración es una superficie de presentación: muestra ejemplos sintéticos, pero no consulta ni modifica configuración, permisos, secretos o datos corporativos.", estado_pendiente_1: "Superficie administrativa segregada, identidad reforzada y autorización administrativa exacta", estado_pendiente_2: "Backend de configuración versionada con auditoría segregada, recibos e idempotencia", estado_pendiente_3: "Cifrado, custodia y rotación de secretos fuera de la web", estado_pendiente_4: "Doble control para roles, retención, conectores, publicación y revocación", estado_conexion: "Sin API, credenciales, almacenamiento web ni conectores activos",
});

export function crearTraductorAdministracion(mensajes = MENSAJES_ADMINISTRACION_ES) {
  return (clave, variables = {}) => {
    if (!Object.hasOwn(mensajes, clave)) throw new Error(`clave i18n de Administración ausente: ${clave}`);
    return mensajes[clave].replace(/\{([a-z_]+)\}/gu, (_, nombre) => String(variables[nombre] ?? ""));
  };
}

export const CLAVES_I18N_ADMINISTRACION = Object.freeze(Object.keys(MENSAJES_ADMINISTRACION_ES));
