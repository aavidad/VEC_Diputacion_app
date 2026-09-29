import { cargarTextos } from "../../../comun/textos.js";

/** Textos de la solicitud de rectificación de Dietas. */
export const MENSAJES_RECTIFICACION_DIETAS = (await cargarTextos("dietas")).seccion("rectificacion_dietas");
