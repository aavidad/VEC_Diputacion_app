import test from "node:test";
import assert from "node:assert/strict";
import { raizPrueba, montar, seleccion } from "./formulario-llamamiento-pruebas.js?v=20261008-alta-rpt-circular-v6";
import { mensajeValidacionPortal } from "../../portal-idioma.js?v=20260930-portales-i18n-integracion-v1";

function escenario({ radio = false, radioExtra = false, anterior = null } = {}) {
  const raiz = raizPrueba(), nodos = new Map();
  let peticiones = 0, confirmaciones = 0;
  const crearNodo = () => ({ children: [], hidden: true, textContent: "", dataset: {},
    append(nodo) { this.children.push(nodo); }, replaceChildren() { this.children = []; },
    closest(selector) { return selector === "[data-ct-llamamiento-error-enlace]" ? this : null; },
  });
  raiz.ownerDocument = { getElementById: (id) => nodos.get(id), createElement: crearNodo };
  const cerrar = montar(raiz, { seleccionarLlamamiento: async () => { peticiones += 1; } }, {
    confirmarOperacion: () => { confirmaciones += 1; return true; },
  });
  const formulario = raiz.preparar("seleccion", seleccion());
  const nombres = radio ? ["respuesta", "respuesta"] : ["expediente_ref", "version_esperada", "clave_idempotencia"];
  if (radioExtra) nombres.push("respuesta", "respuesta");
  const controles = nombres.map((nombre, indice) => {
    const atributos = new Map([["aria-describedby", `pista-${indice}`]]);
    if (anterior !== null) atributos.set("aria-invalid", anterior);
    const control = { name: nombre, id: `ct-llamamiento-seleccion-${nombre}${(radio || radioExtra) && nombre === "respuesta" ? `-${indice}` : ""}`,
      type: (radio || radioExtra) && nombre === "respuesta" ? "radio" : nombre === "version_esperada" ? "number" : "text",
      value: seleccion()[nombre] ?? "", willValidate: true, fallo: {},
      labels: [{ textContent: nombre }],
      get validity() { return { valid: Object.keys(this.fallo).length === 0, ...this.fallo }; },
      getAttribute: (nombre) => atributos.get(nombre) ?? null,
      setAttribute: (nombre, valor) => atributos.set(nombre, valor),
      removeAttribute: (nombre) => atributos.delete(nombre),
      focus() { raiz.foco.push(this.id); }, scrollIntoView() {},
      closest(selector) { return selector === "[data-ct-llamamiento-form]" ? formulario
        : selector === "fieldset" ? { querySelector: () => ({ textContent: "Respuesta" }) } : null; },
    };
    nodos.set(control.id, control);
    const idError = `ct-llamamiento-seleccion-${nombre}-error`;
    nodos.set(idError, { ...crearNodo(), id: idError });
    return control;
  });
  controles.namedItem = (nombre) => controles.find((control) => control.name === nombre);
  formulario.elements = controles;
  formulario.contains = (nodo) => [...nodos.values()].includes(nodo);
  formulario.reportValidity = formulario.checkValidity = () => { throw new Error("invalid global no permitido"); };
  const lista = crearNodo(), resumen = { ...crearNodo(), querySelector: () => lista };
  formulario.querySelector = (selector) => selector === "[data-ct-llamamiento-errores]" ? resumen : null;
  formulario.querySelectorAll = () => [...nodos.entries()].filter(([id]) => id.endsWith("-error")).map(([, nodo]) => nodo);
  return { raiz, controles, cerrar, lista, resumen, error: (control) => nodos.get(`ct-llamamiento-seleccion-${control.name}-error`),
    enviar: () => raiz.enviar("seleccion"),
    evento: (tipo, control) => raiz.eventos.get(tipo)({ target: control, type: tipo }),
    efectos: () => ({ peticiones, confirmaciones }),
  };
}

test("obligatorios: identifica los campos, enlaza el resumen y enfoca el primero sin confirmar ni enviar", async () => {
  const e = escenario();
  try {
    for (const control of e.controles.slice(0, 2)) { control.value = ""; control.fallo = { valueMissing: true }; }
    await e.enviar();
    assert.deepEqual(e.efectos(), { peticiones: 0, confirmaciones: 0 });
    assert.equal(e.raiz.foco.at(-1), e.controles[0].id);
    assert.equal(e.resumen.hidden, false);
    assert.equal(e.lista.children.length, 2);
    for (const control of e.controles.slice(0, 2)) {
      assert.equal(control.getAttribute("aria-invalid"), "true");
      assert.equal(e.error(control).textContent, mensajeValidacionPortal(control));
      assert.equal(e.error(control).hidden, false);
      assert.match(control.getAttribute("aria-describedby"), /pista-\d ct-llamamiento-seleccion-.+-error/u);
    }
    const enlace = e.lista.children[0].children[0];
    e.evento("focusout", e.controles[0]);
    assert.equal(e.lista.children[0].children[0], enlace);
    let prevenido = false;
    e.raiz.eventos.get("click")({ target: enlace, preventDefault() { prevenido = true; } });
    assert.equal(prevenido, true);
    assert.equal(e.raiz.foco.at(-1), e.controles[0].id);
    assert.equal(e.controles[2].getAttribute("aria-invalid"), null);
  } finally { e.cerrar(); }
});

test("corregir retira sólo las marcas propias y conserva la descripción anterior", async () => {
  const e = escenario({ anterior: "false" });
  try {
    const control = e.controles[0];
    control.fallo = { valueMissing: true };
    await e.enviar();
    control.fallo = {};
    e.evento("input", control);
    assert.equal(control.getAttribute("aria-invalid"), "false");
    assert.equal(control.getAttribute("aria-describedby"), "pista-0");
    assert.equal(e.error(control).hidden, true);
    assert.equal(e.resumen.hidden, true);
    assert.equal(e.lista.children.length, 0);
    assert.deepEqual(e.efectos(), { peticiones: 0, confirmaciones: 0 });
  } finally { e.cerrar(); }
});

test("salir del campo valida ese campo sin adelantar errores de otros ni mover el foco", () => {
  const e = escenario();
  try {
    for (const control of e.controles) control.fallo = { valueMissing: true };
    e.raiz.eventos.get("focusout")({ target: e.controles[0], type: "focusout",
      relatedTarget: { closest: () => ({ type: "submit" }) } });
    assert.equal(e.lista.children.length, 0);
    e.evento("focusout", e.controles[0]);
    assert.equal(e.lista.children.length, 1);
    assert.equal(e.controles[0].getAttribute("aria-invalid"), "true");
    assert.equal(e.controles[1].getAttribute("aria-invalid"), null);
    assert.equal(e.raiz.foco.length, 0);
  } finally { e.cerrar(); }
});

test("un número nativamente inválido detiene el envío antes del contrato", async () => {
  const e = escenario();
  try {
    e.controles[1].fallo = { badInput: true };
    await e.enviar();
    assert.equal(e.error(e.controles[1]).textContent, mensajeValidacionPortal(e.controles[1]));
    assert.equal(e.raiz.foco.at(-1), e.controles[1].id);
    assert.deepEqual(e.efectos(), { peticiones: 0, confirmaciones: 0 });
  } finally { e.cerrar(); }
});

test("grupo radio comparte un error y conserva la descripción de cada opción al corregir", () => {
  const e = escenario({ radio: true });
  try {
    for (const control of e.controles) control.fallo = { valueMissing: true };
    e.raiz.eventos.get("focusout")({ target: e.controles[0], type: "focusout",
      relatedTarget: { closest: () => ({ type: "submit" }) } });
    assert.equal(e.lista.children.length, 0);
    e.evento("focusout", e.controles[0]);
    assert.equal(e.lista.children.length, 1);
    for (const control of e.controles) assert.equal(control.getAttribute("aria-invalid"), "true");
    for (const control of e.controles) control.fallo = {};
    e.evento("input", e.controles[1]);
    for (const [indice, control] of e.controles.entries()) {
      assert.equal(control.getAttribute("aria-invalid"), null);
      assert.equal(control.getAttribute("aria-describedby"), `pista-${indice}`);
    }
    assert.equal(e.resumen.hidden, true);
  } finally { e.cerrar(); }
});

test("desmontar y repintar conservan las marcas ajenas y retiran escuchadores propios", async () => {
  const e = escenario({ anterior: "grammar" });
  e.controles[0].fallo = { valueMissing: true };
  await e.enviar();
  e.cerrar.revisarPlazo();
  assert.equal(e.controles[0].getAttribute("aria-invalid"), "grammar");
  assert.equal(e.controles[0].getAttribute("aria-describedby"), "pista-0");
  e.controles[0].fallo = { valueMissing: true };
  e.evento("focusout", e.controles[0]);
  e.controles[0].setAttribute("aria-invalid", "spelling");
  e.controles[0].setAttribute("aria-describedby", e.controles[0].getAttribute("aria-describedby") + " otra-pista");
  e.cerrar();
  assert.equal(e.controles[0].getAttribute("aria-invalid"), "spelling");
  assert.equal(e.controles[0].getAttribute("aria-describedby"), "pista-0 otra-pista");
  assert.equal(e.raiz.eventos.has("focusout"), false);
  assert.equal(e.raiz.eventos.has("input"), false);
});


test("radio vacío y una elección válida permiten el primer intento sin disparar invalid global", async () => {
  const e = escenario({ radioExtra: true });
  try {
    const radios = e.controles.filter((control) => control.type === "radio");
    for (const radio of radios) {
      Object.defineProperty(radio, "validity", { get: () => ({
        valid: radios.some((control) => control.checked === true),
        valueMissing: !radios.some((control) => control.checked === true),
      }) });
      radio.setCustomValidity = () => { throw new Error("customValidity ajeno no permitido"); };
    }
    await e.enviar();
    assert.deepEqual(e.efectos(), { peticiones: 0, confirmaciones: 0 });
    assert.equal(e.lista.children.length, 1);
    radios[0].checked = true;
    e.evento("input", radios[0]);
    assert.equal(e.resumen.hidden, true);
    for (const radio of radios) assert.equal(radio.getAttribute("aria-invalid"), null);
    await e.enviar();
    assert.deepEqual(e.efectos(), { peticiones: 1, confirmaciones: 1 });
  } finally { e.cerrar(); }
});
