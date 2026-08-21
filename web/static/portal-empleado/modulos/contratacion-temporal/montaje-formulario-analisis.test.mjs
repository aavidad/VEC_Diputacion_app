import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { crearExpedienteContratacionTemporalPresentacion } from "./datos-presentacion.js";
import {
  CAPACIDADES_CONTRATACION_TEMPORAL as CAP,
  validarExpedienteContratacionTemporal,
} from "./contrato-expedientes.js";
import { montarModuloContratacionTemporal } from "./vista-expedientes.js";

const HUELLA = "a".repeat(64);
const FORM_DATA_ORIGINAL = globalThis.FormData;

class FormDataFalso {
  constructor(formulario) {
    this.valores = formulario.valores;
  }

  get(nombre) {
    return Object.hasOwn(this.valores, nombre) ? this.valores[nombre] : null;
  }

  *entries() {
    yield* Object.entries(this.valores);
  }
}

globalThis.FormData = FormDataFalso;
test.after(() => { globalThis.FormData = FORM_DATA_ORIGINAL; });

function crearCatalogos() {
  return {
    modalidades: [{ clave: "interinidad", etiqueta: "Interinidad" }],
    categorias: [{
      referencia: "categoria:rrhh:001",
      etiqueta: "Técnica o técnico superior",
      grupos_subgrupos: [{ clave: "A1", etiqueta: "A1" }],
    }],
    causas: [{ clave: "sustitucion", etiqueta: "Sustitución" }],
    entradas_rc: [{
      referencia: "entrada-rc:opaca:001",
      huella_sha256: HUELLA,
      etiqueta: "Retención preparada 001",
    }],
    motivos_rectificacion: [{
      clave: "correccion_datos",
      etiqueta: "Corrección de datos",
    }],
  };
}

function crearAnalisisInicial() {
  return {
    modalidad_clave: "interinidad",
    categoria_ref: "categoria:rrhh:001",
    grupo_subgrupo: "A1",
    causa_clave: "sustitucion",
    periodo: { inicio: "2026-09-01T00:00:00Z", fin: "2027-08-31T00:00:00Z" },
    porcentaje_jornada: 10000,
    entrada_rc: { referencia: "entrada-rc:opaca:001", huella_sha256: HUELLA },
  };
}

function crearValoresFormulario() {
  return {
    modalidad_clave: "interinidad",
    categoria_ref: "categoria:rrhh:001",
    grupo_subgrupo: "A1",
    causa_clave: "sustitucion",
    inicio: "2026-09-01",
    fin: "2027-08-31",
    porcentaje_jornada: "10000",
    entrada_rc_referencia: "entrada-rc:opaca:001",
    motivo_rectificacion_clave: "",
  };
}

function crearExpediente({ disponible = true, capacidad = CAP.analizar } = {}) {
  const entrada = structuredClone(crearExpedienteContratacionTemporalPresentacion());
  entrada.version = 7;
  const tarea = entrada.tareas.find((candidata) => candidata.acciones.some(
    (accion) => accion.capacidad === CAP.analizar,
  ));
  const accion = tarea.acciones.find((candidata) => candidata.capacidad === CAP.analizar);
  accion.capacidad = capacidad;
  accion.disponible = disponible;
  accion.motivo_no_disponible = disponible ? "" : "Actuación no disponible.";
  return {
    expediente: validarExpedienteContratacionTemporal(entrada),
    tareaRef: tarea.tarea_ref,
  };
}

function crearEstado(expediente, tareaRef, sobrescrituras = {}) {
  return {
    vista: "expediente",
    carga: "listo",
    cuadro: null,
    expediente,
    documentos: null,
    auditoria: null,
    expediente_ref: expediente.expediente_ref,
    tarea_ref: tareaRef,
    filtros: { texto: "", estado: "", fase: "" },
    ocupado: false,
    actualizacion_pendiente: false,
    resultado_indeterminado: false,
    recibo: null,
    mensaje_clave: "estado_expediente_listo",
    tipo_mensaje: "informacion",
    ...sobrescrituras,
  };
}

function crearPresentador(estadoInicial) {
  let estado = estadoInicial;
  let desmontajes = 0;
  return {
    obtenerEstado() { return estado; },
    async cargar() { return estado; },
    cambiarVista(vista) { estado = { ...estado, vista }; },
    seleccionarTarea(tareaRef) { estado = { ...estado, vista: "expediente", tarea_ref: tareaRef }; },
    cancelar() {},
    desmontar() { desmontajes += 1; },
    obtenerDesmontajes() { return desmontajes; },
  };
}

function crearContenedorAnalisis() {
  const eventos = new Map();
  const retirados = [];
  return {
    innerHTML: "",
    eventos,
    retirados,
    addEventListener(tipo, manejador) { eventos.set(tipo, manejador); },
    removeEventListener(tipo, manejador) {
      if (eventos.get(tipo) === manejador) eventos.delete(tipo);
      retirados.push(tipo);
    },
    contains() { return true; },
    querySelector() { return { focus() {}, scrollIntoView() {} }; },
    replaceChildren() { this.innerHTML = ""; },
    enviar(valores = crearValoresFormulario()) {
      const formulario = {
        valores,
        closest(selector) {
          return selector === "[data-ct-analisis-form]" ? this : null;
        },
      };
      return eventos.get("submit")({ target: formulario, preventDefault() {} });
    },
  };
}

function crearRaizModulo() {
  const eventos = new Map();
  let html = "";
  let contenedorAnalisis = null;
  const raiz = {
    get innerHTML() { return html; },
    set innerHTML(valor) {
      html = valor;
      contenedorAnalisis = valor.includes("data-ct-exp-analisis")
        ? crearContenedorAnalisis() : null;
    },
    addEventListener(tipo, manejador) { eventos.set(tipo, manejador); },
    removeEventListener(tipo, manejador) {
      if (eventos.get(tipo) === manejador) eventos.delete(tipo);
    },
    contains() { return true; },
    querySelector(selector) {
      if (selector === "[data-ct-exp-analisis]") return contenedorAnalisis;
      return { focus() {}, scrollIntoView() {} };
    },
  };
  return {
    raiz,
    eventos,
    obtenerAnalisis() { return contenedorAnalisis; },
    async seleccionarTarea(tareaRef) {
      const control = {
        dataset: { ctExpTarea: tareaRef },
        closest(selector) {
          return selector === "[data-ct-exp-tarea]" ? this : null;
        },
      };
      await eventos.get("click")({ target: control, preventDefault() {} });
    },
  };
}

function crearComposicion(cliente, sobrescrituras = {}) {
  return {
    cliente,
    catalogos: crearCatalogos(),
    contexto: { operacion: "registrar", artefacto_ref: "artefacto:opaco:001" },
    analisisInicial: crearAnalisisInicial(),
    ...sobrescrituras,
  };
}

async function montarEscenario({
  expediente,
  tareaRef,
  estado = null,
  analisis = null,
} = {}) {
  const raiz = crearRaizModulo();
  const presentador = crearPresentador(estado ?? crearEstado(expediente, tareaRef));
  const modulo = await montarModuloContratacionTemporal({
    raiz: raiz.raiz,
    presentador,
    analisis,
  });
  return { raiz, presentador, modulo };
}

test("la capacidad nominal disponible monta el formulario con expediente y versión validados", async () => {
  const { expediente, tareaRef } = crearExpediente();
  const solicitudes = [];
  const cliente = {
    registrarAnalisis(solicitud) {
      solicitudes.push(solicitud);
      return Promise.resolve({
        esquema: "vec.contratacion-temporal.recibo-analisis-rrhh.v1",
        operacion: "registrar",
        expediente_ref: expediente.expediente_ref,
        version_resultante: expediente.version + 1,
        recibo_ref: "recibo:opaco:analisis:001",
        confirmada_en: "2026-08-21T22:00:00Z",
      });
    },
  };
  const escenario = await montarEscenario({
    expediente,
    tareaRef,
    analisis: crearComposicion(cliente),
  });
  const formulario = escenario.raiz.obtenerAnalisis();

  assert.ok(formulario);
  assert.match(formulario.innerHTML, /data-ct-analisis-form/u);
  assert.match(formulario.innerHTML, /value="interinidad" selected/u);
  assert.doesNotMatch(escenario.raiz.raiz.innerHTML, /data-ct-exp-tarea-form/u);
  await formulario.enviar();
  assert.equal(solicitudes.length, 1);
  assert.equal(solicitudes[0].expediente_ref, expediente.expediente_ref);
  assert.equal(solicitudes[0].version_esperada, expediente.version);
  assert.equal(solicitudes[0].artefacto_ref, "artefacto:opaco:001");
  escenario.modulo.desmontar();
});

test("sin capacidad exacta, disponibilidad o inyección completa conserva la vista genérica", async () => {
  const casos = [
    { expediente: crearExpediente({ disponible: false }), analisis: crearComposicion({ registrarAnalisis() {} }) },
    { expediente: crearExpediente({ capacidad: CAP.decidirCobertura }), analisis: crearComposicion({ registrarAnalisis() {} }) },
    { expediente: crearExpediente(), analisis: null },
    {
      expediente: crearExpediente(),
      analisis: {
        cliente: { registrarAnalisis() {} },
        catalogos: crearCatalogos(),
        contexto: { operacion: "registrar", artefacto_ref: "artefacto:opaco:001" },
      },
    },
  ];
  for (const caso of casos) {
    const escenario = await montarEscenario({
      expediente: caso.expediente.expediente,
      tareaRef: caso.expediente.tareaRef,
      analisis: caso.analisis,
    });
    assert.equal(escenario.raiz.obtenerAnalisis(), null);
    assert.match(escenario.raiz.raiz.innerHTML, /data-ct-exp-tarea-form/u);
    escenario.modulo.desmontar();
  }
});

test("la composición cerrada rechaza identidad y coordenadas autoritativas inyectadas", async () => {
  const { expediente, tareaRef } = crearExpediente();
  const base = crearComposicion({ registrarAnalisis() {} });
  await assert.rejects(montarEscenario({
    expediente,
    tareaRef,
    analisis: { ...base, identidad: "actor:inventado" },
  }), /composición del análisis no válida/u);
  await assert.rejects(montarEscenario({
    expediente,
    tareaRef,
    analisis: {
      ...base,
      contexto: {
        ...base.contexto,
        expediente_ref: "expediente:inyectado:999",
        version_esperada: 999,
      },
    },
  }), /contexto de composición del análisis no válida/u);
});

test("un resultado indeterminado del expediente no se transforma en un formulario reenviable", async () => {
  const { expediente, tareaRef } = crearExpediente();
  const escenario = await montarEscenario({
    expediente,
    tareaRef,
    estado: crearEstado(expediente, tareaRef, {
      actualizacion_pendiente: true,
      resultado_indeterminado: true,
    }),
    analisis: crearComposicion({ registrarAnalisis() {} }),
  });
  assert.equal(escenario.raiz.obtenerAnalisis(), null);
  assert.match(escenario.raiz.raiz.innerHTML, /data-ct-exp-tarea-form/u);
  assert.match(escenario.raiz.raiz.innerHTML, /disabled/u);
  escenario.modulo.desmontar();
});

test("repintar, cambiar de tarea y desmontar retiran listeners y abortan el único vuelo", async () => {
  const { expediente, tareaRef } = crearExpediente();
  let signal;
  const cliente = {
    registrarAnalisis(_solicitud, opciones) {
      signal = opciones.signal;
      return new Promise((_resolve, reject) => {
        signal.addEventListener("abort", () => reject(new Error("vuelo retirado")), { once: true });
      });
    },
  };
  const escenario = await montarEscenario({
    expediente,
    tareaRef,
    analisis: crearComposicion(cliente),
  });
  const primerFormulario = escenario.raiz.obtenerAnalisis();
  const envio = primerFormulario.enviar();
  await Promise.resolve();

  assert.equal(primerFormulario.eventos.size, 3);
  await escenario.raiz.seleccionarTarea(tareaRef);
  await envio;
  assert.equal(signal.aborted, true);
  assert.deepEqual([...primerFormulario.eventos.keys()], []);
  assert.deepEqual(primerFormulario.retirados.sort(), ["change", "click", "submit"]);
  const segundoFormulario = escenario.raiz.obtenerAnalisis();
  assert.notEqual(segundoFormulario, primerFormulario);
  assert.equal(segundoFormulario.eventos.size, 3);

  escenario.modulo.desmontar();
  assert.deepEqual([...segundoFormulario.eventos.keys()], []);
  assert.equal(escenario.presentador.obtenerDesmontajes(), 1);
});

test("el montaje no hardcodea tarea o etiqueta ni incorpora autoridad de navegador", async () => {
  const [componentes, vista] = await Promise.all([
    readFile(new URL("./componentes-expedientes.js", import.meta.url), "utf8"),
    readFile(new URL("./vista-expedientes.js", import.meta.url), "utf8"),
  ]);
  const fuente = `${componentes}\n${vista}`;
  assert.doesNotMatch(fuente, /tarea-analisis|Análisis de RRHH/u);
  assert.match(fuente, /CAPACIDADES_CONTRATACION_TEMPORAL\.analizar/u);
  assert.doesNotMatch(
    fuente,
    /document\.cookie|localStorage|sessionStorage|indexedDB|\b(?:identidad|organizacion|perfil|autorizacion)\s*:/u,
  );
});
