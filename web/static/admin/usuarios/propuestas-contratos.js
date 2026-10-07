import { incompatible } from "./contratos.js?v=20261004-admin-usuarios-metadata-v1";
const SHA = /^[a-f0-9]{64}$/u;
const REF = /^[A-Za-z0-9][A-Za-z0-9_:.-]{2,255}$/u;
const ref = (v) => typeof v === "string" && REF.test(v);
const PROPUESTA = /^propuesta_admin:[a-f0-9]{32}$/u;
const exigir = (v) => { if (!v) throw incompatible(); };
const fecha = (v) => typeof v === "string" && /^\d{4}-\d{2}-\d{2}T/u.test(v) && Number.isFinite(Date.parse(v));
const nombre = (v) => typeof v === "string" && v.trim().length > 0 && v.length <= 500;
const motivoValido = (m) => ref(m?.catalogo_id) && Number.isSafeInteger(m.catalogo_version)
  && m.catalogo_version > 0 && SHA.test(m.catalogo_huella_sha256) && ref(m.entrada_clave) && nombre(m.etiqueta);
export function revisionPropuestaCompleta(p) {
  return Boolean(nombre(p?.proponente_nombre) && nombre(p.proponente_perfil_nombre) && Array.isArray(p.ambitos) && p.ambitos.length > 0 && p.ambitos.length <= 32
    && p.ambitos.every((a) => ref(a?.dimension) && ref(a.referencia) && nombre(a.nombre))
    && fecha(p.vigente_desde) && fecha(p.vigente_hasta) && !p.vigente_desde.startsWith("0001-")
    && Date.parse(p.vigente_hasta) > Date.parse(p.vigente_desde) && motivoValido(p.motivo));
}
export function validarPropuestas(datos) {
  exigir(Array.isArray(datos?.propuestas) && datos.propuestas.length <= 200);
  exigir(new Set(datos.propuestas.map((p) => p?.propuesta_ref)).size === datos.propuestas.length);
  for (const p of datos.propuestas) {
    exigir(PROPUESTA.test(p?.propuesta_ref) && SHA.test(p.huella_sha256) && fecha(p.caduca_en)
      && ref(p.proponente_persona_ref) && ref(p.objetivo_persona_ref) && nombre(p.objetivo_nombre)
      && ref(p.rol_version_ref) && ["otorgar", "revocar"].includes(p.operacion)
      && typeof p.puede_cerrar === "boolean" && Array.isArray(p.motivos_cierre) && p.motivos_cierre.length <= 64);
    if (p.proponente_nombre) exigir(nombre(p.proponente_nombre));
    if (p.proponente_perfil_nombre) exigir(nombre(p.proponente_perfil_nombre));
    if (p.ambitos !== undefined && p.ambitos !== null) exigir(Array.isArray(p.ambitos) && p.ambitos.length <= 32
      && p.ambitos.every((a) => ref(a?.dimension) && ref(a.referencia) && nombre(a.nombre)));
    for (const k of ["vigente_desde", "vigente_hasta"]) if (p[k] !== undefined && p[k] !== null) exigir(typeof p[k] === "string" && (!p[k] || fecha(p[k])));
    if (p.motivo !== undefined && p.motivo !== null) exigir(motivoValido(p.motivo));
    for (const m of p.motivos_cierre) exigir(ref(m?.catalogo_id) && Number.isSafeInteger(m.catalogo_version)
      && m.catalogo_version > 0 && SHA.test(m.catalogo_huella_sha256) && ref(m.entrada_clave) && nombre(m.etiqueta));
  }
  return structuredClone(datos.propuestas);
}
export function puedeCerrarPropuesta(p, actor, capacidades, cliente) {
  return Boolean(p?.puede_cerrar && revisionPropuestaCompleta(p) && p.proponente_persona_ref !== actor && p.objetivo_persona_ref !== actor
    && capacidades.includes("cerrar_propuesta") && typeof cliente?.cerrarPropuesta === "function" && p.motivos_cierre.length);
}
export function prepararCierre(p, actor, capacidades, cliente, decision, indiceMotivo, cripto = globalThis.crypto) {
  exigir(puedeCerrarPropuesta(p, actor, capacidades, cliente) && ["aprobar", "rechazar"].includes(decision)
    && Number.isSafeInteger(indiceMotivo) && p.motivos_cierre[indiceMotivo] && cripto?.getRandomValues);
  const motivo = p.motivos_cierre[indiceMotivo];
  const id = Array.from(cripto.getRandomValues(new Uint8Array(16)), (n) => n.toString(16).padStart(2, "0")).join("");
  return { propuesta: structuredClone(p), actor, decision, motivo: motivo.etiqueta,
    cuerpo: { operacion_ref: `cierre_admin:${id}`, propuesta_huella_sha256: p.huella_sha256, decision: decision === "aprobar" ? "aprobada" : "rechazada",
      motivo: Object.fromEntries(["catalogo_id", "catalogo_version", "catalogo_huella_sha256", "entrada_clave"].map((k) => [k, motivo[k]])) } };
}
export function validarCierre(datos, revision) {
  const c = datos?.cierre, p = revision.propuesta, body = revision.cuerpo;
  exigir(c?.operacion_ref === body.operacion_ref && c.propuesta_ref === p.propuesta_ref
    && c.propuesta_huella_sha256 === p.huella_sha256 && c.decision === body.decision
    && SHA.test(c.huella_cierre_sha256) && fecha(c.confirmado_en));
  if (body.decision === "rechazada") exigir(c.recibo === undefined || c.recibo === null);
  else {
    const r = c.recibo;
    exigir(r?.operacion_ref === body.operacion_ref && r.propuesta_ref === p.propuesta_ref
      && r.actor_persona_ref === revision.actor && r.objetivo_persona_ref === p.objetivo_persona_ref
      && r.rol_version_ref === p.rol_version_ref && /^recibo_admin:[a-f0-9]{32}$/u.test(r.recibo_ref)
      && ref(r.perfil_ref) && ref(r.auditoria_ref) && Number.isSafeInteger(r.version_posterior) && r.version_posterior > 0
      && r.estado_posterior === (p.operacion === "otorgar" ? "activo" : "revocado")
      && SHA.test(r.huella_antes_sha256) && SHA.test(r.huella_despues_sha256)
      && fecha(r.confirmado_en) && Date.parse(r.confirmado_en) === Date.parse(c.confirmado_en)
      && Object.entries(body.motivo).every(([k, v]) => r.motivo?.[k] === v));
  }
  return { decision: body.decision, fecha: c.confirmado_en, referencia: c.recibo?.recibo_ref || c.operacion_ref };
}
