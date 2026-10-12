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
  return { vista, contenedor, pulsar, enviar };
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
