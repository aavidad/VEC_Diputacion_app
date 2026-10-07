import "./test-preparar-textos.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { ErrorClienteSolicitudesCronos } from "./cliente-solicitudes-http.js";
import { filtrarIncidenciasPropiasCronos, marcasPorDiaCronos, montarMovimientosPropiosCronos, renderizarMovimientosPropiosCronos } from "./vista-movimientos-propios.js";

function datos(disponible = true) {
  return {
    periodo: { tipo: "rango", desde: "2026-01-01", hasta: "2026-12-31" },
    calendario: { disponible, dias: disponible ? [{ fecha: "2026-01-06", tipo: "festivo", nombre: "Epifanía" }] : [] },
    marcajes_por_dia: [{ fecha: "2026-09-21", marcajes: 4 }],
    absentismos: [{ solicitud_ref: "permiso:cronos:solicitud:c-0000001", permiso_ref: "permiso:cronos:traslado", nombre: "Traslado de domicilio", desde: "2026-09-22", hasta: "2026-09-23", cantidad: 2, unidad: "dia", pendiente_justificar: true }],
    correcciones: [{ solicitud_ref: "correccion:cronos:corr-0000001", fecha_civil: "2026-09-24", hora_pretendida: "08:00", movimiento: "entrada", estado: "pendiente_responsable", version: 1, solicitada_en: "2026-09-25T07:00:00Z" }],
  };
}
function raizFalsa() {
  const nodo = { dataset: {}, innerHTML: "", eventos: {}, addEventListener(tipo, fn) { this.eventos[tipo] = fn; },
    removeEventListener(tipo) { delete this.eventos[tipo]; }, remove() { this.eliminado = true; }, querySelector() { return null; } };
  return { nodo, raiz: { ownerDocument: { createElement: () => nodo }, append() {} } };
}
function datosConsulta(consulta) {
  const d = datos();
  d.periodo.tipo = consulta.periodo;
  if (consulta.periodo === "rango") { d.periodo.desde = consulta.desde; d.periodo.hasta = consulta.hasta; }
  return d;
}
const esperar = async () => { for (let i = 0; i < 6; i++) await Promise.resolve(); };

test("marca cada día por tipo con los datos recibidos y dice en una línea si falta el calendario", () => {
  const marcas = marcasPorDiaCronos(datos());
  assert.deepEqual([...marcas.get("2026-09-22")], ["ausencia"]);
  assert.ok(marcas.get("2026-09-23").has("ausencia"));
  assert.ok(marcas.get("2026-09-21").has("marcaje"));
  assert.ok(marcas.get("2026-09-24").has("olvido"));
  assert.ok(marcas.get("2026-01-06").has("festivo"));
  const html = renderizarMovimientosPropiosCronos({ estado: "listo", anio: 2026, datos: datos(), hoy: "2026-09-25" });
  assert.match(html, /data-fecha="2026-01-06" data-tipos="festivo"/);
  assert.match(html, /Traslado de domicilio/);
  assert.match(html, /2 días/);
  assert.match(html, /Pendiente de justificar/);
  assert.match(html, /Pendiente de la jefatura/);
  assert.match(html, /data-accion="ayuda"/);
  assert.doesNotMatch(html, /no está publicado/);
  assert.doesNotMatch(html, /permiso:cronos|correccion:cronos|recibo:cronos|DEMO|sintétic/iu);
  const sin = renderizarMovimientosPropiosCronos({ estado: "listo", anio: 2026, datos: datos(false), hoy: "2026-09-25" });
  assert.equal((sin.match(/no está publicado/g) || []).length, 1);
  assert.doesNotMatch(sin, /data-fecha="2026-01-06" data-tipos/);
});

test("el olvido abre la solicitud, conserva la clave al reintentar y recarga el año tras registrarla", async () => {
  const { nodo, raiz } = raizFalsa();
  const envios = []; let fallar = true; let consultas = 0;
  const cliente = {
    consultarMovimientos: async (consulta) => { consultas++; return datosConsulta(consulta); },
    solicitarCorreccion: async (entrada) => {
      envios.push(entrada);
      if (fallar) { fallar = false; throw new ErrorClienteSolicitudesCronos("servicio_no_disponible", 503); }
      return { solicitud_ref: `correccion:cronos:${entrada.clave_operacion}`, actuacion_ref: "actuacion:cronos:1", recibo_ref: "recibo:cronos:1", estado: "pendiente_responsable", version: 1, instante_utc: "2026-09-25T07:00:00Z", replay: false };
    },
  };
  const vista = montarMovimientosPropiosCronos({ raiz, cliente, anio: 2026 });
  await esperar();
  assert.match(nodo.innerHTML, /Comunicar un olvido/);
  nodo.eventos.click({ target: { closest: (sel) => (sel === "[data-cronos-olvido]" ? { dataset: { cronosOlvido: "abrir" } } : null) } });
  assert.match(nodo.innerHTML, /name="hora_pretendida"/);
  const formulario = { matches: () => true, elements: { namedItem: (n) => ({ value: { fecha_civil: "2026-09-24", hora_pretendida: "08:00", movimiento: "entrada" }[n] }) } };
  await nodo.eventos.submit({ target: formulario, preventDefault() {} });
  assert.match(nodo.innerHTML, /Puede reintentarla sin duplicarla/);
  await nodo.eventos.submit({ target: formulario, preventDefault() {} });
  await esperar();
  assert.equal(envios.length, 2);
  assert.equal(envios[0].clave_operacion, envios[1].clave_operacion);
  assert.deepEqual(Object.keys(envios[1]).sort(), ["clave_operacion", "fecha_civil", "hora_pretendida", "movimiento"]);
  assert.match(nodo.innerHTML, /Queda pendiente de la jefatura/);
  assert.ok(consultas >= 2);
  vista.desmontar();
  assert.equal(nodo.eliminado, true);
});

test("sin empleado y denegación muestran su motivo sin hechos anteriores", async () => {
  for (const [codigo, texto] of [["sin_empleado", /relación de empleo vigente/], ["acceso_denegado", /No tiene permiso/], ["servicio_no_disponible", /No se pudieron consultar/]]) {
    const { nodo, raiz } = raizFalsa();
    const vista = montarMovimientosPropiosCronos({ raiz, anio: 2026, cliente: { consultarMovimientos: async () => { throw new ErrorClienteSolicitudesCronos(codigo, 403); }, solicitarCorreccion: async () => ({}) } });
    await esperar();
    assert.match(nodo.innerHTML, texto);
    assert.doesNotMatch(nodo.innerHTML, /Traslado/);
    vista.desmontar();
  }
});

test("la vista no guarda nada en el navegador ni edita marcajes", async () => {
  const fuente = await readFile(new URL("./vista-movimientos-propios.js", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /localStorage|sessionStorage|indexedDB|document\.cookie|geolocation|Math\.random|marcajes\/remoto/u);
});

test("404 de la API: «no disponible» neutro, sin alerta; incrustada sin sobrelínea", async () => {
  const { nodo, raiz } = raizFalsa();
  const vista = montarMovimientosPropiosCronos({ raiz, anio: 2026, incrustada: true, cliente: {
    consultarMovimientos: async () => { throw new ErrorClienteSolicitudesCronos("servicio_no_disponible", 404); }, solicitarCorreccion: async () => ({}) } });
  await esperar();
  assert.match(nodo.innerHTML, /<p class="cronos-vacio"[^>]* role="status">Esta consulta no está disponible\.<\/p>/u);
  assert.doesNotMatch(nodo.innerHTML, /role="alert"|No se pudieron consultar|sobrelinea|<h2/u);
  assert.match(nodo.innerHTML, /<h3 id="cronos-movpropios-titulo">/u);
  vista.desmontar();
});

function pulsar(nodo, selector, dataset = {}) {
  nodo.eventos.click({ target: { closest: (s) => s === selector ? { dataset } : null } });
}
function cambiar(nodo, nombre, value) {
  nodo.eventos.change({ type: "change", target: { dataset: { cronosFiltro: nombre }, value } });
}
function editar(nodo, name, value) {
  nodo.eventos.input({ type: "input", target: { name, value } });
}
function enviar(nodo) {
  return nodo.eventos.submit({ preventDefault() {}, target: { matches: () => true, elements: { namedItem: (n) => ({ value: { fecha_civil: "2026-09-24", hora_pretendida: "08:15", movimiento: "entrada" }[n] }) } } });
}
function diferido() { let resolver; const promesa = new Promise((si) => { resolver = si; }); return { promesa, resolver }; }
function clienteConsulta() { return { consultarMovimientos: async (c) => datosConsulta(c), solicitarCorreccion: async () => ({}) }; }

test("filtra estados y ausencias acreditadas por solapamiento sin inferir ausentismo", () => {
  const d = datos();
  d.correcciones.push({ ...d.correcciones[0], solicitud_ref: "correccion:cronos:corr-2", fecha_civil: "2026-09-23", estado: "aplicada" });
  d.absentismos.push({ ...d.absentismos[0], solicitud_ref: "permiso:cronos:solicitud:p-2", pendiente_justificar: false });
  const f = filtrarIncidenciasPropiasCronos(d, { desde: "2026-09-23", hasta: "2026-09-23", estado: "aplicada", justificante: "pendiente" });
  assert.equal(f.correcciones.length, 1);
  assert.equal(f.absentismos.length, 1);
  assert.equal(f.absentismos[0].pendiente_justificar, true);
  assert.deepEqual(filtrarIncidenciasPropiasCronos(d, { desde: "2026-09-23", hasta: "2026-09-22" }), { correcciones: [], absentismos: [] });
  d.absentismos = []; d.correcciones = [];
  assert.equal(marcasPorDiaCronos(d).has("2026-09-22"), false);
});

test("todos los estados tienen siguiente paso; escapa el contenido y rechaza DTO incompleto", () => {
  const d = datos();
  const estados = ["pendiente_responsable", "pendiente_rrhh", "denegada_responsable", "denegada_rrhh", "pendiente_aplicacion", "aplicada"];
  d.absentismos[0].nombre = '<img src=x onerror="alert(1)">';
  d.correcciones = estados.map((estado, i) => ({ ...d.correcciones[0], solicitud_ref: `correccion:cronos:c-${i}`, estado }));
  const html = renderizarMovimientosPropiosCronos({ estado: "listo", anio: 2026, datos: d });
  assert.match(html, /Siguiente paso/);
  assert.match(html, /La jefatura debe revisar/);
  assert.match(html, /RRHH debe revisar/);
  assert.match(html, /La corrección está aplicada/);
  assert.match(html, /&lt;img src=x onerror=&quot;alert\(1\)&quot;&gt;/);
  assert.doesNotMatch(html, /<img|undefined|data-cronos-decidir|data-cronos-aplicar|data-cronos-justificar/);
  assert.match(html, /La presentación de justificantes no está disponible/);
  delete d.correcciones[0].version;
  const incompleto = renderizarMovimientosPropiosCronos({ estado: "listo", anio: 2026, datos: d });
  assert.match(incompleto, /data-estado="error"/);
  assert.doesNotMatch(incompleto, /La jefatura debe revisar/);
});

test("paginación y recuentos filtran localmente y conservan el borrador al actualizar", async () => {
  const { nodo, raiz } = raizFalsa(); let consultas = 0;
  const vista = montarMovimientosPropiosCronos({ raiz, anio: 2026, tamanoPagina: 1, abrirOlvido: true, cliente: {
    ...clienteConsulta(), consultarMovimientos: async (consulta) => {
      consultas++; const d = datosConsulta(consulta);
      d.correcciones.push({ ...d.correcciones[0], solicitud_ref: "correccion:cronos:otra", hora_pretendida: "09:12", estado: "aplicada" });
      return d;
    },
  } });
  await esperar();
  editar(nodo, "fecha_civil", "2026-09-20"); editar(nodo, "hora_pretendida", "07:35"); editar(nodo, "movimiento", "salida");
  assert.match(nodo.innerHTML, /Mostrando 1 a 1 de 2/);
  assert.doesNotMatch(nodo.innerHTML, /<td>09:12<\/td>/);
  pulsar(nodo, "[data-cronos-pagina]", { cronosPagina: "correcciones", pagina: "2" });
  assert.match(nodo.innerHTML, /<td>09:12<\/td>/);
  cambiar(nodo, "estado", "pendiente_responsable");
  assert.match(nodo.innerHTML, /Pendiente de la jefatura \(1\)/);
  assert.match(nodo.innerHTML, /value="pendiente_responsable" selected/);
  assert.match(nodo.innerHTML, /name="hora_pretendida" required value="07:35"/);
  assert.equal(consultas, 1);
  await vista.actualizar();
  assert.equal(consultas, 2);
  assert.match(nodo.innerHTML, /value="pendiente_responsable" selected/);
  assert.match(nodo.innerHTML, /name="fecha_civil"[^>]*value="2026-09-20"/);
  assert.match(nodo.innerHTML, /name="hora_pretendida" required value="07:35"/);
  assert.match(nodo.innerHTML, /value="salida" selected/);
  pulsar(nodo, "[data-cronos-recuento]", { cronosRecuento: "pendiente" });
  assert.match(nodo.innerHTML, /value="pendiente" selected/);
  assert.equal(consultas, 2);
  vista.desmontar();
  assert.equal(nodo.eventos.input, undefined);
  await vista.actualizar();
  assert.equal(consultas, 2);
});

test("ignora la consulta antigua y no confirma lecturas o recibos incompletos", async () => {
  const { nodo, raiz } = raizFalsa(); const antigua = diferido(); let consultas = 0;
  const vista = montarMovimientosPropiosCronos({ raiz, anio: 2026, cliente: {
    ...clienteConsulta(), consultarMovimientos: async (c) => { consultas++; return consultas === 1 ? antigua.promesa : datosConsulta(c); },
  } });
  await vista.actualizar();
  antigua.resolver({}); await esperar();
  assert.match(nodo.innerHTML, /data-estado="listo"/);
  pulsar(nodo, "[data-cronos-olvido]", { cronosOlvido: "abrir" });
  await enviar(nodo);
  assert.match(nodo.innerHTML, /Puede reintentarla sin duplicarla/);
  assert.doesNotMatch(nodo.innerHTML, /Queda pendiente de la jefatura/);
  vista.desmontar();
  const b = raizFalsa();
  const mala = montarMovimientosPropiosCronos({ raiz: b.raiz, anio: 2026, cliente: { ...clienteConsulta(), consultarMovimientos: async () => ({}) } });
  await esperar(); assert.match(b.nodo.innerHTML, /data-estado="error"/); mala.desmontar();
});

test("el envío pendiente impide cerrar o iniciar otra escritura; desmontar anula respuestas tardías", async () => {
  const { nodo, raiz } = raizFalsa(); const pendiente = diferido(); let escrituras = 0; let senal;
  const avisos = [];
  const vista = montarMovimientosPropiosCronos({ raiz, anio: 2026, abrirOlvido: true, anunciar: (v) => avisos.push(v), cliente: {
    ...clienteConsulta(), solicitarCorreccion: async (_c, { signal }) => { escrituras++; senal = signal; return pendiente.promesa; },
  } });
  await esperar(); const primera = enviar(nodo); await esperar();
  pulsar(nodo, "[data-cronos-olvido]", { cronosOlvido: "cerrar" });
  pulsar(nodo, "[data-cronos-olvido]", { cronosOlvido: "abrir" });
  await enviar(nodo);
  assert.equal(escrituras, 1);
  assert.match(nodo.innerHTML, /data-cronos-olvido="cerrar" disabled/);
  const html = nodo.innerHTML; const n = avisos.length;
  vista.desmontar(); assert.equal(senal.aborted, true);
  pendiente.resolver({}); await primera;
  assert.equal(nodo.innerHTML, html); assert.equal(avisos.length, n);
});

test("el traductor real carga ambas lenguas y exige catálogo completo", async () => {
  const { cargarTextos } = await import("../../../comun/textos.js");
  const { crearTraductorIncidenciasCronos } = await import("./i18n-incidencias.js?v=20261001-cronos-grafo-bandeja-v5");
  for (const idioma of ["es", "en"]) {
    const base = (await cargarTextos("cronos", { idioma })).seccion("solicitudes");
    const extension = (await cargarTextos("cronos-incidencias", { idioma })).seccion("incidencias");
    const html = renderizarMovimientosPropiosCronos({ estado: "listo", anio: 2026, datos: datos(), mensajes: { ...base, ...extension }, locale: idioma });
    assert.match(html, idioma === "en" ? /Next step/ : /Siguiente paso/);
    assert.match(html, idioma === "en" ? /Calendar view/ : /Vista del calendario/);
    assert.match(html, idioma === "en" ? /Day details/ : /Detalle del día/);
    assert.doesNotMatch(html, /undefined/);
  }
  assert.throws(() => crearTraductorIncidenciasCronos({ siguiente_paso: "" }), /incompleto/);
  assert.throws(() => renderizarMovimientosPropiosCronos({ anio: 2026, tamanoPagina: 101 }), /tamaño de página/);
});


test("Actualizar restaura el foco del teclado después de la carga sin robar foco externo", async () => {
  const { nodo, raiz } = raizFalsa(); const carga = diferido(); let consultas = 0; let enfocado = 0;
  const body = {}; const externo = {};
  nodo.ownerDocument = { body, activeElement: body };
  const actualizar = { hasAttribute: (n) => n === "data-cronos-actualizar", focus() {
    if (!nodo.innerHTML.includes("data-cronos-actualizar disabled")) { nodo.ownerDocument.activeElement = this; enfocado++; }
  } };
  nodo.contains = (n) => n === actualizar;
  nodo.querySelector = (s) => s === ":focus" ? (nodo.ownerDocument.activeElement === actualizar ? actualizar : null)
    : s === "[data-cronos-actualizar]" ? actualizar : null;
  let html = "";
  Object.defineProperty(nodo, "innerHTML", { get: () => html, set: (v) => { html = v; if (nodo.ownerDocument.activeElement === actualizar) nodo.ownerDocument.activeElement = body; } });
  const vista = montarMovimientosPropiosCronos({ raiz, anio: 2026, cliente: {
    ...clienteConsulta(), consultarMovimientos: async (c) => { consultas++; return consultas === 1 ? datosConsulta(c) : carga.promesa; },
  } });
  await esperar(); actualizar.focus();
  const n = enfocado; const lectura = vista.actualizar(); await esperar();
  assert.equal(enfocado, n, "el botón disabled no puede recibir foco mientras carga");
  carga.resolver(datosConsulta({ periodo: "anio" })); await lectura;
  assert.equal(enfocado, n + 1, "el botón vuelve a recibir foco al terminar la lectura");
  const segunda = vista.actualizar(); nodo.ownerDocument.activeElement = externo; await segunda;
  assert.equal(enfocado, n + 1, "un control externo conserva el foco");
  assert.equal(nodo.ownerDocument.activeElement, externo);
  vista.desmontar();
});

function cambiarCalendario(nodo, atributo, value) {
  nodo.eventos.change({ type: "change", target: { value, hasAttribute: (nombre) => nombre === atributo } });
}
function datosAnio(consulta) {
  const anio = consulta.desde?.slice(0, 4) || String(new Date().getFullYear());
  const d = JSON.parse(JSON.stringify(datos()).replaceAll("2026", anio));
  d.periodo.tipo = consulta.periodo;
  return d;
}

test("año/mes muestran sólo el periodo elegido y el detalle contiene únicamente marcas recibidas", () => {
  const anual = renderizarMovimientosPropiosCronos({ estado: "listo", anio: 2026, datos: datos(), fechaSeleccionada: "2026-09-22" });
  const mensual = renderizarMovimientosPropiosCronos({ estado: "listo", anio: 2026, datos: datos(), vistaCalendario: "mes", fechaSeleccionada: "2026-09-22" });
  assert.equal((anual.match(/data-fecha=/g) || []).length, 365);
  assert.equal((mensual.match(/data-fecha=/g) || []).length, 30);
  assert.match(mensual, /data-calendario-vista="mes"/);
  assert.doesNotMatch(mensual, /data-fecha="2026-01-06"/);
  assert.match(mensual, /Detalle del día/);
  assert.match(mensual, /Ausencia: Traslado de domicilio/);
  assert.match(mensual, /data-calendario-estado="disponible"/);
  const vacio = renderizarMovimientosPropiosCronos({ estado: "listo", anio: 2026, datos: datos(false), vistaCalendario: "mes", fechaSeleccionada: "2026-09-26" });
  assert.match(vacio, /data-calendario-estado="no_configurado"/);
  assert.match(vacio, /No hay datos registrados para esta fecha/);
  assert.doesNotMatch(vacio, /data-fecha="2026-09-26" data-tipos/);
  const d = datos(); d.calendario.dias[0].nombre = '<img src=x onerror="1">';
  const seguro = renderizarMovimientosPropiosCronos({ estado: "listo", anio: 2026, datos: d, fechaSeleccionada: "2026-01-06" });
  assert.match(seguro, /Festivo: &lt;img/); assert.doesNotMatch(seguro, /<img/);
  for (const fechaSeleccionada of ["2026-02-29", "2025-12-31", "2101-01-01"]) {
    assert.throws(() => renderizarMovimientosPropiosCronos({ anio: 2026, fechaSeleccionada }), /vista de calendario/);
  }
  d.periodo.desde = "2026-01-02";
  assert.match(renderizarMovimientosPropiosCronos({ estado: "listo", anio: 2026, datos: d }), /data-estado="error"/);
});

test("navegar dentro del año reutiliza la consulta y conserva filtros y borrador", async () => {
  const { nodo, raiz } = raizFalsa(); let consultas = 0;
  const vista = montarMovimientosPropiosCronos({ raiz, anio: 2026, fechaSeleccionada: "2026-01-31", abrirOlvido: true, cliente: {
    ...clienteConsulta(), consultarMovimientos: async (c) => { consultas++; return datosConsulta(c); },
  } });
  await esperar();
  editar(nodo, "hora_pretendida", "07:35"); cambiar(nodo, "estado", "pendiente_responsable");
  cambiarCalendario(nodo, "data-cronos-cal-vista", "mes");
  pulsar(nodo, "[data-cronos-cal-mes]", { cronosCalMes: "1" });
  assert.equal((nodo.innerHTML.match(/data-fecha=/g) || []).length, 28);
  assert.match(nodo.innerHTML, /data-cronos-cal-fecha[^>]*value="2026-02-28"/);
  cambiarCalendario(nodo, "data-cronos-cal-mes-elegido", "9");
  cambiarCalendario(nodo, "data-cronos-cal-fecha", "2026-09-21");
  assert.match(nodo.innerHTML, /Marcajes: 4/);
  assert.match(nodo.innerHTML, /value="pendiente_responsable" selected/);
  assert.match(nodo.innerHTML, /name="hora_pretendida" required value="07:35"/);
  assert.equal(consultas, 1);
  await vista.actualizar();
  assert.match(nodo.innerHTML, /data-calendario-vista="mes"/);
  assert.match(nodo.innerHTML, /data-cronos-cal-fecha[^>]*value="2026-09-21"/);
  assert.equal(consultas, 2);
  cambiarCalendario(nodo, "data-cronos-cal-fecha", "2026-02-30");
  cambiarCalendario(nodo, "data-cronos-cal-mes-elegido", "13");
  assert.equal(consultas, 2);
  assert.match(nodo.innerHTML, /data-cronos-cal-fecha[^>]*value="2026-09-21"/);
  vista.desmontar();
});

test("salto mensual de año cancela la lectura anterior y nunca pinta otro año", async () => {
  const { nodo, raiz } = raizFalsa(); const vieja = diferido(); const lecturas = []; const avisos = [];
  const vista = montarMovimientosPropiosCronos({ raiz, anio: 2025, fechaSeleccionada: "2025-12-31", vistaCalendario: "mes", anunciar: (v) => avisos.push(v), cliente: {
    ...clienteConsulta(), consultarMovimientos: async (consulta, { signal }) => {
      lecturas.push({ consulta, signal });
      return lecturas.length === 1 ? vieja.promesa : datosAnio(consulta);
    },
  } });
  // Durante carga el salto anual sigue disponible; la antigua queda cancelada.
  pulsar(nodo, "[data-cronos-anio]", { cronosAnio: "1" });
  await esperar();
  assert.equal(lecturas[0].signal.aborted, true);
  vieja.resolver(datosAnio(lecturas[0].consulta)); await esperar();
  assert.match(nodo.innerHTML, /data-estado="listo"/);
  assert.match(nodo.innerHTML, /data-cronos-cal-fecha[^>]*value="2026-12-31"/);
  pulsar(nodo, "[data-cronos-cal-mes]", { cronosCalMes: "1" }); await esperar();
  assert.deepEqual(lecturas[2].consulta, { periodo: "rango", desde: "2027-01-01", hasta: "2027-12-31" });
  assert.match(nodo.innerHTML, /data-cronos-cal-fecha[^>]*value="2027-01-31"/);
  assert.doesNotMatch(nodo.innerHTML, /data-fecha="2026-/);
  const n = avisos.length;
  vista.desmontar(); assert.equal(lecturas[2].signal.aborted, true);
  await vista.actualizar(); assert.equal(lecturas.length, 3); assert.equal(avisos.length, n);
});

test("límites de calendario no generan lecturas ni fechas fuera de contrato", async () => {
  for (const [anio, fecha, paso] of [[2000, "2000-01-01", "-1"], [2100, "2100-12-31", "1"]]) {
    const { nodo, raiz } = raizFalsa(); let consultas = 0;
    const vista = montarMovimientosPropiosCronos({ raiz, anio, fechaSeleccionada: fecha, vistaCalendario: "mes", cliente: {
      ...clienteConsulta(), consultarMovimientos: async (c) => { consultas++; return datosAnio(c); },
    } });
    await esperar();
    assert.match(nodo.innerHTML, new RegExp(`data-cronos-cal-mes="${paso}" disabled`));
    pulsar(nodo, "[data-cronos-cal-mes]", { cronosCalMes: paso });
    assert.equal(consultas, 1);
    assert.match(nodo.innerHTML, new RegExp(`value="${fecha}"`));
    vista.desmontar();
  }
});
