import { cargarTextos } from "../comun/textos.js";

/** Textos del contacto de origen CONVOCA (duda 45) en la ficha y en el llamamiento: `textos/<idioma>/portal-bolsa.json`. */
export const MENSAJES_CONTACTO_ORIGEN = (await cargarTextos("portal-bolsa")).seccion("contacto_origen");
