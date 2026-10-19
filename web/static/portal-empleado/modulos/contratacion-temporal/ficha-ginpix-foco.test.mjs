import assert from "node:assert/strict";
import test from "node:test";
import { montarFichaGINPIX } from "./ficha-ginpix.js?v=VEC_WEB_VERSION";
import { consultaB2GINPIX } from "./ficha-ginpix-fixture.mjs";

const consulta = consultaB2GINPIX();

function diferida() {
  let resolver, rechazar;
  const promesa = new Promise((si, no) => { resolver = si; rechazar = no; });
  return { promesa, resolver, rechazar };
}

function fichaDOM() {
  const documento = { body: {}, activeElement: null };
  documento.activeElement = documento.body;
  const eventos = new Map();
  const controles = new Map();
  let html = "";
  const nodo = (selector) => ({
    selector, atributos: new Map(), textContent: "",
    matches: (consulta) => consulta === selector,
    setAttribute(clave, valor) { this.atributos.set(clave, valor); },
    getAttribute(clave) { return this.atributos.get(clave); },
    focus() { documento.activeElement = this; },
  });
  const raiz = {
    ownerDocument: documento,
    addEventListener(tipo, escucha) { eventos.set(tipo, escucha); },
    removeEventListener(tipo) { eventos.delete(tipo); },
    replaceChildren() {
      if ([...controles.values()].includes(documento.activeElement)) documento.activeElement = documento.body;
      controles.clear(); html = "";
    },
    querySelector(selector) { return controles.get(selector) ?? null; },
    contains(control) { return [...controles.values()].includes(control); },
    get innerHTML() { return html; },
    set innerHTML(valor) {
      if ([...controles.values()].includes(documento.activeElement)) documento.activeElement = documento.body;
      html = valor; controles.clear();
      for (const selector of ["[data-ct-ficha-ginpix-descargar]", "[data-ct-ficha-ginpix-estado]"]) controles.set(selector, nodo(selector));
    },
  };
  const boton = () => raiz.querySelector("[data-ct-ficha-ginpix-descargar]");
  const estado = () => raiz.querySelector("[data-ct-ficha-ginpix-estado]");
  const pulsar = () => eventos.get("click")({ target: boton() });
  return { raiz, documento, boton, estado, pulsar, eventos };
}

test("teclado: conserva botón enfocado y estado ocupado hasta descargar una sola vez", async () => {
  const dom = fichaDOM(); const pendiente = diferida(); let consultas = 0, archivos = 0;
  montarFichaGINPIX({ raiz: dom.raiz, consulta,
    cliente: { descargarFichaGINPIX: () => { consultas++; return pendiente.promesa; } },
    descargarArchivo: () => { archivos++; },
  });
  const disparador = dom.boton(); disparador.focus();
  const pulsacion = dom.pulsar();
  assert.equal(dom.documento.activeElement, disparador);
  assert.equal(dom.boton(), disparador);
  assert.equal(disparador.getAttribute("aria-disabled"), "true");
  assert.equal(disparador.getAttribute("aria-busy"), "true");
  assert.match(dom.estado().textContent, /Preparando la descarga/u);
  await dom.pulsar(); assert.equal(consultas, 1);
  pendiente.resolver({ contenido: new Uint8Array([123, 125]) }); await pulsacion;
  assert.equal(archivos, 1);
  assert.equal(dom.documento.activeElement, disparador);
  assert.equal(disparador.getAttribute("aria-disabled"), "false");
  assert.equal(disparador.getAttribute("aria-busy"), "false");
  assert.equal(dom.estado().textContent, "");
});

test("teclado: el error mantiene el disparador y no roba foco a otro control", async () => {
  const dom = fichaDOM(); const pendientes = [diferida(), diferida()]; let indice = 0;
  montarFichaGINPIX({ raiz: dom.raiz, consulta,
    cliente: { descargarFichaGINPIX: () => pendientes[indice++].promesa }, descargarArchivo: () => {},
  });
  const disparador = dom.boton(); disparador.focus();
  const primerIntento = dom.pulsar(); pendientes[0].rechazar(new Error("red")); await primerIntento;
  assert.equal(dom.documento.activeElement, disparador);
  assert.match(dom.estado().textContent, /No se ha podido/u);
  assert.equal(disparador.getAttribute("aria-disabled"), "false");
  const segundoIntento = dom.pulsar();
  const controlAjeno = { focus() { dom.documento.activeElement = this; } }; controlAjeno.focus();
  pendientes[1].rechazar(new Error("red")); await segundoIntento;
  assert.equal(dom.documento.activeElement, controlAjeno);
  assert.equal(disparador.getAttribute("aria-busy"), "false");
  assert.match(dom.estado().textContent, /No se ha podido/u);
});

test("desmontaje: una respuesta tardía no actualiza la ficha ni recupera foco", async () => {
  const dom = fichaDOM(); const pendiente = diferida(); let archivos = 0;
  const desmontar = montarFichaGINPIX({ raiz: dom.raiz, consulta,
    cliente: { descargarFichaGINPIX: () => pendiente.promesa },
    descargarArchivo: () => { archivos++; },
  });
  dom.boton().focus(); const pulsacion = dom.pulsar();
  desmontar(); assert.equal(dom.documento.activeElement, dom.documento.body);
  pendiente.resolver({ contenido: new Uint8Array([123, 125]) }); await pulsacion;
  assert.equal(dom.raiz.innerHTML, "");
  assert.equal(dom.documento.activeElement, dom.documento.body);
  assert.equal(archivos, 0);
  assert.equal(dom.eventos.has("click"), false);
});
