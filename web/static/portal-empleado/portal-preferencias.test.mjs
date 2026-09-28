import assert from "node:assert/strict";
import test from "node:test";
import { crearClientePreferencias, ErrorPreferencias } from "./portal-preferencias-api.js";
import { crearSuperficiePreferenciasPortal } from "./portal-preferencias.js";

const valores = Object.freeze({ idioma: "es", tamano_texto: "normal", alto_contraste: false,
  tema: "sistema", inicio: "cuadro", filas: 20, aviso_correo_tareas: false, aviso_correo_plazos: false });
const catalogo = Object.freeze({ version_ref: "usuarios-preferencias-v1",
  idiomas: ["navegador", "es", "en"].map((codigo) => ({ codigo, nombre_key: `ui.usuarios.preferencias.idioma.${codigo}` })),
  tamanos_texto: ["normal", "grande", "muy_grande"].map((codigo) => ({ codigo, nombre_key: `ui.usuarios.preferencias.tamano_texto.${codigo}` })),
  temas: ["sistema", "claro", "oscuro"].map((codigo) => ({ codigo, nombre_key: `ui.usuarios.preferencias.tema.${codigo}` })),
  inicios: ["cuadro", "peticiones", "bolsas"].map((codigo) => ({ codigo, nombre_key: `ui.usuarios.preferencias.inicio.${codigo}` })),
  filas: [20, 50, 100], predeterminados: valores });
const get = { data: { catalogo, estado: { version: 0, catalogo_version_ref: catalogo.version_ref, valores } } };
const respuesta = (json, status = 200) => new Response(JSON.stringify(json), { status, headers: { "Content-Type": "application/json" } });

test("GET conserva la versión cero y el catálogo servidor sin crear ni escribir", async () => {
  const peticiones = [];
  const cliente = crearClientePreferencias({ fetchImpl: async (ruta, opciones) => {
    peticiones.push({ ruta, opciones }); return respuesta(get);
  } });
  const resultado = await cliente.consultar();
  assert.equal(resultado.estado.version, 0);
  assert.equal(resultado.catalogo.inicios[1].codigo, "peticiones");
  assert.equal(peticiones[0].ruta, "/api/vec/usuarios/mis-preferencias");
  assert.equal(peticiones[0].opciones.method, "GET");
  assert.equal(peticiones[0].opciones.credentials, "same-origin");
  assert.equal(peticiones[0].opciones.body, undefined);
});

test("PUT envía solo CAS, catálogo, clave y valores, y conserva el recibo", async () => {
  let cuerpo;
  const cliente = crearClientePreferencias({ fetchImpl: async (_ruta, opciones) => {
    cuerpo = JSON.parse(opciones.body);
    return respuesta({ data: { recibo_ref: "recibo:abc", version: 1,
      catalogo_version_ref: catalogo.version_ref, valores, fecha_utc: "2026-09-29T10:00:00Z", replay: false } }, 201);
  } });
  const resultado = await cliente.guardar({ version: 0, catalogoVersion: catalogo.version_ref,
    clave: "abc123456789012345", valores });
  assert.deepEqual(Object.keys(cuerpo).sort(), ["version_esperada", "catalogo_version_ref", "clave_operacion", "valores"].sort());
  assert.equal(cuerpo.persona_ref, undefined);
  assert.equal(resultado.recibo_ref, "recibo:abc");
});

test("errores 401/403/409/422/503 no exponen valores ni admiten envoltura alternativa", async () => {
  for (const [status, codigo] of [[401, "no_autenticado"], [403, "prohibido"], [409, "conflicto"], [422, "peticion_invalida"], [503, "no_disponible"]]) {
    const cliente = crearClientePreferencias({ fetchImpl: async () => respuesta({ error: { codigo, clave_i18n: `api.usuarios.preferencias.error.${codigo}` } }, status) });
    await assert.rejects(cliente.consultar(), (error) => error instanceof ErrorPreferencias && error.estado === status && error.codigo === codigo);
  }
  const alternativo = crearClientePreferencias({ fetchImpl: async () => respuesta(get.data) });
  await assert.rejects(alternativo.consultar(), /catálogo de preferencias inválido/u);
});

test("la vista usa únicamente opciones del catálogo y no finge datos si GET falla", async () => {
  const sinAcceso = crearSuperficiePreferenciasPortal({ cliente: { consultar: async () => { throw new ErrorPreferencias(403, "prohibido"); } } });
  await sinAcceso.cargar();
  assert.match(sinAcceso.renderizar(), /No tiene permiso/u);
  assert.doesNotMatch(sinAcceso.renderizar(), /formulario-preferencias|Filas por página/u);
  const recortado = { ...catalogo, filas: [20], inicios: catalogo.inicios.slice(0, 1) };
  const vista = crearSuperficiePreferenciasPortal({ cliente: { consultar: async () => ({ catalogo: recortado, estado: get.data.estado }) } });
  await vista.cargar();
  const html = vista.renderizar();
  assert.match(html, /formulario-preferencias/u);
  assert.match(html, /value="20"/u);
  assert.doesNotMatch(html, /value="50"|value="peticiones"/u);
  assert.match(html, /data-pref-ayuda="filas"[^>]*aria-expanded="false"/u);
});
