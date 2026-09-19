import assert from "node:assert/strict";
import test from "node:test";

import { presentarEtiquetasHitoRRHH } from "./adaptador-http-expedientes.js";
import { montarFormularioInformeJuridico } from "./formulario-informe-juridico.js";
import { renderizarModuloContratacionTemporal } from "./vista-expedientes.js";

const EXPEDIENTE = "expediente:ct:sintetico:informe-001";
const CLAVE = "123e4567-e89b-42d3-a456-426614174000";

function recibo() {
  return {
    esquema: "vec.contratacion-temporal.recibo-informe-juridico.v1",
    operacion: "preparar",
    expediente_ref: EXPEDIENTE,
    version_resultante: 5,
    informe_ref: "informe:ct:sintetico-001",
    documento_ref: "documento:ct:sintetico-001",
    version_documento: 1,
    formato: "text/plain; charset=utf-8",
    nombre: "informe-juridico-desarrollo.txt",
    huella_documento_sha256: "a".repeat(64),
    recibo_ref: "recibo:ct:informe:sintetico-001",
    auditoria_ref: "auditoria:ct:informe:sintetico-001",
    evento_ref: "evento:ct:informe:sintetico-001",
    contenido_desarrollo:
      "DOCUMENTO DE DESARROLLO — SIN FIRMA NI VALIDEZ JURIDICA\n",
    confirmada_en: "2026-09-04T18:00:00Z",
  };
}

function raizFalsa() {
  const eventos = new Map();
  return {
    innerHTML: "",
    addEventListener(tipo, manejador) { eventos.set(tipo, manejador); },
    removeEventListener(tipo, manejador) {
      if (eventos.get(tipo) === manejador) eventos.delete(tipo);
    },
    contains() { return true; },
    querySelector() { return { focus() {}, scrollIntoView() {} }; },
    replaceChildren() { this.innerHTML = ""; },
    enviar() {
      const formulario = {
        closest(selector) {
          return selector === "[data-ct-informe-form]" ? this : null;
        },
        checkValidity() { return true; },
        reportValidity() {},
      };
      return eventos.get("submit")({ target: formulario, preventDefault() {} });
    },
  };
}

test("conserva documento y recibo si el historial no corresponde", async () => {
  const raiz = raizFalsa();
  const solicitudes = [];
  const confirmados = [];
  montarFormularioInformeJuridico({
    raiz,
    cliente: {
      async prepararInformeJuridico(solicitud) {
        solicitudes.push(solicitud);
        return recibo();
      },
      async consultarDetalleRRHH() {
        return {
          resumen: { expediente_ref: EXPEDIENTE, version: 4 },
          hitos: [{
            secuencia: 4,
            version_expediente: 4,
            accion_clave: "contratacion_temporal.unidad.asignar",
            fase_destino: "asignacion_unidad",
          }],
        };
      },
    },
    contexto: { expediente_ref: EXPEDIENTE, version_esperada: 4 },
    generarClaveIdempotencia: () => CLAVE,
    confirmarOperacion: () => true,
    alConfirmar: (resultado) => confirmados.push(resultado),
  });

  await raiz.enviar();
  assert.deepEqual(solicitudes, [{
    expediente_ref: EXPEDIENTE,
    version_esperada: 4,
    clave_idempotencia: CLAVE,
  }]);
  assert.match(raiz.innerHTML, /data-ct-informe-recibo/u);
  assert.match(raiz.innerHTML, /data-ct-informe-documento/u);
  assert.match(raiz.innerHTML, /data-ct-informe-historial-error/u);
  assert.deepEqual(confirmados, [recibo()]);
});

test("ofrece el informe al reabrir un expediente asignado", () => {
  const expediente = {
    expediente_ref: EXPEDIENTE,
    numero_visible: "2026/CT-001",
    version: 4,
    flujo_ref: "flujo:ct:sintetico",
    flujo_version: 1,
    flujo_huella: "b".repeat(64),
    cabecera: [{ clave: "unidad", valor: "unidad:seleccion" }],
    fases: [],
    tareas: [],
  };
  const html = renderizarModuloContratacionTemporal({
    vista: "expediente",
    carga: "listo",
    cuadro: { expedientes: [{
      expediente_ref: EXPEDIENTE,
      version: 4,
      fase_clave: "asignacion_unidad",
      estado_clave: "en_curso",
    }] },
    expediente,
    tarea_ref: "",
    mensaje_clave: "estado_expediente_listo",
    tipo_mensaje: "informacion",
  }, { informeJuridicoDisponible: true });

  assert.match(html, /data-ct-exp-informe-juridico/u);
});

test("explica el portafirmas pendiente sin simular una firma o envío", () => {
  const expediente = {
    expediente_ref: EXPEDIENTE, numero_visible: "2026/CT-001", version: 5,
    flujo_ref: "flujo:ct:sintetico", flujo_version: 1, flujo_huella: "b".repeat(64),
    cabecera: [], fases: [], tareas: [], historial: [{
      accion_clave: "contratacion_temporal.informe_juridico.generar", accion: "Informe jurídico generado",
    }],
  };
  const html = renderizarModuloContratacionTemporal({
    vista: "expediente", carga: "listo", cuadro: { expedientes: [] }, expediente,
    tarea_ref: "", mensaje_clave: "", tipo_mensaje: "informacion",
  });
  assert.match(html, /Firma de Jefatura y remisión a Intervención/u);
  assert.match(html, /Pendiente de integración con el portafirmas corporativo/u);
  assert.match(html, /VEC no ha enviado el documento ni acredita firma o remisión/u);
  assert.doesNotMatch(html, /<form|Firmado|Enviar a firma/u);
});

test("no mantiene la firma pendiente cuando la historia ya avanzó", () => {
  const expediente = {
    expediente_ref: EXPEDIENTE, numero_visible: "2026/CT-001", version: 6,
    flujo_ref: "flujo:ct:sintetico", flujo_version: 1, flujo_huella: "b".repeat(64),
    cabecera: [], fases: [], tareas: [], historial: [
      { accion_clave: "contratacion_temporal.informe_juridico.generar", accion: "Informe jurídico generado" },
      { accion_clave: "contratacion_temporal.fiscalizacion.registrar", accion: "Fiscalización registrada" },
    ],
  };
  const html = renderizarModuloContratacionTemporal({
    vista: "expediente", carga: "listo", cuadro: { expedientes: [] }, expediente,
    tarea_ref: "", mensaje_clave: "", tipo_mensaje: "informacion",
  });
  assert.doesNotMatch(html, /Firma de Jefatura y remisión a Intervención/u);
});

test("el historial del informe reutiliza etiquetas legibles y traducciones", () => {
  const etiquetas = presentarEtiquetasHitoRRHH({
    accion_clave: "contratacion_temporal.analisis.registrar",
    fase_origen: null, fase_destino: "solicitud", estado_origen: "pendiente", estado_destino: "en_curso",
  }, { hito_analisis: "Análisis revisado" });
  assert.equal(etiquetas.accion, "Análisis revisado");
  assert.equal(etiquetas.faseOrigen, "—");
  assert.equal(etiquetas.faseDestino, "Solicitud");
  assert.equal(etiquetas.estadoDestino, "En tramitación");
});

test("presenta el historial autorizado tras confirmar y conserva traducciones", async () => {
  const raiz = raizFalsa();
  const desmontar = montarFormularioInformeJuridico({
    raiz,
    cliente: {
      async prepararInformeJuridico() { return recibo(); },
      async consultarDetalleRRHH() {
        return { resumen: { expediente_ref: EXPEDIENTE, version: 5 }, hitos: [{
          secuencia: 5, version_expediente: 5,
          accion_clave: "contratacion_temporal.informe_juridico.generar",
          fase_origen: "asignacion_unidad", fase_destino: "informe_juridico",
          estado_origen: "en_curso", estado_destino: "en_curso",
          realizada_en: recibo().confirmada_en,
        }] };
      },
    },
    contexto: { expediente_ref: EXPEDIENTE, version_esperada: 4 },
    generarClaveIdempotencia: () => CLAVE, confirmarOperacion: () => true,
    mensajes: { hito_informe_juridico: "Informe preparado para revisión" },
  });
  await raiz.enviar();
  assert.match(raiz.innerHTML, /Informe preparado para revisión/u);
  assert.match(raiz.innerHTML, /data-ct-informe-recibo/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-informe-historial-error/u);
  desmontar();
});
