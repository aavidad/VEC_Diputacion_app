import test from "node:test";
import assert from "node:assert/strict";
import { cargarTextos } from "../comun/textos.js";
import { consultarPerfilPropioBolsa, RUTA_PERFIL_PROPIO_BOLSA, validarPerfilPropioBolsa, renderizarConfiguracionBolsa } from "./portal-bolsa-configuracion.js";
const textos = await cargarTextos("bolsa-configuracion");
const esc = (v) => String(v ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;");
const perfil = { esquema: "vec.bolsa.rrhh.perfil.v1", rol: { nombre: "Gestión de Bolsa", version: 9 }, ambito: { unidad_ref: "unidad:rrhh", ambito_ref: "ambito:bolsa" }, acciones: ["bolsa.contacto_participacion.registrar"], canales: [], consultada_en: "2026-10-13T10:00:00Z" };
test("el perfil exige acuse temporal y contrato propio sin datos de otras personas", () => {
  assert.equal(validarPerfilPropioBolsa({ data: perfil }).rol.version, 9);
  for (const dato of [{ ...perfil, consultada_en: "" }, { ...perfil, persona: "ajena" }, { ...perfil, acciones: ["*" ] }, { ...perfil, rol: { ...perfil.rol, version: 0 } }]) assert.throws(() => validarPerfilPropioBolsa({ data: dato }));
});
test("la lectura nominal es única, cancelable y no abre el cuadro de bolsas", async () => {
  const llamadas = [];
  const fetchImpl = async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    return new Response(JSON.stringify({ data: perfil }), { headers: { "Content-Type": "application/json" } });
  };
  assert.equal((await consultarPerfilPropioBolsa({ fetchImpl })).rol.version, 9);
  assert.equal(llamadas.length, 1);
  assert.equal(llamadas[0].ruta, RUTA_PERFIL_PROPIO_BOLSA);
  assert.equal(llamadas[0].opciones.method, "GET");
  assert.equal(llamadas[0].opciones.cache, "no-store");
  const cancelada = new AbortController(); cancelada.abort();
  await assert.rejects(consultarPerfilPropioBolsa({ signal: cancelada.signal, fetchImpl }));
  assert.equal(llamadas.length, 1, "una cancelación previa no emite GET");
  const enCurso = new AbortController(); let avisarInicio;
  const iniciada = new Promise((resolver) => { avisarInicio = resolver; });
  const pendiente = consultarPerfilPropioBolsa({ signal: enCurso.signal, fetchImpl: (_ruta, opciones) => {
    avisarInicio(opciones.signal);
    return new Promise((_resolver, rechazar) => opciones.signal.addEventListener("abort",
      () => rechazar(new DOMException("Cancelada", "AbortError")), { once: true }));
  } });
  const signal = await iniciada; enCurso.abort();
  assert.equal(signal.aborted, true);
  await assert.rejects(pendiente);
  for (const estado of [403, 404]) {
    const denegada = () => consultarPerfilPropioBolsa({ fetchImpl: async () =>
      new Response(JSON.stringify({ error: { codigo: "acceso_denegado" } }), { status: estado,
        headers: { "Content-Type": "application/json" } }) });
    await assert.rejects(denegada());
  }
});
test("el fallo de perfil no oculta el correo y las referencias internas no se muestran", () => {
  const estado = { perfil: { carga: "listo", datos: perfil }, reglas: { carga: "error", datos: null }, cese: { carga: "cargando", datos: null }, correo: { carga: "listo", datos: { asunto: "Llamamiento <A>", cuerpo: "Primera línea\nSegunda línea" } }, ayuda: false };
  const html = renderizarConfiguracionBolsa(estado, { textos, escaparHTML: esc });
  assert.match(html, /Registrar contactos con la persona llamada/u); assert.match(html, /Llamamiento &lt;A&gt;/u);
  assert.match(html, /id="configuracion-ayuda" hidden/u); assert.doesNotMatch(html, /unidad:rrhh|ambito:bolsa|bolsa\.contacto_participacion\.registrar/u);
  const denegado = renderizarConfiguracionBolsa({ ...estado, perfil: { carga: "denegado", datos: null } }, { textos, escaparHTML: esc });
  assert.match(denegado, /Su perfil no permite consultar este apartado/u); assert.match(denegado, /Llamamiento &lt;A&gt;/u);
});
