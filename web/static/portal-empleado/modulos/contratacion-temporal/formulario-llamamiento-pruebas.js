import assert from "node:assert/strict";
import { createHash, webcrypto } from "node:crypto";
import { File } from "node:buffer";
import { readFile } from "node:fs/promises";
import { montarFormularioLlamamiento } from "./formulario-llamamiento.js";
import { montarModuloContratacionTemporal } from "./vista-expedientes.js";

export const CLAVE = "123e4567-e89b-42d3-a456-426614174000";
export const EXPEDIENTE = "expediente:ct:sintetico:001";
export const PUBLICACIONES_PROPUESTA = await readFile(new URL("./formalizacion-desarrollo.json", import.meta.url), "utf8");
export const seleccion = () => ({
  expediente_ref: EXPEDIENTE, version_esperada: "6", clave_idempotencia: CLAVE,
});
export const recibo = {
  esquema: "vec.contratacion-temporal.recibo-seleccion-llamamiento.v1",
  estado: "confirmado", recibo_ref: "recibo:sintetico:001", confirmada_en: "2026-09-05T08:00:00Z",
  organizacion_ref: "organizacion:sintetica:001", llamamiento_ref: "llamamiento:sintetico:001",
  version_llamamiento: 1,
};
export function raizPrueba() {
  const eventos = new Map(), borradores = {}, foco = [];
  const raiz = {
    innerHTML: "", eventos, borradores, foco,
    addEventListener: (tipo, fn) => eventos.set(tipo, fn),
    removeEventListener: (tipo) => eventos.delete(tipo),
    contains: () => true,
    replaceChildren() { this.innerHTML = ""; },
    querySelector(selector) {
      const tipo = selector.match(/^\[data-ct-llamamiento-form="(seleccion|comunicacion|comunicacion_siguiente|respuesta|respuesta_siguiente|resolucion|resolucion_siguiente|siguiente|propuesta)"\]$/u)?.[1];
      if (tipo) return borradores[tipo] ?? null;
      if (selector === "[data-ct-llamamiento-comunicacion]") return { open: false };
      return { focus: () => foco.push(selector), scrollIntoView() {} };
    },
    preparar(tipo, valores) {
      const controles = Object.fromEntries(Object.entries(valores).map(
        ([nombre, value]) => [nombre, typeof value === "boolean"
          ? { value: "on", checked: value } : { value }],
      ));
      borradores[tipo] = { dataset: { ctLlamamientoForm: tipo },
        elements: { namedItem: (nombre) => controles[nombre] },
        closest: () => borradores[tipo],
      };
      return borradores[tipo];
    },
    enviar(tipo, valores) {
      const form = valores ? this.preparar(tipo, valores) : borradores[tipo];
      return eventos.get("submit")({ target: form, preventDefault() {} });
    },
    archivo: (archivo, operacion = "respuesta") => eventos.get("change")({ target: {
      files: archivo ? [archivo] : [], closest(selector) {
        return selector === "[data-ct-llamamiento-form]" ? { dataset: { ctLlamamientoForm: operacion } } : this;
      },
    } }),
  };
  return raiz;
}
export function montar(raiz, cliente = {}, extras = {}) {
  return montarFormularioLlamamiento({
    raiz, cliente: { seleccionarLlamamiento: async () => recibo,
      registrarComunicacionLlamamiento: async () => {},
      registrarRespuestaRecibida: async () => {}, resolverLlamamiento: async () => {},
      continuarLlamamiento: async () => {}, ...cliente },
    confirmarOperacion: () => true, criptografia: webcrypto, ...extras,
  });
}
export function estadoSeleccionado(expedienteRef = EXPEDIENTE) {
  return {
    vista: "expediente", carga: "listo", expediente_ref: expedienteRef,
    cuadro: { demostracion: false, expedientes: [{ expediente_ref: expedienteRef, version: 6, fase_clave: "llamamiento" }] },
    expediente: { demostracion: false, expediente_ref: expedienteRef, version: 6,
      numero_visible: "CT-SINTETICO-001", cabecera: [], fases: [], tareas: [],
    },
    tipo_mensaje: "informacion", mensaje_clave: "estado_expediente_listo", ocupado: false,
    actualizacion_pendiente: false, resultado_indeterminado: false,
  };
}
export async function montarExpedienteSeleccionado(inicial, alta = null) {
  let estado = inicial, html = "", formulario, peticiones = 0;
  const eventos = new Map();
  const raiz = {
    addEventListener: (tipo, fn) => eventos.set(tipo, fn),
    removeEventListener: (tipo) => eventos.delete(tipo),
    contains: () => true,
    get innerHTML() { return html; },
    set innerHTML(valor) { html = valor; formulario = raizPrueba(); },
    querySelector: (selector) => selector === "[data-ct-exp-llamamiento]"
      && html.includes("data-ct-exp-llamamiento") ? formulario : null,
  };
  const modulo = await montarModuloContratacionTemporal({
    raiz, alta,
    presentador: {
      obtenerEstado: () => estado,
      cargar: async () => estado,
      async seleccionarExpediente(referencia) { estado = estadoSeleccionado(referencia); },
    },
    llamamiento: { cliente: {
      seleccionarLlamamiento: async () => { peticiones += 1; return recibo; },
      registrarComunicacionLlamamiento: async () => { peticiones += 1; },
      registrarRespuestaRecibida: async () => { peticiones += 1; },
      resolverLlamamiento: async () => { peticiones += 1; },
      continuarLlamamiento: async () => { peticiones += 1; },
    } },
  });
  return {
    formulario: () => formulario, peticiones: () => peticiones, desmontar: modulo.desmontar,
    abrir: (referencia) => eventos.get("click")({
      target: { closest: (selector) => selector === "[data-ct-exp-abrir]"
        ? { dataset: { ctExpAbrir: referencia } } : null },
      preventDefault() {},
    }),
  };
}
export const CORREO = "Subject: Respuesta sintetica\r\n\r\nAceptacion declarada por RRHH.\r\n";
export const HUELLA = createHash("sha256").update(CORREO).digest("hex");
export const archivoCorreo = (opcion = "aceptacion") => new File([
  CORREO.replace("Aceptacion", opcion === "renuncia" ? "Renuncia" : "Aceptacion"),
], "respuesta-sintetica.eml");
export const comunicacionRegistrada = {
  esquema: "vec.contratacion-temporal.registro-comunicacion-llamamiento.v1",
  estado_local: "registrada_localmente", comunicacion_ref: "comunicacion:sintetica:001",
  recibo_ref: "recibo:comunicacion:001", auditoria_ref: "auditoria:sintetica:001",
  version_resultante: 2, registrada_en: "2026-09-05T08:05:00Z",
  intencion_envio_ref: "intencion:sintetica:001",
};
export const declaracion = () => ({
  clave_idempotencia: "123e4567-e89b-42d3-a456-426614174002",
  respuesta: "aceptacion", correo_ref: "correo:sintetico:001", recibida_en: "2026-09-05T08:30",
});
export const justificante = (solicitud) => ({
  ...solicitud, esquema: "vec.contratacion-temporal.respuesta-recibida-llamamiento.v1",
  justificante_ref: "justificante:sintetico:001", recibo_ref: "recibo:respuesta:001",
  auditoria_ref: "auditoria:respuesta:001", registrada_en: "2026-09-05T09:00:00.123456Z",
  estado: "registrada_por_rrhh",
});
export async function abrirRespuesta(raiz, cliente = {}, extras = {}) {
  const cerrar = montar(raiz, {
    registrarComunicacionLlamamiento: async () => comunicacionRegistrada,
    registrarRespuestaRecibida: async (s) => justificante(s), ...cliente,
  }, extras);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="respuesta"/u);
  await raiz.enviar("seleccion", { ...seleccion(), version_esperada: extras.contexto?.version_esperada ?? 6 });
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="respuesta"/u);
  await raiz.enviar("comunicacion", { clave_idempotencia: CLAVE });
  raiz.preparar("respuesta", declaracion());
  return cerrar;
}
export const CLAVE_RESOLUCION = "123e4567-e89b-42d3-a456-426614174003";
export const CLAVE_RESOLUCION_SIGUIENTE = "123e4567-e89b-42d3-a456-426614174008";
export const revisionManual = { revision_respuesta_rrhh: true, revision_plazo_rrhh: true };
export const resolucionConfirmada = {
  esquema: "vec.contratacion-temporal.resolucion-comunicacion-llamamiento.v1",
  respuesta: "aceptacion", estado_plazo: "vigente", estado_local: "confirmado",
  resolucion_ref: "resolucion:sintetica:001", recibo_local_ref: "recibo:resolucion:001",
  auditoria_ref: "auditoria:resolucion:001", version_resultante: 3,
  resuelta_en: "2026-09-05T09:05:00.123450Z",
};
export const reciboResolucion = (opcion, estado_local = "confirmado") => ({
  ...resolucionConfirmada, respuesta: opcion, estado_local,
  ...(opcion === "renuncia" ? { intencion_siguiente: { referencia: "intencion:siguiente:001",
    estado_local: "pendiente", actualizada_en: "2026-09-05T09:05:00.12345Z" } } : {}),
});
export const reciboResolucionSucesor = (opcion, estado_local = "confirmado") => ({
  ...reciboResolucion(opcion, estado_local), resolucion_ref: "resolucion:sucesor:002",
  recibo_local_ref: "recibo:resolucion:sucesor:002", auditoria_ref: "auditoria:resolucion:sucesor:002",
  resuelta_en: "2026-09-05T09:10:00.123456Z",
  ...(opcion === "renuncia" ? { intencion_siguiente: { referencia: "intencion:sucesor:003",
    estado_local: "pendiente", actualizada_en: "2026-09-05T09:10:00.123456Z" } } : {}),
});
export const solicitudSiguiente = {
  clave_idempotencia: "123e4567-e89b-42d3-a456-426614174004",
  organizacion_ref: recibo.organizacion_ref, expediente_ref: EXPEDIENTE,
  resolucion_ref: resolucionConfirmada.resolucion_ref, intencion_ref: "intencion:siguiente:001",
};
export const continuacionConfirmada = {
  esquema: "vec.contratacion-temporal.continuacion-llamamiento.v1",
  organizacion_ref: recibo.organizacion_ref, expediente_ref: EXPEDIENTE,
  resolucion_ref: solicitudSiguiente.resolucion_ref, intencion_ref: solicitudSiguiente.intencion_ref,
  llamamiento_anterior_ref: recibo.llamamiento_ref, llamamiento_ref: "llamamiento:siguiente:002",
  version_llamamiento: 1, recibo_bolsa_ref: "recibo:bolsa:002", recibo_ref: "recibo:ct:002",
  auditoria_ref: "auditoria:siguiente:002", confirmada_en: "2026-09-05T09:06:00.123456Z",
  estado_intencion: "despachada", estado_local: "confirmado",
};
export const solicitudAvisoSiguiente = {
  clave_idempotencia: "123e4567-e89b-42d3-a456-426614174006",
  organizacion_ref: continuacionConfirmada.organizacion_ref, expediente_ref: continuacionConfirmada.expediente_ref,
  llamamiento_ref: continuacionConfirmada.llamamiento_ref, version_esperada: 1,
  prueba_entrega_ref: continuacionConfirmada.recibo_ref, tipo_antecedente: "continuacion_confirmada",
};
export const avisoSiguienteRegistrado = { ...comunicacionRegistrada,
  comunicacion_ref: "comunicacion:sucesor:002", recibo_ref: "recibo:aviso:sucesor:002",
  auditoria_ref: "auditoria:sucesor:002", registrada_en: "2026-09-05T09:07:00.123456Z",
  intencion_envio_ref: "intencion:aviso:sucesor:002" };
export const declaracionSiguiente = () => ({ ...declaracion(),
  clave_idempotencia: "123e4567-e89b-42d3-a456-426614174007",
  correo_ref: "correo:sintetico:sucesor:002", recibida_en: "2026-09-05T09:08" });
export const justificanteSiguiente = (s) => ({ ...justificante(s),
  justificante_ref: "justificante:sucesor:002", recibo_ref: "recibo:respuesta:sucesor:002",
  auditoria_ref: "auditoria:respuesta:sucesor:002", registrada_en: "2026-09-05T09:09:00.123456Z" });
export async function abrirResolucion(raiz, cliente = {}, extras = {}, opcion = "aceptacion") {
  const cerrar = await abrirRespuesta(raiz, cliente, extras);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="resolucion"/u);
  await raiz.archivo(archivoCorreo(opcion));
  await raiz.enviar("respuesta", { ...declaracion(), respuesta: opcion });
  raiz.preparar("resolucion", { clave_idempotencia: CLAVE_RESOLUCION,
    revision_respuesta_rrhh: false, revision_plazo_rrhh: false });
  return cerrar;
}
export async function abrirSiguiente(raiz, cliente = {}, extras = {}) {
  const cerrar = await abrirResolucion(raiz, {
    resolverLlamamiento: async () => reciboResolucion("renuncia"), ...cliente,
  }, extras, "renuncia");
  await raiz.enviar("resolucion", { clave_idempotencia: CLAVE_RESOLUCION, ...revisionManual });
  return cerrar;
}
export async function abrirRespuestaSiguiente(raiz, registrar = justificanteSiguiente, extras = {}, cliente = {}) {
  const cerrar = await abrirSiguiente(raiz, {
    ...cliente,
    continuarLlamamiento: async () => continuacionConfirmada,
    registrarComunicacionLlamamiento: async (s) => s.tipo_antecedente ? avisoSiguienteRegistrado : comunicacionRegistrada,
    registrarRespuestaRecibida: (s, opciones) => s.comunicacion_ref === avisoSiguienteRegistrado.comunicacion_ref
      ? registrar(s, opciones) : justificante(s),
  }, extras);
  await raiz.enviar("siguiente", solicitudSiguiente);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="respuesta_siguiente"/u);
  await raiz.enviar("comunicacion_siguiente", solicitudAvisoSiguiente);
  raiz.preparar("respuesta_siguiente", declaracionSiguiente());
  return cerrar;
}
export async function abrirResolucionSucesor(raiz, cliente = {}, extras = {}, opcion = "aceptacion") {
  const cerrar = await abrirRespuestaSiguiente(raiz, justificanteSiguiente, { ...extras,
    confirmarOperacion: (d) => d.datos.prueba_respuesta_ref === justificante({}).justificante_ref
      ? true : (extras.confirmarOperacion?.(d) ?? true),
  }, { ...cliente, resolverLlamamiento: (s, opciones) => s.comunicacion_ref === avisoSiguienteRegistrado.comunicacion_ref
    ? cliente.resolverLlamamiento(s, opciones) : reciboResolucion("renuncia") });
  await raiz.archivo(archivoCorreo(opcion), "respuesta_siguiente");
  await raiz.enviar("respuesta_siguiente", { ...declaracionSiguiente(), respuesta: opcion });
  raiz.preparar("resolucion_siguiente", { clave_idempotencia: CLAVE_RESOLUCION_SIGUIENTE,
    revision_respuesta_rrhh: false, revision_plazo_rrhh: false });
  return cerrar;
}
