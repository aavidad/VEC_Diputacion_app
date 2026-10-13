import test from "node:test";
import assert from "node:assert/strict";
import { cargarTextos } from "../comun/textos.js";
import { validarPerfilPropioBolsa, renderizarConfiguracionBolsa } from "./portal-bolsa-configuracion.js";
const textos = await cargarTextos("bolsa-configuracion");
const esc = (v) => String(v ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;");
const perfil = { esquema: "vec.bolsa.rrhh.perfil.v1", rol: { nombre: "Gestión de Bolsa", version: 9 }, ambito: { unidad_ref: "unidad:rrhh", ambito_ref: "ambito:bolsa" }, acciones: ["bolsa.contacto_participacion.registrar"], canales: [], consultada_en: "2026-10-13T10:00:00Z" };
test("el perfil exige acuse temporal y contrato propio sin datos de otras personas", () => {
  assert.equal(validarPerfilPropioBolsa({ data: perfil }).rol.version, 9);
  for (const dato of [{ ...perfil, consultada_en: "" }, { ...perfil, persona: "ajena" }, { ...perfil, acciones: ["*" ] }, { ...perfil, rol: { ...perfil.rol, version: 0 } }]) assert.throws(() => validarPerfilPropioBolsa({ data: dato }));
});
test("el fallo de perfil no oculta el correo y las referencias internas no se muestran", () => {
  const estado = { perfil: { carga: "listo", datos: perfil }, reglas: { carga: "error", datos: null }, cese: { carga: "cargando", datos: null }, correo: { carga: "listo", datos: { asunto: "Llamamiento <A>", cuerpo: "Primera línea\nSegunda línea" } }, ayuda: false };
  const html = renderizarConfiguracionBolsa(estado, { textos, escaparHTML: esc });
  assert.match(html, /Registrar contactos con la persona llamada/u); assert.match(html, /Llamamiento &lt;A&gt;/u);
  assert.match(html, /id="configuracion-ayuda" hidden/u); assert.doesNotMatch(html, /unidad:rrhh|ambito:bolsa|bolsa\.contacto_participacion\.registrar/u);
  const denegado = renderizarConfiguracionBolsa({ ...estado, perfil: { carga: "denegado", datos: null } }, { textos, escaparHTML: esc });
  assert.match(denegado, /Su perfil no permite consultar este apartado/u); assert.match(denegado, /Llamamiento &lt;A&gt;/u);
});
