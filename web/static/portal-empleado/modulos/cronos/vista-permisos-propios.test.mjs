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
function diferido() { let resolver; let rechazar; const promesa = new Promise((si, no) => { resolver = si; rechazar = no; }); return { promesa, resolver, rechazar }; }

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

test("cancelar devuelve el foco a Solicitar del permiso que abrió el formulario", async () => {
  for (const permisoRef of ["permiso:cronos:asuntos-propios", "permiso:cronos:horas-medico"]) {
    const { nodo, raiz } = raizFalsa(); const documento = raiz.ownerDocument;
    const body = {}; let html = ""; documento.activeElement = body;
    Object.defineProperty(nodo, "innerHTML", { get: () => html, set(valor) {
      html = valor; documento.activeElement = body;
    } });
    nodo.querySelectorAll = (selector) => selector !== "[data-cronos-solicitar]" ? []
      : Array.from(html.matchAll(/data-cronos-solicitar="([^"]+)"/gu), ([, ref]) => ({
        dataset: { cronosSolicitar: ref }, focus() { documento.activeElement = this; },
      }));
    let consultas = 0; let escrituras = 0;
    const vista = montarPermisosPropiosCronos({ raiz, anio: 2026,
      cliente: { consultarPermisos: async () => { consultas++; return datos(); },
        solicitarPermiso: async () => { escrituras++; } } });
    await esperar(); pulsar(nodo, "[data-cronos-solicitar]", { cronosSolicitar: permisoRef });
    assert.match(nodo.innerHTML, /data-cronos-permiso-formulario/u);
    documento.activeElement = {};
    pulsar(nodo, "[data-cronos-permiso-cerrar]", {});
    assert.doesNotMatch(nodo.innerHTML, /data-cronos-permiso-formulario/u);
    assert.equal(documento.activeElement.dataset?.cronosSolicitar, permisoRef);
    assert.equal(consultas, 1); assert.equal(escrituras, 0);
    vista.desmontar();
  }
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

// Reproduce el efecto del DOM: al repintar, el botón desaparece y el foco cae
// en body. Los controles nuevos sólo recuperan el foco mediante focus().
function raizConFocoAnio() {
  const { nodo, raiz } = raizFalsa(); const documento = raiz.ownerDocument;
  const body = {}; documento.activeElement = body;
  let html = ""; let botones = [];
  Object.defineProperty(nodo, "innerHTML", { get: () => html, set(valor) {
    if (botones.includes(documento.activeElement)) documento.activeElement = body;
    html = valor;
    botones = ["-1", "1"].map((paso) => {
      const etiqueta = html.match(new RegExp(`<button[^>]*data-cronos-anio="${paso}"[^>]*>`, "u"))?.[0] ?? "";
      return { dataset: { cronosAnio: paso }, disabled: /\sdisabled(?:\s|>)/u.test(etiqueta),
        getAttribute: (nombre) => nombre === "data-cronos-anio" ? paso : null,
        closest(selector) { return selector === "[data-cronos-anio]" ? this : null; },
        focus() { if (botones.includes(this) && !this.disabled) documento.activeElement = this; } };
    });
  } });
  nodo.contains = (elemento) => botones.includes(elemento);
  nodo.querySelector = (selector) => botones.find((b) => selector === `[data-cronos-anio="${b.dataset.cronosAnio}"]`) ?? null;
  const quitar = nodo.remove; nodo.remove = () => { if (nodo.contains(documento.activeElement)) documento.activeElement = body; quitar.call(nodo); };
  const cambiar = (paso) => { const boton = nodo.querySelector(`[data-cronos-anio="${paso}"]`); boton.focus(); nodo.eventos.click({ target: boton }); };
  return { nodo, raiz, documento, cambiar };
}

test("las dos flechas de año conservan foco durante carga y tras éxito o error", async () => {
  for (const paso of ["-1", "1"]) for (const falla of [false, true]) {
    const { nodo, raiz, documento, cambiar } = raizConFocoAnio(); const pendiente = diferido(); let llamadas = 0;
    const vista = montarPermisosPropiosCronos({ raiz, anio: 2026,
      cliente: { consultarPermisos: async () => ++llamadas === 1 ? datos() : pendiente.promesa, solicitarPermiso: async () => ({}) } });
    await esperar(); cambiar(paso);
    assert.match(nodo.innerHTML, /data-estado="cargando"/u);
    assert.equal(documento.activeElement, nodo.querySelector(`[data-cronos-anio="${paso}"]`), "el repintado de carga no deja el foco en body");
    if (falla) pendiente.rechazar(new ErrorClienteSolicitudesCronos("servicio_no_disponible", 503));
    else pendiente.resolver({ ...datos(), anio: 2026 + Number(paso), solicitudes: [] });
    await esperar(); assert.match(nodo.innerHTML, new RegExp(`data-estado="${falla ? "error" : "listo"}"`, "u"));
    assert.equal(documento.activeElement, nodo.querySelector(`[data-cronos-anio="${paso}"]`));
    assert.equal(llamadas, 2); vista.desmontar();
  }
});

test("en los límites de año el foco pasa a la flecha vecina habilitada", async () => {
  for (const [anio, paso, vecino] of [[2001, "-1", "1"], [2099, "1", "-1"]]) {
    const { nodo, raiz, documento, cambiar } = raizConFocoAnio(); const pendiente = diferido(); let llamadas = 0;
    const vista = montarPermisosPropiosCronos({ raiz, anio,
      cliente: { consultarPermisos: async ({ anio: consultado }) => ++llamadas === 1 ? { ...datos(), anio: consultado, solicitudes: [] } : pendiente.promesa, solicitarPermiso: async () => ({}) } });
    await esperar(); cambiar(paso);
    assert.equal(nodo.querySelector(`[data-cronos-anio="${paso}"]`).disabled, true);
    assert.equal(documento.activeElement, nodo.querySelector(`[data-cronos-anio="${vecino}"]`));
    pendiente.resolver({ ...datos(), anio: anio + Number(paso), solicitudes: [] }); await esperar();
    assert.equal(documento.activeElement, nodo.querySelector(`[data-cronos-anio="${vecino}"]`)); vista.desmontar();
  }
});

test("el retorno del año no roba el foco externo y una respuesta tardía o desmontada no lo mueve", async () => {
  const { nodo, raiz, documento, cambiar } = raizConFocoAnio(); const antigua = diferido(); const actual = diferido(); let llamadas = 0;
  const vista = montarPermisosPropiosCronos({ raiz, anio: 2026,
    cliente: { consultarPermisos: async () => ++llamadas === 1 ? antigua.promesa : actual.promesa, solicitarPermiso: async () => ({}) } });
  cambiar("-1"); const foco = documento.activeElement; const html = nodo.innerHTML;
  antigua.resolver(datos()); await esperar();
  assert.equal(documento.activeElement, foco); assert.equal(nodo.innerHTML, html, "el año antiguo no repinta");
  const externo = {}; documento.activeElement = externo;
  actual.resolver({ ...datos(), anio: 2025, solicitudes: [] }); await esperar();
  assert.equal(documento.activeElement, externo);
  vista.desmontar();
  const segunda = raizConFocoAnio(); const tardia = diferido();
  const nueva = montarPermisosPropiosCronos({ raiz: segunda.raiz, anio: 2026,
    cliente: { consultarPermisos: async () => tardia.promesa, solicitarPermiso: async () => ({}) } });
  nueva.desmontar(); segunda.documento.activeElement = externo; const antes = segunda.nodo.innerHTML;
  tardia.resolver(datos()); await esperar();
  assert.equal(segunda.documento.activeElement, externo); assert.equal(segunda.nodo.innerHTML, antes); assert.equal(segunda.nodo.eliminado, true);
});

test("Reintentar recupera lectura y Actualizar conserva filtro, página, borrador y clave sin escribir", async () => {
  const { nodo, raiz } = raizFalsa(); const entradas = []; let consultas = 0; let caer = true;
  const vista = montarPermisosPropiosCronos({ raiz, anio: 2026, tamanoPagina: 2,
    cliente: { consultarPermisos: async ({ anio }) => { consultas++; assert.equal(anio, 2026);
      if (caer) throw new ErrorClienteSolicitudesCronos("servicio_no_disponible", 503); return datosHistorial(); },
    solicitarPermiso: async (entrada) => { entradas.push(entrada); throw new ErrorClienteSolicitudesCronos("servicio_no_disponible", 503); } } });
  await esperar(); assert.match(nodo.innerHTML, /data-cronos-permisos-actualizar="">Reintentar/u);
  caer = false; pulsar(nodo, "[data-cronos-permisos-actualizar]", {}); await esperar();
  assert.match(nodo.innerHTML, /data-cronos-permisos-actualizar="">Actualizar/u);
  pulsar(nodo, "[data-cronos-solicitar]", { cronosSolicitar: "permiso:cronos:asuntos-propios" });
  editar(nodo, "desde", "2026-10-20"); editar(nodo, "hasta", "2026-10-21");
  filtrar(nodo, "denegados"); pulsar(nodo, "[data-cronos-historial-pagina]", { cronosHistorialPagina: "siguiente" });
  await enviar(nodo, { desde: "2026-10-20", hasta: "2026-10-21" });
  caer = true; pulsar(nodo, "[data-cronos-permisos-actualizar]", {}); await esperar();
  assert.match(nodo.innerHTML, /data-estado="error"/u);
  assert.doesNotMatch(nodo.innerHTML, /Asuntos propios|tarjeta-kpi|data-cronos-solicitar|data-cronos-permiso-formulario/u);
  assert.equal(entradas.length, 1);
  caer = false; pulsar(nodo, "[data-cronos-permisos-actualizar]", {}); await esperar();
  assert.match(nodo.innerHTML, /value="denegados" selected/u); assert.match(nodo.innerHTML, /Página 2 de 4/u);
  assert.match(nodo.innerHTML, /value="2026-10-20"/u); assert.match(nodo.innerHTML, /value="2026-10-21"/u);
  assert.equal(consultas, 4); assert.equal(entradas.length, 1);
  await enviar(nodo, { desde: "2026-10-20", hasta: "2026-10-21" });
  assert.equal(entradas[1].clave_operacion, entradas[0].clave_operacion);
  vista.desmontar(); await vista.recargar(); assert.equal(consultas, 4);
});

test("Actualizar oculta datos durante carga, aborta la lectura anterior y descarta su retorno", async () => {
  const { nodo, raiz } = raizFalsa(); const primera = diferido(); const ultima = diferido(); const signals = [];
  const vista = montarPermisosPropiosCronos({ raiz, anio: 2026,
    cliente: { consultarPermisos: async (_entrada, { signal }) => { signals.push(signal);
      return signals.length === 1 ? datos() : signals.length === 2 ? primera.promesa : ultima.promesa; }, solicitarPermiso: async () => assert.fail("escritura") } });
  await esperar(); pulsar(nodo, "[data-cronos-permisos-actualizar]", {});
  assert.match(nodo.innerHTML, /aria-disabled="true">Consultando permisos/u);
  assert.doesNotMatch(nodo.innerHTML, /Asuntos propios|tarjeta-kpi/u);
  pulsar(nodo, "[data-cronos-permisos-actualizar]", {}); assert.equal(signals.length, 2);
  const recarga = vista.recargar(); assert.equal(signals[1].aborted, true);
  ultima.rechazar(new ErrorClienteSolicitudesCronos("acceso_denegado", 403)); await recarga;
  const denegacion = nodo.innerHTML;
  primera.resolver(datos()); await esperar(); assert.equal(nodo.innerHTML, denegacion);
  assert.match(nodo.innerHTML, /data-estado="denegado"/u); assert.doesNotMatch(nodo.innerHTML, /Asuntos propios/u);
  vista.desmontar();
});

test("Actualizar y Reintentar conservan el recibo confirmado sin repetir la escritura", async () => {
  const { nodo, raiz } = raizFalsa(); let caer = false; let escrituras = 0;
  const vista = montarPermisosPropiosCronos({ raiz, anio: 2026,
    cliente: { consultarPermisos: async () => { if (caer) throw new ErrorClienteSolicitudesCronos("servicio_no_disponible", 503); return datos(); },
      solicitarPermiso: async (entrada) => { escrituras++; return reciboDe(entrada); } } });
  await esperar(); pulsar(nodo, "[data-cronos-solicitar]", { cronosSolicitar: "permiso:cronos:asuntos-propios" }); await enviar(nodo);
  caer = true; pulsar(nodo, "[data-cronos-permisos-actualizar]", {}); await esperar();
  assert.match(nodo.innerHTML, /recibo:cronos:conservado/u); assert.match(nodo.innerHTML, /datetime="2026-10-01T08:15:00Z"/u);
  caer = false; pulsar(nodo, "[data-cronos-permisos-actualizar]", {}); await esperar();
  assert.match(nodo.innerHTML, /recibo:cronos:conservado/u); assert.equal(escrituras, 1); vista.desmontar();
});

test("la acción de consulta traduce y escapa sus mensajes en ambos idiomas", async () => {
  const { cargarTextos } = await import("../../../comun/textos.js");
  const { crearTraductorConsultaPermisosCronos } = await import("./i18n-permisos-consulta.js");
  let claves;
  for (const [idioma, actualizar, reintentar] of [["es", "Actualizar", "Reintentar"], ["en", "Refresh", "Retry"]]) {
    const textos = await cargarTextos("cronos-permisos-consulta", { idioma, avisar: (aviso) => assert.fail(aviso) });
    const mensajesConsulta = textos.seccion("consulta"); claves ??= Object.keys(mensajesConsulta); assert.deepEqual(Object.keys(mensajesConsulta), claves);
    const t = crearTraductorConsultaPermisosCronos(mensajesConsulta);
    assert.equal(t("actualizar"), actualizar); assert.equal(t("reintentar"), reintentar);
    assert.throws(() => t("inexistente"), TypeError); assert.throws(() => crearTraductorConsultaPermisosCronos({}), TypeError);
    assert.ok(renderizarPermisosPropiosCronos({ estado: "listo", anio: 2026, datos: datos(), mensajesConsulta }).includes(`>${actualizar}</button>`));
    assert.ok(renderizarPermisosPropiosCronos({ estado: "error", anio: 2026, mensajesConsulta }).includes(`>${reintentar}</button>`));
    const hostil = renderizarPermisosPropiosCronos({ estado: "error", anio: 2026, mensajesConsulta: { ...mensajesConsulta, error: '<img src=x onerror="x()">' } });
    assert.match(hostil, /&lt;img/u); assert.doesNotMatch(hostil, /<img/u);
  }
});

test("Actualizar no interrumpe ni repite una solicitud que está enviándose", async () => {
  const { nodo, raiz } = raizFalsa(); const pendiente = diferido(); let consultas = 0; let escrituras = 0; let entrada;
  const vista = montarPermisosPropiosCronos({ raiz, anio: 2026,
    cliente: { consultarPermisos: async () => { consultas++; return datos(); },
      solicitarPermiso: async (e) => { escrituras++; entrada = e; return pendiente.promesa; } } });
  await esperar(); pulsar(nodo, "[data-cronos-solicitar]", { cronosSolicitar: "permiso:cronos:asuntos-propios" });
  const envio = enviar(nodo);
  assert.match(nodo.innerHTML, /data-cronos-permisos-actualizar="" disabled/u);
  pulsar(nodo, "[data-cronos-permisos-actualizar]", {});
  assert.equal(consultas, 1); assert.equal(escrituras, 1);
  pendiente.resolver(reciboDe(entrada)); await envio;
  assert.equal(consultas, 2); assert.equal(escrituras, 1); assert.match(nodo.innerHTML, /recibo:cronos:conservado/u);
  vista.desmontar();
});

test("el catálogo actualizado bloquea un permiso no solicitable y conserva borrador y clave", async () => {
  for (const retirar of [false, true]) {
    const { nodo, raiz } = raizFalsa(); let disponible = true; const envios = [];
    const vista = montarPermisosPropiosCronos({ raiz, anio: 2026,
      cliente: { consultarPermisos: async () => { const d = datos();
        if (!disponible) { if (retirar) d.permisos.shift(); else d.permisos[0].solicitable = false; }
        return d; }, solicitarPermiso: async (entrada) => { envios.push(entrada); throw new ErrorClienteSolicitudesCronos("servicio_no_disponible", 503); } } });
    await esperar(); pulsar(nodo, "[data-cronos-solicitar]", { cronosSolicitar: "permiso:cronos:asuntos-propios" });
    editar(nodo, "desde", "2026-10-20"); editar(nodo, "hasta", "2026-10-21");
    await enviar(nodo, { desde: "2026-10-20", hasta: "2026-10-21" });
    const clave = envios[0].clave_operacion; envios.length = 0;
    disponible = false; pulsar(nodo, "[data-cronos-permisos-actualizar]", {}); await esperar();
    assert.match(nodo.innerHTML, /El permiso seleccionado ya no admite solicitudes/u);
    assert.doesNotMatch(nodo.innerHTML, /data-cronos-permiso-formulario/u);
    await enviar(nodo, { desde: "2026-10-20", hasta: "2026-10-21" }); assert.equal(envios.length, 0);
    disponible = true; pulsar(nodo, "[data-cronos-permisos-actualizar]", {}); await esperar();
    assert.match(nodo.innerHTML, /value="2026-10-20"/u); assert.match(nodo.innerHTML, /value="2026-10-21"/u);
    await enviar(nodo, { desde: "2026-10-20", hasta: "2026-10-21" }); assert.equal(envios[0].clave_operacion, clave);
    vista.desmontar();
  }
});
