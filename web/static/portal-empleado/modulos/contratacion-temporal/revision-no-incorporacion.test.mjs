import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { montarRevisionNoIncorporacion } from "./revision-no-incorporacion.js";

const EXP = "expediente:ni:001";
const esperar = () => new Promise((resolver) => setImmediate(resolver));

function pagina() {
  const eventos = new Map();
  const panel = { innerHTML: "", eventos, addEventListener: (tipo, fn) => eventos.set(tipo, fn), removeEventListener: (tipo) => eventos.delete(tipo), querySelector: () => null };
  const estado = { textContent: "" }, numero = { textContent: "" };
  const nodos = new Map([["[data-revision-panel]", panel], ["[data-revision-estado]", estado], ["[data-revision-numero]", numero]]);
  const documento = { title: "", querySelector: (s) => nodos.get(s), querySelectorAll: () => [] };
  const ventana = { eventos: new Map(), confirm: () => true,
    addEventListener(tipo, fn) { this.eventos.set(tipo, fn); }, removeEventListener(tipo) { this.eventos.delete(tipo); } };
  return { documento, ventana, panel, estado, numero };
}

test("la página solo monta la lectura acotada y descarta parámetros ajenos", async () => {
  const html = await readFile(new URL("./revision-no-incorporacion.html", import.meta.url), "utf8");
  assert.doesNotMatch(html, /cuadro|detalle-general|name="expediente_ref"/u);
  const { documento, ventana, panel, numero } = pagina();
  let leido = 0;
  const cliente = { consultarSeguimientoCese: async (ref, { signal }) => { leido++; assert.equal(ref, EXP); assert.ok(signal instanceof AbortSignal);
    return { opciones: { no_incorporacion: { motivos: [], segunda_persona: true } }, estado: { expediente_ref: EXP, numero_expediente: "CT-2026-9",
      version_expediente: 9, fase_clave: "nombramiento", estado_clave: "en_curso", incorporacion: null, cese: null, cierre: null } }; } };
  const quitar = montarRevisionNoIncorporacion({ documento, ventana, cliente,
    localizacion: { href: `https://vec.example/revision-no-incorporacion.html?expediente_ref=${encodeURIComponent(EXP)}&version=999&actor_ref=otro` } });
  await esperar();
  assert.equal(leido, 1);
  assert.match(numero.textContent, /CT-2026-9/u);
  assert.doesNotMatch(panel.innerHTML, /data-ct-seg-form=/u);
  quitar();
  assert.equal(panel.eventos.size, 0);
});

test("sin referencia única la página no consulta y al desmontar aborta la lectura", async () => {
  const p = pagina();
  let signal;
  const cliente = { consultarSeguimientoCese: (_ref, opciones) => { signal = opciones.signal; return new Promise(() => {}); } };
  const invalido = montarRevisionNoIncorporacion({ ...p, cliente, localizacion: { href: "https://vec.example/revision-no-incorporacion.html?version=7" } });
  assert.notEqual(p.estado.textContent, "");
  assert.equal(signal, undefined);
  invalido();
  const quitar = montarRevisionNoIncorporacion({ ...p, cliente,
    localizacion: { href: `https://vec.example/revision-no-incorporacion.html?expediente_ref=${encodeURIComponent(EXP)}` } });
  assert.equal(signal.aborted, false);
  quitar();
  assert.equal(signal.aborted, true);
});

test("el idioma se importa con la misma URL que el resto del portal", async () => {
  // Con otra URL el navegador carga otra copia de idioma.js que no recibe el
  // idioma elegido y la página se queda en castellano aunque pida lang=en.
  for (const archivo of ["./revision-no-incorporacion.js", "./seguimiento-cese.js"]) {
    const codigo = await readFile(new URL(archivo, import.meta.url), "utf8");
    assert.match(codigo, /from "\.\.\/\.\.\/\.\.\/comun\/idioma\.js";/u, archivo);
    assert.doesNotMatch(codigo, /comun\/idioma\.js\?/u, archivo);
  }
});
