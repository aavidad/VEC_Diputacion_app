import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { diezmilesimasDesdeHorasMinutos, horasMinutosDesdeDiezmilesimas, montarFormularioAnalisisRRHH } from "./formulario-analisis.js";

const UUID = "123e4567-e89b-42d3-a456-426614174000";
const HUELLA = "a".repeat(64);
const FORM_DATA_ORIGINAL = globalThis.FormData;

class FormDataFalso {
  constructor(formulario) {
    this.valores = formulario.valores;
  }

  get(nombre) {
    return Object.hasOwn(this.valores, nombre) ? this.valores[nombre] : null;
  }
}

globalThis.FormData = FormDataFalso;
test.after(() => { globalThis.FormData = FORM_DATA_ORIGINAL; });

function crearCatalogos(cantidadCategorias = 1) {
  return {
    modalidades: [
      { clave: "sustitucion", etiqueta: "Sustitución" },
      { clave: "vacante", etiqueta: "Vacante" },
      { clave: "acumulacion_tareas", etiqueta: "Acumulación de tareas" },
      { clave: "programa", etiqueta: "Programa temporal" },
      { clave: "relevo", etiqueta: "Contrato de relevo" },
    ],
    categorias: Array.from({ length: cantidadCategorias }, (_valor, indice) => ({
      referencia: "categoria:rrhh:" + String(indice + 1).padStart(3, "0"),
      etiqueta: "Categoría RRHH " + String(indice + 1),
      grupos_subgrupos: [{ clave: "A1", etiqueta: "A1" }],
    })),
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

function crearContexto(operacion = "registrar") {
  return {
    operacion,
    expediente_ref: "expediente:opaco:001",
    version_esperada: 1,
    artefacto_ref: "artefacto:opaco:001",
  };
}

function crearValores(sobrescrituras = {}) {
  return {
    modalidad_clave: "sustitucion",
    categoria_ref: "categoria:rrhh:001",
    grupo_subgrupo: "A1",
    causa_clave: "sustitucion",
    inicio: "2026-09-01",
    fin: "2027-08-31",
    jornada_horas: "37",
    jornada_minutos: "30",
    entrada_rc_referencia: "entrada-rc:opaca:001",
    motivo_rectificacion_clave: "correccion_datos",
    ...sobrescrituras,
  };
}

function crearRecibo(operacion = "registrar") {
  return {
    esquema: "vec.contratacion-temporal.recibo-analisis-rrhh.v1",
    operacion,
    expediente_ref: "expediente:opaco:001",
    version_resultante: 2,
    recibo_ref: "recibo:opaco:analisis:001",
    confirmada_en: "2026-08-21T21:30:00Z",
  };
}

function crearRaiz() {
  const eventos = new Map();
  const retirados = [];
  const focos = [];
  const ayudas = new Map();
  const raiz = {
    innerHTML: "",
    addEventListener(tipo, manejador) { eventos.set(tipo, manejador); },
    removeEventListener(tipo, manejador) {
      if (eventos.get(tipo) === manejador) eventos.delete(tipo);
      retirados.push(tipo);
    },
    contains() { return true; },
    querySelector(selector) {
      if (selector === "#ct-analisis-porcentaje_jornada-equivalencia") return { set textContent(valor) { ayudas.set(selector, valor); } };
      return {
        focus() { focos.push(selector); },
        scrollIntoView() {},
      };
    },
    replaceChildren() { this.innerHTML = ""; },
  };
  return {
    raiz,
    eventos,
    focos,
    retirados,
    ayudas,
    enviar(valores = crearValores()) {
      const formulario = {
        valores,
        closest(selector) {
          return selector === "[data-ct-analisis-form]" ? this : null;
        },
      };
      return eventos.get("submit")({
        target: formulario,
        preventDefault() {},
      });
    },
    pulsar(accion) {
      const control = {
        dataset: { ctAnalisisAccion: accion },
        closest(selector) {
          return selector === "[data-ct-analisis-accion]" ? this : null;
        },
      };
      return eventos.get("click")({ target: control, preventDefault() {} });
    },
    escribirJornada(horas, minutos) {
      const campos = { jornada_horas: { value: horas }, jornada_minutos: { value: minutos } };
      const formulario = {
        closest(selector) { return selector === "[data-ct-analisis-form]" ? this : null; },
        querySelector(selector) { return campos[/name=(\w+)/u.exec(selector)?.[1]] ?? null; },
      };
      const control = { name: "jornada_horas", value: horas, closest(selector) { return selector === "[data-ct-analisis-form]" ? formulario : null; } };
      return eventos.get("input")({ target: control });
    },
  };
}

function montar({ operacion = "registrar", cliente, raiz = crearRaiz(), extras = {} }) {
  const desmontar = montarFormularioAnalisisRRHH({
    raiz: raiz.raiz,
    cliente,
    contexto: crearContexto(operacion),
    catalogos: crearCatalogos(),
    generarClaveIdempotencia: () => UUID,
    ...extras,
  });
  return { ...raiz, desmontar };
}

test("la configuración y el contexto son cerrados y no aceptan autoridad del formulario", () => {
  const raiz = crearRaiz();
  const cliente = { registrarAnalisis() {} };
  assert.throws(() => montarFormularioAnalisisRRHH({
    raiz: raiz.raiz,
    cliente,
    contexto: { ...crearContexto(), identidad: "actor:inventado" },
    catalogos: crearCatalogos(),
  }), /contexto del análisis no válido/u);
  assert.throws(() => montarFormularioAnalisisRRHH({
    raiz: raiz.raiz,
    cliente,
    contexto: crearContexto(),
    catalogos: crearCatalogos(),
    perfil: "rrhh",
  }), /configuración del formulario/u);
  const catalogosIncompletos = crearCatalogos();
  catalogosIncompletos.modalidades[4] = {
    clave: "otra_modalidad", etiqueta: "Otra modalidad",
  };
  assert.throws(() => montarFormularioAnalisisRRHH({
    raiz: raiz.raiz,
    cliente,
    contexto: crearContexto(),
    catalogos: catalogosIncompletos,
  }), /modalidades/u);
});

test("sin motivos publicados ni ejecutor autorizado la rectificación queda solo para consulta", async () => {
  const raiz = crearRaiz();
  const catalogos = crearCatalogos();
  catalogos.motivos_rectificacion = [];
  const desmontar = montarFormularioAnalisisRRHH({
    raiz: raiz.raiz,
    cliente: null,
    contexto: crearContexto("rectificar"),
    catalogos,
    generarClaveIdempotencia: () => UUID,
  });

  assert.match(raiz.raiz.innerHTML, /data-ct-analisis-solo-lectura/u);
  assert.match(raiz.raiz.innerHTML, /Puede consultar el análisis, pero no rectificarlo/u);
  assert.doesNotMatch(raiz.raiz.innerHTML, /data-ct-analisis-form|<button/u);
  await raiz.enviar();
  desmontar();
});

test("la vista usa controles gobernados, etiquetas, ayudas y regiones vivas", () => {
  const catalogos = crearCatalogos();
  catalogos.modalidades[0].etiqueta = "<img src=x onerror=privado>";
  const raiz = crearRaiz();
  const desmontar = montarFormularioAnalisisRRHH({
    raiz: raiz.raiz,
    cliente: { registrarAnalisis() {} },
    contexto: crearContexto(),
    catalogos,
    generarClaveIdempotencia: () => UUID,
  });

  assert.match(raiz.raiz.innerHTML, /<form data-ct-analisis-form novalidate>/u);
  assert.match(raiz.raiz.innerHTML, /<label for="ct-analisis-modalidad_clave">/u);
  // Sin ayudas bajo cada campo: solo la jornada conserva su referencia de 37 h 30 min.
  assert.doesNotMatch(raiz.raiz.innerHTML, /ct-analisis-modalidad_clave-ayuda|campos marcados/u);
  assert.match(raiz.raiz.innerHTML, /aria-describedby="ct-analisis-porcentaje_jornada-ayuda"/u);
  assert.doesNotMatch(raiz.raiz.innerHTML, /provisional|no formaliza|Contratación temporal ·/u);
  assert.match(raiz.raiz.innerHTML, /role="status" aria-live="polite"/u);
  assert.match(raiz.raiz.innerHTML, /&lt;img src=x onerror=privado&gt;/u);
  assert.doesNotMatch(raiz.raiz.innerHTML, /<img|name="(?:identidad|organizacion|perfil|autorizacion)"/u);
  assert.doesNotMatch(raiz.raiz.innerHTML, /tabindex="[1-9]|onkey(?:down|press|up)=/u);
  desmontar();
});

test("la jornada se escribe en horas y minutos semanales y conserva el DTO canónico", async () => {
  const solicitudes = [];
  const vista = montar({ cliente: { registrarAnalisis(solicitud) { solicitudes.push(solicitud); return Promise.resolve(crearRecibo()); } } });
  assert.match(vista.raiz.innerHTML, /Jornada contratada \(media semanal\)/u);
  assert.doesNotMatch(vista.raiz.innerHTML, /diezmil/iu);
  vista.escribirJornada("18", "45");
  assert.equal(vista.ayudas.get("#ct-analisis-porcentaje_jornada-equivalencia"), "Equivale a 50,00\u00a0% de la jornada completa.");
  vista.escribirJornada("37", "30");
  assert.equal(vista.ayudas.get("#ct-analisis-porcentaje_jornada-equivalencia"), "Equivale a 100,00\u00a0% de la jornada completa.");
  vista.escribirJornada("37", "31");
  assert.equal(vista.ayudas.get("#ct-analisis-porcentaje_jornada-equivalencia"), "");
  await vista.enviar(crearValores({ jornada_horas: "18", jornada_minutos: "45" }));
  assert.equal(solicitudes[0].analisis.porcentaje_jornada, 5000);
  vista.desmontar();
});

test("la conversión horas-minutos y diezmilésimas es estable en ambos sentidos", () => {
  assert.equal(diezmilesimasDesdeHorasMinutos("37", "30"), "10000");
  assert.equal(diezmilesimasDesdeHorasMinutos("0", "0"), "");
  assert.equal(diezmilesimasDesdeHorasMinutos("38", "0"), "");
  assert.equal(diezmilesimasDesdeHorasMinutos("10", "60"), "");
  assert.deepEqual(horasMinutosDesdeDiezmilesimas("7500"), { horas: "28", minutos: "8" });
  for (const minutos of [1, 59, 600, 1125, 2249, 2250]) {
    const valor = diezmilesimasDesdeHorasMinutos(String(Math.floor(minutos / 60)), String(minutos % 60));
    const vuelta = horasMinutosDesdeDiezmilesimas(valor);
    assert.equal(Number(vuelta.horas) * 60 + Number(vuelta.minutos), minutos);
  }
});

test("registrar envía una sola vez el DTO exacto y presenta el recibo verificado", async () => {
  const llamadas = [];
  const cliente = {
    registrarAnalisis(solicitud, opciones) {
      llamadas.push({ solicitud, opciones });
      return Promise.resolve(crearRecibo());
    },
  };
  const vista = montar({ cliente });
  await vista.enviar();

  assert.equal(llamadas.length, 1);
  assert.deepEqual(Object.keys(llamadas[0].opciones), ["signal"]);
  assert.ok(llamadas[0].opciones.signal instanceof AbortSignal);
  assert.deepEqual(llamadas[0].solicitud, {
    expediente_ref: "expediente:opaco:001",
    version_esperada: 1,
    clave_idempotencia: UUID,
    artefacto_ref: "artefacto:opaco:001",
    analisis: {
      modalidad_clave: "sustitucion",
      categoria_ref: "categoria:rrhh:001",
      grupo_subgrupo: "A1",
      causa_clave: "sustitucion",
      periodo: {
        inicio: "2026-09-01T00:00:00Z",
        fin: "2027-08-31T00:00:00Z",
      },
      porcentaje_jornada: 10000,
      entrada_rc: { referencia: "entrada-rc:opaca:001", huella_sha256: HUELLA },
    },
  });
  assert.ok(Object.isFrozen(llamadas[0].solicitud));
  assert.match(vista.raiz.innerHTML, /data-ct-analisis-recibo/u);
  assert.match(vista.raiz.innerHTML, /recibo:opaco:analisis:001/u);
  assert.doesNotMatch(vista.raiz.innerHTML, /data-ct-analisis-form/u);
  vista.desmontar();
});

test("rectificar exige el motivo gobernado y usa únicamente rectificarAnalisis", async () => {
  let registro = 0;
  let rectificacion;
  const cliente = {
    registrarAnalisis() { registro += 1; },
    rectificarAnalisis(solicitud) {
      rectificacion = solicitud;
      return Promise.resolve(crearRecibo("rectificar"));
    },
  };
  const vista = montar({ operacion: "rectificar", cliente });

  await vista.enviar(crearValores({ motivo_rectificacion_clave: "" }));
  assert.equal(rectificacion, undefined);
  assert.match(vista.raiz.innerHTML, /aria-invalid="true"/u);
  assert.match(vista.raiz.innerHTML, /Seleccione un motivo disponible/u);

  await vista.enviar();
  assert.match(vista.raiz.innerHTML, /Rectificación confirmada/u);
  assert.match(vista.raiz.innerHTML, /recibo:opaco:analisis:001/u);
  assert.equal(registro, 0);
  assert.equal(rectificacion.motivo_rectificacion_clave, "correccion_datos");
  assert.deepEqual(Object.keys(rectificacion), [
    "expediente_ref", "version_esperada", "clave_idempotencia", "artefacto_ref",
    "analisis", "motivo_rectificacion_clave",
  ]);
  vista.desmontar();
});

test("los errores locales producen resumen accesible y foco sin tocar el cliente", async () => {
  let llamadas = 0;
  const vista = montar({
    cliente: { registrarAnalisis() { llamadas += 1; } },
  });
  await vista.enviar(crearValores({
    modalidad_clave: "inventada",
    inicio: "2027-01-02",
    fin: "2027-01-01",
    jornada_horas: "38",
  }));

  assert.equal(llamadas, 0);
  assert.match(vista.raiz.innerHTML, /data-ct-analisis-error-general role="alert"/u);
  assert.match(vista.raiz.innerHTML, /aria-live="assertive" aria-atomic="true" tabindex="-1"/u);
  assert.match(vista.raiz.innerHTML, /aria-invalid="true"/u);
  assert.equal(vista.focos.at(-1), "[data-ct-analisis-error-general]");
  vista.desmontar();
});

test("dos envíos concurrentes comparten el único vuelo y deshabilitan el formulario", async () => {
  let resolver;
  let llamadas = 0;
  const pendiente = new Promise((resolve) => { resolver = resolve; });
  const vista = montar({
    cliente: {
      registrarAnalisis() {
        llamadas += 1;
        return pendiente;
      },
    },
  });

  const primera = vista.enviar();
  const segunda = vista.enviar();
  await Promise.resolve();
  assert.equal(llamadas, 1);
  assert.match(vista.raiz.innerHTML, /<fieldset class="ct-bloque" disabled>/u);
  assert.match(vista.raiz.innerHTML, /data-ct-analisis-accion="cancelar"/u);
  resolver(crearRecibo());
  await Promise.all([primera, segunda]);
  assert.equal(llamadas, 1);
  vista.desmontar();
});

test("cancelar la espera aborta la señal y bloquea el resultado postenvío", async () => {
  let signal;
  const vista = montar({
    cliente: {
      registrarAnalisis(_solicitud, opciones) {
        signal = opciones.signal;
        return new Promise((_resolve, reject) => {
          signal.addEventListener("abort", () => {
            const error = new Error("causa privada");
            error.resultadoIndeterminado = true;
            reject(error);
          }, { once: true });
        });
      },
    },
  });

  const envio = vista.enviar();
  await Promise.resolve();
  vista.pulsar("cancelar");
  await envio;
  assert.equal(signal.aborted, true);
  assert.match(vista.raiz.innerHTML, /data-ct-analisis-indeterminado/u);
  assert.doesNotMatch(vista.raiz.innerHTML, /causa privada|data-ct-analisis-form/u);
  vista.desmontar();
});

test("un resultado indeterminado bloquea cualquier reenvío y redacta el error privado", async () => {
  let llamadas = 0;
  const error = new Error("dsn=privado contraseña=secreta");
  error.codigo = "operacion_pendiente";
  error.resultadoIndeterminado = true;
  error.reintentoPermitido = true;
  const vista = montar({
    cliente: {
      registrarAnalisis() {
        llamadas += 1;
        return Promise.reject(error);
      },
    },
  });

  await vista.enviar();
  assert.equal(llamadas, 1);
  assert.match(vista.raiz.innerHTML, /data-ct-analisis-indeterminado/u);
  assert.match(vista.raiz.innerHTML, /No repita la operación/u);
  assert.doesNotMatch(vista.raiz.innerHTML, /dsn|contraseña|secreta|Análisis confirmado/u);
  await vista.enviar();
  assert.equal(llamadas, 1);
  vista.desmontar();
});

test("una respuesta no verificable se presenta como indeterminada, nunca como éxito", async () => {
  const vista = montar({
    cliente: {
      registrarAnalisis() {
        return Promise.resolve({ ...crearRecibo(), expediente_ref: "expediente:otro:999" });
      },
    },
  });
  await vista.enviar();
  assert.match(vista.raiz.innerHTML, /Resultado indeterminado/u);
  assert.doesNotMatch(vista.raiz.innerHTML, /data-ct-analisis-recibo/u);
  vista.desmontar();
});

test("un fallo determinado no se reintenta solo y un reenvío manual conserva la intención", async () => {
  const claves = [];
  let llamadas = 0;
  const error = new Error("detalle privado de política");
  error.codigo = "acceso_denegado";
  error.resultadoIndeterminado = false;
  const vista = montar({
    cliente: {
      registrarAnalisis(solicitud) {
        llamadas += 1;
        claves.push(solicitud.clave_idempotencia);
        if (llamadas === 1) throw error;
        return Promise.resolve(crearRecibo());
      },
    },
  });

  await vista.enviar();
  assert.equal(llamadas, 1);
  assert.match(vista.raiz.innerHTML, /No dispone de autorización/u);
  assert.doesNotMatch(vista.raiz.innerHTML, /detalle privado/u);
  await Promise.resolve();
  assert.equal(llamadas, 1);
  await vista.enviar();
  assert.equal(llamadas, 2);
  assert.deepEqual(claves, [UUID, UUID]);
  vista.desmontar();
});

test("desmontar aborta el vuelo, retira escuchas, vacía la vista y descarta respuestas tardías", async () => {
  let signal;
  let resolver;
  const pendiente = new Promise((resolve) => { resolver = resolve; });
  const vista = montar({
    cliente: {
      registrarAnalisis(_solicitud, opciones) {
        signal = opciones.signal;
        return pendiente;
      },
    },
  });
  const envio = vista.enviar();
  await Promise.resolve();
  vista.desmontar();

  assert.equal(signal.aborted, true);
  assert.equal(vista.raiz.innerHTML, "");
  assert.deepEqual([...vista.eventos.keys()], []);
  assert.deepEqual(vista.retirados.sort(), ["change", "click", "input", "submit"]);
  resolver(crearRecibo());
  await envio;
  assert.equal(vista.raiz.innerHTML, "");
});

test("el componente no contiene transporte, autoridad de navegador ni persistencia local", async () => {
  const fuente = await readFile(new URL("./formulario-analisis.js", import.meta.url), "utf8");
  assert.doesNotMatch(
    fuente,
    /\b(?:fetch|XMLHttpRequest|WebSocket|EventSource)\s*\(|credentials\s*:|document\.cookie|localStorage|sessionStorage|indexedDB/u,
  );
  assert.doesNotMatch(fuente, /setTimeout|setInterval/u);
  assert.match(fuente, /registrarAnalisis/u);
  assert.match(fuente, /rectificarAnalisis/u);
  assert.match(fuente, /AbortController/u);
});


test("la rectificación prellena datos anteriores sin seleccionar RC, grupo ni motivo", async () => {
  const solicitudes = [];
  const escenario = montar({
    operacion: "rectificar",
    cliente: { rectificarAnalisis(solicitud) { solicitudes.push(solicitud); } },
    extras: { datosPrevios: {
      modalidad_clave: "sustitucion", categoria_ref: "categoria:rrhh:001",
      causa_clave: "sustitucion",
      periodo: { inicio: "2027-01-01T00:00:00Z", fin: "2027-03-31T00:00:00Z" },
      porcentaje_jornada: 7500,
    } },
  });
  assert.match(escenario.raiz.innerHTML, /value="sustitucion" selected/u);
  assert.match(escenario.raiz.innerHTML, /value="2027-01-01"/u);
  assert.match(escenario.raiz.innerHTML, /name="jornada_horas" type="number" required value="28"/u);
  assert.match(escenario.raiz.innerHTML, /name="jornada_minutos" type="number" required value="8"/u);
  assert.doesNotMatch(escenario.raiz.innerHTML, /value="(?:entrada-rc:opaca:001|A1|correccion_datos)" selected/u);
  await escenario.enviar(crearValores({
    entrada_rc_referencia: "", grupo_subgrupo: "", motivo_rectificacion_clave: "",
  }));
  assert.equal(solicitudes.length, 0);
  escenario.desmontar();
});

test("renderiza el textarea de observaciones con id analisis_observaciones y maxlength 4000", () => {
  const escenario = montar({ cliente: { registrarAnalisis() {} } });
  assert.match(
    escenario.raiz.innerHTML,
    /<textarea id="analisis_observaciones" name="observaciones" maxlength="4000"/u,
  );
  assert.match(escenario.raiz.innerHTML, /<label for="analisis_observaciones">Observaciones<\/label>/u);
  escenario.desmontar();
});

test("envío de formulario incluye observaciones cuando se aportan", async () => {
  const solicitudes = [];
  const escenario = montar({
    cliente: {
      registrarAnalisis(solicitud) {
        solicitudes.push(solicitud);
        return Promise.resolve(crearRecibo());
      },
    },
  });
  await escenario.enviar(crearValores({
    observaciones: "Nota de análisis importante para RRHH",
  }));
  assert.equal(solicitudes.length, 1);
  assert.equal(
    solicitudes[0].analisis.observaciones,
    "Nota de análisis importante para RRHH",
  );
  escenario.desmontar();
});

test("envío de formulario omite la clave observaciones si está vacía", async () => {
  const solicitudes = [];
  const escenario = montar({
    cliente: {
      registrarAnalisis(solicitud) {
        solicitudes.push(solicitud);
        return Promise.resolve(crearRecibo());
      },
    },
  });
  await escenario.enviar(crearValores({
    observaciones: "",
  }));
  assert.equal(solicitudes.length, 1);
  assert.equal(Object.hasOwn(solicitudes[0].analisis, "observaciones"), false);
  escenario.desmontar();
});

test("rechaza observaciones mayores a 4000 caracteres en formulario y enfoca el campo", async () => {
  const solicitudes = [];
  const escenario = montar({
    cliente: {
      registrarAnalisis(solicitud) {
        solicitudes.push(solicitud);
      },
    },
  });
  await escenario.enviar(crearValores({
    observaciones: "x".repeat(4001),
  }));
  assert.equal(solicitudes.length, 0);
  assert.match(escenario.raiz.innerHTML, /analisis_observaciones/u);
  assert.match(escenario.raiz.innerHTML, /ct-analisis-observaciones-error/u);
  escenario.desmontar();
});


test("el formulario monta las 151 categorías del catálogo y mantiene el límite", () => {
  const raiz = crearRaiz();
  const desmontar = montarFormularioAnalisisRRHH({
    raiz: raiz.raiz,
    cliente: { registrarAnalisis() {} },
    contexto: crearContexto(),
    catalogos: crearCatalogos(151),
  });
  assert.match(raiz.raiz.innerHTML, /Categoría RRHH 151/u);
  desmontar();
  assert.throws(() => montarFormularioAnalisisRRHH({
    raiz: crearRaiz().raiz,
    cliente: { registrarAnalisis() {} },
    contexto: crearContexto(),
    catalogos: crearCatalogos(1001),
  }), /categorías/u);
});

test("la jornada previa se reenvía exacta si no se tocan horas ni minutos", async () => {
  const solicitudes = [];
  const vista = montar({ cliente: { registrarAnalisis(solicitud) { solicitudes.push(solicitud); return Promise.resolve(crearRecibo()); } } });
  await vista.enviar(crearValores({ jornada_horas: "28", jornada_minutos: "8", jornada_original: "7501" }));
  assert.equal(solicitudes[0].analisis.porcentaje_jornada, 7501);
  vista.desmontar();
  const otra = [];
  const vista2 = montar({ cliente: { registrarAnalisis(solicitud) { otra.push(solicitud); return Promise.resolve(crearRecibo()); } } });
  await vista2.enviar(crearValores({ jornada_horas: "28", jornada_minutos: "9", jornada_original: "7501" }));
  assert.equal(otra[0].analisis.porcentaje_jornada, 7507);
  vista2.desmontar();
});
