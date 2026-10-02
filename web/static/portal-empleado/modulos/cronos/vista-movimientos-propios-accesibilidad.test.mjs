import assert from "node:assert/strict";
import test from "node:test";
import { ErrorClienteSolicitudesCronos } from "./cliente-solicitudes-http.js";
import { montarMovimientosPropiosCronos, renderizarMovimientosPropiosCronos } from "./vista-movimientos-propios.js";

function datos(c = { periodo: "rango", desde: "2026-01-01", hasta: "2026-12-31" }) {
  return { periodo: { tipo: c.periodo, desde: c.desde || "2026-01-01", hasta: c.hasta || "2026-12-31" },
    calendario: { disponible: true, dias: [] }, marcajes_por_dia: [],
    absentismos: [{ solicitud_ref: "permiso:cronos:solicitud:fixture-1", permiso_ref: "permiso:cronos:traslado", nombre: "Permiso sintético", desde: "2026-09-22", hasta: "2026-09-23", cantidad: 2, unidad: "dia", pendiente_justificar: true }],
    correcciones: [{ solicitud_ref: "correccion:cronos:fixture-1", fecha_civil: "2026-09-24", hora_pretendida: "08:00", movimiento: "entrada", estado: "pendiente_responsable", version: 1, solicitada_en: "2026-09-25T07:00:00Z" }],
  };
}
function diferida() { let resolver; let rechazar; const promesa = new Promise((si, no) => { resolver = si; rechazar = no; }); return { promesa, resolver, rechazar }; }
const esperar = async () => { for (let i = 0; i < 8; i++) await Promise.resolve(); };
const montar = (s, cliente) => montarMovimientosPropiosCronos({ raiz: s.raiz, anio: 2026, vistaCalendario: "mes", fechaSeleccionada: "2026-09-24", cliente });
const recibo = (e) => ({ solicitud_ref: `correccion:cronos:${e.clave_operacion}`, actuacion_ref: "actuacion:cronos:1", recibo_ref: "recibo:cronos:1", estado: "pendiente_responsable", version: 1, instante_utc: "2026-09-25T07:00:00Z", replay: false });

// El repintado retira los controles anteriores, como innerHTML en Chrome.
// La comprobación de navegador complementa este DOM mínimo de foco.
function superficie() {
  const documento = { body: {}, activeElement: null, hasFocus: () => true };
  documento.activeElement = documento.body;
  let html = ""; let controles = [];
  const coincide = (control, selector) => {
    const partes = selector.split(" ").at(-1);
    return [...partes.matchAll(/\[([a-z-]+)(?:="([^"]*)"|=([a-z_-]+))?\]/gu)].every(([, atributo, citado, simple]) => {
      const valor = citado ?? simple;
      return control.hasAttribute(atributo) && (valor === undefined || control.getAttribute(atributo) === valor);
    });
  };
  const nodo = {
    ownerDocument: documento, dataset: {}, eventos: {},
    get innerHTML() { return html; },
    set innerHTML(valor) {
      if (controles.includes(documento.activeElement)) documento.activeElement = documento.body;
      html = valor;
      controles = [...html.matchAll(/<(button|input|select|textarea|p|span|form)\b([^>]*)>/gu)].map(([, tag, texto]) => {
        const atributos = new Map([...texto.matchAll(/([a-z-]+)(?:="([^"]*)")?/gu)].map(([, clave, valor]) => [clave, valor ?? ""]));
        const dataset = Object.fromEntries([...atributos].filter(([clave]) => clave.startsWith("data-")).map(([clave, valor]) =>
          [clave.slice(5).replace(/-([a-z])/gu, (_, letra) => letra.toUpperCase()), valor]));
        const control = { tag, dataset, name: atributos.get("name"), value: atributos.get("value") ?? "", disabled: atributos.has("disabled"),
          getAttribute: (clave) => atributos.get(clave) ?? null,
          hasAttribute: (clave) => atributos.has(clave),
          focus() { if (!this.disabled) documento.activeElement = this; },
          closest: (selector) => coincide(control, selector) ? control : null,
        };
        if (tag === "textarea") control.value = html.match(/<textarea\b[^>]*>([^<]*)<\/textarea>/u)?.[1] ?? "";
        if (tag === "select" && control.name === "movimiento") {
          const opciones = html.match(/<select\b[^>]*name="movimiento"[^>]*>([\s\S]*?)<\/select>/u)?.[1] ?? "";
          control.value = opciones.match(/<option value="([a-z_]+)" selected/u)?.[1] ?? "entrada";
        }
        return control;
      });
    },
    contains: (control) => controles.includes(control),
    querySelector: (selector) => selector === ":focus" ? (controles.includes(documento.activeElement) ? documento.activeElement : null) : controles.find((control) => coincide(control, selector)) ?? null,
    addEventListener(tipo, fn) { this.eventos[tipo] = fn; },
    removeEventListener(tipo) { delete this.eventos[tipo]; }, remove() {},
  };
  documento.createElement = () => nodo;
  const raiz = { ownerDocument: documento, append() {} };
  const elegir = (selector) => { const control = nodo.querySelector(selector); assert.ok(control, selector); control.focus(); return control; };
  const pulsar = (selector) => { const control = elegir(selector); nodo.eventos.click({ target: control }); };
  const editar = (name, value) => { const campo = elegir(`[name="${name}"]`); campo.value = value; nodo.eventos.input({ type: "input", target: campo }); };
  const enviar = () => nodo.eventos.submit({ preventDefault() {}, target: {
    matches: (selector) => selector === "[data-cronos-olvido-formulario]",
    elements: { namedItem: (nombre) => nodo.querySelector(`[name="${nombre}"]`) },
  } });
  return { raiz, nodo, documento, elegir, pulsar, editar, enviar };
}

test("los recuentos tienen valor, etiqueta, icono decorativo y una sola acción nativa", () => {
  const html = renderizarMovimientosPropiosCronos({ estado: "listo", anio: 2026, datos: datos() });
  assert.match(html, /class="rejilla-kpi rejilla-kpi--compacta"/u);
  const tarjetas = [...html.matchAll(/<button type="button" class="tarjeta-kpi" data-cronos-recuento="([a-z]+)">(.*?)<\/button>/gu)];
  assert.deepEqual(tarjetas.map((m) => m[1]), ["correcciones", "ausencias", "pendiente"]);
  for (const [, , contenido] of tarjetas) {
    assert.match(contenido, /<span class="icono-kpi" aria-hidden="true"><svg/u);
    assert.match(contenido, /<strong class="valor-kpi">1<\/strong>/u);
    assert.match(contenido, /<span class="etiqueta-kpi">[^<]+<\/span>/u);
    assert.doesNotMatch(contenido, /<button|aria-pressed/u);
  }
});

test("las tarjetas conservan filtros, destino y borrador sin escribir ni recargar", async () => {
  const s = superficie(); let consultas = 0; let envios = 0;
  montar(s, { consultarMovimientos: async (c) => { consultas++; return datos(c); }, solicitarCorreccion: async () => { envios++; } });
  await esperar(); s.pulsar('[data-cronos-olvido="abrir"]'); s.editar("hora_pretendida", "07:35");
  for (const filtro of ["correcciones", "ausencias", "pendiente"]) {
    s.pulsar(`[data-cronos-recuento="${filtro}"]`);
    const destino = filtro === "correcciones" ? "estado" : "justificante";
    assert.equal(s.documento.activeElement, s.nodo.querySelector(`[data-cronos-filtro="${destino}"]`));
    assert.equal(s.nodo.querySelector('[name="hora_pretendida"]').value, "07:35");
    assert.match(s.nodo.innerHTML, filtro === "pendiente" ? /value="pendiente" selected/u : /value="" selected/u);
  }
  assert.equal(consultas, 1); assert.equal(envios, 0);
});

test("el error de envío devuelve foco y conserva contenido y clave al reintentar", async () => {
  const s = superficie(); let respuesta = diferida(); const peticiones = [];
  montar(s, { consultarMovimientos: async (c) => datos(c), solicitarCorreccion: (e) => { peticiones.push(e); return respuesta.promesa; } });
  await esperar(); s.pulsar('[data-cronos-olvido="abrir"]'); s.editar("fecha_civil", "2026-09-24"); s.editar("hora_pretendida", "07:35"); s.editar("movimiento", "salida");
  s.elegir("[data-cronos-olvido-enviar]"); let envio = s.enviar();
  assert.equal(s.documento.activeElement, s.nodo.querySelector("[data-cronos-olvido-formulario]"));
  respuesta.rechazar(new ErrorClienteSolicitudesCronos("servicio_no_disponible", 503)); await envio;
  assert.equal(s.documento.activeElement, s.nodo.querySelector("[data-cronos-olvido-enviar]"));
  assert.equal(s.nodo.querySelector('[name="hora_pretendida"]').value, "07:35");
  respuesta = diferida(); envio = s.enviar(); respuesta.resolver(recibo(peticiones[1])); await envio;
  assert.equal(peticiones[0].clave_operacion, peticiones[1].clave_operacion);
  assert.equal(s.documento.activeElement, s.nodo.querySelector("[data-cronos-olvido-enviar]"));
});

test("recargar conserva el campo editado y recupera el foco tras la lectura", async () => {
  const s = superficie(); const respuesta = diferida(); let consultas = 0;
  const vista = montar(s, { consultarMovimientos: (c) => ++consultas === 1 ? Promise.resolve(datos(c)) : respuesta.promesa, solicitarCorreccion: async () => ({}) });
  await esperar(); s.pulsar('[data-cronos-olvido="abrir"]'); s.editar("hora_pretendida", "07:35");
  const lectura = vista.recargar();
  assert.equal(s.documento.activeElement, s.nodo.querySelector("[data-cronos-movimientos-estado]"));
  respuesta.resolver(datos({ periodo: "anio" })); await lectura;
  assert.equal(s.documento.activeElement, s.nodo.querySelector('[name="hora_pretendida"]'));
  assert.equal(s.documento.activeElement.value, "07:35");
});

test("lecturas y envíos tardíos respetan el foco externo y la ventana inactiva", async () => {
  for (const operacion of ["lectura", "envio"]) for (const destino of ["externo", "ventana"]) {
    const s = superficie(); const respuesta = diferida(); let consultas = 0;
    const vista = montar(s, { consultarMovimientos: (c) => ++consultas === 1 ? Promise.resolve(datos(c)) : respuesta.promesa, solicitarCorreccion: () => respuesta.promesa });
    await esperar(); s.pulsar('[data-cronos-olvido="abrir"]'); s.editar("hora_pretendida", "07:35"); let pendiente;
    if (operacion === "lectura") pendiente = vista.recargar();
    else { s.elegir("[data-cronos-olvido-enviar]"); pendiente = s.enviar(); }
    const externo = {};
    if (destino === "externo") s.documento.activeElement = externo;
    else s.documento.hasFocus = () => false;
    if (operacion === "lectura") respuesta.resolver(datos({ periodo: "anio" }));
    else respuesta.rechazar(new ErrorClienteSolicitudesCronos("servicio_no_disponible", 503));
    await pendiente;
    assert.equal(s.documento.activeElement, destino === "externo" ? externo : s.documento.body);
  }
});

test("moverse a la ayuda durante el envío sustituye el destino provisional", async () => {
  const s = superficie(); const respuesta = diferida();
  montar(s, { consultarMovimientos: async (c) => datos(c), solicitarCorreccion: () => respuesta.promesa });
  await esperar(); s.pulsar('[data-cronos-olvido="abrir"]'); s.editar("hora_pretendida", "07:35");
  s.elegir("[data-cronos-olvido-enviar]"); const envio = s.enviar();
  s.elegir('[data-accion="ayuda"]');
  respuesta.rechazar(new ErrorClienteSolicitudesCronos("servicio_no_disponible", 503)); await envio;
  assert.equal(s.documento.activeElement, s.nodo.querySelector('[data-accion="ayuda"]'));
});

test("elegir un día muestra el detalle, mantiene el foco y no consulta ni pierde el borrador", async () => {
  const s = superficie(); let consultas = 0; let escrituras = 0;
  montar(s, { consultarMovimientos: async (c) => { consultas++; return datos(c); }, solicitarCorreccion: async () => { escrituras++; } });
  await esperar(); s.pulsar('[data-cronos-olvido="abrir"]'); s.editar("hora_pretendida", "07:35");
  const filtro = s.elegir('[data-cronos-filtro="justificante"]'); filtro.value = "pendiente";
  s.nodo.eventos.change({ type: "change", target: filtro });
  s.pulsar('[data-cronos-cal-dia][data-fecha="2026-09-22"]');
  const seleccionado = s.nodo.querySelector('[data-cronos-cal-dia][data-fecha="2026-09-22"]');
  assert.equal(seleccionado.tag, "button");
  assert.equal(seleccionado.getAttribute("aria-pressed"), "true");
  assert.equal(seleccionado.getAttribute("aria-controls"), "cronos-movpropios-detalle");
  assert.equal(s.documento.activeElement, seleccionado);
  assert.equal(s.nodo.querySelector('[data-cronos-cal-fecha]').value, "2026-09-22");
  assert.equal(s.nodo.querySelector('[name="hora_pretendida"]').value, "07:35");
  assert.match(s.nodo.innerHTML, /value="pendiente" selected/u);
  assert.match(s.nodo.innerHTML, /Ausencia: Permiso sintético/u);
  assert.match(s.nodo.innerHTML, /data-calendario-vista="mes"/u);
  s.pulsar('[data-cronos-cal-dia][data-fecha="2026-09-26"]');
  assert.match(s.nodo.innerHTML, /No hay datos registrados para esta fecha/u);
  assert.equal(consultas, 1); assert.equal(escrituras, 0);
});

test("un día anual conserva la vista; fechas ajenas o no válidas no consultan ni cambian selección", async () => {
  const s = superficie(); let consultas = 0;
  const vista = montarMovimientosPropiosCronos({ raiz: s.raiz, anio: 2026, fechaSeleccionada: "2026-09-24", cliente: {
    consultarMovimientos: async (c) => { consultas++; const d = datos(c); d.calendario.disponible = false; return d; }, solicitarCorreccion: async () => ({}),
  } });
  await esperar(); s.pulsar('[data-cronos-cal-dia][data-fecha="2026-01-06"]');
  assert.match(s.nodo.innerHTML, /data-calendario-vista="anio"/u);
  assert.match(s.nodo.innerHTML, /No hay datos registrados para esta fecha/u);
  assert.doesNotMatch(s.nodo.innerHTML, /data-fecha="2026-01-06" data-tipos/u);
  for (const fecha of ["2025-12-31", "2027-01-01", "2026-02-30", '<img src="x">']) {
    s.nodo.eventos.click({ target: { closest: (selector) => selector === "[data-cronos-cal-dia]" ? { dataset: { fecha } } : null } });
  }
  assert.equal(s.nodo.querySelector('[data-cronos-cal-fecha]').value, "2026-01-06");
  assert.equal(consultas, 1);
  vista.desmontar();
});
