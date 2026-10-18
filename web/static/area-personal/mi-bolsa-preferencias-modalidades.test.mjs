import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { cargarPreferenciasModalidades, crearControladorPreferenciasModalidades, guardarPreferenciasModalidades, renderizarPreferenciasModalidades, validarEstadoPreferenciasModalidades } from "./mi-bolsa-preferencias-modalidades.js";
import { consultarTodasOfertasMiBolsa, datosMinimosMiBolsa, montarPreferenciasModalidadesSiAplica } from "./aplicacion.js";
import { renderizarLlamamientos, renderizarSeccionOfertasMiBolsa } from "./vistas/seguimiento-tramites.js";

const reglasCT = JSON.parse(await readFile(new URL("../../../data/catalogos/contratacion-temporal/reglas.v3.json", import.meta.url), "utf8"));
const entradasModalidad = reglasCT.catalogo.entradas.filter((entrada) => entrada.clave.startsWith("c12.modalidad."));
const fuente = (entrada) => ({ catalogo_id: reglasCT.catalogo.id, catalogo_version: reglasCT.catalogo.version,
  catalogo_huella_sha256: "a".repeat(64), entrada_clave: entrada.clave });
const opciones = entradasModalidad.map((entrada) => ({ clave: entrada.clave.slice("c12.modalidad.".length), fuente: fuente(entrada) }));
const estado = (seleccion = null, extra = {}) => ({ version: 1, modalidades_guardadas: seleccion, modalidades_disponibles: opciones, recibo_ref: null, registrado_en: null, ...extra });
const respuesta = (valor, status = 200) => ({ status, text: async () => JSON.stringify(valor) });
const tick = () => new Promise((resolver) => setImmediate(resolver));

function host() {
  const eventos = new Map();
  return { innerHTML: "", eventos,
    addEventListener(tipo, fn) { eventos.set(tipo, fn); }, removeEventListener(tipo) { eventos.delete(tipo); },
    querySelector() { return { focus() {} }; },
    pulsar(accion) { eventos.get("click")({ target: { closest: () => ({ dataset: { modalidades: accion } }) } }); },
    modo(valor) { eventos.get("change")({ target: { value: valor, hasAttribute: (nombre) => nombre === "data-modalidades-modo" } }); },
    opcion(clave, checked) { eventos.get("change")({ target: { dataset: { modalidadesOpcion: clave }, checked, hasAttribute: (nombre) => nombre === "data-modalidades-opcion" } }); },
  };
}

test("GET usa el selector exacto y exige catálogo gobernado conocido", async () => {
  assert.equal(opciones.length, 5);
  const llamadas = [];
  const datos = await cargarPreferenciasModalidades({ api: async (ruta, config) => { llamadas.push([ruta, config]); return respuesta(estado(null)); } });
  assert.equal(datos.modalidades_guardadas, null);
  assert.equal(llamadas[0][0], "/api/vec/bolsa/mi-bolsa/preferencias?tipo=modalidades");
  assert.equal(llamadas[0][1].method, "GET");
  assert.equal(llamadas[0][1].credentials, "same-origin");
  assert.equal(llamadas[0][1].cache, "no-store");
  assert.throws(() => validarEstadoPreferenciasModalidades(estado(null, { modalidades_disponibles: [...opciones, { clave: "ajena", fuente: fuente({ clave: "c12.modalidad.ajena" }) }] })), /respuesta/u);
  assert.throws(() => validarEstadoPreferenciasModalidades(estado(null, { modalidades_disponibles: [{ ...opciones[0], fuente: fuente(entradasModalidad[1]) }, ...opciones.slice(1)] })), /respuesta/u);
  assert.throws(() => validarEstadoPreferenciasModalidades(estado(null, { modalidades_disponibles: [{ ...opciones[0], fuente: { ...opciones[0].fuente, catalogo_id: "otro.catalogo" } }, ...opciones.slice(1)] })), /respuesta/u);
  assert.throws(() => validarEstadoPreferenciasModalidades(estado(["ajena"])), /respuesta/u);
  assert.throws(() => validarEstadoPreferenciasModalidades(estado(null, { persona_ref: "ajena" })), /respuesta/u);
  await assert.rejects(cargarPreferenciasModalidades({ api: async () => ({ status: 403 }) }), (error) => error.codigo === "denegado");
});

test("null muestra todas, el vacío explícito avisa antes de guardar y los textos no revelan claves", () => {
  const todas = renderizarPreferenciasModalidades(estado(null), { estado: "correcto", borrador: null });
  assert.match(todas, /Todas las ofertas/u);
  assert.match(todas, /<details /u);
  assert.doesNotMatch(todas, /data-modalidades-opcion/u);
  const ninguna = renderizarPreferenciasModalidades(estado(null), { estado: "correcto", borrador: [], abierto: true });
  assert.match(ninguna, /no mostrará ofertas publicadas ni recibirá sus avisos/u);
  assert.match(ninguna, /data-modalidades-opcion="sustitucion"/u);
  assert.match(ninguna, /data-modalidades="guardar"/u);
  assert.doesNotMatch(ninguna, /ct\.modalidad|catalogo_huella_sha256/u);
  assert.match(renderizarPreferenciasModalidades(estado([]), { estado: "correcto", borrador: [] }), /data-modalidades="guardar" disabled/u);
  const sinPreferencias = renderizarPreferenciasModalidades(null, { estado: "servicio" });
  assert.match(sinPreferencias, /data-modalidades="ver-todas"/u);
  assert.doesNotMatch(sinPreferencias, /data-modalidades="guardar"/u);
});

test("Ver todas solo llama al callback y deja intactos borrador y servidor", async () => {
  const solicitudes = [], vista = host(); let vistas = 0;
  const c = crearControladorPreferenciasModalidades({ host: vista, api: async (ruta, config) => { solicitudes.push(config.method); return respuesta(estado(["vacante"])); }, onVerTodas: () => { vistas++; } });
  c.montar(); await tick(); vista.modo("elegir"); vista.opcion("sustitucion", true);
  const antes = vista.innerHTML;
  vista.pulsar("ver-todas");
  await tick();
  assert.equal(vistas, 1); assert.deepEqual(solicitudes, ["GET"]);
  assert.match(vista.innerHTML, /data-modalidades-opcion="sustitucion" checked/u);
  assert.match(vista.innerHTML, /Consulta de todas las ofertas completada/u);
  assert.match(antes, /data-modalidades-opcion="sustitucion" checked/u);
  c.destruir();
});

test("Ver todas valida la respuesta completa y conserva la vista filtrada al fallar", async () => {
  const origen = { data: { esquema: "vec.bolsa.mi-bolsa.v1", consultada_en: "2026-10-18T10:00:00.000000Z", participaciones: [] } };
  let ruta, config;
  const todas = await consultarTodasOfertasMiBolsa(async (r, c) => { ruta = r; config = c; return respuesta(origen); });
  assert.equal(ruta, "/api/vec/bolsa/mi-bolsa?ofertas=todas");
  assert.equal(config.method, "GET"); assert.equal(config.credentials, "same-origin");
  assert.deepEqual(todas.ofertas, []);
  assert.match(renderizarSeccionOfertasMiBolsa(todas.ofertas, { mostrarVacio: true }), /No hay ofertas publicadas/u);
  assert.match(renderizarSeccionOfertasMiBolsa(null, { noDisponibles: true }), /Las ofertas no se pudieron cargar/u);
  assert.match(renderizarSeccionOfertasMiBolsa(null, { noDisponibles: true }), /data-accion="ver-todas-ofertas"/u);
  const parcial = renderizarLlamamientos(datosMinimosMiBolsa({ consultada_en: "2026-10-18T10:00:00.000000Z" }), { ofertasNoDisponibles: true });
  assert.match(parcial, /id="historial-mi-bolsa"/u);
  assert.match(parcial, /Las ofertas no se pudieron cargar/u);
  assert.doesNotMatch(parcial, /No hay ofertas publicadas/u);
  await assert.rejects(consultarTodasOfertasMiBolsa(async () => ({ status: 403 })), (e) => e.codigo === "verTodasDenegado");
  await assert.rejects(consultarTodasOfertasMiBolsa(async () => respuesta({ data: { ofertas: [] } })), (e) => e.codigo === "verTodasError");
  await assert.rejects(consultarTodasOfertasMiBolsa(async () => respuesta({ ...origen, data: { ...origen.data, ofertas_no_disponibles: true } })), (e) => e.codigo === "verTodasError");
});

test("el panel solo se monta con la marca del portal externo", () => {
  const datos = datosMinimosMiBolsa({ consultada_en: "2026-10-18T10:00:00.000000Z" });
  const antigua = renderizarLlamamientos(datos, {});
  assert.doesNotMatch(antigua, /id="preferencias-modalidades"/u);
  assert.match(antigua, /id="historial-mi-bolsa"/u);
  const nueva = renderizarLlamamientos(datos, { ofertasSegunModalidades: true });
  assert.match(nueva, /id="preferencias-modalidades"/u);
  let montajes = 0, consultas = 0;
  const fabrica = () => { montajes++; return { montar() { consultas++; } }; };
  const estadoBase = { fetchImpl: async () => { consultas++; }, vista: "llamamientos", generacionVista: 1 };
  assert.equal(montarPreferenciasModalidadesSiAplica(estadoBase, {}, fabrica), null);
  assert.equal(montajes, 0); assert.equal(consultas, 0);
  const montado = montarPreferenciasModalidadesSiAplica({ ...estadoBase, ofertasSegunModalidades: true }, {}, fabrica);
  assert.ok(montado); assert.equal(montajes, 1); assert.equal(consultas, 1);
});

test("el callback fallido se explica dentro del panel sin cambiar la elección", async () => {
  const vista = host();
  const c = crearControladorPreferenciasModalidades({ host: vista, api: async () => respuesta(estado(["vacante"])),
    onVerTodas: async () => { throw Object.assign(new Error("denegada"), { codigo: "verTodasDenegado" }); } });
  c.montar(); await tick(); vista.pulsar("ver-todas"); await tick();
  assert.match(vista.innerHTML, /No tiene permiso para ver todas las ofertas/u);
  assert.match(vista.innerHTML, /data-modalidades-opcion="vacante" checked/u);
  c.destruir();
});

test("resultado incierto repite mismo cuerpo y clave sin segundo PUT simultáneo", async () => {
  const cuerpos = []; let completar;
  const vista = host();
  const c = crearControladorPreferenciasModalidades({ host: vista, api: async (_ruta, config) => {
    if (config.method === "GET") return respuesta(estado(null));
    cuerpos.push(JSON.parse(config.body));
    if (cuerpos.length === 1) throw new TypeError("red");
    return new Promise((resolver) => { completar = resolver; });
  } });
  c.montar(); await tick(); vista.modo("elegir"); vista.pulsar("guardar"); await tick();
  assert.match(vista.innerHTML, /No se pudo confirmar/u);
  vista.pulsar("reintentar"); vista.pulsar("reintentar");
  assert.equal(cuerpos.length, 2); assert.deepEqual(cuerpos[0], cuerpos[1]);
  assert.deepEqual(cuerpos[0].modalidades, []);
  completar(respuesta({ recibo_ref: "recibo:1", version: 2, modalidades: [], registrado_en: "2026-10-18T10:00:00.000000Z", reutilizado: true }));
  await tick(); assert.match(vista.innerHTML, /Elección guardada/u); assert.doesNotMatch(vista.innerHTML, /recibo:1/u);
  c.destruir();
});

test("destruir aborta GET y evita una respuesta tardía; errores bloquean el guardado", async () => {
  let completar, signal;
  const vista = host();
  const c = crearControladorPreferenciasModalidades({ host: vista, api: (_ruta, config) => { signal = config.signal; return new Promise((resolver) => { completar = resolver; }); } });
  c.montar(); c.destruir(); assert.equal(signal.aborted, true); vista.innerHTML = "otra vista";
  completar(respuesta(estado(null))); await tick(); assert.equal(vista.innerHTML, "otra vista"); assert.equal(vista.eventos.size, 0);
  assert.match(renderizarPreferenciasModalidades(estado(null), { estado: "servicio", borrador: [] }), /data-modalidades="guardar" disabled/u);
  await assert.rejects(guardarPreferenciasModalidades({ modalidades: [], version_esperada: 1, clave_idempotencia: "clave-12345678" }, { api: async () => ({ status: 409 }) }), (error) => error.codigo === "conflicto");
});

test("los catálogos español e inglés cubren las opciones y mensajes", async () => {
  for (const idioma of ["es", "en"]) {
    const catalogo = JSON.parse(await readFile(new URL(`../textos/${idioma}/area-personal.json`, import.meta.url), "utf8"));
    const textos = catalogo.miBolsa.preferenciasModalidades;
    for (const clave of ["titulo", "cargando", "sinConsulta", "todas", "elegir", "avisoNinguna", "guardar", "verTodas", "servicio", "incierto", "conflicto", "reintentarGuardado"]) assert.equal(typeof textos[clave], "string");
    for (const { clave } of opciones) assert.equal(typeof textos.opciones[clave], "string");
  }
});
