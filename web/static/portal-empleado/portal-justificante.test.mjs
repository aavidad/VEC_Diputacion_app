import assert from "node:assert/strict";
import test from "node:test";
import { instalarCopiaJustificantes, justificanteTraducido, renderizarJustificante } from "./portal-justificante.js";

const escapar = (valor) => String(valor).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll('"', "&quot;");

test("el justificante no muestra la referencia opaca y la ofrece para copiar", () => {
  const html = renderizarJustificante('recibo:ct-alta:"x"', { escapar, etiqueta: "Justificante registrado", copiar: "Copiar referencia", copiado: "Referencia copiada" });
  assert.match(html, /^<span class="justificante-registrado">Justificante registrado<\/span> <button type="button"/u);
  assert.match(html, /data-copiar-justificante="recibo:ct-alta:&quot;x&quot;"/u);
  assert.doesNotMatch(html.replace(/data-copiar-justificante="[^"]*"/u, ""), /recibo:/u);
  assert.equal(renderizarJustificante("", { escapar, etiqueta: "a", copiar: "b", copiado: "c" }), "—");
  const t = (clave) => ({ justificante_registrado: "J", justificante_copiar: "C", justificante_copiado: "OK" })[clave];
  assert.match(justificanteTraducido("recibo:1", escapar, t), />J<\/span>.*>C<\/button>/u);
});

test("un único oyente copia la referencia del botón y confirma con su texto", async () => {
  const oyentes = [];
  const documento = { addEventListener: (tipo, fn) => oyentes.push([tipo, fn]) };
  const copiados = [];
  const portapapeles = { writeText: async (texto) => { copiados.push(texto); } };
  assert.equal(instalarCopiaJustificantes(documento, portapapeles), true);
  assert.equal(instalarCopiaJustificantes(documento, portapapeles), false);
  assert.equal(oyentes.length, 1);
  const boton = { dataset: { copiarJustificante: "recibo:1", textoCopiado: "Referencia copiada" }, textContent: "Copiar referencia" };
  oyentes[0][1]({ target: { closest: () => boton } });
  await new Promise((resolver) => setImmediate(resolver));
  assert.deepEqual(copiados, ["recibo:1"]);
  assert.equal(boton.textContent, "Referencia copiada");
  oyentes[0][1]({ target: { closest: () => null } });
  assert.equal(copiados.length, 1);
});
