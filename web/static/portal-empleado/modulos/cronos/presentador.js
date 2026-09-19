import {
  CAPACIDAD_CONSULTAR_FICHAJES,
  CAPACIDAD_CONSULTAR_HORARIO,
  CAPACIDAD_CONSULTAR_PERMISOS,
  CAPACIDAD_REGISTRAR_FICHAJE,
  CAPACIDAD_SOLICITAR_PERMISO,
  exigirContextoActorCronos,
  normalizarTextoCronosMultilinea,
  tieneCapacidadCronos,
  validarCapacidadesCronos,
  validarDatosCronos,
} from "./contrato.js";
import { prepararDescriptorReciboCronos } from "./documentos.js";
import { crearTraductorCronos } from "./i18n.js";
import { renderizarAreaCronos } from "./vista.js";

const PROPIETARIO_MONTAJE = Symbol("cronos-presentacion-propietario");
let generacionMontaje = 0;

function copiar(valor) {
  return JSON.parse(JSON.stringify(valor));
}

function cadenaAcotada(valor, nombre, maximo) {
  const texto = String(valor ?? "").trim();
  if (texto.length > maximo) throw new Error(`${nombre} supera el tamaño permitido`);
  return texto;
}

function falloCronos(codigo, detalle) {
  const error = new Error(detalle);
  error.codigo = codigo;
  return error;
}

function mensajeSeguro(error, t, alternativa) {
  const codigos = new Set([
    "acceso_denegado", "puerto_no_disponible", "comando_invalido",
    "respuesta_invalida", "operacion_no_disponible", "operacion_ocupada", "descarga_no_disponible",
  ]);
  const codigo = typeof error?.codigo === "string" && codigos.has(error.codigo)
    ? error.codigo : alternativa;
  return t(`error_${codigo}`);
}

function normalizarComando(comando) {
  if (!comando || typeof comando !== "object" || Array.isArray(comando)) {
    throw new Error("comando de Cronos no válido");
  }
  if (comando.tipo === "registrar_fichaje") {
    if (!new Set(["entrada", "salida", "inicio_pausa", "fin_pausa"]).has(comando.movimiento)) {
      throw new Error("movimiento de fichaje no válido");
    }
    return Object.freeze({ tipo: comando.tipo, movimiento: comando.movimiento });
  }
  if (comando.tipo === "solicitar_permiso") {
    const cantidad = Number(comando.cantidad);
    if (!Number.isSafeInteger(cantidad) || cantidad < 1 || cantidad > 100_000) {
      throw new Error("cantidad de permiso no válida");
    }
    const fecha = /^\d{4}-\d{2}-\d{2}$/;
    if (!fecha.test(comando.desde) || !fecha.test(comando.hasta) || comando.hasta < comando.desde) {
      throw new Error("periodo de permiso no válido");
    }
    return Object.freeze({
      tipo: comando.tipo,
      permiso_id: cadenaAcotada(comando.permiso_id, "tipo de permiso", 80),
      desde: comando.desde,
      hasta: comando.hasta,
      cantidad,
      motivo: cadenaAcotada(comando.motivo, "motivo", 500),
      documento_ref: cadenaAcotada(comando.documento_ref, "referencia documental", 120),
    });
  }
  if (comando.tipo === "preparar_observacion") {
    const observacion = normalizarTextoCronosMultilinea(comando.observacion, "observación", 8);
    return Object.freeze({ tipo: comando.tipo, incidencia_id: cadenaAcotada(comando.incidencia_id, "incidencia", 80), observacion });
  }
  throw new Error("operación de Cronos no reconocida");
}

function validarRecibo(recibo, actorRef, demostracion, ambito) {
  if (!recibo || typeof recibo !== "object" || Array.isArray(recibo)
    || recibo.esquema !== "vec.cronos.recibo.v1"
    || recibo.actor_ref !== actorRef
    || typeof recibo.referencia !== "string" || !/^[A-Za-z0-9][A-Za-z0-9:._-]{5,127}$/.test(recibo.referencia)
    || typeof recibo.instante !== "string"
    || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{3})?Z$/.test(recibo.instante)
    || !Number.isFinite(Date.parse(recibo.instante)) || typeof recibo.operacion !== "string"
    || typeof recibo.estado !== "string"
    || recibo.ambito_clave !== ambito
    || !new Set(["registrado", "simulado"]).has(recibo.estado_clave)) {
    throw new Error("el ejecutor no ha devuelto un recibo válido");
  }
  if (demostracion && recibo.efectos_reales !== false) throw new Error("un recibo de presentación debe declarar sin efectos reales");
  if (demostracion && !recibo.referencia.startsWith("DEMO-")) {
    throw new Error("un recibo de presentación debe estar marcado como DEMO");
  }
  return Object.freeze({
    referencia: recibo.referencia,
    instante: recibo.instante,
    operacion: recibo.operacion,
    estado: recibo.estado,
    estado_clave: recibo.estado_clave,
    ambito_clave: recibo.ambito_clave,
  });
}

/**
 * Presentador definitivo, independiente del transporte. `ejecutor` es el puerto
 * de comandos: en presentación se conecta al adaptador volátil; en producción
 * se sustituye por el cliente de la API interna sin modificar esta vista.
 */
export function crearPresentadorCronos({
  contextoActor, capacidades = [], datos, ejecutor, mensajes,
  descargarRecibo: descargarReciboInyectado, origenComprobacion = "",
  locale = "es-ES", zonaHoraria = "Europe/Madrid",
} = {}) {
  const contexto = exigirContextoActorCronos(contextoActor);
  const capacidadesValidadas = validarCapacidadesCronos(capacidades);
  let estadoDatos = validarDatosCronos(datos, contexto);
  let recibos = [];
  let mensaje = "";
  let enEjecucion = false;

  function ambitosDeRecibo(referencia, datosEntrada = estadoDatos, recibosEntrada = recibos) {
    const ambitos = new Set();
    if (datosEntrada.fichajes.some((item) => item.recibo_ref === referencia)) ambitos.add("fichaje");
    if (datosEntrada.solicitudes.some((item) => item.recibo_ref === referencia)) ambitos.add("permiso");
    for (const item of datosEntrada.historial) {
      if (item.recibo_ref === referencia) ambitos.add(item.ambito_clave);
    }
    for (const item of recibosEntrada) {
      if (item.referencia === referencia) ambitos.add(item.ambito_clave);
    }
    return ambitos;
  }

  function proyectar(datosEntrada, recibosEntrada = recibos) {
    const fichajes = tieneCapacidadCronos(capacidadesValidadas, CAPACIDAD_CONSULTAR_FICHAJES);
    const permisos = tieneCapacidadCronos(capacidadesValidadas, CAPACIDAD_CONSULTAR_PERMISOS);
    const horario = tieneCapacidadCronos(capacidadesValidadas, CAPACIDAD_CONSULTAR_HORARIO);
    const datosProyectados = copiar(datosEntrada);
    if (!fichajes) {
      datosProyectados.fichajes = []; datosProyectados.incidencias = [];
      for (const campo of ["trabajadas_hoy", "saldo_hoy", "saldo_periodo", "incidencias_abiertas"]) {
        datosProyectados.resumen[campo] = null;
      }
    }
    if (!permisos) {
      datosProyectados.saldos = []; datosProyectados.solicitudes = [];
      datosProyectados.resumen.solicitudes_pendientes = null;
    }
    if (!horario) {
      datosProyectados.perfil_jornada = null;
      datosProyectados.resumen.teoricas_hoy = null;
      datosProyectados.resumen.saldo_hoy = null;
      datosProyectados.resumen.saldo_periodo = null;
    }
    datosProyectados.historial = datosProyectados.historial.filter((item) =>
      (item.ambito_clave === "fichaje" ? fichajes : permisos) && (!item.requiere_horario || horario));
    return Object.freeze({ datos: datosProyectados, recibos: recibosEntrada.filter((item) =>
      (item.ambito_clave === "fichaje" ? fichajes : permisos)
      && (horario || !datosEntrada.historial.some((evento) => evento.recibo_ref === item.referencia && evento.requiere_horario))) });
  }

  function datosParaVista(datosProyectados) {
    const datosVista = copiar(datosProyectados);
    // Adaptación interna al contrato estricto de la vista. Sus indicadores
    // ocultan estos sentinelas por capacidad; el estado público conserva null.
    datosVista.perfil_jornada ??= { referencia: "—", nombre: "—", jornada_diaria: "—", jornada_semanal: "—", ventana_entrada: "—", tramo_obligatorio: "—", teletrabajo: "—" };
    for (const campo of ["teoricas_hoy", "trabajadas_hoy", "saldo_hoy", "saldo_periodo"]) datosVista.resumen[campo] ??= "—";
    for (const campo of ["incidencias_abiertas", "solicitudes_pendientes"]) datosVista.resumen[campo] ??= 0;
    return datosVista;
  }

  function renderizar() {
    return renderizarAreaCronos({
      contextoActor: contexto,
      capacidades: capacidadesValidadas,
      datos: datosParaVista(proyectar(estadoDatos).datos),
      recibos: proyectar(estadoDatos).recibos,
      mensaje,
      descargaRecibosDisponible: typeof descargarReciboInyectado === "function",
      locale,
      zonaHoraria,
      ...(mensajes ? { mensajes } : {}),
    });
  }

  function obtenerEstado() {
    return Object.freeze({
      identidad: contexto,
      capacidades: capacidadesValidadas,
      datos: copiar(proyectar(estadoDatos).datos),
      recibos: copiar(proyectar(estadoDatos).recibos),
      mensaje,
    });
  }

  async function ejecutar(comandoSinValidar) {
    if (enEjecucion) throw falloCronos("operacion_ocupada", "Cronos ya está preparando una acción");
    const comando = normalizarComando(comandoSinValidar);
    const capacidad = comando.tipo === "preparar_observacion" ? CAPACIDAD_CONSULTAR_FICHAJES : comando.tipo === "registrar_fichaje"
      ? CAPACIDAD_REGISTRAR_FICHAJE
      : CAPACIDAD_SOLICITAR_PERMISO;
    if (!tieneCapacidadCronos(capacidadesValidadas, capacidad)) {
      throw falloCronos("acceso_denegado", "operación denegada por la política de mínimo privilegio");
    }
    if (typeof ejecutor !== "function") {
      throw falloCronos("puerto_no_disponible", "puerto de comandos de Cronos no conectado");
    }
    if (comando.tipo === "preparar_observacion" && !tieneCapacidadCronos(capacidadesValidadas, CAPACIDAD_REGISTRAR_FICHAJE)) {
      throw falloCronos("acceso_denegado", "operación denegada por la política de mínimo privilegio");
    }
    if (comando.tipo === "solicitar_permiso" && !tieneCapacidadCronos(capacidadesValidadas, CAPACIDAD_CONSULTAR_PERMISOS)) {
      throw falloCronos("acceso_denegado", "operación denegada por la política de mínimo privilegio");
    }
    enEjecucion = true;
    let resultado;
    try {
      resultado = await ejecutor(comando, Object.freeze({ identidad: contexto, capacidades: capacidadesValidadas, datos: copiar(estadoDatos) }));
    } finally { enEjecucion = false; }
    if (!resultado || typeof resultado !== "object") {
      throw falloCronos("respuesta_invalida", "respuesta del ejecutor de Cronos no válida");
    }
    const nuevosDatos = validarDatosCronos(resultado.datos, contexto);
    const ambito = comando.tipo === "solicitar_permiso" ? "permiso" : "fichaje";
    const recibo = validarRecibo(resultado.recibo, contexto.actor.actor_ref, nuevosDatos.demostracion, ambito);
    const recibosSiguientes = [recibo, ...recibos];
    for (const item of recibosSiguientes) {
      if (ambitosDeRecibo(item.referencia, nuevosDatos, recibosSiguientes).size !== 1) {
        throw falloCronos("respuesta_invalida", "recibo de sesión con ámbitos incompatibles");
      }
    }
    estadoDatos = nuevosDatos;
    recibos = recibosSiguientes.slice(0, 12);
    mensaje = `${recibo.operacion}. ${recibo.estado}. Recibo ${recibo.referencia}.`;
    return recibo;
  }

  function prepararDescriptorRecibo(referencia) {
    const ambitos = ambitosDeRecibo(referencia);
    if (ambitos.size !== 1) throw falloCronos("acceso_denegado", "recibo fuera del ámbito propio o ambiguo");
    const [ambito] = ambitos;
    const capacidad = ambito === "fichaje" ? CAPACIDAD_CONSULTAR_FICHAJES : CAPACIDAD_CONSULTAR_PERMISOS;
    const horario = tieneCapacidadCronos(capacidadesValidadas, CAPACIDAD_CONSULTAR_HORARIO);
    if (!tieneCapacidadCronos(capacidadesValidadas, capacidad)
      || (!horario && estadoDatos.historial.some((item) => item.recibo_ref === referencia && item.requiere_horario))) {
      throw falloCronos("acceso_denegado", "descarga denegada por la política de mínimo privilegio");
    }
    // El generador documental recibe únicamente el recibo del ámbito concedido.
    // No puede escoger otro registro por orden de búsqueda ni recibe sus datos.
    const datosRecibo = datosParaVista({
      ...estadoDatos,
      perfil_jornada: null,
      resumen: Object.fromEntries(Object.keys(estadoDatos.resumen).map((campo) => [campo, null])),
      fichajes: ambito === "fichaje" ? estadoDatos.fichajes.filter((item) => item.recibo_ref === referencia) : [],
      solicitudes: ambito === "permiso" ? estadoDatos.solicitudes.filter((item) => item.recibo_ref === referencia) : [],
      historial: estadoDatos.historial.filter((item) => item.recibo_ref === referencia
        && item.ambito_clave === ambito && (!item.requiere_horario || horario)),
      saldos: [], incidencias: [],
    });
    return prepararDescriptorReciboCronos({
      contextoActor: contexto,
      datos: datosRecibo,
      recibo_ref: referencia,
      origenComprobacion,
    });
  }

  async function descargarReciboPDF(referencia) {
    if (typeof descargarReciboInyectado !== "function") {
      throw falloCronos("descarga_no_disponible", "puerto de descarga PDF no conectado");
    }
    const descriptor = prepararDescriptorRecibo(referencia);
    const resultado = await descargarReciboInyectado(descriptor);
    mensaje = `Recibo ${descriptor.referencia} preparado como PDF institucional verificable.`;
    return resultado ?? descriptor;
  }

  function instalarEventos({ raiz, alCambiar = () => {}, anunciar = () => {} } = {}) {
    if (!raiz || typeof raiz.addEventListener !== "function") {
      throw new Error("raíz DOM de Cronos no válida");
    }
    let ocupado = false;
    let activo = true;
    const generacion = ++generacionMontaje;
    raiz[PROPIETARIO_MONTAJE] = generacion;
    const esPropietario = () => activo && raiz[PROPIETARIO_MONTAJE] === generacion;
    const repintar = () => {
      if (!esPropietario()) return;
      raiz.innerHTML = renderizar();
      alCambiar(obtenerEstado());
      anunciar(mensaje);
    };
    const procesar = async (comando) => {
      if (!esPropietario() || ocupado) return;
      ocupado = true;
      try {
        await ejecutar(comando);
      } catch (error) {
        mensaje = mensajeSeguro(error, crearTraductorCronos(mensajes), "operacion_no_disponible");
      } finally {
        ocupado = false;
        repintar();
      }
    };
    const procesarDescarga = async (referencia) => {
      if (!esPropietario() || ocupado) return;
      ocupado = true;
      try {
        await descargarReciboPDF(referencia);
      } catch (error) {
        mensaje = mensajeSeguro(error, crearTraductorCronos(mensajes), "descarga_no_disponible");
      } finally {
        ocupado = false;
        repintar();
      }
    };
    const alPulsar = (evento) => {
      if (!esPropietario()) return;
      const navegacion = evento.target?.closest?.("[data-cronos-destino]");
      if (navegacion && raiz.contains(navegacion)) {
        evento.preventDefault();
        const destinos = new Set(["cronos-resumen", "cronos-fichajes", "cronos-permisos", "cronos-historial"]);
        const id = navegacion.dataset.cronosDestino;
        if (destinos.has(id)) raiz.querySelector(`#${id}`)?.scrollIntoView?.({ block: "start" });
        return;
      }
      const descarga = evento.target?.closest?.("[data-cronos-accion='descargar-recibo']");
      if (descarga && raiz.contains(descarga)) {
        evento.preventDefault();
        void procesarDescarga(descarga.dataset.cronosReciboRef);
        return;
      }
      const observacion = evento.target?.closest?.("[data-cronos-accion='preparar-observacion']");
      if (observacion && raiz.contains(observacion)) {
        evento.preventDefault();
        const campo = raiz.querySelector?.(`[data-cronos-observacion="${observacion.dataset.cronosIncidenciaRef}"]`);
        void procesar({ tipo: "preparar_observacion", incidencia_id: observacion.dataset.cronosIncidenciaRef, observacion: campo?.value ?? "" });
        return;
      }
      const boton = evento.target?.closest?.("[data-cronos-accion='registrar-fichaje']");
      if (!boton || !raiz.contains(boton)) return;
      evento.preventDefault();
      void procesar({ tipo: "registrar_fichaje", movimiento: boton.dataset.cronosTipo });
    };
    const alEnviar = (evento) => {
      if (!esPropietario()) return;
      const formulario = evento.target?.closest?.("[data-cronos-formulario='solicitud-permiso']");
      if (!formulario || !raiz.contains(formulario)) return;
      evento.preventDefault();
      if (typeof formulario.reportValidity === "function" && !formulario.reportValidity()) return;
      const campos = new FormData(formulario);
      void procesar({
        tipo: "solicitar_permiso",
        permiso_id: campos.get("tipo"),
        desde: campos.get("desde"),
        hasta: campos.get("hasta"),
        cantidad: campos.get("cantidad"),
        motivo: campos.get("motivo"),
        documento_ref: campos.get("documento_ref"),
      });
    };

    raiz.addEventListener("click", alPulsar);
    raiz.addEventListener("submit", alEnviar);
    return () => {
      activo = false;
      if (raiz[PROPIETARIO_MONTAJE] === generacion) delete raiz[PROPIETARIO_MONTAJE];
      raiz.removeEventListener("click", alPulsar);
      raiz.removeEventListener("submit", alEnviar);
    };
  }

  return Object.freeze({
    descargarReciboPDF,
    ejecutar,
    instalarEventos,
    obtenerEstado,
    prepararDescriptorRecibo,
    renderizar,
  });
}
