import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";
import vm from "node:vm";

import {
  validarRespuestaBolsasPublicas,
  validarRespuestaListaPublica,
  PATRON_DOCUMENTO_ENMASCARADO,
} from "./contrato-publico-bolsas.js";
import {
  consultarBolsasPublicas,
  consultarListaBolsaPublica,
} from "./lista-bolsas-api.js";
const directorio = dirname(fileURLToPath(import.meta.url));
const rutaRaiz = join(directorio, "../../..");
vm.runInThisContext(readFileSync(join(directorio, "i18n-publica.js"), "utf8"));
const { crearControladorListaBolsas } = await import("./lista-bolsas.js");
const datasetDemo = JSON.parse(
  readFileSync(join(rutaRaiz, "data/demo/bolsa/v1.bolsas-demo.json"), "utf8")
);

// Mapeo canónico de estados de demo al catálogo SituacionParticipacionBolsa
function mapearEstadoBolsa(estadoOrigen) {
  switch (estadoOrigen) {
    case "trabajando":
    case "pendiente_incorporacion":
      return "ocupado";
    case "disponible_desde":
      return "no_disponible";
    case "renuncia":
      return "renuncia_pendiente";
    case "disponible":
    case "no_disponible":
    case "excluido":
      return estadoOrigen;
    default:
      return "disponible";
  }
}

// Fixture canónica de bolsas públicas derivada del dataset
function generarFixtureBolsasPublicas() {
  const bolsas = (datasetDemo.bolsas || []).map((b) => ({
    bolsa_ref: b.bolsa_ref.replace(":demo:", ":sintetico:"),
    categoria: b.categoria,
    grupos: b.grupos || ["C1"],
    tipo_lista: b.tipo_lista || "ordinaria",
    vigente_desde: b.vigente_desde ? `${b.vigente_desde}T00:00:00Z` : "2025-01-01T00:00:00Z",
    vigente_hasta: b.vigente_hasta ? `${b.vigente_hasta}T00:00:00Z` : null,
    total: b.candidaturas || 0,
  }));

  return {
    data: {
      esquema: "vec.bolsa.publico.bolsas.v1",
      generado_en: "2026-09-17T20:00:00Z",
      bolsas,
    },
  };
}

// Fixture canónica de lista pública de posiciones
function generarFixtureListaPublica(bolsaRef = "bolsa:sintetico:administrativo") {
  const bolsaOrigen = (datasetDemo.bolsas || []).find((b) =>
    b.bolsa_ref.replace(":demo:", ":sintetico:") === bolsaRef
  ) || datasetDemo.bolsas[0];

  const bolsa = {
    bolsa_ref: bolsaOrigen.bolsa_ref.replace(":demo:", ":sintetico:"),
    categoria: bolsaOrigen.categoria,
    grupos: bolsaOrigen.grupos || ["C1"],
    tipo_lista: bolsaOrigen.tipo_lista || "ordinaria",
    vigente_desde: bolsaOrigen.vigente_desde ? `${bolsaOrigen.vigente_desde}T00:00:00Z` : "2025-01-01T00:00:00Z",
    vigente_hasta: bolsaOrigen.vigente_hasta ? `${bolsaOrigen.vigente_hasta}T00:00:00Z` : null,
    total: bolsaOrigen.candidaturas || 10,
  };

  const candidaturasBolsa = (datasetDemo.candidaturas || [])
    .filter((c) => c.bolsa_ref === bolsaOrigen.bolsa_ref)
    .slice(0, 15);

  const posiciones = candidaturasBolsa.map((c, idx) => ({
    orden: idx + 1,
    documento_enmascarado: c.documento_enmascarado || `***${String(idx + 1000).slice(0, 4)}**`,
    estado_clave: mapearEstadoBolsa(c.estado_clave),
  }));

  return {
    data: {
      esquema: "vec.bolsa.publico.lista.v1",
      generado_en: "2026-09-17T20:00:00Z",
      bolsa,
      posiciones,
      hay_mas: true,
      cursor_siguiente: "cursor:posicion:16",
    },
  };
}

test("contrato público: valida correctamente la respuesta de bolsas públicas canónicas", () => {
  const fixture = generarFixtureBolsasPublicas();
  const validado = validarRespuestaBolsasPublicas(fixture);

  assert.equal(validado.esquema, "vec.bolsa.publico.bolsas.v1");
  assert.ok(validado.bolsas.length > 0);
  const primera = validado.bolsas[0];
  assert.ok(primera.bolsa_ref);
  assert.ok(primera.categoria);
  assert.ok(Array.isArray(primera.grupos));
  assert.ok(primera.total >= 0);
});

test("contrato público: rechaza bolsas con esquema erróneo o propiedades extra no permitidas", () => {
  const fixture = generarFixtureBolsasPublicas();

  // Esquema incorrecto
  assert.throws(() => {
    validarRespuestaBolsasPublicas({
      data: { ...fixture.data, esquema: "vec.bolsa.rrhh.bolsas.v1" },
    });
  }, /Esquema inválido/);

  // Propiedad no autorizada
  const bolsaConExtra = { ...fixture.data.bolsas[0], administracion_interna: true };
  assert.throws(() => {
    validarRespuestaBolsasPublicas({
      data: { ...fixture.data, bolsas: [bolsaConExtra] },
    });
  }, /Propiedad no permitida/);
});

test("contrato público: valida la lista de aspirantes y verifica que solo contenga orden, documento y estado", () => {
  const fixture = generarFixtureListaPublica();
  const validado = validarRespuestaListaPublica(fixture);

  assert.equal(validado.esquema, "vec.bolsa.publico.lista.v1");
  assert.ok(validado.posiciones.length > 0);

  for (const pos of validado.posiciones) {
    assert.equal(typeof pos.orden, "number");
    assert.match(pos.documento_enmascarado, PATRON_DOCUMENTO_ENMASCARADO);
    assert.ok(
      ["disponible", "ocupado", "no_disponible", "excluido", "renuncia_pendiente"].includes(
        pos.estado_clave
      )
    );
    assert.deepEqual(Object.keys(pos).sort(), [
      "documento_enmascarado",
      "estado_clave",
      "orden",
    ]);
  }
});

test("anti-fugas: el contrato de lista rechaza estrictamente nombres, emails, teléfonos o DNI sin enmascarar", () => {
  const fixture = generarFixtureListaPublica();

  // Fuga 1: inclusión de nombre visible
  const posicionConNombre = {
    ...fixture.data.posiciones[0],
    nombre_visible: "Persona Pública",
  };
  assert.throws(() => {
    validarRespuestaListaPublica({
      data: { ...fixture.data, posiciones: [posicionConNombre] },
    });
  }, /Propiedad no permitida.*nombre_visible/);

  // Fuga 2: DNI real sin enmascarar
  const posicionConDNIReal = {
    ...fixture.data.posiciones[0],
    documento_enmascarado: "12345678Z",
  };
  assert.throws(() => {
    validarRespuestaListaPublica({
      data: { ...fixture.data, posiciones: [posicionConDNIReal] },
    });
  }, /documento_enmascarado inválido/);

  // Fuga 3: Situación ajena al catálogo
  const posicionEstadoInvalido = {
    ...fixture.data.posiciones[0],
    estado_clave: "activo_sin_comprobar",
  };
  assert.throws(() => {
    validarRespuestaListaPublica({
      data: { ...fixture.data, posiciones: [posicionEstadoInvalido] },
    });
  }, /estado_clave inválido/);
});

test("cliente HTTP público: utiliza exclusivamente credentials: 'omit' y no contiene rutas internas", async () => {
  const codigoCliente = readFileSync(join(directorio, "lista-bolsas-api.js"), "utf8");
  const codigoControlador = readFileSync(join(directorio, "lista-bolsas.js"), "utf8");
  const codigoHTML = readFileSync(join(directorio, "listas.html"), "utf8");

  // Regla DEC-053: credentials: "omit"
  assert.match(codigoCliente, /credentials:\s*"omit"/);
  assert.doesNotMatch(codigoCliente, /credentials:\s*"(?:include|same-origin)"/);

  // Sin almacenamiento de sesión ni cookies
  const todoElCodigo = `${codigoCliente}\n${codigoControlador}\n${codigoHTML}`;
  assert.doesNotMatch(todoElCodigo, /document\.cookie/);
  assert.doesNotMatch(todoElCodigo, /localStorage/);
  assert.doesNotMatch(todoElCodigo, /sessionStorage/);

  // Sin referencias a rutas autenticadas o internas
  assert.doesNotMatch(todoElCodigo, /\/api\/vec\b/);
  assert.doesNotMatch(`${codigoCliente}\n${codigoControlador}`, /\/portal-empleado\b/);
  assert.match(codigoHTML, /<link rel="stylesheet" href="\/portal-empleado\/portal\.css\?v=20260924-f2-shell-v1">/);
  assert.doesNotMatch(codigoHTML, /<(?:a|script)\b[^>]*(?:href|src)="\/portal-empleado\b/);
  assert.doesNotMatch(todoElCodigo, /\/area-personal\b/);
});

test("cliente HTTP público: invoca endpoints canónicos y propaga parámetros seguros", async () => {
  const peticionesRealizadas = [];
  const fakeFetch = async (url, opts) => {
    peticionesRealizadas.push({ url, opts });
    if (url.includes("/lista")) {
      return {
        ok: true,
        status: 200,
        json: async () => generarFixtureListaPublica(),
      };
    }
    return {
      ok: true,
      status: 200,
      json: async () => generarFixtureBolsasPublicas(),
    };
  };

  // Consulta de bolsas
  const respBolsas = await consultarBolsasPublicas({ fetchImpl: fakeFetch });
  assert.ok(respBolsas.bolsas.length > 0);
  assert.equal(peticionesRealizadas[0].url, "/api/publico/bolsa/bolsas");
  assert.equal(peticionesRealizadas[0].opts.credentials, "omit");

  // Consulta de lista con documento enmascarado válido
  const respLista = await consultarListaBolsaPublica({
    bolsa_ref: "bolsa:sintetico:administrativo",
    documento: "***1234**",
    fetchImpl: fakeFetch,
  });
  assert.ok(respLista.posiciones.length > 0);
  const urlLista = peticionesRealizadas[1].url;
  assert.ok(urlLista.includes("/api/publico/bolsa/bolsas/bolsa:sintetico:administrativo/lista"));
  assert.ok(urlLista.includes("documento=***1234**") || urlLista.includes("documento=%2A%2A%2A1234%2A%2A"));

  // Consulta con documento inválido (no enmascarado): no se propaga el parámetro
  await consultarListaBolsaPublica({
    bolsa_ref: "bolsa:sintetico:administrativo",
    documento: "12345678Z",
    fetchImpl: fakeFetch,
  });
  const urlSinParam = peticionesRealizadas[2].url;
  assert.ok(!urlSinParam.includes("documento="));
});

test("controlador público: ciclo de vida de renderizado de bolsas y selección de lista", async () => {
  const fixtureBolsas = generarFixtureBolsasPublicas();
  const fixtureLista = generarFixtureListaPublica();

  // Simulación ligera del DOM
  const elementos = {
    seccionBolsas: { hidden: false },
    seccionLista: { hidden: true },
    bolsasCargando: { hidden: true },
    bolsasError: { hidden: true },
    bolsasVacio: { hidden: true },
    cuerpoTablaBolsas: { innerHTML: "" },
    infoBolsaActiva: { innerHTML: "" },
    cuerpoTablaLista: { innerHTML: "" },
    listaCargando: { hidden: true },
    listaError: { hidden: true },
    listaVacio: { hidden: true },
    contenedorPaginacion: { hidden: true },
    botonSiguiente: { dataset: {} },
    errorDocumento: { hidden: true, textContent: "" },
    inputDocumento: { value: "" },
  };

  const apiMock = {
    consultarBolsasPublicas: async () => fixtureBolsas.data,
    consultarListaBolsaPublica: async () => fixtureLista.data,
  };

  const ctrl = crearControladorListaBolsas({ elementos, api: apiMock });

  // Carga inicial
  await ctrl.cargarBolsas();
  assert.equal(elementos.seccionBolsas.hidden, false);
  assert.equal(elementos.seccionLista.hidden, true);
  assert.ok(elementos.cuerpoTablaBolsas.innerHTML.includes("ADMINISTRATIVO"));
  assert.ok(elementos.cuerpoTablaBolsas.innerHTML.includes("Consultar lista"));

  // Selección de una bolsa
  await ctrl.seleccionarBolsa("bolsa:sintetico:administrativo");
  assert.equal(elementos.seccionBolsas.hidden, true);
  assert.equal(elementos.seccionLista.hidden, false);
  assert.ok(elementos.infoBolsaActiva.innerHTML.includes("ADMINISTRATIVO"));
  assert.ok(elementos.cuerpoTablaLista.innerHTML.includes("***"));
  assert.ok(elementos.cuerpoTablaLista.innerHTML.includes("estado-chip-publico"));

  // Retorno a bolsas
  ctrl.volverABolsas();
  assert.equal(elementos.seccionBolsas.hidden, false);
  assert.equal(elementos.seccionLista.hidden, true);
});

test("controlador público: gestiona adecuadamente estados de error y vacío", async () => {
  const elementos = {
    seccionBolsas: { hidden: false },
    seccionLista: { hidden: true },
    bolsasCargando: { hidden: true },
    bolsasError: { hidden: true },
    bolsasVacio: { hidden: true },
    mensajeErrorBolsas: { textContent: "" },
    cuerpoTablaBolsas: { innerHTML: "" },
    infoBolsaActiva: { innerHTML: "" },
    cuerpoTablaLista: { innerHTML: "" },
    listaCargando: { hidden: true },
    listaError: { hidden: true },
    listaVacio: { hidden: true },
    mensajeErrorLista: { textContent: "" },
    contenedorPaginacion: { hidden: true },
    errorDocumento: { hidden: true },
    inputDocumento: { value: "" },
  };

  // Simulación de error en bolsas
  const apiError = {
    consultarBolsasPublicas: async () => {
      throw new Error("Fallo de conexión 503");
    },
    consultarListaBolsaPublica: async () => {
      throw new Error("Bolsa no encontrada 404");
    },
  };

  const ctrlError = crearControladorListaBolsas({ elementos, api: apiError });
  await ctrlError.cargarBolsas();
  assert.equal(elementos.bolsasError.hidden, false);
  assert.equal(elementos.mensajeErrorBolsas.textContent, "Error al consultar bolsas");

  // Simulación de lista vacía
  const apiVacia = {
    consultarBolsasPublicas: async () => ({ bolsas: [] }),
    consultarListaBolsaPublica: async () => ({
      bolsa: { bolsa_ref: "bolsa:sintetico:vacia", categoria: "PRUEBA", grupos: [], tipo_lista: "ordinaria", total: 0 },
      posiciones: [],
      hay_mas: false,
      cursor_siguiente: null,
    }),
  };

  const ctrlVacio = crearControladorListaBolsas({ elementos, api: apiVacia });
  await ctrlVacio.cargarBolsas();
  assert.equal(elementos.bolsasVacio.hidden, false);

  await ctrlVacio.seleccionarBolsa("bolsa:sintetico:vacia");
  assert.equal(elementos.listaVacio.hidden, false);
});

test("interfaz pública: listas.html es accesible, semántica y sin textos demo visibles", () => {
  const html = readFileSync(join(directorio, "listas.html"), "utf8");
  const css = readFileSync(join(directorio, "listas.css"), "utf8");

  // Estructura semántica
  assert.match(html, /<main\b[^>]*\bid="contenido-principal"/);
  assert.match(html, /<header\b[^>]*\bclass="cabecera-publica"/);
  assert.match(html, /<table\b[^>]*\bclass="tabla-listas-publica"/);
  assert.match(html, /<caption>/);
  assert.match(html, /<th scope="col">/);

  // Enlace de salto
  assert.match(html, /<a class="salto-contenido" href="#contenido-principal">/);

  // Protección anti desbordamiento a 390 px
  assert.match(css, /overflow-x:\s*auto/);
  assert.match(css, /@media\s*\(max-width:\s*480px\)/);

  // No contiene la palabra "demo" en textos visibles de usuario
  // (permitido solo en nombres de archivos o atributos internos si existieran, pero en texto de interfaz debe ser sintético/referencia)
  const textoSinEtiquetas = html.replace(/<[^>]+>/g, " ");
  assert.doesNotMatch(textoSinEtiquetas, /\bdemo\b/i);
});

test("controlador público: restaura lista y filtro con Atrás/Adelante", async () => {
  const fixtureBolsas = generarFixtureBolsasPublicas();
  const consultas = [];
  const elementos = {
    seccionBolsas: { hidden: false, addEventListener() {} },
    seccionLista: { hidden: true },
    bolsasCargando: { hidden: true },
    bolsasError: { hidden: true },
    bolsasVacio: { hidden: true },
    cuerpoTablaBolsas: { innerHTML: "" },
    infoBolsaActiva: { innerHTML: "" },
    cuerpoTablaLista: { innerHTML: "" },
    listaCargando: { hidden: true },
    listaError: { hidden: true },
    listaVacio: { hidden: true },
    contenedorPaginacion: { hidden: true },
    botonSiguiente: { dataset: {}, addEventListener() {} },
    errorDocumento: { hidden: true, textContent: "" },
    inputDocumento: { value: "" },
  };
  const listeners = new Map();
  let href = "https://vec.test/bolsa/listas.html";
  const location = {
    get href() { return href; },
    set href(url) { href = new URL(url, href).toString(); },
    get search() { return new URL(href).search; },
  };
  const ventana = {
    location,
    history: {
      pushState(_estado, _titulo, url) { location.href = url; },
      replaceState(_estado, _titulo, url) { location.href = url; },
    },
    addEventListener(tipo, receptor) { listeners.set(tipo, receptor); },
  };
  const api = {
    consultarBolsasPublicas: async () => fixtureBolsas.data,
    consultarListaBolsaPublica: async (consulta) => {
      consultas.push(consulta);
      return generarFixtureListaPublica(consulta.bolsa_ref).data;
    },
  };
  const ctrl = crearControladorListaBolsas({ elementos, api, ventana });
  ctrl.instalar();
  await new Promise((resolver) => setImmediate(resolver));

  await ctrl.seleccionarBolsa("bolsa:sintetico:administrativo", "***1234**");
  assert.match(ventana.location.href, /bolsa=bolsa%3Asintetico%3Aadministrativo/);

  ventana.location.href = "https://vec.test/bolsa/listas.html?bolsa=bolsa%3Asintetico%3Aadministrativo&documento=***1234**";
  listeners.get("popstate")();
  await new Promise((resolver) => setImmediate(resolver));

  assert.equal(elementos.inputDocumento.value, "***1234**");
  assert.deepEqual(consultas.at(-1), {
    bolsa_ref: "bolsa:sintetico:administrativo",
    documento: "***1234**",
    cursor: "",
  });

  ventana.location.href = "https://vec.test/bolsa/listas.html?bolsa=bolsa%3Asintetico%3Aadministrativo&documento=12345678Z";
  listeners.get("popstate")();
  await new Promise((resolver) => setImmediate(resolver));
  assert.equal(elementos.inputDocumento.value, "");
  assert.equal(consultas.at(-1).documento, "");
  assert.doesNotMatch(ventana.location.href, /12345678Z/);

  ventana.location.href = "https://vec.test/bolsa/listas.html";
  listeners.get("popstate")();
  await new Promise((resolver) => setImmediate(resolver));

  assert.equal(elementos.seccionBolsas.hidden, false);
  assert.equal(elementos.seccionLista.hidden, true);
});

test("error de lista pública redacta datos del servidor y permite reintentar un enlace directo", async () => {
  let reintentar;
  let intentos = 0;
  const elementos = {
    seccionBolsas: { hidden: false, addEventListener() {} },
    seccionLista: { hidden: true },
    bolsasCargando: { hidden: true }, bolsasError: { hidden: true }, bolsasVacio: { hidden: true },
    cuerpoTablaBolsas: { innerHTML: "" },
    listaCargando: { hidden: true }, listaError: { hidden: true }, listaVacio: { hidden: true },
    mensajeErrorLista: { textContent: "" }, infoBolsaActiva: { innerHTML: "" },
    cuerpoTablaLista: { innerHTML: "" }, contenedorPaginacion: { hidden: true },
    botonSiguiente: { dataset: {}, disabled: false, addEventListener() {} },
    botonReintentarLista: { addEventListener(_tipo, fn) { reintentar = fn; } },
  };
  let href = "https://vec.test/bolsa/listas.html?bolsa=bolsa:sintetico:administrativo&documento=12345678Z";
  const ventana = {
    location: { get href() { return href; }, get search() { return new URL(href).search; } },
    history: { replaceState(_estado, _titulo, url) { href = url; } }, addEventListener() {},
  };
  const api = {
    consultarBolsasPublicas: async () => ({ bolsas: [] }),
    consultarListaBolsaPublica: async () => {
      if (++intentos === 1) throw Object.assign(new Error("DNI 12345678Z no disponible"), { status: 404 });
      return generarFixtureListaPublica().data;
    },
  };
  const ctrl = crearControladorListaBolsas({ elementos, api, ventana });
  ctrl.instalar();
  await new Promise((resolver) => setImmediate(resolver));
  assert.equal(elementos.mensajeErrorLista.textContent, "La bolsa solicitada no está disponible para consulta pública.");
  assert.doesNotMatch(elementos.mensajeErrorLista.textContent, /12345678Z/);
  assert.doesNotMatch(ventana.location.href, /12345678Z/);
  assert.equal(ctrl.estado.bolsaRefSolicitada, "bolsa:sintetico:administrativo");
  reintentar();
  await new Promise((resolver) => setImmediate(resolver));
  assert.equal(elementos.listaError.hidden, true);
  assert.match(elementos.cuerpoTablaLista.innerHTML, /\*\*\*/);
});

test("al cambiar de bolsa, un fallo no deja visible la posición ni el filtro anteriores", async () => {
  const refAnterior = "bolsa:sintetico:administrativo";
  const refNueva = "bolsa:sintetico:otra";
  const tabla = { hidden: false };
  const campos = { disabled: false };
  const elementos = {
    seccionBolsas: { hidden: false }, seccionLista: { hidden: true },
    bolsasCargando: { hidden: true }, bolsasError: { hidden: true }, bolsasVacio: { hidden: true },
    cuerpoTablaBolsas: { innerHTML: "" },
    listaCargando: { hidden: true }, listaError: { hidden: true }, listaVacio: { hidden: true },
    mensajeErrorLista: { textContent: "" },
    infoBolsaActiva: { innerHTML: "" },
    cuerpoTablaLista: { innerHTML: "", closest: () => tabla },
    formularioBusqueda: { querySelector: () => campos },
    inputDocumento: { value: "" }, errorDocumento: { hidden: true },
    contenedorPaginacion: { hidden: true }, botonSiguiente: { dataset: {}, disabled: false },
  };
  const api = {
    consultarBolsasPublicas: async () => ({ bolsas: [] }),
    consultarListaBolsaPublica: async ({ bolsa_ref }) => {
      if (bolsa_ref === refNueva) throw Object.assign(new Error("DNI 12345678Z"), { status: 404 });
      return generarFixtureListaPublica(refAnterior).data;
    },
  };
  const ctrl = crearControladorListaBolsas({ elementos, api });
  await ctrl.seleccionarBolsa(refAnterior, "***1234**");
  assert.equal(tabla.hidden, false);
  assert.ok(elementos.cuerpoTablaLista.innerHTML.includes("***"));
  await ctrl.seleccionarBolsa(refNueva, "12345678Z");
  assert.equal(ctrl.estado.bolsaSeleccionada, null);
  assert.equal(ctrl.estado.filtroDocumento, "");
  assert.equal(elementos.inputDocumento.value, "");
  assert.equal(elementos.infoBolsaActiva.innerHTML, "");
  assert.equal(elementos.cuerpoTablaLista.innerHTML, "");
  assert.equal(tabla.hidden, true);
  assert.equal(campos.disabled, true);
  assert.equal(elementos.mensajeErrorLista.textContent, "La bolsa solicitada no está disponible para consulta pública.");
});

test("un error al cargar la página siguiente conserva las posiciones y reintenta el mismo cursor", async () => {
  const bolsaRef = "bolsa:sintetico:administrativo";
  const base = generarFixtureListaPublica(bolsaRef).data;
  const cursores = [];
  let siguiente;
  let reintentar;
  const elementos = {
    seccionBolsas: { hidden: false, addEventListener() {} }, seccionLista: { hidden: true },
    bolsasCargando: { hidden: true }, bolsasError: { hidden: true }, bolsasVacio: { hidden: true },
    cuerpoTablaBolsas: { innerHTML: "" },
    listaCargando: { hidden: true }, listaError: { hidden: true }, listaVacio: { hidden: true },
    mensajeErrorLista: { textContent: "" }, infoBolsaActiva: { innerHTML: "" },
    cuerpoTablaLista: { innerHTML: "" }, contenedorPaginacion: { hidden: true },
    botonSiguiente: { dataset: {}, disabled: false, addEventListener(_tipo, fn) { siguiente = fn; } },
    botonReintentarLista: { addEventListener(_tipo, fn) { reintentar = fn; } },
  };
  const ventana = {
    location: { href: `https://vec.test/bolsa/listas.html?bolsa=${bolsaRef}`, search: `?bolsa=${bolsaRef}` },
    history: { replaceState() {} }, addEventListener() {},
  };
  const api = {
    consultarBolsasPublicas: async () => ({ bolsas: [] }),
    consultarListaBolsaPublica: async ({ cursor }) => {
      cursores.push(cursor);
      if (cursor && cursores.length === 2) throw Object.assign(new Error("Fallo temporal"), { status: 503 });
      return cursor
        ? { ...base, posiciones: base.posiciones.slice(2, 4), hay_mas: false, cursor_siguiente: null }
        : { ...base, posiciones: base.posiciones.slice(0, 2), hay_mas: true, cursor_siguiente: "cursor:pagina:2" };
    },
  };
  const ctrl = crearControladorListaBolsas({ elementos, api, ventana });
  ctrl.instalar();
  await new Promise((resolver) => setImmediate(resolver));
  assert.equal(ctrl.estado.posiciones.length, 2);
  siguiente();
  await new Promise((resolver) => setImmediate(resolver));
  assert.equal(elementos.listaError.hidden, false);
  assert.equal(ctrl.estado.posiciones.length, 2);
  assert.equal(ctrl.estado.cursorSolicitado, "cursor:pagina:2");
  reintentar();
  await new Promise((resolver) => setImmediate(resolver));
  assert.deepEqual(cursores, ["", "cursor:pagina:2", "cursor:pagina:2"]);
  assert.equal(ctrl.estado.posiciones.length, 4);
  assert.equal(elementos.listaError.hidden, true);
});

test("una respuesta de otra bolsa no se presenta como posición de la solicitada", async () => {
  const elementos = {
    seccionBolsas: { hidden: false }, seccionLista: { hidden: true },
    bolsasCargando: { hidden: true }, bolsasError: { hidden: true }, bolsasVacio: { hidden: true },
    listaCargando: { hidden: true }, listaError: { hidden: true }, listaVacio: { hidden: true },
    mensajeErrorLista: { textContent: "" }, infoBolsaActiva: { innerHTML: "" },
    cuerpoTablaLista: { innerHTML: "" }, contenedorPaginacion: { hidden: true },
  };
  const api = {
    consultarBolsasPublicas: async () => ({ bolsas: [] }),
    consultarListaBolsaPublica: async () => generarFixtureListaPublica("bolsa:sintetico:administrativo").data,
  };
  const ctrl = crearControladorListaBolsas({ elementos, api });
  await ctrl.seleccionarBolsa("bolsa:sintetico:otra");
  assert.equal(elementos.listaError.hidden, false);
  assert.equal(elementos.infoBolsaActiva.innerHTML, "");
  assert.equal(elementos.cuerpoTablaLista.innerHTML, "");
  assert.equal(ctrl.estado.bolsaSeleccionada, null);
});
