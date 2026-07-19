/**
 * Puerto de comandos VOLÁTIL y EXCLUSIVO DE PRESENTACIÓN.
 *
 * Docker/producción debe excluir este fichero junto con datos-presentacion.js.
 * No realiza HTTP, no persiste y no produce efectos administrativos.
 */

function copiar(valor) {
  return JSON.parse(JSON.stringify(valor));
}

export function crearEjecutorCronosPresentacion({ reloj = () => new Date() } = {}) {
  let secuencia = 0;
  return async (comando, contexto) => {
    const actorRef = contexto?.identidad?.actor?.actor_ref;
    if (!contexto?.identidad?.demostracion || !contexto?.datos?.demostracion
      || contexto.datos.actor_ref !== actorRef) {
      throw new Error("el adaptador de presentación solo admite el actor DEMO de la sesión");
    }
    const ahora = reloj();
    if (!(ahora instanceof Date) || !Number.isFinite(ahora.getTime())) throw new Error("reloj de presentación no válido");
    secuencia += 1;
    const sufijo = String(secuencia).padStart(4, "0");
    const datos = copiar(contexto.datos);
    const referencia = `DEMO-CRONOS-REC-${sufijo}`;
    let operacion;

    if (comando.tipo === "registrar_fichaje") {
      const etiquetas = {
        entrada: "Entrada",
        salida: "Salida",
        inicio_pausa: "Inicio de pausa",
        fin_pausa: "Fin de pausa",
      };
      const movimiento = etiquetas[comando.movimiento];
      if (!movimiento) throw new Error("movimiento no soportado por el adaptador de presentación");
      operacion = `${movimiento} preparada en presentación`;
      datos.fichajes.unshift({
        id: `DEMO-FIC-VOLATIL-${sufijo}`,
        actor_ref: actorRef,
        instante: ahora.toISOString().replace(/\.\d{3}Z$/, "Z"),
        tipo_clave: comando.movimiento,
        canal: "Portal interno DEMO",
        modalidad: "Presencial",
        estado_clave: "simulado",
        recibo_ref: referencia,
      });
      datos.historial.unshift({
        id: `DEMO-HIS-VOLATIL-${sufijo}`,
        actor_ref: actorRef,
        instante: ahora.toISOString().replace(/\.\d{3}Z$/, "Z"),
        evento: "Fichaje de presentación",
        detalle: movimiento,
        estado_clave: "simulado",
        recibo_ref: referencia,
      });
    } else if (comando.tipo === "solicitar_permiso") {
      const saldo = datos.saldos.find((item) => item.id === comando.permiso_id);
      if (!saldo) throw new Error("tipo de permiso no disponible");
      if (comando.cantidad > Number(saldo.restante)) throw new Error("la cantidad supera el saldo DEMO disponible");
      operacion = "Solicitud de permiso preparada en presentación";
      saldo.solicitado = Number(saldo.solicitado) + comando.cantidad;
      saldo.restante = Number(saldo.restante) - comando.cantidad;
      datos.resumen.solicitudes_pendientes = Number(datos.resumen.solicitudes_pendientes) + 1;
      datos.solicitudes.unshift({
        id: `DEMO-PER-VOLATIL-${sufijo}`,
        actor_ref: actorRef,
        tipo: saldo.nombre,
        desde: comando.desde,
        hasta: comando.hasta,
        cantidad_valor: comando.cantidad,
        unidad_clave: saldo.unidad_clave,
        estado_clave: "preparado_no_registrado",
        recibo_ref: referencia,
      });
      datos.historial.unshift({
        id: `DEMO-HIS-VOLATIL-${sufijo}`,
        actor_ref: actorRef,
        instante: ahora.toISOString().replace(/\.\d{3}Z$/, "Z"),
        evento: "Solicitud preparada",
        detalle: `${saldo.nombre}: ${comando.desde}–${comando.hasta}`,
        estado_clave: "sin_registrar",
        recibo_ref: referencia,
      });
    } else {
      throw new Error("operación no soportada por el adaptador de presentación");
    }

    datos.actualizado_en = ahora.toISOString().replace(/\.\d{3}Z$/, "Z");
    return {
      datos,
      recibo: {
        esquema: "vec.cronos.recibo.v1",
        referencia,
        instante: ahora.toISOString().replace(/\.\d{3}Z$/, "Z"),
        operacion,
        estado: "Simulado · sin persistencia ni efectos",
        estado_clave: "simulado",
        actor_ref: actorRef,
        demostracion: true,
      },
    };
  };
}
