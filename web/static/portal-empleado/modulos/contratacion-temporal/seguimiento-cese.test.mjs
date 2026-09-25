import assert from "node:assert/strict";
import test from "node:test";
import {
  CONFLICTOS_SEGUIMIENTO_CESE, RUTA_CESES_NOMBRAMIENTO, RUTA_SEGUIMIENTO_CESE, crearClienteSeguimientoCeseHTTP,
  validarConsultaSeguimientoCese, validarReciboSeguimiento, validarSolicitudCese, validarSolicitudCierre, validarSolicitudModificacion,
} from "./cliente-http-seguimiento-cese.js";
import { contextoSeguimientoCeseDesdeEstado, montarPanelSeguimientoCese } from "./seguimiento-cese.js";

const EXP = "expediente:cese:001";
const cese = Object.freeze({ expediente_ref: EXP, version_esperada: 7, clave_idempotencia: "123e4567-e89b-42d3-a456-426614174000",
  causa_clave: "fin_sustitucion", fecha_efecto: "2027-02-15", justificante_ref: "documento:reincorporacion:1", justificante_sha256: "a".repeat(64), observaciones: "" });
const recibo = Object.freeze({ esquema: "vec.contratacion-temporal.recibo-seguimiento.v1", operacion: "registrar_cese", expediente_ref: EXP,
  version_anterior: 7, version_resultante: 8, fase_resultante: "nombramiento", estado_resultante: "en_curso", recibo_ref: "recibo:cese:1",
  auditoria_ref: "auditoria:cese:1", evento_ref: "evento:cese:1", registrada_en: "2026-09-25T10:00:00Z", causa_clave: "fin_sustitucion", fecha_efecto: "2027-02-15" });
const opciones = { causas_cese: [{ clave: "fin_sustitucion", etiqueta: "Fin de la sustitución", clave_i18n: "x.y", justificante_tipo: "comunicacion_reincorporacion" }],
  condiciones_cierre: ["cese_registrado", "ginpix_confirmado"], fase_retorno_modificacion: "fiscalizacion",
  motivos_modificacion: [{ clave: "cambio_jornada", etiqueta: "Cambio de jornada", clave_i18n: "x.z" }] };
const consulta = (estado = {}) => ({ esquema: "vec.contratacion-temporal.seguimiento-cese.v1", opciones,
  estado: { expediente_ref: EXP, incorporacion: { inicio: "2026-10-01" }, cese: null, cierre: null, ...estado } });

test("las solicitudes se validan antes de salir y rechazan fechas imposibles y campos ajenos", () => {
  assert.deepEqual(validarSolicitudCese({ ...cese }), cese);
  assert.throws(() => validarSolicitudCese({ ...cese, fecha_efecto: "2027-02-30" }), TypeError);
  assert.throws(() => validarSolicitudCese({ ...cese, actor_ref: "x" }), TypeError);
  assert.throws(() => validarSolicitudCese({ ...cese, justificante_sha256: "A".repeat(64) }), TypeError);
  const cierre = { expediente_ref: EXP, version_esperada: 8, clave_idempotencia: cese.clave_idempotencia, ginpix_numero: "", ginpix_confirmada_en: "", observaciones: "" };
  assert.equal(validarSolicitudCierre(cierre).ginpix_numero, "");
  assert.throws(() => validarSolicitudCierre({ ...cierre, ginpix_numero: "G-1" }), TypeError, "número sin fecha");
  assert.equal(validarSolicitudCierre({ ...cierre, ginpix_numero: "G-1", ginpix_confirmada_en: "2027-02-20" }).ginpix_numero, "G-1");
  const mod = { expediente_ref: EXP, version_esperada: 7, clave_idempotencia: cese.clave_idempotencia, motivo_clave: "cambio_jornada",
    periodo_inicio: "2026-10-01", periodo_fin: "2027-03-31", porcentaje_jornada: 5000, observaciones: "Media jornada." };
  assert.equal(validarSolicitudModificacion(mod).porcentaje_jornada, 5000);
  assert.throws(() => validarSolicitudModificacion({ ...mod, periodo_fin: "2026-09-30" }), TypeError);
  assert.throws(() => validarSolicitudModificacion({ ...mod, observaciones: "" }), TypeError);
  assert.throws(() => validarSolicitudModificacion({ ...mod, porcentaje_jornada: 10001 }), TypeError);
});

test("el recibo debe corresponder a la solicitud y a la operación", () => {
  assert.equal(validarReciboSeguimiento({ ...recibo }, cese, "registrar_cese").recibo_ref, "recibo:cese:1");
  assert.throws(() => validarReciboSeguimiento({ ...recibo }, cese, "cerrar_expediente"), TypeError);
  assert.throws(() => validarReciboSeguimiento({ ...recibo, version_resultante: 9 }, cese, "registrar_cese"), TypeError);
  assert.throws(() => validarReciboSeguimiento({ ...recibo, expediente_ref: "expediente:otro:1" }, cese, "registrar_cese"), TypeError);
  assert.throws(() => validarReciboSeguimiento({ ...recibo, coste_centimos: 0 }, cese, "registrar_cese"), TypeError);
});

test("la consulta exige el esquema, el expediente pedido y un cierre solo tras el cese", () => {
  assert.equal(validarConsultaSeguimientoCese(consulta(), EXP).estado.cese, null);
  assert.throws(() => validarConsultaSeguimientoCese(consulta(), "expediente:otro:1"), TypeError);
  assert.throws(() => validarConsultaSeguimientoCese({ ...consulta(), opciones: { ...opciones, condiciones_cierre: ["ginpix_confirmado"] } }, EXP), TypeError);
  assert.throws(() => validarConsultaSeguimientoCese(consulta({ cierre: { condiciones: [], ginpix_numero: "", ginpix_confirmada_en: "", observaciones: "", recibo_ref: "recibo:c:1", registrada_en: "" } }), EXP), TypeError);
});

test("el cliente usa la ruta propia, espera 201 y trata como rechazo cierto solo los conflictos conocidos", async () => {
  const llamadas = [];
  const cliente = crearClienteSeguimientoCeseHTTP({ ejecutar: async (p) => { llamadas.push(p); return p.validarRespuesta(p.efecto ? { ...recibo } : consulta()); },
    validarOpciones: (o) => ({ signal: o?.signal }), serializarAcotado: (v) => JSON.stringify(v) });
  assert.equal((await cliente.registrarCese({ ...cese })).recibo_ref, "recibo:cese:1");
  assert.equal(llamadas[0].ruta, RUTA_CESES_NOMBRAMIENTO);
  assert.equal(llamadas[0].estadoEsperado, 201);
  assert.equal(llamadas[0].efecto, true);
  for (const codigo of CONFLICTOS_SEGUIMIENTO_CESE) assert.equal(llamadas[0].rechazoDeterminado({ envelopeValido: true, estado: 409, codigo }), true);
  assert.equal(llamadas[0].rechazoDeterminado({ envelopeValido: true, estado: 409, codigo: "otro" }), false);
  assert.equal(llamadas[0].rechazoDeterminado({ envelopeValido: true, estado: 503, codigo: "no_disponible" }), false);
  await cliente.consultarSeguimientoCese(EXP);
  assert.equal(llamadas[1].ruta, RUTA_SEGUIMIENTO_CESE);
  assert.equal(llamadas[1].efecto, false);
  assert.throws(() => cliente.registrarCese({ ...cese, fecha_efecto: "15/02/2027" }), TypeError);
  assert.equal(llamadas.length, 2);
});

test("el panel solo aparece en nombramiento con el detalle vigente y cuadra la versión", () => {
  const estado = (fase, version = 7) => ({ vista: "expediente", carga: "listo", expediente: { demostracion: false, expediente_ref: EXP, version: 7 },
    cuadro: { expedientes: [{ expediente_ref: EXP, version, fase_clave: fase, estado_clave: "en_curso" }] } });
  assert.deepEqual({ ...contextoSeguimientoCeseDesdeEstado(estado("nombramiento")) }, { expediente_ref: EXP, version: 7, fase_clave: "nombramiento", estado_clave: "en_curso" });
  assert.equal(contextoSeguimientoCeseDesdeEstado(estado("fiscalizacion")), null);
  assert.equal(contextoSeguimientoCeseDesdeEstado(estado("nombramiento", 6)), null);
});

function contenedorFalso() {
  const eventos = new Map();
  return { innerHTML: "", eventos, addEventListener: (n, f) => eventos.set(n, f), removeEventListener: (n) => eventos.delete(n), querySelector: () => null };
}
const esperar = () => new Promise((r) => setImmediate(r));
const contexto = Object.freeze({ expediente_ref: EXP, version: 7, fase_clave: "nombramiento", estado_clave: "en_curso" });

test("el panel muestra cese y modificación antes del cese, y solo el cierre después", async () => {
  const c = contenedorFalso();
  let respuesta = validarConsultaSeguimientoCese(consulta(), EXP);
  const cliente = { consultarSeguimientoCese: async () => respuesta };
  let desmontar = montarPanelSeguimientoCese({ contenedor: c, cliente, contexto });
  assert.match(c.innerHTML, /aria-busy="true"/u);
  await esperar();
  assert.match(c.innerHTML, /data-ct-seg-form="cese"/u);
  assert.match(c.innerHTML, /data-ct-seg-form="modificacion"/u);
  assert.doesNotMatch(c.innerHTML, /data-ct-seg-form="cierre"/u);
  assert.match(c.innerHTML, /El expediente volverá a: Fiscalización/u);
  assert.match(c.innerHTML, /aria-expanded="false"[^>]*>\?<\/button>/u, "la ayuda solo se abre con «?»");
  assert.match(c.innerHTML, /id="ct-seg-cese-ayuda"[^>]*hidden/u);
  desmontar();
  assert.equal(c.eventos.size, 0);

  respuesta = validarConsultaSeguimientoCese(consulta({ cese: { causa_clave: "fin_sustitucion", fecha_efecto: "2027-02-15", justificante_tipo: "comunicacion_reincorporacion",
    justificante_ref: "documento:r:1", justificante_sha256: "a".repeat(64), observaciones: "", recibo_ref: "recibo:cese:1", registrada_en: "2026-09-25T10:00:00Z" } }), EXP);
  desmontar = montarPanelSeguimientoCese({ contenedor: c, cliente, contexto });
  await esperar();
  assert.match(c.innerHTML, /data-ct-seg-form="cierre"/u);
  assert.match(c.innerHTML, /name="ginpix_numero"/u, "el catálogo exige GINPIX");
  assert.doesNotMatch(c.innerHTML, /data-ct-seg-form="cese"/u);
  assert.match(c.innerHTML, /Cese registrado: Fin de la sustitución/u);
  desmontar();
});

test("sin incorporación el cese queda deshabilitado con su motivo y un fallo de consulta ofrece reintentar", async () => {
  const c = contenedorFalso();
  let fallar = false;
  const cliente = { consultarSeguimientoCese: async () => { if (fallar) throw new Error("red"); return validarConsultaSeguimientoCese(consulta({ incorporacion: null }), EXP); } };
  let desmontar = montarPanelSeguimientoCese({ contenedor: c, cliente, contexto });
  await esperar();
  assert.match(c.innerHTML, /Para registrar el cese hace falta la incorporación acreditada/u);
  assert.match(c.innerHTML, /<button type="submit" class="boton-primario"\s+disabled>Registrar el cese/u);
  desmontar();
  fallar = true;
  desmontar = montarPanelSeguimientoCese({ contenedor: c, cliente, contexto });
  await esperar();
  assert.match(c.innerHTML, /role="alert"/u);
  assert.match(c.innerHTML, /data-ct-seg-reintentar/u);
  desmontar();
});
