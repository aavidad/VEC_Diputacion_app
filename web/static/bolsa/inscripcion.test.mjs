import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import vm from "node:vm";

const fuente = readFileSync(new URL("./inscripcion.js", import.meta.url), "utf8");
const contexto = { globalThis: {} };
vm.runInNewContext(fuente, contexto);
const { enlace } = contexto.globalThis.VECBolsaInscripcion;
const plazo = (clave, situacion) => ({ tipo: { clave }, situacion });

test("«Inscribirme» solo aparece con un plazo de inscripción que el servidor declara abierto", () => {
  const convocatoria = { identificador_publico: "bolsa-operario-diputacion-2026" };
  assert.equal(enlace({ convocatoria, plazos: [plazo("inscripcion", "abierto")] }),
    "/area-personal/?vista=solicitud&id=bolsa-operario-diputacion-2026");
  assert.equal(enlace({ convocatoria, plazos: [plazo("inscripcion", "cerrado")] }), "");
  assert.equal(enlace({ convocatoria, plazos: [plazo("alegaciones", "abierto")] }), "");
  assert.equal(enlace({ convocatoria: { identificador_publico: "../x" }, plazos: [plazo("inscripcion", "abierto")] }), "");
});

test("la ficha pública pinta el enlace con texto del catálogo y el destino es la única ruta privada", () => {
  const controlador = readFileSync(new URL("./bolsa.js", import.meta.url), "utf8");
  const html = readFileSync(new URL("./index.html", import.meta.url), "utf8");
  assert.match(controlador, /texto\("a", t\("inscribirme"\), "boton-primario enlace-inscripcion"\)/u);
  assert.match(html, /<script src="\/bolsa\/inscripcion\.js\?v=[\w-]+" defer><\/script>/u);
  assert.equal(fuente.match(/\/area-personal/gu).length, 1);
  assert.doesNotMatch(controlador, /\/area-personal|\/api\/vec/u);
});
