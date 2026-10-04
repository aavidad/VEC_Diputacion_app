import { cargarTextos } from "../../../comun/textos.js";

const catalogo = await cargarTextos("personal-exportacion-servicios");
export const traducirExportacionServicios = (clave) => catalogo.traducir(`general.${clave}`);
export const nombreArchivoExportacionServicios = catalogo.traducir("formato.nombre_archivo");
