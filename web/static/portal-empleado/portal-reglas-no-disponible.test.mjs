import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

// Reglas y versiones: un 404 de /api/vec/bolsa/politica-cese abre la vista para
// decir «no disponible» (antes devolvía a Inicio); un 403 sigue en Inicio.
const fuente = await readFile(new URL("portal.js", import.meta.url), "utf8");
const cuerpo = (nombre) => { const i = fuente.indexOf(`function ${nombre}(`); return fuente.slice(i, fuente.indexOf("\n}\n", i)); };

test("el 404 de la política de cese abre Reglas como no disponible y el 403 vuelve a Inicio", () => {
  const comprobar = cuerpo("comprobarAccesoPoliticaCese");
  assert.match(comprobar, /estado\.politicaCeseAusente = error\?\.estado === 404;/u);
  assert.match(comprobar, /if \(estado\.politicaCeseAusente && estado\.vista === "portal"\) navegar\("reglas"\);\s*else history\.replaceState\(null, "", rutaDeVista\("portal"\)\);/u);
  assert.match(cuerpo("vistaPermitida"), /vista === "reglas" && estado\.politicaCeseAusente\) return coordinadorModulos\.obtenerCatalogo\(\)\.some\(\(m\) => m\.clave === "bolsa"\)/u);
  assert.match(cuerpo("montarVistaBolsa"), /noDisponible: estado\.politicaCeseAusente/u);
  assert.match(cuerpo("cargarFuenteDatos"), /estado\.politicaCeseAusente = false;/u);
});
