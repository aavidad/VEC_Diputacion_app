import assert from "node:assert/strict";
import test from "node:test";
import { crearClientePreferencias, ErrorPreferencias } from "./portal-preferencias-api.js";
import { crearSuperficiePreferenciasPortal } from "./portal-preferencias.js?v=20261001-ct-a-i18n-v1";
import { peticionesEnSerie } from "../comun/imagen-propia.js";

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

test("dos vistas comparten GET en vuelo y una cancelación no corta la otra", async () => {
  let resolver;
  let llamadas = 0;
  const cliente = crearClientePreferencias({ fetchImpl: () => {
    llamadas++;
    return new Promise((resolve) => { resolver = resolve; });
  } });
  const una = new AbortController();
  const otra = new AbortController();
  const primera = cliente.consultar({ signal: una.signal });
  const segunda = cliente.consultar({ signal: otra.signal });
  await Promise.resolve();
  assert.equal(llamadas, 1);
  una.abort();
  await assert.rejects(primera);
  resolver(respuesta(get));
  assert.equal((await segunda).estado.version, 0);
  assert.equal(llamadas, 1);
});

test("GET recupera después de una respuesta 503 grande sin romper la cola de Usuarios", async () => {
  let llamadas = 0;
  const enSerie = peticionesEnSerie(async () => {
    llamadas++;
    return llamadas === 1
      ? new Response("x".repeat(70 * 1024), { status: 503, headers: { "Content-Type": "application/json" } })
      : respuesta(get);
  });
  const cliente = crearClientePreferencias({ fetchImpl: enSerie });
  assert.equal((await cliente.consultar()).estado.version, 0);
  assert.equal(llamadas, 2);
});

test("v2 permite los seis temas; v1 rechaza un tema adelantado", async () => {
  const temasNuevos = ["diputacion_granada", "arena", "salvia", "lavanda", "azul_sereno", "noche_suave"];
  const catalogoV2 = { ...catalogo, version_ref: "usuarios-preferencias-v2",
    temas: [...catalogo.temas, ...temasNuevos.map((codigo) => ({ codigo, nombre_key: `ui.usuarios.preferencias.tema.${codigo}` }))] };
  for (const tema of temasNuevos) {
    const elegidos = { ...valores, tema };
    const cliente = crearClientePreferencias({ fetchImpl: async (_ruta, opciones) => opciones.method === "GET"
      ? respuesta({ data: { catalogo: catalogoV2, estado: { version: 0, catalogo_version_ref: catalogoV2.version_ref, valores: elegidos } } })
      : respuesta({ data: { recibo_ref: "recibo:tema", version: 1, catalogo_version_ref: catalogoV2.version_ref,
        valores: elegidos, fecha_utc: "2026-09-30T00:00:00Z" } }, 201) });
    assert.equal((await cliente.consultar()).estado.valores.tema, tema);
    assert.equal((await cliente.guardar({ version: 0, catalogoVersion: catalogoV2.version_ref,
      clave: "tema-123456789012345", valores: elegidos })).valores.tema, tema);
    await assert.rejects(cliente.guardar({ version: 0, catalogoVersion: catalogo.version_ref,
      clave: "tema-123456789012345", valores: elegidos }), /valores de preferencias inválidos/u);
  }
  const fueraDeCatalogo = { ...catalogoV2, temas: catalogoV2.temas.slice(0, -1) };
  const cliente = crearClientePreferencias({ fetchImpl: async () => respuesta({ data: { catalogo: fueraDeCatalogo,
    estado: { version: 0, catalogo_version_ref: fueraDeCatalogo.version_ref, valores: { ...valores, tema: "noche_suave" } } } }) });
  await assert.rejects(cliente.consultar(), /estado de preferencias inválido|valores de preferencias inválidos/u);
});

test("GET lee estado histórico v1 con catálogo actual v2 y guarda con v2", async () => {
  const catalogoV2 = { ...catalogo, version_ref: "usuarios-preferencias-v2",
    temas: [...catalogo.temas, { codigo: "salvia", nombre_key: "ui.usuarios.preferencias.tema.salvia" }] };
  const estadoHistorico = { version: 4, catalogo_version_ref: "usuarios-preferencias-v1", valores };
  const llamadas = [];
  const cliente = crearClientePreferencias({ fetchImpl: async (_ruta, opciones) => {
    llamadas.push(opciones);
    return opciones.method === "GET" ? respuesta({ data: { catalogo: catalogoV2, estado: estadoHistorico } })
      : respuesta({ data: { recibo_ref: "recibo:actualizado", version: 5,
        catalogo_version_ref: "usuarios-preferencias-v2", valores: { ...valores, tema: "salvia" },
        fecha_utc: "2026-09-30T00:00:00Z" } }, 201);
  } });
  const leido = await cliente.consultar();
  assert.equal(leido.catalogo.version_ref, "usuarios-preferencias-v2");
  assert.equal(leido.estado.version, 4);
  const nuevo = { ...leido.estado.valores, tema: "salvia" };
  await cliente.guardar({ version: leido.estado.version, catalogoVersion: leido.catalogo.version_ref,
    clave: "tema-123456789012345", valores: nuevo });
  assert.equal(JSON.parse(llamadas[1].body).catalogo_version_ref, "usuarios-preferencias-v2");
  const adelantado = crearClientePreferencias({ fetchImpl: async () => respuesta({ data: { catalogo,
    estado: { ...estadoHistorico, catalogo_version_ref: "usuarios-preferencias-v2" } } }) });
  await assert.rejects(adelantado.consultar(), /estado de preferencias inválido/u);
  const falsoHistorico = crearClientePreferencias({ fetchImpl: async () => respuesta({ data: { catalogo: catalogoV2,
    estado: { ...estadoHistorico, valores: nuevo } } }) });
  await assert.rejects(falsoHistorico.consultar(), /valores de preferencias inválidos/u);
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
  assert.match(sinAcceso.renderizar(), /no tiene permiso/u);
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

test("un PUT 503 conserva el borrador, exige nueva consulta y enfoca el resultado", async () => {
  const formulario = { id: "formulario-preferencias", valores: { ...valores, tema: "oscuro" } };
  const eventos = {};
  let html = "";
  let focos = 0;
  let consultas = 0;
  let escrituras = 0;
  let versionEnviada = -1;
  const contenedor = {
    addEventListener: (nombre, funcion) => { eventos[nombre] = funcion; },
    removeEventListener: () => {},
    querySelector: (selector) => selector === "[data-pref-resultado]" && html.includes("data-pref-resultado")
      ? { focus: () => { focos++; } } : null,
  };
  const cliente = {
    consultar: async () => {
      consultas++;
      return { catalogo, estado: { ...get.data.estado, version: consultas === 1 ? 0 : 1 } };
    },
    guardar: async ({ version, valores: elegidos }) => {
      escrituras++;
      versionEnviada = version;
      assert.equal(elegidos.tema, "oscuro");
      if (escrituras === 1) throw new ErrorPreferencias(503, "no_disponible");
      return { recibo_ref: "recibo:confirmado", version: 2, valores: elegidos };
    },
  };
  const originalFormData = globalThis.FormData;
  globalThis.FormData = class { constructor(form) { this.form = form; } get(campo) { return String(this.form.valores[campo]); } };
  try {
    const superficie = crearSuperficiePreferenciasPortal({ cliente, actualizar: () => { html = superficie.renderizar(); } });
    const desmontar = superficie.instalar(contenedor);
    await superficie.cargar();
    const enviar = () => eventos.submit({ target: formulario, preventDefault() {} });
    enviar();
    await new Promise((resolver) => setImmediate(resolver));
    assert.equal(escrituras, 1);
    assert.equal(superficie.leerCarga(), "guardado_incierto");
    assert.match(html, /No se pudo confirmar el guardado.*operación podría haberse aplicado/u);
    assert.match(html, /id="pref-tema"[^>]*>[\s\S]*?<option value="oscuro" selected/u);
    assert.match(html, /type="submit" disabled/u);
    assert.equal(focos, 2);
    enviar();
    await new Promise((resolver) => setImmediate(resolver));
    assert.equal(escrituras, 1);
    await superficie.cargar({ enfocar: true });
    assert.equal(consultas, 2);
    assert.equal(focos, 4);
    assert.match(html, /id="pref-tema"[^>]*>[\s\S]*?<option value="oscuro" selected/u);
    enviar();
    await new Promise((resolver) => setImmediate(resolver));
    assert.equal(escrituras, 2);
    assert.equal(versionEnviada, 1);
    assert.match(html, /pref-aviso--exito[^>]*>Preferencias guardadas\./u);
    assert.doesNotMatch(html, /recibo:confirmado/u);
    assert.equal(focos, 6);
    desmontar();
  } finally { globalThis.FormData = originalFormData; }
});

test("los errores de sesión, permiso, conflicto y validación mantienen avisos específicos", async () => {
  const originalFormData = globalThis.FormData;
  globalThis.FormData = class { get(campo) { return String(valores[campo]); } };
  try {
    for (const [estado, frase] of [[401, "sesión ha caducado"], [403, "no tiene permiso"],
      [409, "cambiaron en otra sesión"], [422, "guardar una de las opciones"]]) {
      let html = "";
      let enviar;
      const contenedor = { addEventListener: (nombre, funcion) => { if (nombre === "submit") enviar = funcion; },
        removeEventListener: () => {}, querySelector: () => null };
      const superficie = crearSuperficiePreferenciasPortal({
        cliente: { consultar: async () => get.data, guardar: async () => { throw new ErrorPreferencias(estado, "no_disponible"); } },
        actualizar: () => { html = superficie.renderizar(); },
      });
      superficie.instalar(contenedor);
      await superficie.cargar();
      enviar({ target: { id: "formulario-preferencias" }, preventDefault() {} });
      await new Promise((resolver) => setImmediate(resolver));
      assert.match(html, new RegExp(frase, "u"));
      assert.doesNotMatch(html, /No se pudo confirmar el guardado/u);
    }
  } finally { globalThis.FormData = originalFormData; }
});
