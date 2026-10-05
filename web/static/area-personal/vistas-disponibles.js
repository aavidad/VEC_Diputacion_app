/**
 * Vistas del área personal que se ofrecen a la persona.
 *
 * El catálogo `vistas.json` decide qué vista aparece en el menú y se puede
 * abrir. Una vista sin servicio en el servidor queda desactivada (su código se
 * conserva) y se activa cambiando el catálogo cuando exista su ruta. Lo que el
 * catálogo no nombra queda cerrado.
 */
import catalogo from "./vistas.json" with { type: "json" };

const VERSION_CATALOGO = "area-personal-vistas-v1";
// Destinos a los que vuelve la navegación ante una vista desconocida: no pueden faltar.
const VISTAS_IMPRESCINDIBLES = Object.freeze(["inicio", "llamamientos"]);

export function leerVistasDisponibles(datos) {
  const vistas = datos?.vistas;
  if (!datos || typeof datos !== "object" || Array.isArray(datos)
    || datos.version !== VERSION_CATALOGO || Object.keys(datos).length !== 2
    || !vistas || typeof vistas !== "object" || Array.isArray(vistas)
    || Object.entries(vistas).some(([vista, activa]) => !/^[a-z]+$/u.test(vista) || typeof activa !== "boolean")
    || VISTAS_IMPRESCINDIBLES.some((vista) => vistas[vista] !== true)) {
    throw new TypeError("vistas.json");
  }
  return Object.freeze(new Set(Object.entries(vistas).filter(([, activa]) => activa).map(([vista]) => vista)));
}

export const VISTAS_DISPONIBLES = leerVistasDisponibles(catalogo);
