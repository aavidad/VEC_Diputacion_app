import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { ErrorClienteSolicitudesCronos } from "./cliente-solicitudes-http.js";
import { estadoSolicitudPermisoCronos, montarPermisosPropiosCronos, renderizarPermisosPropiosCronos } from "./vista-permisos-propios.js";

function permiso(ref, nombre, extra = {}) {
  return { permiso_ref: `permiso:cronos:${ref}`, version_ref: `catalogo:cronos:${ref}:v1`, nombre, unidad: "dia", computo: "laborables", circuito: "A",
    minimo: 1, maximo_solicitud: null, maximo_mensual: null, maximo_anual: 6, justificante_exigido: false, solicitable: true, sintetico: true,
    solicitado: 0, concedido: 0, pendiente_justificar: 0, resta: 6, ...extra };
}
function datos() {
  return {
    anio: 2026,
    permisos: [
      permiso("asuntos-propios", "Asuntos propios", { solicitado: 1, concedido: 2, resta: 3 }),
      permiso("horas-medico", "Horas de médico", { unidad: "hora", circuito: "J-A", minimo: 15, maximo_anual: null, maximo_solicitud: 180, resta: null }),
      permiso("horas-sindicales", "Horas sindicales", { unidad: "hora", minimo: 15, maximo_anual: null, maximo_mensual: 3600, resta: null }),
      permiso("nacimiento", "Nacimiento", { computo: "naturales", solicitable: false, maximo_anual: 7, resta: 7 }),
    ],
    solicitudes: [
      { solicitud_ref: "permiso:cronos:solicitud:a-1", catalogo_version_ref: "catalogo:cronos:asuntos-propios:v1", permiso_ref: "permiso:cronos:asuntos-propios", desde: "2026-03-02", hasta: "2026-03-03", cantidad: 2, unidad: "dia", estado: "concedido", version: 2, pendiente_justificar: true, solicitada_en: "2026-02-20T08:00:00Z" },
      { solicitud_ref: "permiso:cronos:solicitud:a-2", catalogo_version_ref: "catalogo:cronos:asuntos-propios:v1", permiso_ref: "permiso:cronos:asuntos-propios", desde: "2026-10-05", hasta: "2026-10-05", cantidad: 1, unidad: "dia", estado: "solicitado", version: 1, pendiente_justificar: false, solicitada_en: "2026-09-25T08:00:00Z", circuito: "J-A" },
      { solicitud_ref: "permiso:cronos:solicitud:m-1", catalogo_version_ref: "catalogo:cronos:horas-medico:v1", permiso_ref: "permiso:cronos:horas-medico", desde: "2026-10-07", hasta: "2026-10-07", hora_inicio: "09:00", hora_fin: "11:30", cantidad: 150, unidad: "hora", estado: "solicitado", version: 1, pendiente_justificar: false, solicitada_en: "2026-09-25T08:00:00Z", circuito: "A" },
    ],
  };
}
function raizFalsa() {
  const nodo = { dataset: {}, innerHTML: "", eventos: {}, addEventListener(tipo, fn) { this.eventos[tipo] = fn; },
    removeEventListener(tipo) { delete this.eventos[tipo]; }, remove() { this.eliminado = true; }, querySelector() { return null; } };
  return { nodo, raiz: { ownerDocument: { createElement: () => nodo }, append() {} } };
}
const esperar = async () => { for (let i = 0; i < 6; i++) await Promise.resolve(); };

test("listado anual con máximo, mínimo, solicitado, concedido y resta en días u horas y minutos", () => {
  const html = renderizarPermisosPropiosCronos({ estado: "listo", anio: 2026, datos: datos() });
  assert.match(html, /Asuntos propios/);
  assert.match(html, /6 días/);
  assert.match(html, /3 días/);
  assert.match(html, /3 h por solicitud/);
  assert.match(html, /60 h al mes/);
  assert.match(html, /15 min/);
  assert.match(html, /2 h 30 min/);
  // El circuito del catálogo ya no decide: no se muestra ni en la tabla ni
  // en el formulario; cada solicitud dice en qué punto está.
  assert.doesNotMatch(html, /Jefatura y administración|>Administración<|>Concede</u);
  assert.match(html, /Cuantías pendientes de confirmar por RRHH/);
  assert.equal((html.match(/data-cronos-solicitar=/g) || []).length, 3, "el permiso no solicitable no ofrece Solicitar");
  assert.match(html, /Pendiente de jefatura/);
  assert.match(html, /Pendiente de RRHH/);
  assert.doesNotMatch(html, /Pendiente de administración|Pendiente de la jefatura/);
  assert.match(html, /Pendientes de justificar/);
  assert.doesNotMatch(html, /catalogo:cronos|solicitud:a-|Conceder|Denegar|DEMO/u);
  const reales = datos(); reales.permisos.forEach((p) => { p.sintetico = false; });
  assert.doesNotMatch(renderizarPermisosPropiosCronos({ estado: "listo", anio: 2026, datos: reales }), /pendientes de confirmar/);
});

test("con solo cupo mensual la resta es lo que queda del mes en curso; una fila sin conciliar no tumba el listado", () => {
  const d = datos();
  d.solicitudes.push(
    { solicitud_ref: "permiso:cronos:solicitud:s-1", catalogo_version_ref: "catalogo:cronos:horas-sindicales:v1", permiso_ref: "permiso:cronos:horas-sindicales", desde: "2026-10-02", hasta: "2026-10-02", hora_inicio: "09:00", hora_fin: "10:00", cantidad: 60, unidad: "hora", estado: "concedido", version: 2, pendiente_justificar: false, solicitada_en: "2026-09-25T08:00:00Z" },
    { solicitud_ref: "permiso:cronos:solicitud:s-2", catalogo_version_ref: "catalogo:cronos:horas-sindicales:v1", permiso_ref: "permiso:cronos:horas-sindicales", desde: "2026-10-05", hasta: "2026-10-05", hora_inicio: "09:00", hora_fin: "09:30", cantidad: 30, unidad: "hora", estado: "denegado", version: 2, pendiente_justificar: false, solicitada_en: "2026-09-25T08:00:00Z" },
    { solicitud_ref: "permiso:cronos:solicitud:s-3", catalogo_version_ref: "catalogo:cronos:horas-sindicales:v1", permiso_ref: "permiso:cronos:horas-sindicales", desde: "2026-09-10", hasta: "2026-09-10", hora_inicio: "09:00", hora_fin: "11:00", cantidad: 120, unidad: "hora", estado: "concedido", version: 2, pendiente_justificar: false, solicitada_en: "2026-09-01T08:00:00Z" },
  );
  d.permisos[3].sin_conciliar = true; d.permisos[3].resta = null;
  const html = renderizarPermisosPropiosCronos({ estado: "listo", anio: 2026, datos: d, hoy: "2026-10-15" });
  assert.match(html, /59 h este mes/);
  assert.match(html, /A revisar por RRHH/);
  assert.match(html, /Asuntos propios/);
  const anterior = { ...d, anio: 2025, solicitudes: d.solicitudes.map((s) => ({ ...s, desde: s.desde.replace("2026", "2025"), hasta: s.hasta.replace("2026", "2025") })) };
  assert.doesNotMatch(renderizarPermisosPropiosCronos({ estado: "listo", anio: 2025, datos: anterior, hoy: "2026-10-15" }), /este mes/);
});

test("solicitar un permiso por horas envía un solo día con su tramo y muestra el rechazo nominal", async () => {
  const { nodo, raiz } = raizFalsa(); const envios = []; let respuesta = new ErrorClienteSolicitudesCronos("fuera_de_limites", 422);
  const cliente = {
    consultarPermisos: async () => datos(),
    solicitarPermiso: async (e) => { envios.push(e); if (respuesta instanceof Error) throw respuesta; return respuesta; },
  };
  const vista = montarPermisosPropiosCronos({ raiz, cliente, anio: 2026 });
  await esperar();
  nodo.eventos.click({ target: { closest: (sel) => (sel === "[data-cronos-solicitar]" ? { dataset: { cronosSolicitar: "permiso:cronos:horas-medico" } } : null) } });
  assert.match(nodo.innerHTML, /name="hora_inicio"/);
  assert.doesNotMatch(nodo.innerHTML, /name="hasta"/);
  const formulario = { matches: () => true, elements: { namedItem: (n) => ({ value: { desde: "2026-10-08", hora_inicio: "09:00", hora_fin: "10:00" }[n] }) } };
  await nodo.eventos.submit({ target: formulario, preventDefault() {} });
  assert.match(nodo.innerHTML, /supera el máximo/);
  assert.deepEqual(envios[0], { clave_operacion: envios[0].clave_operacion, permiso_ref: "permiso:cronos:horas-medico", desde: "2026-10-08", hasta: "2026-10-08", hora_inicio: "09:00", hora_fin: "10:00" });
  respuesta = { solicitud_ref: `permiso:cronos:solicitud:${envios[0].clave_operacion}`, catalogo_version_ref: "catalogo:cronos:horas-medico:v1", recibo_ref: "recibo:cronos:1", estado: "solicitado", version: 1, cantidad: 60, unidad: "hora", instante_utc: "2026-09-25T08:00:00Z", replay: false };
  await nodo.eventos.submit({ target: formulario, preventDefault() {} });
  await esperar();
  assert.equal(envios[1].clave_operacion, envios[0].clave_operacion, "un reintento conserva la clave");
  assert.match(nodo.innerHTML, /Solicitud registrada: 1 h\. Queda pendiente de resolver\./);
  assert.doesNotMatch(nodo.innerHTML, /Jefatura y administración|cronos-circuito/);
  vista.desmontar();
  assert.equal(nodo.eliminado, true);
});

test("sin empleado o sin servicio no muestra cifras", async () => {
  for (const [codigo, texto] of [["sin_empleado", /relación de empleo vigente/], ["servicio_no_disponible", /No se pudieron consultar/]]) {
    const { nodo, raiz } = raizFalsa();
    const vista = montarPermisosPropiosCronos({ raiz, anio: 2026, cliente: { consultarPermisos: async () => { throw new ErrorClienteSolicitudesCronos(codigo, 503); }, solicitarPermiso: async () => ({}) } });
    await esperar();
    assert.match(nodo.innerHTML, texto);
    assert.doesNotMatch(nodo.innerHTML, /días|tarjeta-kpi/);
    vista.desmontar();
  }
});

test("la vista no guarda nada en el navegador ni concede permisos", async () => {
  const fuente = await readFile(new URL("./vista-permisos-propios.js", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /localStorage|sessionStorage|indexedDB|document\.cookie|Math\.random|decidir|conceder\(/u);
});

test("el estado de cada solicitud sale del circuito aplicado por el servidor, no del catálogo", () => {
  const casos = [
    [{ estado: "solicitado", circuito: "J-A" }, "estado_permiso_pendiente_jefatura"],
    [{ estado: "solicitado", circuito: "A" }, "estado_permiso_pendiente_rrhh"],
    [{ estado: "solicitado", circuito: "J-A", pendiente_asignacion: true }, "estado_permiso_pendiente_asignacion"],
    [{ estado: "solicitado" }, "estado_permiso_pendiente"],
    [{ estado: "pendiente_administracion", circuito: "J-A" }, "estado_permiso_pendiente_rrhh"],
    [{ estado: "pendiente_administracion" }, "estado_permiso_pendiente_rrhh"],
    [{ estado: "concedido", circuito: "A" }, "estado_concedido"],
    [{ estado: "denegado" }, "estado_permiso_denegado"],
    [{ estado: "cancelado" }, "estado_permiso_cancelado"],
  ];
  for (const [s, clave] of casos) assert.equal(estadoSolicitudPermisoCronos(s), clave, JSON.stringify(s));
  const d = datos();
  d.solicitudes[1] = { ...d.solicitudes[1], pendiente_asignacion: true };
  d.solicitudes[2] = { ...d.solicitudes[2] }; delete d.solicitudes[2].circuito;
  const html = renderizarPermisosPropiosCronos({ estado: "listo", anio: 2026, datos: d });
  assert.match(html, /Pendiente de asignar jefatura/);
  assert.match(html, /Pendiente de resolver/);
  assert.doesNotMatch(html, /Pendiente de RRHH/, "sin circuito del servidor no se deduce del catálogo (A)");
});

function datosHistorial(cantidad = 27) {
  const d = datos();
  d.solicitudes = Array.from({ length: cantidad }, (_, i) => ({ ...d.solicitudes[1],
    solicitud_ref: `permiso:cronos:solicitud:historial-${i}`, estado: ["solicitado", "concedido", "denegado", "cancelado"][i % 4] }));
  return d;
}
function pulsar(nodo, selector, dataset) {
  return nodo.eventos.click({ target: { closest: (s) => s === selector ? { dataset } : null } });
}
function filtrar(nodo, valor) {
  nodo.eventos.change({ target: { matches: (s) => s === "[data-cronos-historial-filtro]", value: valor } });
}
function editar(nodo, nombre, valor) {
  nodo.eventos.input({ target: { closest: () => ({}), name: nombre, value: valor } });
}
function enviar(nodo, campos = { desde: "2026-10-08", hasta: "2026-10-09" }) {
  return nodo.eventos.submit({ target: { matches: () => true, elements: { namedItem: (n) => ({ value: campos[n] }) } }, preventDefault() {} });
}
function reciboDe(entrada) {
  return { solicitud_ref: `permiso:cronos:solicitud:${entrada.clave_operacion}`, catalogo_version_ref: "catalogo:cronos:asuntos-propios:v1",
    recibo_ref: "recibo:cronos:conservado", estado: "solicitado", version: 1, cantidad: 2, unidad: "dia", instante_utc: "2026-10-01T08:15:00Z", replay: false };
}
function diferido() { let resolver; const promesa = new Promise((si) => { resolver = si; }); return { promesa, resolver }; }

test("el historial hace visibles denegadas y canceladas sin inventar motivos ni recibos", () => {
  const d = datosHistorial();
  const html = renderizarPermisosPropiosCronos({ estado: "listo", anio: 2026, datos: d });
  assert.match(html, /Denegado/); assert.match(html, /Cancelado/);
  assert.match(html, /Todas \(27\)/); assert.match(html, /Denegadas \(7\)/); assert.match(html, /Canceladas \(6\)/);
  assert.match(html, /Solicitudes 1–20 de 27. Página 1 de 2/);
  assert.doesNotMatch(html, /recibo:|Motivo|Fecha de resolución/);
  assert.equal((html.match(/data-estado="denegado"/gu) || []).length, 5);
  const filtrado = renderizarPermisosPropiosCronos({ estado: "listo", anio: 2026, datos: d, filtro: "cancelados", tamanoPagina: 2, pagina: 2 });
  assert.match(filtrado, /Solicitudes 3–4 de 6. Página 2 de 3/);
  assert.doesNotMatch(filtrado, /data-estado="denegado"/);
  assert.equal((filtrado.match(/data-estado="cancelado"/gu) || []).length, 2);
  const hostil = datosHistorial(4); hostil.permisos[0].nombre = '<img src=x onerror="x()">';
  assert.match(renderizarPermisosPropiosCronos({ estado: "listo", anio: 2026, datos: hostil }), /&lt;img/);
});

test("filtrar y paginar son locales y conservan el borrador al dibujar o recargar", async () => {
  const { nodo, raiz } = raizFalsa(); let consultas = 0;
  const vista = montarPermisosPropiosCronos({ raiz, anio: 2026, tamanoPagina: 2,
    cliente: { consultarPermisos: async () => { consultas++; return datosHistorial(); }, solicitarPermiso: async () => ({}) } });
  await esperar();
  pulsar(nodo, "[data-cronos-solicitar]", { cronosSolicitar: "permiso:cronos:asuntos-propios" });
  editar(nodo, "desde", "2026-10-20"); editar(nodo, "hasta", "2026-10-21");
  filtrar(nodo, "denegados");
  assert.match(nodo.innerHTML, /value="2026-10-20"/); assert.match(nodo.innerHTML, /value="2026-10-21"/);
  pulsar(nodo, "[data-cronos-historial-pagina]", { cronosHistorialPagina: "siguiente" });
  assert.match(nodo.innerHTML, /Solicitudes 3–4 de 7. Página 2 de 4/);
  assert.equal(consultas, 1);
  await vista.recargar();
  assert.match(nodo.innerHTML, /value="denegados" selected/);
  assert.match(nodo.innerHTML, /Página 2 de 4/);
  assert.match(nodo.innerHTML, /value="2026-10-20"/);
  filtrar(nodo, "cancelados"); assert.match(nodo.innerHTML, /Página 1 de 3/);
  vista.desmontar();
});

test("conserva el recibo original tras recargas y caída del listado, sin inventar otro", async () => {
  const { nodo, raiz } = raizFalsa(); let caer = false; let envios = 0;
  const vista = montarPermisosPropiosCronos({ raiz, anio: 2026,
    cliente: { consultarPermisos: async () => { if (caer) throw new ErrorClienteSolicitudesCronos("servicio_no_disponible", 503); return datos(); },
      solicitarPermiso: async (entrada) => { envios++; return reciboDe(entrada); } } });
  await esperar(); pulsar(nodo, "[data-cronos-solicitar]", { cronosSolicitar: "permiso:cronos:asuntos-propios" });
  await enviar(nodo); assert.match(nodo.innerHTML, /recibo:cronos:conservado/);
  assert.match(nodo.innerHTML, /datetime="2026-10-01T08:15:00Z"/);
  await vista.recargar(); assert.match(nodo.innerHTML, /recibo:cronos:conservado/);
  caer = true; await vista.recargar();
  assert.match(nodo.innerHTML, /No se pudieron consultar/);
  assert.match(nodo.innerHTML, /recibo:cronos:conservado/);
  assert.match(nodo.innerHTML, /datetime="2026-10-01T08:15:00Z"/);
  assert.equal(envios, 1);
  vista.desmontar();
});

test("una promesa sin recibo válido no confirma la solicitud ni una lista incompatible produce cifras", async () => {
  for (const mutar of [(r) => ({ ...r, solicitud_ref: "permiso:cronos:solicitud:ajena" }), (r) => ({ ...r, extra: "desconocido" }), (r) => ({ ...r, cantidad: 0 }), (r) => ({ ...r, unidad: "hora" }), () => ({})]) {
    const { nodo, raiz } = raizFalsa();
    const vista = montarPermisosPropiosCronos({ raiz, anio: 2026,
      cliente: { consultarPermisos: async () => datos(), solicitarPermiso: async (e) => mutar(reciboDe(e)) } });
    await esperar(); pulsar(nodo, "[data-cronos-solicitar]", { cronosSolicitar: "permiso:cronos:asuntos-propios" });
    await enviar(nodo);
    assert.doesNotMatch(nodo.innerHTML, /Última solicitud registrada|Solicitud registrada:/);
    vista.desmontar();
  }
  const { nodo, raiz } = raizFalsa();
  const vista = montarPermisosPropiosCronos({ raiz, anio: 2026,
    cliente: { consultarPermisos: async () => ({ ...datos(), extra: "desconocido" }), solicitarPermiso: async () => ({}) } });
  await esperar(); assert.match(nodo.innerHTML, /data-estado="error"/); assert.doesNotMatch(nodo.innerHTML, /tarjeta-kpi/);
  vista.desmontar();
});

test("el recibo tardío de un formulario cerrado no modifica otra solicitud", async () => {
  const { nodo, raiz } = raizFalsa(); const pendiente = diferido(); let entrada;
  const vista = montarPermisosPropiosCronos({ raiz, anio: 2026,
    cliente: { consultarPermisos: async () => datos(), solicitarPermiso: async (e) => { entrada = e; return pendiente.promesa; } } });
  await esperar(); pulsar(nodo, "[data-cronos-solicitar]", { cronosSolicitar: "permiso:cronos:asuntos-propios" });
  const envio = enviar(nodo);
  pulsar(nodo, "[data-cronos-permiso-cerrar]", {});
  pulsar(nodo, "[data-cronos-solicitar]", { cronosSolicitar: "permiso:cronos:horas-medico" });
  pendiente.resolver(reciboDe(entrada)); await envio;
  assert.match(nodo.innerHTML, /name="hora_inicio"/);
  assert.doesNotMatch(nodo.innerHTML, /Última solicitud registrada|Solicitud registrada:/);
  vista.desmontar();
});

test("los catálogos del historial contienen los mismos mensajes y variables y la página está acotada", async () => {
  const { crearTraductorHistorialCronos } = await import("./i18n-historial.js");
  const { cargarTextos } = await import("../../../comun/textos.js");
  const raiz = new URL("../../../textos/", import.meta.url);
  const indice = JSON.parse(await readFile(new URL("idiomas.json", raiz), "utf8")); let claves;
  for (const { codigo } of indice.idiomas) {
    const texto = await cargarTextos("cronos-historial", { idioma: codigo, avisar: (aviso) => assert.fail(aviso) });
    const mensajes = texto.seccion("historial"); claves ??= Object.keys(mensajes); assert.deepEqual(Object.keys(mensajes), claves);
    const t = crearTraductorHistorialCronos(mensajes);
    assert.match(t("recuento", { desde: 1, hasta: 20, total: 27, pagina: 1, paginas: 2 }), /27/);
    assert.throws(() => t("inexistente"), TypeError);
  }
  for (const tamanoPagina of [0, -1, 1.5, 5001]) assert.throws(() => renderizarPermisosPropiosCronos({ anio: 2026, tamanoPagina }), RangeError);
  assert.throws(() => renderizarPermisosPropiosCronos({ anio: 2026, filtro: "desconocido" }), RangeError);
  const vacio = renderizarPermisosPropiosCronos({ estado: "listo", anio: 2026, datos: datos(), filtro: "denegados" });
  assert.match(vacio, /No hay solicitudes con este estado/); assert.match(vacio, /Solicitudes 0–0 de 0/);
});

test("el filtro conserva foco al cambiar y una respuesta tardía no muestra otro año", async () => {
  const { nodo, raiz } = raizFalsa(); const pendiente = diferido(); let llamadas = 0;
  const controlFiltro = { getAttribute: (a) => a === "data-cronos-historial-filtro" ? "" : null,
    focus() { raiz.ownerDocument.activeElement = this; } };
  nodo.contains = (elemento) => elemento === controlFiltro;
  nodo.querySelector = (selector) => selector === '[data-cronos-historial-filtro=""]' ? controlFiltro : null;
  const vista = montarPermisosPropiosCronos({ raiz, anio: 2026,
    cliente: { consultarPermisos: async ({ anio }) => { llamadas++; return llamadas === 1 ? pendiente.promesa : { ...datos(), anio, solicitudes: [] }; }, solicitarPermiso: async () => ({}) } });
  pulsar(nodo, "[data-cronos-anio]", { cronosAnio: "-1" }); await esperar();
  pendiente.resolver(datos()); await esperar();
  assert.match(nodo.innerHTML, /Permisos de 2025/);
  assert.doesNotMatch(nodo.innerHTML, /Solicitudes 1–3/);
  controlFiltro.focus(); filtrar(nodo, "cancelados");
  assert.equal(raiz.ownerDocument.activeElement, controlFiltro);
  assert.match(nodo.innerHTML, /value="cancelados" selected/);
  vista.desmontar();
});

test("la justificación muestra solo la marca del servidor sobre permisos concedidos", () => {
  const d = datos();
  d.permisos[0].justificante_exigido = false;
  d.solicitudes.push({ ...d.solicitudes[0], solicitud_ref: "permiso:cronos:solicitud:sin-pendiente", pendiente_justificar: false });
  d.solicitudes[1].pendiente_justificar = true;
  const html = renderizarPermisosPropiosCronos({ estado: "listo", anio: 2026, datos: d });
  const panel = html.slice(html.indexOf('id="cronos-permisos-just-panel"'));
  assert.equal((panel.match(/<tbody><tr>/gu) || []).length, 1);
  assert.equal((panel.match(/data-estado="concedido"/gu) || []).length, 1);
  assert.doesNotMatch(panel, /data-estado="solicitado"/u);
  assert.match(panel, /Pendiente de justificar/u);
  assert.match(panel, /El registro de justificantes todavía no está disponible/u);
  assert.doesNotMatch(panel, /type="file"|<form|Registrar justificante|data-cronos-solicitar/u);
  assert.match(html, /data-cronos-ver-justificacion aria-controls="cronos-permisos-just-panel"/u);
  assert.match(panel, /id="cronos-justificacion-ayuda" hidden/u);
  d.solicitudes.forEach((s) => { s.pendiente_justificar = false; });
  const vacio = renderizarPermisosPropiosCronos({ estado: "listo", anio: 2026, datos: d });
  assert.match(vacio, /No tienes permisos pendientes de justificar en este año/u);
  assert.doesNotMatch(vacio, /El registro de justificantes todavía no está disponible/u);
});

test("ir a justificar enfoca el panel sin peticiones y la ayuda conserva borrador y filtro", async () => {
  const { nodo, raiz } = raizFalsa(); let consultas = 0; let escrituras = 0; let desplazamientos = 0;
  const titulo = { focus() { raiz.ownerDocument.activeElement = this; }, scrollIntoView(opciones) { assert.deepEqual(opciones, { block: "nearest" }); desplazamientos++; } };
  const botonAyuda = { focus() { raiz.ownerDocument.activeElement = this; } };
  nodo.querySelector = (selector) => selector === "#cronos-permisos-just" ? titulo
    : selector === "[data-cronos-justificacion-ayuda]" ? botonAyuda : null;
  const vista = montarPermisosPropiosCronos({ raiz, anio: 2026,
    cliente: { consultarPermisos: async () => { consultas++; return datos(); }, solicitarPermiso: async () => { escrituras++; } } });
  await esperar();
  pulsar(nodo, "[data-cronos-solicitar]", { cronosSolicitar: "permiso:cronos:asuntos-propios" });
  editar(nodo, "desde", "2026-10-20"); filtrar(nodo, "denegados");
  pulsar(nodo, "[data-cronos-ver-justificacion]", {});
  assert.equal(raiz.ownerDocument.activeElement, titulo); assert.equal(desplazamientos, 1);
  pulsar(nodo, "[data-cronos-justificacion-ayuda]", {});
  assert.equal(raiz.ownerDocument.activeElement, botonAyuda);
  assert.match(nodo.innerHTML, /aria-expanded="true" aria-controls="cronos-justificacion-ayuda"/u);
  assert.doesNotMatch(nodo.innerHTML, /id="cronos-justificacion-ayuda" hidden/u);
  assert.match(nodo.innerHTML, /value="2026-10-20"/u); assert.match(nodo.innerHTML, /value="denegados" selected/u);
  pulsar(nodo, "[data-cronos-justificacion-ayuda]", {});
  assert.match(nodo.innerHTML, /id="cronos-justificacion-ayuda" hidden/u);
  assert.equal(consultas, 1); assert.equal(escrituras, 0);
  vista.desmontar();
});

test("la denegación elimina los pendientes anteriores y no permite navegar a ellos", async () => {
  const { nodo, raiz } = raizFalsa(); let denegado = false; let focos = 0;
  nodo.querySelector = () => ({ focus() { focos++; } });
  const vista = montarPermisosPropiosCronos({ raiz, anio: 2026,
    cliente: { consultarPermisos: async () => { if (denegado) throw new ErrorClienteSolicitudesCronos("acceso_denegado", 403); return datos(); }, solicitarPermiso: async () => {} } });
  await esperar(); assert.match(nodo.innerHTML, /data-cronos-ver-justificacion/u);
  denegado = true; await vista.recargar();
  assert.doesNotMatch(nodo.innerHTML, /data-cronos-ver-justificacion|cronos-permisos-just-panel|Asuntos propios/u);
  pulsar(nodo, "[data-cronos-ver-justificacion]", {}); assert.equal(focos, 0);
  vista.desmontar();
});

test("los textos de justificación se cargan y traducen en los idiomas del portal", async () => {
  const { cargarTextos } = await import("../../../comun/textos.js");
  const { crearTraductorJustificacionCronos } = await import("./i18n-permisos.js");
  let claves;
  for (const [idioma, esperado] of [["es", "Ver pendientes de justificar"], ["en", "View leave awaiting evidence"]]) {
    const catalogo = await cargarTextos("cronos-permisos", { idioma, avisar: (aviso) => assert.fail(aviso) });
    const mensajes = catalogo.seccion("justificacion"); claves ??= Object.keys(mensajes);
    assert.deepEqual(Object.keys(mensajes), claves);
    const t = crearTraductorJustificacionCronos(mensajes);
    assert.equal(t("ver_pendientes"), esperado); assert.throws(() => t("inexistente"), TypeError);
    assert.throws(() => crearTraductorJustificacionCronos({}), TypeError);
    const html = renderizarPermisosPropiosCronos({ estado: "listo", anio: 2026, datos: datos(), mensajesJustificacion: mensajes });
    assert.ok(html.includes(esperado));
  }
});
