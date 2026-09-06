import assert from "node:assert/strict";
import test from "node:test";
import { renderizarCuadro } from "../modulos/contratacion-temporal/componentes-expedientes.js";
import {
  API_ORGANIZACION,
  API_CAMBIOS,
  ESQUEMA_ORGANIZACION,
  FUENTE_RPT,
  filtrarUnidades,
  validarOrganizacion,
  crearCliente,
  crearEstadoFormulario,
} from "./organizacion.js";

const unidad = (extra = {}) => ({
  clave: "u-1",
  etiqueta: "Centro <uno>",
  tipo: "centro",
  ...extra,
});

test("acepta GET antiguo solo como consulta y exige revisión cuando se habilita edición", () => {
  const antiguo = structuredClone(base());
  delete antiguo.data.catalogo_revision;
  delete antiguo.data.edicion_habilitada;
  assert.equal(validarOrganizacion(antiguo).edicion_habilitada, false);
  assert.throws(
    () =>
      validarOrganizacion({
        data: {
          ...base().data,
          edicion_habilitada: true,
          catalogo_revision: undefined,
        },
      }),
    /no válida/,
  );
});

test("GET usa timeout de 10 segundos", async () => {
  let signal;
  const fetchImpl = async (_url, options) => {
    signal = options.signal;
    await new Promise((_resolve, reject) => {
      signal.addEventListener("abort", () => reject(new Error("abortado")));
    });
  };
  await assert.rejects(crearCliente(fetchImpl, 1).obtener(), /abortado/);
  assert.equal(signal.aborted, true);
});

test("red, 503 y recibo inválido son inciertos; 409 es conflicto confirmado", async () => {
  const body = {
    catalogo_version: 1,
    catalogo_revision: 1,
    huella_esperada: "a".repeat(64),
    clave_idempotencia: "00000000-0000-4000-8000-000000000001",
    unidad: { clave: "local-1", etiqueta: "Centro", tipo: "centro" },
    motivo: "ajuste",
  };
  for (const response of [
    { status: 503, data: { error: "temporal" } },
    { status: 200, data: { data: {} } },
  ]) {
    let llamada;
    const cliente = crearCliente(async (url, options) => {
      llamada = { url, options };
      return {
        ok: response.status < 400,
        status: response.status,
        json: async () => response.data,
      };
    });
    await assert.rejects(
      cliente.guardar(body),
      (error) => error.incierto === true && error.body === body,
    );
    assert.equal(llamada.url, API_CAMBIOS);
    assert.equal(
      JSON.parse(llamada.options.body).clave_idempotencia,
      body.clave_idempotencia,
    );
  }
  await assert.rejects(
    crearCliente(async () => {
      throw new Error("red caída");
    }).guardar(body),
    (error) => error.incierto === true && error.body === body,
  );
  await assert.rejects(
    crearCliente(async () => ({ ok: false, status: 409 })).guardar(body),
    (error) => error.conflicto === true && error.incierto !== true,
  );
});
test("el rechazo confirmado permite corregir sin conservar una operación incierta", async () => {
  for (const status of [400, 401, 403, 405]) {
    await assert.rejects(
      crearCliente(async () => ({ ok: false, status })).guardar({}),
      (error) => error.rechazado === true && error.incierto !== true,
    );
  }
  const estado = crearEstadoFormulario();
  estado.preparar({ unidad: { clave: "centro-520" } });
  estado.iniciarEnvio();
  estado.marcarRechazo();
  assert.equal(estado.consultar().bloqueado, false);
  assert.equal(estado.consultar().bodyPendiente, undefined);
});

test("el timeout también abarca la lectura del cuerpo después de las cabeceras", async () => {
  let signal;
  const cliente = crearCliente(async (_url, options) => {
    signal = options.signal;
    return {
      ok: true,
      json: () => new Promise((_resolve, reject) => {
        signal.addEventListener("abort", () => reject(new Error("cuerpo abortado")));
      }),
    };
  }, 1);
  await assert.rejects(cliente.obtener(), /cuerpo abortado/);
  assert.equal(signal.aborted, true);
});

test("un envío incierto inmoviliza el cuerpo y solo reintenta el mismo", () => {
  const estado = crearEstadoFormulario();
  const original = {
    catalogo_version: 1,
    catalogo_revision: 2,
    huella_esperada: "a".repeat(64),
    clave_idempotencia: "00000000-0000-4000-8000-000000000001",
    unidad: { clave: "local-1", etiqueta: "Centro", tipo: "centro" },
    motivo: "ajuste",
  };
  const preparado = estado.preparar(original);
  const primerEnvio = estado.iniciarEnvio();
  original.unidad.etiqueta = "Alterado fuera";
  assert.equal(Object.isFrozen(primerEnvio), true);
  assert.equal(Object.isFrozen(primerEnvio.unidad), true);
  assert.equal(primerEnvio.unidad.etiqueta, "Centro");
  assert.equal(estado.iniciarEnvio(), null, "impide el doble envío");
  estado.marcarIncierto();
  assert.equal(estado.consultar().bloqueado, true);
  assert.equal(estado.preparar({}), null, "impide sustituir la operación");
  assert.equal(
    estado.iniciarEnvio(),
    preparado,
    "reintenta cuerpo y clave exactos",
  );
});

test("el éxito limpia pendiente y el conflicto exige renovar la revisión", () => {
  const cuerpo = {
    unidad: { clave: "u-1", etiqueta: "Centro", tipo: "centro" },
  };
  const exito = crearEstadoFormulario();
  exito.preparar(cuerpo);
  exito.iniciarEnvio();
  exito.marcarExito();
  assert.deepEqual(exito.consultar(), {
    fase: "borrador",
    bodyPendiente: undefined,
    bloqueado: false,
  });

  const conflicto = crearEstadoFormulario();
  conflicto.preparar(cuerpo);
  conflicto.iniciarEnvio();
  conflicto.marcarConflicto();
  assert.equal(conflicto.preparar(cuerpo), null);
  conflicto.renovarRevision();
  assert.notEqual(conflicto.preparar(cuerpo), null);
  assert.equal(conflicto.consultar().fase, "revision");
});
const base = () => ({
  data: {
    esquema: ESQUEMA_ORGANIZACION,
    catalogo_id: "rpt",
    catalogo_version: 1,
    catalogo_revision: 1,
    edicion_habilitada: true,
    catalogo_huella_sha256: "a".repeat(64),
    estado: "borrador",
    fuente_ref: "fuente-servidor",
    descripcion: "Procedencia declarada por el servidor.",
    unidades: [unidad()],
  },
});
test("la consulta de organización es accesible desde el cuadro real, sin adaptador DEMO", () => {
  const html = renderizarCuadro(
    {
      cuadro: { demostracion: false, indicadores: [], expedientes: [] },
      filtros: { texto: "", estado: "", fase: "" },
      carga: "listo",
    },
    (clave) => clave,
  );
  assert.match(html, /href="\/portal-empleado\/organizacion\/"/);
  assert.match(html, /organizacion_referencia/);
  assert.doesNotMatch(html, /ct-exp-operativo-titulo/);
});
test("valida envelope, tipos y límites de organización", () => {
  assert.equal(validarOrganizacion(base()).unidades[0].clave, "u-1");
  assert.throws(
    () => validarOrganizacion({ data: { ...base().data, esquema: "otro" } }),
    /no válida/,
  );
  assert.throws(
    () =>
      validarOrganizacion({
        data: {
          ...base().data,
          unidades: Array.from({ length: 1001 }, (_, i) =>
            unidad({ clave: `u-${i}` }),
          ),
        },
      }),
    /no válida/,
  );
});
test("filtra por texto, padre y acentos sin alterar la fuente", () => {
  const datos = [
    unidad({
      clave: "padre",
      etiqueta: "Área de Informática",
      tipo: "delegacion",
    }),
    unidad({
      clave: "u-2",
      etiqueta: "Puesto técnico",
      tipo: "puesto_responsabilidad",
      adscripcion_clave: "padre",
    }),
  ];
  assert.equal(filtrarUnidades(datos, "informatica", "").length, 2);
  assert.equal(filtrarUnidades(datos, "", "delegacion").length, 1);
  assert.equal(datos.length, 2);
});
test("expone endpoint real y no añade almacenamiento ni datos de demostración", async () => {
  let llamada;
  const body = new TextEncoder().encode(JSON.stringify(base()));
  const cliente = (await import("./organizacion.js")).crearCliente(
    async (url, options) => {
      llamada = { url, options };
      return {
        ok: true,
        body: {
          getReader() {
            let done = false;
            return {
              read: async () => {
                if (done) return { done: true };
                done = true;
                return { done: false, value: body };
              },
              cancel: async () => {},
            };
          },
        },
      };
    },
  );
  await cliente.obtener();
  assert.equal(llamada.url, API_ORGANIZACION);
  assert.equal(llamada.options.credentials, "same-origin");
  assert.equal(llamada.options.redirect, "error");
  assert.equal(llamada.options.cache, "no-store");
});
