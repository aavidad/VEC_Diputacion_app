import test from "node:test";
import assert from "node:assert/strict";

import {
  crearHerramientasCatalogoCategorias,
  normalizarPaginaCategoriasProfesionales,
  ofertaPerteneceAlCatalogoActual,
} from "./catalogo-categorias.js";

const HUELLA = "2a9aa4a903b765c2f46ceb7f429f342a13b229e54ca45813472cb9d0aa1a4f3e";

function herramientas() {
  return crearHerramientasCatalogoCategorias({
    getPersonalCatalog: (vista) => vista.personalCatalog,
    workTableTextCollator: new Intl.Collator("es", { sensitivity: "base" }),
  });
}

test("normaliza la respuesta gobernada sin perder categorías ni metadatos", () => {
  const items = Array.from({ length: 58 }, (_, indice) => ({
    clave: `categoria-${indice + 1}`,
    etiqueta: `Categoría ${indice + 1}`,
  }));
  const pagina = normalizarPaginaCategoriasProfesionales({
    items,
    total: 58,
    catalogo: { catalogo_id: "categorias-profesionales", catalogo_version: 1 },
    fuente: { demostracion: true, aviso: "Contenido de demostración" },
  });

  assert.equal(pagina.items.length, 58);
  assert.equal(pagina.total, 58);
  assert.equal(pagina.catalogo.catalogo_id, "categorias-profesionales");
  assert.equal(pagina.demostracion, true);
  assert.equal(pagina.aviso, "Contenido de demostración");
});

test("el directorio conserva las 58 categorías y sus claves estables", () => {
  const items = Array.from({ length: 58 }, (_, indice) => ({
    slug: `categoria-${indice + 1}`,
    name: `Categoría ${indice + 1}`,
    orden: 58 - indice,
  }));
  const vista = { personalCatalog: { categories: { items } } };
  const categorias = herramientas().obtenerCategoriasProfesionales(vista);

  assert.equal(categorias.length, 58);
  assert.equal(categorias[0].clave, "categoria-58");
  assert.equal(categorias.at(-1).clave, "categoria-1");
});

test("solo reutiliza ofertas vinculadas a la versión y huella vigentes", () => {
  const referencia = {
    id: "categorias-profesionales",
    version: 1,
    huellaSHA256: HUELLA,
  };
  const oferta = {
    categoryKey: "administrativo",
    categoryCatalogID: referencia.id,
    categoryCatalogVersion: referencia.version,
    categoryCatalogSHA256: referencia.huellaSHA256,
  };

  assert.equal(ofertaPerteneceAlCatalogoActual(oferta, ["administrativo"], referencia), true);
  assert.equal(ofertaPerteneceAlCatalogoActual({ ...oferta, categoryCatalogSHA256: "0".repeat(64) }, ["administrativo"], referencia), false);
  assert.equal(ofertaPerteneceAlCatalogoActual({ categoryKey: "administrativo" }, ["administrativo"], referencia), false);
  assert.equal(ofertaPerteneceAlCatalogoActual(oferta, ["otra-categoria"], referencia), false);
});
