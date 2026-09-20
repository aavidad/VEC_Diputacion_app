import { renderizarEstadoEntrega } from "../../estado-entrega.js";
import { DATOS_ADMINISTRACION_PRESENTACION, crearEstadoAdministracion, PESTANAS_ADMINISTRACION } from "./datos-presentacion.js";
import { crearTraductorAdministracion } from "./i18n.js";

const crear = (d, etiqueta, texto = "", clase = "") => { const n = d.createElement(etiqueta); if (texto) n.textContent = texto; if (clase) n.className = clase; return n; };
const accionBloqueada = (d, t, texto, motivo = t("accion_bloqueada")) => { const b = crear(d, "button", texto, "administracion-accion"); b.type = "button"; b.disabled = true; b.setAttribute("aria-disabled", "true"); b.title = motivo; return b; };

function tabla(d, t, titulo, cabeceras, filas, seleccion, alSeleccionar, indices = filas.map((_, indice) => indice)) {
  const region = crear(d, "div", "", "administracion-tabla"); region.tabIndex = 0; region.setAttribute("role", "region"); region.setAttribute("aria-label", titulo);
  const table = crear(d, "table"); table.append(crear(d, "caption", titulo)); const head = crear(d, "thead"); const filaCabecera = crear(d, "tr"); cabeceras.forEach((cabecera) => { const th = crear(d, "th", cabecera); th.scope = "col"; filaCabecera.append(th); }); head.append(filaCabecera); const cuerpo = crear(d, "tbody");
  filas.forEach((fila, i) => { const indiceOriginal = indices[i]; const tr = crear(d, "tr"); if (seleccion === indiceOriginal) tr.dataset.seleccionada = "true"; fila.forEach((valor) => tr.append(crear(d, "td", valor))); if (alSeleccionar) { const td = crear(d, "td"); const boton = crear(d, "button", t("ver_resumen"), "administracion-enlace-local"); boton.type = "button"; boton.addEventListener("click", () => alSeleccionar(indiceOriginal)); td.append(boton); tr.append(td); } cuerpo.append(tr); });
  table.append(head, cuerpo); region.append(table); return region;
}

export function filtrarFilasAdministracion(filas, filtro = "") {
  return filas.map((fila, indice) => ({ fila, indice })).filter(({ fila }) => !filtro || fila.join(" ").toLocaleLowerCase("es").includes(filtro.toLocaleLowerCase("es")));
}

function listado(d, t, titulo, filas, seleccion, seleccionar, filtro = "") {
  const visibles = filtrarFilasAdministracion(filas, filtro);
  const seccion = crear(d, "section", "", "panel administracion-listado"); seccion.append(crear(d, "h3", titulo));
  seccion.append(tabla(d, t, titulo, [t("elemento"), t("referencia"), t("estado"), t("accion")], visibles.map(({ fila }) => fila), seleccion, seleccionar, visibles.map(({ indice }) => indice)));
  if (!visibles.length) seccion.append(crear(d, "p", t("sin_resultados"))); return seccion;
}

function detalle(d, t, fila, clave) {
  const aside = crear(d, "aside", "", "panel administracion-detalle"); aside.tabIndex = -1; aside.append(crear(d, "h3", t("detalle")), crear(d, "p", fila?.[0] || t("seleccionar")));
  const dl = crear(d, "dl", "", "administracion-datos"); [[t("referencia"), fila?.[1] || t("sin_valor")], [t("estado"), fila?.[2] || t("sin_valor")], [t("tratamiento"), t("consulta_local")]].forEach(([etiqueta, valor]) => { const x = crear(d, "div"); x.append(crear(d, "dt", etiqueta), crear(d, "dd", valor)); dl.append(x); });
  aside.append(dl, crear(d, "p", t(clave === "roles" ? "limite_roles" : "limite_configuracion"), "administracion-limite"), accionBloqueada(d, t, t("solicitar_cambio"))); return aside;
}

function formularioIA(d, t) {
  const seccion = crear(d, "section", "", "panel administracion-ia"); seccion.append(crear(d, "h3", t("ia_titulo")), crear(d, "p", t("ia_ayuda")));
  const formulario = crear(d, "form"); [[t("ia_endpoint"), "http://localhost:11434"], [t("ia_modelo"), t("ia_modelo_ejemplo")], [t("ia_indice"), t("ia_indice_ejemplo")]].forEach(([etiqueta, ejemplo]) => { const label = crear(d, "label", etiqueta); const input = crear(d, "input"); input.value = ejemplo; input.disabled = true; input.setAttribute("aria-describedby", "administracion-ia-limite"); label.append(input); formulario.append(label); });
  const controles = crear(d, "div", "", "administracion-acciones"); controles.append(accionBloqueada(d, t, t("probar_conexion"), t("prueba_bloqueada")), accionBloqueada(d, t, t("activar_ia"), t("activar_bloqueada"))); formulario.append(controles); seccion.append(formulario, crear(d, "p", t("ia_limite"), "administracion-limite")); return seccion;
}

function controlesPendientes(d, t, clave) {
  const claves = { roles: ["guardar_asignacion", "revocar_permiso"], catalogos: ["guardar_catalogo", "publicar_version"], calendarios: ["guardar_calendario", "publicar_calendario"], reglas: ["guardar_regla", "publicar_regla"], conectores: ["probar_conexion", "rotar_secreto"], modulos: ["activar_modulo", "publicar_modulo"], privacidad: ["guardar_conservacion", "revocar_acceso"] }[clave] || [];
  const grupo = crear(d, "div", "", "administracion-acciones"); claves.forEach((claveAccion) => grupo.append(accionBloqueada(d, t, t(claveAccion)))); return grupo;
}

function contenido(d, t, clave, estado, actualizar) {
  const datos = DATOS_ADMINISTRACION_PRESENTACION;
  if (clave === "resumen") { const seccion = crear(d, "section", "", "administracion-resumen"); const tarjetas = crear(d, "div", "", "administracion-kpis"); [[t("kpi_roles"), String(datos.roles.length)], [t("kpi_catalogos"), String(datos.catalogos.length)], [t("kpi_conectores"), "0"], [t("kpi_ia"), t("ia_desactivada")]].forEach(([etiqueta, valor]) => { const tarjeta = crear(d, "article", "", "administracion-kpi"); tarjeta.append(crear(d, "span", etiqueta), crear(d, "strong", valor), crear(d, "small", t("presentacion_sin_conexion"))); tarjetas.append(tarjeta); }); seccion.append(tarjetas, crear(d, "p", t("responsable", datos), "administracion-aviso")); return seccion; }
  if (clave === "ia") return formularioIA(d, t);
  const filas = datos[clave]; const titulo = t(`tab_${clave}`); const seleccion = estado.selecciones[clave] ?? 0; const grid = crear(d, "div", "", "administracion-espacio"); grid.append(listado(d, t, titulo, filas, seleccion, (indice) => actualizar({ selecciones: { ...estado.selecciones, [clave]: indice } }), estado.filtro), detalle(d, t, filas[seleccion], clave)); const salida = crear(d, "div", "", "administracion-contenido"); salida.append(grid, controlesPendientes(d, t, clave)); return salida;
}

/** Superficie visual local de Administración; no ejecuta red, efectos ni almacenamiento. */
export function montarVistaAdministracion({ raiz, anunciar = () => {}, registrarDesmontar } = {}) {
  if (!raiz?.replaceChildren || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("vista de Administración no disponible");
  const documento = raiz.ownerDocument; if (!documento?.createElement) throw new TypeError("documento de Administración no disponible"); const t = crearTraductorAdministracion(); const estadoEntrega = crearEstadoAdministracion(t); let activa = true; let estado = { pestana: "resumen", filtro: "", selecciones: {} };
  const pintar = () => { if (!activa) return; raiz.replaceChildren(); const seccion = crear(documento, "section", "", "modulo-administracion"); seccion.dataset.administracionVista = ""; seccion.dataset.estadoEntrega = "visual_pendiente_backend"; const cabecera = crear(documento, "header", "", "cabecera-vista"); cabecera.append(crear(documento, "p", t("sobrelinea"), "sobrelinea"), crear(documento, "h2", t("titulo")), crear(documento, "p", t("descripcion"))); const aviso = crear(documento, "p", DATOS_ADMINISTRACION_PRESENTACION.aviso, "administracion-aviso"); const entrega = crear(documento, "div"); entrega.innerHTML = renderizarEstadoEntrega(estadoEntrega); const nav = crear(documento, "nav", "", "administracion-pestanas"); nav.setAttribute("role", "tablist"); nav.setAttribute("aria-label", t("pestanas")); PESTANAS_ADMINISTRACION.forEach((clave) => { const boton = crear(documento, "button", t(`tab_${clave}`)); boton.type = "button"; boton.dataset.administracionPestana = clave; boton.setAttribute("role", "tab"); boton.setAttribute("aria-selected", String(clave === estado.pestana)); boton.addEventListener("click", () => { estado = { ...estado, pestana: clave, filtro: "" }; pintar(); anunciar(t("seccion_seleccionada", { etiqueta: t(`tab_${clave}`) }), "info"); }); nav.append(boton); }); const filtros = crear(documento, "form", "administracion-filtros"); const label = crear(documento, "label", t("filtro")); const input = crear(documento, "input"); input.name = "filtro"; input.value = estado.filtro; input.maxLength = 120; input.autocomplete = "off"; label.append(input); const aplicar = crear(documento, "button", t("aplicar_filtro"), "boton-secundario"); aplicar.type = "submit"; filtros.append(label, aplicar); filtros.addEventListener("submit", (evento) => { evento.preventDefault(); estado = { ...estado, filtro: input.value.trim() }; pintar(); anunciar(t("filtro_aplicado"), "info"); }); seccion.append(cabecera, aviso, entrega, nav); if (!["resumen", "ia"].includes(estado.pestana)) seccion.append(filtros); seccion.append(contenido(documento, t, estado.pestana, estado, (cambios) => { estado = { ...estado, ...cambios }; pintar(); })); raiz.append(seccion); };
  pintar(); const desmontar = () => { if (!activa) return; activa = false; raiz.replaceChildren(); }; registrarDesmontar?.(desmontar); return Object.freeze({ desmontar });
}
