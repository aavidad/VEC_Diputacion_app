import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { datosMinimosMiBolsa } from "./aplicacion.js";
import { iniciarI18nAreaPersonal, traducir } from "./i18n.js";
import { renderizarPerfil } from "./vistas/perfil-meritos-solicitud.js";

test("Mi bolsa vacía no fabrica persona, iniciales ni referencias para el área personal", () => {
  const datos = datosMinimosMiBolsa({ consultada_en: "2026-09-24T09:00:00Z", participaciones: [] });
  assert.equal(datos.meta.presentacion, false);
  assert.equal(datos.sesion.nombre_visible, "");
  assert.equal(datos.sesion.iniciales, "—");
  assert.equal(datos.sesion.persona_ref, null);
  assert.equal(datos.perfil.referencia, null);
  assert.equal(datos.perfil.nombre_visible, "");
  assert.deepEqual(datos.capacidades, {});
  assert.equal(datos.disponibilidad.disponible, false);
  const texto = `${JSON.stringify(datos)}\n${renderizarPerfil(datos)}`;
  assert.doesNotMatch(texto, /Candidato identificado|candidato:identificado|perfil:pendiente|"CI"|DEMO-/u);
  assert.doesNotMatch(texto, /Identidad no facilitada|sint[ée]tic/iu);
});

test("la identidad no facilitada usa las claves del catálogo real y el respaldo común", async () => {
  const catalogo = JSON.parse(await readFile(new URL("./locales/es.json", import.meta.url), "utf8"));
  const claves = ["noFacilitada", "metodoNoFacilitado", "valorNoFacilitado"]
    .map((sufijo) => `areaPersonal.miBolsa.identidad.${sufijo}`);
  for (const clave of claves) assert.equal(typeof catalogo[clave], "string", clave);
  await iniciarI18nAreaPersonal({ querySelectorAll: () => [] }, async (ruta, opciones) => {
    assert.equal(ruta, "/area-personal/locales/es.json");
    assert.equal(opciones.credentials, "omit");
    return { ok: true, json: async () => catalogo };
  });
  const datos = datosMinimosMiBolsa({ consultada_en: "2026-09-24T09:00:00Z", participaciones: [] });
  assert.equal(datos.sesion.nombre_visible, "");
  assert.equal(datos.sesion.metodo, traducir(claves[1]));
  assert.equal(datos.perfil.identificador_visible, traducir(claves[2]));
  const respaldo = await import("./i18n.js?prueba-respaldo-mi-bolsa");
  for (const clave of claves) assert.equal(respaldo.traducir(clave), catalogo[clave]);
});
