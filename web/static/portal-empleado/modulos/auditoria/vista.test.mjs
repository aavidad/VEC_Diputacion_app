import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { renderizarVistaAuditoria } from "./vista.js";

test("Auditoría abre cerrada y no presenta la muestra ficticia como resultado", () => {
  const html = renderizarVistaAuditoria();
  assert.match(html, /Consulta no configurada/u);
  assert.match(html, /No se han solicitado ni mostrado eventos/u);
  assert.match(html, /Referencia exacta del recurso/u);
  assert.match(html, /competencia, recurso y finalidad/u);
  assert.match(html, /data-auditoria-ayuda aria-controls="auditoria-ayuda" aria-expanded="false"/u);
  assert.match(html, /disabled aria-disabled="true"/u);
  assert.doesNotMatch(html, /AUD-2026|rec_sint_|cor_sint_|E\. M\. R\.|<table/u);
});

test("la denegación tampoco revela eventos, identidad o recurso", () => {
  const html = renderizarVistaAuditoria({ estado: "denegado", ayudaAbierta: true });
  assert.match(html, /Acceso denegado/u);
  assert.match(html, /No hay eventos visibles con este acceso/u);
  assert.match(html, /id="auditoria-ayuda" class="panel auditoria-ayuda"  aria-labelledby/u);
  assert.match(html, /aria-expanded="true"/u);
  assert.doesNotMatch(html, /AUD-2026|rec_sint_|cor_sint_|E\. M\. R\.|<table/u);
});

test("la vista no añade red, almacenamiento ni datos locales de auditoría", async () => {
  const fuente = await readFile(new URL("./vista.js", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /(?:fetch\(|XMLHttpRequest|localStorage|sessionStorage|indexedDB|document\.cookie|datos-presentacion\.js)/iu);
  assert.doesNotMatch(fuente, /EVENTOS_AUDITORIA_PRESENTACION|ESTADO_AUDITORIA_PRESENTACION/u);
});

test("la vista carga la versión actual del catálogo de Auditoría", async () => {
  const fuente = await readFile(new URL("./vista.js", import.meta.url), "utf8");
  assert.match(fuente, /from "\.\/i18n\.js\?v=20260924-f2-web2"/u);
  const catalogo = await import("./i18n.js?v=20260924-f2-web2");
  assert.equal(catalogo.crearTraductorAuditoria()("estado_no_configurado"), "Consulta no configurada");
  assert.match(renderizarVistaAuditoria(), /Consulta no configurada/u);
});
