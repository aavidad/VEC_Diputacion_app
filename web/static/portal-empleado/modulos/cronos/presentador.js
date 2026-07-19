import {
  CAPACIDAD_CONSULTAR_FICHAJES,
  CAPACIDAD_CONSULTAR_PERMISOS,
  CAPACIDAD_REGISTRAR_FICHAJE,
  CAPACIDAD_SOLICITAR_PERMISO,
  exigirContextoActorCronos,
  tieneCapacidadCronos,
  validarCapacidadesCronos,
  validarDatosCronos,
} from "./contrato.js";
import { prepararDescriptorReciboCronos } from "./documentos.js";
import { renderizarAreaCronos } from "./vista.js";

function copiar(valor) {
  return JSON.parse(JSON.stringify(valor));
}

function cadenaAcotada(valor, nombre, maximo) {
  const texto = String(valor ?? "").trim();
  if (texto.length > maximo) throw new Error(`${nombre} supera el tamaño permitido`);
  return texto;
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
  throw new Error("operación de Cronos no reconocida");
}

function validarRecibo(recibo, actorRef, demostracion) {
  if (!recibo || typeof recibo !== "object" || Array.isArray(recibo)
    || recibo.esquema !== "vec.cronos.recibo.v1"
    || recibo.actor_ref !== actorRef
    || typeof recibo.referencia !== "string" || !/^[A-Za-z0-9][A-Za-z0-9:._-]{5,127}$/.test(recibo.referencia)
    || typeof recibo.instante !== "string"
    || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{3})?Z$/.test(recibo.instante)
    || !Number.isFinite(Date.parse(recibo.instante)) || typeof recibo.operacion !== "string"
    || typeof recibo.estado !== "string"
    || !new Set(["registrado", "simulado"]).has(recibo.estado_clave)) {
    throw new Error("el ejecutor no ha devuelto un recibo válido");
  }
  if (demostracion && !recibo.referencia.startsWith("DEMO-")) {
    throw new Error("un recibo de presentación debe estar marcado como DEMO");
  }
  return Object.freeze({
    referencia: recibo.referencia,
    instante: recibo.instante,
    operacion: recibo.operacion,
    estado: recibo.estado,
    estado_clave: recibo.estado_clave,
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

  function renderizar() {
    return renderizarAreaCronos({
      contextoActor: contexto,
      capacidades: capacidadesValidadas,
      datos: estadoDatos,
      recibos,
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
      datos: copiar(estadoDatos),
      recibos: copiar(recibos),
      mensaje,
    });
  }

  async function ejecutar(comandoSinValidar) {
    const comando = normalizarComando(comandoSinValidar);
    const capacidad = comando.tipo === "registrar_fichaje"
      ? CAPACIDAD_REGISTRAR_FICHAJE
      : CAPACIDAD_SOLICITAR_PERMISO;
    if (!tieneCapacidadCronos(capacidadesValidadas, capacidad)) {
      throw new Error("operación denegada por la política de mínimo privilegio");
    }
    if (typeof ejecutor !== "function") {
      throw new Error("puerto de comandos de Cronos no conectado");
    }
    const resultado = await ejecutor(comando, Object.freeze({
      identidad: contexto,
      capacidades: capacidadesValidadas,
      datos: copiar(estadoDatos),
    }));
    if (!resultado || typeof resultado !== "object") {
      throw new Error("respuesta del ejecutor de Cronos no válida");
    }
    const nuevosDatos = validarDatosCronos(resultado.datos, contexto);
    const recibo = validarRecibo(resultado.recibo, contexto.actor.actor_ref, nuevosDatos.demostracion);
    estadoDatos = nuevosDatos;
    recibos = [recibo, ...recibos].slice(0, 12);
    mensaje = `${recibo.operacion}. ${recibo.estado}. Recibo ${recibo.referencia}.`;
    return recibo;
  }

  function prepararDescriptorRecibo(referencia) {
    const esFichaje = estadoDatos.fichajes.some((item) => item.recibo_ref === referencia);
    const capacidad = esFichaje ? CAPACIDAD_CONSULTAR_FICHAJES : CAPACIDAD_CONSULTAR_PERMISOS;
    if (!tieneCapacidadCronos(capacidadesValidadas, capacidad)) {
      throw new Error("descarga denegada por la política de mínimo privilegio");
    }
    return prepararDescriptorReciboCronos({
      contextoActor: contexto,
      datos: estadoDatos,
      recibo_ref: referencia,
      origenComprobacion,
    });
  }

  async function descargarReciboPDF(referencia) {
    if (typeof descargarReciboInyectado !== "function") throw new Error("puerto de descarga PDF no conectado");
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

    const repintar = () => {
      raiz.innerHTML = renderizar();
      alCambiar(obtenerEstado());
      anunciar(mensaje);
    };
    const procesar = async (comando) => {
      if (ocupado) return;
      ocupado = true;
      try {
        await ejecutar(comando);
      } catch (error) {
        mensaje = error instanceof Error ? error.message : "No se pudo completar la operación";
      } finally {
        ocupado = false;
        repintar();
      }
    };
    const procesarDescarga = async (referencia) => {
      if (ocupado) return;
      ocupado = true;
      try {
        await descargarReciboPDF(referencia);
      } catch (error) {
        mensaje = error instanceof Error ? error.message : "No se pudo descargar el recibo";
      } finally {
        ocupado = false;
        repintar();
      }
    };
    const alPulsar = (evento) => {
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
      const boton = evento.target?.closest?.("[data-cronos-accion='registrar-fichaje']");
      if (!boton || !raiz.contains(boton)) return;
      evento.preventDefault();
      void procesar({ tipo: "registrar_fichaje", movimiento: boton.dataset.cronosTipo });
    };
    const alEnviar = (evento) => {
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
