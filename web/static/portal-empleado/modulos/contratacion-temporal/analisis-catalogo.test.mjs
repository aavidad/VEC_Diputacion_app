import assert from "node:assert/strict";
import test from "node:test";

import {
  periodoSuperaDuracionMaxima,
  validarConfiguracionAnalisis,
} from "./contrato-analisis.js";
import { montarFormularioAnalisisRRHH } from "./formulario-analisis.js";

// Opciones del análisis publicadas por el catálogo de reglas: modalidades sin
// lista fija, duración máxima por modalidad (aviso o bloqueo) y urgencia.

const HUELLA = "b".repeat(64);
const FORM_DATA_ORIGINAL = globalThis.FormData;

class FormDataFalso {
  constructor(formulario) { this.valores = formulario.valores; }

  get(nombre) { return Object.hasOwn(this.valores, nombre) ? this.valores[nombre] : null; }
}

globalThis.FormData = FormDataFalso;
test.after(() => { globalThis.FormData = FORM_DATA_ORIGINAL; });

function configuracion(extra = {}) {
  return {
    esquema: "vec.contratacion_temporal.configuracion_analisis.v1",
    artefacto_ref: "artefacto:analisis:desarrollo:v1",
    modalidades: [
      { clave: "vacante", etiqueta: "Vacante" },
      { clave: "acumulacion_tareas", etiqueta: "Acumulación de tareas" },
      { clave: "interinidad_programa", etiqueta: "Interinidad por programa" },
    ],
    categorias: [{
      referencia: "categoria:rrhh:001", etiqueta: "Auxiliar administrativo",
      grupos_subgrupos: [{ clave: "C2", etiqueta: "C2" }],
    }],
    causas: [{ clave: "necesidad_temporal", etiqueta: "Necesidad temporal" }],
    entradas_rc: [{ referencia: "rc:desarrollo:001", huella_sha256: HUELLA, etiqueta: "Retención 001" }],
    motivos_rectificacion: [],
    jornada_completa_minutos_semanales: 2250,
    ...extra,
  };
}

const DURACIONES = Object.freeze([
  { modalidad_clave: "vacante", unidad: "anios", cantidad: 3, bloquear: false },
  { modalidad_clave: "acumulacion_tareas", unidad: "meses", cantidad: 9, bloquear: true },
]);

test("la configuración publica duraciones y urgencia solo con la forma correcta", () => {
  const salida = validarConfiguracionAnalisis(configuracion({
    duraciones_maximas: DURACIONES.map((d) => ({ ...d })), urgencia_disponible: true,
  }));
  assert.equal(salida.duraciones_maximas.length, 2);
  assert.equal(salida.urgencia_disponible, true);
  assert.equal(Object.isFrozen(salida.duraciones_maximas[0]), true);
  const sinExtras = validarConfiguracionAnalisis(configuracion());
  assert.equal(Object.hasOwn(sinExtras, "duraciones_maximas"), false);
  assert.equal(Object.hasOwn(sinExtras, "urgencia_disponible"), false);
  const malas = [
    [{ modalidad_clave: "relevo", unidad: "meses", cantidad: 6, bloquear: false }],
    [{ modalidad_clave: "vacante", unidad: "semanas", cantidad: 6, bloquear: false }],
    [{ modalidad_clave: "vacante", unidad: "meses", cantidad: 0, bloquear: false }],
    [{ modalidad_clave: "vacante", unidad: "meses", cantidad: 6, bloquear: "no" }],
    [{ modalidad_clave: "vacante", unidad: "meses", cantidad: 6 }],
    [DURACIONES[0], DURACIONES[0]],
  ];
  for (const duraciones of malas) {
    assert.throws(() => validarConfiguracionAnalisis(configuracion({ duraciones_maximas: duraciones })), /duraci/u);
  }
  assert.throws(() => validarConfiguracionAnalisis(configuracion({ urgencia_disponible: "si" })), /configuración/u);
});

test("la duración máxima se cuenta de fecha a fecha como en el servidor", () => {
  const meses = { unidad: "meses", cantidad: 9 };
  assert.equal(periodoSuperaDuracionMaxima(meses, "2026-01-01", "2026-09-30"), false);
  assert.equal(periodoSuperaDuracionMaxima(meses, "2026-01-01", "2026-10-01"), true);
  const unMes = { unidad: "meses", cantidad: 1 };
  assert.equal(periodoSuperaDuracionMaxima(unMes, "2026-01-31", "2026-02-27"), false);
  assert.equal(periodoSuperaDuracionMaxima(unMes, "2026-01-31", "2026-02-28"), true);
  const anios = { unidad: "anios", cantidad: 3 };
  assert.equal(periodoSuperaDuracionMaxima(anios, "2026-03-01", "2029-02-28"), false);
  assert.equal(periodoSuperaDuracionMaxima(anios, "2026-03-01", "2029-03-01"), true);
  const dias = { unidad: "dias_naturales", cantidad: 10 };
  assert.equal(periodoSuperaDuracionMaxima(dias, "2026-03-01", "2026-03-10"), false);
  assert.equal(periodoSuperaDuracionMaxima(dias, "2026-03-01", "2026-03-11"), true);
  assert.equal(periodoSuperaDuracionMaxima(null, "2026-01-01", "2099-01-01"), false);
  assert.equal(periodoSuperaDuracionMaxima(meses, "", "2026-10-01"), false);
});

function crearEntorno(catalogos, operacion = "registrar") {
  const eventos = new Map();
  const avisos = { texto: null };
  const llamadas = [];
  const raiz = {
    innerHTML: "",
    addEventListener(tipo, manejador) { eventos.set(tipo, manejador); },
    removeEventListener(tipo) { eventos.delete(tipo); },
    contains() { return true; },
    querySelector(selector) {
      if (selector === "[data-ct-analisis-aviso-duracion]") {
        return {
          get textContent() { return avisos.texto ?? ""; },
          set textContent(valor) { avisos.texto = valor; },
        };
      }
      return { focus() {}, scrollIntoView() {} };
    },
    replaceChildren() { this.innerHTML = ""; },
  };
  const cliente = {
    async registrarAnalisis(solicitud) {
      llamadas.push(solicitud);
      return {
        esquema: "vec.contratacion-temporal.recibo-analisis-rrhh.v1", operacion,
        expediente_ref: "expediente:opaco:001", version_resultante: 2,
        recibo_ref: "recibo:opaco:analisis:001", confirmada_en: "2026-09-26T08:00:00Z",
      };
    },
    async rectificarAnalisis(solicitud) { return this.registrarAnalisis(solicitud); },
  };
  const desmontar = montarFormularioAnalisisRRHH({
    raiz, cliente, catalogos,
    contexto: {
      operacion, expediente_ref: "expediente:opaco:001", version_esperada: 1,
      artefacto_ref: "artefacto:analisis:desarrollo:v1",
    },
    generarClaveIdempotencia: () => "123e4567-e89b-42d3-a456-426614174000",
  });
  function formulario(valores) {
    return { valores, closest(selector) { return selector === "[data-ct-analisis-form]" ? this : null; } };
  }
  return {
    raiz, avisos, llamadas, desmontar,
    cambiar(nombre, valores) {
      const f = formulario(valores);
      eventos.get("change")({ target: { name: nombre, closest: (s) => f.closest(s) } });
    },
    enviar(valores) {
      return eventos.get("submit")({ target: formulario(valores), preventDefault() {} });
    },
  };
}

function catalogosFormulario(extra = {}) {
  const base = configuracion();
  return {
    modalidades: base.modalidades, categorias: base.categorias, causas: base.causas,
    entradas_rc: base.entradas_rc, motivos_rectificacion: base.motivos_rectificacion,
    jornada_completa_minutos_semanales: 2250, ...extra,
  };
}

function valoresFormulario(extra = {}) {
  return {
    modalidad_clave: "vacante", categoria_ref: "categoria:rrhh:001", grupo_subgrupo: "C2",
    causa_clave: "necesidad_temporal", inicio: "2026-01-01", fin: "2026-06-30",
    jornada_horas: "37", jornada_minutos: "30", entrada_rc_referencia: "rc:desarrollo:001",
    ...extra,
  };
}

test("el formulario pinta una modalidad nueva del catálogo sin lista fija", () => {
  const entorno = crearEntorno(catalogosFormulario());
  assert.match(entorno.raiz.innerHTML, /Interinidad por programa/u);
  entorno.desmontar();
});

test("superar un máximo que solo avisa muestra el aviso y deja registrar", async () => {
  const entorno = crearEntorno(catalogosFormulario({ duraciones_maximas: DURACIONES.map((d) => ({ ...d })) }));
  assert.doesNotMatch(entorno.raiz.innerHTML, /supera la duración máxima/u);
  const valores = valoresFormulario({ fin: "2029-01-01" });
  entorno.cambiar("fin", valores);
  assert.match(entorno.avisos.texto, /supera la duración máxima de esta modalidad \(3 años\)/u);
  entorno.cambiar("fin", valoresFormulario({ fin: "2028-12-31" }));
  assert.equal(entorno.avisos.texto, "");
  const recibo = await entorno.enviar(valores);
  assert.notEqual(recibo, null);
  assert.equal(entorno.llamadas.length, 1);
  entorno.desmontar();
});

test("superar un máximo que bloquea impide enviar y lo explica en el campo", async () => {
  const entorno = crearEntorno(catalogosFormulario({ duraciones_maximas: DURACIONES.map((d) => ({ ...d })) }));
  const recibo = await entorno.enviar(valoresFormulario({ modalidad_clave: "acumulacion_tareas", fin: "2026-10-01" }));
  assert.equal(recibo, null);
  assert.equal(entorno.llamadas.length, 0);
  assert.match(entorno.raiz.innerHTML, /El periodo supera la duración máxima de esta modalidad\./u);
  const valido = await entorno.enviar(valoresFormulario({ modalidad_clave: "acumulacion_tareas", fin: "2026-09-30" }));
  assert.notEqual(valido, null);
  entorno.desmontar();
});

test("sin la regla de urgencia el formulario no la ofrece", () => {
  const entorno = crearEntorno(catalogosFormulario());
  assert.doesNotMatch(entorno.raiz.innerHTML, /urgencia_motivo|Tramitación urgente/u);
  entorno.desmontar();
});

test("con la regla de urgencia se declara con motivo obligatorio", async () => {
  const entorno = crearEntorno(catalogosFormulario({ urgencia_disponible: true }));
  assert.match(entorno.raiz.innerHTML, /Tramitación urgente/u);
  assert.match(entorno.raiz.innerHTML, /<label for="ct-analisis-urgencia_motivo">Motivo de la urgencia<\/label>/u);
  const sinMotivo = await entorno.enviar(valoresFormulario({ urgente: "si", urgencia_motivo: "  " }));
  assert.equal(sinMotivo, null);
  assert.equal(entorno.llamadas.length, 0);
  assert.match(entorno.raiz.innerHTML, /Indique el motivo de la urgencia/u);
  const recibo = await entorno.enviar(valoresFormulario({
    urgente: "si", urgencia_motivo: "Cierre del servicio de ayuda a domicilio si no se cubre antes del lunes.",
  }));
  assert.notEqual(recibo, null);
  assert.equal(entorno.llamadas[0].analisis.urgencia_motivo,
    "Cierre del servicio de ayuda a domicilio si no se cubre antes del lunes.");
  entorno.desmontar();
});

test("sin marcar la urgencia el motivo escrito no viaja", async () => {
  const entorno = crearEntorno(catalogosFormulario({ urgencia_disponible: true }));
  await entorno.enviar(valoresFormulario({ urgencia_motivo: "Texto sin marcar" }));
  assert.equal(Object.hasOwn(entorno.llamadas[0].analisis, "urgencia_motivo"), false);
  entorno.desmontar();
});

test("la bandeja marca los expedientes urgentes y rechaza una urgencia falsa", async () => {
  const { crearAdaptadorHTTPExpedientesContratacionTemporal } = await import("./adaptador-http-expedientes.js");
  const { crearClienteHTTPContratacionTemporal } = await import("./cliente-http.js");
  const { renderizarCuadro } = await import("./componentes-expedientes.js");
  const { crearTraductorExpedientesContratacion } = await import("./i18n-expedientes.js");
  const resumen = {
    expediente_ref: "expediente:ct:001", numero_visible: "2026/CT-0001", version: 3,
    flujo_ref: "flujo:ct:general", flujo_version: 1, flujo_huella_sha256: "a".repeat(64),
    fase_clave: "fiscalizacion", estado_clave: "en_curso", centro_ref: "centro:001",
    categoria_ref: "categoria:auxiliar", creado_en: "2026-09-03T08:00:00Z",
    actualizado_en: "2026-09-15T09:00:00Z",
  };
  const cliente = (expedientes) => crearClienteHTTPContratacionTemporal({
    fetchImpl: async () => new Response(JSON.stringify({ data: {
      esquema: "vec.contratacion-temporal.cuadro-rrhh.v1", generada_en: "2026-09-30T08:00:00Z",
      expedientes, hay_mas: false,
    } }), { status: 200, headers: { "Content-Type": "application/json; charset=utf-8" } }),
  });
  const filtros = { filtros: { texto: "", estado: "", fase: "" } };
  const adaptador = crearAdaptadorHTTPExpedientesContratacionTemporal({
    cliente: cliente([{ ...resumen, urgente: true }, { ...resumen, expediente_ref: "expediente:ct:002", numero_visible: "2026/CT-0002" }]),
  });
  const cuadro = await adaptador.listar(filtros);
  assert.equal(cuadro.expedientes[0].urgente, true);
  assert.equal(Object.hasOwn(cuadro.expedientes[1], "urgente"), false);
  const html = renderizarCuadro({ vista: "cuadro", carga: "listo", filtros: { texto: "", estado: "", fase: "" }, cuadro },
    crearTraductorExpedientesContratacion());
  assert.equal(html.match(/<span class="ct-marca-urgente">Urgente<\/span>/gu)?.length, 1);
  await assert.rejects(cliente([{ ...resumen, urgente: false }]).consultarCuadroRRHH({
    filtros: { texto: "", estado_clave: "", fase_clave: "" }, paginacion: { limite: 50, cursor: "" },
  }), (error) => error?.codigo === "respuesta_incompatible");
});

test("el contrato admite el motivo de urgencia solo bien formado", async () => {
  const { validarSolicitudRegistroAnalisis } = await import("./contrato-analisis.js");
  const solicitud = (extra) => ({
    expediente_ref: "expediente:opaco:001", version_esperada: 1,
    clave_idempotencia: "123e4567-e89b-42d3-a456-426614174000", artefacto_ref: "artefacto:opaco:001",
    analisis: {
      modalidad_clave: "vacante", categoria_ref: "categoria:rrhh:001", grupo_subgrupo: "C2",
      causa_clave: "necesidad_temporal", periodo: { inicio: "2026-01-01T00:00:00Z", fin: "2026-06-30T00:00:00Z" },
      porcentaje_jornada: 10000, entrada_rc: { referencia: "rc:desarrollo:001", huella_sha256: HUELLA }, ...extra,
    },
  });
  assert.equal(validarSolicitudRegistroAnalisis(solicitud({ urgencia_motivo: "Refuerzo por temporal de nieve" }))
    .analisis.urgencia_motivo, "Refuerzo por temporal de nieve");
  for (const malo of ["", " con espacios ", "x".repeat(1001), "control\u0007"]) {
    assert.throws(() => validarSolicitudRegistroAnalisis(solicitud({ urgencia_motivo: malo })), TypeError);
  }
});
