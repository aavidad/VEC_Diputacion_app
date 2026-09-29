import { cargarTextos } from "../../../comun/textos.js";

/** Textos de la rectificación administrativa de asignación de Dietas. */
export const MENSAJES_RECTIFICACION_ADMIN = (await cargarTextos("dietas")).seccion("rectificacion_admin");
