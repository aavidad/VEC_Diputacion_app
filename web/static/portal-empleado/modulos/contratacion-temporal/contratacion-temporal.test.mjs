import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import {
  CAPACIDAD_CREAR_SOLICITUD,
  ErrorValidacionAlta,
  LIMITES_ALTA_CONTRATACION,
  crearBorradorAlta,
  crearComandoAlta,
  validarBorradorAlta,
  validarCatalogosAlta,
  validarComandoAlta,
  validarReciboAlta,
} from "./contrato.js";
import { MENSAJES_CONTRATACION_TEMPORAL_ES } from "./i18n.js";
import { crearPresentadorAltaContratacionTemporal } from "./presentador.js";
import {
  montarAltaContratacionTemporal,
  renderizarAltaContratacionTemporal,
} from "./vista.js";

const directorio = new URL("./", import.meta.url);
const [contratoFuente, vistaFuente] = await Promise.all([
  readFile(new URL("contrato.js", directorio), "utf8"),
  readFile(new URL("vista.js", directorio), "utf8"),
]);

const CLAVE_PRUEBA = "12345678-1234-4abc-8def-1234567890ab";

function catalogosPrueba() {
  return {
    esquema: "vec.contratacion_temporal.catalogos_alta.v1",
    centros: [{
      referencia: "cen_sintetico_001",
      etiqueta: "Centro sintético",
      contactos: [{
        referencia: "con_sintetico_001",
        etiqueta: "Responsable referenciado",
      }],
    }],
    categorias: [{
      referencia: "cat_sintetica_001",
      etiqueta: "Categoría sintética",
      grupos_subgrupos: [{ clave: "A2", etiqueta: "Grupo A2" }],
    }],
    motivos: [{ clave: "sustitucion", etiqueta: "Sustitución temporal" }],
    documentos: [
      { referencia: "doc_sintetico_001", etiqueta: "Petición incorporada" },
      { referencia: "doc_sintetico_002", etiqueta: "Retención incorporada" },
    ],
  };
}

function borradorValido(cambios = {}) {
  return {
    ...crearBorradorAlta(),
    centro_ref: "cen_sintetico_001",
    contacto_ref: "con_sintetico_001",
    categoria_ref: "cat_sintetica_001",
    grupo_subgrupo: "A2",
    motivo_clave: "sustitucion",
    detalle: "Necesidad administrativa temporal.",
    inicio: "2026-08-01",
    fin: "2026-08-31",
    rc_existe: true,
    rc_numero: "rc_sintetica_001",
    rc_fecha: "2026-07-23",
    rc_importe: "32450,00",
    rc_documento_ref: "doc_sintetico_002",
    documentos_adjuntos: ["doc_sintetico_001"],
    observaciones: "Tramitación ordinaria sintética.",
    ...cambios,
  };
}

function reciboValido(cambios = {}) {
  return {
    expediente_ref: "exp_sintetico_001",
    numero_visible: "2026/CT-0001",
    version: 1,
    recibo_ref: "rec_sintetico_001",
    confirmada_en: "2026-07-23T09:15:00Z",
    ...cambios,
  };
}

function crearPresentador({
  catalogos = catalogosPrueba(),
  capacidad = CAPACIDAD_CREAR_SOLICITUD,
  ejecutor = async () => reciboValido(),
  claves = [CLAVE_PRUEBA],
} = {}) {
  let indice = 0;
  return crearPresentadorAltaContratacionTemporal({
    catalogos,
    capacidad,
    ejecutor,
    generarClaveIdempotencia: () => claves[Math.min(indice++, claves.length - 1)],
  });
}

function crearRaizDOMFalsa() {
  const escuchas = new Map();
  const desplazamientos = [];
  const focos = [];
  const raiz = {
    innerHTML: "",
    addEventListener(tipo, escucha) {
      escuchas.set(tipo, escucha);
    },
    removeEventListener(tipo, escucha) {
      if (escuchas.get(tipo) === escucha) escuchas.delete(tipo);
    },
    querySelector(selector) {
      return {
        focus() {
          focos.push(selector);
        },
        scrollIntoView() {
          desplazamientos.push(selector);
        },
      };
    },
    contains() {
      return true;
    },
  };
  return {
    desplazamientos, escuchas, focos, raiz,
  };
}

function eventoAccion(accion) {
  const control = { dataset: { ctAccion: accion } };
  return {
    preventDefault() {},
    target: {
      closest(selector) {
        if (selector === "[data-ct-enfocar]") return null;
        return selector === "[data-ct-accion]" ? control : null;
      },
    },
  };
}

test("el contrato es cerrado, clona catálogos y conserva relaciones inyectadas", () => {
  const entrada = catalogosPrueba();
  const catalogos = validarCatalogosAlta(entrada);
  entrada.centros[0].etiqueta = "Mutación posterior";
  entrada.centros[0].contactos.push({
    referencia: "con_sintetico_002",
    etiqueta: "Contacto añadido",
  });
  assert.equal(catalogos.centros[0].etiqueta, "Centro sintético");
  assert.equal(catalogos.centros[0].contactos.length, 1);
  assert.ok(Object.isFrozen(catalogos.centros[0].contactos));

  assert.throws(
    () => validarCatalogosAlta({ ...catalogosPrueba(), campo_extra: true }),
    /contrato cerrado/,
  );
  const catalogosConExtra = catalogosPrueba();
  catalogosConExtra.centros[0].contactos[0].dato_personal = "prohibido";
  assert.throws(() => validarCatalogosAlta(catalogosConExtra), /contrato cerrado/);
  const catalogosCruzados = catalogosPrueba();
  catalogosCruzados.centros[0].contactos = [];
  assert.equal(
    validarBorradorAlta(borradorValido(), catalogosCruzados).errores.contacto_ref,
    "opcion_catalogo",
  );
});

test("el DTO rechaza campos extra, tipos, tamaños, fechas e importes inválidos", () => {
  const catalogos = catalogosPrueba();
  assert.throws(
    () => crearComandoAlta({ ...borradorValido(), campo_extra: true }, catalogos, CLAVE_PRUEBA),
    ErrorValidacionAlta,
  );
  assert.throws(
    () => crearComandoAlta(borradorValido({ detalle: "😀".repeat(4001) }), catalogos, CLAVE_PRUEBA),
    ErrorValidacionAlta,
  );
  assert.doesNotThrow(
    () => crearComandoAlta(borradorValido({ detalle: "😀".repeat(4000) }), catalogos, CLAVE_PRUEBA),
  );
  for (const cambios of [
    { inicio: "2026-02-30" },
    { fin: "2026-07-31" },
    { rc_importe: "0,00" },
    { rc_importe: "12,3" },
    { rc_importe: "90071992547409,92" },
    { observaciones: " texto con espacios " },
    { detalle: "texto\u0000control" },
    { detalle: "texto\uD800incompleto" },
  ]) {
    assert.throws(() => crearComandoAlta(borradorValido(cambios), catalogos, CLAVE_PRUEBA));
  }
  assert.throws(
    () => crearComandoAlta(
      borradorValido({
        rc_existe: false,
        rc_numero: "rc_sintetica_001",
        rc_fecha: "",
        rc_importe: "",
        rc_documento_ref: "",
      }),
      catalogos,
      CLAVE_PRUEBA,
    ),
    ErrorValidacionAlta,
  );
  assert.throws(() => crearComandoAlta(
    borradorValido({ inicio: "0000-08-01" }),
    catalogos,
    CLAVE_PRUEBA,
  ));

  const comando = crearComandoAlta(
    borradorValido(),
    catalogos,
    CLAVE_PRUEBA,
  );
  const conSimbolo = { ...comando };
  conSimbolo[Symbol("campo_oculto")] = "prohibido";
  assert.throws(() => validarComandoAlta(conSimbolo));
  let accesos = 0;
  const conAccesor = { ...comando };
  Object.defineProperty(conAccesor, "solicitud", {
    enumerable: true,
    get() {
      accesos += 1;
      return comando.solicitud;
    },
  });
  assert.throws(() => validarComandoAlta(conAccesor));
  assert.equal(accesos, 0);
  assert.throws(() => validarComandoAlta({
    ...comando,
    solicitud: {
      ...comando.solicitud,
      documentos_adjuntos: Array(1),
    },
  }));
});

test("el periodo y la RC respetan los límites exactos de la frontera interna", async () => {
  assert.equal(LIMITES_ALTA_CONTRATACION.maximoAniosPeriodo, 100);
  assert.equal(LIMITES_ALTA_CONTRATACION.maximoCentimosRC, 922_337_203_685_477);

  const comandoLimite = crearComandoAlta(
    borradorValido({
      inicio: "2026-08-01",
      fin: "2126-08-01",
      rc_importe: "9223372036854,77",
    }),
    catalogosPrueba(),
    CLAVE_PRUEBA,
  );
  assert.equal(comandoLimite.solicitud.periodo.fin, "2126-08-01T00:00:00Z");
  assert.equal(comandoLimite.solicitud.rc.importe.centimos, 922_337_203_685_477);
  assert.throws(() => validarComandoAlta({
    ...comandoLimite,
    solicitud: {
      ...comandoLimite.solicitud,
      periodo: {
        ...comandoLimite.solicitud.periodo,
        fin: "2126-08-02T00:00:00Z",
      },
    },
  }));
  assert.throws(() => validarComandoAlta({
    ...comandoLimite,
    solicitud: {
      ...comandoLimite.solicitud,
      rc: {
        ...comandoLimite.solicitud.rc,
        importe: { centimos: 922_337_203_685_478, moneda: "EUR" },
      },
    },
  }));

  for (const rcImporte of [
    "9223372036854,78",
    "90071992547409,91",
  ]) {
    const validacion = validarBorradorAlta(
      borradorValido({ rc_importe: rcImporte }),
      catalogosPrueba(),
    );
    assert.equal(validacion.valido, false);
    assert.equal(validacion.errores.rc_importe, "importe_maximo");
  }

  let invocaciones = 0;
  const presentador = crearPresentador({
    ejecutor: async () => {
      invocaciones += 1;
      return reciboValido();
    },
  });
  assert.equal(presentador.prepararRevision(borradorValido({
    inicio: "2026-08-01",
    fin: "2126-08-02",
  })), false);
  assert.equal(presentador.obtenerEstado().errores.fin, "periodo_maximo");
  assert.match(MENSAJES_CONTRATACION_TEMPORAL_ES.error_periodo_maximo, /cien años civiles/);
  assert.match(MENSAJES_CONTRATACION_TEMPORAL_ES.error_importe_maximo, /9\.223\.372/);
  await assert.rejects(presentador.enviar());
  assert.equal(invocaciones, 0);
});

test("el comando replica el dominio, usa céntimos y aplica copia defensiva", () => {
  const borrador = borradorValido();
  const comando = crearComandoAlta(borrador, catalogosPrueba(), CLAVE_PRUEBA);
  assert.deepEqual(Object.keys(comando), ["clave_idempotencia", "solicitud"]);
  assert.deepEqual(Object.keys(comando.solicitud), [
    "centro_ref",
    "contacto_ref",
    "categoria_ref",
    "grupo_subgrupo",
    "motivo_clave",
    "detalle",
    "periodo",
    "rc",
    "documentos_adjuntos",
    "observaciones",
  ]);
  assert.deepEqual(comando.solicitud.periodo, {
    inicio: "2026-08-01T00:00:00Z",
    fin: "2026-08-31T00:00:00Z",
  });
  assert.deepEqual(comando.solicitud.rc.importe, { centimos: 3_245_000, moneda: "EUR" });
  borrador.documentos_adjuntos.push("doc_sintetico_002");
  assert.deepEqual(comando.solicitud.documentos_adjuntos, ["doc_sintetico_001"]);
  assert.ok(Object.isFrozen(comando.solicitud.documentos_adjuntos));

  const sinRC = crearComandoAlta(borradorValido({
    rc_existe: false,
    rc_numero: "",
    rc_fecha: "",
    rc_importe: "",
    rc_documento_ref: "",
  }), catalogosPrueba(), CLAVE_PRUEBA);
  assert.deepEqual(sinRC.solicitud.rc, { existe: false });
  assert.throws(() => validarComandoAlta({ ...comando, actor_ref: "act_prohibido_001" }));
});

test("el comando y el recibo público minimizan identidad y material privado", () => {
  const comando = crearComandoAlta(borradorValido(), catalogosPrueba(), CLAVE_PRUEBA);
  const recibo = validarReciboAlta(reciboValido());
  const prohibido =
    /(?:^|[^a-z0-9])(?:dni|nif|nombre|correo|telefono|rol|permiso|actor|perfil|sesion|autenticacion|organizacion|capacidad|hmac|decision|atestacion|token|auditoria|evento)(?:[^a-z0-9]|$)/i;
  assert.doesNotMatch(JSON.stringify(comando), prohibido);
  assert.doesNotMatch(JSON.stringify(recibo), prohibido);
  assert.deepEqual(Object.keys(recibo), [
    "expediente_ref",
    "numero_visible",
    "version",
    "recibo_ref",
    "confirmada_en",
  ]);
  assert.throws(() => validarReciboAlta({ ...reciboValido(), auditoria_ref: "aud_privada_001" }));
  assert.throws(() => validarReciboAlta({ ...reciboValido(), evento_ref: "evt_privado_001" }));
  assert.throws(() => validarReciboAlta({ ...reciboValido(), version: 0 }));
  assert.throws(() => validarReciboAlta({ ...reciboValido(), confirmada_en: "2026-07-23" }));
  assert.throws(
    () => validarReciboAlta({ ...reciboValido(), confirmada_en: "2026-02-30T09:15:00Z" }),
  );
  for (const confirmadaEn of [
    "2026-07-23T24:00:00Z",
    "2026-07-23T23:60:00Z",
    "2026-07-23T23:59:60Z",
    "2026-07-23T09:15:00.1234567Z",
    "2026-07-23T09:15:00+00:00",
  ]) {
    assert.throws(
      () => validarReciboAlta({ ...reciboValido(), confirmada_en: confirmadaEn }),
      `se aceptó el instante no canónico ${confirmadaEn}`,
    );
  }
  assert.doesNotThrow(
    () => validarReciboAlta({
      ...reciboValido(),
      confirmada_en: "2026-07-23T09:15:00.123456Z",
    }),
  );
});

test("el presentador envía una vez, no expone la clave y acepta solo un recibo válido", async () => {
  const comandos = [];
  const presentador = crearPresentador({
    ejecutor: async (comando) => {
      comandos.push(comando);
      return reciboValido();
    },
  });
  assert.equal(presentador.prepararRevision(borradorValido()), true);
  assert.doesNotMatch(JSON.stringify(presentador.obtenerEstado()), /idempotencia/i);
  const recibo = await presentador.enviar();
  assert.equal(comandos.length, 1);
  assert.equal(comandos[0].clave_idempotencia, CLAVE_PRUEBA);
  assert.equal(recibo.numero_visible, "2026/CT-0001");
  assert.equal(presentador.obtenerEstado().fase, "recibo");
});

test("el mutex bloquea la doble pulsación antes del primer await", async () => {
  let resolver;
  let llamadas = 0;
  const espera = new Promise((resuelve) => { resolver = resuelve; });
  const presentador = crearPresentador({
    ejecutor: async () => {
      llamadas += 1;
      return espera;
    },
  });
  presentador.prepararRevision(borradorValido());
  const primerEnvio = presentador.enviar();
  const segundoEnvio = presentador.enviar();
  assert.strictEqual(primerEnvio, segundoEnvio);
  assert.equal(llamadas, 1);
  assert.equal(presentador.obtenerEstado().ocupado, true);
  assert.match(
    renderizarAltaContratacionTemporal(presentador.obtenerEstado()),
    /aria-busy="true"/,
  );
  resolver(reciboValido());
  await primerEnvio;
  assert.equal(presentador.obtenerEstado().fase, "recibo");
});

test("la cancelación es segura, conserva contenido y reutiliza la misma clave al reintentar", async () => {
  const clavesRecibidas = [];
  let intento = 0;
  const presentador = crearPresentador({
    ejecutor: async (comando, contexto) => {
      clavesRecibidas.push(comando.clave_idempotencia);
      intento += 1;
      if (intento === 1) {
        return new Promise((_resolver, rechazar) => {
          contexto.signal.addEventListener(
            "abort",
            () => rechazar(new Error("detalle privado de transporte")),
            { once: true },
          );
        });
      }
      return reciboValido();
    },
  });
  presentador.prepararRevision(borradorValido());
  const envio = presentador.enviar();
  assert.equal(presentador.cancelarEnvio(), true);
  assert.equal(await envio, null);
  assert.equal(presentador.obtenerEstado().fase, "revision");
  assert.equal(presentador.obtenerEstado().mensaje_clave, "estado_cancelado");
  assert.equal((await presentador.enviar()).recibo_ref, "rec_sintetico_001");
  assert.deepEqual(clavesRecibidas, [CLAVE_PRUEBA, CLAVE_PRUEBA]);
});

test("los errores se redactan, permiten reintento y nunca filtran el mensaje privado", async () => {
  const comandos = [];
  let intento = 0;
  const presentador = crearPresentador({
    ejecutor: async (comando) => {
      comandos.push(comando);
      intento += 1;
      if (intento === 1) throw new Error("SQL secreto DSN=privado");
      return reciboValido();
    },
  });
  presentador.prepararRevision(borradorValido());
  assert.equal(await presentador.enviar(), null);
  assert.equal(presentador.obtenerEstado().mensaje_clave, "estado_error");
  assert.doesNotMatch(
    renderizarAltaContratacionTemporal(presentador.obtenerEstado()),
    /SQL|DSN|privado/,
  );
  await presentador.enviar();
  assert.equal(comandos[0].clave_idempotencia, comandos[1].clave_idempotencia);
});

test("un resultado indeterminado bloquea el reenvío hasta recuperación protegida", async () => {
  let invocaciones = 0;
  const presentador = crearPresentador({
    ejecutor: async () => {
      invocaciones += 1;
      const error = new Error("detalle privado");
      error.resultadoIndeterminado = true;
      error.reintentoPermitido = false;
      throw error;
    },
  });
  presentador.prepararRevision(borradorValido());
  assert.equal(await presentador.enviar(), null);
  const estado = presentador.obtenerEstado();
  assert.equal(estado.fase, "pendiente");
  assert.equal(estado.mensaje_clave, "estado_operacion_pendiente");
  const html = renderizarAltaContratacionTemporal(estado);
  assert.match(html, /data-ct-operacion-pendiente/u);
  assert.doesNotMatch(html, /data-ct-accion="confirmar"/u);
  assert.doesNotMatch(html, /data-ct-accion="volver"/u);
  await assert.rejects(presentador.enviar());
  assert.equal(invocaciones, 1);
});

test("un error hostil no rompe el estado ni expone su contenido", async () => {
  const privado = new Proxy(new Error("secreto interno"), {
    get() {
      throw new Error("getter privado");
    },
  });
  const presentador = crearPresentador({
    ejecutor: async () => {
      throw privado;
    },
  });
  presentador.prepararRevision(borradorValido());
  assert.equal(await presentador.enviar(), null);
  assert.equal(presentador.obtenerEstado().fase, "revision");
  assert.doesNotMatch(
    renderizarAltaContratacionTemporal(presentador.obtenerEstado()),
    /secreto|getter privado/u,
  );
});

test("editar conserva la clave solo si la solicitud canónica es idéntica", async () => {
  const CLAVE_NUEVA = "22345678-1234-4abc-8def-1234567890ab";
  const clavesSinCambio = [];
  let intentoSinCambio = 0;
  const sinCambio = crearPresentador({
    claves: [CLAVE_PRUEBA, CLAVE_NUEVA],
    ejecutor: async (comando) => {
      clavesSinCambio.push(comando.clave_idempotencia);
      intentoSinCambio += 1;
      if (intentoSinCambio === 1) throw new Error("privado");
      return reciboValido();
    },
  });
  sinCambio.prepararRevision(borradorValido());
  await sinCambio.enviar();
  sinCambio.volverAEdicion();
  sinCambio.prepararRevision(borradorValido({ rc_importe: "32450.00" }));
  await sinCambio.enviar();
  assert.deepEqual(clavesSinCambio, [CLAVE_PRUEBA, CLAVE_PRUEBA]);

  const clavesConCambio = [];
  const conCambio = crearPresentador({
    claves: [CLAVE_PRUEBA, CLAVE_NUEVA],
    ejecutor: async (comando) => {
      clavesConCambio.push(comando.clave_idempotencia);
      throw new Error("privado");
    },
  });
  conCambio.prepararRevision(borradorValido());
  await conCambio.enviar();
  conCambio.volverAEdicion();
  conCambio.prepararRevision(borradorValido({ detalle: "Contenido realmente distinto." }));
  await conCambio.enviar();
  assert.deepEqual(clavesConCambio, [CLAVE_PRUEBA, CLAVE_NUEVA]);
});

test("el presentador limpia relaciones dependientes fuera de la vista", () => {
  const catalogos = catalogosPrueba();
  catalogos.centros.push({
    referencia: "cen_sintetico_002",
    etiqueta: "Segundo centro",
    contactos: [{
      referencia: "con_sintetico_002",
      etiqueta: "Segundo responsable",
    }],
  });
  catalogos.categorias.push({
    referencia: "cat_sintetica_002",
    etiqueta: "Segunda categoría",
    grupos_subgrupos: [{ clave: "C1", etiqueta: "Grupo C1" }],
  });
  const presentador = crearPresentador({ catalogos });
  presentador.actualizarBorrador({
    ...crearBorradorAlta(),
    centro_ref: "cen_sintetico_001",
    categoria_ref: "cat_sintetica_001",
  });
  presentador.actualizarBorrador({
    ...presentador.obtenerEstado().borrador,
    contacto_ref: "con_sintetico_001",
    grupo_subgrupo: "A2",
  });
  presentador.actualizarBorrador({
    ...presentador.obtenerEstado().borrador,
    centro_ref: "cen_sintetico_002",
    categoria_ref: "cat_sintetica_002",
  });
  assert.equal(presentador.obtenerEstado().borrador.contacto_ref, "");
  assert.equal(presentador.obtenerEstado().borrador.grupo_subgrupo, "");
  assert.doesNotMatch(vistaFuente, /campo === "centro_ref".*contacto_ref/s);
  assert.doesNotMatch(vistaFuente, /campo === "categoria_ref".*grupo_subgrupo/s);
});

test("un recibo adulterado o incompleto no produce un falso éxito", async () => {
  for (const respuesta of [
    { ...reciboValido(), decision: "concedida" },
    { ...reciboValido(), expediente_ref: "x" },
    { numero_visible: "2026/CT-0001" },
  ]) {
    const presentador = crearPresentador({ ejecutor: async () => respuesta });
    presentador.prepararRevision(borradorValido());
    assert.equal(await presentador.enviar(), null);
    assert.equal(presentador.obtenerEstado().fase, "revision");
    assert.equal(presentador.obtenerEstado().mensaje_clave, "estado_recibo_invalido");
    assert.equal(presentador.obtenerEstado().recibo, null);
  }
});

test("sin capacidad o sin ejecutor el módulo falla cerrado y no invoca efectos", () => {
  let llamadas = 0;
  const sinCapacidad = crearPresentador({
    capacidad: null,
    ejecutor: async () => { llamadas += 1; },
  });
  const estado = sinCapacidad.obtenerEstado();
  assert.equal(estado.disponible, false);
  assert.match(renderizarAltaContratacionTemporal(estado), /Servicio no disponible/);
  assert.match(renderizarAltaContratacionTemporal(estado), /type="submit" disabled/);
  assert.throws(() => sinCapacidad.prepararRevision(borradorValido()), /no está disponible/);

  const sinEjecutor = crearPresentador({ ejecutor: null });
  assert.equal(sinEjecutor.obtenerEstado().disponible, false);
  assert.throws(() => sinEjecutor.prepararRevision(borradorValido()));
  assert.equal(llamadas, 0);
});

test("las opciones proceden solo del catálogo y el contenido hostil queda escapado", () => {
  const catalogos = catalogosPrueba();
  const etiquetaHostil = '\"><img src=x onerror=\"alert(1)\">';
  catalogos.centros[0].etiqueta = etiquetaHostil;
  catalogos.documentos[0].etiqueta = etiquetaHostil;
  const presentador = crearPresentador({ catalogos });
  const edicion = renderizarAltaContratacionTemporal(presentador.obtenerEstado());
  assert.doesNotMatch(edicion, /<img src=x/);
  assert.match(edicion, /&lt;img src=x/);
  assert.doesNotMatch(contratoFuente, /Trabajador Social|Auxiliar Administrativo|Vacante/);
  assert.doesNotMatch(vistaFuente, /Trabajador Social|Auxiliar Administrativo|Vacante/);

  const detalleHostil = "<img src=x onerror=alert(2)>";
  presentador.prepararRevision(borradorValido({ detalle: detalleHostil }));
  const revision = renderizarAltaContratacionTemporal(presentador.obtenerEstado());
  assert.doesNotMatch(revision, /<img src=x/);
  assert.match(revision, /&lt;img src=x/);
  const comando = crearComandoAlta(borradorValido(), catalogos, CLAVE_PRUEBA);
  assert.doesNotMatch(JSON.stringify(comando), new RegExp(etiquetaHostil.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")));
});

test("la estructura permite teclado, etiquetas, errores asociados y anuncios aria", () => {
  const presentador = crearPresentador();
  const html = renderizarAltaContratacionTemporal(presentador.obtenerEstado());
  for (const campo of [
    "centro_ref",
    "contacto_ref",
    "categoria_ref",
    "grupo_subgrupo",
    "motivo_clave",
    "detalle",
    "inicio",
    "fin",
    "observaciones",
  ]) {
    assert.match(html, new RegExp(`<label[^>]+for=\"ct-${campo}\"`));
    assert.match(html, new RegExp(`id=\"ct-${campo}\"`));
  }
  assert.ok((html.match(/<fieldset/g) || []).length >= 5);
  assert.ok((html.match(/<legend/g) || []).length >= 5);
  assert.match(html, /aria-live="polite"/);
  assert.match(html, /Los campos marcados con \* son obligatorios\./);
  assert.doesNotMatch(html, /aria-required="true"/);
  assert.match(html, /id="ct-detalle-contador"/);
  assert.doesNotMatch(html, /(?:detalle|observaciones)"[^>]+maxlength="4000"/);
  assert.doesNotMatch(html, /tabindex="[1-9]/);
  assert.doesNotMatch(html, /type="file"|onclick=|javascript:/i);

  presentador.prepararRevision(borradorValido());
  const revision = renderizarAltaContratacionTemporal(presentador.obtenerEstado());
  assert.ok(revision.indexOf('data-ct-accion="volver"')
    < revision.indexOf('data-ct-accion="confirmar"'));
  assert.match(revision, /id="ct-revision-titulo" tabindex="-1"/);

  const conErrores = crearPresentador();
  assert.equal(conErrores.prepararRevision(crearBorradorAlta()), false);
  const htmlErrores = renderizarAltaContratacionTemporal(conErrores.obtenerEstado());
  assert.match(htmlErrores, /role="alert"[^>]+aria-live="assertive"/);
  assert.match(htmlErrores, /aria-invalid="true"/);
  assert.match(htmlErrores, /aria-describedby="ct-centro_ref-ayuda ct-centro_ref-error"/);
  const htmlErroresGrupos = renderizarAltaContratacionTemporal({
    ...conErrores.obtenerEstado(),
    errores: {
      ...conErrores.obtenerEstado().errores,
      rc_existe: "booleano",
      documentos_adjuntos: "adjuntos",
    },
  });
  assert.match(
    htmlErroresGrupos,
    /id="ct-rc_existe"[^>]+aria-describedby="ct-rc_existe-ayuda ct-rc_existe-error"[^>]+aria-invalid="true"/,
  );
  assert.match(
    htmlErroresGrupos,
    /id="ct-documentos_adjuntos"[^>]+aria-describedby="ct-documentos_adjuntos-ayuda ct-documentos_adjuntos-error"[^>]+aria-invalid="true"/,
  );
});

test("el montaje conserva foco, usa controles nativos y no repinta tras desmontar", async () => {
  let resolver;
  const respuestaPendiente = new Promise((resuelve) => { resolver = resuelve; });
  const presentador = crearPresentador({
    ejecutor: async () => respuestaPendiente,
  });
  presentador.prepararRevision(borradorValido());
  const {
    desplazamientos, escuchas, focos, raiz,
  } = crearRaizDOMFalsa();
  const anuncios = [];
  const desmontar = montarAltaContratacionTemporal({
    raiz,
    presentador,
    anunciar: (...argumentos) => anuncios.push(argumentos),
  });
  assert.deepEqual([...escuchas.keys()].sort(), ["change", "click", "input", "submit"]);
  assert.doesNotMatch(raiz.innerHTML, /tabindex="[1-9]|onkeydown=|onkeypress=/);

  const pulsacion = escuchas.get("click")(eventoAccion("confirmar"));
  assert.equal(focos.at(-1), "[data-ct-accion='cancelar']");
  assert.equal(desplazamientos.at(-1), "[data-ct-accion='cancelar']");
  const htmlDuranteEnvio = raiz.innerHTML;
  const anunciosAntesDesmontar = anuncios.length;
  desmontar();
  assert.equal(escuchas.size, 0);
  resolver(reciboValido());
  await pulsacion;
  assert.equal(raiz.innerHTML, htmlDuranteEnvio);
  assert.equal(anuncios.length, anunciosAntesDesmontar);
});

test("la cancelación montada mantiene un destino de foco estable", async () => {
  const presentador = crearPresentador({
    ejecutor: async (_comando, contexto) => new Promise((_resolver, rechazar) => {
      contexto.signal.addEventListener("abort", () => rechazar(new Error("privado")), {
        once: true,
      });
    }),
  });
  presentador.prepararRevision(borradorValido());
  const {
    desplazamientos, escuchas, focos, raiz,
  } = crearRaizDOMFalsa();
  const desmontar = montarAltaContratacionTemporal({ raiz, presentador });
  const envio = escuchas.get("click")(eventoAccion("confirmar"));
  await escuchas.get("click")(eventoAccion("cancelar"));
  assert.ok(focos.includes("[data-ct-estado]"));
  assert.ok(desplazamientos.includes("[data-ct-estado]"));
  await envio;
  assert.equal(focos.at(-1), "[data-ct-accion='confirmar']");
  desmontar();
});

test("la revisión localiza fechas civiles e importe sin alterar el DTO", () => {
  const presentador = crearPresentador();
  presentador.prepararRevision(borradorValido());
  const html = renderizarAltaContratacionTemporal(presentador.obtenerEstado(), {
    locale: "en-US",
  });
  assert.match(html, /August 1, 2026/);
  assert.match(html, /July 23, 2026/);
  assert.match(html, /€32,450\.00/);
  assert.doesNotMatch(html, /32450,00 EUR/);
  const comando = crearComandoAlta(borradorValido(), catalogosPrueba(), CLAVE_PRUEBA);
  assert.equal(comando.solicitud.periodo.inicio, "2026-08-01T00:00:00Z");
  assert.equal(comando.solicitud.rc.importe.centimos, 3_245_000);
});
