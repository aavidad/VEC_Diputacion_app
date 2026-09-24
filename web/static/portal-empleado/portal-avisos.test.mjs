import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { obtenerDatosPresentacion } from "./datos-presentacion.js";
import { instalarDeeplinkAvisosBorradores } from "./portal-borradores-ui.js";

const [portal, estilos, html] = await Promise.all([
  readFile(new URL("portal.js", import.meta.url), "utf8"),
  readFile(new URL("portal-flujos.css", import.meta.url), "utf8"),
  readFile(new URL("index.html", import.meta.url), "utf8"),
]);

test("el aviso R5 apunta al borrador DEMO exacto mediante una vista interna", () => {
  const [aviso] = obtenerDatosPresentacion().avisos;
  assert.deepEqual(aviso, {
    texto: "Informe jurídico pendiente en DEMO-BORRADOR-001.",
    destino: {
      vista: "elaboracion",
      etiqueta: "Borradores de convocatorias",
      estado: "disponible",
      referencia: "DEMO-BORRADOR-001",
    },
  });
});

test("el botón nativo conserva teclado, foco y navegación opaca sin construir URL", async () => {
  let escuchar;
  let enfocado = false;
  const orden = [];
  const dialogo = { open: false, showModal() { this.open = true; }, close() { this.open = false; } };
  const titulo = { textContent: "" };
  const contenido = { innerHTML: "", querySelector: () => ({ focus: () => { enfocado = true; } }) };
  const elementos = { "dialogo-detalle": dialogo, "titulo-dialogo": titulo, "contenido-dialogo": contenido };
  const documento = {
    addEventListener: (_tipo, manejador) => { escuchar = manejador; },
    removeEventListener: () => {},
  };
  instalarDeeplinkAvisosBorradores({
    documento,
    escaparHTML: String,
    porId: (id) => elementos[id],
    obtenerAvisos: () => obtenerDatosPresentacion().avisos,
    disponible: () => true,
    navegar: (vista, opciones) => orden.push([vista, opciones]),
    anunciar: (mensaje) => orden.push(mensaje),
  });
  escuchar({ target: { closest: (selector) => selector.includes("boton-avisos") ? {} : null },
    preventDefault() {}, stopImmediatePropagation() {} });
  await Promise.resolve();
  assert.equal(dialogo.open, true);
  assert.equal(titulo.textContent, "Avisos");
  assert.match(contenido.innerHTML, /data-aviso-borrador-ref="DEMO-BORRADOR-001"/);
  assert.equal(enfocado, true);
  escuchar({ target: { closest: (selector) => selector.includes("data-aviso-borrador-ref")
    ? { dataset: { avisoBorradorRef: "DEMO-BORRADOR-001" } } : null },
  preventDefault() {}, stopImmediatePropagation() {} });
  assert.deepEqual(orden, [["elaboracion", { referencia: "DEMO-BORRADOR-001" }], "Aviso: Borradores de convocatorias"]);
  assert.equal(dialogo.open, false);
  assert.doesNotMatch(portal, /location\.href\s*=.*aviso|window\.open\(.*aviso/);
});

test("el deeplink permanece cerrado sin autorización positiva de Elaboración", () => {
  let escuchar;
  let impedido = false;
  const dialogo = { open: false, showModal() { this.open = true; } };
  const contenido = { innerHTML: "", querySelector: () => null };
  instalarDeeplinkAvisosBorradores({
    documento: {
      addEventListener: (_tipo, manejador) => { escuchar = manejador; },
      removeEventListener: () => {},
    },
    escaparHTML: String,
    porId: (id) => id === "dialogo-detalle" ? dialogo
      : id === "contenido-dialogo" ? contenido : { textContent: "" },
    obtenerAvisos: () => obtenerDatosPresentacion("tecnico").avisos,
    disponible: () => false,
    navegar: () => assert.fail("no debe navegar sin capacidad"),
    anunciar: () => assert.fail("no debe anunciar un acceso denegado como disponible"),
  });
  escuchar({
    target: { closest: (selector) => selector.includes("boton-avisos") ? {} : null },
    preventDefault: () => { impedido = true; },
  });
  assert.equal(dialogo.open, true);
  assert.equal(impedido, true);
  assert.doesNotMatch(contenido.innerHTML, /DEMO-BORRADOR-001|data-aviso-borrador-ref/);
  assert.match(contenido.innerHTML, /Tres llamamientos previstos/);
  assert.doesNotMatch(portal, /instalarDeeplinkAvisosBorradores/);
});

test("Avisos permanece visible en móvil y ambos activos avanzan de caché", () => {
  const movil = estilos.slice(estilos.indexOf("@media (max-width: 780px)"));
  assert.match(movil, /\.acciones-cabecera \.boton-avisos\s*\{[\s\S]*display:\s*inline-flex/);
  assert.match(movil, /\.acciones-cabecera \.boton-avisos\s*\{[\s\S]*min-width:\s*38px/);
  assert.doesNotMatch(movil, /\.boton-avisos\s*\{\s*display:\s*none/);
  assert.match(html, /portal-flujos\.css\?v=20260923-pweb16-v1/);
  assert.match(html, /portal\.js\?v=20260924-rescate-web-v2/);
  assert.doesNotMatch(html, /portal\.js\?v=20260924-web-c-ayuda-v5/);
  assert.doesNotMatch(html, /portal\.js\?v=20260924-web-c-ayuda-v4/);
  assert.doesNotMatch(html, /portal\.js\?v=20260924-web-c-v3/);
  assert.doesNotMatch(html, /portal\.js\?v=20260924-f2-dietas-consulta-v2/);
  assert.doesNotMatch(html, /portal\.js\?v=20260924-f2-cronos-permisos-v2/);
  assert.doesNotMatch(html, /portal\.js\?v=20260924-f2-cronos-permisos-v1/);
  assert.doesNotMatch(html, /portal\.js\?v=20260924-f2-personal-estados-v4/);
  assert.doesNotMatch(html, /portal\.js\?v=20260924-p1-personal-interno-v2/);
  assert.doesNotMatch(html, /portal\.js\?v=20260923-p4-reintento-v2/);
});
