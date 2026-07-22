import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";
import vm from "node:vm";

const directorio = dirname(fileURLToPath(import.meta.url));
const contexto = {};
contexto.globalThis = contexto;
vm.runInNewContext(readFileSync(join(directorio, "contrato-v2.js"), "utf8"), contexto);
const contrato = contexto.VECBolsaContratoV2;

function categoria(clave = "auxiliar-administrativo", version = 1) {
  return { clave, version, etiqueta: "Auxiliar administrativo", descripcion: "Categoría pública.", semantica: "informacion" };
}

test("V2 resuelve referencias nominales por clave y versión exactas", () => {
  const datos = {
    esquema: "vec.bolsa.publico.convocatorias.v2",
    facetas: { categorias: [categoria()] },
    convocatorias: [{
      identificador_publico: "auxiliares-2026",
      categorias: [{ clave: "auxiliar-administrativo", version: 1 }],
    }],
  };
  const resueltas = contrato.validarListado(datos);
  assert.equal(resueltas.get("auxiliares-2026")[0].etiqueta, "Auxiliar administrativo");
});

test("V2 falla cerrado ante referencia desconocida o versión distinta", () => {
  for (const referencia of [
    { clave: "categoria-inexistente", version: 1 },
    { clave: "auxiliar-administrativo", version: 2 },
  ]) {
    assert.throws(() => contrato.validarListado({
      esquema: "vec.bolsa.publico.convocatorias.v2",
      facetas: { categorias: [categoria()] },
      convocatorias: [{ identificador_publico: "auxiliares-2026", categorias: [referencia] }],
    }), /desconocida|duplicada/);
  }
});

test("V2 rechaza diccionarios duplicados o ambiguos", () => {
  for (const entradas of [
    [categoria(), categoria()],
    [categoria(), categoria("auxiliar-administrativo", 2)],
  ]) {
    assert.throws(() => contrato.crearDiccionario(entradas), /ambiguo/);
  }
});

test("el detalle V2 exige diccionario propio biyectivo", () => {
  const base = {
    esquema: "vec.bolsa.publico.convocatoria.v2",
    convocatoria: { categorias: [{ clave: "auxiliar-administrativo", version: 1 }] },
  };
  assert.equal(contrato.validarDetalle({ ...base, diccionario_categorias: [categoria()] })[0].clave, "auxiliar-administrativo");
  assert.throws(() => contrato.validarDetalle({
    ...base,
    diccionario_categorias: [categoria(), categoria("administrativo", 1)],
  }), /no biyectivo/);
  assert.throws(() => contrato.validarDetalle({ ...base, diccionario_categorias: [] }), /fuera de límites/);
});

test("V1 no tiene fallback implícito en el runtime productivo", () => {
  assert.throws(() => contrato.validarListado({
    esquema: "vec.bolsa.publico.convocatorias.v1", facetas: { categorias: [] }, convocatorias: [],
  }), /esquema público inesperado/);
  assert.throws(() => contrato.validarDetalle({
    esquema: "vec.bolsa.publico.convocatoria.v1", convocatoria: { categorias: [] }, diccionario_categorias: [],
  }), /esquema de detalle inesperado/);
});

test("V2 rechaza categorías vacías y límites técnicos excedidos", () => {
  const listado = (categorias, facetas = [categoria()]) => ({
    esquema: "vec.bolsa.publico.convocatorias.v2",
    facetas: { categorias: facetas },
    convocatorias: [{ identificador_publico: "auxiliares-2026", categorias }],
  });
  assert.throws(() => contrato.validarListado(listado([])), /inválidas/);
  assert.throws(() => contrato.validarListado(listado(
    Array.from({ length: 129 }, (_, indice) => ({ clave: `categoria-${indice}`, version: 1 })),
    Array.from({ length: 129 }, (_, indice) => categoria(`categoria-${indice}`, 1)),
  )), /inválidas/);
  assert.throws(() => contrato.crearDiccionario(
    Array.from({ length: 1025 }, (_, indice) => categoria(`categoria-${indice}`, 1)),
  ), /excesivo/);
  assert.throws(() => contrato.validarDetalle({
    esquema: "vec.bolsa.publico.convocatoria.v2",
    convocatoria: { categorias: [] },
    diccionario_categorias: [],
  }), /fuera de límites/);
  assert.throws(() => contrato.validarDetalle({
    esquema: "vec.bolsa.publico.convocatoria.v2",
    convocatoria: { categorias: [{ clave: "categoria-0", version: 1 }] },
    diccionario_categorias: Array.from({ length: 129 }, (_, indice) => categoria(`categoria-${indice}`, 1)),
  }), /fuera de límites/);
});

test("V2 aplica la sintaxis canónica exacta de categoría", () => {
  for (const clave of ["Auxiliar", "auxiliar.admin", "auxiliar-", "auxiliar\u0000admin", "áuxiliar", `${"a".repeat(80)}b`]) {
    assert.throws(() => contrato.crearDiccionario([categoria(clave, 1)]), /inválida/);
  }
});
