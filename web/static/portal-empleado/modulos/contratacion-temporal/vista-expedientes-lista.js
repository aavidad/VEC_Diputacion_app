/**
 * Lista de peticiones según la dirección de diseño del 29/09/2026: «N
 * peticiones en trámite» (el mismo número que la portada), buscador que
 * filtra al escribir, filtros de fase, centro, categoría y «Mostrar», etiquetas
 * de filtros activos que se quitan con un clic y una tabla de cinco columnas
 * ordenada por plazo que en móvil pasa a fichas.
 *
 * Filtra en pantalla sobre la consulta autorizada ya cargada; los filtros que
 * pidió el servidor (estado o fase de origen) se muestran también como
 * etiquetas quitables. HTML puro: los eventos los atiende vista-expedientes.js.
 */
import { FASES_RRHH, faseRRHH } from "./fases-rrhh-datos.js?v=20261007-pantallas-textos-final-v1";
import { diaConsulta, diasEntre, filtrarPeticiones, OPCIONES_MOSTRAR, resumirPeticiones } from "./recuentos-peticiones.js?v=20261007-pantallas-textos-final-v1";

function escapar(valor) {
  return String(valor ?? "")
    .replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;").replaceAll("'", "&#039;");
}

function opcion(valor, etiqueta, seleccionado) {
  return `<option value="${escapar(valor)}"${valor === seleccionado ? " selected" : ""}>${escapar(etiqueta)}</option>`;
}

function distintos(expedientes, campo) {
  return [...new Set(expedientes.map((e) => e[campo]).filter((v) => typeof v === "string" && v !== ""))]
    .sort((a, b) => a.localeCompare(b, "es"));
}

// «Mostrar» por omisión es «En trámite»; si el servidor ya filtró terminadas,
// no se esconden en pantalla.
function filtroEfectivo(estado, filtro) {
  const terminadasPedidas = ["completado", "cancelado"].includes(estado.filtros?.estado);
  return terminadasPedidas && filtro.mostrar === "en_tramite" ? { ...filtro, mostrar: "todas" } : filtro;
}

function etiquetaRelativa(expediente, generadoEn, t) {
  const hoy = diaConsulta(generadoEn);
  const ultimoDia = expediente.plazo_ultimo_dia;
  if (!hoy || !/^\d{4}-\d{2}-\d{2}$/u.test(ultimoDia ?? "")) return "";
  const dias = diasEntre(hoy, ultimoDia);
  if (expediente.plazo_estado === "vencido") {
    return dias < 0 ? t(dias === -1 ? "lista_plazo_vencio_ayer" : "lista_plazo_vencio_hace", { dias: -dias }) : t("lista_plazo_vencio_hoy");
  }
  if (expediente.plazo_estado === "vence_hoy" || dias === 0) return t("lista_plazo_vence_hoy");
  return dias > 0 ? t(dias === 1 ? "lista_plazo_vence_manana" : "lista_plazo_vence_en", { dias }) : "";
}

function celdaPlazo(expediente, t, generadoEn) {
  if (!expediente.plazo_estado || expediente.plazo_estado === "no_calculado") {
    return `<span class="texto-secundario">${escapar(expediente.plazo_estado ? expediente.plazo : t("lista_sin_plazo"))}</span>`;
  }
  const urgente = ["vencido", "vence_hoy"].includes(expediente.plazo_estado);
  const relativa = etiquetaRelativa(expediente, generadoEn, t);
  return `${urgente ? `<span class="ct-exp-chip ct-plazo-${escapar(expediente.plazo_estado)}">${escapar(t(`plazo_fase_${expediente.plazo_estado}`))}</span>` : ""}
    <small>${relativa ? `${escapar(relativa)} · ` : ""}${escapar(expediente.plazo)}</small>`;
}

function fila(expediente, t, { numeroVisible, centroVisible }, generadoEn) {
  const fase = faseRRHH(expediente.fase_clave);
  const centro = centroVisible(expediente.centro);
  const numero = numeroVisible(expediente.numero_visible);
  return `<tr class="ct-exp-fila" data-ct-fase="${escapar(fase?.clave ?? "otra")}">
    <th scope="row" class="celda-titulo"><button type="button" class="enlace-tabla" data-ct-exp-abrir="${escapar(expediente.expediente_ref)}"
      aria-label="${escapar(t("lista_abrir_aria", { expediente: numero }))}">${escapar(numero)}</button>${expediente.urgente
    ? ` <span class="ct-marca-urgente">${escapar(t("marca_urgente"))}</span>` : ""}</th>
    <td class="envuelve" data-etiqueta="${escapar(t("lista_col_centro_categoria"))}"${centro.referencia ? ` title="${escapar(centro.referencia)}"` : ""}>${escapar(centro.etiqueta)}<small>${escapar(expediente.categoria)}</small></td>
    <td data-etiqueta="${escapar(t("lista_col_fase"))}">${escapar(expediente.fase_actual)}${fase
    ? `<small>${escapar(t("fase_rrhh_orden", { orden: fase.orden, total: fase.total }))}</small>` : ""}</td>
    <td data-etiqueta="${escapar(t("lista_col_estado"))}"><span class="ct-exp-chip ct-fase-${escapar(expediente.estado_clave)}">${escapar(expediente.estado)}</span></td>
    <td data-etiqueta="${escapar(t("lista_col_plazo"))}">${celdaPlazo(expediente, t, generadoEn)}</td>
  </tr>`;
}

function etiquetasActivas(estado, filtro, t, ayudas, opcionesFaseServidor = null) {
  const chips = [];
  const chip = (clave, texto) => chips.push(`<button type="button" class="chip-quitar" data-ct-exp-quitar-filtro="${escapar(clave)}"
    aria-label="${escapar(t("lista_quitar_filtro", { filtro: texto }))}">${escapar(texto)} <span aria-hidden="true">×</span></button>`);
  if (filtro.texto) chip("texto", t("lista_filtro_texto", { valor: filtro.texto }));
  const claveFase = opcionesFaseServidor?.find(([clave]) => clave === filtro.fase)?.[1];
  if (filtro.fase) chip("fase", t(claveFase ?? `etiqueta_fase_${filtro.fase}`));
  if (filtro.centro) chip("centro", ayudas.centroVisible(filtro.centro).etiqueta);
  if (filtro.categoria) chip("categoria", filtro.categoria);
  if (filtro.mostrar !== (opcionesFaseServidor ? "todas" : "en_tramite"))
    chip("mostrar", t(`lista_mostrar_${filtro.mostrar}`));
  const servidor = estado.filtros ?? {};
  if (servidor.estado) chip("servidor", t("lista_filtro_servidor_estado", { valor: t(`fase_${servidor.estado}`) }));
  else if (servidor.fase || servidor.texto) chip("servidor", [servidor.fase, servidor.texto].filter(Boolean).join(" · "));
  if (chips.length === 0) return "";
  return `<div class="filtros-activos"><span>${escapar(t("lista_filtros_activos"))}</span>${chips.join("")}
    <button type="button" class="boton-terciario" data-ct-exp-quitar-filtro="todos">${escapar(t("lista_quitar_todos"))}</button></div>`;
}

// Solo los filtros que realmente se aplican en pantalla se limitan a la página
// cargada. Un texto o estado ya aplicado por el servidor no lleva este aviso.
function busquedaParcial(cuadro, filtro) {
  return Boolean(cuadro.paginacion?.cursor_siguiente)
    && Boolean(filtro.texto || filtro.fase || filtro.centro || filtro.categoria
      || !["en_tramite", "todas"].includes(filtro.mostrar));
}

/** Etiquetas, recuento y tabla: la parte que cambia al escribir o elegir un filtro. */
export function renderizarResultadosLista(estado, t, filtroEntrada, ayudas, filtroBusqueda = filtroEntrada,
  totalConjunto = null, cuentaSoloPagina = false, opcionesFaseServidor = null) {
  const cuadro = estado.cuadro;
  const filtro = filtroEfectivo(estado, filtroEntrada);
  const filas = filtrarPeticiones(cuadro.expedientes, filtroEfectivo(estado, filtroBusqueda), cuadro.generado_en, (expediente) => [
    ayudas.numeroVisible(expediente.numero_visible),
    ayudas.centroVisible(expediente.centro).etiqueta,
  ]);
  return `<div data-ct-exp-resultados>
    ${etiquetasActivas(estado, filtroEntrada, t, ayudas, opcionesFaseServidor)}
    <p class="solo-lectura" role="status" aria-live="polite">${escapar(cuentaSoloPagina
      ? t("lista_resultados_pagina", { total: filas.length })
      : t("lista_resultados", { total: filas.length, de: totalConjunto ?? cuadro.expedientes.length }))}</p>
    ${busquedaParcial(cuadro, filtroBusqueda) ? `<p class="ct-exp-lista-parcial" role="status" data-ct-exp-busqueda-parcial>${escapar(t("lista_busqueda_parcial"))}</p>` : ""}
    ${filas.length === 0
    ? `<p class="cuerpo-panel vacio-controlado" role="status">${escapar(t(cuadro.expedientes.length === 0 && !opcionesFaseServidor
      ? "lista_vacia_crear" : "lista_sin_resultados"))}</p>`
    : `<div class="ct-exp-tabla-lista"><table class="tabla-datos tabla-apilable ct-exp-tabla-peticiones">
      <caption class="solo-lectura">${escapar(t("tabla_expedientes"))}</caption>
      <thead><tr>
        <th scope="col">${escapar(t("lista_col_expediente"))}</th>
        <th scope="col">${escapar(t("lista_col_centro_categoria"))}</th>
        <th scope="col">${escapar(t("lista_col_fase"))}</th>
        <th scope="col">${escapar(t("lista_col_estado"))}</th>
        <th scope="col">${escapar(t("lista_col_plazo"))}</th>
      </tr></thead>
      <tbody>${filas.map((expediente) => fila(expediente, t, ayudas, cuadro.generado_en)).join("")}</tbody>
    </table></div>`}
  </div>`;
}

/** Pantalla completa de la lista. `ayudas` aporta número y centro legibles. */
export function renderizarListaPeticiones(estado, t, filtro, ayudas, paginacion = "",
  { altaDisponible = true, actualizarDisponible = false, filtroResultados = filtro,
    totalConjunto = null, enTramiteConjunto = null, paginaAnterior = false,
    opcionesFaseServidor = null, opcionesMostrarServidor = null,
    buscadorServidor = false, tituloConjunto = false, ocultarFiltrosLocales = false,
    totalConjuntoExacto = false, filtrosDisponiblesVacios = false } = {}) {
  const cuadro = estado.cuadro;
  const resumen = resumirPeticiones({ expedientes: cuadro.expedientes });
  const parcial = Boolean(cuadro.paginacion?.cursor_siguiente);
  const enTramite = enTramiteConjunto ?? (parcial || paginaAnterior ? null : resumen.enTramite);
  const titulo = tituloConjunto ? t("lista_titulo_conjunto") : enTramite === null ? t("tabla_expedientes")
    : enTramite === 0 ? t("lista_titulo_ninguna")
      : (enTramite === 1 ? t("lista_titulo_uno") : t("lista_titulo_varias", { total: enTramite }));
  // Sin ninguna petición ni filtro del servidor, sobran buscador y filtros.
  const sinPeticiones = cuadro.expedientes.length === 0 && !parcial && !filtrosDisponiblesVacios
    && !Object.values(estado.filtros ?? {}).some((valor) => valor !== "" && valor != null)
    && (!tituloConjunto || !filtro.texto && !filtro.fase && filtro.mostrar === "todas");
  const centros = distintos(cuadro.expedientes, "centro");
  const categorias = distintos(cuadro.expedientes, "categoria");
  return `<header class="cabeza-pagina">
      <div><h3 class="ct-exp-lista-titulo">${escapar(titulo)}</h3></div>
      ${altaDisponible ? `<button type="button" class="boton-primario" data-ct-exp-vista="alta">${escapar(t("lista_nueva_peticion"))}</button>` : ""}
      ${actualizarDisponible ? `<button type="button" class="boton-terciario" data-ct-exp-recargar>${escapar(t("lista_actualizar"))}</button>` : ""}
    </header>
    ${parcial && !(totalConjuntoExacto && Number.isSafeInteger(totalConjunto) && totalConjunto >= 0)
      ? `<p class="ct-exp-lista-parcial" role="status">${escapar(t("lista_recuento_parcial"))}</p>` : ""}
    <section class="panel ct-exp-listado" aria-labelledby="ct-exp-lista-titulo-panel">
      <h3 class="solo-lectura" id="ct-exp-lista-titulo-panel">${escapar(t("tabla_expedientes"))}</h3>
      ${sinPeticiones ? "" : `<form class="filtros-quitables" data-ct-exp-filtros-locales role="search" aria-label="${escapar(t("filtros"))}">
        <label><span>${escapar(t("lista_buscar"))}</span>
          <input type="search" name="texto" value="${escapar(filtro.texto)}" maxlength="80" autocomplete="off"
            placeholder="${escapar(t(buscadorServidor ? "lista_buscar_pista_servidor" : "lista_buscar_pista"))}"></label>
        <label><span>${escapar(t("lista_fase"))}</span>
          <select name="fase">${opcion("", t("lista_fase_todas"), filtro.fase)}${(opcionesFaseServidor
    ? opcionesFaseServidor.map(([fase, clave]) => opcion(fase, t(clave), filtro.fase))
    : FASES_RRHH.map((fase, indice) => opcion(fase, `${indice + 1}. ${t(`etiqueta_fase_${fase}`)}`, filtro.fase))).join("")}</select></label>
        <details class="mas-filtros" open data-ct-exp-mas-filtros${filtro.centro || filtro.categoria
          || filtro.mostrar !== (opcionesMostrarServidor ? "todas" : "en_tramite") ? " data-activos" : ""}>
          <summary>${escapar(t("lista_mas_filtros"))}</summary>
          <div class="mas-filtros-cuerpo">
            ${ocultarFiltrosLocales ? "" : `<label><span>${escapar(t("lista_centro"))}</span>
              <select name="centro">${opcion("", t("lista_centro_todos"), filtro.centro)}${centros.map((c) => opcion(c, ayudas.centroVisible(c).etiqueta, filtro.centro)).join("")}</select></label>
            <label><span>${escapar(t("lista_categoria"))}</span>
              <select name="categoria">${opcion("", t("lista_categoria_todas"), filtro.categoria)}${categorias.map((c) => opcion(c, c, filtro.categoria)).join("")}</select></label>`}
            <label><span>${escapar(t("lista_mostrar"))}</span>
              <select name="mostrar">${(opcionesMostrarServidor ?? OPCIONES_MOSTRAR.map((clave) => [clave, `lista_mostrar_${clave}`]))
                .map(([clave, etiqueta]) => opcion(clave, t(etiqueta), filtro.mostrar)).join("")}</select></label>
          </div>
        </details>
      </form>`}
      ${sinPeticiones ? `<p class="cuerpo-panel vacio-controlado" role="status">${escapar(t(altaDisponible
    ? "lista_vacia_crear" : "lista_vacia_sin_alta"))}</p>`
    : renderizarResultadosLista(estado, t, filtro, ayudas, filtroResultados,
      totalConjunto, totalConjunto === null && (parcial || paginaAnterior), opcionesFaseServidor)}
      ${paginacion}
    </section>`;
}
