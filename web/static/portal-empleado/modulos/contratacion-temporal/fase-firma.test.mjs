import assert from "node:assert/strict";
import test from "node:test";

import { cargarTextos } from "../../../comun/textos.js";
import { calcularFaseDocumento, cargarTextosFaseFirma, renderizarFaseFirma } from "./fase-firma.js?v=20261001-ct-a-i18n-v1";

const es = await cargarTextos("contratacion-temporal-firma", { idioma: "es", porDefecto: "es" });
const en = await cargarTextos("contratacion-temporal-firma", { idioma: "en", porDefecto: "es" });

function documento(estados, pendiente, fechas = []) {
  return {
    documento: "resolucion", etiqueta: "Resolución <x>", paso_pendiente: pendiente,
    pasos: estados.map((estado, i) => ({ orden: i + 1, cargo: `Cargo ${i + 1}`, estado, registrada_en: fechas[i] ?? "" })),
  };
}

test("el estado de la fase sale de la historia registrada", () => {
  const a = "2026-09-28T08:30:00Z";
  const b = "2026-09-29T09:15:00Z";
  assert.deepEqual({ ...calcularFaseDocumento(documento(["pendiente_firma", "en_espera"], 1), true) },
    { estado: "borrador", paso: documento(["pendiente_firma", "en_espera"], 1).pasos[0], total: 2, desde: "", custodiado: null });
  const pendiente = calcularFaseDocumento(documento(["firmado", "pendiente_firma"], 2, [a]), true);
  assert.equal(pendiente.estado, "pendiente_firma");
  assert.equal(pendiente.paso.orden, 2);
  assert.equal(pendiente.desde, a, "desde la firma del paso anterior");
  const firmado = calcularFaseDocumento(documento(["firmado", "firmado"], 0, [a, b]), true);
  assert.equal(firmado.estado, "firmado");
  assert.equal(firmado.paso, null);
  assert.equal(firmado.desde, b);
  const devuelto = calcularFaseDocumento(documento(["pendiente_firma", "devuelto"], 1, ["", b]), true);
  assert.equal(devuelto.estado, "devuelto");
  assert.equal(devuelto.paso.orden, 1, "debe actuar quien recibe la devolución");
  assert.equal(devuelto.desde, b);
});

test("sin historia disponible no se deduce ningún estado del catálogo", () => {
  const fase = calcularFaseDocumento({ documento: "resolucion", etiqueta: "R", pasos: [{ orden: 1, cargo: "C" }] }, false);
  assert.equal(fase.estado, "no_disponible");
  assert.equal(fase.paso.cargo, "C");
  const sinPendiente = calcularFaseDocumento(documento(["firmado"], undefined), true);
  assert.equal(sinPendiente.estado, "no_disponible");
});

test("el apartado dice documento, estado, quién firma y desde cuándo, escapado", () => {
  const real = { documentos: [documento(["firmado", "pendiente_firma", "en_espera"], 2, ["2026-09-28T08:30:00Z"])] };
  const html = renderizarFaseFirma({ catalogo: real, real, textos: es });
  assert.match(html, /<h4 id="ct-fase-firma-titulo">Fase de firma<\/h4>/u);
  assert.match(html, /Resolución &lt;x&gt;/u);
  assert.match(html, /data-ct-fase-firma-estado="pendiente_firma"/u);
  assert.match(html, />Pendiente de firma</u);
  assert.match(html, /Cargo 2 \(paso 2 de 3\)/u);
  assert.match(html, /28 sept 2026, 10:30/u, "fecha y hora de Madrid");
  assert.doesNotMatch(html, /ct-fase-firma-aviso/u);
});

test("sin estado real se avisa y se muestra el primer firmante", () => {
  const catalogo = { documentos: [{ documento: "resolucion", etiqueta: "Resolución", pasos: [{ orden: 1, cargo: "Jefatura" }, { orden: 2, cargo: "Diputado" }] }] };
  const html = renderizarFaseFirma({ catalogo, real: null, textos: es });
  assert.match(html, /role="status">Ahora no se puede saber/u);
  assert.match(html, />Estado no disponible</u);
  assert.match(html, /Jefatura \(paso 1 de 2\)/u);
  assert.match(html, /Fecha no disponible/u);
});

test("en inglés, sin texto en castellano, y sin catálogo no pinta nada", () => {
  const real = { documentos: [documento(["firmado", "firmado"], 0, ["2026-09-28T08:30:00Z", "2026-09-29T09:15:00Z"])] };
  const html = renderizarFaseFirma({ catalogo: real, real, textos: en, nombrar: (tipo, valor) => `${tipo}:${valor}` });
  assert.match(html, /Signing stage/u);
  assert.match(html, />Signed \(test\)</u);
  assert.match(html, /No one: test workflow complete/u);
  assert.match(html, /documento:Resolución &lt;x&gt;/u);
  assert.doesNotMatch(html, /Fase de firma|>Firmado<|Quién debe/u);
  assert.equal(renderizarFaseFirma({ catalogo: null, real: null, textos: es }), "");
  assert.equal(renderizarFaseFirma({ catalogo: real, real, textos: null }), "");
});

test("los dos idiomas tienen las mismas claves y un fallo de carga no rompe la ficha", async () => {
  const claves = (m, prefijo = "") => Object.entries(m).flatMap(([k, v]) => (typeof v === "object" ? claves(v, `${prefijo}${k}.`) : [`${prefijo}${k}`])).sort();
  assert.deepEqual(claves(en.mensajes), claves(es.mensajes));
  assert.equal(en.faltantes.length, 0);
  for (const estado of ["borrador", "pendiente_firma", "enviado_portafirmas", "firmado", "devuelto", "no_disponible"]) {
    assert.ok(es.traducir(`estado.${estado}`));
  }
  assert.equal(await cargarTextosFaseFirma(() => { throw new Error("sin catálogo"); }), null);
  assert.equal(await cargarTextosFaseFirma(async () => { throw new Error("sin catálogo"); }), null);
});

test("un documento de un solo paso no repite «paso 1 de 1»", () => {
  const real = { documentos: [documento(["pendiente_firma"], 1)] };
  const html = renderizarFaseFirma({ catalogo: real, real, textos: es });
  assert.match(html, /<span>Cargo 1<\/span>/u);
  assert.doesNotMatch(html, /paso 1 de 1/u);
});

test("una devolución dice quién la hizo y el motivo, y la firma completa se declara de prueba", () => {
  const devuelto = documento(["pendiente_firma", "devuelto"], 1, ["", "2026-09-29T07:20:00Z"]);
  devuelto.pasos[1].motivo_devolucion = "Falta <fecha>";
  const html = renderizarFaseFirma({ catalogo: { documentos: [devuelto] }, real: { documentos: [devuelto] }, textos: es });
  assert.match(html, /Devuelto por<\/span>\s*<span>Cargo 2: Falta &lt;fecha&gt;<\/span>/u);
  assert.doesNotMatch(html, /Quién debe firmar/u);
  const sinMotivo = documento(["devuelto"], 1, ["2026-09-29T07:20:00Z"]);
  assert.match(renderizarFaseFirma({ catalogo: { documentos: [sinMotivo] }, real: { documentos: [sinMotivo] }, textos: es }),
    /Devuelto por<\/span>\s*<span>Cargo 1<\/span>/u);
  const firmado = documento(["firmado"], 0, ["2026-09-29T07:20:00Z"]);
  const completo = renderizarFaseFirma({ catalogo: { documentos: [firmado] }, real: { documentos: [firmado] }, textos: es });
  assert.match(completo, />Firmado en prueba</u);
  assert.match(completo, /Nadie: circuito de prueba completo/u);
  const pendiente = documento(["pendiente_firma"], 1);
  assert.match(renderizarFaseFirma({ catalogo: { documentos: [pendiente] }, real: { documentos: [pendiente] }, textos: es }), /Sin empezar/u);
});

test("con permiso denegado la fase no añade su propio aviso", () => {
  const catalogo = { documentos: [documento(["pendiente_firma"], 1)] };
  assert.doesNotMatch(renderizarFaseFirma({ catalogo, real: null, textos: es, aviso: false }), /ct-fase-firma-aviso/u);
});

test("un documento con PDF firmado guardado ofrece descargarlo, con la terna que autoriza Documentos", () => {
  const c = { expediente_ref: `ref:${"e".repeat(64)}`, documento_ref: `ref:${"d".repeat(64)}`, version: 1, huella_sha256: "1".repeat(64) };
  const doc = documento(["firmado"], 0, ["2026-09-29T09:15:00Z"]);
  doc.pasos[0].documento_custodiado = c;
  const fase = calcularFaseDocumento(doc, true);
  assert.equal(fase.custodiado, doc.pasos[0]);
  const html = renderizarFaseFirma({ catalogo: { documentos: [doc] }, real: { documentos: [doc] }, textos: es });
  assert.match(html, /data-ct-descargar-firmado/u);
  assert.match(html, new RegExp(`data-ct-firmado-expediente="${c.expediente_ref}"`, "u"));
  assert.match(html, new RegExp(`data-ct-firmado-documento="${c.documento_ref}"`, "u"));
  assert.match(html, /data-ct-firmado-version="1"/u);
  assert.match(html, new RegExp(`data-ct-firmado-huella="${c.huella_sha256}"`, "u"));
  assert.match(html, />Descargar PDF firmado</u);
  assert.match(html, /aria-label="Descargar PDF firmado: Resolución &lt;x&gt;"/u, "el nombre accesible empieza por el texto visible");
  assert.match(html, /sin validez oficial/u);
  assert.match(renderizarFaseFirma({ catalogo: { documentos: [doc] }, real: { documentos: [doc] }, textos: en }), />Download signed PDF</u);
  // Sin PDF guardado, sin estado real o con dos pasos: lo que corresponde.
  const sin = documento(["firmado"], 0);
  assert.doesNotMatch(renderizarFaseFirma({ catalogo: { documentos: [sin] }, real: { documentos: [sin] }, textos: es }), /data-ct-descargar-firmado/u);
  assert.doesNotMatch(renderizarFaseFirma({ catalogo: { documentos: [doc] }, real: null, textos: es }), /data-ct-descargar-firmado/u);
  const dos = documento(["firmado", "pendiente_firma"], 2);
  dos.pasos[0].documento_custodiado = c;
  assert.match(renderizarFaseFirma({ catalogo: { documentos: [dos] }, real: { documentos: [dos] }, textos: es }), /Firma del paso 1 de 2\./u);
});
