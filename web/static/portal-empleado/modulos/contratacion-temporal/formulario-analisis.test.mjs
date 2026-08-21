import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { montarFormularioAnalisisRRHH } from "./formulario-analisis.js";

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
    modalidad_clave: "interinidad",
    categoria_ref: "categoria:rrhh:001",
    grupo_subgrupo: "A1",
    causa_clave: "sustitucion",
    inicio: "2026-09-01",
    fin: "2027-08-31",
    porcentaje_jornada: "10000",
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
  const raiz = {
    innerHTML: "",
    addEventListener(tipo, manejador) { eventos.set(tipo, manejador); },
    removeEventListener(tipo, manejador) {
      if (eventos.get(tipo) === manejador) eventos.delete(tipo);
      retirados.push(tipo);
    },
    contains() { return true; },
    querySelector(selector) {
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
  assert.match(raiz.raiz.innerHTML, /aria-describedby="ct-analisis-modalidad_clave-ayuda"/u);
  assert.match(raiz.raiz.innerHTML, /role="status" aria-live="polite"/u);
  assert.match(raiz.raiz.innerHTML, /&lt;img src=x onerror=privado&gt;/u);
  assert.doesNotMatch(raiz.raiz.innerHTML, /<img|name="(?:identidad|organizacion|perfil|autorizacion)"/u);
  assert.doesNotMatch(raiz.raiz.innerHTML, /tabindex="[1-9]|onkey(?:down|press|up)=/u);
  desmontar();
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
      modalidad_clave: "interinidad",
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
  assert.match(vista.raiz.innerHTML, /Seleccione un motivo gobernado/u);

  await vista.enviar();
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
    porcentaje_jornada: "10001",
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
  assert.deepEqual(vista.retirados.sort(), ["change", "click", "submit"]);
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
