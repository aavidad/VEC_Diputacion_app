import assert from "node:assert/strict";
import test from "node:test";
import { renderizarViasExpediente } from "./vias-expediente.js";
import {
  MENSAJES_EXPEDIENTES_CONTRATACION_ES, SOBRESCRITURAS_VIAS_EXPEDIENTE_EN,
  crearTraductorExpedientesContratacion,
} from "./i18n-expedientes.js";
import { MENSAJES_VIAS_EXPEDIENTE_ES, MENSAJES_VIAS_EXPEDIENTE_EN } from "./i18n-vias-expediente.js";

const traducir = crearTraductorExpedientesContratacion(MENSAJES_VIAS_EXPEDIENTE_ES);

function estadoReal(documentos = true) {
  return {
    vista: "expediente", carga: "listo", navegacion: { documentos },
    expediente: { demostracion: false, cabecera: [
      { clave: "categoria", valor: "Auxiliar & <administración>" },
      { clave: "via_cobertura", valor: "Bolsa vigente" },
      { clave: "bolsa_cobertura", valor: "bolsa:autorizada:123" },
    ] },
  };
}

test("las dos entradas muestran datos autorizados sin crear oferta SAE ni selección", () => {
  const html = renderizarViasExpediente(estadoReal(), traducir);
  assert.match(html, /data-ct-via="bolsa"/);
  assert.match(html, /data-ct-via="sae"/);
  assert.match(html, /POR BOLSA DE TRABAJO/);
  assert.match(html, /POR OFERTA AL SAE/);
  assert.match(html, /Auxiliar &amp; &lt;administración&gt;/);
  assert.doesNotMatch(html, /Auxiliar & <administración>/);
  assert.match(html, /Número de oferta SAE/);
  assert.match(html, /Acta de selección/);
  assert.match(html, /no crea ni envía una oferta/);
  assert.match(html, /<summary aria-label="Ayuda sobre esta vía">\?<\/summary>/);
  assert.match(html, /data-ct-exp-vista="documentos"/);
  assert.doesNotMatch(html, /<form|type="submit"|data-ct-cobertura-accion|data-ct-llamamiento-accion/);
});

test("la lectura no ofrece índice documental sin capacidad y se omite sin detalle real", () => {
  const html = renderizarViasExpediente(estadoReal(false), traducir);
  assert.doesNotMatch(html, /data-ct-exp-vista="documentos"/);
  assert.match(html, /perfil activo/);
  assert.equal(renderizarViasExpediente({ ...estadoReal(), carga: "denegado" }, traducir), "");
  assert.equal(renderizarViasExpediente({ ...estadoReal(), expediente: {
    demostracion: true, cabecera: [],
  } }, traducir), "");
});

test("el traductor real carga las claves nuevas y acepta las sobrescrituras inglesas", () => {
  assert.deepEqual(Object.keys(MENSAJES_VIAS_EXPEDIENTE_ES).sort(), Object.keys(MENSAJES_VIAS_EXPEDIENTE_EN).sort());
  for (const clave of Object.keys(MENSAJES_VIAS_EXPEDIENTE_ES)) {
    assert.equal(MENSAJES_EXPEDIENTES_CONTRATACION_ES[clave], MENSAJES_VIAS_EXPEDIENTE_ES[clave]);
  }
  assert.equal(crearTraductorExpedientesContratacion()("vias_bolsa"), "POR BOLSA DE TRABAJO");
  assert.equal(crearTraductorExpedientesContratacion(SOBRESCRITURAS_VIAS_EXPEDIENTE_EN)("vias_sae"), "Through an SAE job offer");
});
