import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { MENSAJES_AUDITORIA_ES, crearTraductorAuditoria } from "./i18n.js";

test("el catálogo español de Auditoría está cerrado y cubre las etiquetas de interfaz", () => {
  const requeridas = ["titulo", "vista_accesos", "verificar_evidencia", "exportar_resultados", "anuncio_filtros", "contador_visibles"];
  requeridas.forEach((clave) => assert.equal(typeof MENSAJES_AUDITORIA_ES[clave], "string"));
  const t = crearTraductorAuditoria();
  assert.equal(t("contador_visibles", { visibles: 2, total: 6 }), "2 de 6");
  assert.throws(() => t("clave_inexistente"), /no definido/u);
});

test("la vista usa el catálogo para las etiquetas, avisos y acciones", async () => {
  const fuente = await readFile(new URL("./vista.js", import.meta.url), "utf8");
  ["sobrelinea", "tabla_caption", "actor_enmascarado", "verificar_evidencia", "exportar_resultados", "anuncio_filtros"].forEach((clave) => assert.ok(fuente.includes(`t("${clave}")`)));
  assert.match(fuente, /crearTraductorAuditoria/u);
});
