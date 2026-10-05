import { bytesABase64, comprobarFichero } from "./cliente.js?v=20261005-b1-carga-v1";
import {
  claveCategoria, filtrarFilas, nombrePersona, paginar, textoAviso, textoBloqueo, textoError, textoIncidencia,
} from "./modelo.js?v=20261005-b1-carga-v1";

const ZONA = "Europe/Madrid";

function traducirFijos(doc, t) {
  doc.querySelectorAll("[data-i18n]").forEach((e) => { e.textContent = t(e.getAttribute("data-i18n")); });
  doc.querySelectorAll("[data-i18n-label]").forEach((e) => e.setAttribute("aria-label", t(e.getAttribute("data-i18n-label"))));
  doc.title = t("tituloDocumento");
}

/**
 * Pantalla de tres pasos: elegir fichero, revisar la vista previa y resultado.
 * El fichero solo vive en memoria de esta página; nada va a almacenamiento web.
 */
export function montarVistaCargaConvoca({ doc, cliente, categorias, textos }) {
  if (!doc || typeof cliente?.previsualizar !== "function" || typeof cliente?.confirmar !== "function"
    || typeof categorias?.listarOpciones !== "function" || typeof textos?.traducir !== "function") {
    throw new TypeError("montaje de la carga no disponible");
  }
  const t = (clave, variables) => textos.traducir(`general.${clave}`, variables);
  const n = (valor) => textos.numero(valor);
  const $ = (id) => doc.getElementById(id);
  doc.documentElement.lang = textos.idioma;
  doc.querySelectorAll('a[href^="/portal-empleado/"]').forEach((a) => {
    const [ruta, ancla = ""] = a.getAttribute("href").split("#");
    a.setAttribute("href", `${ruta}?lang=${encodeURIComponent(textos.idioma)}${ancla ? `#${ancla}` : ""}`);
  });
  traducirFijos(doc, t);

  const pasos = { 1: $("paso-elegir"), 2: $("paso-revisar"), 3: $("paso-hecho") };
  const estado = { nombre: "", base64: "", categoria: null, vista: null, filtro: "todas", pagina: 1, ocupado: false };
  let controlador = null;

  function irAPaso(numero) {
    for (const [clave, seccion] of Object.entries(pasos)) seccion.hidden = Number(clave) !== numero;
    for (let i = 1; i <= 3; i += 1) {
      const marca = $(`marca-paso-${i}`);
      if (i === numero) marca.setAttribute("aria-current", "step"); else marca.removeAttribute("aria-current");
      marca.classList.toggle("carga-pasos--hecho", i < numero);
    }
    pasos[numero].focus();
  }

  function nuevaPeticion() {
    controlador?.abort();
    controlador = new AbortController();
    return controlador.signal;
  }

  // Paso 1: categorías y validación del formulario.
  async function cargarCategorias() {
    const select = $("categoria");
    select.disabled = true;
    $("categorias-reintentar").hidden = true;
    $("categorias-estado").textContent = t("categoriasCargando");
    try {
      const opciones = await categorias.listarOpciones({ signal: nuevaPeticion() });
      select.replaceChildren(select.options[0]);
      for (const opcion of opciones) {
        if (!claveCategoria(opcion.referencia)) continue;
        const elemento = doc.createElement("option");
        elemento.value = opcion.referencia;
        elemento.textContent = opcion.etiqueta;
        select.append(elemento);
      }
      select.disabled = false;
      $("categorias-estado").textContent = "";
    } catch (error) {
      if (error?.name === "AbortError") return;
      const denegado = error?.estado === 401 || error?.estado === 403;
      $("categorias-estado").textContent = denegado ? textoError(textos, error) : t("categoriasError");
      $("categorias-reintentar").hidden = denegado;
    }
  }

  function marcarCampo(id, mensaje) {
    const error = $(`${id}-error`);
    error.hidden = !mensaje;
    error.textContent = mensaje;
    if (mensaje) $(id).setAttribute("aria-invalid", "true"); else $(id).removeAttribute("aria-invalid");
  }

  function validarFormulario() {
    const fichero = $("fichero").files?.[0];
    const errores = [];
    const errorFichero = comprobarFichero(fichero);
    marcarCampo("fichero", errorFichero ? t(errorFichero) : "");
    if (errorFichero) errores.push(["fichero", t(errorFichero)]);
    const select = $("categoria");
    const elegida = select.selectedOptions?.[0];
    const valida = select.value && claveCategoria(select.value);
    marcarCampo("categoria", valida ? "" : t("faltaCategoria"));
    if (!valida) errores.push(["categoria", t("faltaCategoria")]);
    const resumen = $("resumen-errores");
    $("lista-errores").replaceChildren(...errores.map(([id, texto]) => {
      const li = doc.createElement("li");
      const a = doc.createElement("a");
      a.href = `#${id}`;
      a.textContent = texto;
      a.addEventListener("click", (evento) => { evento.preventDefault(); $(id).focus(); });
      li.append(a);
      return li;
    }));
    resumen.hidden = errores.length === 0;
    if (errores.length) { resumen.focus(); return null; }
    return { fichero, categoria: { referencia: select.value, etiqueta: elegida?.textContent ?? "" } };
  }

  async function revisar(evento) {
    evento.preventDefault();
    if (estado.ocupado) return;
    const datos = validarFormulario();
    if (!datos) return;
    estado.ocupado = true;
    $("revisar").disabled = true;
    $("estado-elegir").textContent = t("revisando");
    try {
      const bytes = new Uint8Array(await datos.fichero.arrayBuffer());
      estado.nombre = datos.fichero.name;
      estado.base64 = bytesABase64(bytes);
      bytes.fill(0);
      estado.categoria = datos.categoria;
      estado.vista = await cliente.previsualizar({ nombre: estado.nombre, base64: estado.base64, signal: nuevaPeticion() });
      estado.filtro = "todas";
      estado.pagina = 1;
      $("estado-elegir").textContent = "";
      pintarRevision();
      irAPaso(2);
    } catch (error) {
      if (error?.name === "AbortError") return;
      estado.base64 = "";
      $("estado-elegir").textContent = textoError(textos, error);
    } finally {
      estado.ocupado = false;
      $("revisar").disabled = false;
    }
  }

  // Paso 2: revisión.
  function celda(contenido, clase = "") {
    const td = doc.createElement("td");
    if (clase) td.className = clase;
    td.textContent = contenido;
    return td;
  }

  function pintarFilas() {
    const visibles = filtrarFilas(estado.vista.filas, estado.filtro);
    const pagina = paginar(visibles, estado.pagina);
    estado.pagina = pagina.pagina;
    $("cuenta").textContent = visibles.length ? t("cuentaFilas", { cuenta: visibles.length }) : t("sinFilasFiltro");
    $("tabla-contenedor").hidden = visibles.length === 0;
    $("paginacion").hidden = pagina.total <= 1;
    $("pagina-estado").textContent = t("paginaEstado", { pagina: n(pagina.pagina), total: n(pagina.total) });
    $("anterior").disabled = pagina.pagina <= 1;
    $("siguiente").disabled = pagina.pagina >= pagina.total;
    $("filas").replaceChildren(...pagina.elementos.map((fila) => {
      const tr = doc.createElement("tr");
      const aceptada = fila.estado === "aceptada";
      tr.append(celda(aceptada ? n(fila.posicion) : "—", "carga-numero"));
      const persona = doc.createElement("th");
      persona.scope = "row";
      persona.textContent = aceptada ? nombrePersona(fila) : t("fila", { numero: n(fila.numero) });
      tr.append(persona, celda(fila.documento ?? ""),
        celda(fila.total ? n(Number(fila.total)) : "", "carga-numero"));
      const chip = doc.createElement("span");
      const conAviso = fila.avisos.length > 0;
      chip.className = `estado-chip ${aceptada ? (conAviso ? "" : "exito") : "peligro"}`.trim();
      chip.textContent = t(aceptada ? (conAviso ? "estadoAviso" : "estadoCargara") : "estadoError");
      const tdEstado = doc.createElement("td");
      tdEstado.append(chip);
      const tdDetalle = doc.createElement("td");
      const mensajes = [...fila.errores.map((e) => textoIncidencia(textos, e)), ...fila.avisos.map((a) => textoAviso(textos, a))];
      // En las aceptadas se añade la fila del Excel para poder localizarla.
      tdDetalle.textContent = aceptada && mensajes.length
        ? `${t("fila", { numero: n(fila.numero) })}: ${mensajes.join(" ")}` : mensajes.join(" ");
      tr.append(tdEstado, tdDetalle);
      return tr;
    }));
  }

  function pintarRevision() {
    const v = estado.vista;
    $("resumen-fichero").textContent = t("resumenFichero", { fichero: v.nombre_fichero, categoria: estado.categoria.etiqueta });
    $("kpi-leidas").textContent = n(v.filas_leidas);
    $("kpi-cargaran").textContent = n(v.aceptadas);
    $("kpi-errores").textContent = n(v.rechazadas);
    $("kpi-avisos").textContent = n(v.con_avisos);
    $("bloqueo").hidden = !v.bloqueo;
    $("bloqueo").textContent = v.bloqueo ? textoBloqueo(textos, v.bloqueo) : "";
    $("excluir-marco").hidden = v.rechazadas === 0 || Boolean(v.bloqueo);
    $("excluir").checked = false;
    $("excluir-texto").textContent = t("excluir", { cuenta: v.rechazadas });
    $("cargar").disabled = Boolean(v.bloqueo);
    $("error-revisar").hidden = true;
    $("estado-revisar").textContent = "";
    doc.querySelectorAll('input[name="filtro"]').forEach((r) => { r.checked = r.value === estado.filtro; });
    pintarFilas();
  }

  function mostrarErrorRevision(texto) {
    const nodo = $("error-revisar");
    nodo.textContent = texto;
    nodo.hidden = false;
    nodo.focus();
  }

  function pedirConfirmacion() {
    const v = estado.vista;
    if (!v || v.bloqueo || estado.ocupado) return;
    if (v.rechazadas > 0 && !$("excluir").checked) { mostrarErrorRevision(t("faltaExcluir")); return; }
    $("error-revisar").hidden = true;
    $("confirmar-texto").textContent = t("confirmarTexto", { cuenta: v.aceptadas, categoria: estado.categoria.etiqueta });
    $("confirmar").showModal?.();
  }

  async function cargar() {
    $("confirmar").close?.();
    if (estado.ocupado) return;
    estado.ocupado = true;
    $("cargar").disabled = true;
    $("otro-fichero").disabled = true;
    $("estado-revisar").textContent = t("cargando");
    try {
      const recibo = await cliente.confirmar({ nombre: estado.nombre, base64: estado.base64,
        categoria: claveCategoria(estado.categoria.referencia), excluir: $("excluir").checked, signal: nuevaPeticion() });
      estado.base64 = "";
      pintarHecho(recibo);
      irAPaso(3);
    } catch (error) {
      if (error?.name === "AbortError") return;
      $("estado-revisar").textContent = "";
      mostrarErrorRevision(textoError(textos, error));
    } finally {
      estado.ocupado = false;
      $("cargar").disabled = Boolean(estado.vista?.bloqueo);
      $("otro-fichero").disabled = false;
    }
  }

  // Paso 3: resultado.
  function pintarHecho(recibo) {
    $("hecho-mensaje").textContent = recibo.reutilizada ? t("hechoRepetida") : t("hechoNueva", { categoria: estado.categoria.etiqueta });
    const datos = [t("hechoCargadas", { cuenta: recibo.filas_cargadas })];
    if (recibo.filas_excluidas) datos.push(t("hechoExcluidas", { cuenta: recibo.filas_excluidas }));
    if (recibo.sustituye_a.length) datos.push(t("hechoSustituye", { cuenta: recibo.sustituye_a.length }));
    datos.push(t("hechoFecha", { fecha: textos.fecha(recibo.confirmada_en, { dateStyle: "long", timeStyle: "short", timeZone: ZONA }) }));
    $("hecho-datos").replaceChildren(...datos.map((texto) => { const li = doc.createElement("li"); li.textContent = texto; return li; }));
    $("hecho-pendientes").hidden = recibo.pendientes_revision.length === 0;
    $("hecho-pendientes-lista").replaceChildren(...recibo.pendientes_revision.map((p) => {
      const li = doc.createElement("li");
      li.textContent = `${t("fila", { numero: n(p.fila) })}: ${textoAviso(textos, p.motivo)}`;
      return li;
    }));
    $("detalle-registro").textContent = t("detalleRegistro", { referencia: recibo.auditoria_ref });
    $("detalle-huella").textContent = t("detalleHuella", { huella: recibo.huella_sha256 });
  }

  function reiniciar() {
    controlador?.abort();
    Object.assign(estado, { nombre: "", base64: "", vista: null, ocupado: false });
    $("fichero").value = "";
    $("resumen-errores").hidden = true;
    marcarCampo("fichero", "");
    marcarCampo("categoria", "");
    $("revisar").disabled = false;
    irAPaso(1);
    void cargarCategorias();
  }

  $("form-elegir").addEventListener("submit", revisar);
  $("categorias-reintentar").addEventListener("click", () => void cargarCategorias());
  doc.querySelectorAll('input[name="filtro"]').forEach((radio) => radio.addEventListener("change", () => {
    estado.filtro = radio.value;
    estado.pagina = 1;
    pintarFilas();
  }));
  $("anterior").addEventListener("click", () => { estado.pagina -= 1; pintarFilas(); });
  $("siguiente").addEventListener("click", () => { estado.pagina += 1; pintarFilas(); });
  $("excluir").addEventListener("change", () => { $("error-revisar").hidden = true; });
  $("cargar").addEventListener("click", pedirConfirmacion);
  $("confirmar-si").addEventListener("click", () => void cargar());
  $("confirmar-no").addEventListener("click", () => $("confirmar").close?.());
  $("otro-fichero").addEventListener("click", reiniciar);
  $("otra-carga").addEventListener("click", reiniciar);
  $("ayuda-abrir").addEventListener("click", () => $("ayuda").showModal?.());
  $("ayuda-cerrar").addEventListener("click", () => $("ayuda").close?.());
  doc.defaultView?.addEventListener("pagehide", () => controlador?.abort());
  return Object.freeze({ iniciar: cargarCategorias });
}
