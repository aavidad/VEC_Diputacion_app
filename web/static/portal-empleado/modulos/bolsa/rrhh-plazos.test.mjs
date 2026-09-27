import assert from "node:assert/strict";
import test from "node:test";
import { crearClientePoliticaOfertas, ESQUEMA_POLITICA_OFERTAS, RUTA_POLITICA_OFERTAS,
  validarPoliticaRecibida } from "./rrhh-plazos-api.js";
import { crearTraductorRRHHPlazos } from "./rrhh-plazos-i18n.js";
import { crearSuperficieRRHHPlazos } from "./rrhh-plazos-ui.js";

const POLITICA = Object.freeze({
  plazo: { unidad: "dias_habiles", cantidad: 3, computo: "administrativo", municipio_sede: "18087" },
  adjudicacion: { criterio: "orden_vigente", elegibilidad: "disposicion_en_plazo" },
  no_cubierta: { accion: "llamamiento_directo", condicion: "sin_disposiciones_elegibles" },
});
const HUELLA = "a".repeat(64);
const vacia = (bolsaRef = "bolsa:1") => ({ data: { esquema: ESQUEMA_POLITICA_OFERTAS,
  bolsa_ref: bolsaRef, version: 0, configurada: false, ejemplo: true, politica: null } });
const vigente = (bolsaRef = "bolsa:1", extra = {}) => ({ data: { esquema: ESQUEMA_POLITICA_OFERTAS,
  bolsa_ref: bolsaRef, version: 1, configurada: true, ejemplo: true, huella_sha256: HUELLA,
  publicada_en: "2026-09-28T10:00:00Z", politica: structuredClone(POLITICA), ...extra } });
const respuesta = (status, body) => ({ ok: status >= 200 && status < 300, status, json: async () => body });
const turno = () => new Promise((resolver) => setTimeout(resolver, 0));

test("consulta la política por bolsa y publica una versión con recibo e idempotencia", async () => {
  const llamadas = [];
  const cliente = crearClientePoliticaOfertas({ fetchImpl: async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    return opciones.method === "GET" ? respuesta(200, vacia())
      : respuesta(201, vigente("bolsa:1", { recibo_ref: "recibo:politica:1" }));
  } });
  const consulta = await cliente.consultar("bolsa:1");
  assert.equal(consulta.politica.configurada, false);
  const alta = await cliente.publicar({ bolsa_ref: "bolsa:1", version_esperada: 0,
    clave_idempotencia: "politica-1", politica: POLITICA });
  assert.equal(alta.politica.recibo_ref, "recibo:politica:1");
  assert.equal(llamadas[0].ruta, `${RUTA_POLITICA_OFERTAS}?bolsa_ref=bolsa%3A1`);
  assert.equal(llamadas[1].opciones.headers["Idempotency-Key"], "politica-1");
  assert.deepEqual(Object.keys(JSON.parse(llamadas[1].opciones.body)),
    ["bolsa_ref", "version_esperada", "clave_idempotencia", "politica"]);
  assert.equal(llamadas[1].opciones.credentials, "same-origin");
});

test("rechaza respuestas que omiten recibo, falsean regla de ejemplo o cambian criterio", async () => {
  assert.throws(() => validarPoliticaRecibida(vigente("bolsa:1", { ejemplo: false })));
  const alterada = vigente();
  alterada.data.politica.adjudicacion.criterio = "otro";
  assert.throws(() => validarPoliticaRecibida(alterada));
  const cliente = crearClientePoliticaOfertas({ fetchImpl: async () => respuesta(201, vigente()) });
  await assert.rejects(cliente.publicar({ bolsa_ref: "bolsa:1", version_esperada: 0,
    clave_idempotencia: "politica-1", politica: POLITICA }), /recibo/);
});

test("la superficie muestra ejemplo, no cubierta y recibo solo después de POST válido", async () => {
  let intentos = 0;
  const claves = [];
  const cliente = { consultar: async () => ({ ok: true, politica: vacia().data }),
    publicar: async (comando) => {
      claves.push(comando.clave_idempotencia); intentos++;
      return intentos === 1 ? { ok: false, status: 503 } :
        { ok: true, politica: vigente("bolsa:1", { recibo_ref: "recibo:politica:1" }).data };
    } };
  const superficie = crearSuperficieRRHHPlazos({ cliente, traducir: crearTraductorRRHHPlazos(),
    puedeEditar: true, generarClave: () => "misma-clave" });
  superficie.activar("bolsa:1"); await turno();
  assert.match(superficie.renderizar(), /Regla de ejemplo/);
  assert.match(superficie.renderizar(), /Oferta no cubierta/);
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

test("al cambiar de bolsa ignora una consulta tardía del ámbito anterior", async () => {
  let resolverAnterior;
  const cliente = { consultar: async (bolsaRef) => bolsaRef === "bolsa:1"
    ? new Promise((resolver) => { resolverAnterior = resolver; })
    : { ok: true, politica: vigente("bolsa:2").data } };
  const superficie = crearSuperficieRRHHPlazos({ cliente, traducir: crearTraductorRRHHPlazos() });
  superficie.activar("bolsa:1"); superficie.activar("bolsa:2"); await turno();
  resolverAnterior({ ok: true, politica: vigente("bolsa:1").data }); await turno();
  assert.equal(superficie.estado().vigente.bolsa_ref, "bolsa:2");
});

test("sin concesión de publicación la política es visible y el conflicto conserva el borrador", async () => {
  const cliente = { consultar: async () => ({ ok: true, politica: vigente().data }),
    publicar: async () => ({ ok: false, status: 409 }) };
  const traductor = crearTraductorRRHHPlazos();
  const lectura = crearSuperficieRRHHPlazos({ cliente, traducir: traductor });
  lectura.activar("bolsa:1"); await turno();
  assert.match(lectura.renderizar(), /Orden vigente de la bolsa/);
  assert.doesNotMatch(lectura.renderizar(), /type="submit"/);
  const edicion = crearSuperficieRRHHPlazos({ cliente, traducir: traductor, puedeEditar: true,
    generarClave: () => "politica-clave-1" });
  edicion.activar("bolsa:1"); await turno();
  edicion.manejarCambio({ target: { name: "cantidad", value: "5", closest: () => ({}) } });
  const form = { closest: () => form, reportValidity: () => true };
  edicion.manejarSubmit({ target: form, preventDefault() {} }); await turno();
  assert.equal(edicion.estado().borrador.plazo.cantidad, 5);
  assert.equal(edicion.estado().conflicto, true);
  assert.match(edicion.renderizar(), /Revisar versión vigente/);
  assert.match(edicion.renderizar(), /value="5"/);
});
