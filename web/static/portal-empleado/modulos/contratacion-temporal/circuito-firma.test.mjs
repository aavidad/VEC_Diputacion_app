import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import {
  crearClienteHTTPCircuitoFirma, crearGestorCircuitoFirma, renderizarCircuitoFirma,
  RUTA_CIRCUITO_FIRMA, validarCircuitoFirma,
} from "./circuito-firma.js";
import { crearTraductorCircuitoFirma, MENSAJES_CIRCUITO_FIRMA_ES } from "./i18n-circuito-firma.js";

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
  assert.doesNotMatch(html, /Circuito de ejemplo/u);
  assert.match(html, /Pendiente de firma por Cargo &lt;1&gt;/u);
  assert.match(html, /En espera del paso anterior/u);
  assert.match(html, /Permite remitir a Intervención/u);
  assert.equal((html.match(/aria-current="step"/gu) ?? []).length, 1);
  assert.doesNotMatch(html, /<1>/u);
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

test("la ayuda explica el circuito de ejemplo y la falta de eficacia sin portafirmas", async () => {
  const ayuda = await readFile(new URL("../../portal-i18n-ayuda.js", import.meta.url), "utf8");
  assert.match(ayuda, /ayuda_contenido_421: "El «Circuito de firma»/u);
  assert.match(ayuda, /ayuda_contenido_422: ".*no tiene eficacia administrativa.*portafirmas corporativo/u);
});
