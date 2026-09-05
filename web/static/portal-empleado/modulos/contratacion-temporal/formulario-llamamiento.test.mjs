import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { montarFormularioLlamamiento } from "./formulario-llamamiento.js";
import { crearClienteHTTPContratacionTemporal } from "./cliente-http.js";
import {
  montarModuloContratacionTemporal, renderizarModuloContratacionTemporal,
} from "./vista-expedientes.js";

const CLAVE = "123e4567-e89b-42d3-a456-426614174000";
const EXPEDIENTE = "expediente:ct:sintetico:001";
const seleccion = () => ({
  expediente_ref: EXPEDIENTE, version_esperada: "6", clave_idempotencia: CLAVE,
});
const recibo = {
  esquema: "vec.contratacion-temporal.recibo-seleccion-llamamiento.v1",
  estado: "confirmado", recibo_ref: "recibo:sintetico:001", confirmada_en: "2026-09-05T08:00:00Z",
  organizacion_ref: "organizacion:sintetica:001", llamamiento_ref: "llamamiento:sintetico:001",
  version_llamamiento: 1,
};
function raizPrueba() {
  const eventos = new Map();
  const borradores = {};
  const foco = [];
  const raiz = {
    innerHTML: "", eventos, borradores, foco,
    addEventListener: (tipo, fn) => eventos.set(tipo, fn),
    removeEventListener: (tipo) => eventos.delete(tipo),
    contains: () => true,
    replaceChildren() { this.innerHTML = ""; },
    querySelector(selector) {
      const tipo = selector.match(/^\[data-ct-llamamiento-form="(seleccion|comunicacion)"\]$/u)?.[1];
      if (tipo) return borradores[tipo] ?? null;
      if (selector === "[data-ct-llamamiento-comunicacion]") return { open: false };
      return { focus: () => foco.push(selector), scrollIntoView() {} };
    },
    preparar(tipo, valores) {
      const controles = Object.fromEntries(Object.entries(valores).map(
        ([nombre, value]) => [nombre, { value }],
      ));
      borradores[tipo] = {
        dataset: { ctLlamamientoForm: tipo },
        elements: { namedItem: (nombre) => controles[nombre] },
        closest: () => borradores[tipo],
      };
      return borradores[tipo];
    },
    enviar(tipo, valores) {
      const form = valores ? this.preparar(tipo, valores) : borradores[tipo];
      return eventos.get("submit")({ target: form, preventDefault() {} });
    },
  };
  return raiz;
}
function montar(raiz, cliente = {}, extras = {}) {
  return montarFormularioLlamamiento({
    raiz, cliente: { seleccionarLlamamiento: async () => recibo,
      registrarComunicacionLlamamiento: async () => {}, ...cliente },
    confirmarOperacion: () => true, ...extras,
  });
}

function estadoSeleccionado(expedienteRef = EXPEDIENTE) {
  return {
    vista: "expediente", carga: "listo", expediente_ref: expedienteRef,
    cuadro: { demostracion: false, expedientes: [
      { expediente_ref: expedienteRef, version: 6, fase_clave: "fiscalizacion" },
    ] },
    expediente: {
      demostracion: false, expediente_ref: expedienteRef, version: 6,
      numero_visible: "CT-SINTETICO-001", cabecera: [], fases: [], tareas: [],
    },
    tipo_mensaje: "informacion", mensaje_clave: "estado_expediente_listo",
    ocupado: false, actualizacion_pendiente: false, resultado_indeterminado: false,
  };
}

async function montarExpedienteSeleccionado(inicial) {
  let estado = inicial, html = "", formulario;
  let peticiones = 0;
  const eventos = new Map();
  const raiz = {
    addEventListener: (tipo, fn) => eventos.set(tipo, fn),
    removeEventListener: (tipo) => eventos.delete(tipo),
    contains: () => true,
    get innerHTML() { return html; },
    set innerHTML(valor) { html = valor; formulario = raizPrueba(); },
    querySelector: (selector) => selector === "[data-ct-exp-llamamiento]"
      && html.includes("data-ct-exp-llamamiento") ? formulario : null,
  };
  const modulo = await montarModuloContratacionTemporal({
    raiz,
    presentador: {
      obtenerEstado: () => estado,
      cargar: async () => estado,
      async seleccionarExpediente(referencia) { estado = estadoSeleccionado(referencia); },
    },
    llamamiento: { cliente: {
      seleccionarLlamamiento: async () => { peticiones += 1; return recibo; },
      registrarComunicacionLlamamiento: async () => { peticiones += 1; },
    } },
  });
  return {
    formulario: () => formulario,
    peticiones: () => peticiones,
    desmontar: modulo.desmontar,
    abrir: (referencia) => eventos.get("click")({
      target: { closest: (selector) => selector === "[data-ct-exp-abrir]"
        ? { dataset: { ctExpAbrir: referencia } } : null },
      preventDefault() {},
    }),
  };
}

test("el expediente fiscalizado seleccionado rellena el formulario existente sin POST ni clave automática", async () => {
  const montaje = await montarExpedienteSeleccionado(estadoSeleccionado());
  const html = montaje.formulario().innerHTML;
  assert.match(html, /Datos del expediente fiscalizado/u);
  assert.match(html, /id="ct-llamamiento-seleccion-expediente_ref"[^>]*value="expediente:ct:sintetico:001"/u);
  assert.match(html, /id="ct-llamamiento-seleccion-version_esperada"[^>]*value="6"/u);
  assert.match(html, /id="ct-llamamiento-seleccion-clave_idempotencia"[^>]*value=""/u);
  assert.equal(montaje.peticiones(), 0);
  montaje.desmontar();
});

test("no enlaza un detalle sin cargar, desfasado, de otro expediente o no fiscalizado", async () => {
  const casos = [
    (e) => { e.carga = "cargando"; },
    (e) => { e.carga = "error"; },
    (e) => { e.expediente = null; },
    (e) => { e.expediente_ref = "expediente:otro"; },
    (e) => { e.cuadro.expedientes[0].expediente_ref = "expediente:otro"; },
    (e) => { e.cuadro.expedientes[0].version = 5; },
    (e) => { e.cuadro.expedientes[0].fase_clave = "subsanacion_unidad"; },
    (e) => { e.actualizacion_pendiente = true; },
    (e) => { e.expediente.demostracion = true; },
  ];
  for (const modificar of casos) {
    const estado = estadoSeleccionado();
    modificar(estado);
    const montaje = await montarExpedienteSeleccionado(estado);
    assert.match(montaje.formulario().innerHTML,
      /id="ct-llamamiento-seleccion-expediente_ref"[^>]*value=""/u);
    assert.doesNotMatch(montaje.formulario().innerHTML, /Datos del expediente fiscalizado/u);
    assert.equal(montaje.peticiones(), 0);
    montaje.desmontar();
  }
});

test("abrir otro expediente no reutiliza referencias ni clave del formulario anterior", async () => {
  const montaje = await montarExpedienteSeleccionado(estadoSeleccionado());
  const anterior = montaje.formulario();
  anterior.preparar("seleccion", seleccion());
  await montaje.abrir("expediente:ct:sintetico:002");
  const actual = montaje.formulario();
  assert.notEqual(actual, anterior);
  assert.equal(anterior.eventos.size, 0);
  assert.match(actual.innerHTML, /id="ct-llamamiento-seleccion-expediente_ref"[^>]*value="expediente:ct:sintetico:002"/u);
  assert.match(actual.innerHTML, /id="ct-llamamiento-seleccion-clave_idempotencia"[^>]*value=""/u);
  assert.doesNotMatch(actual.innerHTML, /expediente:ct:sintetico:001/u);
  assert.equal(montaje.peticiones(), 0);
  montaje.desmontar();
});

test("manifiestos publican todos los recursos del llamamiento sin duplicados", async () => {
  const recursos = [
    "cliente-http-llamamiento.js", "contrato-llamamiento.js",
    "formulario-llamamiento.js", "i18n-llamamiento.js", "renderizado-llamamiento.js",
  ];
  for (const nombre of ["interno.manifest", "produccion.manifest"]) {
    const contenido = await readFile(new URL(`../../../../${nombre}`, import.meta.url), "utf8");
    const rutas = contenido.trim().split(/\r?\n/u);
    for (const recurso of recursos) {
      const ruta = `static/portal-empleado/modulos/contratacion-temporal/${recurso}`;
      assert.doesNotMatch(ruta, /presentacion|demo/iu, "No debe activar la exclusión de material de presentación");
      assert.equal(rutas.filter((entrada) => entrada === ruta).length, 1, `${nombre}: ${recurso}`);
      assert.ok((await readFile(new URL(`./${recurso}`, import.meta.url), "utf8")).length > 0);
    }
  }
});

test("enlaza fiscalización y exige confirmación sin inventar candidato ni autoridad", async () => {
  const raiz = raizPrueba();
  let llamadas = 0;
  const desmontar = montar(raiz, { seleccionarLlamamiento: () => { llamadas += 1; } },
    { confirmarOperacion: () => false });
  assert.equal(desmontar.actualizarContexto({ expediente_ref: EXPEDIENTE, version_esperada: 6 }), true);
  assert.match(raiz.innerHTML, /Datos del expediente fiscalizado/u);
  assert.match(raiz.innerHTML, /value="expediente:ct:sintetico:001"/u);
  await raiz.enviar("seleccion", seleccion());
  assert.equal(llamadas, 0);
  assert.doesNotMatch(raiz.innerHTML, /candidatura_ref|name="actor_ref"/u);
  desmontar();
  assert.equal(raiz.eventos.size, 0);
});

test("doble envío no duplica operación y el recibo minimizado abre comunicación", async () => {
  const raiz = raizPrueba();
  let resolver;
  const solicitudes = [];
  montar(raiz, {
    seleccionarLlamamiento: (solicitud) => {
      solicitudes.push(solicitud);
      return new Promise((resolve) => { resolver = resolve; });
    },
  });
  const primera = raiz.enviar("seleccion", seleccion());
  await raiz.enviar("seleccion", { ...seleccion(), clave_idempotencia: "otra" });
  assert.equal(solicitudes.length, 1);
  resolver(recibo);
  await primera;
  assert.match(raiz.innerHTML, /data-ct-llamamiento-recibo="seleccion"/u);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-comunicacion open/u);
  assert.match(raiz.innerHTML, /No expone identidad/u);
  assert.equal(solicitudes[0].version_esperada, 6);
});

test("respuesta perdida: recuperación usa petición congelada aunque cambien controles", async () => {
  const raiz = raizPrueba();
  const solicitudes = [];
  montar(raiz, { seleccionarLlamamiento: async (solicitud) => {
    solicitudes.push(solicitud);
    if (solicitudes.length === 1) throw new Error("red");
    return recibo;
  } });
  await raiz.enviar("seleccion", seleccion());
  assert.match(raiz.innerHTML, /Recuperar con los mismos datos/u);
  await raiz.enviar("seleccion", { ...seleccion(), expediente_ref: "expediente:distinto" });
  assert.deepEqual(solicitudes[0], solicitudes[1]);
});

test("tras desmontar permite recuperar mediante los inputs visibles, sin memoria web", async () => {
  const solicitudes = [];
  const cliente = { seleccionarLlamamiento: async (solicitud) => {
    solicitudes.push(solicitud);
    return recibo;
  } };
  const primera = raizPrueba();
  const cerrar = montar(primera, cliente);
  await primera.enviar("seleccion", seleccion());
  cerrar();
  const segunda = raizPrueba();
  montar(segunda, cliente);
  await segunda.enviar("seleccion", seleccion());
  assert.deepEqual(solicitudes[0], solicitudes[1]);
  assert.match(segunda.innerHTML, /Recibo verificado/u);
});

test("conflicto no reintentable mantiene datos y no permite otra escritura", async () => {
  const raiz = raizPrueba();
  let llamadas = 0;
  montar(raiz, { seleccionarLlamamiento: async () => {
    llamadas += 1;
    throw Object.assign(new Error("conflicto"), { codigo: "conflicto_no_reintentable" });
  } });
  await raiz.enviar("seleccion", seleccion());
  await raiz.enviar("seleccion", seleccion());
  assert.equal(llamadas, 1);
  assert.match(raiz.innerHTML, /requiere revisión del servidor/u);
});

test("comunicación exige sus referencias y muestra registro, no envío de correo", async () => {
  const raiz = raizPrueba();
  let solicitud;
  montar(raiz, { registrarComunicacionLlamamiento: async (entrada) => {
    solicitud = entrada;
    return {
      esquema: "vec.contratacion-temporal.registro-comunicacion-llamamiento.v1",
      estado_local: "confirmado", comunicacion_ref: "comunicacion:sintetica:001",
      recibo_ref: "recibo:comunicacion:001", auditoria_ref: "auditoria:sintetica:001",
      version_resultante: 2, respuesta_hasta: "2026-09-06T08:00:00Z",
    };
  } });
  await raiz.enviar("seleccion", seleccion());
  await raiz.enviar("comunicacion", { clave_idempotencia: CLAVE });
  assert.equal(solicitud.version_esperada, 1);
  assert.match(raiz.innerHTML, /Comunicación registrada/u);
  assert.match(raiz.innerHTML, /no acredita envío de correo real/u);
});

test("desmontar cancela la espera y una respuesta tardía no repinta", async () => {
  const raiz = raizPrueba();
  let resolver;
  let signal;
  const cerrar = montar(raiz, { seleccionarLlamamiento: (_, opciones) => {
    signal = opciones.signal;
    return new Promise((resolve) => { resolver = resolve; });
  } });
  const vuelo = raiz.enviar("seleccion", seleccion());
  cerrar();
  assert.equal(signal.aborted, true);
  resolver(recibo);
  await vuelo;
  assert.equal(raiz.innerHTML, "");
});

test("el llamamiento queda dentro del shell incluso con consultas temporalmente no disponibles", () => {
  const html = renderizarModuloContratacionTemporal({
    vista: "cuadro", carga: "error", cuadro: null, expediente: null,
    tipo_mensaje: "error", mensaje_clave: "estado_error_carga",
  }, { llamamientoDisponible: true });
  assert.match(html, /ct-exp-navegacion/u);
  assert.match(html, /data-ct-exp-llamamiento/u);
});

test("encadenado selección y replay autorrellenan comunicación local sin transcribir referencias", async () => {
  const claveComunicacion = "123e4567-e89b-42d3-a456-426614174001";
  const llamadas = [];
  for (const estadoHTTP of [201, 200]) {
    const raiz = raizPrueba();
    const cliente = crearClienteHTTPContratacionTemporal({
      fetchImpl: async (ruta, opciones) => {
        const entrada = JSON.parse(opciones.body);
        llamadas.push({ ruta, entrada });
        const data = ruta.endsWith("/seleccion") ? recibo : {
          esquema: "vec.contratacion-temporal.registro-comunicacion-llamamiento.v1",
          estado_local: estadoHTTP === 201 ? "registrada_localmente" : "replay_registrada_localmente",
          comunicacion_ref: "comunicacion:sintetica:001", recibo_ref: "recibo:comunicacion:001",
          auditoria_ref: "auditoria:sintetica:001", version_resultante: 2,
          registrada_en: "2026-09-05T08:05:00Z", intencion_envio_ref: "intencion:sintetica:001",
        };
        return new Response(JSON.stringify({ data }), {
          status: estadoHTTP, headers: { "Content-Type": "application/json; charset=utf-8" },
        });
      },
    });
    const cerrar = montar(raiz, cliente);
    assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="comunicacion"/u);
    await raiz.enviar("seleccion", seleccion());
    for (const [campo, valor] of Object.entries({
      organizacion_ref: recibo.organizacion_ref, expediente_ref: EXPEDIENTE,
      llamamiento_ref: recibo.llamamiento_ref, version_esperada: "1",
      prueba_entrega_ref: recibo.recibo_ref,
    })) {
      assert.match(raiz.innerHTML, new RegExp(
        'id="ct-llamamiento-comunicacion-' + campo + '"[^>]*value="' + valor + '"[^>]*readonly', "u",
      ));
    }
    // Ni siquiera al manipular controles se sustituyen los antecedentes del recibo.
    await raiz.enviar("comunicacion", {
      clave_idempotencia: claveComunicacion, organizacion_ref: "organizacion:inventada",
      expediente_ref: "expediente:inventado", llamamiento_ref: "llamamiento:inventado",
      version_esperada: "99", prueba_entrega_ref: "prueba:inventada",
    });
    assert.deepEqual(llamadas.at(-1).entrada, {
      clave_idempotencia: claveComunicacion,
      organizacion_ref: recibo.organizacion_ref, expediente_ref: EXPEDIENTE,
      llamamiento_ref: recibo.llamamiento_ref, version_esperada: 1,
      prueba_entrega_ref: recibo.recibo_ref,
    });
    assert.match(raiz.innerHTML, /data-ct-llamamiento-recibo="comunicacion"/u);
    assert.match(raiz.innerHTML, /Sin entrega acreditada/u);
    assert.doesNotMatch(raiz.innerHTML, /Plazo de respuesta registrado/u);
    cerrar();
  }
  assert.deepEqual(llamadas.slice(0, 2), llamadas.slice(2));
});
