import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { crearTraductorSolicitudes, MENSAJES_SOLICITUDES_ES } from "./i18n.js";
import { montarVistaSolicitudes, renderizarSolicitudes } from "./vista.js";

const datos = {
  tramites: [
    { referencia: "SOL-001", titulo: "Solicitud <propia>", fecha: "2026-09-20", unidad: "Personal", estado: "pendiente_subsanacion", paso: "Revisión", descripcion: "Falta <documento>", siguienteAccion: "Aportar prueba", historial: [{ fecha: "2026-09-20", titulo: "Registrada", detalle: "Entrada recibida" }] },
    { referencia: "SOL-002", titulo: "Reconocimiento", fecha: "2026-09-21", estado: "finalizada", historial: [] },
  ],
  catalogo: [{ categoria: "Personal", titulo: "Servicios previos", descripcion: "Preparar información" }],
  certificados: [{ tipo: "Servicios", alcance: "Períodos reconocidos", situacion: "Tipo consultable" }],
};

function raizFalsa() {
  const eventos = new Map();
  return {
    innerHTML: "",
    eventos,
    focos: [],
    replaceChildren() { this.innerHTML = ""; },
    addEventListener(tipo, callback) { eventos.set(tipo, callback); },
    removeEventListener(tipo) { eventos.delete(tipo); },
    querySelector(selector) { return { focus: () => this.focos.push(selector) }; },
    querySelectorAll(selector) {
      if (selector !== "[data-solicitudes-detalle]") return [];
      return [...this.innerHTML.matchAll(/data-solicitudes-detalle="([^"]+)"/g)].map(([, referencia]) => ({
        dataset: { solicitudesDetalle: referencia },
        focus: () => this.focos.push(`detalle:${referencia}`),
      }));
    },
  };
}
const tick = () => new Promise((resolver) => setImmediate(resolver));
const objetivo = (atributo, valor = "") => ({ closest: (selector) => selector === `[${atributo}]` ? { dataset: { solicitudesDetalle: valor } } : null });
const objetivoTab = (pestana) => ({ closest: (selector) => selector === "[data-solicitudes-tab]" ? { dataset: { solicitudesTab: pestana } } : null });

test("vista carga el catálogo i18n con URL F2 inmutable", async () => {
  const vista = await readFile(new URL("vista.js", import.meta.url), "utf8");
  assert.match(vista, /from "\.\/i18n\.js\?v=20260924-f2-web2"/);
  const modulo = await import("./vista.js?v=20260926-integracion-bolsa-ct-v1");
  assert.equal(typeof modulo.montarVistaSolicitudes, "function");
  assert.match(modulo.renderizarSolicitudes(), /Consulta no configurada/);
});

test("sin fuente autorizada muestra estado no configurado y ningún recibo ni expediente sintético", () => {
  const html = renderizarSolicitudes();
  assert.match(html, /Consulta no configurada/);
  assert.match(html, /Consulta el estado y el seguimiento de tus trámites/);
  assert.doesNotMatch(html, /SOL-2026|Antonio López|recibo:/);
  assert.match(html, /Ningún trámite coincide con los filtros/);
  assert.match(html, /data-solicitudes-tab="certificados"/);
  assert.doesNotMatch(html, /localStorage|sessionStorage|document\.cookie|javascript:/i);
});

test("lista, filtros, ficha e historia usan solo la fuente inyectada y escapan sus datos", () => {
  const html = renderizarSolicitudes({ situacion: "disponible", datos, filtro: "pendiente_subsanacion", busqueda: "SOL-001" });
  assert.match(html, /SOL-001/);
  assert.doesNotMatch(html, /SOL-002/);
  assert.match(html, /Solicitud &lt;propia&gt;/);
  assert.doesNotMatch(html, /Solicitud <propia>/);
  assert.match(html, /Pendiente de subsanación/);
  const ficha = renderizarSolicitudes({ situacion: "disponible", datos, pestana: "seguimiento", seleccionada: "SOL-001" });
  assert.match(ficha, /Entrada recibida/);
  assert.match(ficha, /Falta &lt;documento&gt;/);
  assert.match(ficha, /Siguiente acción indicada/);
  assert.match(ficha, /disabled aria-disabled="true"/);
  assert.doesNotMatch(ficha, /<documento>/);
  const sinHistoria = renderizarSolicitudes({ situacion: "disponible", datos, pestana: "seguimiento", seleccionada: "SOL-002" });
  assert.match(sinHistoria, /La fuente no ha devuelto hitos/);
});

test("catálogo y certificados son consultables, las operaciones permanecen deshabilitadas", () => {
  const nueva = renderizarSolicitudes({ situacion: "disponible", datos, pestana: "nueva" });
  assert.match(nueva, /Servicios previos/);
  assert.match(nueva, /Iniciar trámite/);
  assert.match(nueva, /Pendiente de conectar el caso de uso/);
  assert.match(nueva, /disabled aria-disabled="true"/);
  const certificados = renderizarSolicitudes({ situacion: "disponible", datos, pestana: "certificados" });
  assert.match(certificados, /Períodos reconocidos/);
  assert.match(certificados, /Emitir o descargar/);
  assert.match(certificados, /disabled aria-disabled="true"/);
  assert.doesNotMatch(certificados, /CERT-2026/);
});

test("la vista pura descarta trámites y selección si la consulta no está disponible", () => {
  for (const situacion of ["denegado", "error", "no_configurado", "invalido", "vacio"]) {
    const html = renderizarSolicitudes({ situacion, datos, pestana: "seguimiento", seleccionada: "SOL-001" });
    assert.doesNotMatch(html, /SOL-001|SOL-002|Entrada recibida|Solicitud &lt;propia&gt;/);
    assert.match(html, /Selecciona un trámite de la bandeja/);
  }
});

test("una bandeja vacía conserva catálogo y certificados autorizados sin inventar trámites", () => {
  const vacio = { situacion: "vacio", datos, pestana: "nueva" };
  assert.match(renderizarSolicitudes(vacio), /Servicios previos/);
  assert.match(renderizarSolicitudes({ ...vacio, pestana: "certificados" }), /Períodos reconocidos/);
  assert.doesNotMatch(renderizarSolicitudes({ ...vacio, pestana: "bandeja" }), /SOL-001|SOL-002/);
});

test("de la bandeja se abre la ficha seleccionada y se vuelve con foco al listado", async () => {
  const raiz = raizFalsa();
  montarVistaSolicitudes({ raiz, fuente: { consultar: () => datos } });
  await tick();
  raiz.eventos.get("click")({ target: objetivo("data-solicitudes-detalle", "SOL-001") });
  assert.match(raiz.innerHTML, /Entrada recibida/);
  assert.match(raiz.innerHTML, /Volver a mis trámites/);
  assert.match(raiz.innerHTML, /aria-labelledby="solicitudes-tab-seguimiento"/);
  assert.equal(raiz.focos.at(-1), "#solicitudes-panel-actual");
  raiz.eventos.get("click")({ target: objetivo("data-solicitudes-volver") });
  assert.match(raiz.innerHTML, /Trámites propios consultados/);
  assert.match(raiz.innerHTML, /SOL-001/);
  assert.equal(raiz.focos.at(-1), "detalle:SOL-001");
  raiz.eventos.get("click")({ target: objetivo("data-solicitudes-detalle", "SOL-002") });
  assert.match(raiz.innerHTML, /Reconocimiento/);
  raiz.eventos.get("click")({ target: objetivo("data-solicitudes-volver") });
  assert.equal(raiz.focos.at(-1), "detalle:SOL-002");
});

test("clic y flechas, Home y End conservan foco en la pestaña seleccionada", async () => {
  const raiz = raizFalsa();
  const montada = montarVistaSolicitudes({ raiz, fuente: { consultar: () => datos } });
  await tick();
  raiz.eventos.get("click")({ target: objetivoTab("nueva") });
  assert.match(raiz.innerHTML, /aria-labelledby="solicitudes-tab-nueva"/);
  assert.equal(raiz.focos.at(-1), '[data-solicitudes-tab="nueva"]');
  const tecla = (desde, key, destino) => {
    let evitado = false;
    raiz.eventos.get("keydown")({ target: objetivoTab(desde), key, preventDefault() { evitado = true; } });
    assert.equal(evitado, true);
    assert.match(raiz.innerHTML, new RegExp(`aria-labelledby="solicitudes-tab-${destino}"`));
    assert.equal(raiz.focos.at(-1), `[data-solicitudes-tab="${destino}"]`);
  };
  tecla("nueva", "ArrowRight", "seguimiento");
  tecla("seguimiento", "Home", "bandeja");
  tecla("bandeja", "End", "certificados");
  tecla("certificados", "ArrowLeft", "seguimiento");
  tecla("seguimiento", "ArrowLeft", "nueva");
  montada.desmontar();
  assert.equal(raiz.eventos.size, 0);
  assert.equal(raiz.innerHTML, "");
});

test("cambiar pestaña en denegación conserva foco sin mostrar datos", async () => {
  const raiz = raizFalsa();
  montarVistaSolicitudes({ raiz, fuente: { consultar: () => ({ ...datos, estado: "denegado" }) } });
  await tick();
  raiz.eventos.get("click")({ target: objetivoTab("certificados") });
  assert.equal(raiz.focos.at(-1), '[data-solicitudes-tab="certificados"]');
  assert.doesNotMatch(raiz.innerHTML, /SOL-001|Períodos reconocidos|Servicios previos/);
});

test("respuesta asíncrona conserva foco en la pestaña activa sin revelar denegados", async () => {
  const raiz = raizFalsa();
  let resolver;
  raiz.ownerDocument = { activeElement: null };
  raiz.contains = () => true;
  montarVistaSolicitudes({ raiz, fuente: { consultar: () => new Promise((resuelve) => { resolver = resuelve; }) } });
  await tick();
  raiz.eventos.get("click")({ target: objetivoTab("certificados") });
  raiz.ownerDocument.activeElement = objetivoTab("certificados");
  resolver({ ...datos, estado: "denegado" });
  await tick();
  assert.equal(raiz.focos.at(-1), '[data-solicitudes-tab="certificados"]');
  assert.doesNotMatch(raiz.innerHTML, /SOL-001|Períodos reconocidos|Servicios previos/);
});

test("estados expresos no filtran filas, mientras vacío conserva catálogos", async () => {
  for (const estado of ["denegado", "no_configurado", "error", "invalido"]) {
    const raiz = raizFalsa();
    montarVistaSolicitudes({ raiz, fuente: { consultar: () => ({ ...datos, estado }) } });
    await tick();
    assert.doesNotMatch(raiz.innerHTML, /SOL-001|Servicios previos|Períodos reconocidos/);
    raiz.eventos.get("click")({ target: objetivo("data-solicitudes-detalle", "SOL-001") });
    assert.doesNotMatch(raiz.innerHTML, /Entrada recibida/);
  }
  const vacia = raizFalsa();
  montarVistaSolicitudes({ raiz: vacia, fuente: { consultar: () => ({ ...datos, estado: "vacio" }) } });
  await tick();
  assert.doesNotMatch(vacia.innerHTML, /SOL-001/);
  vacia.eventos.get("click")({ target: { closest: (selector) => selector === "[data-solicitudes-tab]" ? { dataset: { solicitudesTab: "nueva" } } : null } });
  assert.match(vacia.innerHTML, /Servicios previos/);
  vacia.eventos.get("click")({ target: { closest: (selector) => selector === "[data-solicitudes-tab]" ? { dataset: { solicitudesTab: "certificados" } } : null } });
  assert.match(vacia.innerHTML, /Períodos reconocidos/);
});
test("montaje consulta una vez, representa vacío, denegado y error, y cancela al desmontar", async () => {
  const raiz = raizFalsa();
  let consultas = 0;
  let señal;
  const montada = montarVistaSolicitudes({ raiz, fuente: { consultar: async ({ signal }) => { consultas++; señal = signal; return datos; } } });
  assert.match(raiz.innerHTML, /Cargando trámites/);
  await tick();
  assert.equal(consultas, 1);
  assert.match(raiz.innerHTML, /SOL-001/);
  montada.desmontar();
  assert.equal(señal.aborted, true);
  assert.equal(raiz.innerHTML, "");
  assert.equal(raiz.eventos.size, 0);
  const vacia = raizFalsa();
  montarVistaSolicitudes({ raiz: vacia, fuente: { consultar: () => ({ tramites: [] }) } });
  await tick();
  assert.match(vacia.innerHTML, /No hay trámites en esta consulta/);
  const denegada = raizFalsa();
  montarVistaSolicitudes({ raiz: denegada, fuente: { consultar: () => ({ estado: "denegado" }) } });
  await tick();
  assert.match(denegada.innerHTML, /Acceso denegado/);
  const error = raizFalsa();
  montarVistaSolicitudes({ raiz: error, fuente: { consultar: () => { throw new Error("secreto"); } } });
  await tick();
  assert.match(error.innerHTML, /No se pudo cargar la consulta/);
  assert.doesNotMatch(error.innerHTML, /secreto/);
});

test("respuesta tardía no repinta tras desmontaje y el catálogo i18n exige claves completas", async () => {
  const raiz = raizFalsa();
  let resolver;
  const pendiente = new Promise((resuelve) => { resolver = resuelve; });
  const montada = montarVistaSolicitudes({ raiz, fuente: { consultar: () => pendiente } });
  await tick();
  montada.desmontar();
  resolver(datos);
  await tick();
  assert.equal(raiz.innerHTML, "");
  assert.throws(() => crearTraductorSolicitudes({ titulo: "Parcial" }), /incompleto/);
  assert.equal(crearTraductorSolicitudes(MENSAJES_SOLICITUDES_ES)("anuncio_detalle", { referencia: "SOL-001" }), "Ficha de SOL-001 seleccionada.");
});

test("CSS conserva paneles, tabla con scroll y foco, adaptación 1024/390 y alto contraste", async () => {
  const css = await readFile(new URL("solicitudes.css", import.meta.url), "utf8");
  assert.match(css, /var\(--portal-fondo|var\(--portal-superficie-alterna/);
  assert.match(css, /overflow-x: auto/);
  assert.match(css, /focus-within/);
  assert.match(css, /max-width: 1024px/);
  assert.match(css, /max-width: 390px/);
  assert.match(css, /forced-colors: active/);
  assert.doesNotMatch(css, /#[0-9a-f]{3,8}\b/i);
});
