const ESQUEMA_PANEL = "vec.bolsa.area-personal.v1";
const ESQUEMA_RECIBO = "vec.bolsa.area-personal.recibo.v1";
const ESQUEMA_RECIBO_PRESENTACION = "vec.bolsa.area-personal.recibo-demo.v1";
const PATRON_DNI_NIE = /\b(?:[XYZ]\d{7}[A-Z]|\d{8}[A-Z])\b/i;
const PATRON_CORREO = /\b[A-Z0-9._%+-]+@([A-Z0-9.-]+)\b/gi;
const MAXIMO_ELEMENTOS = 200;

function exigirObjeto(valor, nombre) {
  if (valor === null || typeof valor !== "object" || Array.isArray(valor)) {
    throw new TypeError(`${nombre} debe ser un objeto.`);
  }
  return valor;
}

function exigirCadena(valor, nombre, maximo = 500) {
  if (typeof valor !== "string" || valor.trim() === "" || valor.length > maximo) {
    throw new TypeError(`${nombre} debe ser una cadena no vacía de hasta ${maximo} caracteres.`);
  }
  return valor;
}

function exigirBooleano(valor, nombre) {
  if (typeof valor !== "boolean") throw new TypeError(`${nombre} debe ser booleano.`);
  return valor;
}

function exigirNumero(valor, nombre, { minimo = 0, maximo = 1_000_000 } = {}) {
  if (!Number.isFinite(valor) || valor < minimo || valor > maximo) {
    throw new TypeError(`${nombre} debe ser un número entre ${minimo} y ${maximo}.`);
  }
  return valor;
}

function exigirLista(valor, nombre) {
  if (!Array.isArray(valor) || valor.length > MAXIMO_ELEMENTOS) {
    throw new TypeError(`${nombre} debe ser una lista de hasta ${MAXIMO_ELEMENTOS} elementos.`);
  }
  return valor;
}

function exigirReferencia(valor, nombre, { demostracion = false } = {}) {
  const referencia = exigirCadena(valor, nombre, 100);
  if (!/^[A-Z0-9][A-Z0-9._/-]*$/i.test(referencia)) throw new TypeError(`${nombre} no es una referencia opaca válida.`);
  if (demostracion && !referencia.startsWith("DEMO-")) throw new TypeError(`${nombre} debe comenzar por DEMO-.`);
  return referencia;
}

function exigirInstante(valor, nombre) {
  const instante = exigirCadena(valor, nombre, 40);
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{3})?Z$/.test(instante)) {
    throw new TypeError(`${nombre} debe ser un instante UTC ISO 8601.`);
  }
  return instante;
}

function recorrerCadenas(valor, visita) {
  if (typeof valor === "string") {
    visita(valor);
    return;
  }
  if (Array.isArray(valor)) {
    valor.forEach((item) => recorrerCadenas(item, visita));
    return;
  }
  if (valor && typeof valor === "object") {
    Object.values(valor).forEach((item) => recorrerCadenas(item, visita));
  }
}

function validarPrivacidadPresentacion(datos) {
  recorrerCadenas(datos, (cadena) => {
    if (PATRON_DNI_NIE.test(cadena)) throw new TypeError("La presentación no admite DNI o NIE formalmente válidos.");
    PATRON_CORREO.lastIndex = 0;
    for (const coincidencia of cadena.matchAll(PATRON_CORREO)) {
      if (!coincidencia[1].toLowerCase().endsWith(".test")) {
        throw new TypeError("Los correos de presentación deben usar un dominio .test.");
      }
    }
  });
}

function validarListaObjetos(datos, campo, camposCadena = []) {
  exigirLista(datos[campo], campo).forEach((elemento, indice) => {
    const objeto = exigirObjeto(elemento, `${campo}[${indice}]`);
    camposCadena.forEach((nombre) => exigirCadena(objeto[nombre], `${campo}[${indice}].${nombre}`));
  });
}

function congelarProfundo(valor) {
  if (valor && typeof valor === "object" && !Object.isFrozen(valor)) {
    Object.values(valor).forEach(congelarProfundo);
    Object.freeze(valor);
  }
  return valor;
}

export function validarDatosAreaPersonal(entrada, { presentacionEsperada = false } = {}) {
  const datos = structuredClone(exigirObjeto(entrada, "datos"));
  const meta = exigirObjeto(datos.meta, "meta");
  if (exigirCadena(meta.esquema, "meta.esquema", 80) !== ESQUEMA_PANEL) {
    throw new TypeError("El esquema del área personal no es compatible.");
  }
  const presentacion = exigirBooleano(meta.presentacion, "meta.presentacion");
  if (presentacion !== presentacionEsperada) throw new TypeError("El origen no coincide con el modo solicitado.");
  exigirCadena(meta.origen, "meta.origen", 120);
  exigirInstante(meta.generado_en, "meta.generado_en");

  const sesion = exigirObjeto(datos.sesion, "sesion");
  exigirCadena(sesion.nombre_visible, "sesion.nombre_visible", 100);
  exigirCadena(sesion.iniciales, "sesion.iniciales", 4);
  exigirCadena(sesion.metodo, "sesion.metodo", 100);
  exigirReferencia(sesion.persona_ref, "sesion.persona_ref", { demostracion: presentacion });

  const resumen = exigirObjeto(datos.resumen, "resumen");
  ["acciones_pendientes", "convocatorias_abiertas", "solicitudes_activas", "mensajes_no_leidos"]
    .forEach((campo) => exigirNumero(resumen[campo], `resumen.${campo}`, { maximo: 10_000 }));
  exigirNumero(resumen.puntuacion_provisional, "resumen.puntuacion_provisional", { maximo: 10_000 });

  const perfil = exigirObjeto(datos.perfil, "perfil");
  exigirReferencia(perfil.referencia, "perfil.referencia", { demostracion: presentacion });
  ["nombre_visible", "identificador_visible", "correo", "telefono", "domicilio", "estado_verificacion"]
    .forEach((campo) => exigirCadena(perfil[campo], `perfil.${campo}`));

  validarListaObjetos(datos, "plazos", ["id", "dia", "mes", "titulo", "detalle", "estado"]);
  validarListaObjetos(datos, "convocatorias", ["id", "referencia", "titulo", "categoria", "estado", "plazo", "descripcion"]);
  validarListaObjetos(datos, "meritos", ["id", "tipo", "titulo", "detalle", "estado", "documento_ref"]);
  validarListaObjetos(datos, "solicitudes", ["id", "convocatoria_id", "referencia", "titulo", "estado", "actualizado"]);
  validarListaObjetos(datos, "baremo", ["id", "nombre", "detalle", "estado"]);
  validarListaObjetos(datos, "llamamientos", ["id", "bolsa", "puesto", "plazo", "estado"]);
  validarListaObjetos(datos, "subsanaciones", ["id", "solicitud_ref", "motivo", "plazo", "estado"]);
  validarListaObjetos(datos, "alegaciones", ["id", "solicitud_ref", "asunto", "estado", "fecha"]);
  validarListaObjetos(datos, "mensajes", ["id", "asunto", "resumen", "fecha", "estado"]);
  validarListaObjetos(datos, "certificados", ["id", "tipo", "descripcion", "estado"]);
  validarListaObjetos(datos, "documentos", ["id", "nombre", "tipo", "fecha", "estado"]);
  validarListaObjetos(datos, "actividad", ["id", "titulo", "detalle", "fecha", "actor"]);
  exigirObjeto(datos.disponibilidad, "disponibilidad");
  exigirBooleano(datos.disponibilidad.disponible, "disponibilidad.disponible");
  exigirCadena(datos.disponibilidad.estado, "disponibilidad.estado");
  exigirLista(datos.ayuda, "ayuda").forEach((item, indice) => {
    exigirCadena(exigirObjeto(item, `ayuda[${indice}]`).pregunta, `ayuda[${indice}].pregunta`);
    exigirCadena(item.respuesta, `ayuda[${indice}].respuesta`, 2_000);
  });
  const capacidades = exigirObjeto(datos.capacidades, "capacidades");
  Object.entries(capacidades).forEach(([nombre, valor]) => exigirBooleano(valor, `capacidades.${nombre}`));
  if (datos.preferencias_notificacion !== undefined) {
    const preferencias = exigirObjeto(datos.preferencias_notificacion, "preferencias_notificacion");
    Object.entries(preferencias).forEach(([nombre, valor]) => exigirBooleano(valor, `preferencias_notificacion.${nombre}`));
  }
  if (datos.resultado_autobaremo !== undefined) {
    const resultado = exigirObjeto(datos.resultado_autobaremo, "resultado_autobaremo");
    exigirReferencia(resultado.convocatoria_id, "resultado_autobaremo.convocatoria_id", { demostracion: presentacion });
    exigirLista(resultado.meritos_ids, "resultado_autobaremo.meritos_ids")
      .forEach((id, indice) => exigirReferencia(id, `resultado_autobaremo.meritos_ids[${indice}]`, { demostracion: presentacion }));
    exigirNumero(resultado.puntos, "resultado_autobaremo.puntos", { maximo: 10_000 });
    exigirInstante(resultado.calculado_en, "resultado_autobaremo.calculado_en");
  }

  if (presentacion) validarPrivacidadPresentacion(datos);
  return congelarProfundo(datos);
}

export function validarRecibo(entrada, { presentacionEsperada = false } = {}) {
  const recibo = structuredClone(exigirObjeto(entrada, "recibo"));
  const esquemaEsperado = presentacionEsperada ? ESQUEMA_RECIBO_PRESENTACION : ESQUEMA_RECIBO;
  if (exigirCadena(recibo.esquema, "recibo.esquema", 80) !== esquemaEsperado) {
    throw new TypeError("El esquema del recibo no es compatible.");
  }
  if (exigirBooleano(recibo.presentacion, "recibo.presentacion") !== presentacionEsperada) {
    throw new TypeError("El recibo no corresponde al modo activo.");
  }
  exigirReferencia(recibo.referencia, "recibo.referencia", { demostracion: presentacionEsperada });
  exigirCadena(recibo.accion, "recibo.accion", 80);
  exigirReferencia(recibo.objetivo, "recibo.objetivo", { demostracion: presentacionEsperada });
  exigirCadena(recibo.resultado, "recibo.resultado", 80);
  exigirCadena(recibo.actor, "recibo.actor", 100);
  exigirInstante(recibo.fecha, "recibo.fecha");
  exigirCadena(recibo.advertencia, "recibo.advertencia", 300);
  return congelarProfundo(recibo);
}

export function esModoPresentacion(parametros = new URLSearchParams()) {
  const selectores = parametros.getAll("presentacion");
  if (selectores.length > 1) throw new TypeError("El selector de presentación es ambiguo.");
  return selectores.length === 1 && selectores[0] === "rrhh";
}

export const CONTRATO_AREA_PERSONAL = Object.freeze({
  esquemaPanel: ESQUEMA_PANEL,
  esquemaRecibo: ESQUEMA_RECIBO,
  esquemaReciboPresentacion: ESQUEMA_RECIBO_PRESENTACION,
});
