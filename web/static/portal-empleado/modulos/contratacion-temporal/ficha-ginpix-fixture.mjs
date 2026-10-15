const sha = "a".repeat(64);
const exp = "expediente:ct:001";
const origen = "recibo:incorporacion-personal-b2:001";
export function consultaB2GINPIX() {
  return { esquema: "vec.contratacion-temporal.incorporacion-personal-b2.consulta.v1", expediente_ref: exp,
    version_expediente_actual: 9, estado: "incorporacion_confirmada", prerrequisitos: [],
    opciones: { vacantes: [], regimenes: [], modalidades: [], catalogo_clases_ocupacion: { ref: "", version: 0, huella_sha256: "" },
      clases_ocupacion: [], motivos: [], documentos: [], periodo: { desde: "", hasta: "", fuente_ref: "" } },
    plan: { plan_ref: "plan:b2:001", version: 1, sha256: sha,
      intencion: { expediente_ref: exp, version_expediente: 8, puesto_ref: "puesto:001", plaza_ref: "plaza:001",
        version_plantilla_ref: "plantilla:001", version_rpt_ref: "rpt:001", regimen: { ref: "regimen:001", version: 1 },
        modalidad: { ref: "modalidad:001", version: 1 }, clase_ocupacion: "temporal", desde: "2026-10-01", hasta: "",
        motivo_clave: "incorporar", documento_ref: "documento:001", documento_sha256: sha,
        clave_idempotencia: "10000000-0000-4000-8000-000000000001" } },
    recibo: { esquema: "vec.contratacion-temporal.incorporacion-personal-b2.recibo.v1", expediente_ref: exp,
      plan_ref: "plan:b2:001", plan_version: 1, recibo_ref: origen, registrada_en: "2026-10-01T10:00:00Z",
      empleado_ref: "empleado:001", relacion_ref: "relacion:001", ocupacion_ref: "ocupacion:001",
      firma_oficial: false, eficacia_administrativa: false } };
}
export function ficheroB2GINPIX() {
  return { esquema: "vec.dipgra.contratacion-temporal.ginpix.fichero.v1", version: 1,
    metadatos: { esquema_modelo: "vec.dipgra.contratacion-temporal.ginpix.modelo.v1",
      esquema_mapeo: "vec.dipgra.contratacion-temporal.ginpix.mapeo.v1",
      esquema_carga: "vec.dipgra.contratacion-temporal.ginpix.carga.v1", version_expediente: 9,
      expediente_ref: exp, incorporacion_ref: origen, procedencia_modelo_ref: origen,
      correlacion_ref: "solicitud:personal:001", idempotencia_ref: "10000000-0000-4000-8000-000000000002",
      huella_modelo_sha256: sha, mapeo_ref: "mapeo:ct:ginpix:ficha-personal-b2", mapeo_version: 1,
      procedencia_mapeo_ref: "configuracion:ct:ginpix:ficha-trazabilidad-personal-b2:v1",
      huella_mapeo_sha256: sha, huella_carga_sha256: sha },
    campos: [
      { clave: "actuacion_ref", estado: "valor", valor: origen },
      { clave: "expediente_ref", estado: "valor", valor: exp },
      { clave: "recibo_ref", estado: "valor", valor: "recibo:personal:alta:001" },
      { clave: "relacion_ref", estado: "valor", valor: "relacion:001" },
      { clave: "seguimiento_ref", estado: "valor", valor: "recibo:personal:ejecucion:001" },
      { clave: "solicitud_personal_ref", estado: "valor", valor: "solicitud:personal:001" },
    ] };
}
