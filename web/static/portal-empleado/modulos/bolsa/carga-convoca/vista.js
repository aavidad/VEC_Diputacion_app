import { bytesABase64, comprobarFichero, ErrorCargaConvoca } from "./cliente.js?v=20261008-u-b1-preview-v6";
import { rutaCandidatosBolsaCompartible } from "../../../portal-bolsas-ruta-filtros.js";
import {
  claveCategoria, escribirEstadoRuta, filtroControl, filtroServidor, leerEstadoRuta, nombrePersona,
  paginaServidor, TAMANO_PAGINA, textoAviso, textoBloqueo, textoError, textoIncidencia,
} from "./modelo.js?v=20261008-u-b1-paginacion-v1";

const ZONA = "Europe/Madrid";
const CODIGOS_FICHERO = new Set(["fichero_no_valido", "fichero_demasiado_grande", "demasiadas_filas", "peticion_no_valida"]);

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
  const rutaInicial = leerEstadoRuta(doc.defaultView?.location?.search ?? "");
  const estado = { nombre: "", base64: "", categoria: null, vista: null, filtro: rutaInicial.filtro,
    pagina: rutaInicial.pagina, ocupado: false, cargandoPagina: false, generacion: 0, secuencia: 0,
    peticionActual: null, recibo: null, soloLectura: false, indeterminado: false };
  let controladorCategorias = null;
  let controladorVista = null;
  let controladorConfirmacion = null;

  function irAPaso(numero) {
    for (const [clave, seccion] of Object.entries(pasos)) seccion.hidden = Number(clave) !== numero;
    for (let i = 1; i <= 3; i += 1) {
      const marca = $(`marca-paso-${i}`);
      if (i === numero) marca.setAttribute("aria-current", "step"); else marca.removeAttribute("aria-current");
      marca.classList.toggle("carga-pasos--hecho", i < numero);
      marca.querySelector(".carga-pasos__numero").textContent = i < numero ? "✓" : String(i);
      marca.querySelector(".carga-pasos__estado").textContent = i < numero ? ` (${t("pasoHecho")})` : i === numero ? ` (${t("pasoActual")})` : "";
    }
    pasos[numero].focus();
  }

  // Paso 1: categorías y validación del formulario.
  async function cargarCategorias() {
    const select = $("categoria");
    if (select.options.length > 1) return;
    select.disabled = true;
    $("categorias-reintentar").hidden = true;
    $("categorias-estado").textContent = t("categoriasCargando");
    try {
      controladorCategorias?.abort();
      controladorCategorias = new AbortController();
      const opciones = await categorias.listarOpciones({ signal: controladorCategorias.signal });
      const previa = select.value;
      select.replaceChildren(select.options[0]);
      for (const opcion of opciones) {
        if (!claveCategoria(opcion.referencia)) continue;
        const elemento = doc.createElement("option");
        elemento.value = opcion.referencia;
        elemento.textContent = opcion.etiqueta;
        select.append(elemento);
      }
      // Volver a elegir fichero no obliga a repetir la categoría.
      if ([...select.options].some((o) => o.value === previa)) select.value = previa;
      const hay = select.options.length > 1;
      select.disabled = !hay;
      $("categorias-estado").textContent = hay ? "" : t("sinCategorias");
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
    if (errores.length) { pintarResumenErrores(errores); return null; }
    $("resumen-errores").hidden = true;
    return { fichero, categoria: { referencia: select.value, etiqueta: elegida?.textContent ?? "" } };
  }

  function pintarResumenErrores(errores) {
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
    resumen.hidden = false;
    resumen.focus();
  }

  // Un fallo del fichero se marca en su campo, como los del formulario; los
  // demás salen en un aviso destacado.
  function mostrarErrorElegir(error) {
    const texto = textoError(textos, error);
    if (CODIGOS_FICHERO.has(error?.codigo)) {
      marcarCampo("fichero", texto);
      pintarResumenErrores([["fichero", texto]]);
      return;
    }
    const aviso = $("error-elegir");
    aviso.textContent = texto;
    aviso.hidden = false;
    aviso.focus();
  }

  function actualizarRuta(filtro, pagina, modo) {
    const ventana = doc.defaultView;
    if (!ventana?.history || !modo) return;
    const { pathname, search, hash } = ventana.location;
    const destino = `${pathname}${escribirEstadoRuta(search, filtro, pagina)}${hash}`;
    if (destino !== `${pathname}${search}${hash}`) ventana.history[modo](null, "", destino);
  }

  function mismaLectura(antes, despues) {
    return !antes || ["huella_sha256", "filas_leidas", "aceptadas", "rechazadas", "con_avisos", "bloqueo"]
      .every((campo) => antes[campo] === despues[campo]);
  }

  function solicitarPagina({ filtro, pagina, ruta = null, forzar = false, soloLectura = null }) {
    if (!estado.base64) return Promise.resolve(null);
    const paginaSolicitada = pagina;
    const clave = `${estado.generacion}:${filtro}:${pagina}`;
    if (estado.peticionActual?.clave === clave) return estado.peticionActual.promesa;
    if (!forzar && estado.vista?.filtro === filtro && estado.pagina === pagina && !estado.cargandoPagina) return Promise.resolve(estado.vista);
    controladorVista?.abort();
    controladorVista = new AbortController();
    const signal = controladorVista.signal;
    const secuencia = ++estado.secuencia;
    estado.cargandoPagina = true;
    $("estado-revisar").textContent = t("cargandoPagina");
    $("error-revisar").hidden = true;
    $("cargar").disabled = true;
    const promesa = (async () => {
      try {
        let respuesta = await cliente.previsualizar({ nombre: estado.nombre, base64: estado.base64,
          categoria: claveCategoria(estado.categoria.referencia),
          filtro, limite: TAMANO_PAGINA, desplazamiento: (pagina - 1) * TAMANO_PAGINA, signal });
        if (secuencia !== estado.secuencia || signal.aborted) return null;
        const ultimaPagina = Math.max(1, Math.ceil(respuesta.total_filtrado / TAMANO_PAGINA));
        if (pagina > ultimaPagina) {
          pagina = ultimaPagina;
          respuesta = await cliente.previsualizar({ nombre: estado.nombre, base64: estado.base64,
            categoria: claveCategoria(estado.categoria.referencia),
            filtro, limite: TAMANO_PAGINA, desplazamiento: (pagina - 1) * TAMANO_PAGINA, signal });
        }
        if (secuencia !== estado.secuencia || signal.aborted) return null;
        if (estado.recibo && (respuesta.huella_sha256 !== estado.recibo.huella_sha256
          || respuesta.aceptadas !== estado.recibo.filas_cargadas
          || respuesta.rechazadas !== estado.recibo.filas_excluidas)) {
          throw new ErrorCargaConvoca(0, "recibo_incoherente");
        }
        if (!mismaLectura(estado.vista, respuesta)) throw new TypeError("el fichero cambió durante la revisión");
        const primera = !estado.vista;
        if (soloLectura !== null) estado.soloLectura = soloLectura;
        estado.vista = respuesta;
        estado.filtro = filtro;
        estado.pagina = pagina;
        pintarRevision(primera);
        if (pasos[2].hidden) irAPaso(2);
        actualizarRuta(filtro, pagina, pagina !== paginaSolicitada ? "replaceState" : ruta);
        return respuesta;
      } catch (error) {
        if (secuencia !== estado.secuencia || error?.name === "AbortError") return null;
        throw error;
      } finally {
        if (secuencia === estado.secuencia) {
          estado.cargandoPagina = false;
          estado.peticionActual = null;
          $("estado-revisar").textContent = "";
          $("cargar").disabled = Boolean(estado.vista?.bloqueo) || estado.ocupado
            || estado.soloLectura || estado.indeterminado;
        }
      }
    })();
    estado.peticionActual = { clave, promesa };
    return promesa;
  }

  async function revisar(evento) {
    evento.preventDefault();
    if (estado.ocupado) return;
    $("error-elegir").hidden = true;
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
      estado.vista = null;
      estado.generacion += 1;
      const ruta = leerEstadoRuta(doc.defaultView?.location?.search ?? "");
      const respuesta = await solicitarPagina({ filtro: ruta.filtro, pagina: ruta.pagina });
      if (!respuesta) return;
      $("estado-elegir").textContent = "";
    } catch (error) {
      if (error?.name === "AbortError") return;
      estado.base64 = "";
      $("estado-elegir").textContent = "";
      mostrarErrorElegir(error);
    } finally {
      estado.ocupado = false;
      $("revisar").disabled = false;
      if (estado.vista) $("cargar").disabled = Boolean(estado.vista.bloqueo)
        || estado.cargandoPagina || estado.indeterminado;
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
    const visibles = estado.vista.filas;
    const pagina = paginaServidor(estado.vista);
    $("cuenta").textContent = estado.vista.total_filtrado
      ? t("cuentaFilas", { cuenta: estado.vista.total_filtrado }) : t("sinFilasFiltro");
    $("tabla-contenedor").hidden = visibles.length === 0;
    $("paginacion").hidden = pagina.total <= 1;
    $("pagina-estado").textContent = t("paginaEstado", { pagina: n(pagina.pagina), total: n(pagina.total) });
    $("anterior").disabled = pagina.pagina <= 1;
    $("siguiente").disabled = pagina.pagina >= pagina.total;
    $("filas").replaceChildren(...visibles.map((fila) => {
      const tr = doc.createElement("tr");
      const aceptada = fila.estado === "aceptada";
      tr.append(celda(aceptada ? n(fila.posicion) : "—", "carga-numero"));
      const persona = doc.createElement("th");
      persona.scope = "row";
      persona.textContent = aceptada ? nombrePersona(fila) : t("fila", { numero: n(fila.numero) });
      const chip = doc.createElement("span");
      const conAviso = fila.avisos.length > 0;
      chip.className = `estado-chip ${aceptada ? (conAviso ? "" : "exito") : "peligro"}`.trim();
      chip.textContent = t(aceptada ? (conAviso ? "filtroAvisos" : "estadoCargara") : "estadoError");
      const tdEstado = doc.createElement("td");
      tdEstado.append(chip);
      tr.append(persona, tdEstado, celda(fila.documento ?? ""), celda(fila.total ? n(Number(fila.total)) : "", "carga-numero"));
      const tdDetalle = doc.createElement("td");
      const mensajes = [...fila.errores.map((e) => textoIncidencia(textos, e)), ...fila.avisos.map((a) => textoAviso(textos, a))];
      // En las aceptadas se añade la fila del Excel para poder localizarla.
      tdDetalle.textContent = aceptada && mensajes.length
        ? `${t("fila", { numero: n(fila.numero) })}: ${mensajes.join(" ")}` : mensajes.join(" ");
      tr.append(tdDetalle);
      return tr;
    }));
  }

  function pintarRevision(primera = false) {
    const v = estado.vista;
    $("resumen-fichero").textContent = t("resumenFichero", { fichero: v.nombre_fichero, categoria: estado.categoria.etiqueta });
    $("kpi-leidas").textContent = n(v.filas_leidas);
    $("kpi-cargaran").textContent = n(v.aceptadas);
    $("kpi-errores").textContent = n(v.rechazadas);
    $("kpi-avisos").textContent = n(v.con_avisos);
    $("bloqueo").hidden = !v.bloqueo;
    $("bloqueo").textContent = v.bloqueo ? textoBloqueo(textos, v.bloqueo) : "";
    $("excluir-marco").hidden = estado.soloLectura || v.rechazadas === 0 || Boolean(v.bloqueo);
    if (primera) $("excluir").checked = false;
    $("excluir-texto").textContent = t("excluir", { cuenta: v.rechazadas });
    $("cargar").hidden = estado.soloLectura || estado.indeterminado;
    $("cargar").disabled = estado.soloLectura || estado.indeterminado || Boolean(v.bloqueo) || estado.cargandoPagina;
    $("recuperar-resultado").hidden = !estado.indeterminado;
    $("recuperar-ayuda").hidden = !estado.indeterminado;
    $("ver-bolsas-tras-error").hidden = !estado.indeterminado;
    $("otro-fichero").hidden = estado.soloLectura;
    $("volver-resultado").hidden = !estado.soloLectura;
    $("error-revisar").hidden = !estado.indeterminado;
    $("estado-revisar").textContent = "";
    doc.querySelectorAll('input[name="filtro"]').forEach((r) => { r.checked = r.value === filtroControl(estado.filtro); });
    pintarFilas();
  }

  function mostrarErrorRevision(texto) {
    const nodo = $("error-revisar");
    nodo.textContent = texto;
    nodo.hidden = false;
    nodo.focus();
  }

  function avisarRecuperacionSinFichero() {
    if (!estado.indeterminado || estado.base64) return;
    $("recuperar-resultado").disabled = true;
    $("recuperar-ayuda").textContent = t("recuperacionSinFichero");
    $("recuperar-ayuda").hidden = false;
    doc.querySelectorAll('input[name="filtro"], .carga-kpi-boton, #anterior, #siguiente')
      .forEach((control) => { control.disabled = true; });
  }

  function pedirConfirmacion() {
    const v = estado.vista;
    if (!v || v.bloqueo || estado.ocupado || estado.cargandoPagina || estado.soloLectura || estado.indeterminado) return;
    if (v.rechazadas > 0 && !$("excluir").checked) { mostrarErrorRevision(t("faltaExcluir")); return; }
    $("error-revisar").hidden = true;
    $("confirmar-texto").textContent = `${t("confirmarTexto", { cuenta: v.aceptadas, categoria: estado.categoria.etiqueta })} ${v.rechazadas ? t("confirmarDescartes", { cuenta: v.rechazadas }) : ""}`.trim();
    $("confirmar").showModal?.();
  }

  async function cargar({ recuperar = false } = {}) {
    if ($("confirmar").open) $("confirmar").close?.();
    if (recuperar && !estado.base64) { avisarRecuperacionSinFichero(); return; }
    if (estado.ocupado || estado.cargandoPagina || estado.soloLectura
      || (estado.indeterminado && !recuperar) || (recuperar && !estado.indeterminado)) return;
    estado.ocupado = true;
    $("cargar").disabled = true;
    $("recuperar-resultado").disabled = true;
    $("otro-fichero").disabled = true;
    $("estado-revisar").textContent = t("cargando");
    try {
      controladorConfirmacion = new AbortController();
      const recibo = await cliente.confirmar({ nombre: estado.nombre, base64: estado.base64,
        categoria: claveCategoria(estado.categoria.referencia), excluir: $("excluir").checked,
        signal: controladorConfirmacion.signal });
      if (recibo.huella_sha256 !== estado.vista.huella_sha256
        || recibo.filas_cargadas !== estado.vista.aceptadas
        || recibo.filas_excluidas !== estado.vista.rechazadas) {
        throw new ErrorCargaConvoca(0, "recibo_incoherente");
      }
      estado.indeterminado = false;
      estado.recibo = recibo;
      pintarHecho(recibo);
      irAPaso(3);
    } catch (error) {
      const fallo = error?.codigo === "recibo_incoherente" ? error
        : !Number.isInteger(error?.estado) || error.estado === 0 || error.estado >= 500
          ? new ErrorCargaConvoca(0, "resultado_indeterminado") : error;
      $("estado-revisar").textContent = "";
      if (fallo?.codigo === "recibo_incoherente" || fallo?.codigo === "resultado_indeterminado") {
        estado.indeterminado = true;
      }
      if (estado.indeterminado) {
        $("cargar").hidden = true;
        $("recuperar-resultado").hidden = false;
        $("recuperar-ayuda").hidden = false;
        $("ver-bolsas-tras-error").hidden = false;
        if (!estado.base64) avisarRecuperacionSinFichero();
      }
      mostrarErrorRevision(textoError(textos, fallo));
    } finally {
      controladorConfirmacion = null;
      estado.ocupado = false;
      $("cargar").disabled = Boolean(estado.vista?.bloqueo) || estado.soloLectura || estado.indeterminado;
      $("recuperar-resultado").disabled = estado.indeterminado && !estado.base64;
      $("otro-fichero").disabled = false;
    }
  }

  // Paso 3: resultado.
  function pintarHecho(recibo) {
    $("hecho-mensaje").textContent = recibo.reutilizada ? t("hechoRepetida") : t("hechoNueva", { categoria: estado.categoria.etiqueta });
    $("error-hecho").hidden = true;
    const busqueda = escribirEstadoRuta(doc.defaultView?.location?.search ?? "", "todas", 1);
    const rutaBolsa = (referencia) => `/portal-empleado/${rutaCandidatosBolsaCompartible(busqueda, referencia)}`;
    const cargadas = doc.createElement("li");
    const enlace = doc.createElement("a");
    enlace.href = rutaBolsa(recibo.bolsa_ref);
    enlace.textContent = t("verCargadas", { cuenta: recibo.filas_cargadas });
    cargadas.append(enlace);
    const datos = [];
    if (recibo.filas_excluidas) {
      const excluidas = doc.createElement("li");
      const boton = doc.createElement("button");
      boton.type = "button";
      boton.id = "ver-excluidas";
      boton.className = "boton-secundario";
      boton.textContent = t("verExcluidas", { cuenta: recibo.filas_excluidas });
      boton.addEventListener("click", () => void abrirExcluidas());
      excluidas.append(boton);
      datos.push(excluidas);
    }
    if (recibo.sustituye_a.length) {
      const grupo = doc.createElement("li");
      grupo.append(doc.createTextNode(t("bolsasSustituidas")));
      const lista = doc.createElement("ul");
      recibo.sustituye_a.forEach((bolsa, indice) => {
        const li = doc.createElement("li");
        const a = doc.createElement("a");
        a.href = rutaBolsa(bolsa.bolsa_ref);
        a.textContent = t("verBolsaAnterior", { numero: n(indice + 1) });
        li.append(a);
        lista.append(li);
      });
      grupo.append(lista);
      datos.push(grupo);
    }
    datos.push(t("hechoFecha", { fecha: textos.fecha(recibo.confirmada_en, { dateStyle: "long", timeStyle: "short", timeZone: ZONA }) }));
    $("hecho-datos").replaceChildren(cargadas, ...datos.map((dato) => {
      if (typeof dato !== "string") return dato;
      const li = doc.createElement("li");
      li.textContent = dato;
      return li;
    }));
    $("hecho-pendientes").hidden = recibo.pendientes_revision.length === 0;
    $("hecho-pendientes-lista").replaceChildren(...recibo.pendientes_revision.map((p) => {
      const li = doc.createElement("li");
      li.textContent = `${t("fila", { numero: n(p.fila) })}: ${textoAviso(textos, p.motivo)}`;
      return li;
    }));
    $("detalle-registro").textContent = t("detalleRegistro", { referencia: recibo.auditoria_ref });
  }

  function avisarFicheroNoDisponible(enfocar = false) {
    const boton = $("ver-excluidas");
    if (boton) boton.disabled = true;
    const aviso = $("error-hecho");
    aviso.textContent = t("ficheroNoDisponible");
    aviso.hidden = false;
    if (enfocar) aviso.focus();
  }

  async function abrirExcluidas() {
    if (!estado.recibo || !estado.recibo.filas_excluidas || estado.ocupado) return;
    if (!estado.base64) { avisarFicheroNoDisponible(true); return; }
    $("error-hecho").hidden = true;
    try {
      const respuesta = await solicitarPagina({ filtro: "rechazadas", pagina: 1, ruta: "pushState",
        forzar: true, soloLectura: true });
      if (!respuesta) return;
      $("cargar").hidden = true;
    } catch (error) {
      const aviso = $("error-hecho");
      aviso.textContent = textoError(textos, error);
      aviso.hidden = false;
      aviso.focus();
    }
  }

  function reiniciar() {
    controladorVista?.abort();
    estado.secuencia += 1;
    Object.assign(estado, { nombre: "", base64: "", vista: null, ocupado: false,
      cargandoPagina: false, peticionActual: null, recibo: null, soloLectura: false, indeterminado: false });
    $("ver-bolsas-tras-error").hidden = true;
    $("recuperar-resultado").hidden = true;
    $("recuperar-ayuda").hidden = true;
    $("recuperar-ayuda").textContent = t("recuperarResultadoAyuda");
    doc.querySelectorAll('input[name="filtro"], .carga-kpi-boton, #anterior, #siguiente')
      .forEach((control) => { control.disabled = false; });
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
  function navegarPagina(filtro, pagina, { ruta = "pushState", enfocar = true } = {}) {
    if (!estado.vista || estado.ocupado) return;
    void solicitarPagina({ filtro, pagina, ruta }).then((respuesta) => {
      if (!respuesta || !enfocar) return;
      $("tabla-contenedor").hidden ? $("cuenta").focus() : $("tabla-contenedor").focus();
    }).catch((error) => {
      doc.querySelectorAll('input[name="filtro"]').forEach((radio) => {
        radio.checked = radio.value === filtroControl(estado.filtro);
      });
      actualizarRuta(estado.filtro, estado.pagina, "replaceState");
      mostrarErrorRevision(textoError(textos, error));
    });
  }
  doc.querySelectorAll('input[name="filtro"]').forEach((radio) => radio.addEventListener("change", () => {
    navegarPagina(filtroServidor(radio.value), 1);
  }));
  doc.querySelectorAll(".carga-kpi-boton").forEach((boton) => boton.addEventListener("click", () => {
    navegarPagina(filtroServidor(boton.dataset.filtro), 1);
  }));
  $("anterior").addEventListener("click", () => navegarPagina(estado.filtro, Math.max(1, estado.pagina - 1)));
  $("siguiente").addEventListener("click", () => navegarPagina(estado.filtro, estado.pagina + 1));
  $("excluir").addEventListener("change", () => {
    if (!estado.indeterminado) $("error-revisar").hidden = true;
  });
  $("cargar").addEventListener("click", pedirConfirmacion);
  $("confirmar-si").addEventListener("click", () => void cargar());
  $("recuperar-resultado").addEventListener("click", () => void cargar({ recuperar: true }));
  $("confirmar-no").addEventListener("click", () => $("confirmar").close?.());
  $("otro-fichero").addEventListener("click", reiniciar);
  $("volver-resultado").addEventListener("click", () => {
    estado.soloLectura = false;
    irAPaso(3);
  });
  $("otra-carga").addEventListener("click", reiniciar);
  $("ayuda-abrir").addEventListener("click", () => $("ayuda").showModal?.());
  $("ayuda-cerrar").addEventListener("click", () => $("ayuda").close?.());
  doc.defaultView?.addEventListener("popstate", () => {
    if (!pasos[3].hidden) return;
    const ruta = leerEstadoRuta(doc.defaultView.location.search);
    if (!estado.base64) { estado.filtro = ruta.filtro; estado.pagina = ruta.pagina; return; }
    navegarPagina(ruta.filtro, ruta.pagina, { ruta: null, enfocar: false });
  });
  doc.defaultView?.addEventListener("pagehide", () => {
    controladorCategorias?.abort();
    controladorVista?.abort();
    estado.base64 = "";
    $("fichero").value = "";
  });
  doc.defaultView?.addEventListener("pageshow", () => {
    if (estado.recibo && !estado.base64 && !pasos[3].hidden) avisarFicheroNoDisponible();
    if (estado.indeterminado && !estado.base64 && !pasos[2].hidden) avisarRecuperacionSinFichero();
  });
  return Object.freeze({ iniciar: cargarCategorias });
}
