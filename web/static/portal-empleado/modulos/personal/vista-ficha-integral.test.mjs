import assert from "node:assert/strict";
import test from "node:test";
import { prepararTextosPersonal } from "./i18n.js?v=20261007-pantallas-textos-final-v1";

test.before(async () => { await prepararTextosPersonal(); });
import { exigirVersiones, posterior } from "../../versiones-cache.test-helper.mjs";
import { readFileSync } from "node:fs";
import { montarVistaFichaIntegralPersonal } from "./vista-ficha-integral.js";

test("la ficha carga el catálogo i18n del corte F2 con versión de caché", () => {
  const codigo = readFileSync(new URL("./vista-ficha-integral.js", import.meta.url), "utf8");
  // i18n de Personal renovado (organización histórica): nunca la URL immutable previa.
  exigirVersiones(codigo, "./i18n.js", posterior("20260924-f2-web2"));
});

function raizFalsa({ geometriaPestanas = false } = {}) {
  class Nodo {
    constructor(documento, etiqueta = "div") {
      this.ownerDocument = documento; this.tagName = etiqueta; this.children = []; this.dataset = {};
      this.listeners = new Map(); this.parent = null; this.textContent = ""; this.atributos = new Map(); this.disabled = false;
      this.scrollLeft = 0; this.clientLeft = 0;
      if (documento.geometriaPestanas) this.getBoundingClientRect = () => {
        if (this.className === "personal-ficha-pestanas") return { left: 24, right: 366 };
        const origen = { ficha: [28, 94], servicios: [170, 266], catalogos: [439, 531] }[this.dataset.personalFichaTab];
        return origen ? { left: origen[0] - this.parent.scrollLeft, right: origen[1] - this.parent.scrollLeft }
          : { left: 0, right: 0 };
      };
    }
    append(...hijos) { this.children.push(...hijos); hijos.forEach((hijo) => { hijo.parent = this; }); }
    replaceChildren(...hijos) { this.children = []; this.append(...hijos); }
    remove() { if (this.parent) this.parent.children = this.parent.children.filter((hijo) => hijo !== this); }
    addEventListener(tipo, fn) { this.listeners.set(tipo, fn); }
    setAttribute(clave, valor) { this.atributos.set(clave, valor); }
    click() { this.listeners.get("click")?.(); }
    focus(opciones) { this.enfocado = true; this.opcionesFoco = opciones; this.ownerDocument.activeElement = this; }
    matches(selector) { const coincide = selector.match(/^\[data-([a-z-]+)(?:="([a-z_-]+)")?\]$/u); if (!coincide) return false; const clave = coincide[1].replace(/-([a-z])/g, (_m, letra) => letra.toUpperCase()); return this.dataset[clave] !== undefined && (coincide[2] === undefined || this.dataset[clave] === coincide[2]); }
    querySelector(selector) { if (this.matches(selector)) return this; for (const hijo of this.children) { const encontrado = hijo.querySelector(selector); if (encontrado) return encontrado; } return null; }
  }
  const documento = { createElement: (etiqueta) => new Nodo(documento, etiqueta), activeElement: null, tieneFoco: true, geometriaPestanas, hasFocus() { return this.tieneFoco; } };
  return new Nodo(documento, "root");
}
function nodos(n) { return [n, ...n.children.flatMap(nodos)]; }
function texto(n) { return nodos(n).map((item) => item.textContent).join(" "); }
function tab(ficha, clave) { return nodos(ficha).find((n) => n.dataset.personalFichaTab === clave); }
const completar = () => new Promise((resolve) => setImmediate(resolve));

test("Contacto consulta su fuente sólo al abrir la pestaña y limpia una vez al salir", async () => {
  const raiz = raizFalsa(); let consultas = 0, limpiezas = 0;
  montarVistaFichaIntegralPersonal({ raiz, montarContacto(entrada) {
    consultas += 1; const limpiar = () => { limpiezas += 1; }; entrada.registrarDesmontar(limpiar);
    return { desmontar: limpiar };
  } });
  assert.equal(consultas, 0);
  const ficha = raiz.querySelector("[data-personal-ficha-integral]"); tab(ficha, "contacto").click(); await completar();
  assert.equal(consultas, 1); tab(ficha, "ficha").click(); await completar(); assert.equal(limpiezas, 1);
});

test("una sesión caducada en Contacto cierra toda la ficha y retira sus fuentes cacheadas", async () => {
  const raiz = raizFalsa(); let caducar, invalidaciones = 0, limpiezas = 0, lecturas = 0;
  montarVistaFichaIntegralPersonal({ raiz, fuentes: {
    servicios: { consultarPropios() { lecturas += 1; return { estado: "disponible", fuente: "Personal", actualizado_en: "2026-10-04T00:00:00Z", items: [{ procedencia: "Periodo reconocido" }] }; }, actualizar() { invalidaciones += 1; } },
  }, montarContacto(entrada) { caducar = entrada.alCaducarSesion; const limpiar = () => { limpiezas += 1; }; entrada.registrarDesmontar(limpiar); return { desmontar: limpiar }; } });
  const ficha = raiz.querySelector("[data-personal-ficha-integral]");
  const servicios = tab(ficha, "servicios"); servicios.click(); await completar(); assert.match(texto(raiz), /Periodo reconocido/);
  tab(ficha, "contacto").click(); await completar(); caducar();
  assert.equal(raiz.querySelector("[data-personal-ficha-integral]"), null); assert.equal(invalidaciones, 1); assert.equal(limpiezas, 1);
  assert.doesNotMatch(texto(raiz), /Periodo reconocido/); assert.equal(raiz.children[0].atributos.get("role"), "alert");
  assert.match(texto(raiz), /Su sesión ha caducado. Identifíquese de nuevo para volver a ver su ficha/u);
  assert.equal(raiz.ownerDocument.activeElement, raiz.children[0]); servicios.click(); await completar(); assert.equal(lecturas, 1);
});

test("el acceso a correos abre su vista existente sin consultar ni trasladar datos de Personal", () => {
  const raiz = raizFalsa(); let aperturas = 0; let consultas = 0;
  montarVistaFichaIntegralPersonal({ raiz, abrirCorreos: () => { aperturas += 1; }, fuentes: {
    servicios: { consultarPropios() { consultas += 1; throw new Error("no debe consultar"); } },
  } });
  const boton = raiz.querySelector("[data-personal-ficha-correos]");
  assert.equal(boton.textContent, "Ver mis correos");
  assert.match(boton.title, /Mis preferencias/);
  boton.focus(); boton.click();
  assert.equal(aperturas, 1); assert.equal(consultas, 0);
  assert.doesNotMatch(texto(raiz), /@/);
  const sinNavegacion = raizFalsa(); montarVistaFichaIntegralPersonal({ raiz: sinNavegacion });
  assert.equal(sinNavegacion.querySelector("[data-personal-ficha-correos]"), null);
});

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

test("un fallo inicial conserva relaciones y servicios; Reintentar y Actualizar recuperan con foco", async () => {
  const raiz = raizFalsa(); let llamadas = 0; let revision = 0;
  const grupoActualizacion = {};
  const fuente = {
    estadoInicial: "error",
    grupoActualizacion,
    actualizar() { revision += 1; },
    consultarPropios() {
      llamadas += 1;
      if (revision === 0 || revision === 2) return { estado: "error" };
      return { estado: "disponible", fuente: "Registro de Personal", actualizado_en: "2026-09-24T08:00:00Z", items: [{ desde: "2026-01-01", puesto: `Puesto ${revision}` }] };
    },
  };
  montarVistaFichaIntegralPersonal({ raiz, ocultarSinFuente: true, fuentes: { relaciones: fuente, servicios: { ...fuente, consultarPropios: () => ({ estado: "error" }) } } });
  const ficha = raiz.querySelector("[data-personal-ficha-integral]");
  assert.deepEqual(nodos(ficha).filter((n) => n.dataset.personalFichaTab).map((n) => n.dataset.personalFichaTab), ["ficha", "relaciones", "servicios"]);
  assert.equal(nodos(ficha).filter((n) => n.dataset.personalFichaEstado === "error").length, 2);
  tab(ficha, "relaciones").listeners.get("click")(); await completar();
  assert.match(texto(ficha), /No se pudo consultar/);
  let accion = ficha.querySelector('[data-personal-ficha-actualizar="relaciones"]');
  assert.equal(accion.textContent, "Reintentar consulta");
  accion.listeners.get("click")();
  assert.match(texto(ficha), /Consultando este apartado/);
  assert.doesNotMatch(texto(ficha), /Puesto 1/);
  await completar();
  accion = ficha.querySelector('[data-personal-ficha-actualizar="relaciones"]');
  assert.equal(accion.textContent, "Actualizar datos");
  assert.equal(accion.enfocado, true);
  assert.equal(raiz.ownerDocument.activeElement, accion);
  assert.match(texto(ficha), /Puesto 1/);
  tab(ficha, "ficha").listeners.get("click")();
  assert.deepEqual(nodos(ficha).filter((n) => n.dataset.personalFichaEstado).map((n) => n.dataset.personalFichaEstado), ["disponible", "sin_consulta"]);
  tab(ficha, "relaciones").listeners.get("click")(); await completar();
  accion = ficha.querySelector('[data-personal-ficha-actualizar="relaciones"]');
  accion.listeners.get("click")(); await completar();
  assert.doesNotMatch(texto(ficha), /Puesto 1/, "el fallo posterior no muestra datos anteriores");
  accion = ficha.querySelector('[data-personal-ficha-actualizar="relaciones"]');
  assert.equal(accion.textContent, "Reintentar consulta");
  assert.equal(accion.enfocado, true);
  accion.listeners.get("click")(); await completar();
  assert.match(texto(ficha), /Puesto 3/);
  assert.equal(llamadas, 5);
});

test("el reintento conserva el foco del usuario y una denegación lo deja en el panel", async () => {
  const raiz = raizFalsa(); let resolver; let consultas = 0;
  montarVistaFichaIntegralPersonal({ raiz, fuentes: { relaciones: {
    estadoInicial: "error",
    actualizar() {},
    consultarPropios() {
      consultas += 1;
      if (consultas === 1) return { estado: "error" };
      return new Promise((resolve) => { resolver = resolve; });
    },
  } } });
  const ficha = raiz.querySelector("[data-personal-ficha-integral]");
  const documento = raiz.ownerDocument;
  const principal = nodos(ficha).find((n) => n.className === "personal-ficha-principal");
  const ayuda = ficha.querySelector("[data-personal-ficha-ayuda]").children[0];
  tab(ficha, "relaciones").listeners.get("click")(); await completar();
  let accion = ficha.querySelector('[data-personal-ficha-actualizar="relaciones"]');
  accion.focus(); accion.listeners.get("click")();
  assert.equal(documento.activeElement, principal, "la carga conserva un foco útil");
  await completar(); ayuda.focus();
  resolver({ estado: "disponible", fuente: "Personal", actualizado_en: "2026-09-24T08:00:00Z", items: [{ puesto: "Nuevo" }] });
  await completar();
  assert.equal(documento.activeElement, ayuda, "la respuesta no recupera el foco si se abrió Ayuda");

  accion = ficha.querySelector('[data-personal-ficha-actualizar="relaciones"]');
  accion.focus(); accion.listeners.get("click")(); await completar();
  documento.tieneFoco = false;
  resolver({ estado: "disponible", fuente: "Personal", actualizado_en: "2026-09-24T08:00:00Z", items: [{ puesto: "Otro" }] });
  await completar();
  assert.equal(documento.activeElement, principal, "una ventana sin foco no recibe foco nuevo");
  documento.tieneFoco = true;

  accion = ficha.querySelector('[data-personal-ficha-actualizar="relaciones"]');
  accion.focus(); accion.listeners.get("click")(); await completar();
  resolver({ estado: "denegado" }); await completar();
  assert.equal(ficha.querySelector('[data-personal-ficha-actualizar="relaciones"]'), null);
  assert.equal(documento.activeElement, principal, "sin acción permitida queda el foco en el panel");
  assert.match(texto(ficha), /No tiene permiso/);
});

test("más filas de las que se muestran: estado propio visible, no desaparece; celdas hasta 300", async () => {
  const raiz = raizFalsa(); const fila = (n) => ({ desde: "2026-01-01", hasta: "Actualidad", regimen: "R".repeat(n) });
  montarVistaFichaIntegralPersonal({ raiz, ocultarSinFuente: true, fuentes: {
    relaciones: { consultarPropios: () => ({ estado: "excede_limite" }) },
    servicios: { consultarPropios: () => ({ estado: "disponible", fuente: "Registro de Personal", actualizado_en: "2026-09-24T08:00:00Z", items: [{ desde: "2026-01-01", procedencia: "S".repeat(300) }] }) },
  } });
  const ficha = raiz.querySelector("[data-personal-ficha-integral]");
  tab(ficha, "relaciones").listeners.get("click")(); await completar();
  assert.match(texto(ficha), /más registros de los que se pueden mostrar/);
  tab(ficha, "servicios").listeners.get("click")(); await completar();
  assert.ok(texto(ficha).includes("S".repeat(300)), "una celda de 300 caracteres se admite");
  tab(ficha, "ficha").listeners.get("click")();
  const bloque = nodos(ficha).find((n) => n.dataset.personalFichaEstado === "excede_limite");
  assert.ok(bloque, "el apartado sigue en la portada con su estado");
  assert.match(texto(bloque), /Demasiados registros/);
  const otra = raizFalsa();
  montarVistaFichaIntegralPersonal({ raiz: otra, fuentes: { relaciones: { consultarPropios: () => ({ estado: "disponible", fuente: "P", actualizado_en: "2026-09-24T08:00:00Z", items: [fila(301)] }) } } });
  const fichaOtra = otra.querySelector("[data-personal-ficha-integral]");
  tab(fichaOtra, "relaciones").listeners.get("click")(); await completar();
  assert.match(texto(fichaOtra), /No se pudo consultar/);
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
  assert.equal(dietas.disabled, false); assert.equal(cronos, undefined, "un módulo no ofrecido no se pinta");
  dietas.listeners.get("click")();
  assert.deepEqual(destinos, [["dietas"]]);
});

test("un callback de navegación no habilita por sí solo Dietas ni Cronos", () => {
  const raiz = raizFalsa(); const destinos = [];
  montarVistaFichaIntegralPersonal({ raiz, navegarModulo: (destino) => destinos.push(destino) });
  const ficha = raiz.querySelector("[data-personal-ficha-integral]");
  assert.equal(nodos(ficha).filter((n) => n.dataset.personalFichaDestino).length, 0, "sin disponibilidad no se ofrece ningún destino");
  assert.deepEqual(destinos, []);
});

test("un destino del catálogo aún no disponible se ofrece desactivado", () => {
  const raiz = raizFalsa(); const destinos = [];
  montarVistaFichaIntegralPersonal({ raiz, navegarModulo: (destino) => destinos.push(destino), destinosDisponibles: { dietas: false, cronos: false } });
  const ficha = raiz.querySelector("[data-personal-ficha-integral]");
  for (const destino of ["dietas", "cronos"]) {
    const boton = nodos(ficha).find((n) => n.dataset.personalFichaDestino === destino);
    assert.equal(boton.disabled, true); assert.match(boton.title, /no está montada/i);
    boton.listeners.get("click")();
  }
  assert.deepEqual(destinos, []);
});

test("Cronos y Dietas ocultos por el despliegue no aparecen en Mi ficha", () => {
  const raiz = raizFalsa();
  montarVistaFichaIntegralPersonal({ raiz, navegarModulo: () => {}, destinosDisponibles: {}, ocultarSinFuente: true });
  const ficha = raiz.querySelector("[data-personal-ficha-integral]");
  tab(ficha, "ficha").listeners.get("click")();
  assert.equal(nodos(ficha).filter((n) => n.dataset.personalFichaDestino).length, 0);
  assert.doesNotMatch(texto(ficha), /Cronos|Dietas/);
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
  assert.equal(nodos(ficha).some((n) => n.dataset.personalFichaDestino === "cronos"), false, "sin catálogo no se ofrece Cronos");
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


test("Servicios selecciona fecha, anuncia el corte real y elimina filas durante carga o denegación", async () => {
  const raiz = raizFalsa(); const llamadas = []; let resolver;
  montarVistaFichaIntegralPersonal({ raiz, fuentes: { servicios: {
    seleccionarFecha: true, fechaReferencia: "2026-09-25", actualizar() {},
    consultarPropios({ fechaReferencia, signal }) {
      llamadas.push({ fechaReferencia, signal });
      if (llamadas.length === 1) return { estado: "disponible", fuente: "Personal", actualizado_en: "2026-09-25T08:00:00Z", fecha_referencia: "2026-09-25", items: [{ procedencia: "Servicios propios" }] };
      return new Promise((r) => { resolver = r; });
    },
  } } });
  const ficha = raiz.querySelector("[data-personal-ficha-integral]");
  tab(ficha, "servicios").listeners.get("click")(); await completar();
  assert.match(texto(ficha), /Servicios a fecha de 25 sept 2026/);
  assert.match(texto(ficha), /Esta fecha se aplica solo a Servicios/);
  const fecha = ficha.querySelector("[data-personal-ficha-fecha]");
  assert.equal(fecha.value, "2026-09-25");
  const form = ficha.querySelector("[data-personal-ficha-corte]");
  fecha.value = "2020-02-29"; form.listeners.get("submit")({ preventDefault() {} }); await completar();
  assert.equal(llamadas[1].fechaReferencia, "2020-02-29");
  assert.doesNotMatch(texto(ficha), /Servicios propios|Servicios a fecha de/);
  resolver({ estado: "denegado" }); await completar();
  assert.doesNotMatch(texto(ficha), /Servicios propios|Servicios a fecha de/);
  assert.equal(ficha.querySelector("[data-personal-ficha-corte]"), null);
});

test("un nuevo corte cancela el anterior y descarta su respuesta aunque ignore abort", async () => {
  const raiz = raizFalsa(); const pendientes = [];
  montarVistaFichaIntegralPersonal({ raiz, fuentes: { servicios: {
    seleccionarFecha: true, fechaReferencia: "2026-09-25", actualizar() {},
    consultarPropios({ signal }) { return new Promise((resolver) => pendientes.push({ signal, resolver })); },
  } } });
  const ficha = raiz.querySelector("[data-personal-ficha-integral]");
  tab(ficha, "servicios").listeners.get("click")(); await completar();
  const fecha = ficha.querySelector("[data-personal-ficha-fecha]"); fecha.value = "2020-02-29";
  ficha.querySelector("[data-personal-ficha-corte]").listeners.get("submit")({ preventDefault() {} }); await completar();
  assert.equal(pendientes[0].signal.aborted, true);
  pendientes[1].resolver({ estado: "error" }); await completar();
  pendientes[0].resolver({ estado: "disponible", fuente: "Personal", actualizado_en: "2026-09-25T08:00:00Z", fecha_referencia: "2026-09-25", items: [{ procedencia: "Respuesta anterior" }] }); await completar();
  assert.doesNotMatch(texto(ficha), /Respuesta anterior|Servicios a fecha de/);
});


test("fecha inválida señala el campo y no inicia otra consulta", async () => {
  const raiz = raizFalsa(); let llamadas = 0;
  montarVistaFichaIntegralPersonal({ raiz, fuentes: { servicios: {
    seleccionarFecha: true, fechaReferencia: "2026-09-25", actualizar() {},
    consultarPropios() { llamadas += 1; return { estado: "vacio", fuente: "Personal", actualizado_en: "2026-09-25T08:00:00Z", fecha_referencia: "2026-09-25", items: [] }; },
  } } });
  const ficha = raiz.querySelector("[data-personal-ficha-integral]");
  tab(ficha, "servicios").listeners.get("click")(); await completar();
  const fecha = ficha.querySelector("[data-personal-ficha-fecha]"); fecha.value = "2025-02-29";
  fecha.listeners.get("blur")();
  assert.equal(fecha.atributos.get("aria-invalid"), "true");
  assert.match(texto(ficha), /Introduzca una fecha válida/);
  ficha.querySelector("[data-personal-ficha-corte]").listeners.get("submit")({ preventDefault() {} }); await completar();
  assert.equal(llamadas, 1);
  assert.equal(fecha.enfocado, true);
  fecha.value = "2020-06-30"; fecha.listeners.get("input")();
  assert.equal(fecha.atributos.get("aria-invalid"), "false");
  assert.doesNotMatch(texto(ficha), /Introduzca una fecha válida/);
  ficha.querySelector("[data-personal-ficha-corte]").listeners.get("submit")({ preventDefault() {} }); await completar();
  assert.equal(llamadas, 2, "corregir la fecha permite enviar a la primera");
});


const reciboServicios = "fichapropia:0f0e0d0c-0b0a-4908-8706-050403020100";
const corteServicios = { vigente_en: "2026-10-01", conocido_en: "2026-10-02T07:59:59.000000Z" };
const archivoServicios = () => ({ bytes: new TextEncoder().encode("Inicio,Fin\n"), mime: "text/csv; charset=utf-8", nombre: "servicios.csv", huella: "a".repeat(64) });
function documentoDescarga(raiz) {
  const blobs = [], urls = []; const d = raiz.ownerDocument; d.body = raiz;
  d.defaultView = { Blob, URL: { createObjectURL(blob) { blobs.push(blob); return "blob:local"; }, revokeObjectURL(url) { urls.push(url); } } };
  return { blobs, urls };
}
const datosServicios = () => ({ estado: "disponible", exportacion_servicios_disponible: true, fuente: "Personal", actualizado_en: "2026-10-02T08:00:00Z", fecha_referencia: corteServicios.vigente_en, recibo_ref: reciboServicios, corte: { ...corteServicios }, items: [{ procedencia: "Diputación" }] });

test("CSV servidor: mismo recibo/corte, estado y foco; reintentar no renueva la consulta", async () => {
  const raiz = raizFalsa(), { blobs, urls } = documentoDescarga(raiz);
  let consultas = 0, completarExportacion; const exportaciones = [], avisos = [];
  montarVistaFichaIntegralPersonal({ raiz, anunciar: (...args) => avisos.push(args), fuentes: { servicios: {
    actualizar() {}, consultarPropios() { consultas += 1; return datosServicios(); },
    exportarPropios(entrada) { exportaciones.push(entrada); return new Promise((resolver, rechazar) => { completarExportacion = { resolver, rechazar }; }); },
  } } });
  const ficha = raiz.querySelector("[data-personal-ficha-integral]");
  tab(ficha, "servicios").click(); await completar();
  const boton = ficha.querySelector("[data-personal-servicios-descargar]"); boton.focus(); boton.click(); boton.click();
  assert.equal(boton.disabled, true); assert.equal(boton.atributos.get("aria-busy"), "true");
  assert.match(texto(ficha), /Preparando el resumen/u); assert.equal(exportaciones.length, 1);
  assert.equal(exportaciones[0].reciboRef, reciboServicios); assert.deepEqual(exportaciones[0].corte, corteServicios);
  completarExportacion.rechazar({ codigo: "no_disponible" }); await completar();
  assert.match(texto(ficha), /no está disponible ahora/u); assert.equal(boton.disabled, false); assert.equal(boton.enfocado, true);
  assert.equal(consultas, 1); assert.equal(blobs.length, 0);
  boton.click(); completarExportacion.resolver(archivoServicios()); await completar();
  assert.equal(consultas, 1); assert.equal(exportaciones.length, 2); assert.equal(blobs.length, 1); assert.equal(urls.length, 1);
  assert.match(texto(ficha), /servidor ha preparado el resumen/u);
  assert.doesNotMatch(texto(ficha), /guardado|entregado|certificado/u);
  assert.equal(avisos.length, 2);
});

test("CSV tardío se cancela al actualizar, cambiar apartado o desmontar", async () => {
  for (const operacion of ["actualizar", "salir", "desmontar"]) {
    const raiz = raizFalsa(), { blobs } = documentoDescarga(raiz); let resolver; let entrada; let consultas = 0;
    const montaje = montarVistaFichaIntegralPersonal({ raiz, fuentes: { servicios: {
      actualizar() {}, consultarPropios() { consultas += 1; return datosServicios(); },
      exportarPropios(solicitud) { entrada = solicitud; return new Promise((r) => { resolver = r; }); },
    } } });
    const ficha = raiz.querySelector("[data-personal-ficha-integral]"); tab(ficha, "servicios").click(); await completar();
    const boton = ficha.querySelector("[data-personal-servicios-descargar]"); boton.click();
    if (operacion === "actualizar") ficha.querySelector('[data-personal-ficha-actualizar="servicios"]').click();
    else if (operacion === "salir") tab(ficha, "ficha").click();
    else montaje.desmontar();
    assert.equal(entrada.signal.aborted, true); resolver(archivoServicios()); await completar();
    boton.click(); assert.equal(blobs.length, 0); assert.equal(consultas, operacion === "actualizar" ? 2 : 1);
  }
});

test("sin recibo/corte o cliente nominal no se ofrece CSV local; 403 pide actualizar voluntariamente", async () => {
  for (const quitar of ["recibo_ref", "corte", "exportador"]) {
    const raiz = raizFalsa(); const datos = datosServicios(); if (quitar !== "exportador") delete datos[quitar];
    const fuente = { consultarPropios: () => datos, ...(quitar !== "exportador" ? { exportarPropios: async () => archivoServicios() } : {}) };
    montarVistaFichaIntegralPersonal({ raiz, fuentes: { servicios: fuente } });
    const ficha = raiz.querySelector("[data-personal-ficha-integral]"); tab(ficha, "servicios").click(); await completar();
    assert.equal(ficha.querySelector("[data-personal-servicios-descargar]"), null);
  }
  const raiz = raizFalsa(); let consultas = 0;
  montarVistaFichaIntegralPersonal({ raiz, fuentes: { servicios: { actualizar() {}, consultarPropios() { consultas += 1; return datosServicios(); }, exportarPropios: async () => { throw { codigo: "denegado" }; } } } });
  const ficha = raiz.querySelector("[data-personal-ficha-integral]"); tab(ficha, "servicios").click(); await completar();
  ficha.querySelector("[data-personal-servicios-descargar]").click(); await completar();
  assert.match(texto(ficha), /No se puede exportar esta consulta. Actualice/u);
  assert.equal(consultas, 1); assert.ok(ficha.querySelector('[data-personal-ficha-actualizar="servicios"]'));
});


test("disponibilidad false o ausente nunca ofrece acción ni inicia POST con servidor antiguo", async () => {
  for (const valor of [undefined, false, true]) {
    const raiz = raizFalsa(); let posts = 0; const datos = datosServicios();
    if (valor === undefined) delete datos.exportacion_servicios_disponible;
    else datos.exportacion_servicios_disponible = valor;
    montarVistaFichaIntegralPersonal({ raiz, fuentes: { servicios: { consultarPropios: () => datos, exportarPropios: async () => { posts += 1; return archivoServicios(); } } } });
    const ficha = raiz.querySelector("[data-personal-ficha-integral]"); tab(ficha, "servicios").click(); await completar();
    assert.equal(Boolean(ficha.querySelector("[data-personal-servicios-descargar]")), valor === true);
    assert.equal(posts, 0); assert.match(texto(ficha), /Diputación/u);
  }
});


test("sesión caducada en descarga retira tabla y conserva aviso y actualización al cambiar pestaña", async () => {
  const { crearFuentesFichaPropia } = await import("./cliente-http-ficha-propia.js");
  const raiz = raizFalsa(); const metodos = [];
  const sobre = { data: { exportacion_servicios_disponible: true, ficha: { corte: { vigente_en: "2026-09-25", conocido_en: "2026-09-25T08:59:59.000000Z" }, relaciones: [], servicios: [{ inicio: "2019-01-01", fin: "2019-12-31", clase: "Servicios previos", dias: 365, estado: "reconocido" }] }, recibo_ref: "fichapropia:0f0e0d0c-0b0a-4908-8706-050403020100", consultada_en: "2026-09-25T09:00:00.000000Z" } };
  const fuentes = await crearFuentesFichaPropia({ fetchImpl: async (ruta, opciones) => {
    metodos.push(opciones.method);
    const cuerpo = JSON.stringify(opciones.method === "GET" ? sobre : { error: "autenticacion_requerida" });
    return new Response(cuerpo, { status: opciones.method === "GET" ? 200 : 401, headers: { "Content-Type": "application/json; charset=utf-8", "Content-Length": String(Buffer.byteLength(cuerpo)) } });
  } }).preparar();
  montarVistaFichaIntegralPersonal({ raiz, fuentes });
  const ficha = raiz.querySelector("[data-personal-ficha-integral]");
  tab(ficha, "servicios").click(); await completar();
  assert.ok(nodos(ficha).some((n) => n.tagName === "table"));
  ficha.querySelector("[data-personal-servicios-descargar]").click(); await completar(); await completar();
  for (const pestana of ["servicios", "relaciones", "servicios"]) {
    tab(ficha, pestana).click(); await completar();
    assert.equal(nodos(ficha).some((n) => n.tagName === "table"), false);
    assert.equal(ficha.querySelector("[data-personal-servicios-descargar]"), null);
    assert.equal(ficha.querySelector("[data-personal-ficha-fecha]"), null);
    assert.match(texto(ficha), /Su sesión ha finalizado. Identifíquese de nuevo/u);
    assert.ok(ficha.querySelector(`[data-personal-ficha-actualizar="${pestana}"]`));
  }
  assert.deepEqual(metodos, ["GET", "POST"]);
});

test("historia conectada se abre desde Servicios sin consultar sola y se desmonta al salir", async () => {
 const raiz=raizFalsa();let llamadas=0;
 const datos={...datosServicios(),historia_servicios_disponible:true};
 montarVistaFichaIntegralPersonal({raiz,fuentes:{servicios:{consultarPropios:()=>datos,clienteHistoria:{consultar:async()=>{llamadas+=1;}}}}});
 const ficha=raiz.querySelector("[data-personal-ficha-integral]");tab(ficha,"servicios").click();await completar();
 const abrir=ficha.querySelector("[data-personal-historia-abrir]");assert.ok(abrir);abrir.click();
 assert.ok(ficha.querySelector("[data-personal-historia-servicios]"));assert.equal(llamadas,0);
 tab(ficha,"ficha").click();assert.equal(ficha.querySelector("[data-personal-historia-servicios]"),null);assert.equal(llamadas,0);
});


test("401 de historia retira también la ficha padre y mantiene sesión invalidada sin GET", async () => {
 const {crearFuentesFichaPropia}=await import("./cliente-http-ficha-propia.js");
 const raiz=raizFalsa();const metodos=[];
 const sobre={data:{exportacion_servicios_disponible:true,historia_servicios_disponible:true,ficha:{corte:{vigente_en:"2026-09-25",conocido_en:"2026-09-25T08:59:59.000000Z"},relaciones:[],servicios:[{inicio:"2019-01-01",fin:"2019-12-31",clase:"Servicios previos",dias:365,estado:"reconocido"}]},recibo_ref:"fichapropia:0f0e0d0c-0b0a-4908-8706-050403020100",consultada_en:"2026-09-25T09:00:00.000000Z"}};
 const fuentes=await crearFuentesFichaPropia({fetchImpl:async(_,opciones)=>{
  metodos.push(opciones.method);const body=JSON.stringify(opciones.method==="GET"?sobre:{error:"autenticacion_requerida"});
  return new Response(body,{status:opciones.method==="GET"?200:401,headers:{"Content-Type":"application/json; charset=utf-8","Content-Length":String(Buffer.byteLength(body))}});
 }}).preparar();
 montarVistaFichaIntegralPersonal({raiz,fuentes});const ficha=raiz.querySelector("[data-personal-ficha-integral]");
 tab(ficha,"servicios").click();await completar();ficha.querySelector("[data-personal-historia-abrir]").click();
 const historia=ficha.querySelector("[data-personal-historia-servicios]");
 historia.querySelector('[data-personal-historia-fecha="desde"]').value="2020-01-01";
 historia.querySelector('[data-personal-historia-fecha="hasta"]').value="2027-01-01";
 const form=nodos(historia).find(n=>n.tagName==="form");form.listeners.get("submit")({preventDefault(){}});
 await completar();await completar();
 assert.equal(nodos(ficha).some(n=>n.tagName==="table"),false);
 assert.equal(ficha.querySelector("[data-personal-historia-servicios]"),null);
 assert.equal(ficha.querySelector("[data-personal-servicios-descargar]"),null);
 assert.match(texto(ficha),/Su sesión ha finalizado/u);
 assert.ok(ficha.querySelector('[data-personal-ficha-actualizar="servicios"]'));
 tab(ficha,"ficha").click();tab(ficha,"servicios").click();await completar();
 assert.equal(nodos(ficha).some(n=>n.tagName==="table"),false);assert.deepEqual(metodos,["GET","POST"]);
 await assert.rejects(fuentes.servicios.clienteHistoria.consultar({efectosDesde:"2020-01-01",efectosHasta:"2027-01-01"}),{codigo:"sesion_caducada",estado:401});
});

test("enlace RPT abre Catálogos aunque existan fuentes propias, sólo si la sonda lo ofrece", async () => {
  const anterior = globalThis.window;
  globalThis.window = { location: { search: "?lang=es&rpt_vista=puestos&rpt_categoria=administrativo", hash: "#personal" } };
  try {
    const fuentes = { servicios: { consultarPropios: () => ({ estado: "vacio", fuente: "Personal", actualizado_en: "2026-09-24T08:00:00Z", items: [] }) } };
    const raiz = raizFalsa(); let montajes = 0;
    montarVistaFichaIntegralPersonal({ raiz, fuentes, ocultarSinFuente: true, rptDisponible: true,
      montarCatalogos: () => { montajes += 1; return { desmontar() {} }; } });
    const ficha = raiz.querySelector("[data-personal-ficha-integral]");
    assert.equal(tab(ficha, "catalogos").atributos.get("aria-selected"), "true");
    await completar(); assert.equal(montajes, 1);
    const noDisponible = raizFalsa(); let montajesSinSonda = 0;
    montarVistaFichaIntegralPersonal({ raiz: noDisponible, fuentes, ocultarSinFuente: true, rptDisponible: false,
      montarCatalogos: () => { montajesSinSonda += 1; return { desmontar() {} }; } });
    assert.equal(tab(noDisponible.querySelector("[data-personal-ficha-integral]"), "ficha").atributos.get("aria-selected"), "true");
    await completar(); assert.equal(montajesSinSonda, 0);
    assert.throws(() => montarVistaFichaIntegralPersonal({ raiz: raizFalsa(), rptDisponible: "sí" }), /no disponible/);
  } finally { globalThis.window = anterior; }
});

test("enlace RPT inválido llega al visor, que canoniza el filtro y muestra aviso", async () => {
  const { montarModuloRPTPublica } = await import("./vista-rpt-publica.js");
  const anterior = globalThis.window;
  const location = { pathname: "/portal-empleado/", search: "?lang=es&rpt_vista=puestos&rpt_centro=%20", hash: "#personal" };
  globalThis.window = { location, history: { replaceState(_a, _b, ruta) { const url = new URL(ruta, "http://vec.local"); location.search = url.search; location.hash = url.hash; } } };
  try {
    const raiz = raizFalsa(), consultas = [];
    montarVistaFichaIntegralPersonal({ raiz, ocultarSinFuente: true, rptDisponible: true,
      fuentes: { servicios: { consultarPropios: () => ({ estado: "vacio", fuente: "Personal", actualizado_en: "2026-09-24T08:00:00Z", items: [] }) } },
      montarCatalogos: (entrada) => montarModuloRPTPublica({ ...entrada, cliente: { async listar(consulta) { consultas.push(consulta); return {
        items: [], total: 0, limit: consulta.limit, offset: consulta.offset, vista: consulta.vista,
        fuente: { documento: "RPT publicada", importacion: "rpt-v1", generado_en: "2026-09-17", aviso: "Sin ocupantes", huella_sha256: "a".repeat(64) },
        resumen: { puestos: 842, dotacion: 1714, categorias: 145, centros: 41 },
      }; } } }),
    });
    await completar(); await completar();
    const ficha = raiz.querySelector("[data-personal-ficha-integral]");
    assert.equal(tab(ficha, "catalogos").atributos.get("aria-selected"), "true");
    assert.equal(consultas[0].vista, "categorias"); assert.equal(consultas[0].centro_codigo, "");
    assert.equal(location.search, "?lang=es");
    assert.match(texto(ficha), /enlace tenía un filtro no válido/u);
  } finally { globalThis.window = anterior; }
});

test("la pestaña RPT activa queda visible en móvil sin desplazar página ni mover foco", () => {
  const anterior = globalThis.window;
  globalThis.window = { location: { search: "?rpt_vista=puestos&rpt_categoria=administrativo", hash: "#personal" }, scrollY: 420,
    scrollTo() { throw new Error("la página no debe desplazarse"); } };
  try {
    const raiz = raizFalsa({ geometriaPestanas: true });
    montarVistaFichaIntegralPersonal({ raiz, ocultarSinFuente: true, rptDisponible: true,
      montarCatalogos: () => ({ desmontar() {} }), fuentes: {
        servicios: { consultarPropios: () => ({ estado: "vacio", fuente: "Personal", actualizado_en: "2026-09-24T08:00:00Z", items: [] }) },
      } });
    const ficha = raiz.querySelector("[data-personal-ficha-integral]");
    const barra = nodos(ficha).find((n) => n.className === "personal-ficha-pestanas");
    const catalogos = tab(ficha, "catalogos");
    assert.equal(catalogos.atributos.get("aria-selected"), "true");
    assert.ok(barra.scrollLeft >= 165, "solo la barra alcanza la pestaña fuera de 24..366");
    assert.ok(catalogos.getBoundingClientRect().right <= 366);
    assert.equal(raiz.ownerDocument.activeElement, null, "la apertura no mueve el foco");
    assert.equal(globalThis.window.scrollY, 420);
    tab(ficha, "ficha").click();
    assert.ok(barra.scrollLeft <= 4); assert.ok(tab(ficha, "ficha").getBoundingClientRect().left >= 24);
    tab(ficha, "ficha").listeners.get("keydown")({ key: "End", preventDefault() {} });
    assert.ok(catalogos.getBoundingClientRect().right <= 366);
    assert.equal(catalogos.opcionesFoco?.preventScroll, true);
    assert.equal(globalThis.window.scrollY, 420);
  } finally { globalThis.window = anterior; }
});
