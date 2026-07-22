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

const HUELLA_A = "a".repeat(64);
const HUELLA_B = "b".repeat(64);

function snapshot(version = 1, huella_sha256 = HUELLA_A) {
  return { referencia: "categorias-profesionales", version, huella_sha256 };
}

function categoria(clave = "auxiliar-administrativo", version = 1, catalogo_categorias = snapshot(version)) {
  return {
    clave, version, catalogo_categorias, etiqueta: "Auxiliar administrativo",
    descripcion: "Categoría pública.", semantica: "informacion",
  };
}

test("V2 resuelve una página mixta con categorías históricas desde el diccionario", () => {
  const datos = {
    esquema: "vec.bolsa.publico.convocatorias.v2",
    facetas: { categorias: [categoria("auxiliar-administrativo", 2, snapshot(2, HUELLA_B))] },
    diccionario_categorias: [
      categoria("auxiliar-administrativo", 1, snapshot(1, HUELLA_A)),
      { ...categoria("auxiliar-administrativo", 2, snapshot(2, HUELLA_B)), etiqueta: "Auxiliar administrativo actualizado" },
    ],
    convocatorias: [
      { identificador_publico: "historica-a", catalogo_categorias: snapshot(1, HUELLA_A), categorias: [{ clave: "auxiliar-administrativo", version: 1 }] },
      { identificador_publico: "actual-b", catalogo_categorias: snapshot(2, HUELLA_B), categorias: [{ clave: "auxiliar-administrativo", version: 2 }] },
    ],
  };
  const resueltas = contrato.validarListado(datos);
  assert.equal(resueltas.get("historica-a")[0].etiqueta, "Auxiliar administrativo");
  assert.equal(resueltas.get("actual-b")[0].etiqueta, "Auxiliar administrativo actualizado");
});

test("V2 falla cerrado ante referencia desconocida o versión distinta", () => {
  for (const referencia of [
    { clave: "categoria-inexistente", version: 1 },
    { clave: "auxiliar-administrativo", version: 2 },
  ]) {
    assert.throws(() => contrato.validarListado({
      esquema: "vec.bolsa.publico.convocatorias.v2",
      facetas: { categorias: [categoria()] },
      diccionario_categorias: [categoria()],
      convocatorias: [{ identificador_publico: "auxiliares-2026", catalogo_categorias: snapshot(), categorias: [referencia] }],
    }), /desconocida|duplicada/);
  }
});

test("V2 rechaza diccionarios duplicados o ambiguos", () => {
  for (const entradas of [
    [categoria(), categoria()],
    [categoria(), categoria()],
  ]) {
    assert.throws(() => contrato.crearDiccionario(entradas), /ambiguo/);
  }
  assert.equal(contrato.crearDiccionario([
    categoria("auxiliar-administrativo", 1, snapshot(1, HUELLA_A)),
    categoria("auxiliar-administrativo", 2, snapshot(2, HUELLA_B)),
  ]).size, 2);
});

test("el detalle V2 exige diccionario propio biyectivo", () => {
  const base = {
    esquema: "vec.bolsa.publico.convocatoria.v2",
    convocatoria: { catalogo_categorias: snapshot(), categorias: [{ clave: "auxiliar-administrativo", version: 1 }] },
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
    esquema: "vec.bolsa.publico.convocatorias.v1", facetas: { categorias: [] }, diccionario_categorias: [], convocatorias: [],
  }), /esquema público inesperado/);
  assert.throws(() => contrato.validarDetalle({
    esquema: "vec.bolsa.publico.convocatoria.v1", convocatoria: { catalogo_categorias: snapshot(), categorias: [] }, diccionario_categorias: [],
  }), /esquema de detalle inesperado/);
});

test("V2 rechaza categorías vacías y límites técnicos excedidos", () => {
  const listado = (categorias, diccionario = [categoria()]) => ({
    esquema: "vec.bolsa.publico.convocatorias.v2",
    facetas: { categorias: [categoria()] }, diccionario_categorias: diccionario,
    convocatorias: [{ identificador_publico: "auxiliares-2026", catalogo_categorias: snapshot(), categorias }],
  });
  assert.throws(() => contrato.validarListado(listado([])), /inválidas/);
  assert.throws(() => contrato.validarListado(listado(
    Array.from({ length: 129 }, (_, indice) => ({ clave: `categoria-${indice}`, version: 1 })),
    Array.from({ length: 129 }, (_, indice) => categoria(`categoria-${indice}`, 1)),
  )), /inválidas/);
  assert.equal(contrato.crearDiccionario(
    Array.from({ length: 1025 }, (_, indice) => categoria(`categoria-${indice}`, 1)),
  ).size, 1025);
  assert.throws(() => contrato.crearDiccionario(
    Array.from({ length: 4097 }, (_, indice) => categoria(`categoria-${indice}`, 1)),
  ), /excesivo/);
  assert.throws(() => contrato.validarDetalle({
    esquema: "vec.bolsa.publico.convocatoria.v2",
    convocatoria: { catalogo_categorias: snapshot(), categorias: [] },
    diccionario_categorias: [],
  }), /fuera de límites/);
  assert.throws(() => contrato.validarDetalle({
    esquema: "vec.bolsa.publico.convocatoria.v2",
    convocatoria: { catalogo_categorias: snapshot(), categorias: [{ clave: "categoria-0", version: 1 }] },
    diccionario_categorias: Array.from({ length: 129 }, (_, indice) => categoria(`categoria-${indice}`, 1)),
  }), /fuera de límites/);
});

test("V2 falla cerrado cuando la referencia, huella o versión no son coherentes con el snapshot", () => {
  const base = {
    esquema: "vec.bolsa.publico.convocatorias.v2", facetas: { categorias: [categoria()] },
    diccionario_categorias: [categoria()],
    convocatorias: [{ identificador_publico: "historica-a", catalogo_categorias: snapshot(), categorias: [{ clave: "auxiliar-administrativo", version: 1 }] }],
  };
  for (const manipulacion of [
    { referencia: "otro-catalogo", version: 1, huella_sha256: HUELLA_A },
    snapshot(2, HUELLA_A),
    snapshot(1, HUELLA_B),
    { referencia: "categorias-profesionales", version: 1, huella_sha256: "invalida" },
  ]) {
    const datos = structuredClone(base);
    datos.convocatorias[0].catalogo_categorias = manipulacion;
    assert.throws(() => contrato.validarListado(datos), /snapshot/);
  }
});

test("V2 aplica patrón y fronteras exactas al snapshot de catálogo", () => {
  const referenciaMaxima = `a${"0._-".repeat(31)}xyz`;
  assert.equal(referenciaMaxima.length, 128);
  assert.doesNotThrow(() => contrato.validarCatalogoCategorias({
    referencia: "a", version: 1, huella_sha256: HUELLA_A,
  }));
  assert.doesNotThrow(() => contrato.validarCatalogoCategorias({
    referencia: referenciaMaxima, version: 2147483647, huella_sha256: HUELLA_B,
  }));

  for (const referencia of ["", "1catalogo", "Catalogo", "cat/privado", "categoría", `${referenciaMaxima}x`]) {
    assert.throws(() => contrato.validarCatalogoCategorias({
      referencia, version: 1, huella_sha256: HUELLA_A,
    }), /snapshot/);
  }
  for (const version of [0, 2147483648, 1.5, "1"]) {
    assert.throws(() => contrato.validarCatalogoCategorias({
      referencia: "categorias-profesionales", version, huella_sha256: HUELLA_A,
    }), /snapshot/);
  }
});

test("V2 liga cada entrada a su versión y mantiene un snapshot único por versión", () => {
  assert.throws(() => contrato.crearDiccionario([
    categoria("auxiliar-administrativo", 1, snapshot(2, HUELLA_A)),
  ]), /ambiguo/);

  const conflictos = [
    { referencia: "otro-catalogo", version: 1, huella_sha256: HUELLA_A },
    snapshot(1, HUELLA_B),
  ];
  for (const catalogoConflicto of conflictos) {
    assert.throws(() => contrato.crearDiccionario([
      categoria("auxiliar-administrativo", 1, snapshot(1, HUELLA_A)),
      categoria("administrativo", 1, catalogoConflicto),
    ]), /ambiguo/);
  }

  assert.equal(contrato.crearDiccionario([
    categoria("auxiliar-administrativo", 1, snapshot(1, HUELLA_A)),
    categoria("administrativo", 1, snapshot(1, HUELLA_A)),
  ]).size, 2);
});

test("V2 aplica la sintaxis canónica exacta de categoría", () => {
  for (const clave of ["Auxiliar", "auxiliar.admin", "auxiliar-", "auxiliar\u0000admin", "áuxiliar", `${"a".repeat(80)}b`]) {
    assert.throws(() => contrato.crearDiccionario([categoria(clave, 1)]), /inválida/);
  }
});
