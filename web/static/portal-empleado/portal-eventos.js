/**
 * Controladores de interacción del Portal del Empleado.
 *
 * Recibe sus dependencias desde `portal.js`: no conoce repositorios ni decide
 * negocio. Las acciones sin comando de servidor compuesto permanecen
 * informativas y nunca producen efectos administrativos en el navegador.
 */
import { traducirPortal } from "./portal-i18n.js?v=20260928-rrhh-i18n-unificada-v1";
import { validarAvisosPortal } from "./portal-contrato.js?v=20260925-sin-demo2-v1";
import { LOCALIZACION_ACTUAL } from "../comun/idioma.js";

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

function usuarioPrefiereMovimientoReducido() {
  return globalThis.matchMedia?.("(prefers-reduced-motion: reduce)")?.matches === true;
}

/**
 * Conserva el foco accesible en un resultado dinámico y lo lleva al área
 * visible. La animación se desactiva cuando el usuario solicita reducir el
 * movimiento.
 */
export function enfocarYMostrarResultado(elemento, {
  movimientoReducido = usuarioPrefiereMovimientoReducido(),
} = {}) {
  if (!elemento || typeof elemento.focus !== "function") return false;
  elemento.focus({ preventScroll: true });
  elemento.scrollIntoView?.({
    behavior: movimientoReducido ? "auto" : "smooth",
    block: "center",
    inline: "nearest",
  });
  return true;
}

export function restaurarFocoTrasReintentoBorradores(raiz = globalThis.document) {
  const tarjeta = raiz?.querySelector?.('[data-modulo-catalogo="bolsa"]');
  if (!tarjeta || typeof tarjeta.focus !== "function") return false;
  const control = tarjeta.querySelector?.(
    '[data-accion="reintentar-borradores"], [data-vista="elaboracion"]',
  );
  (control && typeof control.focus === "function" ? control : tarjeta)
    .focus({ preventScroll: true });
  return true;
}

export function renderizarAvisosNavegables(avisos, escaparHTML, traducir = traducirPortal) {
  const avisosValidados = validarAvisosPortal(avisos);
  if (avisosValidados.length === 0) return `<p>${escaparHTML(traducir("avisos_vacio"))}</p>`;
  return `<ul class="lista-avisos-navegables">${avisosValidados.map((aviso, indice) => {
    const destino = aviso.destino;
    if (!destino) return `<li><p>${escaparHTML(aviso.texto)}</p></li>`;
    const contexto = traducir("avisos_ir_a", { destino: destino.etiqueta });
    if (destino.estado === "pendiente") {
      return `<li><p>${escaparHTML(aviso.texto)}</p><button type="button" class="boton-secundario" disabled aria-disabled="true" title="${escaparHTML(traducir("avisos_destino_pendiente"))}">${escaparHTML(contexto)} · ${escaparHTML(traducir("avisos_pendiente"))}</button></li>`;
    }
    return `<li><p>${escaparHTML(aviso.texto)}</p><button type="button" class="boton-secundario" data-aviso-destino="${indice}" aria-label="${escaparHTML(contexto)}">${escaparHTML(contexto)}</button></li>`;
  }).join("")}</ul>`;
}

export function instalarDestinosAvisos(contenedor, avisos, { cerrar, navegar, anunciar, traducir = traducirPortal }) {
  const avisosValidados = validarAvisosPortal(avisos);
  const manejarClick = (evento) => {
    const boton = evento.target?.closest?.("[data-aviso-destino]");
    if (!boton || !contenedor?.contains?.(boton)) return;
    const indice = Number(boton.dataset.avisoDestino);
    if (!Number.isSafeInteger(indice) || indice < 0 || indice >= avisosValidados.length) return;
    const destino = avisosValidados[indice].destino;
    if (destino.estado !== "disponible") return;
    evento.preventDefault?.();
    cerrar();
    navegar(destino.vista, destino.referencia ? { referencia: destino.referencia } : {});
    anunciar(traducir("avisos_anuncio", { destino: destino.etiqueta }));
  };
  contenedor?.addEventListener?.("click", manejarClick);
  return () => contenedor?.removeEventListener?.("click", manejarClick);
}

export function crearControladorPortal(dependencias) {
  const {
    anunciar,
    cargarFuenteDatos,
    cerrarMenuMovil,
    comprobarDisponibilidadBorradores,
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
  const traducir = typeof dependencias.traducir === "function"
    ? dependencias.traducir
    : traducirPortal;
  const numeroLocal = (valor) => new Intl.NumberFormat(LOCALIZACION_ACTUAL).format(Number(valor) || 0);
  const porcentajeLocal = (valor) => new Intl.NumberFormat(LOCALIZACION_ACTUAL, { style: "percent", maximumFractionDigits: 1 }).format(porcentajeSeguro(valor) / 100);
  const htmlTraducido = (clave, variables) => escaparHTML(traducir(clave, variables));

  let limpiarDialogo = null;

  function limpiarContenidoDialogo() {
    limpiarDialogo?.();
    limpiarDialogo = null;
  }

  const dialogoDetalle = porId("dialogo-detalle");
  dialogoDetalle?.addEventListener?.("close", limpiarContenidoDialogo);
  dialogoDetalle?.addEventListener?.("cancel", limpiarContenidoDialogo);

  function cerrarDialogo() {
    const dialogo = porId("dialogo-detalle");
    limpiarContenidoDialogo();
    if (typeof dialogo.close === "function" && dialogo.open) dialogo.close();
    else dialogo.removeAttribute?.("open");
  }

  function abrirDialogo(titulo, contenido, instalar = null) {
    const dialogo = porId("dialogo-detalle");
    limpiarContenidoDialogo();
    porId("titulo-dialogo").textContent = titulo;
    const contenedor = porId("contenido-dialogo");
    contenedor.innerHTML = contenido;
    if (typeof dialogo.showModal === "function") dialogo.showModal();
    else dialogo.setAttribute("open", "");
    if (typeof instalar === "function") {
      limpiarDialogo = instalar({ contenedor, documento: document, navegar, anunciar, cerrar: cerrarDialogo }) || null;
    }
  }

  function detalleLimitacion(titulo) {
    abrirDialogo(titulo, `<p class="nota-pendiente">${escaparHTML(notaOperacionNoCompuesta())}</p><p>${htmlTraducido("operacion_requisitos")}</p>`);
  }

  function reiniciarLlamamiento(necesidadId) {
    estado.pasoLlamamiento = 1;
    estado.necesidadSeleccionada = necesidadId;
    estado.confirmacionPropuestaLlamamiento = null;
    estado.errorPropuesta = "";
  }

  async function manejarAccion(boton) {
    const accion = boton.dataset.accion;
    const id = boton.dataset.id;
    const datosPanel = obtenerDatosPanel();
    switch (accion) {
      case "recargar-fuente": {
        estado.errorFuente = "";
        estado.fuenteLista = false;
        renderizar();
        // El botón de reintento desaparece al repintar: el foco pasa al
        // contenido principal. cargarFuenteDatos pinta su propio resultado sin
        // volver a montar la vista que ya montó durante la carga.
        const activo = document.activeElement;
        if (!activo || activo === document.body || activo.isConnected === false) {
          porId("contenido-principal")?.focus?.({ preventScroll: true });
        }
        cargarFuenteDatos().catch(() => {
          estado.errorFuente = traducir("fuente_reintento_error");
          renderizar();
        });
        break;
      }
      case "reintentar-borradores": {
        const disponible = await comprobarDisponibilidadBorradores({ forzar: true });
        anunciar(disponible
          ? traducir("anuncio_acceso_borradores_comprobado")
          : traducir("anuncio_acceso_borradores_no_disponible"));
        renderizar();
        queueMicrotask(() => restaurarFocoTrasReintentoBorradores());
        break;
      }
      case "nuevo-llamamiento":
        reiniciarLlamamiento(datosPanel.necesidades_llamamiento[0]?.id || "");
        navegar("llamamientos");
        break;
      case "nueva-bolsa":
        abrirDialogo(traducir("bolsa_nueva_titulo"), `<ol><li>${htmlTraducido("bolsa_nueva_identificacion")}</li><li>${htmlTraducido("bolsa_nueva_bases")}</li><li>${htmlTraducido("bolsa_nueva_baremo")}</li><li>${htmlTraducido("bolsa_nueva_calendario")}</li><li>${htmlTraducido("bolsa_nueva_firmas")}</li></ol><p class="nota-pendiente">${escaparHTML(notaOperacionNoCompuesta())}</p>`);
        break;
      case "seleccionar-elaboracion":
        estado.elaboracionSeleccionada = id;
        renderizar();
        anunciar(traducir("elaboracion_seleccionada"));
        break;
      case "configurar-bases":
        abrirDialogo(traducir("bases_titulo"), `<dl class="resumen-expediente"><div class="fila-resumen"><dt>${htmlTraducido("bases_experiencia")}</dt><dd>${htmlTraducido("bases_experiencia_detalle")}</dd></div><div class="fila-resumen"><dt>${htmlTraducido("bases_formacion")}</dt><dd>${htmlTraducido("bases_formacion_detalle")}</dd></div><div class="fila-resumen"><dt>${htmlTraducido("bases_otros")}</dt><dd>${htmlTraducido("bases_otros_detalle")}</dd></div><div class="fila-resumen"><dt>${htmlTraducido("bases_garantia")}</dt><dd>${htmlTraducido("bases_garantia_detalle")}</dd></div></dl><p class="nota-pendiente">${htmlTraducido("bases_ejemplos")}</p>`);
        break;
      case "seleccionar-necesidad": {
        if (!datosPanel.necesidades_llamamiento.some((item) => item.id === id)) {
          anunciar(traducir("necesidad_fuera_ambito"));
          break;
        }
        reiniciarLlamamiento(id);
        renderizar();
        anunciar(traducir("necesidad_seleccionada"));
        break;
      }
      case "ver-bolsa": {
        const bolsa = datosPanel.bolsas.find((item) => item.id === id);
        if (bolsa) abrirDialogo(bolsa.nombre, `<dl class="resumen-expediente"><div class="fila-resumen"><dt>${htmlTraducido("bolsa_categoria")}</dt><dd>${escaparHTML(bolsa.categoria)}</dd></div><div class="fila-resumen"><dt>${htmlTraducido("bolsa_integrantes")}</dt><dd>${numeroLocal(bolsa.integrantes)}</dd></div><div class="fila-resumen"><dt>${htmlTraducido("bolsa_disponibles")}</dt><dd>${numeroLocal(bolsa.disponibles)}</dd></div><div class="fila-resumen"><dt>${htmlTraducido("bolsa_cobertura")}</dt><dd>${porcentajeLocal(bolsa.cobertura)}</dd></div><div class="fila-resumen"><dt>${htmlTraducido("bolsa_estado")}</dt><dd>${escaparHTML(bolsa.estado)}</dd></div></dl><p class="nota-informativa">${htmlTraducido("bolsa_fuente", { fuente: etiquetaFuentePanel() })}</p>`);
        break;
      }
      case "solicitar-propuesta": {
        const resultado = await solicitarPropuestaLlamamiento();
        if (!resultado.ok) {
          anunciar(resultado.mensaje || traducir("propuesta_error"));
          renderizar();
          break;
        }
        estado.pasoLlamamiento = resultado.confirmacion ? 2 : 1;
        renderizar();
        porId("contenido-principal")?.focus({ preventScroll: true });
        anunciar(resultado.mensaje || (resultado.confirmacion
          ? traducir("propuesta_confirmada_sin_detalle")
          : traducir("llamamiento_detalle_bloqueado")));
        break;
      }
      case "siguiente-paso":
        estado.pasoLlamamiento = 1;
        renderizar();
        anunciar(traducir("llamamiento_detalle_bloqueado"));
        break;
      case "anterior-paso":
        estado.pasoLlamamiento = Math.max(1, estado.pasoLlamamiento - 1);
        renderizar();
        anunciar(traducir("llamamiento_paso", { paso: numeroLocal(estado.pasoLlamamiento) }));
        break;
      case "ir-paso": {
        const paso = Number(boton.dataset.paso);
        if (paso > 2 || (paso === 2 && !estado.confirmacionPropuestaLlamamiento)) {
          estado.pasoLlamamiento = 1;
          renderizar();
          anunciar(traducir("llamamiento_pasos_no_conectados"));
          break;
        }
        if (paso >= 1 && paso <= 2 && paso <= estado.pasoLlamamiento) {
          estado.pasoLlamamiento = paso;
          renderizar();
        } else if (paso > estado.pasoLlamamiento) {
          anunciar(traducir("llamamiento_complete_paso"));
        }
        break;
      }
      case "bloqueo-presentacion":
        abrirDialogo(traducir("bloqueo_titulo"), `<p class="nota-pendiente"><strong>${htmlTraducido("bloqueo_sin_accion")}</strong> ${escaparHTML(boton.dataset.motivo || traducir("bloqueo_capacidad_no_conectada"))}</p>`);
        break;
      case "imprimir":
        window.print();
        break;
      case "ayuda": {
        const ayuda = renderizarContenidoAyuda();
        if (typeof ayuda === "object" && ayuda !== null && "contenido" in ayuda) {
          abrirDialogo(ayuda.titulo || traducir("ayuda_titulo"), ayuda.contenido, ayuda.instalar);
        } else {
          abrirDialogo(traducir("ayuda_titulo"), ayuda);
        }
        break;
      }
      case "avisos":
        try {
          abrirDialogo(traducir("avisos_titulo"), renderizarAvisosNavegables(datosPanel.avisos, escaparHTML, traducir),
            ({ contenedor, cerrar, navegar: navegarAviso, anunciar: anunciarAviso }) => instalarDestinosAvisos(
              contenedor, datosPanel.avisos, { cerrar, navegar: navegarAviso, anunciar: anunciarAviso, traducir },
            ));
        } catch {
          abrirDialogo(traducir("avisos_titulo"), `<p class="nota-pendiente">${htmlTraducido("avisos_contrato_invalido")}</p>`);
          anunciar(traducir("avisos_rechazados"));
        }
        break;
      case "exportar":
      case "aplicar-filtros":
      case "comparar-versiones":
      case "nueva-regla":
      case "detalle-regla":
        detalleLimitacion(boton.textContent.trim() || traducir("accion_administrativa"));
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
    anunciar(traducir(`preferencia_${nombre}_${activo ? "activado" : "desactivado"}`));
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
        const opciones = {};
        if (botonVista.dataset.ctExpVista) {
          opciones.subvista = botonVista.dataset.ctExpVista;
        }
        if (botonVista.dataset.ctExpAbrirInicio) {
          opciones.expedienteRef = botonVista.dataset.ctExpAbrirInicio;
        }
        if (botonVista.dataset.ctExpFiltroEstado || botonVista.dataset.ctExpFiltroFase) {
          opciones.filtros = {
            texto: "",
            estado: botonVista.dataset.ctExpFiltroEstado ?? "",
            fase: botonVista.dataset.ctExpFiltroFase ?? "",
          };
        }
        navegar(botonVista.dataset.vista, opciones);
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
        anunciar(traducir(total === 1 ? "filtros_resultado_uno" : "filtros_resultado_varios", { total: numeroLocal(total) }));
      } catch {
        abrirDialogo(traducir("filtros_error_titulo"), `<p class="nota-pendiente">${htmlTraducido("filtros_error_detalle")}</p>`);
        anunciar(traducir("filtros_rechazados"));
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
      if (dialogoDetalle?.open) return;
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
