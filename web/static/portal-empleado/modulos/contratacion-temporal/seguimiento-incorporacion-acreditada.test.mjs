import assert from "node:assert/strict";
import test from "node:test";
import {
  CONFLICTOS_SEGUIMIENTO_CESE, RUTA_CONFIRMACIONES_GINPIX, RUTAS_SEGUIMIENTO_CESE, crearClienteSeguimientoCeseHTTP,
  validarConsultaSeguimientoCese, validarSolicitudConfirmacionGINPIX,
} from "./cliente-http-seguimiento-cese.js";
import { montarPanelSeguimientoCese } from "./seguimiento-cese.js";
import { codigoValidoParaRuta, CODIGO_CIERRE_SIN_CESE_NO_CONTEMPLADO } from "./cliente-http-transporte.js";

const EXP = "expediente:cese:001";
const opciones = { causas_cese: [{ clave: "fin_sustitucion", etiqueta: "Fin de la sustitución", clave_i18n: "x.y", justificante_tipo: "comunicacion_reincorporacion" }],
  condiciones_cierre: ["cese_registrado", "ginpix_confirmado"], fase_retorno_modificacion: "fiscalizacion",
  motivos_modificacion: [{ clave: "cambio_jornada", etiqueta: "Cambio de jornada", clave_i18n: "x.z" }], confirmacion_ginpix: true };
const cese = { causa_clave: "fin_sustitucion", fecha_efecto: "2027-02-15", justificante_tipo: "comunicacion_reincorporacion",
  justificante_ref: "documento:r:1", justificante_sha256: "a".repeat(64), observaciones: "", recibo_ref: "recibo:cese:1", registrada_en: "2026-09-25T10:00:00Z" };
const ginpix = { ginpix_numero: "GX-2027-0042", ginpix_confirmada_en: "2027-02-10", recibo_ref: "recibo:ginpix:1", registrada_en: "2027-02-10T09:00:00.123456Z" };
const centro = { fecha_incorporacion: "2026-09-01", documento_tipo: "toma_posesion", documento_ref: "registro:centro:1", documento_sha256: "f".repeat(64),
  recibo_ref: "recibo:centro:1", registrada_en: "2026-09-01T08:00:00Z" };
const consulta = (estado = {}) => ({ esquema: "vec.contratacion-temporal.seguimiento-cese.v1", opciones,
  estado: { expediente_ref: EXP, incorporacion: { inicio: "2026-10-01" }, cese: null, cierre: null, ginpix: null, confirmacion_centro: null, ...estado } });
const contexto = Object.freeze({ expediente_ref: EXP, version: 8, fase_clave: "nombramiento", estado_clave: "en_curso" });
const esperar = () => new Promise((r) => setImmediate(r));
function contenedorFalso() {
  const eventos = new Map();
  return { innerHTML: "", eventos, addEventListener: (n, f) => eventos.set(n, f), removeEventListener: (n) => eventos.delete(n), querySelector: () => null };
}

test("la confirmación de GINPIX se valida y viaja por su ruta, con sus conflictos conocidos", async () => {
  const s = { expediente_ref: EXP, version_esperada: 8, clave_idempotencia: "123e4567-e89b-42d3-a456-426614174000", ginpix_numero: "GX-2027-0042",
    ginpix_confirmada_en: "2027-02-10", observaciones: "" };
  assert.deepEqual(validarSolicitudConfirmacionGINPIX({ ...s }), s);
  assert.throws(() => validarSolicitudConfirmacionGINPIX({ ...s, ginpix_numero: "" }), TypeError);
  assert.throws(() => validarSolicitudConfirmacionGINPIX({ ...s, ginpix_confirmada_en: "2027-02-30" }), TypeError);
  assert.ok(RUTAS_SEGUIMIENTO_CESE.includes(RUTA_CONFIRMACIONES_GINPIX));
  for (const c of ["ginpix_existente", "ginpix_no_confirmado", "ginpix_distinto"]) assert.ok(CONFLICTOS_SEGUIMIENTO_CESE.includes(c));
  const llamadas = [];
  const cliente = crearClienteSeguimientoCeseHTTP({ ejecutar: async (p) => { llamadas.push(p); return p.validarRespuesta({ esquema: "vec.contratacion-temporal.recibo-seguimiento.v1",
    operacion: "confirmar_ginpix", expediente_ref: EXP, version_anterior: 8, version_resultante: 9, fase_resultante: "nombramiento", estado_resultante: "en_curso",
    recibo_ref: "recibo:g:1", auditoria_ref: "auditoria:g:1", evento_ref: "evento:g:1", registrada_en: "2026-09-26T10:00:00Z", ginpix_numero: "GX-2027-0042" }); },
  validarOpciones: (o) => ({ signal: o?.signal }), serializarAcotado: (v) => JSON.stringify(v) });
  assert.equal((await cliente.confirmarGINPIX({ ...s })).ginpix_numero, "GX-2027-0042");
  assert.equal(llamadas[0].ruta, RUTA_CONFIRMACIONES_GINPIX);
});

test("la consulta exige las dos confirmaciones solo con la incorporación acreditada", () => {
  assert.equal(validarConsultaSeguimientoCese(consulta({ ginpix, confirmacion_centro: centro }), EXP).estado.ginpix.ginpix_numero, "GX-2027-0042");
  const sinClaves = consulta(); delete sinClaves.estado.ginpix; delete sinClaves.estado.confirmacion_centro;
  assert.throws(() => validarConsultaSeguimientoCese(sinClaves, EXP), TypeError, "con confirmacion_ginpix el estado trae las dos confirmaciones");
  assert.throws(() => validarConsultaSeguimientoCese(consulta({ ginpix: { ...ginpix, ginpix_numero: "" } }), EXP), TypeError);
  assert.throws(() => validarConsultaSeguimientoCese(consulta({ confirmacion_centro: { ...centro, documento_sha256: "x" } }), EXP), TypeError);
});

test("el panel ofrece confirmar GINPIX y, sin confirmación, no ofrece el cierre", async () => {
  const c = contenedorFalso();
  const respuesta = validarConsultaSeguimientoCese(consulta({ cese, confirmacion_centro: centro }), EXP);
  const desmontar = montarPanelSeguimientoCese({ contenedor: c, cliente: { consultarSeguimientoCese: async () => respuesta }, contexto });
  await esperar();
  assert.match(c.innerHTML, /data-ct-seg-form="ginpix"/u);
  assert.doesNotMatch(c.innerHTML, /data-ct-seg-form="cierre"/u, "sin GINPIX confirmado el cierre no se ofrece");
  assert.match(c.innerHTML, /registre antes la confirmación de la ficha de GINPIX/u);
  assert.match(c.innerHTML, /El centro confirma la incorporación el 1 de septiembre de 2026 con la toma de posesión/u);
  assert.doesNotMatch(c.innerHTML, /registro:centro:1/u, "sin referencias internas en pantalla");
  desmontar();
});

test("con GINPIX confirmado el cierre muestra su número y no lo deja teclear", async () => {
  const c = contenedorFalso();
  const llamadas = [];
  const respuesta = validarConsultaSeguimientoCese(consulta({ cese, ginpix }), EXP);
  const cliente = { consultarSeguimientoCese: async () => respuesta, cerrarExpediente: async (s) => { llamadas.push(s); throw new Error("fin"); } };
  const desmontar = montarPanelSeguimientoCese({ contenedor: c, cliente, contexto, confirmarOperacion: () => true });
  await esperar();
  assert.match(c.innerHTML, /data-ct-seg-form="cierre"/u);
  assert.match(c.innerHTML, /data-ct-seg-ginpix-cierre/u);
  assert.match(c.innerHTML, /GX-2027-0042/u);
  assert.doesNotMatch(c.innerHTML, /name="ginpix_numero"/u);
  assert.doesNotMatch(c.innerHTML, /data-ct-seg-form="ginpix"/u);
  desmontar();
});

test("el transporte reconoce que la regla de cierre no contempla el cierre sin cese", () => {
  const rutas = { preparacionCierreSinCese: "/api/vec/contratacion-temporal/seguimiento/cerrar-sin-cese/preparacion" };
  assert.equal(codigoValidoParaRuta(`${rutas.preparacionCierreSinCese}?expediente_ref=x`, 409, CODIGO_CIERRE_SIN_CESE_NO_CONTEMPLADO, rutas), true);
  assert.equal(codigoValidoParaRuta("/api/vec/contratacion-temporal/seguimiento/cerrar-sin-cese", 409, CODIGO_CIERRE_SIN_CESE_NO_CONTEMPLADO, rutas), true);
  assert.equal(codigoValidoParaRuta("/api/vec/contratacion-temporal/ceses", 409, CODIGO_CIERRE_SIN_CESE_NO_CONTEMPLADO, rutas), false);
});
