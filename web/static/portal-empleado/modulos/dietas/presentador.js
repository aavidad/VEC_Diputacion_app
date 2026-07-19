import {
  CAPACIDAD_CONSULTAR_AUDITORIA,
  CAPACIDAD_CONSULTAR_GASTO,
  CAPACIDAD_CONSULTAR_RUTA,
  CAPACIDAD_GESTIONAR_GASTO,
  CAPACIDAD_GESTIONAR_RUTA,
  ATRIBUCION_OSM_INTERNA,
  ESQUEMA_RESUMEN_ANUAL_DIETAS,
  ESQUEMA_RECIBO_DIETAS,
  PLANTILLA_TESELAS_OSM_INTERNA,
  copiarDietas,
  exigirContextoActorDietas,
  tieneCapacidadDietas,
  validarCapacidadesDietas,
  validarGeometriaRutaDietas,
  validarPanelDietas,
} from "./contrato.js";
import { crearPresentadorRutasDietas } from "./presentador-rutas.js";

const ESTADOS_PERMITIDOS = new Set([
  "todos", "borrador", "pendiente_jefatura", "aprobada", "enviada_rrhh", "enviada_nomina", "pagada",
]);
const CLAVES_I18N_ESTADO = Object.freeze({
  borrador: "estado_borrador",
  pendiente_jefatura: "estado_pendiente_jefatura",
  aprobada: "estado_aprobada",
  enviada_rrhh: "estado_enviada_rrhh",
  enviada_nomina: "estado_enviada_nomina",
  pagada: "estado_pagada",
});

function suma(registros, campo) {
  return registros.reduce((total, registro) => total + Number(registro[campo] || 0), 0);
}

function resumenDe(comisiones) {
  const pendientes = comisiones.filter((item) => item.etapa_actual < 5 && item.estado !== "pagada");
  const pagadas = comisiones.filter((item) => item.estado === "pagada");
  return Object.freeze({
    expedientes: comisiones.length,
    pendientes: pendientes.length,
    kilometros: Math.round(suma(comisiones, "kilometros") * 10) / 10,
    total_euros: Math.round(suma(comisiones, "total_euros") * 100) / 100,
    pagado_euros: Math.round(suma(pagadas, "total_euros") * 100) / 100,
  });
}

function redondear(valor, decimales = 2) {
  const factor = 10 ** decimales;
  return Math.round(Number(valor || 0) * factor) / factor;
}

function resumenAnualDe(comisiones, anioSolicitado = null) {
  const anios = [...new Set(comisiones.map((item) => Number(String(item.fecha).slice(0, 4))))]
    .filter((anio) => Number.isInteger(anio) && anio >= 2000 && anio <= 2200)
    .sort((a, b) => b - a);
  const anio = anioSolicitado === null ? anios[0] : Number(anioSolicitado);
  if (!Number.isInteger(anio) || !anios.includes(anio)) throw new Error("año de resumen de Dietas no disponible");
  const registros = comisiones.filter((item) => Number(String(item.fecha).slice(0, 4)) === anio);
  const kilometraje = suma(registros, "kilometraje_euros");
  const devengado = suma(registros, "total_euros");
  return Object.freeze({
    anio,
    meses_con_actividad: new Set(registros.map((item) => String(item.fecha).slice(0, 7))).size,
    expedientes: registros.length,
    pendientes: registros.filter((item) => item.estado !== "pagada").length,
    kilometros: redondear(suma(registros, "kilometros"), 1),
    kilometraje_euros: redondear(kilometraje),
    dietas_gastos_euros: redondear(devengado - kilometraje),
    devengado_euros: redondear(devengado),
    pagado_euros: redondear(suma(registros.filter((item) => item.estado === "pagada"), "total_euros")),
  });
}

function mesesDe(comisiones) {
  const meses = new Map();
  for (const comision of comisiones) {
    const clave = String(comision.fecha).slice(0, 7);
    const actual = meses.get(clave) || { mes: clave, expedientes: 0, kilometros: 0, total_euros: 0, pagado_euros: 0 };
    actual.expedientes += 1;
    actual.kilometros += Number(comision.kilometros || 0);
    actual.total_euros += Number(comision.total_euros || 0);
    if (comision.estado === "pagada") actual.pagado_euros += Number(comision.total_euros || 0);
    meses.set(clave, actual);
  }
  return [...meses.values()].sort((a, b) => b.mes.localeCompare(a.mes)).map((item) => ({
    ...item,
    kilometros: Math.round(item.kilometros * 10) / 10,
    total_euros: Math.round(item.total_euros * 100) / 100,
    pagado_euros: Math.round(item.pagado_euros * 100) / 100,
  }));
}

function textoFiltro(valor) {
  return String(valor ?? "").trim().slice(0, 100);
}

export function crearPresentadorDietas({
  datos: datosIniciales, contextoActor: contextoInyectado, capacidades: capacidadesInyectadas = [],
  origenComprobacion = "", catalogoRutas = null,
} = {}) {
  const contextoActor = exigirContextoActorDietas(contextoInyectado);
  const capacidades = validarCapacidadesDietas(capacidadesInyectadas);
  const permisos = Object.freeze({
    consultarGastos: tieneCapacidadDietas(capacidades, CAPACIDAD_CONSULTAR_GASTO),
    gestionarGastos: tieneCapacidadDietas(capacidades, CAPACIDAD_GESTIONAR_GASTO),
    consultarRutas: tieneCapacidadDietas(capacidades, CAPACIDAD_CONSULTAR_RUTA),
    gestionarRutas: tieneCapacidadDietas(capacidades, CAPACIDAD_GESTIONAR_RUTA),
    consultarAuditoria: tieneCapacidadDietas(capacidades, CAPACIDAD_CONSULTAR_AUDITORIA),
    consultarHistorialPropio: tieneCapacidadDietas(capacidades, CAPACIDAD_CONSULTAR_GASTO),
  });
  let datos = validarPanelDietas(datosIniciales, contextoActor.actor.actor_ref, capacidades);
  if (datos.origen.demostracion !== contextoActor.demostracion) {
    throw new Error("el origen de Dietas no coincide con el contexto de actor");
  }
  const rutas = crearPresentadorRutasDietas({
    catalogo: catalogoRutas,
    permisos,
    demostracion: datos.origen.demostracion === true,
  });
  const estado = {
    seleccionada: datos.comisiones[0]?.referencia || "",
    filtroEstado: "todos",
    filtroTexto: "",
  };

  function actualizarDatos(siguientesDatos) {
    datos = validarPanelDietas(siguientesDatos, contextoActor.actor.actor_ref, capacidades);
    const objetivoRecibo = datos.ultimo_recibo?.objetivo;
    if (objetivoRecibo && datos.comisiones.some((item) => item.referencia === objetivoRecibo)) {
      estado.seleccionada = objetivoRecibo;
    } else if (!datos.comisiones.some((item) => item.referencia === estado.seleccionada)) {
      estado.seleccionada = datos.comisiones[0]?.referencia || "";
    }
    return obtenerModelo();
  }

  function obtenerModelo() {
    const termino = estado.filtroTexto.toLocaleLowerCase("es");
    const visiblesSinProyectar = permisos.consultarGastos ? datos.comisiones.filter((item) => {
      const coincideEstado = estado.filtroEstado === "todos" || item.estado === estado.filtroEstado;
      const texto = `${item.referencia} ${item.motivo} ${item.ruta.join(" ")} ${item.estado}`.toLocaleLowerCase("es");
      return coincideEstado && (!termino || texto.includes(termino));
    }) : [];
    const proyectar = (item) => {
      if (!item) return null;
      const salida = copiarDietas(item);
      if (!permisos.consultarRutas) {
        salida.ruta = [];
        salida.kilometros = null;
        salida.kilometraje_euros = null;
        delete salida.mapa_ruta;
      } else if (salida.geometria_ruta) {
        salida.mapa_ruta = Object.freeze({
          vista_ref: `expediente-${salida.referencia}`,
          proveedor: "openstreetmap",
          despliegue: "red_interna",
          plantilla_teselas: PLANTILLA_TESELAS_OSM_INTERNA,
          atribucion: ATRIBUCION_OSM_INTERNA,
          demostracion: datos.origen.demostracion === true,
          geometria: validarGeometriaRutaDietas(salida.geometria_ruta, salida.ruta),
        });
      }
      delete salida.geometria_ruta;
      return salida;
    };
    const seleccionadaSinProyectar = permisos.consultarGastos
      ? datos.comisiones.find((item) => item.referencia === estado.seleccionada) || visiblesSinProyectar[0] || null
      : null;
    const resumen = { ...(permisos.consultarGastos ? resumenDe(datos.comisiones) : resumenDe([])) };
    if (!permisos.consultarRutas) resumen.kilometros = null;
    const resumenAnual = permisos.consultarGastos ? { ...resumenAnualDe(datos.comisiones) } : null;
    if (resumenAnual && !permisos.consultarRutas) {
      delete resumenAnual.kilometros;
      delete resumenAnual.kilometraje_euros;
      // Esta cifra permitiría reconstruir el importe kilométrico por diferencia.
      delete resumenAnual.dietas_gastos_euros;
    }
    return Object.freeze({
      esquema: datos.esquema,
      demostracion: datos.origen.demostracion === true,
      efectos_reales: datos.origen.efectos_reales === true,
      identidad: contextoActor,
      capacidades: permisos,
      politica: copiarDietas(datos.politica),
      borradorInicial: copiarDietas(datos.borrador_inicial || {
        fecha: "", motivo: "", origen: "", destino: "", kilometros: 0,
        manutencion_euros: 0, alojamiento_euros: 0, otros_gastos_euros: 0,
      }),
      etapas: [...datos.etapas],
      filtros: Object.freeze({ estado: estado.filtroEstado, texto: estado.filtroTexto }),
      resumen: Object.freeze(resumen),
      resumenAnual: resumenAnual ? Object.freeze(resumenAnual) : null,
      herramientaRutas: rutas?.obtenerModelo() || null,
      historialMensual: permisos.consultarGastos ? mesesDe(datos.comisiones).map((item) => ({
        ...item, kilometros: permisos.consultarRutas ? item.kilometros : null,
      })) : [],
      comisiones: visiblesSinProyectar.map(proyectar),
      totalSinFiltrar: permisos.consultarGastos ? datos.comisiones.length : 0,
      seleccionada: proyectar(seleccionadaSinProyectar),
      ultimoRecibo: datos.ultimo_recibo ? Object.freeze(copiarDietas(datos.ultimo_recibo)) : null,
    });
  }

  function filtrar({ estado: siguienteEstado = estado.filtroEstado, texto = estado.filtroTexto } = {}) {
    if (!ESTADOS_PERMITIDOS.has(siguienteEstado)) throw new Error("estado de filtro no permitido");
    estado.filtroEstado = siguienteEstado;
    estado.filtroTexto = textoFiltro(texto);
    return obtenerModelo();
  }

  function seleccionar(referencia) {
    if (!permisos.consultarGastos) throw new Error("la sesión no tiene capacidad para consultar Dietas");
    const objetivo = String(referencia ?? "").trim();
    if (!datos.comisiones.some((item) => item.referencia === objetivo)) throw new Error("comision no encontrada");
    estado.seleccionada = objetivo;
    return obtenerModelo();
  }

  function prepararDescriptorRecibo(referenciaRecibo, traducir) {
    if (!permisos.consultarGastos) {
      throw new Error("la sesión no tiene capacidad para descargar recibos de Dietas");
    }
    if (typeof traducir !== "function") throw new Error("traductor documental de Dietas no válido");
    const referencia = String(referenciaRecibo ?? "").trim();
    const comision = datos.comisiones.find((item) => item.historial.some((evento) => evento.recibo === referencia));
    const evento = comision?.historial.find((item) => item.recibo === referencia);
    const referenciaDemo = /^DEMO-[A-Z0-9-]{8,90}$/.test(referencia);
    const referenciaOpaca = /^[A-Za-z0-9][A-Za-z0-9:._-]{5,127}$/.test(referencia);
    if (!comision || !evento || !referenciaOpaca
      || (datos.origen.demostracion === true && !referenciaDemo)
      || (datos.origen.demostracion !== true && referenciaDemo)) {
      throw new Error("recibo de Dietas no encontrado");
    }
    if (!CLAVES_I18N_ESTADO[evento.estado]) throw new Error("estado de recibo de Dietas no válido");
    const rutaComprobacion = `/verificar/?ref=${encodeURIComponent(referencia)}${datos.origen.demostracion === true ? "&presentacion=rrhh" : ""}`;
    let contenidoQR = rutaComprobacion;
    if (origenComprobacion) {
      try {
        contenidoQR = new URL(rutaComprobacion, origenComprobacion).href;
      } catch {
        throw new Error("origen de comprobacion no valido");
      }
    }
    return Object.freeze({
      esquema: ESQUEMA_RECIBO_DIETAS,
      modulo: "dietas",
      formato: "pdf",
      referencia,
      expediente_ref: comision.referencia,
      titulo: traducir("documento_titulo"),
      subtitulo: traducir("documento_subtitulo"),
      identidad_visual: Object.freeze({
        entidad: traducir("documento_entidad"),
        logo_src: "/portal-empleado/assets/logo-diputacion-granada.svg",
        texto_alternativo: traducir("documento_entidad"),
      }),
      demostracion: datos.origen.demostracion === true,
      marca: datos.origen.demostracion === true ? traducir("documento_marca_demo") : "",
      filas: Object.freeze([
        Object.freeze({ etiqueta: traducir("documento_actuacion"), valor: traducir(CLAVES_I18N_ESTADO[evento.estado]) }),
        Object.freeze({ etiqueta: traducir("documento_fecha_comision"), valor: comision.fecha }),
        ...(permisos.consultarRutas ? [
          Object.freeze({ etiqueta: traducir("documento_ruta"), valor: comision.ruta.join(" -> ") }),
          Object.freeze({ etiqueta: traducir("documento_kilometros"), valor: String(comision.kilometros) }),
        ] : []),
        Object.freeze({ etiqueta: traducir("documento_importe_total"), valor: String(comision.total_euros) }),
        Object.freeze({ etiqueta: traducir("documento_instante"), valor: evento.instante }),
      ]),
      comprobacion: Object.freeze({
        ruta: rutaComprobacion,
        qr_contenido: contenidoQR,
        contiene_datos_personales: false,
        metodo: datos.origen.demostracion === true ? "consulta_estatica_demo" : "post_servicio_cotejo",
      }),
    });
  }

  function prepararDescriptorResumenAnual(anioEntrada, traducir) {
    if (!permisos.consultarGastos) throw new Error("la sesión no tiene capacidad para exportar Dietas");
    if (datos.origen.demostracion !== true) {
      throw new Error("el resumen anual productivo debe generarse en el servicio documental autorizado");
    }
    if (typeof traducir !== "function") throw new Error("traductor documental de Dietas no válido");
    const anual = resumenAnualDe(datos.comisiones, anioEntrada);
    const referencia = `DEMO-DIE-REC-ANUAL-${anual.anio}-01`;
    const rutaComprobacion = `/verificar/?ref=${encodeURIComponent(referencia)}&presentacion=rrhh`;
    let contenidoQR = rutaComprobacion;
    if (origenComprobacion) {
      try {
        contenidoQR = new URL(rutaComprobacion, origenComprobacion).href;
      } catch {
        throw new Error("origen de comprobacion no valido");
      }
    }
    const filas = [
      { etiqueta: traducir("documento_anual_anio"), valor: String(anual.anio) },
      { etiqueta: traducir("documento_anual_meses"), valor: String(anual.meses_con_actividad) },
      { etiqueta: traducir("documento_anual_expedientes"), valor: String(anual.expedientes) },
      { etiqueta: traducir("documento_anual_pendientes"), valor: String(anual.pendientes) },
      ...(permisos.consultarRutas ? [
        { etiqueta: traducir("documento_anual_kilometros"), valor: `${anual.kilometros.toFixed(1)} km` },
        { etiqueta: traducir("documento_anual_kilometraje"), valor: `${anual.kilometraje_euros.toFixed(2)} EUR` },
      ] : []),
      { etiqueta: traducir("documento_anual_devengado"), valor: `${anual.devengado_euros.toFixed(2)} EUR` },
      { etiqueta: traducir("documento_anual_pagado"), valor: `${anual.pagado_euros.toFixed(2)} EUR` },
    ];
    return Object.freeze({
      esquema: ESQUEMA_RESUMEN_ANUAL_DIETAS,
      modulo: "dietas",
      formato: "pdf",
      referencia,
      periodo: String(anual.anio),
      titulo: traducir("documento_anual_titulo", { anio: anual.anio }),
      subtitulo: traducir("documento_subtitulo"),
      demostracion: true,
      marca: traducir("documento_marca_demo"),
      nombre_archivo: `resumen-anual-dietas-demo-${anual.anio}.pdf`,
      filas: Object.freeze(filas.map((fila) => Object.freeze(fila))),
      texto_certificacion: traducir("documento_anual_certificacion"),
      comprobacion: Object.freeze({
        ruta: rutaComprobacion,
        qr_contenido: contenidoQR,
        contiene_datos_personales: false,
        metodo: "consulta_estatica_demo",
      }),
    });
  }

  return Object.freeze({
    actualizarDatos,
    obtenerModelo,
    filtrar,
    seleccionar,
    prepararDescriptorRecibo,
    prepararDescriptorResumenAnual,
    rutas,
  });
}
