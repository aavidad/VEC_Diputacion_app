import assert from "node:assert/strict";
import test from "node:test";
import { crearClientePoliticaOfertas, ESQUEMA_POLITICA_OFERTAS, RUTA_POLITICA_OFERTAS,
  RUTA_CAPACIDAD_POLITICA_OFERTAS,
  validarPoliticaEditable, validarPoliticaRecibida, cargarEjemploPlazas, plazasCompletas } from "./rrhh-plazos-api.js";
import { crearTraductorRRHHPlazos } from "./rrhh-plazos-i18n.js";
import { crearSuperficieRRHHPlazos } from "./rrhh-plazos-ui.js";

const POLITICA = Object.freeze({
  plazo: { unidad: "dias_habiles", cantidad: 3, computo: "administrativo", municipio_sede: "18087" },
  adjudicacion: { criterio: "orden_vigente", elegibilidad: "disposicion_en_plazo" },
  no_cubierta: { accion: "llamamiento_directo", condicion: "sin_disposiciones_elegibles" },
});
const HUELLA = "a".repeat(64);
const EJEMPLO_PLAZAS = Object.freeze({ llamada: "simultanea", respuesta_horas: 24, tras_renuncia: "siguiente_en_orden" });
const conEjemplo = async () => structuredClone(EJEMPLO_PLAZAS);
const vacia = (bolsaRef = "bolsa:1", extra = {}) => ({ data: { esquema: ESQUEMA_POLITICA_OFERTAS,
  bolsa_ref: bolsaRef, version: 0, configurada: false, ejemplo: true, politica: null,
  puede_publicar: false, ...extra } });
const vigente = (bolsaRef = "bolsa:1", extra = {}) => ({ data: { esquema: ESQUEMA_POLITICA_OFERTAS,
  bolsa_ref: bolsaRef, version: 1, configurada: true, ejemplo: true, huella_sha256: HUELLA,
  publicada_en: "2026-09-28T10:00:00Z", politica: structuredClone(POLITICA),
  puede_publicar: false, ...extra } });
const respuesta = (status, body) => ({ ok: status >= 200 && status < 300, status, json: async () => body });
const turno = () => new Promise((resolver) => setTimeout(resolver, 0));

test("consulta la política por bolsa y publica una versión con recibo e idempotencia", async () => {
  const llamadas = [];
  const cliente = crearClientePoliticaOfertas({ fetchImpl: async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
  return opciones.method === "GET" ? respuesta(200, vacia())
      : ruta === RUTA_CAPACIDAD_POLITICA_OFERTAS ? respuesta(200, { puede_publicar: true })
        : respuesta(201, vigente("bolsa:1", { recibo_ref: "recibo:politica:1" }));
  } });
  const consulta = await cliente.consultar("bolsa:1");
  assert.equal(consulta.politica.configurada, false);
  assert.equal(Object.hasOwn(consulta.politica, "puede_publicar"), false);
  const capacidad = await cliente.consultarCapacidad("bolsa:1");
  assert.equal(capacidad.puede_publicar, true);
  const alta = await cliente.publicar({ bolsa_ref: "bolsa:1", version_esperada: 0,
    clave_idempotencia: "politica-1", politica: POLITICA });
  assert.equal(alta.politica.recibo_ref, "recibo:politica:1");
  assert.equal(llamadas[0].ruta, `${RUTA_POLITICA_OFERTAS}?bolsa_ref=bolsa%3A1`);
  assert.equal(llamadas[1].ruta, RUTA_CAPACIDAD_POLITICA_OFERTAS);
  assert.deepEqual(JSON.parse(llamadas[1].opciones.body), { bolsa_ref: "bolsa:1" });
  assert.equal(llamadas[1].opciones.credentials, "same-origin");
  assert.equal(llamadas[2].opciones.headers["Idempotency-Key"], "politica-1");
  assert.deepEqual(Object.keys(JSON.parse(llamadas[2].opciones.body)),
    ["bolsa_ref", "version_esperada", "clave_idempotencia", "politica"]);
  assert.equal(llamadas[2].opciones.credentials, "same-origin");
});

test("rechaza respuestas que omiten recibo, falsean regla de ejemplo o cambian criterio", async () => {
  const sinPermiso = vacia(); delete sinPermiso.data.puede_publicar;
  const clienteSinPermiso = crearClientePoliticaOfertas({ fetchImpl: async () => respuesta(200, sinPermiso) });
  assert.equal((await clienteSinPermiso.consultar("bolsa:1")).politica.configurada, false);
  const clienteOtraBolsa = crearClientePoliticaOfertas({ fetchImpl: async () => respuesta(200, vacia("bolsa:otra")) });
  await assert.rejects(clienteOtraBolsa.consultar("bolsa:1"), /otra bolsa/u);
  assert.throws(() => validarPoliticaRecibida(vigente("bolsa:1", { ejemplo: false })));
  const alterada = vigente();
  alterada.data.politica.adjudicacion.criterio = "otro";
  assert.throws(() => validarPoliticaRecibida(alterada));
  const cliente = crearClientePoliticaOfertas({ fetchImpl: async () => respuesta(201, vigente()) });
  await assert.rejects(cliente.publicar({ bolsa_ref: "bolsa:1", version_esperada: 0,
    clave_idempotencia: "politica-1", politica: POLITICA }), /recibo/);
});

test("capacidad exige un booleano explícito y falla cerrada ante denegación", async () => {
  for (const cuerpo of [{}, { puede_publicar: "true" }, { data: { puede_publicar: true } }]) {
    const cliente = crearClientePoliticaOfertas({ fetchImpl: async () => respuesta(200, cuerpo) });
    await assert.rejects(cliente.consultarCapacidad("bolsa:1"), /capacidad/u);
  }
  const clienteDenegado = crearClientePoliticaOfertas({ fetchImpl: async () => respuesta(403, { error: { codigo: "denegado" } }) });
  assert.deepEqual(await clienteDenegado.consultarCapacidad("bolsa:1"),
    { ok: false, status: 403, codigo: "denegado" });
});

test("la política acepta 48 horas naturales continuas y conserva el rango propio de cada unidad", () => {
  const horas = structuredClone(POLITICA);
  horas.plazo = { ...horas.plazo, unidad: "horas_naturales", cantidad: 48, computo: "continuo_utc" };
  assert.deepEqual(validarPoliticaEditable(horas), horas);
  assert.deepEqual(validarPoliticaRecibida(vigente("bolsa:1", { politica: horas })).politica, horas);
  horas.plazo.cantidad = 720;
  assert.deepEqual(validarPoliticaEditable(horas), horas);
  horas.plazo.cantidad = 721;
  assert.throws(() => validarPoliticaEditable(horas), /política/u);
  horas.plazo.cantidad = 48;
  horas.plazo.computo = "administrativo";
  assert.throws(() => validarPoliticaEditable(horas), /política/u);
  const dias = structuredClone(POLITICA);
  dias.plazo.cantidad = 31;
  assert.throws(() => validarPoliticaEditable(dias), /política/u);
});

test("la superficie muestra ejemplo, no cubierta y recibo solo después de POST válido", async () => {
  let intentos = 0;
  const claves = [];
  const cliente = { consultar: async () => ({ ok: true, politica: vacia("bolsa:1", { puede_publicar: true }).data }),
    consultarCapacidad: async () => ({ ok: true, puede_publicar: true }),
    publicar: async (comando) => {
      claves.push(comando.clave_idempotencia); intentos++;
      return intentos === 1 ? { ok: false, status: 503 } :
        { ok: true, politica: vigente("bolsa:1", { recibo_ref: "recibo:politica:1" }).data };
    } };
  const superficie = crearSuperficieRRHHPlazos({ cliente, traducir: crearTraductorRRHHPlazos(),
    generarClave: () => "misma-clave", cargarEjemplo: conEjemplo });
  superficie.activar("bolsa:1"); await turno();
  assert.match(superficie.renderizar(), /Regla de ejemplo/);
  assert.match(superficie.renderizar(), /Oferta no cubierta/);
  assert.match(superficie.renderizar(), /Horas naturales/);
  assert.match(superficie.renderizar(), /value="48"/);
  assert.match(superficie.renderizar(), /id="rrhh-plazos-ayuda"[^>]*hidden/u);
  superficie.manejarClick({ target: { disabled: false, dataset: { rrhhPlazosAccion: "ayuda" }, closest: () => ({ disabled: false, dataset: { rrhhPlazosAccion: "ayuda" } }) } });
  assert.doesNotMatch(superficie.renderizar(), /id="rrhh-plazos-ayuda"[^>]*hidden/u);
  assert.match(superficie.renderizar(), /Europe\/Madrid/);
  assert.match(superficie.renderizar(), /correo no acredita notificación/);
  assert.doesNotMatch(superficie.renderizar(), /recibo:politica:1/);
  const control = { name: "municipio_sede", value: "18087", closest: () => ({}) };
  superficie.manejarCambio({ target: control });
  const formulario = { closest: () => formulario, reportValidity: () => true };
  superficie.manejarSubmit({ target: formulario, preventDefault() {} }); await turno();
  assert.match(superficie.renderizar(), /Reintentar el mismo guardado/);
  superficie.manejarSubmit({ target: formulario, preventDefault() {} }); await turno();
  assert.deepEqual(claves, ["misma-clave", "misma-clave"]);
  assert.match(superficie.renderizar(), /recibo:politica:1/);
});

test("el selector cambia cómputo y valor inicial sin mezclar horas con días", async () => {
  const cliente = { consultar: async () => ({ ok: true, politica: vacia().data }),
    consultarCapacidad: async () => ({ ok: true, puede_publicar: true }) };
  const superficie = crearSuperficieRRHHPlazos({ cliente, traducir: crearTraductorRRHHPlazos() });
  superficie.activar("bolsa:1"); await turno();
  const control = (name, value) => ({ target: { name, value, closest: () => ({}) } });
  assert.equal(superficie.estado().borrador.plazo.cantidad, 48);
  superficie.manejarCambio(control("unidad", "dias_naturales"));
  assert.deepEqual([superficie.estado().borrador.plazo.cantidad, superficie.estado().borrador.plazo.computo],
    [1, "administrativo"]);
  assert.match(superficie.renderizar(), /max="30"/);
  superficie.manejarCambio(control("unidad", "horas_naturales"));
  assert.deepEqual([superficie.estado().borrador.plazo.cantidad, superficie.estado().borrador.plazo.computo],
    [48, "continuo_utc"]);
  assert.match(superficie.renderizar(), /max="720"/);
  assert.match(superficie.renderizar(), /id="rrhh-plazos-ayuda"[^>]*hidden>[^<]*plazo legal/);
});

test("al cambiar de bolsa ignora una consulta tardía del ámbito anterior", async () => {
  let resolverAnterior;
  const cliente = { consultar: async (bolsaRef) => bolsaRef === "bolsa:1"
    ? new Promise((resolver) => { resolverAnterior = resolver; })
    : { ok: true, politica: vigente("bolsa:2").data },
  consultarCapacidad: async () => ({ ok: true, puede_publicar: false }) };
  const superficie = crearSuperficieRRHHPlazos({ cliente, traducir: crearTraductorRRHHPlazos() });
  superficie.activar("bolsa:1"); superficie.activar("bolsa:2"); await turno();
  resolverAnterior({ ok: true, politica: vigente("bolsa:1").data }); await turno();
  assert.equal(superficie.estado().vigente.bolsa_ref, "bolsa:2");
});

test("sin concesión de publicación la política es visible y el conflicto conserva el borrador", async () => {
  const cliente = { consultar: async () => ({ ok: true, politica: vigente().data }),
    consultarCapacidad: async () => ({ ok: true, puede_publicar: false }),
    publicar: async () => ({ ok: false, status: 409 }) };
  const traductor = crearTraductorRRHHPlazos();
  const lectura = crearSuperficieRRHHPlazos({ cliente, traducir: traductor, cargarEjemplo: conEjemplo });
  lectura.activar("bolsa:1"); await turno();
  assert.match(lectura.renderizar(), /Orden vigente de la bolsa/);
  assert.doesNotMatch(lectura.renderizar(), /type="submit"/);
  const edicion = crearSuperficieRRHHPlazos({ cliente: { ...cliente,
    consultar: async () => ({ ok: true, politica: vigente("bolsa:1", { puede_publicar: true }).data }),
    consultarCapacidad: async () => ({ ok: true, puede_publicar: true }) }, traducir: traductor,
    generarClave: () => "politica-clave-1", cargarEjemplo: conEjemplo });
  edicion.activar("bolsa:1"); await turno();
  edicion.manejarCambio({ target: { name: "cantidad", value: "5", closest: () => ({}) } });
  const form = { closest: () => form, reportValidity: () => true };
  edicion.manejarSubmit({ target: form, preventDefault() {} }); await turno();
  assert.equal(edicion.estado().borrador.plazo.cantidad, 5);
  assert.equal(edicion.estado().conflicto, true);
  assert.match(edicion.renderizar(), /Revisar versión vigente/);
  assert.match(edicion.renderizar(), /value="5"/);
});

test("la proyección GET nunca habilita edición y respuestas tardías no cruzan bolsas", async () => {
  let resolverAnterior;
  const cliente = { consultar: async (bolsaRef) => ({ ok: true,
    politica: vacia(bolsaRef, { puede_publicar: true }).data }),
  consultarCapacidad: async (bolsaRef) => bolsaRef === "bolsa:1"
    ? new Promise((resolver) => { resolverAnterior = resolver; })
    : { ok: true, puede_publicar: false } };
  const superficie = crearSuperficieRRHHPlazos({ cliente, traducir: crearTraductorRRHHPlazos() });
  superficie.activar("bolsa:1"); await turno();
  assert.doesNotMatch(superficie.renderizar(), /type="submit"/);
  superficie.activar("bolsa:2"); await turno();
  resolverAnterior({ ok: true, puede_publicar: true }); await turno();
  assert.equal(superficie.estado().bolsaRef, "bolsa:2");
  assert.equal(superficie.estado().puedePublicar, false);
  assert.doesNotMatch(superficie.renderizar(), /type="submit"/);
});

test("la revocación de capacidad tras 403 deshabilita la edición", async () => {
  const cliente = { consultar: async () => ({ ok: true, politica: vacia("bolsa:1").data }),
    consultarCapacidad: async () => ({ ok: true, puede_publicar: true }),
    publicar: async () => ({ ok: false, status: 403 }) };
  const superficie = crearSuperficieRRHHPlazos({ cliente, traducir: crearTraductorRRHHPlazos(), cargarEjemplo: conEjemplo });
  superficie.activar("bolsa:1"); await turno();
  assert.match(superficie.renderizar(), /type="submit"/);
  superficie.manejarCambio({ target: { name: "municipio_sede", value: "18087", closest: () => ({}) } });
  const form = { closest: () => form, reportValidity: () => true };
  superficie.manejarSubmit({ target: form, preventDefault() {} }); await turno();
  assert.equal(superficie.estado().puedePublicar, false);
  assert.doesNotMatch(superficie.renderizar(), /type="submit"/);
});

test("durante la comprobación y ante un fallo de capacidad mantiene solo lectura y permite reintentar", async () => {
  let resolverPrimera;
  let intentos = 0;
  const cliente = { consultar: async () => ({ ok: true, politica: vacia().data }),
    consultarCapacidad: async () => ++intentos === 1
      ? new Promise((resolver) => { resolverPrimera = resolver; })
      : { ok: true, puede_publicar: true } };
  const superficie = crearSuperficieRRHHPlazos({ cliente, traducir: crearTraductorRRHHPlazos() });
  superficie.activar("bolsa:1"); await turno();
  assert.match(superficie.renderizar(), /Comprobando permiso de edición/);
  assert.doesNotMatch(superficie.renderizar(), /type="submit"/);
  resolverPrimera({ ok: false, status: 503 }); await turno();
  assert.match(superficie.renderizar(), /La política sigue en solo lectura/);
  const boton = { disabled: false, dataset: { rrhhPlazosAccion: "capacidad" } };
  superficie.manejarClick({ target: { closest: () => boton } }); await turno();
  assert.equal(intentos, 2);
  assert.match(superficie.renderizar(), /type="submit"/);
});

test("el apartado de plazas admite solo los valores que ejecuta Bolsa", () => {
  const politica = { ...structuredClone(POLITICA), plazas: { ...EJEMPLO_PLAZAS } };
  assert.deepEqual(validarPoliticaEditable(politica), politica);
  assert.deepEqual(validarPoliticaEditable({ ...structuredClone(POLITICA), plazas: null }), POLITICA);
  for (const cambio of [{ llamada: "al_azar" }, { respuesta_horas: 0 }, { respuesta_horas: 721 }, { respuesta_horas: "24" },
    { tras_renuncia: "baja" }, { otra: 1 }]) {
    assert.throws(() => validarPoliticaEditable({ ...structuredClone(POLITICA), plazas: { ...EJEMPLO_PLAZAS, ...cambio } }), /política/u);
  }
  assert.equal(plazasCompletas({ llamada: "", respuesta_horas: null, tras_renuncia: "" }), false);
});

test("la propuesta de ejemplo sale del paquete de reglas y se ignora si falta o es ajena", async () => {
  const regla = (clave, extra) => ({ clave, ...extra });
  const catalogo = (reglas) => ({ reglas: async () => ({ catalogos: [{ modulo: "bolsa", estado: "disponible", reglas }] }) });
  assert.deepEqual(await cargarEjemploPlazas({ cliente: catalogo([regla("b30.plazas_llamada", { valor: "sucesiva" }),
    regla("b30.plazas_plazo_respuesta", { cantidad: 12 }), regla("b30.plazas_tras_renuncia", { valor: "llamamiento_directo" })]) }),
  { llamada: "sucesiva", respuesta_horas: 12, tras_renuncia: "llamamiento_directo" });
  assert.equal(await cargarEjemploPlazas({ cliente: catalogo([regla("b30.plazas_llamada", { valor: "sucesiva" })]) }), null);
  assert.equal(await cargarEjemploPlazas({ cliente: catalogo([regla("b30.plazas_llamada", { valor: "otra" }),
    regla("b30.plazas_plazo_respuesta", { cantidad: 12 }), regla("b30.plazas_tras_renuncia", { valor: "llamamiento_directo" })]) }), null);
  assert.equal(await cargarEjemploPlazas({ cliente: { reglas: async () => { throw new Error("sin red"); } } }), null);
});

test("la superficie rellena las plazas con el ejemplo, respeta lo que cambia RRHH y las envía al guardar", async () => {
  const enviadas = [];
  const cliente = { consultar: async () => ({ ok: true, politica: vigente().data }),
    consultarCapacidad: async () => ({ ok: true, puede_publicar: true }),
    publicar: async (comando) => { enviadas.push(comando.politica); return { ok: true, politica: vigente("bolsa:1", { version: 2, recibo_ref: "recibo:politica:2",
      politica: { ...structuredClone(POLITICA), plazas: comando.politica.plazas } }).data }; } };
  const superficie = crearSuperficieRRHHPlazos({ cliente, traducir: crearTraductorRRHHPlazos(), cargarEjemplo: conEjemplo,
    generarClave: () => "politica-clave-2" });
  superficie.activar("bolsa:1"); await turno();
  const html = superficie.renderizar();
  assert.match(html, /<h3>Plazas<\/h3>/u);
  assert.match(html, /<option value="simultanea" selected>A la vez, tantas personas como plazas libres<\/option>/u);
  assert.match(html, /name="plazas_respuesta_horas"[^>]*value="24"/u);
  superficie.manejarCambio({ target: { name: "plazas_tras_renuncia", value: "llamamiento_directo", closest: () => ({}) } });
  superficie.manejarCambio({ target: { name: "plazas_respuesta_horas", value: "6", closest: () => ({}) } });
  const form = { closest: () => form, reportValidity: () => true };
  superficie.manejarSubmit({ target: form, preventDefault() {} }); await turno();
  assert.deepEqual(enviadas, [{ ...POLITICA, plazas: { llamada: "simultanea", respuesta_horas: 6, tras_renuncia: "llamamiento_directo" } }]);
  assert.equal(superficie.estado().borrador.plazas.tras_renuncia, "llamamiento_directo");
});

test("sin paquete de ejemplo las plazas quedan por elegir y no se guarda una política incompleta", async () => {
  let publicadas = 0;
  const cliente = { consultar: async () => ({ ok: true, politica: vacia("bolsa:1", { puede_publicar: true }).data }),
    consultarCapacidad: async () => ({ ok: true, puede_publicar: true }), publicar: async () => { publicadas++; return { ok: false, status: 500 }; } };
  const superficie = crearSuperficieRRHHPlazos({ cliente, traducir: crearTraductorRRHHPlazos(), cargarEjemplo: async () => null });
  superficie.activar("bolsa:1"); await turno();
  assert.match(superficie.renderizar(), /<option value="" selected>Elija una opción<\/option>/u);
  superficie.manejarCambio({ target: { name: "municipio_sede", value: "18087", closest: () => ({}) } });
  const form = { closest: () => form, reportValidity: () => true };
  superficie.manejarSubmit({ target: form, preventDefault() {} }); await turno();
  assert.equal(publicadas, 0);
  assert.match(superficie.renderizar(), /Revise los campos señalados/u);
});
