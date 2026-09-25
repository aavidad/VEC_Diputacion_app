import test from "node:test";
import assert from "node:assert/strict";
import {
  RUTA_PLANTILLA_CORREO, RUTA_VISTA_PREVIA_CORREO, aplicarPlantillaAlFlujo, consultarPlantillaCorreo, consultarVistaPreviaCorreo,
  crearControladorCorreoLlamamiento, renderizarMarcadoresCorreo, renderizarVistaPreviaCorreo, validarPlantillaCorreo,
} from "./portal-bolsas-correo.js";
import { crearTraductorCorreoLlamamiento, MENSAJES_CORREO_LLAMAMIENTO_ES } from "./portal-i18n-correo-llamamiento.js";
import { crearPresentadorPanelInterno } from "./portal-panel-interno.js";

const plantillaServidor = { plantilla_version: "bolsa-llamamiento-v2", personalizada: true, limite: 4000, limite_asunto: 250, asunto: "Llamamiento de {bolsa}", cuerpo: "Estimado/a {nombre}:", marcadores: [{ clave: "nombre" }, { clave: "posicion" }] };
const respuesta = (status, cuerpo) => ({ status, ok: status < 400, json: async () => cuerpo });
const configuracion = { referencia: "NEC-01", descripcion: "Cobertura", categoria: "Auxiliar", centro: "Centro", modalidad: "Sustitución", fecha_inicio: "2026-10-01", plazo: "48 horas", plantilla_version: "bolsa-llamamiento-v2", asunto: "Aviso", cuerpo: "Hola {nombre}" };

test("valida el contrato de la plantilla y rechaza marcadores mal formados", () => {
  const p = validarPlantillaCorreo(plantillaServidor);
  assert.deepEqual(p.marcadores, ["nombre", "posicion"]);
  assert.equal(validarPlantillaCorreo({ ...plantillaServidor, marcadores: [{ clave: "<script>" }] }), null);
  assert.equal(validarPlantillaCorreo({ ...plantillaServidor, plantilla_version: "otra" }), null);
  assert.equal(validarPlantillaCorreo({ ...plantillaServidor, limite: "4000" }), null);
});

test("consulta la plantilla con la política de red del portal", async () => {
  let llamada;
  const res = await consultarPlantillaCorreo({ fetchImpl: async (ruta, opciones) => { llamada = { ruta, opciones }; return respuesta(200, { data: plantillaServidor }); } });
  assert.equal(res.ok, true);
  assert.equal(llamada.ruta, RUTA_PLANTILLA_CORREO);
  assert.equal(llamada.opciones.credentials, "same-origin");
  assert.equal(llamada.opciones.redirect, "error");
  assert.equal((await consultarPlantillaCorreo({ fetchImpl: async () => respuesta(503, {}) })).ok, false);
  assert.equal((await consultarPlantillaCorreo({ fetchImpl: async () => { throw new Error("red"); } })).ok, false);
});

test("la vista previa envía la configuración y traduce los motivos del servidor", async () => {
  let cuerpo;
  const ok = await consultarVistaPreviaCorreo({ bolsa_ref: "bolsa:01", participacion_ref: "part:01", configuracion }, { fetchImpl: async (ruta, o) => { assert.equal(ruta, RUTA_VISTA_PREVIA_CORREO); cuerpo = JSON.parse(o.body); return respuesta(200, { data: { asunto: "Aviso", cuerpo: "Hola Ana", caracteres: 8, limite: 4000 } }); } });
  assert.deepEqual(ok.datos, { asunto: "Aviso", cuerpo: "Hola Ana", caracteres: 8, limite: 4000 });
  assert.deepEqual(Object.keys(cuerpo), ["bolsa_ref", "participacion_ref", "configuracion"]);
  for (const [codigo, clave] of [["correo_excede_limite", "error_correo_excede_limite"], ["datos_incompletos", "error_datos_incompletos"], ["plantilla_invalida", "error_plantilla_invalida"]]) {
    const r = await consultarVistaPreviaCorreo({ bolsa_ref: "b", participacion_ref: "p", configuracion }, { fetchImpl: async () => respuesta(422, { error: { codigo } }) });
    assert.equal(r.mensaje, MENSAJES_CORREO_LLAMAMIENTO_ES[clave]);
  }
  const denegada = await consultarVistaPreviaCorreo({ bolsa_ref: "b", participacion_ref: "p", configuracion }, { fetchImpl: async () => respuesta(403, {}) });
  assert.equal(denegada.mensaje, MENSAJES_CORREO_LLAMAMIENTO_ES.error_acceso_denegado);
});

test("aplicar la plantilla no pisa lo que RRHH ya escribió", () => {
  const p = validarPlantillaCorreo(plantillaServidor);
  const nuevo = { paso: 1, configuracion: null };
  aplicarPlantillaAlFlujo(nuevo, p);
  assert.equal(nuevo.cuerpoBorrador, "Estimado/a {nombre}:");
  assert.equal(nuevo.configuracion.asunto, "Llamamiento de {bolsa}");
  assert.equal(nuevo.plantilla_version, "bolsa-llamamiento-v2");
  const escrito = { cuerpoBorrador: "Mi texto", configuracion: { asunto: "Mi asunto", plazo: "24 horas" } };
  aplicarPlantillaAlFlujo(escrito, p);
  assert.equal(escrito.cuerpoBorrador, "Mi texto");
  assert.deepEqual(escrito.configuracion, { asunto: "Mi asunto", plazo: "24 horas" });
});

test("marcadores y vista previa se presentan accesibles y sin texto de ayuda", () => {
  const flujo = { correo: validarPlantillaCorreo(plantillaServidor), participaciones: ["part:01", "part:02"], configuracion };
  const marcadores = renderizarMarcadoresCorreo(flujo);
  assert.match(marcadores, /role="group" aria-label="Datos de cada persona que se pueden insertar"/);
  assert.match(marcadores, /data-marcador="nombre" aria-label="Insertar el dato \{nombre\} en el texto">\{nombre\}<\/button>/);
  const vista = renderizarVistaPreviaCorreo(flujo, [{ participacion_ref: "part:01", nombre_visible: "Ana <b>Ruiz</b>" }]);
  assert.match(vista, /Ana &lt;b&gt;Ruiz&lt;\/b&gt;/);
  assert.match(vista, /Persona 2/);
  assert.match(vista, /aria-live="polite"/);
  assert.equal(renderizarMarcadoresCorreo({ correo: { ...flujo.correo, personalizada: false } }), "");
  assert.equal(renderizarVistaPreviaCorreo({ participaciones: ["p"] }), "");
});

test("el controlador inserta marcadores y pinta la vista previa sin re-renderizar el paso", async () => {
  const escuchas = {};
  const campo = { value: "Hola ", selectionStart: 5, selectionEnd: 5, focus() {}, setSelectionRange() {} };
  const region = { innerHTML: "" };
  const selector = { value: "part:02" };
  const documento = {
    addEventListener(tipo, fn) { escuchas[tipo] = fn; },
    querySelector(s) {
      if (s === '[data-bolsa-form="b7-paso3"]') return { querySelector: (q) => (q === 'textarea[name="cuerpo"]' ? campo : null) };
      if (s === "[data-b7-vista-previa-resultado]") return region;
      if (s === '[data-b7-correo="destinatario"]') return selector;
      return null;
    },
  };
  const flujo = { paso: 4, participaciones: ["part:01", "part:02"], configuracion };
  const estado = { bolsaSeleccionada: "bolsa:01", filtrosBolsa: { nuevo_llamamiento: flujo } };
  let renders = 0;
  let enviado;
  const controlador = crearControladorCorreoLlamamiento({ estado, renderizar: () => renders++, fetchImpl: async (ruta, o) => { enviado = JSON.parse(o.body); return respuesta(200, { data: { asunto: "Aviso", cuerpo: "Hola Luis\nPosición 2", caracteres: 20, limite: 4000 } }); } });
  controlador.instalar(documento);
  escuchas.click({ preventDefault() {}, target: { closest: () => ({ dataset: { b7Correo: "insertar", marcador: "nombre" } }) } });
  assert.equal(campo.value, "Hola {nombre}");
  escuchas.click({ preventDefault() {}, target: { closest: () => ({ dataset: { b7Correo: "insertar", marcador: "no válido" } }) } });
  assert.equal(campo.value, "Hola {nombre}");
  await controlador.vistaPrevia(documento);
  assert.equal(enviado.participacion_ref, "part:02");
  assert.match(region.innerHTML, /Hola Luis<br>Posición 2/);
  assert.match(region.innerHTML, /20 de 4000 caracteres/);
  assert.equal(renders, 0);
  selector.value = "ajena";
  enviado = null;
  await controlador.vistaPrevia(documento);
  assert.equal(enviado, null, "no se pide la vista previa de una persona no seleccionada");
});

test("la plantilla se carga una vez y rellena el flujo nuevo", async () => {
  let llamadas = 0;
  const flujo = { paso: 1, configuracion: null };
  const estado = { filtrosBolsa: { nuevo_llamamiento: flujo } };
  let renders = 0;
  const controlador = crearControladorCorreoLlamamiento({ estado, renderizar: () => renders++, fetchImpl: async () => { llamadas++; return respuesta(200, { data: plantillaServidor }); } });
  controlador.prepararFlujo(flujo);
  await new Promise(setImmediate);
  assert.equal(flujo.plantilla_version, "bolsa-llamamiento-v2");
  assert.equal(renders, 1);
  const otro = { paso: 1, configuracion: null };
  controlador.prepararFlujo(otro);
  assert.equal(otro.cuerpoBorrador, "Estimado/a {nombre}:");
  assert.equal(llamadas, 1);
});

test("el asistente usa la versión del catálogo o conserva la literal sin catálogo", () => {
  const candidato = { participacion_ref: "part:01", nombre_visible: "Ana", estado_clave: "disponible", orden: 1 };
  const datos = { generado_en: "2026-09-23T10:00:00Z", bolsa: { bolsa_ref: "bolsa:01", categoria: "Auxiliar", total: 1, por_estado: { disponible: 1 } }, candidatos: [candidato], contactos: [], hay_mas: false, cursor_siguiente: null };
  const html = (flujo) => crearPresentadorPanelInterno({
    claseEstado: () => "info", etiquetaClave: (v) => v, encabezadoVista: () => "", escaparHTML: (v) => String(v ?? ""), numero: (v) => String(v ?? 0),
    obtenerDatosPanel: () => ({ esquema: "vec.bolsa.panel.interno.v1" }), tituloVista: (v) => v,
    obtenerDatosCandidatosBolsa: () => ({ carga: "listo", datos, error: "" }), obtenerEstadoCandidatos: () => ({ estado: "", texto: "", nuevo_llamamiento: flujo }),
  }).renderizarVista("bolsa-candidatos");
  const flujo = { paso: 3, estados: ["disponible"], participaciones: ["part:01"], configuracion: null, error: "", recibo: "" };
  assert.match(html(flujo), /name="plantilla_version" value="bolsa-llamamiento-v1"/);
  assert.doesNotMatch(html(flujo), /data-b7-correo/);
  aplicarPlantillaAlFlujo(flujo, validarPlantillaCorreo(plantillaServidor));
  const paso3 = html(flujo);
  assert.match(paso3, /name="plantilla_version" value="bolsa-llamamiento-v2"/);
  assert.match(paso3, /data-b7-correo="insertar" data-marcador="posicion"/);
  assert.match(paso3, /Estimado\/a \{nombre\}:<\/textarea>/);
  flujo.paso = 4;
  flujo.configuracion = { ...configuracion };
  assert.match(html(flujo), /data-b7-correo="vista-previa"[\s\S]*data-bolsa-form="b7-paso4"/);
});

test("el catálogo i18n del correo está completo", () => {
  assert.throws(() => crearTraductorCorreoLlamamiento({}), TypeError);
  const t = crearTraductorCorreoLlamamiento();
  assert.equal(t("vista_previa_longitud", { caracteres: 5, limite: 10 }), "5 de 10 caracteres");
  assert.throws(() => t("inexistente"), TypeError);
});
