import { cargarTextos } from "../../../comun/textos.js";

/** Textos de estado del catálogo C6; los tipos proceden de una consulta autorizada. */
export const MENSAJES_CRONOS_C6 = (await cargarTextos("cronos")).seccion("c6");
