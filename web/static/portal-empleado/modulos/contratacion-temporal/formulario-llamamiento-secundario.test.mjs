import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { File } from "node:buffer";
import test from "node:test";
import { crearClienteHTTPContratacionTemporal } from "./cliente-http.js";
import {
  CLAVE, EXPEDIENTE, PUBLICACIONES_PROPUESTA, seleccion, recibo,
  raizPrueba, montar, CORREO, HUELLA, archivoCorreo, comunicacionRegistrada,
  declaracion, justificante, abrirRespuesta, CLAVE_RESOLUCION,
  CLAVE_RESOLUCION_SIGUIENTE, revisionManual, resolucionConfirmada,
  reciboResolucion, reciboResolucionSucesor, solicitudSiguiente,
  continuacionConfirmada, solicitudAvisoSiguiente, avisoSiguienteRegistrado,
  declaracionSiguiente, justificanteSiguiente, abrirResolucion,
  abrirSiguiente, abrirRespuestaSiguiente, abrirResolucionSucesor,
} from "./formulario-llamamiento-pruebas.js";

for (const sucesor of [false, true])
for (const caso of ["confirmado", "renuncia", "asset_invalido", "ambiguo", "fecha_anterior", "conflicto", "tardia"]) test(`propuesta ${sucesor ? "sucesor" : "original"}/${caso}: aceptación y publicaciones reales, misma clave y ningún efecto implícito`, async () => {
  const raiz = raizPrueba(), solicitudes = [], confirmaciones = []; let lecturas = 0, actualizaciones = 0, confirmar = false, liberar, avisar, señal;
  const inicio = new Promise((resolve) => { avisar = resolve; });
  const operacionId = sucesor ? "123e4567-e89b-42d3-a456-426614174009" : "123e4567-e89b-42d3-a456-426614174005";
  const operacionResolucion = sucesor ? "resolucion_siguiente" : "resolucion";
  const resolucionId = sucesor ? CLAVE_RESOLUCION_SIGUIENTE : CLAVE_RESOLUCION;
  const aceptacion = sucesor ? reciboResolucionSucesor("aceptacion") : resolucionConfirmada;
  const llamamientoRef = sucesor ? continuacionConfirmada.llamamiento_ref : recibo.llamamiento_ref;
  const previos = () => [...raiz.innerHTML.matchAll(/<section[^>]*data-ct-llamamiento-recibo="(?!propuesta")[^"]+"[\s\S]*?<\/section>/gu)].map((m) => m[0]);
  const resultado = { esquema: "vec.contratacion-temporal.propuesta-formalizacion-local.v1", estado_local: "confirmado",
    propuesta_ref: "propuesta:sintetica:001", recibo_local_ref: "recibo:propuesta:001", version_resultante: 7,
    confirmada_en: "2026-09-06T10:00:00.123456Z" };
  const respuestaPublica = () => new Response(caso === "asset_invalido" ? "{}" : PUBLICACIONES_PROPUESTA,
    { status: 200, headers: { "Content-Type": "application/json" } });
  const opcion = caso === "renuncia" ? "renuncia" : "aceptacion";
  const cerrar = await (sucesor ? abrirResolucionSucesor : abrirResolucion)(raiz, {
    resolverLlamamiento: async () => (sucesor ? reciboResolucionSucesor : reciboResolucion)(opcion),
    prepararPropuestaFormalizacion: async (s) => {
      solicitudes.push(s);
      if (caso === "ambiguo" && solicitudes.length === 1) throw new Error("red");
      if (caso === "conflicto") throw Object.assign(new Error(), { envelopeValido: true,
        codigo: "resolucion_no_aceptada", resultadoIndeterminado: true });
      return { ...resultado, estado_local: solicitudes.length > 1 ? "replay_confirmado" : "confirmado",
        // En sucesor, una fecha posterior a la resolución raíz pero anterior a
        // su aceptación también debe rechazarse: se coteja el antecedente propio.
        ...(caso === "fecha_anterior" && solicitudes.length === 1
          ? { confirmada_en: sucesor ? "2026-09-05T09:07:00Z" : "2026-09-05T09:04:00Z" } : {}) };
    },
  }, { fetchPublicaciones: async (_, opciones) => {
    lecturas += 1; señal = opciones.signal; avisar();
    if (caso === "tardia") return new Promise((resolve) => { liberar = resolve; });
    return respuestaPublica();
  }, alActualizarPropuesta: async () => { actualizaciones += 1; return false; }, confirmarOperacion: ({ titulo, advertencia }) => {
    if (!titulo.startsWith("Propuesta")) return true;
    confirmaciones.push(advertencia);
    assert.ok(advertencia.includes(llamamientoRef)); assert.ok(advertencia.includes(aceptacion.resolucion_ref));
    assert.match(advertencia, /borrador sin datos ni firma/u); return confirmar;
  } }, opcion);
  assert.equal(lecturas, 0); assert.equal(solicitudes.length, 0);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="propuesta"/u);
  await raiz.enviar("propuesta", { clave_idempotencia: operacionId });
  assert.equal(solicitudes.length, 0); assert.equal(confirmaciones.length, 0);
  delete raiz.borradores.propuesta;
  const resolviendo = raiz.enviar(operacionResolucion, { clave_idempotencia: resolucionId, ...revisionManual });
  if (caso === "tardia") {
    await inicio; cerrar(); assert.equal(señal.aborted, true); liberar(respuestaPublica());
    await resolviendo; assert.equal(raiz.innerHTML, ""); assert.equal(solicitudes.length, 0); return;
  }
  await resolviendo; assert.equal(solicitudes.length, 0);
  if (caso === "renuncia" || caso === "asset_invalido") {
    await raiz.enviar("propuesta", { clave_idempotencia: operacionId }); assert.equal(solicitudes.length, 0);
    if (caso === "renuncia") { assert.equal(lecturas, 0); assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="propuesta"/u); }
    else assert.match(raiz.innerHTML, /Publicaciones de formalización no disponibles/u);
    cerrar(); return;
  }
  const anteriores = previos(); assert.equal(anteriores.length, sucesor ? 8 : 4);
  assert.equal(lecturas, 1); assert.match(raiz.innerHTML, /id="ct-llamamiento-propuesta-version_esperada"[^>]*value="6"[^>]*readonly/u);
  assert.equal([...raiz.innerHTML.matchAll(/data-ct-llamamiento-form="propuesta"/gu)].length, 1);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="(?:propuesta_siguiente|siguiente_siguiente)"/u);
  for (const clave_idempotencia of [CLAVE, declaracion().clave_idempotencia, CLAVE_RESOLUCION,
    ...(sucesor ? [solicitudSiguiente.clave_idempotencia, solicitudAvisoSiguiente.clave_idempotencia,
      declaracionSiguiente().clave_idempotencia, CLAVE_RESOLUCION_SIGUIENTE] : [])])
    await raiz.enviar("propuesta", { clave_idempotencia });
  assert.equal(solicitudes.length, 0); assert.equal(confirmaciones.length, 0);
  await raiz.enviar("propuesta", { clave_idempotencia: operacionId }); assert.equal(solicitudes.length, 0);
  confirmar = true;
  await raiz.enviar("propuesta", { clave_idempotencia: operacionId, version_esperada: 3, expediente_ref: "expediente:ajeno",
    llamamiento_ref: "llamamiento:ajeno", resolucion_llamamiento_aceptada_ref: "resolucion:ajena",
    recibo_resolucion_aceptada_ref: "recibo:ajeno",
    tipo_formalizacion: {}, plantilla: {}, politica_firma: {}, plan_firma: {}, anexos: [{}] });
  assert.equal(solicitudes.length, 1); assert.equal(solicitudes[0].version_esperada, 6); assert.equal(solicitudes[0].expediente_ref, EXPEDIENTE); assert.equal(solicitudes[0].llamamiento_ref, llamamientoRef); assert.equal(solicitudes[0].resolucion_llamamiento_aceptada_ref, aceptacion.resolucion_ref);
  assert.equal(solicitudes[0].recibo_resolucion_aceptada_ref, aceptacion.recibo_local_ref); assert.equal(Object.keys(solicitudes[0]).length, 11); assert.ok(Object.isFrozen(solicitudes[0])); assert.ok(Object.isFrozen(solicitudes[0].plantilla)); assert.deepEqual(solicitudes[0].anexos, []);
  if (caso !== "confirmado") assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-recibo="propuesta"/u);
  await raiz.enviar("propuesta", { clave_idempotencia: CLAVE, version_esperada: 99 });
  assert.equal(solicitudes.length, ["ambiguo", "fecha_anterior"].includes(caso) ? 2 : 1);
  if (["ambiguo", "fecha_anterior"].includes(caso)) {
    assert.strictEqual(solicitudes[0], solicitudes[1]);
    assert.match(raiz.innerHTML, /Recuperado sin repetir el efecto/u);
    assert.equal(solicitudes[1].version_esperada, 6); assert.match(raiz.innerHTML, /2026-09-06T10:00:00.123456Z/u);
  }
  if (caso !== "conflicto") assert.match(raiz.innerHTML, /Propuesta registrada · ejercicio sintético/u);
  if (caso === "confirmado") {
    assert.match(raiz.innerHTML, /data-ct-llamamiento-actualizar-propuesta/u);
    await raiz.eventos.get("click")({ target: { closest: (selector) => selector === "[data-ct-llamamiento-actualizar-propuesta]"
      ? { dataset: { ctLlamamientoActualizarPropuesta: "" } } : null }, preventDefault() {} });
    assert.equal(actualizaciones, 1);
    assert.match(raiz.innerHTML, /La actualización del expediente sigue pendiente; no repita la propuesta/u);
    assert.match(raiz.innerHTML, /data-ct-llamamiento-recibo="propuesta"/u);
    assert.equal(solicitudes.length, 1);
  }
  assert.deepEqual(previos(), anteriores); assert.equal(lecturas, 1); cerrar();
});
test("la carga fallida de publicaciones se puede recuperar sin repetir la aceptación", async () => {
  const raiz = raizPrueba(); let lecturas = 0;
  const cerrar = await abrirResolucion(raiz, {
    resolverLlamamiento: async () => reciboResolucion("aceptacion"),
    prepararPropuestaFormalizacion: async () => assert.fail("no debe registrar propuesta"),
  }, {
    fetchPublicaciones: async () => {
      lecturas += 1;
      return new Response(lecturas === 1 ? "{}" : PUBLICACIONES_PROPUESTA,
        { status: 200, headers: { "Content-Type": "application/json" } });
    },
  });
  await raiz.enviar("resolucion", { clave_idempotencia: CLAVE_RESOLUCION, ...revisionManual });
  assert.equal(lecturas, 1);
  assert.match(raiz.innerHTML, /Publicaciones de formalización no disponibles/u);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-reintentar-publicaciones/u);
  await raiz.eventos.get("click")({
    target: { closest: (selector) => selector === "[data-ct-llamamiento-reintentar-publicaciones]"
      ? { dataset: { ctLlamamientoReintentarPublicaciones: "" } } : null },
    preventDefault() {},
  });
  assert.equal(lecturas, 2);
  assert.match(raiz.innerHTML, /Publicaciones de desarrollo disponibles/u);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-form="propuesta"/u);
  cerrar();
});
test("siguiente exige renuncia, clave propia y confirmación; no modifica recibos ni la primera comunicación", async () => {
  const raiz = raizPrueba(), solicitudes = [], confirmaciones = []; let confirmar = false, claves = 0;
  const renuncia = reciboResolucion("renuncia"), original = JSON.stringify(renuncia);
  const cerrar = await abrirResolucion(raiz, { resolverLlamamiento: async () => renuncia,
    continuarLlamamiento: async (s) => { solicitudes.push(s); return continuacionConfirmada; },
  }, { confirmarOperacion: (d) => { if (!d.datos.intencion_ref) return true; confirmaciones.push(d); return confirmar; },
    generarClaveIdempotencia: () => { claves += 1; return solicitudSiguiente.clave_idempotencia; } }, "renuncia");
  await raiz.enviar("siguiente", solicitudSiguiente); assert.equal(solicitudes.length, 0);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="siguiente"/u);
  delete raiz.borradores.siguiente; // El intento simulado no crea un formulario real.
  await raiz.enviar("resolucion", { clave_idempotencia: CLAVE_RESOLUCION, ...revisionManual });
  const form = raiz.innerHTML.match(/<form data-ct-llamamiento-form="siguiente"[\s\S]*?<\/form>/u)[0];
  assert.match(form, /name="clave_idempotencia" value=""/u);
  for (const campo of Object.keys(solicitudSiguiente).slice(1)) assert.match(form, new RegExp(`name="${campo}"[^>]*readonly`, "u"));
  assert.doesNotMatch(form, /checkbox|type="file"|name="(?:actor_ref|version_esperada|llamamiento_ref)"/u);
  const pulsarClave = () => raiz.eventos.get("click")({ preventDefault() {}, target: { closest: () => ({ dataset: { ctLlamamientoClave: "siguiente" } }) } });
  assert.equal(claves, 0); pulsarClave(); assert.equal(claves, 1);
  for (const clave_idempotencia of [CLAVE, CLAVE_RESOLUCION, declaracion().clave_idempotencia]) await raiz.enviar("siguiente", { clave_idempotencia });
  await raiz.enviar("siguiente", solicitudSiguiente); assert.equal(solicitudes.length, 0);
  confirmar = true;
  await raiz.enviar("siguiente", { ...solicitudSiguiente, organizacion_ref: "org:ajena", expediente_ref: "exp:ajeno",
    resolucion_ref: "resolucion:ajena", intencion_ref: "intencion:ajena", actor_ref: "actor:inventado" });
  assert.deepEqual(solicitudes, [solicitudSiguiente]); assert.ok(Object.isFrozen(solicitudes[0])); assert.match(confirmaciones.at(-1).advertencia, /abrirá un único nuevo llamamiento después de la renuncia/u); assert.equal(confirmaciones.at(-1).referencia, EXPEDIENTE); assert.match(raiz.innerHTML, /Siguiente llamamiento abierto · ejercicio sintético/u);
  assert.match(raiz.innerHTML, /Recibo histórico de renuncia/u); assert.match(raiz.innerHTML, /2026-09-05T09:06:00.123456Z/u); assert.doesNotMatch(raiz.innerHTML.match(/<form data-ct-llamamiento-form="siguiente"[\s\S]*?<\/form>/u)[0], /type="submit"/u); assert.equal(JSON.stringify(renuncia), original);
  for (const campo of ["llamamiento_anterior_ref", "llamamiento_ref", "recibo_bolsa_ref", "auditoria_ref", "intencion_ref"])
    assert.ok(raiz.innerHTML.includes(continuacionConfirmada[campo])); assert.match(raiz.innerHTML, /id="ct-llamamiento-comunicacion-llamamiento_ref"[^>]*value="llamamiento:sintetico:001"/u);
  await raiz.enviar("siguiente", solicitudSiguiente); pulsarClave();
  assert.equal(solicitudes.length, 1); assert.equal(claves, 1); cerrar();
});
for (const fallo of ["red", "anterior_ajeno", "rechazo", "conflicto"]) test(`siguiente ${fallo}: conserva clave e intento, sin reintento automático ni falso recibo`, async () => {
  const raiz = raizPrueba(), solicitudes = [];
  const cerrar = await abrirSiguiente(raiz, { continuarLlamamiento: async (s) => {
    solicitudes.push(s);
    if (solicitudes.length === 1 && fallo === "anterior_ajeno") return { ...continuacionConfirmada, llamamiento_anterior_ref: "llamamiento:ajeno" };
    if (solicitudes.length === 1 && fallo === "red") throw new Error("red");
    if (solicitudes.length < 3) throw Object.assign(new Error(), { envelopeValido: true, resultadoIndeterminado: false,
      codigo: fallo === "conflicto" ? "clave_idempotencia_reutilizada" : "acceso_denegado" });
    return { ...continuacionConfirmada, estado_local: "replay_confirmado" };
  } });
  await raiz.enviar("siguiente", solicitudSiguiente);
  assert.equal(solicitudes.length, 1); assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-recibo="siguiente"|data-ct-llamamiento-clave=["]siguiente["]/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="comunicacion_siguiente"/u);
  for (let intento = 0; intento < 2; intento += 1) await raiz.enviar("siguiente", {
    ...solicitudSiguiente, clave_idempotencia: CLAVE, resolucion_ref: "resolucion:ajena", intencion_ref: "intencion:ajena" });
  assert.equal(solicitudes.length, fallo === "conflicto" ? 1 : 3); assert.ok(solicitudes.every((s) => JSON.stringify(s) === JSON.stringify(solicitudSiguiente)));
  if (fallo !== "conflicto") assert.match(raiz.innerHTML, /data-ct-llamamiento-recibo="siguiente"/u);
  cerrar();
});
test("aviso local al sucesor exige continuación, clave propia y confirmación; conserva solicitudes y recibos previos", async () => {
  const raiz = raizPrueba(), solicitudes = [], confirmaciones = []; let confirmar = false, claves = 0;
  const cerrar = await abrirSiguiente(raiz, {
    continuarLlamamiento: async () => continuacionConfirmada,
    registrarComunicacionLlamamiento: async (s) => {
      solicitudes.push(s); return s.tipo_antecedente ? avisoSiguienteRegistrado : comunicacionRegistrada;
    },
  }, { confirmarOperacion: (d) => {
    if (!d.datos.tipo_antecedente) return true;
    confirmaciones.push(d); return confirmar;
  }, generarClaveIdempotencia: () => { claves += 1; return solicitudAvisoSiguiente.clave_idempotencia; } });
  const pulsarClave = () => raiz.eventos.get("click")({ preventDefault() {}, target: {
    closest: () => ({ dataset: { ctLlamamientoClave: "comunicacion_siguiente" } }),
  } });
  assert.equal(solicitudes.length, 1); assert.equal(Object.keys(solicitudes[0]).length, 6);
  const solicitudOriginal = JSON.stringify(solicitudes[0]);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="comunicacion_siguiente"/u);
  pulsarClave(); await raiz.enviar("comunicacion_siguiente", solicitudAvisoSiguiente);
  assert.equal(claves, 0); assert.equal(solicitudes.length, 1);
  delete raiz.borradores.comunicacion_siguiente;
  await raiz.enviar("siguiente", solicitudSiguiente);
  const formulario = () => raiz.innerHTML.match(/<form data-ct-llamamiento-form="comunicacion_siguiente"[\s\S]*?<\/form>/u)[0];
  assert.match(formulario(), /name="clave_idempotencia" value=""/u);
  assert.match(formulario(), /Recibo de continuación antecedente/u);
  assert.doesNotMatch(formulario(), /name="(?:tipo_antecedente|actor_ref|politica_ref)"|type="file"|type="checkbox"/u);
  for (const campo of Object.keys(solicitudAvisoSiguiente).slice(1, -1))
    assert.match(formulario(), new RegExp(`name="${campo}"[^>]*readonly`, "u"));
  const previos = () => ["seleccion", "comunicacion", "respuesta", "resolucion", "siguiente"].map((op) => [
    raiz.innerHTML.match(new RegExp(`<form data-ct-llamamiento-form="${op}"[\\s\\S]*?<\\/form>`, "u"))[0],
    raiz.innerHTML.match(new RegExp(`<section[^>]*data-ct-llamamiento-recibo="${op}"[\\s\\S]*?<\\/section>`, "u"))[0],
  ]);
  const anteriores = previos();
  assert.equal(claves, 0); assert.equal(solicitudes.length, 1);
  pulsarClave(); assert.equal(claves, 1);
  for (const clave_idempotencia of [CLAVE, declaracion().clave_idempotencia, CLAVE_RESOLUCION, solicitudSiguiente.clave_idempotencia])
    await raiz.enviar("comunicacion_siguiente", { clave_idempotencia });
  assert.equal(confirmaciones.length, 0);
  await raiz.enviar("comunicacion_siguiente", solicitudAvisoSiguiente);
  assert.equal(solicitudes.length, 1); confirmar = true;
  await raiz.enviar("comunicacion_siguiente", { ...solicitudAvisoSiguiente,
    organizacion_ref: "org:ajena", expediente_ref: "exp:ajeno", llamamiento_ref: "llamamiento:ajeno",
    version_esperada: 99, prueba_entrega_ref: "recibo:ajeno", tipo_antecedente: "inventado", actor_ref: "actor:ajeno" });
  assert.deepEqual(solicitudes[1], solicitudAvisoSiguiente); assert.ok(Object.isFrozen(solicitudes[1]));
  assert.equal(JSON.stringify(solicitudes[0]), solicitudOriginal); assert.deepEqual(previos(), anteriores);
  assert.match(confirmaciones.at(-1).advertencia, /No acredita envío, entrega, apertura de plazo/u);
  assert.ok(confirmaciones.at(-1).advertencia.includes(continuacionConfirmada.recibo_ref));
  assert.match(raiz.innerHTML, /Aviso local al sucesor registrado · No enviado/u);
  assert.doesNotMatch(formulario(), /type="submit"/u);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-form="respuesta_siguiente"/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="resolucion_siguiente"/u);
  await raiz.enviar("comunicacion_siguiente", solicitudAvisoSiguiente); pulsarClave();
  assert.equal(solicitudes.length, 2); assert.equal(claves, 1); cerrar();
});
for (const fallo of ["red", "recibo_no_local", "rechazo", "conflicto"]) test(`aviso al sucesor ${fallo}: intento congelado, sin reintento automático ni recibo falso`, async () => {
  const raiz = raizPrueba(), solicitudes = [];
  const cerrar = await abrirSiguiente(raiz, {
    continuarLlamamiento: async () => continuacionConfirmada,
    registrarComunicacionLlamamiento: async (s) => {
      if (!s.tipo_antecedente) return comunicacionRegistrada;
      solicitudes.push(s);
      if (solicitudes.length === 1 && fallo === "red") throw new Error("red");
      if (solicitudes.length === 1 && fallo === "recibo_no_local") {
        const noLocal = { ...avisoSiguienteRegistrado, estado_local: "confirmado", respuesta_hasta: "2026-09-06T08:00:00Z" };
        delete noLocal.registrada_en; delete noLocal.intencion_envio_ref; return noLocal;
      }
      if (solicitudes.length < 3) throw Object.assign(new Error(), { envelopeValido: true, resultadoIndeterminado: false,
        codigo: fallo === "conflicto" ? "clave_idempotencia_reutilizada" : "acceso_denegado" });
      return { ...avisoSiguienteRegistrado, estado_local: "replay_registrada_localmente" };
    },
  });
  await raiz.enviar("siguiente", solicitudSiguiente); assert.equal(solicitudes.length, 0);
  await raiz.enviar("comunicacion_siguiente", solicitudAvisoSiguiente);
  assert.equal(solicitudes.length, 1);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-recibo="comunicacion_siguiente"|data-ct-llamamiento-clave=["]comunicacion_siguiente["]/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="respuesta_siguiente"/u);
  for (let intento = 0; intento < 2; intento += 1) await raiz.enviar("comunicacion_siguiente", {
    ...solicitudAvisoSiguiente, clave_idempotencia: CLAVE, prueba_entrega_ref: "recibo:ajeno", tipo_antecedente: "inventado" });
  assert.equal(solicitudes.length, fallo === "conflicto" ? 1 : 3);
  assert.ok(solicitudes.every((s) => JSON.stringify(s) === JSON.stringify(solicitudAvisoSiguiente)));
  if (fallo !== "conflicto") assert.match(raiz.innerHTML, /Registro local recuperado/u);
  cerrar();
});
test("aviso al sucesor en vuelo evita duplicación y se cancela al desmontar sin repintado tardío", async () => {
  const raiz = raizPrueba(); let resolver, signal, llamadas = 0;
  const cerrar = await abrirSiguiente(raiz, {
    continuarLlamamiento: async () => continuacionConfirmada,
    registrarComunicacionLlamamiento: (s, opciones) => {
      if (!s.tipo_antecedente) return comunicacionRegistrada;
      llamadas += 1; signal = opciones.signal;
      return new Promise((resolve) => { resolver = resolve; });
    },
  });
  await raiz.enviar("siguiente", solicitudSiguiente);
  const pendiente = raiz.enviar("comunicacion_siguiente", solicitudAvisoSiguiente);
  await raiz.enviar("comunicacion_siguiente", solicitudAvisoSiguiente); assert.equal(llamadas, 1);
  cerrar(); assert.equal(signal.aborted, true); resolver(avisoSiguienteRegistrado); await pendiente;
  assert.equal(raiz.innerHTML, "");
});

for (const respuesta of ["aceptacion", "renuncia"]) test(`respuesta del sucesor ${respuesta}: diez campos derivados, confirmación y recibos anteriores intactos sin resolver`, async () => {
  const raiz = raizPrueba(), solicitudes = [], confirmaciones = []; let confirmar = false;
  const cerrar = await abrirRespuestaSiguiente(raiz, (s) => { solicitudes.push(s); return justificanteSiguiente(s); }, {
    confirmarOperacion: (d) => {
      if (d.datos.comunicacion_ref !== avisoSiguienteRegistrado.comunicacion_ref) return true;
      confirmaciones.push(d); return confirmar;
    },
  });
  const valores = { ...declaracionSiguiente(), respuesta };
  const formulario = () => raiz.innerHTML.match(/<form data-ct-llamamiento-form="respuesta_siguiente"[\s\S]*?<\/form>/u)[0];
  const previos = () => ["seleccion", "comunicacion", "respuesta", "resolucion", "siguiente", "comunicacion_siguiente"].map((op) => [
    raiz.innerHTML.match(new RegExp(`<form data-ct-llamamiento-form="${op}"[\\s\\S]*?<\\/form>`, "u"))[0],
    raiz.innerHTML.match(new RegExp(`<section[^>]*data-ct-llamamiento-recibo="${op}"[\\s\\S]*?<\\/section>`, "u"))[0],
  ]);
  const anteriores = previos(), ids = [...raiz.innerHTML.matchAll(/\bid="([^"]+)"/gu)].map((m) => m[1]);
  assert.equal(new Set(ids).size, ids.length);
  assert.match(formulario(), /id="ct-llamamiento-respuesta_siguiente-correo"/u);
  assert.match(formulario(), /aria-describedby="ct-llamamiento-respuesta_siguiente-correo-ayuda"/u);
  assert.match(formulario(), /Fecha de recepción[^<]*UTC/u);
  assert.doesNotMatch(formulario(), /name="(?:tipo_antecedente|actor_ref|politica_ref|revision_plazo_rrhh)"/u);
  await raiz.enviar("respuesta_siguiente", { ...valores, correo_sha256: HUELLA });
  assert.equal(confirmaciones.length, 0); // No admite una huella escrita en el DOM.
  await raiz.archivo(archivoCorreo(respuesta), "respuesta_siguiente");
  assert.match(formulario(), /Huella calculada/u);
  assert.match(formulario(), /name="correo_sha256"[^>]*readonly/u);
  for (const clave_idempotencia of [CLAVE, declaracion().clave_idempotencia, CLAVE_RESOLUCION,
    solicitudSiguiente.clave_idempotencia, solicitudAvisoSiguiente.clave_idempotencia])
    await raiz.enviar("respuesta_siguiente", { ...valores, clave_idempotencia });
  assert.equal(confirmaciones.length, 0);
  await raiz.enviar("respuesta_siguiente", valores); assert.equal(solicitudes.length, 0);
  confirmar = true;
  await raiz.enviar("respuesta_siguiente", { ...valores, organizacion_ref: "org:ajena",
    expediente_ref: "exp:ajeno", llamamiento_ref: "llamamiento:ajeno", comunicacion_ref: "comunicacion:ajena",
    version_comunicacion_esperada: 99, correo_sha256: "f".repeat(64), tipo_antecedente: "inventado" });
  const esperada = {
    clave_idempotencia: valores.clave_idempotencia, organizacion_ref: solicitudAvisoSiguiente.organizacion_ref,
    expediente_ref: solicitudAvisoSiguiente.expediente_ref, llamamiento_ref: solicitudAvisoSiguiente.llamamiento_ref,
    comunicacion_ref: avisoSiguienteRegistrado.comunicacion_ref, version_comunicacion_esperada: 2,
    respuesta, correo_ref: valores.correo_ref,
    correo_sha256: createHash("sha256").update(await archivoCorreo(respuesta).text()).digest("hex"),
    recibida_en: "2026-09-05T09:08:00Z",
  };
  assert.equal(JSON.stringify(solicitudes[0]), JSON.stringify(esperada)); assert.ok(Object.isFrozen(solicitudes[0]));
  assert.match(confirmaciones.at(-1).advertencia, /no cambia la candidatura/iu);
  assert.deepEqual(previos(), anteriores);
  assert.match(raiz.innerHTML, /Declaración de respuesta del sucesor registrada · Sin resolución/u);
  assert.match(raiz.innerHTML, /justificante:sucesor:002/u);
  assert.doesNotMatch(raiz.innerHTML, /Subject:|respuesta-sintetica.eml|data-ct-llamamiento-recibo="resolucion_siguiente"/u);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-form="resolucion_siguiente"/u);
  assert.doesNotMatch(formulario(), /type="submit"/u);
  await raiz.enviar("respuesta_siguiente", valores); assert.equal(solicitudes.length, 1); cerrar();
});
test("respuesta del sucesor no admite envío ni lectura antes de su aviso local validado v2", async () => {
  const raiz = raizPrueba(); let llamadas = 0;
  const cerrar = await abrirSiguiente(raiz, {
    continuarLlamamiento: async () => continuacionConfirmada,
    registrarComunicacionLlamamiento: async (s) => s.tipo_antecedente
      ? { ...avisoSiguienteRegistrado, version_resultante: 3 } : comunicacionRegistrada,
  });
  const archivo = { name: "sintetico.eml", size: 1, arrayBuffer: () => { llamadas += 1; } };
  await raiz.archivo(archivo, "respuesta_siguiente");
  await raiz.enviar("respuesta_siguiente", declaracionSiguiente());
  await raiz.enviar("siguiente", solicitudSiguiente);
  await raiz.enviar("comunicacion_siguiente", solicitudAvisoSiguiente);
  await raiz.archivo(archivo, "respuesta_siguiente");
  await raiz.enviar("respuesta_siguiente", declaracionSiguiente());
  assert.equal(llamadas, 0); assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="respuesta_siguiente"/u);
  cerrar();
});
for (const fallo of ["red", "recibo_cruzado", "rechazo", "conflicto"]) test(`respuesta del sucesor ${fallo}: sin recibo falso, misma clave y recuperación explícita`, async () => {
  const raiz = raizPrueba(), solicitudes = [];
  const cerrar = await abrirRespuestaSiguiente(raiz, (s) => {
    solicitudes.push(s);
    if (solicitudes.length === 1 && fallo === "red") throw new Error("red");
    if (solicitudes.length === 1 && fallo === "recibo_cruzado") return {
      ...justificanteSiguiente(s), comunicacion_ref: comunicacionRegistrada.comunicacion_ref };
    if (solicitudes.length < 3) throw Object.assign(new Error(), { envelopeValido: true, resultadoIndeterminado: false,
      codigo: fallo === "conflicto" ? "clave_idempotencia_reutilizada" : "acceso_denegado" });
    return { ...justificanteSiguiente(s), estado: "replay_registrada_por_rrhh" };
  });
  await raiz.archivo(archivoCorreo(), "respuesta_siguiente");
  await raiz.enviar("respuesta_siguiente", declaracionSiguiente());
  assert.equal(solicitudes.length, 1);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-recibo="respuesta_siguiente"|data-ct-llamamiento-clave=["]respuesta_siguiente["]/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="resolucion_siguiente"/u);
  for (let intento = 0; intento < 2; intento += 1) {
    if (fallo !== "rechazo") await raiz.archivo(archivoCorreo("renuncia"), "respuesta_siguiente");
    await raiz.enviar("respuesta_siguiente", { ...declaracionSiguiente(), clave_idempotencia: CLAVE,
      comunicacion_ref: "comunicacion:ajena", ...(fallo !== "rechazo" ? { respuesta: "renuncia", recibida_en: "2026-09-06T12:00" } : {}) });
  }
  assert.equal(solicitudes.length, fallo === "conflicto" ? 1 : 3);
  assert.ok(solicitudes.every((s) => JSON.stringify(s) === JSON.stringify(solicitudes[0])));
  if (fallo !== "conflicto") assert.match(raiz.innerHTML, /Misma declaración recuperada, sin nuevo registro/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-recibo="resolucion_siguiente"/u); cerrar();
});
for (const fase of ["huella", "peticion"]) test(`respuesta del sucesor cancela ${fase} sin duplicado ni repintado tardío`, async () => {
  const raiz = raizPrueba(); let liberar, signal, llamadas = 0, solicitud;
  const cerrar = await abrirRespuestaSiguiente(raiz, (s, opciones) => {
    llamadas += 1; solicitud = s; signal = opciones.signal;
    return new Promise((resolve) => { liberar = resolve; });
  });
  const bytes = new Uint8Array([65]);
  let pendiente;
  if (fase === "huella") {
    await raiz.archivo({ name: "grande.eml", size: 2 * 1024 * 1024 + 1,
      arrayBuffer: () => assert.fail("no leer más de 2 MiB") }, "respuesta_siguiente");
    pendiente = raiz.archivo({ name: "pendiente.eml", size: 1,
      arrayBuffer: () => new Promise((resolve) => { liberar = resolve; }) }, "respuesta_siguiente");
  } else {
    await raiz.archivo(archivoCorreo(), "respuesta_siguiente");
    pendiente = raiz.enviar("respuesta_siguiente", declaracionSiguiente());
  }
  await raiz.enviar("respuesta_siguiente", declaracionSiguiente());
  assert.equal(llamadas, fase === "huella" ? 0 : 1); cerrar();
  if (fase === "peticion") assert.equal(signal.aborted, true);
  liberar(fase === "huella" ? bytes.buffer : justificanteSiguiente(solicitud)); await pendiente;
  assert.equal(raiz.innerHTML, ""); assert.equal(raiz.eventos.size, 0);
  if (fase === "huella") assert.equal(bytes[0], 0);
});

for (const respuesta of ["aceptacion", "renuncia"]) test(`octava operación ${respuesta}: justificante propio, once campos, siete recibos intactos y ninguna continuación automática`, async () => {
  const raiz = raizPrueba(), solicitudes = [], confirmaciones = []; let confirmar = false;
  const cerrar = await abrirRespuestaSiguiente(raiz, justificanteSiguiente, {
    confirmarOperacion: (d) => {
      if (d.datos.prueba_respuesta_ref !== justificanteSiguiente({}).justificante_ref) return true;
      confirmaciones.push(d); return confirmar;
    },
  }, { resolverLlamamiento: (s) => {
    if (s.comunicacion_ref !== avisoSiguienteRegistrado.comunicacion_ref) return reciboResolucion("renuncia");
    solicitudes.push(s); return reciboResolucionSucesor(respuesta);
  } });
  const valores = { clave_idempotencia: CLAVE_RESOLUCION_SIGUIENTE, ...revisionManual };
  await raiz.enviar("resolucion_siguiente", valores);
  assert.equal(solicitudes.length, 0); assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="resolucion_siguiente"/u);
  delete raiz.borradores.resolucion_siguiente;
  await raiz.archivo(archivoCorreo(respuesta), "respuesta_siguiente");
  await raiz.enviar("respuesta_siguiente", { ...declaracionSiguiente(), respuesta });
  const formulario = () => raiz.innerHTML.match(/<form data-ct-llamamiento-form="resolucion_siguiente"[\s\S]*?<\/form>/u)[0];
  const previos = () => ["seleccion", "comunicacion", "respuesta", "resolucion", "siguiente", "comunicacion_siguiente", "respuesta_siguiente"].map((op) => [
    raiz.innerHTML.match(new RegExp(`<form data-ct-llamamiento-form="${op}"[\\s\\S]*?<\\/form>`, "u"))[0],
    raiz.innerHTML.match(new RegExp(`<section[^>]*data-ct-llamamiento-recibo="${op}"[\\s\\S]*?<\\/section>`, "u"))[0],
  ]);
  const anteriores = previos();
  assert.equal(solicitudes.length, 0); assert.match(formulario(), /name="clave_idempotencia" value=""/u);
  assert.doesNotMatch(formulario(), /type="file"|name="(?:actor_ref|estado_plazo|tipo_antecedente)"/u);
  assert.match(formulario(), /name="respuesta" required disabled/u);
  for (const campo of ["organizacion_ref", "expediente_ref", "llamamiento_ref", "comunicacion_ref", "version_esperada", "prueba_respuesta_ref", "criterio_validacion_ref"])
    assert.match(formulario(), new RegExp(`name="${campo}"[^>]*readonly`, "u"));
  for (const nombre of Object.keys(revisionManual)) {
    const control = formulario().match(new RegExp(`<input[^>]*name="${nombre}"[^>]*>`, "u"))[0];
    assert.match(control, /type="checkbox"/u); assert.doesNotMatch(control, /\schecked|\sdisabled/u);
    assert.match(control, /aria-describedby="ct-llamamiento-resolucion_siguiente-validacion-ayuda"/u);
  }
  const ids = [...raiz.innerHTML.matchAll(/\bid="([^"]+)"/gu)].map((m) => m[1]);
  assert.equal(new Set(ids).size, ids.length);
  for (const revision of [{}, { revision_respuesta_rrhh: true }, { revision_plazo_rrhh: true }])
    await raiz.enviar("resolucion_siguiente", { clave_idempotencia: CLAVE_RESOLUCION_SIGUIENTE, ...revision });
  for (const clave_idempotencia of [CLAVE, declaracion().clave_idempotencia, CLAVE_RESOLUCION,
    solicitudSiguiente.clave_idempotencia, solicitudAvisoSiguiente.clave_idempotencia, declaracionSiguiente().clave_idempotencia])
    await raiz.enviar("resolucion_siguiente", { ...valores, clave_idempotencia });
  assert.equal(confirmaciones.length, 0);
  await raiz.enviar("resolucion_siguiente", valores); assert.equal(solicitudes.length, 0); confirmar = true;
  await raiz.enviar("resolucion_siguiente", { ...valores, respuesta: respuesta === "aceptacion" ? "renuncia" : "aceptacion",
    organizacion_ref: "org:ajena", expediente_ref: "exp:ajeno", llamamiento_ref: recibo.llamamiento_ref,
    comunicacion_ref: comunicacionRegistrada.comunicacion_ref, version_esperada: 99,
    prueba_respuesta_ref: justificante({}).justificante_ref, criterio_validacion_ref: "politica:inventada" });
  const esperada = { clave_idempotencia: CLAVE_RESOLUCION_SIGUIENTE,
    organizacion_ref: solicitudAvisoSiguiente.organizacion_ref, expediente_ref: EXPEDIENTE,
    llamamiento_ref: continuacionConfirmada.llamamiento_ref, comunicacion_ref: avisoSiguienteRegistrado.comunicacion_ref,
    version_esperada: 2, respuesta, prueba_respuesta_ref: justificanteSiguiente({}).justificante_ref,
    ...revisionManual, criterio_validacion_ref: "politica:ct:revision-manual-sintetica:20260906" };
  assert.equal(JSON.stringify(solicitudes[0]), JSON.stringify(esperada)); assert.ok(Object.isFrozen(solicitudes[0]));
  assert.match(confirmaciones.at(-1).advertencia, /justificante:sucesor:002/u);
  assert.match(confirmaciones.at(-1).advertencia, /El vencimiento no se evalúa porque faltan inicio y política gobernados/u);
  assert.deepEqual(previos(), anteriores);
  const resultado = raiz.innerHTML.match(/<section[^>]*data-ct-llamamiento-recibo="resolucion_siguiente"[\s\S]*?<\/section>/u)[0];
  assert.match(resultado, new RegExp(`${respuesta === "aceptacion" ? "Aceptación" : "Renuncia"} del sucesor registrada · ejercicio sintético`, "u"));
  assert.match(resultado, /recibo:resolucion:sucesor:002|2026-09-05T09:10:00.123456Z/u);
  const resumen = raiz.innerHTML.match(/<section class="ct-llamamiento-resultado"[\s\S]*?<\/section>/u)[0];
  assert.match(resumen, /Estado de la respuesta del sucesor/u);
  assert.match(resumen, new RegExp(`Resolución de ${respuesta === "aceptacion" ? "aceptación" : "renuncia"}; resultado manual sintético registrado`, "u"));
  assert.match(resumen, /Declarada el/u);
  assert.match(resumen, /Resuelta el/u);
  assert.doesNotMatch(resumen, /Registrar el aviso local del sucesor/u);
  if (respuesta === "renuncia") assert.match(resumen, /todavía no hay otra apertura disponible/u);
  if (respuesta === "renuncia") {
    assert.match(resultado, /intencion:sucesor:003/u); assert.match(resultado, /No se ha seleccionado ni avisado a otra persona/u);
    assert.doesNotMatch(resultado, /Recibo histórico de renuncia/u);
  } else assert.doesNotMatch(resultado, /intencion:sucesor:003/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="(?:propuesta_siguiente|siguiente_siguiente)"/u);
  if (respuesta === "renuncia") assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="propuesta"/u);
  else assert.match(raiz.innerHTML, /Publicaciones de formalización no disponibles/u);
  assert.equal([...raiz.innerHTML.matchAll(/data-ct-llamamiento-form="siguiente"/gu)].length, 1);
  assert.doesNotMatch(formulario(), /type="submit"/u);
  await raiz.enviar("resolucion_siguiente", valores); assert.equal(solicitudes.length, 1); cerrar();
});

for (const operacion of ["resolucion", "resolucion_siguiente"])
for (const opcion of ["aceptacion", "renuncia"]) test(`HTTP 409 pendiente ${operacion}/${opcion} libera casillas pero conserva clave y exige otra confirmación`, async () => {
  const raiz = raizPrueba(), solicitudes = []; let claves = 0, confirmaciones = 0;
  const sucesor = operacion === "resolucion_siguiente";
  const operacionId = sucesor ? CLAVE_RESOLUCION_SIGUIENTE : CLAVE_RESOLUCION;
  const formulario = () => raiz.innerHTML.match(new RegExp(`<form data-ct-llamamiento-form="${operacion}"[\\s\\S]*?<\\/form>`, "u"))[0];
  const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async (ruta, opciones) => {
    assert.ok(ruta.endsWith("/resoluciones"));
    solicitudes.push(opciones.body);
    return new Response(JSON.stringify({ error: {
      codigo: "validacion_respuesta_pendiente",
      clave_i18n: "api.contratacion_temporal.comunicacion_llamamiento.error.validacion_respuesta_pendiente",
      correlacion_ref: "corr_0123456789abcdef0123456789abcdef",
    } }), { status: 409, headers: { "content-type": "application/json; charset=utf-8" } });
  } });
  const cerrar = await (sucesor ? abrirResolucionSucesor : abrirResolucion)(raiz, { resolverLlamamiento: cliente.resolverLlamamiento }, {
    confirmarOperacion: () => { confirmaciones += 1; return true; },
    generarClaveIdempotencia: () => { claves += 1; return CLAVE; },
  }, opcion);
  assert.equal(solicitudes.length, 0);
  for (let intento = 0; intento < 2; intento += 1) {
    await raiz.enviar(operacion, { clave_idempotencia: intento === 0 ? operacionId : CLAVE, ...revisionManual });
    assert.equal(solicitudes.length, intento + 1); assert.match(raiz.innerHTML, /Pendiente de revisión manual sintética por RRHH\. El vencimiento no es evaluable y la resolución no se ha confirmado\./u); assert.match(formulario(), /Revisar y solicitar resolución/u); assert.match(formulario(), new RegExp(`name="clave_idempotencia" value="${operacionId}"[^>]*readonly`, "u"));
    const control = formulario().match(/<input[^>]*name="revision_plazo_rrhh"[^>]*>/u)[0];
    assert.match(control, /\schecked/u); assert.doesNotMatch(control, /\sdisabled/u); assert.doesNotMatch(raiz.innerHTML, new RegExp(`data-ct-llamamiento-recibo="${operacion}"|No se ha podido confirmar el resultado`, "u"));
    assert.doesNotMatch(formulario(), /data-ct-llamamiento-clave/u);
    raiz.eventos.get("click")({ target: { closest: () => ({ dataset: { ctLlamamientoClave: operacion } }) },
      preventDefault() {} });
    assert.equal(claves, 0);
    await raiz.enviar(operacion, { clave_idempotencia: CLAVE,
      ...revisionManual, revision_plazo_rrhh: false });
    assert.equal(solicitudes.length, intento + 1); // Corregir no autoriza a enviar sin ambas casillas.
    assert.doesNotMatch(formulario().match(/<input[^>]*name="revision_plazo_rrhh"[^>]*>/u)[0], /\schecked/u);
  }
  assert.equal(solicitudes[0], solicitudes[1]); assert.equal(JSON.parse(solicitudes[1]).clave_idempotencia, operacionId);
  assert.equal(confirmaciones, sucesor ? 8 : 5); // Antecedentes y dos solicitudes; primera resolución propia del helper.
  cerrar();
});
for (const operacion of ["resolucion", "resolucion_siguiente"])
for (const opcion of ["aceptacion", "renuncia"]) test(`${operacion} ${opcion} ambigua o inválida conserva intento incluso si el replay pierde permiso`, async () => {
  const sucesor = operacion === "resolucion_siguiente";
  const operacionId = sucesor ? CLAVE_RESOLUCION_SIGUIENTE : CLAVE_RESOLUCION;
  const resultado = sucesor ? reciboResolucionSucesor : reciboResolucion;
  for (const invalido of [false, true, "replay_pendiente", ...(opcion === "renuncia" ? ["sin_intencion"] : [])]) {
    const raiz = raizPrueba(), solicitudes = [];
    const cerrar = await (sucesor ? abrirResolucionSucesor : abrirResolucion)(raiz, { resolverLlamamiento: async (s) => {
      solicitudes.push(s);
      if (solicitudes.length === 1) {
        if (invalido === true) return { ...resultado(opcion), estado_plazo: "vencido" };
        if (invalido === "sin_intencion") return { ...resolucionConfirmada, respuesta: opcion };
        throw new Error("transporte interrumpido");
      }
      if (solicitudes.length === 2) throw Object.assign(new Error(), {
        codigo: invalido === "replay_pendiente" ? "validacion_respuesta_pendiente" : "acceso_denegado",
        estado: invalido === "replay_pendiente" ? 409 : 403, envelopeValido: true, resultadoIndeterminado: false,
      });
      return resultado(opcion, "replay_confirmado");
    } }, {}, opcion);
    for (let intento = 0; intento < 3; intento += 1) {
      await raiz.enviar(operacion, { clave_idempotencia: intento === 0 ? operacionId : CLAVE,
        revision_respuesta_rrhh: intento === 0, revision_plazo_rrhh: intento === 0 });
      assert.deepEqual(solicitudes[intento], solicitudes[0]);
      if (intento < 2) {
        assert.doesNotMatch(raiz.innerHTML, new RegExp(`data-ct-llamamiento-recibo="${operacion}"`, "u")); assert.match(raiz.innerHTML.match(new RegExp(`<input[^>]*id="ct-llamamiento-${operacion}-revision_plazo_rrhh"[^>]*>`, "u"))[0], /checked disabled/u); assert.match(raiz.innerHTML, /No se ha podido confirmar el resultado/u);
      }
      assert.doesNotMatch(raiz.innerHTML, new RegExp(`data-ct-llamamiento-clave=["]${operacion}["]`, "u"));
    }
    assert.match(raiz.innerHTML, /Recuperado sin repetir el efecto/u);
    cerrar();
  }
});
for (const opcion of ["aceptacion", "renuncia", "siguiente", "resolucion_siguiente"]) test(`${opcion} no duplica; desmontar aborta y descarta el recibo tardío`, async () => {
  const raiz = raizPrueba(); let resolver, signal, llamadas = 0;
  const sucesor = opcion === "resolucion_siguiente";
  const esSiguiente = opcion === "siguiente", operacion = sucesor ? opcion : esSiguiente ? "siguiente" : "resolucion";
  const cliente = { [esSiguiente ? "continuarLlamamiento" : "resolverLlamamiento"]: (_, opciones) => {
    llamadas += 1; signal = opciones.signal;
    return new Promise((resolve) => { resolver = resolve; });
  } };
  const cerrar = sucesor ? await abrirResolucionSucesor(raiz, cliente)
    : esSiguiente ? await abrirSiguiente(raiz, cliente) : await abrirResolucion(raiz, cliente, {}, opcion);
  raiz.preparar(operacion, esSiguiente ? solicitudSiguiente : { clave_idempotencia: sucesor ? CLAVE_RESOLUCION_SIGUIENTE : CLAVE_RESOLUCION, ...revisionManual });
  const pendiente = raiz.enviar(operacion);
  await raiz.enviar(operacion);
  assert.equal(llamadas, 1);
  cerrar();
  assert.equal(signal.aborted, true);
  resolver(sucesor ? reciboResolucionSucesor("aceptacion") : esSiguiente ? continuacionConfirmada : reciboResolucion(opcion));
  await pendiente;
  assert.equal(raiz.innerHTML, ""); assert.equal(raiz.eventos.size, 0);
});
