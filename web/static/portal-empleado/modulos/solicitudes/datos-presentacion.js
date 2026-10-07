import { cargarTextos } from "../../../comun/textos.js";

// Ejemplos para presentación; no se conectan a la consulta autorizada de la vista.
const ejemplos = (await cargarTextos("solicitudes-ejemplos")).seccion("presentacion");
export const DATOS_SOLICITUDES_PRESENTACION = Object.freeze({
  ...ejemplos,
  tramites: Object.freeze(Object.values(ejemplos.tramites)),
  catalogo: Object.freeze(Object.values(ejemplos.catalogo)),
  certificados: Object.freeze(Object.values(ejemplos.certificados)),
});
