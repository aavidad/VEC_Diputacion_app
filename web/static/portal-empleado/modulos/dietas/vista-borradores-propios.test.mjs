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
  const selectorRelacion = contenedor.querySelectorAll("select").find((selector) => selector.name === "relacion_ref");
  assert.deepEqual(selectorRelacion.children.map((opcion) => opcion.value), ["", "rel_1234567890123456789012", "rel_abcdefghijklmnopqrstuv"]);
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
test("muestra km, importe y tramos provisionales de la comisión recuperada", async () => {
  const calculado={...item,comision:{...item.comision,codigos_ruta:["18087","18003"],calculo:{
    rotulo:"PROVISIONAL · pendiente de confirmación por RRHH",version_tarifa:"provisional:rd462:20260923",version_grafo:"grafo-sintetico-v1",kilometros:"12.0000",importe_kilometraje_centimos:312,
    tramos_ruta:[{origen_codigo:"18087",destino_codigo:"18003",kilometros:"12.0000"}],
    opciones_dieta:[1,2,3].map((grupo)=>({grupo,calculo:{total_maximo_orientativo_centimos:2500,tramos:[{fecha:"2026-09-20",tipo:"manutencion",porcentaje:50,importe_centimos:2500}]}})),
  }}};
  const contenedor=raiz(); const vista=montarVistaBorradoresPropios(contenedor,{cliente:{listar:async()=>({items:[calculado]}),obtener:async()=>calculado,crear:async()=>calculado}});
  await Promise.resolve(); await Promise.resolve();
  const boton=contenedor.querySelector("[data-dietas-borrador-detalle]");
  await contenedor.querySelector("[data-dietas-borradores-propios]").listeners.click({target:boton});
  const texto=textoVisible(contenedor);
  assert.match(texto,/12 km/u); assert.match(texto,/3,12/u); assert.match(texto,/Granada → Albolote/u); assert.match(texto,/Grupo 3/u);
  vista.desmontar();
});
