// Contratos del cambio de perfiles por lote: valida la preparación que devuelve
// el servidor y construye el cuerpo del lote sin inventar datos. No toca el DOM.
const PERSONA = /^per_[A-Za-z0-9_-]{22,128}$/u;
const OPACA = (prefijo) => new RegExp(`^${prefijo}[A-Za-z0-9_-]{22,128}$`, "u");
const PERFIL = OPACA("prf_"), VINCULO = OPACA("vca_"), CUENTA = OPACA("cta_");
const ROL = /^rol:[A-Za-z0-9_.:-]+:v[1-9][0-9]*$/u;
const HUELLA = /^[0-9a-f]{64}$/u;
const UNIDAD = /^[a-z][a-z0-9_:-]{2,127}$/u;
const CLAVE = /^[a-z][a-z0-9_]{1,63}$/u;
const ZONA = "Europe/Madrid";

function incompatible() { return Object.assign(new Error("respuesta_incompatible"), { codigo: "respuesta_incompatible" }); }
function exigir(valor) { if (!valor) throw incompatible(); }
function cerrado(valor, claves) {
  exigir(valor && typeof valor === "object" && !Array.isArray(valor));
  exigir(claves.every((k) => Object.hasOwn(valor, k)) && Object.keys(valor).every((k) => claves.includes(k)));
}
const version = (v) => Number.isSafeInteger(v) && v > 0;
const instante = (v) => typeof v === "string" && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d{1,6})?Z$/u.test(v) && Number.isFinite(Date.parse(v));
const texto = (v, max = 256) => typeof v === "string" && new TextEncoder().encode(v).byteLength <= max;

function motivo(m) {
  cerrado(m, ["catalogo_id", "catalogo_version", "catalogo_huella_sha256", "entrada_clave", "clave_i18n"]);
  exigir(texto(m.catalogo_id, 128) && m.catalogo_id && version(m.catalogo_version) && HUELLA.test(m.catalogo_huella_sha256)
    && texto(m.entrada_clave, 160) && m.entrada_clave && CLAVE.test(m.clave_i18n));
}

/** Valida la preparación (GET …/preparacion-lote) para una persona y unidad. */
export function validarPreparacion(datos, personaRef, unidadRef) {
  cerrado(datos, ["preparacion"]);
  const p = datos.preparacion;
  cerrado(p, ["operacion_ref", "auditoria_ref", "preparada_en", "persona_ref", "persona_version", "cuenta_ref", "cuenta_version",
    "procedencia_ref", "procedencia_version", "procedencia_huella_sha256", "unidad_ref", "altas", "bajas", "truncado", "motivos"]);
  exigir(/^prep_admin:[0-9a-f]{32}$/u.test(p.operacion_ref) && texto(p.auditoria_ref) && instante(p.preparada_en)
    && p.persona_ref === personaRef && PERSONA.test(p.persona_ref) && p.unidad_ref === unidadRef && UNIDAD.test(p.unidad_ref)
    && version(p.persona_version) && CUENTA.test(p.cuenta_ref) && version(p.cuenta_version) && texto(p.procedencia_ref)
    && version(p.procedencia_version) && HUELLA.test(p.procedencia_huella_sha256) && typeof p.truncado === "boolean"
    && Array.isArray(p.altas) && Array.isArray(p.bajas) && Array.isArray(p.motivos)
    && p.altas.length <= 64 && p.bajas.length <= 64 && p.motivos.length >= 1 && p.motivos.length <= 16);
  const refs = new Set();
  for (const a of p.altas) {
    cerrado(a, ["rol_version_ref", "nombre", "unidad_requerida", "vigente_hasta_maxima", "duracion_propuesta_segundos", "perfil_ref", "vinculo_ref", "huella_sha256"]);
    exigir(ROL.test(a.rol_version_ref) && texto(a.nombre) && typeof a.unidad_requerida === "boolean" && instante(a.vigente_hasta_maxima)
      && Number.isSafeInteger(a.duracion_propuesta_segundos) && a.duracion_propuesta_segundos > 0
      && PERFIL.test(a.perfil_ref) && VINCULO.test(a.vinculo_ref) && HUELLA.test(a.huella_sha256) && !refs.has(a.perfil_ref));
    refs.add(a.perfil_ref);
  }
  for (const b of p.bajas) {
    cerrado(b, ["rol_version_ref", "nombre", "perfil_ref", "perfil_version", "vinculo_ref", "vinculo_version", "vigente_desde", "vigente_hasta", "huella_sha256"]);
    exigir(ROL.test(b.rol_version_ref) && texto(b.nombre) && PERFIL.test(b.perfil_ref) && version(b.perfil_version)
      && VINCULO.test(b.vinculo_ref) && version(b.vinculo_version) && instante(b.vigente_desde) && instante(b.vigente_hasta)
      && HUELLA.test(b.huella_sha256) && !refs.has(b.perfil_ref));
    refs.add(b.perfil_ref);
  }
  const claves = new Set();
  for (const m of p.motivos) { motivo(m); exigir(!claves.has(m.clave_i18n)); claves.add(m.clave_i18n); }
  return structuredClone(p);
}

// Desfase de Madrid (minutos) en un instante dado, a partir de Intl.
function desfaseMadrid(ms) {
  const partes = Object.fromEntries(new Intl.DateTimeFormat("en-GB", { timeZone: ZONA, hourCycle: "h23",
    year: "numeric", month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit", second: "2-digit" })
    .formatToParts(new Date(ms)).filter((x) => x.type !== "literal").map((x) => [x.type, Number(x.value)]));
  return (Date.UTC(partes.year, partes.month - 1, partes.day, partes.hour, partes.minute, partes.second) - ms) / 60000;
}

/** Instante UTC de una hora local de Madrid para un día «aaaa-mm-dd». */
export function instanteMadrid(dia, hora, minuto, segundo) {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/u.exec(dia || "");
  if (!m) return null;
  const local = Date.UTC(Number(m[1]), Number(m[2]) - 1, Number(m[3]), hora, minuto, segundo);
  let ms = local - desfaseMadrid(local) * 60000;
  ms = local - desfaseMadrid(ms) * 60000;
  return Number.isFinite(ms) ? ms : null;
}

/** Día «aaaa-mm-dd» en Madrid de un instante. */
export function diaMadrid(ms) {
  const p = Object.fromEntries(new Intl.DateTimeFormat("en-GB", { timeZone: ZONA, year: "numeric", month: "2-digit", day: "2-digit" })
    .formatToParts(new Date(ms)).filter((x) => x.type !== "literal").map((x) => [x.type, x.value]));
  return `${p.year}-${p.month}-${p.day}`;
}

const iso = (ms) => new Date(Math.floor(ms / 1000) * 1000).toISOString().replace(".000Z", "Z");

/** Fechas de un alta a partir de lo que elige la persona; null si no valen. */
export function fechasAlta(alta, { hasta, empieza, desde }, ahora) {
  const maxima = Date.parse(alta.vigente_hasta_maxima);
  const fin = instanteMadrid(hasta, 23, 59, 59);
  // Un día posterior al máximo es un error que se explica; el mismo día del
  // máximo termina a la hora máxima permitida.
  if (fin === null || hasta > diaMadrid(maxima)) return { error: "fecha_hasta" };
  const finReal = Math.min(fin, maxima);
  if (empieza === "fecha") {
    const inicio = instanteMadrid(desde, 0, 0, 0);
    if (inicio === null || inicio <= ahora) return { error: "fecha_desde" };
    if (finReal <= inicio) return { error: "fecha_hasta_inicio" };
    return { inicio_vigencia: "programado", vigente_desde: iso(inicio), vigente_hasta: iso(finReal) };
  }
  if (empieza !== "ahora") return { error: "fecha_desde" };
  if (finReal <= ahora + 60000) return { error: "fecha_hasta" };
  return { inicio_vigencia: "inmediato", vigente_hasta: iso(finReal) };
}

/** Día propuesto para «Hasta»: hoy más la duración propuesta, sin pasar del máximo. */
export function hastaPropuesto(alta, ahora) {
  return diaMadrid(Math.min(ahora + alta.duracion_propuesta_segundos * 1000, Date.parse(alta.vigente_hasta_maxima)));
}

function hex32(cripto) {
  const b = new Uint8Array(16); cripto.getRandomValues(b);
  return [...b].map((x) => x.toString(16).padStart(2, "0")).join("");
}

/** Cuerpo del POST /lotes-ordinarios con un único cambio. */
export function construirLote(prep, cambio, motivoElegido, referencia, cripto) {
  const m = prep.motivos.find((x) => x.clave_i18n === motivoElegido);
  if (!m) throw Object.assign(new Error("motivo_requerido"), { codigo: "motivo_requerido" });
  const ref = String(referencia ?? "").trim();
  if (new TextEncoder().encode(ref).byteLength > 256 || /[\u0000-\u001f\u007f\u2028\u2029]/u.test(ref)) throw Object.assign(new Error("referencia"), { codigo: "referencia" });
  const comun = { unidad_ref: prep.unidad_ref, cuenta_ref: prep.cuenta_ref, cuenta_version: prep.cuenta_version,
    persona_ref: prep.persona_ref, persona_version: prep.persona_version, revision_continuidad: 0,
    procedencia_ref: prep.procedencia_ref, procedencia_version: prep.procedencia_version,
    procedencia_huella_sha256: prep.procedencia_huella_sha256 };
  let fila;
  if (cambio.operacion === "otorgar") {
    const { alta, fechas } = cambio;
    fila = { operacion: "otorgar", inicio_vigencia: fechas.inicio_vigencia, rol_version_ref: alta.rol_version_ref,
      objetivo: { ...comun, perfil_ref: alta.perfil_ref, perfil_version: 0, vinculo_ref: alta.vinculo_ref, vinculo_version: 0,
        huella_sha256: alta.huella_sha256, ...(fechas.vigente_desde ? { vigente_desde: fechas.vigente_desde } : {}), vigente_hasta: fechas.vigente_hasta } };
  } else {
    const { baja } = cambio;
    fila = { operacion: "revocar", rol_version_ref: baja.rol_version_ref,
      objetivo: { ...comun, perfil_ref: baja.perfil_ref, perfil_version: baja.perfil_version, vinculo_ref: baja.vinculo_ref,
        vinculo_version: baja.vinculo_version, huella_sha256: baja.huella_sha256 } };
  }
  const { clave_i18n: _, ...referenciaMotivo } = m;
  return { operacion_ref: `acto_admin:${hex32(cripto)}`, cambios: [fila], motivo: referenciaMotivo, ...(ref ? { referencia_acto: ref } : {}) };
}

/** Recibo del lote: debe ser el de esta orden, con un cambio confirmado. */
export function validarReciboLote(datos, cuerpo) {
  exigir(datos && typeof datos === "object" && datos.recibo && typeof datos.recibo === "object");
  const r = datos.recibo;
  exigir(r.operacion_ref === cuerpo.operacion_ref && Array.isArray(r.cambios) && r.cambios.length === 1
    && /^recibo_admin:[0-9a-f]{32}$/u.test(r.recibo_ref || "") && instante(r.confirmado_en) && Array.isArray(r.inicios) && r.inicios.length === 1);
  const c = r.cambios[0];
  exigir(c && c.perfil_ref === cuerpo.cambios[0].objetivo.perfil_ref
    && c.estado_posterior === (cuerpo.cambios[0].operacion === "otorgar" ? "activo" : "revocado"));
  return structuredClone(r);
}
