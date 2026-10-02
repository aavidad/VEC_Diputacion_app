import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { runInNewContext } from "node:vm";
import { aMicropuntos, crearEditorBaremo, normalizarMinimoFormacion } from "../baremo-editor.js";

const fuente = await readFile(new URL("./montaje.js", import.meta.url), "utf8");
const entrada = fuente.slice(fuente.indexOf('raiz.addEventListener("input",'), fuente.indexOf('raiz.addEventListener("change",'));
const reglas = JSON.parse(await readFile(new URL("../../../../../../internal/modules/bolsa/application/simulacionbaremo/testdata/meritos_reglas_a.json", import.meta.url), "utf8"));
const ejemplo = { referencia: "meritos_sinteticos_v1", modo: "meritos", reglas };
const ruta = '["reglas",0,"minimo_unidades"]';
const resultado = () => ({ resultado: { estado: "completado", total: "4000000", secciones: [] } });

function montar(editor) {
  let escuchar;
  const exportar = { disabled: false }, estado = {}, boton = {}, panel = { setAttribute() {} };
  const controles = new Map([
    ['[data-accion="exportar"]', exportar], [".baremo-resultados", panel], ['[type="submit"]', boton],
    [".baremo-cabecera [role='status']", estado], ["#baremo-error", { dataset: {} }],
  ]);
  const control = {
    dataset: { ruta, indice: "0" }, value: "20/1", validacion: "", atributos: new Set(["data-minimo-formacion"]),
    hasAttribute(nombre) { return this.atributos.has(nombre); },
    setAttribute(nombre) { this.atributos.add(nombre); }, removeAttribute(nombre) { this.atributos.delete(nombre); },
    setCustomValidity(mensaje) { this.validacion = mensaje; },
  };
  const raiz = { addEventListener(_nombre, callback) { escuchar = callback; }, querySelector(selector) { return controles.get(selector); } };
  const compatibilidad = fuente.slice(fuente.indexOf("function falloRestosBorrador("), fuente.indexOf("function claveErrorCampo("));
  runInNewContext(`${compatibilidad}\n${entrada}`, { raiz, editor, panel: "bolsa", lecturaId: 0, error: "", aMicropuntos,
    t: (clave) => clave, claveErrorCampo: () => "minimo_formacion_invalido", mostrarErrores() {} });
  return { control, exportar, escribir(valor) { control.value = valor; escuchar({ target: control }); } };
}

test("el input real edita horas exactas sin convertirlas a puntos y conserva el ejemplo", async () => {
  const solicitudes = [];
  const editor = crearEditorBaremo({ cliente: { simular: async (s) => { solicitudes.push(s); return resultado(); } } });
  editor.cargar(ejemplo); const vista = montar(editor); vista.escribir("62/2");
  assert.equal(editor.estado().borrador.reglas[0].minimo_unidades, "31/1");
  assert.equal(ejemplo.reglas.reglas[0].minimo_unidades, "20/1");
  await editor.comparar();
  assert.equal(solicitudes.length, 2);
  assert.equal(solicitudes[1].reglas.reglas[0].minimo_unidades, "31/1");
});

test("el montaje conserva entradas inválidas y bloquea envío y exportación hasta corregirlas", async () => {
  let llamadas = 0;
  const editor = crearEditorBaremo({ cliente: { simular: async () => { llamadas++; return resultado(); } } });
  editor.cargar(ejemplo); const vista = montar(editor);
  for (const valor of ["1/0", "1000000001/1", ""]) {
    vista.escribir(valor);
    assert.equal(editor.estado().invalidos[ruta], valor);
    assert.equal(vista.control.value, valor);
    assert.equal(vista.control.validacion, "minimo_formacion_invalido");
    assert.equal(vista.exportar.disabled, true);
    await editor.comparar(); assert.equal(llamadas, 0);
  }
  vista.escribir("0/1");
  assert.deepEqual(editor.estado().invalidos, {});
  assert.equal(vista.control.validacion, ""); assert.equal(vista.exportar.disabled, false);
  await editor.comparar(); assert.equal(llamadas, 2);
});

test("editar desde el montaje cancela una comparación anterior y descarta su respuesta tardía", async () => {
  let resolver, signal;
  const editor = crearEditorBaremo({ cliente: { simular: (_s, o) => {
    signal = o.signal; return new Promise((r) => { resolver = r; });
  } } });
  editor.cargar(ejemplo); const vista = montar(editor), vuelo = editor.comparar();
  vista.escribir("31/1"); assert.equal(signal.aborted, true);
  resolver(resultado()); await vuelo;
  assert.equal(editor.estado().comparacion, null); assert.equal(editor.estado().trabajando, false);
});

test("la validación del montaje detecta límites semánticos que el patrón HTML permite", () => {
  const pista = {}, control = { value: "1000000001/1", validacion: "", atributos: new Set(["data-minimo-formacion"]),
    hasAttribute(nombre) { return this.atributos.has(nombre); }, setAttribute() {},
    setCustomValidity(mensaje) { this.validacion = mensaje; }, checkValidity() { return !this.validacion; },
    closest: () => ({ querySelector: () => pista }),
  };
  const raiz = { querySelectorAll: (selector) => selector.startsWith("select") ? [] : [control], querySelector: () => null };
  const validacion = fuente.slice(fuente.indexOf("function claveErrorCampo("), fuente.indexOf('raiz.addEventListener("focusout",'));
  const contexto = { raiz, editor: { estado: () => ({}) }, normalizarMinimoFormacion, t: (clave) => clave };
  runInNewContext(`${validacion}\nresultado = mostrarErrores();`, contexto);
  assert.equal(contexto.resultado, false);
  assert.equal(pista.textContent, "minimo_formacion_invalido"); assert.equal(pista.hidden, false);
});
