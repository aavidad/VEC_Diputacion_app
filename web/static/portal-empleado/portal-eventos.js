/**
 * Controladores de interacción del Portal del Empleado.
 *
 * Recibe sus dependencias desde `portal.js`: no conoce repositorios ni decide
 * negocio. Las acciones sin comando de servidor compuesto permanecen
 * informativas y nunca producen efectos administrativos en el navegador.
 */
export function crearControladorPortal(dependencias) {
  const {
    anunciar,
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
        estado.pasoLlamamiento = 1;
        estado.propuestaLlamamiento = null;
        estado.confirmacionPropuestaLlamamiento = null;
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
        estado.necesidadSeleccionada = id;
        estado.propuestaLlamamiento = null;
        estado.confirmacionPropuestaLlamamiento = null;
        estado.errorPropuesta = "";
        estado.pasoLlamamiento = 1;
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
        else estado.pasoLlamamiento = 1;
        renderizar();
        porId("contenido-principal")?.focus({ preventScroll: true });
        anunciar(resultado.mensaje || (resultado.sintetica
          ? "Propuesta sintética cargada sin consultar el servidor"
          : "Confirmación de propuesta recibida; detalle no disponible"));
        break;
      }
      case "siguiente-paso":
        if (!estado.modoPresentacion || estado.propuestaLlamamiento?.demostracion !== true) {
          estado.pasoLlamamiento = 1;
          renderizar();
          anunciar("Detalle no disponible. La configuración del llamamiento permanece bloqueada.");
          break;
        }
        estado.pasoLlamamiento = Math.min(4, estado.pasoLlamamiento + 1);
        renderizar();
        porId("contenido-principal")?.focus({ preventScroll: true });
        anunciar(`Paso ${estado.pasoLlamamiento} del llamamiento`);
        break;
      case "anterior-paso":
        estado.pasoLlamamiento = Math.max(1, estado.pasoLlamamiento - 1);
        renderizar();
        anunciar(`Paso ${estado.pasoLlamamiento} del llamamiento`);
        break;
      case "ir-paso": {
        const paso = Number(boton.dataset.paso);
        if (!estado.modoPresentacion && paso > 1) {
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
      case "operacion-presentacion": {
        if (!estado.modoPresentacion) {
          detalleLimitacion(boton.textContent.trim() || "Acción administrativa");
          break;
        }
        const operacion = boton.dataset.operacion || "";
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
        const recibo = ejecutarOperacionPresentacion(operacion, objetivo, boton.dataset.motivo || "Recorrido funcional de presentación");
        renderizar();
        abrirDialogo("Actuación simulada", `<section class="recibo-presentacion"><p class="nota-seguridad"><strong>Simulación completada.</strong> No tiene efectos administrativos y desaparecerá al recargar.</p><dl class="resumen-expediente"><div class="fila-resumen"><dt>Recibo</dt><dd><code>${escaparHTML(recibo.referencia)}</code></dd></div><div class="fila-resumen"><dt>Actor</dt><dd>${escaparHTML(recibo.actor)}</dd></div><div class="fila-resumen"><dt>Instante</dt><dd><time datetime="${escaparHTML(recibo.instante)}">${escaparHTML(recibo.instante)}</time></dd></div><div class="fila-resumen"><dt>Objetivo</dt><dd>${escaparHTML(recibo.objetivo)}</dd></div><div class="fila-resumen"><dt>Resultado</dt><dd>${escaparHTML(recibo.resultado)}</dd></div><div class="fila-resumen"><dt>Efectos reales</dt><dd>No</dd></div></dl></section>`);
        anunciar(`Simulación completada con recibo ${recibo.referencia}`);
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
    porId("boton-menu").addEventListener("click", () => {
      if (document.body.dataset.menuAbierto === "true") cerrarMenuMovil();
      else abrirMenuMovil();
    });
    porId("velo-menu").addEventListener("click", cerrarMenuMovil);
    porId("boton-texto").addEventListener("click", (evento) => alternarPreferencia("texto", evento.currentTarget));
    porId("boton-contraste").addEventListener("click", (evento) => alternarPreferencia("contraste", evento.currentTarget));
    window.addEventListener("hashchange", () => {
      const vista = vistaDesdeHash();
      if (vista !== estado.vista) {
        estado.vista = vista;
        renderizar();
      }
    });
    window.addEventListener("keydown", (evento) => {
      if (evento.key === "Escape" && document.body.dataset.menuAbierto === "true") cerrarMenuMovil();
    });
  }

  return { instalar, restaurarPreferencias };
}
