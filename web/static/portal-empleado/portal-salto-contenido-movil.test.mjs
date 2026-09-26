import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const raiz = new URL("./", import.meta.url);

test("el enlace de salto enfocado libera el botón de menú solo en móvil pequeño", async () => {
  const [html, css] = await Promise.all([
    readFile(new URL("index.html", raiz), "utf8"),
    readFile(new URL("portal.css", raiz), "utf8"),
  ]);

  assert.match(html, /<a class="salto-contenido" href="#contenido-principal" data-i18n-portal="txt_saltar_al_contenido_principal">Saltar al contenido principal<\/a>/u);
  assert.match(html, /<main id="contenido-principal"[^>]*tabindex="-1"/u);
  assert.match(html, /<button[^>]*id="boton-menu"[^>]*aria-expanded="false"/u);

  assert.match(css, /\.salto-contenido\s*\{[^}]*position:\s*fixed;[^}]*z-index:\s*1000;[^}]*top:\s*8px;[^}]*left:\s*8px;[^}]*transform:\s*translateY\(-150%\);/u);
  assert.match(css, /\.salto-contenido:focus\s*\{\s*transform:\s*none;\s*\}/u);
  assert.match(css, /@media \(max-width:\s*520px\)\s*\{\s*\.salto-contenido:focus\s*\{\s*position:\s*absolute;\s*display:\s*block;\s*top:\s*112px;\s*left:\s*8px;\s*width:\s*max-content;\s*max-width:\s*calc\(100% - 16px\);\s*\}\s*\}/u);
  assert.match(css, /a:focus-visible[\s\S]*?outline:\s*3px solid var\(--portal-azul-700\);/u);
});
