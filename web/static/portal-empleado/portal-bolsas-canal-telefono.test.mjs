import test from "node:test";
import assert from "node:assert/strict";
import { prepararTextosPortal } from "./portal-i18n.js?v=20261007-pantallas-textos-final-v1";
import { prepararMensajesContratos } from "./portal-bolsas-contratos.js?v=20261007-pantallas-textos-final-v1";
await prepararTextosPortal("bolsa");
await prepararMensajesContratos();
import { crearControladorBolsas, rutaCandidatosBolsa } from "./portal-bolsas-api.js?v=20261008-canal-telefono-v1";
import { validarCanalesLlamamiento } from "./portal-bolsas-contrato.js?v=20261008-canal-telefono-v1";
import { crearControladorIntentosContacto, llamamientoDeFicha, prepararTextosTelefono } from "./portal-bolsas-intentos.js?v=20261008-canal-telefono-v1";
import { crearPresentadorPanelInterno } from "./portal-panel-interno.js?v=20261008-canal-telefono-v1";
import { canalesAviso, filasSeguimiento } from "./portal-bolsas-seguimiento.js";
import { leerCandidatosBolsaCompartible, rutaCandidatosBolsaCompartible } from "./portal-bolsas-ruta-filtros.js";
import { origenLlamamientoValido } from "./portal-llamamiento-origen.js";
import { renderizarExpediente } from "./modulos/contratacion-temporal/componentes-expedientes.js?v=20261008-canal-telefono-v1";
import { crearTraductorExpedientesContratacion } from "./modulos/contratacion-temporal/i18n-expedientes.js?v=20261007-pantallas-textos-final-v1";

await prepararTextosTelefono();

const CORREO = { canal: "correo", modo: "automatico", al_emitir: true, seguimiento: false };
const TELEFONO = { canal: "telefono", modo: "manual", al_emitir: false, seguimiento: true,
  resultados: ["contactado", "no_contesta", "comunica", "numero_erroneo", "acepta", "rechaza", "aplazado", "buzon"],
  resultados_cierre: ["acepta", "rechaza", "numero_erroneo"] };
const LLAMAMIENTO = "llamamiento:sintetico:7";
const BOLSA = { bolsa_ref: "bolsa:sintetica:1", categoria_clave: "auxiliar_enfermeria", categoria: "Auxiliar de enfermería", tipo_lista: "rotatoria",
  vigente_desde: "2026-01-15", vigente_hasta: null, total: 3, por_estado: { disponible: 3 }, llamamientos_en_curso: 1,
  politica_orden: { politica_ref: "politica:1", version: 1, criterio: "puntuacion", tipo_lista: "rotatoria", reposicion: "misma_posicion", provisional: false, rotulo: "", actor: "actor:1", vigente_desde: "2026-01-15" } };
const persona = (n, nombre, extra = {}) => ({ participacion_ref: `participacion:sintetica:${n}`, orden: n, orden_acta: n, razon_orden: "orden_acta",
  nombre_visible: nombre, documento_enmascarado: `***${String(1000 + n)}**`, estado_clave: "disponible", estado_desde: "2026-09-01T08:00:00Z",
  disponible_desde: null, contactos_total: 0,
  ultimo_llamamiento: { llamamiento_ref: LLAMAMIENTO, comunicado_en: "2026-10-08T07:00:00Z", canal: "correo", resultado: "pendiente" }, ...extra });
const CANDIDATOS = [persona(1, "Antonio Reyes Álvarez"), persona(2, "Lucía Martín Serrano"), persona(3, "Carmen Molina Ortega")];
const contacto = (n, canal, resultado, instante, llamamiento = LLAMAMIENTO) => ({ contacto_ref: `contacto:${n}:${instante}`,
  participacion_ref: `participacion:sintetica:${n}`, llamamiento_ref: llamamiento, canal, instante, actor_ref: "actor:rrhh", resultado, anotacion: "" });

function html(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;");
}
function presentador({ datos, filtros, modal = null }) {
  return crearPresentadorPanelInterno({
    claseEstado: () => "info", encabezadoVista: (_s, titulo, _d, acciones = "") => `<header><h2>${titulo}</h2>${acciones}</header>`,
    escaparHTML: html, numero: (n) => String(n ?? 0),
    obtenerDatosPanel: () => ({ esquema: "vec.bolsa.panel.interno.v1" }), tituloVista: (v) => v,
    obtenerDatosCandidatosBolsa: () => datos, obtenerEstadoCandidatos: () => filtros, obtenerModalFicha: () => modal,
  });
}
const listo = (extra = {}) => ({ carga: "listo", error: "", datos: { bolsa: BOLSA, candidatos: CANDIDATOS, contactos: [], hay_mas: false, cursor_siguiente: null, ...extra } });

test("el contrato acepta solo canales conocidos y bien formados", () => {
  const canales = validarCanalesLlamamiento([CORREO, TELEFONO, { canal: "telegram", modo: "manual", al_emitir: false, seguimiento: false }]);
  assert.deepEqual(canales.map((c) => c.canal), ["correo", "telefono"]);
  assert.deepEqual(validarCanalesLlamamiento([{ ...TELEFONO, resultados_cierre: ["otro"] }]), []);
  assert.deepEqual(validarCanalesLlamamiento([{ ...TELEFONO, resultados: [] }]), []);
  assert.deepEqual(validarCanalesLlamamiento("telefono"), []);
  assert.equal(canalesAviso({}).telefono, null, "sin el campo, solo correo");
  assert.equal(canalesAviso({ canales_llamamiento: canales }).telefono.canal, "telefono");
});

test("el paso 3 ofrece canales complementarios: correo fijo y teléfono si está activo", () => {
  const flujo = { paso: 3, estados: ["disponible"], participaciones: ["participacion:sintetica:1"], configuracion: null };
  const soloCorreo = presentador({ datos: listo(), filtros: { nuevo_llamamiento: flujo } }).renderizarVista("bolsa-candidatos");
  assert.match(soloCorreo, /<legend>Canales de aviso<\/legend><label><input type="checkbox" checked disabled> Correo electrónico al emitir el llamamiento<\/label><\/fieldset>/u);
  assert.doesNotMatch(soloCorreo, /seguimiento_telefono/u);
  const dos = presentador({ datos: listo({ canales_llamamiento: validarCanalesLlamamiento([CORREO, TELEFONO]) }), filtros: { nuevo_llamamiento: flujo } }).renderizarVista("bolsa-candidatos");
  assert.match(dos, /<input type="checkbox" name="seguimiento_telefono" value="1" checked> Seguimiento por teléfono/u);
  assert.doesNotMatch(dos, /type="radio"/u, "no es un selector excluyente");
  flujo.canales = { correo: true, telefono: false };
  const desmarcado = presentador({ datos: listo({ canales_llamamiento: validarCanalesLlamamiento([CORREO, TELEFONO]) }), filtros: { nuevo_llamamiento: flujo } }).renderizarVista("bolsa-candidatos");
  assert.match(desmarcado, /name="seguimiento_telefono" value="1">/u);
});

test("con origen en una petición, referencia, centro y fecha llegan rellenos y de solo lectura", () => {
  const origen = { referencia: "2026/CT-00042", centro: "Residencia La Milagrosa", fecha_inicio: "2026-10-20" };
  const conOrigen = presentador({ datos: listo(), filtros: { nuevo_llamamiento: { paso: 3, participaciones: [], origen } } }).renderizarVista("bolsa-candidatos");
  assert.match(conOrigen, /<input name="referencia" required minlength="2" maxlength="160" value="2026\/CT-00042" readonly>/u);
  assert.match(conOrigen, /<input name="centro" required minlength="2" maxlength="200" value="Residencia La Milagrosa" readonly>/u);
  assert.match(conOrigen, /<input type="date" name="fecha_inicio" required value="2026-10-20" readonly>/u);
  assert.doesNotMatch(conOrigen, /disabled>.*name="referencia"|name="referencia"[^>]*disabled/u);
  const sinOrigen = presentador({ datos: listo(), filtros: { nuevo_llamamiento: { paso: 3, participaciones: [] } } }).renderizarVista("bolsa-candidatos");
  assert.match(sinOrigen, /<input name="referencia" required minlength="2" maxlength="160" value="">/u);
  assert.doesNotMatch(sinOrigen, /readonly/u);
});

test("la revisión y el recibo dicen los canales elegidos y llevan a las llamadas", () => {
  const configuracion = { referencia: "2026/CT-00042", centro: "Residencia", modalidad: "Sustitución", plazo: "24 horas" };
  const flujo = { paso: 4, participaciones: ["participacion:sintetica:1"], configuracion, canales: { correo: true, telefono: true } };
  let vista = presentador({ datos: listo(), filtros: { nuevo_llamamiento: flujo } }).renderizarVista("bolsa-candidatos");
  assert.match(vista, /<dt>Canales de aviso<\/dt><dd>Correo electrónico y seguimiento por teléfono\.<\/dd>/u);
  assert.doesNotMatch(vista, /por separado/u);
  Object.assign(flujo, { recibo: "recibo:sintetico:1", llamamiento_ref: LLAMAMIENTO });
  vista = presentador({ datos: listo(), filtros: { nuevo_llamamiento: flujo } }).renderizarVista("bolsa-candidatos");
  assert.match(vista, /<a class="boton-primario" href="\?bolsa_ref=bolsa%3Asintetica%3A1&amp;seguimiento=llamamiento%3Asintetico%3A7#bolsa\/bolsa-candidatos" data-accion="ver-bolsa" data-bolsa-ref="bolsa:sintetica:1" data-seguimiento="llamamiento:sintetico:7">Empezar las llamadas<\/a>/u);
  flujo.canales = { correo: true, telefono: false };
  vista = presentador({ datos: listo(), filtros: { nuevo_llamamiento: flujo } }).renderizarVista("bolsa-candidatos");
  assert.match(vista, /<dt>Canales de aviso<\/dt><dd>Correo electrónico\.<\/dd>/u);
  assert.doesNotMatch(vista, /Empezar las llamadas/u);
});

test("el seguimiento marca como siguiente a la primera persona sin resultado de cierre", () => {
  const contactos = [contacto(1, "correo", "enviado", "2026-10-08T07:00:00Z"), contacto(1, "telefono", "no_contesta", "2026-10-08T08:00:00Z"),
    contacto(1, "telefono", "rechaza", "2026-10-08T09:00:00Z"), contacto(2, "telefono", "comunica", "2026-10-08T09:30:00Z"),
    contacto(3, "telefono", "acepta", "2026-10-08T09:40:00Z", "llamamiento:otro")];
  const filas = filasSeguimiento({ candidatos: [...CANDIDATOS].reverse(), contactos, llamamientoRef: LLAMAMIENTO, telefono: TELEFONO });
  assert.deepEqual(filas.map((f) => f.candidato.orden), [1, 2, 3], "orden de la bolsa");
  assert.deepEqual(filas.map((f) => [f.llamadas, f.cerrada, f.siguiente]), [[2, true, false], [1, false, true], [0, false, false]]);
  assert.equal(filas[0].ultimaLlamada.resultado, "rechaza");
  assert.equal(filas[0].correo.resultado, "enviado");

  const filtros = { estado: "", texto: "", seguimiento: { llamamiento_ref: LLAMAMIENTO, bolsa_ref: BOLSA.bolsa_ref } };
  const datos = listo({ contactos, canales_llamamiento: validarCanalesLlamamiento([CORREO, TELEFONO]) });
  const vista = presentador({ datos, filtros }).renderizarVista("bolsa-candidatos");
  assert.match(vista, /<h2>Seguimiento por teléfono: Auxiliar de enfermería<\/h2><a class="boton-secundario" href="\?bolsa_ref=bolsa%3Asintetica%3A1#bolsa\/bolsa-candidatos" data-accion="ver-bolsa" data-bolsa-ref="bolsa:sintetica:1">Volver a la bolsa<\/a>/u);
  assert.match(vista, /class="tabla-contenedor" tabindex="0" role="region"/u);
  assert.equal(vista.match(/Siguiente a llamar<\/span>/gu)?.length, 1);
  assert.match(vista, /Lucía Martín Serrano<\/strong><\/button> <span class="estado-chip info">Siguiente a llamar<\/span>/u);
  assert.match(vista, /Llamadas: 2<\/button><br>Indica que rechaza/u);
  assert.match(vista, /Llamadas: 1<\/button><br>Comunica/u);
  assert.match(vista, /<button type="button" class="boton-primario" data-bolsa-accion="abrir-ficha" data-bolsa-control-principal="true" data-participacion-ref="participacion:sintetica:2"[^>]*aria-label="Abrir la ficha de Lucía Martín Serrano para llamarle">Ver ficha y llamar<\/button>/u);
});

test("el seguimiento presenta carga, error con reintento, vacío y teléfono apagado", () => {
  const filtros = { seguimiento: { llamamiento_ref: LLAMAMIENTO, bolsa_ref: BOLSA.bolsa_ref } };
  const ver = (datos) => presentador({ datos, filtros }).renderizarVista("bolsa-candidatos");
  assert.match(ver({ carga: "cargando" }), /role="status" aria-busy="true"><p><strong>Cargando las personas del llamamiento…/u);
  const error = ver({ carga: "error", error: "" });
  assert.match(error, /role="alert"><p><strong>No se ha podido cargar el seguimiento/u);
  assert.match(error, /data-bolsa-accion="reintentar-candidatos">Reintentar/u);
  assert.match(ver(listo({ candidatos: [], canales_llamamiento: validarCanalesLlamamiento([TELEFONO]) })), /Este llamamiento no tiene personas\./u);
  assert.match(ver(listo()), /El seguimiento por teléfono no está activado\./u);
});

test("la lista de la bolsa ofrece el seguimiento solo con el teléfono activo", () => {
  const filtros = { estado: "", texto: "" };
  const con = presentador({ datos: listo({ canales_llamamiento: validarCanalesLlamamiento([CORREO, TELEFONO]) }), filtros }).renderizarVista("bolsa-candidatos");
  assert.equal(con.match(/data-seguimiento="llamamiento:sintetico:7"/gu)?.length, 3);
  assert.match(con, /aria-label="Seguimiento por teléfono del llamamiento de Antonio Reyes Álvarez">Seguimiento<\/a>/u);
  const sin = presentador({ datos: listo(), filtros }).renderizarVista("bolsa-candidatos");
  assert.doesNotMatch(sin, /data-seguimiento=/u);
});

test("en la ficha, el llamamiento del seguimiento prevalece sobre el último de la persona", async () => {
  const candidato = persona(2, "Lucía Martín Serrano", { ultimo_llamamiento: { ...CANDIDATOS[1].ultimo_llamamiento, llamamiento_ref: "llamamiento:posterior" } });
  const modal = { abierto: true, candidato, bolsa: BOLSA, llamamientoSeguimiento: LLAMAMIENTO };
  assert.equal(llamamientoDeFicha(modal), LLAMAMIENTO);
  assert.equal(llamamientoDeFicha({ candidato }), "llamamiento:posterior");
  const peticiones = [];
  const estado = { bolsaSeleccionada: BOLSA.bolsa_ref, modalFicha: modal };
  const fetchImpl = async (url, opciones) => {
    peticiones.push({ url, opciones });
    if (opciones.method === "POST") {
      const cuerpo = JSON.parse(opciones.body);
      return { status: 201, json: async () => ({ data: { recibo_ref: "recibo:1", participacion_ref: candidato.participacion_ref, llamamiento_ref: cuerpo.llamamiento_ref,
        canal: "telefono", resultado: cuerpo.resultado, anotacion: cuerpo.anotacion, instante: "2026-10-08T10:00:00Z" } }) };
    }
    return { ok: true, status: 200, json: async () => ({ data: { esquema: "vec.bolsa.rrhh.contactos.v1", intentos: { configurado: false,
      registro_telefono: { esquema: "vec.bolsa.registro_telefono.v1", instante_servidor: true, anotacion_opcional: true, resultados: TELEFONO.resultados } } } }) };
  };
  const controlador = crearControladorIntentosContacto({ estado, renderizar: () => {}, fetchImpl });
  await controlador.cargar(modal);
  assert.match(peticiones[0].url, /\?llamamiento_ref=llamamiento%3Asintetico%3A7$/u);
  const formulario = { dataset: { intentosForm: "intento" }, closest: () => formulario };
  const FormDataOriginal = globalThis.FormData;
  globalThis.FormData = class { get(campo) { return { resultado: "comunica", anotacion: "" }[campo] ?? null; } };
  try {
    controlador.manejarSubmit({ target: formulario, preventDefault() {} });
    await new Promise(setImmediate);
  } finally {
    globalThis.FormData = FormDataOriginal;
  }
  const envio = peticiones.find((p) => p.opciones.method === "POST");
  assert.deepEqual(JSON.parse(envio.opciones.body), { canal: "telefono", resultado: "comunica", anotacion: "", llamamiento_ref: LLAMAMIENTO });
  const vista = presentador({ datos: listo({ canales_llamamiento: validarCanalesLlamamiento([TELEFONO]) }),
    filtros: { seguimiento: { llamamiento_ref: LLAMAMIENTO, bolsa_ref: BOLSA.bolsa_ref } }, modal }).renderizarVista("bolsa-candidatos");
  assert.match(vista, /<td colspan="5">/u, "la ficha ocupa las columnas del seguimiento");
});

test("la consulta del seguimiento pide solo las personas de ese llamamiento", async () => {
  assert.equal(rutaCandidatosBolsa("bolsa:1", { llamamiento: LLAMAMIENTO, estado: "disponible", texto: "x", cursor: "c" }),
    "/api/vec/bolsa/bolsas/bolsa:1/candidatos?llamamiento=llamamiento%3Asintetico%3A7&limite=100");
  const escuchas = {};
  const documento = { addEventListener(tipo, fn) { escuchas[tipo] = fn; }, querySelector() { return null; } };
  const estado = { filtrosBolsa: { estado: "", texto: "" }, datosCandidatos: null };
  const rutas = [];
  const fetchOriginal = globalThis.fetch;
  globalThis.fetch = async (url) => { rutas.push(url); return { ok: false, status: 503, json: async () => ({}) }; };
  try {
    crearControladorBolsas({ estado, renderizar: () => {}, navegar: () => {}, documento }).instalar();
    const enlace = { dataset: { accion: "ver-bolsa", bolsaRef: BOLSA.bolsa_ref, seguimiento: LLAMAMIENTO } };
    escuchas.click({ preventDefault() {}, target: { closest: (selector) => (selector.includes("ver-bolsa") ? enlace : null) } });
    await new Promise(setImmediate);
  } finally {
    globalThis.fetch = fetchOriginal;
  }
  assert.deepEqual(estado.filtrosBolsa.seguimiento, { llamamiento_ref: LLAMAMIENTO, bolsa_ref: BOLSA.bolsa_ref });
  assert.ok(rutas.some((ruta) => ruta.endsWith("/candidatos?llamamiento=llamamiento%3Asintetico%3A7&limite=100")), rutas.join(" "));
  assert.equal(estado.datosCandidatos.carga, "error");
});

test("iniciar el llamamiento copia el origen de la petición de esa bolsa", async () => {
  const escuchas = {};
  const documento = { addEventListener(tipo, fn) { escuchas[tipo] = fn; }, querySelector() { return null; } };
  const origen = { bolsa_ref: BOLSA.bolsa_ref, expediente_ref: "expediente:sintetico:42", referencia: "2026/CT-00042", centro: "Residencia La Milagrosa" };
  const estado = { vista: "bolsa-candidatos", bolsaSeleccionada: BOLSA.bolsa_ref, origenLlamamientoB7: origen, filtrosBolsa: {}, datosCandidatos: listo() };
  const fetchOriginal = globalThis.fetch;
  globalThis.fetch = async () => ({ ok: false, status: 404 });
  try {
    crearControladorBolsas({ estado, renderizar: () => {}, navegar: () => {}, documento }).instalar();
    escuchas.click({ preventDefault() {}, target: { closest: (s) => (s === "[data-bolsa-accion]" ? { dataset: { bolsaAccion: "iniciar-b7" } } : null) } });
    await new Promise(setImmediate);
  } finally {
    globalThis.fetch = fetchOriginal;
  }
  assert.deepEqual(estado.filtrosBolsa.nuevo_llamamiento.origen, { referencia: "2026/CT-00042", centro: "Residencia La Milagrosa" });
  estado.bolsaSeleccionada = "bolsa:otra";
  estado.filtrosBolsa = {};
  escuchas.click({ preventDefault() {}, target: { closest: (s) => (s === "[data-bolsa-accion]" ? { dataset: { bolsaAccion: "iniciar-b7" } } : null) } });
  assert.equal(estado.filtrosBolsa.nuevo_llamamiento.origen, null, "otra bolsa no hereda el origen");
});

test("la URL lleva seguimiento u origen validados y nunca ambos", () => {
  const ruta = rutaCandidatosBolsaCompartible("?lang=es", "bolsa:1", "", { seguimiento: LLAMAMIENTO });
  assert.equal(ruta, "?lang=es&bolsa_ref=bolsa%3A1&seguimiento=llamamiento%3Asintetico%3A7#bolsa/bolsa-candidatos");
  const autorizadas = [{ bolsa_ref: "bolsa:1" }];
  assert.deepEqual(leerCandidatosBolsaCompartible(ruta.split("#")[0], autorizadas), { bolsaRef: "bolsa:1", estado: "", seguimiento: LLAMAMIENTO });
  const origen = { expediente_ref: "expediente:sintetico:42", referencia: "2026/CT-00042", centro: "Residencia La Milagrosa", fecha_inicio: "2026-10-20" };
  const conOrigen = rutaCandidatosBolsaCompartible(ruta.split("#")[0], "bolsa:1", "", { origen });
  assert.doesNotMatch(conOrigen, /seguimiento=/u, "el origen sustituye al seguimiento anterior");
  assert.deepEqual(leerCandidatosBolsaCompartible(conOrigen.split("#")[0], autorizadas).origen, origen);
  assert.throws(() => leerCandidatosBolsaCompartible("?bolsa_ref=bolsa%3A1&seguimiento=x&origen_expediente=e&origen_referencia=ab", autorizadas), TypeError);
  assert.throws(() => leerCandidatosBolsaCompartible("?bolsa_ref=bolsa%3A1&seguimiento=x&estado=disponible", autorizadas), TypeError);
  const malCentro = leerCandidatosBolsaCompartible("?bolsa_ref=bolsa%3A1&origen_expediente=e1&origen_referencia=CT-1&origen_centro=%3Cscript%3E&origen_inicio=2026-02-30", autorizadas);
  assert.deepEqual(malCentro.origen, { expediente_ref: "e1", referencia: "CT-1" }, "centro y fecha inválidos se descartan");
  assert.equal(origenLlamamientoValido({ expediente_ref: "e1", referencia: "x".repeat(161) }), null);
  assert.equal(origenLlamamientoValido({ expediente_ref: "e1", referencia: "CT-1", centro: "c".repeat(201) }).centro, undefined);
  const raro = presentador({ datos: listo({ canales_llamamiento: validarCanalesLlamamiento([TELEFONO]),
    candidatos: [persona(1, "Antonio Reyes Álvarez", { ultimo_llamamiento: { llamamiento_ref: "a/b", comunicado_en: "2026-10-08T07:00:00Z", canal: "correo", resultado: "pendiente" } })] }),
  filtros: { estado: "", texto: "" } }).renderizarVista("bolsa-candidatos");
  assert.doesNotMatch(raro, /data-seguimiento=/u, "una referencia no enlazable no rompe la lista");
});

const t = crearTraductorExpedientesContratacion({});
const campoCT = (clave, valor) => ({ clave, etiqueta: clave, valor, tono: "neutro", control: "solo_lectura", obligatorio: false, opciones: [] });
const expedienteCT = (cabecera, extra = {}) => ({ carga: "listo", tarea_ref: "", expediente: { expediente_ref: "expediente:sintetico:42", numero_visible: "2026/CT-00042",
  version: 3, fases: [], tareas: [], hitos: [], cabecera, ...extra } });

test("la ficha CT abre el llamamiento en la bolsa de la categoría con su origen", () => {
  const llamadas = [];
  const resolver = (ref, { categoriaRef = "" } = {}) => {
    llamadas.push([ref, categoriaRef]);
    return !ref && categoriaRef === "categoria:rpt:auxiliar_enfermeria" ? { categoria: BOLSA.categoria, bolsa_ref: BOLSA.bolsa_ref } : null;
  };
  const estado = expedienteCT([campoCT("centro", "Residencia La Milagrosa")], {
    datos_peticion: { categoria_ref: "categoria:rpt:auxiliar_enfermeria", grupo_subgrupo: "C2", periodo: { inicio: "2026-10-20", fin: "2026-12-31" } } });
  const vista = renderizarExpediente(estado, t, "es-ES", "Europe/Madrid", false, resolver);
  assert.match(vista, /<button type="button" class="boton-primario" data-accion="ver-bolsa" data-bolsa-ref="bolsa:sintetica:1" data-origen-expediente="expediente:sintetico:42" data-origen-referencia="2026\/CT-00042" data-origen-centro="Residencia La Milagrosa" data-origen-inicio="2026-10-20">Abrir llamamiento en Bolsa<\/button>/u);
  const cobertura = renderizarExpediente(expedienteCT([campoCT("bolsa_cobertura", "bolsa:sintetica:9")]), t, "es-ES", "Europe/Madrid", false,
    (ref) => (ref === "bolsa:sintetica:9" ? { categoria: "Peón" } : null));
  assert.match(cobertura, /data-accion="ver-bolsa" data-bolsa-ref="bolsa:sintetica:9" data-origen-expediente=/u, "la bolsa de cobertura manda");
});

test("sin bolsa para la categoría la ficha CT no ofrece el llamamiento", () => {
  const estado = expedienteCT([campoCT("centro", "Residencia")], { datos_peticion: { categoria_ref: "categoria:rpt:otra", grupo_subgrupo: "C2", periodo: { inicio: "2026-10-20", fin: "2026-12-31" } } });
  for (const resolver of [null, () => null]) {
    assert.doesNotMatch(renderizarExpediente(estado, t, "es-ES", "Europe/Madrid", false, resolver), /Abrir llamamiento en Bolsa|data-origen-/u);
  }
});
