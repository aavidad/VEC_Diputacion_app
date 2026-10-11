/** Contrato HTTP del vínculo entre expediente y categoría de la relación de puestos (CT154). */
import { registroB2, referenciaB2 } from "./contrato-incorporacion-personal-b2.js";

export const RUTA_VINCULO_RPT = "/api/vec/contratacion-temporal/incorporacion-personal-b2/vinculo-categoria-rpt/v1";
export const RUTA_CATEGORIAS_RPT = "/api/vec/contratacion-temporal/incorporacion-personal-b2/categorias-rpt/v1";

const SHA = /^[0-9a-f]{64}$/u;
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/u;
const CATEGORIA = /^[a-z][a-z0-9_.:-]{2,127}$/u;
const CAMPOS_ENTRADA = ["expediente_ref", "version_expediente_esperada", "analisis_version", "analisis_recibo_ref",
  "analisis_huella_sha256", "categoria_ref", "catalogo_version", "catalogo_huella_sha256", "categoria_id", "fuente_ref",
  "motivo_ref", "aprobacion_ref", "revision_esperada", "anterior_recibo_ref", "clave_idempotencia"];
const CAMPOS_VINCULO = ["revision", "recibo_ref", "catalogo_id", "modulo_id", "catalogo_version", "catalogo_huella_sha256",
  "categoria_id", "fuente_ref", "motivo_ref", "aprobacion_ref", "prospectivo", "acredita_procedencia_historica"];

function exigir(condicion) { if (!condicion) throw new TypeError("contrato_vinculo_categoria_rpt_invalido"); }
const entero = (v, minimo = 1) => Number.isSafeInteger(v) && v >= minimo;
const texto = (v) => typeof v === "string" && v.length > 0 && v.length <= 320 && !/[\u0000-\u001f\u007f]/u.test(v);

function vinculo(v) {
  const x = registroB2(v, CAMPOS_VINCULO);
  exigir(entero(x.revision) && referenciaB2(x.recibo_ref) && CATEGORIA.test(x.catalogo_id) && CATEGORIA.test(x.modulo_id)
    && entero(x.catalogo_version) && SHA.test(x.catalogo_huella_sha256) && CATEGORIA.test(x.categoria_id)
    && ["fuente_ref", "motivo_ref", "aprobacion_ref"].every((c) => referenciaB2(x[c]))
    && x.prospectivo === true && x.acredita_procedencia_historica === false);
  return Object.freeze(x);
}

/** Lectura del GET: anclaje del análisis confirmado y vínculo vigente, si existe. */
export function validarLecturaVinculoRPT(v, expedienteRef) {
  const l = registroB2(v, ["organizacion_ref", "expediente_ref", "analisis", "vinculo"]);
  const a = registroB2(l.analisis, ["version_expediente", "analisis_version", "analisis_recibo_ref", "analisis_huella_sha256", "categoria_ref"]);
  exigir(referenciaB2(l.organizacion_ref) && l.expediente_ref === expedienteRef && entero(a.version_expediente, 2)
    && entero(a.analisis_version, 2) && a.analisis_version <= a.version_expediente && referenciaB2(a.analisis_recibo_ref)
    && SHA.test(a.analisis_huella_sha256) && referenciaB2(a.categoria_ref));
  return Object.freeze({ expediente_ref: l.expediente_ref, analisis: Object.freeze(a), vinculo: l.vinculo === null ? null : vinculo(l.vinculo) });
}

/** Página del listado de categorías publicadas, en orden estricto de clave. */
export function validarPaginaCategoriasRPT(v, cursor = "") {
  const p = registroB2(v, ["categorias", "hay_mas", "siguiente_cursor"]);
  exigir(Array.isArray(p.categorias) && p.categorias.length <= 100 && typeof p.hay_mas === "boolean" && typeof p.siguiente_cursor === "string");
  let anterior = cursor;
  const categorias = p.categorias.map((bruta) => {
    const c = registroB2(bruta, ["categoria_id", "etiqueta", "atributos", "catalogo_version", "catalogo_huella_sha256", "fuente_ref", "aprobacion_ref"]);
    exigir(CATEGORIA.test(c.categoria_id) && c.categoria_id > anterior && texto(c.etiqueta) && entero(c.catalogo_version)
      && SHA.test(c.catalogo_huella_sha256) && referenciaB2(c.fuente_ref) && referenciaB2(c.aprobacion_ref)
      && c.atributos && Object.getPrototypeOf(c.atributos) === Object.prototype
      && Object.values(c.atributos).every(texto));
    anterior = c.categoria_id;
    return Object.freeze({ ...c, atributos: Object.freeze({ ...c.atributos }) });
  });
  exigir(p.hay_mas ? categorias.length > 0 && p.siguiente_cursor === anterior : p.siguiente_cursor === "");
  return Object.freeze({ categorias: Object.freeze(categorias), hay_mas: p.hay_mas, siguiente_cursor: p.siguiente_cursor });
}

export function validarEntradaVinculoRPT(v) {
  const e = registroB2(v, CAMPOS_ENTRADA);
  exigir(referenciaB2(e.expediente_ref) && entero(e.version_expediente_esperada, 2) && entero(e.analisis_version, 2)
    && referenciaB2(e.analisis_recibo_ref) && SHA.test(e.analisis_huella_sha256) && CATEGORIA.test(e.categoria_id)
    && e.categoria_ref === e.categoria_id && entero(e.catalogo_version) && SHA.test(e.catalogo_huella_sha256)
    && ["fuente_ref", "motivo_ref", "aprobacion_ref"].every((c) => referenciaB2(e[c]))
    && entero(e.revision_esperada, 0) && (e.revision_esperada === 0 ? e.anterior_recibo_ref === "" : referenciaB2(e.anterior_recibo_ref))
    && UUID.test(e.clave_idempotencia));
  return Object.freeze(e);
}

/** Recibo del POST, ligado exactamente a la intención enviada. */
export function validarReciboVinculoRPT(v, entrada) {
  const r = registroB2(v, ["expediente_ref", "recibo_ref", "registrado_en", "revision", "prospectivo", "acredita_procedencia_historica", "vinculo"]);
  const x = vinculo(r.vinculo);
  exigir(r.expediente_ref === entrada.expediente_ref && referenciaB2(r.recibo_ref) && r.recibo_ref === x.recibo_ref
    && typeof r.registrado_en === "string" && Number.isFinite(Date.parse(r.registrado_en))
    && r.revision === entrada.revision_esperada + 1 && x.revision === r.revision
    && x.categoria_id === entrada.categoria_id && x.catalogo_version === entrada.catalogo_version
    && x.catalogo_huella_sha256 === entrada.catalogo_huella_sha256 && r.prospectivo === true && r.acredita_procedencia_historica === false);
  return Object.freeze({ ...r, vinculo: x });
}
