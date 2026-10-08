import assert from "node:assert/strict";
import { createHash, webcrypto } from "node:crypto";
import test from "node:test";
import { crearClienteBorradoresPublicados, RUTA_BORRADORES_DISPONIBLES,
  RUTA_BORRADORES_PUBLICADOS, validarBorradoresDisponibles } from "./cliente-http-borradores-publicados.js";
import { montarBorradoresPublicados, renderizarBorradoresPublicados } from "./vista-borradores-publicados.js?v=20261008-alta-circular-v3";

const contexto = Object.freeze({ expediente_ref: "expediente:ct:1", version_observada: 8 });
const catalogo = Object.freeze({ esquema: "vec.contratacion-temporal.borradores-disponibles.v1",
  catalogo_ref: "catalogo:ct:publicado:1", catalogo_huella_sha256: "a".repeat(64),
  procedencia_ref: "recibo:12345678-1234-1234-1234-123456789abc",
  tipos: [{ clave: "acta_ampliada", etiqueta: "Acta ampliada", formatos: ["pdf", "docx"] }] });
const bytesPDF = new TextEncoder().encode("%PDF-1.7\nBorrador sintético\n");
const huellaPDF = createHash("sha256").update(bytesPDF).digest("hex");

test("catálogo publicado valida claves y formatos sin aceptar tipos libres", () => {
  assert.equal(validarBorradoresDisponibles(catalogo).tipos[0].clave, "acta_ampliada");
  assert.throws(() => validarBorradoresDisponibles({ ...catalogo, tipos: [{ ...catalogo.tipos[0], clave: "../../otro" }] }));
  assert.throws(() => validarBorradoresDisponibles({ ...catalogo, tipos: [{ ...catalogo.tipos[0], formatos: ["pdf", "html"] }] }));
  assert.throws(() => validarBorradoresDisponibles({ ...catalogo, tipos: [...catalogo.tipos, ...catalogo.tipos] }));
  assert.throws(() => validarBorradoresDisponibles({ ...catalogo, actor_ref: "ajeno" }));
  assert.throws(() => validarBorradoresDisponibles({ ...catalogo, procedencia_ref: "" }));
  assert.throws(() => validarBorradoresDisponibles({ ...catalogo, tipos: [{ ...catalogo.tipos[0], etiqueta: "Acta\nextra" }] }));
});

test("consulta tipos y conserva los bytes originales solo con catálogo y huellas coincidentes", async () => {
  const llamadas = [];
  const cliente = crearClienteBorradoresPublicados({ cryptoImpl: webcrypto, fetchImpl: async (ruta, opciones) => {
    llamadas.push([ruta, opciones]);
    if (ruta === RUTA_BORRADORES_DISPONIBLES) return new Response(JSON.stringify(catalogo), { status: 200,
      headers: { "content-type": "application/json" } });
    return new Response(bytesPDF, { status: 200, headers: {
      "content-type": "application/pdf", "content-disposition": 'attachment; filename="acta_ampliada-borrador.pdf"',
      "x-vec-catalogo-ref": catalogo.catalogo_ref,
      "x-vec-catalogo-huella-sha256": catalogo.catalogo_huella_sha256,
      "x-vec-plantilla-procedencia-ref": catalogo.procedencia_ref,
      "x-vec-documento-sha256": huellaPDF,
    } });
  } });
  const disponibles = await cliente.consultarDisponibles(contexto);
  const archivo = await cliente.descargar(contexto, disponibles, "acta_ampliada", "pdf");
  assert.deepEqual(archivo.bytes, bytesPDF);
  assert.equal(archivo.huella_sha256, huellaPDF);
  assert.equal(archivo.procedencia_ref, catalogo.procedencia_ref);
  assert.deepEqual(llamadas.map(([ruta]) => ruta), [RUTA_BORRADORES_DISPONIBLES, RUTA_BORRADORES_PUBLICADOS]);
  assert.deepEqual(JSON.parse(llamadas[1][1].body), { ...contexto, tipo: "acta_ampliada", formato: "pdf" });
  assert.equal(llamadas[1][1].headers.Accept, "application/pdf");
  assert.equal(llamadas[1][1].credentials, "same-origin");
  assert.equal(llamadas[1][1].cache, "no-store");
  assert.equal(llamadas[1][0].includes("?"), false);
  await assert.rejects(cliente.descargar(contexto, disponibles, "tipo_ajeno", "pdf"), /tipo_no_publicado/u);
  assert.equal(llamadas.length, 2);
});

test("rechaza documento alterado, catálogo ajeno y ausencia de permiso", async () => {
  const cliente = crearClienteBorradoresPublicados({ cryptoImpl: webcrypto, fetchImpl: async () => new Response(bytesPDF,
    { status: 200, headers: { "content-type": "application/pdf", "content-disposition": 'attachment; filename="acta_ampliada-borrador.pdf"',
      "x-vec-catalogo-ref": catalogo.catalogo_ref, "x-vec-catalogo-huella-sha256": catalogo.catalogo_huella_sha256,
      "x-vec-plantilla-procedencia-ref": catalogo.procedencia_ref,
      "x-vec-documento-sha256": "b".repeat(64) } }) });
  await assert.rejects(cliente.descargar(contexto, catalogo, "acta_ampliada", "pdf"), /huella_no_coincide/u);
  const denegado = crearClienteBorradoresPublicados({ fetchImpl: async () => new Response("", { status: 403 }) });
  await assert.rejects(denegado.consultarDisponibles(contexto), (error) => error.estado === 403);
  const cancelado = new AbortController(); cancelado.abort();
  const sinRed = crearClienteBorradoresPublicados({ fetchImpl: async () => assert.fail("red inesperada") });
  await assert.rejects(sinRed.consultarDisponibles(contexto, { signal: cancelado.signal }), /cancelado/u);
});

test("PDF y DOCX exigen el mismo recibo de publicación y rechazan otro origen", async () => {
  const bytesDOCX = new Uint8Array([0x50, 0x4b, 0x03, 0x04, 0x01]);
  const peticiones = [];
  const cliente = crearClienteBorradoresPublicados({ cryptoImpl: webcrypto, fetchImpl: async (ruta, opciones) => {
    peticiones.push([ruta, JSON.parse(opciones.body)]);
    if (ruta === RUTA_BORRADORES_DISPONIBLES) return respuestaJSON(catalogo);
    const docx = opciones.headers.Accept.includes("officedocument");
    const bytes = docx ? bytesDOCX : bytesPDF;
    return new Response(bytes, { status: 200, headers: {
      "content-type": opciones.headers.Accept,
      "content-disposition": `attachment; filename="acta_ampliada-borrador.${docx ? "docx" : "pdf"}"`,
      "x-vec-catalogo-ref": catalogo.catalogo_ref,
      "x-vec-catalogo-huella-sha256": catalogo.catalogo_huella_sha256,
      "x-vec-plantilla-procedencia-ref": catalogo.procedencia_ref,
      "x-vec-documento-sha256": createHash("sha256").update(bytes).digest("hex"),
    } });
  } });
  const disponibles = await cliente.consultarDisponibles(contexto);
  const pdf = await cliente.descargar(contexto, disponibles, "acta_ampliada", "pdf");
  const docx = await cliente.descargar(contexto, disponibles, "acta_ampliada", "docx");
  assert.equal(pdf.procedencia_ref, docx.procedencia_ref);
  assert.equal(peticiones.length, 3);
  assert.deepEqual(peticiones[1][1], { ...contexto, tipo: "acta_ampliada", formato: "pdf" });
  assert.deepEqual(peticiones[2][1], { ...contexto, tipo: "acta_ampliada", formato: "docx" });

  const alterado = crearClienteBorradoresPublicados({ cryptoImpl: webcrypto, fetchImpl: async () =>
    new Response(bytesPDF, { status: 200, headers: { "content-type": "application/pdf",
      "content-disposition": 'attachment; filename="acta_ampliada-borrador.pdf"',
      "x-vec-catalogo-ref": catalogo.catalogo_ref,
      "x-vec-catalogo-huella-sha256": catalogo.catalogo_huella_sha256,
      "x-vec-plantilla-procedencia-ref": "recibo:ffffffff-ffff-ffff-ffff-ffffffffffff",
      "x-vec-documento-sha256": huellaPDF } }) });
  await assert.rejects(alterado.descargar(contexto, disponibles, "acta_ampliada", "pdf"), /documento_incompatible/u);
});

const respuestaJSON = (valor) => new Response(JSON.stringify(valor), { status: 200,
  headers: { "content-type": "application/json" } });

test("el panel solo ofrece tipos recibidos y ayuda tras ?", () => {
  const html = renderizarBorradoresPublicados({ estado: "lista", catalogo: validarBorradoresDisponibles(catalogo) });
  assert.match(html, /Acta ampliada/u);
  assert.match(html, /data-bp-descargar="acta_ampliada" data-bp-formato="pdf"/u);
  assert.match(html, /data-bp-descargar="acta_ampliada" data-bp-formato="docx"/u);
  assert.match(html, /id="ct-bp-ayuda"[^>]*hidden/u);
  assert.doesNotMatch(html, /Tipos disponibles para este expediente y su versión actual|<small>acta_ampliada<\/small>|catalogo:ct:publicado:1/u);
  assert.match(html, /Recibo de publicación: recibo:12345678/u);
  assert.doesNotMatch(renderizarBorradoresPublicados({ estado: "denegado" }), /data-bp-descargar/u);
});

test("el panel descarga solo la opción del catálogo y muestra huella sin recibo inventado", async () => {
  const eventos = new Map(), clics = [], revocadas = [], solicitudes = [];
  const raiz = { innerHTML: "", hidden: false, contains: () => true,
    addEventListener: (tipo, fn) => eventos.set(tipo, fn), removeEventListener: (tipo) => eventos.delete(tipo),
    replaceChildren() { this.innerHTML = ""; } };
  const cliente = { consultarDisponibles: async () => validarBorradoresDisponibles(catalogo),
    descargar: async (_contexto, _catalogo, tipo, formato) => { solicitudes.push([tipo, formato]);
      return { bytes: bytesPDF, mime: "application/pdf", nombre: "acta_ampliada-borrador.pdf",
        huella_sha256: huellaPDF, catalogo_ref: catalogo.catalogo_ref,
        catalogo_huella_sha256: catalogo.catalogo_huella_sha256,
        procedencia_ref: catalogo.procedencia_ref }; } };
  const entornoDescarga = { URL: { createObjectURL: () => "blob:prueba", revokeObjectURL: (url) => revocadas.push(url) },
    document: { body: { append() {} }, createElement: () => ({ hidden: false, click() { clics.push(this.download); }, remove() {} }) } };
  const panel = montarBorradoresPublicados({ raiz, contexto, cliente, entornoDescarga });
  await new Promise((resolver) => setImmediate(resolver));
  assert.match(raiz.innerHTML, /Acta ampliada/u);
  const boton = { dataset: { bpDescargar: "acta_ampliada", bpFormato: "pdf" },
    hasAttribute: (clave) => clave === "data-bp-descargar" };
  eventos.get("click")({ target: { closest: () => boton } });
  await new Promise((resolver) => setImmediate(resolver));
  assert.deepEqual(solicitudes, [["acta_ampliada", "pdf"]]);
  assert.deepEqual(clics, ["acta_ampliada-borrador.pdf"]);
  assert.match(raiz.innerHTML, new RegExp(huellaPDF));
  assert.match(raiz.innerHTML, /Recibo de publicación: recibo:12345678/u);
  panel.desmontar();
  assert.equal(eventos.size, 0);
  assert.deepEqual(revocadas, ["blob:prueba"]);
});
