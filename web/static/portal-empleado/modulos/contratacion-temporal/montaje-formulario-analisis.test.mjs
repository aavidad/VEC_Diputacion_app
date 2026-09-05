import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { crearExpedienteContratacionTemporalPresentacion } from "./datos-presentacion.js";
import { CAPACIDAD_CREAR_SOLICITUD } from "./contrato.js";
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

  getAll(nombre) {
    const valor = this.get(nombre);
    if (valor === null) return [];
    return Array.isArray(valor) ? valor : [valor];
  }

  *entries() {
    yield* Object.entries(this.valores);
  }
}

globalThis.FormData = FormDataFalso;
test.after(() => { globalThis.FormData = FORM_DATA_ORIGINAL; });

function crearCatalogos() {
  return {
    modalidades: [
      { clave: "sustitucion", etiqueta: "Sustitución" },
      { clave: "vacante", etiqueta: "Vacante" },
      { clave: "acumulacion_tareas", etiqueta: "Acumulación de tareas" },
      { clave: "programa", etiqueta: "Programa temporal" },
      { clave: "relevo", etiqueta: "Contrato de relevo" },
    ],
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
    modalidad_clave: "sustitucion",
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
    modalidad_clave: "sustitucion",
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

function crearDiferida() {
  let resolver;
  let rechazar;
  const promesa = new Promise((resolve, reject) => {
    resolver = resolve;
    rechazar = reject;
  });
  return { promesa, resolver, rechazar };
}

function crearErrorCliente(resultadoIndeterminado, codigo = "conflicto") {
  const error = new Error("detalle privado no presentable");
  error.codigo = codigo;
  error.resultadoIndeterminado = resultadoIndeterminado;
  return error;
}

function crearRecibo(expediente) {
  return {
    esquema: "vec.contratacion-temporal.recibo-analisis-rrhh.v1",
    operacion: "registrar",
    expediente_ref: expediente.expediente_ref,
    version_resultante: expediente.version + 1,
    recibo_ref: "recibo:opaco:analisis:001",
    confirmada_en: "2026-08-21T22:00:00Z",
  };
}

async function esperarTransmision() {
  await Promise.resolve();
  await Promise.resolve();
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

function crearPresentador(estadoInicial, fallarCarga = false) {
  let estado = estadoInicial;
  let desmontajes = 0;
  return {
    obtenerEstado() { return estado; },
    async cargar() {
      if (fallarCarga) throw new Error("listado temporalmente no disponible");
      return estado;
    },
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
  let limpiezas = 0;
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
    replaceChildren() {
      limpiezas += 1;
      this.innerHTML = "";
    },
    obtenerLimpiezas() { return limpiezas; },
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

function crearContenedorAlta() {
  const eventos = new Map();
  return {
    innerHTML: "",
    eventos,
    addEventListener(tipo, manejador) { eventos.set(tipo, manejador); },
    removeEventListener(tipo, manejador) {
      if (eventos.get(tipo) === manejador) eventos.delete(tipo);
    },
    contains() { return true; },
    querySelector() { return { focus() {}, scrollIntoView() {} }; },
    enviar(valores) {
      const formulario = {
        valores,
        closest(selector) {
          return selector === "[data-ct-form]" ? this : null;
        },
      };
      return eventos.get("submit")({ target: formulario, preventDefault() {} });
    },
    confirmar() {
      const control = {
        dataset: { ctAccion: "confirmar" },
        closest(selector) {
          if (selector === "[data-ct-enfocar]") return null;
          return selector === "[data-ct-accion]" ? this : null;
        },
      };
      return eventos.get("click")({ target: control, preventDefault() {} });
    },
  };
}

function crearRaizModulo() {
  const eventos = new Map();
  const atributos = new Map();
  let html = "";
  let contenedorAlta = null;
  let contenedorAnalisis = null;
  let montajesAnalisis = 0;
  let controles = [];
  const raiz = {
    get innerHTML() { return html; },
    set innerHTML(valor) {
      html = valor;
      contenedorAlta = valor.includes("data-ct-exp-alta")
        ? crearContenedorAlta() : null;
      contenedorAnalisis = valor.includes("data-ct-exp-analisis")
        ? crearContenedorAnalisis() : null;
      if (contenedorAnalisis) montajesAnalisis += 1;
      controles = Array.from({ length: 3 }, () => {
        const propios = new Map();
        return {
          disabled: false,
          getAttribute(nombre) { return propios.get(nombre) ?? null; },
          setAttribute(nombre, contenido) { propios.set(nombre, contenido); },
          removeAttribute(nombre) { propios.delete(nombre); },
        };
      });
    },
    addEventListener(tipo, manejador) { eventos.set(tipo, manejador); },
    removeEventListener(tipo, manejador) {
      if (eventos.get(tipo) === manejador) eventos.delete(tipo);
    },
    contains() { return true; },
    getAttribute(nombre) { return atributos.get(nombre) ?? null; },
    setAttribute(nombre, contenido) { atributos.set(nombre, contenido); },
    removeAttribute(nombre) { atributos.delete(nombre); },
    querySelectorAll() { return controles; },
    querySelector(selector) {
      if (selector === "[data-ct-exp-alta]") return contenedorAlta;
      if (selector === "[data-ct-exp-analisis]") return contenedorAnalisis;
      return { focus() {}, scrollIntoView() {} };
    },
  };
  return {
    raiz,
    eventos,
    obtenerAlta() { return contenedorAlta; },
    obtenerAnalisis() { return contenedorAnalisis; },
    obtenerMontajesAnalisis() { return montajesAnalisis; },
    obtenerControles() { return controles; },
    obtenerAtributo(nombre) { return atributos.get(nombre) ?? null; },
    async cambiarVista(vista) {
      const control = {
        dataset: { ctExpVista: vista },
        closest(selector) {
          return selector === "[data-ct-exp-vista]" ? this : null;
        },
      };
      await eventos.get("click")({ target: control, preventDefault() {} });
    },
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
  alta = null,
  fallarCarga = false,
} = {}) {
  const raiz = crearRaizModulo();
  const presentador = crearPresentador(
    estado ?? crearEstado(expediente, tareaRef),
    fallarCarga,
  );
  const modulo = await montarModuloContratacionTemporal({
    raiz: raiz.raiz,
    presentador,
    alta,
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
      return Promise.resolve(crearRecibo(expediente));
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
  assert.match(formulario.innerHTML, /value="sustitucion" selected/u);
  for (const modalidad of [
    "sustitucion", "vacante", "acumulacion_tareas", "programa", "relevo",
  ]) assert.match(formulario.innerHTML, new RegExp(`value="${modalidad}"`, "u"));
  assert.doesNotMatch(escenario.raiz.raiz.innerHTML, /data-ct-exp-tarea-form/u);
  await formulario.enviar();
  assert.equal(solicitudes.length, 1);
  assert.equal(solicitudes[0].expediente_ref, expediente.expediente_ref);
  assert.equal(solicitudes[0].version_esperada, expediente.version);
  assert.equal(solicitudes[0].artefacto_ref, "artefacto:opaco:001");
  escenario.modulo.desmontar();
});

test("el detalle sin tareas solo expone análisis para solicitud v1 en curso", async () => {
  const { expediente, tareaRef } = crearExpediente();
  const expedienteSinTareas = { ...expediente, version: 1, tareas: [] };
  const cliente = { registrarAnalisis() { return Promise.resolve(crearRecibo(expedienteSinTareas)); } };
  const resumen = {
    expediente_ref: expediente.expediente_ref,
    fase_clave: "solicitud",
    estado_clave: "en_curso",
    version: expedienteSinTareas.version,
  };
  const casos = [
    ["válido", {}, true],
    ["fase avanzada", { cuadro: { demostracion: false, expedientes: [{ ...resumen, fase_clave: "fiscalizacion" }] } }, false],
    ["versión avanzada", {
      expediente: { ...expedienteSinTareas, version: 2 },
      cuadro: { demostracion: false, expedientes: [{ ...resumen, version: 2 }] },
    }, false],
    ["resumen desfasado", { cuadro: { demostracion: false, expedientes: [{ ...resumen, version: 2 }] } }, false],
    ["estado avanzado", { cuadro: { demostracion: false, expedientes: [{ ...resumen, estado_clave: "completado" }] } }, false],
  ];
  for (const [nombre, cambios, debeMontar] of casos) {
    const expedienteCaso = cambios.expediente ?? expedienteSinTareas;
    const estado = crearEstado(expedienteCaso, tareaRef, {
      cuadro: { demostracion: false, expedientes: [resumen] },
      ...cambios,
    });
    const escenario = await montarEscenario({
      expediente: expedienteCaso,
      tareaRef,
      estado,
      analisis: crearComposicion(cliente),
    });
    assert.equal(Boolean(escenario.raiz.obtenerAnalisis()), debeMontar, nombre);
    escenario.modulo.desmontar();
  }
});

test("el recibo de alta monta el análisis real sin perderse si falla el listado", async () => {
  const { expediente, tareaRef } = crearExpediente();
  const reciboAlta = {
    expediente_ref: "expediente:alta:analisis:001",
    numero_visible: "2026/CT-9001",
    version: 1,
    recibo_ref: "recibo:alta:analisis:001",
    confirmada_en: "2026-09-04T08:15:00Z",
  };
  const solicitudesAnalisis = [];
  const escenario = await montarEscenario({
    expediente,
    tareaRef,
    estado: crearEstado(expediente, tareaRef, { vista: "alta" }),
    fallarCarga: true,
    alta: {
      catalogos: {
        esquema: "vec.contratacion_temporal.catalogos_alta.v1",
        centros: [{
          referencia: "centro:sintetico:001",
          etiqueta: "Centro sintético",
          contactos: [{
            referencia: "contacto:sintetico:001",
            etiqueta: "Contacto sintético",
          }],
        }],
        categorias: [{
          referencia: "categoria:rrhh:001",
          etiqueta: "Técnica o técnico superior",
          grupos_subgrupos: [{ clave: "A1", etiqueta: "A1" }],
        }],
        motivos: [{ clave: "sustitucion", etiqueta: "Sustitución" }],
        documentos: [{
          referencia: "documento:sintetico:001",
          etiqueta: "Retención sintética",
        }],
      },
      capacidad: CAPACIDAD_CREAR_SOLICITUD,
      ejecutor: async () => reciboAlta,
      generarClaveIdempotencia: () => "12345678-1234-4abc-8def-1234567890ab",
    },
    analisis: crearComposicion({
      registrarAnalisis(solicitud) {
        solicitudesAnalisis.push(solicitud);
        return Promise.resolve({
          esquema: "vec.contratacion-temporal.recibo-analisis-rrhh.v1",
          operacion: "registrar",
          expediente_ref: reciboAlta.expediente_ref,
          version_resultante: 2,
          recibo_ref: "recibo:analisis:alta:001",
          confirmada_en: "2026-09-04T08:16:00Z",
        });
      },
    }, { analisisInicial: null }),
  });
  const alta = escenario.raiz.obtenerAlta();
  alta.enviar({
    centro_ref: "centro:sintetico:001",
    contacto_ref: "contacto:sintetico:001",
    categoria_ref: "categoria:rrhh:001",
    grupo_subgrupo: "A1",
    motivo_clave: "sustitucion",
    detalle: "Necesidad temporal sintética.",
    inicio: "2026-09-10",
    fin: "2027-03-10",
    rc_existe: "si",
    rc_numero: "RC-2026-9001",
    rc_fecha: "2026-09-04",
    rc_importe: "12500,00",
    rc_documento_ref: "documento:sintetico:001",
    documentos_adjuntos: ["documento:sintetico:001"],
    observaciones: "Datos exclusivamente sintéticos.",
  });
  await alta.confirmar();

  assert.match(alta.innerHTML, /data-ct-recibo/u);
  assert.match(alta.innerHTML, /recibo:alta:analisis:001/u);
  const formularioAnalisis = escenario.raiz.obtenerAnalisis();
  assert.match(formularioAnalisis.innerHTML, /data-ct-analisis-form/u);
  await formularioAnalisis.enviar();
  assert.equal(solicitudesAnalisis.length, 1);
  assert.equal(
    solicitudesAnalisis[0].expediente_ref,
    reciboAlta.expediente_ref,
  );
  assert.equal(solicitudesAnalisis[0].version_esperada, reciboAlta.version);
  assert.equal(solicitudesAnalisis[0].artefacto_ref, "artefacto:opaco:001");
  assert.match(formularioAnalisis.innerHTML, /recibo:analisis:alta:001/u);
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

test("un vuelo bloquea navegación y repintado, conserva el formulario y reintenta con la misma clave", async () => {
  const { expediente, tareaRef } = crearExpediente();
  const vuelos = [crearDiferida(), crearDiferida()];
  const solicitudes = [];
  const signals = [];
  const cliente = Object.freeze({
    registrarAnalisis(solicitud, opciones) {
      solicitudes.push(solicitud);
      signals.push(opciones.signal);
      return vuelos[solicitudes.length - 1].promesa;
    },
  });
  const escenario = await montarEscenario({
    expediente,
    tareaRef,
    analisis: crearComposicion(cliente),
  });
  const primerFormulario = escenario.raiz.obtenerAnalisis();
  const envio = primerFormulario.enviar();
  await esperarTransmision();
  const otraTarea = expediente.tareas.find(({ tarea_ref: referencia }) => referencia !== tareaRef);

  assert.equal(primerFormulario.eventos.size, 3);
  await escenario.raiz.seleccionarTarea(otraTarea.tarea_ref);
  await escenario.raiz.cambiarVista("cuadro");
  const mismoEnvio = primerFormulario.enviar();
  assert.strictEqual(mismoEnvio, envio);
  assert.strictEqual(escenario.raiz.obtenerAnalisis(), primerFormulario);
  assert.equal(escenario.raiz.obtenerMontajesAnalisis(), 1);
  assert.equal(escenario.presentador.obtenerEstado().vista, "expediente");
  assert.equal(escenario.presentador.obtenerEstado().tarea_ref, tareaRef);
  assert.equal(solicitudes.length, 1);
  assert.equal(signals[0].aborted, false);
  assert.deepEqual(Object.keys(cliente), ["registrarAnalisis"]);
  assert.ok(escenario.raiz.obtenerControles().every(({ disabled }) => disabled));
  assert.equal(escenario.raiz.obtenerAtributo("aria-busy"), "true");

  vuelos[0].rechazar(crearErrorCliente(false));
  await envio;
  assert.strictEqual(escenario.raiz.obtenerAnalisis(), primerFormulario);
  assert.match(primerFormulario.innerHTML, /data-ct-analisis-form/u);
  assert.equal(escenario.raiz.obtenerAtributo("aria-busy"), null);

  const reintento = primerFormulario.enviar();
  await esperarTransmision();
  assert.equal(solicitudes.length, 2);
  assert.equal(
    solicitudes[1].clave_idempotencia,
    solicitudes[0].clave_idempotencia,
  );
  vuelos[1].resolver(crearRecibo(expediente));
  await reintento;

  escenario.modulo.desmontar();
});

test("una respuesta indeterminada conserva el bloqueo sin aborto, remontaje ni segundo envío", async () => {
  const { expediente, tareaRef } = crearExpediente();
  let llamadas = 0;
  let signal;
  const cliente = {
    registrarAnalisis(_solicitud, opciones) {
      llamadas += 1;
      signal = opciones.signal;
      return Promise.reject(crearErrorCliente(true, "resultado_indeterminado"));
    },
  };
  const escenario = await montarEscenario({
    expediente,
    tareaRef,
    analisis: crearComposicion(cliente),
  });
  const formulario = escenario.raiz.obtenerAnalisis();
  await formulario.enviar();
  const otraTarea = expediente.tareas.find(({ tarea_ref: referencia }) => referencia !== tareaRef);

  assert.match(formulario.innerHTML, /data-ct-analisis-indeterminado/u);
  await escenario.raiz.seleccionarTarea(otraTarea.tarea_ref);
  await escenario.raiz.cambiarVista("documentos");
  await formulario.enviar();
  assert.strictEqual(escenario.raiz.obtenerAnalisis(), formulario);
  assert.equal(escenario.raiz.obtenerMontajesAnalisis(), 1);
  assert.equal(escenario.presentador.obtenerEstado().tarea_ref, tareaRef);
  assert.equal(llamadas, 1);
  assert.equal(signal.aborted, false);
  assert.equal(escenario.raiz.obtenerAtributo("aria-busy"), null);

  escenario.modulo.desmontar();
});

test("el éxito conserva el recibo visible y no reenvía desde la versión obsoleta", async () => {
  const { expediente, tareaRef } = crearExpediente();
  let llamadas = 0;
  const cliente = {
    registrarAnalisis() {
      llamadas += 1;
      return Promise.resolve(crearRecibo(expediente));
    },
  };
  const escenario = await montarEscenario({
    expediente,
    tareaRef,
    analisis: crearComposicion(cliente),
  });
  const formulario = escenario.raiz.obtenerAnalisis();
  await formulario.enviar();

  assert.match(formulario.innerHTML, /data-ct-analisis-recibo/u);
  assert.match(formulario.innerHTML, /recibo:opaco:analisis:001/u);
  await escenario.raiz.cambiarVista("cuadro");
  await formulario.enviar();
  assert.strictEqual(escenario.raiz.obtenerAnalisis(), formulario);
  assert.equal(escenario.raiz.obtenerMontajesAnalisis(), 1);
  assert.equal(escenario.presentador.obtenerEstado().vista, "expediente");
  assert.equal(llamadas, 1);
  assert.equal(escenario.raiz.obtenerAtributo("aria-busy"), null);

  escenario.modulo.desmontar();
});

test("el desmontaje explícito aborta y limpia exactamente una vez sin presentar otro formulario", async () => {
  const { expediente, tareaRef } = crearExpediente();
  let signal;
  let abortos = 0;
  const cliente = {
    registrarAnalisis(_solicitud, opciones) {
      signal = opciones.signal;
      return new Promise((_resolve, reject) => {
        signal.addEventListener("abort", () => {
          abortos += 1;
          reject(crearErrorCliente(true, "operacion_abortada"));
        }, { once: true });
      });
    },
  };
  const escenario = await montarEscenario({
    expediente,
    tareaRef,
    analisis: crearComposicion(cliente),
  });
  const formulario = escenario.raiz.obtenerAnalisis();
  const envio = formulario.enviar();
  await esperarTransmision();

  escenario.modulo.desmontar();
  escenario.modulo.desmontar();
  await envio;
  assert.equal(signal.aborted, true);
  assert.equal(abortos, 1);
  assert.deepEqual([...formulario.eventos.keys()], []);
  assert.deepEqual(formulario.retirados.sort(), ["change", "click", "submit"]);
  assert.equal(formulario.obtenerLimpiezas(), 1);
  assert.equal(escenario.raiz.obtenerMontajesAnalisis(), 1);
  assert.ok(escenario.raiz.obtenerControles().every(({ disabled }) => !disabled));
  assert.equal(escenario.raiz.obtenerAtributo("aria-busy"), null);
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
