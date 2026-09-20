import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { ESTADO_AUDITORIA_PRESENTACION, EVENTOS_AUDITORIA_PRESENTACION } from "./datos-presentacion.js";
import { renderizarVistaAuditoria } from "./vista.js";

test("Auditoría presenta una muestra sintética, minimizada y con límites explícitos", () => {
  const html = renderizarVistaAuditoria();
  assert.match(html, /datos ficticios y minimizados/u);
  assert.match(html, /Actor enmascarado/u);
  assert.match(html, /Sin validez probatoria/u);
  assert.match(html, /Almacén durable, segregado y de solo adición/u);
  assert.match(html, /Exportación firmada y controlada/u);
  assert.match(html, /disabled aria-disabled="true"/u);
  assert.equal(ESTADO_AUDITORIA_PRESENTACION.estado, "visual_pendiente_backend");
  assert.equal(EVENTOS_AUDITORIA_PRESENTACION.length, 6);
});

test("Auditoría filtra localmente por vista, módulo, resultado y periodo", () => {
  const accesos = renderizarVistaAuditoria({ vista: "accesos" });
  const denegados = renderizarVistaAuditoria({ resultado: "Denegado", periodo: "18/09/2026" });
  assert.match(accesos, /Acceso a incidencia/u);
  assert.doesNotMatch(accesos, /Preparación de informe/u);
  assert.match(denegados, /Consulta de justificante/u);
  assert.doesNotMatch(denegados, /Preparación de informe/u);
});

test("La superficie no añade red ni almacenamiento persistente", async () => {
  const fuente = await readFile(new URL("./vista.js", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /(?:fetch\(|XMLHttpRequest|localStorage|sessionStorage|indexedDB|document\.cookie)/iu);
});
