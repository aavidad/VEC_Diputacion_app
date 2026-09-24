import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import vm from "node:vm";

const catalogo = readFileSync(new URL("./i18n-publica.js", import.meta.url), "utf8");
const controlador = readFileSync(new URL("./bolsa.js", import.meta.url), "utf8");
const contexto = { globalThis: {} };
vm.runInNewContext(catalogo, contexto);
const { mensajes, t, numero, plural } = contexto.globalThis.VECBolsaI18n;

test("el catálogo público localiza números y plurales de resultados y directorio", () => {
  assert.equal(numero(12345), "12.345");
  assert.equal(plural("convocatoria_encontrada", 0), "0 convocatorias encontradas");
  assert.equal(plural("convocatoria_encontrada", 1), "1 convocatoria encontrada");
  assert.equal(plural("convocatoria_encontrada", 12345), "12.345 convocatorias encontradas");
  assert.equal(plural("requisito", 1), "1 requisito");
  assert.equal(plural("requisito", 2), "2 requisitos");
  assert.equal(plural("documento", 0), "0 documentos");
  assert.equal(plural("ayuda", 1), "1 ayuda");
  assert.equal(plural("proceso_publicado", 1), "1 proceso publicado");
  assert.equal(plural("plazo_abierto", 2), "2 plazos abiertos");
});

test("las plantillas conservan los valores de la fuente y traducen solo el envoltorio", () => {
  const revision = "ref:publica:2026";
  const huella = "a".repeat(64);
  assert.equal(t("fuente_actualizada", { revision, fecha: "24 sept 2026, 07:14" }),
    "Fuente ref:publica:2026 · actualizada 24 sept 2026, 07:14");
  assert.equal(t("catalogo_resumen", { referencia: "cat:publico", version: "1", total: "2", huella: huella.slice(0, 16) }),
    `Catálogo cat:publico · versión 1 · 2 categorías · huella ${huella.slice(0, 16)}…`);
  assert.equal(t("catalogo_resumen_aria", { referencia: "cat:publico", version: "1", total: "2", huella }),
    `Catálogo cat:publico, versión 1, 2 categorías, huella SHA-256 ${huella}`);
  assert.equal(t("abrir_documento", { formato: "PDF", titulo: "Bases públicas" }), "Abrir PDF: Bases públicas");
  assert.equal(t("ver_procesos_de", { categoria: "Auxiliar administrativo" }), "Ver procesos de Auxiliar administrativo");
});

test("cada clave visible usada por el controlador existe, incluidas las dos formas plurales", () => {
  const claves = [...controlador.matchAll(/\bt\("([a-z_]+)"/g)].map((coincidencia) => coincidencia[1]);
  const plurales = [...controlador.matchAll(/\bplural\("([a-z_]+)"/g)].map((coincidencia) => coincidencia[1]);
  assert.ok(claves.length > 20);
  assert.ok(plurales.length >= 6);
  for (const clave of claves) assert.equal(typeof mensajes[clave], "string", `falta ${clave}`);
  for (const clave of plurales) {
    assert.equal(typeof mensajes[`${clave}_uno`], "string", `falta ${clave}_uno`);
    assert.equal(typeof mensajes[`${clave}_otros`], "string", `falta ${clave}_otros`);
  }
  assert.doesNotMatch(controlador, /"(?:Publicada el|Bases publicadas el|Ver procesos|Cargando el catálogo profesional…|El directorio no está disponible\.)"/);
});
