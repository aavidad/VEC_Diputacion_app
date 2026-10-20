import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

// Copia local 20/10/2026: #bolsa/publicacion-bases por enlace directo se quedaba
// para siempre en «Comprobando» y sin el menú de Bolsa. renderizar() exigía la
// base del panel interno, pero prepararBolsaBase() no la crea para esta vista
// (vistaNecesitaBolsa la excluye), así que nunca se llegaba a montarla.
const fuente = await readFile(new URL("portal.js", import.meta.url), "utf8");
const cuerpo = (nombre) => { const i = fuente.indexOf(`function ${nombre}(`); return fuente.slice(i, fuente.indexOf("\n}\n", i)); };
const lista = JSON.parse(fuente.match(/const VISTAS_BOLSA_SIN_BASE = Object\.freeze\((\[[^\]]*\])\);/u)[1]);

function necesitaBase(vista, { modulo = "bolsa", pendiente = false } = {}) {
  const funcion = new Function("VISTAS_BOLSA_SIN_BASE", "moduloDeVistaPortal", "vistaBolsaPendienteNoCompuesta", "estado",
    `${cuerpo("vistaNecesitaBolsa")}\n}\nreturn vistaNecesitaBolsa;`);
  return funcion(lista, () => modulo, () => pendiente, { vista })(vista);
}

test("la publicación de las bases no espera a la base de Bolsa por enlace directo", () => {
  assert.equal(necesitaBase("publicacion-bases"), false);
  for (const vista of ["solicitudes", "meritos", "convocatorias", "alegaciones"]) assert.equal(necesitaBase(vista), false);
  assert.equal(necesitaBase("resumen"), true);
  assert.equal(necesitaBase("portal", { modulo: "portal" }), true);
});

test("renderizar y prepararBolsaBase deciden la base con la misma regla", () => {
  const renderizar = cuerpo("renderizar");
  assert.match(renderizar, /const necesitaBolsaBase = estado\.vista !== "portal" && vistaNecesitaBolsa\(estado\.vista\);/u);
  assert.match(cuerpo("prepararBolsaBase"), /if \(!vistaNecesitaBolsa\(\)\) \{/u);
  // Ninguna otra copia de la lista que pueda divergir.
  assert.equal(fuente.match(/\["solicitudes", "meritos", "convocatorias", "publicacion-bases", "alegaciones"\]/gu).length, 1);
  assert.doesNotMatch(fuente, /\["solicitudes", "meritos", "convocatorias", "alegaciones"\]/u);
});

test("con 403 o 503 la vista montada se queda: solo cambia la entrada del menú", () => {
  const montar = cuerpo("montarVistaBolsa");
  assert.match(montar, /alDisponibilidad: cambiarDisponibilidadPublicacionBases/u);
  const cambiar = cuerpo("cambiarDisponibilidadPublicacionBases");
  assert.doesNotMatch(cambiar, /renderizar|desmontar/u);
  assert.match(cambiar, /actualizarNavegacionModulos\(\)/u);
  assert.doesNotMatch(cuerpo("actualizarNavegacionModulos"), /renderizar\(|desmontarPublicacionBases/u);
});
