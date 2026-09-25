// Contrato Go → web: el fixture lo produce la prueba Go
// TestContratoDocumentoPropioV2 con el JSON real de resultadoAJSON (documento
// v2 con devolución, en corrección, reenviado y borrador nuevo). Aquí lo
// valida el cliente real cliente-borradores-http.js, sin adaptarlo.
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { crearClienteBorradoresDietasHTTP } from "./cliente-borradores-http.js";

const fixture = new URL("../../../../../internal/modules/dietas/adapters/httpinterno/testdata/documento_propio_v2.json", import.meta.url);
const casos = JSON.parse(await readFile(fixture, "utf8"));
const respuesta = (cuerpo) => new Response(JSON.stringify(cuerpo), {
  status: 200, headers: { "Content-Type": "application/json; charset=utf-8" } });

test("el fixture cubre los estados del documento propio v2", () => {
  assert.deepEqual(casos.map((caso) => caso.nombre), ["devuelta", "en_correccion", "reenviada", "borrador_nuevo"]);
});

for (const { nombre, resultado } of casos) {
  test(`el cliente real acepta el JSON de Go: ${nombre}`, async () => {
    const llamadas = [];
    const cliente = crearClienteBorradoresDietasHTTP({ fetchImpl: async (ruta, opciones) => {
      llamadas.push({ ruta, opciones }); return respuesta(resultado);
    } });
    const item = await cliente.obtener(resultado.comision.referencia);
    assert.equal(llamadas.length, 1);
    assert.equal(item.comision.estado, resultado.comision.estado);
    assert.equal(item.comision.version, resultado.comision.version);
    assert.equal(item.recibo.version, resultado.recibo.version);
    assert.deepEqual(item.comision.devolucion, resultado.comision.devolucion);
    if (resultado.comision.documento) {
      assert.equal(item.comision.centro_ref, "centro:uno");
      assert.equal(item.comision.unidad_ref, "unidad:uno");
      assert.deepEqual(item.comision.documento.lineas.map((linea) => linea.tipo),
        resultado.comision.documento.lineas.map((linea) => linea.tipo));
      assert.equal(item.comision.documento.total_orientativo_centimos, resultado.comision.documento.total_orientativo_centimos);
    }
  });
}

test("el recibo de Go tiene exactamente las cuatro claves del cliente", () => {
  for (const { nombre, resultado } of casos) {
    assert.deepEqual(Object.keys(resultado.recibo).sort(), ["referencia", "registrado_en", "repeticion", "version"], nombre);
  }
});

test("el cliente rechaza un recibo con la regla: Go no la emite", async () => {
  const base = casos.find((caso) => caso.nombre === "devuelta").resultado;
  for (const extra of [{ regla_ref: "provisional:regla:nacional-ordinaria:20260923" }, { regla_huella_sha256: "a".repeat(64) }]) {
    const alterado = structuredClone(base);
    Object.assign(alterado.recibo, extra);
    const cliente = crearClienteBorradoresDietasHTTP({ fetchImpl: async () => respuesta(alterado) });
    await assert.rejects(cliente.obtener(base.comision.referencia), /recibo de Dietas incompatible/u);
  }
});

test("una devolución con blanco de borde que recorta Go o el navegador se rechaza", async () => {
  const base = casos.find((caso) => caso.nombre === "devuelta").resultado;
  for (const borde of ["\ufeff", "\u0085", "\u00a0", " "]) {
    const alterado = structuredClone(base);
    alterado.comision.devolucion.motivo += borde;
    const cliente = crearClienteBorradoresDietasHTTP({ fetchImpl: async () => respuesta(alterado) });
    await assert.rejects(cliente.obtener(base.comision.referencia), `borde U+${borde.codePointAt(0).toString(16)}`);
  }
});
