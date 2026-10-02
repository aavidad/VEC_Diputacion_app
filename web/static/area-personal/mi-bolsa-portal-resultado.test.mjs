import assert from "node:assert/strict";
import test from "node:test";
import { iniciarI18nAreaPersonal, traducir } from "./i18n.js";
import { cuerpoPortalMiBolsa, enviarPortalMiBolsa, renderizarPortalMiBolsa } from "./mi-bolsa-portal.js";
import { catalogoPlano, lectorCatalogos } from "./textos-prueba.test-helper.mjs";

const huella = "a".repeat(64);
const oferta = `oferta:${huella}`;
const bolsa = "bolsa:demo:1";
const escenarios = [
  { accion: "responder", dataset: { bolsa }, tipo: "respuesta", referencia: `respuesta-portal:${huella}`, prefijo: "respuesta-portal", estado: "propuesta_rrhh" },
  { accion: "disposicion", dataset: { oferta }, tipo: "disposicion", referencia: oferta, prefijo: "disposicion", estado: "manifestada" },
  { accion: "contacto", dataset: { bolsa, version: "3" }, tipo: "contacto", referencia: bolsa, prefijo: "confirmacion-contacto", estado: "confirmado" },
];

function reciboDe(escenario, repetida = false) {
  return { data: {
    esquema: `vec.bolsa.mi-bolsa.${escenario.tipo}.v1`, referencia: escenario.referencia,
    recibo: `recibo:${escenario.prefijo}:${huella}`, registrada_en: "2026-10-01T08:00:00.000000Z",
    estado: escenario.estado, repetida,
  } };
}

function formularioDe(escenario) {
  const zona = { textContent: "" };
  const boton = { disabled: false };
  return { dataset: { portalMiBolsa: escenario.accion, ...escenario.dataset }, zona, boton,
    querySelector: (selector) => selector === "[data-portal-resultado]" ? zona : boton };
}

function datosDe() {
  const datos = new FormData();
  datos.set("respuesta", "acepta");
  return datos;
}

test("solo confirma los tres recibos completos y su recuperación HTTP", async (t) => {
  for (const escenario of escenarios) {
    for (const status of [201, 200]) await t.test(`${escenario.accion}: ${status}`, async () => {
      const formulario = formularioDe(escenario);
      let confirmaciones = 0;
      const recibo = reciboDe(escenario, status === 200);
      assert.equal(await enviarPortalMiBolsa(formulario, { datos: datosDe(), alRegistrar: () => confirmaciones++,
        fetchImpl: async () => ({ status, json: async () => recibo }) }), true);
      assert.equal(confirmaciones, 1);
      assert.ok(formulario.zona.textContent.includes(recibo.data.recibo));
      assert.equal(formulario.boton.disabled, false);
    });
  }
});

test("un 200/201 sin recibo válido queda incierto en los dos idiomas sin recargar ni reenviar", async (t) => {
  for (const idioma of ["es", "en"]) {
    await iniciarI18nAreaPersonal({ documentElement: {}, querySelectorAll: () => [] }, {
      leer: lectorCatalogos(), ubicacion: { href: `https://vec.example/area-personal/?lang=${idioma}` },
    });
    const catalogo = await catalogoPlano(idioma);
    for (const escenario of escenarios) {
      const valido = reciboDe(escenario);
      const invalidos = [null, {}, { data: [] }, { data: { recibo: valido.data.recibo } },
        ...["esquema", "referencia", "recibo", "registrada_en", "estado", "repetida"].map((campo) => {
          const copia = structuredClone(valido); delete copia.data[campo]; return copia;
        }),
        { data: { ...valido.data, recibo: " " } }, { data: { ...valido.data, recibo: 42 } },
        { data: { ...valido.data, esquema: "vec.bolsa.area-personal.recibo.v1" } },
        { data: { ...valido.data, referencia: escenario.accion === "disposicion" ? `oferta:${"b".repeat(64)}` : "bolsa:ajena" } },
        { data: { ...valido.data, estado: "desconocido" } },
        { data: { ...valido.data, registrada_en: "fecha no válida" } },
        { data: { ...valido.data, repetida: true } },
        { data: { ...valido.data, vence_antes_de: "fecha no válida" } },
      ];
      for (const [indice, invalido] of invalidos.entries()) await t.test(`${idioma}/${escenario.accion}/${indice}`, async () => {
        const formulario = formularioDe(escenario);
        let envios = 0, confirmaciones = 0;
        assert.equal(await enviarPortalMiBolsa(formulario, { datos: datosDe(), alRegistrar: () => confirmaciones++,
          fetchImpl: async () => { envios++; return { status: 201, json: async () => invalido }; } }), false);
        assert.equal(formulario.zona.textContent, catalogo["areaPersonal.portal.error.servicio_no_disponible"]);
        assert.equal(formulario.boton.disabled, false);
        assert.equal(confirmaciones, 0);
        assert.equal(envios, 1);
      });
    }
    const formulario = formularioDe(escenarios[0]);
    assert.equal(await enviarPortalMiBolsa(formulario, { datos: datosDe(), fetchImpl: async () => ({ status: 200,
      json: async () => { throw new SyntaxError(); } }) }), false);
    assert.equal(formulario.zona.textContent, catalogo["areaPersonal.portal.error.servicio_no_disponible"]);
  }
});

test("el reintento explícito conserva bolsa, versión y clave después de un recibo incompleto", async () => {
  const escenario = escenarios[2];
  const formulario = formularioDe(escenario);
  const enviadas = [];
  let confirmaciones = 0;
  const fetchImpl = async (_ruta, opciones) => {
    enviadas.push(opciones.body);
    return { status: enviadas.length === 1 ? 201 : 200,
      json: async () => enviadas.length === 1 ? {} : reciboDe(escenario, true) };
  };
  const dependencias = { datos: datosDe(), fetchImpl, alRegistrar: () => confirmaciones++ };
  assert.equal(await enviarPortalMiBolsa(formulario, dependencias), false);
  formulario.dataset.bolsa = "bolsa:otra";
  formulario.dataset.version = "9";
  assert.equal(await enviarPortalMiBolsa(formulario, dependencias), true);
  assert.equal(enviadas[1], enviadas[0]);
  assert.equal(JSON.parse(enviadas[1]).version, 3);
  assert.equal(confirmaciones, 1);
});

test("el recibo rechaza fechas civiles imposibles y horas normalizadas en sus dos instantes", async () => {
  for (const campo of ["registrada_en", "vence_antes_de"]) {
    for (const valor of ["2026-02-30T08:00:00Z", "2025-02-29T08:00:00Z", "2026-10-01T24:00:00Z", "2026-10-01T08:60:00Z", "2026-10-01T08:00:60Z"]) {
      const formulario = formularioDe(escenarios[0]);
      const recibo = reciboDe(escenarios[0]);
      recibo.data[campo] = valor;
      let confirmaciones = 0;
      assert.equal(await enviarPortalMiBolsa(formulario, { datos: datosDe(), alRegistrar: () => confirmaciones++,
        fetchImpl: async () => ({ status: 201, json: async () => recibo }) }), false, `${campo}: ${valor}`);
      assert.equal(confirmaciones, 0);
      assert.equal(formulario.zona.textContent, traducir("areaPersonal.portal.error.servicio_no_disponible"));
    }
  }
  const recibo = reciboDe(escenarios[0]);
  recibo.data.registrada_en = "2024-02-29T23:59:59.123456Z";
  recibo.data.vence_antes_de = "2024-03-01T23:59:59.999999Z";
  assert.equal(await enviarPortalMiBolsa(formularioDe(escenarios[0]), { datos: datosDe(),
    fetchImpl: async () => ({ status: 201, json: async () => recibo }) }), true);
});

test("un corte conserva respuesta y justificante originales sin un segundo envío concurrente", async () => {
  const formulario = formularioDe(escenarios[0]);
  const datos = new FormData();
  datos.set("respuesta", "renuncia_justificada"); datos.set("causa", "enfermedad");
  datos.set("justificante_ref", "documento:demo:1"); datos.set("justificante", new Blob(["abc"]));
  const enviadas = [];
  let terminar;
  const dependencias = { datos, fetchImpl: async (_ruta, opciones) => {
    enviadas.push(opciones.body);
    await new Promise((resolver) => { terminar = resolver; });
    throw new TypeError();
  } };
  const primera = enviarPortalMiBolsa(formulario, dependencias);
  assert.equal(await enviarPortalMiBolsa(formulario, dependencias), false);
  while (!terminar) await new Promise((resolver) => setImmediate(resolver));
  terminar();
  assert.equal(await primera, false);
  assert.equal(enviadas.length, 1);
  datos.set("respuesta", "acepta"); datos.set("justificante_ref", "documento:otro");
  assert.equal(await enviarPortalMiBolsa(formulario, { datos, fetchImpl: async (_ruta, opciones) => {
    enviadas.push(opciones.body); return { status: 200, json: async () => reciboDe(escenarios[0], true) };
  } }), true);
  assert.equal(enviadas[1], enviadas[0]);
});

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
        dataset: { portalMiBolsa: "responder", bolsa: "bolsa:demo:1" },
        querySelector: (selector) => selector === "[data-portal-resultado]" ? zona : boton,
      };
      const enviadas = [];
      let confirmaciones = 0;
      const dependencias = {
        datos: datosDe(), alRegistrar: () => confirmaciones++,
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
