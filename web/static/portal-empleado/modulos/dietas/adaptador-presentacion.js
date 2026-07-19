/**
 * Puerto de comandos y consulta volátiles para la presentación.
 *
 * Es el unico fichero que simula persistencia. Produccion lo sustituye por el
 * adaptador HTTP sin modificar contrato, presentador ni vista.
 */
import {
  CAPACIDAD_CONSULTAR_GASTO,
  CAPACIDAD_CONSULTAR_RUTA,
  CAPACIDAD_GESTIONAR_GASTO,
  CAPACIDAD_GESTIONAR_RUTA,
  copiarDietas,
  exigirContextoActorDietas,
  tieneCapacidadDietas,
  validarCapacidadesDietas,
  validarComandoDietas,
  validarGeometriaRutaDietas,
  validarPanelDietas,
} from "./contrato.js";
import {
  crearDatosDietasPresentacion,
  crearGeometriaRutaDietasPresentacion,
} from "./datos-presentacion.js";

function numeroNoNegativo(valor, nombre) {
  const numero = Number(valor ?? 0);
  if (!Number.isFinite(numero) || numero < 0 || numero > 1_000_000) throw new Error(`${nombre} no valido`);
  return Math.round(numero * 100) / 100;
}

function textoAcotado(valor, nombre, maximo = 180) {
  const texto = String(valor ?? "").trim();
  if (!texto || texto.length > maximo || /[\u0000-\u0008\u000B\u000C\u000E-\u001F]/.test(texto)) {
    throw new Error(`${nombre} no valido`);
  }
  return texto;
}

function fechaValida(valor) {
  const fecha = textoAcotado(valor, "fecha", 10);
  if (!/^\d{4}-\d{2}-\d{2}$/.test(fecha) || !Number.isFinite(Date.parse(`${fecha}T00:00:00Z`))) {
    throw new Error("fecha no valida");
  }
  return fecha;
}

function horaValida(valor, nombre) {
  const hora = textoAcotado(valor, nombre, 5);
  if (!/^(?:[01]\d|2[0-3]):[0-5]\d$/.test(hora)) throw new Error(`${nombre} no valida`);
  return hora;
}

function rutaCalculada(campos) {
  if (!Array.isArray(campos.ruta)) return null;
  if (campos.ruta.length < 2 || campos.ruta.length > 12) throw new Error("ruta calculada no valida");
  const ruta = campos.ruta.map((parada) => textoAcotado(parada, "parada", 100));
  const geometria = validarGeometriaRutaDietas(campos.geometria_ruta, ruta);
  const traza = campos.trazabilidad_ruta;
  if (!traza || typeof traza !== "object" || Array.isArray(traza)
    || traza.motor !== "simulacion_osrm_demo" || traza.liquidable !== false
    || typeof traza.calculo_ref !== "string" || !/^DEMO-RUTA-[A-Z0-9-]{6,80}$/.test(traza.calculo_ref)
    || !Array.isArray(traza.ajustes)) {
    throw new Error("trazabilidad de ruta DEMO no valida");
  }
  return { ruta, geometria, traza: copiarDietas(traza) };
}

export function crearAdaptadorDietasPresentacion({
  contextoActor: contextoInyectado, capacidades: capacidadesInyectadas = [], reloj = () => new Date(), crearReferencia,
} = {}) {
  const contextoActor = exigirContextoActorDietas(contextoInyectado);
  const capacidades = validarCapacidadesDietas(capacidadesInyectadas);
  if (contextoActor.demostracion !== true) throw new Error("el adaptador de presentación exige un contexto DEMO");
  let datos = validarPanelDietas(
    crearDatosDietasPresentacion(contextoActor), contextoActor.actor.actor_ref,
  );
  let secuencia = 0;
  const nuevaReferencia = typeof crearReferencia === "function"
    ? crearReferencia
    : () => `DEMO-DIE-NUEVA-${String(++secuencia).padStart(3, "0")}`;

  function instanteActual() {
    const valor = reloj();
    const fecha = valor instanceof Date ? valor : new Date(valor);
    if (!Number.isFinite(fecha.getTime())) throw new Error("reloj de Dietas no valido");
    return fecha.toISOString();
  }

  function emitirRecibo(operacion, objetivo, resultado) {
    const recibo = {
      referencia: `DEMO-REC-DIE-VOL-${String(++secuencia).padStart(4, "0")}`,
      operacion,
      objetivo,
      resultado,
      actor_ref: contextoActor.actor.actor_ref,
      instante: instanteActual(),
      efectos_reales: false,
      persistencia: "memoria_volatil",
    };
    datos.ultimo_recibo = recibo;
    return recibo;
  }

  function crearBorrador(campos) {
    const referencia = textoAcotado(nuevaReferencia(), "referencia", 100);
    if (datos.comisiones.some((item) => item.referencia === referencia)) throw new Error("referencia de comision repetida");
    const fecha = fechaValida(campos.fecha);
    const fechaFin = fechaValida(campos.fecha_fin || campos.fecha);
    const horaInicio = horaValida(campos.hora_inicio || "08:00", "hora de inicio");
    const horaFin = horaValida(campos.hora_fin || "15:00", "hora de fin");
    if (`${fechaFin}T${horaFin}` < `${fecha}T${horaInicio}`) throw new Error("el fin de la comision no puede ser anterior al inicio");
    const motivo = textoAcotado(campos.motivo, "motivo");
    const calculada = rutaCalculada(campos);
    const origen = calculada?.ruta[0] || textoAcotado(campos.origen, "origen", 80);
    const destino = calculada ? textoAcotado(campos.destino, "destino", 100) : textoAcotado(campos.destino, "destino", 80);
    const kilometros = numeroNoNegativo(campos.kilometros, "kilometros");
    const manutencion = numeroNoNegativo(campos.manutencion_euros, "manutencion");
    const alojamiento = numeroNoNegativo(campos.alojamiento_euros, "alojamiento");
    const otros = numeroNoNegativo(campos.otros_gastos_euros, "otros gastos");
    const kilometraje = Math.round(kilometros * Number(datos.politica.tarifa_kilometro_euros) * 100) / 100;
    const total = Math.round((kilometraje + manutencion + alojamiento + otros) * 100) / 100;
    const recibo = emitirRecibo("crear_borrador", referencia, "borrador_creado_demo");
    datos.comisiones.unshift({
      referencia,
      titular_ref: contextoActor.actor.actor_ref,
      fecha,
      fecha_fin: fechaFin,
      hora_inicio: horaInicio,
      hora_fin: horaFin,
      vehiculo_propio: campos.vehiculo_propio === true,
      motivo,
      ruta: calculada?.ruta || (origen === destino ? [origen, origen] : [origen, destino, origen]),
      geometria_ruta: calculada?.geometria || crearGeometriaRutaDietasPresentacion(
        origen === destino ? [origen, destino] : [origen, destino, origen],
      ),
      ...(calculada ? { trazabilidad_ruta: calculada.traza } : {}),
      kilometros,
      kilometraje_euros: kilometraje,
      manutencion_euros: manutencion,
      alojamiento_euros: alojamiento,
      otros_gastos_euros: otros,
      total_euros: total,
      estado: "borrador",
      etapa_actual: 0,
      justificantes: 0,
      siguiente_actuacion: "completar_enviar_validacion",
      historial: [{ estado: "borrador", instante: recibo.instante, actor_ref: recibo.actor_ref, recibo: recibo.referencia }],
    });
  }

  function enviarValidacion(referencia) {
    const comision = datos.comisiones.find((item) => item.referencia === referencia);
    if (!comision) throw new Error("comision no encontrada");
    if (comision.estado !== "borrador") throw new Error("solo se puede enviar un borrador");
    const recibo = emitirRecibo("enviar_validacion", referencia, "envio_jefatura_demo");
    comision.estado = "pendiente_jefatura";
    comision.etapa_actual = 1;
    comision.siguiente_actuacion = "revision_jefatura";
    comision.historial.push({ estado: "pendiente_jefatura", instante: recibo.instante, actor_ref: recibo.actor_ref, recibo: recibo.referencia });
  }

  function proyectarDatosAutorizados() {
    const salida = copiarDietas(datos);
    if (!tieneCapacidadDietas(capacidades, CAPACIDAD_CONSULTAR_GASTO)) {
      salida.comisiones = [];
      salida.ultimo_recibo = null;
      return salida;
    }
    if (!tieneCapacidadDietas(capacidades, CAPACIDAD_CONSULTAR_RUTA)) {
      salida.comisiones.forEach((comision) => {
        comision.ruta = [];
        comision.geometria_ruta = null;
        comision.kilometros = null;
        comision.kilometraje_euros = null;
      });
    }
    return salida;
  }

  return Object.freeze({
    obtenerDatos() {
      return proyectarDatosAutorizados();
    },
    ejecutar(comandoEntrada) {
      const comando = validarComandoDietas(comandoEntrada);
      if (!tieneCapacidadDietas(capacidades, CAPACIDAD_GESTIONAR_GASTO)) {
        throw new Error("la sesión no tiene capacidad para gestionar Dietas");
      }
      if (comando.tipo === "crear_borrador") {
        if (!tieneCapacidadDietas(capacidades, CAPACIDAD_GESTIONAR_RUTA)) {
          throw new Error("la sesión no tiene capacidad para gestionar rutas de Dietas");
        }
        crearBorrador(comando.campos);
      }
      if (comando.tipo === "enviar_validacion") enviarValidacion(comando.referencia);
      return proyectarDatosAutorizados();
    },
  });
}
