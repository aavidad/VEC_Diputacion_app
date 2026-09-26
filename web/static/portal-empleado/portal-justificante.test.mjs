import assert from "node:assert/strict";
import test from "node:test";
import { actorTraducido, claveRecuperacionTraducida, esReferenciaInterna, instalarCopiaJustificantes, justificanteTraducido, referenciaCopiableTraducida, renderizarJustificante } from "./portal-justificante.js";
import { traducirReferencia } from "./portal-referencias-i18n.js";

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

test("la clave de recuperación se ofrece para copiar sin mostrarla", () => {
  const t = (clave) => ({ clave_recuperacion_preparada: "Clave de recuperación preparada", clave_recuperacion_copiar: "Copiar clave", clave_recuperacion_copiada: "Clave copiada" })[clave];
  const html = claveRecuperacionTraducida("11111111-1111-4111-8111-111111111111", escapar, t);
  assert.match(html, /Clave de recuperación preparada<\/span> <button[^>]+data-copiar-justificante="11111111-1111-4111-8111-111111111111"[^>]*>Copiar clave<\/button>/u);
  assert.doesNotMatch(html.replace(/data-copiar-justificante="[^"]*"/u, ""), /11111111/u);
});

test("una referencia de fila se copia sin rótulo y con nombre accesible propio", () => {
  const html = referenciaCopiableTraducida("cnv_0123456789abcdef", escapar, traducirReferencia, "Copiar la referencia de la convocatoria 1");
  assert.match(html, /^<button type="button" class="boton-terciario boton-copiar-justificante" data-copiar-justificante="cnv_0123456789abcdef"/u);
  assert.match(html, /aria-label="Copiar la referencia de la convocatoria 1">Copiar referencia<\/button>$/u);
  assert.doesNotMatch(html, /justificante-registrado/u);
});

test("las referencias de identidad se nombran por su papel y los nombres escritos se respetan", () => {
  for (const ref of ["per_rrhh", "recibo:1", "act_0123456789abcdef", "0f4b8f2e-8c1d-4f5e-9a3b-2c7d6e5f4a3b", "a".repeat(64)]) {
    assert.equal(esReferenciaInterna(ref), true, ref);
  }
  for (const texto of ["Ana Ruiz", "RRHH", "Jefatura de servicio", "", null]) assert.equal(esReferenciaInterna(texto), false, String(texto));
  const actor = actorTraducido("per_rrhh", escapar, traducirReferencia);
  assert.match(actor, /^Personal de RRHH <button/u);
  assert.match(actor, /data-copiar-justificante="per_rrhh"/u);
  assert.equal(actorTraducido("Ana <Ruiz>", escapar, traducirReferencia), "Ana &lt;Ruiz>");
  assert.equal(actorTraducido("", escapar, traducirReferencia), "—");
  assert.throws(() => traducirReferencia("clave_inexistente"), /desconocida/u);
});
