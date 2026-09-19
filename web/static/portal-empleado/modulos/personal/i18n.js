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
