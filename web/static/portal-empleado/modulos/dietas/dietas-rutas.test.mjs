import assert from "node:assert/strict";
import test from "node:test";

import {
  ESQUEMA_CONTEXTO_ACTOR_FRONTEND,
  validarYCongelarContextoActor,
} from "../../identidad/contexto-actor.js";
import { crearAdaptadorDietasPresentacion } from "./adaptador-presentacion.js";
import { crearCalculadorRutasDietasPresentacion } from "./calculador-rutas-presentacion.js";
import {
  ATRIBUCION_OSM_INTERNA,
  CAPACIDAD_CONSULTAR_GASTO,
  CAPACIDAD_CONSULTAR_RUTA,
  CAPACIDAD_GESTIONAR_GASTO,
  CAPACIDAD_GESTIONAR_RUTA,
  CODIGO_ERROR_SERVICIO_RUTAS_DIETAS,
  PLANTILLA_TESELAS_OSM_INTERNA,
} from "./contrato.js";
import { crearTraductorDietas } from "./i18n.js";
import { crearVisorRutaDietas } from "./mapa-ruta.js";
import { crearPresentadorDietas } from "./presentador.js";
import { montarModuloDietas, renderizarDietas } from "./vista.js";

const CONTEXTO_COMPARTIDO = validarYCongelarContextoActor({
  esquema: ESQUEMA_CONTEXTO_ACTOR_FRONTEND,
  revision: 1,
  demostracion: true,
  persona_ref: "per_demo_persona_interna_dietas_000001",
  cuenta_ref: "cta_demo_cuenta_interna_dietas_000001",
  perfil_ref: "prf_demo_perfil_interno_dietas_000001",
  actor: {
    actor_ref: "DEMO-PERFIL-INTERNO-COMPARTIDO-01",
    iniciales: "AI",
    nombre_visible: "Agente interno DEMO",
  },
  rol: {
    clave: "empleado_publico",
    etiqueta: "Personal de la Diputación · escenario DEMO",
  },
  ambito: {
    clase: "personal_interno",
    organizacion_ref: "org_demo_diputacion_granada_000001",
    unidad_ref: "uni_demo_unidad_interna_dietas_000001",
    modulos: ["bolsa", "cronos", "dietas"],
  },
  autenticacion: {
    sesion_ref: "ses_demo_sesion_interna_dietas_000001",
    metodo: "demo",
    garantia: "bajo",
  },
  resuelto_en: "2026-07-19T00:00:00.000Z",
});

const CAPACIDADES_EMPLEADO = Object.freeze([
  CAPACIDAD_CONSULTAR_GASTO,
  CAPACIDAD_GESTIONAR_GASTO,
  CAPACIDAD_CONSULTAR_RUTA,
  CAPACIDAD_GESTIONAR_RUTA,
]);
const t = crearTraductorDietas();

function presentador(capacidades = CAPACIDADES_EMPLEADO) {
  return crearPresentadorDietas({
    datos: adaptador(capacidades).obtenerDatos(),
    contextoActor: CONTEXTO_COMPARTIDO,
    capacidades,
    origenComprobacion: "https://vec.demo.dipgra.es",
  });
}

function adaptador(capacidades = CAPACIDADES_EMPLEADO) {
  return crearAdaptadorDietasPresentacion({
    contextoActor: CONTEXTO_COMPARTIDO,
    capacidades,
    reloj: () => new Date("2026-07-19T10:15:00Z"),
    crearReferencia: () => "DEMO-DIE-NUEVA-999",
  });
}

function calculadorRutas(capacidades = CAPACIDADES_EMPLEADO) {
  return crearCalculadorRutasDietasPresentacion({ contextoActor: CONTEXTO_COMPARTIDO, capacidades });
}

function presentadorConRutas(capacidades = CAPACIDADES_EMPLEADO) {
  const calculador = calculadorRutas(capacidades);
  return {
    calculador,
    modulo: crearPresentadorDietas({
      datos: adaptador(capacidades).obtenerDatos(),
      contextoActor: CONTEXTO_COMPARTIDO,
      capacidades,
      catalogoRutas: calculador.obtenerCatalogo(),
      origenComprobacion: "https://vec.demo.dipgra.es",
    }),
  };
}

function crearRaizDietasMinima() {
  const escuchas = new Map();
  let contenido = "";
  return {
    escuchas,
    ownerDocument: { activeElement: null },
    get innerHTML() { return contenido; },
    set innerHTML(valor) { contenido = String(valor); },
    contains() { return false; },
    querySelector() { return null; },
    querySelectorAll() { return []; },
    addEventListener(tipo, escucha) { escuchas.set(tipo, escucha); },
    removeEventListener(tipo) { escuchas.delete(tipo); },
    replaceChildren() { contenido = ""; },
  };
}

function crearRaizDietasInteractiva() {
  const escuchas = new Map();
  const ownerDocument = { activeElement: null };
  let contenido = "";
  let controles = [];
  const convertirDataset = (atributo) => atributo.slice(5).replace(/-([a-z])/g, (_coincidencia, letra) => letra.toUpperCase());
  const raiz = {
    escuchas,
    ownerDocument,
    get innerHTML() { return contenido; },
    set innerHTML(valor) {
      contenido = String(valor);
      controles = [...contenido.matchAll(/<(?:button|select|input)\b[^>]*>/g)].flatMap(([etiqueta]) => {
        const coincidencia = etiqueta.match(/\s(data-dietas-ruta-[a-z-]+)(?:=\"([^\"]*)\")?/);
        if (!coincidencia) return [];
        const [, atributo, valor = ""] = coincidencia;
        const control = {
          disabled: /\sdisabled(?:\s|>)/.test(etiqueta),
          dataset: { [convertirDataset(atributo)]: valor },
          hasAttribute(nombre) { return nombre === atributo; },
          getAttribute(nombre) { return nombre === atributo ? valor : null; },
          closest(selector) { return selector === `[${atributo}]` ? this : null; },
          focus() { ownerDocument.activeElement = this; },
        };
        return [control];
      });
    },
    contains(control) { return controles.includes(control); },
    querySelector(selector) {
      return controles.find((control) => [...Object.keys(control.dataset)].some((clave) => {
        const atributo = `data-${clave.replace(/[A-Z]/g, (letra) => `-${letra.toLowerCase()}`)}`;
        return selector === `[${atributo}]`;
      })) || null;
    },
    querySelectorAll(selector) {
      return controles.filter((control) => [...Object.keys(control.dataset)].some((clave) => {
        const atributo = `data-${clave.replace(/[A-Z]/g, (letra) => `-${letra.toLowerCase()}`)}`;
        return selector.includes(`[${atributo}]`);
      }));
    },
    buscar(atributo, valor = null) {
      return controles.find((control) => control.hasAttribute(atributo)
        && (valor === null || control.getAttribute(atributo) === String(valor)));
    },
    setAttribute() {},
    removeAttribute() {},
    addEventListener(tipo, escucha) { escuchas.set(tipo, escucha); },
    removeEventListener(tipo) { escuchas.delete(tipo); },
    replaceChildren() { this.innerHTML = ""; },
  };
  return raiz;
}

test("conserva la geometría histórica sin duplicar el mapa del planificador", () => {
  const modelo = presentador().obtenerModelo();
  const mapa = modelo.seleccionada.mapa_ruta;
  assert.equal(mapa.plantilla_teselas, PLANTILLA_TESELAS_OSM_INTERNA);
  assert.equal(mapa.atribucion, ATRIBUCION_OSM_INTERNA);
  assert.equal(mapa.geometria.liquidable, false);
  assert.equal(Object.isFrozen(mapa.geometria), true);
  assert.equal(Object.isFrozen(mapa.geometria.paradas), true);
  assert.equal(Object.isFrozen(mapa.geometria.paradas[0]), true);
  assert.equal(Object.isFrozen(mapa.geometria.trazado), true);
  assert.equal(Object.isFrozen(mapa.geometria.trazado[0]), true);
  assert.throws(() => { mapa.geometria.paradas[0].latitud = 0; }, TypeError);
  assert.throws(() => { mapa.geometria.trazado[0][0] = 0; }, TypeError);
  const html = renderizarDietas(modelo, { descargaDisponible: true });
  assert.doesNotMatch(html, /data-dietas-mapa-canvas/);
  assert.doesNotMatch(html, /data-dietas-mapa-ref="comision-/);
});

test("el visor activa únicamente OpenStreetMap interno y nunca simula un mapa sin Leaflet", () => {
  const descriptorSintetico = presentador().obtenerModelo().seleccionada.mapa_ruta;
  const descriptor = structuredClone(descriptorSintetico);
  descriptor.geometria.origen = "osrm_interno";
  const atributosAcercar = {};
  const atributosAlejar = {};
  const lienzo = {
    innerHTML: '<p class="dietas-mapa-espera">Cargando el mapa interno</p>', dataset: {},
    replaceChildren() { this.innerHTML = ""; },
    querySelector(selector) {
      const atributos = selector === ".leaflet-control-zoom-in" ? atributosAcercar
        : selector === ".leaflet-control-zoom-out" ? atributosAlejar : null;
      return atributos ? { setAttribute(nombre, valor) { atributos[nombre] = valor; } } : null;
    },
  };
  const estado = { textContent: "Cargando mapa interno" };
  const atribucion = { hidden: true };
  const raiz = { querySelector(selector) {
    if (selector === "[data-dietas-mapa-canvas]") return lienzo;
    if (selector === "[data-dietas-mapa-estado]") return estado;
    if (selector === "[data-dietas-mapa-atribucion]") return atribucion;
    return null;
  } };
  assert.equal(crearVisorRutaDietas({ entorno: {} }).montar({ raiz, descriptor }).modo, "mapa_no_disponible");
  assert.doesNotMatch(lienzo.innerHTML, /<svg|polyline|croquis/iu);
  assert.match(estado.textContent, /no está disponible/u);
  assert.equal(atribucion.hidden, true);

  let plantilla;
  let opcionesTeselas;
  let capasSolicitadas = 0;
  let retirado = false;
  let prefijoAtribucion;
  const eventosTeselas = new Map();
  const mapa = {
    attributionControl: { setPrefix(valor) { prefijoAtribucion = valor; } },
    fitBounds() {}, remove() { retirado = true; },
  };
  const capa = () => ({ addTo(destino) { assert.strictEqual(destino, mapa); return this; } });
  const capaTeselas = {
    ...capa(),
    on(tipo, manejador) { eventosTeselas.set(tipo, manejador); return this; },
    off(tipo, manejador) {
      if (eventosTeselas.get(tipo) === manejador) eventosTeselas.delete(tipo);
      return this;
    },
  };
  const entorno = { L: {
    map(destino) { assert.strictEqual(destino, lienzo); return mapa; },
    tileLayer(url, opciones) { capasSolicitadas += 1; plantilla = url; opcionesTeselas = opciones; return capaTeselas; },
    polyline(puntos) {
      assert.deepEqual(puntos, descriptor.geometria.trazado);
      return { ...capa(), getBounds() { return { isValid: () => true }; } };
    },
    circleMarker() { return { ...capa(), bindTooltip() {} }; },
  } };
  const sintetico = crearVisorRutaDietas({ entorno, permitirTeselas: true })
    .montar({ raiz, descriptor: descriptorSintetico });
  assert.equal(sintetico.modo, "mapa_no_disponible");
  assert.equal(capasSolicitadas, 0);
  const sinTeselas = crearVisorRutaDietas({ entorno, permitirTeselas: false }).montar({ raiz, descriptor });
  assert.equal(sinTeselas.modo, "mapa_no_disponible");
  assert.equal(capasSolicitadas, 0);
  const montaje = crearVisorRutaDietas({ entorno, permitirTeselas: true }).montar({ raiz, descriptor });
  assert.equal(montaje.modo, "mapa_cargando");
  assert.match(estado.textContent, /Cargando/u);
  assert.equal(plantilla, "/tiles/osm/{z}/{x}/{y}.png");
  assert.equal(opcionesTeselas.attribution, ATRIBUCION_OSM_INTERNA);
  assert.equal(opcionesTeselas.maxNativeZoom, 14);
  assert.equal(opcionesTeselas.maxZoom, 14);
  assert.equal(prefijoAtribucion, false);
  assert.deepEqual(atributosAcercar, { title: "Acercar el mapa", "aria-label": "Acercar el mapa" });
  assert.deepEqual(atributosAlejar, { title: "Alejar el mapa", "aria-label": "Alejar el mapa" });
  assert.doesNotMatch(plantilla, /^https?:|tile\.openstreetmap\.org/i);
  assert.equal(atribucion.hidden, true);
  assert.equal(capasSolicitadas, 1);
  eventosTeselas.get("load")();
  assert.equal(montaje.modo, "openstreetmap_interno");
  assert.match(estado.textContent, /OpenStreetMap cargado/u);
  montaje.desmontar();
  assert.equal(retirado, true);
});

test("el visor no declara éxito y se retira ante errores de tesela o timeout", () => {
  const descriptor = structuredClone(presentador().obtenerModelo().seleccionada.mapa_ruta);
  descriptor.geometria.origen = "osrm_interno";
  const crearEscenario = () => {
    const eventos = new Map();
    let ejecutarTimeout;
    let retiradas = 0;
    const lienzo = {
      dataset: {}, textContent: "", ownerDocument: null,
      replaceChildren() { this.textContent = ""; },
    };
    const estado = { textContent: "" };
    const atribucion = { hidden: true, textContent: "" };
    const capaBase = { addTo() { return this; } };
    const capaTeselas = {
      ...capaBase,
      on(tipo, manejador) { eventos.set(tipo, manejador); return this; },
      off(tipo, manejador) {
        if (eventos.get(tipo) === manejador) eventos.delete(tipo);
        return this;
      },
    };
    const entorno = {
      setTimeout(tarea, espera) { assert.equal(espera, 25); ejecutarTimeout = tarea; return 1; },
      clearTimeout() {},
      L: {
        map() { return { attributionControl: { setPrefix() {} }, fitBounds() {}, remove() { retiradas += 1; } }; },
        tileLayer() { return capaTeselas; },
        polyline() { return { ...capaBase, getBounds() { return { isValid: () => true }; } }; },
        circleMarker() { return { ...capaBase, bindTooltip() {} }; },
      },
    };
    const raiz = { querySelector(selector) {
      if (selector === "[data-dietas-mapa-canvas]") return lienzo;
      if (selector === "[data-dietas-mapa-estado]") return estado;
      if (selector === "[data-dietas-mapa-atribucion]") return atribucion;
      return null;
    } };
    const montaje = crearVisorRutaDietas({
      entorno, permitirTeselas: true, tiempoEsperaMs: 25,
    }).montar({ raiz, descriptor });
    return { montaje, eventos, ejecutarTimeout: () => ejecutarTimeout(), estado, retiradas: () => retiradas };
  };

  const conErrores = crearEscenario();
  assert.equal(conErrores.montaje.modo, "mapa_cargando");
  conErrores.eventos.get("tileerror")();
  conErrores.eventos.get("tileerror")();
  assert.equal(conErrores.montaje.modo, "mapa_cargando");
  conErrores.eventos.get("tileerror")();
  assert.equal(conErrores.montaje.modo, "mapa_no_disponible");
  assert.match(conErrores.estado.textContent, /no está disponible/u);
  assert.equal(conErrores.retiradas(), 1);

  const conTimeout = crearEscenario();
  conTimeout.ejecutarTimeout();
  assert.equal(conTimeout.montaje.modo, "mapa_no_disponible");
  assert.match(conTimeout.estado.textContent, /no está disponible/u);
  assert.equal(conTimeout.retiradas(), 1);
});

test("compone catálogo provincial y cálculo multiparada sin exponer coordenadas antes de calcular", async () => {
  const { calculador, modulo } = presentadorConRutas();
  const inicial = modulo.obtenerModelo().herramientaRutas;
  assert.equal(inicial.catalogo.completo, true);
  assert.equal(inicial.catalogo.puntos.length, 175);
  assert.deepEqual(inicial.ruta, ["Granada", "Motril", "Granada"]);
  assert.doesNotMatch(JSON.stringify(inicial.catalogo), /latitud|longitud|coordinates/i);

  modulo.rutas.agregarParada();
  let herramienta = modulo.rutas.obtenerModelo();
  assert.equal(herramienta.paradas.length, 4);
  const albolote = herramienta.catalogo.puntos.find((punto) => punto.nombre === "Albolote");
  modulo.rutas.establecerParada(2, albolote.codigo);
  const solicitud = modulo.rutas.prepararSolicitudCalculo();
  assert.deepEqual(solicitud.paradas, ["18087", "18140", "18003", "18087"]);
  const calculo = await calculador.calcular(solicitud);
  assert.equal(calculo.alternativas.length, 3);
  assert.equal(calculo.liquidable, false);
  assert.equal(calculo.motor, "simulacion_osrm_demo");
  assert.equal(Object.isFrozen(calculo.alternativas[0].geometria.trazado[0]), true);
  modulo.rutas.registrarCalculo(calculo);
  herramienta = modulo.rutas.obtenerModelo();
  assert.equal(herramienta.calculado, true);
  assert.equal(herramienta.tramos.length, 3);
  assert.equal(herramienta.mapa_ruta.geometria.origen, "sintetica_demo");
  assert.equal(herramienta.lista_para_borrador, true);
});

test("exige motivo para alternativa y ajuste y conecta la ruta elegida al borrador", async () => {
  const { calculador, modulo } = presentadorConRutas();
  const solicitud = modulo.rutas.prepararSolicitudCalculo();
  modulo.rutas.registrarCalculo(await calculador.calcular(solicitud));
  const alternativa = modulo.rutas.obtenerModelo().alternativas.find((item) => !item.recomendada);
  assert.throws(() => modulo.rutas.seleccionarAlternativa(alternativa.referencia), /motivo/);
  modulo.rutas.seleccionarAlternativa(alternativa.referencia, "Corte de carretera comunicado por el servicio");
  assert.throws(() => modulo.rutas.ajustarTramo(0, 2.5, ""), /motivo/);
  modulo.rutas.ajustarTramo(0, 2.5, "Recorrido adicional acreditado dentro del municipio");
  const rutaBorrador = modulo.rutas.prepararRutaBorrador();
  assert.equal(rutaBorrador.origen, "Granada");
  assert.equal(rutaBorrador.destino, "Motril");
  assert.equal(rutaBorrador.trazabilidad_ruta.motivo_alternativa, "Corte de carretera comunicado por el servicio");
  assert.equal(rutaBorrador.trazabilidad_ruta.ajustes[0].kilometros, 2.5);

  const puerto = adaptador();
  const datos = puerto.ejecutar({ tipo: "crear_borrador", campos: {
    fecha: "2026-07-20", fecha_fin: "2026-07-20", hora_inicio: "08:00", hora_fin: "15:00",
    motivo: "Visita técnica", vehiculo_propio: true,
    manutencion_euros: 20, alojamiento_euros: 0, otros_gastos_euros: 5,
    ...rutaBorrador,
  } });
  const creada = datos.comisiones[0];
  assert.deepEqual(creada.ruta, ["Granada", "Motril", "Granada"]);
  assert.equal(creada.trazabilidad_ruta.calculo_ref, rutaBorrador.trazabilidad_ruta.calculo_ref);
  assert.equal(creada.destino, undefined);
  assert.equal(creada.vehiculo_propio, true);
  assert.equal(creada.hora_inicio, "08:00");
});

test("reinicia por completo la ruta calculada después de cerrar un borrador", async () => {
  const { calculador, modulo } = presentadorConRutas();
  modulo.rutas.agregarParada();
  const albolote = modulo.rutas.obtenerModelo().catalogo.puntos.find((punto) => punto.nombre === "Albolote");
  modulo.rutas.establecerParada(2, albolote.codigo);
  modulo.rutas.registrarCalculo(await calculador.calcular(modulo.rutas.prepararSolicitudCalculo()));
  const alternativa = modulo.rutas.obtenerModelo().alternativas.find((item) => !item.recomendada);
  modulo.rutas.seleccionarAlternativa(alternativa.referencia, "Desvío autorizado por una incidencia del servicio");
  modulo.rutas.ajustarTramo(0, 1.5, "Recorrido adicional acreditado por el responsable");

  const reiniciada = modulo.rutas.reiniciar();
  assert.deepEqual(reiniciada.ruta, ["Granada", "Motril", "Granada"]);
  assert.equal(reiniciada.calculado, false);
  assert.equal(reiniciada.calculo_ref, "");
  assert.equal(reiniciada.alternativa_ref, "");
  assert.equal(reiniciada.motivo_alternativa, "");
  assert.deepEqual(reiniciada.tramos, []);
  assert.equal(reiniciada.kilometros_ajuste, 0);
  assert.equal(reiniciada.lista_para_borrador, false);
});

test("rechaza un cálculo válido que pertenezca a otro entorno", async () => {
  const { calculador, modulo } = presentadorConRutas();
  const solicitud = modulo.rutas.prepararSolicitudCalculo();
  const calculoProducto = structuredClone(await calculador.calcular(solicitud));
  calculoProducto.demostracion = false;
  calculoProducto.motor = "osrm_interno";
  calculoProducto.alternativas.forEach((alternativa) => {
    alternativa.geometria.origen = "osrm_interno";
  });
  assert.throws(
    () => modulo.rutas.registrarCalculo(calculoProducto),
    /no coincide con el entorno de la sesión/u,
  );
  assert.equal(modulo.rutas.obtenerModelo().calculado, false);
});

test("aísla el fallo del catálogo de rutas y mantiene operativos listado y detalle", async () => {
  const raiz = crearRaizDietasMinima();
  const anuncios = [];
  const modulo = await montarModuloDietas({
    raiz,
    contextoActor: CONTEXTO_COMPARTIDO,
    capacidades: CAPACIDADES_EMPLEADO,
    adaptador: adaptador(),
    calculadorRuta: {
      async obtenerCatalogo() { throw new Error("detalle interno que no debe mostrarse"); },
      async calcular() { throw new Error("no debe invocarse"); },
    },
    anunciar: (mensaje, tipo) => anuncios.push({ mensaje, tipo }),
  });

  assert.equal(modulo.obtenerModelo().comisiones.length, 5);
  assert.match(raiz.innerHTML, /DEMO-DIE-2026-0091/);
  assert.match(raiz.innerHTML, /Expediente seleccionado/);
  assert.match(raiz.innerHTML, /La herramienta de rutas no está conectada para esta sesión/);
  assert.doesNotMatch(raiz.innerHTML, /detalle interno que no debe mostrarse/);
  assert.deepEqual(anuncios, [{
    mensaje: "La herramienta de rutas no está conectada para esta sesión.", tipo: "error",
  }]);
  modulo.desmontar();
});

test("un fallo OSRM queda visible dentro de la herramienta con texto i18n gobernado", async () => {
  const raiz = crearRaizDietasInteractiva();
  const anuncios = [];
  const calculadorCatalogo = calculadorRutas();
  const modulo = await montarModuloDietas({
    raiz,
    contextoActor: CONTEXTO_COMPARTIDO,
    capacidades: CAPACIDADES_EMPLEADO,
    adaptador: adaptador(),
    calculadorRuta: {
      obtenerCatalogo: calculadorCatalogo.obtenerCatalogo,
      async calcular() {
        throw {
          codigo: CODIGO_ERROR_SERVICIO_RUTAS_DIETAS,
          message: "HTTP 500: detalle remoto que nunca debe mostrarse",
        };
      },
    },
    anunciar: (mensaje, tipo) => anuncios.push({ mensaje, tipo }),
  });
  const calcular = raiz.buscar("data-dietas-ruta-calcular");
  await raiz.escuchas.get("click")({ target: calcular, preventDefault() {} });

  assert.match(raiz.innerHTML, /data-dietas-ruta-error/);
  assert.match(raiz.innerHTML, /servicio cartográfico interno no está disponible/);
  assert.doesNotMatch(raiz.innerHTML, /HTTP 500|detalle remoto/u);
  assert.deepEqual(anuncios.at(-1), {
    mensaje: "No se ha podido calcular la ruta porque el servicio cartográfico interno no está disponible. Inténtelo de nuevo más tarde.",
    tipo: "error",
  });
  modulo.desmontar();
});

test("conserva el foco de trabajo al repintar las acciones principales de ruta", async () => {
  const raiz = crearRaizDietasInteractiva();
  const calculador = calculadorRutas();
  const modulo = await montarModuloDietas({
    raiz,
    contextoActor: CONTEXTO_COMPARTIDO,
    capacidades: CAPACIDADES_EMPLEADO,
    adaptador: adaptador(),
    calculadorRuta: calculador,
  });
  const pulsarYComprobarFoco = async (atributo, valor = null) => {
    const anterior = raiz.buscar(atributo, valor);
    assert.ok(anterior, `no se encontró ${atributo}`);
    anterior.focus();
    await raiz.escuchas.get("click")({ target: anterior, preventDefault() {} });
    const siguiente = raiz.buscar(atributo, valor);
    assert.ok(siguiente, `no se restauró ${atributo}`);
    assert.notStrictEqual(siguiente, anterior);
    assert.strictEqual(raiz.ownerDocument.activeElement, siguiente);
  };

  await pulsarYComprobarFoco("data-dietas-ruta-anadir");
  await pulsarYComprobarFoco("data-dietas-ruta-quitar", 2);
  await pulsarYComprobarFoco("data-dietas-ruta-calcular");
  const alternativa = raiz.buscar("data-dietas-ruta-alternativa");
  await pulsarYComprobarFoco("data-dietas-ruta-alternativa", alternativa.getAttribute("data-dietas-ruta-alternativa"));
  await pulsarYComprobarFoco("data-dietas-ruta-aplicar-ajuste", 0);
  const ultimaParada = raiz.buscar("data-dietas-ruta-quitar", 2);
  ultimaParada.focus();
  await raiz.escuchas.get("click")({ target: ultimaParada, preventDefault() {} });
  assert.strictEqual(raiz.ownerDocument.activeElement, raiz.buscar("data-dietas-ruta-anadir"));
  modulo.desmontar();
});

test("no compone la herramienta provincial sin ambas capacidades de ruta", () => {
  assert.throws(() => calculadorRutas([CAPACIDAD_CONSULTAR_GASTO]), /capacidad para consultar rutas/);
  const catalogo = calculadorRutas().obtenerCatalogo();
  const modulo = crearPresentadorDietas({
    datos: adaptador([CAPACIDAD_CONSULTAR_GASTO]).obtenerDatos(),
    contextoActor: CONTEXTO_COMPARTIDO,
    capacidades: [CAPACIDAD_CONSULTAR_GASTO],
    catalogoRutas: catalogo,
  });
  assert.equal(modulo.obtenerModelo().herramientaRutas, null);
  assert.doesNotMatch(renderizarDietas(modulo.obtenerModelo()), /Catálogo provincial|data-dietas-ruta-calcular/);
});
