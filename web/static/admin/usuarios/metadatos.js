const REF = /^[A-Za-z0-9][A-Za-z0-9_:-]{2,127}$/u;
const PERSONA = /^per_[A-Za-z0-9_-]{22,124}$/u;
const ROL = /^rol:[A-Za-z0-9_-]+:v[1-9][0-9]*$/u;
const estados = new Set(["vigente", "caducado", "revocado", "pendiente"]);
function exigir(valor) { if (!valor) throw Object.assign(new Error("respuesta_incompatible"), { codigo: "respuesta_incompatible" }); }
function cerrado(valor, obligatorias, opcionales = []) {
  exigir(valor && typeof valor === "object" && !Array.isArray(valor));
  exigir(obligatorias.every((k) => Object.hasOwn(valor, k)) && Object.keys(valor).every((k) => obligatorias.includes(k) || opcionales.includes(k)));
}
const version = (v) => Number.isSafeInteger(v) && v > 0;
const fecha = (v) => typeof v === "string" && /^\d{4}-\d{2}-\d{2}T/u.test(v) && Number.isFinite(Date.parse(v));
function persona(p, ficha = false) {
  cerrado(p, ["persona_ref", "unidad_ref", "denominacion_version", "nombre_estado", "perfiles", ...(ficha ? ["proyeccion", "historia_estado", "actos_estado"] : [])], ["nombre"]);
  exigir(PERSONA.test(p.persona_ref) && REF.test(p.unidad_ref) && Array.isArray(p.perfiles));
  exigir(["no_registrado", "no_consultado", "consultado"].includes(p.nombre_estado));
  if (p.nombre_estado === "no_registrado") exigir(p.denominacion_version === null);
  else exigir(version(p.denominacion_version));
  if (p.nombre_estado === "consultado") exigir(typeof p.nombre === "string" && p.nombre.trim().length > 0 && new TextEncoder().encode(p.nombre).byteLength <= 4096);
  else exigir(!Object.hasOwn(p, "nombre"));
  const vistos = new Set();
  for (const perfil of p.perfiles) {
    cerrado(perfil, ["perfil_ref", "rol_version_ref", "version", "estado", "vigente_desde", "vigente_hasta"]);
    exigir(REF.test(perfil.perfil_ref) && ROL.test(perfil.rol_version_ref) && version(perfil.version) && estados.has(perfil.estado)
      && fecha(perfil.vigente_desde) && fecha(perfil.vigente_hasta) && Date.parse(perfil.vigente_hasta) > Date.parse(perfil.vigente_desde) && !vistos.has(perfil.perfil_ref));
    vistos.add(perfil.perfil_ref);
  }
  if (ficha) exigir(p.proyeccion === "metadatos_v1" && p.historia_estado === "no_consultada" && p.actos_estado === "no_consultados");
  return { ...structuredClone(p), proyeccion: "metadatos_v1" };
}
export function validarPaginaMetadatos(pagina) {
  cerrado(pagina, ["proyeccion", "personas"], ["siguiente_cursor"]);
  exigir(pagina.proyeccion === "metadatos_v1" && Array.isArray(pagina.personas) && pagina.personas.length <= 50);
  exigir(pagina.siguiente_cursor === undefined || typeof pagina.siguiente_cursor === "string" && pagina.siguiente_cursor.length <= 256);
  const personas = pagina.personas.map((p) => persona(p));
  exigir(new Set(personas.map((p) => p.persona_ref)).size === personas.length);
  return { ...pagina, personas };
}
export function validarFichaMetadatos(datos, referencia) {
  const ficha = persona(datos, true);
  exigir(ficha.persona_ref === referencia);
  return ficha;
}
export function etiquetaNombreMetadatos(datos, traducir) {
  return datos.nombre_estado === "consultado" ? datos.nombre : traducir(`metadatos.nombre_${datos.nombre_estado}`);
}
