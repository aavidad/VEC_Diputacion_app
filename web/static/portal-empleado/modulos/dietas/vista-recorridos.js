import { crearTraductorDietas, MENSAJES_DIETAS_ES } from "./i18n.js";
import { montarVistaBorradoresPropios } from "./vista-borradores-propios.js";
import { obtenerAtlasSinteticoRRHH } from "../../datos-sinteticos-rrhh.js";

const ETAPAS = Object.freeze([
  ["solicitante", "recorridos_solicitante"],
  ["jefatura", "recorridos_jefatura"],
  ["gestion", "recorridos_gestion"],
]);

const COMISIONES_PRESENTACION = Object.freeze([
  Object.freeze({ referencia: "DIE-2026-0084", fecha: "19/06/2026 · 08:00–14:30", motivo: "Reunión técnica de coordinación", localidades: "Granada · Albolote · Granada", kilometros: "21,6 km", dietas: "Manutención · 22,78 €", gastos: "Kilometraje · 5,62 €", justificantes: "1 justificante declarado", total: "28,40 €", estado: "Pendiente de jefatura", incidencia: "Sin incidencias declaradas" }),
  Object.freeze({ referencia: "DIE-2026-0091", fecha: "21/06/2026 · 07:30–15:15", motivo: "Visita técnica de obra", localidades: "Granada · Motril · Granada", kilometros: "140,8 km", dietas: "Sin dieta declarada", gastos: "Kilometraje · 36,61 €", justificantes: "Pendiente de adjuntar", total: "36,61 €", estado: "Borrador", incidencia: "Falta justificante de comisión" }),
  Object.freeze({ referencia: "DIE-2026-0073", fecha: "27/05/2026 · 08:15–17:00", motivo: "Inspección de obra provincial", localidades: "Granada · Guadix · Granada", kilometros: "107,2 km", dietas: "Manutención · 34,01 €", gastos: "Kilometraje · 27,87 €", justificantes: "2 justificantes declarados", total: "61,88 €", estado: "Ejemplo liquidado", incidencia: "Seguimiento del pago pendiente de conexión" }),
]);

function elemento(documento, etiqueta, texto = "") {
  const resultado = documento.createElement(etiqueta);
  if (texto) resultado.textContent = texto;
  return resultado;
}
function sigueMontada(contenedor, raiz) {
  return contenedor.querySelector?.("[data-dietas-recorridos]") === raiz;
}
function retirar(contenedor, raiz) {
  if (!sigueMontada(contenedor, raiz)) return;
  if (typeof raiz.remove === "function") raiz.remove();
  else contenedor.removeChild?.(raiz);
}
function botonPendiente(documento, t, clave) {
  const boton = elemento(documento, "button", t(clave));
  boton.type = "button";
  boton.className = "boton-secundario";
  boton.disabled = true;
  return boton;
}

function tablaPendiente(documento, t, columnas, etiqueta) {
  const region = elemento(documento, "div");
  region.className = "tabla-contenedor";
  region.setAttribute("role", "region");
  region.setAttribute("tabindex", "0");
  region.setAttribute("aria-label", t(etiqueta));
  const tabla = elemento(documento, "table");
  tabla.className = "tabla-datos";
  const filaCabecera = elemento(documento, "tr");
  columnas.forEach((clave) => {
    const celda = elemento(documento, "th", t(clave));
    celda.setAttribute("scope", "col");
    filaCabecera.append(celda);
  });
  const thead = elemento(documento, "thead");
  thead.append(filaCabecera);
  const filaVacia = elemento(documento, "tr");
  const celdaVacia = elemento(
    documento,
    "td",
    t("recorridos_pendiente_conexion"),
  );
  celdaVacia.colSpan = columnas.length;
  filaVacia.append(celdaVacia);
  const tbody = elemento(documento, "tbody");
  tbody.append(filaVacia);
  tabla.append(thead, tbody);
  region.append(tabla);
  return region;
}

function tablaPresentacion(documento, t, columnas, seleccionada, incluirUnidad = false) {
  const region = elemento(documento, "div");
  region.className = "tabla-contenedor dietas-presentacion-tabla";
  region.setAttribute("role", "region");
  region.setAttribute("tabindex", "0");
  region.setAttribute("aria-label", t("recorridos_datos_ficticios"));
  const tabla = elemento(documento, "table"); tabla.className = "tabla-datos";
  const cabecera = elemento(documento, "tr");
  columnas.forEach((clave) => { const celda = elemento(documento, "th", t(clave)); celda.scope = "col"; cabecera.append(celda); });
  const thead = elemento(documento, "thead"); thead.append(cabecera);
  const cuerpo = elemento(documento, "tbody");
  COMISIONES_PRESENTACION.forEach((comision) => {
    const fila = elemento(documento, "tr"); fila.dataset.dietasSeleccionada = String(comision.referencia === seleccionada); fila.dataset.dietasComisionRef = comision.referencia;
    const referencia = elemento(documento, "th"); referencia.scope = "row";
    const elegir = elemento(documento, "button", comision.referencia); elegir.type = "button"; elegir.className = "enlace-tabla"; elegir.dataset.dietasSeleccionarComision = comision.referencia; elegir.setAttribute("aria-current", String(comision.referencia === seleccionada)); referencia.append(elegir); fila.append(referencia);
    [comision.fecha, comision.motivo, comision.total].forEach((valor) => fila.append(elemento(documento, "td", valor)));
    const estado = elemento(documento, "td"); const chip = elemento(documento, "span", comision.estado); chip.className = "estado-chip aviso"; estado.append(chip); fila.append(estado);
    if (incluirUnidad) fila.append(elemento(documento, "td", obtenerAtlasSinteticoRRHH().unidad.nombre_visible));
    cuerpo.append(fila);
  });
  tabla.append(thead, cuerpo); region.append(tabla); return region;
}

function detallePresentacion(documento, t, comision, rol) {
  const seccion = elemento(documento, "section"); seccion.className = "panel dietas-presentacion-detalle"; seccion.dataset.dietasDetallePresentacion = comision.referencia; seccion.setAttribute("tabindex", "-1");
  const atlas = obtenerAtlasSinteticoRRHH();
  seccion.append(elemento(documento, "h3", `${t("recorridos_detalle_solicitud")} · ${comision.referencia}`));
  const datos = elemento(documento, "dl"); datos.className = "dietas-datos-clave";
  [["recorridos_persona", atlas.persona_principal.nombre_visible], ["recorridos_fechas", comision.fecha], ["motivo", comision.motivo], ["recorridos_localidades", comision.localidades], ["recorridos_kilometros", comision.kilometros], ["recorridos_justificantes_declarados", comision.justificantes]].forEach(([clave, valor]) => { const grupo = elemento(documento, "div"); grupo.append(elemento(documento, "dt", t(clave)), elemento(documento, "dd", valor)); datos.append(grupo); });
  const conceptos = elemento(documento, "section"); conceptos.className = "dietas-presentacion-conceptos"; conceptos.append(elemento(documento, "h4", t("recorridos_conceptos_declarados")), elemento(documento, "p", `${comision.dietas} · ${comision.gastos}`), elemento(documento, "p", `${t("recorridos_total_ejemplo")}: ${comision.total}`));
  seccion.append(datos, conceptos);
  if (rol === "jefatura") ["recorridos_validar", "recorridos_devolver", "recorridos_rechazar"].forEach((clave) => seccion.append(botonPendiente(documento, t, clave)));
  if (rol === "gestion") ["recorridos_revisar_conceptos", "recorridos_liquidar", "recorridos_seguir_pago"].forEach((clave) => seccion.append(botonPendiente(documento, t, clave)));
  return seccion;
}

function detallesPresentacion(documento, t, seleccionada, rol) {
  const grupo = elemento(documento, "div"); grupo.className = "dietas-presentacion-detalles";
  COMISIONES_PRESENTACION.forEach((comision) => { const detalle = detallePresentacion(documento, t, comision, rol); detalle.hidden = comision.referencia !== seleccionada; grupo.append(detalle); });
  return grupo;
}

function resumenPresentacion(documento, t) {
  const seccion = elemento(documento, "section");
  seccion.className = "rejilla-kpi dietas-kpi";
  seccion.setAttribute("aria-label", t("recorridos_resumen"));
  [
    ["COM", "recorridos_kpi_comisiones", "3"],
    ["REV", "recorridos_kpi_revision", "1"],
    ["EUR", "recorridos_kpi_declarado", "126,89 €"],
    ["PAG", "recorridos_kpi_pago", t("recorridos_no_acreditado")],
  ].forEach(([sigla, clave, valor]) => {
    const tarjeta = elemento(documento, "article");
    tarjeta.className = "tarjeta-kpi";
    const icono = elemento(documento, "span", sigla);
    icono.className = "icono-kpi";
    icono.setAttribute("aria-hidden", "true");
    const contenido = elemento(documento, "div");
    const cifra = elemento(documento, "strong", valor);
    cifra.className = "valor-kpi";
    const etiqueta = elemento(documento, "span", t(clave));
    etiqueta.className = "etiqueta-kpi";
    contenido.append(cifra, etiqueta);
    tarjeta.append(icono, contenido);
    seccion.append(tarjeta);
  });
  return seccion;
}

function formularioGastos(documento, t) {
  const seccion = elemento(documento, "section");
  seccion.className = "panel dietas-recorridos-formulario";
  seccion.append(
    elemento(documento, "h3", t("recorridos_gastos_justificantes")),
  );
  const formulario = elemento(documento, "form");
  formulario.dataset.dietasGastosPendientes = "";
  [
    ["fecha_inicio", "fecha_inicio", "date"],
    ["hora_inicio", "recorridos_hora_inicio", "time"],
    ["fecha_fin", "fecha_fin", "date"],
    ["hora_fin", "recorridos_hora_fin", "time"],
    ["motivo", "motivo", "text"],
    ["localidades", "recorridos_localidades", "text"],
    ["pais", "recorridos_pais", "text"],
    ["nivel_detalle", "recorridos_nivel_detalle", "text"],
    ["descripcion", "recorridos_descripcion", "text"],
    ["fecha", "fecha", "date"],
    ["importe", "recorridos_importe_declarado", "number"],
    ["moneda", "recorridos_moneda", "text"],
    ["documento", "recorridos_documento", "file"],
  ].forEach(([nombre, etiqueta, tipo]) => {
    const label = elemento(documento, "label", t(etiqueta));
    const input = elemento(documento, "input");
    input.name = nombre;
    input.type = tipo;
    input.disabled = true;
    label.append(input);
    formulario.append(label);
  });
  formulario.append(botonPendiente(documento, t, "recorridos_anadir_concepto"));
  const dietas = tablaPendiente(
    documento,
    t,
    [
      "recorridos_tipo_dieta",
      "recorridos_intervalo",
      "recorridos_importe_declarado",
      "recorridos_aceptar",
    ],
    "recorridos_gastos_justificantes",
  );
  const kilometraje = elemento(documento, "section");
  kilometraje.append(
    elemento(documento, "h4", t("kilometraje")),
    botonPendiente(documento, t, "recorridos_vehiculo_propio"),
    botonPendiente(documento, t, "recorridos_salida"),
    botonPendiente(documento, t, "recorridos_llegada"),
    botonPendiente(documento, t, "recorridos_anadir_ruta"),
    elemento(documento, "p", t("recorridos_pendiente_conexion")),
  );
  const otros = elemento(documento, "section");
  otros.append(
    elemento(documento, "h4", t("recorridos_otro_gasto")),
    botonPendiente(documento, t, "recorridos_otro_medio"),
    botonPendiente(documento, t, "recorridos_gasto_justificado"),
    botonPendiente(documento, t, "recorridos_anadir_gasto"),
  );
  seccion.append(
    formulario,
    dietas,
    kilometraje,
    otros,
    elemento(documento, "p", t("recorridos_pendiente_conexion")),
  );
  return seccion;
}

function panelSolicitante(documento, t, areaBorradores, areaItinerario, seleccionada) {
  const panel = elemento(documento, "section");
  panel.className = "panel dietas-recorridos-principal";
  panel.dataset.dietasPanelEtapa = "solicitante";
  panel.append(
    elemento(documento, "h3", t("recorridos_solicitante")),
    resumenPresentacion(documento, t),
  );
  // El itinerario es la tarea principal de la comisión: debe aparecer antes
  // que los formularios administrativos todavía pendientes de conexión.
  if (areaItinerario) panel.append(areaItinerario);
  panel.append(areaBorradores);
  const documentos = elemento(documento, "section");
  documentos.className = "dietas-recorridos-documentos";
  documentos.append(
    elemento(documento, "h3", t("recorridos_documentos_pendientes")),
    tablaPendiente(
      documento,
      t,
      [
        "recorridos_referencia",
        "recorridos_fecha_apertura",
        "recorridos_editar",
        "recorridos_retirar",
        "recorridos_enviar_revision",
      ],
      "recorridos_documentos_pendientes",
    ),
    elemento(documento, "h4", t("recorridos_centro_unidad")),
    elemento(documento, "p", t("recorridos_sin_asignacion_verificada")),
    elemento(documento, "h4", t("recorridos_destinatarios")),
    elemento(documento, "p", t("recorridos_sin_asignacion_verificada")),
    elemento(documento, "h4", t("recorridos_control_documentos")),
    botonPendiente(documento, t, "recorridos_desde"),
    botonPendiente(documento, t, "recorridos_hasta"),
  );
  panel.append(documentos);
  const bandeja = elemento(documento, "section");
  bandeja.className = "panel dietas-presentacion-bandeja";
  bandeja.append(
    elemento(documento, "h3", t("recorridos_mis_solicitudes")),
    tablaPresentacion(documento, t, ["recorridos_solicitud", "recorridos_fechas", "motivo", "recorridos_total_ejemplo", "cab_estado"], seleccionada),
    detallesPresentacion(documento, t, seleccionada, "solicitante"),
  );
  panel.append(bandeja);
  panel.append(formularioGastos(documento, t));
  const envio = elemento(documento, "section");
  envio.className = "dietas-recorridos-acciones";
  envio.append(
    elemento(documento, "h3", t("recorridos_revision_envio")),
    elemento(documento, "p", t("recorridos_pendiente_conexion")),
    botonPendiente(documento, t, "recorridos_enviar_revision"),
  );
  panel.append(envio);
  return panel;
}

function panelJefatura(documento, t, seleccionada) {
  const panel = elemento(documento, "section");
  panel.className = "panel dietas-recorridos-principal";
  panel.dataset.dietasPanelEtapa = "jefatura";
  panel.append(
    elemento(documento, "h3", t("recorridos_jefatura")),
    elemento(documento, "p", t("recorridos_jefatura_ayuda")),
  );
  const filtros = elemento(documento, "form");
  filtros.className = "dietas-recorridos-filtros";
  ["recorridos_filtrar_estado", "recorridos_filtrar_periodo"].forEach(
    (clave) => {
      const label = elemento(documento, "label", t(clave));
      const input = elemento(documento, "input");
      input.type = "search";
      input.placeholder = t("recorridos_filtro_ayuda");
      label.append(input);
      filtros.append(label);
    },
  );
  panel.append(
    filtros,
    tablaPresentacion(documento, t, ["recorridos_solicitud", "recorridos_fechas", "motivo", "recorridos_total_ejemplo", "cab_estado", "recorridos_unidad"], seleccionada, true),
  );
  const detalle = elemento(documento, "section");
  detalle.className = "dietas-recorridos-acciones";
  detalle.append(
    elemento(documento, "h3", t("recorridos_detalle_solicitud")),
    elemento(documento, "p", t("recorridos_detalle_pendiente")),
    botonPendiente(documento, t, "recorridos_validar"),
    botonPendiente(documento, t, "recorridos_devolver"),
    botonPendiente(documento, t, "recorridos_rechazar"),
  );
  panel.append(detalle);
  panel.append(detallesPresentacion(documento, t, seleccionada, "jefatura"));
  return panel;
}

function panelGestion(documento, t, seleccionada) {
  const panel = elemento(documento, "section");
  panel.className = "panel dietas-recorridos-principal";
  panel.dataset.dietasPanelEtapa = "gestion";
  panel.append(
    elemento(documento, "h3", t("recorridos_gestion")),
    elemento(documento, "p", t("recorridos_gestion_ayuda")),
    tablaPresentacion(documento, t, ["recorridos_solicitud", "recorridos_fechas", "motivo", "recorridos_total_ejemplo", "cab_estado"], seleccionada),
  );
  const acciones = elemento(documento, "section");
  acciones.className = "dietas-recorridos-acciones";
  acciones.append(
    elemento(documento, "h3", t("recorridos_liquidacion_seguimiento")),
    elemento(documento, "p", t("recorridos_pendiente_conexion")),
    botonPendiente(documento, t, "recorridos_revisar_conceptos"),
    botonPendiente(documento, t, "recorridos_liquidar"),
    botonPendiente(documento, t, "recorridos_seguir_pago"),
  );
  panel.append(acciones);
  const comision = COMISIONES_PRESENTACION.find((item) => item.referencia === seleccionada) || COMISIONES_PRESENTACION[0];
  panel.append(detallesPresentacion(documento, t, seleccionada, "gestion"));
  const incidencia = elemento(documento, "p", `${t("recorridos_incidencia")}: ${comision.incidencia}`); incidencia.className = "dietas-presentacion-incidencia"; panel.append(incidencia);
  return panel;
}

/** Presenta un recorrido navegable; cada efecto continúa cerrado hasta su puerto autorizado. */
export function montarVistaRecorridosDietas(
  contenedor,
  {
    clienteBorradores,
    traducir = crearTraductorDietas(MENSAJES_DIETAS_ES),
    montarItinerario,
    anunciar = () => {},
    registrarDesmontar,
  } = {},
) {
  if (
    !contenedor?.append ||
    !contenedor?.querySelector ||
    !contenedor.ownerDocument ||
    typeof traducir !== "function" ||
    typeof anunciar !== "function" ||
    (montarItinerario !== undefined &&
      typeof montarItinerario !== "function") ||
    (registrarDesmontar !== undefined &&
      typeof registrarDesmontar !== "function")
  )
    throw new TypeError("recorrido de Dietas no disponible");
  const documento = contenedor.ownerDocument;
  const raiz = elemento(documento, "section");
  raiz.className = "modulo-dietas";
  raiz.dataset.dietasRecorridos = "";
  // El coordinador cede este contenedor al recorrido: sustituimos su indicador
  // inicial para que no quede un "Cargando módulo" junto al contenido real.
  contenedor.replaceChildren(raiz);
  let activa = true;
  let etapa = "solicitante";
  let seleccionada = COMISIONES_PRESENTACION[0].referencia;
  let desmontarBorradores = () => {};
  let desmontarItinerario = () => {};
  const areaBorradores = elemento(documento, "div");
  areaBorradores.dataset.dietasAreaBorradores = "";
  try {
    desmontarBorradores = montarVistaBorradoresPropios(areaBorradores, {
      cliente: clienteBorradores,
      traducir,
      anunciar,
    }).desmontar;
  } catch {
    areaBorradores.append(
      elemento(
        documento,
        "p",
        traducir("borradores_propios_pendiente_conexion"),
      ),
    );
  }
  const areaItinerario = montarItinerario ? elemento(documento, "div") : null;
  if (areaItinerario) {
    areaItinerario.dataset.dietasAreaItinerario = "";
    try {
      Promise.resolve(montarItinerario(areaItinerario)).then(
        (vista) => {
          if (typeof vista?.desmontar !== "function") return;
          if (!activa) {
            vista.desmontar();
            return;
          }
          desmontarItinerario = vista.desmontar;
        },
        () => {
          if (activa && sigueMontada(contenedor, raiz))
            areaItinerario.append(
              elemento(documento, "p", traducir("recorridos_sin_datos")),
            );
        },
      );
    } catch {
      areaItinerario.append(
        elemento(documento, "p", traducir("recorridos_sin_datos")),
      );
    }
  }
  const pasos = elemento(documento, "nav");
  pasos.className = "dietas-recorridos-pasos";
  pasos.setAttribute("aria-label", traducir("recorridos_titulo"));
  const botonesEtapa = new Map();
  ETAPAS.forEach(([valor, etiqueta], indice) => {
    const boton = elemento(
      documento,
      "button",
      `${indice + 1}. ${traducir(etiqueta)}`,
    );
    boton.type = "button";
    boton.dataset.dietasCambiarEtapa = valor;
    botonesEtapa.set(valor, boton);
    pasos.append(boton);
  });
  const solicitante = panelSolicitante(
    documento,
    traducir,
    areaBorradores,
    areaItinerario,
    seleccionada,
  );
  const jefatura = panelJefatura(documento, traducir, seleccionada);
  const gestion = panelGestion(documento, traducir, seleccionada);
  const paneles = new Map([
    ["solicitante", solicitante],
    ["jefatura", jefatura],
    ["gestion", gestion],
  ]);
  const cuerpo = elemento(documento, "div");
  cuerpo.className = "dietas-recorridos-cuerpo";
  cuerpo.append(solicitante, jefatura, gestion);
  raiz.append(pasos, cuerpo);

  function pintar() {
    if (!activa || !sigueMontada(contenedor, raiz)) return;
    paneles.forEach((panel, valor) => { panel.hidden = valor !== etapa; });
    botonesEtapa.forEach((boton, valor) => {
      if (valor === etapa) boton.setAttribute("aria-current", "step");
      else boton.removeAttribute?.("aria-current");
    });
    if (etapa === "solicitante" && areaItinerario) {
      // Leaflet escucha resize: al volver desde una etapa oculta recalcula el
      // lienzo sin desmontar el visor ni perder el cálculo ya mostrado.
      documento.defaultView?.dispatchEvent?.(new Event("resize"));
    }
  }
  function cambiarEtapa(evento) {
    const boton = evento.target?.closest?.("[data-dietas-cambiar-etapa]");
    const seleccion = evento.target?.closest?.("[data-dietas-seleccionar-comision]");
    if (seleccion && activa) {
      seleccionada = seleccion.dataset.dietasSeleccionarComision;
      raiz.querySelectorAll?.("[data-dietas-comision-ref]").forEach((fila) => {
        const esActual = fila.dataset.dietasComisionRef === seleccionada;
        fila.dataset.dietasSeleccionada = String(esActual);
        fila.querySelector?.("[data-dietas-seleccionar-comision]")?.setAttribute("aria-current", String(esActual));
      });
      raiz.querySelectorAll?.("[data-dietas-detalle-presentacion]").forEach((detalle) => {
        detalle.hidden = detalle.dataset.dietasDetallePresentacion !== seleccionada;
      });
      pintar();
      Array.from(raiz.querySelectorAll?.("[data-dietas-detalle-presentacion]") || []).find((detalle) => detalle.dataset.dietasDetallePresentacion === seleccionada)?.focus?.();
      return;
    }
    if (
      !boton ||
      !activa ||
      !ETAPAS.some(([valor]) => valor === boton.dataset.dietasCambiarEtapa)
    )
      return;
    etapa = boton.dataset.dietasCambiarEtapa;
    pintar();
    raiz.querySelector?.('[aria-current="step"]')?.focus?.();
  }
  function desmontar() {
    if (!activa) return;
    activa = false;
    raiz.removeEventListener("click", cambiarEtapa);
    desmontarBorradores();
    desmontarItinerario();
    retirar(contenedor, raiz);
  }
  raiz.addEventListener("click", cambiarEtapa);
  registrarDesmontar?.(desmontar);
  pintar();
  return Object.freeze({ desmontar });
}
