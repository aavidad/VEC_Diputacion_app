import test from "node:test";
import assert from "node:assert/strict";
import { montarMeritosReales } from "./meritos-reales-montaje.js";

function contenedorFalso() {
  const cabeza = [];
  const documento = { head: { append: (nodo) => cabeza.push(nodo) },
    createElement: (tipo) => ({ tipo, dataset: {}, remove() {}, setAttribute() {} }) };
  return { cabeza, contenedor: { ownerDocument: documento, isConnected: true,
    innerHTML: "", replaceChildren() { this.innerHTML = ""; },
    addEventListener() {}, removeEventListener() {}, contains() { return true; } } };
}

test("la cartera abierta consulta una vez y pide motivos en el idioma elegido", async () => {
  const { cabeza, contenedor } = contenedorFalso();
  const llamadas = [];
  const fetchImpl = async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    return { status: 200, ok: true, async json() { return { data: { hechos: [], total: 0,
      filtro: "todos", motivos_retirada: [{ etiqueta: "Incorrect document",
        motivo: { entrada_clave: "motivo_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" } }],
      corte_en: "2026-10-12T16:00:00Z", recibo_acceso: { auditoria_ref: "aud:1" } } }; } };
  };
  const vista = montarMeritosReales({ contenedor, fetchImpl, idioma: "en",
    ubicacion: new URL("https://ejemplo.local/area-personal/?lang=en"), historial: { replaceState() {} } });
  try {
    await vista.preparada;
    assert.equal(llamadas.length, 1);
    assert.equal(llamadas[0].ruta, "/api/vec/meritos/cartera?filtro=todos&limite=25&lang=en");
    assert.equal(llamadas[0].opciones.credentials, "omit");
    assert.match(contenedor.innerHTML, /class="panel meritos-reales"/u);
    assert.equal(cabeza.length, 1);
  } finally { vista.desmontar(); }
});

function montajeInteractivo(fetchImpl, url = "https://ejemplo.local/area-personal/?vista=meritos") {
  const eventos = new Map();
  const documento = { activeElement: null, head: { append() {} },
    createElement: () => ({ remove() {}, setAttribute() {} }) };
  const contenedor = { ownerDocument: documento, isConnected: true, innerHTML: "",
    replaceChildren() { this.innerHTML = ""; }, contains: () => true,
    querySelector: () => null, querySelectorAll: () => [],
    addEventListener(tipo, fn) { eventos.set(tipo, fn); }, removeEventListener() {} };
  const vista = montarMeritosReales({ contenedor, fetchImpl, idioma: "es",
    ubicacion: new URL(url), historial: { replaceState() {} } });
  const pulsar = (datos, atributos = []) => {
    const boton = { dataset: datos, hasAttribute: (nombre) => atributos.includes(nombre) };
    eventos.get("click")({ target: { closest: () => boton } });
  };
  const enviar = () => eventos.get("submit")({ preventDefault() {}, target: {
    id: "meritos-reales-retirada-formulario", elements: { namedItem: () => ({ value: "0" }) },
  } });
  const editar = (name, value, files) => eventos.get(name === "documento" ? "change" : "input")({
    type: name === "documento" ? "change" : "input",
    target: { name, value, files, closest: () => true },
  });
  const enviarAlta = () => eventos.get("submit")({ preventDefault() {}, target: {
    id: "meritos-reales-alta-formulario", reportValidity: () => true,
  } });
  return { vista, contenedor, documento, pulsar, enviar, editar, enviarAlta };
}

function pagina(hechos, filtro = "todos") {
  return { data: { hechos, total: hechos.length, filtro, corte_en: "2026-10-12T16:00:00Z",
    recibo_acceso: { auditoria_ref: "aud:1" }, motivos_retirada: [{ etiqueta: "Duplicado",
      motivo: { catalogo_id: "motivos_meritos_cartera", catalogo_version: 1,
        catalogo_huella_sha256: "a".repeat(64), entrada_clave: "motivo_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" } }] } };
}

const hecho = (version, estado = "disponible") => ({ hecho_ref: "hecho:uno", denominacion: "Curso acreditado",
  tipo: "curso_superacion", version_hecho: version, disponibilidad: { estado, version: estado === "retirado" ? 1 : 0 },
  ...(estado === "retirado" ? { retirada: { recibo_ref: "recibo:uno" } } : {}) });

test("un 409 definitivo permite retirar después con versión y clave nuevas", async () => {
  let lecturas = 0;
  const ordenes = [];
  const fetchImpl = async (_ruta, opciones) => opciones.method === "GET"
    ? { status: 200, ok: true, async json() { return pagina([hecho(++lecturas === 1 ? 1 : 2,
      lecturas >= 3 ? "retirado" : "disponible")]); } }
    : { status: ordenes.push(JSON.parse(opciones.body)) === 1 ? 409 : 201,
      get ok() { return this.status === 201; }, async json() { return this.ok
        ? { data: { recibo: { referencia: "recibo:uno" }, repetida: false, auditoria_acceso_ref: "aud:2" } }
        : { error: { codigo: "conflicto" } }; } };
  const { vista, contenedor, pulsar, enviar } = montajeInteractivo(fetchImpl);
  try {
    await vista.preparada;
    pulsar({ meritosRetirar: "hecho:uno" });
    enviar();
    await new Promise(setImmediate);
    assert.equal(vista.pendiente(), null, "el 409 rechazó definitivamente la orden");
    assert.match(contenedor.innerHTML, /La cartera cambió/u);
    pulsar({}, ["data-meritos-consultar"]);
    await new Promise(setImmediate);
    pulsar({ meritosRetirar: "hecho:uno" });
    enviar();
    await new Promise(setImmediate);
    assert.deepEqual(ordenes.map((o) => o.hecho_version_esperada), [1, 2]);
    assert.notEqual(ordenes[0].clave_idempotencia, ordenes[1].clave_idempotencia);
    assert.equal(vista.pendiente(), null);
    assert.match(contenedor.innerHTML, /Recibo recibo:uno/u);
  } finally { vista.desmontar(); }
});

test("una respuesta incierta conserva clave y aviso aunque Disponibles quede vacío", async () => {
  let lecturas = 0;
  const ordenes = [];
  const fetchImpl = async (_ruta, opciones) => opciones.method === "GET"
    ? { status: 200, ok: true, async json() { return pagina(++lecturas === 1 ? [hecho(1)] : [], "disponibles"); } }
    : { status: 503, ok: false, async json() { ordenes.push(JSON.parse(opciones.body));
      return { error: { codigo: "no_disponible" } }; } };
  const { vista, contenedor, pulsar, enviar } = montajeInteractivo(fetchImpl,
    "https://ejemplo.local/area-personal/?vista=meritos&filtro=disponibles");
  try {
    await vista.preparada;
    pulsar({ meritosRetirar: "hecho:uno" });
    enviar();
    await new Promise(setImmediate);
    const clave = vista.pendiente()?.orden.clave_idempotencia;
    assert.ok(clave);
    pulsar({}, ["data-meritos-consultar"]);
    await new Promise(setImmediate);
    assert.equal(vista.pendiente()?.orden.clave_idempotencia, clave);
    assert.match(contenedor.innerHTML, /Aún no sabemos si se retiró/u);
    assert.match(contenedor.innerHTML, /data-meritos-filtro="todos"/u);
    assert.equal(ordenes.length, 1);
  } finally { vista.desmontar(); }
});

test("al repintar un filtro conserva el foco de teclado en el control equivalente", async () => {
  const fetchImpl = async () => ({ status: 200, ok: true, async json() { return pagina([hecho(1)]); } });
  const { vista, contenedor, documento, pulsar } = montajeInteractivo(fetchImpl);
  try {
    await vista.preparada;
    let restauraciones = 0;
    documento.activeElement = { dataset: { meritosFiltro: "todos" } };
    contenedor.querySelectorAll = (selector) => selector === "[data-meritos-filtro]"
      ? [{ dataset: { meritosFiltro: "todos" }, focus(opciones) {
        assert.equal(opciones.preventScroll, true);
        restauraciones += 1;
      } }] : [];
    pulsar({ meritosFiltro: "todos" });
    await new Promise(setImmediate);
    assert.ok(restauraciones >= 2, "mantiene el foco durante carga y resultado");
  } finally { vista.desmontar(); }
});

test("un PDF incierto conserva archivo y clave; al reintentar registra solo después de admitirlo", async () => {
  const peticiones = [];
  let cargas = 0;
  const fetchImpl = async (ruta, opciones) => {
    peticiones.push({ ruta, opciones });
    if (ruta.includes("/comprobantes/estado?")) return { status: 404, ok: false, async json() {
      return { error: { codigo: "meritos.error.comprobante_no_encontrado" } };
    } };
    if (opciones.method === "GET") return { status: 200, ok: true, async json() { return pagina([]); } };
    if (ruta.endsWith("/comprobantes")) {
      cargas += 1;
      return { status: cargas === 1 ? 503 : 200, ok: cargas > 1, async json() {
        return cargas === 1 ? { error: { codigo: "no_disponible" } }
          : { data: { comprobante_ref: "ref:pdf1", carga_id: "ref:carga1",
            registrado_en: "2026-10-14T12:00:00Z", estado: "admitido" } };
      } };
    }
    const orden = JSON.parse(opciones.body);
    return { status: 200, ok: true, async json() { return { data: { referencia: "ref:merito1",
      auditoria_ref: "aud:1", accion: "meritos.hecho.declarar", clave_idempotencia: orden.clave_idempotencia,
      registrado_en: "2026-10-14T12:00:00Z", registro: { hecho: { estado: "declarado", version: 1,
        denominacion: orden.denominacion } } } }; } };
  };
  const { vista, pulsar, editar, enviarAlta, contenedor } = montajeInteractivo(fetchImpl);
  const archivo = new Blob(["%PDF-1.7"], { type: "application/pdf" });
  archivo.name = "titulo.pdf";
  try {
    await vista.preparada;
    pulsar({}, ["data-meritos-alta"]);
    editar("denominacion", "Título de Administración");
    editar("fecha_desde", "2026-01-01");
    editar("documento", "", [archivo]);
    assert.match(contenedor.innerHTML, /titulo\.pdf/u);
    enviarAlta();
    await new Promise(setImmediate);
    const pendiente = vista.pendiente();
    assert.equal(pendiente.alta.archivo, archivo);
    assert.equal(peticiones.filter((p) => p.ruta.endsWith("/declaraciones")).length, 0);
    assert.match(contenedor.innerHTML, /No se pudo confirmar la carga del PDF/u);
    enviarAlta();
    await new Promise(setImmediate);
    const cargasHechas = peticiones.filter((p) => p.ruta.endsWith("/comprobantes"));
    assert.equal(cargasHechas.length, 2);
    assert.equal(peticiones.filter((p) => p.ruta.includes("/comprobantes/estado?")).length, 1);
    assert.equal(cargasHechas[0].opciones.body.get("clave_idempotencia"), pendiente.orden.clave_idempotencia);
    assert.equal(cargasHechas[1].opciones.body.get("clave_idempotencia"), pendiente.orden.clave_idempotencia);
    const declaracion = JSON.parse(peticiones.find((p) => p.ruta.endsWith("/declaraciones")).opciones.body);
    assert.equal(declaracion.comprobante_ref, "ref:pdf1");
    assert.equal(declaracion.clave_idempotencia, pendiente.orden.clave_idempotencia);
    assert.equal(declaracion.evidencias, undefined);
    assert.equal(vista.pendiente(), null);
  } finally { vista.desmontar(); }
});

test("si el PDF no está disponible muestra una indicación legible y no el código interno", async () => {
  const fetchImpl = async (ruta) => ruta.endsWith("/original")
    ? { status: 503, ok: false, async json() {
      return { error: { codigo: "meritos.error.comprobante_no_disponible" } };
    } }
    : { status: 200, ok: true, async json() { return pagina([
      { ...hecho(1), evidencias: [{ id: "ref:original", version: 1 }] },
    ]); } };
  const { vista, pulsar, contenedor } = montajeInteractivo(fetchImpl);
  try {
    await vista.preparada;
    pulsar({ meritosDescargar: "ref:original" });
    await new Promise(setImmediate);
    assert.match(contenedor.innerHTML, /El comprobante no está disponible ahora/u);
    assert.doesNotMatch(contenedor.innerHTML, /meritos\.error\.comprobante_no_disponible/u);
  } finally { vista.desmontar(); }
});

test("un nuevo intento de descarga retira el aviso anterior y conserva el foco", async () => {
  let lecturas = 0;
  const fetchImpl = async (ruta) => {
    if (ruta.endsWith("/original")) {
      lecturas += 1;
      return lecturas === 1 ? { status: 503, ok: false, async json() {
        return { error: { codigo: "meritos.error.comprobante_no_disponible" } };
      } } : new Promise(() => {});
    }
    return { status: 200, ok: true, async json() { return pagina([
      { ...hecho(1), evidencias: [{ id: "ref:original", version: 1 }] },
    ]); } };
  };
  const { vista, pulsar, contenedor, documento } = montajeInteractivo(fetchImpl);
  try {
    await vista.preparada;
    pulsar({ meritosDescargar: "ref:original" }); await new Promise(setImmediate);
    assert.match(contenedor.innerHTML, /El comprobante no está disponible ahora/u);
    let foco = 0;
    documento.activeElement = { dataset: { meritosDescargar: "ref:original" } };
    contenedor.querySelectorAll = (selector) => selector === "[data-meritos-descargar]"
      ? [{ dataset: { meritosDescargar: "ref:original" }, focus() { foco += 1; } }] : [];
    pulsar({ meritosDescargar: "ref:original" });
    assert.doesNotMatch(contenedor.innerHTML, /El comprobante no está disponible ahora/u);
    assert.equal(foco, 1);
  } finally { vista.desmontar(); }
});

test("muestra recibo y análisis, consulta la misma carga y declara solo tras admisión", async () => {
  const llamadas = [];
  let consultasEstado = 0;
  let cargas = 0;
  const fetchImpl = async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    if (ruta.includes("/comprobantes/estado?")) {
      consultasEstado += 1;
      return { status: 200, ok: true, async json() { return { data: consultasEstado === 1
        ? { carga_id: "ref:carga1", estado: "analizando", recibo_ref: "ref:recibo1" }
        : { carga_id: "ref:carga1", estado: "admitido", recibo_ref: "ref:recibo1",
          comprobante_ref: "ref:pdf1", registrado_en: "2026-10-14T12:00:00Z" } }; } };
    }
    if (ruta.endsWith("/comprobantes")) {
      cargas += 1;
      return { status: 202, ok: true, async json() { return { data: {
        carga_id: "ref:carga1", estado: cargas === 1 ? "recibido" : "analizando",
        recibo_ref: "ref:recibo1",
      } }; } };
    }
    if (ruta.endsWith("/declaraciones")) {
      const orden = JSON.parse(opciones.body);
      return { status: 200, ok: true, async json() { return { data: { referencia: "ref:merito1",
        auditoria_ref: "aud:1", accion: "meritos.hecho.declarar", clave_idempotencia: orden.clave_idempotencia,
        registrado_en: "2026-10-14T12:00:00Z", registro: { hecho: { estado: "declarado", version: 1,
          denominacion: orden.denominacion } } } }; } };
    }
    return { status: 200, ok: true, async json() { return pagina([]); } };
  };
  const { vista, pulsar, editar, enviarAlta, contenedor } = montajeInteractivo(fetchImpl);
  let focoAccion = 0;
  contenedor.querySelector = (selector) => selector === "#meritos-alta-accion"
    ? { focus() { focoAccion += 1; } } : null;
  const archivo = new Blob(["%PDF-1.7"], { type: "application/pdf" }); archivo.name = "titulo.pdf";
  try {
    await vista.preparada;
    pulsar({}, ["data-meritos-alta"]); editar("denominacion", "Título");
    assert.match(contenedor.innerHTML, /id="meritos-alta-titulo" tabindex="-1"/u);
    editar("fecha_desde", "2026-01-01"); editar("documento", "", [archivo]);
    enviarAlta(); await new Promise(setImmediate);
    assert.match(contenedor.innerHTML, /Hemos recibido el PDF/u);
    assert.match(contenedor.innerHTML, /Recibo ref:recibo1/u);
    assert.equal(llamadas.filter((p) => p.ruta.endsWith("/declaraciones")).length, 0);
    const clave = vista.pendiente().orden.clave_idempotencia;
    enviarAlta(); await new Promise(setImmediate);
    assert.match(contenedor.innerHTML, /Estamos comprobando el PDF/u);
    assert.ok(focoAccion >= 2, "el control para continuar recupera el foco");
    assert.equal(llamadas.filter((p) => p.ruta.endsWith("/declaraciones")).length, 0);
    enviarAlta(); await new Promise(setImmediate);
    const subidas = llamadas.filter((p) => p.ruta.endsWith("/comprobantes"));
    assert.equal(subidas.length, 2, "una acción expresa reanuda el análisis pendiente");
    assert.equal(subidas[0].opciones.body.get("clave_idempotencia"), clave);
    assert.equal(subidas[1].opciones.body.get("clave_idempotencia"), clave);
    assert.equal(await subidas[1].opciones.body.get("documento").text(), "%PDF-1.7");
    assert.equal(llamadas.filter((p) => p.ruta.includes("/comprobantes/estado?")).length, 2);
    const declaracion = JSON.parse(llamadas.find((p) => p.ruta.endsWith("/declaraciones")).opciones.body);
    assert.equal(declaracion.clave_idempotencia, clave);
    assert.equal(declaracion.comprobante_ref, "ref:pdf1");
    assert.equal(vista.pendiente(), null);
  } finally { vista.desmontar(); }
});

test("el rechazo conserva el recibo, oculta el código y exige cambiar el PDF", async () => {
  const llamadas = [];
  const fetchImpl = async (ruta, opciones) => {
    llamadas.push(ruta);
    return ruta.endsWith("/comprobantes")
      ? { status: 422, ok: false, async json() { return { data: { carga_id: "ref:carga1",
        estado: "rechazado", recibo_ref: "ref:recibo1", codigo_rechazo: "pdf_no_valido" } }; } }
      : { status: 200, ok: true, async json() { return pagina([]); } };
  };
  const { vista, pulsar, editar, enviarAlta, contenedor } = montajeInteractivo(fetchImpl);
  const archivo = new Blob(["%PDF-1.7"], { type: "application/pdf" }); archivo.name = "titulo.pdf";
  try {
    await vista.preparada;
    pulsar({}, ["data-meritos-alta"]); editar("denominacion", "Título");
    editar("fecha_desde", "2026-01-01"); editar("documento", "", [archivo]);
    enviarAlta(); await new Promise(setImmediate);
    assert.match(contenedor.innerHTML, /No se ha podido admitir este PDF/u);
    assert.match(contenedor.innerHTML, /no tiene un formato PDF válido/u);
    assert.match(contenedor.innerHTML, /Recibo ref:recibo1/u);
    assert.doesNotMatch(contenedor.innerHTML, /pdf_no_valido/u);
    assert.equal(vista.pendiente(), null);
    enviarAlta(); await new Promise(setImmediate);
    assert.equal(llamadas.filter((ruta) => ruta.endsWith("/comprobantes")).length, 1);
    assert.equal(llamadas.filter((ruta) => ruta.endsWith("/declaraciones")).length, 0);
  } finally { vista.desmontar(); }
});

test("una consulta incierta impide reenviar el PDF por segunda vez", async () => {
  const rutas = [];
  const fetchImpl = async (ruta) => {
    rutas.push(ruta);
    if (ruta.includes("/comprobantes/estado?")) return { status: 503, ok: false, async json() {
      return { error: { codigo: "meritos.error.comprobante_no_disponible" } };
    } };
    if (ruta.endsWith("/comprobantes")) return { status: 503, ok: false, async json() {
      return { error: { codigo: "meritos.error.comprobante_pendiente" } };
    } };
    return { status: 200, ok: true, async json() { return pagina([]); } };
  };
  const { vista, pulsar, editar, enviarAlta, contenedor } = montajeInteractivo(fetchImpl);
  const archivo = new Blob(["%PDF-1.7"], { type: "application/pdf" }); archivo.name = "titulo.pdf";
  try {
    await vista.preparada;
    pulsar({}, ["data-meritos-alta"]); editar("denominacion", "Título");
    editar("fecha_desde", "2026-01-01"); editar("documento", "", [archivo]);
    enviarAlta(); await new Promise(setImmediate);
    const clave = vista.pendiente().orden.clave_idempotencia;
    enviarAlta(); await new Promise(setImmediate);
    assert.equal(rutas.filter((ruta) => ruta.endsWith("/comprobantes")).length, 1);
    assert.ok(rutas.find((ruta) => ruta.includes("/comprobantes/estado?")).includes(encodeURIComponent(clave)));
    assert.equal(vista.pendiente().alta.archivo, archivo);
    assert.match(contenedor.innerHTML, /No se pudo consultar el estado del PDF/u);
    assert.equal(rutas.filter((ruta) => ruta.endsWith("/declaraciones")).length, 0);
  } finally { vista.desmontar(); }
});
