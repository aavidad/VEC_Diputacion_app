const SHA = /^[a-f0-9]{64}$/u;
const REF = /^[A-Za-z0-9][A-Za-z0-9_:.-]{2,255}$/u;
const ACCIONES = new Set(["consultar", "aplicar_ordinario", "proponer", "cerrar_propuesta", "aplicar_lote", "proponer_lote"]);
const CLASES = new Set(["ordinario", "administrador", "intervencion"]);
const ESTADOS = new Set(["activo", "caducado", "revocado", "pendiente"]);
export function incompatible() { return Object.assign(new Error("respuesta_incompatible"), { codigo: "respuesta_incompatible" }); }
const exigir = (valor) => { if (!valor) throw incompatible(); };
const texto = (valor) => typeof valor === "string" && valor.trim().length > 0 && valor.length <= 500;
const ref = (valor) => typeof valor === "string" && REF.test(valor);
const fecha = (valor) => typeof valor === "string" && /^\d{4}-\d{2}-\d{2}T/u.test(valor) && Number.isFinite(Date.parse(valor));
const version = (valor) => Number.isSafeInteger(valor) && valor > 0;
function lista(valor, maximo = 200) { exigir(Array.isArray(valor) && valor.length <= maximo); return valor; }
function unicos(valores, campo) { exigir(new Set(valores.map((v) => v[campo])).size === valores.length); return valores; }
export function validarCapacidades(datos) {
  exigir(datos?.version === "v1" && ref(datos.actor_persona_ref));
  const acciones = lista(datos.acciones, 6); exigir(acciones.every((a) => ACCIONES.has(a)) && new Set(acciones).size === acciones.length);
  return Object.freeze({ ...datos, acciones: Object.freeze([...acciones]) });
}
export function validarRoles(datos) {
  const roles = unicos(lista(datos?.roles, 128), "version_ref");
  for (const rol of roles) exigir(ref(rol?.version_ref) && texto(rol.etiqueta) && CLASES.has(rol.clase)
    && SHA.test(rol.huella_sha256) && typeof rol.fijo === "boolean");
  return Object.freeze(roles.map((r) => Object.freeze({ ...r })));
}
export function validarUnidades(datos) {
  const unidades = unicos(lista(datos?.unidades, 128), "unidad_ref");
  for (const u of unidades) exigir(ref(u?.unidad_ref) && texto(u.nombre));
  return Object.freeze(unidades.map((u) => Object.freeze({ ...u })));
}
export function validarPersonas(datos) {
  const personas = unicos(lista(datos?.personas), "persona_ref");
  for (const persona of personas) {
    exigir(ref(persona?.persona_ref) && texto(persona.nombre) && texto(persona.unidad_nombre));
    if (persona.unidad_ref !== undefined) exigir(ref(persona.unidad_ref));
    if (persona.perfiles !== undefined) for (const p of lista(persona.perfiles, 128)) exigir(ref(p.rol_version_ref) && ESTADOS.has(p.estado));
  }
  if (datos.siguiente_cursor !== undefined) exigir(typeof datos.siguiente_cursor === "string" && datos.siguiente_cursor.length <= 256);
  return { ...datos, personas: personas.map((p) => ({ ...p })) };
}
function validarMotivo(m) {
  exigir(ref(m?.catalogo_id) && version(m.catalogo_version) && SHA.test(m.catalogo_huella_sha256)
    && ref(m.entrada_clave) && texto(m.etiqueta));
}
export function validarFicha(datos, personaRef) {
  exigir(datos?.persona_ref === personaRef && ref(personaRef) && texto(datos.nombre) && texto(datos.unidad_nombre));
  const perfiles = unicos(lista(datos.perfiles, 128), "perfil_ref");
  for (const p of perfiles) {
    exigir(ref(p?.perfil_ref) && ref(p.rol_version_ref) && ESTADOS.has(p.estado) && version(p.version) && fecha(p.vigente_hasta));
    if (p.vigente_desde !== undefined) exigir(fecha(p.vigente_desde));
    if (p.ambito_etiqueta !== undefined) exigir(texto(p.ambito_etiqueta));
  }
  for (const acto of lista(datos.actos_disponibles, 128)) {
    exigir(["otorgar", "revocar"].includes(acto?.operacion) && ref(acto.rol_version_ref)
      && acto.objetivo?.persona_ref === personaRef && ref(acto.objetivo.perfil_ref) && SHA.test(acto.objetivo.huella_sha256));
    for (const campo of ["cuenta", "persona", "perfil", "vinculo", "procedencia"]) {
      exigir(ref(acto.objetivo[`${campo}_ref`]) && version(acto.objetivo[`${campo}_version`]));
    }
    exigir(version(acto.objetivo.revision_continuidad) && SHA.test(acto.objetivo.procedencia_huella_sha256) && fecha(acto.objetivo.vigente_hasta));
    for (const motivo of lista(acto.motivos, 64)) validarMotivo(motivo);
    if (acto.operacion === "revocar") exigir(perfiles.some((p) => p.perfil_ref === acto.objetivo.perfil_ref && p.rol_version_ref === acto.rol_version_ref));
  }
  for (const item of lista(datos.historia)) {
    exigir(["otorgar", "revocar"].includes(item?.operacion) && fecha(item.confirmado_en) && ref(item.acto_ref));
    for (const identidad of [item.actor, item.proponente, item.aprobador].filter(Boolean)) {
      exigir(ref(identidad.persona_ref) && texto(identidad.nombre) && ref(identidad.perfil_activo_ref) && ref(identidad.asignacion_ref));
      if (identidad.perfil_activo_nombre !== undefined) exigir(texto(identidad.perfil_activo_nombre));
    }
    if (item.objetivo_nombre !== undefined) exigir(texto(item.objetivo_nombre));
    if (item.motivo !== undefined) validarMotivo(item.motivo);
  }
  return structuredClone(datos);
}
export function seleccionarActos(ficha, roles, capacidades, operacion, indices) {
  exigir(["otorgar", "revocar"].includes(operacion) && Array.isArray(indices) && indices.length > 0 && indices.length <= 32);
  exigir(new Set(indices).size === indices.length);
  return indices.map((indice) => {
    const acto = ficha?.actos_disponibles?.[indice];
    const rol = roles.find((r) => r.version_ref === acto?.rol_version_ref);
    exigir(acto?.operacion === operacion && rol && !rol.fijo && acto.motivos.length > 0);
    if (operacion === "otorgar") exigir(fecha(acto.objetivo.vigente_desde));
    exigir(capacidades.includes(rol.clase === "ordinario" ? "aplicar_ordinario" : "proponer"));
    if (operacion === "revocar") exigir(ficha.perfiles.some((p) => p.perfil_ref === acto.objetivo.perfil_ref && p.estado === "activo"));
    return { indice, acto, rol };
  });
}
export function prepararDecision(ficha, roles, capacidades, operacion, indices, motivos, cripto = globalThis.crypto) {
  const seleccion = seleccionarActos(ficha, roles, capacidades, operacion, indices);
  const sensible = seleccion.some((s) => s.rol.clase !== "ordinario");
  exigir(cripto?.getRandomValues);
  const id = Array.from(cripto.getRandomValues(new Uint8Array(16)), (n) => n.toString(16).padStart(2, "0")).join("");
  const solicitudes = seleccion.map(({ indice, acto }) => {
    const motivo = acto.motivos[motivos[indice]]; exigir(motivo);
    return { operacion: acto.operacion, rol_version_ref: acto.rol_version_ref, objetivo: structuredClone(acto.objetivo),
      motivo: Object.fromEntries(["catalogo_id", "catalogo_version", "catalogo_huella_sha256", "entrada_clave"].map((c) => [c, motivo[c]])) };
  });
  exigir(new Set(solicitudes.map((s) => s.objetivo.perfil_ref)).size === solicitudes.length);
  return Object.freeze({ sensible, seleccion, cuerpo: Object.freeze({ operacion_ref: `${sensible ? "propuesta_admin:" : "acto_admin:"}${id}`,
    ...(solicitudes.length === 1 ? solicitudes[0] : { solicitudes }) }) });
}
export function puedeConfirmar(decision, capacidades, cliente) {
  const lote = decision.seleccion.length > 1;
  const accion = lote ? (decision.sensible ? "proponer_lote" : "aplicar_lote") : (decision.sensible ? "proponer" : "aplicar_ordinario");
  const metodo = lote ? (decision.sensible ? "proponerLote" : "aplicarLote") : (decision.sensible ? "proponer" : "aplicar");
  return capacidades.includes(accion) && typeof cliente?.[metodo] === "function" ? metodo : null;
}
export function validarResultado(datos, decision) {
  const { cuerpo, seleccion, sensible } = decision;
  if (sensible) {
    const p = datos?.propuesta;
    exigir(p?.propuesta_ref === cuerpo.operacion_ref && /^propuesta_admin:[a-f0-9]{32}$/u.test(p.propuesta_ref)
      && SHA.test(p.huella_sha256) && fecha(p.caduca_en));
    return { tipo: "propuesta", referencia: p.propuesta_ref, fecha: p.caduca_en };
  }
  const r = datos?.recibo;
  exigir(r?.operacion_ref === cuerpo.operacion_ref && /^recibo_admin:[a-f0-9]{32}$/u.test(r.recibo_ref)
    && r.objetivo_persona_ref === seleccion[0].acto.objetivo.persona_ref && fecha(r.confirmado_en)
    && ref(r.auditoria_ref) && SHA.test(r.huella_antes_sha256) && SHA.test(r.huella_despues_sha256));
  const perfiles = seleccion.length === 1 ? [r] : lista(r.perfiles, 32);
  exigir(perfiles.length === seleccion.length && new Set(perfiles.map((p) => p.perfil_ref)).size === perfiles.length);
  for (const s of seleccion) exigir(perfiles.some((p) => p.perfil_ref === s.acto.objetivo.perfil_ref && version(p.version_posterior)
    && p.estado_posterior === (s.acto.operacion === "revocar" ? "revocado" : "activo")));
  return { tipo: "recibo", referencia: r.recibo_ref, fecha: r.confirmado_en };
}
