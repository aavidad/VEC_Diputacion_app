import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import {
  ESQUEMA_CONTEXTO_ACTOR_FRONTEND,
  validarYCongelarContextoActor,
} from "../../identidad/contexto-actor.js";
import { crearAdaptadorDietasPresentacion } from "./adaptador-presentacion.js";
import { crearCalculadorRutasDietasPresentacion } from "./calculador-rutas-presentacion.js";
import { crearCalculadorRutasDietasPresentacionOSRM } from "./calculador-rutas-presentacion-osrm.js";
import {
  ATRIBUCION_OSM_INTERNA,
  CAPACIDAD_CONSULTAR_AUDITORIA,
  CAPACIDAD_CONSULTAR_GASTO,
  CAPACIDAD_CONSULTAR_RUTA,
  CAPACIDAD_GESTIONAR_GASTO,
  CAPACIDAD_GESTIONAR_RUTA,
  CODIGO_ERROR_SERVICIO_RUTAS_DIETAS,
  ESQUEMA_RESUMEN_ANUAL_DIETAS,
  PLANTILLA_TESELAS_OSM_INTERNA,
} from "./contrato.js";
import { crearDatosDietasPresentacion } from "./datos-presentacion.js";
import { crearTraductorDietas, MENSAJES_DIETAS_ES } from "./i18n.js";
import { crearVisorRutaDietas } from "./mapa-ruta.js";
import { crearPresentadorDietas } from "./presentador.js";
import { montarModuloDietas, renderizarDietas } from "./vista.js";
import { crearDescargadorRecibosPresentacion } from "../../documentos/descarga-recibos-presentacion.js";
import { cotejarDocumentoPresentacion } from "../../../verificar/adaptador-presentacion.js";

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

const CONTEXTO_PRODUCTIVO = validarYCongelarContextoActor({
  esquema: ESQUEMA_CONTEXTO_ACTOR_FRONTEND,
  revision: 2,
  demostracion: false,
  persona_ref: "per_persona_interna_dietas_000001",
  cuenta_ref: "cta_cuenta_interna_dietas_000001",
  perfil_ref: "prf_perfil_interno_dietas_000001",
  actor: {
    actor_ref: "prf_perfil_interno_dietas_000001",
    iniciales: "PI",
    nombre_visible: "Persona interna",
  },
  rol: {
    clave: "empleado_publico",
    etiqueta: "Personal de la Diputación",
  },
  ambito: {
    clase: "personal_interno",
    organizacion_ref: "org_diputacion_granada_interna_000001",
    unidad_ref: "uni_unidad_interna_dietas_000001",
    modulos: ["bolsa", "cronos", "dietas"],
  },
  autenticacion: {
    sesion_ref: "ses_sesion_interna_dietas_000001",
    metodo: "kerberos_ad",
    garantia: "alto",
  },
  resuelto_en: "2026-07-19T08:00:00.000Z",
});

const CAPACIDADES_EMPLEADO = Object.freeze([
  CAPACIDAD_CONSULTAR_GASTO,
  CAPACIDAD_GESTIONAR_GASTO,
  CAPACIDAD_CONSULTAR_RUTA,
  CAPACIDAD_GESTIONAR_RUTA,
]);
const CAPACIDADES_COMPLETAS = Object.freeze([...CAPACIDADES_EMPLEADO, CAPACIDAD_CONSULTAR_AUDITORIA]);
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
  const atributos = new Map();
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
        const coincidencia = etiqueta.match(/\s(data-dietas-ruta-[a-z-]+)(?:="([^"]*)")?/);
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
    setAttribute(nombre, valor) { atributos.set(nombre, String(valor)); },
    removeAttribute(nombre) { atributos.delete(nombre); },
    getAttribute(nombre) { return atributos.get(nombre) || null; },
    addEventListener(tipo, escucha) { escuchas.set(tipo, escucha); },
    removeEventListener(tipo) { escuchas.delete(tipo); },
    replaceChildren() { this.innerHTML = ""; },
  };
  return raiz;
}

function datosProductivos() {
  const datos = crearDatosDietasPresentacion(CONTEXTO_COMPARTIDO);
  datos.origen = { demostracion: false, efectos_reales: true, adaptador: "http_interno" };
  datos.comisiones.forEach((comision, indiceComision) => {
    comision.titular_ref = CONTEXTO_PRODUCTIVO.actor.actor_ref;
    comision.historial.forEach((evento, indiceEvento) => {
      evento.recibo = `recibo:dietas:${indiceComision}:${indiceEvento}:2026`;
    });
  });
  return datos;
}

test("conserva por referencia el ContextoActor común y rechaza expedientes ajenos", () => {
  assert.throws(() => crearDatosDietasPresentacion(), /contexto debe estar validado/);
  const datos = crearDatosDietasPresentacion(CONTEXTO_COMPARTIDO);
  const modelo = crearPresentadorDietas({
    datos, contextoActor: CONTEXTO_COMPARTIDO, capacidades: CAPACIDADES_EMPLEADO,
  }).obtenerModelo();
  assert.strictEqual(modelo.identidad, CONTEXTO_COMPARTIDO);
  assert.ok(datos.comisiones.every((item) => item.titular_ref === CONTEXTO_COMPARTIDO.actor.actor_ref));

  const contaminados = structuredClone(datos);
  contaminados.comisiones[0].titular_ref = "DEMO-PERFIL-INTERNO-AJENO-01";
  assert.throws(
    () => crearPresentadorDietas({
      datos: contaminados, contextoActor: CONTEXTO_COMPARTIDO, capacidades: CAPACIDADES_EMPLEADO,
    }),
    /expediente ajeno/,
  );
});

test("deniega por defecto y no proyecta referencias, importes ni rutas", () => {
  const modelo = presentador([]).obtenerModelo();
  assert.equal(modelo.capacidades.consultarGastos, false);
  assert.deepEqual(modelo.comisiones, []);
  assert.equal(modelo.seleccionada, null);
  const html = renderizarDietas(modelo);
  assert.match(html, /Acceso a Dietas no autorizado/);
  assert.doesNotMatch(html, /DEMO-DIE-2026|Granada →|267,79/);
  assert.doesNotMatch(html, /data-dietas-formulario|data-dietas-enviar/);
});

test("cierra las filas completas sin lectura de rutas para impedir inferencia de kilometraje", () => {
  const puerto = adaptador([CAPACIDAD_CONSULTAR_GASTO]);
  const inicial = puerto.obtenerDatos();
  assert.deepEqual(inicial.comisiones, []);
  assert.deepEqual(inicial.borrador_inicial, {});
  assert.deepEqual(inicial.politica, {});
  assert.equal(inicial.ultimo_recibo, null);
  const modulo = crearPresentadorDietas({
    datos: inicial, contextoActor: CONTEXTO_COMPARTIDO, capacidades: [CAPACIDAD_CONSULTAR_GASTO],
  });
  const modelo = modulo.obtenerModelo();
  assert.deepEqual(modelo.comisiones, []);
  assert.equal(modelo.seleccionada, null);
  assert.equal(modelo.capacidades.consultarGastos, false);
  assert.ok(Object.values(modelo.resumen).every((valor) => valor === null));
  assert.equal(modelo.resumenAnual, null);
  assert.deepEqual(modelo.historialMensual, []);
  assert.equal(modelo.totalSinFiltrar, null);
  assert.deepEqual(modelo.borradorInicial, {});
  assert.deepEqual(modelo.politica, {});
  const serializado = JSON.stringify({ inicial, modelo });
  assert.doesNotMatch(serializado, /latitud|longitud|geometria|mapa_ruta|trazabilidad_ruta|Motril|140\.8|0\.26|59\.36/);
  assert.throws(() => modulo.seleccionar("DEMO-DIE-BORR-001"), /no tiene capacidad/);
  assert.throws(() => modulo.prepararDescriptorRecibo("DEMO-REC-DIE-BORR-001", t), /no tiene capacidad/);
  assert.throws(() => modulo.prepararDescriptorResumenAnual(2026, t), /no tiene capacidad/);
  const html = renderizarDietas(modelo);
  assert.match(html, /Acceso a Dietas no autorizado/);
  assert.doesNotMatch(html, /tarjeta-kpi/);
  assert.doesNotMatch(html, /Albolote|Motril|Guadix|Loja|Baza/);
  assert.doesNotMatch(html, /data-dietas-mapa|OpenStreetMap|croquis SVG/i);
  assert.doesNotMatch(html, /data-dietas-formulario/);
});

test("el presentador también cierra una proyección parcial heredada sin convertir ausencia en cero", () => {
  const datos = crearDatosDietasPresentacion(CONTEXTO_COMPARTIDO);
  for (const comision of datos.comisiones) {
    comision.ruta = [];
    comision.geometria_ruta = null;
    comision.kilometros = null;
    comision.kilometraje_euros = null;
  }
  // La preimagen dejaba plantilla, tarifa, total y gastos inferibles.
  const modulo = crearPresentadorDietas({
    datos, contextoActor: CONTEXTO_COMPARTIDO, capacidades: [CAPACIDAD_CONSULTAR_GASTO],
  });
  for (const modelo of [modulo.obtenerModelo(), modulo.actualizarDatos(datos)]) {
    assert.deepEqual(modelo.comisiones, []);
    assert.equal(modelo.seleccionada, null);
    assert.deepEqual(modelo.borradorInicial, {});
    assert.deepEqual(modelo.politica, {});
    assert.equal(modelo.resumen.total_euros, null);
    assert.equal(modelo.resumen.kilometros, null);
    assert.equal(modelo.resumenAnual, null);
    assert.deepEqual(modelo.historialMensual, []);
    assert.doesNotMatch(JSON.stringify(modelo), /Motril|140\.8|0\.26|59\.36|18\.5|4\.25/);
  }
});

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
  const recomendada = modulo.rutas.prepararRutaBorrador();
  const rutaRecomendada = recomendada.trazabilidad_ruta.ruta;
  assert.deepEqual(Object.keys(rutaRecomendada).sort(), [
    "alternativa", "calculo", "catalogo_version", "esquema", "kilometros", "liquidable", "paradas", "tramos", "trazado",
  ]);
  assert.equal(rutaRecomendada.alternativa.recomendada, true);
  assert.equal(rutaRecomendada.alternativa.motivo, "");
  assert.equal(rutaRecomendada.liquidable, false);
  assert.equal(rutaRecomendada.catalogo_version, modulo.rutas.obtenerModelo().catalogo.version);
  assert.equal(rutaRecomendada.paradas[0].codigo, "18087");
  assert.equal(rutaRecomendada.paradas[0].nombre, "Granada");
  assert.equal(typeof rutaRecomendada.paradas[0].latitud, "number");
  assert.equal(rutaRecomendada.tramos[0].origen.codigo, rutaRecomendada.paradas[0].codigo);
  assert.equal(rutaRecomendada.tramos[0].destino.codigo, rutaRecomendada.paradas[1].codigo);
  assert.equal(rutaRecomendada.tramos[0].kilometros_finales, rutaRecomendada.tramos[0].kilometros);

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
  const rutaAlternativa = rutaBorrador.trazabilidad_ruta.ruta;
  assert.equal(rutaAlternativa.alternativa.recomendada, false);
  assert.equal(rutaAlternativa.alternativa.motivo, "Corte de carretera comunicado por el servicio");
  assert.equal(rutaAlternativa.tramos[0].ajuste_kilometros, 2.5);
  assert.equal(rutaAlternativa.tramos[0].motivo_ajuste, "Recorrido adicional acreditado dentro del municipio");
  assert.equal(rutaAlternativa.tramos[0].kilometros_finales,
    Math.round((rutaAlternativa.tramos[0].kilometros + 2.5) * 10) / 10);
  assert.equal(rutaAlternativa.kilometros.final,
    Math.round((rutaAlternativa.kilometros.base + rutaAlternativa.kilometros.ajuste) * 10) / 10);
  assert.equal(rutaAlternativa.liquidable, false);
  assert.equal(rutaBorrador.trazabilidad_ruta.liquidable, false);
  assert.equal(Object.isFrozen(rutaAlternativa), true);
  assert.equal(Object.isFrozen(rutaAlternativa.paradas[0]), true);
  assert.equal(Object.isFrozen(rutaAlternativa.trazado[0]), true);
  const copiaRuta = structuredClone(rutaAlternativa);
  copiaRuta.paradas[0].nombre = "Cambio local";
  copiaRuta.tramos[0].origen.codigo = "OTRO";
  copiaRuta.trazado[0][0] = 0;
  assert.equal(rutaAlternativa.paradas[0].nombre, "Granada");
  assert.equal(rutaAlternativa.tramos[0].origen.codigo, rutaAlternativa.paradas[0].codigo);
  assert.notEqual(rutaAlternativa.trazado[0][0], 0);

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
  assert.equal(creada.kilometros, rutaBorrador.kilometros);
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

test("rechaza cálculo de ruta incompleto antes de crear su snapshot", async () => {
  const { calculador, modulo } = presentadorConRutas();
  const solicitud = modulo.rutas.prepararSolicitudCalculo();
  const incompleto = structuredClone(await calculador.calcular(solicitud));
  delete incompleto.alternativas[0].geometria.trazado;
  assert.throws(() => modulo.rutas.registrarCalculo(incompleto), /geometria de ruta de Dietas no valid[oa]/u);
  assert.equal(modulo.rutas.obtenerModelo().calculado, false);
  assert.throws(() => modulo.rutas.prepararRutaBorrador(), /calcule y justifique/u);
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

  assert.equal(modulo.obtenerModelo().comisiones.length, 1);
  assert.match(raiz.innerHTML, /DEMO-DIE-BORR-001/);
  assert.match(raiz.innerHTML, /Expediente seleccionado/);
  assert.match(raiz.innerHTML, /La herramienta de rutas no está conectada para esta sesión/);
  assert.doesNotMatch(raiz.innerHTML, /detalle interno que no debe mostrarse/);
  assert.deepEqual(anuncios, [{
    mensaje: "La herramienta de rutas no está conectada para esta sesión.", tipo: "error",
  }]);
  modulo.desmontar();
});

for (const etapa of ["catalogo", "datos"]) {
  for (const desenlace of ["resuelve", "rechaza"]) {
    test("un montaje pendiente en " + etapa + " no altera Cronos cuando " + desenlace, async () => {
      const raiz = crearRaizDietasInteractiva();
      let resolverInicial;
      let rechazarInicial;
      const pendiente = new Promise((resolver, rechazar) => {
        resolverInicial = resolver;
        rechazarInicial = rechazar;
      });
      const primeroPendiente = montarModuloDietas({
        raiz,
        contextoActor: CONTEXTO_COMPARTIDO,
        capacidades: CAPACIDADES_EMPLEADO,
        adaptador: etapa === "datos"
          ? { obtenerDatos: () => pendiente, ejecutar: adaptador().ejecutar }
          : adaptador(),
        calculadorRuta: etapa === "catalogo"
          ? { obtenerCatalogo: () => pendiente, calcular: calculadorRutas().calcular }
          : undefined,
      });

      // El coordinador compartido sustituye Dietas por Cronos sin invocar
      // desmontar de Dietas mientras la carga inicial sigue pendiente.
      const htmlCronos = '<section data-modulo="cronos">Cronos activo</section>';
      const escuchaCronos = () => "cronos";
      raiz.innerHTML = htmlCronos;
      raiz.escuchas.set("click", escuchaCronos);

      if (desenlace === "rechaza") rechazarInicial(new Error("fallo tardio de Dietas"));
      else resolverInicial({ respuesta_tardia: true });
      const montajeObsoleto = await primeroPendiente;
      montajeObsoleto.desmontar();

      assert.equal(montajeObsoleto.obtenerModelo(), null);
      assert.equal(raiz.innerHTML, htmlCronos);
      assert.strictEqual(raiz.escuchas.get("click"), escuchaCronos);
      assert.equal(raiz.escuchas.size, 1);
      assert.equal(raiz.getAttribute("aria-busy"), null);
    });
  }

  for (const desenlace of ["resuelve", "rechaza"]) {
    test(`el montaje inicial pendiente en ${etapa} que ${desenlace} no retira ni altera su reemplazo`, async () => {
      const raiz = crearRaizDietasInteractiva();
      const anuncios = [];
      let resolverInicial;
      let rechazarInicial;
      let lecturasDatos = 0;
      const inicialPendiente = new Promise((resolver, rechazar) => {
        resolverInicial = resolver;
        rechazarInicial = rechazar;
      });
      const calculador = calculadorRutas();
      const datos = adaptador();
      const primeroPendiente = montarModuloDietas({
        raiz, contextoActor: CONTEXTO_COMPARTIDO, capacidades: CAPACIDADES_EMPLEADO,
        adaptador: {
          obtenerDatos() {
            lecturasDatos += 1;
            return etapa === "datos" ? inicialPendiente : datos.obtenerDatos();
          },
          ejecutar: datos.ejecutar,
        },
        calculadorRuta: etapa === "catalogo" ? {
          obtenerCatalogo: () => inicialPendiente,
          calcular: calculador.calcular,
        } : undefined,
        anunciar: (...mensaje) => anuncios.push(mensaje),
      });
      assert.equal(raiz.escuchas.size, 0);
      let resolverCalculo;
      let solicitudActual;
      const segundo = await montarModuloDietas({
        raiz, contextoActor: CONTEXTO_COMPARTIDO, capacidades: CAPACIDADES_EMPLEADO,
        adaptador: adaptador(),
        calculadorRuta: {
          obtenerCatalogo: calculador.obtenerCatalogo,
          calcular(solicitud) {
            solicitudActual = solicitud;
            return new Promise((resolver) => { resolverCalculo = resolver; });
          },
        },
        anunciar: (...mensaje) => anuncios.push(mensaje),
      });
      const escuchasActuales = [...raiz.escuchas];
      const calcular = raiz.buscar("data-dietas-ruta-calcular");
      const operacionActual = raiz.escuchas.get("click")({ target: calcular, preventDefault() {} });
      const htmlActual = raiz.innerHTML;
      assert.equal(raiz.getAttribute("aria-busy"), "true");
      assert.equal(calcular.disabled, true);
      if (desenlace === "rechaza") rechazarInicial(new Error("fallo inicial obsoleto"));
      else resolverInicial({ respuesta_obsoleta: true });
      const primero = await primeroPendiente;
      // El coordinador puede retirar A después de recibir su controlador tardío.
      primero.desmontar();
      primero.desmontar();
      assert.equal(primero.obtenerModelo(), null);
      assert.equal(raiz.innerHTML, htmlActual);
      assert.deepEqual([...raiz.escuchas], escuchasActuales);
      assert.equal(raiz.getAttribute("aria-busy"), "true");
      assert.equal(calcular.disabled, true);
      assert.deepEqual(anuncios, []);
      assert.equal(lecturasDatos, etapa === "datos" ? 1 : 0);
      // B mantiene sus escuchadores y completa su propia operación normalmente.
      resolverCalculo(await calculador.calcular(solicitudActual));
      await operacionActual;
      assert.equal(segundo.obtenerModelo().herramientaRutas.calculado, true);
      assert.equal(raiz.getAttribute("aria-busy"), null);
      assert.equal(anuncios.length, 1);
      segundo.desmontar();
      assert.equal(raiz.innerHTML, "");
      assert.equal(raiz.escuchas.size, 0);
      assert.equal(raiz.getAttribute("aria-busy"), null);
    });
  }
}

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

test("una respuesta tardía no altera una raíz Dietas reutilizada ni desbloquea su operación", async () => {
  const raiz = crearRaizDietasInteractiva();
  const anuncios = [];
  const pendientes = [];
  const calculadorPendiente = {
    obtenerCatalogo() { return calculadorRutas().obtenerCatalogo(); },
    calcular(_solicitud, opciones = {}) {
      return new Promise((resolver, rechazar) => pendientes.push({ resolver, rechazar, signal: opciones.signal }));
    },
  };
  const primero = await montarModuloDietas({
    raiz, contextoActor: CONTEXTO_COMPARTIDO, capacidades: CAPACIDADES_EMPLEADO,
    adaptador: adaptador(), calculadorRuta: calculadorPendiente,
    anunciar: (...mensaje) => anuncios.push(mensaje),
  });
  const iniciar = (montaje) => raiz.escuchas.get("click")({
    target: raiz.buscar("data-dietas-ruta-calcular"), preventDefault() {},
  });
  const viejo = iniciar(primero);
  assert.equal(pendientes.length, 1);
  assert.equal(pendientes[0].signal instanceof AbortSignal, true);
  primero.desmontar();
  // El desmontaje limpia su indicador antes de abortar la promesa antigua.
  assert.equal(raiz.getAttribute("aria-busy"), null);
  assert.equal(pendientes[0].signal.aborted, true);

  const segundo = await montarModuloDietas({
    raiz, contextoActor: CONTEXTO_COMPARTIDO, capacidades: CAPACIDADES_EMPLEADO,
    adaptador: adaptador(), calculadorRuta: calculadorPendiente,
    anunciar: (...mensaje) => anuncios.push(mensaje),
  });
  // Un remonte sin operación propia sigue estando inactivo.
  assert.equal(raiz.getAttribute("aria-busy"), null);
  const actual = iniciar(segundo);
  assert.equal(pendientes.length, 2);
  assert.equal(raiz.getAttribute("aria-busy"), "true");
  const htmlActivo = raiz.innerHTML;

  // El valor no necesita ser válido: la instancia desmontada no debe ni leerlo.
  pendientes[0].resolver({ respuesta_tardia: true });
  await viejo;
  assert.equal(raiz.getAttribute("aria-busy"), "true");
  assert.equal(raiz.innerHTML, htmlActivo);
  assert.deepEqual(anuncios, []);

  pendientes[1].rechazar(new Error("fallo tardío actual"));
  await actual;
  assert.equal(raiz.getAttribute("aria-busy"), null);
  assert.equal(anuncios.length, 1);
  segundo.desmontar();
});

test("un rechazo tardío de una instancia desmontada no pinta ni anuncia en su reemplazo", async () => {
  const raiz = crearRaizDietasInteractiva();
  const anuncios = [];
  let rechazar;
  const calculadorPendiente = {
    obtenerCatalogo() { return calculadorRutas().obtenerCatalogo(); },
    calcular() { return new Promise((_resolver, rechazo) => { rechazar = rechazo; }); },
  };
  const primero = await montarModuloDietas({
    raiz, contextoActor: CONTEXTO_COMPARTIDO, capacidades: CAPACIDADES_EMPLEADO,
    adaptador: adaptador(), calculadorRuta: calculadorPendiente,
    anunciar: (...mensaje) => anuncios.push(mensaje),
  });
  const pendiente = raiz.escuchas.get("click")({
    target: raiz.buscar("data-dietas-ruta-calcular"), preventDefault() {},
  });
  primero.desmontar();
  const reemplazo = await montarModuloDietas({
    raiz, contextoActor: CONTEXTO_COMPARTIDO, capacidades: CAPACIDADES_EMPLEADO,
    adaptador: adaptador(), calculadorRuta: calculadorRutas(),
    anunciar: (...mensaje) => anuncios.push(mensaje),
  });
  const htmlReemplazo = raiz.innerHTML;
  rechazar(new Error("fallo de una instancia retirada"));
  await pendiente;
  assert.equal(raiz.innerHTML, htmlReemplazo);
  assert.deepEqual(anuncios, []);
  reemplazo.desmontar();
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

test("renderiza abierta la herramienta operativa completa de nueva comisión", async () => {
  const { calculador, modulo } = presentadorConRutas();
  modulo.rutas.registrarCalculo(await calculador.calcular(modulo.rutas.prepararSolicitudCalculo()));
  const html = renderizarDietas(modulo.obtenerModelo(), { descargaDisponible: true, confirmacionDisponible: true });
  assert.match(html, /<details class="panel dietas-nueva" open>/);
  assert.match(html, /Catálogo provincial de localidades/);
  assert.match(html, /data-dietas-ruta-anadir/);
  assert.match(html, /data-dietas-ruta-calcular/);
  assert.match(html, /data-dietas-ruta-alternativa/);
  assert.match(html, /data-dietas-ruta-aplicar-ajuste/);
  assert.match(html, /Fecha de inicio/);
  assert.match(html, /Hora de fin/);
  assert.match(html, /Vehículo propio/);
  assert.match(html, /data-dietas-resumen-borrador="total"/);
  assert.match(html, /data-dietas-mapa-ref="borrador-ruta-calculada"/);
  assert.match(html, /aria-label="Quitar Granada, posición 1"/);
  assert.match(html, /aria-label="Quitar Motril, posición 2"/);
});

test("resume comisiones, kilometraje, importes y pagos con capacidades explícitas", () => {
  const modelo = presentador().obtenerModelo();
  assert.deepEqual(modelo.resumen, {
    expedientes: 1, pendientes: 1, kilometros: 140.8, total_euros: 59.36, pagado_euros: 0,
  });
  assert.equal(modelo.historialMensual.length, 1);
  assert.equal(modelo.comisiones[0].ruta.join(" → "), "Granada → Motril → Granada");
  assert.equal(modelo.seleccionada.historial.length, 1);
});

test("filtra mediante códigos canónicos independientes del idioma", () => {
  const modulo = presentador();
  assert.equal(modulo.filtrar({ estado: "borrador", texto: "Motril" }).comisiones.length, 1);
  assert.equal(modulo.filtrar({ estado: "todos", texto: "sin coincidencia" }).comisiones.length, 0);
  assert.throws(() => modulo.filtrar({ estado: "Pagada" }), /no permitido/);
});

test("el adaptador DEMO crea un borrador volátil con ruta calculada y recibo del actor común", async () => {
  const { calculador, modulo } = presentadorConRutas();
  modulo.rutas.registrarCalculo(await calculador.calcular(modulo.rutas.prepararSolicitudCalculo()));
  const ruta = modulo.rutas.prepararRutaBorrador();
  const puerto = adaptador();
  assert.throws(() => puerto.ejecutar({ tipo: "crear_borrador", campos: {
    fecha: "2026-07-20", motivo: "Visita técnica", manutencion_euros: "20", alojamiento_euros: "0", otros_gastos_euros: "5",
  } }), (error) => error.codigo === "DIETAS_RUTA_PENDIENTE");
  assert.throws(() => puerto.ejecutar({ tipo: "crear_borrador", campos: {
    fecha: "2026-07-20", motivo: "Visita técnica", vehiculo_propio: false, manutencion_euros: "", alojamiento_euros: "0", otros_gastos_euros: "5", ...ruta,
  } }), (error) => error.codigo === "DIETAS_GASTO_REQUERIDO");
  const antesVehiculoInvalido = puerto.obtenerDatos();
  for (const vehiculo_propio of [undefined, null, "true", 1]) {
    assert.throws(() => puerto.ejecutar({ tipo: "crear_borrador", campos: {
      fecha: "2026-07-20", motivo: "Visita técnica", vehiculo_propio,
      manutencion_euros: "20", alojamiento_euros: "0", otros_gastos_euros: "5", ...ruta,
    } }), (error) => error.codigo === "DIETAS_VEHICULO_PROPIO_INVALIDO");
  }
  assert.deepEqual(puerto.obtenerDatos(), antesVehiculoInvalido);
  const datos = puerto.ejecutar({ tipo: "crear_borrador", campos: {
    fecha: "2026-07-20", motivo: "Visita técnica", vehiculo_propio: false,
    manutencion_euros: "20", alojamiento_euros: "0", otros_gastos_euros: "5", ...ruta,
  } });
  const modelo = crearPresentadorDietas({ datos, contextoActor: CONTEXTO_COMPARTIDO, capacidades: CAPACIDADES_EMPLEADO }).obtenerModelo();
  assert.equal(modelo.resumen.expedientes, 2);
  assert.equal(modelo.comisiones[0].referencia, "DEMO-DIE-NUEVA-999");
  assert.equal(modelo.comisiones[0].total_euros, Math.round((modelo.comisiones[0].kilometraje_euros + 25) * 100) / 100);
  assert.equal(modelo.comisiones[0].kilometros, ruta.kilometros);
  assert.equal(modelo.comisiones[0].trazabilidad_ruta.ruta.kilometros.final, ruta.kilometros);
  assert.equal(modelo.ultimoRecibo.efectos_reales, false);
  const sinConsultaRuta = adaptador([CAPACIDAD_CONSULTAR_GASTO, CAPACIDAD_GESTIONAR_GASTO, CAPACIDAD_GESTIONAR_RUTA]);
  assert.deepEqual(sinConsultaRuta.obtenerDatos().comisiones, []);
  const creadaSinConsultaRuta = sinConsultaRuta.ejecutar({ tipo: "crear_borrador", campos: {
    fecha: "2026-07-20", motivo: "Visita técnica", vehiculo_propio: false,
    manutencion_euros: "20", alojamiento_euros: "0", otros_gastos_euros: "5", ...ruta,
  } });
  assert.deepEqual(creadaSinConsultaRuta.comisiones, []);
  assert.deepEqual(creadaSinConsultaRuta.borrador_inicial, {});
  assert.deepEqual(creadaSinConsultaRuta.politica, {});
  assert.equal(creadaSinConsultaRuta.ultimo_recibo, null);
  assert.deepEqual(sinConsultaRuta.obtenerDatos(), creadaSinConsultaRuta);
  const modeloSinRuta = crearPresentadorDietas({
    datos: creadaSinConsultaRuta, contextoActor: CONTEXTO_COMPARTIDO,
    capacidades: [CAPACIDAD_CONSULTAR_GASTO, CAPACIDAD_GESTIONAR_GASTO, CAPACIDAD_GESTIONAR_RUTA],
  }).obtenerModelo();
  assert.deepEqual(modeloSinRuta.comisiones, []);
  assert.equal(modeloSinRuta.seleccionada, null);
  assert.equal(modeloSinRuta.ultimoRecibo, null);
  assert.equal(modeloSinRuta.resumen.total_euros, null);
  assert.equal(modeloSinRuta.resumen.kilometros, null);
  assert.deepEqual(modeloSinRuta.borradorInicial, {});
  assert.deepEqual(modeloSinRuta.politica, {});
  assert.doesNotMatch(JSON.stringify({ creadaSinConsultaRuta, modeloSinRuta }), /Motril|140\.8|0\.26|trazabilidad_ruta|kilometraje_euros/);
  assert.throws(() => adaptador([]).ejecutar({ tipo: "enviar_validacion", referencia: "DEMO-DIE-BORR-001" }), /no tiene capacidad/);
});

test("crear sin lectura de ruta no revela distancias ni importes con o sin vehículo propio", async () => {
  const { calculador, modulo } = presentadorConRutas();
  modulo.rutas.registrarCalculo(await calculador.calcular(modulo.rutas.prepararSolicitudCalculo()));
  const ruta = modulo.rutas.prepararRutaBorrador();
  const capacidades = [CAPACIDAD_CONSULTAR_GASTO, CAPACIDAD_GESTIONAR_GASTO, CAPACIDAD_GESTIONAR_RUTA];
  for (const vehiculo_propio of [true, false]) {
    const puerto = adaptador(capacidades);
    const inicial = puerto.obtenerDatos();
    const proyeccion = crearPresentadorDietas({ datos: inicial, contextoActor: CONTEXTO_COMPARTIDO, capacidades });
    const creado = puerto.ejecutar({ tipo: "crear_borrador", campos: {
      fecha: "2026-07-20", motivo: "Visita sintética Motril", vehiculo_propio,
      manutencion_euros: "17", alojamiento_euros: "11", otros_gastos_euros: "7", ...ruta,
    } });
    // La respuesta tampoco revela que se añadió una fila oculta o su recibo.
    assert.deepEqual(creado, inicial);
    const modelo = proyeccion.actualizarDatos(creado);
    assert.deepEqual(modelo.comisiones, []);
    assert.deepEqual(modelo.borradorInicial, {});
    assert.deepEqual(modelo.politica, {});
    assert.equal(modelo.seleccionada, null);
    assert.equal(modelo.ultimoRecibo, null);
    assert.equal(modelo.resumenAnual, null);
    assert.deepEqual(modelo.historialMensual, []);
    assert.ok(Object.values(modelo.resumen).every((valor) => valor === null));
    assert.doesNotMatch(JSON.stringify({ creado, modelo }), /Motril|140\.8|0\.26|36\.61|71\.61|trazabilidad_ruta|kilometraje_euros/);
    assert.match(renderizarDietas(modelo), /Acceso a Dietas no autorizado/);
  }
});

test("conecta OSRM interno al presentador, borrador, lista, detalle y recibo sin efectos", async () => {
  const respuesta = { code: "Ok", engine: "osrm_on_premise", route_scope: "Granada provincia + 15 km", graph_version: "granada-buffer-osrm-v1-53aba0ad43c4", routes: [{ distance: 140800, duration: 6600, legs: [{ distance: 70400, duration: 3300 }, { distance: 70400, duration: 3300 }], geometry: { type: "LineString", coordinates: [[-3.59869101, 37.17428891], [-3.52045559, 36.74535308], [-3.59869101, 37.17428891]] } }] };
  const calculador = crearCalculadorRutasDietasPresentacionOSRM({ contextoActor: CONTEXTO_COMPARTIDO, capacidades: CAPACIDADES_EMPLEADO, fetchImpl: async () => new Response(JSON.stringify(respuesta), { headers: { "Content-Type": "application/json" } }) });
  const modulo = crearPresentadorDietas({ datos: adaptador().obtenerDatos(), contextoActor: CONTEXTO_COMPARTIDO, capacidades: CAPACIDADES_EMPLEADO, catalogoRutas: calculador.obtenerCatalogo() });
  modulo.rutas.registrarCalculo(await calculador.calcular(modulo.rutas.prepararSolicitudCalculo()));
  const ruta = modulo.rutas.prepararRutaBorrador();
  const datos = adaptador().ejecutar({ tipo: "crear_borrador", campos: { fecha: "2026-07-20", motivo: "Visita técnica", vehiculo_propio: false, manutencion_euros: 0, alojamiento_euros: 0, otros_gastos_euros: 0, ...ruta } });
  const final = crearPresentadorDietas({ datos, contextoActor: CONTEXTO_COMPARTIDO, capacidades: CAPACIDADES_EMPLEADO }).obtenerModelo();
  assert.equal(final.comisiones[0].trazabilidad_ruta.motor, "osrm_interno");
  assert.equal(final.seleccionada.referencia, "DEMO-DIE-NUEVA-999");
  assert.equal(final.ultimoRecibo.efectos_reales, false);
  assert.equal(final.comisiones[0].vehiculo_propio, false);
  assert.equal(final.comisiones[0].kilometros, ruta.kilometros);
  assert.equal(final.comisiones[0].trazabilidad_ruta.ruta.kilometros.final, ruta.kilometros);
  assert.equal(final.comisiones[0].kilometraje_euros, 0);
  assert.equal(final.comisiones[0].total_euros, 0);
  const html = renderizarDietas(final, { descargaDisponible: true });
  assert.match(html, /0,00 €/);
  assert.match(html, /DEMO-REC-DIE-VOL-/);
});

test("la presentación rechaza enviar_validacion sin cambiar el borrador", () => {
  const puerto = adaptador();
  assert.throws(
    () => puerto.ejecutar({ tipo: "enviar_validacion", referencia: "DEMO-DIE-BORR-001" }),
    /no envía borradores ni simula validaciones/,
  );
  const item = puerto.obtenerDatos().comisiones[0];
  assert.equal(item.estado, "borrador");
  assert.equal(item.historial.length, 1);
});

test("prepara PDF DEMO con logo y QR resoluble sin datos personales", () => {
  const descriptor = presentador().prepararDescriptorRecibo("DEMO-REC-DIE-BORR-001", t);
  assert.equal(descriptor.formato, "pdf");
  assert.equal(descriptor.identidad_visual.logo_src, "/assets/logo-diputacion-granada.svg");
  assert.equal(descriptor.comprobacion.qr_contenido, "https://vec.demo.dipgra.es/verificar/?ref=DEMO-REC-DIE-BORR-001&presentacion=rrhh");
  assert.equal(descriptor.comprobacion.metodo, "consulta_estatica_demo");
  assert.equal(descriptor.comprobacion.contiene_datos_personales, false);
  assert.doesNotMatch(descriptor.comprobacion.qr_contenido, /Agente|DNI|nombre/i);
  assert.match(descriptor.marca, /SIN EFECTOS ADMINISTRATIVOS/);
});

test("genera el resumen anual PDF real por el puerto documental común", async () => {
  const modulo = presentador();
  const descriptor = modulo.prepararDescriptorResumenAnual(2026, t);
  assert.equal(descriptor.esquema, ESQUEMA_RESUMEN_ANUAL_DIETAS);
  assert.equal(descriptor.referencia, "DEMO-DIE-REC-ANUAL-2026-01");
  assert.equal(descriptor.filas.length, 8);
  assert.equal(descriptor.comprobacion.contiene_datos_personales, false);
  assert.deepEqual(cotejarDocumentoPresentacion(descriptor.referencia), {
    valido: true,
    titulo: "Documento de demostración reconocido",
    mensaje: "La referencia corresponde a un recibo generado localmente para revisar el recorrido. No acredita una actuación administrativa real.",
    referencia: descriptor.referencia,
    estado: "DEMO · sin validez administrativa",
    alcance: "Comprobación local, sin consulta a registros, firma o sello reales",
  });
  assert.doesNotMatch(JSON.stringify(descriptor), /persona_ref|actor_ref|dni|latitud|longitud/i);

  let blob;
  let nombre;
  const entorno = {
    URL: { createObjectURL(valor) { blob = valor; return "blob:resumen-anual"; }, revokeObjectURL() {} },
    location: { origin: "https://vec.demo.dipgra.es" },
    document: {
      body: { append() {} },
      createElement() {
        return { click() {}, remove() {}, set href(_valor) {}, set download(valor) { nombre = valor; } };
      },
    },
    setTimeout(funcion) { funcion(); },
  };
  const resultado = await crearDescargadorRecibosPresentacion(entorno)(descriptor);
  assert.equal(resultado.formato, "application/pdf");
  assert.equal(nombre, "resumen-anual-dietas-demo-2026.pdf");
  const pdf = Buffer.from(await blob.arrayBuffer()).toString("latin1");
  assert.match(pdf, /^%PDF-1\.4/);
  assert.match(pdf, /Resumen anual de Dietas 2026/);
  assert.match(pdf, /DEMO-DIE-REC-ANUAL-2026-01/);
  assert.ok(blob.size > 10_000);

  assert.throws(() => presentador([CAPACIDAD_CONSULTAR_GASTO])
    .prepararDescriptorResumenAnual(2026, t), /no tiene capacidad/);
  assert.throws(() => crearPresentadorDietas({
    datos: datosProductivos(), contextoActor: CONTEXTO_PRODUCTIVO,
    capacidades: CAPACIDADES_EMPLEADO,
  }).prepararDescriptorResumenAnual(2026, t), /servicio documental autorizado/);
});

test("acepta referencias opacas productivas, rechaza DEMO y prepara cotejo POST", () => {
  const modulo = crearPresentadorDietas({
    datos: datosProductivos(), contextoActor: CONTEXTO_PRODUCTIVO,
    capacidades: CAPACIDADES_EMPLEADO, origenComprobacion: "https://vec.dipgra.es",
  });
  const referencia = "recibo:dietas:0:0:2026";
  const descriptor = modulo.prepararDescriptorRecibo(referencia, t);
  assert.equal(descriptor.marca, "");
  assert.equal(descriptor.comprobacion.metodo, "post_servicio_cotejo");
  assert.equal(descriptor.comprobacion.qr_contenido, "https://vec.dipgra.es/verificar/?ref=recibo%3Adietas%3A0%3A0%3A2026");

  const datosConDemo = datosProductivos();
  datosConDemo.comisiones[0].historial[0].recibo = "DEMO-REC-DIE-PROHIBIDO-01";
  const moduloConDemo = crearPresentadorDietas({
    datos: datosConDemo, contextoActor: CONTEXTO_PRODUCTIVO, capacidades: CAPACIDADES_EMPLEADO,
  });
  assert.throws(() => moduloConDemo.prepararDescriptorRecibo("DEMO-REC-DIE-PROHIBIDO-01", t), /no encontrado/);
});

test("renderiza un espacio administrativo accesible, denso y traducido", () => {
  const html = renderizarDietas(presentador(CAPACIDADES_COMPLETAS).obtenerModelo(), {
    descargaDisponible: true, confirmacionDisponible: true,
  });
  assert.match(html, /Portal del Empleado → Dietas/);
  assert.match(html, /Agente interno DEMO/);
  assert.match(html, /Presentación · datos sintéticos · no liquidable · no enviado/);
  assert.match(html, /Desglose de gastos/);
  assert.match(html, /Historial y trazabilidad/);
  assert.match(html, /<caption>/);
  assert.match(html, /<th scope="col"/);
  assert.match(html, /data-dietas-descargar-recibo="DEMO-REC-DIE-BORR-001"/);
  assert.match(html, /aria-current="true"/);
  assert.doesNotMatch(html, /aria-selected=/);
  assert.doesNotMatch(html, /onclick=|javascript:/i);
});

test("permite sustituir todo el catálogo de interfaz sin cambiar estados", () => {
  const alternativo = { ...MENSAJES_DIETAS_ES, titulo: "Expense workspace", buscar: "Search" };
  const html = renderizarDietas(presentador().obtenerModelo(), { mensajes: alternativo });
  assert.match(html, /Expense workspace/);
  assert.match(html, />Search<input/);
  assert.match(html, /value="borrador"/);
  assert.doesNotMatch(html, />Mis dietas y comisiones de servicio</);
  assert.throws(() => crearTraductorDietas({ titulo: "incompleto" }), /incompleto/);
});

test("las etiquetas ordinales OSRM proceden íntegramente del catálogo i18n", async () => {
  const { calculador, modulo } = presentadorConRutas();
  const solicitudRuta = modulo.rutas.prepararSolicitudCalculo();
  const calculo = structuredClone(await calculador.calcular(solicitudRuta));
  calculo.motor = "osrm_interno";
  calculo.version_grafo = "grafo-osrm-prueba-i18n";
  calculo.alternativas.forEach((alternativa, indice) => {
    alternativa.etiqueta = `ruta_alternativa_osrm_${indice + 1}`;
    alternativa.geometria.origen = "osrm_interno";
  });
  modulo.rutas.registrarCalculo(calculo);
  const mensajes = {
    ...MENSAJES_DIETAS_ES,
    ruta_etiqueta_osrm_1: "Itinerario corporativo preferente",
    ruta_etiqueta_osrm_2: "Itinerario corporativo alternativo B",
    ruta_etiqueta_osrm_3: "Itinerario corporativo alternativo C",
  };
  const html = renderizarDietas(modulo.obtenerModelo(), { mensajes });
  assert.match(html, /Itinerario corporativo preferente/);
  assert.match(html, /Itinerario corporativo alternativo B/);
  assert.match(html, /Itinerario corporativo alternativo C/);
  assert.doesNotMatch(html, /ruta_alternativa_osrm_|Ruta OSRM interna · primera/u);
  assert.match(html, /data-dietas-mapa-estado role="status" aria-live="polite" aria-atomic="true"/u);
});

test("la vista final no importa fixtures y gobierna concurrencia e interfaces por puertos", async () => {
  const fuentes = await Promise.all([
    "contrato.js", "datos-presentacion.js", "adaptador-presentacion.js", "presentador.js", "vista.js", "i18n.js", "mapa-ruta.js",
    "presentador-rutas.js", "calculador-rutas-presentacion.js",
  ].map((archivo) => readFile(new URL(archivo, import.meta.url), "utf8")));
  const codigo = fuentes.join("\n");
  assert.doesNotMatch(codigo, /\bfetch\s*\(|XMLHttpRequest|WebSocket|\.cookie\b|localStorage|sessionStorage|indexedDB/);
  assert.doesNotMatch(codigo, /tile\.openstreetmap\.org|google\s*maps|mapbox/i);
  assert.match(codigo, /\/tiles\/osm\/\{z\}\/\{x\}\/\{y\}\.png/);
  const vista = fuentes[4];
  assert.doesNotMatch(vista, /datos-presentacion|adaptador-presentacion/);
  assert.doesNotMatch(vista, /Visita técnica|2026-07-20|value="Granada"|value="Motril"|value="140\.8"/);
  assert.match(vista, /let ocupado = false/);
  assert.match(vista, /conBloqueo/);
  assert.match(vista, /aria-busy/);
  assert.match(vista, /confirmarOperacion/);
  assert.match(vista, /descargarRecibo/);
});

test("la hoja de estilos conserva densidad, foco visible y adaptación móvil", async () => {
  const css = await readFile(new URL("dietas.css", import.meta.url), "utf8");
  assert.match(css, /:focus-visible/);
  assert.match(css, /@media \(max-width: 1180px\)/);
  assert.match(css, /@media \(max-width: 720px\)/);
  assert.match(css, /@media \(max-width: 420px\)/);
  assert.match(css, /overflow-x|tabla-contenedor/);
  // Las columnas de escritorio se reparten el ancho real del shell, incluso
  // entre el breakpoint y 1440 px o al aumentar el texto al 125 %.
  const rejillaTrabajo = css.match(/\.modulo-dietas \.dietas-espacio-trabajo\s*\{([^}]+)\}/)[1];
  assert.match(rejillaTrabajo, /grid-template-columns: minmax\(0, 1\.45fr\) minmax\(0, 0\.85fr\)/);
  assert.doesNotMatch(rejillaTrabajo, /minmax\(\d+(?:\.\d+)?(?:rem|px)/);
  // A 390 px la tabla conserva su desplazamiento local y las cabeceras,
  // chips y selects no pueden ensanchar el documento.
  assert.match(css, /\.modulo-dietas > \*/);
  assert.match(css, /\.estado-chip[\s\S]*?overflow-wrap: anywhere/);
  assert.match(css, /\.modulo-dietas select[\s\S]*?inline-size: 100%/);
  assert.match(css, /\.tabla-contenedor[\s\S]*?overflow-x: auto/);
  assert.match(css, /prefers-reduced-motion/);
  assert.doesNotMatch(css, /font-family\s*:/);
});

test("la tabla de comisiones mantiene visible el estado sin ocultar la ruta", () => {
  const html = renderizarDietas(presentador().obtenerModelo());
  assert.match(html, /data-tabla-prioritaria="estado"/);
  for (const columna of ["referencia", "fecha", "ruta", "kilometros", "total", "estado"]) {
    assert.match(html, new RegExp(`data-columna="${columna}"`));
  }
  assert.match(html, /tabla-datos--dietas/);
  assert.match(html, /tabindex="0" role="region"/);
  assert.equal((html.match(/<tr data-seleccionada="true">/gu) || []).length, 1);
  assert.match(html, /<tr data-seleccionada="true">[\s\S]*?aria-current="true"/u);
});
