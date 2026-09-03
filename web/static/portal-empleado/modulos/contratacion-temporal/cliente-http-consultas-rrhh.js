const RUTA_CUADRO = "/api/vec/contratacion-temporal/cuadro/consultas";
const RUTA_DETALLE = "/api/vec/contratacion-temporal/expedientes/consultas";
const ESQUEMA_CUADRO = "vec.contratacion-temporal.cuadro-rrhh.v1";
const ESQUEMA_DETALLE = "vec.contratacion-temporal.detalle-rrhh.v1";
const MAXIMO_SOLICITUD = 4 * 1024;
const MAXIMO_RESPUESTA = 256 * 1024;
const MAXIMO_EXPEDIENTES = 100;
const MAXIMO_HITOS = 2_000;
const PATRON_REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const PATRON_CLAVE = /^[a-z][a-z0-9._-]{1,79}$/u;
const PATRON_NUMERO = /^[0-9]{4}\/[A-Za-z0-9._-]{1,40}$/u;
const PATRON_TEXTO_CUADRO = /^[0-9A-Za-zÁÉÍÓÚÜÑáéíóúüñ/._ -]{0,80}$/u;
const PATRON_CURSOR = /^[A-Za-z0-9_-]{42}[AEIMQUYcgkosw048]$/u;
const PATRON_INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/u;
const ESTADOS_OPERATIVOS = new Set([
  "pendiente", "en_curso", "espera_externa", "completado", "incidencia", "cancelado",
]);

export const RUTAS_CONSULTA_RRHH = Object.freeze({
  cuadroRRHH: RUTA_CUADRO,
  detalleRRHH: RUTA_DETALLE,
});

function esRegistro(valor) {
  return valor !== null && typeof valor === "object" && !Array.isArray(valor)
    && Object.getPrototypeOf(valor) === Object.prototype;
}

function camposCerrados(valor, obligatorios, opcionales = []) {
  if (!esRegistro(valor)) return false;
  const permitidos = new Set([...obligatorios, ...opcionales]);
  const recibidos = Object.keys(valor);
  return obligatorios.every((campo) => Object.hasOwn(valor, campo))
    && recibidos.every((campo) => permitidos.has(campo));
}

function cadena(valor, { vacia = false, maximo = 4_000, patron } = {}) {
  return typeof valor === "string" && (vacia || valor.length > 0)
    && valor.length <= maximo && valor === valor.trim()
    && valor.normalize("NFC") === valor
    && !/[\u0000-\u001f\u007f-\u009f]/u.test(valor)
    && (patron === undefined || patron.test(valor));
}

function referencia(valor, vacia = false) {
  return cadena(valor, { vacia, maximo: 160 })
    && (valor === "" ? vacia : PATRON_REFERENCIA.test(valor));
}

function clave(valor, vacia = false) {
  return cadena(valor, { vacia, maximo: 80 })
    && (valor === "" ? vacia : PATRON_CLAVE.test(valor));
}

function estadoOperativo(valor, vacio = false) {
  return valor === "" ? vacio : ESTADOS_OPERATIVOS.has(valor);
}

function cursor(valor, vacio = false) {
  return valor === "" ? vacio : cadena(valor, { maximo: 43, patron: PATRON_CURSOR });
}

function instante(valor) {
  return cadena(valor, { maximo: 40, patron: PATRON_INSTANTE })
    && Number.isFinite(Date.parse(valor));
}

function entero(valor, minimo = 0) {
  return Number.isSafeInteger(valor) && valor >= minimo;
}

function validarSolicitudCuadro(entrada) {
  if (!camposCerrados(entrada, ["filtros", "paginacion"])
    || !camposCerrados(entrada.filtros, ["texto", "estado_clave", "fase_clave"])
    || !camposCerrados(entrada.paginacion, ["limite", "cursor"])
    || !cadena(entrada.filtros.texto, {
      vacia: true, maximo: 80, patron: PATRON_TEXTO_CUADRO,
    })
    || !estadoOperativo(entrada.filtros.estado_clave, true)
    || !clave(entrada.filtros.fase_clave, true)
    || !entero(entrada.paginacion.limite, 1)
    || entrada.paginacion.limite > MAXIMO_EXPEDIENTES
    || !cursor(entrada.paginacion.cursor, true)) {
    throw new TypeError("solicitud de cuadro RRHH no válida");
  }
  return structuredClone(entrada);
}

function validarSolicitudDetalle(entrada) {
  if (!camposCerrados(entrada, ["expediente_ref", "version_observada"])
    || !referencia(entrada.expediente_ref)
    || !entero(entrada.version_observada, 0)) {
    throw new TypeError("solicitud de detalle RRHH no válida");
  }
  return structuredClone(entrada);
}

function validarResumen(entrada) {
  const obligatorios = [
    "expediente_ref", "numero_visible", "version", "flujo_ref",
    "flujo_version", "flujo_huella_sha256", "fase_clave", "estado_clave",
    "centro_ref", "categoria_ref", "creado_en", "actualizado_en",
  ];
  if (!camposCerrados(entrada, obligatorios, ["modalidad_clave", "unidad_ref"])
    || !referencia(entrada.expediente_ref) || !referencia(entrada.flujo_ref)
    || !referencia(entrada.centro_ref) || !referencia(entrada.categoria_ref)
    || !cadena(entrada.numero_visible, { maximo: 45, patron: PATRON_NUMERO })
    || !entero(entrada.version, 1) || !entero(entrada.flujo_version, 1)
    || typeof entrada.flujo_huella_sha256 !== "string"
    || !/^[a-f0-9]{64}$/u.test(entrada.flujo_huella_sha256)
    || !clave(entrada.fase_clave) || !estadoOperativo(entrada.estado_clave)
    || !instante(entrada.creado_en) || !instante(entrada.actualizado_en)
    || (Object.hasOwn(entrada, "modalidad_clave") && !clave(entrada.modalidad_clave))
    || (Object.hasOwn(entrada, "unidad_ref") && !referencia(entrada.unidad_ref))) {
    throw new TypeError("resumen RRHH no válido");
  }
  return structuredClone(entrada);
}

function validarPagina(entrada) {
  if (!camposCerrados(
    entrada,
    ["esquema", "generada_en", "expedientes", "hay_mas"],
    ["cursor_siguiente"],
  ) || entrada.esquema !== ESQUEMA_CUADRO || !instante(entrada.generada_en)
    || !Array.isArray(entrada.expedientes)
    || entrada.expedientes.length > MAXIMO_EXPEDIENTES
    || typeof entrada.hay_mas !== "boolean"
    || (Object.hasOwn(entrada, "cursor_siguiente")
      && !cursor(entrada.cursor_siguiente))) {
    throw new TypeError("página de cuadro RRHH no válida");
  }
  const expedientes = entrada.expedientes.map(validarResumen);
  if (new Set(expedientes.map(({ expediente_ref: ref }) => ref)).size
    !== expedientes.length || (entrada.hay_mas !== Object.hasOwn(
    entrada,
    "cursor_siguiente",
  ))) {
    throw new TypeError("página de cuadro RRHH incoherente");
  }
  return Object.freeze({ ...structuredClone(entrada), expedientes });
}

function validarSolicitudProyectada(entrada) {
  if (!camposCerrados(
    entrada,
    ["grupo_subgrupo", "motivo_clave", "periodo_inicio", "periodo_fin"],
  ) || !cadena(entrada.grupo_subgrupo, { maximo: 80 })
    || !clave(entrada.motivo_clave) || !instante(entrada.periodo_inicio)
    || !instante(entrada.periodo_fin)) {
    throw new TypeError("solicitud proyectada RRHH no válida");
  }
  return structuredClone(entrada);
}

function validarAnalisis(entrada) {
  const obligatorios = [
    "modalidad_clave", "categoria_ref", "causa_clave", "periodo_inicio",
    "periodo_fin", "porcentaje_jornada", "resultado_rc",
  ];
  if (!camposCerrados(entrada, obligatorios, ["coste_previsto", "fuente_coste_ref"])
    || !clave(entrada.modalidad_clave) || !referencia(entrada.categoria_ref)
    || !clave(entrada.causa_clave) || !instante(entrada.periodo_inicio)
    || !instante(entrada.periodo_fin)
    || !entero(entrada.porcentaje_jornada, 1) || entrada.porcentaje_jornada > 10_000
    || !clave(entrada.resultado_rc)
    || (Object.hasOwn(entrada, "fuente_coste_ref")
      && !referencia(entrada.fuente_coste_ref))
    || (Object.hasOwn(entrada, "coste_previsto")
      && (!camposCerrados(entrada.coste_previsto, ["centimos", "moneda"])
        || !entero(entrada.coste_previsto.centimos)
        || !/^[A-Z]{3}$/u.test(entrada.coste_previsto.moneda)))) {
    throw new TypeError("análisis RRHH no válido");
  }
  return structuredClone(entrada);
}

function validarCobertura(entrada) {
  if (!camposCerrados(
    entrada,
    ["via_clave", "decision_gobernada", "comprobaciones"],
    ["procedimiento_ref", "bolsa_ref"],
  ) || !clave(entrada.via_clave) || typeof entrada.decision_gobernada !== "boolean"
    || !Array.isArray(entrada.comprobaciones) || entrada.comprobaciones.length > 128
    || (Object.hasOwn(entrada, "procedimiento_ref")
      && !referencia(entrada.procedimiento_ref))
    || (Object.hasOwn(entrada, "bolsa_ref") && !referencia(entrada.bolsa_ref))) {
    throw new TypeError("cobertura RRHH no válida");
  }
  entrada.comprobaciones.forEach((item) => {
    if (!camposCerrados(item, ["clave", "resultado"])
      || !clave(item.clave) || !clave(item.resultado)) {
      throw new TypeError("comprobación RRHH no válida");
    }
  });
  return structuredClone(entrada);
}

function validarAsignacion(entrada) {
  if (!camposCerrados(entrada, ["unidad_ref", "asignada_en"], ["motivo_clave"])
    || !referencia(entrada.unidad_ref) || !instante(entrada.asignada_en)
    || (Object.hasOwn(entrada, "motivo_clave") && !clave(entrada.motivo_clave))) {
    throw new TypeError("asignación RRHH no válida");
  }
  return structuredClone(entrada);
}

function validarHito(entrada) {
  const obligatorios = [
    "secuencia", "version_expediente", "accion_clave", "realizada_en",
    "fase_destino", "estado_origen", "estado_destino",
  ];
  if (!camposCerrados(entrada, obligatorios, ["fase_origen"])
    || !entero(entrada.secuencia, 1) || !entero(entrada.version_expediente, 1)
    || !clave(entrada.accion_clave) || !instante(entrada.realizada_en)
    || !clave(entrada.fase_destino) || !estadoOperativo(entrada.estado_origen)
    || !estadoOperativo(entrada.estado_destino)
    || (Object.hasOwn(entrada, "fase_origen") && !clave(entrada.fase_origen))) {
    throw new TypeError("hito RRHH no válido");
  }
  return structuredClone(entrada);
}

function validarDetalle(entrada) {
  if (!camposCerrados(
    entrada,
    ["esquema", "resumen", "solicitud", "hitos"],
    ["analisis", "cobertura", "asignacion"],
  ) || entrada.esquema !== ESQUEMA_DETALLE || !Array.isArray(entrada.hitos)
    || entrada.hitos.length > MAXIMO_HITOS) {
    throw new TypeError("detalle RRHH no válido");
  }
  const salida = {
    ...structuredClone(entrada),
    resumen: validarResumen(entrada.resumen),
    solicitud: validarSolicitudProyectada(entrada.solicitud),
    hitos: entrada.hitos.map(validarHito),
  };
  if (Object.hasOwn(entrada, "analisis")) salida.analisis = validarAnalisis(entrada.analisis);
  if (Object.hasOwn(entrada, "cobertura")) salida.cobertura = validarCobertura(entrada.cobertura);
  if (Object.hasOwn(entrada, "asignacion")) salida.asignacion = validarAsignacion(entrada.asignacion);
  return Object.freeze(salida);
}

export function crearConsultasRRHHClienteHTTP({ ejecutar, validarOpciones } = {}) {
  if (typeof ejecutar !== "function" || typeof validarOpciones !== "function") {
    throw new TypeError("dependencias de consultas RRHH no disponibles");
  }
  return Object.freeze({
    consultarCuadroRRHH(solicitud, opciones) {
      const { signal } = validarOpciones(opciones);
      return ejecutar({
        ruta: RUTA_CUADRO,
        entrada: validarSolicitudCuadro(solicitud),
        signal,
        estadoEsperado: 200,
        maximoSolicitud: MAXIMO_SOLICITUD,
        maximoRespuesta: MAXIMO_RESPUESTA,
        validarRespuesta: validarPagina,
        efecto: false,
        tipoContenido: "application/json",
      });
    },
    consultarDetalleRRHH(solicitud, opciones) {
      const { signal } = validarOpciones(opciones);
      return ejecutar({
        ruta: RUTA_DETALLE,
        entrada: validarSolicitudDetalle(solicitud),
        signal,
        estadoEsperado: 200,
        maximoSolicitud: MAXIMO_SOLICITUD,
        maximoRespuesta: MAXIMO_RESPUESTA,
        validarRespuesta: validarDetalle,
        efecto: false,
        tipoContenido: "application/json",
      });
    },
  });
}
