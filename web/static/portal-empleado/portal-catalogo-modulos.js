/**
 * Adaptador del registro de módulos del núcleo para el shell web.
 *
 * La lista productiva procede de `/api/vec/modules`; este archivo no enumera
 * módulos funcionales. Las pantallas disponibles se resuelven después mediante
 * adaptadores registrados, de modo que manifiesto y composición son decisiones
 * independientes y de mínimo privilegio.
 */
const RUTA_MANIFIESTOS = "/api/vec/modules";
const RUTA_TRADUCCIONES = "/locales/es.json";
const CAMPOS_MANIFIESTO = new Set([
  "id", "name_key", "description_key", "version", "group", "base_path", "permissions", "menu",
]);

function cadena(valor, nombre, patron, maximo = 256) {
  if (typeof valor !== "string" || valor.length < 2 || valor.length > maximo
    || valor !== valor.trim() || (patron && !patron.test(valor))) {
    throw new TypeError(`${nombre} no válido`);
  }
  return valor;
}

function validarManifiesto(manifiesto) {
  if (!manifiesto || typeof manifiesto !== "object" || Array.isArray(manifiesto)
    || Object.keys(manifiesto).length !== CAMPOS_MANIFIESTO.size
    || Object.keys(manifiesto).some((campo) => !CAMPOS_MANIFIESTO.has(campo))) {
    throw new TypeError("manifiesto de módulo no válido");
  }
  const id = cadena(manifiesto.id, "id de módulo", /^vec\.module\.[a-z][a-z0-9_.-]{1,79}$/);
  const clave = id.slice("vec.module.".length);
  cadena(manifiesto.name_key, "clave de nombre", /^[a-z][a-z0-9_.-]+$/);
  cadena(manifiesto.description_key, "clave de descripción", /^[a-z][a-z0-9_.-]+$/);
  cadena(manifiesto.version, "versión", /^v[0-9]+\.[0-9]+\.[0-9]+$/);
  cadena(manifiesto.group, "grupo", /^[a-z][a-z0-9_.-]+$/);
  cadena(manifiesto.base_path, "ruta base", /^\/modules\/[a-z][a-z0-9/_-]*$/);
  if (!Array.isArray(manifiesto.permissions) || manifiesto.permissions.length < 1
    || !Array.isArray(manifiesto.menu)) throw new TypeError("colecciones del manifiesto no válidas");
  return { clave, manifiesto };
}

function traducir(diccionario, clave, nombre) {
  const valor = diccionario?.[clave];
  return cadena(valor, nombre, null, 300);
}

function siglaDe(clave) {
  return clave.replace(/[^a-z0-9]/g, "").slice(0, 3).toUpperCase().padEnd(3, "·");
}

export function crearCatalogoModulosDesdeManifiestos(manifiestos, traducciones) {
  if (!Array.isArray(manifiestos) || manifiestos.length < 1 || manifiestos.length > 128
    || !traducciones || typeof traducciones !== "object" || Array.isArray(traducciones)) {
    throw new TypeError("catálogo de módulos no válido");
  }
  const vistos = new Set();
  const catalogo = manifiestos.map((entrada) => {
    const { clave, manifiesto } = validarManifiesto(entrada);
    if (vistos.has(clave)) throw new TypeError("módulo repetido");
    vistos.add(clave);
    return Object.freeze({
      clave,
      sigla: siglaDe(clave),
      titulo: traducir(traducciones, manifiesto.name_key, "nombre traducido"),
      texto: traducir(traducciones, manifiesto.description_key, "descripción traducida"),
      version: manifiesto.version,
      grupo: manifiesto.group,
      rutaBase: manifiesto.base_path,
    });
  });
  return Object.freeze(catalogo);
}

export function validarCatalogoModulosPresentacion(catalogo) {
  if (!Array.isArray(catalogo) || catalogo.length < 1 || catalogo.length > 128) {
    throw new TypeError("catálogo de presentación no válido");
  }
  const vistos = new Set();
  return Object.freeze(catalogo.map((modulo) => {
    if (!modulo || typeof modulo !== "object" || Array.isArray(modulo)) throw new TypeError("módulo de presentación no válido");
    const clave = cadena(modulo.clave, "clave de módulo", /^[a-z][a-z0-9_.-]{1,79}$/);
    if (vistos.has(clave)) throw new TypeError("módulo de presentación repetido");
    vistos.add(clave);
    return Object.freeze({ clave, sigla: cadena(modulo.sigla || siglaDe(clave), "sigla", /^[A-Z0-9·]{2,4}$/, 4),
      titulo: cadena(modulo.titulo, "título", null, 160), texto: cadena(modulo.texto, "descripción", null, 300),
      version: "presentacion", grupo: "presentacion", rutaBase: `/modules/${clave}` });
  }));
}

export async function cargarCatalogoModulosInterno(fetchImpl = globalThis.fetch) {
  if (typeof fetchImpl !== "function") throw new TypeError("cliente HTTP no disponible");
  const [respuestaModulos, respuestaTraducciones] = await Promise.all([
    fetchImpl(RUTA_MANIFIESTOS, { method: "GET", credentials: "omit", headers: { Accept: "application/json" } }),
    fetchImpl(RUTA_TRADUCCIONES, { method: "GET", credentials: "omit", headers: { Accept: "application/json" } }),
  ]);
  if (!respuestaModulos.ok || !respuestaTraducciones.ok) throw new Error("no se pudo cargar el catálogo interno de módulos");
  const [envoltura, traducciones] = await Promise.all([respuestaModulos.json(), respuestaTraducciones.json()]);
  if (!envoltura || Object.keys(envoltura).length !== 1 || !Array.isArray(envoltura.modules)) {
    throw new TypeError("respuesta de manifiestos no válida");
  }
  return crearCatalogoModulosDesdeManifiestos(envoltura.modules, traducciones);
}

export function renderizarNavegacionModulos({ catalogo, resolverAcceso, escaparHTML }) {
  if (!Array.isArray(catalogo) || typeof resolverAcceso !== "function" || typeof escaparHTML !== "function") {
    throw new TypeError("navegación de módulos no válida");
  }
  return catalogo.map((modulo) => {
    const acceso = resolverAcceso(modulo.clave);
    const habilitado = acceso?.disponible === true && typeof acceso?.vista === "string";
    return `<button type="button" class="enlace-lateral${habilitado ? " modulo-habilitado" : ""}"
      data-modulo-portal="${escaparHTML(modulo.clave)}"${habilitado ? ` data-vista="${escaparHTML(acceso.vista)}"` : " disabled"}>
      <span class="indicador-menu" aria-hidden="true">${escaparHTML(modulo.sigla.slice(0, 1))}</span>
      <span>${escaparHTML(modulo.titulo)}</span>
      <span class="etiqueta-menu${habilitado ? "" : " etiqueta-bloqueada"}">${habilitado ? "Activo" : "No habilitado"}</span>
    </button>`;
  }).join("");
}
