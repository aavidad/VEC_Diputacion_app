import { aDecimal, errorRestosRegla } from "./baremo-editor.js?v=20261002-a-restos-recuperacion-v1";
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
  const jornada = (regla, i) => {
    const opciones = estado.catalogoJornada?.opciones ?? [], modo = regla.jornada?.modo ?? "";
    const actual = opciones.find((o) => o.modo === modo), ruta = ["reglas_experiencia", i, "jornada", "modo"];
    const id = idCampo(ruta), umbralRuta = ["reglas_experiencia", i, "jornada", "umbral"], umbralId = idCampo(umbralRuta);
    const invalidado = Object.hasOwn(estado.invalidos ?? {}, JSON.stringify(umbralRuta));
    const motivo = !opciones.length ? t("catalogo_jornada_no_disponible") : actual && !actual.disponible ? t(actual.motivo) : !actual ? t("jornada_no_disponible") : opciones.filter((o) => !o.disponible).map((o) => t(o.motivo)).join(" ");
    return `<td><label class="campo"><span>${t("jornada")}</span><select id="${escapar(id)}" data-ruta="${escapar(JSON.stringify(ruta))}" data-jornada-modo data-indice="${i}" aria-label="${t("jornada")}" aria-describedby="${escapar(id)}-motivo"${opciones.length ? "" : " disabled"}>${!actual ? `<option value="${escapar(modo)}" selected disabled>${t(textos.mensajes.editor[`jornada_${modo}`] ? `jornada_${modo}` : "jornada_no_disponible")}</option>` : ""}${opciones.map((o) => `<option value="${escapar(o.modo)}"${o.modo === modo ? " selected" : ""}${o.disponible ? "" : " disabled"}>${t(o.etiqueta)}</option>`).join("")}</select></label><span id="${escapar(id)}-motivo" class="baremo-condicion">${motivo}</span>${actual?.requiere_umbral ? `<label class="campo"><span>${t("jornada_umbral")}</span><input id="${escapar(umbralId)}" data-ruta="${escapar(JSON.stringify(umbralRuta))}" data-umbral data-indice="${i}" type="text" required maxlength="39" pattern="[1-9][0-9]{0,18}/[1-9][0-9]{0,18}" aria-label="${t("jornada_umbral")}" aria-describedby="${escapar(umbralId)}-error ${escapar(umbralId)}-formato" value="${escapar(estado.invalidos?.[JSON.stringify(umbralRuta)] ?? regla.jornada.umbral ?? "")}"${invalidado ? ' aria-invalid="true"' : ""}><span id="${escapar(umbralId)}-formato" class="baremo-condicion">${t("jornada_formato")}</span><span id="${escapar(umbralId)}-error" data-error-campo hidden></span></label>` : ""}</td>`;
  };
  const restos = (regla, i) => {
    const catalogo = estado.catalogoRestos, opciones = catalogo?.opciones ?? [];
    const ruta = ["reglas_experiencia", i, "restos", "modo"], id = escapar(idCampo(ruta));
    const modo = estado.invalidos?.[JSON.stringify(ruta)] ?? regla.restos?.modo ?? "";
    const actual = opciones.find((o) => o.modo === modo), error = errorRestosRegla(regla, catalogo, modo);
    return `<td><label class="campo">${t("restos")}<select id="${id}" data-ruta="${escapar(JSON.stringify(ruta))}" data-restos-modo data-indice="${i}" aria-label="${t("restos_regla", { n: textos.numero(i + 1) })}" aria-describedby="${id}-explicacion ${id}-error"${catalogo ? "" : " disabled"}${error && catalogo ? ` aria-invalid="true" data-restos-error="${escapar(error)}"` : ""}>${!actual ? `<option value="${escapar(modo)}" selected disabled>${t("restos_no_disponibles")}</option>` : ""}${opciones.map((o) => `<option value="${escapar(o.modo)}"${o.modo === modo ? " selected" : ""}${errorRestosRegla(regla, catalogo, o.modo) ? " disabled" : ""}>${t(o.etiqueta)}</option>`).join("")}</select><span id="${id}-explicacion" class="baremo-condicion" data-restos-explicacion>${actual ? t(actual.explicacion) : ""}</span><span id="${id}-error" data-error-campo${error ? "" : " hidden"}>${error ? t(error) : ""}</span></label></td>`;
  };
  const minimoFormacion = (regla, i) => {
    if (s.esquema !== "vec.bolsa.reglas_meritos.v1" || regla.familia !== "formacion" || regla.unidad !== "hora") return "";
    const ruta = ["reglas", i, "minimo_unidades"], id = escapar(idCampo(ruta));
    const invalido = Object.hasOwn(estado.invalidos ?? {}, JSON.stringify(ruta));
    return `<label class="campo">${t("minimo_formacion")}<input id="${id}" data-ruta="${escapar(JSON.stringify(ruta))}" data-minimo-formacion data-indice="${i}" type="text" required maxlength="39" pattern="(0|[1-9][0-9]{0,18})/[1-9][0-9]{0,18}" aria-label="${t("minimo_formacion_regla", { n: textos.numero(i + 1) })}" aria-describedby="${id}-formato ${id}-error" value="${escapar(estado.invalidos?.[JSON.stringify(ruta)] ?? regla.minimo_unidades)}"${invalido ? ' aria-invalid="true"' : ""}><span id="${id}-formato" class="baremo-condicion">${t("minimo_formacion_formato")}</span><span id="${id}-error" data-error-campo${invalido ? "" : " hidden"}>${invalido ? t("minimo_formacion_invalido") : ""}</span></label>`;
  };
  const fila = (regla, i) => {
    const ruta = [s.reglas_experiencia ? "reglas_experiencia" : "reglas", i];
    return `<tr><th scope="row">${t("regla_nombre", { n: i + 1 })}<span class="baremo-condicion">${regla.clase ? t(textos.mensajes.editor[`clase_${regla.clase}`] ? `clase_${regla.clase}` : "condicion_catalogo") : (regla.criterios ?? []).flatMap((c) => c.valores).map((v) => t(textos.mensajes.editor[`condicion_${v}`] ? `condicion_${v}` : "condicion_catalogo")).join(" · ")}</span></th><td>${seccion(regla.seccion_clave, i + 1)}</td><td>${campo([...ruta, "puntos_por_unidad"], regla.puntos_por_unidad, t("puntuacion"))}</td><td>${t(`unidad_${regla.unidad_temporal?.unidad_puntuable ?? regla.unidad ?? "merito"}`)}${minimoFormacion(regla, i)}</td><td>${typeof regla.maximo_puntos === "string" ? campo([...ruta, "maximo_puntos"], regla.maximo_puntos, t("maximo_regla")) : regla.maximo_puntos?.modo === "limitado" ? campo([...ruta, "maximo_puntos", "valor"], regla.maximo_puntos.valor, t("maximo_regla")) : t("sin_tope")}</td>${s.reglas_experiencia ? jornada(regla, i) + restos(regla, i) : ""}</tr>`;
  };
  const resumenTotal = (resultado) => {
    // El resumen pertenece a esta respuesta, nunca al borrador que se edita.
    const global = Object.hasOwn(resultado, "suma_secciones") || Object.hasOwn(resultado, "maximo_total");
    return `<dl>${global ? `<div><dt>${t("suma_apartados")}</dt><dd>${puntos(resultado.suma_secciones)}</dd></div><div><dt>${t("tope_global")}</dt><dd>${puntos(resultado.maximo_total)}</dd></div>` : ""}<div><dt>${t("total")}</dt><dd>${puntos(resultado.total)}</dd></div></dl>`;
  };
  const resultados = estado.comparacion ? ["antes", "despues"].map((lado) => {
    const r = estado.comparacion[lado].resultado;
    return `<section class="baremo-resultado"><h3>${t(lado)}</h3>${r.estado === "bloqueado" ? `<p role="status">${t("bloqueado")}</p><ul>${(r.bloqueos ?? r.incidencias ?? []).map((b) => `<li>${t(textos.mensajes.editor[`bloqueo_${b.codigo}`] ? `bloqueo_${b.codigo}` : "bloqueo_desconocido")}</li>`).join("")}</ul>` : `${resumenTotal(r)}<div class="tabla-contenedor" tabindex="0" aria-label="${t("desglose")}"><table class="tabla-datos"><caption>${t("desglose")}</caption><thead><tr><th scope="col">${t("apartado")}</th><th scope="col" class="columna-numero">${t("antes_tope")}</th><th scope="col" class="columna-numero">${t("maximo_apartado")}</th><th scope="col" class="columna-numero">${t("despues_tope")}</th></tr></thead><tbody>${(r.secciones ?? []).map((apartado, n) => `<tr><th scope="row">${seccion(apartado.seccion_clave ?? apartado.seccion ?? apartado.clave, n + 1)}</th><td class="columna-numero">${puntosExactos(apartado.antes_tope ?? apartado.suma_reglas)}</td><td class="columna-numero">${limiteApartado(apartado)}</td><td class="columna-numero">${puntos(apartado.subtotal ?? apartado.puntos_finales ?? apartado.puntos)}</td></tr>`).join("")}</tbody></table></div>`}</section>`;
  }).join("") : `<p role="status">${t(estado.trabajando ? "trabajando" : "pendiente")}</p>`;
  return `<section class="baremo-editor" aria-label="${t("titulo")}"><div class="panel">${cabecera}<div class="cuerpo-panel baremo-cabecera"><span class="estado-modulo">${t("estado")}</span><span>${t("version", { version: s.identidad?.version ?? s.version })}</span><span role="status">${t(estado.cambiado ? "sin_guardar" : "sin_cambios")}</span><p class="baremo-limite">${t("limite")}</p><p id="baremo-ayuda" ${estado.ayuda ? "" : "hidden"}>${t("ayuda_texto")}</p></div></div>
    <form id="baremo-formulario" novalidate><section class="panel"><div class="cuerpo-panel baremo-controles"><label class="campo"><span>${t("ejemplo")}</span><select name="ejemplo">${ejemplos.map((e, n) => `<option value="${escapar(e.referencia)}"${e.referencia === estado.ejemplo.referencia ? " selected" : ""}>${t(`caso_${e.modo}`, { n: n + 1 })}</option>`).join("")}</select></label><label class="campo"><span>${t("fecha")}</span><input id="baremo-fecha" aria-label="${t("fecha")}" aria-describedby="baremo-fecha-error" type="date" name="fecha" required value="${escapar(s.fecha_corte_inclusiva)}" data-ruta='["fecha_corte_inclusiva"]'><span id="baremo-fecha-error" data-error-campo hidden></span></label><label class="boton-secundario baremo-importar">${t("cargar")}<input type="file" accept=".json,application/json" name="archivo"></label><button type="button" class="boton-secundario" data-accion="restablecer">${t("restablecer")}</button></div></section>
    <div class="baremo-trabajo"><section class="panel baremo-reglas"><div class="cabecera-panel"><h2>${t("reglas")}</h2><label class="campo"><span>${t("filtrar")}</span><input type="search" name="filtro" value="${escapar(filtro)}"></label></div><div class="tabla-contenedor" tabindex="0" aria-label="${t("reglas")}"><table class="tabla-datos${s.esquema === "vec.bolsa.reglas_meritos.v1" || s.reglas_experiencia ? " tabla-datos--prioritaria" : ""}"><thead><tr><th scope="col">${t("regla")}</th><th scope="col">${t("apartado")}</th><th scope="col">${t("puntuacion")}</th><th scope="col">${t("unidad")}</th><th scope="col">${t("maximo_regla")}</th>${s.reglas_experiencia ? `<th scope="col">${t("jornada")}</th><th scope="col">${t("restos")}</th>` : ""}</tr></thead><tbody>${(s.reglas_experiencia ?? s.reglas).map((r, i) => fila(r, i)).join("")}<tr data-sin-reglas hidden><td colspan="${s.reglas_experiencia ? 7 : 5}">${t("sin_reglas")}</td></tr></tbody></table></div></section>
    <section class="panel baremo-topes"><div class="cabecera-panel"><h2>${t("estructura")}</h2></div><div class="cuerpo-panel">${s.maximo_total ? campo(["maximo_total"], s.maximo_total, t("tope_total")) : ""}${s.secciones.map((apartado, i) => campo(["secciones", i, Object.hasOwn(apartado, "puntos_maximos") ? "puntos_maximos" : "maximo_puntos"], apartado.puntos_maximos ?? apartado.maximo_puntos, `${seccion(apartado.clave, i + 1)} · ${t("tope")}`)).join("")}</div></section></div>
    <section class="panel"><div class="cabecera-panel"><h2>${t("resultado")}</h2></div><div class="cuerpo-panel baremo-resultados" aria-live="polite" aria-busy="${estado.trabajando}">${resultados}</div></section>
    <footer class="panel cuerpo-panel baremo-acciones"><div id="baremo-error" role="alert" ${estado.error ? "" : "hidden"}>${estado.error ? t(estado.error) : ""}</div><button type="button" class="boton-secundario" data-accion="exportar"${Object.keys(estado.invalidos ?? {}).length || estado.catalogoRestos && (s.reglas_experiencia ?? []).some((r) => errorRestosRegla(r, estado.catalogoRestos)) ? " disabled" : ""}>${t("exportar")}</button><button type="submit" class="boton-primario"${estado.trabajando ? " disabled" : ""}>${t(estado.trabajando ? "trabajando" : "comparar")}</button><button type="button" class="boton-secundario" disabled aria-describedby="baremo-activacion">${t("activar")}</button><p id="baremo-activacion">${t("activacion_pendiente")}</p></footer></form></section>`;
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
