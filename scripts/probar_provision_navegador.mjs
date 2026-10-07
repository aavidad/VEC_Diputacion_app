#!/usr/bin/env node
// Chrome y HTTP locales reales. Las expectativas no sustituyen respuestas.
import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { createHash } from "node:crypto";
import { access, mkdir, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { createRequire } from "node:module";
import { homedir, tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const propia = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const ayuda = `node scripts/probar_provision_navegador.mjs [--source-root RUTA] [--server-bin RUTA] [--escenarios ID,ID] [--caso es:390]
--verificar-configuracion comprueba el contrato sin arrancar Go ni Chrome.
CHROME_BIN, PLAYWRIGHT_MODULE y GO_BIN seleccionan herramientas locales ya instaladas.
La fuente debe contener vec-baremador-web y la interfaz Provisión agrupados.
Acta/capturas: ~/.local/state/vec-codexb-provision-20261001/run-*/`;
const args = process.argv.slice(2), opciones = {};
for (let i = 0; i < args.length; i++) {
  const clave = args[i];
  if (["--help", "--verificar-configuracion"].includes(clave)) opciones[clave] = true;
  else {
    assert(["--source-root", "--server-bin", "--escenarios", "--caso"].includes(clave), `argumento desconocido: ${clave}`);
    assert(args[i + 1] && !args[i + 1].startsWith("--"), `falta valor: ${clave}`);
    assert(!(clave in opciones), `argumento repetido: ${clave}`);
    opciones[clave] = args[++i];
  }
}
if (opciones["--help"]) { console.log(ayuda); process.exit(0); }
const fuente = resolve(opciones["--source-root"] ?? propia);
const casoUnico = opciones["--caso"];
assert(!casoUnico || /^(?:es|en):(?:1440|390|720)$/u.test(casoUnico), "caso inválido; use idioma:ancho");
const contrato = JSON.parse(await readFile(join(propia, "scripts/recorridos/provision_expectativas.json"), "utf8"));
assert.equal(contrato.schema_version, "vec.recorrido_provision.v1");
for (const clave of ["pagina", "configuracion", "simulacion", "endpoint_institucional_ausente"]) {
  assert(/^\/(?!\/)[^?#]+$/u.test(contrato[clave]), `ruta local inválida: ${clave}`);
}
assert.deepEqual(contrato.idiomas, ["es", "en"]);
assert.deepEqual(contrato.anchos, [1440, 390]);
assert(Object.values(contrato.selectores).every((s) => typeof s === "string" && s.length > 0));
const elegidos = opciones["--escenarios"]?.split(",") ?? [];
const escenarios = elegidos.map((id) => {
  const escenario = contrato.escenarios.find((e) => e.id === id);
  assert(escenario, `escenario pendiente o desconocido: ${id}`);
  assert(escenario.selector && /^\/api\/provision\//u.test(escenario.ruta));
  assert(Number.isInteger(escenario.estado_http) && escenario.esperado && Object.keys(escenario.esperado).length);
  return escenario;
});
if (opciones["--verificar-configuracion"]) { console.log("Contrato Provisión válido; navegador no ejecutado."); process.exit(0); }

const informe = { esquema: contrato.schema_version, inicio: new Date().toISOString(), estado: "en_curso",
  modo: casoUnico ? "focal" : "matriz", ...(casoUnico ? { casoSolicitado: casoUnico } : {}),
  alcance: "simulador_interno_sintetico_local", transporte: "HTTP real, sin interceptar APIs",
  pendientes: ["portal_institucional", "PostgreSQL", "identidad_y_autorizacion", "firma", "efectos_administrativos",
    "zoom_nativo_200_por_ciento", "revision_visual_independiente"],
  escenariosSolicitados: elegidos, casos: [] };
const procesos = new Set(), cancelacion = new AbortController();
let temporal, artefactos, browser;
const matar = (p, senal = "SIGTERM") => {
  if (p?.exitCode === null && p.signalCode === null) {
    try { process.kill(-p.pid, senal); } catch (e) { if (e.code !== "ESRCH") throw e; }
  }
};
const cancelar = () => {
  cancelacion.abort(); for (const p of procesos) matar(p); void browser?.close().catch(() => {});
};
process.on("SIGINT", cancelar); process.on("SIGTERM", cancelar);
const limite = setTimeout(cancelar, 8 * 60 * 1000);

function proceso(programa, argumentos, listo) {
  cancelacion.signal.throwIfAborted();
  return new Promise((aceptar, rechazar) => {
    const p = spawn(programa, argumentos, { cwd: fuente, detached: true, stdio: ["ignore", "pipe", "pipe"],
      env: { ...process.env, GOTOOLCHAIN: "local", GOPROXY: "off", GOSUMDB: "off" } });
    procesos.add(p); let salida = "", error = "", entregado = false;
    const tiempo = setTimeout(() => { matar(p, "SIGKILL"); rechazar(new Error("proceso excede su límite")); }, listo ? 15000 : 180000);
    p.stdout.on("data", (b) => {
      salida += b;
      if (salida.length > 1048576) { matar(p); rechazar(new Error("salida excesiva")); }
      const valor = listo?.(salida);
      if (valor && !entregado) { entregado = true; clearTimeout(tiempo); aceptar(valor); }
    });
    p.stderr.on("data", (b) => { error = (error + b).slice(-2000); });
    p.once("error", (e) => { clearTimeout(tiempo); procesos.delete(p); rechazar(e); });
    p.once("close", (codigo) => {
      clearTimeout(tiempo); procesos.delete(p);
      if (listo && entregado) return;
      if (codigo === 0 && !listo) aceptar(salida.trim());
      else rechazar(new Error(`${programa} terminó (${codigo}): ${error}`));
    });
  });
}

function vigilarAlmacenamiento() {
  globalThis.__vecAlmacenamiento = [];
  const registrar = (nombre) => globalThis.__vecAlmacenamiento.push(nombre);
  for (const nombre of ["setItem", "removeItem", "clear"]) {
    const original = Storage.prototype[nombre];
    Storage.prototype[nombre] = function (...args) { registrar(`Storage.${nombre}`); return original.apply(this, args); };
  }
  const cookie = Object.getOwnPropertyDescriptor(Document.prototype, "cookie");
  if (cookie?.set) Object.defineProperty(Document.prototype, "cookie", {
    ...cookie, set(v) { registrar("cookie"); return cookie.set.call(this, v); },
  });
  for (const [objeto, nombre] of [[indexedDB, "open"], [indexedDB, "deleteDatabase"],
    [globalThis.caches, "open"], [navigator.serviceWorker, "register"]]) {
    if (objeto?.[nombre]) {
      const original = objeto[nombre].bind(objeto);
      objeto[nombre] = (...args) => { registrar(nombre); return original(...args); };
    }
  }
}

function observar(page, origen) {
  const errores = [], solicitudes = [], respuestas = [], pendientes = new Set();
  page.on("pageerror", (e) => errores.push({ tipo: "js", texto: e.message }));
  page.on("console", (m) => { if (m.type() === "error") errores.push({ tipo: "consola", texto: m.text() }); });
  page.on("request", (r) => solicitudes.push({ fuera: new URL(r.url()).origin !== origen,
    credenciales: Boolean(r.headers().cookie || r.headers().authorization) }));
  page.on("response", (r) => {
    const tarea = r.allHeaders().then((h) => respuestas.push({ ruta: new URL(r.url()).pathname,
      metodo: r.request().method(), estado: r.status(), cookie: Boolean(h["set-cookie"]) }));
    pendientes.add(tarea); tarea.then(() => pendientes.delete(tarea), (e) => {
      pendientes.delete(tarea); errores.push({ tipo: "monitor", texto: e.message });
    });
  });
  return async (context, caso) => {
    while (pendientes.size) await Promise.all([...pendientes]);
    assert(solicitudes.every((s) => !s.fuera && !s.credenciales), "petición externa o con credenciales");
    assert(!respuestas.some((r) => r.cookie), "Set-Cookie recibido");
    assert.deepEqual(await context.cookies(), [], "cookies conservadas");
    const storage = await page.evaluate(async () => ({ intentos: globalThis.__vecAlmacenamiento,
      local: localStorage.length, sesion: sessionStorage.length, bases: await indexedDB.databases(), caches: await caches.keys() }));
    assert.deepEqual(storage, { intentos: [], local: 0, sesion: 0, bases: [], caches: [] }, "almacenamiento web usado");
    const provocados = respuestas.filter((r) => r.ruta === contrato.simulacion && r.estado === 400
      || r.ruta === contrato.endpoint_institucional_ausente && r.estado === 404
      || r.estado >= 400 && escenarios.some((e) => e.ruta === r.ruta && e.estado_http === r.estado));
    const disponibles = provocados.map((r) => r.estado);
    const red = errores.filter((e) => {
      const codigo = e.tipo === "consola" && e.texto.match(/^Failed to load resource:.*\b(\d{3})\b/u);
      const indice = codigo ? disponibles.indexOf(Number(codigo[1])) : -1;
      if (indice < 0) return false;
      disponibles.splice(indice, 1); return true;
    });
    assert.deepEqual(errores.filter((e) => !red.includes(e)), [], "errores JS o consola inesperados");
    assert(respuestas.every((r) => r.estado < 400 || provocados.includes(r)), "respuesta HTTP inesperada");
    caso.vigilancia = { erroresJS: 0, erroresConsolaInesperados: 0, diagnosticosRedProvocados: red.length,
      llamadasExternas: 0, cookies: 0, almacenamiento: storage };
    caso.http = respuestas;
  };
}

async function geometria(page) {
  const medidas = await page.evaluate(() => {
    const limites = (e) => {
      let izquierda = 0, derecha = innerWidth;
      for (let padre = e.parentElement; padre; padre = padre.parentElement) {
        if (["hidden", "clip", "auto", "scroll"].includes(getComputedStyle(padre).overflowX)) {
          const r = padre.getBoundingClientRect(); izquierda = Math.max(izquierda, r.left); derecha = Math.min(derecha, r.right);
        }
      }
      const r = e.getBoundingClientRect();
      return { izquierda: r.left, derecha: r.right, sinRecorte: r.left >= izquierda - 1 && r.right <= derecha + 1 };
    };
    return { ancho: innerWidth, documento: document.documentElement.scrollWidth,
    cuerpo: document.body.scrollWidth, tablas: [...document.querySelectorAll(".tabla-contenedor")]
      .filter((e) => e.getClientRects().length).map((e) => ({ ancho: e.clientWidth, contenido: e.scrollWidth,
        overflow: getComputedStyle(e).overflowX, ...limites(e) })),
    tecnicos: [...document.querySelectorAll("details[open] dd,details[open] p")].filter(e => e.getClientRects().length)
      .map(e => ({ ...limites(e), sinContenidoRecortado: e.scrollWidth <= e.clientWidth + 1 })),
    acciones: [...document.querySelectorAll('[data-foco="simular"],[data-foco="corregir-configuracion"],[data-foco$="-simular"],[data-foco$="-resolver"],[data-foco$="-alegar"]')]
      .filter((e) => e.getClientRects().length).map((e) => ({ accion: e.dataset.foco, ...limites(e) })) };
  });
  assert(medidas.documento <= medidas.ancho + 1 && medidas.cuerpo <= medidas.ancho + 1, "desbordamiento horizontal global");
  assert(medidas.tablas.every((t) => t.contenido <= t.ancho + 1 || ["auto", "scroll"].includes(t.overflow)), "tabla sin scroll interno");
  assert(medidas.tablas.every((t) => t.sinRecorte), `contenedor de tabla recortado: ${JSON.stringify(medidas.tablas)}`);
  assert(medidas.acciones.every((a) => a.sinRecorte), `acción recortada: ${JSON.stringify(medidas.acciones)}`);
  assert(medidas.tecnicos.every(t => t.sinRecorte && t.sinContenidoRecortado), `detalle técnico recortado: ${JSON.stringify(medidas.tecnicos)}`);
  return medidas;
}

async function teclado(page) {
  const fechas = new Map();
  for (const campo of await page.locator('input[type="date"]:visible').all()) {
    fechas.set(await campo.getAttribute("data-foco"), createHash("sha256").update(await campo.screenshot({ caret: "hide" })).digest("hex"));
  }
  await page.evaluate(() => { document.body.tabIndex = -1; document.body.focus(); document.body.removeAttribute("tabindex"); });
  const vistos = new Set();
  const maximo = await page.locator('button:visible:not([disabled]),input:visible:not([disabled]),select:visible:not([disabled]),summary:visible,a:visible,[tabindex="0"]:visible').count() * 3 + 5;
  let primero;
  for (let i = 0; i < maximo; i++) {
    await page.keyboard.press("Tab");
    // Espera el desplazamiento nativo de foco; no desplaza la página por código.
    await page.waitForFunction(() => {
      const e = document.activeElement, r = e.getBoundingClientRect();
      const encima = document.elementFromPoint(Math.max(0, Math.min(innerWidth - 1, r.x + r.width / 2)),
        Math.max(0, Math.min(innerHeight - 1, r.y + r.height / 2)));
      return r.width > 0 && r.height > 0 && r.right > 0 && r.left < innerWidth && r.bottom > 0 && r.top < innerHeight
        && (encima === e || e.contains(encima));
    }, undefined, { timeout: 1200 }).catch(() => {});
    const foco = await page.evaluate(() => {
      const e = document.activeElement, r = e.getBoundingClientRect(), s = getComputedStyle(e);
      const encima = document.elementFromPoint(Math.max(0, Math.min(innerWidth - 1, r.x + r.width / 2)),
        Math.max(0, Math.min(innerHeight - 1, r.y + r.height / 2)));
      return { tag: e.tagName, clave: e.dataset.foco || e.id || e.name || e.outerHTML.slice(0, 100),
        rect: { x: r.x, y: r.y, ancho: r.width, alto: r.height },
        tipo: e.type, focusWithin: e.matches(":focus-within"),
        visible: r.width > 0 && r.height > 0 && r.right > 0 && r.left < innerWidth && r.bottom > 0 && r.top < innerHeight,
        tapado: encima !== e && !e.contains(encima),
        indicador: parseFloat(s.outlineWidth) > 0 && s.outlineStyle !== "none" || s.boxShadow !== "none" };
    });
    if (foco.tag === "BODY") continue;
    if (!foco.indicador && foco.tipo === "date" && foco.focusWithin) {
      const campo = page.locator(`[data-foco="${foco.clave}"]`);
      const captura = await campo.screenshot({ caret: "hide" });
      foco.indicador = createHash("sha256").update(captura).digest("hex") !== fechas.get(foco.clave);
      foco.indicadorNativo = foco.indicador;
    }
    assert(foco.visible && !foco.tapado && foco.indicador, `foco invisible, tapado o sin indicador: ${JSON.stringify(foco)}`);
    primero ??= foco.clave;
    if (vistos.size > 1 && foco.clave === primero) break;
    vistos.add(foco.clave);
  }
  assert(vistos.size >= 4, "teclado no alcanza controles principales");
  return [...vistos];
}

async function responder(page, selector, ruta, estado = 200) {
  const espera = page.waitForResponse((r) => new URL(r.url()).pathname === ruta && r.request().method() === "POST");
  await page.locator(selector).click();
  const r = await espera; assert.equal(r.status(), estado);
  const bytes = await r.body();
  return { datos: JSON.parse(bytes.toString()), sha256: createHash("sha256").update(bytes).digest("hex"),
    entrada: r.request().postData() };
}

async function compararVista(page, datos, idioma, catalogo) {
  assert.equal(datos.alcance, "simulacion"); assert.equal(datos.estado, "borrador");
  assert(datos.valoraciones.length > 0, "respuesta sin valoraciones");
  const bloques = page.locator(".cuerpo-panel > details");
  await page.waitForFunction((cantidad) => document.querySelectorAll(".cuerpo-panel > details").length === cantidad,
    datos.valoraciones.length);
  for (const [i, valoracion] of datos.valoraciones.entries()) {
    const r = valoracion.resultado, bloque = bloques.nth(i);
    const puntos = (v) => {
      const n = BigInt(v), fraccion = String(n % 1000000n).padStart(6, "0").replace(/0+$/u, "");
      const formato = new Intl.NumberFormat(idioma);
      const separador = formato.formatToParts(1.1).find((p) => p.type === "decimal").value;
      return formato.format(n / 1000000n) + (fraccion ? separador + fraccion : "");
    };
    assert.equal(await bloque.locator("dd").last().textContent(), r.total === null ? catalogo.estados.sin_dato : puntos(r.total));
    const filas = bloque.locator("table").nth(1).locator("tbody > tr");
    assert.equal(await filas.count(), r.desglose.length);
    for (const [j, d] of r.desglose.entries()) {
      assert.equal(await filas.nth(j).locator("td").last().textContent(),
        d.estado === "pendiente_dato" ? catalogo.estados.sin_dato : puntos(d.resultado), "desglose visible distinto del servidor");
    }
  }
}

async function preferencias(page) {
  const s = contrato.selectores;
  await page.locator(s.personal).click();
  for (const selector of [s.quitar_primero, s.quitar_segundo]) {
    if (await page.locator(selector).count()) await page.locator(selector).click();
  }
  await page.locator(s.puestos).click();
  for (const selector of [s.seleccionar_primero, s.seleccionar_segundo]) {
    if (await page.locator(selector).isEnabled()) await page.locator(selector).click();
  }
  await page.locator(s.personal).click();
  for (const [selector, siguiente] of [[s.bajar_primero, s.subir_primero], [s.subir_primero, s.bajar_primero]]) {
    await page.locator(selector).focus(); await page.keyboard.press("Enter");
    await page.evaluate(() => new Promise((r) => requestAnimationFrame(r)));
    assert(await page.locator(siguiente).evaluate((e) => e === document.activeElement), "mover preferencia pierde foco en el extremo");
  }
}

const punteroJSON = (datos, puntero) => puntero.split("/").slice(1)
  .reduce((v, k) => v?.[k.replaceAll("~1", "/").replaceAll("~0", "~")], datos);
function puntosEnsayo(valor, idioma) {
  const n = BigInt(valor), fraccion = String(n % 1000000n).padStart(6, "0").replace(/0+$/u, "");
  const formato = new Intl.NumberFormat(idioma);
  return formato.format(n / 1000000n) + (fraccion ? formato.formatToParts(1.1).find(p => p.type === "decimal").value + fraccion : "");
}
async function recorrerEnsayo(page, escenario, caso, prefijo, catalogo, idioma) {
  for (const paso of escenario.pasos ?? []) {
    const control = page.locator(paso.selector);
    if (paso.accion === "click") await control.click();
    else if (paso.accion === "fill") await control.fill(paso.valor);
    else if (paso.accion === "selectOption") await control.selectOption(paso.valor);
    else throw new Error(`acción no admitida: ${paso.accion}`);
  }
  const r = await responder(page, escenario.selector, escenario.ruta, escenario.estado_http);
  for (const [p, esperado] of Object.entries(escenario.esperado)) assert.deepEqual(punteroJSON(r.datos, p), esperado, `${escenario.id}: ${p}`);
  for (const [p, cantidad] of Object.entries(escenario.cantidades ?? {})) assert.equal(punteroJSON(r.datos, p)?.length, cantidad, `${escenario.id}: ${p}`);
  const repetida = await responder(page, escenario.selector, escenario.ruta, escenario.estado_http);
  assert.equal(repetida.entrada, r.entrada, "ensayo cambia petición al repetir");
  assert.equal(repetida.sha256, r.sha256, "ensayo cambia bytes al repetir");
  assert.deepEqual(repetida.datos, r.datos, "ensayo cambia resultado al repetir");
  const entrada = JSON.parse(r.entrada), esCiclo = escenario.tipo === "ciclo";
  assert.deepEqual(Object.keys(entrada).sort(), esCiclo ? ["caso_ref", "ejemplo_ref"] : ["configuracion", "ejemplo_ref"]);
  const espacio = page.locator(esCiclo ? '[data-ensayo-ciclo]' : '[data-ensayo-adjudicacion]');
  const titulo = esCiclo ? catalogo.ciclo.cronologia : catalogo.ensayos.asignaciones;
  const filas = espacio.getByRole("table", { name: titulo, exact: true }).locator("tbody > tr");
  await filas.first().waitFor();
  await page.waitForFunction(selector => document.activeElement === document.querySelector(selector), escenario.selector);
  if (esCiclo) {
    assert.equal(await filas.count(), r.datos.valoraciones.length);
    for (const [i, v] of r.datos.valoraciones.entries()) {
      assert.equal(await filas.nth(i).locator("td").nth(1).textContent(), v.resultado.total === null ? catalogo.estados.sin_dato : puntosEnsayo(v.resultado.total, idioma));
      assert.match(v.huella_revision, /^[a-f0-9]{64}$/u);
      if (i) assert.equal(v.huella_anterior, r.datos.valoraciones[i - 1].huella_revision);
    }
    if (entrada.caso_ref === "rectificar") assert.notEqual(r.datos.valoraciones[1].resultado.total, r.datos.valoraciones[0].resultado.total);
    if (entrada.caso_ref === "mantener") assert.equal(r.datos.valoraciones[1].resultado.total, r.datos.valoraciones[0].resultado.total);
    for (const clave of ["ciclo-alegar", "ciclo-resolver"]) assert(await page.locator(`[data-foco="${clave}"]`).isDisabled());
  } else {
    assert.match(r.datos.huella_resultado, /^[a-f0-9]{64}$/u);
    assert.equal(new Set(r.datos.asignaciones.map(a => a.persona_ref)).size, r.datos.asignaciones.length);
    assert.equal(new Set(r.datos.asignaciones.map(a => a.vacante_ref)).size, r.datos.asignaciones.length);
    assert.equal(await filas.count(), r.datos.asignaciones.length);
    for (const [i, a] of r.datos.asignaciones.entries()) {
      assert.equal(await filas.nth(i).locator("th").textContent(), catalogo.ensayos.personas[a.persona_ref.replaceAll(":", "_")]);
      assert.equal(await filas.nth(i).locator("td").first().textContent(), catalogo.ensayos.puestos[a.puesto_ref.replaceAll(":", "_")]);
    }
    assert(await page.locator('[data-foco="adjudicacion-resolver"]').isDisabled());
  }
  for (const detalle of await espacio.locator("details").all()) {
    if (!(await detalle.evaluate(e => e.open))) { await detalle.locator("summary").focus(); await page.keyboard.press("Enter"); }
    assert(await detalle.evaluate(e => e.open), "detalle no abre por teclado");
  }
  const registro = { id: escenario.id, sha256: r.sha256, entrada, respuesta: r.datos,
    focoRetenidoTrasRespuesta: true, geometria: await geometria(page), teclado: await teclado(page) };
  await page.screenshot({ path: join(artefactos, `${prefijo}-${escenario.id}.png`), fullPage: false });
  if (!esCiclo) {
    const campo = page.locator('[data-foco="adjudicacion-politica_ref"]'), valor = await campo.inputValue();
    await campo.fill(""); await page.keyboard.press("Tab");
    assert.equal(await campo.getAttribute("aria-invalid"), "true");
    assert(await page.locator(escenario.selector).isDisabled());
    assert.equal(await page.locator("#adjudicacion-error-politica_ref").textContent(), catalogo.ensayos.configuracion_invalida);
    registro.geometriaError = await geometria(page);
    await page.screenshot({ path: join(artefactos, `${prefijo}-adjudicacion-error.png`), fullPage: false });
    await campo.fill(valor); await page.keyboard.press("Tab"); assert(await page.locator(escenario.selector).isEnabled());
    registro.validacionVisible = true;
  }
  caso.escenarios.push(registro); return r.datos;
}

async function recorrer(url, idioma, ancho, reflujo = false) {
  cancelacion.signal.throwIfAborted();
  const caso = { idioma, anchoCSS: ancho, estado: "en_curso",
    ...(reflujo ? { reflujo: "720 CSS px con escala 2; no acredita zoom nativo" } : {}) };
  informe.casos.push(caso);
  const context = await browser.newContext({ viewport: { width: ancho, height: reflujo ? 450 : 900 },
    deviceScaleFactor: reflujo ? 2 : 1, locale: idioma, serviceWorkers: "block" });
  await context.addInitScript(vigilarAlmacenamiento);
  const page = await context.newPage(); page.setDefaultTimeout(12000);
  const comprobar = observar(page, url.origin), prefijo = `${idioma}-${ancho}`;
  try {
    const destino = new URL(contrato.pagina, url); destino.searchParams.set("lang", idioma);
    const configuracion = page.waitForResponse((r) => new URL(r.url()).pathname === contrato.configuracion);
    assert.equal((await page.goto(destino.href, { waitUntil: "networkidle" })).status(), 200);
    assert.equal((await configuracion).status(), 200);
    await page.locator(contrato.selectores.raiz).waitFor();
    const catalogo = JSON.parse(await readFile(join(fuente, `web/static/textos/${idioma}/provision.json`), "utf8"));
    assert.equal(await page.locator("html").getAttribute("lang"), idioma);
    if (!escenarios.length) {
    caso.inicio = await geometria(page);
    await page.screenshot({ path: join(artefactos, `${prefijo}-inicio.png`), fullPage: false });
    caso.teclado = await teclado(page);
    await preferencias(page);
    caso.preferencias = await geometria(page);
    await page.screenshot({ path: join(artefactos, `${prefijo}-preferencias.png`), fullPage: false });
    await page.locator(contrato.selectores.valoracion).click();
    const primera = await responder(page, contrato.selectores.simular, contrato.simulacion);
    const repetida = await responder(page, contrato.selectores.simular, contrato.simulacion);
    assert.equal(primera.entrada, repetida.entrada, "UI cambia entrada al repetir");
    assert.deepEqual(primera.datos, repetida.datos, "desglose no determinista");
    assert.equal(primera.sha256, repetida.sha256, "mismos bytes cambian respuesta");
    await compararVista(page, repetida.datos, idioma, catalogo);
    caso.simulacion = { sha256: primera.sha256, respuesta: primera.datos, reproducida: true };
    caso.resultado = await geometria(page);
    await page.screenshot({ path: join(artefactos, `${prefijo}-valoracion.png`), fullPage: false });
    await page.locator(contrato.selectores.convocatoria).click();
    const campo = page.locator(contrato.selectores.maximo), valor = await campo.inputValue();
    await campo.fill("abc"); await page.keyboard.press("Tab");
    assert.equal(await campo.getAttribute("aria-invalid"), "true", "campo inválido sin indicación accesible");
    assert.equal(await campo.inputValue(), "abc", "validación pierde dato que debe corregirse");
    const explicaciones = await campo.evaluate((e) => (e.getAttribute("aria-describedby") ?? "").split(/\s+/u)
      .map((id) => e.ownerDocument.getElementById(id)?.textContent).filter(Boolean));
    assert(explicaciones.includes(catalogo.configuracion.invalida), "error del campo sin asociación o sin catálogo");
    await page.locator(contrato.selectores.valoracion).click();
    assert(await page.locator(contrato.selectores.simular).isDisabled(), "configuración inválida permite simular");
    assert.equal(await page.locator('[data-provision-aviso]').textContent(), catalogo.configuracion.errores_pendientes,
      "configuración inválida sin explicación localizada");
    caso.errorConfiguracion = await geometria(page);
    await page.screenshot({ path: join(artefactos, `${prefijo}-configuracion-invalida.png`), fullPage: false });
    await page.locator(contrato.selectores.convocatoria).click(); await campo.fill(valor); await page.keyboard.press("Tab");
    await page.evaluate(() => new Promise((r) => requestAnimationFrame(r)));
    assert(!(await campo.evaluate((e) => e === document.activeElement)), "editar configuración atrapa Tab en el campo");
    await page.locator(contrato.selectores.valoracion).click();
    assert(await page.locator(contrato.selectores.simular).isEnabled(), "corrección no recupera simulación");
    caso.validacionVisible = { datoConservado: true, configuracionInvalidaBloqueada: true, correccionRecuperada: true };
    }
    caso.escenarios = [];
    let inicialCiclo;
    for (const escenario of escenarios) {
      const resultado = await recorrerEnsayo(page, escenario, caso, prefijo, catalogo, idioma);
      if (escenario.tipo === "ciclo") {
        inicialCiclo ??= resultado.valoraciones[0];
        assert.deepEqual(resultado.valoraciones[0], inicialCiclo, "otro caso reescribe la versión inicial");
      }
    }
    if (!escenarios.length) {
      const errores = await page.evaluate(async ({ simulacion, endpoint_institucional_ausente }) => {
        const roto = await fetch(simulacion, { method: "POST", credentials: "omit", headers: { "Content-Type": "application/json" }, body: "{" });
        const ausente = await fetch(endpoint_institucional_ausente, { credentials: "omit" });
        return { roto: roto.status, ausente: ausente.status };
      }, contrato);
      assert.deepEqual(errores, { roto: 400, ausente: 404 }); caso.rechazos = errores;
    }
    await comprobar(context, caso); caso.estado = "pasado";
  } catch (e) {
    caso.estado = "fallido"; caso.error = e.message;
    await page.screenshot({ path: join(artefactos, `${prefijo}-fallo.png`), fullPage: true }).catch(() => {});
    throw e;
  } finally { await context.close().catch(() => {}); }
}

try {
  const evidencia = join(homedir(), ".local/state/vec-codexb-provision-20261001");
  await mkdir(evidencia, { recursive: true, mode: 0o700 });
  artefactos = await mkdtemp(join(evidencia, "run-")); temporal = await mkdtemp(join(tmpdir(), "vec-provision-proceso-"));
  console.log(`Artefactos: ${artefactos}`);
  informe.sha256Guion = createHash("sha256").update(await readFile(fileURLToPath(import.meta.url))).digest("hex");
  informe.sha256Expectativas = createHash("sha256").update(await readFile(join(propia, "scripts/recorridos/provision_expectativas.json"))).digest("hex");
  informe.commitFuente = await proceso("git", ["rev-parse", "HEAD"]);
  informe.cambiosFuente = await proceso("git", ["status", "--short"]);
  const require = createRequire(import.meta.url); let sdk;
  for (const modulo of [process.env.PLAYWRIGHT_MODULE, "playwright", "playwright-core",
    "/home/alberto/.local/share/openclaw-operativo/app/node_modules/playwright-core"].filter(Boolean)) {
    try { sdk = require(modulo); break; } catch (e) { if (e.code !== "MODULE_NOT_FOUND") throw e; }
  }
  assert(sdk?.chromium, "Playwright no instalado; no se descargan dependencias");
  const chrome = process.env.CHROME_BIN ?? "/usr/bin/google-chrome"; await access(chrome);
  let binario = opciones["--server-bin"] && resolve(opciones["--server-bin"]);
  if (!binario) {
    const go = process.env.GO_BIN ?? await proceso("bash", ["scripts/seleccionar_toolchain_go_local.sh"]);
    binario = join(temporal, "vec-baremador-web");
    await proceso(go, ["build", "-buildvcs=false", "-o", binario, "./cmd/vec-baremador-web"]);
  }
  informe.sha256Binario = createHash("sha256").update(await readFile(binario)).digest("hex");
  const url = await proceso(binario, ["--puerto", "0", "--web-dir", join(fuente, "web/static")], (salida) => {
    const direccion = salida.match(/http:\/\/127\.0\.0\.1:\d+\/[^\s]+/u); return direccion && new URL(direccion[0]);
  });
  browser = await sdk.chromium.launch({ executablePath: chrome, headless: true,
    args: ["--disable-background-networking", "--disable-component-update", "--no-first-run"] });
  informe.chrome = browser.version();
  for (const idioma of contrato.idiomas) {
    for (const ancho of contrato.anchos) {
      if (!casoUnico || casoUnico === `${idioma}:${ancho}`) {
        console.log(`Provisión ${idioma} ${ancho}px`); await recorrer(url, idioma, ancho);
      }
    }
    if (!casoUnico || casoUnico === `${idioma}:720`) await recorrer(url, idioma, 720, true);
  }
  informe.estado = "pasado";
} catch (e) {
  informe.estado = cancelacion.signal.aborted ? "interrumpido" : "fallido"; informe.error = e.message;
  console.error(e.message); process.exitCode = 1;
} finally {
  clearTimeout(limite); await browser?.close().catch(() => {});
  for (const p of procesos) matar(p);
  for (const p of [...procesos]) {
    if (p.exitCode === null && p.signalCode === null) {
      await Promise.race([new Promise((r) => p.once("close", r)), new Promise((r) => setTimeout(r, 2500))]); matar(p, "SIGKILL");
    }
  }
  if (temporal) await rm(temporal, { recursive: true, force: true });
  informe.fin = new Date().toISOString();
  if (artefactos) await writeFile(join(artefactos, "resultado.json"), JSON.stringify(informe, null, 2) + "\n", { mode: 0o600 });
  console.log(`${informe.estado}: ${artefactos ? join(artefactos, "resultado.json") : "sin artefactos"}`);
  process.removeListener("SIGINT", cancelar); process.removeListener("SIGTERM", cancelar);
}
