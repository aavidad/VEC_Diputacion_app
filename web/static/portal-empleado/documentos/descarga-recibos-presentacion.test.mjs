import assert from "node:assert/strict";
import test from "node:test";
import { crearDescargadorRecibosPresentacion } from "./descarga-recibos-presentacion.js";

test("adapta el descriptor final de Dietas y descarga un PDF institucional", async () => {
  let blob;
  let nombre;
  const cuerpo = { append() {} };
  const entorno = {
    URL: { createObjectURL(valor) { blob = valor; return "blob:demo"; }, revokeObjectURL() {} },
    document: { body: cuerpo, createElement() { return { hidden: false, click() {}, remove() {}, set download(valor) { nombre = valor; }, set href(_valor) {} }; } },
    setTimeout(funcion) { funcion(); },
  };
  const descargar = crearDescargadorRecibosPresentacion(entorno);
  const resultado = await descargar({
    referencia: "DEMO-REC-DIE-0073-06",
    titulo: "Recibo de actuación en Dietas",
    subtitulo: "Portal del Empleado · Diputación de Granada",
    marca: "DOCUMENTO DEMO · SIN EFECTOS ADMINISTRATIVOS",
    filas: [
      { etiqueta: "Actuación", valor: "Pagada" },
      { etiqueta: "Ruta", valor: "Granada -> Guadix -> Granada" },
      { etiqueta: "Importe", valor: "61,88 EUR" },
    ],
    comprobacion: { qr_contenido: "http://127.0.0.2:8081/verificar/?ref=DEMO-REC-DIE-0073-06&presentacion=rrhh" },
  });
  assert.equal(resultado.formato, "application/pdf");
  assert.equal(nombre, "recibo-demo-rec-die-0073-06.pdf");
  assert.equal(blob.type, "application/pdf");
  const pdf = Buffer.from(await blob.arrayBuffer()).toString("latin1");
  assert.match(pdf, /Diputacion de Granada/);
  assert.match(pdf, /DEMO-REC-DIE-0073-06/);
  assert.ok(blob.size > 10_000);
});
