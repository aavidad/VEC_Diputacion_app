import test from "node:test";
import assert from "node:assert/strict";
import { cargarTextos } from "../comun/textos.js";
import { renderizarRankingBolsa, rutaRankingBolsa } from "./portal-baremacion-ranking.js";
import { validarCandidato } from "./portal-bolsas-contrato.js";
import { rutaCandidatosBolsa } from "./portal-bolsas-api.js";
const textos = await cargarTextos("bolsa-ranking");
const e = (v) => String(v ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;");
const candidato = { participacion_ref: "participacion:uno", orden: null, orden_acta: 1, razon_orden: "sin_turno", nombre_visible: "Carmen Molina <A>", documento_enmascarado: "***1234**", estado_clave: "trabajando", estado_desde: "2026-10-13T10:00:00Z", disponible_desde: null, ultimo_llamamiento: null, contactos_total: 0 };
test("puntos originales opcionales, cero válido y ausencia distinta; se rechaza HTML", () => {
  assert.equal(validarCandidato({ ...candidato, puntuacion: "0" }).puntuacion, "0");
  assert.equal(validarCandidato({ ...candidato, puntuacion: null }).puntuacion, null);
  assert.equal(Object.hasOwn(validarCandidato(candidato), "puntuacion"), false);
  for (const valor of ["<b>3</b>", 3, "NaN", "-3", "1e9"]) assert.throws(() => validarCandidato({ ...candidato, puntuacion: valor }));
});
test("ranking pagina por acta, conserva idioma y limpia filtros de llamamiento", () => {
  assert.match(rutaCandidatosBolsa("bolsa:uno", { orden: "acta", cursor: "participacion:uno" }), /orden=acta/u);
  const ruta = rutaRankingBolsa("?lang=en&seguimiento=llamamiento:uno&estado=renuncia", "bolsa:uno", "participacion:uno");
  assert.match(ruta, /lang=en/u); assert.match(ruta, /cursor=participacion%3Auno/u);
  assert.match(ruta, /#bolsa\/baremacion$/u); assert.doesNotMatch(ruta, /seguimiento|estado/u);
});
test("tabla conserva decimales, no revela referencias y enlaza el total al mismo filtro", () => {
  const bolsa = { bolsa_ref: "bolsa:uno", categoria: "Administrativo", total: 2 };
  const html = renderizarRankingBolsa({ bolsas: [bolsa], referencia: bolsa.bolsa_ref, pagina: { bolsa, candidatos: [{ ...candidato, puntuacion: "12.3456" }, { ...candidato, nombre_visible: "Antonio Pérez", puntuacion: null }], hay_mas: false }, carga: "listo", ayuda: false, cursor: "" }, { textos, escaparHTML: e, search: "?lang=es" });
  assert.match(html, /12,3456/u); assert.match(html, /No consta/u); assert.match(html, /Carmen Molina &lt;A&gt;/u);
  assert.match(html, /href="\?lang=es&amp;bolsa_ref=bolsa%3Auno#bolsa\/bolsa-candidatos"/u);
  assert.doesNotMatch(html, /participacion:uno|\*\*\*1234\*\*|calcular|publicar/u);
  assert.match(html, /id="ranking-ayuda" hidden/u);
});
