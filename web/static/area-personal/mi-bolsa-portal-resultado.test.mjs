import assert from "node:assert/strict";
import test from "node:test";
import { iniciarI18nAreaPersonal, traducir } from "./i18n.js";
import { cuerpoPortalMiBolsa, enviarPortalMiBolsa, renderizarPortalMiBolsa } from "./mi-bolsa-portal.js";
import { catalogoPlano, lectorCatalogos } from "./textos-prueba.test-helper.mjs";

test("Mi bolsa explica el resultado incierto y el límite del justificante en ambos idiomas", async (t) => {
  for (const idioma of ["es", "en"]) {
    const documento = { documentElement: { lang: idioma }, querySelectorAll: () => [] };
    await iniciarI18nAreaPersonal(documento, {
      leer: lectorCatalogos(), ubicacion: { href: `https://vec.example/area-personal/?lang=${idioma}` },
    });
    const catalogo = await catalogoPlano(idioma);

    await t.test(`${idioma}: perder la respuesta no afirma que la petición no se registró`, async () => {
      const zona = { textContent: "" };
      const boton = { disabled: false };
      const formulario = {
        dataset: { portalMiBolsa: "solicitar", tipo: "reactivacion", bolsa: "bolsa:demo:1" },
        querySelector: (selector) => selector === "[data-portal-resultado]" ? zona : boton,
      };
      const enviadas = [];
      let confirmaciones = 0;
      const dependencias = {
        datos: new FormData(), alRegistrar: () => confirmaciones++,
        fetchImpl: async (_ruta, opciones) => {
          enviadas.push(JSON.parse(opciones.body));
          throw new TypeError("Respuesta perdida después de emitir la petición");
        },
      };
      assert.equal(await enviarPortalMiBolsa(formulario, dependencias), false);
      assert.equal(zona.textContent, catalogo["areaPersonal.portal.error.servicio_no_disponible"]);
      assert.match(zona.textContent, idioma === "es" ? /confirmar el resultado.*Consulte Mi bolsa/u : /outcome.*confirmed.*Check My job pool/u);
      assert.equal(confirmaciones, 0);
      assert.equal(boton.disabled, false);
      assert.equal(enviadas.length, 1, "no repite la petición automáticamente");
      assert.equal(await enviarPortalMiBolsa(formulario, dependencias), false);
      assert.deepEqual(enviadas[1], enviadas[0], "un reintento explícito conserva la petición y su clave");
    });

    await t.test(`${idioma}: la etiqueta explica que el fichero no se envía y solo salen referencia y huella`, async () => {
      const html = renderizarPortalMiBolsa(
        [{ bolsa: "bolsa:demo:1", categoria: "Auxiliar" }],
        [{ bolsa: "bolsa:demo:1", llamamiento_abierto: {
          contacto_en: "2026-10-01T08:00:00Z", vence_antes_de: "2026-10-02T21:59:59Z",
        } }],
        { pausa_maxima: "2027-10-01T21:59:59Z", causas_renuncia: ["enfermedad"], modo_respuesta: "propuesta_rrhh" },
      );
      const etiqueta = traducir("areaPersonal.portal.justificante");
      assert.equal(etiqueta, catalogo["areaPersonal.portal.justificante"]);
      assert.ok(html.includes(`>${etiqueta}</label>`));
      assert.match(etiqueta, idioma === "es" ? /el archivo no se envía/u : /the file is not sent/u);
      const datos = new FormData();
      datos.set("respuesta", "renuncia_justificada");
      datos.set("causa", "enfermedad");
      datos.set("justificante_ref", "documento:demo:1");
      datos.set("justificante", new Blob(["abc"]));
      const { cuerpo } = await cuerpoPortalMiBolsa({
        dataset: { portalMiBolsa: "responder", bolsa: "bolsa:demo:1" },
      }, datos);
      assert.deepEqual(Object.keys(cuerpo).sort(), ["bolsa", "causa", "clave", "justificante_ref", "justificante_sha256", "respuesta"]);
      assert.equal(cuerpo.justificante_ref, "documento:demo:1");
      assert.equal(cuerpo.justificante_sha256, "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad");
    });
  }
});
