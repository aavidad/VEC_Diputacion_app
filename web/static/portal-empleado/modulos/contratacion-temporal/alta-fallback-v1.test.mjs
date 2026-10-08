import test from "node:test";
import assert from "node:assert/strict";
import { crearGestorTramitacion } from "./vista-expedientes-tramitacion.js";
import { crearClienteHTTPContratacionTemporal } from "./cliente-http.js";
import { claveI18nValida, codigoValidoParaRuta } from "./cliente-http-transporte.js";

const RUTA = "/api/vec/contratacion-temporal/catalogos-alta";
const CLAVE = "api.contratacion_temporal.catalogos_alta.error.capacidad_no_configurada";
const rutas = { catalogosAlta: RUTA, alta: "/api/vec/contratacion-temporal/solicitudes" };
const esperar = () => new Promise((resolver) => setImmediate(resolver));

function catalogosV1() {
  return { esquema: "vec.contratacion_temporal.catalogos_alta.v1",
    numero_expediente_moad: { referencia: "catalogo:moad:ct", version: 1,
      patron: "^[0-9]{4}/[1-9][0-9]{0,9}$", ejemplo: "2026/12345" },
    centros: [{ referencia: "centro:001", etiqueta: "Centro",
      contactos: [{ referencia: "contacto:001", etiqueta: "Responsable" }] }],
    categorias: [{ referencia: "categoria:001", etiqueta: "Técnico",
      grupos_subgrupos: [{ clave: "A2", etiqueta: "A2" }] }],
    motivos: [{ clave: "sustitucion", etiqueta: "Sustitución" }], documentos: [] };
}

function nodo(documento) {
  return { innerHTML: "", children: [], ownerDocument: documento,
    addEventListener(tipo, escuchar) { (this.eventos ??= {})[tipo] = escuchar; },
    removeEventListener(tipo) { delete this.eventos?.[tipo]; },
    setAttribute() {}, removeAttribute() {},
    append(...hijos) { this.children.push(...hijos); },
    prepend(hijo) { this.children.unshift(hijo); },
    replaceChildren(...hijos) { this.innerHTML = ""; this.children = hijos; },
    querySelector() { return null; }, contains() { return true; } };
}

function escenario(respuestas, { catalogos = catalogosV1(), mensajes = {}, esMontada = () => true } = {}) {
  const documento = { createElement: () => nodo(documento) };
  const contenedor = nodo(documento);
  const raiz = { querySelector: (selector) => selector === "[data-ct-exp-alta]" ? contenedor : null };
  let consultas = 0;
  const gestor = crearGestorTramitacion({ raiz, mensajes, esMontada,
    presentador: { obtenerEstado: () => ({ vista: "alta" }) }, altaDisponible: true,
    alta: { catalogos, capacidad: "contratacion_temporal.solicitud.crear",
      ejecutor: async () => { throw Error("sin envío"); },
      obtenerCatalogosNecesidadesAlta: () => {
        const indice = consultas++;
        return respuestas[indice]();
      } },
    crearEjecutorAltaConRefresco: (ejecutor) => ejecutor });
  return { gestor, contenedor, consultas: () => consultas };
}

function noConfigurada(extra = {}) {
  return { estado: 503, codigo: "capacidad_no_configurada", envelopeValido: true,
    claveI18n: CLAVE, ...extra };
}

function respuestaError(estado, codigo = "capacidad_no_configurada", clave = CLAVE) {
  const texto = JSON.stringify({ error: { codigo, clave_i18n: clave,
    correlacion_ref: "corr_no_disponible" } });
  return new Response(texto, { status: estado, headers: {
    "Content-Type": "application/json; charset=utf-8",
    "Content-Length": String(new TextEncoder().encode(texto).byteLength),
  } });
}

test("el error de capacidad sólo es válido para el GET v2 con su clave exacta", () => {
  assert.equal(codigoValidoParaRuta(`${RUTA}?version=2`, 503, "capacidad_no_configurada", rutas), true);
  assert.equal(claveI18nValida(`${RUTA}?version=2`, "capacidad_no_configurada", CLAVE, rutas), true);
  for (const [ruta, estado] of [[RUTA, 503], [rutas.alta, 503],
    [`${RUTA}?version=2`, 401], [`${RUTA}?version=2`, 403]]) {
    assert.equal(codigoValidoParaRuta(ruta, estado, "capacidad_no_configurada", rutas), false);
  }
  assert.equal(claveI18nValida(`${RUTA}?version=2`, "capacidad_no_configurada",
    "api.contratacion_temporal.cobertura.error.capacidad_no_configurada", rutas), false);
});

test("el cliente HTTP conserva el rechazo validado y rechaza sobres incompatibles", async () => {
  for (const [respuesta, codigo, valido] of [
    [respuestaError(503), "capacidad_no_configurada", true],
    [respuestaError(503, "servicio_no_disponible",
      "api.contratacion_temporal.cobertura.error.servicio_no_disponible"), "servicio_no_disponible", true],
    [respuestaError(503, "capacidad_no_configurada", "clave.falsa"), "respuesta_error_no_valida", false],
    [respuestaError(403), "respuesta_error_no_valida", false],
  ]) {
    const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async () => respuesta });
    await assert.rejects(cliente.obtenerCatalogosNecesidadesAlta(), (error) =>
      error.codigo === codigo && error.envelopeValido === valido);
  }
});

test("el rechazo validado monta el formulario v1 limitado a sustituciones y permite reconsultar v2", async () => {
  const actual = escenario([
    () => Promise.reject(noConfigurada()),
    () => Promise.reject({ estado: 503, codigo: "servicio_no_disponible", envelopeValido: true }),
  ], { mensajes: { alta_solo_sustituciones: "Puede registrar sustituciones. Las demás necesidades todavía no están disponibles." } });
  actual.gestor.montarAltaSiProcede();
  actual.gestor.montarAltaSiProcede();
  await esperar();
  assert.equal(actual.consultas(), 1);
  const [aviso, formulario] = actual.contenedor.children;
  assert.equal(aviso.children[0].textContent,
    "Puede registrar sustituciones. Las demás necesidades todavía no están disponibles.");
  assert.match(formulario.innerHTML, /data-modulo="contratacion-temporal"/u);
  assert.match(formulario.innerHTML, /Sustitución/u);
  assert.doesNotMatch(formulario.innerHTML, /Vacante/u);
  formulario.innerHTML = "edición";
  assert.equal(actual.contenedor.children[0], aviso, "el aviso queda fuera del formulario repintado");
  aviso.children[1].eventos.click();
  actual.gestor.montarAltaSiProcede();
  await esperar();
  assert.equal(actual.consultas(), 2);
  assert.match(actual.contenedor.innerHTML, /data-ct-exp-accion="reintentar"/u);
  actual.gestor.retirarComponentes();
});

test("avería, denegación y sobre falso no activan el formulario v1", async () => {
  for (const error of [
    { estado: 503, codigo: "servicio_no_disponible", envelopeValido: true },
    { estado: 401, codigo: "capacidad_no_configurada", envelopeValido: true, claveI18n: CLAVE },
    { estado: 403, codigo: "capacidad_no_configurada", envelopeValido: true, claveI18n: CLAVE },
    noConfigurada({ envelopeValido: false }), noConfigurada({ claveI18n: "clave.falsa" }),
  ]) {
    const actual = escenario([() => Promise.reject(error)]);
    actual.gestor.montarAltaSiProcede();
    await esperar();
    assert.deepEqual(actual.contenedor.children, []);
    assert.match(actual.contenedor.innerHTML, /data-ct-exp-accion="reintentar"/u);
    actual.gestor.retirarComponentes();
  }
});

test("una fuente v1 con más motivos no se presenta como solo sustituciones", async () => {
  const catalogos = catalogosV1();
  catalogos.motivos.push({ clave: "vacante", etiqueta: "Vacante" });
  const actual = escenario([() => Promise.reject(noConfigurada())], { catalogos });
  actual.gestor.montarAltaSiProcede();
  await esperar();
  assert.deepEqual(actual.contenedor.children, []);
  assert.match(actual.contenedor.innerHTML, /data-ct-exp-accion="reintentar"/u);
  actual.gestor.retirarComponentes();
});

test("una respuesta tardía después del desmontaje no monta ni altera la vista", async () => {
  let rechazar;
  let montada = true;
  const pendiente = new Promise((_, fallo) => { rechazar = fallo; });
  const actual = escenario([() => pendiente], { esMontada: () => montada });
  actual.gestor.montarAltaSiProcede();
  await Promise.resolve();
  actual.gestor.retirarComponentes();
  montada = false;
  const anterior = actual.contenedor.innerHTML;
  rechazar(noConfigurada());
  await esperar();
  assert.equal(actual.contenedor.innerHTML, anterior);
  assert.deepEqual(actual.contenedor.children, []);
  assert.equal(actual.consultas(), 1);
});
