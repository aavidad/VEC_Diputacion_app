/** Catálogo de la vista. Los hechos y decisiones llegan como datos, nunca como etiquetas. */
export const MENSAJES_MERITOS_ES = Object.freeze({
  sobrelinea: "Área personal", titulo: "Méritos", descripcion: "Un inventario de hechos aportados una vez y reutilizables con su procedencia.",
  ayuda: "Ayuda", ayuda_texto: "Una titulación o un servicio se aporta una vez. Cada convocatoria comprueba por separado sus requisitos de acceso y calcula, si procede, su propia puntuación. Una declaración no equivale a acreditación y los puntos de un proceso no se trasladan a otro.",
  inventario: "Inventario", requisitos: "Requisitos de acceso", valoraciones: "Valoraciones por convocatoria", navegacion: "Secciones de Méritos",
  no_configurado: "Fuente de méritos no conectada", no_configurado_texto: "Aún no se puede consultar el expediente personal de méritos. No se han cargado datos de ejemplo.",
  cargando: "Cargando méritos", cargando_texto: "Esperando una respuesta de la fuente autorizada.",
  vacio: "Sin méritos aportados", vacio_texto: "La fuente consultada no devolvió méritos para esta persona.",
  denegado: "Acceso denegado", denegado_texto: "No hay autorización para consultar estos datos con el perfil y finalidad actuales.",
  error: "No se pudieron consultar los méritos", error_texto: "La consulta falló. Inténtalo de nuevo cuando esté disponible el servicio.",
  disponible: "Consulta disponible", solo_lectura: "Consulta de solo lectura", consulta_pendiente: "La consulta y aportación requieren fuente, autorización, evidencia e historial con recibo.",
  total: "Hechos registrados", acreditados: "Acreditados", pendientes: "Pendientes", rechazados: "Rechazados", sin_dato: "Sin dato", registros: "{numero} registros",
  inventario_ayuda: "Cada fila representa un hecho aportado una vez. Abre su detalle para consultar fuente, evidencia y vigencia.",
  requisitos_ayuda: "El acceso se verifica contra las bases versionadas de cada convocatoria. La puntuación no decide si se cumple un requisito.",
  valoraciones_ayuda: "Cada puntuación pertenece exclusivamente a su convocatoria y a la versión de sus bases.",
  sin_resultados: "No hay méritos con este estado.", sin_requisitos: "No hay comprobaciones de acceso disponibles para esta consulta.", sin_valoraciones: "No hay valoraciones disponibles para esta consulta.",
  nombre: "Hecho aportado", tipo: "Tipo", fuente: "Fuente", evidencia: "Evidencia", vigencia: "Vigencia", estado: "Estado", detalle: "Detalle del mérito", abrir_detalle: "Abrir detalle de {nombre}", cerrar_detalle: "Cerrar detalle de {nombre}",
  convocatoria: "Convocatoria", bases: "Bases", requisito: "Requisito", resultado: "Resultado", motivo: "Motivo", procedencia: "Procedencia", hito: "Hito de cumplimiento", merito: "Mérito", puntos: "Puntos", criterio: "Criterio",
  declarado: "Declarado", pendiente: "Pendiente", acreditado: "Acreditado", rechazado: "Rechazado", cumple: "Cumple", no_cumple: "No cumple",
  filtrar: "Filtrar por estado", todos: "Todos", aportar: "Aportar mérito", aportar_pendiente: "La aportación estará disponible al conectar el expediente, los documentos, la autorización y el recibo.",
  aviso_sin_evidencia: "Evidencia no disponible", aviso_sin_vigencia: "Vigencia no informada", aviso_sin_fuente: "Fuente no informada", aviso_sin_version: "Versión de bases no informada",
  anuncio_inicial: "Méritos: fuente personal no conectada.", anuncio_seccion: "Sección: {nombre}", anuncio_actualizado: "Méritos actualizados.",
});

const CLAVES = Object.freeze(Object.keys(MENSAJES_MERITOS_ES));
export function crearTraductorMeritos(mensajes = MENSAJES_MERITOS_ES) {
  if (!mensajes || typeof mensajes !== "object" || Array.isArray(mensajes)
    || Object.keys(mensajes).length !== CLAVES.length
    || CLAVES.some((clave) => typeof mensajes[clave] !== "string")
    || Object.keys(mensajes).some((clave) => !CLAVES.includes(clave))) throw new TypeError("catálogo de Méritos incompleto");
  return (clave, variables = {}) => {
    if (!Object.hasOwn(mensajes, clave)) throw new TypeError(`clave de Méritos no disponible: ${clave}`);
    return mensajes[clave].replace(/\{([a-z_]+)\}/gu, (_, nombre) => String(variables[nombre] ?? ""));
  };
}
