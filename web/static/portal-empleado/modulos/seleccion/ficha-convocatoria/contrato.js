const objeto = (v) => v !== null && typeof v === "object" && !Array.isArray(v);
const ref = (v) => typeof v === "string" && v.length > 0 && v.length <= 512 && !/[\s\p{Cc}\p{Bidi_Control}*\uFFFD]/u.test(v);
const huella = (v) => typeof v === "string" && /^[a-f0-9]{64}$/u.test(v);
const entero = (v) => Number.isSafeInteger(v) && v > 0;
const texto = (v, n, opcional = false) => typeof v === "string" && [...v].length <= n && (opcional || v.trim().length > 0);
const config = (v) => objeto(v) && ref(v.id) && entero(v.version) && huella(v.huella_contenido_sha256);

export class ErrorFichaConvocatoria extends Error {
  constructor(codigo, estado = 0) { super(codigo); this.name = "ErrorFichaConvocatoria"; this.codigo = codigo; this.estado = estado; }
}

export function validarSelectorFicha(v) {
  if (!objeto(v) || Object.keys(v).length !== 2 || !ref(v.convocatoria_id) || v.convocatoria_id.includes("#") || !entero(v.secuencia)) {
    throw new ErrorFichaConvocatoria("consulta_invalida");
  }
  return Object.freeze({ convocatoria_id: v.convocatoria_id, secuencia: v.secuencia });
}

function unicas(lista, clave) { return new Set(lista.map((v) => v?.[clave])).size === lista.length; }

/** Valida correspondencia y evidencia. No concede ni comprueba permisos. */
export function validarLecturaFicha(v, selector) {
  const f = v?.ficha, e = v?.evidencia;
  if (!objeto(v) || !objeto(f) || !objeto(e) || f.convocatoria_id !== selector.convocatoria_id || f.secuencia !== selector.secuencia
    || !entero(f.revision) || f.fuente_ref !== `${selector.convocatoria_id}#${selector.secuencia}` || !huella(f.huella_version_sha256)
    || !config(f.flujo_proceso) || !config(f.reglas_baremacion) || f.fases_estado !== "pendiente_fuente"
    || f.referencias_cobertura_estado !== "pendiente_fuente" || !Array.isArray(f.bases) || !f.bases.length || f.bases.length > 256
    || !Array.isArray(f.requisitos) || f.requisitos.length > 256 || !unicas(f.bases, "publicacion_ref") || !unicas(f.requisitos, "referencia")
    || !f.bases.every((b) => objeto(b) && texto(b.rol, 64) && ref(b.publicacion_ref) && ref(b.documento_ref) && entero(b.version_documento)
      && ref(b.representacion_ref) && huella(b.huella_contenido_sha256) && ref(b.firma_validada_ref) && ref(b.recibo_custodia_ref))
    || !f.requisitos.every((r) => objeto(r) && ref(r.referencia) && entero(r.orden) && texto(r.titulo, 180)
      && texto(r.descripcion, 3000, true) && typeof r.obligatorio === "boolean")
    || !ref(e.recibo_ref) || !ref(e.decision_ref) || !huella(e.consumo_huella_sha256) || !ref(e.auditoria_ref) || !ref(e.correlacion_ref)
    || typeof e.consultada_en !== "string" || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,6})?Z$/u.test(e.consultada_en)
    || !Number.isFinite(Date.parse(e.consultada_en)) || new Date(e.consultada_en).toISOString().slice(0, 19) !== e.consultada_en.slice(0, 19)) {
    throw new ErrorFichaConvocatoria("respuesta_incompatible");
  }
  return v;
}
