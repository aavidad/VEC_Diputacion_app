import { isDeepStrictEqual } from "node:util";
import { readFile, readdir } from "node:fs/promises";
import { resolve, relative, join } from "node:path";
import { pathToFileURL } from "node:url";

const raiz = resolve(import.meta.dirname, "../web/static");
const cobertura = JSON.parse(await readFile(resolve(import.meta.dirname, "i18n_cobertura.json"), "utf8")).catalogos;
const compatibilidad = JSON.parse(await readFile(join(raiz, "textos/es/contratacion-temporal-compatibilidad.json"), "utf8"));
const { ES: codigoES, EN: codigoEN } = compatibilidad.idiomas_exportados;
const indice = JSON.parse(await readFile(join(raiz, "textos/idiomas.json"), "utf8"));
const codigos = new Set(indice.idiomas.map(({ codigo }) => codigo));
const errores = [];
if (!codigos.has(codigoES) || !codigos.has(codigoEN) || codigoES === codigoEN) {
  errores.push("CT: idiomas de compatibilidad ausentes o repetidos en el índice");
}
const marcador = /\{[a-z_]+\}/giu;
const DIR_CT = "portal-empleado/modulos/contratacion-temporal/";

// Cada fuente lazy se solicita expresamente en el idioma que se comprueba.
// El catálogo del idioma inactivo nunca se exige durante el arranque del módulo.
const recursosCT = new Map([
  ["i18n-analisis-catalogo.js", [["contratacion-temporal-analisis-catalogo", "general"]]],
  ["i18n-avisos-via-cobertura.js", [["contratacion-temporal-avisos-via-cobertura", "general"]]],
  ["i18n-borradores-publicados.js", [["contratacion-temporal-borradores-publicados", "general"]]],
  ["i18n-cambios-expediente.js", [["contratacion-temporal-cambios-expediente", "general"]]],
  ["i18n-circuito-firma.js", [["contratacion-temporal-circuito-firma", "general"]]],
  ["i18n-expedientes.js", [["portal", "fases_rrhh"], ["contratacion-temporal-analisis-catalogo", "general"],
    ["contratacion-temporal-ficha-lista", "general"], ["contratacion-temporal-lista-plazos", "lista"],
    ...["incorporacion", "hito", "continuidad", "firma", "borrador", "firma_pendiente"]
      .map((seccion) => ["contratacion-temporal-firma-incorporacion-expedientes", seccion])]],
  ["i18n-ficha-lista.js", [["contratacion-temporal-lista-plazos", "lista"], ["contratacion-temporal-ficha-lista", "general"]]],
  ["i18n-firma-remision.js", [["contratacion-temporal-firma-remision", "general"]]],
  ["i18n-informe-tras-subsanacion.js", [["contratacion-temporal-informe-tras-subsanacion", "general"]]],
  ["i18n-llamamiento.js", [["contratacion-temporal-llamamiento", "general"]]],
  ["i18n-subsanacion-reparos.js", [["contratacion-temporal-subsanacion-reparos", "general"]]],
  ["i18n-textos-vistas.js", [["contratacion-temporal-textos-vistas", "general"]]],
  ["i18n.js", [["portal", "fases_rrhh"], ["contratacion-temporal-moad", "general"],
    ["contratacion-temporal-llamamiento", "general"], ["contratacion-temporal-subsanacion-reparos", "general"],
    ["contratacion-temporal-avisos-via-cobertura", "general"], ["contratacion-temporal-analisis-catalogo", "general"],
    ["contratacion-temporal-textos-vistas", "general"], ["contratacion-temporal-borradores-publicados", "general"],
    ["contratacion-temporal-firma-incorporacion-portal", "general"]]],
]);

export function validarRespuesta(etiqueta, solicitado, respuesta, origen) {
  const fallos = [];
  if (!codigos.has(solicitado)) fallos.push(`${etiqueta}: locale ${solicitado} ausente del índice`);
  if (!respuesta || respuesta.idioma !== solicitado || respuesta.incidenciaCatalogo || respuesta.incidenciaIndice) {
    fallos.push(`${etiqueta}: el idioma solicitado ${solicitado} no se cargó íntegro`);
  }
  if (!respuesta?.actual || typeof respuesta.actual !== "object" || Array.isArray(respuesta.actual)) {
    fallos.push(`${etiqueta}: sección ausente o inválida`);
  } else if (!isDeepStrictEqual(respuesta.actual, origen)) {
    fallos.push(`${etiqueta}: la respuesta no coincide con el JSON del idioma solicitado`);
  }
  return fallos;
}

export function validarPar(etiqueta, es, en) {
  const fallos = [];
  if (!es || !en || typeof es !== "object" || typeof en !== "object"
    || Array.isArray(es) || Array.isArray(en)) return [`${etiqueta}: falta catálogo ES o EN`];
  const clavesES = Object.keys(es).sort();
  const clavesEN = Object.keys(en).sort();
  if (!isDeepStrictEqual(clavesES, clavesEN)) {
    fallos.push(`${etiqueta}: claves distintas ES/EN`);
    return fallos;
  }
  let diferencias = 0;
  for (const clave of clavesES) {
    const original = es[clave], traducido = en[clave];
    if (typeof original !== "string" || !original.trim() || typeof traducido !== "string" || !traducido.trim()) {
      fallos.push(`${etiqueta}: traducción vacía o inválida en ${clave}`);
    } else {
      if (original !== traducido) diferencias++;
      if (!isDeepStrictEqual([...original.matchAll(marcador)].map((m) => m[0]).sort(),
        [...traducido.matchAll(marcador)].map((m) => m[0]).sort())) {
        fallos.push(`${etiqueta}: marcadores distintos en ${clave}`);
      }
    }
  }
  if (clavesES.length && diferencias === 0) fallos.push(`${etiqueta}: EN repite íntegramente el catálogo ES`);
  return fallos;
}

async function* ficheros(dir) {
  for (const entrada of await readdir(dir, { withFileTypes: true })) {
    if (entrada.name === "vendor") continue;
    const ruta = join(dir, entrada.name);
    if (entrada.isDirectory()) yield* ficheros(ruta);
    else if (entrada.name.endsWith(".js")) yield ruta;
  }
}

const helperCT = await import(pathToFileURL(join(raiz, DIR_CT, "i18n-catalogos.js")).href);
const JSON_CT = new Map();
async function origen(idioma, modulo, seccion) {
  const clave = `${idioma}/${modulo}`;
  if (!JSON_CT.has(clave)) JSON_CT.set(clave,
    readFile(join(raiz, "textos", idioma, `${modulo}.json`), "utf8").then(JSON.parse));
  const datos = await JSON_CT.get(clave);
  if (!datos?.[seccion] || typeof datos[seccion] !== "object" || Array.isArray(datos[seccion])) {
    throw new Error(`sección ${seccion} ausente de ${clave}`);
  }
  return datos[seccion];
}

async function catalogoCT(etiqueta, idioma, modulo, seccion) {
  const [respuesta, fuente] = await Promise.all([
    helperCT.cargarCatalogosContratacionEnIdioma(modulo, idioma, seccion),
    origen(idioma, modulo, seccion),
  ]);
  errores.push(...validarRespuesta(etiqueta, idioma, respuesta, fuente));
  return respuesta.actual;
}

async function verificarCT(ruta, archivo, modulo) {
  const fuentes = recursosCT.get(archivo);
  if (!fuentes) return false;
  const catalogos = {};
  for (const idioma of [codigoES, codigoEN]) {
    catalogos[idioma] = [];
    for (const [nombre, seccion] of fuentes) {
      catalogos[idioma].push(await catalogoCT(`${ruta}:${nombre}.${seccion}`, idioma, nombre, seccion));
    }
  }
  let es, en;
  if (archivo === "i18n.js" || archivo === "i18n-expedientes.js") {
    const funcion = archivo === "i18n.js"
      ? "cargarMensajesContratacionTemporalEnIdioma" : "cargarMensajesExpedientesContratacionEnIdioma";
    if (typeof modulo[funcion] !== "function") throw new Error(`falta la API explícita ${funcion}`);
    [es, en] = await Promise.all([modulo[funcion](codigoES), modulo[funcion](codigoEN)]);
  } else if (archivo === "i18n-ficha-lista.js") {
    es = Object.assign({}, ...catalogos[codigoES]);
    en = Object.assign({}, ...catalogos[codigoEN]);
  } else {
    [es] = catalogos[codigoES];
    [en] = catalogos[codigoEN];
  }
  errores.push(...validarPar(ruta, es, en));
  // El nombre legacy debe representar solo el idioma cargado al importar.
  const exportES = [...Object.keys(modulo)].find((nombre) => nombre.startsWith("MENSAJES_") && nombre.endsWith("_ES"));
  if (exportES && modulo[exportES] && !isDeepStrictEqual(modulo[exportES], es)) {
    errores.push(`${ruta}: la exportación ${exportES} no coincide con su idioma`);
  }
  return true;
}

for await (const ruta of ficheros(raiz)) {
  const relativa = relative(raiz, ruta).split("\\").join("/");
  if (!cobertura.some((prefijo) => relativa.startsWith(prefijo))) continue;
  const fuente = await readFile(ruta, "utf8");
  const nombres = [...fuente.matchAll(/export const ([A-Z][A-Z_0-9]*_ES)\s*=/gu)].map((coincidencia) => coincidencia[1]);
  if (!nombres.length) continue;
  let modulo;
  try { modulo = await import(pathToFileURL(ruta).href); }
  catch (error) { errores.push(`${relativa}: no se puede cargar catálogo: ${error.message}`); continue; }
  try {
    if (relativa.startsWith(DIR_CT) && await verificarCT(relativa, relativa.slice(DIR_CT.length), modulo)) continue;
  } catch (error) { errores.push(`${relativa}: no se puede verificar CT: ${error.message}`); continue; }
  for (const nombre of nombres) {
    const par = nombre.slice(0, -3) + "_EN";
    errores.push(...validarPar(`${relativa}:${nombre}/${par}`, modulo[nombre], modulo[par]));
  }
}

if (errores.length) {
  for (const error of errores.slice(0, 100)) process.stderr.write(`${error}\n`);
  process.stderr.write(`i18n JS: ${errores.length} incumplimientos\n`);
  process.exitCode = 1;
} else {
  process.stdout.write("i18n JS: catálogos ES/EN simétricos\n");
}
