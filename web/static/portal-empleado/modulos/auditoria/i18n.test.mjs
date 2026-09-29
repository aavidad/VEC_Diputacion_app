import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { spawnSync } from "node:child_process";
import test from "node:test";
import { MENSAJES_AUDITORIA, crearTraductorAuditoria } from "./i18n.js";

test("las etiquetas visibles proceden del catálogo de Auditoría", async () => {
  const fuente = await readFile(new URL("./vista.js", import.meta.url), "utf8");
  for (const [, clave] of fuente.matchAll(/\bt\("([a-z_]+)"\)/gu)) {
    assert.equal(typeof MENSAJES_AUDITORIA[clave], "string", clave);
  }
  for (const estado of ["no_configurado", "cargando_opciones", "esperando", "cargando", "disponible", "vacio", "denegado", "error", "invalido"]) {
    assert.equal(typeof MENSAJES_AUDITORIA[`estado_${estado}`], "string");
  }
  assert.equal(crearTraductorAuditoria()("pagina", { numero: 3 }), "Página 3");
  assert.throws(() => crearTraductorAuditoria()("desconocida"), /no definido/u);
});

test("catálogo inglés completo y elegido por el idioma común", async () => {
  const catalogoEs = JSON.parse(await readFile(new URL("../../../textos/es/auditoria.json", import.meta.url), "utf8")).general;
  const catalogoEn = JSON.parse(await readFile(new URL("../../../textos/en/auditoria.json", import.meta.url), "utf8")).general;
  assert.deepEqual(Object.keys(catalogoEn).sort(), Object.keys(catalogoEs).sort());
  for (const [clave, valor] of Object.entries(catalogoEn)) assert.ok(valor.trim(), clave);
  assert.equal(crearTraductorAuditoria(catalogoEn)("pagina", { numero: 3 }), "Page 3");
  const codigo = `globalThis.location = { href: "https://example.test/?lang=en" };
    const { renderizarVistaAuditoria } = await import(${JSON.stringify(new URL("./vista.js", import.meta.url).href)});
    const denegado = renderizarVistaAuditoria({ estado: "denegado" });
    const filtrosPendientes = renderizarVistaAuditoria({ estado: "esperando", habilitada: true });
    const registro = { id: "aud_1", modulo_id: "personal", accion: "relacion.actualizada", actor_ref: "per_1",
      ocurrido_en: "2026-09-28T08:00:00Z", resultado: "confirmado", expediente_ref: "exp_1",
      recibo_ref: "rec_1", antes_sha256: "a".repeat(64), despues_sha256: "b".repeat(64),
      motivo: "rectificacion", fuente: "Personal", datos_disponibles: true, antes: {}, despues: {} };
    const html = renderizarVistaAuditoria({ estado: "disponible", habilitada: true, ejemplo: true, registros: [registro] });
    if (!denegado.includes("Access denied") || !filtrosPendientes.includes("Press Search to apply the filters") || html.includes("Acceso denegado") ||
      !html.includes("Updated the employment record") || !html.includes("Fictitious sample") ||
      !html.includes("View technical details") || !html.includes("Sept 2026")) process.exit(1);`;
  const resultado = spawnSync(process.execPath, ["--input-type=module", "-e", codigo], { encoding: "utf8" });
  assert.equal(resultado.status, 0, resultado.stderr);
});
