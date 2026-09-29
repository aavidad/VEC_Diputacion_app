import { cargarTextos } from "../../../comun/textos.js";

/** Textos del recorrido sin fuente de permisos. No habilita operaciones. */
export const MENSAJES_CRONOS_PERMISOS = (await cargarTextos("cronos")).seccion("permisos");
