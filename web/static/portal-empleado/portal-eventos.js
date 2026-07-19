/**
 * Controladores de interacción del Portal del Empleado.
 *
 * Recibe sus dependencias desde `portal.js`: no conoce repositorios ni decide
 * negocio. Las acciones sin comando de servidor compuesto permanecen
 * informativas y nunca producen efectos administrativos en el navegador.
 */
const NOMBRES_CAMPOS_OPERACION = new Set([
  "denominacion", "categoria", "expediente", "tipo_proceso", "apertura", "cierre",
  "subsanacion_desde", "subsanacion_hasta", "version_bases", "medio_publicacion", "plantilla",
  "circuito_firma", "criterio", "motivo_tipificado", "observacion", "unidad_tiempo",
  "puntos_unidad", "fraccion_jornada", "tope_bloque", "ambito_experiencia", "redondeo",
  "desempate_1", "desempate_2", "desempate_3", "ultimo_recurso", "bolsa", "destino",
  "jornada", "duracion", "regla", "plazo_respuesta", "canales", "formato", "circuito",
  "cotejo", "plazo", "asunto", "contenido", "informe", "ambito", "finalidad", "nombre",
  "unidad", "recursos", "acciones", "vigencia_desde", "revision",
]);

const NOMBRES_FILTRO = Object.freeze({
  convocatorias: new Set(["texto", "estado", "unidad"]),
  solicitudes: new Set(["referencia", "convocatoria", "estado"]),
  meritos: new Set(["referencia", "tipo", "estado"]),
});

function serializarEntradasCerradas(entradas, nombresPermitidos) {
  if (!entradas || typeof entradas[Symbol.iterator] !== "function") {
    throw new TypeError("el formulario no proporciona entradas válidas");
  }
  const salida = {};
  let cantidad = 0;
  for (const entrada of entradas) {
    if (!Array.isArray(entrada) || entrada.length !== 2) throw new TypeError("entrada de formulario no válida");
    cantidad += 1;
    if (cantidad > 32) throw new RangeError("el formulario supera el máximo de campos");
    const [nombre, valor] = entrada;
    if (typeof nombre !== "string" || !/^[a-z][a-z0-9_]{0,47}$/.test(nombre)
      || !nombresPermitidos.has(nombre)) {
      throw new TypeError("nombre de campo no canónico");
    }
    if (Object.hasOwn(salida, nombre)) throw new TypeError("el formulario contiene campos duplicados");
    if (typeof valor !== "string") throw new TypeError("el formulario no admite ficheros");
    if (valor.length > 2_000 || /[\u0000-\u0008\u000B\u000C\u000E-\u001F]/.test(valor)) {
      throw new RangeError("valor de formulario no válido");
    }
    salida[nombre] = valor.trim();
  }
  return Object.freeze(salida);
}

export function serializarCamposOperacion(formData) {
  return serializarEntradasCerradas(formData, NOMBRES_CAMPOS_OPERACION);
}

export function serializarCamposFiltro(tipo, formData) {
  const nombresPermitidos = NOMBRES_FILTRO[tipo];
  if (!nombresPermitidos) throw new TypeError("tipo de filtro no permitido");
  return serializarEntradasCerradas(formData, nombresPermitidos);
}

export function contenerTabulacionMenu(evento, lateral, elementoActivo) {
  if (evento?.key !== "Tab" || !lateral) return false;
  const controles = [...lateral.querySelectorAll('a[href], button:not(:disabled), input:not(:disabled), select:not(:disabled), textarea:not(:disabled), [tabindex]:not([tabindex="-1"])')]
    .filter((control) => control.getAttribute?.("aria-hidden") !== "true" && !control.closest?.("[hidden]"));
  if (controles.length === 0) {
    evento.preventDefault();
    return true;
  }
  const primero = controles[0];
  const ultimo = controles.at(-1);
  const activoDentro = lateral.contains(elementoActivo);
  if (evento.shiftKey && (!activoDentro || elementoActivo === primero)) {
    evento.preventDefault();
    ultimo.focus();
    return true;
  }
  if (!evento.shiftKey && (!activoDentro || elementoActivo === ultimo)) {
    evento.preventDefault();
    primero.focus();
    return true;
  }
  return false;
}

export function crearControladorPortal(dependencias) {
  const {
    anunciar,
    asistenteLlamamientos,
    cargarFuenteDatos,
    cerrarMenuMovil,
    confirmarOperacionPresentacion,
    describirOperacionPresentacion,
    escaparHTML,
    estado,
    etiquetaFuentePanel,
    ejecutarOperacionPresentacion,
    navegar,
    notaOperacionNoCompuesta,
    numero,
    obtenerDatosPanel,
    operacionPermitida,
    porcentajeSeguro,
    porId,
    renderizar,
    renderizarContenidoAyuda,
    solicitarPropuestaLlamamiento,
    vistaDesdeHash,
  } = dependencias;

  function abrirDialogo(titulo, contenido) {
    const dialogo = porId("dialogo-detalle");
    porId("titulo-dialogo").textContent = titulo;
    porId("contenido-dialogo").innerHTML = contenido;
    if (typeof dialogo.showModal === "function") dialogo.showModal();
    else dialogo.setAttribute("open", "");
  }

  function detalleLimitacion(titulo) {
    abrirDialogo(titulo, `<p class="nota-pendiente">${escaparHTML(notaOperacionNoCompuesta())}</p><p>Su activación exige autorización por expediente, validación de estado, persistencia, auditoría y, cuando proceda, firma o recibo verificable del conector.</p>`);
  }

  async function manejarAccion(boton) {
    const accion = boton.dataset.accion;
    const id = boton.dataset.id;
    const datosPanel = obtenerDatosPanel();
    switch (accion) {
      case "recargar-fuente":
        estado.errorFuente = "";
        estado.fuenteLista = false;
        renderizar();
        cargarFuenteDatos().then(renderizar).catch(() => {
          estado.errorFuente = "No se pudo volver a comprobar la fuente interna.";
          renderizar();
        });
        break;
      case "nuevo-llamamiento":
        asistenteLlamamientos.reiniciar(estado, datosPanel.necesidades_llamamiento[0]?.id || "");
        estado.errorPropuesta = "";
        navegar("llamamientos");
        break;
      case "nueva-bolsa":
        abrirDialogo("Nueva bolsa", `<ol><li>Identificación y categoría.</li><li>Bases y documentación.</li><li>Requisitos y baremo versionado.</li><li>Calendario y tribunal.</li><li>Firmas, publicación y transparencia.</li></ol><p class="nota-pendiente">${escaparHTML(notaOperacionNoCompuesta())}</p>`);
        break;
      case "seleccionar-elaboracion":
        estado.elaboracionSeleccionada = id;
        renderizar();
        anunciar("Expediente seleccionado");
        break;
      case "configurar-bases":
        abrirDialogo("Configurar bases y baremo", '<dl class="resumen-expediente"><div class="fila-resumen"><dt>Experiencia</dt><dd>Unidad, ámbito, jornada, topes y redondeo</dd></div><div class="fila-resumen"><dt>Formación</dt><dd>Titulaciones, cursos, horas, relación y límites</dd></div><div class="fila-resumen"><dt>Otros méritos</dt><dd>Tipos definidos por las bases</dd></div><div class="fila-resumen"><dt>Garantía</dt><dd>Versión inmutable, simulación y validación antes de publicar</dd></div></dl><p class="nota-pendiente">Los valores visibles serían ejemplos; no se activará una regla sin bases e informe aplicables.</p>');
        break;
      case "seleccionar-necesidad": {
        if (!datosPanel.necesidades_llamamiento.some((item) => item.id === id)) {
          anunciar("La necesidad seleccionada no pertenece al ámbito visible");
          break;
        }
        asistenteLlamamientos.reiniciar(estado, id);
        estado.errorPropuesta = "";
        renderizar();
        anunciar("Necesidad de cobertura seleccionada");
        break;
      }
      case "ver-bolsa": {
        const bolsa = datosPanel.bolsas.find((item) => item.id === id);
        if (bolsa) abrirDialogo(bolsa.nombre, `<dl class="resumen-expediente"><div class="fila-resumen"><dt>Categoría</dt><dd>${escaparHTML(bolsa.categoria)}</dd></div><div class="fila-resumen"><dt>Integrantes</dt><dd>${numero(bolsa.integrantes)}</dd></div><div class="fila-resumen"><dt>Disponibles</dt><dd>${numero(bolsa.disponibles)}</dd></div><div class="fila-resumen"><dt>Cobertura</dt><dd>${porcentajeSeguro(bolsa.cobertura)}%</dd></div><div class="fila-resumen"><dt>Estado</dt><dd>${escaparHTML(bolsa.estado)}</dd></div></dl><p class="nota-informativa">Fuente: ${escaparHTML(etiquetaFuentePanel())}.</p>`);
        break;
      }
      case "solicitar-propuesta": {
        const resultado = await solicitarPropuestaLlamamiento();
        if (!resultado.ok) {
          anunciar(resultado.mensaje || "No se pudo obtener la propuesta");
          renderizar();
          break;
        }
        if (resultado.avanzar === true) estado.pasoLlamamiento = 2;
        else if (resultado.confirmacion) estado.pasoLlamamiento = 2;
        else estado.pasoLlamamiento = 1;
        renderizar();
        porId("contenido-principal")?.focus({ preventScroll: true });
        anunciar(resultado.mensaje || (resultado.sintetica
          ? "Propuesta sintética cargada sin consultar el servidor"
          : "Confirmación de propuesta recibida; detalle no disponible"));
        break;
      }
      case "siguiente-paso":
        if (!estado.modoPresentacion) {
          estado.pasoLlamamiento = 1;
          renderizar();
          anunciar("Detalle no disponible. La configuración del llamamiento permanece bloqueada.");
          break;
        }
        {
          const resultado = asistenteLlamamientos.avanzar(datosPanel, estado);
          renderizar();
          if (!resultado.ok) {
            document.querySelector("#errores-configuracion-llamamiento")?.focus({ preventScroll: true });
            anunciar(resultado.mensaje);
            break;
          }
          porId("contenido-principal")?.focus({ preventScroll: true });
          anunciar(resultado.mensaje);
        }
        break;
      case "anterior-paso":
        estado.pasoLlamamiento = Math.max(1, estado.pasoLlamamiento - 1);
        renderizar();
        anunciar(`Paso ${estado.pasoLlamamiento} del llamamiento`);
        break;
      case "ir-paso": {
        const paso = Number(boton.dataset.paso);
        if (!estado.modoPresentacion && (paso > 2
          || (paso === 2 && !estado.confirmacionPropuestaLlamamiento))) {
          estado.pasoLlamamiento = 1;
          renderizar();
          anunciar("Detalle no disponible. Los pasos posteriores no están conectados.");
          break;
        }
        if (paso >= 1 && paso <= 4 && paso <= estado.pasoLlamamiento) {
          estado.pasoLlamamiento = paso;
          renderizar();
        } else if (paso > estado.pasoLlamamiento) {
          anunciar("Complete primero el paso actual");
        }
        break;
      }
      case "validar-recorrido":
        abrirDialogo("Presentación comprobada", '<p class="nota-seguridad">Se ha revisado únicamente el recorrido sintético. No se ha creado expediente, enviado comunicación ni modificado dato alguno.</p>');
        break;
      case "preparar-llamamiento-demo": {
        if (!estado.modoPresentacion || estado.pasoLlamamiento !== 4
          || !operacionPermitida("emitir-llamamiento")) {
          abrirDialogo("Operación no autorizada", '<p class="nota-pendiente"><strong>No se ha preparado ningún llamamiento.</strong> El modo o perfil activo no concede esta simulación.</p>');
          anunciar("Preparación rechazada por falta de autorización explícita");
          break;
        }
        const preparacion = asistenteLlamamientos.prepararOperacion(datosPanel, estado);
        if (!preparacion.ok) {
          estado.erroresConfiguracionLlamamiento = preparacion.errores;
          estado.pasoLlamamiento = 3;
          renderizar();
          document.querySelector("#errores-configuracion-llamamiento")?.focus({ preventScroll: true });
          anunciar(preparacion.errores[0]?.mensaje || "Revise la configuración del llamamiento");
          break;
        }
        const descripcion = describirOperacionPresentacion("emitir-llamamiento", preparacion.objetivo);
        if (!descripcion) {
          detalleLimitacion("Preparación no disponible");
          break;
        }
        const pregunta = `Preparar un llamamiento exclusivamente DEMO.\n\nObjetivo: ${descripcion.objetivo}\nActor resuelto: ${descripcion.actor}\n\nNo se enviará ninguna comunicación ni se producirá un efecto administrativo. ¿Continuar?`;
        if (!confirmarOperacionPresentacion(pregunta)) {
          anunciar("Preparación DEMO cancelada; no se ha emitido recibo");
          break;
        }
        try {
          const recibo = ejecutarOperacionPresentacion(
            "emitir-llamamiento", preparacion.objetivo,
            "Preparación del asistente de llamamientos, sin envío real", preparacion.campos,
          );
          estado.reciboLlamamiento = recibo;
          renderizar();
          document.querySelector("[data-recibo-llamamiento]")?.focus({ preventScroll: true });
          anunciar(`Preparación DEMO confirmada con recibo ${recibo.referencia}; no se ha enviado nada`);
        } catch {
          estado.reciboLlamamiento = null;
          abrirDialogo("Preparación no realizada", '<p class="nota-pendiente"><strong>No se ha emitido un recibo de éxito.</strong> El permiso, la configuración o el estado del llamamiento no cumplen el contrato DEMO.</p>');
          anunciar("Preparación rechazada de forma segura; no se ha enviado nada");
        }
        break;
      }
      case "operacion-presentacion": {
        if (!estado.modoPresentacion) {
          detalleLimitacion(boton.textContent.trim() || "Acción administrativa");
          break;
        }
        const operacion = boton.dataset.operacion || "";
        const comando = boton.dataset.comando || "";
        if (comando !== operacion || !operacionPermitida(operacion)) {
          abrirDialogo("Operación no autorizada", '<p class="nota-pendiente"><strong>No se ha ejecutado ninguna actuación.</strong> El perfil actual no dispone de autorización explícita para este comando.</p>');
          anunciar("Operación rechazada por falta de autorización explícita");
          break;
        }
        const objetivo = boton.dataset.objetivo || "DEMO-SIN-OBJETIVO";
        const descripcion = describirOperacionPresentacion(operacion, objetivo);
        if (!descripcion) {
          detalleLimitacion("Operación no permitida");
          break;
        }
        const pregunta = `${descripcion.efecto}.\n\nObjetivo: ${descripcion.objetivo}\nActor: ${descripcion.actor}\n\n¿Desea ejecutar esta simulación sin efectos reales?`;
        if (!confirmarOperacionPresentacion(pregunta)) {
          anunciar("Simulación cancelada; no se ha modificado el estado en memoria");
          break;
        }
        try {
          const formulario = boton.closest("form.formulario-gobernado");
          if (formulario?.dataset.comando && formulario.dataset.comando !== comando) {
            throw new Error("el formulario no corresponde al comando");
          }
          if (formulario && typeof formulario.reportValidity === "function" && !formulario.reportValidity()) {
            anunciar("Revise los campos obligatorios antes de continuar");
            break;
          }
          const campos = formulario ? serializarCamposOperacion(new FormData(formulario)) : {};
          const recibo = ejecutarOperacionPresentacion(operacion, objetivo,
            boton.dataset.motivo || "Recorrido funcional de presentación", campos);
          renderizar();
          abrirDialogo("Actuación simulada", `<section class="recibo-presentacion"><p class="nota-seguridad"><strong>Simulación completada.</strong> No tiene efectos administrativos y desaparecerá al recargar.</p><dl class="resumen-expediente"><div class="fila-resumen"><dt>Recibo</dt><dd><code>${escaparHTML(recibo.referencia)}</code></dd></div><div class="fila-resumen"><dt>Actor</dt><dd>${escaparHTML(recibo.actor)}</dd></div><div class="fila-resumen"><dt>Instante</dt><dd><time datetime="${escaparHTML(recibo.instante)}">${escaparHTML(recibo.instante)}</time></dd></div><div class="fila-resumen"><dt>Objetivo</dt><dd>${escaparHTML(recibo.objetivo)}</dd></div><div class="fila-resumen"><dt>Resultado</dt><dd>${escaparHTML(recibo.resultado)}</dd></div><div class="fila-resumen"><dt>Campos aplicados</dt><dd>${escaparHTML(recibo.campos_aplicados)}</dd></div><div class="fila-resumen"><dt>Efectos reales</dt><dd>No</dd></div></dl></section>`);
          anunciar(`Simulación completada con recibo ${recibo.referencia}`);
        } catch {
          abrirDialogo("Actuación no realizada", '<p class="nota-pendiente"><strong>La operación se ha rechazado de forma segura.</strong> Los datos del formulario o el estado del expediente no cumplen el contrato vigente. No se ha emitido un recibo de éxito.</p>');
          anunciar("Operación rechazada de forma segura; revise el formulario y el estado del expediente");
        }
        break;
      }
      case "bloqueo-presentacion":
        abrirDialogo("Funcionalidad bloqueada", `<p class="nota-pendiente"><strong>No se ejecutará ninguna acción.</strong> ${escaparHTML(boton.dataset.motivo || "La capacidad productiva no está conectada ni autorizada.")}</p><p>El modo real permanece cerrado hasta recibir una capacidad positiva del servidor.</p>`);
        break;
      case "imprimir":
        window.print();
        break;
      case "ayuda":
        abrirDialogo("Ayuda del Portal del Empleado", renderizarContenidoAyuda());
        break;
      case "avisos":
        abrirDialogo("Avisos", `<ul>${datosPanel.avisos.map((aviso) => `<li>${escaparHTML(aviso.texto)}</li>`).join("") || "<li>No hay avisos accesibles.</li>"}</ul>`);
        break;
      case "exportar":
      case "aplicar-filtros":
      case "comparar-versiones":
      case "nueva-regla":
      case "detalle-regla":
        detalleLimitacion(boton.textContent.trim() || "Acción administrativa");
        break;
      default:
        break;
    }
  }

  function abrirMenuMovil() {
    document.body.dataset.menuAbierto = "true";
    porId("boton-menu").setAttribute("aria-expanded", "true");
    porId("velo-menu").hidden = false;
    document.querySelector(".portal-lateral button:not(:disabled)")?.focus();
  }

  function alternarPreferencia(nombre, boton) {
    const atributo = nombre === "texto" ? "textoGrande" : "contraste";
    const activo = document.body.dataset[atributo] !== "true";
    document.body.dataset[atributo] = String(activo);
    if (atributo === "textoGrande") {
      document.documentElement.dataset.textoGrande = String(activo);
    }
    boton.setAttribute("aria-pressed", String(activo));
    anunciar(activo ? `${nombre} activado` : `${nombre} desactivado`);
  }

  function restaurarPreferencias() {
    document.body.dataset.textoGrande = "false";
    document.documentElement.dataset.textoGrande = "false";
    document.body.dataset.contraste = "false";
    porId("boton-texto").setAttribute("aria-pressed", "false");
    porId("boton-contraste").setAttribute("aria-pressed", "false");
  }

  function instalar() {
    document.addEventListener("click", (evento) => {
      const botonVista = evento.target.closest("[data-vista]");
      if (botonVista && !botonVista.disabled) {
        evento.preventDefault();
        navegar(botonVista.dataset.vista);
        return;
      }
      const botonAccion = evento.target.closest("[data-accion]");
      if (botonAccion) void manejarAccion(botonAccion);
    });
    document.addEventListener("submit", (evento) => {
      const formulario = evento.target.closest?.("form[data-filtro]");
      if (!formulario) return;
      evento.preventDefault();
      try {
        const tipo = formulario.dataset.filtro;
        const campos = serializarCamposFiltro(tipo, new FormData(formulario));
        estado.filtros[tipo] = campos;
        renderizar();
        const resultado = document.querySelector(`[data-total-filtro="${tipo}"]`);
        const total = Number(resultado?.dataset.total || 0);
        anunciar(`${total} resultados tras aplicar los filtros`);
      } catch {
        abrirDialogo("Filtros no aplicados", '<p class="nota-pendiente">El formulario de filtros no cumple el contrato cerrado. No se ha modificado la vista.</p>');
        anunciar("Filtros rechazados de forma segura");
      }
    });
    document.addEventListener("input", (evento) => {
      const control = evento.target.closest?.("[data-llamamiento-campo]");
      if (!control || estado.vista !== "llamamientos" || estado.pasoLlamamiento !== 3) return;
      try {
        asistenteLlamamientos.actualizarCampo(estado, control.dataset.llamamientoCampo, control.value);
      } catch {
        anunciar("El campo del llamamiento no admite ese valor");
      }
    });
    porId("boton-menu").addEventListener("click", () => {
      if (document.body.dataset.menuAbierto === "true") cerrarMenuMovil({ restaurarFoco: true });
      else abrirMenuMovil();
    });
    porId("velo-menu").addEventListener("click", () => cerrarMenuMovil({ restaurarFoco: true }));
    porId("boton-texto").addEventListener("click", (evento) => alternarPreferencia("texto", evento.currentTarget));
    porId("boton-contraste").addEventListener("click", (evento) => alternarPreferencia("contraste", evento.currentTarget));
    window.addEventListener("hashchange", () => {
      const vista = vistaDesdeHash();
      if (vista !== estado.vista) {
        navegar(vista, { enfocar: false });
      }
    });
    window.addEventListener("keydown", (evento) => {
      if (document.body.dataset.menuAbierto === "true" && evento.key === "Tab") {
        contenerTabulacionMenu(evento, document.querySelector(".portal-lateral"), document.activeElement);
        return;
      }
      if (evento.key === "Escape" && document.body.dataset.menuAbierto === "true") {
        evento.preventDefault();
        cerrarMenuMovil({ restaurarFoco: true });
      }
    });
  }

  return { instalar, restaurarPreferencias };
}
