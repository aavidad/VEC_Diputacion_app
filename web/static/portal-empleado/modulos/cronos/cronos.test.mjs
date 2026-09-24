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
import { montarJornadaCronos, renderizarJornadaCronos, validarSeleccionPeriodoCronos } from "./vista.js";
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

test("Jornada sin servicio muestra estados honestos y no expone cifras ni acciones", () => {
  for (const estado of ["cargando", "vacio", "no_configurado", "denegado", "error"]) {
    const html = renderizarJornadaCronos({ estado });
    assert.match(html, new RegExp(`data-estado="${estado}"`));
    assert.doesNotMatch(html, /07:36|DEMO-REC-FIC-1900|tabla-cronos-jornada/);
    assert.ok((html.match(/disabled aria-disabled="true"/g) || []).length === 2);
  }
  assert.match(renderizarJornadaCronos({ estado: "cargando" }), /aria-busy="true"/);
  assert.throws(() => renderizarJornadaCronos({ estado: "inventado" }), /estado de jornada/);
});

test("Jornada permite elegir escala y rango, y distingue un periodo sin proyección", () => {
  const inicial = renderizarJornadaCronos();
  for (const tipo of ["dia", "semana", "mes", "anio", "periodo"]) {
    assert.match(inicial, new RegExp(`value="${tipo}"`));
  }
  assert.match(inicial, /data-cronos-form-periodo/);
  assert.match(inicial, /Fecha de referencia/);
  const contexto = contextoReal();
  const seleccion = validarSeleccionPeriodoCronos({ tipo: "semana", desde: "2026-09-24" });
  const html = renderizarJornadaCronos({
    estado: "disponible", contextoActor: contexto, datos: datosRealesPara(contexto),
    capacidades: [CAPACIDAD_CONSULTAR_HORARIO, CAPACIDAD_CONSULTAR_FICHAJES], seleccion,
  });
  assert.match(html, /Semana del 21\/09\/2026 al 27\/09\/2026/);
  assert.match(html, /Falta una proyección autorizada para el periodo seleccionado/);
  assert.match(html, /Jornada teórica.*Tiempo trabajado.*Permisos.*Saldo/s);
  assert.doesNotMatch(html, /tabla-cronos-jornada|Jornada diaria|08:01|DEMO-REC-FIC-1900/);
  for (const [tipo, esperado] of [
    ["dia", "Día 24/09/2026"], ["mes", "Mes de septiembre de 2026"], ["anio", "Año 2026"],
  ]) {
    assert.match(renderizarJornadaCronos({ seleccion: validarSeleccionPeriodoCronos({ tipo, desde: "2026-09-24" }) }),
      new RegExp(esperado));
  }
  const denegado = renderizarJornadaCronos({ estado: "denegado", seleccion });
  assert.match(denegado, /Acceso denegado/);
  assert.doesNotMatch(denegado, /Falta una proyección autorizada|data-cronos-form-periodo/);
});

test("La selección rechaza fechas imposibles y rangos invertidos", () => {
  assert.throws(() => validarSeleccionPeriodoCronos({ tipo: "dia", desde: "2026-02-30" }), /fecha/);
  assert.throws(() => validarSeleccionPeriodoCronos({ tipo: "periodo", desde: "2026-09-25", hasta: "2026-09-24" }), /rango/);
  assert.deepEqual(validarSeleccionPeriodoCronos({ tipo: "periodo", desde: "2026-09-24", hasta: "2026-09-24" }),
    { tipo: "periodo", desde: "2026-09-24", hasta: "2026-09-24" });
});

test("El formulario aplica la selección y retira sus oyentes al desmontar", () => {
  let contenedor;
  const raiz = {
    ownerDocument: { createElement: () => {
      contenedor = { dataset: {}, innerHTML: "", oyentes: new Map(),
        addEventListener(tipo, fn) { this.oyentes.set(tipo, fn); },
        removeEventListener(tipo) { this.oyentes.delete(tipo); },
        remove() {},
      };
      return contenedor;
    } },
    append() {},
  };
  const anuncios = [];
  const vista = montarJornadaCronos({ raiz, anunciar: (texto) => anuncios.push(texto) });
  let focoHasta = false;
  const controles = new Map(Object.entries({ tipo: { value: "periodo" }, desde: { value: "2026-09-24" },
    hasta: { value: "2026-09-23", focus() { focoHasta = true; } } }));
  let prevenido = false;
  const error = { hidden: true, textContent: "" };
  const formulario = { matches: () => true, elements: { namedItem: (nombre) => controles.get(nombre) },
    querySelector: () => error };
  contenedor.oyentes.get("submit")({ target: formulario, preventDefault() { prevenido = true; } });
  assert.equal(focoHasta, true);
  assert.equal(error.hidden, false);
  assert.match(error.textContent, /Revise las fechas/);
  controles.get("hasta").value = "2026-09-25";
  contenedor.oyentes.get("submit")({ target: formulario, preventDefault() { prevenido = true; } });
  assert.equal(prevenido, true);
  assert.match(contenedor.innerHTML, /Periodo del 24\/09\/2026 al 25\/09\/2026/);
  assert.match(anuncios.at(-1), /Periodo del 24\/09\/2026 al 25\/09\/2026/);
  controles.get("hasta").value = "2026-09-26";
  contenedor.oyentes.get("submit")({ target: formulario, preventDefault() {} });
  assert.match(contenedor.innerHTML, /Periodo del 24\/09\/2026 al 26\/09\/2026/);
  vista.desmontar();
  assert.equal(contenedor.oyentes.size, 0);
});

test("La selección conserva la denegación de una proyección real sin capacidades", () => {
  const contexto = contextoReal();
  const datos = datosRealesPara(contexto);
  const seleccion = validarSeleccionPeriodoCronos({ tipo: "semana", desde: "2026-09-24" });
  const proyeccion = { estado: "disponible", contextoActor: contexto, datos, capacidades: [] };
  const html = renderizarJornadaCronos({ ...proyeccion, seleccion });
  assert.match(html, /data-estado="denegado"/);
  assert.match(html, /Acceso denegado/);
  assert.match(html, /data-cronos-jornada-resultado tabindex="-1"/);
  assert.doesNotMatch(html, /Servicio pendiente|Falta una proyección autorizada|data-cronos-form-periodo/);

  let contenedor;
  let focos = 0;
  let focoDenegado = 0;
  const documento = { activeElement: {}, createElement: () => {
    contenedor = { dataset: {}, innerHTML: "", oyentes: new Map(),
      addEventListener(tipo, fn) { this.oyentes.set(tipo, fn); },
      removeEventListener(tipo) { this.oyentes.delete(tipo); },
      contains: () => true,
      querySelector(selector) {
        if (selector.includes("data-cronos-periodo-resultado") && this.innerHTML.includes("data-cronos-periodo-resultado")) {
          return { focus() { focos += 1; } };
        }
        if (selector.includes("data-cronos-jornada-resultado") && this.innerHTML.includes('data-estado="denegado"')) {
          return { focus() { focoDenegado += 1; } };
        }
        return null;
      },
      remove() {},
    };
    return contenedor;
  } };
  const raiz = { ownerDocument: documento, append() {} };
  const anuncios = [];
  const vista = montarJornadaCronos({ raiz, anunciar: (mensaje) => anuncios.push(mensaje) });
  const controles = new Map(Object.entries({ tipo: { value: "semana" }, desde: { value: "2026-09-24" }, hasta: { value: "" } }));
  const formulario = { matches: () => true, elements: { namedItem: (nombre) => controles.get(nombre) } };
  contenedor.oyentes.get("submit")({ target: formulario, preventDefault() {} });
  assert.ok(focos >= 1);
  vista.actualizar(proyeccion);
  assert.match(contenedor.innerHTML, /data-estado="denegado"/);
  assert.equal(focoDenegado, 1);
  assert.equal(anuncios.at(-1), "Acceso denegado");
  vista.desmontar();
});

test("Jornada disponible exige proyección propia y oculta bloques sin capacidad", () => {
  const contexto = contextoReal();
  const datos = datosRealesPara(contexto);
  const contextoDemo = contextoPresentacion();
  assert.match(renderizarJornadaCronos({ estado: "disponible", contextoActor: contexto, capacidades: [CAPACIDAD_CONSULTAR_FICHAJES] }), /data-estado="no_configurado"/);
  assert.match(renderizarJornadaCronos({
    estado: "disponible", contextoActor: contextoDemo,
    datos: crearDatosCronosPresentacion(contextoDemo), capacidades: [CAPACIDAD_CONSULTAR_FICHAJES],
  }), /data-estado="no_configurado"/);
  const soloHorario = renderizarJornadaCronos({
    estado: "disponible", contextoActor: contexto, datos,
    capacidades: [CAPACIDAD_CONSULTAR_HORARIO],
  });
  assert.match(soloHorario, /Datos de la consulta recibida/);
  assert.match(soloHorario, /La sesión no tiene capacidad para consultar fichajes/);
  assert.doesNotMatch(soloHorario, /DEMO-REC-FIC-1900|08:01|tabla-cronos-jornada/);
  const todo = renderizarJornadaCronos({
    estado: "disponible", contextoActor: contexto, datos,
    capacidades: [CAPACIDAD_CONSULTAR_HORARIO, CAPACIDAD_CONSULTAR_FICHAJES],
  });
  assert.match(todo, /tabla-cronos-jornada/);
  assert.match(todo, /Jornada diaria/);
  assert.match(todo, /Calendario laboral/);
  assert.match(todo, /Sin calendario autorizado para el centro y el periodo/);
  assert.doesNotMatch(todo, /cronos-mini-calendario|data-cronos-control|data-cronos-detalle/);
  assert.match(todo, /data-accion="ayuda"/);
  assert.doesNotMatch(todo, /data-cronos-accion="registrar-fichaje"/);
  const inyectado = renderizarJornadaCronos({
    estado: "disponible", contextoActor: contexto,
    datos: { ...datos, perfil_jornada: { ...datos.perfil_jornada, nombre: "<script>alert(1)</script>" } },
    capacidades: [CAPACIDAD_CONSULTAR_HORARIO],
  });
  assert.match(inyectado, /&lt;script&gt;alert\(1\)&lt;\/script&gt;/);
  assert.doesNotMatch(inyectado, /<script>/);
  assert.throws(() => renderizarJornadaCronos({
    estado: "disponible", contextoActor: contexto, datos: { ...datos, actor_ref: "REAL-OTRO-ACTOR" },
    capacidades: [CAPACIDAD_CONSULTAR_FICHAJES],
  }), /no pertenecen/);
});

test("Jornada se monta y desmonta sin listeners ni efectos", () => {
  const montajes = [];
  const raiz = {
    ownerDocument: { createElement: () => ({ dataset: {}, innerHTML: "", remove() { this.eliminado = true; } }) },
    append: (elemento) => montajes.push(elemento),
  };
  let desmontarRegistrado;
  const anuncios = [];
  const vista = montarJornadaCronos({ raiz, anunciar: (texto) => anuncios.push(texto), registrarDesmontar: (desmontar) => { desmontarRegistrado = desmontar; } });
  assert.equal(montajes.length, 1);
  assert.match(montajes[0].innerHTML, /Servicio pendiente/);
  vista.actualizar({ estado: "error" });
  assert.match(montajes[0].innerHTML, /Consulta no disponible/);
  vista.actualizar({ estado: "disponible" });
  assert.match(montajes[0].innerHTML, /data-estado="no_configurado"/);
  assert.equal(anuncios.at(-1), "Servicio pendiente");
  desmontarRegistrado();
  assert.equal(montajes[0].eliminado, true);
  assert.throws(() => vista.actualizar({ estado: "cargando" }), /desmontada/);
});

test("Jornada inserta el calendario civil y retira sus oyentes al actualizar y salir", () => {
  const creados = [];
  const destino = { ownerDocument: null, append(nodo) { this.ultimo = nodo; } };
  const documento = { createElement() {
    const nodo = { dataset: {}, innerHTML: "", oyentes: new Map(), addEventListener(tipo, fn) { this.oyentes.set(tipo, fn); },
      removeEventListener(tipo) { this.oyentes.delete(tipo); }, remove() { this.eliminado = true; },
      querySelector(selector) { return selector === "[data-cronos-calendario-raiz]" ? destino : null; } };
    creados.push(nodo);
    return nodo;
  } };
  destino.ownerDocument = documento;
  const vista = montarJornadaCronos({ raiz: { ownerDocument: documento, append() {} } });
  assert.match(creados[1].innerHTML, /Calendario civil/u);
  assert.match(creados[1].innerHTML, /Calendario laboral no configurado/u);
  assert.equal(creados[1].oyentes.size, 2);
  vista.actualizar({ estado: "error" });
  assert.equal(creados[1].eliminado, true);
  assert.equal(creados[1].oyentes.size, 0);
  assert.match(creados[2].innerHTML, /Calendario civil/u);
  vista.desmontar();
  assert.equal(creados[2].eliminado, true);
  assert.equal(creados[2].oyentes.size, 0);
});

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
  assert.match(html, /data-accion="ayuda"/);
  assert.match(html, /Entorno DEMO · datos sintéticos/);
  assert.match(html, /DEMO · sin registro, notificación ni aprobación/);
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

test("las operaciones DEMO son volátiles, trazadas y no contaminan el fixture", async () => {
  const contexto = contextoPresentacion();
  const inicial = crearDatosCronosPresentacion(contexto);
  const ejecutor = crearEjecutorCronosPresentacion({ reloj: () => new Date("2026-07-19T10:30:00Z") });
  const presentador = crearPresentadorCronos({ contextoActor: contexto, capacidades: TODAS_LAS_CAPACIDADES, datos: inicial, ejecutor });

  const reciboFichaje = await presentador.ejecutar({ tipo: "registrar_fichaje", movimiento: "entrada" });
  assert.equal(reciboFichaje.referencia, "DEMO-CRONOS-REC-0001");
  assert.match(reciboFichaje.estado, /sin persistencia/);
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
        actor_ref: contextoEjecucion.identidad.actor.actor_ref,
      },
    }),
  });
  assert.doesNotMatch(presentador.renderizar(), /Entorno DEMO/);
  assert.equal((await presentador.ejecutar({ tipo: "registrar_fichaje", movimiento: "salida" })).referencia, "CRONOS-REC-OPACA-0001");
  assert.match(presentadorFuente, /ejecutor\(comando/);
  assert.doesNotMatch(`${presentadorFuente}\n${vistaFuente}`, /fetch\(|XMLHttpRequest|document\.cookie|localStorage|sessionStorage/);
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
  assert.match(presentador.obtenerEstado().mensaje, /sesión efímera; no acredita registro ni efectos administrativos/);
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
  assert.match(datosFuente, /estado_clave: "registrado"/);
  assert.match(datosFuente, /tipo_clave: "inicio_pausa"/);
  assert.doesNotMatch(adaptadorFuente, /getUTC(?:Hours|Date|Month)/);
});

test("Cronos deja la orientación detallada al ayudante contextual sin ocultar límites", () => {
  const html = crearPresentadorCronos({
    contextoActor: contextoPresentacion(),
    capacidades: TODAS_LAS_CAPACIDADES,
    datos: crearDatosCronosPresentacion(contextoPresentacion()),
  }).renderizar();
  assert.ok((html.match(/data-accion="ayuda"/g) || []).length >= 8);
  assert.match(html, /DEMO · sin registro, notificación ni aprobación\./);
  assert.match(html, /Sin geolocalización ni datos de terceros\./);
  assert.doesNotMatch(html, /Introduzca días completos o minutos/);
  assert.doesNotMatch(html, /No incluya datos de salud u otros datos innecesarios/);
  assert.doesNotMatch(html, /Solo movimientos asociados al actor de esta sesión interna/);
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
