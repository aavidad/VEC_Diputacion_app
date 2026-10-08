import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { test } from "node:test";
import { aplicarPreferenciasInicialesAplazadas, iniciarAreaPersonal } from "./aplicacion.js";
import { iniciarI18nAreaPersonal } from "./i18n.js";
import { crearClienteHTTPAreaPersonal } from "./cliente-http.js";
import { cargarPreferenciasIniciales } from "./cliente-http.js";
import { peticionesEnSerie } from "../comun/imagen-propia.js";
import { catalogoPlano, lectorCatalogos } from "./textos-prueba.test-helper.mjs";

const valores = { idioma: "navegador", tamano_texto: "normal", alto_contraste: false,
  tema: "sistema", inicio: "bolsas", filas: 20, aviso_correo_tareas: false, aviso_correo_plazos: false };
const preferencias = { catalogo: { version_ref: "usuarios-preferencias-v1", idiomas: [], tamanos_texto: [],
  temas: [], inicios: [], filas: [20], predeterminados: valores },
estado: { persona_ref: "persona:autorizada", version: 0, catalogo_version_ref: "usuarios-preferencias-v1", valores } };

function documentoFalso() {
  const eventos = new Map();
  const elementos = new Map();
  const documento = { title: "", activeElement: null, body: { dataset: { menuAbierto: "false" } },
    documentElement: { lang: "es" },
    addEventListener(tipo, fn) { eventos.set(tipo, fn); },
    querySelectorAll() { return []; },
    querySelector(selector) {
      if (selector === '[data-accion="ver-sesion"]') return this.getElementById("boton-sesion");
      return null;
    },
    getElementById(id) {
      if (id === "enlace-inicio-institucional") return null;
      if (!elementos.has(id)) elementos.set(id, { id, dataset: {}, hidden: id === "menu-identidad",
        textContent: "", innerHTML: "", className: "", setAttribute() {}, removeAttribute() {},
        addEventListener() {}, querySelector() { return { focus() {} }; }, replaceChildren() { this.innerHTML = ""; },
        focus() { documento.activeElement = this; } });
      return elementos.get(id);
    },
  };
  return { documento, eventos };
}

function pulsar(eventos, selector, valor) {
  const control = { dataset: selector === "[data-ruta]" ? { ruta: valor } : { accion: valor } };
  eventos.get("click")({ preventDefault() {}, target: { closest(patron) {
    return patron === selector ? control : null;
  } } });
}

async function escenario(idioma, lectura, errorPreferencias = null) {
  const original = { document: globalThis.document, window: globalThis.window };
  const { documento, eventos } = documentoFalso();
  const ventana = { location: { search: "", pathname: "/area-personal/", origin: "https://vec.example" },
    history: { pushState() {} }, addEventListener() {}, scrollTo() {} };
  globalThis.document = documento;
  globalThis.window = ventana;
  try {
    await iniciarI18nAreaPersonal(documento, { leer: lectorCatalogos(), preferidos: [idioma],
      ubicacion: { href: `https://vec.example/area-personal/?lang=${idioma}` } });
    let consultasBolsa = 0;
    const cliente = { async cargar() { consultasBolsa += 1; throw { codigo: "servicio_no_disponible" }; } };
    const vistasDisponibles = new Set(["inicio", "llamamientos", "preferencias"]);
    const estado = await iniciarAreaPersonal({ cliente, vistasDisponibles, preferencias: lectura, errorPreferencias });
    assert.equal(consultasBolsa, 1);
    assert.equal(estado.datos, null);
    pulsar(eventos, "[data-accion]", "ver-sesion");
    assert.equal(documento.getElementById("menu-identidad").hidden, false);
    pulsar(eventos, "[data-accion]", "abrir-preferencias");
    assert.equal(documento.getElementById("menu-identidad").hidden, true);
    const vistaPreferencias = documento.getElementById("espacio-trabajo").innerHTML;
    assert.equal(estado.soloPreferencias, true);
    assert.equal(estado.datos.meta.origen, "GET /api/vec/usuarios/area-personal/mis-preferencias");
    const metodo = estado.datos.sesion.metodo;
    pulsar(eventos, "[data-ruta]", "llamamientos");
    await new Promise((resolve) => setImmediate(resolve));
    assert.equal(consultasBolsa, 2);
    const vueltaBolsa = documento.getElementById("espacio-trabajo").innerHTML;
    assert.equal(estado.datos, null);
    assert.doesNotMatch(vueltaBolsa, /formulario-preferencias/u);
    pulsar(eventos, "[data-accion]", "ver-sesion");
    pulsar(eventos, "[data-accion]", "abrir-preferencias");
    const reabierta = documento.getElementById("espacio-trabajo").innerHTML;
    return { vistaPreferencias, vueltaBolsa, reabierta, metodo };
  } finally {
    globalThis.document = original.document;
    globalThis.window = original.window;
  }
}

test("GET preferencias 200 y Mi Bolsa 503 permite abrir el formulario y volver al error real, ES/EN", async () => {
  for (const [idioma, titulo] of [["es", "Mis preferencias"], ["en", "My preferences"]]) {
    const resultado = await escenario(idioma, preferencias);
    assert.match(resultado.vistaPreferencias, /id="formulario-preferencias"/u);
    assert.ok(resultado.vistaPreferencias.includes(titulo));
    assert.match(resultado.vueltaBolsa, /estado-error/u);
    assert.match(resultado.reabierta, /id="formulario-preferencias"/u);
  }
});

test("GET preferencias 403 conserva su denegación aunque Mi Bolsa falle, ES/EN", async () => {
  for (const [idioma, mensaje] of [["es", "No dispone de permiso"], ["en", "You do not have permission"]]) {
    const resultado = await escenario(idioma, null, { codigo: "denegado" });
    assert.doesNotMatch(resultado.vistaPreferencias, /id="formulario-preferencias"/u);
    assert.ok(resultado.vistaPreferencias.includes(mensaje));
    assert.doesNotMatch(resultado.reabierta, /id="formulario-preferencias"/u);
    assert.match(resultado.metodo, /no confirmada|not confirmed/u);
  }
});

test("la preferencia tardía conserva la navegación, datos, capacidades y foco", async () => {
  const documentoAnterior = globalThis.document;
  const { documento } = documentoFalso();
  globalThis.document = documento;
  try {
    const datos = { capacidades: { consultar: true } };
    const foco = documento.getElementById("foco-actual");
    documento.activeElement = foco;
    const aplicadas = [];
    const estado = { vista: "perfil", datos, preferencias: { estado: null, catalogo: null, error: null },
      filasPreferidas: 20, inicioTardio: true, navegacionVersion: 2, interaccionVersion: 1,
      ajusteVisualVersion: 0, controladorVisual: { aplicarPreferenciasServidor: (valor) => aplicadas.push(valor) } };
    const lectura = { ...preferencias, estado: { ...preferencias.estado,
      valores: { ...valores, inicio: "cuadro", filas: 50, tema: "oscuro" } } };
    aplicarPreferenciasInicialesAplazadas(estado, lectura);
    assert.equal(estado.vista, "perfil");
    assert.equal(estado.datos, datos);
    assert.equal(estado.datos.capacidades.consultar, true);
    assert.equal(documento.activeElement, foco);
    assert.equal(estado.filasPreferidas, 50);
    assert.deepEqual(aplicadas, [lectura.estado.valores]);
    assert.equal(estado.preferencias.estado, lectura.estado);
  } finally { globalThis.document = documentoAnterior; }
});

test("un atajo usado durante la carga prevalece sobre el tema tardío y un error no cambia datos ni rol", () => {
  const documentoAnterior = globalThis.document;
  const { documento } = documentoFalso();
  globalThis.document = documento;
  try {
    const datos = { capacidades: { consultar: true } };
    const estado = { vista: "perfil", datos, preferencias: { estado: null, catalogo: null, error: null },
      filasPreferidas: 20, inicioTardio: true, navegacionVersion: 1, interaccionVersion: 1,
      ajusteVisualVersion: 1, controladorVisual: { aplicarPreferenciasServidor() { assert.fail("no pisar atajo"); } } };
    aplicarPreferenciasInicialesAplazadas(estado, preferencias);
    for (const codigo of ["autenticacion", "denegado", "servicio"]) {
      aplicarPreferenciasInicialesAplazadas(estado, null, { codigo });
      assert.equal(estado.preferencias.error.codigo, codigo);
      assert.equal(estado.datos, datos);
      assert.equal(estado.datos.capacidades.consultar, true);
    }
  } finally { globalThis.document = documentoAnterior; }
});

test("con respuesta de preferencias a 250 ms, la carga de Mi Bolsa ya puede terminar", async () => {
  const original = { document: globalThis.document, window: globalThis.window };
  const { documento } = documentoFalso();
  globalThis.document = documento;
  globalThis.window = { location: { search: "?lang=en", pathname: "/area-personal/", origin: "https://vec.example" },
    history: { pushState() {} }, addEventListener() {}, scrollTo() {} };
  try {
    await iniciarI18nAreaPersonal(documento, { leer: lectorCatalogos(),
      ubicacion: { href: "https://vec.example/area-personal/?lang=en" } });
    let consultasBolsa = 0;
    const estado = await iniciarAreaPersonal({
      cliente: { async cargar() { consultasBolsa += 1; throw { codigo: "servicio_no_disponible" }; } },
      vistasDisponibles: new Set(["inicio", "llamamientos", "preferencias"]),
    });
    const lectura = new Promise((resolve) => setTimeout(() => resolve(preferencias), 250));
    assert.equal(consultasBolsa, 1);
    assert.equal(estado.preferencias.estado, null);
    assert.match(documento.getElementById("espacio-trabajo").innerHTML, /estado-error/u);
    aplicarPreferenciasInicialesAplazadas(estado, await lectura);
    assert.equal(estado.preferencias.estado, preferencias.estado);
    assert.equal(consultasBolsa, 1);
  } finally {
    globalThis.document = original.document;
    globalThis.window = original.window;
  }
});

test("un consumidor Bolsa conserva una respuesta válida mayor de 64 KiB fuera de la cola de Usuarios", async () => {
  const original = { document: globalThis.document, window: globalThis.window };
  const { documento } = documentoFalso();
  globalThis.document = documento;
  globalThis.window = { location: { search: "?vista=preferencias&lang=en", pathname: "/area-personal/", origin: "https://vec.example" },
    history: { pushState() {} }, addEventListener() {}, scrollTo() {} };
  try {
    await iniciarI18nAreaPersonal(documento, { leer: lectorCatalogos(),
      ubicacion: { href: "https://vec.example/area-personal/?vista=preferencias&lang=en" } });
    const muestra = JSON.parse(await readFile(new URL("../../../internal/modules/bolsa/adapters/httppersonal/testdata/mi_bolsa_situacion.json", import.meta.url), "utf8"));
    muestra.data.participaciones = Array.from({ length: 200 }, (_, numero) => ({
      ...muestra.data.participaciones[0], bolsa: `bolsa:${numero + 1}`, categoria: `Auxiliar ${"X".repeat(190)}`,
    }));
    const cuerpo = JSON.stringify(muestra);
    assert.ok(new TextEncoder().encode(cuerpo).byteLength > 65536);
    const rutas = [];
    const fetchNormal = async (ruta) => {
      rutas.push(ruta);
      return ruta === "/api/vec/bolsa/mi-bolsa"
        ? new Response(cuerpo, { status: 200, headers: { "Content-Type": "application/json" } })
        : new Response("", { status: 503 });
    };
    const estado = await iniciarAreaPersonal({ cliente: crearClienteHTTPAreaPersonal({ fetchImpl: fetchNormal }),
      vistasDisponibles: new Set(["preferencias", "llamamientos", "inicio"]),
      fetchImpl: fetchNormal, fetchUsuarios: peticionesEnSerie(fetchNormal) });
    const posterior = await crearClienteHTTPAreaPersonal({ fetchImpl: estado.fetchImpl }).cargar();
    assert.equal(estado.participaciones.length, 200);
    assert.equal(posterior.consulta.participaciones.length, 200);
    assert.equal(rutas.filter((ruta) => ruta === "/api/vec/bolsa/mi-bolsa").length, 2);
  } finally {
    globalThis.document = original.document;
    globalThis.window = original.window;
  }
});

test("GET tardío pinta el formulario y devuelve el foco a Recargar en el panel", async () => {
  const original = { document: globalThis.document, window: globalThis.window };
  const { documento } = documentoFalso();
  globalThis.document = documento;
  globalThis.window = { location: { search: "?vista=preferencias&lang=en", pathname: "/area-personal/", origin: "https://vec.example" },
    history: { pushState() {} }, addEventListener() {}, scrollTo() {} };
  try {
    await iniciarI18nAreaPersonal(documento, { leer: lectorCatalogos(),
      ubicacion: { href: "https://vec.example/area-personal/?vista=preferencias&lang=en" } });
    const estado = await iniciarAreaPersonal({ cliente: { async cargar() { return { consulta: {
      consultada_en: "2026-10-08T10:00:00Z", participaciones: [] } }; } },
    vistasDisponibles: new Set(["preferencias", "llamamientos", "inicio"]), preferenciasAplazadas: true });
    const contenido = documento.getElementById("espacio-trabajo");
    assert.match(contenido.innerHTML, /Loading authorised information/u);
    assert.doesNotMatch(contenido.innerHTML, /could not be retrieved/u);
    const anterior = { dataset: { accion: "recargar-preferencias" } };
    const restaurado = { focus() { documento.activeElement = this; } };
    contenido.contains = (elemento) => elemento === anterior;
    contenido.querySelector = (selector) => selector === '[data-accion="recargar-preferencias"]' ? restaurado : null;
    documento.activeElement = anterior;
    aplicarPreferenciasInicialesAplazadas(estado, preferencias);
    assert.match(contenido.innerHTML, /id="formulario-preferencias"/u);
    assert.equal(documento.activeElement, restaurado);
  } finally { globalThis.document = original.document; globalThis.window = original.window; }
});

test("Recargar antes y después de Mi Bolsa comparte la lectura diferida y no crea dos GET iniciales", async () => {
  const original = { document: globalThis.document, window: globalThis.window };
  const { documento, eventos } = documentoFalso();
  globalThis.document = documento;
  globalThis.window = { location: { search: "?vista=preferencias&lang=es", pathname: "/area-personal/", origin: "https://vec.example" },
    history: { pushState() {} }, addEventListener() {}, scrollTo() {} };
  try {
    await iniciarI18nAreaPersonal(documento, { leer: lectorCatalogos(),
      ubicacion: { href: "https://vec.example/area-personal/?vista=preferencias&lang=es" } });
    let terminarBolsa;
    const bolsa = new Promise((_resolver, rechazar) => { terminarBolsa = rechazar; });
    let consultas = 0;
    const clientePreferencias = { async cargar() { consultas += 1; return preferencias; } };
    const inicio = iniciarAreaPersonal({ cliente: { cargar: () => bolsa },
      vistasDisponibles: new Set(["inicio", "llamamientos", "preferencias"]),
      clientePreferencias, preferenciasAplazadas: true });
    pulsar(eventos, "[data-accion]", "recargar-preferencias");
    assert.equal(consultas, 0);
    terminarBolsa({ codigo: "servicio_no_disponible" });
    const estado = await inicio;
    const contenido = documento.getElementById("espacio-trabajo");
    assert.match(contenido.innerHTML, /Cargando información autorizada/u);
    assert.doesNotMatch(contenido.innerHTML, /No se pudieron consultar sus preferencias/u);
    pulsar(eventos, "[data-accion]", "recargar-preferencias");
    assert.equal(consultas, 0);
    aplicarPreferenciasInicialesAplazadas(estado, await cargarPreferenciasIniciales(clientePreferencias));
    await Promise.resolve();
    assert.equal(consultas, 1);
    assert.equal(estado.preferencias.estado, preferencias.estado);
    assert.match(contenido.innerHTML, /id="formulario-preferencias"/u);
  } finally { globalThis.document = original.document; globalThis.window = original.window; }
});

test("sin lang explícito, Recargar hace una lectura nueva tras el arranque previo", async () => {
  const original = { document: globalThis.document, window: globalThis.window };
  const { documento, eventos } = documentoFalso();
  globalThis.document = documento;
  globalThis.window = { location: { search: "?vista=preferencias", pathname: "/area-personal/", origin: "https://vec.example" },
    history: { pushState() {} }, addEventListener() {}, scrollTo() {} };
  try {
    await iniciarI18nAreaPersonal(documento, { leer: lectorCatalogos(),
      ubicacion: { href: "https://vec.example/area-personal/?vista=preferencias" } });
    let consultas = 0;
    const estado = await iniciarAreaPersonal({ cliente: { async cargar() { assert.fail("preferencias precargadas"); } },
      vistasDisponibles: new Set(["preferencias", "llamamientos", "inicio"]), preferencias,
      clientePreferencias: { async cargar() { consultas += 1; return preferencias; } } });
    assert.equal(estado.lecturaPreferenciasInicial, null);
    pulsar(eventos, "[data-accion]", "recargar-preferencias");
    await new Promise((resolve) => setImmediate(resolve));
    assert.equal(consultas, 1);
  } finally { globalThis.document = original.document; globalThis.window = original.window; }
});

test("un fallo del GET inicial permite una nueva lectura explícita desde Preferencias", async () => {
  const original = { document: globalThis.document, window: globalThis.window };
  const { documento, eventos } = documentoFalso();
  globalThis.document = documento;
  globalThis.window = { location: { search: "?vista=preferencias&lang=es", pathname: "/area-personal/", origin: "https://vec.example" },
    history: { pushState() {} }, addEventListener() {}, scrollTo() {} };
  try {
    await iniciarI18nAreaPersonal(documento, { leer: lectorCatalogos(),
      ubicacion: { href: "https://vec.example/area-personal/?vista=preferencias&lang=es" } });
    let consultas = 0;
    const estado = await iniciarAreaPersonal({ cliente: { async cargar() { throw { codigo: "servicio_no_disponible" }; } },
      vistasDisponibles: new Set(["preferencias", "llamamientos", "inicio"]), preferenciasAplazadas: true,
      clientePreferencias: { async cargar() { consultas += 1; return preferencias; } } });
    aplicarPreferenciasInicialesAplazadas(estado, null, { codigo: "servicio" });
    assert.equal(consultas, 0);
    pulsar(eventos, "[data-accion]", "recargar-preferencias");
    await new Promise((resolve) => setImmediate(resolve));
    assert.equal(consultas, 1);
    assert.equal(estado.preferencias.estado, preferencias.estado);
  } finally { globalThis.document = original.document; globalThis.window = original.window; }
});
