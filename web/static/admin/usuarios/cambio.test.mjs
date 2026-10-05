import assert from "node:assert/strict";
import test from "node:test";
import { cargarTextos } from "../../comun/textos.js";
import { validarPreparacion, fechasAlta, hastaPropuesto, construirLote, validarReciboLote, instanteMadrid, diaMadrid } from "./cambio-contratos.js";
import { montarCambioPerfiles } from "./cambio-perfiles.js";

const PERSONA = `per_${"a".repeat(24)}`;
const UNIDAD = "unidad:prueba";
const AHORA = Date.parse("2026-10-05T10:00:00Z");
const cripto = { getRandomValues(b) { b.fill(9); return b; } };
const motivo = { catalogo_id: "motivos_admin_perfiles", catalogo_version: 1, catalogo_huella_sha256: "c".repeat(64), entrada_clave: "motivo_" + "1".repeat(32), clave_i18n: "alta_funciones" };
function preparacion() {
  return { preparacion: { operacion_ref: `prep_admin:${"0".repeat(32)}`, auditoria_ref: "aud_v3_" + "a".repeat(32), preparada_en: "2026-10-05T10:00:00.123456Z",
    persona_ref: PERSONA, persona_version: 1, cuenta_ref: `cta_${"b".repeat(24)}`, cuenta_version: 2, procedencia_ref: "prc_" + "d".repeat(32),
    procedencia_version: 1, procedencia_huella_sha256: "e".repeat(64), unidad_ref: UNIDAD,
    altas: [{ rol_version_ref: "rol:tecnico_rrhh_desarrollo:v1", nombre: "Técnico de RRHH", unidad_requerida: false, vigente_hasta_maxima: "2027-09-30T15:11:06.000000Z",
      duracion_propuesta_segundos: 86400 * 30, perfil_ref: `prf_${"1".repeat(32)}`, vinculo_ref: `vca_${"2".repeat(32)}`, huella_sha256: "3".repeat(64) }],
    bajas: [{ rol_version_ref: "rol:llamamiento_desarrollo:v1", nombre: "Llamamientos", perfil_ref: `prf_${"4".repeat(32)}`, perfil_version: 1,
      vinculo_ref: `vca_${"5".repeat(32)}`, vinculo_version: 2, vigente_desde: "2026-01-01T00:00:00Z", vigente_hasta: "2036-01-01T00:00:00Z", huella_sha256: "6".repeat(64) }],
    truncado: false, motivos: [motivo] } };
}

test("la preparación se valida cerrada y para la persona y unidad pedidas", () => {
  const p = validarPreparacion(preparacion(), PERSONA, UNIDAD);
  assert.equal(p.altas.length, 1);
  for (const [nombre, cambiar] of Object.entries({
    otra_persona: (d) => { d.preparacion.persona_ref = `per_${"z".repeat(24)}`; },
    otra_unidad: (d) => { d.preparacion.unidad_ref = "unidad:otra"; },
    campo_extra: (d) => { d.preparacion.extra = 1; },
    baja_sin_nombre: (d) => { delete d.preparacion.bajas[0].nombre; },
    sin_motivos: (d) => { d.preparacion.motivos = []; },
    motivo_repetido: (d) => { d.preparacion.motivos.push({ ...motivo }); },
    perfil_repetido: (d) => { d.preparacion.bajas[0].perfil_ref = d.preparacion.altas[0].perfil_ref; },
  })) {
    const d = preparacion(); cambiar(d);
    assert.throws(() => validarPreparacion(d, PERSONA, UNIDAD), /respuesta_incompatible/u, nombre);
  }
});

test("las fechas se interpretan en Madrid y respetan el máximo del perfil", () => {
  assert.equal(new Date(instanteMadrid("2026-10-05", 23, 59, 59)).toISOString(), "2026-10-05T21:59:59.000Z");
  assert.equal(new Date(instanteMadrid("2026-12-01", 0, 0, 0)).toISOString(), "2026-11-30T23:00:00.000Z");
  assert.equal(diaMadrid(Date.parse("2026-10-05T22:30:00Z")), "2026-10-06");
  const alta = preparacion().preparacion.altas[0];
  assert.equal(hastaPropuesto(alta, AHORA), "2026-11-04");
  const inmediato = fechasAlta(alta, { hasta: "2026-11-04", empieza: "ahora" }, AHORA);
  assert.deepEqual(inmediato, { inicio_vigencia: "inmediato", vigente_hasta: "2026-11-04T22:59:59Z" });
  const programado = fechasAlta(alta, { hasta: "2026-11-04", empieza: "fecha", desde: "2026-10-10" }, AHORA);
  assert.equal(programado.inicio_vigencia, "programado");
  assert.equal(programado.vigente_desde, "2026-10-09T22:00:00Z");
  assert.equal(fechasAlta(alta, { hasta: "2030-01-01", empieza: "ahora" }, AHORA).error, "fecha_hasta");
  assert.equal(fechasAlta(alta, { hasta: "2027-09-30", empieza: "ahora" }, AHORA).vigente_hasta, "2027-09-30T15:11:06Z");
  assert.equal(fechasAlta(alta, { hasta: "2026-10-04", empieza: "ahora" }, AHORA).error, "fecha_hasta");
  assert.equal(fechasAlta(alta, { hasta: "2026-11-04", empieza: "fecha", desde: "2026-10-05" }, AHORA).error, "fecha_desde");
  assert.equal(fechasAlta(alta, { hasta: "2026-10-10", empieza: "fecha", desde: "2026-10-20" }, AHORA).error, "fecha_hasta_inicio");
  assert.equal(fechasAlta(alta, { hasta: "", empieza: "ahora" }, AHORA).error, "fecha_hasta");
});

test("el cuerpo del lote copia la preparación sin inventar nada", () => {
  const p = validarPreparacion(preparacion(), PERSONA, UNIDAD);
  const fechas = fechasAlta(p.altas[0], { hasta: "2026-11-04", empieza: "ahora" }, AHORA);
  const alta = construirLote(p, { operacion: "otorgar", alta: p.altas[0], fechas }, "alta_funciones", "  Resolución 12/2026 ", cripto);
  assert.match(alta.operacion_ref, /^acto_admin:[0-9a-f]{32}$/u);
  assert.equal(alta.referencia_acto, "Resolución 12/2026");
  assert.deepEqual(alta.motivo, { catalogo_id: motivo.catalogo_id, catalogo_version: 1, catalogo_huella_sha256: motivo.catalogo_huella_sha256, entrada_clave: motivo.entrada_clave });
  const c = alta.cambios[0];
  assert.equal(c.operacion, "otorgar"); assert.equal(c.inicio_vigencia, "inmediato");
  assert.equal(c.objetivo.perfil_version, 0); assert.equal(c.objetivo.huella_sha256, p.altas[0].huella_sha256);
  assert.equal(Object.hasOwn(c.objetivo, "vigente_desde"), false);
  const baja = construirLote(p, { operacion: "revocar", baja: p.bajas[0] }, "alta_funciones", "", cripto);
  const b = baja.cambios[0];
  assert.equal(b.operacion, "revocar"); assert.equal(Object.hasOwn(b, "inicio_vigencia"), false);
  assert.equal(Object.hasOwn(b.objetivo, "vigente_hasta"), false); assert.equal(b.objetivo.vinculo_version, 2);
  assert.equal(Object.hasOwn(baja, "referencia_acto"), false);
  assert.throws(() => construirLote(p, { operacion: "revocar", baja: p.bajas[0] }, "otro", "", cripto), /motivo_requerido/u);
  assert.throws(() => construirLote(p, { operacion: "revocar", baja: p.bajas[0] }, "alta_funciones", "a\u0007b", cripto), /referencia/u);
});

test("el recibo debe ser el de la orden enviada", () => {
  const p = validarPreparacion(preparacion(), PERSONA, UNIDAD);
  const cuerpo = construirLote(p, { operacion: "revocar", baja: p.bajas[0] }, "alta_funciones", "", cripto);
  const recibo = { recibo: { operacion_ref: cuerpo.operacion_ref, recibo_ref: `recibo_admin:${"f".repeat(32)}`, confirmado_en: "2026-10-05T10:01:00.5Z",
    cambios: [{ perfil_ref: p.bajas[0].perfil_ref, estado_posterior: "revocado" }], inicios: [{}] } };
  assert.equal(validarReciboLote(recibo, cuerpo).recibo_ref, recibo.recibo.recibo_ref);
  assert.throws(() => validarReciboLote({ recibo: { ...recibo.recibo, operacion_ref: "acto_admin:" + "0".repeat(32) } }, cuerpo));
  assert.throws(() => validarReciboLote({ recibo: { ...recibo.recibo, cambios: [{ perfil_ref: p.bajas[0].perfil_ref, estado_posterior: "activo" }] } }, cuerpo));
});

// Contenedor mínimo: guarda el HTML y responde a los selectores por id.
class Hijo { constructor() { this.html = ""; this.textContent = ""; this.disabled = false; } set innerHTML(v) { this.html = v; } get innerHTML() { return this.html; } focus() {} }
class Contenedor {
  constructor() { this.id = "c"; this.html = ""; this.listeners = {}; this.hijos = new Map(); }
  set innerHTML(v) { this.html = v; for (const [, i] of v.matchAll(/id="([^"]+)"/gu)) if (!this.hijos.has(i)) this.hijos.set(i, new Hijo()); }
  get innerHTML() { return this.html; }
  get panel() { return this.hijos.get("c-panel").html; }
  querySelector(sel) { return this.hijos.get(sel.slice(1)) || new Hijo(); }
  addEventListener(t, f) { (this.listeners[t] ??= []).push(f); }
  removeEventListener() {}
  contains() { return true; }
  replaceChildren() { this.html = ""; }
}

test("la pantalla traduce los fallos de preparación y ofrece las dos vías", async () => {
  const textos = await cargarTextos("admin-usuarios");
  const base = { textos, cripto, persona: PERSONA, nombre: "Antonio Reyes Álvarez", unidadRef: UNIDAD, ahora: () => AHORA, alVolver() {} };
  const fallo = new Contenedor();
  await montarCambioPerfiles(fallo, { ...base, cliente: { preparar: async () => { throw Object.assign(new Error("x"), { estado: 404 }); }, aplicarLote: async () => ({}) } }).listo;
  assert.match(fallo.panel, new RegExp(textos.traducir("lote.errores.no_disponible").slice(0, 20), "u"));
  assert.doesNotMatch(fallo.panel, /data-cambio="preparar"/u);
  const bien = new Contenedor();
  await montarCambioPerfiles(bien, { ...base, cliente: { preparar: async () => preparacion(), aplicarLote: async () => ({}) } }).listo;
  assert.match(bien.panel, /data-cambio="via-asignar"/u);
  assert.match(bien.panel, /data-cambio="via-retirar"/u);
  assert.doesNotMatch(bien.panel, /per_|prf_|rol:/u);
  for (const clave of ["lote.titulo", "lote.errores.incierto", "lote.motivos.alta_funciones", "lote.paso"]) {
    assert.notEqual(textos.traducir(clave), clave, clave);
  }
});
