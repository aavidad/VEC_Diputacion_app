import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import {
  cargarPreferenciasModalidades, crearControladorPreferenciasModalidades, guardarPreferenciasModalidades,
  modoEleccion, renderizarPreferenciasModalidades, validarEstadoPreferenciasModalidades,
} from "./mi-bolsa-preferencias-modalidades.js";
import { renderizarSeccionOfertasMiBolsa } from "./vistas/seguimiento-tramites.js";

const reglasCT = JSON.parse(await readFile(new URL("../../../data/catalogos/contratacion-temporal/reglas.v3.json", import.meta.url), "utf8"));
const entradas = reglasCT.catalogo.entradas.filter((e) => e.clave.startsWith("c12.modalidad."));
const fuente = (clave) => ({ catalogo_id: "vec.contratacion_temporal.reglas", catalogo_version: 3, catalogo_huella_sha256: "a".repeat(64), entrada_clave: `c12.modalidad.${clave}` });
const opciones = entradas.map((e) => e.clave.slice("c12.modalidad.".length)).map((clave) => ({ clave, fuente: fuente(clave) }));
const FECHA = "2026-10-20T09:15:00.000000Z";
const estado = (guardadas, extra = {}) => ({
  version: 1, modalidades_guardadas: guardadas, modalidades_disponibles: opciones,
  recibo_ref: "recibo:preferencias-modalidades:" + "b".repeat(64), registrado_en: FECHA, ...extra,
});
const inicial = () => ({ version: 0, modalidades_guardadas: null, modalidades_disponibles: opciones, recibo_ref: null, registrado_en: null });
const respuesta = (valor, status = 200) => ({ status, text: async () => JSON.stringify(valor) });
const tick = () => new Promise((r) => setImmediate(r));

function host() {
  const eventos = new Map();
  const h = {
    innerHTML: "", hidden: false, enfocado: "",
    addEventListener(tipo, fn) { eventos.set(tipo, fn); }, removeEventListener(tipo) { eventos.delete(tipo); },
    querySelector(selector) { return { focus() { h.enfocado = selector; } }; },
    pulsar(accion) { eventos.get("click")({ target: { closest: () => ({ dataset: { modalidades: accion }, disabled: false }) } }); },
    modo(valor) { eventos.get("change")({ target: { value: valor, hasAttribute: (n) => n === "data-modalidades-modo" } }); },
    opcion(clave, checked) { eventos.get("change")({ target: { dataset: { modalidadesOpcion: clave }, checked, hasAttribute: (n) => n === "data-modalidades-opcion" } }); },
    eventos,
  };
  return h;
}

test("las opciones del catálogo vigente tienen nombre en los dos idiomas", async () => {
  assert.ok(opciones.length >= 1);
  for (const idioma of ["es", "en"]) {
    const catalogo = JSON.parse(await readFile(new URL(`../textos/${idioma}/area-personal.json`, import.meta.url), "utf8"));
    const textos = catalogo.miBolsa.preferenciasModalidades;
    for (const { clave } of opciones) assert.equal(typeof textos.opciones[clave], "string", `${idioma}: ${clave}`);
    for (const clave of ["titulo", "pregunta", "avisoNinguna", "marqueUna", "guardar", "guardado", "ultimoGuardado", "conflicto", "incierto", "reintentarGuardado", "recargar"]) {
      assert.equal(typeof textos[clave], "string", `${idioma}: ${clave}`);
    }
    for (const clave of ["todas", "algunas", "ninguna"]) assert.equal(typeof textos.modo[clave], "string");
    for (const clave of ["verTodas", "verElegidas", "sinOfertas", "sinOfertasElegidas", "mostrandoTodas", "mostrandoElegidas", "cargandoOfertas", "errorOfertas"]) {
      assert.equal(typeof catalogo.ofertas[clave], "string", `${idioma}: ofertas.${clave}`);
    }
  }
});

test("GET usa la consulta exacta y rechaza opciones ajenas o sin nombre", async () => {
  const llamadas = [];
  const datos = await cargarPreferenciasModalidades({ api: async (ruta, config) => { llamadas.push([ruta, config]); return respuesta(inicial()); } });
  assert.equal(datos.modalidades_guardadas, null);
  assert.equal(llamadas[0][0], "/api/vec/bolsa/mi-bolsa/preferencias?tipo=modalidades");
  assert.equal(llamadas[0][1].method, "GET");
  assert.equal(llamadas[0][1].credentials, "same-origin");
  const sinNombre = { clave: "desconocida", fuente: fuente("desconocida") };
  assert.throws(() => validarEstadoPreferenciasModalidades(estado(null, { modalidades_disponibles: [...opciones, sinNombre] })), /respuesta/u);
  assert.throws(() => validarEstadoPreferenciasModalidades(estado(null, { modalidades_disponibles: [{ ...opciones[0], fuente: fuente("otra") }] })), /respuesta/u);
  assert.throws(() => validarEstadoPreferenciasModalidades(estado(null, { modalidades_disponibles: [] })), /respuesta/u);
  assert.throws(() => validarEstadoPreferenciasModalidades(estado(null, { persona: "x" })), /respuesta/u);
  await assert.rejects(cargarPreferenciasModalidades({ api: async () => ({ status: 403 }) }), (e) => e.codigo === "denegado");
  await assert.rejects(cargarPreferenciasModalidades({ api: async () => ({ status: 404 }) }), (e) => e.codigo === "noDisponible");
});

test("todas, algunas y ninguna se distinguen; «ninguna» avisa antes de guardar", () => {
  assert.equal(modoEleccion(null), "todas");
  assert.equal(modoEleccion([]), "ninguna");
  assert.equal(modoEleccion(["vacante"]), "algunas");
  const todas = renderizarPreferenciasModalidades(estado(null), { estado: "correcto", borrador: null });
  assert.match(todas, /value="todas" data-modalidades-modo checked/u);
  assert.doesNotMatch(todas, /data-modalidades-opcion/u);
  assert.match(todas, /data-modalidades="guardar" disabled/u);
  assert.match(todas, /Última elección guardada el/u);
  const ninguna = renderizarPreferenciasModalidades(estado(null), { estado: "correcto", borrador: null, modo: "ninguna" });
  assert.match(ninguna, /No verá ofertas nuevas en Mi bolsa ni recibirá avisos de ofertas/u);
  assert.match(ninguna, /le pueden seguir llamando/u);
  assert.match(ninguna, /aria-describedby="preferencias-modalidades-ninguna"/u);
  assert.doesNotMatch(ninguna, /data-modalidades="guardar" disabled/u);
  const algunas = renderizarPreferenciasModalidades(estado(null), { estado: "correcto", borrador: [], modo: "algunas" });
  assert.match(algunas, /data-modalidades-opcion="vacante"/u);
  assert.match(algunas, /Marque al menos un tipo/u);
  assert.match(algunas, /data-modalidades="guardar" disabled/u);
  assert.doesNotMatch(algunas, /c12\.modalidad|catalogo_huella|recibo:/u);
});

test("guardar envía versión, clave y elección, y muestra la fecha del recibo", async () => {
  const peticiones = [];
  const vista = host();
  const guardados = [];
  const api = async (ruta, config) => {
    peticiones.push(config);
    if (config.method === "GET") return respuesta(estado(null));
    const cuerpo = JSON.parse(config.body);
    return respuesta({ recibo_ref: "recibo:preferencias-modalidades:" + "c".repeat(64), version: cuerpo.version_esperada + 1, modalidades: cuerpo.modalidades, registrado_en: FECHA, reutilizado: false }, 201);
  };
  crearControladorPreferenciasModalidades({ host: vista, api, alGuardar: (e) => guardados.push(e) }).montar();
  await tick();
  vista.modo("ninguna");
  assert.match(vista.innerHTML, /No verá ofertas nuevas/u);
  vista.pulsar("guardar"); await tick(); await tick();
  const put = peticiones.find((p) => p.method === "PUT");
  const cuerpo = JSON.parse(put.body);
  assert.deepEqual(cuerpo.modalidades, []);
  assert.equal(cuerpo.version_esperada, 1);
  assert.match(cuerpo.clave_idempotencia, /^[0-9a-f-]{36}$/u);
  assert.match(vista.innerHTML, /Elección guardada el/u);
  assert.deepEqual(guardados, [[]]);
  vista.modo("algunas"); vista.opcion("vacante", true);
  assert.match(vista.innerHTML, /data-modalidades-opcion="vacante" checked/u);
});

test("409: recarga la elección vigente y lo explica en llano", async () => {
  let gets = 0;
  const vista = host();
  const api = async (ruta, config) => {
    if (config.method === "GET") { gets += 1; return respuesta(estado(gets === 1 ? null : ["vacante"], { version: gets })); }
    return { status: 409 };
  };
  crearControladorPreferenciasModalidades({ host: vista, api }).montar();
  await tick();
  vista.modo("ninguna"); vista.pulsar("guardar");
  await tick(); await tick(); await tick();
  assert.equal(gets, 2);
  assert.match(vista.innerHTML, /cambió desde otra ventana o dispositivo/u);
  assert.match(vista.innerHTML, /data-modalidades-opcion="vacante" checked/u);
  assert.doesNotMatch(vista.innerHTML, /409|versi[oó]n/u);
});

test("resultado incierto repite la misma clave y cuerpo", async () => {
  const puts = [];
  const vista = host();
  let fallar = true;
  const api = async (ruta, config) => {
    if (config.method === "GET") return respuesta(inicial());
    puts.push(config.body);
    if (fallar) { fallar = false; throw new Error("red"); }
    const c = JSON.parse(config.body);
    return respuesta({ recibo_ref: "recibo:x", version: 1, modalidades: c.modalidades, registrado_en: FECHA, reutilizado: true });
  };
  crearControladorPreferenciasModalidades({ host: vista, api }).montar();
  await tick();
  vista.modo("algunas"); vista.opcion("vacante", true); vista.pulsar("guardar");
  await tick(); await tick();
  assert.match(vista.innerHTML, /No sabemos si su elección se guardó/u);
  assert.match(vista.innerHTML, /data-modalidades="reintentar"/u);
  vista.pulsar("reintentar"); await tick(); await tick();
  assert.equal(puts.length, 2);
  assert.equal(puts[0], puts[1]);
  assert.match(vista.innerHTML, /Elección guardada el/u);
});

test("sin concesión o sin ruta el panel no se muestra; destruir aborta la lectura", async () => {
  for (const status of [403, 404]) {
    const vista = host();
    crearControladorPreferenciasModalidades({ host: vista, api: async () => ({ status }) }).montar();
    await tick(); await tick();
    assert.equal(vista.hidden, true);
    assert.equal(vista.innerHTML, "");
  }
  const vista = host();
  crearControladorPreferenciasModalidades({ host: vista, api: async () => ({ status: 503 }) }).montar();
  await tick(); await tick();
  assert.equal(vista.hidden, false);
  assert.match(vista.innerHTML, /El servicio no está disponible/u);
  let senal;
  const otra = host();
  const c = crearControladorPreferenciasModalidades({ host: otra, api: (r, config) => { senal = config.signal; return new Promise(() => {}); } }).montar();
  c.destruir();
  assert.equal(senal.aborted, true);
});

test("la sección de ofertas ofrece «Ver todas» solo si la lista ya aplica la elección", () => {
  assert.equal(renderizarSeccionOfertasMiBolsa([], {}), "");
  const filtrada = renderizarSeccionOfertasMiBolsa([], { segunModalidades: true });
  assert.match(filtrada, /data-accion="ver-todas-ofertas"/u);
  assert.match(filtrada, /No hay ofertas abiertas de los tipos de contrato que eligió/u);
  const todas = renderizarSeccionOfertasMiBolsa([], { segunModalidades: true, todas: true });
  assert.match(todas, /data-accion="ver-ofertas-elegidas"/u);
  assert.match(todas, /también las de tipos que no eligió/u);
  assert.match(renderizarSeccionOfertasMiBolsa([], { segunModalidades: true, carga: "cargando" }), /data-accion="ver-todas-ofertas" disabled/u);
  assert.match(renderizarSeccionOfertasMiBolsa([], { segunModalidades: true, carga: "error" }), /role="alert"/u);
});
