import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { runInNewContext } from "node:vm";
import { accesoBolsaEfectivo, resumenAccesosModulos } from "./portal-menu-bolsa.js?v=VEC_WEB_VERSION";

const fuente = await readFile(new URL("portal.js", import.meta.url), "utf8");
const inicio = fuente.indexOf("function disponibilidadBolsa()");
const fin = fuente.indexOf("function resolverAccesoPerfil(", inicio);
assert.ok(inicio > 0 && fin > inicio);

function disponibilidad(estado) {
  return runInNewContext(`${fuente.slice(inicio, fin)}; disponibilidadBolsa()`, {
    estado,
    vistaNecesitaBolsa: () => estado.vista === "portal" || estado.vista === "resumen",
    accesoBolsaEfectivo,
    traducirPortal: (clave) => clave,
    coordinadorModulos: { obtenerCatalogo: () => [{ clave: "bolsa" }, { clave: "contratacion_temporal" }] },
    superficieBorradores: { obtenerAcceso: () => ({ disponible: false, estado: "cargando" }) },
  });
}

test("F1-79: el enlace de Intervención termina sin esperar una lectura de Bolsa no iniciada", () => {
  const bolsa = disponibilidad({ vista: "contratacion-temporal", datosBolsas: null });
  assert.equal(bolsa.estado, "no_disponible");
  assert.equal(bolsa.disponible, false);
  assert.equal(resumenAccesosModulos([{ disponible: true }], bolsa.estado === "cargando"), "1 área disponible");
});

test("F1-79: Inicio comprueba Bolsa y conserva disponibilidad y denegación según la respuesta", () => {
  for (const vista of ["portal", "resumen"]) {
    assert.equal(disponibilidad({ vista, datosBolsas: null }).estado, "cargando");
    assert.equal(disponibilidad({ vista, datosBolsas: { carga: "cargando" } }).disponible, true);
    const denegado = disponibilidad({ vista, datosBolsas: { carga: "denegado" } });
    assert.equal(denegado.disponible, false);
    assert.equal(denegado.estado, "denegado");
    assert.equal(disponibilidad({ vista, datosBolsas: { carga: "listo", datos: { bolsas: [] } } }).disponible, true);
  }
});
