import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { crearTraductorDietas, MENSAJES_DIETAS_ES } from "./i18n.js";
import { crearTraductorBorradoresDietas } from "./i18n-borradores.js";
import { MENSAJES_CIRCUITO_DIETAS_ES } from "./i18n-circuito.js";

test("el catálogo común traduce Dietas, el documento y el circuito con el mismo traductor", () => {
  for (const clave of ["borradores_propios_titulo_registrados", "comision_bloque_kilometraje", "comision_total_provisional", "circuito_etapa_fiscalizacion"])
    assert.equal(crearTraductorDietas()(clave), MENSAJES_DIETAS_ES[clave]);
  assert.equal(MENSAJES_DIETAS_ES.circuito_recibo, MENSAJES_CIRCUITO_DIETAS_ES.circuito_recibo);
  const traducir = (clave) => clave === "comision_bloque_dietas" ? "Allowances" : clave;
  assert.equal(crearTraductorBorradoresDietas(traducir)("comision_bloque_dietas"), "Allowances");
  assert.equal(crearTraductorDietas()("circuito_periodo_valor", { inicio: "1 sep", fin: "2 sep" }), "1 sep a 2 sep");
});

test("el recorrido interno importa solo vista, mapa y clientes conectados", async () => {
  const [comun, borradores, recorridos] = await Promise.all([
    readFile(new URL("./i18n.js", import.meta.url), "utf8"),
    readFile(new URL("./vista-borradores-propios.js", import.meta.url), "utf8"),
    readFile(new URL("./vista-recorridos.js", import.meta.url), "utf8"),
  ]);
  assert.match(comun, /i18n-borradores\.js/u);
  assert.match(comun, /i18n-circuito\.js/u);
  assert.match(borradores, /vista-mapa-comision\.js/u);
  assert.match(recorridos, /vista-borradores-propios\.js/u);
  assert.match(recorridos, /vista-bandeja-circuito\.js/u);
  for (const fuente of [comun, borradores, recorridos])
    assert.doesNotMatch(fuente, /adaptador-presentacion|datos-presentacion|calculador-rutas-presentacion|vista-itinerario|DEMO|sintétic/iu);
});
