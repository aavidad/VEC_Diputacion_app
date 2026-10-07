import { isDeepStrictEqual } from "node:util";
import { execFile } from "node:child_process";
import { readFile, readdir } from "node:fs/promises";
import { resolve, relative, join } from "node:path";
import { pathToFileURL } from "node:url";
import { promisify } from "node:util";

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
const ejecutar = promisify(execFile);

// Contrato público de cada módulo CT lazy. Una exportación inactiva existe,
// pero vale undefined; la activa contiene exactamente el idioma solicitado.
const exportacionesCT = new Map([
  ["i18n-analisis-catalogo.js", "MENSAJES_ANALISIS_CATALOGO"],
  ["i18n-avisos-via-cobertura.js", "MENSAJES_AVISOS_VIA_COBERTURA"],
  ["i18n-borradores-publicados.js", "MENSAJES_BORRADORES_PUBLICADOS"],
  ["i18n-cambios-expediente.js", "MENSAJES_CAMBIOS_EXPEDIENTE"],
  ["i18n-circuito-firma.js", "MENSAJES_CIRCUITO_FIRMA"],
  ["i18n-expedientes.js", "MENSAJES_EXPEDIENTES_CONTRATACION"],
  ["i18n-ficha-lista.js", "MENSAJES_FICHA_LISTA"],
  ["i18n-firma-remision.js", "MENSAJES_FIRMA_REMISION"],
  ["i18n-informe-tras-subsanacion.js", "MENSAJES_INFORME_TRAS_SUBSANACION"],
  ["i18n-llamamiento.js", "MENSAJES_LLAMAMIENTO"],
  ["i18n-subsanacion-reparos.js", "MENSAJES_SUBSANACION_REPAROS"],
  ["i18n-textos-vistas.js", "MENSAJES_TEXTOS_VISTAS"],
  ["i18n.js", "MENSAJES_CONTRATACION_TEMPORAL"],
]);

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
const esperadosCT = new Map();
if (!isDeepStrictEqual([...recursosCT.keys()].sort(), [...exportacionesCT.keys()].sort())) {
  errores.push("CT: contrato de fuentes y exportaciones desalineado");
}

export function validarRespuesta(etiqueta, solicitado, respuesta, origen) {
  const fallos = [];
  if (!codigos.has(solicitado)) fallos.push(`${etiqueta}: locale ${solicitado} ausente del índice`);
  if (!respuesta || respuesta.idioma !== solicitado
    || respuesta.incidenciaCatalogo !== null || respuesta.incidenciaIndice !== null) {
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

const lectorAislado = `
  const [idioma, codigoES, rutaIdioma, especificaciones] = process.argv.slice(1);
  globalThis.location = { href: 'https://example.invalid/portal-empleado/?lang=' + encodeURIComponent(idioma) };
  const idiomas = await import(rutaIdioma);
  await idiomas.prepararIdiomas();
  if (idiomas.IDIOMA_ACTUAL !== idioma) throw new Error('idioma activo distinto del solicitado');
  const salida = {};
  for (const [clave, url, prefijo] of JSON.parse(especificaciones)) {
    const modulo = await import(url);
    const nombreActivo = prefijo + (idioma === codigoES ? '_ES' : '_EN');
    const nombreInactivo = prefijo + (idioma === codigoES ? '_EN' : '_ES');
    salida[clave] = {
      tieneActivo: Object.hasOwn(modulo, nombreActivo),
      activo: modulo[nombreActivo] ?? null,
      tieneInactivo: Object.hasOwn(modulo, nombreInactivo),
      inactivo: modulo[nombreInactivo] ?? null,
    };
  }
  process.stdout.write(JSON.stringify(salida));
`;

/** Ejecuta los módulos reales tras seleccionar un solo idioma por proceso. */
export async function leerExportacionesEnProceso(idioma, especificaciones) {
  const rutaIdioma = pathToFileURL(join(raiz, "comun/idioma.js")).href;
  const { stdout } = await ejecutar(process.execPath, ["--input-type=module", "-e", lectorAislado,
    idioma, codigoES, rutaIdioma, JSON.stringify(especificaciones)], { maxBuffer: 8 * 1024 * 1024 });
  return JSON.parse(stdout);
}

export function validarExportacionCT(etiqueta, resultado, esperado) {
  const fallos = [];
  if (!resultado?.tieneActivo || !resultado.activo || typeof resultado.activo !== "object"
    || Array.isArray(resultado.activo)) {
    fallos.push(`${etiqueta}: exportación activa ausente o indefinida`);
  } else if (!isDeepStrictEqual(resultado.activo, esperado)) {
    fallos.push(`${etiqueta}: exportación activa enlazada a otro catálogo`);
  }
  if (!resultado?.tieneInactivo || resultado.inactivo !== null) {
    fallos.push(`${etiqueta}: exportación inactiva ausente o precargada`);
  }
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
  esperadosCT.set(archivo, { [codigoES]: es, [codigoEN]: en });
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

for (const archivo of exportacionesCT.keys()) {
  if (!esperadosCT.has(archivo)) errores.push(`${DIR_CT}${archivo}: módulo CT fuera del barrido de catálogos`);
}
try {
  const especificaciones = [...exportacionesCT].map(([archivo, prefijo]) => [
    archivo, pathToFileURL(join(raiz, DIR_CT, archivo)).href, prefijo,
  ]);
  const [exportES, exportEN] = await Promise.all([
    leerExportacionesEnProceso(codigoES, especificaciones),
    leerExportacionesEnProceso(codigoEN, especificaciones),
  ]);
  for (const archivo of exportacionesCT.keys()) {
    const esperado = esperadosCT.get(archivo);
    if (!esperado) continue;
    errores.push(...validarExportacionCT(`${DIR_CT}${archivo}:${codigoES}`,
      exportES[archivo], esperado[codigoES]));
    errores.push(...validarExportacionCT(`${DIR_CT}${archivo}:${codigoEN}`,
      exportEN[archivo], esperado[codigoEN]));
  }
} catch (error) {
  errores.push(`CT: no se pudieron verificar exportaciones en procesos aislados: ${error.message}`);
}

if (errores.length) {
  for (const error of errores.slice(0, 100)) process.stderr.write(`${error}\n`);
  process.stderr.write(`i18n JS: ${errores.length} incumplimientos\n`);
  process.exitCode = 1;
} else {
  process.stdout.write("i18n JS: catálogos ES/EN simétricos\n");
}
