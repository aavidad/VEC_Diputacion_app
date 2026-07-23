/**
 * Adaptador exclusivo de presentacion para la identidad interna ya usada por
 * Bolsa. Cada perfil recibe un único ContextoActor y solo los módulos de su
 * punto de vista; este fichero no crea una identidad distinta por módulo.
 */

import {
  ESQUEMA_CONTEXTO_ACTOR_FRONTEND,
  validarYCongelarContextoActor,
} from "./contexto-actor.js";

const MODULOS_GESTION = Object.freeze(["bolsa", "contratacion_temporal"]);
const MODULOS_AUTOSERVICIO = Object.freeze(["cronos", "dietas"]);

const PERFILES_PRESENTACION = Object.freeze({
  "DEMO-PERFIL-ADMIN-FUNCIONAL-BOLSA-01": Object.freeze({
    nombre: "Administrador DEMO 01",
    iniciales: "AD",
    etiqueta_perfil: "Administrador funcional de Bolsa · ámbito DEMO completo",
    persona_ref: "per_demo_persona_interna_admin_000001",
    cuenta_ref: "cta_demo_cuenta_interna_admin_000001",
    perfil_ref: "prf_demo_perfil_admin_bolsa_000001",
    sesion_ref: "ses_demo_sesion_interna_admin_000001",
    rol: "administrador_funcional_bolsa",
    modulos: MODULOS_GESTION,
  }),
  "DEMO-PERFIL-TECNICO-RRHH-01": Object.freeze({
    nombre: "Técnico DEMO 01",
    iniciales: "TR",
    etiqueta_perfil: "Técnico revisor de RRHH · ámbito DEMO restringido",
    persona_ref: "per_demo_persona_interna_tecnica_000001",
    cuenta_ref: "cta_demo_cuenta_interna_tecnica_000001",
    perfil_ref: "prf_demo_perfil_tecnico_rrhh_000001",
    sesion_ref: "ses_demo_sesion_interna_tecnica_000001",
    rol: "tecnico_revisor_rrhh",
    modulos: MODULOS_GESTION,
  }),
  "DEMO-PERFIL-FUNCIONARIO-01": Object.freeze({
    nombre: "Funcionario DEMO 01",
    iniciales: "FU",
    etiqueta_perfil: "Funcionario · autoservicio interno DEMO",
    persona_ref: "per_demo_persona_interna_funcionaria_000001",
    cuenta_ref: "cta_demo_cuenta_interna_funcionaria_000001",
    perfil_ref: "prf_demo_perfil_funcionario_000001",
    sesion_ref: "ses_demo_sesion_interna_funcionaria_000001",
    rol: "funcionario_autoservicio",
    modulos: MODULOS_AUTOSERVICIO,
  }),
});

function exigirCadenaSesion(sesion, campo) {
  const valor = sesion?.[campo];
  if (typeof valor !== "string" || valor.length === 0 || valor !== valor.trim()) {
    throw new TypeError(`sesion.${campo} no valido`);
  }
  return valor;
}

/**
 * Traduce la sesion sintetica existente de Bolsa a la proyeccion comun. Solo
 * acepta los actores ya declarados por datos-presentacion.js y comprueba
 * sus datos visibles para no atribuir una sesion mal formada a otro empleado.
 */
export function crearContextoActorPresentacionDesdeSesion(sesion) {
  if (sesion === null || typeof sesion !== "object" || Array.isArray(sesion)) {
    throw new TypeError("sesion de presentacion no valida");
  }
  const actorRef = exigirCadenaSesion(sesion, "actor_ref");
  const definicion = PERFILES_PRESENTACION[actorRef];
  if (!definicion) throw new TypeError("actor de presentacion no reconocido");

  const nombre = exigirCadenaSesion(sesion, "nombre");
  const iniciales = exigirCadenaSesion(sesion, "iniciales");
  const etiquetaPerfil = exigirCadenaSesion(sesion, "perfil");
  if (nombre !== definicion.nombre || iniciales !== definicion.iniciales
    || etiquetaPerfil !== definicion.etiqueta_perfil) {
    throw new TypeError("la sesion no coincide con la identidad sintetica declarada");
  }

  return validarYCongelarContextoActor({
    esquema: ESQUEMA_CONTEXTO_ACTOR_FRONTEND,
    revision: 1,
    demostracion: true,
    persona_ref: definicion.persona_ref,
    cuenta_ref: definicion.cuenta_ref,
    perfil_ref: definicion.perfil_ref,
    actor: {
      actor_ref: actorRef,
      nombre_visible: nombre,
      iniciales,
    },
    rol: {
      clave: definicion.rol,
      etiqueta: etiquetaPerfil,
    },
    ambito: {
      clase: "personal_interno",
      organizacion_ref: "org_demo_diputacion_granada_000001",
      unidad_ref: "uni_demo_recursos_humanos_000001",
      modulos: definicion.modulos,
    },
    autenticacion: {
      sesion_ref: definicion.sesion_ref,
      metodo: "demo",
      garantia: "bajo",
    },
    resuelto_en: "2026-07-19T00:00:00.000Z",
  });
}
