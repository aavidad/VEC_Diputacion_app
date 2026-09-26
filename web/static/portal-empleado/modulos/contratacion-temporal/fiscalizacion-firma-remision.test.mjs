import assert from "node:assert/strict";
import test from "node:test";

import { montarFormularioFiscalizacion } from "./formulario-fiscalizacion.js";
import { RUTA_RESULTADOS_FISCALIZACION } from "./cliente-http-fiscalizacion.js";
import { claveI18nValida, codigoValidoParaRuta } from "./cliente-http-transporte.js";
import { RUTAS_HTTP_CONTRATACION_TEMPORAL } from "./cliente-http.js";
import { MENSAJES_FIRMA_REMISION_ES } from "./i18n-firma-remision.js";

function raizFalsa() {
  const eventos = new Map();
  const observaciones = { value: "", required: false, disabled: false, setAttribute() {}, removeAttribute() {} };
  return {
    innerHTML: "",
    addEventListener(tipo, manejador) { eventos.set(tipo, manejador); },
    removeEventListener(tipo, manejador) { if (eventos.get(tipo) === manejador) eventos.delete(tipo); },
    contains() { return true; },
    querySelector(selector) {
      if (selector === "[name=\"observaciones\"]") return observaciones;
      return { focus() {}, scrollIntoView() {} };
    },
    replaceChildren() { this.innerHTML = ""; },
    enviar(resultado) {
      const controles = { resultado: { value: resultado }, observaciones: { value: "" } };
      const formulario = {
        elements: { namedItem: (nombre) => controles[nombre] },
        closest(selector) { return selector === "[data-ct-fiscalizacion-form]" ? this : null; },
        checkValidity() { return true; },
        reportValidity() {},
      };
      return eventos.get("submit")({ target: formulario, preventDefault() {} });
    },
  };
}

function montarConRechazo(codigo) {
  const raiz = raizFalsa();
  montarFormularioFiscalizacion({
    raiz,
    cliente: {
      registrarResultadoFiscalizacion() {
        const error = new Error("rechazada");
        error.codigo = codigo;
        return Promise.reject(error);
      },
    },
    contexto: {
      expediente_ref: "expediente:ct:fiscalizacion:firma-001", version_esperada: 5,
      fase_clave: "informe_juridico", informe_ref: "informe:juridico:firma:001",
    },
    generarClaveIdempotencia: () => "123e4567-e89b-42d3-a456-426614174000",
    confirmarOperacion: () => true,
  });
  return raiz;
}

test("explica que falta la firma que habilita la remisión a Intervención", async () => {
  const raiz = montarConRechazo("firma_remision_pendiente");
  await raiz.enviar("favorable");
  const texto = MENSAJES_FIRMA_REMISION_ES.fiscalizacion_estado_firma_pendiente;
  assert.ok(raiz.innerHTML.includes(texto.slice(0, 40)), raiz.innerHTML);
  assert.match(raiz.innerHTML, /role="alert"/u);
  assert.doesNotMatch(raiz.innerHTML, /firma_remision_pendiente|remision_intervencion/u);
});

test("otros conflictos conservan el mensaje general de rechazo", async () => {
  const raiz = montarConRechazo("conflicto");
  await raiz.enviar("favorable");
  assert.ok(!raiz.innerHTML.includes(MENSAJES_FIRMA_REMISION_ES.fiscalizacion_estado_firma_pendiente.slice(0, 40)));
});

test("el transporte acepta el conflicto de firma y las claves de fiscalización", () => {
  const rutas = RUTAS_HTTP_CONTRATACION_TEMPORAL;
  assert.equal(codigoValidoParaRuta(RUTA_RESULTADOS_FISCALIZACION, 409, "firma_remision_pendiente", rutas), true);
  assert.equal(codigoValidoParaRuta(RUTA_RESULTADOS_FISCALIZACION, 409, "conflicto", rutas), true);
  assert.equal(codigoValidoParaRuta(RUTA_RESULTADOS_FISCALIZACION, 409, "otro_codigo", rutas), false);
  assert.equal(codigoValidoParaRuta(rutas.propuestaCobertura, 409, "firma_remision_pendiente", rutas), false);
  assert.equal(claveI18nValida(RUTA_RESULTADOS_FISCALIZACION, "firma_remision_pendiente",
    "api.contratacion_temporal.fiscalizacion.error.firma_remision_pendiente", rutas), true);
  assert.equal(claveI18nValida(RUTA_RESULTADOS_FISCALIZACION, "conflicto",
    "api.contratacion_temporal.cobertura.error.conflicto", rutas), false);
});
