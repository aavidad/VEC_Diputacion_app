import assert from "node:assert/strict";
import test from "node:test";
import { crearClientePoliticaOfertas, ESQUEMA_POLITICA_OFERTAS, RUTA_POLITICA_OFERTAS,
  RUTA_CAPACIDAD_POLITICA_OFERTAS,
  validarPoliticaEditable, validarPoliticaRecibida, cargarEjemploPlazas, cargarConfirmacionAdjudicacion, plazasCompletas, crearLectorReglasCompartido } from "./rrhh-plazos-api.js";
import { crearTraductorRRHHPlazos } from "./rrhh-plazos-i18n.js";
import { crearSuperficieRRHHPlazos as crearSuperficieRRHHPlazosReal, cargarPlazoCatalogo } from "./rrhh-plazos-ui.js?v=20261001-ct-a-i18n-v1";

const crearSuperficieRRHHPlazos = (opciones) => crearSuperficieRRHHPlazosReal({ cargarConfirmacion: async () => null, ...opciones });

const POLITICA = Object.freeze({
  plazo: { unidad: "dias_habiles", cantidad: 3, computo: "administrativo", municipio_sede: "18087", inicio: "notificacion" },
  adjudicacion: { criterio: "orden_vigente", elegibilidad: "disposicion_en_plazo" },
  no_cubierta: { accion: "llamamiento_directo", condicion: "sin_disposiciones_elegibles" },
});
const HUELLA = "a".repeat(64);
const EJEMPLO_PLAZAS = Object.freeze({ llamada: "simultanea", respuesta_horas: 24, tras_renuncia: "siguiente_en_orden" });
const conEjemplo = async () => structuredClone(EJEMPLO_PLAZAS);
const PLAZO_CATALOGO = Object.freeze({ unidad: "dias_habiles", cantidad: 2, computo: "administrativo", municipio_sede: "", inicio: "notificacion" });
const conPlazoCatalogo = async () => structuredClone(PLAZO_CATALOGO);
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
  assert.equal(JSON.parse(llamadas[2].opciones.body).politica.plazo.inicio, "notificacion");
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
  const sinInicio = vigente("bolsa:1", { recibo_ref: "recibo:politica:1" });
  delete sinInicio.data.politica.plazo.inicio;
  const clienteRespuestaLegada = crearClientePoliticaOfertas({ fetchImpl: async () => respuesta(201, sinInicio) });
  await assert.rejects(clienteRespuestaLegada.publicar({ bolsa_ref: "bolsa:1", version_esperada: 0,
    clave_idempotencia: "politica-2", politica: POLITICA }), /recibo/);
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
  delete dias.plazo.inicio;
  dias.plazo.cantidad = 2;
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
    generarClave: () => "misma-clave", cargarEjemplo: conEjemplo, cargarPlazo: conPlazoCatalogo });
  superficie.activar("bolsa:1"); await turno();
  assert.match(superficie.renderizar(), /Regla de ejemplo/);
  assert.match(superficie.renderizar(), /Oferta no cubierta/);
  assert.match(superficie.renderizar(), /Días hábiles/);
  assert.match(superficie.renderizar(), /name="cantidad"[^>]*value="2"/u);
  assert.match(superficie.renderizar(), /id="rrhh-plazos-ayuda"[^>]*hidden/u);
  superficie.manejarClick({ target: { disabled: false, dataset: { rrhhPlazosAccion: "ayuda" }, closest: () => ({ disabled: false, dataset: { rrhhPlazosAccion: "ayuda" } }) } });
  assert.doesNotMatch(superficie.renderizar(), /id="rrhh-plazos-ayuda"[^>]*hidden/u);
  assert.match(superficie.renderizar(), /fecha y hora del correo externo/u);
  assert.match(superficie.renderizar(), /Correo externo declarado por RRHH/u);
  assert.match(superficie.renderizar(), /catálogo de Bolsa/);
  assert.match(superficie.renderizar(), /Correo externo declarado por RRHH/);
  assert.match(superficie.renderizar(), /Inicio del plazo/);
  assert.match(superficie.renderizar(), /Guardar política de esta bolsa/);
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

test("el selector recupera el plazo del catálogo y deja vacía la unidad sin propuesta", async () => {
  const cliente = { consultar: async () => ({ ok: true, politica: vacia().data }),
    consultarCapacidad: async () => ({ ok: true, puede_publicar: true }) };
  const superficie = crearSuperficieRRHHPlazos({ cliente, traducir: crearTraductorRRHHPlazos(),
    cargarPlazo: async () => ({ ...PLAZO_CATALOGO, cantidad: 12 }) });
  superficie.activar("bolsa:1"); await turno();
  const control = (name, value) => ({ target: { name, value, closest: () => ({}) } });
  assert.equal(superficie.estado().borrador.plazo.cantidad, 12);
  superficie.manejarCambio(control("unidad", "horas_naturales"));
  assert.deepEqual([superficie.estado().borrador.plazo.cantidad, superficie.estado().borrador.plazo.computo],
    [null, "continuo_utc"]);
  assert.match(superficie.renderizar(), /max="720"/);
  superficie.manejarCambio(control("unidad", "dias_habiles"));
  assert.deepEqual([superficie.estado().borrador.plazo.cantidad, superficie.estado().borrador.plazo.computo],
    [12, "administrativo"]);
  assert.match(superficie.renderizar(), /max="30"/);
  assert.match(superficie.renderizar(), /id="rrhh-plazos-ayuda"[^>]*hidden>[^<]*correo externo/u);
});

test("el plazo inicial sale de la regla publicada y válida del catálogo de Bolsa", async () => {
  const lector = (regla) => ({ reglas: async () => ({ catalogos: [{ modulo: "bolsa", estado: "disponible",
    paquete_ejemplo: true, reglas: [regla] }] }) });
  const regla = { clave: "b10.plazo_publicacion", origen: "reglamento", unidad: "dias_habiles",
    cantidad: 3, computo: "administrativo", inicio: "notificacion" };
  assert.deepEqual(await cargarPlazoCatalogo({ cliente: lector(regla) }), { ...PLAZO_CATALOGO, cantidad: 3 });
  assert.deepEqual(await cargarPlazoCatalogo({ cliente: lector({ ...regla, inicio: "notificacion" }) }), { ...PLAZO_CATALOGO, cantidad: 3 });
  assert.deepEqual(await cargarPlazoCatalogo({ cliente: lector({ ...regla, unidad: "dias_naturales" }) }),
    { ...PLAZO_CATALOGO, cantidad: 3, unidad: "dias_naturales" });
  assert.equal(await cargarPlazoCatalogo({ cliente: lector({ ...regla, inicio: "publicacion" }) }), null);
  assert.equal(await cargarPlazoCatalogo({ cliente: lector({ ...regla, origen: "ejemplo" }) }), null);
  assert.equal(await cargarPlazoCatalogo({ cliente: lector({ ...regla, unidad: "horas_naturales" }) }), null);
  assert.equal(await cargarPlazoCatalogo({ cliente: lector({ ...regla, cantidad: 31 }) }), null);
  assert.equal(await cargarPlazoCatalogo({ cliente: { reglas: async () => { throw new Error("sin red"); } } }), null);
});

test("la política vigente tiene prioridad y la falta de catálogo no propone cifras", async () => {
  const cliente = { consultar: async () => ({ ok: true, politica: vigente().data }),
    consultarCapacidad: async () => ({ ok: true, puede_publicar: true }) };
  const superficie = crearSuperficieRRHHPlazos({ cliente, traducir: crearTraductorRRHHPlazos(),
    cargarPlazo: conPlazoCatalogo });
  superficie.activar("bolsa:1"); await turno();
  assert.deepEqual(superficie.estado().borrador.plazo, POLITICA.plazo);
  const sinCatalogo = crearSuperficieRRHHPlazos({ cliente: { ...cliente,
    consultar: async () => ({ ok: true, politica: vacia().data }) }, traducir: crearTraductorRRHHPlazos(),
    cargarPlazo: async () => null });
  sinCatalogo.activar("bolsa:1"); await turno();
  assert.equal(sinCatalogo.estado().borrador.plazo.cantidad, null);
  assert.match(sinCatalogo.renderizar(), /name="cantidad"[^>]*value=""/u);
});

test("la versión legada permanece intacta y el borrador guarda inicio desde B10 con CAS", async () => {
  const legada = vigente();
  delete legada.data.politica.plazo.inicio;
  const recibida = validarPoliticaRecibida(legada);
  assert.equal(Object.hasOwn(recibida.politica.plazo, "inicio"), false);
  assert.throws(() => validarPoliticaEditable(recibida.politica), /política/u);
  const enviadas = [];
  const cliente = { consultar: async () => ({ ok: true, politica: recibida }),
    consultarCapacidad: async () => ({ ok: true, puede_publicar: true }),
    publicar: async (comando) => { enviadas.push(comando); return { ok: true,
      politica: vigente("bolsa:1", { version: 2, recibo_ref: "recibo:politica:2", politica: comando.politica }).data }; } };
  const superficie = crearSuperficieRRHHPlazos({ cliente, traducir: crearTraductorRRHHPlazos(),
    cargarEjemplo: conEjemplo, cargarPlazo: conPlazoCatalogo, generarClave: () => "politica-legada-2" });
  superficie.activar("bolsa:1"); await turno();
  assert.equal(Object.hasOwn(superficie.estado().vigente.politica.plazo, "inicio"), false);
  assert.equal(superficie.estado().borrador.plazo.inicio, "notificacion");
  assert.match(superficie.renderizar(), /versión no registra el inicio/);
  const form = { closest: () => form, reportValidity: () => true };
  superficie.manejarSubmit({ target: form, preventDefault() {} }); await turno();
  assert.equal(enviadas.length, 1);
  assert.deepEqual([enviadas[0].version_esperada, enviadas[0].clave_idempotencia, enviadas[0].politica.plazo.inicio],
    [1, "politica-legada-2", "notificacion"]);
  assert.equal(recibida.politica.plazo.inicio, undefined, "la proyección histórica no se reescribe");
});

test("sin la regla B10 vigente una versión antigua no permite guardar", async () => {
  const legada = vigente();
  delete legada.data.politica.plazo.inicio;
  let publicadas = 0;
  const cliente = { consultar: async () => ({ ok: true, politica: validarPoliticaRecibida(legada) }),
    consultarCapacidad: async () => ({ ok: true, puede_publicar: true }),
    publicar: async () => { publicadas++; return { ok: false, status: 500 }; } };
  const superficie = crearSuperficieRRHHPlazos({ cliente, traducir: crearTraductorRRHHPlazos(),
    cargarEjemplo: conEjemplo, cargarPlazo: async () => null });
  superficie.activar("bolsa:1"); await turno();
  assert.match(superficie.renderizar(), /Espere a que se cargue la regla vigente/);
  assert.match(superficie.renderizar(), /type="submit" class="boton-primario" disabled/);
  const form = { closest: () => form, reportValidity: () => true };
  superficie.manejarSubmit({ target: form, preventDefault() {} }); await turno();
  assert.equal(publicadas, 0);
});

test("si el catálogo tarda, no sustituye un plazo que RRHH ya ha introducido", async () => {
  let resolverCatalogo;
  const cliente = { consultar: async () => ({ ok: true, politica: vacia().data }),
    consultarCapacidad: async () => ({ ok: true, puede_publicar: true }) };
  const superficie = crearSuperficieRRHHPlazos({ cliente, traducir: crearTraductorRRHHPlazos(),
    cargarPlazo: () => new Promise((resolver) => { resolverCatalogo = resolver; }) });
  superficie.activar("bolsa:1"); await turno();
  superficie.manejarCambio({ target: { name: "unidad", value: "horas_naturales", closest: () => ({}) } });
  superficie.manejarCambio({ target: { name: "cantidad", value: "6", closest: () => ({}) } });
  resolverCatalogo(PLAZO_CATALOGO); await turno();
  assert.deepEqual(superficie.estado().borrador.plazo,
    { unidad: "horas_naturales", cantidad: 6, computo: "continuo_utc", municipio_sede: "", inicio: "" });
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
  const superficie = crearSuperficieRRHHPlazos({ cliente, traducir: crearTraductorRRHHPlazos(),
    cargarEjemplo: conEjemplo, cargarPlazo: conPlazoCatalogo });
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
  assert.match(html, /<option value="simultanea" selected>A la vez: una persona por plaza libre<\/option>/u);
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


test("la confirmación de la oferta se carga del catálogo y rechaza ausencia, duplicados y modos ajenos", async () => {
  const cliente = (reglas) => ({ reglas: async () => ({ catalogos: [{ modulo: "bolsa", estado: "disponible", reglas }] }) });
  const regla = { clave: "b30.confirmacion_adjudicacion", valor: "aceptacion_previa" };
  assert.equal(await cargarConfirmacionAdjudicacion({ cliente: cliente([regla]) }), "aceptacion_previa");
  for (const reglas of [[], [regla, regla], [{ ...regla, valor: "segunda_respuesta" }]]) {
    assert.equal(await cargarConfirmacionAdjudicacion({ cliente: cliente(reglas) }), null);
  }
  const telematica = structuredClone(POLITICA);
  telematica.adjudicacion.confirmacion = "aceptacion_previa";
  assert.deepEqual(validarPoliticaEditable(telematica), telematica);
  telematica.adjudicacion.confirmacion = "segunda_respuesta";
  assert.throws(() => validarPoliticaEditable(telematica));
});

test("guardar una versión nueva adopta la aceptación previa del catálogo sin alterar la política histórica leída", async () => {
  let enviado;
  const antigua = vigente("bolsa:1", { politica: { ...structuredClone(POLITICA), plazas: structuredClone(EJEMPLO_PLAZAS) } }).data;
  const cliente = {
    consultar: async () => ({ ok: true, politica: antigua }),
    consultarCapacidad: async () => ({ ok: true, puede_publicar: true }),
    publicar: async (comando) => {
      enviado = comando;
      return { ok: true, politica: { ...antigua, version: 2, politica: comando.politica, recibo_ref: "recibo:politica:2" } };
    },
  };
  const s = crearSuperficieRRHHPlazos({ cliente, traducir: crearTraductorRRHHPlazos(),
    cargarEjemplo: conEjemplo, cargarConfirmacion: async () => "aceptacion_previa" });
  s.activar("bolsa:1"); await turno();
  assert.equal(antigua.politica.adjudicacion.confirmacion, undefined);
  assert.match(s.renderizar(), /No se pide una segunda respuesta/u);
  assert.doesNotMatch(s.renderizar(), /name="plazas_respuesta_horas"|name="plazas_tras_renuncia"/u);
  const form = { closest: () => form, reportValidity: () => true };
  s.manejarSubmit({ target: form, preventDefault() {} }); await turno();
  assert.equal(enviado.politica.adjudicacion.confirmacion, "aceptacion_previa");
  assert.equal(enviado.version_esperada, 1);
  assert.equal(antigua.politica.adjudicacion.confirmacion, undefined);
});

test("el lector compartido hace una sola lectura para peticiones simultáneas y vuelve a leer después", async () => {
  let lecturas = 0;
  let soltar;
  const lector = crearLectorReglasCompartido(async () => ({ reglas: () => { lecturas++; return new Promise((r) => { soltar = r; }); } }));
  const tres = [lector(), lector(), lector()];
  await turno();
  soltar({ catalogos: [] });
  const resultados = await Promise.all(tres);
  assert.equal(lecturas, 1);
  assert.ok(resultados.every((r) => r === resultados[0]));
  const otra = lector(); await turno(); soltar({ catalogos: [] }); await otra;
  assert.equal(lecturas, 2, "terminada la lectura, la siguiente pide lo vigente");
});

test("abrir el llamamiento pide las reglas vigentes una sola vez", async () => {
  const original = globalThis.fetch;
  let lecturas = 0;
  globalThis.fetch = async (ruta, opciones) => {
    if (ruta !== "/api/vec/reglas/vigentes") return original(ruta, opciones);
    lecturas++;
    await turno();
    const cuerpo = JSON.stringify({ data: { esquema: "vec.reglas.vigentes.v1", catalogos: [] } });
    return { ok: true, status: 200, text: async () => cuerpo };
  };
  try {
    const cliente = { consultar: async () => ({ ok: true, politica: vacia("bolsa:1").data }),
      consultarCapacidad: async () => ({ ok: true, puede_publicar: false }) };
    const superficie = crearSuperficieRRHHPlazosReal({ cliente, traducir: crearTraductorRRHHPlazos() });
    superficie.activar("bolsa:1");
    for (let i = 0; i < 20; i++) await turno();
    assert.equal(lecturas, 1);
  } finally {
    globalThis.fetch = original;
  }
});
