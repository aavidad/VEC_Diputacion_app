import { aDecimal } from "./baremo-editor.js?v=20261001-concursos-v6";
export function escapar(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}
export function renderizarBaremo(estado, { textos, ejemplos = [], filtro = "" }) {
  const tr = (clave, variables = {}) => textos.traducir(`editor.${clave}`, variables);
  const t = (clave, variables = {}) => escapar(tr(clave, variables));
  const s = estado.borrador;
  const puntos = (valor) => {
    try {
      const [entero, fraccion] = aDecimal(valor).split(".");
      const separador = new Intl.NumberFormat(textos.localizacion).formatToParts(1.5).find((p) => p.type === "decimal").value;
      return escapar(`${textos.numero(BigInt(entero))}${fraccion ? separador + fraccion : ""}`);
    } catch { return t("dato_no_disponible"); }
  };
  // Solo cambia la representación de los valores exactos devueltos por el motor.
  const puntosExactos = (valor) => {
    if (typeof valor !== "string") return t("dato_no_disponible");
    const fraccion = /^(0|[1-9][0-9]{0,79})\/([1-9][0-9]{0,79})$/u.exec(valor);
    if (!fraccion) return puntos(valor);
    const numerador = BigInt(fraccion[1]), denominador = BigInt(fraccion[2]);
    if (numerador % denominador === 0n) return puntos(String(numerador / denominador));
    return t("puntos_fraccion", { numerador: puntos(fraccion[1]), denominador: textos.numero(denominador) });
  };
  const limiteApartado = (apartado) => {
    if (Object.hasOwn(apartado, "tope") && apartado.tope && Object.hasOwn(apartado.tope, "limite")) {
      return apartado.tope.limite === null ? t("sin_tope") : puntosExactos(apartado.tope.limite);
    }
    return puntosExactos(apartado.maximo_puntos);
  };
  const seccion = (clave, n) => textos.mensajes.editor[`seccion_${clave}`] ? t(`seccion_${clave}`) : t("seccion_generica", { n });
  const idCampo = (ruta) => `baremo-campo-${ruta.join("-")}`;
  const campo = (ruta, valor, etiqueta) => `<label class="campo"><span>${etiqueta}</span><input id="${escapar(idCampo(ruta))}" aria-describedby="${escapar(idCampo(ruta))}-error" type="text" inputmode="decimal" required pattern="(0|[1-9][0-9]{0,12})([.,][0-9]{1,6})?" data-ruta="${escapar(JSON.stringify(ruta))}" data-puntos value="${escapar(estado.invalidos?.[JSON.stringify(ruta)] ?? aDecimal(valor))}"${Object.hasOwn(estado.invalidos ?? {}, JSON.stringify(ruta)) ? ' aria-invalid="true"' : ""} aria-label="${etiqueta}"><span id="${escapar(idCampo(ruta))}-error" data-error-campo hidden></span></label>`;
  const cabecera = `<header class="cabecera-panel"><h1>${t("titulo")}</h1><button type="button" class="boton-secundario" data-accion="ayuda" aria-expanded="${Boolean(estado.ayuda)}" aria-controls="baremo-ayuda">${t("ayuda")}</button></header>`;
  if (!s) return `<section class="panel">${cabecera}<div class="cuerpo-panel"><p role="status">${t(estado.error || "cargando")}</p>${estado.error ? `<button type="button" class="boton-secundario" data-accion="recargar">${t("reintentar")}</button>` : ""}</div></section>`;
  const fila = (regla, i) => {
    const ruta = [s.reglas_experiencia ? "reglas_experiencia" : "reglas", i];
    return `<tr><th scope="row">${t("regla_nombre", { n: i + 1 })}<span class="baremo-condicion">${regla.clase ? t(textos.mensajes.editor[`clase_${regla.clase}`] ? `clase_${regla.clase}` : "condicion_catalogo") : (regla.criterios ?? []).flatMap((c) => c.valores).map((v) => t(textos.mensajes.editor[`condicion_${v}`] ? `condicion_${v}` : "condicion_catalogo")).join(" · ")}</span></th><td>${seccion(regla.seccion_clave, i + 1)}</td><td>${campo([...ruta, "puntos_por_unidad"], regla.puntos_por_unidad, t("puntuacion"))}</td><td>${t(`unidad_${regla.unidad_temporal?.unidad_puntuable ?? regla.unidad ?? "merito"}`)}</td><td>${typeof regla.maximo_puntos === "string" ? campo([...ruta, "maximo_puntos"], regla.maximo_puntos, t("maximo_regla")) : regla.maximo_puntos?.modo === "limitado" ? campo([...ruta, "maximo_puntos", "valor"], regla.maximo_puntos.valor, t("maximo_regla")) : t("sin_tope")}</td></tr>`;
  };
  const resultados = estado.comparacion ? ["antes", "despues"].map((lado) => {
    const r = estado.comparacion[lado].resultado;
    return `<section class="baremo-resultado"><h3>${t(lado)}</h3>${r.estado === "bloqueado" ? `<p role="status">${t("bloqueado")}</p><ul>${(r.bloqueos ?? r.incidencias ?? []).map((b) => `<li>${t(textos.mensajes.editor[`bloqueo_${b.codigo}`] ? `bloqueo_${b.codigo}` : "bloqueo_desconocido")}</li>`).join("")}</ul>` : `<dl><div><dt>${t("total")}</dt><dd>${puntos(r.total)}</dd></div></dl><div class="tabla-contenedor" tabindex="0" aria-label="${t("desglose")}"><table class="tabla-datos"><caption>${t("desglose")}</caption><thead><tr><th scope="col">${t("apartado")}</th><th scope="col" class="columna-numero">${t("antes_tope")}</th><th scope="col" class="columna-numero">${t("maximo_apartado")}</th><th scope="col" class="columna-numero">${t("despues_tope")}</th></tr></thead><tbody>${(r.secciones ?? []).map((apartado, n) => `<tr><th scope="row">${seccion(apartado.seccion_clave ?? apartado.seccion ?? apartado.clave, n + 1)}</th><td class="columna-numero">${puntosExactos(apartado.antes_tope ?? apartado.suma_reglas)}</td><td class="columna-numero">${limiteApartado(apartado)}</td><td class="columna-numero">${puntos(apartado.subtotal ?? apartado.puntos_finales ?? apartado.puntos)}</td></tr>`).join("")}</tbody></table></div>`}</section>`;
  }).join("") : `<p role="status">${t(estado.trabajando ? "trabajando" : "pendiente")}</p>`;
  return `<section class="baremo-editor" aria-label="${t("titulo")}"><div class="panel">${cabecera}<div class="cuerpo-panel baremo-cabecera"><span class="estado-modulo">${t("estado")}</span><span>${t("version", { version: s.identidad?.version ?? s.version })}</span><span role="status">${t(estado.cambiado ? "sin_guardar" : "sin_cambios")}</span><p class="baremo-limite">${t("limite")}</p><p id="baremo-ayuda" ${estado.ayuda ? "" : "hidden"}>${t("ayuda_texto")}</p></div></div>
    <form id="baremo-formulario" novalidate><section class="panel"><div class="cuerpo-panel baremo-controles"><label class="campo"><span>${t("ejemplo")}</span><select name="ejemplo">${ejemplos.map((e, n) => `<option value="${escapar(e.referencia)}"${e.referencia === estado.ejemplo.referencia ? " selected" : ""}>${t(`caso_${e.modo}`, { n: n + 1 })}</option>`).join("")}</select></label><label class="campo"><span>${t("fecha")}</span><input id="baremo-fecha" aria-label="${t("fecha")}" aria-describedby="baremo-fecha-error" type="date" name="fecha" required value="${escapar(s.fecha_corte_inclusiva)}" data-ruta='["fecha_corte_inclusiva"]'><span id="baremo-fecha-error" data-error-campo hidden></span></label><label class="boton-secundario baremo-importar">${t("cargar")}<input type="file" accept=".json,application/json" name="archivo"></label><button type="button" class="boton-secundario" data-accion="restablecer">${t("restablecer")}</button></div></section>
    <div class="baremo-trabajo"><section class="panel baremo-reglas"><div class="cabecera-panel"><h2>${t("reglas")}</h2><label class="campo"><span>${t("filtrar")}</span><input type="search" name="filtro" value="${escapar(filtro)}"></label></div><div class="tabla-contenedor" tabindex="0" aria-label="${t("reglas")}"><table class="tabla-datos"><thead><tr><th scope="col">${t("regla")}</th><th scope="col">${t("apartado")}</th><th scope="col">${t("puntuacion")}</th><th scope="col">${t("unidad")}</th><th scope="col">${t("maximo_regla")}</th></tr></thead><tbody>${(s.reglas_experiencia ?? s.reglas).map((r, i) => fila(r, i)).join("")}<tr data-sin-reglas hidden><td colspan="5">${t("sin_reglas")}</td></tr></tbody></table></div></section>
    <section class="panel baremo-topes"><div class="cabecera-panel"><h2>${t("estructura")}</h2></div><div class="cuerpo-panel">${s.maximo_total ? campo(["maximo_total"], s.maximo_total, t("tope_total")) : ""}${s.secciones.map((apartado, i) => campo(["secciones", i, Object.hasOwn(apartado, "puntos_maximos") ? "puntos_maximos" : "maximo_puntos"], apartado.puntos_maximos ?? apartado.maximo_puntos, `${seccion(apartado.clave, i + 1)} · ${t("tope")}`)).join("")}</div></section></div>
    <section class="panel"><div class="cabecera-panel"><h2>${t("resultado")}</h2></div><div class="cuerpo-panel baremo-resultados" aria-live="polite" aria-busy="${estado.trabajando}">${resultados}</div></section>
    <footer class="panel cuerpo-panel baremo-acciones"><div id="baremo-error" role="alert" ${estado.error ? "" : "hidden"}>${estado.error ? t(estado.error) : ""}</div><button type="button" class="boton-secundario" data-accion="exportar"${Object.keys(estado.invalidos ?? {}).length ? " disabled" : ""}>${t("exportar")}</button><button type="submit" class="boton-primario"${estado.trabajando ? " disabled" : ""}>${t(estado.trabajando ? "trabajando" : "comparar")}</button><button type="button" class="boton-secundario" disabled aria-describedby="baremo-activacion">${t("activar")}</button><p id="baremo-activacion">${t("activacion_pendiente")}</p></footer></form></section>`;
}

/** Cada proceso conserva su editor y su cálculo; solo comparten navegación. */
export function renderizarPanelesBaremo(estado, { textos, panel = "bolsa", concursosHTML = "", ...opciones }) {
  const t = (clave) => escapar(textos.traducir(`editor.${clave}`));
  const activo = panel === "concursos" ? "concursos" : "bolsa";
  const boton = (clave) => `<button type="button" class="${activo === clave ? "boton-primario" : "boton-secundario"}" data-panel="${clave}" aria-controls="baremo-panel-${clave}"${activo === clave ? ' aria-current="page"' : ""}>${t(clave)}</button>`;
  return `<div class="baremo-modulos"><nav class="baremo-navegacion" aria-label="${t("tipo_proceso")}">${boton("bolsa")}${boton("concursos")}</nav>
    <div id="baremo-panel-bolsa"${activo === "bolsa" ? "" : " hidden"}>${renderizarBaremo(estado, { textos, ...opciones })}</div>
    <section id="baremo-panel-concursos"${activo === "concursos" ? "" : " hidden"}>${concursosHTML || `<section class="panel"><header class="cabecera-panel"><h1>${t("concursos")}</h1></header><div class="cuerpo-panel"><h2>${t("concursos_vacio")}</h2><p>${t("concursos_siguiente")}</p><p class="baremo-limite">${t("concursos_limite")}</p></div></section>`}</section></div>`;
}
