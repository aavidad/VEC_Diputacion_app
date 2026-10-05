import { cargarTextos } from "../../../comun/textos.js";
const catalogo = await cargarTextos("personal-preparacion-rectificacion");
export const traducirPreparacionRectificacion = (clave, variables) => catalogo.traducir(clave, variables);
