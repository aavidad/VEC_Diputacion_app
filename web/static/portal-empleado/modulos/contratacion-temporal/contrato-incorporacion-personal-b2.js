/** Contrato HTTP minimizado; los hechos propietarios nunca se aceptan del formulario. */
export const ESQUEMA_CONSULTA_B2 = "vec.contratacion-temporal.incorporacion-personal-b2.consulta.v1";
export const ESQUEMA_RECIBO_B2 = "vec.contratacion-temporal.incorporacion-personal-b2.recibo.v1";
const REF = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const SHA = /^[0-9a-f]{64}$/u;
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/u;
const CLAVE = /^[a-z][a-z0-9_.:-]{1,159}$/u;
const CAMPOS_PLAN = ["expediente_ref", "version_expediente", "puesto_ref", "plaza_ref", "version_plantilla_ref",
  "version_rpt_ref", "regimen", "modalidad", "clase_ocupacion", "desde", "hasta", "motivo_clave", "documento_ref", "documento_sha256", "clave_idempotencia"];
function fallo() { throw new TypeError("contrato_incorporacion_personal_b2_invalido"); }
export function registroB2(v, campos) {
  if (!v || Object.getPrototypeOf(v) !== Object.prototype || Object.getOwnPropertySymbols(v).length
    || Object.keys(v).length !== campos.length || !campos.every((c) => Object.hasOwn(v, c))) fallo();
  return Object.fromEntries(campos.map((c) => [c, v[c]]));
}
export function referenciaB2(v) { return typeof v === "string" && REF.test(v); }
function clave(v) { return typeof v === "string" && CLAVE.test(v); }
function version(v) { return Number.isSafeInteger(v) && v > 0; }
function exigir(condicion) { if (!condicion) fallo(); }
export function fechaCivilB2(v) {
  return typeof v === "string" && /^(?!0000)\d{4}-\d{2}-\d{2}$/u.test(v)
    && Number.isFinite(Date.parse(v)) && new Date(v).toISOString().slice(0, 10) === v;
}
function periodo(v, vacio = false) {
  exigir(vacio && v.desde === "" && v.hasta === "" || fechaCivilB2(v.desde)
    && (v.hasta === "" || fechaCivilB2(v.hasta) && v.hasta > v.desde));
}
function catalogo(v) {
  const c = registroB2(v, ["ref", "version"]); exigir(referenciaB2(c.ref) && version(c.version)); return Object.freeze(c);
}
function lista(v, maximo, validar, identidad) {
  exigir(Array.isArray(v) && v.length <= maximo);
  const a = Array.from(v, validar);
  if (identidad) exigir(new Set(a.map(identidad)).size === a.length);
  return Object.freeze(a);
}
function etiqueta(v) { return typeof v === "string" && v.length > 0 && v.length <= 320 && !/[\u0000-\u001f\u007f]/u.test(v); }
export function validarSolicitudPlanB2(v) {
  const p = registroB2(v, CAMPOS_PLAN);
  exigir(["expediente_ref", "version_plantilla_ref", "version_rpt_ref", "documento_ref"].every((c) => referenciaB2(p[c]))
    && ["puesto_ref", "plaza_ref"].every((c) => referenciaB2(p[c]) || UUID.test(p[c]))
    && version(p.version_expediente) && clave(p.clase_ocupacion) && clave(p.motivo_clave) && SHA.test(p.documento_sha256)
    && UUID.test(p.clave_idempotencia));
  p.regimen = catalogo(p.regimen); p.modalidad = catalogo(p.modalidad); periodo(p);
  return Object.freeze(p);
}
export function validarSolicitudConfirmacionB2(v) {
  const p = registroB2(v, ["expediente_ref", "plan_ref", "version_plan", "clave_idempotencia"]);
  exigir(referenciaB2(p.expediente_ref) && referenciaB2(p.plan_ref) && version(p.version_plan) && UUID.test(p.clave_idempotencia));
  return Object.freeze(p);
}
export function validarReciboB2(v, contexto) {
  const r = registroB2(v, ["esquema", "expediente_ref", "plan_ref", "plan_version", "recibo_ref", "registrada_en",
    "empleado_ref", "relacion_ref", "ocupacion_ref", "firma_oficial", "eficacia_administrativa"]);
  exigir(r.esquema === ESQUEMA_RECIBO_B2 && r.expediente_ref === contexto.expediente_ref
    && ["expediente_ref", "plan_ref", "recibo_ref", "empleado_ref", "relacion_ref", "ocupacion_ref"].every((c) => referenciaB2(r[c]))
    && version(r.plan_version) && r.firma_oficial === false && r.eficacia_administrativa === false
    && typeof r.registrada_en === "string" && /^(?!0000)\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/u.test(r.registrada_en)
    && Number.isFinite(Date.parse(r.registrada_en)) && new Date(r.registrada_en).toISOString().slice(0, 19) === r.registrada_en.slice(0, 19));
  if (contexto.plan_ref !== undefined) exigir(r.plan_ref === contexto.plan_ref && r.plan_version === (contexto.version_plan ?? contexto.version));
  return Object.freeze(r);
}
function validarOpciones(v) {
  const o = registroB2(v, ["vacantes", "regimenes", "modalidades", "catalogo_clases_ocupacion", "clases_ocupacion", "motivos", "documentos", "periodo"]);
  o.vacantes = lista(o.vacantes, 100, (v) => {
    const x = registroB2(v, ["plaza_ref", "puesto_ref", "version_plantilla_ref", "version_rpt_ref", "unidad_ref", "categoria_ref", "plaza_etiqueta", "puesto_etiqueta"]);
    exigir(["plaza_ref", "puesto_ref", "version_plantilla_ref", "version_rpt_ref", "unidad_ref", "categoria_ref"].every((c) => referenciaB2(x[c]))
      && etiqueta(x.plaza_etiqueta) && etiqueta(x.puesto_etiqueta)); return Object.freeze(x);
  }, (x) => [x.plaza_ref, x.puesto_ref, x.version_plantilla_ref, x.version_rpt_ref].join("|"));
  for (const k of ["regimenes", "modalidades"]) o[k] = lista(o[k], 100, (v) => {
    const x = registroB2(v, ["ref", "version", "denominacion"]);
    exigir(referenciaB2(x.ref) && version(x.version) && etiqueta(x.denominacion)); return Object.freeze(x);
  }, (x) => `${x.ref}|${x.version}`);
  o.catalogo_clases_ocupacion = registroB2(o.catalogo_clases_ocupacion, ["ref", "version", "huella_sha256"]);
  const catalogoClases = o.catalogo_clases_ocupacion;
  const clasesEntrada = o.clases_ocupacion === null ? [] : o.clases_ocupacion;
  exigir(o.clases_ocupacion !== null || catalogoClases.ref === "");
  exigir(referenciaB2(catalogoClases.ref) && version(catalogoClases.version) && SHA.test(catalogoClases.huella_sha256)
    || catalogoClases.ref === "" && catalogoClases.version === 0 && catalogoClases.huella_sha256 === "" && clasesEntrada.length === 0);
  o.catalogo_clases_ocupacion = Object.freeze(catalogoClases);
  o.clases_ocupacion = lista(clasesEntrada, 100, (v) => {
    const x = registroB2(v, ["valor", "texto_clave"]);
    exigir(clave(x.valor) && clave(x.texto_clave)); return Object.freeze(x);
  }, (x) => x.valor);
  o.motivos = lista(o.motivos, 32, (x) => { exigir(clave(x)); return x; }, (x) => x);
  o.documentos = lista(o.documentos, 32, (v) => {
    const x = registroB2(v, ["documento_ref", "documento_sha256", "etiqueta_clave_i18n"]);
    exigir(referenciaB2(x.documento_ref) && SHA.test(x.documento_sha256) && clave(x.etiqueta_clave_i18n)); return Object.freeze(x);
  }, (x) => x.documento_ref);
  o.periodo = registroB2(o.periodo, ["desde", "hasta", "fuente_ref"]); periodo(o.periodo, true);
  exigir(o.periodo.fuente_ref === "" || referenciaB2(o.periodo.fuente_ref)); o.periodo = Object.freeze(o.periodo);
  return Object.freeze(o);
}
export function validarConsultaB2(v, expedienteRef) {
  const c = registroB2(v, ["esquema", "expediente_ref", "version_expediente_actual", "estado", "prerrequisitos", "opciones", "plan", "recibo"]);
  exigir(c.esquema === ESQUEMA_CONSULTA_B2 && referenciaB2(expedienteRef) && c.expediente_ref === expedienteRef
    && version(c.version_expediente_actual) && ["sin_plan", "plan_preparado", "incorporacion_confirmada"].includes(c.estado)
    && ((c.estado === "sin_plan") === (c.plan === null)) && ((c.estado === "incorporacion_confirmada") === (c.recibo !== null)));
  c.prerrequisitos = lista(c.prerrequisitos, 32, (v) => {
    const p = registroB2(v, ["clave_i18n", "cumplido"]); exigir(clave(p.clave_i18n) && typeof p.cumplido === "boolean"); return Object.freeze(p);
  }, (x) => x.clave_i18n);
  c.opciones = validarOpciones(c.opciones);
  if (c.plan !== null) {
    const p = registroB2(c.plan, ["plan_ref", "version", "sha256", "intencion"]);
    exigir(referenciaB2(p.plan_ref) && version(p.version) && SHA.test(p.sha256)); p.intencion = validarSolicitudPlanB2(p.intencion);
    exigir(p.intencion.expediente_ref === expedienteRef && p.intencion.version_expediente <= c.version_expediente_actual);
    c.plan = Object.freeze(p);
  }
  if (c.recibo !== null) c.recibo = validarReciboB2(c.recibo, { expediente_ref: expedienteRef, ...c.plan });
  return Object.freeze(c);
}
