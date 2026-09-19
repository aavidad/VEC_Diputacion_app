import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";
import vm from "node:vm";

const directorio = dirname(fileURLToPath(import.meta.url));
const contexto = { globalThis: {} };
contexto.globalThis.globalThis = contexto.globalThis;
vm.runInNewContext(readFileSync(join(directorio, "contrato-v1.js"), "utf8"), contexto.globalThis);
const contrato = contexto.globalThis.VECBolsaContratoV1;

const huella = "a".repeat(64);
const categoria = { clave: "auxiliar-administrativo", version: 1, etiqueta: "Auxiliar administrativo", semantica: "informacion" };
const fuente = { revision: "demo-v1", actualizada_en: "2026-09-19T10:00:00Z", demostracion: true };
const plazo = { tipo: categoria, titulo: "Presentación", abre_en: "2026-09-20T00:00:00Z", cierra_en: "2026-10-01T00:00:00Z", etiqueta_situacion: "Abierto", semantica_situacion: "exito" };
const convocatoria = {
  identificador_publico: "auxiliares-2026", version: "v1", huella_sha256: huella,
  titulo: "Bolsa de auxiliares", resumen: "Información pública.", tipo: categoria, estado: categoria,
  categorias: [categoria], plazo_destacado: plazo, numero_requisitos: 1, numero_documentos: 2,
  numero_ayudas: 0, publicada_en: "2026-09-19T10:00:00Z",
};

test("V1 acepta categorías ya resueltas por el runtime", () => {
  const resueltas = contrato.validarListado({
    esquema: "vec.bolsa.publico.convocatorias.v1", fuente,
    facetas: { tipos: [categoria], categorias: [{ ...categoria, numero_resultados: 1 }], estados: [categoria] },
    paginacion: { pagina: 1, tamano: 12, total: 1, paginas: 1 }, convocatorias: [convocatoria],
  });
  assert.equal(resueltas.get("auxiliares-2026")[0].etiqueta, "Auxiliar administrativo");
  assert.equal(contrato.validarDetalle({
    esquema: "vec.bolsa.publico.convocatoria.v1", fuente, convocatoria,
    descripcion: "Detalle.", plazos: [plazo], requisitos: [], documentos: [], ayuda: [],
  })[0].clave, "auxiliar-administrativo");
});

test("V1 rechaza esquemas distintos y categorías sin resolver", () => {
  const listado = {
    esquema: "vec.bolsa.publico.convocatorias.v1", fuente,
    facetas: { tipos: [], categorias: [], estados: [] },
    paginacion: { pagina: 1, tamano: 12, total: 1, paginas: 1 }, convocatorias: [convocatoria],
  };
  assert.throws(() => contrato.validarListado({ ...listado, esquema: "vec.bolsa.publico.convocatorias.v2" }), /v1/);
  assert.throws(() => contrato.validarListado({
    ...listado, convocatorias: [{ ...convocatoria, categorias: [{ clave: categoria.clave, version: 1 }] }],
  }), /catálogo/);
});
