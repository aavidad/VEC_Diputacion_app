import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { File } from "node:buffer";
import test from "node:test";
import { fechaRespuestaMadridUTC } from "./formulario-llamamiento.js?v=20261001-ct-a-i18n-v1";
import { MENSAJES_LLAMAMIENTO_EN } from "./i18n-llamamiento.js?v=20261001-ct-a-i18n-v1";
import { crearClienteHTTPContratacionTemporal } from "./cliente-http.js";
import {
  CLAVE, EXPEDIENTE, PUBLICACIONES_PROPUESTA, seleccion, recibo,
  raizPrueba, montar, estadoSeleccionado, montarExpedienteSeleccionado,
  CORREO, HUELLA, archivoCorreo, comunicacionRegistrada, declaracion,
  justificante, abrirRespuesta, CLAVE_RESOLUCION, revisionManual,
  resolucionConfirmada, reciboResolucion, abrirResolucion,
} from "./formulario-llamamiento-pruebas.js?v=20261001-ct-firma-verificador-v2";

test("la vista de alta no monta el bloque de llamamiento sin expediente fiscalizado", async () => {
  const alta = { catalogos: {}, ejecutor: () => { throw new Error("no debe registrar otra petición"); } };
  for (const carga of ["listo", "error"]) {
    for (let reinicio = 0; reinicio < 2; reinicio += 1) {
      const montaje = await montarExpedienteSeleccionado({
        ...estadoSeleccionado(), vista: "alta", carga, cuadro: null,
        expediente: null, expediente_ref: "",
        mensaje_clave: carga === "error" ? "estado_error_carga" : "estado_inicial",
      }, alta);
      assert.equal(montaje.formulario().innerHTML, "");
      assert.equal(montaje.peticiones(), 0);
      montaje.desmontar();
    }
  }
});

test("cuarta operación exige justificante confirmado de aceptación o renuncia y no se solicita automáticamente", async () => {
  for (const opcion of ["aceptacion", "renuncia", "recibo_invalido"]) {
    const raiz = raizPrueba(); let llamadas = 0;
    const cerrar = await abrirRespuesta(raiz, {
      registrarRespuestaRecibida: async (s) => opcion === "recibo_invalido"
        ? { ...justificante(s), estado: "aceptada" } : justificante(s),
      resolverLlamamiento: async () => { llamadas += 1; },
    });
    await raiz.enviar("resolucion", { clave_idempotencia: CLAVE_RESOLUCION });
    assert.equal(llamadas, 0);
    delete raiz.borradores.resolucion; // El envío simulado no crea un formulario real.
    await raiz.archivo(archivoCorreo(opcion));
    await raiz.enviar("respuesta", { ...declaracion(), respuesta: opcion === "renuncia" ? opcion : "aceptacion" });
    assert.equal(llamadas, 0);
    if (opcion !== "recibo_invalido") {
      const formulario = raiz.innerHTML.match(/<form data-ct-llamamiento-form="resolucion"[\s\S]*?<\/form>/u)[0];
      assert.doesNotMatch(formulario, /clave_idempotencia|data-ct-llamamiento-clave|Clave de operación/u); assert.doesNotMatch(formulario, /type="file"|name="(?:actor_ref|estado_plazo|evaluacion_plazo_ref|politica_ref)"/u); assert.match(formulario, /name="respuesta" required disabled/u); assert.match(formulario, new RegExp(`value="${opcion}"\\s+selected`, "u"));
      for (const nombre of Object.keys(revisionManual)) {
        const control = formulario.match(new RegExp(`<input[^>]*name="${nombre}"[^>]*>`, "u"))[0];
        assert.match(control, /type="checkbox"/u); assert.doesNotMatch(control, /\schecked|\sdisabled/u); assert.match(formulario, new RegExp(`label for="ct-llamamiento-resolucion-${nombre}"`, "u"));
      }
      assert.match(formulario, /name="criterio_validacion_ref" value="politica:ct:revision-manual-sintetica:20260906"[^>]*readonly/u); assert.doesNotMatch(formulario, /sintétic|validacion-ayuda/u); assert.match(formulario, /He comprobado la respuesta y el correo/u);
      assert.match(formulario, /He revisado el plazo de respuesta/u);
      for (const campo of ["organizacion_ref", "expediente_ref", "llamamiento_ref",
        "comunicacion_ref", "version_esperada", "prueba_respuesta_ref"]) {
        assert.match(formulario, new RegExp(`name="${campo}"[^>]*readonly`, "u"));
      }
    } else {
      assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="resolucion"/u);
      await raiz.enviar("resolucion", { clave_idempotencia: CLAVE_RESOLUCION });
      assert.equal(llamadas, 0);
    }
    cerrar();
  }
});

for (const opcion of ["aceptacion", "renuncia"]) test(`resolución ${opcion} exige clave propia y confirmación; DOM no concede respuesta, antecedentes ni plazo`, async () => {
  const raiz = raizPrueba(), solicitudes = [], confirmaciones = []; let confirmar = false;
  const cerrar = await abrirResolucion(raiz, { resolverLlamamiento: async (s) => {
    solicitudes.push(s); return reciboResolucion(opcion);
  } }, { confirmarOperacion: (datos) => {
    if (!datos.datos.prueba_respuesta_ref) return true;
    confirmaciones.push(datos); return confirmar;
  } }, opcion);
  for (const [revision_respuesta_rrhh, revision_plazo_rrhh] of [[false, false], [true, false], [false, true], ["true", "true"]]) {
    await raiz.enviar("resolucion", { clave_idempotencia: CLAVE_RESOLUCION, revision_respuesta_rrhh, revision_plazo_rrhh });
    assert.equal(solicitudes.length, 0); assert.equal(confirmaciones.length, 0); assert.match(raiz.innerHTML, /Marque las dos casillas de comprobación\./u);
  }
  assert.equal(confirmaciones.length, 0);
  await raiz.enviar("resolucion", { clave_idempotencia: CLAVE_RESOLUCION, ...revisionManual });
  assert.equal(solicitudes.length, 0);
  confirmar = true;
  await raiz.enviar("resolucion", { clave_idempotencia: CLAVE_RESOLUCION, ...revisionManual,
    organizacion_ref: "org:inventada", expediente_ref: "exp:inventado", llamamiento_ref: "llam:inventado",
    comunicacion_ref: "com:inventada", version_esperada: "99", respuesta: opcion === "renuncia" ? "aceptacion" : "renuncia",
    prueba_respuesta_ref: "prueba:inventada", estado_plazo: "vigente", actor_ref: "actor:inventado",
    criterio_validacion_ref: "politica:inventada" });
  assert.deepEqual(solicitudes, [{ clave_idempotencia: CLAVE_RESOLUCION,
    organizacion_ref: recibo.organizacion_ref, expediente_ref: EXPEDIENTE,
    llamamiento_ref: recibo.llamamiento_ref, comunicacion_ref: comunicacionRegistrada.comunicacion_ref,
    version_esperada: 2, respuesta: opcion, prueba_respuesta_ref: justificante({}).justificante_ref,
    ...revisionManual, criterio_validacion_ref: "politica:ct:revision-manual-sintetica:20260906" }]);
  assert.ok(Object.isFrozen(solicitudes[0])); assert.match(confirmaciones.at(-1).advertencia, /Va a confirmar esta respuesta:/u); assert.doesNotMatch(confirmaciones.at(-1).advertencia, /politica:|justificante:|versión/u);
  assert.match(confirmaciones.at(-1).advertencia, new RegExp(`esta respuesta: ${opcion === "renuncia" ? "Renuncia" : "Aceptación"}\\.`, "u")); assert.match(raiz.innerHTML, /data-ct-llamamiento-recibo="resolucion"/u); assert.match(raiz.innerHTML, new RegExp(`${opcion === "renuncia" ? "Renuncia" : "Aceptación"} confirmada`, "u"));
  if (opcion === "renuncia") {
    assert.match(raiz.innerHTML, /Pendiente de llamar/u); assert.match(raiz.innerHTML, /intencion:siguiente:001/u); assert.match(raiz.innerHTML, /2026-09-05T09:05:00.12345Z/u); assert.doesNotMatch(raiz.innerHTML, /Todavía no se ha llamado a la siguiente persona\./u); assert.doesNotMatch(raiz.innerHTML, /Aceptación confirmada/u);
  } else assert.doesNotMatch(raiz.innerHTML, /Pendiente de llamar|intencion:siguiente:001|Renuncia confirmada|data-ct-llamamiento-form="siguiente"/u);
  assert.doesNotMatch(raiz.innerHTML, /El servidor confirma el registro manual en Contratación temporal y Bolsa/u); assert.match(raiz.innerHTML, /2026-09-05T09:05:00.123450Z/u); assert.equal(raiz.foco.at(-1), '[data-ct-llamamiento-recibo="resolucion"]');
  await raiz.enviar("resolucion", { clave_idempotencia: CLAVE_RESOLUCION });
  assert.equal(solicitudes.length, 1);
  cerrar();
});

test("respuesta RRHH se deriva del recibo v2; confirma datos y envía solo declaración y huella", async () => {
  const raiz = raizPrueba(), confirmaciones = [], solicitudes = [];
  const cerrar = await abrirRespuesta(raiz, { registrarRespuestaRecibida: async (s) => {
    solicitudes.push(s); return justificante(s);
  } }, { confirmarOperacion: (datos) => { confirmaciones.push(datos); return true; } });
  assert.match(raiz.innerHTML, /data-ct-llamamiento-form="respuesta"/u);
  for (const campo of ["organizacion_ref", "expediente_ref", "llamamiento_ref",
    "comunicacion_ref", "version_comunicacion_esperada", "correo_sha256"]) {
    assert.match(raiz.innerHTML, new RegExp(`name="${campo}"[^>]*readonly`, "u"));
  }
  await raiz.archivo(archivoCorreo());
  assert.equal(solicitudes.length, 0, "calcular la huella no registra nada"); assert.match(raiz.innerHTML, /Correo comprobado\./u); assert.match(raiz.innerHTML, /Fecha y hora en que llegó la respuesta/u);
  await raiz.enviar("respuesta", { ...declaracion(), organizacion_ref: "org:inventada",
    expediente_ref: "exp:inventado", llamamiento_ref: "llam:inventado",
    comunicacion_ref: "com:inventada", version_comunicacion_esperada: "99",
    correo_sha256: "f".repeat(64), contenido: CORREO, actor_ref: "actor:inventado" });
  assert.deepEqual(solicitudes, [{
    organizacion_ref: recibo.organizacion_ref, expediente_ref: EXPEDIENTE,
    llamamiento_ref: recibo.llamamiento_ref, comunicacion_ref: comunicacionRegistrada.comunicacion_ref,
    version_comunicacion_esperada: 2, respuesta: "aceptacion", correo_ref: declaracion().correo_ref,
    correo_sha256: HUELLA, recibida_en: "2026-09-05T08:30:00Z",
  }]);
  const confirmacion = confirmaciones.at(-1);
  assert.equal(confirmacion.referencia, EXPEDIENTE); assert.match(confirmacion.advertencia, /Va a anotar esta respuesta:/u); assert.match(raiz.innerHTML, /data-ct-llamamiento-recibo="respuesta"/u);
  assert.doesNotMatch(raiz.innerHTML, /Respuesta anotada\. El correo sigue en su buzón\./u); assert.match(raiz.innerHTML, /2026-09-05T09:00:00.123456Z/u); assert.doesNotMatch(raiz.innerHTML, /Subject:|respuesta-sintetica.eml|name="actor_ref"/u); assert.equal(raiz.foco.at(-1), '[data-ct-llamamiento-recibo="respuesta"]');
  assert.match(raiz.innerHTML, /Cómo va el llamamiento/u);
  assert.match(raiz.innerHTML, /<dd>Ha aceptado<\/dd>/u);
  assert.match(raiz.innerHTML, /Pendiente de que RRHH revise y confirme la respuesta\./u);
  assert.doesNotMatch(raiz.innerHTML, /Anotar la respuesta no la confirma: la confirma RRHH después/u);
  assert.match(raiz.innerHTML, /Plazo para responder/u);
  await raiz.enviar("respuesta", declaracion());
  assert.equal(solicitudes.length, 1);
  cerrar();
});

test("respuesta omite clave cliente y convierte la hora de Madrid sin cambiar el recibo", async () => {
  const raiz = raizPrueba(), solicitudes = [];
  await abrirRespuesta(raiz, { registrarRespuestaRecibida: async (s) => {
    solicitudes.push(s); return justificante(s);
  } }, { generarClaveIdempotencia: (operacion) => operacion === "respuesta"
    ? assert.fail("la respuesta no genera clave") : CLAVE });
  const formulario = raiz.innerHTML.match(/<form data-ct-llamamiento-form="respuesta"[\s\S]*?<\/form>/u)[0];
  assert.match(raiz.innerHTML, /Anotar la respuesta de la persona llamada/u);
  assert.match(formulario, /Ha aceptado/u);
  assert.match(formulario, /Ha renunciado/u);
  assert.doesNotMatch(formulario, /data-ct-llamamiento-clave="respuesta"|name="clave_idempotencia"|Huella SHA-256 declarada/u);
  await raiz.archivo(archivoCorreo());
  await raiz.enviar("respuesta", { ...declaracion(), clave_idempotencia: "", recibida_en: "2026-09-05T10:30" });
  assert.equal(Object.hasOwn(solicitudes[0], "clave_idempotencia"), false);
  assert.equal(solicitudes[0].recibida_en, "2026-09-05T08:30:00Z");
  assert.match(raiz.innerHTML, /Siguiente paso/u);
});

for (const estado of ["registrada_por_rrhh", "replay_registrada_por_rrhh"])
for (const ingles of [false, true]) test(`consulta 404 y POST ${estado} ${ingles ? "EN" : "ES"}: oculta aviso antiguo y conserva resolución`, async () => {
  const raiz = raizPrueba();
  const cerrar = await abrirRespuesta(raiz, { registrarRespuestaRecibida: async (solicitud) => ({
    ...justificante(solicitud), estado,
  }) }, ingles ? { mensajes: MENSAJES_LLAMAMIENTO_EN, locale: "en-GB" } : {});
  assert.match(raiz.innerHTML, ingles ? /No reply has been recorded yet\./u : /Todavía no hay ninguna respuesta anotada\./u);
  await raiz.archivo(archivoCorreo());
  await raiz.enviar("respuesta", declaracion());
  assert.doesNotMatch(raiz.innerHTML, /No consta una respuesta consultable|No reply can be viewed/u);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-recibo="respuesta"/u);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-form="resolucion"/u);
  cerrar();
});

const consultaRespuesta = { organizacion_ref: recibo.organizacion_ref,
  expediente_ref: EXPEDIENTE,
  comunicacion_ref: comunicacionRegistrada.comunicacion_ref };
const reciboConsultado = { esquema: "vec.contratacion-temporal.recibo-respuesta-llamamiento.v1",
  ...consultaRespuesta, respuesta: "aceptacion", justificante_ref: "justificante:sintetico:001",
  recibo_ref: "recibo:respuesta:001", auditoria_ref: "auditoria:respuesta:001",
  registrada_en: "2026-09-05T09:00:00.123456Z", estado: "registrada_por_rrhh" };

test("recarga con contexto autorizado consulta y muestra solo la respuesta ya registrada", async () => {
  const raiz = raizPrueba(); let consultas = 0, escrituras = 0;
  const cerrar = montar(raiz, {
    consultarReciboRespuesta: async (entrada, { signal }) => {
      consultas += 1; assert.deepEqual(entrada, consultaRespuesta); assert.equal(signal.aborted, false);
      return reciboConsultado;
    },
    registrarRespuestaRecibida: async () => { escrituras += 1; },
  }, { contexto: { expediente_ref: EXPEDIENTE, version_esperada: 6,
    consulta_respuesta: consultaRespuesta } });
  await new Promise(setImmediate);
  assert.equal(consultas, 1);
  assert.match(raiz.innerHTML, /Ya hay una respuesta anotada\./u);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-recibo="consultaRespuesta"/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form=/u);
  await raiz.enviar("respuesta", declaracion());
  assert.equal(escrituras, 0);
  cerrar();
});

test("consulta 404 permite registrar tras el aviso; 401/403 ocultan datos y bloquean POST", async () => {
  for (const estado of [404, 401, 403]) {
    const raiz = raizPrueba(); let escrituras = 0;
    const codigo = estado === 404 ? "recurso_no_encontrado"
      : estado === 401 ? "autenticacion_requerida" : "acceso_denegado";
    const cerrar = await abrirRespuesta(raiz, {
      consultarReciboRespuesta: async () => { throw Object.assign(new Error(), {
        estado, codigo, envelopeValido: true,
      }); },
      registrarRespuestaRecibida: async () => { escrituras += 1; },
    });
    if (estado === 404) {
      assert.match(raiz.innerHTML, /Todavía no hay ninguna respuesta anotada\./u);
      assert.match(raiz.innerHTML, /data-ct-llamamiento-form="respuesta"/u);
    } else {
      assert.match(raiz.innerHTML, /No tiene permiso para ver esta respuesta\. Si lo necesita, pí/u);
      assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form=|justificante:sintetico/u);
      await raiz.archivo(archivoCorreo());
      await raiz.enviar("respuesta", declaracion());
      assert.equal(escrituras, 0);
    }
    cerrar();
  }
});

test("404 tras recarga sin antecedente autorizado conserva cerrado el POST", async () => {
  const raiz = raizPrueba(); let escrituras = 0;
  const cerrar = montar(raiz, {
    consultarReciboRespuesta: async () => { throw Object.assign(new Error(), {
      estado: 404, codigo: "recurso_no_encontrado", envelopeValido: true,
    }); },
    registrarRespuestaRecibida: async () => { escrituras += 1; },
  }, { contexto: { expediente_ref: EXPEDIENTE, version_esperada: 6,
    consulta_respuesta: consultaRespuesta } });
  await new Promise(setImmediate);
  assert.match(raiz.innerHTML, /No hay ninguna respuesta anotada y no se puede anotar desde/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form=/u);
  await raiz.enviar("respuesta", declaracion());
  assert.equal(escrituras, 0);
  cerrar();
});

test("consulta temporal fallida bloquea POST hasta un 404 autorizado y permite reintentar", async () => {
  const raiz = raizPrueba(); let consultas = 0, escrituras = 0;
  const cerrar = await abrirRespuesta(raiz, {
    consultarReciboRespuesta: async () => {
      consultas += 1;
      if (consultas === 1) throw Object.assign(new Error(), {
        estado: 503, codigo: "servicio_no_disponible", envelopeValido: true,
      });
      throw Object.assign(new Error(), { estado: 404, codigo: "recurso_no_encontrado", envelopeValido: true });
    },
    registrarRespuestaRecibida: async () => { escrituras += 1; },
  });
  assert.match(raiz.innerHTML, /No se ha podido comprobar si ya hay respuesta\. Pulse «Volver/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form=/u);
  await raiz.archivo(archivoCorreo());
  await raiz.enviar("respuesta", declaracion());
  assert.equal(escrituras, 0);
  raiz.eventos.get("click")({ preventDefault() {}, target: { closest: (selector) =>
    selector === "[data-ct-llamamiento-reintentar-consulta]"
      ? { dataset: { ctLlamamientoReintentarConsulta: "" } } : null } });
  await new Promise(setImmediate);
  assert.equal(consultas, 2);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-form="respuesta"/u);
  cerrar();
});

test("consulta en inglés explica la denegación sin mostrar formulario ni recibo", async () => {
  const raiz = raizPrueba();
  const cerrar = montar(raiz, { consultarReciboRespuesta: async () => {
    throw Object.assign(new Error(), { estado: 403, codigo: "acceso_denegado", envelopeValido: true });
  } }, { mensajes: MENSAJES_LLAMAMIENTO_EN, locale: "en-GB",
    contexto: { expediente_ref: EXPEDIENTE, version_esperada: 6,
      consulta_respuesta: consultaRespuesta } });
  await new Promise(setImmediate);
  assert.match(raiz.innerHTML, /You do not have permission to see this reply\. If you need it/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form=|recibo:respuesta:001/u);
  cerrar();
});

test("consulta pendiente bloquea POST y se cancela al desmontar o cambiar de expediente", async () => {
  for (const cambioExpediente of [false, true]) {
    const raiz = raizPrueba(); let resolver, signal, escrituras = 0;
    const cerrar = montar(raiz, {
      consultarReciboRespuesta: (_entrada, opciones) => {
        signal = opciones.signal;
        return new Promise((resolve) => { resolver = resolve; });
      },
      registrarRespuestaRecibida: async () => { escrituras += 1; },
    }, { contexto: { expediente_ref: EXPEDIENTE, version_esperada: 6,
      consulta_respuesta: consultaRespuesta } });
    await new Promise(setImmediate);
    assert.match(raiz.innerHTML, /Comprobando si ya hay una respuesta anotada…/u);
    assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form=/u);
    await raiz.enviar("respuesta", declaracion());
    assert.equal(escrituras, 0);
    if (cambioExpediente) cerrar.actualizarContexto({ expediente_ref: "expediente:otro:002", version_esperada: 6 });
    else cerrar();
    assert.equal(signal.aborted, true);
    resolver(reciboConsultado);
    await new Promise(setImmediate);
    assert.equal(raiz.innerHTML, "");
  }
});

test("hora civil de Madrid distingue verano e invierno y rechaza saltos ambiguos", () => {
  assert.equal(fechaRespuestaMadridUTC("2026-01-15T10:30"), "2026-01-15T09:30:00Z");
  assert.equal(fechaRespuestaMadridUTC("2026-09-05T10:30"), "2026-09-05T08:30:00Z");
  assert.throws(() => fechaRespuestaMadridUTC("2026-03-29T02:30"), TypeError);
  assert.throws(() => fechaRespuestaMadridUTC("2026-10-25T02:30"), TypeError);
});

test("hora ambigua muestra una acción clara y no registra la respuesta", async () => {
  const raiz = raizPrueba(); let llamadas = 0;
  await abrirRespuesta(raiz, { registrarRespuestaRecibida: async () => { llamadas += 1; } });
  await raiz.archivo(archivoCorreo());
  await raiz.enviar("respuesta", { ...declaracion(), recibida_en: "2026-10-25T02:30" });
  assert.equal(llamadas, 0);
  assert.match(raiz.innerHTML, /Esa hora no existe o se repite por el cambio de hora\. Avise/u);
});

test("respuesta en inglés conserva campos obligatorios y explica el límite del correo", async () => {
  const raiz = raizPrueba();
  await abrirRespuesta(raiz, {}, { mensajes: MENSAJES_LLAMAMIENTO_EN, locale: "en-GB" });
  assert.match(raiz.innerHTML, /Record the reply from the person called/u);
  assert.match(raiz.innerHTML, /They accepted/u);
  assert.match(raiz.innerHTML, /They declined/u);
  assert.match(raiz.innerHTML, /Date and time the reply arrived/u);
  assert.match(raiz.innerHTML, /class="opcion-grande"[\s\S]*?You can then prepare the appointment proposal/u);
  // Los pasos para exportar el correo y sus límites están en la ayuda «?», no en pantalla.
  assert.doesNotMatch(raiz.innerHTML, /This is the proof of the reply|its content is not stored|2 MiB/u);
});

test("respuesta exige comunicación confirmada, archivo, datos y confirmación explícita", async () => {
  const raiz = raizPrueba(); let llamadas = 0;
  const cerrar = montar(raiz, { registrarRespuestaRecibida: async () => { llamadas += 1; } });
  await raiz.enviar("respuesta", declaracion());
  assert.equal(llamadas, 0);
  cerrar();
  const otra = raizPrueba();
  await abrirRespuesta(otra, { registrarRespuestaRecibida: async () => { llamadas += 1; } }, {
    confirmarOperacion: (datos) => !datos.datos.respuesta,
  });
  await otra.enviar("respuesta", declaracion());
  assert.match(otra.innerHTML, /Elija la respuesta, escriba la referencia del correo y la fe/u);
  await otra.archivo(archivoCorreo());
  await otra.enviar("respuesta", { ...declaracion(), respuesta: "expiracion_gobernada" });
  await otra.enviar("respuesta", declaracion());
  assert.equal(llamadas, 0); assert.doesNotMatch(otra.innerHTML, /data-ct-llamamiento-recibo="respuesta"/u);
});

test("correo limita tamaño antes de leer, vacía huella anterior y falla cerrado sin WebCrypto", async () => {
  for (const archivo of [null, { name: "otro.pdf", size: 10 },
    { name: "vacio.eml", size: 0 }, { name: "grande.eml", size: 2 * 1024 * 1024 + 1 }]) {
    const raiz = raizPrueba(); let llamadas = 0;
    await abrirRespuesta(raiz, { registrarRespuestaRecibida: async () => { llamadas += 1; } });
    await raiz.archivo(archivoCorreo());
    if (archivo) archivo.arrayBuffer = () => assert.fail("no debe leer");
    await raiz.archivo(archivo);
    await raiz.enviar("respuesta", declaracion());
    assert.equal(llamadas, 0); assert.match(raiz.innerHTML, /name="correo_sha256" type="hidden" value=""/u);
  }
  const raiz = raizPrueba();
  await abrirRespuesta(raiz, {}, { criptografia: null });
  await raiz.archivo(archivoCorreo());
  assert.match(raiz.innerHTML, /No se ha podido leer el correo\. Elija un archivo \.eml de has/u); assert.match(raiz.innerHTML, /name="correo_sha256" type="hidden" value=""/u);
});

test("huella admite exactamente 2 MiB y descarta bytes locales al terminar", async () => {
  const raiz = raizPrueba();
  await abrirRespuesta(raiz);
  const bytes = new Uint8Array(2 * 1024 * 1024).fill(65);
  const esperada = createHash("sha256").update(bytes).digest("hex");
  await raiz.archivo({ name: "limite.eml", size: bytes.length, arrayBuffer: async () => bytes.buffer });
  assert.match(raiz.innerHTML, new RegExp(`name="correo_sha256" type="hidden" value="${esperada}"`, "u")); assert.ok(bytes.every((b) => b === 0));
});

test("huella en curso bloquea envío y desmontar descarta su resolución tardía", async () => {
  const raiz = raizPrueba(); let resolver, llamadas = 0;
  const cerrar = await abrirRespuesta(raiz, { registrarRespuestaRecibida: () => { llamadas += 1; } });
  const bytes = new Uint8Array([65]);
  const pendiente = raiz.archivo({ name: "pendiente.eml", size: 1,
    arrayBuffer: () => new Promise((resolve) => { resolver = resolve; }) });
  assert.match(raiz.innerHTML, /Comprobando el correo…/u);
  await raiz.enviar("respuesta", declaracion());
  assert.equal(llamadas, 0);
  cerrar();
  resolver(bytes.buffer);
  await pendiente;
  assert.equal(raiz.innerHTML, ""); assert.equal(raiz.eventos.size, 0); assert.equal(bytes[0], 0);
});

test("respuesta perdida conserva fecha, declaración y huella; reintento usa el mismo contenido", async () => {
  const raiz = raizPrueba(), solicitudes = [];
  await abrirRespuesta(raiz, { registrarRespuestaRecibida: async (s) => {
    solicitudes.push(s);
    if (solicitudes.length === 1) throw new Error("transporte interrumpido");
    if (solicitudes.length === 2) throw Object.assign(new Error("permiso de replay denegado"), {
      codigo: "acceso_denegado", envelopeValido: true, resultadoIndeterminado: false,
    });
    return { ...justificante(s), estado: "replay_registrada_por_rrhh" };
  } });
  await raiz.archivo(archivoCorreo());
  await raiz.enviar("respuesta", declaracion());
  assert.match(raiz.innerHTML, /Volver a intentar/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-clave="respuesta"/u); // gitleaks:allow — selector HTML, no credencial.
  await raiz.archivo(new File(["otro"], "otro.eml"));
  await raiz.enviar("respuesta", { ...declaracion(), respuesta: "renuncia",
    clave_idempotencia: CLAVE, correo_ref: "correo:otro", recibida_en: "2026-09-06T10:00" });
  assert.deepEqual(solicitudes[0], solicitudes[1]); assert.equal(solicitudes[0].correo_sha256, HUELLA);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-clave="respuesta"/u); // gitleaks:allow — selector HTML, no credencial.
  await raiz.enviar("respuesta", { ...declaracion(), respuesta: "renuncia" });
  assert.deepEqual(solicitudes[0], solicitudes[2]); assert.match(raiz.innerHTML, /Ya estaba anotada; no se ha repetido/u);
});

test("doble envío de respuesta no duplica y conflicto impide reintentar", async () => {
  const raiz = raizPrueba(); let resolver; const solicitudes = [];
  await abrirRespuesta(raiz, { registrarRespuestaRecibida: (s) => {
    solicitudes.push(s); return new Promise((resolve) => { resolver = resolve; });
  } });
  await raiz.archivo(archivoCorreo());
  const pendiente = raiz.enviar("respuesta", declaracion());
  await raiz.enviar("respuesta", declaracion());
  assert.equal(solicitudes.length, 1);
  resolver(justificante(solicitudes[0]));
  await pendiente;
  for (const codigo of ["version_en_conflicto", "contenido_respuesta_en_conflicto"]) {
    const otra = raizPrueba(); let llamadas = 0;
    await abrirRespuesta(otra, { registrarRespuestaRecibida: async () => {
      llamadas += 1; throw Object.assign(new Error(), { codigo, envelopeValido: true });
    } });
    await otra.archivo(archivoCorreo());
    await otra.enviar("respuesta", declaracion());
    await otra.enviar("respuesta", declaracion());
    assert.equal(llamadas, 1);
    assert.match(otra.innerHTML, codigo === "contenido_respuesta_en_conflicto"
      ? /Ya hay otra respuesta anotada para este llamamiento\. Abra el/u : /No se ha podido guardar porque el expediente ha cambiado\. No/u);
    assert.doesNotMatch(otra.innerHTML, /data-ct-llamamiento-recibo="respuesta"/u);
  }
});

test("conflicto de contenido en inglés bloquea otro POST y remite al expediente", async () => {
  const raiz = raizPrueba(); let llamadas = 0;
  await abrirRespuesta(raiz, { registrarRespuestaRecibida: async () => {
    llamadas += 1;
    throw Object.assign(new Error(), { codigo: "contenido_respuesta_en_conflicto", envelopeValido: true });
  } }, { mensajes: MENSAJES_LLAMAMIENTO_EN, locale: "en-GB" });
  await raiz.archivo(archivoCorreo());
  await raiz.enviar("respuesta", declaracion());
  await raiz.enviar("respuesta", declaracion());
  assert.equal(llamadas, 1);
  assert.match(raiz.innerHTML, /Another reply is already recorded for this call-up\. Open the/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-recibo="respuesta"/u);
});

test("el expediente fiscalizado seleccionado rellena el formulario sin POST y sin enseñar referencia, versión ni clave", async () => {
  const montaje = await montarExpedienteSeleccionado(estadoSeleccionado());
  const html = montaje.formulario().innerHTML;
  assert.match(html, /Llamamiento de este expediente/u); assert.match(html, /Llamar a la siguiente persona/u);
  assert.match(html, /id="ct-llamamiento-seleccion-expediente_ref"[^>]*value="expediente:ct:sintetico:001"[^>]*type="hidden"/u); assert.match(html, /id="ct-llamamiento-seleccion-version_esperada"[^>]*value="6"[^>]*type="hidden"/u);
  assert.doesNotMatch(html, /clave_idempotencia|data-ct-llamamiento-clave|Versión 6 del expediente|Referencia del expediente|Versión esperada/u);
  assert.equal(montaje.peticiones(), 0);
  montaje.desmontar();
});

test("no monta llamamiento para un detalle sin cargar, desfasado, ajeno o no fiscalizado", async () => {
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
    assert.equal(montaje.formulario().innerHTML, ""); assert.equal(montaje.formulario().eventos.size, 0); assert.equal(montaje.peticiones(), 0);
    montaje.desmontar();
  }
});

test("abrir otro expediente no reutiliza referencias ni clave del formulario anterior", async () => {
  const montaje = await montarExpedienteSeleccionado(estadoSeleccionado());
  const anterior = montaje.formulario();
  anterior.preparar("seleccion", seleccion());
  await montaje.abrir("expediente:ct:sintetico:002");
  const actual = montaje.formulario();
  assert.notEqual(actual, anterior); assert.equal(anterior.eventos.size, 0); assert.match(actual.innerHTML, /id="ct-llamamiento-seleccion-expediente_ref"[^>]*value="expediente:ct:sintetico:002"/u); assert.doesNotMatch(actual.innerHTML, /clave_idempotencia/u);
  assert.doesNotMatch(actual.innerHTML, /expediente:ct:sintetico:001/u); assert.equal(montaje.peticiones(), 0);
  montaje.desmontar();
});

test("enlaza fiscalización y exige confirmación sin inventar candidato ni autoridad", async () => {
  const raiz = raizPrueba(); let llamadas = 0;
  const desmontar = montar(raiz, { seleccionarLlamamiento: () => { llamadas += 1; } },
    { confirmarOperacion: () => false });
  assert.equal(desmontar.actualizarContexto({ expediente_ref: EXPEDIENTE, version_esperada: 6 }), true); assert.match(raiz.innerHTML, /Llamamiento de este expediente/u); assert.match(raiz.innerHTML, /value="expediente:ct:sintetico:001"/u);
  await raiz.enviar("seleccion", seleccion());
  assert.equal(llamadas, 0); assert.doesNotMatch(raiz.innerHTML, /candidatura_ref|name="actor_ref"/u);
  desmontar();
  assert.equal(raiz.eventos.size, 0);
});

test("doble envío no duplica operación y el recibo minimizado abre comunicación", async () => {
  const raiz = raizPrueba(), solicitudes = []; let resolver;
  montar(raiz, { seleccionarLlamamiento: (solicitud) => {
    solicitudes.push(solicitud); return new Promise((resolve) => { resolver = resolve; });
  } });
  const primera = raiz.enviar("seleccion", seleccion());
  await raiz.enviar("seleccion", { ...seleccion(), clave_idempotencia: "otra" });
  assert.equal(solicitudes.length, 1);
  resolver(recibo);
  await primera;
  assert.match(raiz.innerHTML, /<details data-ct-llamamiento-datos-registrados="seleccion"><summary>Ver los datos guardados<\/summary>/u);
  assert.match(raiz.innerHTML, /<\/form>\s*<\/details>[\s\S]*?data-ct-llamamiento-recibo="seleccion"/u);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-recibo="seleccion"/u); assert.match(raiz.innerHTML, /data-ct-llamamiento-comunicacion open/u); assert.doesNotMatch(raiz.innerHTML, /Persona llamada\. El justificante no muestra sus datos\./u); assert.equal(solicitudes[0].version_esperada, 6);
});

test("respuesta perdida: recuperación usa petición congelada aunque cambien controles", async () => {
  const raiz = raizPrueba(), solicitudes = [];
  montar(raiz, { seleccionarLlamamiento: async (solicitud) => {
    solicitudes.push(solicitud);
    if (solicitudes.length === 1) throw new Error("red");
    return recibo;
  } });
  await raiz.enviar("seleccion", seleccion());
  assert.match(raiz.innerHTML, /Volver a intentar/u);
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
  assert.deepEqual(solicitudes[0], solicitudes[1]); assert.match(segunda.innerHTML, /Guardado/u);
});

test("conflicto no reintentable mantiene datos y no permite otra escritura", async () => {
  const raiz = raizPrueba(); let llamadas = 0;
  montar(raiz, { seleccionarLlamamiento: async () => {
    llamadas += 1;
    throw Object.assign(new Error("conflicto"), { codigo: "conflicto_no_reintentable" });
  } });
  await raiz.enviar("seleccion", seleccion());
  await raiz.enviar("seleccion", seleccion());
  assert.equal(llamadas, 1); assert.match(raiz.innerHTML, /No se ha podido guardar porque el expediente ha cambiado\. No/u);
});

test("comunicación exige sus referencias y muestra registro, no envío de correo", async () => {
  const raiz = raizPrueba(); let solicitud;
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
  assert.equal(solicitud.version_esperada, 1); assert.doesNotMatch(raiz.innerHTML, /Aviso anotado\. VEC no envía el correo\./u); assert.doesNotMatch(raiz.innerHTML, /Aviso anotado\. VEC no envía el correo\./u);
});

test("desmontar cancela la espera y una respuesta tardía no repinta", async () => {
  const raiz = raizPrueba(); let resolver, signal;
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

test("encadenado selección y replay autorrellenan comunicación local sin transcribir referencias", async () => {
  const claveComunicacion = "123e4567-e89b-42d3-a456-426614174001";
  const llamadas = [];
  for (const estadoHTTP of [201, 200]) {
    const raiz = raizPrueba();
    const cliente = crearClienteHTTPContratacionTemporal({
      fetchImpl: async (ruta, opciones) => {
        if (opciones.method === "GET") return new Response(JSON.stringify({ error: {
          codigo: "recurso_no_encontrado",
          clave_i18n: "api.contratacion_temporal.respuesta_recibida.error.recurso_no_encontrado",
          correlacion_ref: "corr_0123456789abcdef0123456789abcdef",
        } }), { status: 404, headers: { "Content-Type": "application/json; charset=utf-8" } });
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
    // Sin lista de llamamientos previos: el expediente parte de cero.
    const cerrar = montar(raiz, { ...cliente, consultarComunicacionesExpediente: undefined });
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
    assert.match(raiz.innerHTML, /data-ct-llamamiento-recibo="comunicacion"/u); assert.match(raiz.innerHTML, /Anotado/u); assert.doesNotMatch(raiz.innerHTML, /Plazo para responder/u);
    cerrar();
  }
  assert.deepEqual(llamadas.slice(0, 2), llamadas.slice(2));
});

test("aceptación tras subsanación ofrece propuesta v8 y conserva recibo v9", async () => {
 const raiz = raizPrueba(), solicitudes = [];
 const cerrar = await abrirResolucion(raiz, {
  resolverLlamamiento: async () => resolucionConfirmada,
  prepararPropuestaFormalizacion: async (s) => {
   solicitudes.push(s);
   return { esquema: "vec.contratacion-temporal.propuesta-formalizacion-local.v1", estado_local: "confirmado",
    propuesta_ref: "propuesta:sintetica:posterior", recibo_local_ref: "recibo:propuesta:posterior",
    version_resultante: 9, confirmada_en: "2026-09-06T10:00:00.123456Z" };
  },
 }, { contexto: { expediente_ref: EXPEDIENTE, version_esperada: 8 },
  fetchPublicaciones: async () => new Response(PUBLICACIONES_PROPUESTA,
   { status: 200, headers: { "Content-Type": "application/json" } }) });
 await raiz.enviar("resolucion", { clave_idempotencia: CLAVE_RESOLUCION, ...revisionManual });
 assert.match(raiz.innerHTML, /id="ct-llamamiento-propuesta-version_esperada"[^>]*value="8"[^>]*readonly/u);
 assert.equal(solicitudes.length, 0);
 await raiz.enviar("propuesta", { clave_idempotencia: "123e4567-e89b-42d3-a456-426614174005", version_esperada: 6 });
 assert.equal(solicitudes.length, 1);
 assert.equal(solicitudes[0].version_esperada, 8);
 assert.match(raiz.innerHTML, /Propuesta guardada/u);
 cerrar();
});

test("revisar propuesta conserva la navegación y enfoca su formulario sin enviar", () => {
  const raiz = raizPrueba();
  const desmontar = montar(raiz);
  const anterior = raiz.innerHTML;
  let cancelado = false;
  const enlace = { dataset: { ctPropuestaFormalizacionSiguiente: "" } };
  raiz.eventos.get("click")({
    target: { closest: selector => selector === "[data-ct-propuesta-formalizacion-siguiente]" ? enlace : null },
    preventDefault() { cancelado = true; },
  });
  assert.equal(cancelado, true);
  assert.equal(raiz.foco.at(-1), "#ct-llamamiento-propuesta-titulo");
  assert.equal(raiz.innerHTML, anterior);
  desmontar();
});

test("sin expediente abierto no hay formulario ni campos de referencia o versión que teclear", async () => {
  const raiz = raizPrueba(); let llamadas = 0;
  const cerrar = montar(raiz, { seleccionarLlamamiento: async () => { llamadas += 1; return recibo; } }, { contexto: null });
  assert.match(raiz.innerHTML, /data-ct-llamamiento-sin-expediente/u);
  assert.match(raiz.innerHTML, /Abra el expediente desde la lista de peticiones/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form=|expediente_ref|version_esperada|clave_idempotencia/u);
  await raiz.enviar("seleccion", seleccion());
  assert.equal(llamadas, 0);
  cerrar();
});

test("la clave se genera sola al enviar, no se ve y se reutiliza al reintentar; tras un rechazo, otra", async () => {
  const raiz = raizPrueba(), solicitudes = [], claves = [];
  let fallo = "red";
  montar(raiz, { seleccionarLlamamiento: async (solicitud) => {
    solicitudes.push(solicitud);
    if (fallo === "red") { fallo = "rechazo"; throw new Error("red"); }
    if (fallo === "rechazo") {
      fallo = "";
      throw Object.assign(new Error(), { envelopeValido: true, resultadoIndeterminado: false, codigo: "acceso_denegado" });
    }
    return recibo;
  } }, { generarClaveIdempotencia: (operacion) => {
    const clave = `123e4567-e89b-42d3-a456-42661417400${claves.length}`;
    claves.push([operacion, clave]); return clave;
  } });
  assert.doesNotMatch(raiz.innerHTML, /clave_idempotencia|data-ct-llamamiento-clave|Clave de operación/u);
  await raiz.enviar("seleccion", {});
  assert.deepEqual(claves, [["seleccion", "123e4567-e89b-42d3-a456-426614174000"]]);
  assert.equal(solicitudes[0].clave_idempotencia, claves[0][1]);
  // Resultado incierto: el reintento envía la misma petición, con la misma clave.
  await raiz.enviar("seleccion", {});
  assert.equal(solicitudes.length, 2); assert.strictEqual(solicitudes[1], solicitudes[0]); assert.equal(claves.length, 1);
  assert.doesNotMatch(raiz.innerHTML, /123e4567-e89b-42d3-a456-426614174000/u);
  // Rechazo de ese reintento: el original pudo surtir efecto, así que la clave se conserva.
  await raiz.enviar("seleccion", {});
  assert.equal(solicitudes.length, 3); assert.equal(solicitudes[2].clave_idempotencia, claves[0][1]); assert.equal(claves.length, 1);
});

test("tras rechazar el primer intento sin efecto, el siguiente lleva una clave nueva", async () => {
  const raiz = raizPrueba(), solicitudes = []; let generadas = 0;
  montar(raiz, { seleccionarLlamamiento: async (solicitud) => {
    solicitudes.push(solicitud);
    if (solicitudes.length === 1) {
      throw Object.assign(new Error(), { envelopeValido: true, resultadoIndeterminado: false, codigo: "acceso_denegado" });
    }
    return recibo;
  } }, { generarClaveIdempotencia: () => `123e4567-e89b-42d3-a456-42661417401${generadas++}` });
  await raiz.enviar("seleccion", {});
  assert.match(raiz.innerHTML, /No se ha guardado\. Revise los datos/u);
  await raiz.enviar("seleccion", {});
  assert.equal(solicitudes.length, 2); assert.notEqual(solicitudes[0].clave_idempotencia, solicitudes[1].clave_idempotencia);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-recibo="seleccion"/u);
});

test("una clave generada que coincide con la de otra operación se descarta y se genera otra", async () => {
  const raiz = raizPrueba(), solicitudes = [];
  const generadas = [CLAVE, "123e4567-e89b-42d3-a456-426614174009"];
  const cerrar = await abrirResolucion(raiz, { resolverLlamamiento: async (s) => {
    solicitudes.push(s); return reciboResolucion("aceptacion");
  } }, { generarClaveIdempotencia: (operacion) => operacion === "resolucion" ? generadas.shift() : CLAVE });
  await raiz.enviar("resolucion", revisionManual);
  assert.equal(solicitudes.length, 0); assert.equal(generadas.length, 1);
  await raiz.enviar("resolucion", revisionManual);
  assert.equal(solicitudes.length, 1); assert.equal(solicitudes[0].clave_idempotencia, "123e4567-e89b-42d3-a456-426614174009");
  cerrar();
});
