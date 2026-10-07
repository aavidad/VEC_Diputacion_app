import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { cargarTextos, esMensajePlural } from "../comun/textos.js";

const MODULOS = [
  "bolsa", "bolsa-ofertas", "bolsas-publicas", "portal-bolsa",
  "contratacion-temporal-cobertura", "contratacion-temporal-lista-plazos",
  "portal", "area-personal", "portal-ayuda", "preferencias",
];
const VARIABLE = /\{([A-Za-z_][A-Za-z0-9_]*)\}/gu;
const leer = async (modulo, idioma) => JSON.parse(await readFile(new URL(`${idioma}/${modulo}.json`, import.meta.url), "utf8"));

function hojas(objeto, prefijo = "", resultado = new Map()) {
  for (const [clave, valor] of Object.entries(objeto)) {
    const ruta = prefijo ? `${prefijo}.${clave}` : clave;
    if (typeof valor === "string" || esMensajePlural(valor)) resultado.set(ruta, valor);
    else hojas(valor, ruta, resultado);
  }
  return resultado;
}

function variables(texto) {
  return [...texto.matchAll(VARIABLE)].map(([, nombre]) => nombre).sort();
}

for (const modulo of MODULOS) {
  test(`${modulo}: inglés completo, variables intactas y lectura sin respaldo`, async () => {
    const [base, ingles, textos] = await Promise.all([
      leer(modulo, "es"), leer(modulo, "en"), cargarTextos(modulo, { idioma: "en" }),
    ]);
    const originales = hojas(base);
    const traducidos = hojas(ingles);
    assert.deepEqual([...traducidos.keys()].sort(), [...originales.keys()].sort());
    assert.equal(textos.idioma, "en");
    assert.deepEqual(textos.faltantes, []);
    for (const [ruta, original] of originales) {
      const traducido = traducidos.get(ruta);
      assert.equal(typeof traducido, typeof original, ruta);
      const formas = typeof original === "string" ? [null] : Object.keys(original);
      if (typeof original !== "string") assert.deepEqual(Object.keys(traducido).sort(), formas.slice().sort(), ruta);
      for (const forma of formas) {
        const fuente = forma === null ? original : original[forma];
        const destino = forma === null ? traducido : traducido[forma];
        assert.ok(destino.trim(), `${ruta}: mensaje vacío`);
        assert.deepEqual(variables(destino), variables(fuente), `${ruta}: variables de ${forma ?? "texto"}`);
        const datos = Object.fromEntries(variables(destino).map((nombre) => [nombre, `dato_${nombre}`]));
        if (forma !== null) datos.cuenta = forma === "one" ? 1 : 2;
        const resultado = textos.traducir(ruta, datos);
        assert.doesNotMatch(resultado, VARIABLE, `${ruta}: variables sin resolver`);
        for (const nombre of variables(destino).filter((nombre) => nombre !== "cuenta")) {
          assert.ok(resultado.includes(`dato_${nombre}`), `${ruta}: pierde ${nombre}`);
        }
      }
    }
  });
}

test("emitir un llamamiento no presenta el correo como enviado o entregado", async () => {
  const portal = await cargarTextos("portal", { idioma: "en" });
  const ayuda = await cargarTextos("portal-ayuda", { idioma: "en" });
  for (const clave of ["panel_b7_emitido", "panel_b7_llamamiento_emitido", "panel_b7_estado_emitido"]) {
    const texto = portal.traducir(`panel_interno.${clave}`);
    assert.match(texto, /issued/i);
    assert.doesNotMatch(texto, /sent|delivered/i);
  }
  assert.match(portal.traducir("panel_interno.panel_b7_correo_limite"), /does not prove delivery/);
  assert.match(ayuda.traducir("ayuda.ayuda_b7_emitir_instruccion"), /Issue call-up/);
  assert.match(ayuda.traducir("ayuda.ayuda_b7_emitir_resultado"), /issued and awaiting response/);
  assert.match(ayuda.traducir("ayuda.ayuda_b7_emitir_titulo"), /issue/i);
});

test("ofrecerse y responder a una adjudicación conservan plazos distintos", async () => {
  const [politica, ofertas] = await Promise.all(["bolsa", "bolsa-ofertas"].map((modulo) => cargarTextos(modulo, { idioma: "en" })));
  assert.match(politica.traducir("plazos.rrhh_plazos_plazo_titulo"), /express interest/i);
  assert.match(ofertas.traducir("ofertas.col_plazo"), /express interest/i);
  assert.match(ofertas.traducir("ofertas.error_409_plazo_abierto"), /express interest.*still open/i);
  assert.match(ofertas.traducir("ofertas.error_409_respuesta_abierta"), /time to reply/i);
  assert.match(ofertas.traducir("ofertas.confirmar_llamamiento_directo", { plaza: 2 }), /direct call-up/);
});

test("renunciar a un llamamiento no se presenta como retirada de la bolsa", async () => {
  const [portal, area] = await Promise.all(["portal", "area-personal"].map((modulo) => cargarTextos(modulo, { idioma: "en" })));
  assert.match(portal.traducir("textos.txt_renuncia"), /declined/i);
  assert.doesNotMatch(portal.traducir("textos.txt_personas_en_renuncia"), /withdrew|withdrawn/i);
  assert.match(area.traducir("portal.respuesta.renuncia"), /decline/i);
  assert.match(portal.traducir("panel_interno.panel_contacto_no_respuesta"), /accepts or declines/);
});
