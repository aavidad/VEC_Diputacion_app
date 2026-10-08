/**
 * Vistas del área personal que se ofrecen a la persona.
 *
 * El catálogo `vistas.json` decide qué vista aparece en el menú y se puede
 * abrir. Las vistas sin recorrido conectado no se registran en la aplicación.
 * Lo que el catálogo no nombra queda cerrado. El servidor sirve los JSON sin caché, así
 * que el cambio no exige renovar versiones.
 */
const RUTA_CATALOGO = "/area-personal/vistas.json";
const VERSION_CATALOGO = "area-personal-vistas-v1";
const MAXIMO_BYTES = 8 * 1024;
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

// Sin catálogo válido el área no arranca: nunca se abren vistas por defecto.
export async function cargarVistasDisponibles({ fetchImpl = globalThis.fetch } = {}) {
  const respuesta = await fetchImpl(RUTA_CATALOGO, {
    method: "GET", credentials: "same-origin", cache: "no-store", redirect: "error",
    referrerPolicy: "no-referrer", headers: { Accept: "application/json" },
  });
  if (respuesta?.status !== 200) throw new TypeError("vistas.json");
  const texto = await respuesta.text();
  if (new TextEncoder().encode(texto).byteLength > MAXIMO_BYTES) throw new TypeError("vistas.json");
  return leerVistasDisponibles(JSON.parse(texto));
}
