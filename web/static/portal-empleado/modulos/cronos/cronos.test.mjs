import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { crearEjecutorCronosPresentacion } from "./adaptador-presentacion.js";
import {
  CAPACIDAD_CONSULTAR_FICHAJES,
  CAPACIDAD_CONSULTAR_HORARIO,
  CAPACIDAD_CONSULTAR_PERMISOS,
  CAPACIDAD_REGISTRAR_FICHAJE,
  CAPACIDAD_SOLICITAR_PERMISO,
  validarDatosCronos,
} from "./contrato.js";
import { crearDatosCronosPresentacion } from "./datos-presentacion.js";
import { crearSolicitudPDFReciboCronos, solicitarPDFReciboCronos } from "./documentos.js";
import { MENSAJES_CRONOS_ES } from "./i18n.js";
import { crearPresentadorCronos } from "./presentador.js";
import { renderizarAreaCronos } from "./vista.js";
import { crearContextoActorPresentacionDesdeSesion } from "../../identidad/presentacion.js";
import {
  ESQUEMA_CONTEXTO_ACTOR_FRONTEND,
  validarYCongelarContextoActor,
} from "../../identidad/contexto-actor.js";

const directorio = new URL("./", import.meta.url);
const [css, presentadorFuente, vistaFuente, datosFuente, adaptadorFuente] = await Promise.all([
  readFile(new URL("cronos.css", directorio), "utf8"),
  readFile(new URL("presentador.js", directorio), "utf8"),
  readFile(new URL("vista.js", directorio), "utf8"),
  readFile(new URL("datos-presentacion.js", directorio), "utf8"),
  readFile(new URL("adaptador-presentacion.js", directorio), "utf8"),
]);

const TODAS_LAS_CAPACIDADES = [
  CAPACIDAD_CONSULTAR_FICHAJES,
  CAPACIDAD_REGISTRAR_FICHAJE,
  CAPACIDAD_CONSULTAR_HORARIO,
  CAPACIDAD_CONSULTAR_PERMISOS,
  CAPACIDAD_SOLICITAR_PERMISO,
];

function contextoPresentacion() {
  return crearContextoActorPresentacionDesdeSesion({
    actor_ref: "DEMO-PERFIL-FUNCIONARIO-01",
    iniciales: "FU",
    nombre: "Funcionario DEMO 01",
    perfil: "Funcionario · autoservicio interno DEMO",
  });
}

function contextoReal() {
  return validarYCongelarContextoActor({
    esquema: ESQUEMA_CONTEXTO_ACTOR_FRONTEND,
    revision: 1,
    demostracion: false,
    persona_ref: "per_persona_interna_real_000001",
    cuenta_ref: "cta_cuenta_interna_real_000001",
    perfil_ref: "prf_perfil_interno_real_000001",
    actor: { actor_ref: "act_actor_interno_real_000001", nombre_visible: "Persona interna", iniciales: "PI" },
    rol: { clave: "personal_interno", etiqueta: "Personal interno" },
    ambito: {
      clase: "personal_interno",
      organizacion_ref: "org_diputacion_granada_real_000001",
      unidad_ref: "uni_recursos_humanos_real_000001",
      modulos: ["bolsa", "cronos", "dietas"],
    },
    autenticacion: { sesion_ref: "ses_sesion_interna_real_000001", metodo: "kerberos_ad", garantia: "alto" },
    resuelto_en: "2026-07-19T09:00:00.000Z",
  });
}

function datosRealesPara(contexto) {
  const datosDemo = crearDatosCronosPresentacion(contextoPresentacion());
  const actorRef = contexto.actor.actor_ref;
  return {
    ...datosDemo,
    demostracion: false,
    actor_ref: actorRef,
    fichajes: datosDemo.fichajes.map((item) => ({ ...item, actor_ref: actorRef })),
    solicitudes: datosDemo.solicitudes.map((item) => ({ ...item, actor_ref: actorRef })),
    historial: datosDemo.historial.map((item) => ({ ...item, actor_ref: actorRef })),
    incidencias: datosDemo.incidencias.map((item) => ({ ...item, actor_ref: actorRef })),
  };
}

test("Cronos reutiliza el actor del portal y no crea una identidad propia", () => {
  const contexto = contextoPresentacion();
  const datos = crearDatosCronosPresentacion(contexto);
  assert.equal(datos.actor_ref, contexto.actor.actor_ref);
  assert.ok(datos.fichajes.every((item) => item.actor_ref === contexto.actor.actor_ref));
  assert.ok(datos.solicitudes.every((item) => item.actor_ref === contexto.actor.actor_ref));
  assert.doesNotMatch(JSON.stringify(datos), /\b(?:dni|nif|correo|telefono|gps|latitud|longitud)\b/i);
  assert.throws(() => validarDatosCronos({ ...datos, actor_ref: "DEMO-ACTOR-AJENO-02" }, contexto), /no pertenecen/);
  assert.throws(() => validarDatosCronos({
    ...datos,
    fichajes: [{ ...datos.fichajes[0], actor_ref: "DEMO-ACTOR-AJENO-02" }],
  }, contexto), /registros ajenos/);
});

test("la primera pantalla es un espacio de trabajo denso, semántico y trazable", () => {
  const contexto = contextoPresentacion();
  const presentador = crearPresentadorCronos({
    contextoActor: contexto,
    capacidades: TODAS_LAS_CAPACIDADES,
    datos: crearDatosCronosPresentacion(contexto),
  });
  assert.strictEqual(presentador.obtenerEstado().identidad, contexto);
  const html = presentador.renderizar();
  for (const id of ["cronos-resumen", "cronos-fichajes", "cronos-permisos", "cronos-historial"]) {
    assert.match(html, new RegExp(`id="${id}"`));
  }
  assert.match(html, /<nav[^>]+aria-label="Contenido de Cronos"/);
  assert.match(html, /<caption>Fichajes propios<\/caption>/);
  assert.match(html, /<caption>Saldos personales<\/caption>/);
  assert.match(html, /<caption>Historial personal<\/caption>/);
  assert.match(html, /<label><span>Tipo de permiso<\/span>/);
  assert.match(html, /aria-live="polite"/);
  assert.match(html, /Ámbito personal propio/);
  assert.match(html, /misma identidad interna que Bolsa/);
  assert.match(html, /Entorno DEMO · datos sintéticos/);
  assert.doesNotMatch(html, /<div[^>]+onclick=|javascript:|document\.cookie|localStorage|sessionStorage/i);
});

test("mínimo privilegio oculta valores, registros y acciones sin capacidad", () => {
  const contexto = contextoPresentacion();
  const presentador = crearPresentadorCronos({
    contextoActor: contexto,
    capacidades: [],
    datos: crearDatosCronosPresentacion(contexto),
  });
  const html = presentador.renderizar();
  assert.doesNotMatch(html, /07:36|\+02:18|DEMO-REC-FIC-1900|DEMO-REC-VAC-0031/);
  assert.match(html, /La sesión no tiene capacidad para consultar fichajes/);
  assert.match(html, /La sesión no tiene capacidad para consultar permisos/);
  assert.match(html, /La sesión no tiene capacidad para consultar el historial/);
  assert.match(html, /data-cronos-accion="registrar-fichaje"[^>]+disabled/);
  assert.match(html, /No disponible para esta sesión/);
  assert.rejects(
    presentador.ejecutar({ tipo: "registrar_fichaje", movimiento: "entrada" }),
    /mínimo privilegio/,
  );
});

test("la observación exige consultar y gestionar, y deniega antes de llamar al ejecutor", async () => {
  const contexto = contextoPresentacion();
  const datos = crearDatosCronosPresentacion(contexto);
  let llamadas = 0;
  const soloConsulta = crearPresentadorCronos({
    contextoActor: contexto,
    capacidades: [CAPACIDAD_CONSULTAR_FICHAJES],
    datos,
    ejecutor: async () => { llamadas += 1; },
  });
  const html = soloConsulta.renderizar();
  assert.match(html, /Preparar observación de muestra en memoria<\/button>/);
  assert.match(html, /disabled aria-disabled="true"/);
  await assert.rejects(
    soloConsulta.ejecutar({ tipo: "preparar_observacion", incidencia_id: "DEMO-INC-20260719-01", observacion: "Texto de observación DEMO" }),
    /mínimo privilegio/,
  );
  assert.equal(llamadas, 0);
  const estadoRestringido = soloConsulta.obtenerEstado().datos;
  assert.equal(estadoRestringido.saldos.length, 0);
  assert.equal(estadoRestringido.solicitudes.length, 0);
  assert.equal(estadoRestringido.historial.every((item) => item.ambito_clave === "fichaje"), true);
  const sinConsulta = crearPresentadorCronos({
    contextoActor: contexto, capacidades: [CAPACIDAD_REGISTRAR_FICHAJE], datos,
  });
  assert.doesNotMatch(sinConsulta.renderizar(), /data-cronos-accion="preparar-observacion"/);
});

test("preparar observación solo añade historial y recibo DEMO sin efectos reales", async () => {
  const contexto = contextoPresentacion();
  const inicial = crearDatosCronosPresentacion(contexto);
  const presentador = crearPresentadorCronos({
    contextoActor: contexto,
    capacidades: [CAPACIDAD_CONSULTAR_FICHAJES, CAPACIDAD_REGISTRAR_FICHAJE],
    datos: inicial,
    ejecutor: crearEjecutorCronosPresentacion({ reloj: () => new Date("2026-07-19T10:30:00Z") }),
  });
  const recibo = await presentador.ejecutar({ tipo: "preparar_observacion", incidencia_id: "DEMO-INC-20260719-01", observacion: "Texto de observación DEMO" });
  const final = presentador.obtenerEstado().datos;
  assert.equal(recibo.estado_clave, "simulado");
  assert.match(recibo.estado, /no enviado.*no registrado.*sin efectos.*memoria temporal/);
  assert.deepEqual(final.incidencias, inicial.incidencias);
  assert.deepEqual(final.fichajes, inicial.fichajes);
  assert.equal(final.saldos.length, 0);
  assert.equal(final.solicitudes.length, 0);
  assert.equal(final.historial.length, inicial.historial.filter((item) => item.ambito_clave === "fichaje" && !item.requiere_horario).length + 1);
  assert.equal(presentador.obtenerEstado().recibos.length, 1);
});

test("el evento no muestra detalles internos y usa un mensaje i18n cerrado", async () => {
  const contexto = contextoPresentacion();
  const presentador = crearPresentadorCronos({
    contextoActor: contexto,
    capacidades: TODAS_LAS_CAPACIDADES,
    datos: crearDatosCronosPresentacion(contexto),
    ejecutor: async () => { throw new Error("detalle interno que no debe llegar al DOM"); },
  });
  const manejadores = {};
  const raiz = {
    addEventListener: (tipo, manejador) => { manejadores[tipo] = manejador; },
    removeEventListener: () => {},
    contains: () => true,
    innerHTML: "",
  };
  const quitar = presentador.instalarEventos({ raiz });
  manejadores.click({
    preventDefault: () => {},
    target: { closest: (selector) => selector.includes("registrar-fichaje") ? { dataset: { cronosTipo: "entrada" } } : null },
  });
  await new Promise((resolver) => setTimeout(resolver, 0));
  assert.match(raiz.innerHTML, /No se pudo completar la acción de presentación/);
  assert.doesNotMatch(raiz.innerHTML, /detalle interno/);
  quitar();
});

test("las operaciones DEMO son volátiles, trazadas y no contaminan el fixture", async () => {
  const contexto = contextoPresentacion();
  const inicial = crearDatosCronosPresentacion(contexto);
  const ejecutor = crearEjecutorCronosPresentacion({ reloj: () => new Date("2026-07-19T10:30:00Z") });
  const presentador = crearPresentadorCronos({ contextoActor: contexto, capacidades: TODAS_LAS_CAPACIDADES, datos: inicial, ejecutor });

  const reciboFichaje = await presentador.ejecutar({ tipo: "registrar_fichaje", movimiento: "entrada" });
  assert.equal(reciboFichaje.referencia, "DEMO-CRONOS-REC-0001");
  assert.match(reciboFichaje.estado, /no enviado/);
  assert.equal(presentador.obtenerEstado().datos.fichajes.length, inicial.fichajes.length + 1);

  const reciboPermiso = await presentador.ejecutar({
    tipo: "solicitar_permiso",
    permiso_id: "asuntos_propios",
    desde: "2026-08-20",
    hasta: "2026-08-20",
    cantidad: 1,
    motivo: "Escenario de presentación",
    documento_ref: "",
  });
  assert.equal(reciboPermiso.referencia, "DEMO-CRONOS-REC-0002");
  assert.equal(presentador.obtenerEstado().recibos.length, 2);
  assert.match(presentador.renderizar(), /Recibos volátiles DEMO/);

  const nuevaSesion = crearPresentadorCronos({ contextoActor: contexto, capacidades: TODAS_LAS_CAPACIDADES, datos: crearDatosCronosPresentacion(contexto) });
  assert.equal(nuevaSesion.obtenerEstado().datos.fichajes.length, inicial.fichajes.length);
  assert.equal(nuevaSesion.obtenerEstado().recibos.length, 0);
});

test("el presentador definitivo acepta datos reales y un puerto sustituible", async () => {
  const contexto = contextoReal();
  const reales = datosRealesPara(contexto);
  const presentador = crearPresentadorCronos({
    contextoActor: contexto,
    capacidades: TODAS_LAS_CAPACIDADES,
    datos: reales,
    ejecutor: async (_comando, contextoEjecucion) => ({
      datos: contextoEjecucion.datos,
      recibo: {
        esquema: "vec.cronos.recibo.v1",
        referencia: "CRONOS-REC-OPACA-0001",
        instante: "2026-07-19T08:45:00Z",
        operacion: "Fichaje registrado",
        estado: "Registrado",
        estado_clave: "registrado",
        ambito_clave: "fichaje",
        actor_ref: contextoEjecucion.identidad.actor.actor_ref,
      },
    }),
  });
  assert.doesNotMatch(presentador.renderizar(), /Entorno DEMO/);
  assert.equal((await presentador.ejecutar({ tipo: "registrar_fichaje", movimiento: "salida" })).referencia, "CRONOS-REC-OPACA-0001");
  assert.match(presentadorFuente, /ejecutor\(comando/);
  assert.doesNotMatch(`${presentadorFuente}\n${vistaFuente}`, /fetch\(|XMLHttpRequest|document\.cookie|localStorage|sessionStorage/);
  assert.doesNotMatch(presentadorFuente, /error\.message/);
});

test("fixture y adaptador de presentación quedan físicamente separados", async () => {
  assert.match(datosFuente, /Fixture aislado de presentación/);
  assert.match(adaptadorFuente, /EXCLUSIVO DE PRESENTACIÓN/);
  const contexto = contextoPresentacion();
  const datosReales = { ...crearDatosCronosPresentacion(contexto), demostracion: false };
  const ejecutor = crearEjecutorCronosPresentacion({ reloj: () => new Date("2026-07-19T11:00:00Z") });
  await assert.rejects(
    ejecutor({ tipo: "registrar_fichaje", movimiento: "entrada" }, { identidad: contexto, datos: datosReales }),
    /solo admite el actor DEMO/,
  );
});

test("los recibos PDF pasan por el puerto documental institucional", async () => {
  const contexto = contextoPresentacion();
  const solicitud = crearSolicitudPDFReciboCronos({ contextoActor: contexto, recibo_ref: "DEMO-CRONOS-REC-0001" });
  assert.equal(solicitud.formato, "pdf");
  assert.equal(solicitud.plantilla, "recibo_cronos_institucional");
  assert.equal(solicitud.marca_institucional, true);
  assert.equal(solicitud.verificacion_qr, true);
  assert.equal(solicitud.incluir_datos_sensibles_en_verificacion, false);

  let recibida;
  const resultado = await solicitarPDFReciboCronos({
    contextoActor: contexto,
    recibo_ref: "DEMO-CRONOS-REC-0001",
    puertoDocumental: async (peticion) => {
      recibida = peticion;
      return {
        esquema: "vec.documentos.resultado-generacion.v1",
        medio: "application/pdf",
        nombre: "recibo-cronos-demo.pdf",
        documento_ref: "doc_opaco_cronos_0001",
        verificacion_ref: "ver_opaca_cronos_0001",
      };
    },
  });
  assert.equal(recibida.actor_ref, contexto.actor.actor_ref);
  assert.equal(resultado.medio, "application/pdf");
  await assert.rejects(
    solicitarPDFReciboCronos({ contextoActor: contexto, recibo_ref: "DEMO-CRONOS-REC-0001" }),
    /puerto documental no conectado/,
  );
});

test("la descarga visible entrega un descriptor PDF común sin identidad en el QR", async () => {
  const contexto = contextoPresentacion();
  let recibido;
  const presentador = crearPresentadorCronos({
    contextoActor: contexto,
    capacidades: TODAS_LAS_CAPACIDADES,
    datos: crearDatosCronosPresentacion(contexto),
    origenComprobacion: "https://empleados.demo.invalid",
    descargarRecibo: async (descriptor) => { recibido = descriptor; },
  });
  const html = presentador.renderizar();
  assert.ok((html.match(/data-cronos-accion="descargar-recibo"/g) || []).length >= 9);
  assert.doesNotMatch(html, /data-cronos-accion="descargar-recibo"[^>]+disabled/);
  await presentador.descargarReciboPDF("DEMO-REC-FIC-1900");
  assert.equal(recibido.formato, "pdf");
  assert.equal(recibido.titulo, "Recibo de actuación en Cronos");
  assert.equal(recibido.subtitulo, "Portal del Empleado · Diputación de Granada");
  assert.equal(recibido.comprobacion.contiene_datos_personales, false);
  assert.match(recibido.comprobacion.qr_contenido, /\/verificar\/\?ref=DEMO-REC-FIC-1900/);
  assert.doesNotMatch(recibido.comprobacion.qr_contenido, new RegExp(contexto.actor.actor_ref));
  assert.doesNotMatch(recibido.comprobacion.qr_contenido, /Administrador|persona|cuenta/i);
  assert.ok(recibido.filas.some((item) => item.etiqueta === "Actuación" && /Entrada/.test(item.valor)));
  assert.match(recibido.nombre_archivo, /\.pdf$/);
});

test("la navegación interna no altera el hash gestionado por el portal", () => {
  const contexto = contextoPresentacion();
  const html = crearPresentadorCronos({
    contextoActor: contexto,
    capacidades: TODAS_LAS_CAPACIDADES,
    datos: crearDatosCronosPresentacion(contexto),
  }).renderizar();
  assert.match(html, /data-cronos-destino="cronos-fichajes"/);
  assert.doesNotMatch(html, /href="#cronos-/);
  assert.match(presentadorFuente, /scrollIntoView/);
});

test("los instantes se conservan en UTC y se presentan en Europe/Madrid", async () => {
  const contexto = contextoPresentacion();
  const presentador = crearPresentadorCronos({
    contextoActor: contexto,
    capacidades: TODAS_LAS_CAPACIDADES,
    datos: crearDatosCronosPresentacion(contexto),
    ejecutor: crearEjecutorCronosPresentacion({ reloj: () => new Date("2026-07-19T10:30:00Z") }),
  });
  await presentador.ejecutar({ tipo: "registrar_fichaje", movimiento: "inicio_pausa" });
  assert.equal(presentador.obtenerEstado().datos.fichajes[0].instante, "2026-07-19T10:30:00Z");
  assert.match(presentador.renderizar(), /12:30[^<]*(?:CEST|GMT\+2)/);
  assert.throws(() => validarDatosCronos({
    ...crearDatosCronosPresentacion(contexto),
    fichajes: [{
      ...crearDatosCronosPresentacion(contexto).fichajes[0],
      fecha: "19/07/2026",
      hora: "15:08",
    }],
  }, contexto), /contrato cerrado|instante UTC canónico/);
});

test("permisos horarios usan minutos canónicos y fechas de calendario ISO", async () => {
  const contexto = contextoPresentacion();
  const datos = crearDatosCronosPresentacion(contexto);
  const conciliacion = datos.saldos.find((item) => item.id === "bolsa_conciliacion");
  assert.deepEqual(
    { unidad: conciliacion.unidad_clave, concedido: conciliacion.concedido, restante: conciliacion.restante },
    { unidad: "minuto", concedido: 1800, restante: 1440 },
  );
  assert.ok(datos.solicitudes.every((item) => /^\d{4}-\d{2}-\d{2}$/.test(item.desde) && /^\d{4}-\d{2}-\d{2}$/.test(item.hasta)));
  const presentador = crearPresentadorCronos({
    contextoActor: contexto,
    capacidades: TODAS_LAS_CAPACIDADES,
    datos,
    ejecutor: crearEjecutorCronosPresentacion({ reloj: () => new Date("2026-07-19T12:00:00Z") }),
  });
  assert.match(presentador.renderizar(), /30:00 h/);
  await presentador.ejecutar({
    tipo: "solicitar_permiso",
    permiso_id: "bolsa_conciliacion",
    desde: "2026-08-10",
    hasta: "2026-08-10",
    cantidad: 120,
    motivo: "Escenario de prueba",
    documento_ref: "",
  });
  const solicitud = presentador.obtenerEstado().datos.solicitudes[0];
  assert.equal(solicitud.cantidad_valor, 120);
  assert.equal(solicitud.unidad_clave, "minuto");
  assert.match(presentador.renderizar(), /02:00 h/);
});

test("el contrato anidado rechaza campos y referencias repetidas", () => {
  const contexto = contextoPresentacion();
  const datos = crearDatosCronosPresentacion(contexto);
  assert.throws(() => validarDatosCronos({
    ...datos,
    fichajes: [datos.fichajes[0], { ...datos.fichajes[1], id: datos.fichajes[0].id }],
  }, contexto), /id repetida/);
  assert.throws(() => validarDatosCronos({
    ...datos,
    solicitudes: [{ ...datos.solicitudes[0], campo_inesperado: true }],
  }, contexto), /contrato cerrado/);
});

test("i18n cubre la interfaz completa y la lógica visual usa códigos canónicos", () => {
  const contexto = contextoPresentacion();
  const alternativo = Object.fromEntries(Object.keys(MENSAJES_CRONOS_ES).map((clave) => [clave, `XX_${clave}`]));
  const html = crearPresentadorCronos({
    contextoActor: contexto,
    capacidades: TODAS_LAS_CAPACIDADES,
    datos: crearDatosCronosPresentacion(contexto),
    mensajes: alternativo,
  }).renderizar();
  for (const clave of [
    "titulo", "fichajes_titulo", "horario_titulo", "permisos_titulo",
    "solicitud_titulo", "solicitudes_titulo", "historial_titulo", "descargar_recibo",
  ]) assert.match(html, new RegExp(`XX_${clave}`));
  const clavesEstaticas = [...vistaFuente.matchAll(/\bt\("([^"]+)"/g)].map((coincidencia) => coincidencia[1]);
  assert.ok(clavesEstaticas.length > 80);
  for (const clave of clavesEstaticas) assert.ok(Object.hasOwn(MENSAJES_CRONOS_ES, clave), `falta ${clave}`);
  assert.doesNotMatch(vistaFuente, /aprobad\|registrad\|pendiente/);
  assert.match(datosFuente, /estado_clave: "simulado"/);
  assert.match(datosFuente, /tipo_clave: "inicio_pausa"/);
  assert.doesNotMatch(adaptadorFuente, /getUTC(?:Hours|Date|Month)/);
});

test("CSS hereda el tema central y conserva el modelo en portátil y móvil", () => {
  assert.match(css, /var\(--portal-(?:tinta|fondo|superficie|borde|azul-600)\)/);
  assert.match(css, /@media \(max-width: 1180px\)/);
  assert.match(css, /@media \(max-width: 720px\)/);
  assert.match(css, /@media \(max-width: 420px\)/);
  assert.match(css, /overflow-x: auto/);
  assert.match(css, /position: sticky/);
  assert.match(css, /\.cronos-area/);
  assert.doesNotMatch(css, /font-family:/);
});

test("proyección estricta, observación escapada y recibo por ámbito", async () => {
  const contexto = contextoPresentacion();
  const datos = crearDatosCronosPresentacion(contexto);
  const soloPermiso = crearPresentadorCronos({ contextoActor: contexto, capacidades: [CAPACIDAD_CONSULTAR_PERMISOS, CAPACIDAD_SOLICITAR_PERMISO], datos });
  const estado = soloPermiso.obtenerEstado();
  assert.equal(estado.datos.fichajes.length, 0);
  assert.equal(estado.datos.incidencias.length, 0);
  assert.equal(estado.datos.historial.every((item) => item.ambito_clave === "permiso"), true);
  assert.doesNotMatch(soloPermiso.renderizar(), /DEMO-REC-FIC/);
  const p = crearPresentadorCronos({ contextoActor: contexto, capacidades: [CAPACIDAD_CONSULTAR_FICHAJES, CAPACIDAD_REGISTRAR_FICHAJE], datos, ejecutor: crearEjecutorCronosPresentacion(), descargarRecibo: async () => ({}) });
  const recibo = await p.ejecutar({ tipo: "preparar_observacion", incidencia_id: "DEMO-INC-20260719-01", observacion: "<img src=x onerror=alert(1)> texto" });
  assert.equal(recibo.ambito_clave, "fichaje");
  assert.doesNotMatch(p.renderizar(), /<img src=x/);
  await assert.doesNotReject(p.descargarReciboPDF.bind(null, recibo.referencia));
});

test("contrato cierra extras y ejecutar rechaza concurrencia antes del puerto", async () => {
  const contexto = contextoPresentacion(); const datos = crearDatosCronosPresentacion(contexto);
  assert.throws(() => validarDatosCronos({ ...datos, resumen: { ...datos.resumen, extra: true } }, contexto), /contrato cerrado/);
  assert.throws(() => validarDatosCronos({ ...datos, incidencias: [{ ...datos.incidencias[0], instante: "2026-07-19T09:00:00+02:00" }] }, contexto), /no válido/);
  let resolver; let llamadas = 0;
  const p = crearPresentadorCronos({ contextoActor: contexto, capacidades: [CAPACIDAD_REGISTRAR_FICHAJE], datos, ejecutor: () => { llamadas += 1; return new Promise((r) => { resolver = r; }); } });
  const primera = p.ejecutar({ tipo: "registrar_fichaje", movimiento: "entrada" });
  await assert.rejects(p.ejecutar({ tipo: "registrar_fichaje", movimiento: "salida" }), /ya está preparando/);
  assert.equal(llamadas, 1);
  resolver({ datos, recibo: { esquema: "vec.cronos.recibo.v1", referencia: "DEMO-REC-CON-9999", instante: "2026-07-19T10:00:00Z", operacion: "Muestra", estado: "no enviado", efectos_reales: false, estado_clave: "simulado", ambito_clave: "fichaje", actor_ref: contexto.actor.actor_ref, demostracion: true } });
  await primera;
});

test("montaje retirado ignora resolución y rechazo tardíos", async () => {
  const contexto = contextoPresentacion(); let resolver; let rechazar; let llamada = 0;
  const p = crearPresentadorCronos({ contextoActor: contexto, capacidades: [CAPACIDAD_REGISTRAR_FICHAJE], datos: crearDatosCronosPresentacion(contexto), ejecutor: () => new Promise((resolve, reject) => { llamada += 1; resolver = resolve; rechazar = reject; }) });
  const handlers = {}; let cambios = 0; let anuncios = 0;
  const raiz = { innerHTML: "sin tocar", addEventListener: (tipo, fn) => { handlers[tipo] = fn; }, removeEventListener: () => {}, contains: () => true };
  const quitar = p.instalarEventos({ raiz, alCambiar: () => { cambios += 1; }, anunciar: () => { anuncios += 1; } });
  handlers.click({ preventDefault() {}, target: { closest: (sel) => sel.includes("registrar-fichaje") ? { dataset: { cronosTipo: "entrada" } } : null } });
  assert.equal(llamada, 1); quitar();
  resolver({ datos: crearDatosCronosPresentacion(contexto), recibo: { esquema: "vec.cronos.recibo.v1", referencia: "DEMO-REC-LATE-0001", instante: "2026-07-19T10:00:00Z", operacion: "Muestra", estado: "no enviado", efectos_reales: false, estado_clave: "simulado", ambito_clave: "fichaje", actor_ref: contexto.actor.actor_ref, demostracion: true } });
  await new Promise((r) => setTimeout(r, 0));
  assert.equal(raiz.innerHTML, "sin tocar"); assert.equal(cambios, 0); assert.equal(anuncios, 0);
  let falloTardio;
  const pError = crearPresentadorCronos({ contextoActor: contexto, capacidades: [CAPACIDAD_REGISTRAR_FICHAJE], datos: crearDatosCronosPresentacion(contexto), ejecutor: () => new Promise((_resolve, reject) => { falloTardio = reject; }) });
  const handlersError = {}; const raizError = { innerHTML: "sin tocar error", addEventListener: (tipo, fn) => { handlersError[tipo] = fn; }, removeEventListener: () => {}, contains: () => true };
  const quitarError = pError.instalarEventos({ raiz: raizError, alCambiar: () => { cambios += 1; }, anunciar: () => { anuncios += 1; } });
  handlersError.click({ preventDefault() {}, target: { closest: (sel) => sel.includes("registrar-fichaje") ? { dataset: { cronosTipo: "entrada" } } : null } });
  quitarError(); falloTardio(new Error("secreto interno")); await new Promise((r) => setTimeout(r, 0));
  assert.equal(raizError.innerHTML, "sin tocar error"); assert.equal(cambios, 0); assert.equal(anuncios, 0);
  void rechazar;
});

test("estado público proyecta cada resumen con ausencia null en las 32 combinaciones de capacidades", () => {
  const contexto = contextoPresentacion();
  const datos = crearDatosCronosPresentacion(contexto);
  datos.resumen = { teoricas_hoy: "TEORICA_CANARIO", trabajadas_hoy: "TRABAJADO_CANARIO", saldo_hoy: "SALDO_DIA_CANARIO", saldo_periodo: "SALDO_PERIODO_CANARIO", incidencias_abiertas: 713, solicitudes_pendientes: 917 };
  const capacidadPorCampo = {
    teoricas_hoy: CAPACIDAD_CONSULTAR_HORARIO,
    trabajadas_hoy: CAPACIDAD_CONSULTAR_FICHAJES,
    saldo_hoy: CAPACIDAD_CONSULTAR_FICHAJES,
    saldo_periodo: CAPACIDAD_CONSULTAR_FICHAJES,
    incidencias_abiertas: CAPACIDAD_CONSULTAR_FICHAJES,
    solicitudes_pendientes: CAPACIDAD_CONSULTAR_PERMISOS,
  };
  for (let mascara = 0; mascara < 32; mascara += 1) {
    const capacidades = TODAS_LAS_CAPACIDADES.filter((_capacidad, indice) => mascara & (1 << indice));
    const p = crearPresentadorCronos({ contextoActor: contexto, capacidades, datos });
    const proyectados = p.obtenerEstado().datos;
    const html = p.renderizar();
    for (const [campo, capacidad] of Object.entries(capacidadPorCampo)) {
      const autorizada = capacidades.includes(capacidad)
        && (!["saldo_hoy", "saldo_periodo"].includes(campo) || capacidades.includes(CAPACIDAD_CONSULTAR_HORARIO));
      assert.equal(proyectados.resumen[campo], autorizada ? datos.resumen[campo] : null, `${mascara}:${campo}`);
      if (!autorizada) assert.ok(!html.includes(String(datos.resumen[campo])), `${mascara}:${campo} no llega al DOM`);
    }
    assert.deepEqual(proyectados.perfil_jornada, capacidades.includes(CAPACIDAD_CONSULTAR_HORARIO) ? datos.perfil_jornada : null);
    assert.deepEqual(proyectados.saldos, capacidades.includes(CAPACIDAD_CONSULTAR_PERMISOS) ? datos.saldos : []);
    assert.deepEqual(proyectados.fichajes, capacidades.includes(CAPACIDAD_CONSULTAR_FICHAJES) ? datos.fichajes : []);
    // Alterar la copia pública nunca permite recuperar datos ni cambia la fuente.
    proyectados.resumen.teoricas_hoy = "MODIFICADO";
    assert.notEqual(p.obtenerEstado().datos.resumen.teoricas_hoy, "MODIFICADO");
  }
  const vacio = crearPresentadorCronos({ contextoActor: contexto, capacidades: [], datos }).obtenerEstado().datos;
  assert.ok(Object.values(vacio.resumen).every((valor) => valor === null));
  assert.equal(vacio.historial.length, 0);
  assert.equal(vacio.incidencias.length, 0);
  assert.throws(() => validarDatosCronos(vacio, contexto), /incompleto/);
});

test("contrato rechaza recibos homónimos y contradicciones de ámbito entre historial y registros", () => {
  const contexto = contextoPresentacion();
  const inicial = crearDatosCronosPresentacion(contexto);
  const ataques = [
    (datos) => { datos.solicitudes[0].recibo_ref = datos.fichajes[0].recibo_ref; },
    (datos) => { datos.historial[0].recibo_ref = datos.solicitudes[0].recibo_ref; datos.historial.splice(1, 1); },
    (datos) => { datos.historial[1].recibo_ref = datos.fichajes[0].recibo_ref; },
    (datos) => { datos.historial[1].ambito_clave = "fichaje"; },
  ];
  for (const alterar of ataques) {
    const datos = structuredClone(inicial); alterar(datos);
    assert.throws(() => validarDatosCronos(datos, contexto), /ámbitos incompatibles/);
    for (const capacidades of [[], [CAPACIDAD_CONSULTAR_FICHAJES], [CAPACIDAD_CONSULTAR_PERMISOS], TODAS_LAS_CAPACIDADES]) {
      assert.throws(() => crearPresentadorCronos({ contextoActor: contexto, capacidades, datos }), /ámbitos incompatibles/);
    }
  }
  const coherentes = structuredClone(inicial);
  coherentes.historial[0].recibo_ref = coherentes.fichajes[0].recibo_ref;
  assert.doesNotThrow(() => validarDatosCronos(coherentes, contexto));
});

test("recibo de observación sin registro solo se descarga en el ámbito fichaje explícito", async () => {
  const contexto = contextoPresentacion();
  const p = crearPresentadorCronos({ contextoActor: contexto, capacidades: TODAS_LAS_CAPACIDADES, datos: crearDatosCronosPresentacion(contexto), ejecutor: crearEjecutorCronosPresentacion() });
  const recibo = await p.ejecutar({ tipo: "preparar_observacion", incidencia_id: "DEMO-INC-20260719-01", observacion: "Observación sintética reservada a fichaje" });
  const datos = p.obtenerEstado().datos;
  assert.equal(datos.fichajes.some((fila) => fila.recibo_ref === recibo.referencia), false);
  const lector = (capacidades) => crearPresentadorCronos({ contextoActor: contexto, capacidades, datos });
  const ficha = lector([CAPACIDAD_CONSULTAR_FICHAJES]);
  const descriptor = ficha.prepararDescriptorRecibo(recibo.referencia);
  assert.equal(descriptor.clase, "historial");
  assert.ok(descriptor.filas.some((fila) => fila.valor === "Observación sintética reservada a fichaje"));
  assert.doesNotMatch(JSON.stringify(descriptor), /Vacaciones|Asuntos propios|Conciliación/);
  for (const capacidades of [[], [CAPACIDAD_CONSULTAR_PERMISOS], [CAPACIDAD_REGISTRAR_FICHAJE]]) {
    assert.throws(() => lector(capacidades).prepararDescriptorRecibo(recibo.referencia), /mínimo privilegio/);
  }
  const permiso = lector([CAPACIDAD_CONSULTAR_PERMISOS]);
  assert.equal(permiso.prepararDescriptorRecibo(datos.solicitudes[0].recibo_ref).clase, "permiso");
  assert.throws(() => ficha.prepararDescriptorRecibo(datos.solicitudes[0].recibo_ref), /mínimo privilegio/);
  assert.throws(() => permiso.prepararDescriptorRecibo(datos.fichajes[0].recibo_ref), /mínimo privilegio/);
  assert.throws(() => ficha.prepararDescriptorRecibo("DEMO-REC-DESCONOCIDO"), /fuera del ámbito/);
});

test("recibos de sesión no pueden contradecir un registro ni reasignarse a otro ámbito", async () => {
  const contexto = contextoPresentacion();
  const inicial = crearDatosCronosPresentacion(contexto);
  const respuesta = (datos, referencia, ambito) => ({ datos, recibo: {
    esquema: "vec.cronos.recibo.v1", actor_ref: contexto.actor.actor_ref,
    referencia, ambito_clave: ambito, instante: "2026-07-19T10:00:00Z",
    operacion: "Muestra", estado: "Sin efectos", estado_clave: "simulado", efectos_reales: false,
  } });
  for (const referencia of [inicial.solicitudes[0].recibo_ref, inicial.historial[1].recibo_ref]) {
    const p = crearPresentadorCronos({ contextoActor: contexto, capacidades: TODAS_LAS_CAPACIDADES, datos: inicial, ejecutor: async () => respuesta(inicial, referencia, "fichaje") });
    const antes = p.obtenerEstado();
    await assert.rejects(p.ejecutar({ tipo: "registrar_fichaje", movimiento: "entrada" }), /ámbitos incompatibles/);
    assert.deepEqual(p.obtenerEstado(), antes);
  }
  for (const colisionEnRegistro of [false, true]) {
    let llamadas = 0;
    const p = crearPresentadorCronos({ contextoActor: contexto, capacidades: TODAS_LAS_CAPACIDADES, datos: inicial, ejecutor: async () => {
      llamadas += 1;
      if (llamadas === 1) return respuesta(inicial, "DEMO-REC-SESION-CRUCE", "fichaje");
      const nuevos = structuredClone(inicial);
      if (colisionEnRegistro) nuevos.solicitudes[0].recibo_ref = "DEMO-REC-SESION-CRUCE";
      return respuesta(nuevos, colisionEnRegistro ? "DEMO-REC-SESION-NUEVO" : "DEMO-REC-SESION-CRUCE", "permiso");
    } });
    await p.ejecutar({ tipo: "registrar_fichaje", movimiento: "entrada" });
    const antes = p.obtenerEstado();
    await assert.rejects(p.ejecutar({ tipo: "solicitar_permiso", permiso_id: "asuntos_propios", cantidad: 1, desde: "2026-08-01", hasta: "2026-08-01" }), /ámbitos incompatibles/);
    assert.deepEqual(p.obtenerEstado(), antes);
  }
});

test("cada campo de listas rechaza estructuras, tipos impropios, exceso y caracteres de control", () => {
  const contexto = contextoPresentacion();
  const inicial = crearDatosCronosPresentacion(contexto);
  for (const lista of ["fichajes", "saldos", "solicitudes", "historial", "incidencias"]) {
    for (const [campo, original] of Object.entries(inicial[lista][0])) {
      const invalidos = [{ dato_privado_canario: "NO_FILTRAR" }, ["NO_FILTRAR"], null, "x".repeat(501), "texto\u0000canario", "texto\u0085canario"];
      if (typeof original !== "boolean") invalidos.push(true);
      if (lista !== "historial" || campo !== "detalle") invalidos.push("texto\ncanario");
      invalidos.push(typeof original === "number" ? "123" : 123);
      for (const valor of invalidos) {
        const datos = structuredClone(inicial); datos[lista][0][campo] = valor;
        assert.throws(() => validarDatosCronos(datos, contexto), undefined, `${lista}.${campo}: ${JSON.stringify(valor)}`);
      }
    }
  }
  const limites = [
    ["fichajes", "canal", 180], ["fichajes", "modalidad", 180], ["saldos", "nombre", 180], ["solicitudes", "tipo", 180],
    ["historial", "evento", 180], ["historial", "detalle", 500], ["incidencias", "categoria_clave", 80], ["incidencias", "resumen", 180], ["incidencias", "detalle", 500],
  ];
  for (const [lista, campo, maximo] of limites) {
    const datos = structuredClone(inicial); datos[lista][0][campo] = "a".repeat(maximo);
    assert.doesNotThrow(() => validarDatosCronos(datos, contexto));
    datos[lista][0][campo] += "a";
    assert.throws(() => validarDatosCronos(datos, contexto), /no válido/);
  }
  for (const seccion of ["resumen", "perfil_jornada"]) {
    for (const [campo, valor] of Object.entries(inicial[seccion])) {
      if (typeof valor !== "string") continue;
      const datos = structuredClone(inicial); datos[seccion][campo] = "canario\u007f";
      assert.throws(() => validarDatosCronos(datos, contexto), /no válido/);
    }
  }
});

test("trabajadas no permite reconstruir teóricas por saldo ni por historial sin horario", () => {
  const contexto = contextoPresentacion(); const datos = crearDatosCronosPresentacion(contexto);
  const minutos = (valor) => { const [h, m] = valor.replace("+", "").split(":").map(Number); return h * 60 + m; };
  assert.equal(minutos(datos.resumen.trabajadas_hoy) - minutos(datos.resumen.saldo_hoy), minutos(datos.resumen.teoricas_hoy));
  const sensible = datos.historial.find((item) => item.requiere_horario);
  const soloFicha = crearPresentadorCronos({ contextoActor: contexto, capacidades: [CAPACIDAD_CONSULTAR_FICHAJES], datos });
  const publico = soloFicha.obtenerEstado().datos;
  assert.equal(publico.resumen.trabajadas_hoy, "07:36");
  for (const campo of ["teoricas_hoy", "saldo_hoy", "saldo_periodo"]) assert.equal(publico.resumen[campo], null);
  assert.ok(!publico.historial.some((item) => item.id === sensible.id));
  assert.doesNotMatch(soloFicha.renderizar(), /07:30|\+00:06|\+02:18|Saldo diario calculado|DEMO-REC-JOR-0719/);
  assert.throws(() => soloFicha.prepararDescriptorRecibo(sensible.recibo_ref), /mínimo privilegio/);
  const autorizado = crearPresentadorCronos({ contextoActor: contexto, capacidades: [CAPACIDAD_CONSULTAR_FICHAJES, CAPACIDAD_CONSULTAR_HORARIO], datos });
  assert.equal(autorizado.obtenerEstado().datos.resumen.saldo_hoy, "+00:06");
  assert.ok(autorizado.prepararDescriptorRecibo(sensible.recibo_ref).filas.some((fila) => fila.valor === sensible.detalle));
  // Una referencia compartida dentro del mismo ámbito tampoco rebaja la protección.
  datos.historial[0].recibo_ref = datos.fichajes[0].recibo_ref;
  const compartido = crearPresentadorCronos({ contextoActor: contexto, capacidades: [CAPACIDAD_CONSULTAR_FICHAJES], datos });
  assert.throws(() => compartido.prepararDescriptorRecibo(datos.fichajes[0].recibo_ref), /mínimo privilegio/);
  assert.doesNotMatch(JSON.stringify(compartido.prepararDescriptorRecibo(datos.fichajes[1].recibo_ref)), /Saldo diario|\+00:06/);
});

test("render directo filtra cada historial y recibo por ámbito y horario en las 32 combinaciones", () => {
  const contexto = contextoPresentacion(); const datos = crearDatosCronosPresentacion(contexto);
  datos.historial.push({ ...datos.historial[0], id: "DEMO-HIS-NORMAL", recibo_ref: "DEMO-REC-NORMAL", requiere_horario: false, evento: "FICHA_NORMAL_VISIBLE", detalle: "Detalle ordinario" });
  datos.historial[0].evento = "FICHA_HORARIO_VISIBLE";
  datos.historial[1].evento = "PERMISO_NORMAL_VISIBLE";
  datos.historial[2].evento = "PERMISO_HORARIO_VISIBLE"; datos.historial[2].requiere_horario = true;
  const referencias = [datos.fichajes[0].recibo_ref, datos.solicitudes[0].recibo_ref, datos.historial[1].recibo_ref, datos.historial[2].recibo_ref];
  const recibos = ["fichaje", "permiso", "desconocido", undefined].map((ambito, indice) => ({
    referencia: referencias[indice], instante: datos.actualizado_en, ambito_clave: ambito,
    operacion: `RECIBO_DIRECTO_${indice}`, estado_clave: "simulado",
  }));
  for (let mascara = 0; mascara < 32; mascara += 1) {
    const capacidades = TODAS_LAS_CAPACIDADES.filter((_c, indice) => mascara & (1 << indice));
    const ficha = capacidades.includes(CAPACIDAD_CONSULTAR_FICHAJES);
    const permiso = capacidades.includes(CAPACIDAD_CONSULTAR_PERMISOS);
    const horario = capacidades.includes(CAPACIDAD_CONSULTAR_HORARIO);
    const html = renderizarAreaCronos({ contextoActor: contexto, capacidades, datos, recibos });
    for (const [texto, esperado] of [["FICHA_NORMAL_VISIBLE", ficha], ["FICHA_HORARIO_VISIBLE", ficha && horario], ["PERMISO_NORMAL_VISIBLE", permiso], ["PERMISO_HORARIO_VISIBLE", permiso && horario], ["RECIBO_DIRECTO_0", ficha], ["RECIBO_DIRECTO_1", permiso], ["RECIBO_DIRECTO_2", false], ["RECIBO_DIRECTO_3", false]]) {
      assert.equal(html.includes(texto), esperado, `${mascara}:${texto}`);
    }
    if (!ficha || !horario) assert.doesNotMatch(html, /<strong>\+00:06<\/strong>|<strong>\+02:18<\/strong>/);
    const proyectado = crearPresentadorCronos({ contextoActor: contexto, capacidades, datos }).obtenerEstado().datos;
    assert.deepEqual(proyectado.historial, datos.historial.filter((item) => (item.ambito_clave === "fichaje" ? ficha : permiso) && (!item.requiere_horario || horario)));
  }
  for (const valor of [undefined, null, 0, "false", {}]) {
    const alterados = structuredClone(datos);
    if (valor === undefined) delete alterados.historial[0].requiere_horario;
    else alterados.historial[0].requiere_horario = valor;
    assert.throws(() => validarDatosCronos(alterados, contexto));
  }
});

test("textarea conserva dos líneas y escapa HTML; normaliza retornos y rechaza controles antes del puerto", async () => {
  const contexto = contextoPresentacion(); const datos = crearDatosCronosPresentacion(contexto);
  let llamadas = 0; const ejecutar = crearEjecutorCronosPresentacion();
  const p = crearPresentadorCronos({ contextoActor: contexto, capacidades: TODAS_LAS_CAPACIDADES, datos,
    ejecutor: async (...args) => { llamadas += 1; return ejecutar(...args); } });
  const manejadores = {};
  const raiz = { innerHTML: "", addEventListener: (tipo, fn) => { manejadores[tipo] = fn; }, removeEventListener() {}, contains: () => true,
    querySelector: () => ({ value: "Primera línea\r\n<img src=x onerror=alert(1)>\rTercera línea" }) };
  const quitar = p.instalarEventos({ raiz });
  manejadores.click({ preventDefault() {}, target: { closest: (sel) => sel.includes("preparar-observacion") ? { dataset: { cronosIncidenciaRef: datos.incidencias[0].id } } : null } });
  await new Promise((r) => setTimeout(r, 0)); quitar();
  const texto = "Primera línea\n<img src=x onerror=alert(1)>\nTercera línea";
  assert.equal(llamadas, 1); assert.equal(p.obtenerEstado().datos.historial[0].detalle, texto);
  assert.equal(p.obtenerEstado().datos.historial[0].requiere_horario, false);
  assert.match(raiz.innerHTML, /Primera línea<br>&lt;img src=x onerror=alert\(1\)&gt;<br>Tercera línea/);
  assert.doesNotMatch(raiz.innerHTML, /<img src=x/);
  assert.ok(p.prepararDescriptorRecibo(p.obtenerEstado().recibos[0].referencia).filas.some((fila) => fila.valor === texto));
  const antes = p.obtenerEstado();
  for (const observacion of [{ toString: () => "texto objeto" }, ["texto array"], "corto", "x".repeat(501), ...["\u0000", "\t", "\u000b", "\u007f", "\u0085"].map((c) => `Primera${c}segunda`)]) {
    await assert.rejects(p.ejecutar({ tipo: "preparar_observacion", incidencia_id: datos.incidencias[0].id, observacion }), /no válido/);
    assert.equal(llamadas, 1); assert.deepEqual(p.obtenerEstado(), antes);
  }
  const multilinea = structuredClone(datos); multilinea.historial[0].detalle = "Primera\r\nSegunda\rTercera";
  assert.equal(validarDatosCronos(multilinea, contexto).historial[0].detalle, "Primera\nSegunda\nTercera");
});

test("render directo sólo muestra recibos con procedencia coherente del actor y ámbito", () => {
  const contexto = contextoPresentacion();
  const datos = crearDatosCronosPresentacion(contexto);
  const [fichaje] = datos.fichajes;
  const [solicitud] = datos.solicitudes;
  const historicoHorario = datos.historial.find((item) => item.requiere_horario);
  const historicoNormal = datos.historial.find((item) => item.ambito_clave === "permiso" && !item.requiere_horario);
  const recibos = [
    { referencia: fichaje.recibo_ref, instante: datos.actualizado_en, ambito_clave: "fichaje", operacion: "RECIBO_FICHAJE_VALIDO", estado_clave: "simulado" },
    { referencia: fichaje.recibo_ref, instante: datos.actualizado_en, ambito_clave: "permiso", operacion: "RECIBO_FICHAJE_DISFRAZADO", estado_clave: "simulado" },
    { referencia: solicitud.recibo_ref, instante: datos.actualizado_en, ambito_clave: "fichaje", operacion: "RECIBO_PERMISO_DISFRAZADO", estado_clave: "simulado" },
    { referencia: solicitud.recibo_ref, instante: datos.actualizado_en, ambito_clave: "permiso", operacion: "RECIBO_PERMISO_VALIDO", estado_clave: "simulado" },
    { referencia: historicoHorario.recibo_ref, instante: datos.actualizado_en, ambito_clave: "fichaje", operacion: "RECIBO_HORARIO", estado_clave: "simulado" },
    { referencia: historicoNormal.recibo_ref, instante: datos.actualizado_en, ambito_clave: "permiso", operacion: "RECIBO_SESION_OBSERVACION_ASOCIADA", estado_clave: "simulado" },
    { referencia: fichaje.recibo_ref, instante: datos.actualizado_en, ambito_clave: "fichaje", actor_ref: "DEMO-ACTOR-AJENO-02", operacion: "RECIBO_ACTOR_AJENO", estado_clave: "simulado" },
    { referencia: "DEMO-REC-SIN-ASOCIACION", instante: datos.actualizado_en, ambito_clave: "fichaje", operacion: "RECIBO_SIN_ASOCIACION", estado_clave: "simulado" },
  ];
  const sinHorario = renderizarAreaCronos({
    contextoActor: contexto,
    capacidades: [CAPACIDAD_CONSULTAR_FICHAJES, CAPACIDAD_CONSULTAR_PERMISOS],
    datos, recibos,
  });
  for (const texto of ["RECIBO_FICHAJE_VALIDO", "RECIBO_PERMISO_VALIDO", "RECIBO_SESION_OBSERVACION_ASOCIADA"]) assert.match(sinHorario, new RegExp(texto));
  for (const texto of ["RECIBO_FICHAJE_DISFRAZADO", "RECIBO_PERMISO_DISFRAZADO", "RECIBO_HORARIO", "RECIBO_ACTOR_AJENO", "RECIBO_SIN_ASOCIACION"]) assert.doesNotMatch(sinHorario, new RegExp(texto));
  const conHorario = renderizarAreaCronos({
    contextoActor: contexto,
    capacidades: [CAPACIDAD_CONSULTAR_FICHAJES, CAPACIDAD_CONSULTAR_PERMISOS, CAPACIDAD_CONSULTAR_HORARIO],
    datos, recibos,
  });
  assert.match(conHorario, /RECIBO_HORARIO/);
});

test("montajes concurrentes de la misma raíz sólo ejecutan y conservan al propietario actual", async () => {
  const contexto = contextoPresentacion();
  const datos = crearDatosCronosPresentacion(contexto);
  let ejecucionesA = 0;
  let ejecucionesB = 0;
  const reciboB = () => ({
    esquema: "vec.cronos.recibo.v1", actor_ref: contexto.actor.actor_ref,
    referencia: "DEMO-REC-PROPIETARIO-" + String(ejecucionesB).padStart(4, "0"),
    instante: "2026-07-19T10:00:00Z", operacion: "Muestra",
    estado: "no enviado", efectos_reales: false, estado_clave: "simulado", ambito_clave: "fichaje",
  });
  const a = crearPresentadorCronos({
    contextoActor: contexto, capacidades: [CAPACIDAD_REGISTRAR_FICHAJE], datos,
    ejecutor: async () => { ejecucionesA += 1; return { datos, recibo: reciboB() }; },
  });
  const b = crearPresentadorCronos({
    contextoActor: contexto, capacidades: [CAPACIDAD_REGISTRAR_FICHAJE], datos,
    ejecutor: async () => { ejecucionesB += 1; return { datos, recibo: reciboB() }; },
  });
  const manejadores = { click: [], submit: [] };
  const raiz = {
    innerHTML: "sin tocar",
    addEventListener: (tipo, fn) => { manejadores[tipo].push(fn); },
    removeEventListener: (tipo, fn) => { manejadores[tipo] = manejadores[tipo].filter((actual) => actual !== fn); },
    contains: () => true,
  };
  const quitarA = a.instalarEventos({ raiz });
  const quitarB = b.instalarEventos({ raiz });
  const evento = { preventDefault() {}, target: { closest: (selector) => selector.includes("registrar-fichaje") ? { dataset: { cronosTipo: "entrada" } } : null } };
  for (const manejador of [...manejadores.click]) manejador(evento);
  await new Promise((resolver) => setTimeout(resolver, 0));
  assert.equal(ejecucionesA, 0);
  assert.equal(ejecucionesB, 1);
  quitarA();
  for (const manejador of [...manejadores.click]) manejador(evento);
  await new Promise((resolver) => setTimeout(resolver, 0));
  assert.equal(ejecucionesA, 0);
  assert.equal(ejecucionesB, 2);
  quitarB();
});
