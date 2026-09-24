import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { montarVistaCorreccionesCronos, renderizarCorreccionesCronos } from "./vista-correcciones.js";

test("C5 prepara fecha, hora, original, motivo y revisión sin simular un registro", () => {
  const html = renderizarCorreccionesCronos();
  for (const nombre of ["fecha", "hora", "tipo", "original", "motivo"]) assert.match(html, new RegExp(`name="${nombre}"`, "u"));
  assert.match(html, /Preparación local · sin registrar/u);
  assert.match(html, /lo escrito no se guarda/u);
  assert.match(html, /El marcaje original se conservará/u);
  assert.match(html, /Enviar para validación<\/button>/u);
  assert.match(html, /disabled aria-disabled="true" title="Acción deshabilitada/u);
  assert.match(html, /name="original"><option value="">No hay marcaje original que relacionar<\/option><\/select>/u);
  assert.doesNotMatch(html, /recibo:|Registrada correctamente/u);
});

test("solo muestra marcajes suministrados desde una proyección propia disponible", () => {
  const fichajes = [{ id: "fichaje:propio-001", instante: "2026-09-24T08:00:00Z", tipo_clave: "entrada" }];
  const html = renderizarCorreccionesCronos({ estado: "disponible", fichajes });
  assert.match(html, /value="fichaje:propio-001"[^>]*>24\/9\/26, 10:00 · Entrada/u);
  assert.doesNotMatch(renderizarCorreccionesCronos({ estado: "denegado", fichajes }), /fichaje:propio-001/u);
  assert.match(renderizarCorreccionesCronos({ estado: "error" }), /No se pudieron consultar los movimientos propios/u);
  assert.match(renderizarCorreccionesCronos({ estado: "vacio" }), /No hay movimientos propios/u);
  assert.throws(() => renderizarCorreccionesCronos({ estado: "disponible", fichajes: [{ ...fichajes[0], tipo_clave: "otro" }] }), /movimiento C5/u);
  assert.throws(() => renderizarCorreccionesCronos({ estado: "inventado" }), /estado C5/u);
});

test("el texto externo se escapa y el resumen usa textContent", () => {
  const html = renderizarCorreccionesCronos({ mensajes: { c5_titulo: '<img src=x onerror="x()">' } });
  assert.match(html, /&lt;img src=x onerror=&quot;x\(\)&quot;&gt;/u);
  assert.doesNotMatch(html, /<img/u);

  const escuchas = new Map();
  const campos = {
    fecha: { value: "2026-09-24" }, hora: { value: "10:15" }, tipo: { value: "entrada" },
    original: { value: "", selectedOptions: [{ textContent: "No hay marcaje original que relacionar" }] },
    motivo: { value: "<img src=x onerror=x()>" },
  };
  const resumen = Object.fromEntries(["fecha", "hora", "tipo", "original", "motivo"].map((clave) => [clave, { textContent: "" }]));
  const form = {
    querySelector: (selector) => campos[selector.match(/\[name="([^"]+)"\]/u)?.[1]],
    addEventListener: (tipo, fn) => escuchas.set(tipo, fn),
    removeEventListener: (tipo) => escuchas.delete(tipo),
    reset() { this.limpio = true; },
  };
  const contenedor = {
    dataset: {}, innerHTML: "",
    querySelector(selector) { return selector === "[data-cronos-c5-formulario]" ? form : resumen[selector.match(/data-cronos-c5-resumen="([^"]+)"/u)?.[1]]; },
    remove() { this.retirado = true; },
  };
  let desmontarRegistrado;
  const vista = montarVistaCorreccionesCronos({
    raiz: { ownerDocument: { createElement: () => contenedor }, append() {} },
    registrarDesmontar: (desmontar) => { desmontarRegistrado = desmontar; },
  });
  escuchas.get("input")();
  assert.equal(resumen.fecha.textContent, "24/09/2026");
  assert.equal(resumen.tipo.textContent, "Entrada");
  assert.equal(resumen.hora.textContent, "10:15");
  assert.equal(resumen.motivo.textContent, "<img src=x onerror=x()>");
  let impedido = false;
  escuchas.get("submit")({ target: form, preventDefault() { impedido = true; } });
  assert.equal(impedido, true);
  desmontarRegistrado();
  vista.desmontar();
  assert.equal(form.limpio, true);
  assert.equal(contenedor.innerHTML, "");
  assert.equal(contenedor.retirado, true);
  assert.equal(escuchas.size, 0);
});

test("C5 consume el tema común y conserva foco y disposición móvil", async () => {
  const css = await readFile(new URL("./vista-correcciones.css", import.meta.url), "utf8");
  assert.match(css, /\.cronos-c5-rejilla\s*\{[^}]*grid-template-columns/u);
  assert.match(css, /@media \(max-width: 480px\)/u);
  assert.match(css, /:focus-visible/u);
  assert.match(css, /var\(--portal-superficie/u);
  assert.doesNotMatch(css, /#[0-9a-fA-F]{3,8}\b/u);
});
