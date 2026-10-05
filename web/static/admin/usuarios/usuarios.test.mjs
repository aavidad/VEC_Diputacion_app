import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { cargarTextos } from "../../comun/textos.js";
import { IDIOMAS_DISPONIBLES } from "../../comun/idioma.js";
import { validarCapacidades, validarRoles, validarUnidades, validarPersonas, validarFicha, seleccionarActos, prepararDecision, puedeConfirmar, validarResultado } from "./contratos.js";
import { crearClienteUsuarios } from "./cliente.js";
import { crearClienteLecturasUsuarios } from "./lecturas-http.js";
import { montarUsuarios } from "./vista.js";
const original = JSON.parse(await readFile(new URL("./fixture.test.json", import.meta.url), "utf8"));
const fuente = () => structuredClone(original);
const cripto = { getRandomValues(bytes) { bytes.fill(7); return bytes; } };
const textos = await cargarTextos("admin-usuarios");
function decision(indices = [0], f = fuente()) {
  return prepararDecision(f.ficha, validarRoles(f.catalogo), f.capacidades.acciones, "otorgar", indices, Object.fromEntries(indices.map((i) => [i, 0])), cripto);
}
function recibo(d) {
  const s = d.seleccion[0].acto;
  return { recibo: { operacion_ref: d.cuerpo.operacion_ref, recibo_ref: `recibo_admin:${"b".repeat(32)}`,
    objetivo_persona_ref: s.objetivo.persona_ref, perfil_ref: s.objetivo.perfil_ref, rol_version_ref:s.rol_version_ref, vinculo_ref:s.objetivo.vinculo_ref, motivo:d.cuerpo.motivo, estado_posterior: s.operacion==="revocar"?"revocado":"activo", version_posterior: s.operacion==="revocar"?s.objetivo.vinculo_version+1:1,
    unidad_ref:s.objetivo.unidad_ref,centro_ref:s.objetivo.centro_ref,vigente_desde:s.objetivo.vigente_desde,vigente_hasta:s.objetivo.vigente_hasta,
    confirmado_en: s.objetivo.vigente_desde, auditoria_ref: "auditoria:sintetica:01", huella_antes_sha256: "a".repeat(64), huella_despues_sha256: "b".repeat(64) } };
}
class Nodo {
  constructor(root) { this.root = root; this.attrs = {}; this.value = ""; this.hidden = false; this.disabled = false; this.listeners = {}; }
  set innerHTML(html) { this.html = html; for (const [, id] of html.matchAll(/id="([^"]+)"/gu)) this.root.nodos.set(id, new Nodo(this.root)); }
  get innerHTML() { return this.html || ""; }
  getAttribute(c) { return this.attrs[c] ?? null; }
  setAttribute(c, v) { this.attrs[c] = v; }
  removeAttribute(c) { delete this.attrs[c]; }
  querySelector(s) { return this.root.nodos.get(s.slice(1)); }
  querySelectorAll() { return []; }
  addEventListener(c, f) { (this.listeners[c] ??= []).push(f); }
  removeEventListener(c, f) { this.listeners[c] = (this.listeners[c] || []).filter((fn) => fn !== f); }
  focus() { this.enfocado = true; }
  replaceChildren() { this.innerHTML = ""; }
}
class Raiz extends Nodo {
  constructor() { super(null); this.root = this; this.nodos = new Map(); }
  campo(c) { return [...this.nodos.entries()].find(([id]) => id.endsWith(`-${c}`))?.[1]; }
}
const diferida = () => { let resolver; const promesa = new Promise((r) => { resolver = r; }); return { promesa, resolver }; };
const cliente = (f = fuente()) => ({ capacidades: async () => f.capacidades, roles: async () => f.catalogo,
  buscar: async () => f.pagina, persona: async () => f.ficha });
const esperarVista = () => new Promise((resolve) => setImmediate(resolve));
function pulsar(root, accion, ref) {
  root.contains = () => true;
  root.listeners.click[0]({ target: { closest: () => ({ dataset: { accion, ref }, disabled: false }) } });
}
function buscar(root) {
  root.listeners.submit[0]({ preventDefault() {}, target: root.campo("buscar") });
}

test("lecturas conservan referencia, ámbito e historia de la fuente; datos malformados se rechazan", () => {
  const f = fuente();
  assert.deepEqual(validarFicha(f.ficha, f.ficha.persona_ref), f.ficha);
  assert.equal(validarRoles(f.catalogo).length, 4);
  assert.equal(validarPersonas(f.pagina).personas.length, 1);
  assert.throws(() => validarUnidades({ unidades: [{ unidad_ref: "", nombre: "dato" }] }));
  assert.equal(validarCapacidades(f.capacidades).acciones.length, 3);
  assert.throws(() => validarCapacidades({...f.capacidades, version:"v1"}));
  assert.throws(() => validarCapacidades({...f.capacidades, version:1}));
  assert.throws(() => validarCapacidades({ ...f.capacidades, acciones: ["superusuario"] }));
  assert.throws(() => validarRoles({ roles: [{ ...f.catalogo.roles[0], fijo: undefined }] }));
  assert.throws(() => validarPersonas({ personas: [f.pagina.personas[0], f.pagina.personas[0]] }));
  assert.throws(() => validarFicha(f.ficha, "per_otra_persona"));
  f.ficha.actos_disponibles[0].objetivo.persona_ref = "per_otra_persona";
  assert.throws(() => validarFicha(f.ficha, f.ficha.persona_ref));
});

test("selección exige acto autorizado exacto, motivo, fecha inicial y asignación disponible", () => {
  const f = fuente(), roles = validarRoles(f.catalogo), acciones = f.capacidades.acciones;
  assert.throws(() => seleccionarActos(f.ficha, roles, [], "otorgar", [0]));
  assert.equal(seleccionarActos(f.ficha, roles, acciones, "otorgar", [3]).length, 1);
  assert.throws(() => seleccionarActos(f.ficha, roles, acciones, "revocar", [0]));
  assert.throws(() => prepararDecision(f.ficha, roles, acciones, "otorgar", [0], {}, cripto));
  assert.throws(() => decision([0, 0]));
  delete f.ficha.actos_disponibles[0].objetivo.vigente_desde;
  assert.throws(() => decision([0], f));
  assert.equal(seleccionarActos(f.ficha, roles, acciones, "revocar", [4]).length, 1);
  f.ficha.perfiles[0].estado = "revocado";
  assert.throws(() => seleccionarActos(f.ficha, roles, acciones, "revocar", [4]));
});

test("selección múltiple nunca usa el puerto singular ni divide las escrituras", () => {
  const d = decision([0, 1]);
  assert.equal(d.cuerpo.cambios.length, 2);
  assert.equal(puedeConfirmar(d, original.capacidades.acciones, { aplicar() {} }), null);
  assert.equal(puedeConfirmar(d, [...original.capacidades.acciones, "aplicar_lote_ordinario"], { aplicarLote() {} }), "aplicarLote");
  assert.equal(puedeConfirmar(decision([2]), original.capacidades.acciones, { proponer() {} }), "proponer");
  assert.equal(puedeConfirmar(decision([0, 2]), [...original.capacidades.acciones, "aplicar_lote_ordinario"], { aplicarLote() {} }), null);
  const f = fuente(), preimagen = structuredClone(f.ficha); decision([0, 1], f); assert.deepEqual(f.ficha, preimagen);
  f.ficha.actos_disponibles[1].motivos[0].entrada_clave="motivo:diferente";
  assert.equal(puedeConfirmar(decision([0,1],f),[...original.capacidades.acciones,"aplicar_lote_ordinario"],{aplicarLote(){}}),null);
});

test("solo confirma recibo ligado a la operación, persona, perfiles, estado y huellas", () => {
  const d = decision(); assert.equal(validarResultado(recibo(d), d).tipo, "recibo");
  for (const cambio of [{ operacion_ref: "acto_admin:ajeno" }, { objetivo_persona_ref: "per_otra_persona" },
    { perfil_ref: "perfil:otro" }, { estado_posterior: "revocado" }, { huella_despues_sha256: "" }, { version_posterior: 0 }]) {
    assert.throws(() => validarResultado({ recibo: { ...recibo(d).recibo, ...cambio } }, d));
  }
  const lote = decision([0, 1]); assert.throws(() => validarResultado(recibo(lote), lote));
  const r = {recibo:{operacion_ref:lote.cuerpo.operacion_ref,acto_ref:`acto_admin:${"b".repeat(32)}`,recibo_ref:`recibo_admin:${"c".repeat(32)}`,auditoria_ref:"auditoria:000001",huella_solicitud_sha256:"c".repeat(64),confirmado_en:lote.seleccion[0].acto.objetivo.vigente_desde,
    cambios:lote.seleccion.map(s=>({...recibo({cuerpo:lote.cuerpo,seleccion:[s]}).recibo,acto_ref:`acto_admin:${"b".repeat(32)}`,recibo_ref:`recibo_admin:${"c".repeat(32)}`,auditoria_ref:"auditoria:000001",correlacion_ref:"correlacion:000001"}))}};
  assert.equal(validarResultado(r, lote).tipo, "recibo");
  r.recibo.cambios[1].perfil_ref = r.recibo.cambios[0].perfil_ref; assert.throws(() => validarResultado(r, lote));
});

test("propuesta sensible informa pendiente de aprobación y no acepta un recibo ordinario", () => {
  const d = decision([2]);
  assert.throws(() => validarResultado(recibo(d), d));
  assert.equal(validarResultado({ propuesta: { propuesta_ref: d.cuerpo.operacion_ref, huella_sha256: "c".repeat(64), caduca_en: "2026-10-10T08:00:00Z" } }, d).tipo, "propuesta");
});

test("transporte filtra en servidor, no sigue redirecciones, acota bytes y nunca escribe", async () => {
  const llamadas = [];
  const c = crearClienteLecturasUsuarios({ origen: "https://admin.invalid", fetchImpl: async (url, opciones) => {
    llamadas.push({ url, opciones }); return new Response(JSON.stringify(original.pagina), { headers: { "Content-Type": "application/json" } });
  } });
  await c.buscar({ busqueda: "Carmen", perfil_ref: "rol:rrhh:v1", unidad_ref: "unidad:rrhh", estado: "vigente", cursor: "cursor:01" });
  const l = llamadas[0]; assert.equal(new URL(l.url).searchParams.get("estado"), "vigente");
  assert.equal(l.opciones.credentials, "same-origin"); assert.equal(l.opciones.redirect, "error");
  assert.equal(l.opciones.cache, "no-store"); assert.equal(l.opciones.referrerPolicy, "no-referrer");
  assert.equal(c.aplicar, undefined); assert.equal(c.proponer, undefined);
  assert.throws(() => c.buscar({ estado: "activo" })); assert.throws(() => c.buscar({ busqueda: "x" }));
  assert.throws(() => c.persona("../../secreto"));
  for (const respuesta of [new Response("{}", { status: 403, headers: { "Content-Type": "application/json" } }),
    new Response("{}", { headers: { "Content-Type": "text/html" } }),
    new Response("x".repeat(262145), { headers: { "Content-Type": "application/json" } })]) {
    const malo = crearClienteLecturasUsuarios({ origen: "https://admin.invalid", fetchImpl: async () => respuesta });
    await assert.rejects(malo.capacidades());
  }
});

test("adaptador E permanece cerrado para filtros no soportados y conserva los puertos existentes", async () => {
  assert.deepEqual(crearClienteUsuarios(), {});
  const llamadas = [], a = { buscar(q, s) { llamadas.push([q, s]); return q; }, aplicar(c) { return c; } };
  const c = crearClienteUsuarios({ administracion: a });
  assert.equal(c.buscar({ busqueda: "persona" }), "persona"); assert.throws(() => c.buscar({ estado: "vigente" }));
  assert.equal(c.aplicar("dato"), "dato"); assert.equal(llamadas.length, 1);
});

test("nombres de la fuente se escapan y no introducen HTML activo", async () => {
  const f = fuente(); f.pagina.personas[0].nombre = '<img src=x onerror="alert(1)">';
  const root = new Raiz(), v = montarUsuarios(root, { textos, cliente: cliente(f) }); await v.listo;
  assert.doesNotMatch(root.campo("resultados").innerHTML, /<img/u);
  assert.match(root.campo("resultados").innerHTML, /&lt;img/u); v.desmontar();
});

test("vista sin puerto explica la dependencia y no carga ni habilita cambios", async () => {
  const root = new Raiz(), vista = montarUsuarios(root, { textos }); await vista.listo;
  assert.equal(root.campo("estado").textContent, textos.traducir("errores.sin_conexion"));
  assert.equal(root.campo("buscar-boton").disabled, true); vista.desmontar(); assert.equal(root.innerHTML, "");
});

test("vista elimina información al denegar e ignora respuestas tardías tras desmontar", async () => {
  const root = new Raiz(), c = cliente(); c.buscar = async () => { throw { estado: 403 }; };
  const v = montarUsuarios(root, { textos, cliente: c }); await v.listo;
  assert.equal(root.campo("estado").textContent, textos.traducir("errores.denegado")); assert.equal(root.campo("panel-perfiles").innerHTML, "");
  const otra = new Raiz(), d = diferida(); let señal;
  const tarde = montarUsuarios(otra, { textos, cliente: { ...cliente(), capacidades: (s) => { señal = s; return d.promesa; } } });
  tarde.desmontar(); d.resolver(original.capacidades); await tarde.listo;
  assert.equal(señal.aborted, true); assert.equal(otra.innerHTML, "");
});

test("una lectura fresca fallida retira datos y filtros; solo una recarga validada recupera la consulta", async () => {
  for (const fallo of [{ estado: 503 }, { estado: 403 }, { codigo: "respuesta_incompatible" }]) {
    const root = new Raiz(), c = cliente(), v = montarUsuarios(root, { textos, cliente: c }); await v.listo;
    assert.match(root.campo("panel-perfiles").innerHTML, /<table/u);
    root.campo("consulta").value = original.ficha.nombre;
    c.buscar = async () => { throw fallo; }; buscar(root); await esperarVista();
    for (const parte of ["resultados", "detalle", "revision", "panel-perfiles", "panel-propuestas", "filtros-activos"]) assert.equal(root.campo(parte).innerHTML, "");
    assert.equal(root.campo("consulta").value, "");
    assert.equal(root.campo("buscar-boton").disabled, true);
    assert.equal(root.campo("tab-perfiles").disabled, true);
    assert.equal(root.campo("recargar").disabled, false);
    assert.equal(root.campo("recargar").enfocado, true);
    const aviso = textos.traducir(fallo.estado === 403 ? "errores.denegado" : "errores.lectura_retirada");
    pulsar(root, "perfiles"); assert.equal(root.campo("estado").textContent, aviso);
    c.buscar = cliente().buscar; await v.cargar();
    assert.match(root.campo("resultados").innerHTML, /Carmen Molina/u);
    assert.equal(root.campo("buscar-boton").disabled, false); v.desmontar();
  }
});

test("fallo de ficha retira el listado anterior y el catálogo", async () => {
  const root = new Raiz(), c = cliente(), v = montarUsuarios(root, { textos, cliente: c }); await v.listo;
  c.persona = async () => { throw { estado: 503 }; };
  pulsar(root, "persona", original.ficha.persona_ref); await esperarVista();
  assert.equal(root.campo("resultados").innerHTML, ""); assert.equal(root.campo("panel-perfiles").innerHTML, "");
  assert.equal(root.campo("estado").textContent, textos.traducir("errores.lectura_retirada")); v.desmontar();
});

test("una nueva búsqueda cancela la ficha pendiente y descarta su respuesta tardía", async () => {
  const root = new Raiz(), pendiente = diferida(); let señal;
  const c = { ...cliente(), persona: (_ref, signal) => { señal = signal; return pendiente.promesa; } };
  const v = montarUsuarios(root, { textos, cliente: c }); await v.listo;
  pulsar(root, "persona", original.ficha.persona_ref);
  buscar(root); await esperarVista(); assert.equal(señal.aborted, true);
  pendiente.resolver(original.ficha); await esperarVista();
  assert.equal(root.campo("detalle").innerHTML, ""); assert.equal(root.campo("detalle").hidden, true);
  assert.equal(root.campo("listado").hidden, false); v.desmontar();
});

test("catálogos resuelven ES/EN sin faltantes y el grafo interno usa una URL por módulo", async () => {
  const claves = (o, p = "") => Object.entries(o).flatMap(([k, v]) => typeof v === "string" ? [p + k] : claves(v, `${p}${k}.`));
  for (const idioma of IDIOMAS_DISPONIBLES) {
    const c = await cargarTextos("admin-usuarios", { idioma: idioma.codigo }); assert.deepEqual(c.faltantes, []);
    for (const clave of claves(textos.mensajes)) assert.ok(c.traducir(clave));
  }
  for (const archivo of ["entry.js", "vista.js", "propuestas.js", "propuestas-contratos.js"]) {
    const s = await readFile(new URL(archivo, import.meta.url), "utf8");
    for (const [, modulo] of s.matchAll(/from "(\.\/[^"]+)"/gu)) assert.equal(new URL(modulo, import.meta.url).search, "?v=20261004-admin-usuarios-metadata-v1");
  }
});

function metadataPersona() {
  return { persona_ref: "per_aaaaaaaaaaaaaaaaaaaaaaaa", unidad_ref: "unidad:ensayo", denominacion_version: null,
    nombre_estado: "no_registrado", perfiles: [{ perfil_ref: "prf_aaaaaaaaaaaaaaaaaaaaaaaa", rol_version_ref: "rol:administracion_perfiles:v5",
      version: 1, estado: "vigente", vigente_desde: "2026-10-01T00:00:00Z", vigente_hasta: "2026-10-10T00:00:00Z" }] };
}
test("metadatos separan ausencia de nombre y consulta pendiente sin inventar historial", () => {
  const p = metadataPersona();
  const ficha = { ...p, proyeccion: "metadatos_v1", historia_estado: "no_consultada", actos_estado: "no_consultados" };
  assert.equal(validarFicha(ficha, p.persona_ref).nombre_estado, "no_registrado");
  assert.throws(() => validarFicha({ ...ficha, historia: [] }, p.persona_ref));
  assert.throws(() => validarFicha({ ...ficha, actos_disponibles: [] }, p.persona_ref));
  assert.throws(() => validarFicha({ ...ficha, nombre: "Elena Marquez" }, p.persona_ref));
  assert.throws(() => validarFicha({ ...ficha, denominacion_version: 1 }, p.persona_ref));
  assert.equal(validarFicha({ ...ficha, nombre_estado: "no_consultado", denominacion_version: 1 }, p.persona_ref).nombre_estado, "no_consultado");
});
test("metadatos mantienen 51 perfiles de una persona y limitan únicamente las personas de la página", () => {
  const p = metadataPersona();
  p.perfiles = Array.from({ length: 51 }, (_, i) => ({ ...p.perfiles[0], perfil_ref: `prf_${String(i).padStart(24, "0")}` }));
  assert.equal(validarPersonas({ proyeccion: "metadatos_v1", personas: [p] }).personas[0].perfiles.length, 51);
  assert.throws(() => validarPersonas({ proyeccion: "metadatos_v1", personas: Array(51).fill(p) }));
});
test("vista de metadatos consulta la lista directamente y mantiene cerrados cambios y consultas no realizadas", async () => {
  const root = new Raiz(), p = metadataPersona();
  let auxiliares = 0;
  const c = { proyeccion: "metadatos_v1", capacidades: async () => { auxiliares++; throw new Error(); }, roles: async () => { auxiliares++; throw new Error(); },
    buscar: async () => ({ proyeccion: "metadatos_v1", personas: [p] }),
    persona: async () => ({ ...p, proyeccion: "metadatos_v1", historia_estado: "no_consultada", actos_estado: "no_consultados" }) };
  const vista = montarUsuarios(root, { textos, cliente: c }); await vista.listo;
  assert.equal(auxiliares, 0);
  assert.equal(root.campo("consulta").disabled, true);
  assert.equal(root.campo("tab-propuestas").disabled, true);
  assert.match(root.campo("resultados").innerHTML, /Nombre sin registrar/u);
  pulsar(root, "persona", p.persona_ref); await esperarVista();
  assert.match(root.campo("detalle").innerHTML, /El historial no se ha consultado/u);
  assert.doesNotMatch(root.campo("detalle").innerHTML, /No hay cambios|data-seleccion/u);
  vista.desmontar();
});
