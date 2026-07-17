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
    escaparHTML,
    estado,
    etiquetaFuentePanel,
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
        estado.errorPropuesta = "";
        estado.claveIdempotenciaPropuesta = "";
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
        estado.errorPropuesta = "";
        estado.claveIdempotenciaPropuesta = "";
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
        estado.pasoLlamamiento = 2;
        renderizar();
        porId("contenido-principal")?.focus({ preventScroll: true });
        anunciar(resultado.sintetica ? "Propuesta sintética cargada sin consultar el servidor" : "Propuesta recibida del servidor");
        break;
      }
      case "siguiente-paso":
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
        if (paso >= 1 && paso <= 4 && paso <= estado.pasoLlamamiento) {
          estado.pasoLlamamiento = paso;
          renderizar();
        } else if (paso > estado.pasoLlamamiento) {
          anunciar("Complete primero el paso actual");
        }
        break;
      }
      case "validar-recorrido":
        abrirDialogo("Recorrido validado", `<p class="nota-seguridad">La preparación se ha revisado sin enviar comunicaciones ni modificar datos.</p><p>${escaparHTML(notaOperacionNoCompuesta())}</p>`);
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

  function cerrarMenuMovil() {
    delete document.body.dataset.menuAbierto;
    porId("boton-menu")?.setAttribute("aria-expanded", "false");
    if (porId("velo-menu")) porId("velo-menu").hidden = true;
  }

  function alternarPreferencia(nombre, boton) {
    const atributo = nombre === "texto" ? "textoGrande" : "contraste";
    const activo = document.body.dataset[atributo] !== "true";
    document.body.dataset[atributo] = String(activo);
    boton.setAttribute("aria-pressed", String(activo));
    try { localStorage.setItem(`vec_portal_${nombre}`, String(activo)); } catch { /* preferencia no crítica */ }
    anunciar(activo ? `${nombre} activado` : `${nombre} desactivado`);
  }

  function restaurarPreferencias() {
    try {
      const texto = localStorage.getItem("vec_portal_texto") === "true";
      const contraste = localStorage.getItem("vec_portal_contraste") === "true";
      document.body.dataset.textoGrande = String(texto);
      document.body.dataset.contraste = String(contraste);
      porId("boton-texto").setAttribute("aria-pressed", String(texto));
      porId("boton-contraste").setAttribute("aria-pressed", String(contraste));
    } catch { /* el portal funciona sin almacenamiento local */ }
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
