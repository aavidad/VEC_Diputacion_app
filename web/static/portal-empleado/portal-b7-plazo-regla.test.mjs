import test from "node:test";
import assert from "node:assert/strict";
import { consultarPlazoRespuestaLlamamiento, crearControladorBolsas, propuestaPlazoRespuesta, RUTA_PLAZO_RESPUESTA_LLAMAMIENTO } from "./portal-bolsas-api.js";
import { crearPresentadorPanelInterno } from "./portal-panel-interno.js";

const ESQUEMA = "vec.bolsa.llamamiento.plazo_respuesta.v1";
const conRegla = (regla = {}) => ({
  esquema: ESQUEMA, configurada: true,
  regla: { etiqueta: "Plazo de respuesta al llamamiento", texto: "Un día hábil desde el día siguiente al contacto efectivo.", referencia: "vec.bolsa.reglas:1:b05.plazo_respuesta", origen: "ejemplo", ejemplo: true, ...regla },
  vencimiento: { calculado_en: "2026-09-28T08:00:00Z", ultimo_dia: "2026-09-29", vence_en: "2026-09-29T21:59:59Z", vence_antes_de: "2026-09-29T22:00:00Z" },
});
const sinCatalogo = { esquema: ESQUEMA, configurada: false };

function presentador(flujo) {
  const bolsa = { bolsa_ref: "bolsa:01", categoria: "Auxiliar", tipo_lista: "ordinaria", vigente_desde: "2026-09-01", total: 1, por_estado: { disponible: 1 } };
  return crearPresentadorPanelInterno({
    claseEstado: () => "info", etiquetaClave: (valor) => valor,
    encabezadoVista: (_seccion, titulo) => `<header><h2>${titulo}</h2></header>`,
    escaparHTML: (valor) => String(valor ?? "").replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/"/g, "&quot;"),
    numero: (valor) => String(valor ?? 0),
    obtenerDatosPanel: () => ({ esquema: "vec.bolsa.panel.interno.v1" }), tituloVista: (valor) => valor,
    obtenerDatosCandidatosBolsa: () => ({ carga: "listo", datos: { bolsa, candidatos: [], contactos: [] }, error: "" }),
    obtenerEstadoCandidatos: () => ({ nuevo_llamamiento: flujo }),
  });
}

test("B7 consulta el plazo de respuesta por GET de mismo origen y sin caché", async () => {
  let llamada;
  const resultado = await consultarPlazoRespuestaLlamamiento({ fetchImpl: async (url, opciones) => {
    llamada = { url, opciones };
    return { ok: true, status: 200, json: async () => ({ data: conRegla() }) };
  } });
  assert.equal(llamada.url, RUTA_PLAZO_RESPUESTA_LLAMAMIENTO);
  assert.equal(llamada.opciones.method, "GET");
  assert.equal(llamada.opciones.credentials, "same-origin");
  assert.equal(llamada.opciones.cache, "no-store");
  assert.equal(resultado.ok, true);
  const denegada = await consultarPlazoRespuestaLlamamiento({ fetchImpl: async () => ({ ok: false, status: 403 }) });
  assert.deepEqual(denegada, { ok: false, status: 403 });
  const ajena = await consultarPlazoRespuestaLlamamiento({ fetchImpl: async () => ({ ok: true, status: 200, json: async () => ({ data: { esquema: "otro", configurada: true } }) }) });
  assert.equal(ajena.ok, false);
  const caida = await consultarPlazoRespuestaLlamamiento({ fetchImpl: async () => { throw new TypeError("red"); } });
  assert.equal(caida.ok, false);
});

test("B7 convierte la regla en texto editable y procedencia solo para RRHH", () => {
  const ejemplo = propuestaPlazoRespuesta(conRegla());
  assert.equal(ejemplo.texto, "Hasta las 23:59:59 del martes, 29 de septiembre de 2026 (hora peninsular)");
  assert.equal(ejemplo.procedencia, "Regla de ejemplo");
  assert.equal(ejemplo.ejemplo, true);
  assert.equal(ejemplo.referencia, "vec.bolsa.reglas:1:b05.plazo_respuesta");
  const reglamento = propuestaPlazoRespuesta(conRegla({ origen: "reglamento", articulo: "art. 9", ejemplo: false }));
  assert.equal(reglamento.procedencia, "Reglamento, art. 9");
  assert.equal(reglamento.ejemplo, false);
  assert.equal(propuestaPlazoRespuesta(conRegla({ origen: "reglamento", articulo: "art. 9", ejemplo: true })).procedencia, "Reglamento, art. 9, con parte de ejemplo");
  assert.equal(propuestaPlazoRespuesta(sinCatalogo), null);
  assert.equal(propuestaPlazoRespuesta(conRegla({ origen: "reglamento", articulo: "" })), null);
  assert.equal(propuestaPlazoRespuesta(conRegla({ referencia: "sin formato" })), null);
  assert.equal(propuestaPlazoRespuesta({ ...conRegla(), vencimiento: { vence_en: "no es fecha" } }), null);
});

async function iniciarAsistente(respuesta) {
  const escuchas = {};
  const documento = { addEventListener(tipo, fn) { escuchas[tipo] = fn; }, querySelector() { return null; } };
  const estado = { filtrosBolsa: {}, datosCandidatos: { carga: "listo", datos: { candidatos: [] } } };
  const fetchOriginal = globalThis.fetch;
  const rutas = [];
  globalThis.fetch = async (url) => { rutas.push(url); return respuesta; };
  try {
    crearControladorBolsas({ estado, renderizar: () => {}, navegar: () => {}, documento }).instalar();
    escuchas.click({ preventDefault() {}, target: { closest(selector) { return selector === "[data-bolsa-accion]" ? { dataset: { bolsaAccion: "iniciar-b7" } } : null; } } });
    await new Promise(setImmediate);
  } finally {
    globalThis.fetch = fetchOriginal;
  }
  return { flujo: estado.filtrosBolsa.nuevo_llamamiento, rutas };
}

test("B7 con regla rellena el plazo editable y rotula su procedencia", async () => {
  const { flujo, rutas } = await iniciarAsistente({ ok: true, status: 200, json: async () => ({ data: conRegla() }) });
  // Al abrir B7 se pide el plazo una sola vez; la plantilla del correo B7
  // personalizado se pide aparte y no forma parte de esta comprobación.
  assert.deepEqual(rutas.filter((ruta) => ruta === RUTA_PLAZO_RESPUESTA_LLAMAMIENTO), [RUTA_PLAZO_RESPUESTA_LLAMAMIENTO]);
  assert.equal(flujo.reglaPlazo.procedencia, "Regla de ejemplo");
  flujo.paso = 3;
  const html = presentador(flujo).renderizarVista("bolsa-candidatos");
  assert.match(html, /<input name="plazo" required minlength="2" maxlength="160" value="Hasta las 23:59:59 del martes, 29 de septiembre de 2026 \(hora peninsular\)" aria-describedby="b7-plazo-procedencia">/);
  assert.match(html, /id="b7-plazo-procedencia" data-b7-plazo-procedencia>Regla de ejemplo<\/span>/);
  assert.match(html, /<summary aria-label="Ayuda sobre el plazo propuesto">\?<\/summary><p>Un día hábil/);
  assert.match(html, /Regla aplicada: vec\.bolsa\.reglas:1:b05\.plazo_respuesta/);
  // Lo que RRHH escriba prevalece sobre la propuesta.
  flujo.configuracion = { plazo: "Hasta el viernes a las 14:00" };
  assert.match(presentador(flujo).renderizarVista("bolsa-candidatos"), /name="plazo"[^>]*value="Hasta el viernes a las 14:00"/);
});

test("B7 no manda al candidato la procedencia ni la referencia de la regla", async () => {
  const { flujo } = await iniciarAsistente({ ok: true, status: 200, json: async () => ({ data: conRegla() }) });
  Object.assign(flujo, { paso: 3, participaciones: ["participacion:001"] });
  const campos = { referencia: "NEC-01", descripcion: "Cobertura", categoria: "Auxiliar", centro: "Centro", modalidad: "Sustitución",
    fecha_inicio: "2026-10-01", plazo: flujo.reglaPlazo.texto, plantilla_version: "bolsa-llamamiento-v1", asunto: "Llamamiento", cuerpo: "Mensaje" };
  const escuchas = {};
  const documento = { addEventListener(tipo, fn) { escuchas[tipo] = fn; }, querySelector() { return null; } };
  const FormDataOriginal = globalThis.FormData;
  globalThis.FormData = class { get(clave) { return campos[clave] ?? null; } };
  try {
    crearControladorBolsas({ estado: { bolsaSeleccionada: "bolsa:01", filtrosBolsa: { nuevo_llamamiento: flujo } }, renderizar: () => {}, navegar: () => {}, documento }).instalar();
    escuchas.submit({ preventDefault() {}, target: { closest(selector) { return selector === '[data-bolsa-form="b7-paso3"]' ? {} : null; } } });
  } finally {
    globalThis.FormData = FormDataOriginal;
  }
  assert.equal(flujo.paso, 4);
  assert.match(flujo.configuracion.cuerpo, /Plazo de respuesta: Hasta las 23:59:59 del martes, 29 de septiembre de 2026 \(hora peninsular\)/);
  const enviado = JSON.stringify(flujo.configuracion);
  assert.doesNotMatch(enviado, /Regla de ejemplo|Reglamento|vec\.bolsa\.reglas|b05/);
});

test("B7 sin catálogo o con la consulta caída queda como hoy, con texto libre", async () => {
  for (const respuesta of [
    { ok: true, status: 200, json: async () => ({ data: sinCatalogo }) },
    { ok: false, status: 503 },
    { ok: false, status: 403 },
  ]) {
    const { flujo } = await iniciarAsistente(respuesta);
    assert.equal(flujo.reglaPlazo, undefined);
    flujo.paso = 3;
    const html = presentador(flujo).renderizarVista("bolsa-candidatos");
    assert.match(html, /<input name="plazo" required minlength="2" maxlength="160" value="">/);
    assert.doesNotMatch(html, /data-b7-plazo-procedencia|data-b7-plazo-regla|Regla de ejemplo/);
  }
});
