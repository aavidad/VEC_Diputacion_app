import assert from "node:assert/strict";
import test from "node:test";

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
});

test("ofrece el informe al reabrir un expediente asignado", () => {
  const expediente = {
    expediente_ref: EXPEDIENTE,
    numero_visible: "2026/CT-001",
    version: 4,
    flujo_ref: "flujo:ct:sintetico",
    flujo_version: 1,
    flujo_huella: "b".repeat(64),
    cabecera: [],
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
    }] },
    expediente,
    tarea_ref: "",
    mensaje_clave: "estado_expediente_listo",
    tipo_mensaje: "informacion",
  }, { informeJuridicoDisponible: true });

  assert.match(html, /data-ct-exp-informe-juridico/u);
});
