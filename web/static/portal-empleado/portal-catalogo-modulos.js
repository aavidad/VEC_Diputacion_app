/**
 * Adaptador del registro de módulos del núcleo para el shell web.
 *
 * La lista productiva procede de `/api/vec/modules`; este archivo no enumera
 * módulos funcionales. Las pantallas disponibles se resuelven después mediante
 * adaptadores registrados, de modo que manifiesto y composición son decisiones
 * independientes y de mínimo privilegio. También lee la sesión del núcleo
 * (`/api/vec/session`) que la cabecera muestra.
 */
import { traducirPortal } from "./portal-i18n.js?v=20260925-d5d6-cronos-v1";

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
    || manifiesto.permissions.length > 512 || (manifiesto.menu !== null
      && (!Array.isArray(manifiesto.menu) || manifiesto.menu.length > 512))) {
    throw new TypeError("colecciones del manifiesto no válidas");
  }
  const permisosDeclarados = manifiesto.permissions;
  // Go serializa una porción nil como null: usuarios declara permisos, pero no menú.
  const menuDeclarado = manifiesto.menu === null ? [] : manifiesto.menu;

  const permisos = new Set();
  for (const permiso of permisosDeclarados) {
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
  for (const entrada of menuDeclarado) {
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
  // Un manifiesto defectuoso descarta sólo ese módulo: los demás se siguen
  // mostrando. Sólo si no queda ninguno válido se informa del fallo.
  const vistos = new Set();
  const catalogo = [];
  let primerError = null;
  for (const entrada of manifiestos) {
    try {
      const { clave, manifiesto } = validarManifiesto(entrada);
      if (vistos.has(clave)) throw new TypeError("módulo repetido");
      const modulo = Object.freeze({
        clave,
        sigla: siglaDe(clave),
        titulo: traducir(traducciones, manifiesto.name_key, "nombre traducido"),
        texto: traducir(traducciones, manifiesto.description_key, "descripción traducida"),
        version: manifiesto.version,
        grupo: manifiesto.group,
        rutaBase: manifiesto.base_path,
      });
      vistos.add(clave);
      catalogo.push(modulo);
    } catch (error) {
      primerError ??= error;
    }
  }
  if (catalogo.length === 0) throw primerError ?? new TypeError("catálogo de módulos no válido");
  return Object.freeze(catalogo);
}

export async function cargarCatalogoModulosInterno(fetchImpl = globalThis.fetch) {
  if (typeof fetchImpl !== "function") throw new TypeError("cliente HTTP no disponible");
  // El servidor interno exige certificado cliente también en estos GET.
  // Limitarlo al mismo origen y no seguir redirecciones; no usar cookies.
  const [respuestaModulos, respuestaTraducciones] = await Promise.all([
    fetchImpl(RUTA_MANIFIESTOS, {
      method: "GET", credentials: "same-origin", mode: "same-origin", redirect: "error",
      cache: "no-store", headers: { Accept: "application/json" },
    }),
    fetchImpl(RUTA_TRADUCCIONES, {
      method: "GET", credentials: "same-origin", mode: "same-origin", redirect: "error",
      cache: "no-store", headers: { Accept: "application/json" },
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
  // Solo se ofrecen los módulos disponibles y los que aún se comprueban: un
  // módulo sin acceso para el perfil o sin servicio no ocupa el menú.
  return catalogo.map((modulo) => [modulo, resolverAcceso(modulo.clave)])
    .filter(([, acceso]) => acceso?.disponible === true || acceso?.estado === "cargando")
    .map(([modulo, acceso]) => {
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

const RUTA_SESION = "/api/vec/session";
const PATRON_ROL = /^[a-z][a-z0-9_]{0,63}$/;
const MAXIMO_ROLES = 64;
// Tiempo máximo de la consulta de sesión: una frontera colgada no deja la
// cabecera esperando para siempre; al vencer se aborta y el shell reintenta.
export const LIMITE_CONSULTA_SESION_MS = 8000;
// Perfil visible en la cabecera según el rol atestado (clave i18n). Solo se
// muestra: no concede nada, cada consulta la autoriza el servidor.
const PERFILES_VISIBLES = Object.freeze({
  tecnico_rrhh: "perfil_sesion_rrhh",
  jefatura_rrhh: "perfil_sesion_rrhh",
  administrativo: "perfil_sesion_rrhh",
  intervencion: "perfil_sesion_intervencion",
  personal_interno: "perfil_sesion_personal",
  jefe_servicio: "perfil_sesion_jefatura",
  jefe_seccion: "perfil_sesion_jefatura",
});

/**
 * Sesión del núcleo (`GET /api/vec/session`) para la cabecera y para elegir
 * qué pantalla se ofrece. La identidad la atesta la frontera del servidor
 * (certificado cliente); aquí no se guarda en cookies ni almacenamiento web.
 * Devuelve solo el nombre visible y los roles. La consulta se aborta si vence
 * `limiteMs` o si se aborta la señal externa; en ambos casos se rechaza.
 */
export async function consultarSesionPortal({
  fetchImpl = globalThis.fetch, signal, limiteMs = LIMITE_CONSULTA_SESION_MS,
  temporizadores = globalThis,
} = {}) {
  if (typeof fetchImpl !== "function") throw new TypeError("cliente HTTP no disponible");
  if (!Number.isSafeInteger(limiteMs) || limiteMs <= 0) throw new TypeError("límite de sesión no válido");
  const controlador = new AbortController();
  const abortar = () => controlador.abort();
  if (signal?.aborted) abortar();
  else signal?.addEventListener?.("abort", abortar, { once: true });
  const temporizador = temporizadores.setTimeout(abortar, limiteMs);
  try {
    const respuesta = await fetchImpl(RUTA_SESION, {
      method: "GET", credentials: "same-origin", mode: "same-origin", redirect: "error",
      cache: "no-store", referrerPolicy: "no-referrer", headers: { Accept: "application/json" },
      signal: controlador.signal,
    });
    if (!respuesta.ok) throw new Error("no se pudo consultar la sesión");
    const principal = (await respuesta.json())?.data?.principal;
    if (!principal || typeof principal !== "object" || !Array.isArray(principal.roles)
      || principal.roles.length > MAXIMO_ROLES
      || !principal.roles.every((rol) => typeof rol === "string" && PATRON_ROL.test(rol))) {
      throw new TypeError("sesión no válida");
    }
    const nombre = typeof principal.display_name === "string" && principal.display_name.length <= 512
      ? principal.display_name.trim() : "";
    return Object.freeze({ nombre, roles: Object.freeze([...principal.roles]) });
  } finally {
    temporizadores.clearTimeout(temporizador);
    signal?.removeEventListener?.("abort", abortar);
  }
}

/** Nombre, perfil e iniciales que la cabecera muestra de una sesión. */
export function presentarSesionPortal(sesion) {
  const nombre = typeof sesion?.nombre === "string" ? sesion.nombre : "";
  const roles = Array.isArray(sesion?.roles) ? sesion.roles : [];
  const clavePerfil = roles.length === 1 && Object.hasOwn(PERFILES_VISIBLES, roles[0])
    ? PERFILES_VISIBLES[roles[0]] : "";
  const perfil = clavePerfil ? traducirPortal(clavePerfil) : "";
  const iniciales = nombre.split(/\s+/u).filter(Boolean).slice(0, 2)
    .map((parte) => parte[0].toLocaleUpperCase("es-ES")).join("");
  return Object.freeze({ nombre, perfil, iniciales: iniciales || "—" });
}
