import assert from "node:assert/strict";
import test from "node:test";

import { montarVistaBorradoresPropios } from "./vista-borradores-propios.js";
import { crearClienteBorradoresDietasHTTP } from "./cliente-borradores-http.js";

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

test("añade, reordena y quita paradas para revisar y enviar la secuencia completa", async () => {
  const contenedor = raiz();
  const solicitudes = [];
  const vista = montarVistaBorradoresPropios(contenedor, {
    cliente: { listar: async () => ({ items: [] }), obtener: async () => item,
      crear: async (solicitud) => { solicitudes.push(solicitud); return item; } },
    generarClaveIdempotencia: () => "operacion-paradas-20260924",
  });
  await Promise.resolve(); await Promise.resolve();
  const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
  const form = contenedor.querySelector("[data-dietas-borrador-form]");
  form.checkValidity = () => true;
  const datos = { fecha_inicio: "2026-09-20", fecha_fin: "2026-09-21", motivo: "Visita",
    hora_inicio: "09:00", hora_fin: "18:00", origen_codigo: "18087", destino_codigo: "18003" };
  const original = globalThis.FormData;
  globalThis.FormData = class { get(nombre) { return datos[nombre] ?? null; } };
  try {
    await panel.listeners.click({ target: form.querySelector("[data-dietas-parada-anadir]") });
    await panel.listeners.click({ target: form.querySelector("[data-dietas-parada-anadir]") });
    let paradas = form.querySelectorAll("select").filter((nodo) => nodo.name === "parada_codigo");
    assert.equal(paradas.length, 2);
    paradas[0].value = "18175";
    paradas[1].value = "18061";
    await panel.listeners.click({ target: form.querySelectorAll("[data-dietas-parada-subir]")[1] });
    paradas = form.querySelectorAll("select").filter((nodo) => nodo.name === "parada_codigo");
    assert.deepEqual(paradas.map((nodo) => nodo.value), ["18061", "18175"]);
    await panel.listeners.click({ target: form.querySelector("[data-dietas-borrador-revisar]") });
    assert.equal(solicitudes.length, 0);
    assert.match(textoVisible(form.querySelector("[data-dietas-borrador-preparacion]")), /Granada → .* → .* → Albolote/u);
    await panel.listeners.click({ target: form.querySelectorAll("[data-dietas-parada-quitar]")[1] });
    await panel.listeners.submit({ target: form, preventDefault() {} });
    assert.deepEqual(solicitudes[0].codigos_ruta, ["18087", "18061", "18003"]);
  } finally { globalThis.FormData = original; vista.desmontar(); }
});

test("limita a doce localidades y bloquea una parada repetida antes de POST", async () => {
  const contenedor = raiz(); let escrituras = 0;
  const vista = montarVistaBorradoresPropios(contenedor, {
    cliente: { listar: async () => ({ items: [] }), obtener: async () => item,
      crear: async () => { escrituras++; return item; } },
  });
  await Promise.resolve(); await Promise.resolve();
  const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
  const form = contenedor.querySelector("[data-dietas-borrador-form]"); form.checkValidity = () => true;
  const datos = { fecha_inicio: "2026-09-20", fecha_fin: "2026-09-21", motivo: "Visita",
    hora_inicio: "09:00", hora_fin: "18:00", origen_codigo: "18087", destino_codigo: "18003" };
  const original = globalThis.FormData;
  globalThis.FormData = class { get(nombre) { return datos[nombre] ?? null; } };
  try {
    for (let i = 0; i < 11; i++) await panel.listeners.click({ target: form.querySelector("[data-dietas-parada-anadir]") });
    const paradas = form.querySelectorAll("select").filter((nodo) => nodo.name === "parada_codigo");
    assert.equal(paradas.length, 10);
    assert.equal(form.querySelector("[data-dietas-parada-anadir]").disabled, true);
    const disponibles = paradas[0].children.map((opcion) => opcion.value)
      .filter((codigo) => codigo && codigo !== "18087" && codigo !== "18003");
    paradas.forEach((parada, indice) => { parada.value = disponibles[indice]; });
    paradas[0].value = "18087";
    await panel.listeners.submit({ target: form, preventDefault() {} });
    assert.equal(escrituras, 0);
    assert.match(textoVisible(contenedor), /localidades distintas/u);
  } finally { globalThis.FormData = original; vista.desmontar(); }
});

test("un 503 con paradas conserva orden, contenido y clave en el reintento", async () => {
  const contenedor = raiz(); const solicitudes = [];
  const vista = montarVistaBorradoresPropios(contenedor, {
    generarClaveIdempotencia: () => "operacion-paradas-503-20260924",
    cliente: { listar: async () => ({ items: [] }), obtener: async () => item,
      crear: async (solicitud) => {
        solicitudes.push(solicitud);
        if (solicitudes.length === 1) {
          const error = new Error("servicio no disponible"); error.resultadoIndeterminado = true; throw error;
        }
        return item;
      } },
  });
  await Promise.resolve(); await Promise.resolve();
  const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
  const form = contenedor.querySelector("[data-dietas-borrador-form]"); form.checkValidity = () => true;
  const datos = { fecha_inicio: "2026-09-20", fecha_fin: "2026-09-21", motivo: "Visita",
    hora_inicio: "09:00", hora_fin: "18:00", origen_codigo: "18087", destino_codigo: "18003" };
  const original = globalThis.FormData;
  globalThis.FormData = class { get(nombre) { return datos[nombre] ?? null; } };
  try {
    await panel.listeners.click({ target: form.querySelector("[data-dietas-parada-anadir]") });
    form.querySelectorAll("select").find((nodo) => nodo.name === "parada_codigo").value = "18175";
    await panel.listeners.submit({ target: form, preventDefault() {} });
    assert.equal(contenedor.querySelector("[data-dietas-borrador-recibo]"), null);
    await panel.listeners.submit({ target: form, preventDefault() {} });
    assert.deepEqual(solicitudes[0], solicitudes[1]);
    assert.deepEqual(solicitudes[1].codigos_ruta, ["18087", "18175", "18003"]);
    assert.equal(solicitudes[1].clave_idempotencia, "operacion-paradas-503-20260924");
  } finally { globalThis.FormData = original; vista.desmontar(); }
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

test("al guardar otra vez el mismo borrador confirma el recibo y reserva la instrucción para ?", async () => {
  const contenedor = raiz();
  let escrituras = 0;
  const vista = montarVistaBorradoresPropios(contenedor, {
    cliente: {
      listar: async () => ({ items: [] }),
      obtener: async () => item,
      crear: async () => { escrituras += 1; return item; },
    },
    generarClaveIdempotencia: () => "operacion-confirmada",
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
    await panel.listeners.submit({ target: form, preventDefault() {} });
    assert.equal(escrituras, 1);
    assert.equal(contenedor.querySelector("[data-dietas-borradores-estado]").textContent,
      "Este borrador ya se registró.");
    assert.match(textoVisible(contenedor.querySelector("[data-dietas-borrador-recibo]")),
      /rcd_1234567890123456789012/u);
    const ayuda = form.querySelectorAll("details").find((nodo) => nodo.className === "dietas-borradores-ayuda");
    assert.ok(ayuda);
    assert.equal(Boolean(ayuda.open), false);
    assert.equal(ayuda.querySelector("summary").textContent, "?");
    assert.match(textoVisible(ayuda), /Consulte su recibo o cambie los datos para iniciar otro/u);
  } finally {
    globalThis.FormData = FormDataOriginal;
    vista.desmontar();
  }
});

test("la vista conserva la clave si el POST real responde 201 sin recibo válido", async () => {
  const contenedor = raiz();
  const solicitudes = [];
  const confirmado = {
    comision: { ...item.comision, codigos_ruta: ["18087", "18003"] },
    recibo: { ...item.recibo, registrado_en: "2026-09-20T10:00:00.123456Z", repeticion: true },
  };
  const cliente = crearClienteBorradoresDietasHTTP({ fetchImpl: async (ruta, opciones) => {
    if (opciones.method === "GET")
      return new Response(JSON.stringify({ items: [] }), { status: 200, headers: { "Content-Type": "application/json; charset=utf-8" } });
    assert.equal(ruta, "/api/vec/dietas/comisiones");
    solicitudes.push(JSON.parse(opciones.body));
    return solicitudes.length === 1
      ? new Response("no-json", { status: 201, headers: { "Content-Type": "application/json; charset=utf-8" } })
      : new Response(JSON.stringify(confirmado), { status: 200, headers: { "Content-Type": "application/json; charset=utf-8" } });
  } });
  const vista = montarVistaBorradoresPropios(contenedor, {
    cliente,
    generarClaveIdempotencia: () => "operacion-estable-20260924",
  });
  await new Promise((resolver) => setImmediate(resolver));
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
    assert.equal(contenedor.querySelector("[data-dietas-borrador-recibo]"), null);
    assert.match(textoVisible(contenedor), /No se ha podido confirmar/u);
    await panel.listeners.submit({ target: form, preventDefault() {} });
    assert.equal(solicitudes.length, 2);
    assert.deepEqual(solicitudes[0], solicitudes[1]);
    assert.equal(solicitudes[1].clave_idempotencia, "operacion-estable-20260924");
    assert.match(textoVisible(contenedor), /2026-09-20T10:00:00\.123456Z/u);
    assert.match(textoVisible(contenedor), /recuperado sin crear otro borrador/u);
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

test("tras desmontar un POST incierto consulta la lista sin repetirlo y abre solo el recibo elegido", async () => {
  const primerContenedor = raiz();
  const llamadas = { listar: [], obtener: [], crear: [] };
  let lecturas = 0;
  const cliente = {
    listar: async (consulta) => {
      llamadas.listar.push(consulta);
      lecturas += 1;
      return { items: lecturas < 3 ? [] : [item] };
    },
    obtener: async (referencia, opciones) => {
      llamadas.obtener.push([referencia, opciones.relacion_ref]);
      return item;
    },
    crear: async (solicitud) => {
      llamadas.crear.push(solicitud);
      const error = new Error("respuesta incierta");
      error.resultadoIndeterminado = true;
      throw error;
    },
  };
  const primeraVista = montarVistaBorradoresPropios(primerContenedor, {
    cliente, generarClaveIdempotencia: () => "operacion-incierta-20260924",
  });
  await Promise.resolve();
  await Promise.resolve();
  const form = primerContenedor.querySelector("[data-dietas-borrador-form]");
  form.checkValidity = () => true;
  const datos = {
    fecha_inicio: "2026-09-20", fecha_fin: "2026-09-21", motivo: "Visita",
    hora_inicio: "09:00", hora_fin: "18:00", origen_codigo: "18087", destino_codigo: "18003",
  };
  const FormDataOriginal = globalThis.FormData;
  globalThis.FormData = class { get(nombre) { return datos[nombre] ?? null; } };
  try {
    await primerContenedor.querySelector("[data-dietas-borradores-propios]").listeners.submit({ target: form, preventDefault() {} });
  } finally {
    globalThis.FormData = FormDataOriginal;
    primeraVista.desmontar();
  }
  assert.equal(llamadas.crear.length, 1);
  assert.equal(primerContenedor.querySelector("[data-dietas-borradores-propios]"), null);

  const segundoContenedor = raiz();
  const segundaVista = montarVistaBorradoresPropios(segundoContenedor, {
    cliente,
    generarClaveIdempotencia: () => { throw new Error("no debe reconstruirse la clave"); },
  });
  await Promise.resolve();
  await Promise.resolve();
  const panel = segundoContenedor.querySelector("[data-dietas-borradores-propios]");
  const consultar = segundoContenedor.querySelector("[data-dietas-borrador-consultar-registrados]");
  assert.ok(consultar);
  await panel.listeners.click({ target: consultar });
  assert.deepEqual(llamadas.listar.at(-1), { limit: 6 });
  assert.equal(llamadas.crear.length, 1);
  assert.match(segundoContenedor.querySelector("[data-dietas-borradores-estado]").textContent, /La lista no confirma altas anteriores/u);
  assert.doesNotMatch(segundoContenedor.querySelector("[data-dietas-borradores-estado]").textContent, /Elija un borrador|aquel intento/u);
  const ayuda = segundoContenedor.querySelector("[data-dietas-borradores-ayuda]");
  assert.equal(ayuda.open, false);
  assert.match(textoVisible(ayuda), /Elija un borrador para ver su recibo/u);
  assert.equal(ayuda.querySelector("summary").attrs["aria-label"], "? Ayuda");
  const lecturasAntesAyuda = llamadas.listar.length;
  await panel.listeners.click({ target: ayuda.querySelector("summary") });
  assert.equal(llamadas.listar.length, lecturasAntesAyuda);
  assert.equal(segundoContenedor.querySelector("[data-dietas-borrador-recibo]"), null);
  assert.equal(segundoContenedor.ownerDocument.activeElement.dataset.dietasBorradorConsultarRegistrados, "");
  const elegir = segundoContenedor.querySelector("[data-dietas-borrador-detalle]");
  await panel.listeners.click({ target: elegir });
  assert.deepEqual(llamadas.obtener, [[item.comision.referencia, item.comision.relacion_ref]]);
  assert.match(textoVisible(segundoContenedor.querySelector("[data-dietas-borrador-recibo]")), /rcd_1234567890123456789012/u);
  assert.equal(llamadas.crear.length, 1);
  segundaVista.desmontar();
});

test("Consultar borradores reinicia el cursor y distingue lista vacía, 401, 403 y error", async () => {
  for (const caso of ["vacia", "401", "403", "error"]) {
    const contenedor = raiz();
    const consultas = [];
    let llamadas = 0;
    let escrituras = 0;
    const vista = montarVistaBorradoresPropios(contenedor, {
      cliente: {
        listar: async (consulta) => {
          consultas.push(consulta);
          llamadas += 1;
          if (llamadas === 1) return { items: [item], siguiente_cursor: "cursor-servidor" };
          if (llamadas === 2) return { items: [item] };
          if (caso === "vacia") return { items: [] };
          const error = new Error("consulta fallida");
          if (caso === "401") error.codigo = "autenticacion_requerida";
          if (caso === "403") error.codigo = "acceso_denegado";
          throw error;
        },
        obtener: async () => item,
        crear: async () => { escrituras += 1; return item; },
      },
    });
    await Promise.resolve();
    await Promise.resolve();
    const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
    await panel.listeners.click({ target: contenedor.querySelector('[data-dietas-borrador-pagina="siguiente"]') });
    assert.equal(consultas.at(-1).cursor, "cursor-servidor");
    await panel.listeners.click({ target: contenedor.querySelector("[data-dietas-borrador-consultar-registrados]") });
    assert.deepEqual(consultas.at(-1), { limit: 6 });
    assert.equal(escrituras, 0);
    assert.ok(contenedor.querySelector("[data-dietas-borrador-consultar-registrados]"));
    if (caso === "vacia") {
      assert.ok(contenedor.querySelector("[data-dietas-borradores-vacio]"));
      assert.match(contenedor.querySelector("[data-dietas-borradores-estado]").textContent, /La lista no confirma altas anteriores/u);
      assert.equal(contenedor.querySelector("[data-dietas-borradores-ayuda]").open, false);
    } else {
      assert.equal(contenedor.querySelector("[data-dietas-borradores-vacio]"), null);
      assert.equal(contenedor.querySelector("[data-dietas-borradores-estado]").dataset.tono, "error");
      const esperado = caso === "401" ? /Debe identificarse de nuevo/u : caso === "403" ? /No tiene permiso para consultar/u : /No se ha podido completar/u;
      assert.match(textoVisible(contenedor), esperado);
    }
    vista.desmontar();
  }
});

test("una consulta pendiente no modifica ni enfoca la vista desmontada", async () => {
  const contenedor = raiz();
  let resolver;
  let lecturas = 0;
  const vista = montarVistaBorradoresPropios(contenedor, {
    cliente: {
      listar: async () => {
        lecturas += 1;
        if (lecturas === 1) return { items: [] };
        return new Promise((resuelve) => { resolver = resuelve; });
      },
      obtener: async () => item,
      crear: async () => item,
    },
  });
  await Promise.resolve();
  await Promise.resolve();
  const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
  const peticion = panel.listeners.click({ target: contenedor.querySelector("[data-dietas-borrador-consultar-registrados]") });
  contenedor.ownerDocument.activeElement = null;
  vista.desmontar();
  resolver({ items: [item] });
  await peticion;
  assert.equal(contenedor.querySelector("[data-dietas-borradores-propios]"), null);
  assert.equal(contenedor.ownerDocument.activeElement, null);
});

test("una denegación 401/403 retira el recibo elegido por GET", async () => {
  for (const codigo of ["autenticacion_requerida", "acceso_denegado"]) {
    const contenedor = raiz();
    let lecturas = 0;
    let escrituras = 0;
    const vista = montarVistaBorradoresPropios(contenedor, {
      cliente: {
        listar: async () => {
          lecturas += 1;
          if (lecturas === 1) return { items: [item] };
          const error = new Error("denegado"); error.codigo = codigo; throw error;
        },
        obtener: async () => item,
        crear: async () => { escrituras += 1; return item; },
      },
    });
    await Promise.resolve();
    await Promise.resolve();
    const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
    await panel.listeners.click({ target: contenedor.querySelector("[data-dietas-borrador-detalle]") });
    assert.ok(contenedor.querySelector("[data-dietas-borrador-recibo]"));
    await panel.listeners.click({ target: contenedor.querySelector("[data-dietas-borrador-consultar-registrados]") });
    assert.equal(contenedor.querySelector("[data-dietas-borrador-recibo]"), null);
    const ayuda = contenedor.querySelector("[data-dietas-borradores-ayuda]");
    assert.equal(ayuda.open, false);
    assert.equal(ayuda.querySelector("summary").attrs["aria-label"], "? Ayuda");
    assert.equal(escrituras, 0);
    assert.equal(contenedor.querySelector("[data-dietas-borradores-estado]").dataset.tono, "error");
    vista.desmontar();
  }
});

test("la denegación posterior a otra ficha conserva solo el recibo de un POST confirmado", async () => {
  const contenedor = raiz();
  const alta = {
    comision: { ...item.comision, referencia: "dco_abcdefghijklmnopqrstuv", motivo: "Alta confirmada" },
    recibo: { ...item.recibo, referencia: "rcd_abcdefghijklmnopqrstuv" },
  };
  let lecturas = 0;
  let detalles = 0;
  const vista = montarVistaBorradoresPropios(contenedor, {
    cliente: {
      listar: async () => {
        lecturas += 1;
        if (lecturas < 3) return { items: [item] };
        const error = new Error("denegado"); error.codigo = "acceso_denegado"; throw error;
      },
      obtener: async () => {
        detalles += 1;
        if (detalles === 1) return item;
        const error = new Error("denegado"); error.codigo = "acceso_denegado"; throw error;
      },
      crear: async () => alta,
    },
  });
  await Promise.resolve();
  await Promise.resolve();
  const form = contenedor.querySelector("[data-dietas-borrador-form]");
  form.checkValidity = () => true;
  const datos = {
    fecha_inicio: "2026-09-20", fecha_fin: "2026-09-21", motivo: "Alta confirmada",
    hora_inicio: "09:00", hora_fin: "18:00", origen_codigo: "18087", destino_codigo: "18003",
  };
  const FormDataOriginal = globalThis.FormData;
  globalThis.FormData = class { get(nombre) { return datos[nombre] ?? null; } };
  try {
    const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
    await panel.listeners.submit({ target: form, preventDefault() {} });
    await panel.listeners.click({ target: contenedor.querySelector("[data-dietas-borrador-detalle]") });
    assert.match(textoVisible(contenedor.querySelector("[data-dietas-borrador-recibo]")), /rcd_1234567890123456789012/u);
    await panel.listeners.click({ target: contenedor.querySelector("[data-dietas-borrador-detalle]") });
    assert.equal(contenedor.querySelector("[data-dietas-borrador-detalle]"), null);
    assert.doesNotMatch(textoVisible(contenedor), /Reunión/u);
    assert.match(textoVisible(contenedor.querySelector("[data-dietas-borrador-recibo]")), /rcd_abcdefghijklmnopqrstuv/u);
    await panel.listeners.click({ target: contenedor.querySelector("[data-dietas-borrador-consultar-registrados]") });
    const recibo = textoVisible(contenedor.querySelector("[data-dietas-borrador-recibo]"));
    assert.match(recibo, /rcd_abcdefghijklmnopqrstuv/u);
    assert.doesNotMatch(recibo, /rcd_1234567890123456789012/u);
  } finally {
    globalThis.FormData = FormDataOriginal;
    vista.desmontar();
  }
});

test("una respuesta lenta no roba el foco que pasó del botón al formulario", async () => {
  const contenedor = raiz();
  let lecturas = 0;
  let resolver;
  const vista = montarVistaBorradoresPropios(contenedor, {
    cliente: {
      listar: async () => {
        lecturas += 1;
        if (lecturas === 1) return { items: [] };
        return new Promise((resuelve) => { resolver = resuelve; });
      },
      obtener: async () => item,
      crear: async () => item,
    },
  });
  await Promise.resolve();
  await Promise.resolve();
  const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
  const consultar = contenedor.querySelector("[data-dietas-borrador-consultar-registrados]");
  consultar.focus();
  const peticion = panel.listeners.click({ target: consultar });
  const campo = contenedor.querySelector("[data-dietas-borrador-form]").querySelector("input");
  campo.focus();
  resolver({ items: [item] });
  await peticion;
  assert.equal(contenedor.ownerDocument.activeElement, campo);
  vista.desmontar();
});

test("la preparación local muestra país y se puede revisar sin crear un expediente", async () => {
  const contenedor = raiz();
  let escrituras = 0;
  const vista = montarVistaBorradoresPropios(contenedor, {
    cliente: { listar: async () => ({ items: [] }), obtener: async () => item,
      crear: async () => { escrituras += 1; return item; } },
  });
  await Promise.resolve(); await Promise.resolve();
  const form = contenedor.querySelector("[data-dietas-borrador-form]");
  form.checkValidity = () => true;
  const datos = { fecha_inicio: "2026-09-20", fecha_fin: "2026-09-21", motivo: "Visita",
    hora_inicio: "09:00", hora_fin: "18:00", origen_codigo: "18087", destino_codigo: "18003" };
  const FormDataOriginal = globalThis.FormData;
  globalThis.FormData = class { get(nombre) { return datos[nombre] ?? null; } };
  try {
    const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
    const pais = form.querySelectorAll("input").find((campo) => campo.name === "pais");
    assert.equal(pais.value, "España");
    assert.equal(pais.readOnly, true);
    await panel.listeners.click({ target: form.querySelector("[data-dietas-borrador-revisar]") });
    const resumen = form.querySelector("[data-dietas-borrador-preparacion]");
    assert.match(textoVisible(resumen), /Preparación local sin registrar.*Visita.*España/u);
    assert.doesNotMatch(textoVisible(resumen), /Puede seguir editando los campos antes de crear el borrador/u);
    assert.equal(escrituras, 0);
    assert.equal(contenedor.querySelector("[data-dietas-borrador-recibo]"), null);
    datos.motivo = "Visita corregida";
    panel.listeners.input({ target: form.querySelector("input") });
    assert.equal(resumen.hidden, true);
    assert.equal(escrituras, 0);
  } finally { globalThis.FormData = FormDataOriginal; vista.desmontar(); }
});

test("la denegación de detalle elimina una ficha GET anterior", async () => {
  const contenedor = raiz();
  let consultas = 0;
  const vista = montarVistaBorradoresPropios(contenedor, {
    cliente: { listar: async () => ({ items: [item] }), crear: async () => item,
      obtener: async () => {
        consultas += 1;
        if (consultas === 1) return item;
        const error = new Error("denegado"); error.codigo = "autenticacion_requerida"; throw error;
      } },
  });
  await Promise.resolve(); await Promise.resolve();
  const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
  await panel.listeners.click({ target: contenedor.querySelector("[data-dietas-borrador-detalle]") });
  assert.ok(contenedor.querySelector("[data-dietas-borrador-recibo]"));
  await panel.listeners.click({ target: contenedor.querySelector("[data-dietas-borrador-detalle]") });
  assert.equal(contenedor.querySelector("[data-dietas-borrador-recibo]"), null);
  vista.desmontar();
});

test("401 y 403 de detalle purgan filas, ficha y cursor con o sin ficha previa", async () => {
  for (const codigo of ["autenticacion_requerida", "acceso_denegado"]) {
    for (const fichaPrevia of [false, true]) {
      const contenedor = raiz();
      const privado = {
        comision: { ...item.comision, motivo: `Motivo privado ${codigo}`, fecha_inicio: "2026-10-31" },
        recibo: { ...item.recibo, referencia: "rcd_privado_1234567890123456789012" },
      };
      const consultas = [];
      let detalles = 0;
      let escrituras = 0;
      const vista = montarVistaBorradoresPropios(contenedor, {
        cliente: {
          listar: async (consulta) => {
            consultas.push(consulta);
            return consultas.length === 1
              ? { items: [privado], siguiente_cursor: "cursor-privado" }
              : { items: [] };
          },
          obtener: async () => {
            detalles += 1;
            if (fichaPrevia && detalles === 1) return privado;
            const error = new Error("denegado"); error.codigo = codigo; throw error;
          },
          crear: async () => { escrituras += 1; return privado; },
        },
      });
      await Promise.resolve(); await Promise.resolve();
      const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
      const fechaVisible = new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeZone: "UTC" })
        .format(new Date(`${privado.comision.fecha_inicio}T00:00:00Z`));
      assert.ok(textoVisible(contenedor).includes(privado.comision.motivo));
      assert.ok(textoVisible(contenedor).includes(fechaVisible));
      assert.ok(contenedor.querySelector('[data-dietas-borrador-pagina="siguiente"]'));
      if (fichaPrevia)
        await panel.listeners.click({ target: contenedor.querySelector("[data-dietas-borrador-detalle]") });
      if (fichaPrevia) {
        assert.ok(textoVisible(contenedor).includes(privado.comision.referencia));
        assert.ok(textoVisible(contenedor).includes(privado.recibo.referencia));
      }
      await panel.listeners.click({ target: contenedor.querySelector("[data-dietas-borrador-detalle]") });
      const visible = textoVisible(contenedor);
      for (const dato of [privado.comision.motivo, fechaVisible, privado.comision.referencia, privado.recibo.referencia])
        assert.ok(!visible.includes(dato), `${codigo}: permanece ${dato}`);
      assert.equal(contenedor.querySelector("[data-dietas-borrador-detalle]"), null);
      assert.equal(contenedor.querySelector("[data-dietas-borrador-pagina]"), null);
      assert.equal(contenedor.querySelector("[data-dietas-borrador-recibo]"), null);
      assert.equal(contenedor.querySelector("[data-dietas-borradores-estado]").dataset.tono, "error");
      await panel.listeners.click({ target: contenedor.querySelector("[data-dietas-borrador-consultar-registrados]") });
      assert.equal(Object.hasOwn(consultas[1], "cursor"), false, "el reintento empieza sin cursor anterior");
      assert.equal(escrituras, 0);
      vista.desmontar();
    }
  }
});

test("401 y 403 de POST purgan la lista, ficha y cursor obtenidos por GET", async () => {
  for (const codigo of ["autenticacion_requerida", "acceso_denegado"]) {
    const contenedor = raiz();
    const privado = {
      comision: { ...item.comision, motivo: `Motivo privado ${codigo}`, fecha_inicio: "2026-10-31" },
      recibo: { ...item.recibo, referencia: "rcd_privado_1234567890123456789012" },
    };
    const consultas = [];
    const solicitudes = [];
    const vista = montarVistaBorradoresPropios(contenedor, {
      generarClaveIdempotencia: () => `clave-post-${codigo}`,
      cliente: {
        listar: async (consulta) => {
          consultas.push(consulta);
          return consultas.length === 1
            ? { items: [privado], siguiente_cursor: "cursor-privado" }
            : { items: [] };
        },
        obtener: async () => privado,
        crear: async (solicitud) => {
          solicitudes.push(solicitud);
          const error = new Error("denegado"); error.codigo = codigo; throw error;
        },
      },
    });
    await Promise.resolve(); await Promise.resolve();
    const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
    await panel.listeners.click({ target: contenedor.querySelector("[data-dietas-borrador-detalle]") });
    const form = contenedor.querySelector("[data-dietas-borrador-form]");
    form.checkValidity = () => true;
    const datos = { fecha_inicio: "2026-09-20", fecha_fin: "2026-09-21", motivo: "Nueva comisión",
      hora_inicio: "09:00", hora_fin: "18:00", origen_codigo: "18087", destino_codigo: "18003" };
    const FormDataOriginal = globalThis.FormData;
    globalThis.FormData = class { get(nombre) { return datos[nombre] ?? null; } };
    try {
      await panel.listeners.click({ target: form.querySelector("[data-dietas-parada-anadir]") });
      form.querySelectorAll("select").find((selector) => selector.name === "parada_codigo").value = "18175";
      await panel.listeners.submit({ target: form, preventDefault() {} });
      assert.deepEqual(solicitudes[0].codigos_ruta, ["18087", "18175", "18003"]);
      const visible = textoVisible(contenedor);
      const fechaPrivada = new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeZone: "UTC" })
        .format(new Date(`${privado.comision.fecha_inicio}T00:00:00Z`));
      for (const dato of [privado.comision.motivo, fechaPrivada, privado.comision.referencia, privado.recibo.referencia])
        assert.ok(!visible.includes(dato), `${codigo}: permanece ${dato}`);
      assert.equal(contenedor.querySelector("[data-dietas-borrador-detalle]"), null);
      assert.equal(contenedor.querySelector("[data-dietas-borrador-pagina]"), null);
      assert.equal(contenedor.querySelector("[data-dietas-borrador-recibo]"), null);
      assert.equal(contenedor.querySelector("[data-dietas-borradores-vacio]"), null);
      assert.equal(contenedor.querySelector("[data-dietas-borradores-estado]").dataset.tono, "error");
      assert.match(visible, codigo === "autenticacion_requerida"
        ? /Debe identificarse de nuevo/u : /No tiene permiso para crear un borrador/u);
      await panel.listeners.click({ target: contenedor.querySelector("[data-dietas-borrador-consultar-registrados]") });
      assert.deepEqual(consultas[1], { limit: 6 });
    } finally { globalThis.FormData = FormDataOriginal; vista.desmontar(); }
  }
});

test("un POST denegado conserva solo el último recibo confirmado por POST", async () => {
  for (const codigo of ["autenticacion_requerida", "acceso_denegado"]) {
    const contenedor = raiz();
    const alta = {
      comision: { ...item.comision, referencia: "dco_confirmada_1234567890123456789012", motivo: "Alta confirmada" },
      recibo: { ...item.recibo, referencia: "rcd_confirmado_1234567890123456789012" },
    };
    const altaPosterior = {
      comision: { ...item.comision, referencia: "dco_posterior_1234567890123456789012", motivo: "Alta posterior" },
      recibo: { ...item.recibo, referencia: "rcd_posterior_1234567890123456789012" },
    };
    const privado = {
      comision: { ...item.comision, motivo: "Motivo solo por GET" },
      recibo: { ...item.recibo, referencia: "rcd_solo_get_1234567890123456789012" },
    };
    let escrituras = 0;
    const vista = montarVistaBorradoresPropios(contenedor, {
      cliente: {
        listar: async () => ({ items: [privado] }),
        obtener: async () => privado,
        crear: async () => {
          escrituras += 1;
          if (escrituras === 1) return alta;
          if (escrituras === 2) return altaPosterior;
          const error = new Error("denegado"); error.codigo = codigo; throw error;
        },
      },
    });
    await Promise.resolve(); await Promise.resolve();
    const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
    const form = contenedor.querySelector("[data-dietas-borrador-form]");
    form.checkValidity = () => true;
    const datos = { fecha_inicio: "2026-09-20", fecha_fin: "2026-09-21", motivo: "Primera comisión",
      hora_inicio: "09:00", hora_fin: "18:00", origen_codigo: "18087", destino_codigo: "18003" };
    const FormDataOriginal = globalThis.FormData;
    globalThis.FormData = class { get(nombre) { return datos[nombre] ?? null; } };
    try {
      await panel.listeners.submit({ target: form, preventDefault() {} });
      await panel.listeners.click({ target: contenedor.querySelector("[data-dietas-borrador-detalle]") });
      assert.match(textoVisible(contenedor), /Motivo solo por GET/u);
      datos.motivo = "Segunda comisión";
      await panel.listeners.submit({ target: form, preventDefault() {} });
      datos.motivo = "Tercera comisión";
      await panel.listeners.submit({ target: form, preventDefault() {} });
      const visible = textoVisible(contenedor);
      assert.equal(escrituras, 3);
      assert.match(visible, /rcd_posterior_1234567890123456789012/u);
      assert.doesNotMatch(visible, /Motivo solo por GET|rcd_solo_get_1234567890123456789012|rcd_confirmado_1234567890123456789012/u);
      assert.equal(contenedor.querySelector("[data-dietas-borrador-detalle]"), null);
      assert.equal(contenedor.querySelector("[data-dietas-borradores-estado]").dataset.tono, "error");
      datos.motivo = "Primera comisión";
      await panel.listeners.submit({ target: form, preventDefault() {} });
      assert.equal(escrituras, 4, "el recibo anterior exige volver a consultar al servidor");
      assert.doesNotMatch(textoVisible(contenedor), /rcd_confirmado_1234567890123456789012/u);
    } finally { globalThis.FormData = FormDataOriginal; vista.desmontar(); }
  }
});

test("un GET pendiente bloquea Guardar y Revisar sin iniciar otra creación", async () => {
  const contenedor = raiz();
  let resolver;
  let consultas = 0;
  let escrituras = 0;
  const vista = montarVistaBorradoresPropios(contenedor, {
    cliente: {
      listar: async () => {
        consultas += 1;
        if (consultas === 1) return { items: [] };
        return new Promise((resuelve) => { resolver = resuelve; });
      },
      obtener: async () => item,
      crear: async () => { escrituras += 1; return item; },
    },
  });
  await Promise.resolve(); await Promise.resolve();
  const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
  const form = contenedor.querySelector("[data-dietas-borrador-form]");
  form.checkValidity = () => true;
  const consulta = panel.listeners.click({ target: contenedor.querySelector("[data-dietas-borrador-consultar-registrados]") });
  assert.equal(form.querySelector("[data-dietas-borrador-guardar]").disabled, true);
  assert.equal(form.querySelector("[data-dietas-borrador-revisar]").disabled, true);
  await panel.listeners.submit({ target: form, preventDefault() {} });
  assert.equal(escrituras, 0);
  resolver({ items: [] });
  await consulta;
  assert.equal(form.querySelector("[data-dietas-borrador-guardar]").disabled, false);
  assert.equal(form.querySelector("[data-dietas-borrador-revisar]").disabled, false);
  vista.desmontar();
});

test("A incierta, B válida y vuelta a A reutilizan la clave de A sin duplicar B", async () => {
  const contenedor = raiz();
  const solicitudes = [];
  let secuencia = 0;
  const vista = montarVistaBorradoresPropios(contenedor, {
    generarClaveIdempotencia: () => `clave-${++secuencia}`,
    cliente: {
      listar: async () => ({ items: [] }),
      obtener: async () => item,
      crear: async (solicitud) => {
        solicitudes.push(solicitud);
        if (solicitudes.length === 1) {
          const error = new Error("respuesta incierta"); error.resultadoIndeterminado = true; throw error;
        }
        return { ...item, comision: { ...item.comision, motivo: solicitud.motivo } };
      },
    },
  });
  await Promise.resolve(); await Promise.resolve();
  const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
  const form = contenedor.querySelector("[data-dietas-borrador-form]");
  form.checkValidity = () => true;
  const datos = { fecha_inicio: "2026-09-20", fecha_fin: "2026-09-21", motivo: "A",
    hora_inicio: "09:00", hora_fin: "18:00", origen_codigo: "18087", destino_codigo: "18003" };
  const FormDataOriginal = globalThis.FormData;
  globalThis.FormData = class { get(nombre) { return datos[nombre] ?? null; } };
  try {
    await panel.listeners.submit({ target: form, preventDefault() {} });
    datos.motivo = "B";
    await panel.listeners.submit({ target: form, preventDefault() {} });
    datos.motivo = "A";
    await panel.listeners.submit({ target: form, preventDefault() {} });
    datos.motivo = "B";
    await panel.listeners.submit({ target: form, preventDefault() {} });
    assert.deepEqual(solicitudes.map(({ clave_idempotencia, motivo }) => [clave_idempotencia, motivo]), [
      ["clave-1", "A"], ["clave-2", "B"], ["clave-1", "A"],
    ]);
    assert.equal(secuencia, 2);
    assert.ok(contenedor.querySelector("[data-dietas-borrador-recibo]"));
  } finally { globalThis.FormData = FormDataOriginal; vista.desmontar(); }
});

test("A incierta conserva su clave tras un 403 posterior y no duplica B al reautorizar", async () => {
  const contenedor = raiz();
  const solicitudes = [];
  let secuencia = 0;
  const vista = montarVistaBorradoresPropios(contenedor, {
    generarClaveIdempotencia: () => `operacion-clave-${String(++secuencia).padStart(4, "0")}`,
    cliente: {
      listar: async () => ({ items: [] }),
      obtener: async () => item,
      crear: async (solicitud) => {
        solicitudes.push(solicitud);
        if (solicitudes.length === 1) {
          const error = new Error("503 incierto"); error.codigo = "resultado_incierto";
          error.resultadoIndeterminado = true; throw error;
        }
        if (solicitudes.length === 2) {
          const error = new Error("403 tras el reintento"); error.codigo = "acceso_denegado";
          error.resultadoIndeterminado = false; throw error;
        }
        return { ...item, comision: { ...item.comision, motivo: solicitud.motivo } };
      },
    },
  });
  await Promise.resolve(); await Promise.resolve();
  const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
  const form = contenedor.querySelector("[data-dietas-borrador-form]");
  form.checkValidity = () => true;
  const datos = { fecha_inicio: "2026-09-20", fecha_fin: "2026-09-21", motivo: "A",
    hora_inicio: "09:00", hora_fin: "18:00", origen_codigo: "18087", destino_codigo: "18003" };
  const FormDataOriginal = globalThis.FormData;
  globalThis.FormData = class { get(nombre) { return datos[nombre] ?? null; } };
  try {
    await panel.listeners.submit({ target: form, preventDefault() {} });
    await panel.listeners.submit({ target: form, preventDefault() {} });
    assert.equal(contenedor.querySelector("[data-dietas-borrador-recibo]"), null);
    datos.motivo = "B";
    await panel.listeners.submit({ target: form, preventDefault() {} });
    datos.motivo = "A";
    await panel.listeners.submit({ target: form, preventDefault() {} });
    datos.motivo = "B";
    await panel.listeners.submit({ target: form, preventDefault() {} });
    assert.deepEqual(solicitudes.map(({ clave_idempotencia, motivo }) => [clave_idempotencia, motivo]), [
      ["operacion-clave-0001", "A"], ["operacion-clave-0001", "A"],
      ["operacion-clave-0002", "B"], ["operacion-clave-0001", "A"],
    ]);
    assert.equal(secuencia, 2);
  } finally { globalThis.FormData = FormDataOriginal; vista.desmontar(); }
});
