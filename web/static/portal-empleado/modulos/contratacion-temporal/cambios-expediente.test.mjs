import test from "node:test";
import assert from "node:assert/strict";
import {
  ACCEPT_CAMBIOS_EXPEDIENTE, ESQUEMA_CAMBIOS_EXPEDIENTE, crearClienteHTTPCambiosExpediente, validarCambiosExpediente,
} from "./cliente-http-cambios-expediente.js";
import {
  crearTraductorCambiosExpediente, etiquetaRutaCambio, renderizarCambiosExpediente, renderizarTablaCambios, valorVisibleCambio,
} from "./vista-expedientes-cambios.js";

const protegido = "*protegido";
const cuerpo = (cambios, recortado = false) => ({ data: { esquema: ESQUEMA_CAMBIOS_EXPEDIENTE, expediente_ref: "expediente:ct:1", version_expediente: 3, cambios, recortado } });
const cambio = (extra = {}) => ({ version_expediente: 2, registrada_en: "2026-09-25T09:00:00.000000Z", origen_version: "analisis_o3", ruta: "solicitud.grupo_subgrupo", valor_anterior: "C2", valor_nuevo: "C1", ...extra });

test("petición RRHH p.4: el cliente pide los cambios por la ruta del detalle con su Accept", async () => {
  let peticion;
  const cliente = crearClienteHTTPCambiosExpediente({ fetchImpl: async (ruta, opciones) => {
    peticion = { ruta, opciones };
    return new Response(JSON.stringify(cuerpo([cambio()])), { status: 200, headers: { "Content-Type": "application/json; charset=utf-8" } });
  } });
  const resultado = await cliente.consultarCambios({ expediente_ref: "expediente:ct:1", version_observada: 3 });
  assert.equal(peticion.ruta, "/api/vec/contratacion-temporal/expedientes/consultas");
  assert.equal(peticion.opciones.headers.Accept, ACCEPT_CAMBIOS_EXPEDIENTE);
  assert.deepEqual([peticion.opciones.method, peticion.opciones.credentials, peticion.opciones.cache, peticion.opciones.redirect], ["POST", "same-origin", "no-store", "error"]);
  assert.equal(resultado.cambios.length, 1);
  await assert.rejects(crearClienteHTTPCambiosExpediente({ fetchImpl: async () => new Response("{}", { status: 403 }) })
    .consultarCambios({ expediente_ref: "expediente:ct:1", version_observada: 3 }));
});

test("petición RRHH p.4: el contrato rechaza campos, rutas u otro expediente", () => {
  assert.equal(validarCambiosExpediente(cuerpo([cambio(), cambio({ version_expediente: 3, valor_anterior: null, valor_nuevo: protegido })]), "expediente:ct:1").cambios.length, 2);
  for (const malo of [
    cuerpo([cambio({ actor: "per_x" })]),
    cuerpo([cambio({ ruta: "solicitud..x" })]),
    cuerpo([cambio({ version_expediente: 4 })]),
    cuerpo([cambio({ valor_anterior: null, valor_nuevo: null })]),
    cuerpo([cambio({ valor_nuevo: "linea\nsalto" })]),
    { data: { ...cuerpo([]).data, expediente_ref: "expediente:ct:2" } },
    { data: { ...cuerpo([]).data, recortado: "no" } },
    { data: { esquema: ESQUEMA_CAMBIOS_EXPEDIENTE, expediente_ref: "expediente:ct:1", version_expediente: 3, cambios: [] } },
  ]) assert.throws(() => validarCambiosExpediente(malo, "expediente:ct:1"));
});

test("petición RRHH p.4: la tabla muestra anterior y nuevo sin texto libre en claro", () => {
  const t = crearTraductorCambiosExpediente();
  assert.equal(etiquetaRutaCambio("analisis.periodo.inicio", t), "Análisis · Período · Inicio");
  assert.equal(etiquetaRutaCambio("solicitud.documentos_adjuntos[0]", t), "Solicitud · documentos adjuntos 1");
  assert.equal(valorVisibleCambio(null, t), "—");
  assert.equal(valorVisibleCambio(protegido, t), "Dato protegido (cambiado)");
  const html = renderizarTablaCambios(validarCambiosExpediente(cuerpo([cambio(), cambio({ version_expediente: 3, ruta: "analisis.observaciones", valor_anterior: null, valor_nuevo: protegido })]), "expediente:ct:1"), t);
  assert.match(html, /<caption>Valor anterior y nuevo de cada campo cambiado, por versión<\/caption>/u);
  assert.match(html, /<th scope="row">Solicitud · Grupo\/Subgrupo<\/th><td>C2<\/td><td>C1<\/td>/u);
  assert.match(html, /<th scope="row">Análisis · Observaciones<\/th><td>—<\/td><td>Dato protegido \(cambiado\)<\/td>/u);
  assert.doesNotMatch(html, /\*protegido|hay más/u);
  // Solo el indicador de la base anuncia el recorte, no el número de filas.
  const recortado = renderizarTablaCambios(validarCambiosExpediente(cuerpo([cambio()], true), "expediente:ct:1"), t);
  assert.match(recortado, /<p role="note">Se muestran los primeros 1 cambios; hay más\.<\/p>/u);
  assert.match(renderizarTablaCambios({ cambios: [] }, t), /No hay cambios de datos/u);
});

test("petición RRHH p.4: el apartado solo aparece en expedientes reales", () => {
  assert.equal(renderizarCambiosExpediente({ demostracion: true, expediente_ref: "expediente:ct:1", version: 3 }), "");
  const html = renderizarCambiosExpediente({ demostracion: false, expediente_ref: "expediente:ct:1", version: 3 });
  assert.match(html, /data-ct-cambios data-expediente-ref="expediente:ct:1" data-version="3"/u);
  assert.match(html, /<button type="button" class="boton-secundario" data-ct-cambios-consultar>Consultar cambios<\/button>/u);
  assert.match(html, /aria-live="polite"/u);
});
