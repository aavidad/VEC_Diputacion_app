#!/usr/bin/env node
// Recorrido sintético contra Go y Chrome del sistema; no intercepta respuestas.
import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { createHash } from "node:crypto";
import { access, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { createRequire } from "node:module";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const raiz = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const argumentos = process.argv.slice(2);
if (argumentos.includes("--help")) {
  console.log("node scripts/probar_baremador_navegador.mjs [--solo-bolsa | --explicaciones]\nCHROME_BIN: Chrome del sistema. PLAYWRIGHT_MODULE: módulo ya instalado. GO_BIN: toolchain local.\nEl recorrido completo exige Bolsa y Concursos integrados. --explicaciones comprueba únicamente motivos y corte en es/en a 1440/390. Los artefactos quedan en un directorio temporal único.");
  process.exit(0);
}
assert(argumentos.every((a) => ["--solo-bolsa", "--explicaciones"].includes(a)) && argumentos.length <= 1, "argumento no admitido");
const soloBolsa = argumentos.includes("--solo-bolsa");
const soloExplicaciones = argumentos.includes("--explicaciones");
const informe = { esquema: "vec.recorrido_baremador_local.v1", inicio: new Date().toISOString(),
  alcance: "simulacion_sintetica_local", modulos: soloBolsa ? ["bolsa"] : ["bolsa", "concursos"],
  modo: soloExplicaciones ? "explicaciones" : "matriz",
  transporte: "HTTP real, sin sustitución de respuestas", casos: [], estado: "en_curso" };
let temporal, artefactos, browser, servidor;
const procesos = new Set();
const cancelacion = new AbortController();
const matar = (p, senal = "SIGTERM") => {
  if (p?.exitCode === null && p.signalCode === null) {
    try { process.kill(-p.pid, senal); } catch (error) { if (error.code !== "ESRCH") throw error; }
  }
};
function cancelar() {
  if (cancelacion.signal.aborted) return;
  cancelacion.abort();
  for (const proceso of procesos) matar(proceso);
  void browser?.close();
}
process.on("SIGINT", cancelar);
process.on("SIGTERM", cancelar);
const limite = setTimeout(cancelar, 8 * 60 * 1000);

async function ejecutar(programa, args, opciones = {}) {
  cancelacion.signal.throwIfAborted();
  return new Promise((aceptar, rechazar) => {
    const p = spawn(programa, args, { cwd: raiz, detached: true, stdio: ["ignore", "pipe", "pipe"], ...opciones });
    procesos.add(p);
    let salida = "", diagnostico = "";
    const tiempo = setTimeout(() => matar(p, "SIGKILL"), 180000);
    p.stdout.on("data", (b) => { salida += b; if (salida.length > 1048576) matar(p, "SIGKILL"); });
    p.stderr.on("data", (b) => { diagnostico += b; if (diagnostico.length > 1048576) matar(p, "SIGKILL"); });
    p.once("error", (error) => { clearTimeout(tiempo); procesos.delete(p); rechazar(error); });
    p.once("close", (codigo) => {
      clearTimeout(tiempo); procesos.delete(p);
      if (codigo === 0) aceptar(salida.trim());
      else rechazar(new Error(`${programa} terminó con ${codigo}: ${diagnostico.slice(0, 2000)}`));
    });
  });
}

async function arrancar(binario) {
  return new Promise((aceptar, rechazar) => {
    servidor = spawn(binario, ["--puerto", "0", "--web-dir", join(raiz, "web/static")],
      { cwd: raiz, detached: true, stdio: ["ignore", "pipe", "pipe"] });
    procesos.add(servidor);
    let salida = "", diagnostico = "", resuelto = false;
    const tiempo = setTimeout(() => { matar(servidor); rechazar(new Error("Go no anunció su dirección en 15 segundos")); }, 15000);
    servidor.stderr.on("data", (b) => { diagnostico = (diagnostico + b).slice(-2000); });
    servidor.stdout.on("data", (b) => {
      salida += b;
      const coincidencia = salida.match(/http:\/\/127\.0\.0\.1:\d+\/[^\s]+/u);
      if (!resuelto && coincidencia) {
        resuelto = true; clearTimeout(tiempo); aceptar(new URL(coincidencia[0]));
      }
    });
    servidor.once("error", (error) => { clearTimeout(tiempo); rechazar(error); });
    servidor.once("close", (codigo) => {
      clearTimeout(tiempo); procesos.delete(servidor);
      if (!resuelto) rechazar(new Error(`Go no arrancó (${codigo}): ${diagnostico}`));
    });
  });
}

function instalarVigilancia() {
  globalThis.__vecAlmacenamiento = [];
  const registrar = (nombre) => globalThis.__vecAlmacenamiento.push(nombre);
  for (const nombre of ["setItem", "removeItem", "clear"]) {
    const original = Storage.prototype[nombre];
    Storage.prototype[nombre] = function (...args) { registrar(`Storage.${nombre}`); return original.apply(this, args); };
  }
  const cookie = Object.getOwnPropertyDescriptor(Document.prototype, "cookie");
  if (cookie?.set) Object.defineProperty(Document.prototype, "cookie", {
    ...cookie, set(valor) { registrar("cookie"); return cookie.set.call(this, valor); },
  });
  for (const [objeto, nombre, etiqueta] of [[globalThis.indexedDB, "open", "IndexedDB.open"],
    [globalThis.indexedDB, "deleteDatabase", "IndexedDB.deleteDatabase"], [globalThis.caches, "open", "CacheStorage.open"],
    [globalThis.navigator.serviceWorker, "register", "ServiceWorker.register"]]) {
    if (objeto?.[nombre]) {
      const original = objeto[nombre].bind(objeto);
      objeto[nombre] = (...args) => { registrar(etiqueta); return original(...args); };
    }
  }
}

function observar(page, origen, caso) {
  const pendientes = new Set(), mensajes = [], solicitudes = [], respuestas = [];
  page.on("pageerror", (error) => mensajes.push({ tipo: "pageerror", mensaje: error.message }));
  page.on("console", (m) => { if (m.type() === "error") mensajes.push({ tipo: "console", mensaje: m.text() }); });
  page.on("request", (r) => {
    if (new URL(r.url()).origin !== origen) solicitudes.push({ fuera: true, url: r.url() });
    else if (r.headers().cookie || r.headers().authorization) solicitudes.push({ credenciales: true });
  });
  page.on("response", (r) => {
    const tarea = (async () => {
      const ruta = new URL(r.url()).pathname;
      const respuesta = { ruta, metodo: r.request().method(), estado: r.status() };
      if (r.headers()["set-cookie"]) respuesta.cookie = true;
      if (respuesta.metodo === "POST") {
        try { respuesta.solicitud = JSON.parse(r.request().postData()); } catch { respuesta.solicitud = null; }
        // El cliente puede abandonar el cuerpo de un rechazo HTTP. Leerlo por
        // CDP puede quedar pendiente; los errores JSON se consumen por fetch.
        if (r.ok()) { try { respuesta.datos = await r.json(); } catch { respuesta.jsonInvalido = true; } }
      }
      respuestas.push(respuesta);
    })();
    pendientes.add(tarea); tarea.finally(() => pendientes.delete(tarea));
  });
  return { respuestas, async vaciar() { while (pendientes.size) await Promise.all([...pendientes]); },
    async comprobar(context) {
      await this.vaciar();
      assert.deepEqual(solicitudes, [], "llamadas externas o credenciales");
      assert(!respuestas.some((r) => r.cookie), "Set-Cookie recibido");
      assert.deepEqual(await context.cookies(), [], "cookies conservadas");
      const almacenamiento = await page.evaluate(async () => ({ intentos: globalThis.__vecAlmacenamiento,
        local: localStorage.length, sesion: sessionStorage.length, bases: await indexedDB.databases(), caches: await caches.keys() }));
      assert.deepEqual(almacenamiento, { intentos: [], local: 0, sesion: 0, bases: [], caches: [] }, "almacenamiento web usado");
      // Chrome escribe un diagnóstico de red ante los 400/422 provocados deliberadamente.
      const esperados = respuestas.filter((r) => r.metodo === "POST" && [400, 422].includes(r.estado));
      const redEsperada = mensajes.filter((m) => m.tipo === "console" && /^Failed to load resource:.*(?:400|422)/u.test(m.mensaje));
      assert(redEsperada.length <= esperados.length, "diagnósticos de red no explicados");
      assert.deepEqual(mensajes.filter((m) => !redEsperada.includes(m)), [], "errores JS o consola inesperados");
      caso.vigilancia = { cookies: 0, almacenamiento, erroresJS: 0, erroresConsolaInesperados: 0,
        diagnosticosRedProvocados: redEsperada.length, llamadasExternas: 0 };
      caso.http = respuestas.map(({ ruta, metodo, estado, datos }) => ({ ruta, metodo, estado,
        ...(datos?.resultado ? { estadoCalculo: datos.resultado.estado, total: datos.resultado.total,
          huella: datos.huella_resultado_sha256 } : {}) }));
    } };
}

async function geometria(page, nombre) {
  const medidas = await page.evaluate(() => ({ ancho: innerWidth, documento: document.documentElement.scrollWidth,
    cuerpo: document.body.scrollWidth, alto: innerHeight, altoDocumento: document.documentElement.scrollHeight,
    altoCuerpo: document.body.scrollHeight, acciones: [...document.querySelectorAll("form footer button:not([disabled])")]
      .filter((e) => e.getClientRects().length).map((e) => { const r = e.getBoundingClientRect();
        return { accion: e.dataset.accion ?? e.dataset.concursoAccion ?? e.type, visible: r.top >= 0 && r.bottom <= innerHeight && r.left >= 0 && r.right <= innerWidth }; }),
    tablas: [...document.querySelectorAll(".tabla-contenedor")]
      .filter((e) => e.getClientRects().length).map((e) => ({ ancho: e.clientWidth, contenido: e.scrollWidth,
        overflow: getComputedStyle(e).overflowX })) }));
  assert(medidas.documento <= medidas.ancho + 1 && medidas.cuerpo <= medidas.ancho + 1,
    `${nombre}: desbordamiento global ${JSON.stringify(medidas)}`);
  if (medidas.ancho >= 1024) {
    assert(medidas.altoDocumento <= medidas.alto + 1 && medidas.altoCuerpo <= medidas.alto + 1,
      `${nombre}: desplazamiento vertical global en escritorio ${JSON.stringify(medidas)}`);
    assert(medidas.acciones.every((a) => a.visible), `${nombre}: acción frecuente fuera de escritorio ${JSON.stringify(medidas.acciones)}`);
  }
  assert(medidas.tablas.every((t) => t.contenido <= t.ancho + 1 || ["auto", "scroll"].includes(t.overflow)), `${nombre}: tabla sin scroll interno`);
  return medidas;
}

async function fotografiar(page, nombre, destino) {
  await page.evaluate(async () => {
    window.scrollTo({ top: 0, left: 0, behavior: "instant" });
    for (const e of document.querySelectorAll("main,.tabla-contenedor,.baremo-resultados,.baremo-topes .cuerpo-panel")) {
      e.scrollTo({ top: 0, left: 0, behavior: "instant" });
    }
    await new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r)));
  });
  if (destino) await destino.scrollIntoViewIfNeeded();
  await page.screenshot({ path: join(artefactos, nombre), fullPage: false });
}

function presentarPuntos(valor, idioma) {
  const formato = new Intl.NumberFormat(idioma);
  const decimal = formato.formatToParts(1.5).find((p) => p.type === "decimal").value;
  const fraccion = String(BigInt(valor) % 1000000n).padStart(6, "0").replace(/0+$/u, "");
  return formato.format(BigInt(valor) / 1000000n) + (fraccion ? decimal + fraccion : "");
}

async function teclado(page) {
  await page.evaluate(() => { document.body.tabIndex = -1; document.body.focus(); document.body.removeAttribute("tabindex"); });
  const fechas = new Map();
  for (const fecha of await page.locator('input[type="date"]:visible').all()) {
    fechas.set(await fecha.getAttribute("id"), createHash("sha256").update(await fecha.screenshot({ caret: "hide" })).digest("hex"));
  }
  const vistos = new Set();
  const maximo = await page.locator('button:visible:not([disabled]),input:visible:not([disabled]),select:visible:not([disabled]),summary:visible,[tabindex="0"]:visible').count() * 2 + 8;
  for (let i = 0; i < maximo; i++) {
    await page.keyboard.press("Tab");
    const foco = await page.evaluate(() => {
      const e = document.activeElement, r = e.getBoundingClientRect(), s = getComputedStyle(e);
      const label = e.closest("label"), sLabel = label && getComputedStyle(label);
      const x = Math.min(innerWidth - 1, Math.max(0, r.x + r.width / 2));
      const y = Math.min(innerHeight - 1, Math.max(0, r.y + r.height / 2));
      const encima = document.elementFromPoint(x, y);
      return { tag: e.tagName, id: e.id || e.name || e.dataset.panel || e.dataset.accion || e.dataset.concursoAccion || e.type,
        visible: r.width > 0 && r.height > 0 && r.right > 0 && r.left < innerWidth && r.bottom > 0 && r.top < innerHeight,
        tapado: encima !== e && !e.contains(encima) && !encima?.contains(e),
        focusVisible: e.matches(":focus-visible"), focusWithin: e.matches(":focus-within"),
        outline: s.outline, tipo: e.type,
        indicador: parseFloat(s.outlineWidth) > 0 && s.outlineStyle !== "none" || s.boxShadow !== "none"
          || sLabel && parseFloat(sLabel.outlineWidth) > 0 && sLabel.outlineStyle !== "none" };
    });
    if (foco.tag === "BODY") { if (vistos.has("submit")) break; else continue; }
    assert(foco.visible && !foco.tapado, `foco invisible o tapado: ${JSON.stringify(foco)}`);
    if (!foco.indicador && foco.tipo === "date" && foco.focusWithin) {
      const imagen = await page.locator(`#${foco.id}`).screenshot({ caret: "hide" });
      foco.indicador = createHash("sha256").update(imagen).digest("hex") !== fechas.get(foco.id);
      foco.indicadorNativo = foco.indicador;
    }
    assert(foco.indicador, `foco sin indicador: ${JSON.stringify(foco)}`);
    vistos.add(foco.id);
  }
  assert(vistos.size >= 6, `recorrido de teclado demasiado corto: ${[...vistos]}`);
  assert(vistos.has("submit"), "teclado no alcanzó la comparación");
  return [...vistos];
}

async function exportar(page, boton) {
  const espera = page.waitForEvent("download");
  await boton.click();
  const descarga = await espera;
  const archivo = join(temporal, "borrador.json");
  await descarga.saveAs(archivo);
  const contenido = await readFile(archivo, "utf8");
  await rm(archivo);
  return JSON.parse(contenido);
}
async function importar(page, selector, contenido) {
  const control = await page.locator(selector).elementHandle();
  await page.locator(selector).setInputFiles({ name: "reglas-sinteticas.json", mimeType: "application/json",
    buffer: Buffer.from(typeof contenido === "string" ? contenido : JSON.stringify(contenido)) });
  // Tanto el éxito como el rechazo pintan otra vista. Esperar esa sustitución
  // acredita el consumo de ESTE archivo, incluso si ya había un error visible.
  try { await page.waitForFunction((anterior) => !anterior.isConnected, control); }
  finally { await control.dispose(); }
  await page.waitForFunction((s) => !document.querySelector(s)?.disabled, selector);
}
async function comparar(page, monitor, formulario, resultado, ruta, antes, despues) {
  const inicio = monitor.respuestas.length;
  await page.locator(`${formulario} [type=submit]`).click();
  await page.locator(resultado).nth(1).waitFor();
  await monitor.vaciar();
  const recibos = monitor.respuestas.slice(inicio).filter((r) => r.ruta === ruta && r.metodo === "POST");
  assert.equal(recibos.length, 2, "comparación debe consultar original y borrador");
  for (const [i, esperado] of [antes, despues].entries()) {
    const r = recibos[i];
    assert.equal(r.estado, 200); assert.equal(r.datos.alcance, "simulacion");
    assert.equal(r.datos.resultado.estado, ruta === "/simular" ? "completado" : "simulacion_local_sin_efectos");
    assert.equal(r.datos.resultado.total, esperado, "total Go distinto al esperado");
    if (ruta === "/simular") {
      assert.match(r.datos.huella_resultado_sha256, /^[a-f0-9]{64}$/u);
      assert.equal(createHash("sha256").update(JSON.stringify(r.datos.resultado)).digest("hex"), r.datos.huella_resultado_sha256);
    } else {
      assert.equal(r.datos.schema_version, "provision.simulacion.v1");
      assert.equal(r.datos.resultado.version_motor, "provision.v1");
      assert.equal(r.datos.resultado.completo, true);
      for (const clave of ["huella_reglas", "huella_entrada", "huella_resultado"]) assert.match(r.datos.resultado[clave], /^[a-f0-9]{64}$/u);
      assert.equal(createHash("sha256").update(JSON.stringify({ ...r.datos.resultado, huella_resultado: "" })).digest("hex"),
        r.datos.resultado.huella_resultado, "huella propia de Provisión no reproducida");
    }
    const idioma = await page.locator("html").getAttribute("lang");
    const texto = presentarPuntos(esperado, idioma);
    const total = ruta === "/simular" ? page.locator(resultado).nth(i).locator("dd") : page.locator(resultado).nth(i);
    assert.equal((await total.textContent()).trim(), texto, "total visible distinto a Go o idioma incorrecto");
  }
  return recibos;
}

async function bolsa(page, monitor, caso) {
  const campo = page.locator("#baremo-campo-reglas_experiencia-0-puntos_por_unidad");
  await campo.waitFor();
  const primera = await comparar(page, monitor, "#baremo-formulario", ".baremo-resultado", "/simular", "101667", "101667");
  await campo.fill("0.2");
  assert(await campo.evaluate((e) => e === document.activeElement), "editar perdió el foco");
  const cambiada = await comparar(page, monitor, "#baremo-formulario", ".baremo-resultado", "/simular", "101667", "203333");
  assert.notEqual(primera[1].datos.huella_resultado_sha256, cambiada[1].datos.huella_resultado_sha256);
  const botonExportar = page.locator('[data-accion="exportar"]');
  const borrador = await exportar(page, botonExportar);
  assert.equal(borrador.reglas_experiencia[0].puntos_por_unidad, "200000");
  await importar(page, '[name="archivo"]', "{json-roto");
  await page.locator("#baremo-error:not([hidden])").waitFor();
  assert.deepEqual(await exportar(page, botonExportar), borrador, "JSON roto alteró borrador");
  await importar(page, '[name="archivo"]', "x".repeat(2 * 1024 * 1024 + 1));
  await page.locator("#baremo-error:not([hidden])").waitFor();
  assert.deepEqual(await exportar(page, botonExportar), borrador, "archivo excesivo alteró borrador");
  await campo.fill("-1"); await page.keyboard.press("Tab");
  assert.equal(await campo.getAttribute("aria-invalid"), "true");
  assert(await botonExportar.isDisabled(), "campo inválido permitió exportar");
  await page.locator("#baremo-formulario [type=submit]").click();
  assert(await campo.evaluate((e) => e === document.activeElement), "validación no enfoca primer error");
  await campo.fill("0.2");
  await page.locator('[data-panel="concursos"]').click();
  assert(await page.locator('[data-panel="concursos"]').evaluate((e) => e === document.activeElement), "cambio panel perdió foco");
  await page.locator('[data-panel="bolsa"]').click();
  assert.deepEqual(await exportar(page, botonExportar), borrador, "cambio panel perdió borrador");
  // La importación pasa la forma del editor, pero Go rechaza el campo desconocido.
  const invalido = { ...borrador, campo_sintetico_no_admitido: true };
  await importar(page, '[name="archivo"]', invalido);
  await page.locator("#baremo-formulario [type=submit]").click();
  await page.locator("#baremo-error:not([hidden])").waitFor(); await monitor.vaciar();
  assert(monitor.respuestas.some((r) => r.ruta === "/simular" && r.estado === 422), "falta rechazo real de Go");
  assert.deepEqual(await exportar(page, botonExportar), invalido, "rechazo Go perdió borrador");
  await importar(page, '[name="archivo"]', borrador);
  const repetida = await comparar(page, monitor, "#baremo-formulario", ".baremo-resultado", "/simular", "101667", "203333");
  assert.deepEqual(repetida[1].datos, cambiada[1].datos, "mismos bytes cambian resultado");
  const jsonRoto = await page.evaluate(async () => {
    const r = await fetch("/simular", { method: "POST", credentials: "omit", headers: { "Content-Type": "application/json" }, body: "{" });
    return { estado: r.status, datos: await r.json() };
  });
  assert.equal(jsonRoto.estado, 400); assert.equal(jsonRoto.datos.error, "solicitud_invalida");
  assert.deepEqual(await exportar(page, botonExportar), borrador, "error HTTP alteró borrador");
  await page.locator('[name="ejemplo"]').selectOption("meritos_sinteticos_v1");
  await page.locator("#baremo-campo-maximo_total").fill("3");
  await comparar(page, monitor, "#baremo-formulario", ".baremo-resultado", "/simular", "4000000", "3000000");
  const meritos = await exportar(page, botonExportar);
  assert.equal(meritos.maximo_total, "3000000");
  await importar(page, '[name="archivo"]', borrador);
  await page.locator("#baremo-error:not([hidden])").waitFor();
  assert.deepEqual(await exportar(page, botonExportar), meritos, "otra familia alteró borrador");
  await page.locator('[data-accion="ayuda"]').click();
  assert.equal(await page.locator('[data-accion="ayuda"]').getAttribute("aria-expanded"), "true");
  await page.locator("#baremo-ayuda").waitFor();
  await page.locator('[data-accion="ayuda"]').click();
  caso.bolsa = { experiencia: ["101667", "203333"], meritos: ["4000000", "3000000"],
    errores: ["json_roto", "archivo_excesivo", "campo_invalido", "familia_distinta", "Go_422", "HTTP_400"], borradorConservado: true, resultadoReproducido: true };
}

async function concursos(page, monitor, caso) {
  await page.locator('[data-panel="concursos"]').click();
  await page.locator("#concursos-formulario").waitFor();
  const configuracion = await page.evaluate(async () => {
    const r = await fetch("/api/provision/v1/configuracion-local", { credentials: "omit" });
    if (!r.ok) throw new Error(`configuración Concursos: ${r.status}`);
    return r.json();
  });
  assert(configuracion.ejemplos?.length, "Concursos no ofrece ejemplo propio");
  assert(configuracion.ejemplos.some((e) => e.referencia === "ejemplo:concursos:v1"));
  const form = "#concursos-formulario", resultado = "[data-concurso-total]", ruta = "/api/provision/v1/simulaciones";
  const campo = page.locator("#concurso-reglas-2-coeficiente");
  assert.equal(await campo.inputValue(), "0.1");
  const original = await comparar(page, monitor, form, resultado, ruta, "28386027", "28386027");
  const desgloses = Object.fromEntries(original[0].datos.resultado.desglose.map((d) => [d.familia, d.resultado]));
  assert.deepEqual(desgloses, { antiguedad: "1200000", cursos: "480000", grado: "19000000",
    permanencia: "2500000", titulaciones: "3000000", valoracion_trabajo: "2206027" });
  await campo.fill("1");
  assert(await campo.evaluate((e) => e === document.activeElement), "editar Concursos perdió foco");
  const cambiada = await comparar(page, monitor, form, resultado, ruta, "28386027", "39186027");
  assert.notEqual(original[1].datos.resultado.huella_resultado, cambiada[1].datos.resultado.huella_resultado);
  const boton = page.locator('[data-concurso-accion="exportar"]'), archivo = '[name="archivo-concurso"]';
  const borrador = await exportar(page, boton);
  assert.equal(borrador.reglas[2].coeficiente, "1000000");
  for (const contenido of ["{json-roto", "x".repeat(256 * 1024 + 1),
    { ...borrador, schema_version: "vec.bolsa.reglas_meritos.v1" }, { ...borrador, campo_no_admitido: true }]) {
    await importar(page, archivo, contenido);
    await page.locator("#concursos-error:not([hidden])").waitFor();
    assert.deepEqual(await exportar(page, boton), borrador, "archivo rechazado perdió borrador de Concursos");
  }
  await monitor.vaciar();
  assert(monitor.respuestas.some((r) => r.ruta === ruta && [400, 422].includes(r.estado)), "falta rechazo Go propio Concursos");
  await campo.fill("-1"); await page.keyboard.press("Tab");
  assert.equal(await campo.getAttribute("aria-invalid"), "true");
  assert(await boton.isDisabled(), "Concursos exporta campo inválido");
  await page.locator(`${form} [type=submit]`).click();
  assert(await campo.evaluate((e) => e === document.activeElement), "validación Concursos no enfoca error");
  await campo.fill("1");
  await page.locator('[data-panel="bolsa"]').click();
  await page.locator('[data-panel="concursos"]').click();
  assert.deepEqual(await exportar(page, boton), borrador, "cambio panel pierde Concursos");
  await importar(page, archivo, borrador);
  const repetida = await comparar(page, monitor, form, resultado, ruta, "28386027", "39186027");
  assert.deepEqual(repetida[1].datos, cambiada[1].datos, "Concursos cambia al repetir borrador");
  const jsonRoto = await page.evaluate(async (url) => {
    const r = await fetch(url, { method: "POST", credentials: "omit", headers: { "Content-Type": "application/json" }, body: "{" });
    return { estado: r.status, datos: await r.json() };
  }, ruta);
  assert.equal(jsonRoto.estado, 400); assert.equal(jsonRoto.datos.error, "solicitud_invalida");
  assert.deepEqual(await exportar(page, boton), borrador, "error HTTP propio altera Concursos");
  await page.locator('[data-concurso-accion="ayuda"]').click();
  assert.equal(await page.locator('[data-concurso-accion="ayuda"]').getAttribute("aria-expanded"), "true");
  await page.locator("#concursos-ayuda").waitFor();
  await page.locator('[data-concurso-accion="ayuda"]').click();
  caso.concursos = { original: "28386027", antiguedadEditada: "39186027", desgloses,
    borradorConservado: true, resultadoReproducido: true, errores: ["json_roto", "archivo_excesivo", "familia_distinta", "Go_rechaza", "campo_invalido", "HTTP_400"] };
}

async function explicaciones(page, monitor, caso) {
  const prefijo = `${caso.idioma}-${caso.anchoCSS}-explicaciones`;
  await fotografiar(page, `${prefijo}-bolsa-inicio.png`);
  await page.locator('[data-panel="concursos"]').click();
  await page.locator("#concursos-formulario").waitFor();
  caso.geometriaConcursos = await geometria(page, "Concursos inicial");
  await fotografiar(page, `${prefijo}-concursos-inicio.png`);
  const form = "#concursos-formulario", resultado = "[data-concurso-total]", ruta = "/api/provision/v1/simulaciones";
  const recibos = await comparar(page, monitor, form, resultado, ruta, "28386027", "28386027");
  const catalogo = JSON.parse(await readFile(join(raiz, `web/static/textos/${caso.idioma}/baremo-concursos.json`), "utf8")).concursos;
  const fila = (familia, lado = 0) => page.locator(".concursos-resultados .baremo-resultado").nth(lado)
    .locator("tbody > tr").filter({ hasText: catalogo[`familia_${familia}`] });
  const abrir = async (f) => {
    const resumen = f.locator("details > summary");
    await resumen.focus(); await page.keyboard.press("Enter");
    assert(await f.locator("details").evaluate((e) => e.open), "explicación no abre por teclado");
  };
  const motivos = [];
  for (const d of recibos[0].datos.resultado.desglose) {
    const f = fila(d.familia); await abrir(f);
    const lineas = await f.locator("details li").allTextContents();
    assert.equal(lineas.length, d.detalles.length);
    for (const [i, detalle] of d.detalles.entries()) {
      const plantilla = catalogo[`motivo_${detalle.motivo}`];
      assert(plantilla, `motivo sin catálogo propio: ${detalle.motivo}`);
      const u = detalle.unidades;
      const variables = { unidades: typeof u === "object" ? `${u.numerador ?? u.n} / ${u.denominador ?? u.d}` : u,
        coeficiente: presentarPuntos(detalle.coeficiente, caso.idioma), bruto: presentarPuntos(detalle.bruto, caso.idioma),
        tope: presentarPuntos(detalle.maximo, caso.idioma), neto: presentarPuntos(detalle.resultado, caso.idioma) };
      assert.equal(lineas[i], plantilla.replace(/\{([^}]+)\}/gu, (_m, clave) => variables[clave]), "motivo visible pierde precisión o significado");
      motivos.push({ familia: d.familia, motivo: detalle.motivo, texto: lineas[i] });
    }
  }
  const curso = recibos[0].datos.resultado.desglose.find((d) => d.familia === "cursos");
  assert.equal(curso.resultado, "480000");
  assert(curso.detalles.some((d) => d.coeficiente === "12000" && d.motivo === "suma_horas"));
  await fotografiar(page, `${prefijo}-curso.png`, fila("cursos").locator("details"));
  const tablaGrado = page.locator("#concurso-reglas-0-tramos-0-coeficiente");
  const resumenGrado = page.locator("[data-concurso-regla]").first().locator("details > summary");
  await resumenGrado.focus(); await page.keyboard.press("Enter");
  assert.equal(await tablaGrado.inputValue(), "19");
  assert.equal(await page.locator("#concurso-reglas-0-tramos-0-maximo").inputValue(), "19");
  assert.equal(await page.locator("#concurso-reglas-0-tramos-0-min_diferencia").inputValue(), "-999");
  assert.equal(await page.locator("#concurso-reglas-0-tramos-0-max_diferencia").inputValue(), "-1");
  const grado = recibos[0].datos.resultado.desglose.find((d) => d.familia === "grado");
  assert.equal(grado.resultado, "19000000");
  await fotografiar(page, `${prefijo}-tabla-grado.png`, tablaGrado);
  await page.locator("#concurso-fecha_corte").fill("2024-05-01");
  const cortados = await comparar(page, monitor, form, resultado, ruta, "28386027", "23129315");
  const excluido = cortados[1].datos.resultado.desglose.find((d) => d.familia === "cursos");
  assert.equal(excluido.resultado, "0");
  assert(excluido.detalles.some((d) => d.motivo === "posterior_corte"));
  const f = fila("cursos", 1); await abrir(f);
  assert((await f.locator("details li").allTextContents()).includes(catalogo.motivo_posterior_corte), "falta explicar corte exclusivo");
  await fotografiar(page, `${prefijo}-corte-excluido.png`, f.locator("details"));
  caso.explicaciones = { motivos, grado: "19000000", curso: "480000", coeficienteCurso: "12000",
    corte: "2024-05-01", cursoCortado: "0", totalCortado: "23129315", corteExclusivo: true };
}

async function recorrer(url, idioma, ancho, escala = 1) {
  cancelacion.signal.throwIfAborted();
  const caso = { idioma, anchoCSS: ancho, escalaDispositivo: escala,
    ...(escala === 2 ? { reflujo: "equivalente 200%; no zoom nativo del navegador" } : {}), estado: "en_curso" };
  informe.casos.push(caso);
  const context = await browser.newContext({ viewport: { width: ancho, height: escala === 2 ? 450 : 900 },
    deviceScaleFactor: escala, locale: idioma, acceptDownloads: true, serviceWorkers: "block" });
  await context.addInitScript(instalarVigilancia);
  const page = await context.newPage();
  page.setDefaultTimeout(12000);
  page.on("dialog", async (dialogo) => { await dialogo.accept(); });
  const monitor = observar(page, url.origin, caso);
  try {
    const destino = new URL(url); destino.searchParams.set("lang", idioma);
    const documento = await page.goto(destino.href, { waitUntil: "networkidle" });
    assert.equal(documento.status(), 200);
    await page.locator("#baremo-formulario").waitFor();
    assert.equal(await page.locator("html").getAttribute("lang"), idioma);
    caso.geometriaInicial = await geometria(page, "inicial");
    if (soloExplicaciones) await explicaciones(page, monitor, caso);
    else {
      caso.teclado = await teclado(page);
      if (escala === 1) await bolsa(page, monitor, caso);
      await fotografiar(page, `${idioma}-${ancho}-${escala}-bolsa.png`);
      caso.geometriaBolsa = await geometria(page, "Bolsa");
      if (!soloBolsa) {
        if (escala === 1) await concursos(page, monitor, caso);
        else { await page.locator('[data-panel="concursos"]').click(); await page.locator("#concursos-formulario").waitFor(); }
        caso.geometriaConcursos = await geometria(page, "Concursos");
        caso.tecladoConcursos = await teclado(page);
        await fotografiar(page, `${idioma}-${ancho}-${escala}-concursos.png`);
      }
    }
    await monitor.comprobar(context);
    caso.estado = "pasado";
  } catch (error) {
    caso.estado = "fallido"; caso.error = error.message;
    await page.screenshot({ path: join(artefactos, `${idioma}-${ancho}-${escala}-fallo.png`), fullPage: true }).catch(() => {});
    throw error;
  } finally { await context.close().catch(() => {}); }
}

try {
  temporal = await mkdtemp(join(tmpdir(), "vec-baremador-proceso-"));
  artefactos = await mkdtemp(join(tmpdir(), "vec-baremador-evidencia-"));
  console.log(`Artefactos: ${artefactos}`);
  informe.commit = await ejecutar("git", ["rev-parse", "HEAD"]);
  informe.cambios = await ejecutar("git", ["status", "--short"]);
  const require = createRequire(import.meta.url);
  let sdk;
  const modulos = [process.env.PLAYWRIGHT_MODULE, "playwright", "playwright-core",
    "/home/alberto/.local/share/openclaw-operativo/app/node_modules/playwright-core"].filter(Boolean);
  for (const modulo of modulos) { try { sdk = require(modulo); break; } catch (error) { if (error.code !== "MODULE_NOT_FOUND") throw error; } }
  assert(sdk?.chromium, "Playwright no está instalado; no se descargan dependencias");
  const chrome = process.env.CHROME_BIN ?? "/usr/bin/google-chrome";
  await access(chrome);
  const go = process.env.GO_BIN ?? await ejecutar("bash", ["scripts/seleccionar_toolchain_go_local.sh"]);
  informe.go = await ejecutar(go, ["version"], { env: { ...process.env, GOTOOLCHAIN: "local", GOPROXY: "off", GOSUMDB: "off" } });
  console.log("Compilando sólo cmd/vec-baremador-web con dependencias locales…");
  const binario = join(temporal, "vec-baremador-web");
  await ejecutar(go, ["build", "-buildvcs=false", "-o", binario, "./cmd/vec-baremador-web"],
    { env: { ...process.env, GOTOOLCHAIN: "local", GOPROXY: "off", GOSUMDB: "off" } });
  const url = await arrancar(binario);
  browser = await sdk.chromium.launch({ executablePath: chrome, headless: true,
    args: ["--disable-background-networking", "--disable-component-update", "--no-first-run"] });
  informe.chrome = browser.version();
  for (const idioma of ["es", "en"]) {
    for (const ancho of soloExplicaciones ? [1440, 390] : [1440, 1024, 390]) {
      console.log(`Recorrido ${idioma}, ${ancho}px…`);
      await recorrer(url, idioma, ancho);
    }
    if (!soloExplicaciones) {
      console.log(`Reflujo equivalente 200%, ${idioma}…`);
      await recorrer(url, idioma, 720, 2);
    }
  }
  informe.estado = "pasado";
} catch (error) {
  informe.estado = cancelacion.signal.aborted ? "interrumpido" : "fallido";
  informe.error = error.message; process.exitCode = 1;
  console.error(error.message);
} finally {
  clearTimeout(limite);
  await browser?.close().catch(() => {});
  for (const p of procesos) matar(p);
  for (const p of [...procesos]) {
    await Promise.race([new Promise((r) => p.once("close", r)), new Promise((r) => setTimeout(r, 2500))]);
    matar(p, "SIGKILL");
  }
  if (temporal) await rm(temporal, { recursive: true, force: true });
  informe.fin = new Date().toISOString();
  if (artefactos) await writeFile(join(artefactos, "resultado.json"), JSON.stringify(informe, null, 2) + "\n");
  console.log(`${informe.estado}: ${artefactos ? join(artefactos, "resultado.json") : "sin artefactos"}`);
  process.removeListener("SIGINT", cancelar);
  process.removeListener("SIGTERM", cancelar);
}
