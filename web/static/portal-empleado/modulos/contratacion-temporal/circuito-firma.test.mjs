import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import {
  crearClienteHTTPCircuitoFirma, crearGestorCircuitoFirma, renderizarCircuitoFirma,
  RUTA_CIRCUITO_FIRMA, validarCircuitoFirma,
} from "./circuito-firma.js";
import { fusionarEstadoFirmas } from "./circuito-firma-acciones.js";
import { crearTraductorCircuitoFirma, MENSAJES_CIRCUITO_FIRMA_ES, MENSAJES_CIRCUITO_FIRMA_EN_506 } from "./i18n-circuito-firma.js";

function paso(orden, total, extra = {}) {
  return {
    orden, cargo: `Cargo <${orden}>`, perfil_ref: "perfil:ct:jefatura_servicio_rrhh",
    accion: "firma", condicion: orden === 1 ? "borrador_generado" : "firma_paso_anterior",
    habilita: orden === total ? "remision_intervencion" : "siguiente_paso",
    devolucion: "vuelve_a_redaccion", sustitucion: "suplente_designado",
    estado: orden === 1 ? "pendiente_firma" : "en_espera",
    referencia: `vec.contratacion_temporal.circuito_firma:1:informe_definitivo.p${orden}`, ...extra,
  };
}

function circuito() {
  return {
    esquema: "vec.contratacion_temporal.circuito_firma.v1",
    catalogo_ref: "vec.contratacion_temporal.circuito_firma:1",
    huella_sha256: "a".repeat(64), ejemplo: true, firma_eficaz: false,
    documentos: [{ documento: "informe_definitivo", etiqueta: "Informe definitivo", pasos: [paso(1, 2), paso(2, 2)] }],
  };
}

function respuestaJSON(cuerpo, estado = 200) {
  return new Response(JSON.stringify(cuerpo), { status: estado, headers: { "Content-Type": "application/json; charset=utf-8" } });
}

test("valida el contrato exacto y rechaza desviaciones", () => {
  assert.ok(validarCircuitoFirma(circuito()));
  const casos = [
    (c) => { c.firma_eficaz = true; },
    (c) => { c.extra = 1; },
    (c) => { c.documentos[0].pasos[1].orden = 3; },
    (c) => { c.documentos[0].pasos[0].estado = "aprobado"; },
    (c) => { c.documentos[0].pasos[1].habilita = "siguiente_paso"; },
    (c) => { c.documentos[0].pasos[0].habilita = "cierre_circuito"; },
    (c) => { c.documentos = []; },
    (c) => { c.huella_sha256 = "x"; },
  ];
  for (const alterar of casos) {
    const copia = circuito();
    alterar(copia);
    assert.equal(validarCircuitoFirma(copia), null);
  }
});

test("el cliente pide la ruta de solo lectura y falla cerrado", async () => {
  let peticion;
  const cliente = crearClienteHTTPCircuitoFirma({
    fetchImpl: async (ruta, opciones) => { peticion = { ruta, opciones }; return respuestaJSON({ data: circuito() }); },
  });
  const resultado = await cliente.obtenerCircuito();
  assert.equal(peticion.ruta, RUTA_CIRCUITO_FIRMA);
  assert.equal(peticion.opciones.method, "GET");
  assert.equal(peticion.opciones.redirect, "error");
  assert.equal(resultado.documentos[0].pasos.length, 2);
  for (const fetchImpl of [
    async () => respuestaJSON({ error: { codigo: "servicio_no_disponible" } }, 503),
    async () => respuestaJSON({ data: { ...circuito(), firma_eficaz: true } }),
    async () => new Response("x".repeat(70 * 1024), { headers: { "Content-Type": "application/json" } }),
    async () => { throw new TypeError("red"); },
  ]) {
    assert.equal(await crearClienteHTTPCircuitoFirma({ fetchImpl }).obtenerCircuito(), null);
  }
});

test("el bloque muestra cada paso con su estado, escapa el catálogo y marca el ejemplo", () => {
  const t = crearTraductorCircuitoFirma();
  const html = renderizarCircuitoFirma(validarCircuitoFirma(circuito()), t);
  assert.match(html, /aria-labelledby="ct-circuito-firma-titulo"/u);
  assert.match(html, /Firma oficial en Firmadoc/u);
  assert.match(html, /Conexión pendiente/u);
  assert.match(html, /Sin constancia de envío ni firma oficial en VEC/u);
  assert.match(html, /Firmas de prueba con AutoFirma/u);
  assert.doesNotMatch(html, /Enviad[ao] a Firmadoc|Firma oficial completada/u);
  assert.doesNotMatch(html, /Circuito de ejemplo/u);
  assert.match(html, /Pendiente de firma por Cargo &lt;1&gt;/u);
  assert.match(html, /En espera del paso anterior/u);
  assert.match(html, /Permite remitir a Intervención/u);
  assert.equal((html.match(/aria-current="step"/gu) ?? []).length, 1);
  assert.doesNotMatch(html, /<1>/u);
});

test("si falla el catálogo, conserva visible la fase oficial sin afirmar estado de firma", () => {
  const t = crearTraductorCircuitoFirma();
  const html = renderizarCircuitoFirma(null, t);
  assert.match(html, /Firma oficial en Firmadoc/u);
  assert.match(html, /Conexión pendiente/u);
  assert.match(html, /No se puede consultar el circuito de firma/u);
  assert.doesNotMatch(html, /data-ct-firma-accion|Firmado por/u);
});

test("dos pasos CT118 firmados no convierten Firmadoc en envío o firma oficial", () => {
  const catalogo = validarCircuitoFirma(circuito());
  const estado = {
    huella_sha256: catalogo.huella_sha256, verificacion_disponible: true,
    documentos: [{ documento: "informe_definitivo", paso_pendiente: 0,
      pasos: [{ estado: "firmado" }, { estado: "firmado" }] }],
  };
  const unido = fusionarEstadoFirmas(catalogo, estado);
  const html = renderizarCircuitoFirma(unido, crearTraductorCircuitoFirma());
  assert.equal((html.match(/Firmado por/gu) ?? []).length, 2);
  assert.match(html, /Conexión pendiente/u);
  assert.match(html, /Sin constancia de envío ni firma oficial en VEC/u);
  assert.equal(fusionarEstadoFirmas(catalogo, { ...estado, huella_sha256: "b".repeat(64) }), null);
});

test("todas las claves de vocabulario tienen traducción", () => {
  const claves = Object.keys(MENSAJES_CIRCUITO_FIRMA_ES);
  for (const prefijo of ["accion_firma", "accion_visto_bueno", "habilita_siguiente_paso", "habilita_remision_intervencion",
    "habilita_envio_notificacion", "habilita_envio_comunicacion", "habilita_cierre_circuito",
    "devolucion_vuelve_a_redaccion", "devolucion_vuelve_paso_anterior", "sustitucion_suplente_designado",
    "sustitucion_no_admitida", "estado_pendiente_firma", "estado_en_espera", "estado_firmado", "estado_devuelto"]) {
    assert.ok(claves.includes(`circuito_firma_${prefijo}`), prefijo);
  }
});

test("la fase Firmadoc usa el idioma del portal", () => {
  for (const clave of Object.keys(MENSAJES_CIRCUITO_FIRMA_EN_506)) {
    assert.ok(Object.hasOwn(MENSAJES_CIRCUITO_FIRMA_ES, clave));
  }
  const html = renderizarCircuitoFirma(null, crearTraductorCircuitoFirma({}, "en-GB"));
  assert.match(html, /Official signing in Firmadoc/u);
  assert.match(html, /Connection pending/u);
  assert.match(html, /No recorded submission or official signature in VEC/u);
});

test("el gestor inserta el bloque tras la cabecera solo en el expediente vigente", async () => {
  const insertados = [];
  const cabecera = { insertAdjacentHTML: (posicion, html) => insertados.push({ posicion, html }) };
  let estado = { vista: "expediente", expediente: { expediente_ref: "exp:1" } };
  const raiz = { querySelector: (selector) => (selector === ".ct-exp-cabecera-expediente" ? cabecera : null) };
  let consultas = 0;
  const cliente = { obtenerCircuito: async () => { consultas += 1; return validarCircuitoFirma(circuito()); } };
  const gestor = crearGestorCircuitoFirma({ raiz, obtenerEstado: () => estado, cliente });
  gestor.montarSiProcede(estado);
  await new Promise((resolver) => setTimeout(resolver, 0));
  assert.equal(insertados.length, 1);
  assert.equal(insertados[0].posicion, "afterend");
  gestor.montarSiProcede({ vista: "cuadro" });
  const anterior = estado;
  gestor.montarSiProcede(anterior);
  estado = { vista: "expediente", expediente: { expediente_ref: "exp:2" } };
  await new Promise((resolver) => setTimeout(resolver, 0));
  assert.equal(insertados.length, 1, "un expediente ya sustituido no recibe el bloque");
  assert.equal(consultas, 1, "el catálogo se consulta una vez por montaje");
});

test("un fallo de consulta deja el estado pendiente visible en el expediente actual", async () => {
  const insertados = [];
  const cabecera = { insertAdjacentHTML: (_, html) => insertados.push(html) };
  const estado = { vista: "expediente", expediente: { expediente_ref: "exp:1" } };
  const raiz = { querySelector: (selector) => (selector === ".ct-exp-cabecera-expediente" ? cabecera : null) };
  const gestor = crearGestorCircuitoFirma({ raiz, obtenerEstado: () => estado,
    cliente: { obtenerCircuito: async () => null }, clienteFirma: { consultar: () => { throw new Error("no debe consultarse"); } } });
  gestor.montarSiProcede(estado);
  await new Promise((resolver) => setTimeout(resolver, 0));
  assert.equal(insertados.length, 1);
  assert.match(insertados[0], /Conexión pendiente/u);
  assert.match(insertados[0], /No se puede consultar el circuito de firma/u);
  gestor.retirar();
});

test("la ayuda explica el circuito de ejemplo y la falta de eficacia sin portafirmas", async () => {
  const ayuda = await readFile(new URL("../../portal-i18n-ayuda.js", import.meta.url), "utf8");
  assert.match(ayuda, /ayuda_contenido_421: "El «Circuito de firma»/u);
  assert.match(ayuda, /ayuda_contenido_422: ".*no tiene eficacia administrativa.*portafirmas corporativo/u);
});
