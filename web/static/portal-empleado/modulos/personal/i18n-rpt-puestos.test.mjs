import assert from "node:assert/strict";
import test from "node:test";
import { readFileSync } from "node:fs";
import { crearTraductorRPTPuestos, formatearCentimosRPT, formatearRecuentoRPTPuestos, formatearResumenRPTPuestos, formatearEnlaceAgrupacionRPT, IDIOMA_EFECTIVO_RPT_PUESTOS, LOCALIZACION_EFECTIVA_RPT_PUESTOS } from "./i18n-rpt-puestos.js";
import { cargarTextos } from "../../../comun/textos.js";
import { localizacionDe } from "../../../comun/idioma.js";

test("i18n RPT localiza recuentos e importes sin tocar el catálogo común", () => {
  assert.equal(LOCALIZACION_EFECTIVA_RPT_PUESTOS, localizacionDe(IDIOMA_EFECTIVO_RPT_PUESTOS));
  const t = crearTraductorRPTPuestos();
  assert.match(t("ayuda"), /No contiene ocupantes/);
  assert.equal(formatearResumenRPTPuestos({ puestos: 842, dotacion: 1714, categorias: 145, centros: 41 }), "842 puestos · 1.714 dotaciones · 145 categorías · 41 centros");
  assert.equal(formatearResumenRPTPuestos({ puestos: 2, dotacion: 3, categorias: 4, centros: 5 }), "2 puestos · 3 dotaciones · 4 categorías · 5 centros");
  assert.equal(formatearRecuentoRPTPuestos(1, "puestos"), "1 puesto");
  assert.equal(formatearRecuentoRPTPuestos(145, "categorias"), "145 categorías");
  assert.equal(formatearRecuentoRPTPuestos(41, "centros"), "41 centros");
  assert.match(formatearCentimosRPT(1427496), /14[. ]274,96/);
  assert.throws(() => formatearRecuentoRPTPuestos(-1, "puestos"), /no válido/);
});

test("RPT toma idioma y localización del catálogo realmente cargado", async () => {
  const textos = await cargarTextos("personal");
  assert.equal(IDIOMA_EFECTIVO_RPT_PUESTOS, textos.idioma);
  assert.equal(LOCALIZACION_EFECTIVA_RPT_PUESTOS, textos.localizacion);
});

test("etiqueta de agrupación pluraliza puestos y dotaciones en ES y EN", () => {
  for (const [idioma, localizacion, palabraPuesto, palabraPuestos, palabraDotacion, palabraDotaciones] of [
    ["es", "es-ES", "puesto", "puestos", "dotación", "dotaciones"],
    ["en", "en-GB", "post", "posts", "allocation", "allocations"],
  ]) {
    const catalogo = JSON.parse(readFileSync(new URL(`../../../textos/${idioma}/personal.json`, import.meta.url), "utf8")).rpt_puestos;
    for (const [puestos, dotacion, esperadoPuestos, esperadoDotacion] of [
      [1, 1, palabraPuesto, palabraDotacion], [1, 2, palabraPuesto, palabraDotaciones],
      [2, 1, palabraPuestos, palabraDotacion], [0, 0, palabraPuestos, palabraDotaciones],
    ]) {
      for (const vista of ["categorias", "centros"]) {
        const etiqueta = formatearEnlaceAgrupacionRPT(vista, "GABINETE", puestos, dotacion, localizacion, catalogo);
        assert.match(etiqueta, new RegExp(`${puestos} ${esperadoPuestos}.*${dotacion} ${esperadoDotacion}`, "u"));
        assert.doesNotMatch(etiqueta, /\{(?:puestos|dotacion)\}/u);
      }
    }
  }
});
