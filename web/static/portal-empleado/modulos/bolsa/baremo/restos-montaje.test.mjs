import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { runInNewContext } from "node:vm";
import { aMicropuntos, crearEditorBaremo, errorRestosRegla, leerReglas, MAXIMO_ARCHIVO, normalizarMinimoFormacion } from "../baremo-editor.js";

const fuente = await readFile(new URL("./montaje.js", import.meta.url), "utf8");
const auxiliares = fuente.slice(fuente.indexOf("function falloRestosBorrador("), fuente.indexOf('raiz.addEventListener("focusout",'));
const eventos = fuente.slice(fuente.indexOf('raiz.addEventListener("input",'), fuente.indexOf('window.addEventListener("beforeunload",'));
const reglas = JSON.parse(await readFile(new URL("../../../../../../internal/modules/bolsa/application/simulacionbaremo/testdata/reglas_a.json", import.meta.url), "utf8"));
const catalogo = JSON.parse(await readFile(new URL("../../../../catalogos/baremo-restos-v1.json", import.meta.url), "utf8"));
const ejemplo = { referencia: "experiencia_sintetica_v1", modo: "experiencia", reglas };
const rutaRestos = '["reglas_experiencia",0,"restos","modo"]';
const resultado = () => ({ resultado: { estado: "completado", total: "101667", secciones: [] } });

function montar(editor) {
  const escuchas = new Map(), exportar = { disabled: false }, boton = {}, pista = {};
  let descargas = 0;
  function control(ruta, valor, atributo) {
    return { dataset: { ruta, indice: "0" }, value: valor, validacion: "", atributos: new Set([atributo]),
      hasAttribute(nombre) { return this.atributos.has(nombre); }, setAttribute(nombre) { this.atributos.add(nombre); },
      removeAttribute(nombre) { this.atributos.delete(nombre); }, setCustomValidity(mensaje) { this.validacion = mensaje; },
      checkValidity() { return !this.validacion; }, closest: () => ({ querySelector: () => pista }),
    };
  }
  const restos = control(rutaRestos, editor.estado().borrador.reglas_experiencia[0].restos.modo, "data-restos-modo");
  const puntos = control('["reglas_experiencia",0,"puntos_por_unidad"]', "0.1", "data-puntos");
  const nodos = new Map([['[data-accion="exportar"]', exportar], ['[type="submit"]', boton],
    [".baremo-cabecera [role='status']", {}], [".baremo-resultados", { setAttribute() {} }],
  ]);
  const raiz = {
    addEventListener(nombre, callback) { escuchas.set(nombre, callback); }, querySelector: (selector) => nodos.get(selector),
    querySelectorAll(selector) {
      if (selector === "input[data-minimo-formacion]") return [];
      const disponibles = editor.estado().catalogoRestos ? [restos] : [];
      return selector.startsWith("select") ? disponibles : [puntos, ...disponibles];
    },
  };
  const contexto = { raiz, editor, panel: "bolsa", lecturaId: 0, focoComparacion: true, error: "", ejemplos: [ejemplo],
    aMicropuntos, normalizarMinimoFormacion, errorRestosRegla, leerReglas, MAXIMO_ARCHIVO, Blob,
    t: (clave) => clave, pintar() {}, descartar: () => true,
    URL: { createObjectURL() { descargas++; return "blob:local"; } },
    document: { createElement: () => ({ click() {} }) }, setTimeout() {},
  };
  runInNewContext(`${auxiliares}\n${eventos}`, contexto);
  return { restos, puntos, exportar, contexto, descargas: () => descargas,
    input: (nodo) => escuchas.get("input")({ target: nodo }), change: (nodo) => escuchas.get("change")({ target: nodo }),
    enviar: () => escuchas.get("submit")({ preventDefault() {} }),
    descargar: () => escuchas.get("click")({ target: { closest: (selector) => selector === "[data-panel]" ? null : { dataset: { accion: "exportar" } } } }),
  };
}

test("input del selector no escribe por la ruta genérica; change cancela y descarta la respuesta anterior", async () => {
  let resolver, signal;
  const editor = crearEditorBaremo({ catalogoRestos: catalogo, cliente: { simular: (_s, opciones) => {
    signal = opciones.signal; return new Promise((r) => { resolver = r; });
  } } });
  editor.cargar(ejemplo); const vista = montar(editor), vuelo = editor.comparar();
  vista.restos.value = "descartar_por_regla"; vista.input(vista.restos);
  assert.equal(editor.estado().borrador.reglas_experiencia[0].restos.modo, reglas.reglas_experiencia[0].restos.modo);
  assert.equal(signal.aborted, false);
  await vista.change(vista.restos); assert.equal(signal.aborted, true);
  assert.equal(vista.contexto.lecturaId, 1); assert.equal(vista.contexto.focoComparacion, false);
  resolver(resultado()); await vuelo; assert.equal(editor.estado().comparacion, null);
});

test("un borrador incompatible importado sigue cerrado después de editar puntos", async () => {
  for (const modo of ["futuro", "acumular_por_regla"]) {
    let llamadas = 0;
    const editor = crearEditorBaremo({ catalogoRestos: catalogo, cliente: { simular: async () => { llamadas++; return resultado(); } } });
    const importadas = structuredClone(reglas); importadas.reglas_experiencia[0].restos.modo = modo;
    if (modo === "acumular_por_regla") importadas.reglas_experiencia[0].redondeo.momento = "periodo";
    editor.cargar(ejemplo, importadas); const vista = montar(editor);
    vista.puntos.value = "0.2"; vista.input(vista.puntos);
    assert.equal(vista.exportar.disabled, true);
    vista.enviar(); vista.descargar(); await editor.comparar();
    assert.equal(llamadas, 0); assert.equal(vista.descargas(), 0);
    assert.equal(editor.estado().borrador.reglas_experiencia[0].restos.modo, modo);
    assert.ok(vista.restos.validacion.startsWith("restos_"));
  }
});

test("una selección inválida no se borra al editar puntos y admite corrección posterior", async () => {
  let llamadas = 0;
  const editor = crearEditorBaremo({ catalogoRestos: catalogo, cliente: { simular: async () => { llamadas++; return resultado(); } } });
  editor.cargar(ejemplo); const vista = montar(editor);
  vista.restos.value = "futuro"; await vista.change(vista.restos);
  vista.input(vista.puntos); assert.equal(vista.exportar.disabled, true);
  assert.equal(editor.estado().invalidos[rutaRestos], "futuro");
  vista.enviar(); vista.descargar(); await editor.comparar(); assert.equal(llamadas, 0); assert.equal(vista.descargas(), 0);
  vista.restos.value = "descartar_por_regla"; await vista.change(vista.restos); vista.input(vista.puntos);
  assert.equal(vista.exportar.disabled, false); assert.deepEqual(editor.estado().invalidos, {});
  await editor.comparar(); assert.equal(llamadas, 2);
});

test("cambiar Restos mientras se lee un archivo impide que una importación tardía lo sustituya", async () => {
  let resolver;
  const editor = crearEditorBaremo({ catalogoRestos: catalogo, cliente: {} }); editor.cargar(ejemplo); const vista = montar(editor);
  const importacion = vista.change({ name: "archivo", files: [{ size: 100, text: () => new Promise((r) => { resolver = r; }) }], hasAttribute: () => false });
  vista.restos.value = "descartar_por_regla"; await vista.change(vista.restos);
  resolver(JSON.stringify(reglas)); await importacion;
  assert.equal(editor.estado().borrador.reglas_experiencia[0].restos.modo, "descartar_por_regla");
});

test("sin catálogo Restos el montaje conserva comparación y exportación del consumidor anterior", async () => {
  let llamadas = 0;
  const editor = crearEditorBaremo({ cliente: { simular: async () => { llamadas++; return resultado(); } } });
  editor.cargar(ejemplo); const vista = montar(editor); vista.enviar();
  await new Promise((r) => setImmediate(r)); assert.equal(llamadas, 2);
  vista.descargar(); assert.equal(vista.descargas(), 1);
  assert.deepEqual(JSON.parse(editor.exportar()), reglas);
});
