import { renderizarEstadoEntrega } from "../../estado-entrega.js";
import { obtenerAtlasSinteticoRRHH, TEXTO_DATOS_FICTICIOS_RRHH } from "../../datos-sinteticos-rrhh.js";
import { DATOS_MERITOS_PRESENTACION } from "./datos-presentacion.js";
import { crearTraductorMeritos } from "./i18n.js";

const VISTAS = Object.freeze([
  ["resumen", "resumen"], ["titulos", "titulos"], ["cursos", "cursos"],
  ["servicios", "servicios"], ["evidencias", "evidencias"], ["homologaciones", "homologaciones"], ["formacion", "formacion"], ["procesos", "procesos"],
]);
const escap = (valor) => String(valor).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
const sigue = (raiz, contenedor) => raiz.querySelector?.("[data-modulo-meritos]") === contenedor;
const tabla = (titulo, cabeceras, filas) => `<div class="meritos-tabla-contenedor"><table class="meritos-tabla"><caption>${escap(titulo)}</caption><thead><tr>${cabeceras.map((x) => `<th scope="col">${escap(x)}</th>`).join("")}</tr></thead><tbody>${filas.map((fila) => `<tr>${fila.map((celda) => `<td>${escap(celda)}</td>`).join("")}</tr>`).join("")}</tbody></table></div>`;
const accion = (texto, motivo) => `<div class="meritos-accion"><button type="button" disabled aria-disabled="true">${escap(texto)}</button><small>${escap(motivo)}</small></div>`;
const cabeceraPanel = (titulo, ayuda) => `<header><h3>${escap(titulo)}</h3><p>${escap(ayuda)}</p></header>`;
const estado = (t) => renderizarEstadoEntrega({ estado: "visual_pendiente_backend", resumen: t("estado_resumen"), pendientes: ["falta_fuente", "falta_evidencia", "falta_validacion", "falta_autorizacion", "falta_historial"].map(t), fuente: { etiqueta: t("fuente_catalogo") }, conexion: t("sin_conexion") });

function contenido(vista, filtro, atlas, t) {
  const d = DATOS_MERITOS_PRESENTACION;
  const seleccion = filtro === "todos" ? null : filtro;
  if (vista === "resumen") return `<div class="meritos-resumen">${[[d.titulos.length, "titulos_declarados"], [d.cursos.length, "cursos_mostrados"], [d.servicios.length, "servicios_mostrados"], [d.evidencias.length, "evidencias_conectar"]].map(([n, x]) => `<article class="meritos-kpi"><strong>${n}</strong><span>${t(x)}</span></article>`).join("")}</div><section class="meritos-panel">${cabeceraPanel(t("expediente"), `${atlas.persona_principal.nombre_visible} · ${atlas.unidad.nombre_visible}`)}<p class="meritos-ayuda">${t("estados_demo")}</p>${accion(t("aportacion"), t("accion_general_pendiente"))}</section>`;
  const fuentes = { titulos: ["titulos_tabla", ["cab_titulo", "cab_entidad", "cab_ano", "cab_estado"], d.titulos.map((x) => [x.nombre, x.entidad, x.fecha, x.estado])], cursos: ["cursos_tabla", ["cab_curso", "cab_entidad", "cab_duracion", "cab_estado"], d.cursos.map((x) => [x.nombre, x.entidad, x.horas, x.estado])], servicios: ["servicios_tabla", ["cab_puesto", "cab_centro", "cab_periodo", "cab_estado"], d.servicios.map((x) => [x.puesto, x.centro, x.periodo, x.estado])], evidencias: ["evidencias_tabla", ["cab_evidencia", "cab_tipo", "cab_procedencia", "cab_estado"], d.evidencias.map((x) => [x.nombre, x.tipo, x.procedencia, x.estado])], formacion: ["formacion_tabla", ["cab_actividad", "cab_modalidad", "cab_plazas", "cab_fechas"], d.formacion.map((x) => [x.nombre, x.modalidad, x.plazas, x.fecha])], procesos: ["procesos_tabla", ["cab_proceso", "cab_situacion", "cab_aplicacion"], d.procesos.map((x) => [x.nombre, x.fase, x.aplicacion])] };
  if (vista === "homologaciones") return `<section class="meritos-panel">${cabeceraPanel(t("homologaciones_titulo"), t("homologaciones_ayuda"))}<ul class="meritos-lista"><li><strong>${t("homologacion_uno")}</strong><span>${t("homologacion_uno_ayuda")}</span></li><li><strong>${t("homologacion_dos")}</strong><span>${t("homologacion_dos_ayuda")}</span></li></ul>${accion(t("solicitar_homologacion"), t("homologacion_pendiente"))}</section>`;
  const [tituloClave, cabeceras, filas] = fuentes[vista]; const titulo = t(tituloClave);
  const visibles = seleccion ? filas.filter((fila) => fila.join(" ").toLocaleLowerCase().includes(seleccion)) : filas;
  const boton = t({ titulos: "aportar_titulo", cursos: "aportar_curso", servicios: "contraste", evidencias: "validar", formacion: "inscribir", procesos: "reutilizar" }[vista]);
  return `<section class="meritos-panel">${cabeceraPanel(titulo, t("consulta_local"))}<div class="meritos-filtros"><label>${t("filtro")}<select data-meritos-filtro aria-label="${t("filtro")}"><option value="todos">${t("todos")}</option><option value="pendiente">${t("pendientes")}</option><option value="acreditado">${t("acreditados")}</option></select></label><span class="meritos-chip" role="status">${t("registros_visibles", { numero: visibles.length })}</span></div>${tabla(titulo, cabeceras.map(t), visibles)}${accion(boton, t("accion_pendiente"))}</section>`;
}

/** Monta una superficie local de demostración sin red, persistencia ni efectos administrativos. */
export function montarVistaMeritos({ raiz, anunciar = () => {}, registrarDesmontar } = {}) {
  if (!raiz?.append || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("vista de Méritos no disponible");
  const documento = raiz.ownerDocument;
  if (!documento?.createElement) throw new TypeError("documento de Méritos no disponible");
  const atlas = obtenerAtlasSinteticoRRHH();
  const t = crearTraductorMeritos();
  const contenedor = documento.createElement("section"); contenedor.className = "modulo-meritos"; contenedor.dataset.moduloMeritos = ""; raiz.append(contenedor);
  let activa = true; let actual = "resumen"; let filtro = "todos";
  const pintar = (vista = actual) => { if (!activa || !sigue(raiz, contenedor)) return; actual = vista; contenedor.innerHTML = `<header class="meritos-cabecera"><div><p class="meritos-sobrelinea">${t("area_personal")}</p><h2>${t("titulo")}</h2><p>${t("descripcion")}</p></div><span class="meritos-demo" role="status">${escap(t("sin_validez", { aviso: TEXTO_DATOS_FICTICIOS_RRHH }))}</span></header><nav class="meritos-tabs" role="tablist" aria-label="${t("navegacion")}">${VISTAS.map(([clave, etiqueta]) => `<button type="button" role="tab" data-meritos-vista="${clave}" aria-selected="${clave === actual}" tabindex="${clave === actual ? "0" : "-1"}">${t(etiqueta)}</button>`).join("")}</nav>${estado(t)}<main class="meritos-grid"><div role="tabpanel">${contenido(actual, filtro, atlas, t)}</div><aside class="meritos-panel">${cabeceraPanel(t("datos_demostracion"), t("personas_ficticias"))}<ul class="meritos-lista"><li><strong>${escap(atlas.persona_principal.nombre_visible)}</strong><span>${escap(atlas.unidad.nombre_visible)} · ${escap(atlas.centro.nombre_visible)}</span></li></ul><p class="meritos-ayuda">${t("sin_fuente")}</p>${accion(t("exportar"), t("exportar_pendiente"))}</aside></main>`; };
  pintar(); anunciar(t("anuncio_inicial"), "informacion");
  const alClick = (evento) => { const boton = evento.target?.closest?.("[data-meritos-vista]"); if (!boton) return; filtro = "todos"; pintar(boton.dataset.meritosVista); anunciar(t("anuncio_seccion", { nombre: boton.textContent }), "informacion"); };
  const alCambio = (evento) => { if (!evento.target?.matches?.("[data-meritos-filtro]")) return; filtro = evento.target.value; pintar(); };
  const alTecla = (evento) => { const boton = evento.target?.closest?.("[data-meritos-vista]"); if (!boton || !["ArrowRight", "ArrowLeft", "Home", "End"].includes(evento.key)) return; const indice = VISTAS.findIndex(([clave]) => clave === boton.dataset.meritosVista); const salto = evento.key === "Home" ? -indice : evento.key === "End" ? VISTAS.length - indice - 1 : evento.key === "ArrowRight" ? 1 : -1; evento.preventDefault(); const destino = VISTAS[(indice + salto + VISTAS.length) % VISTAS.length][0]; filtro = "todos"; pintar(destino); contenedor.querySelector?.(`[data-meritos-vista="${destino}"]`)?.focus?.(); };
  contenedor.addEventListener("click", alClick); contenedor.addEventListener("change", alCambio); contenedor.addEventListener("keydown", alTecla);
  const desmontar = () => { if (!activa) return; activa = false; contenedor.removeEventListener("click", alClick); contenedor.removeEventListener("change", alCambio); contenedor.removeEventListener("keydown", alTecla); if (sigue(raiz, contenedor)) contenedor.remove(); };
  registrarDesmontar?.(desmontar); return Object.freeze({ desmontar });
}
