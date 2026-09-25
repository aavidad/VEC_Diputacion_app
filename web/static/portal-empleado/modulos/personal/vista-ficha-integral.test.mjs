import assert from "node:assert/strict";
import test from "node:test";
import { exigirVersiones, posterior } from "../../versiones-cache.test-helper.mjs";
import { readFileSync } from "node:fs";
import { montarVistaFichaIntegralPersonal } from "./vista-ficha-integral.js";

test("la ficha carga el catálogo i18n del corte F2 con versión de caché", () => {
  const codigo = readFileSync(new URL("./vista-ficha-integral.js", import.meta.url), "utf8");
  // i18n de Personal renovado (organización histórica): nunca la URL immutable previa.
  exigirVersiones(codigo, "./i18n.js", posterior("20260924-f2-web2"));
});

function raizFalsa() {
  class Nodo {
    constructor(documento, etiqueta = "div") { this.ownerDocument = documento; this.tagName = etiqueta; this.children = []; this.dataset = {}; this.listeners = new Map(); this.parent = null; this.textContent = ""; this.atributos = new Map(); this.disabled = false; }
    append(...hijos) { this.children.push(...hijos); hijos.forEach((hijo) => { hijo.parent = this; }); }
    replaceChildren(...hijos) { this.children = []; this.append(...hijos); }
    remove() { if (this.parent) this.parent.children = this.parent.children.filter((hijo) => hijo !== this); }
    addEventListener(tipo, fn) { this.listeners.set(tipo, fn); }
    setAttribute(clave, valor) { this.atributos.set(clave, valor); }
    focus() { this.enfocado = true; }
    matches(selector) { const coincide = selector.match(/^\[data-([a-z-]+)(?:="([a-z_-]+)")?\]$/u); if (!coincide) return false; const clave = coincide[1].replace(/-([a-z])/g, (_m, letra) => letra.toUpperCase()); return this.dataset[clave] !== undefined && (coincide[2] === undefined || this.dataset[clave] === coincide[2]); }
    querySelector(selector) { if (this.matches(selector)) return this; for (const hijo of this.children) { const encontrado = hijo.querySelector(selector); if (encontrado) return encontrado; } return null; }
  }
  const documento = { createElement: (etiqueta) => new Nodo(documento, etiqueta) };
  return new Nodo(documento, "root");
}
function nodos(n) { return [n, ...n.children.flatMap(nodos)]; }
function texto(n) { return nodos(n).map((item) => item.textContent).join(" "); }
function tab(ficha, clave) { return nodos(ficha).find((n) => n.dataset.personalFichaTab === clave); }
const completar = () => new Promise((resolve) => setImmediate(resolve));

test("la portada no fabrica persona, relación, curso, fichaje ni nómina", () => {
  const raiz = raizFalsa(); montarVistaFichaIntegralPersonal({ raiz }); const ficha = raiz.querySelector("[data-personal-ficha-integral]");
  assert.ok(ficha); assert.equal(tab(ficha, "tiempo").textContent, "Tiempo");
  assert.match(texto(ficha), /Abra un apartado para consultar su fuente propia/);
  assert.match(texto(ficha), /No se muestran nombre, empleado, puesto/);
  assert.equal(nodos(ficha).filter((n) => n.dataset.personalFichaEstado === "no_configurado").length, 6);
  assert.doesNotMatch(texto(ficha), /Antonio López|Funcionario de carrera|Junio 2026|Nómina orientativa/);
  const ayuda = raiz.querySelector("[data-personal-ficha-ayuda]");
  assert.equal(ayuda.tagName, "details");
  assert.equal(ayuda.children[0].tagName, "summary");
  assert.equal(ayuda.children[0].textContent, "?");
  assert.equal(ayuda.children[0].atributos.get("aria-label"), "? Ayuda sobre esta ficha");
  assert.equal(ayuda.children[0].atributos.get("tabindex"), "0");
  assert.equal(ayuda.open, false);
  assert.ok(nodos(ficha).filter((n) => n.dataset.personalFichaDestino).every((n) => n.disabled));
});

test("la ayuda contextual permanece tras ? sin ocultar estados ni iniciar consultas", async () => {
  const raiz = raizFalsa(); let consultas = 0; let navegaciones = 0;
  montarVistaFichaIntegralPersonal({ raiz, navegarModulo: () => { navegaciones += 1; }, fuentes: {
    servicios: { consultarPropios() { consultas += 1; return { estado: "vacio", fuente: "Personal", actualizado_en: "2026-09-24T08:00:00Z", items: [] }; } },
  } });
  const ficha = raiz.querySelector("[data-personal-ficha-integral]");
  const ayuda = raiz.querySelector("[data-personal-ficha-ayuda]");
  const principal = nodos(ficha).find((n) => n.className === "personal-ficha-principal");
  assert.match(texto(ayuda), /La navegación no envía identificadores/);
  assert.doesNotMatch(texto(principal), /La navegación no envía identificadores/);
  ayuda.children[0].focus(); ayuda.open = true;
  assert.equal(ayuda.children[0].enfocado, true);
  assert.equal(consultas, 0); assert.equal(navegaciones, 0);
  tab(ficha, "servicios").listeners.get("click")();
  assert.equal(ayuda.open, false);
  assert.match(texto(ayuda), /Periodos reconocidos, procedencia/);
  assert.doesNotMatch(texto(principal), /Periodos reconocidos, procedencia/);
  assert.match(texto(ficha), /Consultando este apartado/);
  await completar();
  assert.equal(consultas, 1);
  assert.match(texto(ficha), /Fuente: Personal/);
  assert.match(texto(ficha), /no devuelve registros/);
});

test("cada apartado se consulta solo al abrirlo y conserva procedencia sin referencias en la petición", async () => {
  const raiz = raizFalsa(); const llamadas = [];
  montarVistaFichaIntegralPersonal({ raiz, fuentes: {
    servicios: { consultarPropios(entrada) { llamadas.push(entrada); return { estado: "disponible", fuente: "Personal", actualizado_en: "2026-09-24T08:00:00Z", items: [{ desde: "2020-01-01", procedencia: "Diputación", reconocimiento: "Confirmado", estado: "Reconocido" }] }; } },
    tiempo: { consultarPropios() { throw new Error("no debe abrirse"); } },
  } });
  const ficha = raiz.querySelector("[data-personal-ficha-integral]"); assert.equal(llamadas.length, 0);
  assert.doesNotMatch(texto(ficha), /No se muestran nombre, empleado, puesto/);
  tab(ficha, "servicios").listeners.get("click")(); await completar();
  assert.equal(llamadas.length, 1); assert.deepEqual(Object.keys(llamadas[0]), ["signal"]);
  assert.match(texto(ficha), /Fuente: Personal/); assert.match(texto(ficha), /Reconocido/); assert.match(texto(ficha), /1 ene 2020/);
  assert.doesNotMatch(texto(ficha), /curso acreditado|trienio concedido|Entrada a las 08:00/i);
});

test("las seis capacidades conservan su estado y nunca muestran filas de otra fuente", async () => {
  const raiz = raizFalsa(); const llamadas = []; const campos = {
    relaciones: "puesto", servicios: "procedencia", tiempo: "tipo",
    formacion: "curso", economia: "documento", documentos: "documento",
  };
  const fuentes = Object.fromEntries(Object.entries(campos).map(([clave, campo]) => [clave, { consultarPropios() {
    llamadas.push(clave); return { estado: "disponible", fuente: `Fuente ${clave}`, actualizado_en: "2026-09-24T08:00:00Z", items: [{ [campo]: `Dato ${clave}` }] };
  } }]));
  montarVistaFichaIntegralPersonal({ raiz, fuentes }); const ficha = raiz.querySelector("[data-personal-ficha-integral]");
  assert.deepEqual(llamadas, []);
  for (const clave of Object.keys(campos)) {
    tab(ficha, clave).listeners.get("click")(); await completar();
    assert.match(texto(ficha), new RegExp(`Dato ${clave}`));
    for (const ajena of Object.keys(campos).filter((otra) => otra !== clave)) assert.doesNotMatch(texto(ficha), new RegExp(`Dato ${ajena}`));
  }
  assert.deepEqual(llamadas, Object.keys(campos));
  tab(ficha, "ficha").listeners.get("click")();
  assert.equal(nodos(ficha).filter((n) => n.dataset.personalFichaEstado === "disponible").length, 6);
  assert.match(texto(ficha), /Datos en la última consulta/);
  assert.doesNotMatch(texto(ficha), /Dato relaciones|Dato tiempo/);
});

test("una capacidad heredada no habilita ninguna consulta propia", () => {
  const raiz = raizFalsa(); let lecturas = 0;
  const fuentes = Object.create({ servicios: { consultarPropios() { lecturas += 1; } } });
  montarVistaFichaIntegralPersonal({ raiz, fuentes }); const ficha = raiz.querySelector("[data-personal-ficha-integral]");
  tab(ficha, "servicios").listeners.get("click")();
  assert.equal(lecturas, 0); assert.match(texto(ficha), /No hay una fuente propia autorizada conectada/);
});

test("estados separados: fuente ausente, vacío autorizado, denegado y error", async () => {
  const raiz = raizFalsa(); const avisos = [];
  montarVistaFichaIntegralPersonal({ raiz, anunciar: (...args) => avisos.push(args), fuentes: {
    servicios: { consultarPropios: () => ({ estado: "vacio", fuente: "Personal", actualizado_en: "2026-09-24T08:00:00Z", items: [] }) },
    tiempo: { consultarPropios: () => ({ estado: "denegado", items: [{ tipo: "oculto" }] }) },
    formacion: { consultarPropios: () => Promise.reject(new Error("detalle interno")) },
    documentos: { consultarPropios: () => ({ estado: "disponible", fuente: "Archivo", actualizado_en: "2026-09-24T08:00:00Z", items: [{}] }) },
  } });
  const ficha = raiz.querySelector("[data-personal-ficha-integral]");
  tab(ficha, "economia").listeners.get("click")(); assert.match(texto(ficha), /No hay una fuente propia autorizada conectada/);
  tab(ficha, "servicios").listeners.get("click")(); await completar(); assert.match(texto(ficha), /no devuelve registros/);
  tab(ficha, "tiempo").listeners.get("click")(); await completar(); assert.match(texto(ficha), /No tiene permiso/); assert.doesNotMatch(texto(ficha), /oculto/);
  tab(ficha, "formacion").listeners.get("click")(); await completar(); assert.match(texto(ficha), /No se pudo consultar/); assert.doesNotMatch(texto(ficha), /detalle interno/); assert.equal(avisos.length, 1);
  tab(ficha, "documentos").listeners.get("click")(); await completar(); assert.match(texto(ficha), /No se pudo consultar/); assert.doesNotMatch(texto(ficha), /No consta.*No consta/);
  tab(ficha, "ficha").listeners.get("click")();
  assert.equal(nodos(ficha).find((n) => n.dataset.personalFichaEstado === "denegado") !== undefined, true);
  assert.equal(nodos(ficha).filter((n) => n.dataset.personalFichaEstado === "error").length, 2);
});

test("cambiar de pestaña y desmontar aborta consultas sin pintar respuestas tardías", async () => {
  const raiz = raizFalsa(); let resolver; let senal;
  const montaje = montarVistaFichaIntegralPersonal({ raiz, fuentes: { relaciones: { consultarPropios({ signal }) { senal = signal; return new Promise((resolve) => { resolver = resolve; }); } } } });
  const ficha = raiz.querySelector("[data-personal-ficha-integral]"); tab(ficha, "relaciones").listeners.get("click")(); await completar();
  tab(ficha, "servicios").listeners.get("click")(); assert.equal(senal.aborted, true);
  resolver({ estado: "disponible", fuente: "Personal", actualizado_en: "2026-09-24T08:00:00Z", items: [{ puesto: "No visible" }] }); await completar();
  assert.doesNotMatch(texto(ficha), /No visible/);
  tab(ficha, "ficha").listeners.get("click")();
  assert.equal(nodos(ficha).find((n) => n.dataset.personalFichaEstado === "sin_consulta") !== undefined, true);
  montaje.desmontar(); assert.equal(raiz.querySelector("[data-personal-ficha-integral]"), null);
});

test("teclado y navegación a otros módulos no transportan identidad", () => {
  const raiz = raizFalsa(); const destinos = [];
  montarVistaFichaIntegralPersonal({ raiz, navegarModulo: (...args) => destinos.push(args), destinosDisponibles: { dietas: true } }); const ficha = raiz.querySelector("[data-personal-ficha-integral]");
  let prevenido = false; tab(ficha, "ficha").listeners.get("keydown")({ key: "ArrowRight", preventDefault() { prevenido = true; } });
  assert.equal(prevenido, true); assert.equal(tab(ficha, "relaciones").atributos.get("aria-selected"), "true"); assert.equal(tab(ficha, "relaciones").enfocado, true);
  tab(ficha, "relaciones").listeners.get("keydown")({ key: "End", preventDefault() {} });
  assert.equal(tab(ficha, "catalogos").atributos.get("aria-selected"), "true"); assert.equal(tab(ficha, "catalogos").enfocado, true);
  tab(ficha, "catalogos").listeners.get("keydown")({ key: "Home", preventDefault() {} });
  assert.equal(tab(ficha, "ficha").atributos.get("aria-selected"), "true"); assert.equal(tab(ficha, "ficha").enfocado, true);
  tab(ficha, "ficha").listeners.get("click")();
  const dietas = nodos(ficha).find((n) => n.dataset.personalFichaDestino === "dietas");
  const cronos = nodos(ficha).find((n) => n.dataset.personalFichaDestino === "cronos");
  assert.equal(dietas.disabled, false); assert.equal(cronos.disabled, true);
  dietas.listeners.get("click")(); cronos.listeners.get("click")();
  assert.deepEqual(destinos, [["dietas"]]);
});

test("un callback de navegación no habilita por sí solo Dietas ni Cronos", () => {
  const raiz = raizFalsa(); const destinos = [];
  montarVistaFichaIntegralPersonal({ raiz, navegarModulo: (destino) => destinos.push(destino) });
  const ficha = raiz.querySelector("[data-personal-ficha-integral]");
  for (const destino of ["dietas", "cronos"]) {
    const boton = nodos(ficha).find((n) => n.dataset.personalFichaDestino === destino);
    assert.equal(boton.disabled, true); assert.match(boton.title, /no está montada/i);
    boton.listeners.get("click")();
  }
  assert.deepEqual(destinos, []);
});

test("disponibilidad heredada o no booleana no habilita destinos", () => {
  const raiz = raizFalsa(); const destinos = [];
  const destinosDisponibles = Object.create({ dietas: true }); destinosDisponibles.cronos = "true";
  montarVistaFichaIntegralPersonal({ raiz, navegarModulo: (destino) => destinos.push(destino), destinosDisponibles });
  const ficha = raiz.querySelector("[data-personal-ficha-integral]");
  const botones = nodos(ficha).filter((n) => n.dataset.personalFichaDestino);
  assert.ok(botones.every((boton) => boton.disabled && boton.atributos.get("aria-disabled") === "true"));
  botones.forEach((boton) => boton.listeners.get("click")()); assert.deepEqual(destinos, []);
});

test("en el portal real no se ofrecen apartados sin fuente ni textos explicativos", async () => {
  const raiz = raizFalsa(); let montajes = 0;
  montarVistaFichaIntegralPersonal({ raiz, ocultarSinFuente: true, montarCatalogos: () => { montajes += 1; return { desmontar() {} }; } });
  const ficha = raiz.querySelector("[data-personal-ficha-integral]");
  const pestanas = nodos(ficha).filter((n) => n.dataset.personalFichaTab).map((n) => n.dataset.personalFichaTab);
  assert.deepEqual(pestanas, ["ficha", "catalogos"]);
  assert.equal(tab(ficha, "catalogos").atributos.get("aria-selected"), "true", "sin apartados propios se abre Catálogos");
  await completar(); assert.equal(montajes, 1);
  assert.doesNotMatch(texto(ficha), /se consultan por separado/);
  tab(ficha, "ficha").listeners.get("click")();
  assert.equal(nodos(ficha).filter((n) => n.dataset.personalFichaEstado).length, 0);
  assert.doesNotMatch(texto(ficha), /Abra un apartado|No se muestran nombre/);
  assert.ok(nodos(ficha).some((n) => n.dataset.personalFichaDestino === "cronos"));
  tab(ficha, "ficha").listeners.get("keydown")({ key: "ArrowRight", preventDefault() {} });
  assert.equal(tab(ficha, "catalogos").atributos.get("aria-selected"), "true");
});

test("en el portal real un apartado con fuente sí se ofrece y la ficha abre primero", () => {
  const raiz = raizFalsa();
  montarVistaFichaIntegralPersonal({ raiz, ocultarSinFuente: true, montarCatalogos: () => ({ desmontar() {} }), fuentes: {
    servicios: { consultarPropios: () => ({ estado: "vacio", fuente: "Personal", actualizado_en: "2026-09-24T08:00:00Z", items: [] }) },
  } });
  const ficha = raiz.querySelector("[data-personal-ficha-integral]");
  assert.deepEqual(nodos(ficha).filter((n) => n.dataset.personalFichaTab).map((n) => n.dataset.personalFichaTab), ["ficha", "servicios", "catalogos"]);
  assert.equal(tab(ficha, "ficha").atributos.get("aria-selected"), "true");
  assert.deepEqual(nodos(ficha).filter((n) => n.dataset.personalFichaEstado).map((n) => n.dataset.personalFichaEstado), ["sin_consulta"]);
  assert.throws(() => montarVistaFichaIntegralPersonal({ raiz: raizFalsa(), ocultarSinFuente: "si" }), /no disponible/);
});

test("catálogos existentes se montan bajo demanda y se limpian al salir", async () => {
  const raiz = raizFalsa(); let montajes = 0; let limpiezas = 0;
  montarVistaFichaIntegralPersonal({ raiz, montarCatalogos: ({ registrarDesmontar }) => {
    montajes += 1; registrarDesmontar(() => { limpiezas += 1; }); return { desmontar() { limpiezas += 1; } };
  } });
  const ficha = raiz.querySelector("[data-personal-ficha-integral]"); tab(ficha, "catalogos").listeners.get("click")(); await completar();
  assert.equal(montajes, 1); assert.match(texto(ficha), /se consultan por separado/);
  tab(ficha, "ficha").listeners.get("click")(); assert.ok(limpiezas >= 1);
});
