import assert from "node:assert/strict";
import test from "node:test";

import { montarVistaBorradoresPropios as montarVistaConPuerto } from "./vista-borradores-propios.js";
import { crearClienteBorradoresDietasHTTP } from "./cliente-borradores-http.js";
import { ESQUEMA_CATALOGO_RUTAS_DIETAS } from "./contrato.js";

const catalogoProyectado = [
  { codigo: "18087", nombre: "Granada" }, { codigo: "18003", nombre: "Albolote" },
  { codigo: "18061", nombre: "Chimeneas" }, { codigo: "18175", nombre: "Santa Fe" },
  ...Array.from({ length: 15 }, (_valor, indice) => ({ codigo: String(18001 + indice).padStart(5, "0"), nombre: `Municipio ${indice + 1}` }))
    .filter((punto) => punto.codigo !== "18003"),
];
const catalogoRuta = (version = "granada-v1") => ({ esquema: ESQUEMA_CATALOGO_RUTAS_DIETAS,
  demostracion: false, completo: false, version, puntos: [
    { codigo: "18087", nombre: "Granada", tipo: "municipio", municipio_codigo: "18087", municipio_nombre: "Granada" },
    { codigo: "18003", nombre: "Albolote", tipo: "municipio", municipio_codigo: "18003", municipio_nombre: "Albolote" },
  ] });
const montarVistaBorradoresPropios = (contenedor, opciones = {}) =>
  montarVistaConPuerto(contenedor, { catalogoProyectado, ...opciones });

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
  removeAttribute(nombre) {
    delete this.attrs[nombre];
  }
  focus() {
    this.ownerDocument.activeElement = this;
  }
  matches(selector) {
    if (!selector.startsWith("[")) return this.tagName === selector;
    const coincidencia = selector.match(/^\[([^=\]]+)(?:="([^"]*)")?\]$/u);
    const valor = coincidencia[1].startsWith("data-")
      ? this.dataset[claveDatos(coincidencia[1])] : this[coincidencia[1]] ?? this.attrs[coincidencia[1]];
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
function entradasFormulario(form) {
  return [...form.querySelectorAll("input"), ...form.querySelectorAll("select")];
}
function entradaFormulario(form, nombre) {
  return entradasFormulario(form).find((campo) => campo.name === nombre);
}
class DatosFormulario {
  constructor(form) {
    this.datos = new Map();
    for (const campo of entradasFormulario(form)) {
      if (campo.name && !campo.disabled && !this.datos.has(campo.name))
        this.datos.set(campo.name, campo.value || "");
    }
  }
  get(nombre) { return this.datos.get(nombre) ?? null; }
}
function rellenarSolicitud(form) {
  const valores = { fecha_inicio: "2026-09-20", fecha_fin: "2026-09-21", motivo: "  Visita conservada  ",
    hora_inicio: "09:00", hora_fin: "18:00", origen_codigo: "18087", destino_codigo: "18003",
    pais: "ES", pais_otro: "Destino escrito", nivel_detalle: "alto" };
  for (const [nombre, valor] of Object.entries(valores)) entradaFormulario(form, nombre).value = valor;
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

test("formulario y mapa comparten una sola lectura del catálogo de rutas", async () => {
  const contenedor = raiz(); let lecturas = 0;
  const vista = montarVistaBorradoresPropios(contenedor, {
    cliente: { listar: async () => ({ items: [] }), obtener: async () => item, crear: async () => item },
    calculadorRuta: { obtenerCatalogo: async () => { lecturas++; return catalogoRuta(); },
      obtenerCatalogoOtrosGastos: async () => null, calcular: async () => { throw new Error("sin ruta"); } },
    visorRuta: { montar: () => ({ desmontar() {} }) },
  });
  await new Promise((resolver) => setImmediate(resolver));
  assert.equal(lecturas, 1);
  assert.ok(contenedor.querySelector("[data-dietas-mapa-comision]"));
  const origen = contenedor.querySelector('[name="origen_codigo"]');
  assert.ok(origen.querySelectorAll("option").some((opcion) => opcion.value === "18087"));
  vista.desmontar();
});

test("un fallo conserva el formulario y Calcular ruta reintenta el catálogo una vez", async () => {
  const contenedor = raiz(); let lecturas = 0;
  const original = globalThis.FormData;
  globalThis.FormData = DatosFormulario;
  const vista = montarVistaBorradoresPropios(contenedor, {
    cliente: { listar: async () => ({ items: [] }), obtener: async () => item, crear: async () => item },
    calculadorRuta: { obtenerCatalogo: async () => {
      if (++lecturas === 1) throw new Error("fuente no disponible");
      return catalogoRuta();
    }, obtenerCatalogoOtrosGastos: async () => null, calcular: async () => { throw new Error("sin ruta"); } },
    visorRuta: { montar: () => ({ desmontar() {} }) },
  });
  try {
    await new Promise((resolver) => setImmediate(resolver));
    const formulario = contenedor.querySelector("[data-dietas-borrador-form]");
    const boton = formulario.querySelector("[data-dietas-calcular-ruta]");
    assert.ok(formulario);
    assert.match(textoVisible(contenedor), /No se ha podido calcular la ruta/u);
    assert.equal(boton.textContent, "Reintentar");
    await contenedor.querySelector("[data-dietas-borradores-propios]").listeners.click({ target: boton });
    assert.equal(lecturas, 2);
    assert.ok(contenedor.querySelector("[data-dietas-mapa-comision]"));
    assert.notEqual(boton.textContent, "Reintentar");
  } finally { globalThis.FormData = original; vista.desmontar(); }
});

test("dos clics en Reintentar comparten la lectura y conservan el foco", async () => {
  const contenedor = raiz(); let lecturas = 0; let completar; let señal; const avisos = [];
  const original = globalThis.FormData;
  globalThis.FormData = DatosFormulario;
  const vista = montarVistaBorradoresPropios(contenedor, {
    cliente: { listar: async () => ({ items: [] }), obtener: async () => item, crear: async () => item },
    calculadorRuta: { obtenerCatalogo: ({ signal }) => {
      if (++lecturas === 1) throw new Error("fuente no disponible");
      señal = signal;
      return new Promise((resolver) => { completar = resolver; });
    }, obtenerCatalogoOtrosGastos: async () => null, calcular: async () => { throw new Error("sin ruta"); } },
    visorRuta: { montar: () => ({ desmontar() {} }) }, anunciar: (mensaje) => avisos.push(mensaje),
  });
  try {
    await new Promise((resolver) => setImmediate(resolver));
    const boton = contenedor.querySelector("[data-dietas-calcular-ruta]");
    const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
    const erroresIniciales = avisos.filter((mensaje) => mensaje.includes("No se ha podido calcular la ruta")).length;
    assert.equal(boton.textContent, "Reintentar");
    boton.focus();
    const primero = panel.listeners.click({ target: boton });
    const segundo = panel.listeners.click({ target: boton });
    await Promise.resolve(); await Promise.resolve();
    assert.equal(lecturas, 2);
    assert.equal(señal.aborted, false);
    assert.equal(boton.attrs["aria-busy"], "true");
    completar(catalogoRuta());
    await Promise.all([primero, segundo]);
    assert.equal(lecturas, 2);
    assert.equal(señal.aborted, false);
    assert.ok(contenedor.querySelector("[data-dietas-mapa-comision]"));
    assert.notEqual(boton.textContent, "Reintentar");
    assert.equal(boton.attrs["aria-busy"], undefined);
    assert.equal(contenedor.ownerDocument.activeElement, boton);
    assert.equal(avisos.filter((mensaje) => mensaje.includes("No se ha podido calcular la ruta")).length, erroresIniciales);
    assert.equal(avisos.filter((mensaje) => mensaje.includes("localidades distintas")).length, 1);
  } finally { globalThis.FormData = original; vista.desmontar(); }
});

test("dos clics con ruta válida solo inician un cálculo mientras sigue pendiente", async () => {
  const contenedor = raiz(); let calculos = 0; let rechazar;
  const original = globalThis.FormData;
  globalThis.FormData = DatosFormulario;
  const vista = montarVistaBorradoresPropios(contenedor, {
    cliente: { listar: async () => ({ items: [] }), obtener: async () => item, crear: async () => item },
    calculadorRuta: { obtenerCatalogo: async () => catalogoRuta(), obtenerCatalogoOtrosGastos: async () => null,
      calcular: () => { calculos++; return new Promise((_resolver, rechazarPromesa) => { rechazar = rechazarPromesa; }); } },
    visorRuta: { montar: () => ({ desmontar() {} }) },
  });
  try {
    await new Promise((resolver) => setImmediate(resolver));
    const form = contenedor.querySelector("[data-dietas-borrador-form]");
    form.querySelector('[name="origen_codigo"]').value = "18087";
    form.querySelector('[name="destino_codigo"]').value = "18003";
    const boton = form.querySelector("[data-dietas-calcular-ruta]");
    const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
    const primero = panel.listeners.click({ target: boton });
    const segundo = panel.listeners.click({ target: boton });
    await Promise.resolve(); await Promise.resolve();
    assert.equal(calculos, 1);
    assert.equal(boton.attrs["aria-busy"], "true");
    rechazar(new Error("servicio temporalmente no disponible"));
    await Promise.all([primero, segundo]);
    assert.equal(boton.attrs["aria-busy"], undefined);
    const nuevoIntento = panel.listeners.click({ target: boton });
    await Promise.resolve(); await Promise.resolve();
    assert.equal(calculos, 2);
    rechazar(new Error("servicio temporalmente no disponible"));
    await nuevoIntento;
  } finally { globalThis.FormData = original; vista.desmontar(); }
});

test("una respuesta del catálogo tras desmontar no pinta mapa ni modifica el formulario", async () => {
  const contenedor = raiz(); let completar; let señal; const avisos = [];
  const vista = montarVistaBorradoresPropios(contenedor, {
    cliente: { listar: async () => ({ items: [] }), obtener: async () => item, crear: async () => item },
    calculadorRuta: { obtenerCatalogo: ({ signal }) => { señal = signal; return new Promise((resolver) => { completar = resolver; }); },
      obtenerCatalogoOtrosGastos: async () => null, calcular: async () => { throw new Error("sin ruta"); } },
    visorRuta: { montar: () => ({ desmontar() {} }) }, anunciar: (mensaje) => avisos.push(mensaje),
  });
  const form = contenedor.querySelector("[data-dietas-borrador-form]");
  vista.desmontar();
  completar(catalogoRuta());
  await new Promise((resolver) => setImmediate(resolver));
  assert.equal(señal.aborted, true);
  assert.equal(contenedor.querySelector("[data-dietas-mapa-comision]"), null);
  assert.equal(form.querySelector('[name="origen_codigo"]').querySelectorAll("option").length, 1);
  assert.deepEqual(avisos, []);
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

test("conserva todos los controles y la misma solicitud al recuperar un alta incierta", async () => {
  const contenedor = raiz();
  const solicitudes = [];
  let rechazar;
  let siguienteClave = 0;
  const vista = montarVistaBorradoresPropios(contenedor, {
    generarClaveIdempotencia: () => `clave-conservada-${++siguienteClave}`,
    cliente: { listar: async () => ({ items: [] }), obtener: async () => item,
      crear: async (solicitud) => {
        solicitudes.push(structuredClone(solicitud));
        if (solicitudes.length === 1) return new Promise((_resolver, rechazo) => { rechazar = rechazo; });
        return { ...item, recibo: { ...item.recibo, repeticion: true } };
      } },
  });
  await Promise.resolve(); await Promise.resolve();
  const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
  const form = contenedor.querySelector("[data-dietas-borrador-form]");
  form.checkValidity = () => true;
  rellenarSolicitud(form);
  await panel.listeners.click({ target: form.querySelector("[data-dietas-parada-anadir]") });
  entradaFormulario(form, "parada_codigo").value = "18175";
  const controles = entradasFormulario(form);
  const antes = controles.map((campo) => [campo, campo.value, campo.disabled]);
  const original = globalThis.FormData;
  globalThis.FormData = DatosFormulario;
  try {
    const pendiente = panel.listeners.submit({ target: form, preventDefault() {} });
    assert.equal(form.attrs["aria-busy"], "true");
    assert.ok([...controles, ...form.querySelectorAll("button")].every((campo) => campo.disabled));
    await panel.listeners.submit({ target: form, preventDefault() {} });
    assert.equal(solicitudes.length, 1);
    const error = new Error("conexión interrumpida"); error.resultadoIndeterminado = true;
    rechazar(error);
    await pendiente;
    assert.equal(form.attrs["aria-busy"], "false");
    assert.equal(contenedor.ownerDocument.activeElement, contenedor.querySelector("[data-dietas-borradores-estado]"));
    assert.match(textoVisible(contenedor), /No se ha podido confirmar/u);
    assert.equal(contenedor.querySelector("[data-dietas-borrador-form]"), form);
    assert.deepEqual(controles.map((campo) => [campo, campo.value, campo.disabled]), antes);
    assert.equal(contenedor.querySelector("[data-dietas-borrador-recibo]"), null);
    await panel.listeners.submit({ target: form, preventDefault() {} });
    assert.equal(siguienteClave, 1);
    assert.deepEqual(solicitudes[1], solicitudes[0]);
    assert.deepEqual(solicitudes[0].codigos_ruta, ["18087", "18175", "18003"]);
    assert.deepEqual(controles.map((campo) => [campo, campo.value, campo.disabled]), antes);
    assert.match(textoVisible(contenedor), /recuperado sin crear otro borrador/u);
  } finally { globalThis.FormData = original; vista.desmontar(); }
});

test("fecha, hora y ruta inválidas conservan la entrada y enfocan el campo que hay que corregir", async () => {
  const contenedor = raiz(); let escrituras = 0;
  const vista = montarVistaBorradoresPropios(contenedor, {
    cliente: { listar: async () => ({ items: [] }), obtener: async () => item,
      crear: async () => { escrituras++; return item; } },
  });
  await Promise.resolve(); await Promise.resolve();
  const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
  const form = contenedor.querySelector("[data-dietas-borrador-form]");
  form.checkValidity = () => true;
  rellenarSolicitud(form);
  const original = globalThis.FormData;
  globalThis.FormData = DatosFormulario;
  try {
    for (const [nombre, valor, revisar] of [["fecha_fin", "2026-09-19", false], ["hora_fin", "09:00", true],
      ["destino_codigo", "18087", false]]) {
      rellenarSolicitud(form);
      if (nombre === "hora_fin") entradaFormulario(form, "fecha_fin").value = "2026-09-20";
      const campo = entradaFormulario(form, nombre);
      campo.value = valor;
      const antes = entradasFormulario(form).map((entrada) => [entrada, entrada.value]);
      if (revisar) await panel.listeners.click({ target: form.querySelector("[data-dietas-borrador-revisar]") });
      else await panel.listeners.submit({ target: form, preventDefault() {} });
      assert.equal(escrituras, 0);
      assert.equal(contenedor.ownerDocument.activeElement, campo);
      assert.equal(campo.attrs["aria-invalid"], "true");
      assert.equal(campo.attrs["aria-describedby"], contenedor.querySelector("[data-dietas-borradores-estado]").id);
      assert.deepEqual(entradasFormulario(form).map((entrada) => [entrada, entrada.value]), antes);
      panel.listeners.input({ target: campo });
      assert.equal(campo.attrs["aria-invalid"], undefined);
      assert.equal(campo.attrs["aria-describedby"], undefined);
    }
  } finally { globalThis.FormData = original; vista.desmontar(); }
});

test("la consulta pendiente conserva los campos y restaura la disponibilidad propia tras un error", async () => {
  const contenedor = raiz(); let rechazar; let consultas = 0;
  const vista = montarVistaBorradoresPropios(contenedor, {
    cliente: { listar: async () => {
      if (++consultas === 1) return { items: [] };
      return new Promise((_resuelve, rechazo) => { rechazar = rechazo; });
    }, obtener: async () => item, crear: async () => item },
  });
  await Promise.resolve(); await Promise.resolve();
  const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
  const form = contenedor.querySelector("[data-dietas-borrador-form]");
  rellenarSolicitud(form);
  const controles = entradasFormulario(form);
  const antes = controles.map((campo) => [campo, campo.value, campo.disabled]);
  const consulta = panel.listeners.click({ target: contenedor.querySelector("[data-dietas-borrador-consultar-registrados]") });
  assert.ok(controles.every((campo) => campo.disabled));
  rechazar(new Error("sin conexión")); await consulta;
  assert.deepEqual(controles.map((campo) => [campo, campo.value, campo.disabled]), antes);
  assert.equal(form.querySelector("[data-dietas-vehiculo-propio]").disabled, true);
  assert.equal(form.querySelector("[data-dietas-otro-anadir]").disabled, true);
  assert.equal(form.querySelector("[data-dietas-borrador-guardar]").disabled, false);
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
    etiquetasRelaciones: [
      { referencia: "rel_1234567890123456789012", etiqueta: "Centro de muestra · Unidad técnica" },
      { referencia: "rel_abcdefghijklmnopqrstuv", etiqueta: "Centro de ensayo · Unidad administrativa" },
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
  assert.deepEqual(selectorRelacion.children.slice(1).map((opcion) => opcion.textContent),
    ["Centro de muestra · Unidad técnica", "Centro de ensayo · Unidad administrativa"]);
  assert.doesNotMatch(textoVisible(selectorRelacion), /rel_/u);
  assert.equal(contenedor.querySelector("[data-dietas-borrador-guardar]").disabled, true);
  selectorRelacion.value = "rel_abcdefghijklmnopqrstuv";
  contenedor.querySelector("[data-dietas-borradores-propios]").listeners.change({ target: selectorRelacion });
  await Promise.resolve(); await Promise.resolve();
  assert.equal(listas, 1);
  assert.equal(contenedor.querySelector("[data-dietas-borrador-guardar]").disabled, false);
  assert.equal(contenedor.querySelector("[data-dietas-borrador-revisar]").disabled, false);
  vista.desmontar();
});

test("una relación única se envía como campo oculto sin rótulo de referencia técnica", async () => {
  const contenedor = raiz();
  const vista = montarVistaBorradoresPropios(contenedor, { relacionesAutorizadas: [item.comision.relacion_ref],
    cliente: { listar: async () => ({ items: [] }), obtener: async () => item, crear: async () => item } });
  await Promise.resolve(); await Promise.resolve();
  const form = contenedor.querySelector("[data-dietas-borrador-form]");
  const relacion = entradaFormulario(form, "relacion_ref");
  assert.equal(relacion.type, "hidden");
  assert.equal(relacion.value, item.comision.relacion_ref);
  assert.equal(relacion.closest("label"), null);
  assert.doesNotMatch(textoVisible(form), /rel_|Referencia/u);
  vista.desmontar();
});

test("varias relaciones sin nombres completos cierran acciones incluso al forzar botones y permiten el GET explícito", async () => {
  const relaciones = [item.comision.relacion_ref, "rel_abcdefghijklmnopqrstuv"];
  for (const etiquetasRelaciones of [undefined, [], [{ referencia: relaciones[0], etiqueta: "Centro de muestra · Unidad técnica" }]]) {
    const contenedor = raiz(); let escrituras = 0; let listas = 0; const lecturas = [];
    const vista = montarVistaBorradoresPropios(contenedor, {
      relacionesAutorizadas: relaciones.map((relacion_ref) => ({ relacion_ref, unidad_ref: "unidad:ensayo" })),
      etiquetasRelaciones, fechaReferenciaPersonal: "2026-09-24",
      clienteAsignacion: { obtener: async () => ({ verificada: true,
        relacion_ref: item.comision.relacion_ref, unidad_ref: "unidad:ensayo", centro_ref: "centro:ensayo",
        administrativo_persona_ref: "per_aaaaaaaaaaaaaaaaaaaaaa", responsable_persona_ref: "per_bbbbbbbbbbbbbbbbbbbbbb",
        grupo_dieta: 2 }) },
      cliente: { listar: async () => { listas++; return { items: [] }; },
        obtener: async (ref, opciones) => {
          lecturas.push([ref, opciones.relacion_ref]);
          return { ...item, comision: { ...item.comision, documento: {} } };
        },
        crear: async () => { escrituras++; return item; }, enviar: async () => { escrituras++; return item; } } });
    await Promise.resolve(); await Promise.resolve();
    const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
    const form = contenedor.querySelector("[data-dietas-borrador-form]");
    assert.equal(entradaFormulario(form, "relacion_ref"), undefined);
    assert.doesNotMatch(textoVisible(form), /rel_/u);
    assert.match(contenedor.querySelector("[data-dietas-borradores-estado]").textContent, /No se puede determinar una relación/u);
    const guardar = form.querySelector("[data-dietas-borrador-guardar]");
    const revisar = form.querySelector("[data-dietas-borrador-revisar]");
    assert.equal(guardar.disabled, true); assert.equal(revisar.disabled, true);
    form.checkValidity = () => true;
    guardar.disabled = false; revisar.disabled = false;
    await panel.listeners.submit({ target: form, preventDefault() {} });
    await panel.listeners.click({ target: revisar });
    assert.equal(escrituras, 0); assert.equal(listas, 0);
    assert.equal(form.querySelector("[data-dietas-borrador-preparacion]").hidden, true);
    // El identificador explícito de una consulta sigue pasando por el cliente
    // autorizado; los nombres no crean ni revocan una concesión del servidor.
    const consultar = contenedor.ownerDocument.createElement("button");
    consultar.dataset.dietasBorradorDetalle = item.comision.referencia;
    consultar.dataset.dietasBorradorRelacion = item.comision.relacion_ref;
    panel.append(consultar);
    await panel.listeners.click({ target: consultar });
    assert.deepEqual(lecturas, [[item.comision.referencia, item.comision.relacion_ref]]);
    const enviar = contenedor.querySelector("[data-dietas-borrador-enviar]");
    assert.equal(enviar.disabled, true);
    enviar.disabled = false;
    await panel.listeners.click({ target: enviar });
    assert.equal(escrituras, 0);
    vista.desmontar();
  }
});

test("las etiquetas no añaden referencias autorizadas ni aceptan duplicados o códigos opacos como nombres", () => {
  const referencias = [item.comision.relacion_ref, "rel_abcdefghijklmnopqrstuv"];
  for (const etiquetasRelaciones of [null,
    [{ referencia: "rel_zzzzzzzzzzzzzzzzzzzzzz", etiqueta: "Centro de muestra" }],
    [{ referencia: referencias[0], etiqueta: "Centro de muestra" }, { referencia: referencias[0], etiqueta: "Centro de ensayo" }],
    [{ referencia: referencias[0], etiqueta: referencias[0] }],
    [{ referencia: referencias[0], etiqueta: "" }],
    [{ referencia: referencias[0], etiqueta: "Centro de muestra\nUnidad" }],
    [{ referencia: referencias[0], etiqueta: "Centro de muestra" }, { referencia: referencias[1], etiqueta: "Centro de muestra" }]]) {
    const contenedor = raiz();
    assert.throws(() => montarVistaBorradoresPropios(contenedor, { relacionesAutorizadas: referencias, etiquetasRelaciones }), TypeError);
    assert.equal(contenedor.querySelector("[data-dietas-borradores-propios]"), null);
  }
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

test("el total no definitivo del documento se lee desde la ayuda «?» del cálculo", async () => {
  const conDocumento = { ...item, comision: { ...item.comision, codigos_ruta: ["18087", "18003"], calculo: {
    rotulo: "PROVISIONAL", version_tarifa: "provisional:rd462:20260923", version_grafo: "grafo:prueba",
    kilometros: "12.0000", importe_kilometraje_centimos: 312,
    tramos_ruta: [{ origen_codigo: "18087", destino_codigo: "18003", kilometros: "12.0000" }], opciones_dieta: [] },
  documento: { grupo_dieta: 2, manutencion_centimos: 2500, alojamiento_tope_centimos: 0, kilometraje_centimos: 312,
    otros_centimos: 0, total_orientativo_centimos: 2812, lineas: [] } } };
  const contenedor = raiz();
  const vista = montarVistaBorradoresPropios(contenedor, { cliente: { listar: async () => ({ items: [conDocumento] }),
    obtener: async () => conDocumento, crear: async () => conDocumento } });
  await Promise.resolve(); await Promise.resolve();
  await contenedor.querySelector("[data-dietas-borradores-propios]").listeners.click({
    target: contenedor.querySelector("[data-dietas-borrador-detalle]") });
  const aviso = /El total definitivo queda pendiente/u;
  const ayudas = contenedor.querySelectorAll("details").filter((nodo) => aviso.test(textoVisible(nodo)));
  assert.equal(ayudas.length, 1);
  assert.equal(ayudas[0].className, "dietas-borradores-ayuda");
  assert.equal(ayudas[0].children[0].textContent, "?");
  assert.equal(contenedor.querySelectorAll("p").filter((nodo) => aviso.test(nodo.textContent)).length, 1);
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
    const pais = form.querySelector("[data-dietas-pais]");
    assert.equal(pais.value, "ES");
    pais.value = "OTRO";
    panel.listeners.change({ target: pais });
    assert.equal(form.querySelector("[data-dietas-borrador-guardar]").disabled, true);
    assert.equal(form.querySelector("[data-dietas-pais-sin-calculo]").hidden, false);
    assert.equal(form.querySelectorAll("select").find((campo) => campo.name === "origen_codigo").required, false);
    pais.value = "ES";
    panel.listeners.change({ target: pais });
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

test("HTTP conserva la clave de un alta confirmada tras purga y 403; 503 recupera sin duplicar", async () => {
  const contenedor = raiz();
  const privado = {
    comision: { ...item.comision, motivo: "Motivo privado GET", codigos_ruta: ["18087", "18003"] },
    recibo: { ...item.recibo, referencia: `rcd_${"p".repeat(22)}`, registrado_en: "2026-09-20T10:00:00.123456Z" },
  };
  const operacionesServidor = new Map();
  const altasPorMotivo = new Map();
  const solicitudes = [];
  let autorizada = true;
  let siguienteClave = 0;
  const json = (cuerpo, estado) => new Response(JSON.stringify(cuerpo), {
    status: estado, headers: { "Content-Type": "application/json; charset=utf-8" },
  });
  const cliente = crearClienteBorradoresDietasHTTP({ fetchImpl: async (ruta, opciones) => {
    if (opciones.method === "GET")
      return ruta.includes("?limit=") ? json({ items: [privado], siguiente_cursor: "cursor-privado" }, 200) : json(privado, 200);
    const solicitud = JSON.parse(opciones.body);
    solicitudes.push(solicitud);
    if (!autorizada) return json({ error: "dietas.error.acceso_denegado" }, 403);
    const anterior = operacionesServidor.get(solicitud.clave_idempotencia);
    if (anterior) return json({ ...anterior, recibo: { ...anterior.recibo, repeticion: true } }, 200);
    const numero = operacionesServidor.size + 1;
    const creado = {
      comision: {
        referencia: `dco_${String(numero).padStart(22, "0")}`, estado: "borrador",
        fecha_inicio: solicitud.fecha_inicio, fecha_fin: solicitud.fecha_fin,
        motivo: solicitud.motivo, codigos_ruta: solicitud.codigos_ruta,
        relacion_ref: "rel_1234567890123456789012",
      },
      recibo: {
        referencia: `rcd_${String(numero).padStart(22, "0")}`, version: 1,
        registrado_en: "2026-09-20T10:00:00.123456Z", repeticion: false,
      },
    };
    operacionesServidor.set(solicitud.clave_idempotencia, creado);
    altasPorMotivo.set(solicitud.motivo, (altasPorMotivo.get(solicitud.motivo) || 0) + 1);
    if (solicitud.motivo === "D") return json({ error: "dietas.error.resultado_incierto" }, 503);
    return json(creado, 201);
  } });
  const vista = montarVistaBorradoresPropios(contenedor, {
    cliente, generarClaveIdempotencia: () => `clave-operacion-${String(++siguienteClave).padStart(4, "0")}`,
  });
  await new Promise((resolver) => setImmediate(resolver));
  const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
  await panel.listeners.click({ target: contenedor.querySelector("[data-dietas-borrador-detalle]") });
  assert.match(textoVisible(contenedor), /Motivo privado GET/u);
  const form = contenedor.querySelector("[data-dietas-borrador-form]");
  form.checkValidity = () => true;
  const datos = { fecha_inicio: "2026-09-20", fecha_fin: "2026-09-21", motivo: "A",
    hora_inicio: "09:00", hora_fin: "18:00", origen_codigo: "18087", destino_codigo: "18003" };
  const FormDataOriginal = globalThis.FormData;
  globalThis.FormData = class { get(nombre) { return datos[nombre] ?? null; } };
  const enviar = () => panel.listeners.submit({ target: form, preventDefault() {} });
  try {
    await panel.listeners.click({ target: form.querySelector("[data-dietas-parada-anadir]") });
    form.querySelectorAll("select").find((selector) => selector.name === "parada_codigo").value = "18175";
    await enviar(); // A: 201.
    datos.motivo = "B"; await enviar(); // B: 201.
    autorizada = false;
    datos.motivo = "C"; await enviar(); // C: 403 y purga.
    assert.doesNotMatch(textoVisible(contenedor), /Motivo privado GET|rcd_pppppppppppppppppppppp/u);
    assert.equal(contenedor.querySelector("[data-dietas-borrador-detalle]"), null);
    assert.equal(contenedor.querySelector("[data-dietas-borrador-pagina]"), null);
    assert.match(textoVisible(contenedor), /rcd_0000000000000000000002/u);
    datos.motivo = "A"; await enviar(); // A: 403; conserva la clave ya confirmada.
    autorizada = true;
    await enviar(); // A: 200 replay.
    const clavesA = solicitudes.filter((solicitud) => solicitud.motivo === "A")
      .map((solicitud) => solicitud.clave_idempotencia);
    assert.deepEqual(clavesA, Array(3).fill("clave-operacion-0001"));
    assert.equal(altasPorMotivo.get("A"), 1);
    assert.match(textoVisible(contenedor), /recuperado sin crear otro borrador/u);
    datos.motivo = "D"; await enviar(); // D: 503 tras confirmar en servidor.
    assert.match(textoVisible(contenedor), /No se ha podido confirmar/u);
    await enviar(); // D: 200 replay con la misma clave.
    assert.deepEqual(solicitudes.filter((solicitud) => solicitud.motivo === "D")
      .map((solicitud) => solicitud.clave_idempotencia), Array(2).fill("clave-operacion-0004"));
    assert.equal(altasPorMotivo.get("D"), 1);
  } finally { globalThis.FormData = FormDataOriginal; vista.desmontar(); }
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

async function abrirEdicionD5(documentoComision, ejecutarEdicion) {
  const calculado = { ...item, comision: { ...item.comision, ...(documentoComision ? { documento: documentoComision } : {}),
    codigos_ruta: ["18087", "18003"], calculo: {
      rotulo: "PROVISIONAL · pendiente de confirmación por RRHH", version_tarifa: "provisional:rd462:20260923",
      version_grafo: "grafo:prueba", hora_inicio: "09:00", hora_fin: "18:00", kilometros: "12.0000",
      importe_kilometraje_centimos: 312, tramos_ruta: [{ origen_codigo: "18087", destino_codigo: "18003", kilometros: "12.0000" }],
      opciones_dieta: [1, 2, 3].map((grupo) => ({ grupo, calculo: { total_maximo_orientativo_centimos: 1000,
        tramos: [{ fecha: "2026-09-20", tipo: "manutencion", porcentaje: 50, importe_centimos: 500 },
          { fecha: "2026-09-21", tipo: "alojamiento", porcentaje: 100, importe_centimos: 500 }] } })),
    },
  } };
  const contenedor = raiz(); const peticiones = [];
  const vista = montarVistaBorradoresPropios(contenedor, {
    relacionesAutorizadas: [{ relacion_ref: item.comision.relacion_ref, unidad_ref: "unidad:uno" }],
    fechaReferenciaPersonal: "2026-09-24",
    clienteAsignacion: { obtener: async () => ({ verificada: true, asignacion_ref: "ads_1234567890123456789012",
      relacion_ref: item.comision.relacion_ref, unidad_ref: "unidad:uno", fecha_referencia: "2026-09-24",
      centro_ref: "centro:uno", administrativo_persona_ref: "per_aaaaaaaaaaaaaaaaaaaaaa",
      responsable_persona_ref: "per_bbbbbbbbbbbbbbbbbbbbbb", grupo_dieta: 2, version: 1 }) },
    cliente: { listar: async () => ({ items: [calculado] }), obtener: async () => calculado,
      crear: async () => calculado, editar: async (ref, entrada) => {
        peticiones.push(entrada); return ejecutarEdicion ? ejecutarEdicion(calculado, ref, entrada) : calculado;
      } },
    generarClaveIdempotencia: () => "editar-comision-aceptada-20260924",
    catalogoOtrosGastos: { version: "provisional:otros-gastos:20260925", tipos: [
      { codigo: "taxi", clase: "otro_medio" }, { codigo: "aparcamiento", clase: "otro_gasto" }] },
  });
  await Promise.resolve(); await Promise.resolve();
  const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
  await panel.listeners.click({ target: contenedor.querySelector("[data-dietas-borrador-detalle]") });
  await Promise.resolve(); await Promise.resolve();
  const editar = contenedor.querySelector("[data-dietas-borrador-editar]");
  assert.equal(editar.disabled, false);
  await panel.listeners.click({ target: editar });
  const form = contenedor.querySelector("[data-dietas-borrador-form]"); form.checkValidity = () => true;
  form.querySelector("[data-dietas-vehiculo-propio]").value = "no";
  return { contenedor, vista, panel, form, peticiones };
}

test("las paradas de una ruta conservan el foco al añadir, mover y quitar sin tomar el foco externo", async () => {
  const { contenedor, vista, panel, form, peticiones } = await abrirEdicionD5();
  const vehiculo = form.querySelector("[data-dietas-vehiculo-propio]");
  vehiculo.value = "si"; panel.listeners.change({ target: vehiculo });
  await panel.listeners.click({ target: form.querySelector("[data-dietas-ruta-anadir]") });
  const fila = form.querySelector("[data-dietas-ruta-linea]");
  const anadir = fila.querySelector("[data-dietas-ruta-parada-anadir]");
  const selectores = () => fila.querySelectorAll("select").filter((control) => control.name === "ruta_parada_codigo");
  const activar = async (boton) => { boton.focus(); await panel.listeners.click({ target: boton }); };
  await activar(anadir);
  assert.equal(contenedor.ownerDocument.activeElement, selectores()[0]);
  selectores()[0].value = "18061";
  await activar(anadir); selectores()[1].value = "18175";
  await activar(fila.querySelector("[data-dietas-ruta-parada-bajar]"));
  assert.deepEqual(selectores().map((control) => control.value), ["18175", "18061"]);
  assert.equal(contenedor.ownerDocument.activeElement, selectores()[1]);
  await activar(fila.querySelectorAll("[data-dietas-ruta-parada-subir]")[1]);
  assert.equal(contenedor.ownerDocument.activeElement, selectores()[0]);
  const externo = new Nodo(contenedor.ownerDocument, "button"); contenedor.append(externo); externo.focus();
  await panel.listeners.click({ target: fila.querySelector("[data-dietas-ruta-parada-bajar]") });
  assert.equal(contenedor.ownerDocument.activeElement, externo);
  await activar(fila.querySelectorAll("[data-dietas-ruta-parada-quitar]")[1]);
  assert.equal(contenedor.ownerDocument.activeElement, selectores()[0]);
  await activar(fila.querySelector("[data-dietas-ruta-parada-quitar]"));
  assert.equal(contenedor.ownerDocument.activeElement, anadir);
  assert.equal(peticiones.length, 0);
  vista.desmontar();
});

for (const tipo of ["ruta", "otro"]) {
  test(`quitar una línea de ${tipo} enfoca la vecina o Añadir y respeta el foco externo`, async () => {
    const { contenedor, vista, panel, form, peticiones } = await abrirEdicionD5();
    if (tipo === "ruta") {
      const vehiculo = form.querySelector("[data-dietas-vehiculo-propio]");
      vehiculo.value = "si"; panel.listeners.change({ target: vehiculo });
    }
    const anadir = form.querySelector(`[data-dietas-${tipo}-anadir]`);
    const filas = () => form.querySelectorAll(`[data-dietas-${tipo}-linea]`);
    const quitar = async (fila) => {
      const boton = fila.querySelector(`[data-dietas-${tipo}-quitar]`);
      boton.focus(); await panel.listeners.click({ target: boton });
    };
    for (let i = 0; i < 3; i++) await panel.listeners.click({ target: anadir });
    const [primera, intermedia, ultima] = filas();
    await quitar(intermedia);
    assert.equal(contenedor.ownerDocument.activeElement, ultima.querySelector("select"));
    await quitar(ultima);
    assert.equal(contenedor.ownerDocument.activeElement, primera.querySelector("select"));
    await quitar(primera);
    assert.equal(contenedor.ownerDocument.activeElement, anadir);
    await panel.listeners.click({ target: anadir });
    const externo = new Nodo(contenedor.ownerDocument, "button"); contenedor.append(externo); externo.focus();
    await panel.listeners.click({ target: filas()[0].querySelector(`[data-dietas-${tipo}-quitar]`) });
    assert.equal(contenedor.ownerDocument.activeElement, externo);
    assert.equal(peticiones.length, 0);
    vista.desmontar();
  });
}

test("editar conserva tramos, rutas, ajustes y justificantes tras resultado incierto y reintenta la misma operación", async () => {
  let rechazar; let intentos = 0;
  const { contenedor, vista, panel, form, peticiones } = await abrirEdicionD5(undefined, async (calculado) => {
    if (++intentos === 1) return new Promise((_resolver, rechazo) => { rechazar = rechazo; });
    return calculado;
  });
  rellenarSolicitud(form);
  const vehiculo = form.querySelector("[data-dietas-vehiculo-propio]");
  vehiculo.value = "si";
  panel.listeners.change({ target: vehiculo });
  await panel.listeners.click({ target: form.querySelector("[data-dietas-ruta-anadir]") });
  const ruta = form.querySelector("[data-dietas-ruta-linea]");
  for (const [nombre, valor] of Object.entries({ ruta_origen_codigo: "18087", ruta_destino_codigo: "18003",
    ajuste_kilometros: "1.0000", motivo_ajuste: "Desvío conservado" })) entradaFormulario(ruta, nombre).value = valor;
  // Una ruta oculta también conserva lo escrito si cambia el vehículo.
  vehiculo.value = "no";
  panel.listeners.change({ target: vehiculo });
  entradaFormulario(form, "modo_tramos").value = "uno";
  entradaFormulario(form, "tramo_indice").value = "1";
  await panel.listeners.click({ target: form.querySelector("[data-dietas-otro-anadir]") });
  const gasto = form.querySelector("[data-dietas-otro-linea]");
  for (const [nombre, valor] of Object.entries({ tipo_gasto: "aparcamiento", fecha: "2026-09-21",
    concepto: "Aparcamiento conservado", importe: "2,50", justificante_ref: "ticket:conservado",
    justificante_sha256: "a".repeat(64) })) entradaFormulario(gasto, nombre).value = valor;
  const controles = entradasFormulario(form);
  const antes = controles.map((campo) => [campo, campo.value, campo.disabled]);
  const original = globalThis.FormData;
  globalThis.FormData = DatosFormulario;
  try {
    const pendiente = panel.listeners.submit({ target: form, preventDefault() {} });
    assert.equal(peticiones.length, 1);
    assert.ok([...controles, ...form.querySelectorAll("button")].every((campo) => campo.disabled));
    const error = new Error("respuesta incierta"); error.resultadoIndeterminado = true;
    rechazar(error); await pendiente;
    assert.deepEqual(controles.map((campo) => [campo, campo.value, campo.disabled]), antes);
    assert.equal(contenedor.ownerDocument.activeElement, contenedor.querySelector("[data-dietas-borradores-estado]"));
    await panel.listeners.submit({ target: form, preventDefault() {} });
    assert.deepEqual(peticiones[1], peticiones[0]);
    assert.deepEqual(peticiones[0].tramos_aceptados, [1]);
    assert.equal(peticiones[0].otros[0].justificante_ref, "ticket:conservado");
    assert.deepEqual(controles.map((campo) => [campo, campo.value]), antes.map(([campo, valor]) => [campo, valor]));
  } finally { globalThis.FormData = original; vista.desmontar(); }
});

test("editar D3 usa grupo acreditado para aceptar tramos y D4 sin vehículo envía otros gastos D5 con justificante", async () => {
  const { contenedor, vista, panel, form, peticiones } = await abrirEdicionD5();
  await panel.listeners.click({ target: form.querySelector("[data-dietas-otro-anadir]") });
  const filaOtro = form.querySelector("[data-dietas-otro-linea]");
  const tipo = filaOtro.querySelectorAll("select").find((entrada) => entrada.name === "tipo_gasto");
  // Tras «Añadir», el foco va al primer campo de la línea: el tipo.
  assert.equal(contenedor.ownerDocument.activeElement, tipo);
  assert.equal(filaOtro.querySelector("[data-dietas-otro-quitar]").attrs["aria-label"], "Quitar línea 1");
  // Los tipos se agrupan por apartado con rótulos de negocio, sin códigos.
  assert.deepEqual(tipo.children.slice(1).map((grupo) => [grupo.label, grupo.children.map((opcion) => opcion.textContent)]),
    [["Otros medios de transporte", ["Taxi"]], ["Otros gastos", ["Aparcamiento"]]]);
  const campoOtro = (nombre) => filaOtro.querySelectorAll("input").find((entrada) => entrada.name === nombre);
  tipo.value = "aparcamiento";
  campoOtro("fecha").value = "2026-09-22";
  campoOtro("concepto").value = "Aparcamiento";
  campoOtro("importe").value = "2,50";
  campoOtro("justificante_ref").value = "ticket:parking-01";
  campoOtro("justificante_sha256").value = "A".repeat(64);
  const datos = { fecha_inicio: "2026-09-20", fecha_fin: "2026-09-21", motivo: "Visita",
    hora_inicio: "09:00", hora_fin: "18:00", origen_codigo: "18087", destino_codigo: "18003" };
  const original = globalThis.FormData;
  globalThis.FormData = class { get(nombre) { return datos[nombre] ?? null; } };
  try {
    // Un gasto fuera de las fechas de la comisión no se envía.
    await panel.listeners.submit({ target: form, preventDefault() {} });
    assert.equal(peticiones.length, 0);
    assert.match(textoVisible(contenedor), /Revise cada gasto/u);
    assert.equal(contenedor.ownerDocument.activeElement, campoOtro("fecha"));
    campoOtro("fecha").value = "2026-09-21";
    await panel.listeners.submit({ target: form, preventDefault() {} });
    assert.equal(peticiones.length, 1);
    assert.equal(peticiones[0].vehiculo_propio, false);
    assert.deepEqual(peticiones[0].rutas, []);
    assert.deepEqual(peticiones[0].tramos_aceptados, [0, 1]);
    assert.equal(peticiones[0].version_tarifa_aceptada, "provisional:rd462:20260923");
    assert.deepEqual(peticiones[0].otros, [{ tipo: "otro_gasto", tipo_gasto: "aparcamiento",
      catalogo_version: "provisional:otros-gastos:20260925", fecha: "2026-09-21", concepto: "Aparcamiento",
      importe_centimos: 250, justificante_ref: "ticket:parking-01", justificante_sha256: "a".repeat(64) }]);
  } finally { globalThis.FormData = original; vista.desmontar(); }
});

test("editar una comisión con gastos anteriores a D5 los marca y al guardar lleva el foco al primer campo que falta", async () => {
  const { contenedor, vista, panel, form, peticiones } = await abrirEdicionD5({ grupo_dieta: 2, manutencion_centimos: 0,
    alojamiento_tope_centimos: 0, kilometraje_centimos: 0, otros_centimos: 310, total_orientativo_centimos: 310,
    lineas: [{ tipo: "otro_gasto", concepto: "Peaje anterior", importe_centimos: 310 }] });
  const filaOtro = form.querySelector("[data-dietas-otro-linea]");
  assert.match(textoVisible(filaOtro.querySelector("[data-dietas-otro-anterior]")), /Faltan el tipo, la fecha o el justificante/u);
  assert.equal(filaOtro.querySelector("[data-dietas-otro-quitar]").attrs["aria-label"], "Quitar línea 1");
  form.querySelector("[data-dietas-vehiculo-propio]").value = "no";
  const datos = { fecha_inicio: "2026-09-20", fecha_fin: "2026-09-21", motivo: "Visita",
    hora_inicio: "09:00", hora_fin: "18:00", origen_codigo: "18087", destino_codigo: "18003" };
  const original = globalThis.FormData;
  globalThis.FormData = class { get(nombre) { return datos[nombre] ?? null; } };
  try {
    await panel.listeners.submit({ target: form, preventDefault() {} });
    assert.equal(peticiones.length, 0);
    const tipo = filaOtro.querySelectorAll("select").find((entrada) => entrada.name === "tipo_gasto");
    assert.equal(contenedor.ownerDocument.activeElement, tipo);
    assert.equal(tipo.attrs["aria-invalid"], "true");
  } finally { globalThis.FormData = original; vista.desmontar(); }
});
