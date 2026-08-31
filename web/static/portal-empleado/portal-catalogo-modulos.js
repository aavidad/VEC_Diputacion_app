/**
 * Adaptador del registro de módulos del núcleo para el shell web.
 *
 * La lista productiva procede de `/api/vec/modules`; este archivo no enumera
 * módulos funcionales. Las pantallas disponibles se resuelven después mediante
 * adaptadores registrados, de modo que manifiesto y composición son decisiones
 * independientes y de mínimo privilegio.
 */
import { traducirPortal } from "./portal-i18n.js?v=20260721-acceso-real-v2";

const RUTA_MANIFIESTOS = "/api/vec/modules";
const RUTA_TRADUCCIONES = "/locales/es.json";
const CAMPOS_MANIFIESTO = new Set([
  "id", "name_key", "description_key", "version", "group", "base_path", "permissions", "menu",
]);
const CAMPOS_PERMISO = new Set(["key", "label_key"]);
const CAMPOS_PERMISO_OPCIONALES = new Set(["description"]);
const CAMPOS_MENU = new Set([
  "id", "module_id", "label_key", "path", "icon", "group", "order", "required_permissions",
]);

function objetoCerrado(valor, obligatorios, opcionales = new Set()) {
  if (!valor || typeof valor !== "object" || Array.isArray(valor)) return false;
  const campos = Object.keys(valor);
  return obligatorios.size <= campos.length
    && campos.length <= obligatorios.size + opcionales.size
    && [...obligatorios].every((campo) => Object.hasOwn(valor, campo))
    && campos.every((campo) => obligatorios.has(campo) || opcionales.has(campo));
}

function cadena(valor, nombre, patron, maximo = 256) {
  if (typeof valor !== "string" || valor.length < 2 || valor.length > maximo
    || valor !== valor.trim() || (patron && !patron.test(valor))) {
    throw new TypeError(`${nombre} no válido`);
  }
  return valor;
}

function validarManifiesto(manifiesto) {
  if (!objetoCerrado(manifiesto, CAMPOS_MANIFIESTO)) {
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
    || manifiesto.permissions.length > 512 || !Array.isArray(manifiesto.menu)
    || manifiesto.menu.length > 512) throw new TypeError("colecciones del manifiesto no válidas");

  const permisos = new Set();
  for (const permiso of manifiesto.permissions) {
    if (!objetoCerrado(permiso, CAMPOS_PERMISO, CAMPOS_PERMISO_OPCIONALES)) {
      throw new TypeError("permiso de módulo no válido");
    }
    const clavePermiso = cadena(permiso.key, "clave de permiso", /^[a-z][a-z0-9_.-]+$/);
    cadena(permiso.label_key, "etiqueta de permiso", /^[a-z][a-z0-9_.-]+$/);
    if (Object.hasOwn(permiso, "description")) cadena(permiso.description, "descripción de permiso", null, 512);
    if (permisos.has(clavePermiso)) throw new TypeError("permiso de módulo repetido");
    permisos.add(clavePermiso);
  }

  const entradas = new Set();
  const rutas = new Set();
  for (const entrada of manifiesto.menu) {
    if (!objetoCerrado(entrada, CAMPOS_MENU)) throw new TypeError("entrada de menú no válida");
    const entradaID = cadena(entrada.id, "id de entrada", /^[a-z][a-z0-9_.-]+$/);
    if (entrada.module_id !== id) throw new TypeError("módulo de entrada no válido");
    cadena(entrada.label_key, "etiqueta de entrada", /^[a-z][a-z0-9_.-]+$/);
    const ruta = cadena(entrada.path, "ruta de entrada", /^\/(?!\/)[a-z0-9/_-]+$/, 512);
    cadena(entrada.icon, "icono de entrada", /^[a-z][a-z0-9-]+$/);
    cadena(entrada.group, "grupo de entrada", /^[a-z][a-z0-9_.-]+$/);
    if (!Number.isSafeInteger(entrada.order)
      || !Array.isArray(entrada.required_permissions)
      || entrada.required_permissions.length < 1 || entrada.required_permissions.length > 32) {
      throw new TypeError("orden o permisos de entrada no válidos");
    }
    const requeridos = new Set();
    for (const permiso of entrada.required_permissions) {
      cadena(permiso, "permiso requerido", /^[a-z][a-z0-9_.-]+$/);
      if (!permisos.has(permiso) || requeridos.has(permiso)) {
        throw new TypeError("permiso requerido no válido");
      }
      requeridos.add(permiso);
    }
    if (entradas.has(entradaID) || rutas.has(ruta)) throw new TypeError("entrada de menú repetida");
    entradas.add(entradaID);
    rutas.add(ruta);
  }
  return { clave, manifiesto };
}

export function extraerModulosEnvelopeCanonico(envoltura) {
  if (!objetoCerrado(envoltura, new Set(["data"]))
    || !objetoCerrado(envoltura.data, new Set(["modules"]))
    || !Array.isArray(envoltura.data.modules)) {
    throw new TypeError("respuesta de manifiestos no válida");
  }
  return envoltura.data.modules;
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
    fetchImpl(RUTA_MANIFIESTOS, {
      method: "GET", credentials: "omit", cache: "no-store", headers: { Accept: "application/json" },
    }),
    fetchImpl(RUTA_TRADUCCIONES, {
      method: "GET", credentials: "omit", cache: "no-store", headers: { Accept: "application/json" },
    }),
  ]);
  if (!respuestaModulos.ok || !respuestaTraducciones.ok) throw new Error("no se pudo cargar el catálogo interno de módulos");
  const [envoltura, traducciones] = await Promise.all([respuestaModulos.json(), respuestaTraducciones.json()]);
  return crearCatalogoModulosDesdeManifiestos(extraerModulosEnvelopeCanonico(envoltura), traducciones);
}

export function renderizarNavegacionModulos({
  catalogo, resolverAcceso, escaparHTML, traducir = traducirPortal,
}) {
  if (!Array.isArray(catalogo) || typeof resolverAcceso !== "function"
    || typeof escaparHTML !== "function" || typeof traducir !== "function") {
    throw new TypeError("navegación de módulos no válida");
  }
  return catalogo.map((modulo) => {
    const acceso = resolverAcceso(modulo.clave);
    const habilitado = acceso?.disponible === true && typeof acceso?.vista === "string";
    const estado = habilitado ? traducir("estado_modulo_activo") : (acceso?.textoEstado || ({
      cargando: traducir("estado_modulo_comprobando"),
      denegado: traducir("estado_modulo_sin_permiso"),
      error: traducir("estado_modulo_no_disponible"),
      no_disponible: traducir("estado_modulo_no_disponible"),
    }[acceso?.estado] || traducir("estado_modulo_no_habilitado")));
    const comprobando = acceso?.estado === "cargando";
    return `<button type="button" class="enlace-lateral${habilitado ? " modulo-habilitado" : ""}"
      data-modulo-portal="${escaparHTML(modulo.clave)}"${habilitado ? ` data-vista="${escaparHTML(acceso.vista)}"` : ' disabled aria-disabled="true"'}${comprobando ? ' aria-busy="true"' : ""}>
      <span class="indicador-menu" aria-hidden="true">${escaparHTML(modulo.sigla.slice(0, 1))}</span>
      <span>${escaparHTML(modulo.titulo)}</span>
      <span class="etiqueta-menu${habilitado ? "" : " etiqueta-bloqueada"}">${escaparHTML(estado)}</span>
    </button>`;
  }).join("");
}
