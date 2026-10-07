import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { cargarFichaIndicadores, montarFichaIndicadores, normalizarCatalogoIndicadores, renderizarFichaIndicadores } from "./ficha-indicadores.js";

const leerJSON = async (url) => JSON.parse(await readFile(url, "utf8"));
const catalogo = await leerJSON(new URL("./catalogo-indicadores.json", import.meta.url));

test("los cinco indicadores conservan el cálculo publicado y su aprobación pendiente", () => {
  const datos = normalizarCatalogoIndicadores(catalogo);
  assert.deepEqual(datos.indicadores.map((i) => i.id), ["altas", "llamamientos", "formalizaciones", "cierres", "incidencias"]);
  assert.equal(datos.indicadores[0].hito, "min(creado_en)");
  assert.deepEqual(datos.indicadores[3].condicion.estado_clave, ["completado", "cancelado"]);
  assert.equal(datos.indicadores[4].condicion.estado_anterior, "distinto_o_ausente");
  assert.equal(datos.indicadores[2].condicion.fase_clave, "nombramiento");
  assert.equal(datos.responsable, null);
  assert.equal(datos.aprobacion, "pendiente");
  assert.equal(datos.intervalo, "periodos_completos");
  assert.equal(datos.zona_horaria, "Europe/Madrid");
  assert.ok(Object.isFrozen(datos.indicadores[3].condicion.estado_clave));
  assert.throws(() => normalizarCatalogoIndicadores({ ...catalogo, aprobacion: "aprobada" }));
});

test("carga las traducciones reales, sin faltantes, y escapa el texto y la procedencia", async () => {
  for (const idioma of ["es", "en"]) {
    const ficha = await cargarFichaIndicadores({ opcionesTextos: { idioma } });
    assert.deepEqual(ficha.textos.faltantes, []);
    const html = renderizarFichaIndicadores(ficha);
    assert.match(html, /aria-labelledby="ayuda-indicadores-titulo"/);
    assert.equal((html.match(/<dt><strong>/g) ?? []).length, 5);
    assert.match(html, idioma === "es" ? /periodos completos/ : /complete period/);
    assert.match(html, idioma === "es" ? /No acredita firma/ : /does not establish a signature/);
    assert.match(html, /data-analitica-cerrar/);
    const hostil = structuredClone(ficha.catalogo);
    hostil.fuente.funcion = '<img src=x onerror="alert(1)">';
    const escaped = renderizarFichaIndicadores({ ...ficha, catalogo: hostil });
    assert.doesNotMatch(escaped, /<img/);
    assert.match(escaped, /&lt;img/);
    assert.throws(() => renderizarFichaIndicadores({ ...ficha, id: '\" onclick=\"x' }));
  }
});

test("abrir concentra el foco, cerrar o Escape lo devuelve y desmontar retira listeners", async () => {
  const ficha = await cargarFichaIndicadores();
  const listeners = new Map();
  let focoOrigen = 0;
  let focoCerrar = 0;
  let cierres = 0;
  const origen = { isConnected: true, focus: () => { focoOrigen++; } };
  const raiz = { hidden: false, innerHTML: "", ownerDocument: { activeElement: origen },
    addEventListener: (tipo, fn) => listeners.set(tipo, fn), removeEventListener: (tipo) => listeners.delete(tipo),
    querySelector: () => ({ focus: () => { focoCerrar++; } }) };
  const vista = montarFichaIndicadores({ raiz, ...ficha, alCerrar: () => { cierres++; } });
  assert.equal(raiz.hidden, true);
  vista.abrir();
  assert.equal(focoCerrar, 1);
  assert.equal(raiz.hidden, false);
  vista.abrir();
  assert.equal(focoCerrar, 1);
  listeners.get("click")({ target: { closest: () => ({}) } });
  assert.equal(focoOrigen, 1);
  assert.equal(raiz.hidden, true);
  vista.cerrar();
  assert.equal(cierres, 1);
  vista.abrir(origen);
  let prevenido = false;
  listeners.get("keydown")({ key: "Escape", preventDefault: () => { prevenido = true; }, stopPropagation: () => {} });
  assert.equal(prevenido, true);
  assert.equal(focoOrigen, 2);
  vista.abrir(origen);
  vista.desmontar();
  assert.equal(raiz.innerHTML, "");
  assert.equal(listeners.size, 0);
  vista.abrir();
  assert.equal(raiz.hidden, true);
  assert.equal(focoOrigen, 3);
});
