import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { crearTraductorCronosC4, MENSAJES_CRONOS_C4_ES } from "./i18n-c4.js";
import { montarCalendarioCivilCronos, renderizarCalendarioCivilCronos } from "./vista-calendario.js";

test("año civil completo, bisiesto y sin festivos ni jornada inferidos", () => {
  const normal = renderizarCalendarioCivilCronos({ anio: 2025, fechaSeleccionada: "2025-05-16" });
  const bisiesto = renderizarCalendarioCivilCronos({ anio: 2024, fechaSeleccionada: "2024-02-29" });
  assert.equal([...normal.matchAll(/data-cronos-cal-dia="\d{4}-\d{2}-\d{2}"/gu)].length, 365);
  assert.equal([...bisiesto.matchAll(/data-cronos-cal-dia="\d{4}-\d{2}-\d{2}"/gu)].length, 366);
  assert.match(bisiesto, /data-cronos-cal-dia="2024-02-29"[^>]*aria-pressed="true"/u);
  assert.match(normal, /data-estado="no_configurado"/u);
  assert.match(normal, /La fecha elegida no determina si es hábil o laborable/u);
  assert.match(normal, /Sábado y domingo: clasificación civil, sin efecto laboral inferido/u);
  assert.doesNotMatch(normal, /data-festivo|data-laborable|data-jornada-minutos|data-centro-abierto/u);
});

test("fecha y año inválidos se rechazan y los textos C4 siguen el catálogo", () => {
  for (const fecha of ["2025-02-29", "2025-13-01", "2025-00-01", "2025-01-32", "<script>"]) {
    assert.throws(() => renderizarCalendarioCivilCronos({ fechaSeleccionada: fecha }), RangeError);
  }
  assert.throws(() => renderizarCalendarioCivilCronos({ anio: 2024, fechaSeleccionada: "2025-01-01" }), RangeError);
  assert.throws(() => renderizarCalendarioCivilCronos({ fechaSeleccionada: "2101-01-01" }), RangeError);
  assert.throws(() => crearTraductorCronosC4({ calendario_estado: "Pendiente" }), /incompleto/u);
  const mensajes = { ...MENSAJES_CRONOS_C4_ES, calendario_estado: '<img src=x onerror="x">' };
  const html = renderizarCalendarioCivilCronos({ fechaSeleccionada: "2025-01-01", mensajes });
  assert.doesNotMatch(html, /<img/u);
  assert.match(html, /&lt;img src=x onerror=&quot;x&quot;&gt;/u);
});

test("navegación anual, selección por ratón y teclado; desmontaje retira listeners", () => {
  const listeners = new Map();
  const enfoques = [];
  const contenedor = {
    innerHTML: "", addEventListener(tipo, fn) { listeners.set(tipo, fn); },
    removeEventListener(tipo) { listeners.delete(tipo); },
    remove() { this.retirado = true; },
    querySelector(selector) { return { focus() { enfoques.push(selector); } }; },
  };
  const raiz = { ownerDocument: { createElement: () => contenedor }, append() {} };
  const anuncios = [];
  let desmontarRegistrado;
  const vista = montarCalendarioCivilCronos({ raiz, fechaSeleccionada: "2024-02-29", anunciar: (mensaje) => anuncios.push(mensaje), registrarDesmontar(fn) { desmontarRegistrado = fn; } });
  assert.match(contenedor.innerHTML, /data-cronos-cal-dia="2024-02-29"[^>]*aria-pressed="true"/u);
  const objetivo = (selector, dataset) => ({ target: { closest: (pedido) => pedido === selector ? { dataset } : null } });
  listeners.get("click")(objetivo("[data-cronos-cal-anio]", { cronosCalAnio: "1" }));
  assert.match(contenedor.innerHTML, /data-cronos-cal-dia="2025-02-28"[^>]*aria-pressed="true"/u);
  listeners.get("click")(objetivo("[data-cronos-cal-dia]", { cronosCalDia: "2025-12-31" }));
  assert.match(contenedor.innerHTML, /data-cronos-cal-dia="2025-12-31"[^>]*aria-pressed="true"/u);
  let impedido = false;
  listeners.get("keydown")({ key: "ArrowRight", target: { closest: (pedido) => pedido === "[data-cronos-cal-dia]" ? { dataset: { cronosCalDia: "2025-12-31" } } : null }, preventDefault() { impedido = true; } });
  assert.equal(impedido, true);
  assert.match(contenedor.innerHTML, /data-cronos-cal-dia="2026-01-01"[^>]*aria-pressed="true"/u);
  assert.ok(enfoques.includes('[data-cronos-cal-dia="2026-01-01"]'));
  assert.ok(anuncios.length >= 3);
  desmontarRegistrado();
  vista.desmontar();
  assert.equal(listeners.size, 0);
  assert.equal(contenedor.retirado, true);
});

test("sin red, almacenamiento, colores locales ni desbordamiento de página impuesto", async () => {
  const fuente = await readFile(new URL("./vista-calendario.js", import.meta.url), "utf8");
  const css = await readFile(new URL("./vista-calendario.css", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /fetch\(|XMLHttpRequest|localStorage|sessionStorage|indexedDB|document\.cookie|navigator\.geolocation/iu);
  assert.doesNotMatch(css, /#[0-9a-fA-F]{3,8}\b/u);
  assert.match(css, /max-height: min\(62vh, 620px\);[\s\S]*overflow: auto/u);
  assert.match(css, /:focus-visible/u);
});
