import assert from "node:assert/strict";
import test from "node:test";

import {
  contextoInformeTrasSubsanacionDesdeEstado,
  crearGestorInformeTrasSubsanacion,
  informeNuevoEmitidoEnSubsanacion,
  montarFormularioInformeTrasSubsanacion,
} from "./informe-tras-subsanacion.js";
import { contextoFiscalizacionDesdeEstado, renderizarModuloContratacionTemporal } from "./vista-expedientes-render.js";
import { montarFormularioFiscalizacion } from "./formulario-fiscalizacion.js";
import { RUTA_RESULTADOS_FISCALIZACION } from "./cliente-http-fiscalizacion.js";
import { RUTA_PREPARACION_INFORME_JURIDICO } from "./cliente-http-informe-juridico.js";
import { codigoValidoParaRuta } from "./cliente-http-transporte.js";
import { RUTAS_HTTP_CONTRATACION_TEMPORAL } from "./cliente-http.js";
import { MENSAJES_INFORME_TRAS_SUBSANACION_ES as M } from "./i18n-informe-tras-subsanacion.js";

const EXP = "expediente:ct:informe-nuevo:001";
const CLAVE = "123e4567-e89b-42d3-a456-426614174000";
const SUBSANACION = "contratacion_temporal.subsanacion_reparos.registrar";
const INFORME = "contratacion_temporal.informe_juridico.generar";

function estadoCon(version, hito) {
  return {
    vista: "expediente", carga: "listo",
    cuadro: { demostracion: false, expedientes: [{ expediente_ref: EXP, version, fase_clave: "subsanacion_unidad", estado_clave: "incidencia" }] },
    expediente: {
      expediente_ref: EXP, version, demostracion: false, numero_visible: "2026/CT-001",
      flujo_ref: "flujo:ct:sintetico", flujo_version: 1, flujo_huella: "b".repeat(64),
      cabecera: [], fases: [], tareas: [],
      historial: [{ secuencia: version, version_expediente: version, fecha: "26 sept 2026", fase: "Subsanación por la unidad",
        accion: "Actuación", estado_clave: "incidencia", estado: "Incidencia", ...hito }],
    },
    tarea_ref: "", mensaje_clave: "estado_expediente_listo", tipo_mensaje: "informacion",
  };
}
const subsanado = () => estadoCon(7, { accion_clave: SUBSANACION, fase_destino: "subsanacion_unidad" });
const conInformeNuevo = () => estadoCon(8, { accion_clave: INFORME, fase_destino: "subsanacion_unidad" });

function reciboInforme(version) {
  return {
    esquema: "vec.contratacion-temporal.recibo-informe-juridico.v1", operacion: "preparar", expediente_ref: EXP,
    version_resultante: version, informe_ref: "informe:ct:nuevo:001", documento_ref: "documento:ct:nuevo:001",
    version_documento: 1, formato: "text/plain; charset=utf-8", nombre: "informe-juridico-desarrollo.txt",
    huella_documento_sha256: "a".repeat(64), recibo_ref: "recibo:ct:informe:nuevo:001", auditoria_ref: "auditoria:ct:001",
    evento_ref: "evento:ct:001", contenido_desarrollo: "DOCUMENTO DE DESARROLLO — SIN FIRMA NI VALIDEZ JURIDICA",
    confirmada_en: "2026-09-26T10:00:00.000000Z",
  };
}

function raizFalsa() {
  const eventos = new Map();
  return {
    innerHTML: "",
    addEventListener(tipo, manejador) { eventos.set(tipo, manejador); },
    removeEventListener(tipo, manejador) { if (eventos.get(tipo) === manejador) eventos.delete(tipo); },
    contains() { return true; },
    querySelector() { return { focus() {}, scrollIntoView() {} }; },
    replaceChildren() { this.innerHTML = ""; },
    pulsar(accion) {
      const control = { dataset: { ctInformeNuevoAccion: accion }, closest() { return this; } };
      return eventos.get("click")({ target: control, preventDefault() {} });
    },
  };
}

test("el botón solo se ofrece con la subsanación registrada y sin informe nuevo", () => {
  assert.deepEqual(contextoInformeTrasSubsanacionDesdeEstado(subsanado()), { expediente_ref: EXP, version_esperada: 7 });
  assert.equal(contextoInformeTrasSubsanacionDesdeEstado(conInformeNuevo()), null);
  assert.equal(informeNuevoEmitidoEnSubsanacion(conInformeNuevo()), true);
  assert.equal(informeNuevoEmitidoEnSubsanacion(subsanado()), false);
  // Tras el informe nuevo se ofrece la nueva fiscalización.
  assert.equal(contextoFiscalizacionDesdeEstado(conInformeNuevo())?.fase_clave, "subsanacion_unidad");
  assert.equal(contextoFiscalizacionDesdeEstado(subsanado())?.version_esperada, 7);
});

test("el detalle pinta el botón tras subsanar y el aviso tras el informe nuevo", () => {
  const opciones = { informeJuridicoDisponible: true, fiscalizacionDisponible: true, subsanacionDisponible: true };
  const tras = renderizarModuloContratacionTemporal(subsanado(), opciones);
  assert.match(tras, /data-ct-exp-informe-nuevo/u);
  const emitido = renderizarModuloContratacionTemporal(conInformeNuevo(), opciones);
  assert.doesNotMatch(emitido, /data-ct-exp-informe-nuevo|data-ct-exp-subsanacion/u);
  assert.ok(emitido.includes(M.informe_nuevo_pendiente_fiscalizacion));
  assert.match(emitido, /data-ct-exp-fiscalizacion/u);
});

test("emite el informe nuevo con la versión vigente, muestra el justificante y avisa", async () => {
  const raiz = raizFalsa();
  const vistas = [];
  let confirmado = null;
  montarFormularioInformeTrasSubsanacion({
    raiz, contexto: { expediente_ref: EXP, version_esperada: 7 },
    cliente: { prepararInformeJuridico(solicitud) { vistas.push(solicitud); return Promise.resolve(reciboInforme(8)); } },
    confirmarOperacion: () => true, generarClaveIdempotencia: () => CLAVE,
    alConfirmar: (recibo) => { confirmado = recibo; },
  });
  assert.ok(raiz.innerHTML.includes(M.informe_nuevo_emitir));
  await raiz.pulsar("emitir");
  assert.deepEqual(vistas, [{ expediente_ref: EXP, version_esperada: 7, clave_idempotencia: CLAVE }]);
  assert.equal(confirmado?.version_resultante, 8);
  assert.ok(raiz.innerHTML.includes(M.informe_nuevo_estado_confirmado));
  assert.doesNotMatch(raiz.innerHTML, /data-ct-informe-nuevo-accion/u);
  assert.doesNotMatch(raiz.innerHTML.replace(/data-copiar-justificante="[^"]*"/gu, ""), /recibo:ct:informe/u);
});

test("sin confirmación no se emite nada", async () => {
  const raiz = raizFalsa();
  let llamadas = 0;
  montarFormularioInformeTrasSubsanacion({
    raiz, contexto: { expediente_ref: EXP, version_esperada: 7 },
    cliente: { prepararInformeJuridico() { llamadas += 1; return Promise.resolve(reciboInforme(8)); } },
    confirmarOperacion: () => false, generarClaveIdempotencia: () => CLAVE,
  });
  await raiz.pulsar("emitir");
  assert.equal(llamadas, 0);
});

test("explica que el catálogo no prevé informe nuevo y recupera un resultado incierto con la misma clave", async () => {
  const noPrevisto = raizFalsa();
  montarFormularioInformeTrasSubsanacion({
    raiz: noPrevisto, contexto: { expediente_ref: EXP, version_esperada: 7 },
    cliente: { prepararInformeJuridico() { const e = new Error("no"); e.codigo = "informe_nuevo_no_previsto"; e.resultadoIndeterminado = false; return Promise.reject(e); } },
    confirmarOperacion: () => true, generarClaveIdempotencia: () => CLAVE,
  });
  await noPrevisto.pulsar("emitir");
  assert.ok(noPrevisto.innerHTML.includes(M.informe_nuevo_estado_no_previsto));
  assert.doesNotMatch(noPrevisto.innerHTML, /informe_nuevo_no_previsto/u);

  const incierto = raizFalsa();
  const claves = [];
  let intentos = 0;
  montarFormularioInformeTrasSubsanacion({
    raiz: incierto, contexto: { expediente_ref: EXP, version_esperada: 7 },
    cliente: {
      prepararInformeJuridico(solicitud) {
        claves.push(solicitud.clave_idempotencia);
        intentos += 1;
        return intentos === 1 ? Promise.reject(new Error("red")) : Promise.resolve(reciboInforme(8));
      },
    },
    confirmarOperacion: () => true,
    generarClaveIdempotencia: () => `123e4567-e89b-42d3-a456-42661417400${claves.length}`,
  });
  await incierto.pulsar("emitir");
  assert.ok(incierto.innerHTML.includes(M.informe_nuevo_estado_indeterminado));
  await incierto.pulsar("recuperar");
  assert.equal(claves.length, 2);
  assert.equal(claves[0], claves[1]);
  assert.ok(incierto.innerHTML.includes(M.informe_nuevo_estado_confirmado));
});

test("el gestor refresca el expediente tras emitir y retira el formulario", async () => {
  const panel = raizFalsa();
  let estado = subsanado();
  const repintados = [];
  const raiz = { querySelector: (s) => (s === "[data-ct-exp-informe-nuevo]" ? panel : null) };
  const gestor = crearGestorInformeTrasSubsanacion({
    raiz, disponible: true, confirmarOperacion: () => true,
    cliente: { prepararInformeJuridico: () => Promise.resolve(reciboInforme(8)) },
    presentador: {
      obtenerEstado: () => estado,
      cargar: async () => {},
      seleccionarExpediente: async () => { estado = conInformeNuevo(); },
    },
    repintar: (foco) => repintados.push(foco),
  });
  assert.equal(gestor.montarSiProcede(), true);
  await panel.pulsar("emitir");
  await new Promise((resolver) => setTimeout(resolver, 0));
  assert.deepEqual(repintados, ["[data-ct-exp-fiscalizacion]"]);
  gestor.retirar();
  assert.equal(panel.innerHTML, "");
  estado = conInformeNuevo();
  assert.equal(gestor.montarSiProcede(), false);
});

test("la fiscalización explica que falta el informe nuevo y el transporte acepta los códigos", async () => {
  const eventos = new Map();
  const raiz = {
    innerHTML: "", addEventListener(t, m) { eventos.set(t, m); }, removeEventListener() {}, contains: () => true,
    querySelector: () => ({ focus() {}, scrollIntoView() {}, value: "", setAttribute() {}, removeAttribute() {} }),
    replaceChildren() {},
  };
  montarFormularioFiscalizacion({
    raiz, contexto: { expediente_ref: EXP, version_esperada: 7, fase_clave: "subsanacion_unidad", informe_ref: "" },
    cliente: { registrarResultadoFiscalizacion() { const e = new Error("x"); e.codigo = "informe_nuevo_pendiente"; return Promise.reject(e); } },
    generarClaveIdempotencia: () => CLAVE, confirmarOperacion: () => true,
  });
  const controles = { resultado: { value: "favorable" }, observaciones: { value: "" } };
  await eventos.get("submit")({ preventDefault() {}, target: {
    elements: { namedItem: (n) => controles[n] }, closest(s) { return s === "[data-ct-fiscalizacion-form]" ? this : null; },
    checkValidity: () => true, reportValidity() {},
  } });
  assert.ok(raiz.innerHTML.includes(M.fiscalizacion_estado_informe_nuevo_pendiente.slice(0, 40)), raiz.innerHTML);
  const rutas = RUTAS_HTTP_CONTRATACION_TEMPORAL;
  assert.equal(codigoValidoParaRuta(RUTA_RESULTADOS_FISCALIZACION, 409, "informe_nuevo_pendiente", rutas), true);
  assert.equal(codigoValidoParaRuta(RUTA_PREPARACION_INFORME_JURIDICO, 409, "informe_nuevo_no_previsto", rutas), true);
  assert.equal(codigoValidoParaRuta(RUTA_PREPARACION_INFORME_JURIDICO, 409, "conflicto", rutas), true);
  assert.equal(codigoValidoParaRuta(RUTA_PREPARACION_INFORME_JURIDICO, 409, "informe_nuevo_pendiente", rutas), false);
});
