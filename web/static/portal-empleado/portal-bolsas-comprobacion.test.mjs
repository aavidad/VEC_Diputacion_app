import test from "node:test";
import assert from "node:assert/strict";
import {
  codificarPropuestaComprobacion, decodificarPropuestaComprobacion, propuestaComprobacionValida, propuestaDesdeComprobacionPendiente,
} from "./portal-bolsas-comprobacion-ruta.js?v=20261019-f157-v1";
import { leerCandidatosBolsaCompartible, rutaCandidatosBolsaCompartible } from "./portal-bolsas-ruta-filtros.js";
import { enlaceComprobacionPendiente } from "./modulos/contratacion-temporal/renderizado-llamamiento.js";
import {
  componerHechos, contratoEvidencia, crearControladorComprobacion, iniciarComprobacion, prepararTextosComprobacion, prerrellenoContrato,
  reciboValido, referenciaDocumentoValida, registrarComprobacion, renderizarComprobacionOfrecimiento, rutaComprobacionOfrecimiento,
} from "./portal-bolsas-comprobacion.js";

await prepararTextosComprobacion();

const escaparHTML = (v) => String(v ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;");
const HEX = "c".repeat(64);
const contexto = Object.freeze({
  ambito_ref: "ambito:rrhh", bolsa_ref: "bolsa:conserjes", categoria_ref: "categoria:c2", causa_ofrecida: "acumulacion",
  expediente_ref: "expediente:ct:7", huella_politica_ofertas_sha256: "d".repeat(64), modalidad_ofrecida: "acumulacion_tareas",
  necesidad_ref: "necesidad:7", orden_acta: 3, organizacion_ref: "organizacion:diputacion", participacion_ref: "participacion:p3",
  unidad_ref: "unidad:servicios", version_bolsa: 4, version_expediente: 2, version_instantanea: 1, version_politica_ofertas: 5,
});
const propuesta = Object.freeze({ contexto, version_esperada: 0 });
const contrato = Object.freeze({
  evento_ref: `evento:ct:contrato-bolsa:${"a".repeat(64)}`, huella_sha256: "b".repeat(64), tipo: "incorporacion",
  inicio: "2026-01-04T00:00:00Z", fin_previsto: "2099-03-31T00:00:00Z", modalidad_clave: "sustitucion",
  causa_clave: "baja_temporal", categoria_ref: "categoria:c2", ocurrido_en: "2026-01-02T09:00:01Z",
});
const recibo = Object.freeze({
  comprobacion_ref: `comprobacion:${"e".repeat(64)}`, recibo_ref: `recibo:comprobacion:${"e".repeat(64)}`, version: 1, version_esperada: 0,
  resultado: "pendiente", motivo_clave: "antecedentes_incompletos", registrada_en: "2026-10-19T08:00:00Z", reutilizada: false,
});

function modalCon(contratos = { carga: "listo", items: [contrato] }) {
  const modal = { candidato: { participacion_ref: "participacion:p3" }, contratosB13: contratos };
  assert.equal(iniciarComprobacion(modal, propuesta, "bolsa:conserjes"), true);
  return modal;
}
function respuesta(status, cuerpo) {
  return { ok: status >= 200 && status < 300, status, json: async () => cuerpo };
}

test("F1-57 la propuesta viaja en la URL con la forma exacta y sin datos personales", () => {
  const ruta = rutaCandidatosBolsaCompartible("", "bolsa:conserjes", "", { comprobacion: propuesta });
  const leido = leerCandidatosBolsaCompartible(ruta.split("#")[0], [{ bolsa_ref: "bolsa:conserjes" }]);
  assert.deepEqual(leido.comprobacion, propuesta);
  assert.equal(decodificarPropuestaComprobacion(codificarPropuestaComprobacion(propuesta)).contexto.orden_acta, 3);
  assert.equal(propuestaComprobacionValida({ contexto: { ...contexto, extra: 1 }, version_esperada: 0 }), null);
  assert.equal(propuestaComprobacionValida({ contexto: { ...contexto, orden_acta: 0 }, version_esperada: 0 }), null);
  assert.equal(decodificarPropuestaComprobacion("{"), null);
  assert.equal(decodificarPropuestaComprobacion("x".repeat(3_073)), null);
});

test("F1-57 una propuesta ilegible o de otra bolsa se descarta sin impedir abrir la bolsa", () => {
  const otra = codificarPropuestaComprobacion({ contexto: { ...contexto, bolsa_ref: "bolsa:otra" }, version_esperada: 0 });
  const autorizadas = [{ bolsa_ref: "bolsa:conserjes" }];
  assert.equal(leerCandidatosBolsaCompartible(`?bolsa_ref=bolsa:conserjes&comprobacion=${encodeURIComponent(otra)}`, autorizadas).comprobacion, undefined);
  assert.equal(leerCandidatosBolsaCompartible("?bolsa_ref=bolsa:conserjes&comprobacion=nada", autorizadas).comprobacion, undefined);
  assert.throws(() => leerCandidatosBolsaCompartible("?bolsa_ref=bolsa:conserjes&estado=activa&comprobacion=x", autorizadas), TypeError);
  assert.throws(() => rutaCandidatosBolsaCompartible("", "bolsa:otra", "", { comprobacion: propuesta }), TypeError);
});

test("F1-57 el 409 de la selección CT produce la propuesta con su versión esperada", () => {
  const comprobacion = { bolsa_ref: "bolsa:conserjes", orden_acta: 3, contexto, estado: { version_esperada: 2 } };
  assert.equal(propuestaDesdeComprobacionPendiente(comprobacion).version_esperada, 2);
  assert.equal(propuestaDesdeComprobacionPendiente({ bolsa_ref: "bolsa:conserjes", orden_acta: 3 }), null);
});

test("F1-57 la referencia del documento sigue la misma guarda que el servidor", () => {
  for (const v of ["registro/2026/0042", "expediente:EXP-2026.17#acta", `ref:${HEX}`]) assert.equal(referenciaDocumentoValida(v), true, v);
  for (const v of ["dni:prueba", "NIE-archivo", "documento:12345678Z", "X1234567L", "ab", "con espacio", "/registro", "a".repeat(257)]) {
    assert.equal(referenciaDocumentoValida(v), false, v);
  }
});

test("F1-57 el contrato registrado prerrellena hechos solo con su evento y su huella", () => {
  assert.equal(contratoEvidencia({ carga: "listo", items: [{ ...contrato, huella_sha256: undefined }] }), null);
  assert.equal(contratoEvidencia({ carga: "listo", items: [contrato] }), contrato);
  assert.deepEqual(prerrellenoContrato(contrato, "2026-10-19"), {
    trabaja_en_diputacion: "si", tipo_relacion: "sustitucion", modalidad: "sustitucion", causa_relacion: "baja_temporal",
    inicio: "2026-01-04", fin_previsto: "2099-03-31",
  });
});

test("F1-57 compone hechos con el contrato y el documento en custodia; la prelación sale del acta", () => {
  const modal = modalCon();
  const st = modal.comprobacionB57;
  st.documentos[0] = { id: "d1", referencia: "registro/2026/0042", huella: HEX };
  Object.assign(st.valores, { es_vacante: "no", antecedentes_completos: "si" });
  Object.assign(st.docs, { es_vacante: "d1", antecedentes_completos: "d1", prelacion: "d1" });
  const { hechos, errores } = componerHechos(st, modal);
  assert.deepEqual(errores, []);
  assert.deepEqual(hechos.inicio, { valor: "2026-01-04", documento_ref: contrato.evento_ref, huella: contrato.huella_sha256 });
  assert.deepEqual(hechos.prelacion, { valor: 3, documento_ref: "registro/2026/0042", huella: HEX });
  assert.deepEqual(hechos.es_vacante, { valor: false, documento_ref: "registro/2026/0042", huella: HEX });
  assert.equal(Object.hasOwn(hechos, "fin_inicial_programa"), false);
});

test("F1-57 sin documento, referencia o archivo válidos no se pasa a la revisión", () => {
  const modal = modalCon({ carga: "listo", items: [] });
  const st = modal.comprobacionB57;
  st.valores.es_vacante = "si";
  assert.deepEqual(componerHechos(st, modal).errores.map((x) => x.campo), ["b57-doc-es_vacante"]);
  st.docs.es_vacante = "d1";
  st.documentos[0].referencia = "dni:12345678Z";
  assert.deepEqual(componerHechos(st, modal).errores.map((x) => x.campo), ["b57-docref-d1", "b57-docarchivo-d1"]);
  st.valores.es_vacante = ""; st.docs.es_vacante = "";
  assert.deepEqual(componerHechos(st, modal).errores.map((x) => x.campo), ["b57-hecho-trabaja_en_diputacion"]);
});

test("F1-57 solo el recibo con forma exacta se presenta como registrado", () => {
  assert.equal(reciboValido(recibo), true);
  assert.equal(reciboValido({ ...recibo, version: 3 }), false);
  assert.equal(reciboValido({ ...recibo, motivo_clave: "sin_supuesto_articulo_9" }), false);
  assert.equal(reciboValido({ ...recibo, recibo_ref: "recibo:otro" }), false);
});

test("F1-57 registra por POST con clave de idempotencia; 4xx rechaza y 5xx queda incierto", async () => {
  const llamadas = [];
  const fetchImpl = (status, cuerpo) => async (url, opciones) => { llamadas.push({ url, opciones }); return respuesta(status, cuerpo); };
  const ok = await registrarComprobacion("bolsa:conserjes", "participacion:p3", "{}", "comprobacion:k1", { fetchImpl: fetchImpl(201, { data: recibo }) });
  assert.deepEqual(ok, { ok: true, recibo });
  assert.equal(llamadas[0].url, rutaComprobacionOfrecimiento("bolsa:conserjes", "participacion:p3"));
  assert.equal(llamadas[0].opciones.method, "POST");
  assert.equal(llamadas[0].opciones.headers["Idempotency-Key"], "comprobacion:k1");
  assert.equal(llamadas[0].opciones.credentials, "same-origin");
  const reutilizada = await registrarComprobacion("b:1", "p:1", "{}", "k", { fetchImpl: fetchImpl(200, { data: { ...recibo, reutilizada: true } }) });
  assert.equal(reutilizada.recibo.reutilizada, true);
  assert.deepEqual(await registrarComprobacion("b:1", "p:1", "{}", "k", { fetchImpl: fetchImpl(409, { error: { codigo: "comprobacion_en_conflicto" } }) }),
    { ok: false, rechazo: true, codigo: "comprobacion_en_conflicto" });
  assert.equal((await registrarComprobacion("b:1", "p:1", "{}", "k", { fetchImpl: fetchImpl(503, null) })).incierto, true);
  assert.equal((await registrarComprobacion("b:1", "p:1", "{}", "k", { fetchImpl: fetchImpl(201, { data: { ...recibo, version: 9 } }) })).incierto, true);
  assert.equal((await registrarComprobacion("b:1", "p:1", "{}", "k", { fetchImpl: async () => { throw new Error("red"); } })).incierto, true);
});

function documentoFalso() {
  return { querySelector: () => null, addEventListener() {} };
}

test("F1-57 un 503 se reintenta con la misma clave y el mismo cuerpo; un rechazo usa clave nueva", async () => {
  const estado = { comprobacionOfrecimiento: propuesta, bolsaSeleccionada: "bolsa:conserjes" };
  const modal = modalCon();
  estado.modalFicha = modal;
  const envios = [];
  const respuestas = [respuesta(503, null), respuesta(201, { data: recibo })];
  let claves = 0;
  const controlador = crearControladorComprobacion({ estado, renderizar() {}, generarClave: () => `comprobacion:k${++claves}`,
    fetchImpl: async (_url, opciones) => { envios.push(opciones); return respuestas.shift(); } });
  const st = modal.comprobacionB57;
  st.documentos[0] = { id: "d1", referencia: "registro/2026/0042", huella: HEX };
  Object.assign(st.valores, { es_vacante: "no" }); st.docs.es_vacante = "d1";
  const evento = (target) => ({ target, preventDefault() {} });
  const formulario = { closest: (s) => (s === "[data-b57-form]" ? formulario : null) };
  assert.equal(controlador.manejarSubmit(evento(formulario), documentoFalso()), true);
  assert.equal(st.paso, "revision");
  const boton = (accion) => { const b = { dataset: { b57Accion: accion } }; b.closest = (s) => (s === "[data-b57-accion]" ? b : null); return b; };
  controlador.manejarClick(evento(boton("registrar")), documentoFalso());
  await new Promise((r) => setImmediate(r));
  assert.equal(st.paso, "incierto");
  controlador.manejarClick(evento(boton("registrar")), documentoFalso());
  await new Promise((r) => setImmediate(r));
  assert.equal(st.paso, "recibo");
  assert.equal(envios[0].headers["Idempotency-Key"], envios[1].headers["Idempotency-Key"]);
  assert.equal(envios[0].body, envios[1].body);
  assert.deepEqual(Object.keys(JSON.parse(envios[0].body)), ["contexto", "hechos", "version_esperada"]);
  assert.equal(estado.comprobacionOfrecimiento, null);
  assert.equal(claves, 1);
});

test("F1-57 el bloque solo aparece para la persona propuesta y explica que la huella no acredita custodia ni firma", () => {
  const otra = { candidato: { participacion_ref: "participacion:otra" } };
  assert.equal(iniciarComprobacion(otra, propuesta, "bolsa:conserjes"), false);
  assert.equal(renderizarComprobacionOfrecimiento({ modal: otra, escaparHTML }), "");
  const modal = modalCon();
  const html = renderizarComprobacionOfrecimiento({ modal, escaparHTML });
  assert.match(html, /<details><summary aria-label="[^"]+">\?<\/summary>/u);
  assert.match(html, /no prueba que el documento esté custodiado ni firmado|no acredita/iu);
  assert.match(html, /type="file" data-b57-doc-archivo="d1"/u);
  const visible = html.replace(/<[^>]+>/gu, " ");
  assert.doesNotMatch(visible, /B57|F1-57|evento:ct|sha-?256|hash/iu);
  modal.comprobacionB57.paso = "recibo"; modal.comprobacionB57.recibo = recibo;
  assert.match(renderizarComprobacionOfrecimiento({ modal, escaparHTML }), /data-b57-recibo/u);
});

test("F1-57 el aviso de la selección CT lleva a la ficha con la propuesta, o a la bolsa si falta contexto", () => {
  const conContexto = enlaceComprobacionPendiente({ bolsa_ref: "bolsa:conserjes", orden_acta: 3, contexto, estado: { version_esperada: 0 } });
  assert.equal(conContexto.clave, "llamamiento_comprobacion_comprobar");
  assert.deepEqual(leerCandidatosBolsaCompartible(conContexto.ruta.split("#")[0], [{ bolsa_ref: "bolsa:conserjes" }]).comprobacion, propuesta);
  const sinContexto = enlaceComprobacionPendiente({ bolsa_ref: "bolsa:conserjes", orden_acta: 3 });
  assert.equal(sinContexto.clave, "llamamiento_comprobacion_ver_bolsa");
  assert.doesNotMatch(sinContexto.ruta, /comprobacion=/u);
});

test("F1-57 los enlaces del resumen de errores llevan el foco al campo sin cambiar de pantalla", () => {
  const estado = { comprobacionOfrecimiento: propuesta, bolsaSeleccionada: "bolsa:conserjes" };
  estado.modalFicha = modalCon();
  const controlador = crearControladorComprobacion({ estado, renderizar() {} });
  let enfocado = "";
  let prevenido = false;
  const radio = { focus: () => { enfocado = "radio"; } };
  const documento = { getElementById: (id) => (id === "b57-hecho-es_vacante" ? { matches: (s) => s === "fieldset", querySelector: () => radio } : null) };
  const enlace = { dataset: { b57Accion: "ir-campo", b57Campo: "b57-hecho-es_vacante" } };
  enlace.closest = (s) => (s === "[data-b57-accion]" ? enlace : null);
  assert.equal(controlador.manejarClick({ target: enlace, preventDefault: () => { prevenido = true; } }, documento), true);
  assert.equal(prevenido, true);
  assert.equal(enfocado, "radio");
});

test("F1-57 el recibo usa el justificante común y no enseña referencias ni versiones", () => {
  const modal = modalCon();
  Object.assign(modal.comprobacionB57, { paso: "recibo", recibo });
  const html = renderizarComprobacionOfrecimiento({ modal, escaparHTML });
  assert.match(html, /data-copiar-justificante="recibo:comprobacion:e{64}"/u);
  assert.doesNotMatch(html.replace(/<[^>]+>/gu, " "), /recibo:comprobacion|Versión/u);
  assert.match(html, /data-vista="contratacion-temporal"/u);
});
