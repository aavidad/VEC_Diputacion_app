import assert from "node:assert/strict";
import test from "node:test";

import { montarVistaBorradoresPropios } from "./vista-borradores-propios.js";

const claveDatos = (atributo) =>
  atributo.slice(5).replace(/-([a-z])/g, (_m, letra) => letra.toUpperCase());
class Nodo {
  constructor(documento, etiqueta = "div") {
    this.ownerDocument = documento;
    this.tagName = etiqueta;
    this.children = [];
    this.dataset = {};
    this.attrs = {};
    this.listeners = {};
    this.parent = null;
    this.textContent = "";
    this.disabled = false;
  }
  append(...nodos) {
    this.children.push(...nodos);
    nodos.forEach((nodo) => {
      nodo.parent = this;
    });
  }
  replaceChildren(...nodos) {
    this.children = [];
    this.append(...nodos);
  }
  removeChild(nodo) {
    this.children = this.children.filter((hijo) => hijo !== nodo);
    nodo.parent = null;
  }
  remove() {
    this.parent?.removeChild(this);
  }
  addEventListener(tipo, listener) {
    this.listeners[tipo] = listener;
  }
  removeEventListener(tipo) {
    delete this.listeners[tipo];
  }
  setAttribute(nombre, valor) {
    this.attrs[nombre] = String(valor);
  }
  matches(selector) {
    if (!selector.startsWith("[")) return this.tagName === selector;
    const coincidencia = selector.match(/^\[([^=\]]+)(?:="([^"]*)")?\]$/u);
    const valor = this.dataset[claveDatos(coincidencia[1])];
    return (
      valor !== undefined &&
      (coincidencia[2] === undefined || valor === coincidencia[2])
    );
  }
  closest(selector) {
    for (let actual = this; actual; actual = actual.parent)
      if (actual.matches(selector)) return actual;
    return null;
  }
  querySelector(selector) {
    return this.querySelectorAll(selector)[0] || null;
  }
  querySelectorAll(selector) {
    const salida = [];
    const visitar = (actual) => {
      if (actual.matches(selector)) salida.push(actual);
      actual.children.forEach(visitar);
    };
    visitar(this);
    return salida;
  }
}
function raiz() {
  const documento = {
    createElement: (etiqueta) => new Nodo(documento, etiqueta),
  };
  return new Nodo(documento, "root");
}
function textoVisible(nodo) {
  return [nodo.textContent, ...nodo.children.map(textoVisible)].join(" ");
}
const item = Object.freeze({
  comision: {
    referencia: "dco_1234567890123456789012",
    estado: "borrador",
    fecha_inicio: "2026-09-20",
    fecha_fin: "2026-09-21",
    motivo: "Reunión",
    codigos_ruta: [],
    relacion_ref: "rel_1234567890123456789012",
  },
  recibo: {
    referencia: "rcd_1234567890123456789012",
    version: 1,
    registrado_en: "2026-09-20T10:00:00Z",
    repeticion: false,
  },
});

test("carga la colección propia real y recupera su detalle mediante el cliente inyectado", async () => {
  const contenedor = raiz();
  const llamadas = [];
  const vista = montarVistaBorradoresPropios(contenedor, {
    cliente: {
      listar: async (consulta) => {
        llamadas.push(["listar", consulta]);
        return { items: [item] };
      },
      obtener: async (referencia) => {
        llamadas.push(["obtener", referencia]);
        return item;
      },
      crear: async () => item,
    },
  });
  await Promise.resolve();
  await Promise.resolve();
  const boton = contenedor.querySelector(
    '[data-dietas-borrador-detalle="dco_1234567890123456789012"]',
  );
  assert.equal(llamadas[0][0], "listar");
  assert.ok(boton);
  await contenedor
    .querySelector("[data-dietas-borradores-propios]")
    .listeners.click({ target: boton });
  assert.deepEqual(llamadas[1], ["obtener", "dco_1234567890123456789012"]);
  assert.match(
    contenedor.querySelector("[data-dietas-borradores-estado]").textContent,
    /Detalle/u,
  );
  vista.desmontar();
});

test("no escoge la primera relación cuando la composición aporta varias autorizadas", async () => {
  const contenedor = raiz();
  let listas = 0;
  const vista = montarVistaBorradoresPropios(contenedor, {
    relacionesAutorizadas: [
      "rel_1234567890123456789012",
      "rel_abcdefghijklmnopqrstuv",
    ],
    cliente: {
      listar: async () => { listas += 1; return { items: [] }; },
      obtener: async () => item,
      crear: async () => item,
    },
  });
  await Promise.resolve();
  assert.equal(listas, 0);
  const opciones = contenedor.querySelectorAll("option");
  assert.deepEqual(opciones.map((opcion) => opcion.value), ["", "rel_1234567890123456789012", "rel_abcdefghijklmnopqrstuv"]);
  vista.desmontar();
});

test("ante resultado incierto conserva una salida comprensible sin fabricar recibo", async () => {
  const contenedor = raiz();
  const avisos = [];
  const vista = montarVistaBorradoresPropios(contenedor, {
    cliente: {
      listar: async () => {
        const error = new Error();
        error.resultadoIndeterminado = true;
        throw error;
      },
      obtener: async () => item,
      crear: async () => item,
    },
    anunciar: (...aviso) => avisos.push(aviso),
  });
  await Promise.resolve();
  await Promise.resolve();
  assert.match(
    contenedor.querySelector("[data-dietas-borradores-estado]").textContent,
    /No se ha podido confirmar/u,
  );
  assert.equal(contenedor.querySelector("[data-dietas-borrador-recibo]"), null);
  assert.equal(avisos.at(-1)[1], "error");
  vista.desmontar();
});

test("reserva la explicación administrativa para la ayuda contextual", () => {
  const contenedor = raiz();
  const vista = montarVistaBorradoresPropios(contenedor);
  assert.doesNotMatch(
    textoVisible(contenedor),
    /Cree un borrador propio|No acredita autorización, liquidación ni pago/u,
  );
  vista.desmontar();
});
