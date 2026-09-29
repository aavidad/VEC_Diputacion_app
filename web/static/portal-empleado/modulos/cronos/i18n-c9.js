import { cargarTextos } from "../../../comun/textos.js";

/** Textos de estado de C9; no hay contrato de envío ni bandeja conectados. */
export const MENSAJES_CRONOS_C9 = (await cargarTextos("cronos")).seccion("c9");
