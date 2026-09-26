import { ErrorAPIBorradores } from "./portal-borradores-api.js";
import { textoPortal, traducirPortal } from "./portal-i18n.js?v=20260926-pulido-portal-v1";

export const FASE_INICIAL = "inicial";
export const FASE_CARGANDO = "cargando";
export const FASE_LISTA = "lista";
export const FASE_ERROR = "error";

export function copiar(valor) {
  return valor === undefined ? undefined : JSON.parse(JSON.stringify(valor));
}

export function errorSeguro(error, mensajeDefecto) {
  if (error instanceof ErrorAPIBorradores) {
    return {
      mensaje: error.message,
      codigo: error.codigo,
      correlacion: error.correlacion,
      estadoHTTP: error.estado,
      tipoConflicto: error.tipoConflicto,
      conservarCambiosLocales: error.conservarCambiosLocales,
    };
  }
  return {
    mensaje: mensajeDefecto,
    codigo: "error_interfaz_borradores",
    correlacion: null,
    estadoHTTP: 0,
    tipoConflicto: null,
    conservarCambiosLocales: false,
  };
}

export function editorNuevo(opciones) {
  return {
    plantilla_indice: 0,
    motivo_indice: 0,
    codigo_version_publica: "",
    identificador_publico: "",
    expediente_ref: "",
    contenido_editable: {
      tipo: opciones.tipos[0]?.clave || "",
      categorias: opciones.categorias[0] ? [opciones.categorias[0].clave] : [],
      titulo: "",
      resumen: "",
      descripcion: "",
      plazos: [{ referencia: "", tipo: "", titulo: "", descripcion: "", abre_en: "", cierra_en: "" }],
      requisitos: [],
      ayuda: [],
    },
  };
}

export function editorDesdeDetalle(detalle) {
  return { motivo_indice: 0, contenido_editable: copiar(detalle.contenido_editable) };
}

export function asignarRutaEditor(editor, ruta, valor) {
  const partes = String(ruta).split(".");
  if (partes.length < 1 || partes.length > 5
    || partes.some((parte) => !/^(?:[a-z_]+|[0-9]+)$/.test(parte)
      || parte === "__proto__" || parte === "constructor" || parte === "prototype")) {
    throw new TypeError("ruta de editor no válida");
  }
  let actual = editor;
  for (let indice = 0; indice < partes.length - 1; indice += 1) {
    if (actual === null || typeof actual !== "object" || !Object.hasOwn(actual, partes[indice])) {
      throw new TypeError("ruta de editor no disponible");
    }
    actual = actual[partes[indice]];
  }
  const ultimo = partes.at(-1);
  if (actual === null || typeof actual !== "object" || !Object.hasOwn(actual, ultimo)) {
    throw new TypeError("campo de editor no disponible");
  }
  actual[ultimo] = valor;
}

export function instalarDeeplinkAvisosBorradores({
  documento, escaparHTML, porId, obtenerAvisos, disponible, navegar, anunciar,
} = {}) {
  if (!documento?.addEventListener || !documento?.removeEventListener
    || [escaparHTML, porId, obtenerAvisos, disponible, navegar, anunciar]
      .some((dependencia) => typeof dependencia !== "function")) {
    throw new TypeError("dependencias del deeplink de avisos no válidas");
  }
  const manejarClick = (evento) => {
    const destino = evento.target?.closest?.("[data-aviso-borrador-ref]");
    if (destino) {
      const referencia = destino.dataset.avisoBorradorRef;
      if (referencia !== "DEMO-BORRADOR-001") return;
      evento.preventDefault?.();
      evento.stopImmediatePropagation?.();
      const dialogo = porId("dialogo-detalle");
      if (dialogo?.open && typeof dialogo.close === "function") dialogo.close();
      navegar("elaboracion", { referencia });
      anunciar(traducirPortal("txt_aviso_borradores_de_convocatorias"));
      return;
    }
    const botonAvisos = evento.target?.closest?.('.boton-avisos[data-accion="avisos"]');
    if (!botonAvisos) return;
    const avisos = obtenerAvisos();
    const aviso = avisos.find((item) => item?.destino?.vista === "elaboracion"
      && item.destino.estado === "disponible" && item.destino.referencia === "DEMO-BORRADOR-001");
    if (!aviso) return;
    evento.preventDefault?.();
    evento.stopImmediatePropagation?.();
    const dialogo = porId("dialogo-detalle");
    const titulo = porId("titulo-dialogo");
    const contenido = porId("contenido-dialogo");
    if (!dialogo || !titulo || !contenido) return;
    titulo.textContent = traducirPortal("txt_avisos");
    const permitido = disponible();
    const elementos = avisos.filter((item) => permitido || !item?.destino).map((item) => {
      const textoAviso = `<p>${escaparHTML(item.texto)}</p>`;
      if (item !== aviso) return `<li>${textoAviso}</li>`;
      return `<li>${textoAviso}<button type="button" class="boton-secundario" data-aviso-borrador-ref="${escaparHTML(aviso.destino.referencia)}" aria-label="${textoPortal("txt_ir_a", { destino: aviso.destino.etiqueta })}">${textoPortal("txt_ir_a", { destino: aviso.destino.etiqueta })}</button></li>`;
    }).join("");
    contenido.innerHTML = `<ul class="lista-avisos-navegables">${elementos || "<li>" + textoPortal("txt_no_hay_avisos_accesibles") + "</li>"}</ul>`;
    if (typeof dialogo.showModal === "function") dialogo.showModal();
    else dialogo.setAttribute("open", "");
    queueMicrotask(() => contenido.querySelector("[data-aviso-borrador-ref]")?.focus());
  };
  documento.addEventListener("click", manejarClick, { capture: true });
  return () => documento.removeEventListener("click", manejarClick, { capture: true });
}
