/** Textos castellanos propios de la presentación DEMO de Personal. */
export const MENSAJES_PERSONAL_ES = Object.freeze({
  sobrelinea: "Portal del Empleado → Personal",
  titulo: "Mi información de Personal",
  presentacion_demo: "Presentación DEMO · datos sintéticos y efímeros",
  aviso_demo: "Esta pantalla no consulta datos reales ni produce efectos administrativos.",
  relacion_titulo: "Relación y puesto actual",
  relacion_ayuda: "Referencia visual DEMO sin eficacia jurídica ni acto de personal.",
  servicios_titulo: "Servicios informativos",
  servicios_ayuda: "Los periodos se muestran solo como ejemplo; no se calcula antigüedad, trienios ni derechos.",
  formacion_titulo: "Formación",
  formacion_ayuda: "Relación demostrativa sin acreditación, baremación ni validez curricular.",
  nominas_titulo: "Nóminas y bases",
  nominas_ayuda: "No se muestran importes, bases, retenciones ni recibos reales.",
  dietas_titulo: "Dietas cobradas",
  dietas_ayuda: "Las referencias DEMO no acreditan liquidación, inclusión en nómina ni pago.",
  actualizado: "Actualizado: {fecha}",
  cab_referencia: "Referencia",
  cab_periodo: "Periodo",
  cab_estado: "Estado",
  cab_descripcion: "Descripción",
  cab_desde: "Desde",
  cab_hasta: "Hasta",
  cab_observacion: "Observación",
  sin_importes: "Sin importes reales",
  sin_pago: "Sin pago real",
  sin_datos: "No hay datos para mostrar en este escenario DEMO.",
  catalogo_sobrelinea: "Portal del Empleado → Personal",
  catalogo_titulo: "Catálogo profesional vigente",
  catalogo_ayuda: "Consulta RRHH de solo lectura; no contiene datos de empleado, nómina ni servicios.",
  catalogo_cargando: "Cargando categorías profesionales…",
  catalogo_error: "No se pudo consultar el catálogo profesional. No se muestran datos anteriores.",
  catalogo_demo: "demostracion:true · El catálogo mostrado es una demostración pendiente de validación RRHH; no es una RPT aprobada.",
  catalogo_publicado: "Procedencia publicada por Personal: el catálogo no acredita por sí solo una RPT aprobada.",
  catalogo_fuente: "Fuente: revisión {revision}; catálogo {catalogo} v{version}. {aviso}",
  catalogo_buscar: "Buscar categoría",
  catalogo_area: "Área",
  catalogo_accion_buscar: "Buscar",
  catalogo_vacio: "No hay categorías profesionales para estos filtros.",
  catalogo_paginacion: "Paginación de categorías profesionales",
  catalogo_anterior: "Anterior",
  catalogo_siguiente: "Siguiente",
  catalogo_recuento_uno: "{total} categoría",
  catalogo_recuento_otro: "{total} categorías",
  catalogo_cabecera_nombre: "Categoría",
  catalogo_cabecera_area: "Área",
  catalogo_cabecera_estado: "Estado",
  catalogo_filtro_invalido: "El filtro local no es válido. Revise la búsqueda o el área.",
  rpt_sobrelinea: "Portal del Empleado → Personal",
  rpt_titulo: "Puestos y categorías RPT",
  rpt_ayuda: "Consulta pública de solo lectura. No muestra ocupantes, datos personales, nivel ni complementos.",
  rpt_buscar: "Buscar puesto o categoría",
  rpt_accion_buscar: "Buscar",
  rpt_cargando: "Cargando RPT pública…",
  rpt_error: "No se pudo consultar la RPT pública. No se muestran datos anteriores.",
  rpt_fuente: "Fuente: {documento}. {aviso}",
  rpt_huella: "Importación: {importacion} · Huella SHA-256: {huella}",
  rpt_sin_escalas: "No consignada",
  rpt_vacio: "No hay puestos o categorías para estos filtros.",
  rpt_tabla: "Tabla de puestos y categorías RPT",
  rpt_paginacion: "Paginación RPT",
  rpt_anterior: "Anterior",
  rpt_siguiente: "Siguiente",
  rpt_recuento_uno: "{total} categoría RPT",
  rpt_recuento_otro: "{total} categorías RPT",
  rpt_cabecera_clave: "Clave",
  rpt_cabecera_denominacion: "Denominación",
  rpt_cabecera_grupos: "Grupos",
  rpt_cabecera_escalas: "Escalas",
  rpt_cabecera_puestos: "Puestos",
  rpt_cabecera_dotacion: "Dotación",
  estructura_sobrelinea: "Portal del Empleado → Personal",
  estructura_titulo: "Estructura organizativa de referencia",
  estructura_ayuda: "Consulta pública de solo lectura con 66 unidades; no contiene personas, ocupación ni permisos.",
  estructura_cargando: "Cargando estructura organizativa…",
  estructura_error: "No se pudo consultar la estructura organizativa. No se muestran datos anteriores.",
  estructura_aviso: "Fuente DEMO en preparación: no acredita vigencia administrativa, ocupación, cadena de mando, gestión efectiva ni autorización. Los títulos de jefatura son puestos de referencia.",
  estructura_fuente: "Fuente: revisión {revision}; actualizada {actualizada_en}. {aviso}",
  estructura_huella: "Huella SHA-256 del paquete: {huella}",
  estructura_tabla: "Tabla de estructura organizativa de referencia",
  estructura_recuento: "14 delegaciones · 41 centros · 11 puestos de responsabilidad",
  estructura_cabecera_clave: "Clave",
  estructura_cabecera_etiqueta: "Unidad",
  estructura_cabecera_tipo: "Tipo",
  estructura_cabecera_adscripcion: "Adscripción de referencia",
  estructura_tipo_delegacion: "Delegación",
  estructura_tipo_centro: "Centro",
  estructura_tipo_puesto_responsabilidad: "Puesto de responsabilidad",
  estructura_sin_adscripcion: "No consignada",
});

const CLAVES = Object.freeze(Object.keys(MENSAJES_PERSONAL_ES));

export function crearTraductorPersonal(catalogo = MENSAJES_PERSONAL_ES) {
  if (!catalogo || typeof catalogo !== "object"
    || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) {
    throw new Error("catálogo i18n de Personal incompleto");
  }
  return (clave, variables = {}) => {
    if (!CLAVES.includes(clave)) throw new Error(`clave i18n de Personal desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_coincidencia, variable) => String(variables[variable] ?? ""));
  };
}

export function formatearRecuentoCategorias(total, locale = "es-ES", catalogo = MENSAJES_PERSONAL_ES) {
  if (!Number.isSafeInteger(total) || total < 0 || typeof locale !== "string" || locale === "") {
    throw new TypeError("recuento de categorías no válido");
  }
  const t = crearTraductorPersonal(catalogo);
  const numero = new Intl.NumberFormat(locale).format(total);
  return t(total === 1 ? "catalogo_recuento_uno" : "catalogo_recuento_otro", { total: numero });
}

export function formatearRecuentoRPT(total, locale = "es-ES", catalogo = MENSAJES_PERSONAL_ES) {
  if (!Number.isSafeInteger(total) || total < 0 || typeof locale !== "string" || locale === "") throw new TypeError("recuento RPT no válido");
  const t = crearTraductorPersonal(catalogo);
  const numero = new Intl.NumberFormat(locale).format(total);
  return t(total === 1 ? "rpt_recuento_uno" : "rpt_recuento_otro", { total: numero });
}
export function formatearRecuentoEstructura(total, catalogo = MENSAJES_PERSONAL_ES) { if (total !== 66) throw new TypeError("recuento de estructura organizativa no válido"); return crearTraductorPersonal(catalogo)("estructura_recuento"); }

// Conserva el ISO en el contrato y lo traduce únicamente al pintar una fecha
// inequívoca para la zona operativa de la presentación.
export function formatearFechaEstructuraOrganizativa(iso) {
  if (typeof iso !== "string" || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$/u.test(iso)) throw new TypeError("fecha de estructura organizativa no válida");
  const fecha = new Date(iso);
  if (!Number.isFinite(fecha.getTime()) || fecha.toISOString() !== `${iso.slice(0, -1)}.000Z`) throw new TypeError("fecha de estructura organizativa no válida");
  return `${new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }).format(fecha)} (Europe/Madrid)`;
}
