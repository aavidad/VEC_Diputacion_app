import test from "node:test";
import assert from "node:assert/strict";
import { crearControladorTema, ErrorTemaVec, TEMAS_VEC, validarEstadoTema } from "./tema-vec.js";

function elemento() {
  const atributos = new Map();
  const dataset = new Proxy({}, {
    get: (_, clave) => atributos.get(`data-${clave}`),
    set: (_, clave, valor) => { atributos.set(`data-${clave}`, String(valor)); return true; },
  });
  return {
    dataset,
    getAttribute: (nombre) => atributos.get(nombre) ?? null,
    removeAttribute: (nombre) => atributos.delete(nombre),
  };
}

function documento() {
  return { documentElement: elemento(), body: elemento() };
}

test("el catálogo acepta solo identificador y revisión incluidos en el cliente", () => {
  assert.deepEqual(Object.keys(TEMAS_VEC), ["institucional", "granate"]);
  assert.equal(validarEstadoTema({ tema_id: "granate", revision: 1 }), TEMAS_VEC.granate);
  for (const estado of [
    { tema_id: "desconocido", revision: 1 },
    { tema_id: "__proto__", revision: 1 },
  ]) {
    assert.throws(() => validarEstadoTema(estado), (error) => error instanceof ErrorTemaVec && error.codigo === "tema_desconocido");
  }
  for (const revision of [0, 2, "1", 1.1, null]) {
    assert.throws(() => validarEstadoTema({ tema_id: "granate", revision }), (error) => error.codigo === "revision_no_soportada");
  }
  assert.throws(() => validarEstadoTema(null), (error) => error.codigo === "estado_invalido");
});

test("la vista previa se cancela y restaura la ausencia inicial del atributo", () => {
  const d = documento();
  const tema = crearControladorTema({ documento: d });
  assert.equal(tema.leerEstado().estado_servidor, null);
  assert.equal(tema.leerEstado().tema_id, "institucional");
  tema.previsualizar({ tema_id: "granate", revision: 1 });
  assert.equal(d.documentElement.getAttribute("data-tema"), "granate");
  assert.equal(tema.leerEstado().previsualizacion, true);
  tema.cancelarPrevisualizacion();
  assert.equal(d.documentElement.getAttribute("data-tema"), null);
  assert.equal(tema.leerEstado().previsualizacion, false);
  assert.equal(tema.leerEstado().estado_servidor, null);
});

test("un atributo inicial válido informa del tema visible sin atribuirlo al servidor", () => {
  const d = documento();
  d.documentElement.dataset.tema = "granate";
  const tema = crearControladorTema({ documento: d });
  assert.equal(tema.leerEstado().tema_id, "granate");
  assert.equal(tema.leerEstado().revision, 1);
  assert.equal(tema.leerEstado().estado_servidor, null);
  tema.previsualizar({ tema_id: "institucional", revision: 1 });
  assert.equal(tema.leerEstado().tema_id, "institucional");
  tema.cancelarPrevisualizacion();
  assert.equal(tema.leerEstado().tema_id, "granate");
  assert.equal(d.documentElement.getAttribute("data-tema"), "granate");
});

test("varias previsualizaciones conservan el estado servidor y su revisión", () => {
  const d = documento();
  const tema = crearControladorTema({ documento: d });
  tema.aplicarEstadoServidor({ tema_id: "granate", revision: 1 });
  tema.previsualizar({ tema_id: "institucional", revision: 1 });
  tema.previsualizar({ tema_id: "granate", revision: 1 });
  tema.previsualizar({ tema_id: "institucional", revision: 1 });
  assert.deepEqual(tema.leerEstado().estado_servidor, TEMAS_VEC.granate);
  tema.cancelarPrevisualizacion();
  assert.equal(d.documentElement.getAttribute("data-tema"), "granate");
  assert.equal(tema.leerEstado().revision, 1);
  assert.equal(tema.leerEstado().previsualizacion, false);
});

test("una respuesta nueva validada sustituye una vista previa; una inválida no cambia el DOM", () => {
  const d = documento();
  const tema = crearControladorTema({ documento: d });
  tema.aplicarEstadoServidor({ tema_id: "institucional", revision: 1 });
  tema.previsualizar({ tema_id: "granate", revision: 1 });
  assert.throws(() => tema.aplicarEstadoServidor({ tema_id: "granate", revision: 2 }), (error) => error.codigo === "revision_no_soportada");
  assert.equal(d.documentElement.getAttribute("data-tema"), "granate");
  assert.equal(tema.leerEstado().previsualizacion, true);
  tema.aplicarEstadoServidor({ tema_id: "institucional", revision: 1 });
  assert.equal(tema.leerEstado().previsualizacion, false);
  assert.equal(d.documentElement.getAttribute("data-tema"), "institucional");
  tema.cancelarPrevisualizacion();
  assert.equal(d.documentElement.getAttribute("data-tema"), "institucional");
});

test("alto contraste es independiente de la paleta y conserva el ajuste previo", () => {
  const d = documento();
  d.body.dataset.contraste = "true";
  const tema = crearControladorTema({ documento: d });
  tema.aplicarEstadoServidor({ tema_id: "institucional", revision: 1 });
  tema.previsualizar({ tema_id: "granate", revision: 1 });
  tema.cancelarPrevisualizacion();
  assert.equal(d.body.dataset.contraste, "true");
  assert.equal(tema.leerEstado().alto_contraste, true);
  tema.establecerAltoContraste(false);
  assert.equal(d.body.dataset.contraste, "false");
  assert.equal(d.documentElement.getAttribute("data-tema"), "institucional");
  assert.throws(() => tema.establecerAltoContraste("true"), (error) => error.codigo === "contraste_invalido");
});

test("un tema inicial ajeno al catálogo falla de forma explícita", () => {
  const d = documento();
  d.documentElement.dataset.tema = "remoto";
  assert.throws(() => crearControladorTema({ documento: d }), (error) => error.codigo === "tema_desconocido");
});
