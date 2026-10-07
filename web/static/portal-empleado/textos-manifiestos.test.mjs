import assert from "node:assert/strict";
import { readFile, readdir } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";

const raiz = fileURLToPath(new URL("../", import.meta.url));
const web = path.dirname(raiz.slice(0, -1));
const token = /(["'`])(?:\\[\s\S]|(?!\1)[^\\])*?\1|\/\*[\s\S]*?\*\/|\/\/[^\n]*|[A-Za-z_$][\w$]*|=>|[^\s]/gu;

function tokens(fuente) {
  return [...fuente.matchAll(token)].filter(([texto]) => !texto.startsWith("/*") && !texto.startsWith("//"))
    .map(([texto]) => ({ texto, literal: /^["'`]/u.test(texto),
      valor: /^["'`]/u.test(texto) ? texto.slice(1, -1).replace(/\\(["'`\\])/gu, "$1") : texto }));
}

function catalogosCargados(fuente, archivo, { lectorDelegado = false, delegacion } = {}) {
  const t = tokens(fuente);
  const lectores = new Set(t.some((x) => x.literal && /(?:^|\/)textos\.js(?:\?|$)/u.test(x.valor)) ? ["cargarTextos"] : []);
  if (lectorDelegado) {
    lectores.add("cargarCatalogosContratacion");
    lectores.add("cargarCatalogosContratacionEnIdioma");
  }
  const constantes = new Map();
  for (let i = 0; i < t.length - 2; i += 1) {
    if (t[i].texto === "cargarTextos" && ["as", ":"].includes(t[i + 1].texto)) lectores.add(t[i + 2].texto);
    if (t[i + 1].texto === "=" && lectores.has(t[i + 2].texto)) {
      assert.ok(!(lectorDelegado && ["cargarCatalogosContratacion", "cargarCatalogosContratacionEnIdioma"].includes(t[i + 2].texto)),
        `${archivo}: alias indirecto del lector delegado sin origen comprobado`);
      lectores.add(t[i].texto);
    }
    if (t[i - 1]?.texto === "const" && t[i + 1].texto === "=" && t[i + 2].literal) constantes.set(t[i].texto, t[i + 2].valor);
  }
  const modulos = new Set();
  const incorporar = (nombre) => {
    assert.match(nombre, /^[a-z0-9]+(?:-[a-z0-9]+)*$/u, `${archivo}: catálogo no resuelto: ${nombre}`);
    modulos.add(nombre);
  };
  const declaracion = (i) => {
    if (t[i - 1]?.texto === "=" && lectores.has(t[i - 2]?.texto)) return true;
    if (t[i + 1]?.texto === "=" && lectores.has(t[i + 2]?.texto)) return true;
    for (let j = i - 1; j >= 0 && t[j].texto !== ";"; j -= 1) {
      if (t[j].texto === "import" && t[j + 1]?.texto !== "(" && !t.slice(j, i).some((x) => x.literal)) return true;
      if (t[j].texto !== "{") continue;
      const cierre = t.findIndex((x, indice) => indice > i && x.texto === "}");
      let siguiente = cierre + 1;
      if (t[siguiente]?.texto !== "=") continue;
      siguiente += 1;
      if (t[siguiente]?.texto === "await") siguiente += 1;
      if (t[siguiente]?.texto === "import" && t[siguiente + 1]?.texto === "("
        && /\/textos\.js(?:\?|$)/u.test(t[siguiente + 2]?.valor ?? "")) return true;
    }
    return false;
  };
  for (let i = 0; i < t.length - 2; i += 1) {
    if (t[i].literal) {
      assert.ok(!(["cargarTextos", "cargarCatalogosContratacion", "cargarCatalogosContratacionEnIdioma"].includes(t[i].valor) && t[i - 1]?.texto === "["), `${archivo}: lector calculado sin origen comprobado`);
      continue;
    }
    if (!lectores.has(t[i].texto)) continue;
    if (t[i + 1].texto !== "(") {
      assert.ok(declaracion(i), `${archivo}: referencia indirecta a ${t[i].texto} sin origen comprobado`);
      continue;
    }
    const argumento = t[i + 2];
    if (["cargarCatalogosContratacion", "cargarCatalogosContratacionEnIdioma"].includes(t[i].texto)) {
      assert.ok(t[i - 1]?.texto !== ".", `${archivo}: referencia indirecta al lector delegado`);
      assert.ok(argumento.literal && [",", ")"].includes(t[i + 3]?.texto), `${archivo}: catálogo delegado sin nombre literal comprobado`);
    }
    if (argumento.literal) {
      incorporar(argumento.valor);
      continue;
    }
    assert.ok([",", ")"].includes(t[i + 3]?.texto), `${archivo}: carga dinámica sin nombre simple comprobado`);
    if (constantes.has(argumento.texto)) {
      incorporar(constantes.get(argumento.texto));
      continue;
    }
    if (delegacion?.posicion === i) {
      delegacion.modulos.forEach(incorporar);
      continue;
    }
    // El cargador común recibe sus nombres de sus consumidores. En el shell,
    // Promise.all carga un array de catálogos mediante array.map(modulo => …).
    let resuelto = false;
    for (let cierre = i - 1; cierre >= 0 && !resuelto; cierre -= 1) {
      if (t[cierre].texto !== "]" || t[cierre + 1]?.texto !== "." || t[cierre + 2]?.texto !== "map") continue;
      let profundidad = 0;
      let finMap = cierre + 3;
      for (; finMap < t.length; finMap += 1) {
        if (t[finMap].texto === "(") profundidad += 1;
        if (t[finMap].texto === ")" && --profundidad === 0) break;
      }
      if (finMap <= i || finMap === t.length) continue;
      const callback = t.slice(cierre + 3, i);
      const flecha = callback.findIndex((x) => x.texto === "=>");
      if (flecha < 0) continue;
      const parametro = callback.slice(0, flecha).filter((x) => !["(", ")", "async"].includes(x.texto));
      if (parametro.length !== 1 || parametro[0].texto !== argumento.texto) continue;
      if (callback.slice(flecha + 1).some((x) => !["(", "await"].includes(x.texto))) continue;
      let apertura = cierre - 1;
      while (apertura >= 0 && (t[apertura].literal || t[apertura].texto === ",")) apertura -= 1;
      if (t[apertura]?.texto !== "[") continue;
      const nombres = t.slice(apertura + 1, cierre).filter((x) => x.literal);
      if (!nombres.length) continue;
      nombres.forEach(({ valor }) => incorporar(valor));
      resuelto = true;
    }
    assert.ok(resuelto, `${archivo}: carga dinámica de ${argumento.texto} sin origen de catálogo comprobado`);
  }
  return modulos;
}

// El helper solo acepta nombres de catálogo literales en sus consumidores.
// La lectura usa el lector común, después de preparar el idioma de navegación.
function comprobarDelegacion(fuentes, helper, resolver) {
  const opciones = new Map();
  if (!fuentes.has(helper)) return opciones;
  const fuenteHelper = fuentes.get(helper), t = tokens(fuenteHelper);
  assert.ok(fuenteHelper.includes('from "../../../comun/textos.js"'), "Helper sin origen común comprobado");
  assert.match(fuenteHelper, /cargarTextos\(modulo, \{ idioma: elegido \}\)/u,
    "Helper sin lector común comprobado");
  assert.match(fuenteHelper, /await prepararIdiomas\(\)[\s\S]*const elegido = idioma \?\? IDIOMA_ACTUAL/u,
    "Helper elige idioma antes de preparar el índice");
  assert.match(fuenteHelper, /\n    cargarTextos\(modulo, \{ idioma: elegido \}\),/u,
    "Helper sin paso del catálogo solicitado comprobado");
  assert.ok(!/modulo\s*=\s*datos\.catalogo/u.test(fuenteHelper), "El parámetro se altera fuera del paso comprobado");
  assert.match(fuenteHelper, /cargarCatalogosContratacionEnIdioma\(modulo, undefined, seccion\)/u,
    "Helper sin idioma activo comprobado");
  assert.ok(!fuenteHelper.includes("IDIOMAS_DISPONIBLES"), "Helper vuelve a pedir todos los idiomas");
  assert.equal(t.filter((x) => x.texto === "cargarCatalogosContratacion").length, 1, "Referencia indirecta al helper");
  const modulos = new Set();
  for (const [archivo, fuente] of fuentes) {
    if (archivo === helper || archivo.endsWith(".html")) continue;
    const ts = tokens(fuente);
    const referencias = ts.filter((x) => x.literal && /(?:^|\/)i18n-catalogos\.js(?:\?|$)/u.test(x.valor));
    const usaHelper = ts.some((x) => x.texto === "cargarCatalogosContratacion");
    if (!referencias.length && !usaHelper) continue;
    assert.equal(referencias.length, 1, `${archivo}: helper sin origen único comprobado`);
    const referencia = referencias[0], indice = ts.indexOf(referencia);
    assert.equal(resolver(archivo, referencia.valor.split("?")[0]), helper, `${archivo}: helper de otro origen`);
    const prefijo = ts.slice(indice - 5, indice).map((x) => x.texto);
    const importacionSimple = ["import", "{", "cargarCatalogosContratacion", "}", "from"];
    const importacionExplicita = ["import", "{", "cargarCatalogosContratacionEnIdioma", "}", "from"];
    const importacionDoble = ["cargarCatalogosContratacion", ",", "cargarCatalogosContratacionEnIdioma", "}", "from"];
    assert.ok(JSON.stringify(prefijo) === JSON.stringify(importacionSimple)
      || JSON.stringify(prefijo) === JSON.stringify(importacionExplicita)
      || JSON.stringify(prefijo) === JSON.stringify(importacionDoble), `${archivo}: importación indirecta del helper`);
    const configuracion = { lectorDelegado: true };
    for (const modulo of catalogosCargados(fuente, archivo, configuracion)) modulos.add(modulo);
    opciones.set(archivo, configuracion);
  }
  assert.ok(modulos.size > 0, "Helper sin consumidores de catálogo comprobados");
  assert.ok(fuenteHelper.includes('cargarTextos("contratacion-temporal-compatibilidad"'),
    "Helper sin catálogo de compatibilidad comprobado");
  const posicion = t.findIndex((x, i) => x.texto === "cargarTextos" && t[i + 1]?.texto === "(" && t[i + 2]?.texto === "modulo");
  assert.ok(posicion >= 0, "Helper sin llamada comprobada al lector común");
  opciones.set(helper, { delegacion: { posicion, modulos } });
  return opciones;
}

async function htmlDelPortal(directorio, salida = []) {
  for (const entrada of await readdir(directorio, { withFileTypes: true })) {
    const nombre = path.join(directorio, entrada.name);
    if (entrada.isDirectory()) await htmlDelPortal(nombre, salida);
    else if (entrada.name.endsWith(".html")) salida.push(nombre);
  }
  return salida;
}

async function recursosDeTexto() {
  const indice = JSON.parse(await readFile(path.join(raiz, "textos/idiomas.json"), "utf8"));
  const idiomas = indice.idiomas.map(({ codigo }) => codigo);
  const requeridos = new Set();
  const pendientes = await htmlDelPortal(path.join(raiz, "portal-empleado"));
  const vistos = new Set(), fuentes = new Map();
  const resolver = (archivo, referencia) => referencia.startsWith("/")
    ? path.join(raiz, referencia.slice(1)) : path.resolve(path.dirname(archivo), referencia);
  while (pendientes.length) {
    const archivo = pendientes.pop();
    if (vistos.has(archivo)) continue;
    vistos.add(archivo);
    assert.ok(archivo.startsWith(raiz), `Dependencia fuera de static: ${archivo}`);
    const fuente = await readFile(archivo, "utf8");
    fuentes.set(archivo, fuente);
    const cadenas = archivo.endsWith(".html")
      ? [...fuente.matchAll(/(?:src|href)=["']([^"']+)["']/gu)].map(([, valor]) => ({ valor }))
      : tokens(fuente).filter(({ literal }) => literal);
    for (const { valor } of cadenas) {
      const referencia = valor.split("?")[0];
      if (/^(?:\.{1,2}\/|\/).*\.js$/u.test(referencia)) pendientes.push(resolver(archivo, referencia));
      if (!referencia.includes("textos/") || !referencia.endsWith(".json")) continue;
      // La plantilla idioma/modulo de textos.js se resuelve con las llamadas
      // reales anteriores, no con una lista de módulos escrita en la prueba.
      if (referencia.includes("${")) {
        if (path.basename(archivo) === "portal-arranque-aviso.js"
          && referencia === "/textos/${candidato}/portal-arranque.json") {
          // El aviso de arranque debe funcionar aunque no se importe textos.js.
          // Sólo se admite su catálogo mínimo, que también lleva cada idioma.
          for (const idioma of idiomas) requeridos.add(`static/textos/${idioma}/portal-arranque.json`);
          continue;
        }
        assert.equal(path.basename(archivo), "textos.js", `JSON dinámico sin origen comprobado: ${archivo}`);
        continue;
      }
      requeridos.add(path.relative(web, resolver(archivo, referencia)).split(path.sep).join("/"));
    }
  }
  const opciones = comprobarDelegacion(fuentes, path.join(raiz, "portal-empleado/modulos/contratacion-temporal/i18n-catalogos.js"), resolver);
  for (const idioma of idiomas) requeridos.add(`static/textos/${idioma}/contratacion-temporal-compatibilidad.json`);
  for (const [archivo, fuente] of fuentes) {
    if (archivo.endsWith(".js") && archivo !== path.join(raiz, "comun/textos.js")) {
      for (const modulo of catalogosCargados(fuente, path.relative(raiz, archivo), opciones.get(archivo))) {
        for (const idioma of idiomas) requeridos.add(`static/textos/${idioma}/${modulo}.json`);
      }
    }
  }
  assert.ok(vistos.has(path.join(raiz, "portal-empleado/portal.js")), "Se recorrió la entrada real del portal");
  return [...requeridos].sort();
}

function comprobar(requeridos, manifiestos) {
  for (const [nombre, contenido] of Object.entries(manifiestos)) {
    const rutas = new Set(contenido.trim().split(/\r?\n/u));
    const faltantes = requeridos.filter((ruta) => !rutas.has(ruta));
    assert.deepEqual(faltantes, [], `${nombre}: faltan recursos de texto cargados por el portal`);
  }
}

const requeridos = await recursosDeTexto();
const manifiestos = Object.fromEntries(await Promise.all(["interno.manifest", "produccion.manifest"]
  .map(async (nombre) => [nombre, await readFile(path.join(web, nombre), "utf8")])));

test("los textos cargados por el portal están en ambos manifiestos", (t) => {
  comprobar(requeridos, manifiestos);
  t.diagnostic(`${requeridos.length} recursos derivados del portal; ${requeridos.length * 2} omisiones independientes comprobadas`);
});

test("cada omisión de un JSON se rechaza en cada manifiesto por separado", () => {
  for (const [nombre, contenido] of Object.entries(manifiestos)) {
    for (const recurso of requeridos) {
      const modificado = contenido.split(/\r?\n/u).filter((ruta) => ruta !== recurso).join("\n");
      assert.throws(() => comprobar(requeridos, { ...manifiestos, [nombre]: modificado }),
        (error) => error.message.includes(nombre) && error.actual.includes(recurso));
    }
  }
});

test("se auditan literales, alias y el cargador dinámico de array.map", () => {
  const analizar = (fuente) => catalogosCargados('import { cargarTextos } from "./textos.js";\n' + fuente, "consumidor.js");
  assert.deepEqual([...analizar('import { cargarTextos as leer } from "./textos.js"; leer("nuevo-catalogo");')], ["nuevo-catalogo"]);
  assert.deepEqual([...analizar('const leer = cargarTextos; leer("nuevo-catalogo");')], ["nuevo-catalogo"]);
  assert.deepEqual([...analizar('const MODULO = "catalogo-firma"; function cargarFirma(cargar = cargarTextos) { return cargar(MODULO); }')], ["catalogo-firma"]);
  assert.deepEqual([...analizar('["uno", "dos"].map((modulo) => cargarTextos(modulo));')], ["uno", "dos"]);
  assert.throws(() => analizar("cargarTextos(datos.catalogo);"), /carga dinámica/u);
  assert.throws(() => analizar('["uno"].map((modulo) => cargarTextos(modulo.catalogo));'), /carga dinámica/u);
  assert.throws(() => analizar('["uno"].map((x) => x); function otra(modulo) { cargarTextos(modulo); }'), /carga dinámica/u);
  assert.throws(() => analizar('["uno"].map((x) => { const modulo = datos.catalogo; cargarTextos(modulo); });'), /carga dinámica/u);
  assert.throws(() => analizar('["uno"].map((modulo) => { modulo = datos.catalogo; cargarTextos(modulo); });'), /carga dinámica/u);
  assert.throws(() => analizar('const helper = { leer: cargarTextos }; helper.leer("otro");'), /referencia indirecta/u);
  assert.throws(() => analizar('textos["cargarTextos"]("otro");'), /lector calculado/u);
  assert.deepEqual([...analizar('// cargarTextos("comentario")\nconst texto = "cargarTextos(otro)";')], []);
});

test("el helper sólo delega catálogos con consumidores y procedencia comprobados", async () => {
  const helper = path.join(raiz, "portal-empleado/modulos/contratacion-temporal/i18n-catalogos.js");
  const consumidor = path.join(path.dirname(helper), "consumidor.js");
  const fuenteHelper = await readFile(helper, "utf8");
  const importacion = 'import { cargarCatalogosContratacion } from "./i18n-catalogos.js?v=20261001-ct-a-i18n-v1";\n';
  const resolver = (archivo, referencia) => path.resolve(path.dirname(archivo), referencia);
  const fuentes = (fuente, cargador = fuenteHelper) => new Map([[helper, cargador], [consumidor, fuente]]);
  const validar = (fuente, cargador) => comprobarDelegacion(fuentes(fuente, cargador), helper, resolver);
  const correcto = importacion + 'cargarCatalogosContratacion("catalogo-prueba");';
  const opciones = validar(correcto);
  assert.ok(opciones.has(consumidor));
  for (const llamada of [
    "cargarCatalogosContratacion(datos.catalogo);",
    "cargarCatalogosContratacion(modulo);",
    'cargarCatalogosContratacion("catalogo-prueba" + datos.sufijo);',
    "const otro = { leer: cargarCatalogosContratacion }; otro.leer(datos.catalogo);",
    'otro.cargarCatalogosContratacion("catalogo-prueba");',
    'const leer = cargarCatalogosContratacion; leer("catalogo-prueba" + datos.sufijo);',
    'const leer = cargarCatalogosContratacion; leer("catalogo-prueba");',
  ]) assert.throws(() => validar(importacion + llamada), /delegado|indirecta/u);
  assert.throws(() => validar(importacion + 'catalogos["cargarCatalogosContratacion"]("catalogo-prueba");'), /lector calculado/u);
  assert.throws(() => validar('cargarCatalogosContratacion("catalogo-prueba");'), /origen/u);
  assert.throws(() => validar(correcto.replace("./i18n-catalogos.js?v=20261001-ct-a-i18n-v1", "../otro/i18n-catalogos.js")), /otro origen/u);
  assert.throws(() => validar(correcto.replace("{ cargarCatalogosContratacion }", "* as catalogos")), /importación indirecta/u);
  assert.throws(() => validar(correcto, fuenteHelper.replace("    cargarTextos(modulo, { idioma: elegido }),", "    cargarTextos(datos.catalogo, { idioma: elegido }),")), /paso|lector común/u);
  assert.throws(() => validar(correcto, fuenteHelper + "\nmodulo = datos.catalogo;"), /fuera del paso/u);
  assert.throws(() => validar(correcto, fuenteHelper.replace("../../../comun/textos.js", "./otro/textos.js")), /origen común/u);
  assert.throws(() => comprobarDelegacion(new Map([[helper, fuenteHelper]]), helper, resolver), /sin consumidores/u);
});
