import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { MENSAJES_DIETAS, crearTraductorDietas, prepararTextosDietas } from "./i18n.js?v=20260929-i18n-dietas-v1";
import { prepararTextosDietas as prepararDesdeVista } from "./vista-recorridos.js";
import { MENSAJES_BORRADORES } from "./i18n-borradores.js?v=20260929-i18n-dietas-v1";
import { MENSAJES_OTROS_GASTOS } from "./i18n-otros-gastos.js?v=20260929-i18n-dietas-v1";

const es = JSON.parse(await readFile(new URL("../../../textos/es/dietas.json", import.meta.url), "utf8"));
const en = JSON.parse(await readFile(new URL("../../../textos/en/dietas.json", import.meta.url), "utf8"));
const textos = (catalogo, idioma, incidenciaCatalogo = null) => ({
  idioma, localizacion: idioma === "es" ? "es-ES" : "en-GB", incidenciaCatalogo,
  incidenciaIndice: null, seccion: (nombre) => catalogo[nombre],
});

test("importar Dietas no prepara ni lee sus secciones", async () => {
  assert.equal(prepararDesdeVista, prepararTextosDietas);
  assert.equal(MENSAJES_DIETAS, undefined);
  assert.equal(MENSAJES_BORRADORES, undefined);
  assert.throws(() => crearTraductorDietas(), /incompleto/u);
  for (const nombre of ["i18n.js", "i18n-borradores.js", "i18n-revision.js", "i18n-circuito.js",
    "i18n-rectificacion-dietas.js", "i18n-rectificacion-admin.js", "i18n-otros-gastos.js"]) {
    const fuente = await readFile(new URL(nombre, import.meta.url), "utf8");
    assert.doesNotMatch(fuente, /\bawait\s+cargarTextos\s*\(/u, nombre);
  }
});

test("prepara un idioma, falla cerrado y delega el reintento al lector común", async () => {
  const llamadas = [];
  const estado = await prepararTextosDietas({ cargar: async (modulo) => {
    llamadas.push(modulo); return textos(es, "es");
  } });
  assert.deepEqual(llamadas, ["dietas"]);
  assert.equal(estado.idioma, "es");
  assert.equal(estado.localizacion, "es-ES");
  assert.equal(MENSAJES_BORRADORES.borradores_propios_titulo_registrados,
    es.borradores.borradores_propios_titulo_registrados);
  assert.equal(MENSAJES_DIETAS.titulo, es.general.titulo);
  const anterior = crearTraductorDietas();
  const incompleto = { ...en, circuito: { ...en.circuito, circuito_recibo: "" } };
  await assert.rejects(prepararTextosDietas({ cargar: async () => textos(incompleto, "en") }), /incompleto/u);
  assert.equal(MENSAJES_DIETAS, undefined);
  assert.equal(MENSAJES_OTROS_GASTOS, undefined);
  assert.throws(() => anterior("titulo"), /sin preparar/u);
  const incidencia = Object.freeze({ codigo: "catalogo_no_disponible", idioma: "es" });
  const recuperado = await prepararTextosDietas({ reintentar: true,
    cargar: () => { throw new Error("la lectura normal no debe ejecutarse"); },
    releer: async (modulo) => { llamadas.push(`reintento:${modulo}`); return textos(en, "en", incidencia); },
  });
  assert.deepEqual(llamadas, ["dietas", "reintento:dietas"]);
  assert.equal(recuperado.idioma, "en");
  assert.equal(recuperado.incidenciaCatalogo, incidencia);
  assert.equal(crearTraductorDietas()("titulo"), en.general.titulo);
  assert.equal(MENSAJES_OTROS_GASTOS.otros_gastos_tipo_desconocido, en.otros_gastos.otros_gastos_tipo_desconocido);
});

test("una lectura anterior no reemplaza el idioma de la vista más reciente", async () => {
  let completar;
  const lenta = prepararTextosDietas({ cargar: () => new Promise((resolver) => { completar = resolver; }) });
  await prepararTextosDietas({ cargar: async () => textos(en, "en") });
  completar(textos(es, "es"));
  await assert.rejects(lenta, /sustituida/u);
  assert.equal(MENSAJES_DIETAS.titulo, en.general.titulo);
});
