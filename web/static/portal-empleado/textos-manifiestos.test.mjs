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

function catalogosCargados(fuente, archivo) {
  const t = tokens(fuente);
  const lectores = new Set(t.some((x) => x.literal && /(?:^|\/)textos\.js(?:\?|$)/u.test(x.valor)) ? ["cargarTextos"] : []);
  const constantes = new Map();
  for (let i = 0; i < t.length - 2; i += 1) {
    if (t[i].texto === "cargarTextos" && ["as", ":"].includes(t[i + 1].texto)) lectores.add(t[i + 2].texto);
    if (t[i + 1].texto === "=" && lectores.has(t[i + 2].texto)) lectores.add(t[i].texto);
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
      assert.ok(!(t[i].valor === "cargarTextos" && t[i - 1]?.texto === "["), `${archivo}: lector calculado sin origen comprobado`);
      continue;
    }
    if (!lectores.has(t[i].texto)) continue;
    if (t[i + 1].texto !== "(") {
      assert.ok(declaracion(i), `${archivo}: referencia indirecta a ${t[i].texto} sin origen comprobado`);
      continue;
    }
    const argumento = t[i + 2];
    if (argumento.literal) {
      incorporar(argumento.valor);
      continue;
    }
    assert.ok([",", ")"].includes(t[i + 3]?.texto), `${archivo}: carga dinámica sin nombre simple comprobado`);
    if (constantes.has(argumento.texto)) {
      incorporar(constantes.get(argumento.texto));
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
  const vistos = new Set();
  const resolver = (archivo, referencia) => referencia.startsWith("/")
    ? path.join(raiz, referencia.slice(1)) : path.resolve(path.dirname(archivo), referencia);
  while (pendientes.length) {
    const archivo = pendientes.pop();
    if (vistos.has(archivo)) continue;
    vistos.add(archivo);
    assert.ok(archivo.startsWith(raiz), `Dependencia fuera de static: ${archivo}`);
    const fuente = await readFile(archivo, "utf8");
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
        assert.equal(path.basename(archivo), "textos.js", `JSON dinámico sin origen comprobado: ${archivo}`);
        continue;
      }
      requeridos.add(path.relative(web, resolver(archivo, referencia)).split(path.sep).join("/"));
    }
    if (archivo.endsWith(".js") && archivo !== path.join(raiz, "comun/textos.js")) {
      for (const modulo of catalogosCargados(fuente, path.relative(raiz, archivo))) {
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
