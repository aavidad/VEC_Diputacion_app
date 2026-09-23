import { crearTraductorAdministracion } from "./i18n.js?v=20260924-f2-web2";
import { montarVistaApariencia } from "./vista-apariencia.js?v=20260924-f2-web2";

export const PESTANAS_ADMINISTRACION = Object.freeze([
  "resumen", "roles", "catalogos", "calendarios", "reglas", "conectores",
  "modulos", "privacidad", "ia", "apariencia",
]);

const ACCIONES_PENDIENTES = Object.freeze({
  roles: ["guardar_asignacion", "revocar_permiso"],
  catalogos: ["guardar_catalogo", "publicar_version"],
  calendarios: ["guardar_calendario", "publicar_calendario"],
  reglas: ["guardar_regla", "publicar_regla"],
  conectores: ["probar_conexion", "rotar_secreto"],
  modulos: ["activar_modulo", "publicar_modulo"],
  privacidad: ["guardar_conservacion", "revocar_acceso"],
  ia: ["probar_conexion", "activar_ia"],
});

let secuenciaVista = 0;

function crear(documento, etiqueta, texto = "", clase = "") {
  const elemento = documento.createElement(etiqueta);
  if (texto) elemento.textContent = texto;
  if (clase) elemento.className = clase;
  return elemento;
}

function contenidoPendiente(documento, t, clave) {
  const seccion = crear(documento, "section", "", "panel administracion-configuracion");
  seccion.dataset.configuracionEstado = "no_configurado";
  const cabecera = crear(documento, "header", "", "cabecera-panel");
  cabecera.append(crear(documento, "h3", t("tab_" + clave)), crear(documento, "span", t("no_configurado"), "estado-chip aviso"));
  const cuerpo = crear(documento, "div", "", "cuerpo-panel administracion-configuracion-cuerpo");
  cuerpo.append(crear(documento, "p", t("sin_fuente")));

  if (clave === "ia") {
    const campos = crear(documento, "div", "", "administracion-ia-campos");
    for (const campo of ["ia_endpoint", "ia_modelo", "ia_indice"]) {
      const etiqueta = crear(documento, "label", t(campo));
      const entrada = crear(documento, "input");
      entrada.type = "text";
      entrada.value = "";
      entrada.disabled = true;
      entrada.setAttribute("aria-label", t(campo) + " · " + t("no_configurado"));
      etiqueta.append(entrada);
      campos.append(etiqueta);
    }
    cuerpo.append(campos);
  }

  const acciones = ACCIONES_PENDIENTES[clave] || [];
  if (acciones.length) {
    const grupo = crear(documento, "div", "", "administracion-acciones");
    for (const accion of acciones) {
      const boton = crear(documento, "button", t(accion), "administracion-accion");
      boton.type = "button";
      boton.disabled = true;
      boton.setAttribute("aria-disabled", "true");
      boton.title = t("accion_bloqueada");
      grupo.append(boton);
    }
    cuerpo.append(grupo, crear(documento, "p", t("accion_bloqueada"), "administracion-limite"));
  }
  seccion.append(cabecera, cuerpo);
  return seccion;
}

/** Sin autoridad de configuración: muestra estados y una vista previa efímera. */
export function montarVistaAdministracion({ raiz, anunciar = () => {}, registrarDesmontar } = {}) {
  if (!raiz?.replaceChildren || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) {
    throw new TypeError("vista de Administración no disponible");
  }
  const documento = raiz.ownerDocument;
  if (!documento?.createElement) throw new TypeError("documento de Administración no disponible");
  const t = crearTraductorAdministracion();
  const idVista = "administracion-" + ++secuenciaVista;
  let activa = true;
  let pestana = "resumen";
  let vistaApariencia = null;

  const pintar = (focoPestana = false) => {
    if (!activa) return;
    vistaApariencia?.desmontar();
    vistaApariencia = null;
    raiz.replaceChildren();

    const seccion = crear(documento, "section", "", "modulo-administracion");
    seccion.dataset.administracionVista = "";
    seccion.dataset.estadoEntrega = "no_configurado";
    const cabecera = crear(documento, "header", "", "cabecera-vista");
    cabecera.append(crear(documento, "p", t("sobrelinea"), "sobrelinea"), crear(documento, "h2", t("titulo")), crear(documento, "p", t("descripcion")));

    const estado = crear(documento, "details", "", "administracion-entrega");
    const resumen = crear(documento, "summary", t("estado_conexion_breve"));
    const explicacion = crear(documento, "div");
    explicacion.append(crear(documento, "p", t("estado_resumen")));
    const pendientes = crear(documento, "ul");
    for (const clave of ["estado_pendiente_1", "estado_pendiente_2", "estado_pendiente_3", "estado_pendiente_4"]) {
      pendientes.append(crear(documento, "li", t(clave)));
    }
    explicacion.append(pendientes);
    estado.append(resumen, explicacion);

    const nav = crear(documento, "nav", "", "administracion-pestanas");
    nav.setAttribute("role", "tablist");
    nav.setAttribute("aria-label", t("pestanas"));
    let tabSeleccionada = null;
    const seleccionarPestana = (clave) => {
      if (clave === pestana) return;
      pestana = clave;
      pintar(true);
      anunciar(t("seccion_seleccionada", { etiqueta: t("tab_" + clave) }), "info");
    };
    PESTANAS_ADMINISTRACION.forEach((clave, indice) => {
      const boton = crear(documento, "button", t("tab_" + clave));
      boton.type = "button";
      boton.id = idVista + "-tab-" + clave;
      boton.dataset.administracionPestana = clave;
      boton.setAttribute("role", "tab");
      boton.setAttribute("aria-selected", String(clave === pestana));
      boton.setAttribute("aria-controls", idVista + "-panel");
      boton.tabIndex = clave === pestana ? 0 : -1;
      if (clave === pestana) tabSeleccionada = boton;
      boton.addEventListener("click", () => seleccionarPestana(clave));
      boton.addEventListener("keydown", (evento) => {
        const destino = evento.key === "ArrowRight" ? (indice + 1) % PESTANAS_ADMINISTRACION.length
          : evento.key === "ArrowLeft" ? (indice - 1 + PESTANAS_ADMINISTRACION.length) % PESTANAS_ADMINISTRACION.length
            : evento.key === "Home" ? 0 : evento.key === "End" ? PESTANAS_ADMINISTRACION.length - 1 : -1;
        if (destino < 0) return;
        evento.preventDefault();
        seleccionarPestana(PESTANAS_ADMINISTRACION[destino]);
      });
      nav.append(boton);
    });

    const cuerpo = pestana === "apariencia"
      ? crear(documento, "div", "", "administracion-apariencia-raiz")
      : contenidoPendiente(documento, t, pestana);
    cuerpo.id = idVista + "-panel";
    cuerpo.setAttribute("role", "tabpanel");
    cuerpo.setAttribute("aria-labelledby", idVista + "-tab-" + pestana);
    seccion.append(cabecera, estado, nav, cuerpo);
    raiz.append(seccion);
    if (pestana === "apariencia") vistaApariencia = montarVistaApariencia({ raiz: cuerpo, anunciar, t });
    if (focoPestana) tabSeleccionada?.focus?.();
  };

  pintar();
  const desmontar = () => {
    if (!activa) return;
    activa = false;
    vistaApariencia?.desmontar();
    raiz.replaceChildren();
  };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar });
}
