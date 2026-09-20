import { crearTraductorDietas, MENSAJES_DIETAS_ES } from "./i18n.js";
import { montarVistaBorradoresPropios } from "./vista-borradores-propios.js";

const ETAPAS = Object.freeze([
  ["solicitante", "recorridos_solicitante"],
  ["jefatura", "recorridos_jefatura"],
  ["gestion", "recorridos_gestion"],
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
  boton.setAttribute(
    "aria-describedby",
    "dietas-recorridos-conexion-pendiente",
  );
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

function panelSolicitante(documento, t, areaBorradores, areaItinerario) {
  const panel = elemento(documento, "section");
  panel.className = "panel dietas-recorridos-principal";
  panel.dataset.dietasPanelEtapa = "solicitante";
  panel.append(
    elemento(documento, "h3", t("recorridos_solicitante")),
    elemento(documento, "p", t("recorridos_solicitante_ayuda")),
    areaBorradores,
  );
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
  if (areaItinerario) panel.append(areaItinerario);
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

function panelJefatura(documento, t) {
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
      input.disabled = true;
      label.append(input);
      filtros.append(label);
    },
  );
  panel.append(
    filtros,
    tablaPendiente(
      documento,
      t,
      [
        "recorridos_solicitud",
        "recorridos_persona",
        "recorridos_fechas",
        "cab_estado",
      ],
      "recorridos_bandeja",
    ),
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
  return panel;
}

function panelGestion(documento, t) {
  const panel = elemento(documento, "section");
  panel.className = "panel dietas-recorridos-principal";
  panel.dataset.dietasPanelEtapa = "gestion";
  panel.append(
    elemento(documento, "h3", t("recorridos_gestion")),
    elemento(documento, "p", t("recorridos_gestion_ayuda")),
    tablaPendiente(
      documento,
      t,
      [
        "recorridos_concepto",
        "recorridos_importe_declarado",
        "recorridos_total_no_calculado",
      ],
      "recorridos_revision_conceptos",
    ),
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
  return panel;
}

function resumen(documento, t, etapa) {
  const lateral = elemento(documento, "aside");
  lateral.className = "dietas-recorridos-lateral panel";
  lateral.dataset.dietasResumenEtapa = "";
  lateral.append(
    elemento(documento, "h3", t("recorridos_resumen")),
    elemento(documento, "p", t(ETAPAS.find(([valor]) => valor === etapa)[1])),
    elemento(documento, "p", t("recorridos_sin_datos")),
    elemento(documento, "small", t("recorridos_pendiente_conexion")),
  );
  return lateral;
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
  const titulo = elemento(documento, "h2", traducir("recorridos_titulo"));
  const avisoConexion = elemento(
    documento,
    "p",
    traducir("recorridos_pendiente_conexion"),
  );
  avisoConexion.id = "dietas-recorridos-conexion-pendiente";
  avisoConexion.className = "dietas-recorridos-aviso";
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
  );
  const jefatura = panelJefatura(documento, traducir);
  const gestion = panelGestion(documento, traducir);
  const paneles = new Map([
    ["solicitante", solicitante],
    ["jefatura", jefatura],
    ["gestion", gestion],
  ]);
  const lateral = resumen(documento, traducir, etapa);
  const cuerpo = elemento(documento, "div");
  cuerpo.className = "dietas-recorridos-cuerpo";
  cuerpo.append(solicitante, jefatura, gestion, lateral);
  raiz.append(titulo, avisoConexion, pasos, cuerpo);

  function pintar() {
    if (!activa || !sigueMontada(contenedor, raiz)) return;
    paneles.forEach((panel, valor) => {
      panel.hidden = valor !== etapa;
    });
    botonesEtapa.forEach((boton, valor) => {
      if (valor === etapa) boton.setAttribute("aria-current", "step");
      else boton.removeAttribute?.("aria-current");
    });
    lateral.replaceChildren(
      elemento(documento, "h3", traducir("recorridos_resumen")),
      elemento(
        documento,
        "p",
        traducir(ETAPAS.find(([valor]) => valor === etapa)[1]),
      ),
      elemento(documento, "p", traducir("recorridos_sin_datos")),
      elemento(documento, "small", traducir("recorridos_pendiente_conexion")),
    );
    if (etapa === "solicitante" && areaItinerario) {
      // Leaflet escucha resize: al volver desde una etapa oculta recalcula el
      // lienzo sin desmontar el visor ni perder el cálculo ya mostrado.
      documento.defaultView?.dispatchEvent?.(new Event("resize"));
    }
  }
  function cambiarEtapa(evento) {
    const boton = evento.target?.closest?.("[data-dietas-cambiar-etapa]");
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
