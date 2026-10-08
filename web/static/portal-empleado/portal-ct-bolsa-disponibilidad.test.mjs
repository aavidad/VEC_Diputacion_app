import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { runInNewContext } from "node:vm";

const fuente = await readFile(new URL("portal.js", import.meta.url), "utf8");
const inicio = fuente.indexOf("resolverBolsa: (");
const fin = fuente.indexOf("    montar:", inicio);
assert.ok(inicio > 0 && fin > inicio);

function resolver({ permiso = true, carga = "listo", bolsas = [] } = {}) {
  return runInNewContext(`({${fuente.slice(inicio, fin)}}).resolverBolsa`, {
    vistaPermitida: () => permiso,
    VISTA_CANDIDATOS_BOLSA: "bolsa-candidatos",
    estado: { datosBolsas: { carga, datos: { bolsas } } },
  });
}

test("la categoría sin bolsa se confirma únicamente desde el conjunto autorizado listo", () => {
  for (const bolsas of [[], [{ bolsa_ref: "bolsa:antigua", categoria_clave: "auxiliar", vigente_hasta: "2026-01-01" }]]) {
    assert.equal(resolver({ bolsas })("", { categoriaRef: "categoria:rpt:auxiliar" }).estado, "sin_bolsa");
  }
  for (const carga of ["cargando", "error", "vacio", "denegado"]) {
    assert.equal(resolver({ carga })("", { categoriaRef: "categoria:rpt:auxiliar" }), null);
  }
  assert.equal(resolver({ permiso: false })("", { categoriaRef: "categoria:rpt:auxiliar" }), null);
  assert.equal(resolver({ bolsas: null })("", { categoriaRef: "categoria:rpt:auxiliar" }), null);
  assert.equal(resolver()("bolsa:oculta", { categoriaRef: "categoria:rpt:auxiliar" }), null);
  assert.equal(resolver()("", { categoriaRef: "" }), null);
});

test("la bolsa vigente conserva el acceso desde CT sin crear otro llamamiento", () => {
  const bolsa = { bolsa_ref: "bolsa:auxiliar", categoria_clave: "auxiliar", categoria: "Auxiliar administrativo", vigente_hasta: null };
  const buscar = resolver({ bolsas: [bolsa] });
  const porCategoria = buscar("", { categoriaRef: "categoria:rpt:auxiliar" });
  assert.equal(porCategoria.bolsa_ref, bolsa.bolsa_ref);
  assert.equal(porCategoria.categoria, bolsa.categoria);
  assert.equal(buscar(bolsa.bolsa_ref).categoria, bolsa.categoria);
  assert.equal(buscar("bolsa:otra"), null);
});
