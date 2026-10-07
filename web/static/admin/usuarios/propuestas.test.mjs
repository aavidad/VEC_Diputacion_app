import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { validarFicha } from "./contratos.js";
import { validarPropuestas, puedeCerrarPropuesta, prepararCierre, validarCierre } from "./propuestas-contratos.js";
import { crearClienteLecturasUsuarios, crearClienteActosUsuarios } from "./lecturas-http.js";
const f = JSON.parse(await readFile(new URL("./fixture.test.json", import.meta.url), "utf8"));
const p = { propuesta_ref: `propuesta_admin:${"c".repeat(32)}`, proponente_persona_ref: "per_primera_persona_00000001",
  proponente_nombre: "Elena Martín", proponente_perfil_nombre:"Administrador de perfiles", objetivo_persona_ref: f.ficha.persona_ref, objetivo_nombre: f.ficha.nombre,
  rol_version_ref: f.catalogo.roles[2].version_ref, operacion: "otorgar", huella_sha256: "c".repeat(64),
  caduca_en: "2026-10-10T08:00:00Z", puede_cerrar: true, ambitos: [{dimension:"unidad",referencia:"unidad:rrhh",nombre:"Recursos Humanos"}], vigente_desde:"2026-10-01T08:00:00Z", vigente_hasta:"2027-01-01T08:00:00Z", motivo:f.ficha.actos_disponibles[2].motivos[0], motivos_cierre: f.ficha.actos_disponibles[2].motivos };
const actor = "per_segunda_persona_00000001", cap = ["cerrar_propuesta"], cliente = { cerrarPropuesta() {} };
const cripto = { getRandomValues(v) { v.fill(9); return v; } };
const preparar = (decision = "rechazar") => prepararCierre(p, actor, cap, cliente, decision, 0, cripto);
function respuesta(d) {
  const c = { operacion_ref: d.cuerpo.operacion_ref, propuesta_ref: p.propuesta_ref, propuesta_huella_sha256: p.huella_sha256,
    decision: d.cuerpo.decision, huella_cierre_sha256: "d".repeat(64), confirmado_en: "2026-10-03T08:00:00Z" };
  if (c.decision === "aprobada") c.recibo = { ...c, rol_version_ref: p.rol_version_ref, actor_persona_ref: actor,
    objetivo_persona_ref: p.objetivo_persona_ref, recibo_ref: `recibo_admin:${"d".repeat(32)}`,
    perfil_ref: "prf_perfil_sintetico_000002", auditoria_ref: "auditoria:000001", version_posterior: 1,
    estado_posterior: "activo", huella_antes_sha256: "a".repeat(64), huella_despues_sha256: "b".repeat(64), motivo: d.cuerpo.motivo };
  return { cierre: c };
}
test("ficha respeta versiones cero del alta y fechas cero de revocación en el contrato HTTP", () => {
  assert.equal(validarFicha(f.ficha, f.ficha.persona_ref).actos_disponibles[0].objetivo.perfil_version, 0);
  for (const mutar of [x => { x.actos_disponibles[0].objetivo.perfil_version = 1; },
    x => { x.actos_disponibles[0].objetivo.vinculo_version = 1; },
    x => { x.actos_disponibles[4].objetivo.vigente_hasta = "2027-01-01T00:00:00Z"; },
    x => { x.actos_disponibles[4].objetivo.perfil_version = 0; }]) {
    const x = structuredClone(f.ficha); mutar(x); assert.throws(() => validarFicha(x, x.persona_ref));
  }
});
test("segunda persona exige capacidad, motivo y servidor disponible; dos perfiles de la misma persona no bastan", () => {
  assert.deepEqual(validarPropuestas({ propuestas: [p] }), [p]);
  assert.equal(puedeCerrarPropuesta(p, actor, cap, cliente), true);
  for (const sujeto of [p.proponente_persona_ref, p.objetivo_persona_ref]) assert.equal(puedeCerrarPropuesta(p, sujeto, cap, cliente), false);
  assert.equal(puedeCerrarPropuesta(p, actor, [], cliente), false);
  assert.equal(puedeCerrarPropuesta({...p, ambitos:undefined}, actor, cap, cliente), false);
  assert.equal(puedeCerrarPropuesta({...p, proponente_perfil_nombre:undefined}, actor, cap, cliente), false);
  assert.equal(puedeCerrarPropuesta({...p, motivo:undefined}, actor, cap, cliente), false);
  assert.equal(puedeCerrarPropuesta(p, actor, cap, {}), false);
  assert.throws(() => prepararCierre(p, actor, cap, cliente, "aprobar", -1, cripto));
  assert.throws(() => validarPropuestas({ propuestas: [p, p] }));
  assert.throws(() => validarPropuestas({ propuestas: [{ ...p, puede_cerrar: undefined }] }));
  const d = preparar(); assert.match(d.cuerpo.operacion_ref, /^cierre_admin:[a-f0-9]{32}$/u);
  assert.equal(d.cuerpo.actor, undefined); assert.equal(d.cuerpo.proponente_persona_ref, undefined);
});
test("rechazo no se presenta como acceso cambiado y aprobación exige recibo ligado a persona y aprobador", () => {
  for (const decision of ["aprobar", "rechazar"]) {
    const d = preparar(decision), r = respuesta(d), protocolaria = decision === "aprobar" ? "aprobada" : "rechazada";
    assert.equal(d.cuerpo.decision, protocolaria); assert.equal(d.decision, decision);
    assert.equal(validarCierre(r, d).decision, protocolaria);
    assert.throws(() => validarCierre({ cierre: { ...r.cierre, decision } }, d));
    for (const cambio of [{ operacion_ref: "cierre_admin:ajeno" }, { propuesta_ref: "propuesta_admin:ajena" },
      { propuesta_huella_sha256: "a".repeat(64) }, { decision: "otra" }, { huella_cierre_sha256: "" }]) {
      assert.throws(() => validarCierre({ cierre: { ...r.cierre, ...cambio } }, d));
    }
    if (decision === "rechazar") assert.throws(() => validarCierre({ cierre: { ...r.cierre, recibo: {} } }, d));
    else for (const cambio of [{ actor_persona_ref: p.proponente_persona_ref }, { objetivo_persona_ref: actor },
      { rol_version_ref: "rol:otro:v1" }, { estado_posterior: "revocado" }, { motivo: {} }]) {
      assert.throws(() => validarCierre({ cierre: { ...r.cierre, recibo: { ...r.cierre.recibo, ...cambio } } }, d));
    }
  }
});
test("transporte inyectable conserva una escritura singular sin identidad HTTP ni bucles de lote", async () => {
  const llamadas = [], opciones = { origen: "https://admin.invalid", fetchImpl: async (url, o) => {
    llamadas.push({ url, o }); return new Response("{}", { headers: { "Content-Type": "application/json" } });
  } };
  const lecturas = crearClienteLecturasUsuarios(opciones), actos = crearClienteActosUsuarios(opciones), d = preparar();
  await lecturas.propuestas(); assert.equal(new URL(llamadas[0].url).search, "?estado=pendiente");
  assert.equal(lecturas.aplicar, undefined); assert.equal(lecturas.cerrarPropuesta, undefined);
  await actos.cerrarPropuesta(p.propuesta_ref, d.cuerpo);
  const l = llamadas[1]; assert.equal(l.o.method, "POST"); assert.equal(l.o.credentials, "same-origin");
  assert.equal(l.o.redirect, "error"); assert.equal(l.o.cache, "no-store"); assert.equal(l.o.referrerPolicy, "no-referrer");
  assert.deepEqual(JSON.parse(l.o.body), d.cuerpo); assert.equal(l.o.headers.Authorization, undefined);
  assert.throws(() => actos.cerrarPropuesta(p.propuesta_ref, { ...d.cuerpo, actor: "suplantacion" }));
  assert.throws(() => actos.aplicar({ solicitudes: [{}, {}] })); assert.equal(typeof actos.aplicarLote, "function");
  assert.equal(llamadas.length, 2);
});

test("lote ordinario usa una petición con motivo común y rechaza cuerpos divididos o suplantados", async () => {
  const llamadas = [], c = crearClienteActosUsuarios({ origen:"https://admin.invalid", fetchImpl:async(url, o)=>{
    llamadas.push({url,o}); return new Response("{}",{headers:{"Content-Type":"application/json"}});
  } });
  const cuerpo={operacion_ref:`acto_admin:${"a".repeat(32)}`,cambios:f.ficha.actos_disponibles.slice(0,2).map(({operacion,rol_version_ref,objetivo})=>({operacion,rol_version_ref,objetivo})),motivo:preparar().cuerpo.motivo};
  await c.aplicarLote(cuerpo); assert.equal(llamadas.length,1);
  assert.equal(new URL(llamadas[0].url).pathname,"/api/admin/perfiles/v1/lotes-ordinarios");
  assert.deepEqual(JSON.parse(llamadas[0].o.body),cuerpo);
  assert.throws(()=>c.aplicarLote({...cuerpo,actor:{persona_ref:actor}}));
  assert.throws(()=>c.aplicarLote({...cuerpo,cambios:[]}));
  const grande={...cuerpo,cambios:Array.from({length:32},()=>structuredClone(cuerpo.cambios[0]))};
  assert.ok(new TextEncoder().encode(JSON.stringify(grande)).byteLength>16384);
  await c.aplicarLote(grande); assert.equal(llamadas.length,2);
  await assert.rejects(c.aplicarLote({...grande,referencia_acto:"x".repeat(65536)}));
  assert.equal(llamadas.length,2);
});
