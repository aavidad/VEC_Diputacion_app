import { renderizarEstadoEntrega } from "../../estado-entrega.js";
import { crearTraductorDocumentos } from "./i18n.js";

function nodo(documento, etiqueta, texto = "", clase = "") {
  const elemento = documento.createElement(etiqueta);
  if (texto) elemento.textContent = texto;
  if (clase) elemento.className = clase;
  return elemento;
}

function panel(documento, titulo, nota, clase = "") {
  const seccion = nodo(documento, "section", "", `panel ${clase}`.trim());
  const cabecera = nodo(documento, "header", "", "cabecera-panel");
  const titulos = nodo(documento, "div");
  titulos.append(nodo(documento, "h3", titulo), nodo(documento, "p", nota));
  cabecera.append(titulos);
  const cuerpo = nodo(documento, "div", "", "cuerpo-panel");
  seccion.append(cabecera, cuerpo);
  return { seccion, cuerpo };
}

function botonBloqueado(documento, texto, motivo) {
  const boton = nodo(documento, "button", texto, "documentos-accion");
  boton.type = "button";
  boton.disabled = true;
  boton.setAttribute("aria-disabled", "true");
  boton.setAttribute("aria-label", `${texto}. ${motivo}`);
  boton.title = motivo;
  return boton;
}

function filaDato(documento, titulo, valor) {
  const fila = nodo(documento, "div");
  fila.append(nodo(documento, "dt", titulo), nodo(documento, "dd", valor));
  return fila;
}

/** Estado sin fuente: no consulta, genera, firma, custodia ni descarga documentos. */
export function montarVistaDocumentos({ raiz, registrarDesmontar } = {}) {
  const t = crearTraductorDocumentos();
  if (!raiz?.append || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) {
    throw new TypeError(t("error_vista"));
  }
  const documento = raiz.ownerDocument;
  if (!documento?.createElement) throw new TypeError(t("error_documento"));

  let activa = true;
  const contenedor = nodo(documento, "section", "", "modulo-documentos");
  contenedor.dataset.documentos = "";
  contenedor.dataset.estado = "no_configurado";
  raiz.append(contenedor);
  const desmontar = () => {
    if (!activa) return;
    activa = false;
    contenedor.remove();
  };
  registrarDesmontar?.(desmontar);

  const cabecera = nodo(documento, "header", "", "documentos-cabecera");
  cabecera.append(
    nodo(documento, "p", t("sobrelinea"), "sobrelinea"),
    nodo(documento, "h2", t("titulo")),
    nodo(documento, "p", t("descripcion")),
    nodo(documento, "p", t("sin_fuente"), "documentos-aviso"),
  );

  const indicadores = nodo(documento, "div", "", "documentos-indicadores");
  const resumen = [
    ["borrador", "estado_borrador", "nota_borrador"],
    ["firmado", "estado_firmado", "nota_firmado"],
    ["descarga", "estado_descarga", "nota_descarga"],
    ["custodia", "estado_custodia", "nota_custodia"],
  ];
  resumen.forEach(([titulo, valor, nota]) => {
    const tarjeta = nodo(documento, "article", "", "documentos-indicador");
    tarjeta.append(nodo(documento, "span", t(titulo)), nodo(documento, "strong", t(valor)), nodo(documento, "small", t(nota)));
    indicadores.append(tarjeta);
  });

  const principal = nodo(documento, "div", "", "documentos-principal");
  const listado = panel(documento, t("repositorio_titulo"), t("repositorio_nota"), "documentos-listado");
  const ficha = panel(documento, t("ficha"), t("ficha_nota"), "documentos-ficha");
  principal.append(listado.seccion, ficha.seccion);

  const vacio = nodo(documento, "div", "", "documentos-vacio");
  vacio.setAttribute("role", "status");
  vacio.append(
    nodo(documento, "strong", t("estado_no_configurado")),
    nodo(documento, "p", t("lista_vacia")),
    nodo(documento, "small", t("lista_dependencia")),
  );
  listado.cuerpo.append(vacio);

  const fichaTitulo = nodo(documento, "h4", t("ficha_vacia"), "documentos-titulo-ficha");
  const fichaDescripcion = nodo(documento, "p", t("ficha_sin_original"), "documentos-ficha-aviso");
  const metadatos = nodo(documento, "dl", "", "documentos-metadatos");
  metadatos.append(
    filaDato(documento, t("version"), t("version_sin_fuente")),
    filaDato(documento, t("firma"), t("firma_sin_evidencia")),
    filaDato(documento, t("huella"), t("huella_sin_fuente")),
    filaDato(documento, t("conservacion"), t("custodia_sin_fuente")),
  );
  const acciones = nodo(documento, "div", "", "documentos-acciones");
  [["generar_version", "generar_motivo"], ["subir_original", "subir_motivo"], ["firmar", "firmar_motivo"], ["verificar", "verificar_motivo"], ["enviar", "enviar_motivo"], ["descargar", "descargar_motivo"]].forEach(([texto, motivo]) => {
    acciones.append(botonBloqueado(documento, t(texto), t(motivo)));
  });
  ficha.cuerpo.append(fichaTitulo, fichaDescripcion, metadatos, nodo(documento, "h5", t("acciones_pendientes")), acciones);

  const ayuda = nodo(documento, "details", "", "panel documentos-ayuda");
  const pregunta = nodo(documento, "summary", t("ayuda_titulo"));
  pregunta.setAttribute("aria-label", t("ayuda_etiqueta"));
  const ayudaContenido = nodo(documento, "div", "", "cuerpo-panel");
  ayudaContenido.append(nodo(documento, "p", t("aclaracion_firma")), nodo(documento, "h4", t("tipos_previstos")));
  const tipos = nodo(documento, "ul", "", "documentos-tipos");
  ["tipo_informe", "tipo_resolucion", "tipo_rc", "tipo_diligencia"].forEach((clave) => tipos.append(nodo(documento, "li", t(clave))));
  ayudaContenido.append(tipos, nodo(documento, "h4", t("circuito_titulo")));
  const pasos = nodo(documento, "ol", "", "documentos-pasos");
  ["paso_1", "paso_2", "paso_3", "paso_4", "paso_5"].forEach((clave) => pasos.append(nodo(documento, "li", t(clave))));
  ayudaContenido.append(pasos);
  const entrega = nodo(documento, "div", "", "documentos-entrega");
  // El componente común escapa el texto de este catálogo cerrado.
  entrega.innerHTML = renderizarEstadoEntrega({ estado: "visual_pendiente_backend", resumen: t("entrega_resumen"), pendientes: [t("entrega_repositorio"), t("entrega_autorizacion"), t("entrega_huella"), t("entrega_firma"), t("entrega_auditoria")], fuente: { etiqueta: t("sin_fuente") }, conexion: t("entrega_conexion") });
  ayudaContenido.append(entrega);
  ayuda.append(pregunta, ayudaContenido);

  contenedor.append(cabecera, indicadores, principal, ayuda);
  return Object.freeze({ desmontar });
}
