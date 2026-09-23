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
  focus() {
    this.ownerDocument.activeElement = this;
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
  assert.match(textoVisible(contenedor), /rcd_1234567890123456789012/u);
  assert.match(textoVisible(contenedor), /2026-09-20T10:00:00Z/u);
  assert.match(
    contenedor.querySelector("[data-dietas-borradores-estado]").textContent,
    /Detalle/u,
  );
  vista.desmontar();
});

test("conserva el formulario y sus datos al recuperar la lista y la ficha", async () => {
  const contenedor = raiz();
  const vista = montarVistaBorradoresPropios(contenedor, {
    cliente: {
      listar: async () => ({ items: [item] }),
      obtener: async () => item,
      crear: async () => item,
    },
  });
  await Promise.resolve();
  await Promise.resolve();
  const form = contenedor.querySelector("[data-dietas-borrador-form]");
  const motivo = form.querySelectorAll("input").find((campo) => campo.name === "motivo");
  motivo.value = "Visita conservada";
  const boton = contenedor.querySelector("[data-dietas-borrador-detalle]");
  await contenedor.querySelector("[data-dietas-borradores-propios]").listeners.click({ target: boton });
  assert.equal(contenedor.querySelector("[data-dietas-borrador-form]"), form);
  assert.equal(motivo.value, "Visita conservada");
  vista.desmontar();
});

test("pagina con el cursor real del cliente y vuelve a la primera página", async () => {
  const contenedor = raiz();
  const consultas = [];
  const vista = montarVistaBorradoresPropios(contenedor, {
    cliente: {
      listar: async (consulta) => {
        consultas.push(consulta);
        return consulta.cursor
          ? { items: [{ ...item, comision: { ...item.comision, motivo: "Segunda página" } }] }
          : { items: [item], siguiente_cursor: "cursor-servidor" };
      },
      obtener: async () => item,
      crear: async () => item,
    },
  });
  await Promise.resolve();
  await Promise.resolve();
  const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
  await panel.listeners.click({ target: contenedor.querySelector('[data-dietas-borrador-pagina="siguiente"]') });
  assert.deepEqual(consultas.at(-1), { limit: 6, cursor: "cursor-servidor" });
  assert.match(textoVisible(contenedor), /Segunda página/u);
  await panel.listeners.click({ target: contenedor.querySelector('[data-dietas-borrador-pagina="anterior"]') });
  assert.deepEqual(consultas.at(-1), { limit: 6 });
  vista.desmontar();
});

test("una denegación de lista no se presenta como lista vacía", async () => {
  const contenedor = raiz();
  const vista = montarVistaBorradoresPropios(contenedor, {
    cliente: {
      listar: async () => { const error = new Error(); error.codigo = "acceso_denegado"; throw error; },
      obtener: async () => item,
      crear: async () => item,
    },
  });
  await Promise.resolve();
  await Promise.resolve();
  assert.match(textoVisible(contenedor), /No tiene permiso/u);
  assert.equal(contenedor.querySelector("[data-dietas-borradores-vacio]"), null);
  vista.desmontar();
});

test("permite reintentar el GET fallido sin ejecutar POST", async () => {
  const contenedor = raiz();
  let lecturas = 0;
  let escrituras = 0;
  const vista = montarVistaBorradoresPropios(contenedor, {
    cliente: {
      listar: async () => {
        lecturas += 1;
        if (lecturas === 1) throw new Error("red");
        return { items: [item] };
      },
      obtener: async () => item,
      crear: async () => { escrituras += 1; return item; },
    },
  });
  await Promise.resolve();
  await Promise.resolve();
  const boton = contenedor.querySelector("[data-dietas-borrador-recargar]");
  assert.ok(boton);
  await contenedor.querySelector("[data-dietas-borradores-propios]").listeners.click({ target: boton });
  assert.equal(lecturas, 2);
  assert.equal(escrituras, 0);
  assert.match(textoVisible(contenedor), /Reunión/u);
  assert.equal(contenedor.querySelector("[data-dietas-borradores-estado]").dataset.tono, "informacion");
  vista.desmontar();
});

test("un POST incierto reintenta el mismo material y la misma clave sin inventar recibo", async () => {
  const contenedor = raiz();
  const solicitudes = [];
  const vista = montarVistaBorradoresPropios(contenedor, {
    generarClaveIdempotencia: () => "operacion-estable",
    cliente: {
      listar: async () => ({ items: [] }),
      obtener: async () => item,
      crear: async (solicitud) => {
        solicitudes.push(solicitud);
        if (solicitudes.length === 1) {
          const error = new Error();
          error.resultadoIndeterminado = true;
          throw error;
        }
        return item;
      },
    },
  });
  await Promise.resolve();
  await Promise.resolve();
  const form = contenedor.querySelector("[data-dietas-borrador-form]");
  form.checkValidity = () => true;
  const datos = {
    fecha_inicio: "2026-09-20", fecha_fin: "2026-09-21", motivo: "Visita",
    hora_inicio: "09:00", hora_fin: "18:00", origen_codigo: "18087", destino_codigo: "18003",
  };
  const FormDataOriginal = globalThis.FormData;
  globalThis.FormData = class { get(nombre) { return datos[nombre] ?? null; } };
  try {
    const raizVista = contenedor.querySelector("[data-dietas-borradores-propios]");
    await raizVista.listeners.submit({ target: form, preventDefault() {} });
    assert.equal(contenedor.querySelector("[data-dietas-borrador-recibo]"), null);
    assert.match(textoVisible(contenedor), /No se ha podido confirmar/u);
    await raizVista.listeners.submit({ target: form, preventDefault() {} });
    assert.deepEqual(solicitudes[0], solicitudes[1]);
    assert.equal(solicitudes[1].clave_idempotencia, "operacion-estable");
    assert.match(textoVisible(contenedor), /rcd_1234567890123456789012/u);
    assert.equal(contenedor.querySelector("[data-dietas-borradores-estado]").dataset.tono, "exito");
  } finally {
    globalThis.FormData = FormDataOriginal;
    vista.desmontar();
  }
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

test("conserva el recibo si el GET posterior al alta resulta denegado y no repite el POST", async () => {
  const contenedor = raiz();
  let lecturas = 0;
  const solicitudes = [];
  const vista = montarVistaBorradoresPropios(contenedor, {
    generarClaveIdempotencia: () => "operacion-post-confirmado",
    cliente: {
      listar: async () => {
        lecturas += 1;
        if (lecturas === 1) return { items: [] };
        const error = new Error("denegado");
        error.codigo = "acceso_denegado";
        throw error;
      },
      obtener: async () => item,
      crear: async (solicitud) => { solicitudes.push(solicitud); return item; },
    },
  });
  await Promise.resolve();
  await Promise.resolve();
  const form = contenedor.querySelector("[data-dietas-borrador-form]");
  form.checkValidity = () => true;
  const datos = {
    fecha_inicio: "2026-09-20", fecha_fin: "2026-09-21", motivo: "Visita",
    hora_inicio: "09:00", hora_fin: "18:00", origen_codigo: "18087", destino_codigo: "18003",
  };
  const FormDataOriginal = globalThis.FormData;
  globalThis.FormData = class { get(nombre) { return datos[nombre] ?? null; } };
  try {
    const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
    await panel.listeners.submit({ target: form, preventDefault() {} });
    assert.equal(lecturas, 2);
    assert.equal(solicitudes.length, 1);
    assert.match(textoVisible(contenedor), /rcd_1234567890123456789012/u);
    assert.match(textoVisible(contenedor), /Borrador registrado\. No tiene permiso para actualizar/u);
    await panel.listeners.submit({ target: form, preventDefault() {} });
    assert.equal(solicitudes.length, 1);
    assert.match(textoVisible(contenedor), /Este borrador ya se registró/u);
    assert.match(textoVisible(contenedor), /rcd_1234567890123456789012/u);
  } finally {
    globalThis.FormData = FormDataOriginal;
    vista.desmontar();
  }
});

test("un GET de detalle fallido mantiene visible el recibo anterior", async () => {
  const contenedor = raiz();
  const vista = montarVistaBorradoresPropios(contenedor, {
    cliente: {
      listar: async () => ({ items: [item] }),
      obtener: async () => { throw new Error("red"); },
      crear: async () => item,
    },
  });
  await Promise.resolve();
  await Promise.resolve();
  const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
  // El detalle puede ser el recibo de un alta, además de una ficha consultada.
  const form = contenedor.querySelector("[data-dietas-borrador-form]");
  form.checkValidity = () => true;
  const datos = {
    fecha_inicio: "2026-09-20", fecha_fin: "2026-09-21", motivo: "Visita",
    hora_inicio: "09:00", hora_fin: "18:00", origen_codigo: "18087", destino_codigo: "18003",
  };
  const FormDataOriginal = globalThis.FormData;
  globalThis.FormData = class { get(nombre) { return datos[nombre] ?? null; } };
  try {
    await panel.listeners.submit({ target: form, preventDefault() {} });
    const boton = contenedor.querySelector("[data-dietas-borrador-detalle]");
    await panel.listeners.click({ target: boton });
    assert.match(textoVisible(contenedor), /Se conserva el último recibo obtenido/u);
    assert.match(textoVisible(contenedor), /rcd_1234567890123456789012/u);
  } finally {
    globalThis.FormData = FormDataOriginal;
    vista.desmontar();
  }
});

test("permite mostrar y cerrar el formulario desde Mis comisiones sin desmontar la lista", async () => {
  const contenedor = raiz();
  const vista = montarVistaBorradoresPropios(contenedor, {
    formularioInicialmenteVisible: false,
    cliente: {
      listar: async () => ({ items: [item] }),
      obtener: async () => item,
      crear: async () => item,
    },
  });
  await Promise.resolve();
  await Promise.resolve();
  const form = contenedor.querySelector("[data-dietas-borrador-form]");
  assert.equal(form.hidden, true);
  assert.match(textoVisible(contenedor), /Borradores registrados/u);
  assert.ok(contenedor.querySelector("[data-dietas-borrador-detalle]"));
  assert.equal(vista.abrirFormulario(), true);
  assert.equal(form.hidden, false);
  assert.equal(contenedor.ownerDocument.activeElement.tagName, "input");
  assert.equal(vista.cerrarFormulario(), true);
  assert.equal(form.hidden, true);
  vista.desmontar();
  assert.equal(vista.abrirFormulario(), false);
});

test("mantiene el foco navegable tras reemplazar la lista y muestra la ficha en castellano", async () => {
  const contenedor = raiz();
  const comisionConRuta = { ...item, comision: { ...item.comision, codigos_ruta: ["18087", "18003"] } };
  const vista = montarVistaBorradoresPropios(contenedor, {
    formularioInicialmenteVisible: false,
    cliente: {
      listar: async () => ({ items: [comisionConRuta] }),
      obtener: async () => comisionConRuta,
      crear: async () => item,
    },
  });
  await Promise.resolve();
  await Promise.resolve();
  const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
  const boton = contenedor.querySelector("[data-dietas-borrador-detalle]");
  await panel.listeners.click({ target: boton });
  assert.equal(contenedor.ownerDocument.activeElement.dataset.dietasBorradorFicha, "");
  assert.match(textoVisible(contenedor), /Granada → Albolote/u);
  const reintento = vista.recargar();
  await reintento;
  assert.match(textoVisible(contenedor), /rcd_1234567890123456789012/u);
  vista.desmontar();
});

test("rechaza fechas y horas incoherentes antes del POST y enfoca el recibo exacto tras confirmar", async () => {
  const contenedor = raiz();
  let escrituras = 0;
  const itemExacto = { ...item, recibo: { ...item.recibo, registrado_en: "2026-09-20T10:00:00.123456Z" } };
  const vista = montarVistaBorradoresPropios(contenedor, {
    cliente: {
      listar: async () => ({ items: [] }),
      obtener: async () => item,
      crear: async () => { escrituras += 1; return itemExacto; },
    },
  });
  await Promise.resolve();
  await Promise.resolve();
  const form = contenedor.querySelector("[data-dietas-borrador-form]");
  form.checkValidity = () => true;
  const datos = {
    fecha_inicio: "2026-09-20", fecha_fin: "2026-09-20", motivo: "Visita",
    hora_inicio: "18:00", hora_fin: "09:00", origen_codigo: "18087", destino_codigo: "18003",
  };
  const FormDataOriginal = globalThis.FormData;
  globalThis.FormData = class { get(nombre) { return datos[nombre] ?? null; } };
  try {
    const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
    await panel.listeners.submit({ target: form, preventDefault() {} });
    assert.equal(escrituras, 0);
    assert.match(textoVisible(contenedor), /El regreso debe ser posterior/u);
    datos.fecha_fin = "2026-09-21";
    await panel.listeners.submit({ target: form, preventDefault() {} });
    assert.equal(escrituras, 1);
    assert.equal(contenedor.ownerDocument.activeElement.dataset.dietasBorradorRecibo, "");
    assert.match(textoVisible(contenedor), /2026-09-20T10:00:00\.123456Z/u);
  } finally {
    globalThis.FormData = FormDataOriginal;
    vista.desmontar();
  }
});

test("distingue carga, vacío y denegación de consulta sin presentar un alta", async () => {
  const contenedor = raiz();
  let resolver;
  const vista = montarVistaBorradoresPropios(contenedor, {
    cliente: {
      listar: () => new Promise((resuelve) => { resolver = resuelve; }),
      obtener: async () => item,
      crear: async () => { throw new Error("POST inesperado"); },
    },
  });
  assert.match(textoVisible(contenedor), /Cargando sus borradores/u);
  assert.equal(contenedor.querySelector("[data-dietas-borradores-vacio]"), null);
  resolver({ items: [] });
  await Promise.resolve();
  await Promise.resolve();
  assert.match(textoVisible(contenedor), /Todavía no hay borradores propios/u);
  vista.desmontar();

  const denegado = raiz();
  const vistaDenegada = montarVistaBorradoresPropios(denegado, {
    cliente: {
      listar: async () => { const error = new Error(); error.codigo = "acceso_denegado"; throw error; },
      obtener: async () => item,
      crear: async () => item,
    },
  });
  await Promise.resolve();
  await Promise.resolve();
  assert.match(textoVisible(denegado), /No tiene permiso para consultar sus borradores/u);
  assert.doesNotMatch(textoVisible(denegado), /No tiene permiso para crear un borrador/u);
  assert.equal(denegado.querySelector("[data-dietas-borradores-vacio]"), null);
  vistaDenegada.desmontar();
});
