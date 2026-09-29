import { cargarTextos } from "../../../comun/textos.js";

/** Complemento del catálogo común de Dietas para el circuito de revisión y autorización. */
export const MENSAJES_CIRCUITO_DIETAS = (await cargarTextos("dietas")).seccion("circuito");
