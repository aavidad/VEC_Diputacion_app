import "./inicializar-i18n.test-helper.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { lectorCatalogos, catalogoPlano } from "./textos-prueba.test-helper.mjs";
import { cambiosDeContacto, claveOperacion, montarFichaAspirante, RUTA_MI_FICHA, validarVista } from "./ficha-aspirante.js";
import { iniciarI18nAreaPersonal } from "./i18n.js";

// Los textos se cargan del catálogo en castellano, como en el arranque.
const documentoIdioma = { documentElement: { lang: "" } };
await iniciarI18nAreaPersonal(documentoIdioma, { leer: lectorCatalogos(), preferidos: ["es"], ubicacion: { href: "https://vec.test/area-personal/" } });
globalThis.document = documentoIdioma;

// DOM mínimo: lo justo para montar, pulsar y enviar el formulario.
class Nodo {
  constructor(documento, etiqueta) {
    Object.assign(this, { ownerDocument: documento, tagName: etiqueta.toUpperCase(), children: [], attrs: {}, handlers: {}, textContent: "", value: "", checked: false });
  }
  setAttribute(k, v) { this.attrs[k] = v; if (k === "value") this.value = v; if (k === "hidden") this.hidden = true; }
  getAttribute(k) { return this.attrs[k] ?? null; }
  append(...h) { for (const x of h) this.children.push(typeof x === "string" ? Object.assign(new Nodo(this.ownerDocument, "#texto"), { textContent: x }) : x); }
  replaceChildren(...h) { this.children = []; this.append(...h); }
  addEventListener(tipo, fn) { this.handlers[tipo] = fn; }
  focus() { this.ownerDocument.activeElement = this; }
  checkValidity() { return true; }
  reportValidity() {}
  set className(v) { this.attrs.class = v; }
  get className() { return this.attrs.class ?? ""; }
  *todos() { for (const h of this.children) { yield h; yield* h.todos(); } }
  querySelectorAll(sel) { return [...this.todos()].filter((n) => coincide(n, sel)); }
  querySelector(sel) { return this.querySelectorAll(sel)[0] ?? null; }
  get elements() { return { namedItem: (nombre) => [...this.todos()].find((n) => n.attrs.name === nombre && n.tagName === "INPUT" && n.attrs.type !== "radio") ?? null }; }
  get texto() { return [this.textContent, ...this.children.map((h) => h.texto)].join(" "); }
}
function coincide(n, sel) {
  const m = /^([a-z]*)(?:\[([a-z-]+)(?:='([^']*)')?\])?(:checked)?$/u.exec(sel);
  if (!m) throw new Error(`selector no soportado: ${sel}`);
  const [, etiqueta, atributo, valor, marcado] = m;
  if (etiqueta && n.tagName !== etiqueta.toUpperCase()) return false;
  if (atributo && !(atributo in n.attrs)) return false;
  if (valor !== undefined && n.attrs[atributo] !== valor) return false;
  return !marcado || n.checked;
}
function montar() {
  const documento = { activeElement: null };
  documento.createElement = (e) => new Nodo(documento, e);
  return new Nodo(documento, "div");
}
const esperar = () => new Promise((r) => setTimeout(r, 0));

const cabecerasJSON = { get: (k) => (k === "Content-Type" ? "application/json; charset=utf-8" : null) };
function respuesta(status, cuerpo, cabeceras = cabecerasJSON) { return { status, headers: cabeceras, text: async () => (typeof cuerpo === "string" ? cuerpo : JSON.stringify(cuerpo)) }; }
function servidor(secuencia) {
  const peticiones = [];
  return { peticiones, fetchImpl: async (ruta, opciones) => {
    peticiones.push({ ruta, opciones, cuerpo: opciones.body ? JSON.parse(opciones.body) : null });
    const siguiente = secuencia.shift();
    if (!siguiente) throw new Error("petición no esperada");
    return respuesta(...siguiente);
  } };
}
// Persona sintética: Lucía Fernández Moreno.
const identidad = { nombre: "LUCÍA", apellidos: "FERNÁNDEZ MORENO", tipo_documento: "dni", pais_documento: "ES", documento: "***4567**" };
const exigencias = [{ campo: "telefono", obligatorio: true }, { campo: "movil", obligatorio: false },
  { campo: "domicilio", obligatorio: false, condicion: "si_elige_notificacion_papel" }, { campo: "codigo_postal", obligatorio: false, condicion: "si_elige_notificacion_papel" }];
const sinFicha = { estado: "sin_ficha", version: 0, identidad, contacto: {}, exigencias, catalogo_disponible: true, catalogo_ejemplo: true };
const activa = (contacto = { telefono: "958123456" }) => ({ ...sinFicha, estado: "activa", version: 3, contacto });
const recibo = (version) => ({ recibo_ref: `asprec_${"a".repeat(32)}`, version, fecha_utc: "2026-09-29T10:00:00.000000Z", replay: false });
const azar = { randomUUID: () => "00000000-0000-4000-8000-000000000001" };

function rellenar(contenedor, valores) {
  const formulario = contenedor.querySelector("form[data-ficha]");
  for (const [campo, valor] of Object.entries(valores)) formulario.elements.namedItem(campo).value = valor;
  formulario.handlers.input?.();
  return formulario;
}
async function enviar(formulario) { formulario.handlers.submit({ preventDefault() {}, currentTarget: formulario }); await esperar(); await esperar(); }

test("sin ficha muestra la identidad del certificado y crea la ficha con lo pedido", async () => {
  const s = servidor([[200, { data: sinFicha }], [201, { data: recibo(1) }], [200, { data: activa() }]]);
  const c = montar();
  montarFichaAspirante({ contenedor: c, fetchImpl: s.fetchImpl, azar });
  await esperar();
  assert.match(c.texto, /LUCÍA/u);
  assert.match(c.texto, /FERNÁNDEZ MORENO/u);
  assert.match(c.texto, /\*\*\*4567\*\*/u);
  const telefono = c.querySelector("form[data-ficha]").elements.namedItem("telefono");
  assert.equal(telefono.attrs.required, "");
  assert.equal(c.querySelector("form[data-ficha]").elements.namedItem("codigo_postal").attrs.pattern, "[0-9]{5}");
  assert.equal(c.querySelector("[data-motivo]"), null, "al crear no se pregunta motivo");
  await enviar(rellenar(c, { telefono: " 958 12 34 56 ", domicilio: "" }));
  const alta = s.peticiones[1];
  assert.equal(alta.ruta, RUTA_MI_FICHA);
  assert.equal(alta.opciones.method, "POST");
  assert.equal(alta.opciones.credentials, "same-origin");
  assert.deepEqual(alta.cuerpo, { operacion: "alta", clave_operacion: claveOperacion(azar), version_esperada: 0, campos: { telefono: "958 12 34 56" } });
  assert.match(c.querySelector("[data-aviso-ficha]").textContent, /^Ficha creada el 29 de septiembre de 2026/u);
});

test("cambiar un dato que ya había pide el motivo antes de enviar", async () => {
  const s = servidor([[200, { data: activa() }], [201, { data: recibo(4) }], [200, { data: activa({ telefono: "612345678" }) }]]);
  const c = montar();
  montarFichaAspirante({ contenedor: c, fetchImpl: s.fetchImpl, azar });
  await esperar();
  const formulario = rellenar(c, { telefono: "612345678" });
  assert.equal(c.querySelector("[data-motivo]").hidden, false);
  await enviar(formulario);
  assert.equal(s.peticiones.length, 1, "sin motivo no se envía");
  const error = c.querySelector("p[id='ficha-motivo-error']");
  assert.equal(c.querySelector("[data-motivo]").attrs["aria-describedby"], "ficha-motivo-error");
  assert.match(error.textContent, /Elija por qué cambia los datos/u);
  assert.equal(c.ownerDocument.activeElement.attrs.name, "motivo", "el foco va a la pregunta");
  const repintado = c.querySelector("form[data-ficha]");
  assert.equal(repintado.elements.namedItem("telefono").value, "612345678", "lo escrito se conserva");
  repintado.querySelectorAll("input[name='motivo']").find((r) => r.attrs.value === "correccion_de_error").checked = true;
  await enviar(repintado);
  assert.deepEqual(s.peticiones[1].cuerpo, { operacion: "rectificar", clave_operacion: claveOperacion(azar), version_esperada: 3, motivo: "correccion_de_error", campos: { telefono: "612345678" } });
});

test("un dato nuevo no pregunta motivo y sin cambios no se envía nada", async () => {
  const s = servidor([[200, { data: activa() }], [201, { data: recibo(4) }], [200, { data: activa() }]]);
  const c = montar();
  montarFichaAspirante({ contenedor: c, fetchImpl: s.fetchImpl, azar });
  await esperar();
  await enviar(rellenar(c, {}));
  assert.equal(s.peticiones.length, 1);
  assert.equal(c.querySelector("[data-aviso-ficha]").className, "nota aviso");
  const formulario = rellenar(c, { movil: "699111222" });
  assert.equal(c.querySelector("[data-motivo]").hidden, true);
  await enviar(formulario);
  assert.equal(s.peticiones[1].cuerpo.motivo, "dato_nuevo");
  assert.deepEqual(s.peticiones[1].cuerpo.campos, { movil: "699111222" });
});

test("un dato que ya no se pide se puede quitar", async () => {
  const soloTelefono = { ...activa({ telefono: "958123456", domicilio: "Calle Recogidas, 12" }), exigencias: [exigencias[0]] };
  const s = servidor([[200, { data: soloTelefono }], [201, { data: recibo(4) }], [200, { data: activa() }]]);
  const c = montar();
  montarFichaAspirante({ contenedor: c, fetchImpl: s.fetchImpl, azar });
  await esperar();
  c.querySelector("[data-quitar]").handlers.click();
  assert.equal(s.peticiones.length, 1, "quitar pide confirmación antes");
  assert.match(c.querySelector("[data-pregunta-quitar]").textContent, /¿Quitar «Domicilio» de su ficha\?/u);
  assert.equal(c.ownerDocument.activeElement, c.querySelector("[data-pregunta-quitar]"));
  c.querySelector("[data-cancelar-quitar]").handlers.click();
  assert.equal(c.querySelector("[data-pregunta-quitar]"), null);
  assert.equal(c.ownerDocument.activeElement, c.querySelector("[data-quitar='domicilio']"), "al cancelar, el foco vuelve a Quitar");
  c.querySelector("[data-quitar]").handlers.click();
  c.querySelector("[data-confirmar-quitar]").handlers.click();
  await esperar(); await esperar();
  assert.deepEqual(s.peticiones[1].cuerpo, { operacion: "rectificar", clave_operacion: claveOperacion(azar), version_esperada: 3, motivo: "cambio_de_dato", campos: { domicilio: "" } });
});

test("quitar un dato sobrante conserva la edición telefónica pendiente", async () => {
  const soloTelefono = { ...activa({ telefono: "958123456", domicilio: "Calle Recogidas, 12" }), exigencias: [exigencias[0]] };
  const sinDomicilio = { ...soloTelefono, version: 4, contacto: { telefono: "958123456" } };
  const s = servidor([[200, { data: soloTelefono }], [201, { data: recibo(4) }], [200, { data: sinDomicilio }]]);
  const c = montar();
  montarFichaAspirante({ contenedor: c, fetchImpl: s.fetchImpl, azar });
  await esperar();
  rellenar(c, { telefono: "612345678" });
  c.querySelector("[data-quitar]").handlers.click();
  assert.equal(c.querySelector("form[data-ficha]").elements.namedItem("telefono").value, "612345678");
  c.querySelector("[data-cancelar-quitar]").handlers.click();
  assert.equal(c.querySelector("form[data-ficha]").elements.namedItem("telefono").value, "612345678");
  c.querySelector("[data-quitar]").handlers.click();
  c.querySelector("[data-confirmar-quitar]").handlers.click();
  await esperar(); await esperar();
  assert.deepEqual(s.peticiones[1].cuerpo.campos, { domicilio: "" }, "el teléfono pendiente no se envía con la retirada");
  assert.equal(c.querySelector("form[data-ficha]").elements.namedItem("telefono").value, "612345678");
});

test("catálogo caído: no se ofrece crear ni guardar", async () => {
  const s = servidor([[200, { data: { ...sinFicha, exigencias: [], catalogo_disponible: false } }]]);
  const c = montar();
  montarFichaAspirante({ contenedor: c, fetchImpl: s.fetchImpl, azar });
  await esperar();
  assert.equal(c.querySelector("form[data-ficha]"), null);
});

test("errores en lenguaje llano, reintento y conflicto que recarga", async () => {
  const s = servidor([[403, { error: { codigo: "prohibido", clave_i18n: "x" } }], [200, { data: activa() }], [409, { error: { codigo: "conflicto" } }], [200, { data: activa() }]]);
  const c = montar();
  montarFichaAspirante({ contenedor: c, fetchImpl: s.fetchImpl, azar });
  await esperar();
  const alerta = c.querySelector("[data-accion-ficha='reintentar']");
  assert.ok(alerta);
  assert.match(c.texto, /Su certificado no permite ver esta ficha/u);
  assert.doesNotMatch(c.texto, /403|prohibido|areaPersonal\./u, "sin códigos técnicos en pantalla");
  alerta.handlers.click();
  await esperar();
  await enviar(rellenar(c, { movil: "699111222" }));
  assert.equal(s.peticiones.length, 4, "el conflicto vuelve a cargar la ficha");
  assert.equal(c.querySelector("[data-aviso-ficha]").className, "nota aviso");
});

test("si el guardado falla, lo escrito sigue en el formulario y el foco va al aviso", async () => {
  const s = servidor([[200, { data: activa() }], [503, "<html>proxy</html>", { get: () => "text/html" }]]);
  const c = montar();
  montarFichaAspirante({ contenedor: c, fetchImpl: s.fetchImpl, azar });
  await esperar();
  const formulario = rellenar(c, { movil: "699111222" });
  const boton = formulario.querySelector("button[type='submit']");
  formulario.handlers.submit({ preventDefault() {}, currentTarget: formulario });
  assert.equal(c.querySelector("form[data-ficha]"), formulario, "no se repinta al empezar a guardar");
  assert.equal(boton.attrs["aria-busy"], "true");
  await esperar(); await esperar();
  const alerta = c.querySelector("[data-error-guardar]");
  assert.equal(alerta.attrs.role, "alert");
  assert.match(alerta.textContent, /Sus cambios siguen en el formulario/u);
  assert.equal(c.ownerDocument.activeElement, alerta);
  assert.equal(c.querySelector("form[data-ficha]").elements.namedItem("movil").value, "699111222");
  assert.equal(c.querySelector("[data-accion-ficha='reintentar']"), null, "reintentar solo al cargar");
});

test("un reintento incierto conserva la clave y el cuerpo exactos", async () => {
  let generadas = 0;
  const azarContado = { randomUUID: () => `00000000-0000-4000-8000-${String(++generadas).padStart(12, "0")}` };
  const s = servidor([[200, { data: sinFicha }], [503, "<html>proxy</html>", { get: () => "text/html" }],
    [201, { data: { ...recibo(1), replay: true } }], [200, { data: activa() }]]);
  const c = montar();
  montarFichaAspirante({ contenedor: c, fetchImpl: s.fetchImpl, azar: azarContado });
  await esperar();
  await enviar(rellenar(c, { telefono: "958123456" }));
  await enviar(c.querySelector("form[data-ficha]"));
  assert.deepEqual(s.peticiones[2].cuerpo, s.peticiones[1].cuerpo);
  assert.equal(generadas, 1);
  assert.match(c.querySelector("[data-aviso-ficha]").textContent, /Ficha creada el 29 de septiembre/u);
});

test("editar la petición tras un 503 crea una clave diferente", async () => {
  let generadas = 0;
  const azarContado = { randomUUID: () => `00000000-0000-4000-8000-${String(++generadas).padStart(12, "0")}` };
  const s = servidor([[200, { data: sinFicha }], [503, "<html>proxy</html>", { get: () => "text/html" }],
    [503, "<html>proxy</html>", { get: () => "text/html" }]]);
  const c = montar();
  montarFichaAspirante({ contenedor: c, fetchImpl: s.fetchImpl, azar: azarContado });
  await esperar();
  await enviar(rellenar(c, { telefono: "958123456" }));
  await enviar(rellenar(c, { telefono: "612345678" }));
  assert.notEqual(s.peticiones[2].cuerpo.clave_operacion, s.peticiones[1].cuerpo.clave_operacion);
  assert.equal(generadas, 2);
});

test("un POST 201 sin recibo válido no confirma ni consulta la ficha", async () => {
  const s = servidor([[200, { data: sinFicha }], [201, { data: {} }]]);
  const c = montar();
  montarFichaAspirante({ contenedor: c, fetchImpl: s.fetchImpl, azar });
  await esperar();
  await enviar(rellenar(c, { telefono: "958123456" }));
  assert.equal(s.peticiones.length, 2);
  assert.equal(c.querySelector("[data-aviso-ficha]"), null);
  assert.ok(c.querySelector("[data-error-guardar]"));
  assert.doesNotMatch(c.texto, /Ficha creada el \.|Ficha creada el/u);
});

test("un GET aún sin ficha tras el recibo no muestra creación y permite replay exacto", async () => {
  let generadas = 0;
  const azarContado = { randomUUID: () => `00000000-0000-4000-8000-${String(++generadas).padStart(12, "0")}` };
  const s = servidor([[200, { data: sinFicha }], [201, { data: recibo(1) }], [200, { data: sinFicha }],
    [201, { data: { ...recibo(1), replay: true } }], [200, { data: activa() }]]);
  const c = montar();
  montarFichaAspirante({ contenedor: c, fetchImpl: s.fetchImpl, azar: azarContado });
  await esperar();
  await enviar(rellenar(c, { telefono: "958123456" }));
  assert.equal(c.querySelector("[data-aviso-ficha]"), null);
  assert.ok(c.querySelector("[data-error-guardar]"));
  assert.equal(c.querySelector("form[data-ficha]").elements.namedItem("telefono").value, "958123456");
  await enviar(c.querySelector("form[data-ficha]"));
  assert.deepEqual(s.peticiones[3].cuerpo, s.peticiones[1].cuerpo);
  assert.equal(generadas, 1);
  assert.match(c.querySelector("[data-aviso-ficha]").textContent, /Ficha creada el 29 de septiembre/u);
});

test("valida teléfono y código postal antes de enviar", async () => {
  const s = servidor([[200, { data: sinFicha }]]);
  const c = montar();
  montarFichaAspirante({ contenedor: c, fetchImpl: s.fetchImpl, azar });
  await esperar();
  assert.equal(c.querySelector("form[data-ficha]").attrs.novalidate, "");
  await enviar(rellenar(c, { telefono: "958 12 34 56", codigo_postal: "180" }));
  assert.equal(s.peticiones.length, 1, "con errores no se envía");
  const cp = c.querySelector("form[data-ficha]").elements.namedItem("codigo_postal");
  assert.equal(cp.attrs["aria-invalid"], "true");
  assert.equal(cp.value, "180");
  assert.equal(c.ownerDocument.activeElement, cp);
  assert.match(c.texto, /Escriba los 5 números del código postal, por ejemplo 18002/u);
  await enviar(rellenar(c, { telefono: "teléfono", codigo_postal: "18002" }));
  assert.match(c.texto, /Revise el teléfono: escriba solo números/u);
});

test("una respuesta que no es JSON se trata como no disponible", async () => {
  const s = servidor([[200, "<html></html>", { get: () => "text/html" }]]);
  const c = montar();
  montarFichaAspirante({ contenedor: c, fetchImpl: s.fetchImpl, azar });
  await esperar();
  assert.ok(c.querySelector("[data-accion-ficha='reintentar']"));
  assert.match(c.texto, /No se han podido cargar sus datos/u);
});

test("validarVista y cambiosDeContacto", () => {
  assert.throws(() => validarVista({ ...sinFicha, contacto: { discapacidad: "33" } }));
  assert.throws(() => validarVista({ ...sinFicha, version: 2 }));
  assert.throws(() => validarVista({ ...sinFicha, extra: undefined, identidad: { ...identidad, tipo_documento: "cedula" } }));
  assert.throws(() => validarVista({ ...sinFicha, exigencias: [{ campo: "telefono", obligatorio: true, condicion: "../x" }] }));
  assert.deepEqual(cambiosDeContacto({ telefono: " 958123456 ", movil: "6" }, { telefono: "958123456" }), { cambios: { movil: "6" }, habiaValor: false });
});

test("los textos están en los dos catálogos y no en el código", async () => {
  const codigo = await readFile(new URL("./ficha-aspirante.js", import.meta.url), "utf8");
  const claves = new Set([...codigo.matchAll(/t\(`?"?([a-zA-Z_.]+)/gu)].map((m) => m[1]).filter((c) => !c.endsWith(".")));
  const es = await catalogoPlano("es");
  const en = await catalogoPlano("en");
  for (const sufijo of ["cargando", "identidad.titulo", "contacto.titulo", "accion.crear", "accion.guardar", "motivo.pregunta", "sinCambios",
    "error.no_disponible", "error.prohibido", "error.conflicto", "error.guardar", "error.motivo", "quitar.pregunta", "resumen",
    "validacion.telefono", "validacion.movil", "validacion.codigo_postal", "validacion.domicilio", "validacion.vacio.telefono", "condicion.si_elige_notificacion_papel", "campo.codigo_postal", "panel.titulo"]) {
    assert.ok(es[`areaPersonal.ficha.${sufijo}`] && en[`areaPersonal.ficha.${sufijo}`], sufijo);
  }
  assert.ok(claves.size > 5);
  assert.doesNotMatch(codigo, /"(Teléfono|Guardar|Crear mi ficha|Domicilio)"/u);
  const codigoI18n = await readFile(new URL("./i18n.js", import.meta.url), "utf8");
  assert.doesNotMatch(codigoI18n, /"areaPersonal\.contacto\.titulo": "Correo electrónico"/u);
  assert.equal(es["areaPersonal.contacto.titulo"], "Correo electrónico");
  assert.equal(en["areaPersonal.contacto.titulo"], "Email address");
});
