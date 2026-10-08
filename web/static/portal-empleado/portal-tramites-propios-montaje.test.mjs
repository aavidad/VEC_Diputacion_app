import test from "node:test";
import assert from "node:assert/strict";
import { crearCoordinadorModulosPortal, moduloDeVistaPortal, rutaDeVistaPortal } from "./portal-modulos-coordinador.js?v=20261008-alta-rpt-circular-v6";
import { crearFuenteTramitesPropios } from "./modulos/solicitudes/fuente-tramites-propios.js";

const permiso = { permiso_ref: "permiso:cronos:formacion", version_ref: "catalogo:cronos:formacion:v1", nombre: "Formación", unidad: "dia", computo: "laborables", circuito: "J-A", minimo: 1, maximo_solicitud: null, maximo_mensual: null, maximo_anual: null, justificante_exigido: false, solicitable: true, sintetico: true, solicitado: 1, concedido: 0, pendiente_justificar: 0, resta: null };
const cronos = (anio) => ({ anio, permisos: [permiso], solicitudes: [{ solicitud_ref: "permiso:cronos:solicitud:formacion0001", catalogo_version_ref: permiso.version_ref, permiso_ref: permiso.permiso_ref, desde: `${anio}-10-05`, hasta: `${anio}-10-05`, cantidad: 1, unidad: "dia", estado: "solicitado", version: 1, pendiente_justificar: false, solicitada_en: `${anio}-10-01T10:00:00Z` }] });
const recibo = { referencia: `rcd_${"c".repeat(22)}`, version: 3, registrado_en: "2026-10-01T10:00:00.123456Z", repeticion: false };
const dietas = { items: [{ comision: { referencia: `dco_${"a".repeat(22)}`, version: 3, numero_documento: "VEC-D-2026-000001", estado: "fiscalizada", fecha_inicio: "2026-10-05", fecha_fin: "2026-10-06", motivo: "Visita de coordinación", relacion_ref: `rel_${"b".repeat(22)}` }, recibo }] };
const diferida = () => { let resolver; const promesa = new Promise((r) => resolver = r); return { resolver, promesa }; };
const raiz = () => ({ innerHTML: "", replaceChildren() { this.innerHTML = ""; } });
const tick = () => new Promise((r) => setImmediate(r));

function entorno({ catalogo = ["cronos", "dietas", "personal"], pendienteCronos, pendienteHoja } = {}) {
  const lecturas = []; const cargas = []; const montajes = []; let fuente; let falloDietas; let falloCronos; let desmontajes = 0;
  const montar = () => ({ desmontar() {} });
  const recursosCronos = {
    saldo: { montarVistaSaldoCronos: montar }, remoto: { montarVistaRemotoCronos: montar }, movimientos: { montarVistaMovimientosCronos: montar }, movimientosPropios: { montarMovimientosPropiosCronos: montar }, permisosPropios: { montarPermisosPropiosCronos: montar },
    clienteSaldo: { crearClienteSaldoCronosHTTP: () => ({}) }, clienteRemoto: { crearClienteRemotoCronosHTTP: () => ({}) }, i18n: { crearTraductorCronos: () => (clave) => clave },
    clienteSolicitudes: { crearClienteSolicitudesCronosHTTP: () => ({ consultarPermisos: async (consulta, opciones) => { lecturas.push(["cronos", consulta, opciones]); if (falloCronos) throw { codigo: falloCronos }; return cronos(consulta.anio); } }) },
  };
  const recursosDietas = { contrato: {}, clienteBorradores: { crearClienteBorradoresDietasHTTP: () => ({ listar: async (consulta, opciones) => { lecturas.push(["dietas", consulta, opciones]); if (falloDietas) throw { codigo: falloDietas, message: "detalle reservado" }; return dietas; } }) }, clienteAsignacion: { crearClienteAsignacionDietasHTTP: () => ({}) }, calculador: { crearCalculadorRutasDietasHTTP: () => ({}) }, mapa: { crearVisorRutaDietas: () => ({}) }, recorridos: { montarVistaRecorridosDietas: montar } };
  const recursosHoja = { fuente: { crearFuenteTramitesPropios }, vista: { montarVistaTramitesPropios: (opciones) => { fuente = opciones.fuente; montajes.push(opciones); return { desmontar: () => desmontajes++ }; } } };
  const coordinador = crearCoordinadorModulosPortal({ escaparHTML: String,
    entorno: { fetch: () => assert.fail("no debe sondear otras APIs") },
    cargarCatalogoInterno: async () => catalogo.map((clave) => ({ clave })),
    cargarTramitesPropios: async () => { cargas.push("hoja"); return pendienteHoja ? pendienteHoja.promesa : recursosHoja; },
    cargadoresInternos: { contratacion_temporal: () => assert.fail("sin CT"), personal: () => assert.fail("Personal sigue diferido"), cronos: async () => { cargas.push("cronos"); return pendienteCronos ? pendienteCronos.promesa : recursosCronos; }, dietas: async () => { cargas.push("dietas"); return recursosDietas; } },
  });
  return { coordinador, cargas, lecturas, montajes, recursosCronos, recursosHoja, get fuente() { return fuente; }, get desmontajes() { return desmontajes; }, fallar(cronos, dietas) { falloCronos = cronos; falloDietas = dietas; } };
}

test("Mis trámites es una vista del portal y carga sólo código registrado al abrir, sin lecturas", async () => {
  const e = entorno(); const c = e.coordinador; await c.cargarInterno();
  assert.equal(moduloDeVistaPortal("mis-tramites"), "portal"); assert.equal(rutaDeVistaPortal("mis-tramites"), "#mis-tramites");
  assert.equal(c.obtenerAccesosEmpleado()["mis-tramites"].estado, "diferido"); assert.deepEqual(e.cargas, []); assert.deepEqual(e.lecturas, []);
  await c.prepararVista("mis-tramites"); assert.deepEqual(e.cargas.sort(), ["cronos", "dietas", "hoja"]); assert.deepEqual(e.lecturas, []);
  assert.equal(c.vistaDisponible("mis-tramites"), true); assert.equal(await c.montarVista("mis-tramites", raiz()), true);
  const signal = new AbortController().signal;
  assert.equal((await e.fuente.consultarCronos({ anio: 2026, signal })).solicitudes[0].nombre, "Formación");
  const pagina = await e.fuente.consultarDietas({ cursor: "pagina:2", signal });
  assert.deepEqual(pagina.items[0].recibo, recibo); assert.equal(pagina.items[0].comision.motivo, undefined);
  assert.deepEqual(e.lecturas, [["cronos", { anio: 2026 }, { signal }], ["dietas", { limit: 20, cursor: "pagina:2" }, { signal }]]);
  e.fallar("acceso_denegado", undefined); await assert.rejects(e.fuente.consultarCronos({ anio: 2026 }), { codigo: "acceso_denegado" });
  assert.deepEqual((await e.fuente.consultarDietas()).items[0].recibo, recibo);
  e.fallar(undefined, "relacion_ambigua"); const antes = e.lecturas.length;
  await assert.rejects(e.fuente.consultarDietas(), (error) => error.codigo === "relacion_ambigua" && error.message === "");
  assert.equal(e.lecturas.length, antes + 1); c.retirarVistaMontada(); assert.equal(e.desmontajes, 1);
});

test("la vista espera una carga Cronos ya iniciada y no monta antes de resolverla", async () => {
  const pendiente = diferida(); const e = entorno({ pendienteCronos: pendiente }); await e.coordinador.cargarInterno();
  const cargaCronos = e.coordinador.prepararVista("cronos"); const cargaTramites = e.coordinador.prepararVista("mis-tramites"); await tick();
  assert.equal(e.coordinador.vistaDisponible("mis-tramites"), false); assert.equal(e.coordinador.vistaPendiente("mis-tramites"), true);
  pendiente.resolver(e.recursosCronos); await Promise.all([cargaCronos, cargaTramites]);
  assert.equal(e.cargas.filter((clave) => clave === "cronos").length, 1); assert.equal(e.coordinador.vistaDisponible("mis-tramites"), true); assert.deepEqual(e.lecturas, []);
});

test("una relación no registrada no se carga y una hoja tardía no sobrevive a otro catálogo", async () => {
  const e = entorno({ catalogo: ["cronos"] }); await e.coordinador.cargarInterno(); await e.coordinador.prepararVista("mis-tramites");
  assert.deepEqual(e.cargas.sort(), ["cronos", "hoja"]); await e.coordinador.montarVista("mis-tramites", raiz()); assert.deepEqual(e.fuente.disponibles, { cronos: true, dietas: false });
  const pendiente = diferida(); const tarde = entorno({ pendienteHoja: pendiente }); await tarde.coordinador.cargarInterno();
  const carga = tarde.coordinador.prepararVista("mis-tramites"); await tick();
  await tarde.coordinador.cargarInterno(); pendiente.resolver(tarde.recursosHoja); await carga;
  assert.equal(tarde.coordinador.vistaDisponible("mis-tramites"), false); assert.equal(tarde.coordinador.obtenerAccesosEmpleado()["mis-tramites"].estado, "diferido"); assert.deepEqual(tarde.lecturas, []);
});
