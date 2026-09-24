import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { crearTraductorInscripciones, MENSAJES_INSCRIPCIONES_ES } from "./i18n.js";
import { montarVistaInscripciones, renderizarVistaInscripciones } from "./vista.js";

const ejemplo = (condicion = "externa") => ({
  estado: "disponible",
  persona: { condicion, nombre_visible: "Ana Ejemplo", contacto_visible: "Correo de la persona" },
  convocatorias: [{ identificador_publico: "conv-publica-1", titulo: "Proceso de prueba", categoria: "Administración", plazo: "Según bases", bases: "Bases publicadas", requisitos: [{ descripcion: "Titulación", estado: "pendiente", motivo: "Falta acreditación", procedencia: "Declaración" }], documentos: ["Solicitud"] }],
  convocatoriaSeleccionada: { identificador_publico: "conv-publica-1", titulo: "Proceso de prueba" },
  inscripciones: [], subsanaciones: [{ titulo: "Subsanar titulación", motivo: "Aportar documento", fecha_limite: "Según requerimiento", estado: "Pendiente" }],
});

function raizDePrueba() {
  class Nodo {
    constructor(documento) { this.ownerDocument = documento; this.listeners = new Map(); this.children = []; this.parent = null; this.innerHTML = ""; }
    append(nodo) { nodo.parent = this; this.children.push(nodo); }
    remove() { if (this.parent) this.parent.children = this.parent.children.filter((n) => n !== this); }
    addEventListener(nombre, fn) { this.listeners.set(nombre, fn); }
    removeEventListener(nombre, fn) { if (this.listeners.get(nombre) === fn) this.listeners.delete(nombre); }
  }
  const documento = { createElement() { return new Nodo(documento); } };
  return new Nodo(documento);
}

test("catálogo español cerrado y estados explícitos sin fuente", () => {
  assert.ok(Object.isFrozen(MENSAJES_INSCRIPCIONES_ES));
  const t = crearTraductorInscripciones();
  assert.match(t("sin_conexion"), /pendiente de conexión/i);
  assert.throws(() => crearTraductorInscripciones({ titulo: "Solo título" }), /incompleto/);
  assert.throws(() => t("no_existe"), /desconocida/);
  for (const estado of ["cargando", "vacio", "no_configurado", "denegado", "error"]) {
    const html = renderizarVistaInscripciones({ estado });
    assert.match(html, new RegExp(t(estado === "no_configurado" ? "sin_conexion" : estado).slice(0, 15), "i"));
    assert.doesNotMatch(html, /recibo:/i);
  }
});

test("pendiente no bloquea el recorrido y los datos de solicitud no se precompletan antes de confirmar", () => {
  const datos = ejemplo();
  const html = renderizarVistaInscripciones(datos, { pestana: "convocatorias" }, { puedeAbrir: true });
  assert.match(html, /Pendiente de comprobar o acreditar/);
  assert.match(html, /no impide por sí solo/);
  assert.match(html, /data-inscripciones-detalle="0"/);
  assert.doesNotMatch(html, /Ana Ejemplo/);
  const preparacion = renderizarVistaInscripciones(datos, { pestana: "preparar" });
  assert.match(preparacion, /Aspirante externo/);
  assert.doesNotMatch(preparacion, /Ana Ejemplo/);
  assert.match(preparacion, /data-inscripciones-preparar disabled/);
  const completando = renderizarVistaInscripciones(datos, { pestana: "preparar", confirmada: true, contacto: "Correo de la persona" });
  assert.match(completando, /value="Ana Ejemplo" readonly/);
  assert.match(completando, /value="Correo de la persona"/);
  const preparado = renderizarVistaInscripciones(datos, { pestana: "preparar", confirmada: true, preparado: true });
  assert.match(preparado, /Ana Ejemplo/);
  assert.match(preparado, /Borrador local sin registro/);
  assert.match(preparado, /Registrar solicitud"?/);
  assert.match(preparado, /disabled aria-disabled="true"/);
  assert.match(renderizarVistaInscripciones(ejemplo("empleada"), { pestana: "preparar" }), /Persona empleada/);
});

test("el montaje abre solo el identificador público, confirma en memoria y limpia al desmontar", () => {
  const raiz = raizDePrueba();
  const abiertos = []; const anuncios = []; let limpieza;
  const vista = montarVistaInscripciones({ raiz, datos: ejemplo(), abrirDetallePublico: (id) => abiertos.push(id), anunciar: (m) => anuncios.push(m), registrarDesmontar: (fn) => { limpieza = fn; } });
  const nodo = raiz.children[0];
  nodo.listeners.get("click")({ target: { closest: (s) => s === "[data-inscripciones-detalle]" ? { dataset: { inscripcionesDetalle: "0" } } : null } });
  assert.deepEqual(abiertos, ["conv-publica-1"]);
  nodo.listeners.get("click")({ target: { closest: (s) => s === "[data-inscripciones-tab]" ? { dataset: { inscripcionesTab: "preparar" } } : null } });
  assert.doesNotMatch(nodo.innerHTML, /Ana Ejemplo/);
  nodo.listeners.get("change")({ target: { closest: (s) => s === "[data-inscripciones-confirmar]" ? { checked: true } : null } });
  assert.doesNotMatch(nodo.innerHTML, /Ana Ejemplo/);
  nodo.listeners.get("click")({ target: { closest: (s) => s === "[data-inscripciones-preparar]" ? { disabled: false } : null } });
  assert.match(nodo.innerHTML, /Ana Ejemplo/);
  assert.doesNotMatch(nodo.innerHTML, /Borrador local sin registro/);
  nodo.listeners.get("click")({ target: { closest: (s) => s === "[data-inscripciones-solicitud-siguiente]" ? {} : null } });
  assert.match(nodo.innerHTML, /Borrador local sin registro/);
  assert.match(anuncios[0], /No se ha presentado/);
  vista.actualizar(ejemplo("empleada"));
  assert.doesNotMatch(nodo.innerHTML, /Ana Ejemplo/);
  assert.equal(limpieza, vista.desmontar);
  vista.desmontar();
  assert.equal(raiz.children.length, 0);
  assert.equal(nodo.listeners.size, 0);
});

test("el asistente valida solicitud y subsanación sin crear envío ni justificante", () => {
  const raiz = raizDePrueba();
  const datos = ejemplo(); datos.persona.contacto_visible = "";
  const avisos = [];
  montarVistaInscripciones({ raiz, datos, anunciar: (texto) => avisos.push(texto) });
  const nodo = raiz.children[0];
  const click = (selector, extras = {}) => nodo.listeners.get("click")({ target: { closest: (s) => s === selector ? extras : null } });
  const cambio = (selector, extras) => nodo.listeners.get("change")({ target: { closest: (s) => s === selector ? extras : null } });
  const entrada = (selector, value) => nodo.listeners.get("input")({ target: { value, closest: (s) => s === selector ? { value } : null } });
  click("[data-inscripciones-tab]", { dataset: { inscripcionesTab: "preparar" } });
  cambio("[data-inscripciones-confirmar]", { checked: true });
  click("[data-inscripciones-preparar]", { disabled: false });
  click("[data-inscripciones-solicitud-siguiente]");
  assert.match(nodo.innerHTML, /Indica un contacto/);
  assert.doesNotMatch(nodo.innerHTML, /Borrador local sin registro/);
  entrada("[data-inscripciones-contacto]", "contacto@example.invalid");
  click("[data-inscripciones-solicitud-siguiente]");
  assert.match(nodo.innerHTML, /Borrador local sin registro/);
  assert.match(nodo.innerHTML, /disabled aria-disabled="true"/);
  click("[data-inscripciones-tab]", { dataset: { inscripcionesTab: "subsanaciones" } });
  click("[data-inscripciones-seleccionar-subsanacion]", { dataset: { inscripcionesSeleccionarSubsanacion: "0" } });
  assert.match(nodo.innerHTML, /He leído el motivo/);
  cambio("[data-inscripciones-confirmar-requerimiento]", { checked: true });
  click("[data-inscripciones-subsanacion-siguiente]");
  entrada("[data-inscripciones-respuesta]", "corta");
  click("[data-inscripciones-subsanacion-siguiente]");
  assert.match(nodo.innerHTML, /entre 10 y 1000 caracteres/);
  entrada("[data-inscripciones-respuesta]", "Adjunto la acreditación cuando exista el canal autorizado.");
  click("[data-inscripciones-subsanacion-siguiente]");
  assert.match(nodo.innerHTML, /Respuesta preparada en memoria/);
  assert.match(nodo.innerHTML, /disabled aria-disabled="true"/);
  assert.ok(avisos.some((a) => /subsanación/i.test(a)));
});

test("escapa datos inyectados y no introduce transporte ni persistencia web", async () => {
  const datos = ejemplo();
  datos.convocatorias[0].titulo = '<img src=x onerror="alert(1)">';
  const html = renderizarVistaInscripciones(datos);
  assert.match(html, /&lt;img/);
  assert.doesNotMatch(html, /<img/);
  const fuente = await readFile(new URL("vista.js", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /fetch\(|XMLHttpRequest|localStorage|sessionStorage|indexedDB|document\.cookie/i);
});

test("la vista carga el catálogo i18n con la versión F2 actual", async () => {
  const fuente = await readFile(new URL("vista.js", import.meta.url), "utf8");
  assert.match(fuente, /from "\.\/i18n\.js\?v=20260924-f2-web2"/);
  const modulo = await import(new URL("./vista.js?v=20260924-f2-web2", import.meta.url));
  assert.equal(typeof modulo.montarVistaInscripciones, "function");
  assert.match(modulo.renderizarVistaInscripciones(), /Vista pendiente de conexión/);
});
