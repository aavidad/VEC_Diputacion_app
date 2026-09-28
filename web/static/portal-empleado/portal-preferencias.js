import { crearClientePreferencias, ErrorPreferencias } from "./portal-preferencias-api.js";
import { traducirPortal } from "./portal-i18n.js?v=20260928-auditoria-expediente-en-v2";

const CAMPOS_SELECT = Object.freeze({ idioma: "idiomas", tamano_texto: "tamanos_texto", tema: "temas", inicio: "inicios", filas: "filas" });
const AYUDAS = Object.freeze({ idioma: "idioma", tamano_texto: "tamano", alto_contraste: "contraste", tema: "tema", inicio: "inicio", filas: "filas", aviso_correo_tareas: "correo_tareas", aviso_correo_plazos: "correo_plazos" });
const ETIQUETAS = Object.freeze({ idioma: "idioma", tamano_texto: "tamano", alto_contraste: "contraste", tema: "tema", inicio: "inicio", filas: "filas", aviso_correo_tareas: "correo_tareas", aviso_correo_plazos: "correo_plazos" });
const ETIQUETAS_CATALOGO = Object.freeze({
  idioma: Object.freeze({ navegador: "ui.usuarios.preferencias.idioma.navegador", es: "ui.usuarios.preferencias.idioma.es", en: "ui.usuarios.preferencias.idioma.en" }),
  tamano_texto: Object.freeze({ normal: "ui.usuarios.preferencias.tamano_texto.normal", grande: "ui.usuarios.preferencias.tamano_texto.grande", muy_grande: "ui.usuarios.preferencias.tamano_texto.muy_grande" }),
  tema: Object.freeze({ sistema: "ui.usuarios.preferencias.tema.sistema", claro: "ui.usuarios.preferencias.tema.claro", oscuro: "ui.usuarios.preferencias.tema.oscuro" }),
  inicio: Object.freeze({ cuadro: "ui.usuarios.preferencias.inicio.cuadro", peticiones: "ui.usuarios.preferencias.inicio.peticiones", bolsas: "ui.usuarios.preferencias.inicio.bolsas" }),
});
function escapar(valor) { return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;"); }
function t(clave, variables) { return traducirPortal(clave, variables); }
function ayuda(campo) {
  const etiqueta = t(`preferencias_${ETIQUETAS[campo]}`);
  return `<button type="button" class="pref-ayuda" data-pref-ayuda="${campo}" aria-controls="pref-ayuda-${campo}" aria-expanded="false" aria-label="${escapar(t("preferencias_ayuda", { campo: etiqueta }))}">?</button><p id="pref-ayuda-${campo}" class="pref-ayuda-texto" hidden>${escapar(t(`preferencias_ayuda_${AYUDAS[campo]}`))}</p>`;
}
function campoSelect(campo, catalogo, valor) {
  const opciones = catalogo[CAMPOS_SELECT[campo]];
  return `<div class="pref-campo"><div class="pref-etiqueta"><label for="pref-${campo}">${escapar(t(`preferencias_${ETIQUETAS[campo]}`))}</label>${ayuda(campo)}</div><select id="pref-${campo}" name="${campo}" required>${opciones.map((opcion) => {
    const codigo = campo === "filas" ? opcion : opcion.codigo;
    const clave = ETIQUETAS_CATALOGO[campo]?.[codigo];
    const texto = campo === "filas" ? String(codigo) : clave ? t(clave) : String(codigo);
    return `<option value="${escapar(codigo)}"${codigo === valor ? " selected" : ""}>${escapar(texto)}</option>`;
  }).join("")}</select></div>`;
}
function campoBooleano(campo, valor) {
  return `<div class="pref-campo"><div class="pref-etiqueta"><label for="pref-${campo}">${escapar(t(`preferencias_${ETIQUETAS[campo]}`))}</label>${ayuda(campo)}</div><select id="pref-${campo}" name="${campo}"><option value="true"${valor ? " selected" : ""}>${escapar(t("preferencias_si"))}</option><option value="false"${!valor ? " selected" : ""}>${escapar(t("preferencias_no"))}</option></select></div>`;
}
function panel(titulo, subtitulo, campos) {
  return `<section class="panel pref-panel"><div class="cabecera-panel"><div><h3>${escapar(t(titulo))}</h3><p>${escapar(t(subtitulo))}</p></div></div><div class="cuerpo-panel pref-campos">${campos}</div></section>`;
}
function mensajeError(error, alGuardar = false) {
  if (error instanceof ErrorPreferencias && error.clave_i18n) return t(error.clave_i18n);
  if (error?.estado === 401) return t("preferencias_sesion");
  if (error?.estado === 403) return t("preferencias_denegado");
  if (error?.estado === 409) return t("preferencias_conflicto");
  if (error?.estado === 422) return t("preferencias_validacion");
  return t(alGuardar ? "preferencias_error_guardar" : "preferencias_error");
}

export function crearSuperficiePreferenciasPortal({ cliente = crearClientePreferencias(), actualizar = () => {}, alCargar = () => {}, alGuardar = () => {} } = {}) {
  let datos = null;
  let estado = "sin_cargar";
  let error = null;
  let recibo = null;
  let controlador = null;
  let generacion = 0;

  async function cargar() {
    controlador?.abort();
    controlador = new AbortController();
    const actual = ++generacion;
    datos = null; recibo = null; error = null; estado = "cargando"; actualizar();
    try {
      const nuevos = await cliente.consultar({ signal: controlador.signal });
      if (actual !== generacion) return;
      datos = nuevos; estado = "lista"; actualizar(); alCargar(nuevos);
    } catch (fallo) {
      if (actual !== generacion || controlador.signal.aborted) return;
      datos = null; estado = "error"; error = fallo; actualizar();
    }
  }
  function desmontarPeticion() { controlador?.abort(); ++generacion; }
  async function guardar(formulario) {
    if (estado !== "lista" || !datos) return;
    const campos = new FormData(formulario);
    const valores = Object.fromEntries(Object.keys(datos.estado.valores).map((campo) => [campo,
      ["alto_contraste", "aviso_correo_tareas", "aviso_correo_plazos"].includes(campo) ? campos.get(campo) === "true"
        : campo === "filas" ? Number(campos.get(campo)) : campos.get(campo)]));
    for (const [campo, nombre] of Object.entries(CAMPOS_SELECT)) {
      const opciones = datos.catalogo[nombre].map((opcion) => campo === "filas" ? opcion : opcion.codigo);
      if (!opciones.includes(valores[campo])) return;
    }
    const clave = globalThis.crypto?.randomUUID?.();
    if (!clave) { error = new ErrorPreferencias(0); actualizar(); return; }
    controlador?.abort(); controlador = new AbortController();
    const actual = ++generacion;
    estado = "guardando"; error = null; recibo = null; actualizar();
    try {
      const nuevoRecibo = await cliente.guardar({ version: datos.estado.version, catalogoVersion: datos.catalogo.version_ref,
        clave, valores, signal: controlador.signal });
      if (actual !== generacion) return;
      recibo = nuevoRecibo;
      datos = { ...datos, estado: { version: recibo.version, valores: recibo.valores } };
      estado = "lista"; actualizar(); alGuardar(nuevoRecibo);
    } catch (fallo) {
      if (actual !== generacion || controlador.signal.aborted) return;
      error = fallo;
      if (fallo?.estado === 401 || fallo?.estado === 403) {
        datos = null; estado = "error";
      } else estado = "guardado_incierto";
      actualizar();
    }
  }
  function renderizar() {
    const encabezado = `<div class="pref-encabezado"><p>${escapar(t("preferencias_intro"))}</p></div>`;
    if (estado === "sin_cargar" || estado === "cargando") return `${encabezado}<section class="panel pref-panel" role="status" aria-busy="true"><div class="cuerpo-panel">${escapar(t("preferencias_cargando"))}</div></section>`;
    if (!datos) return `${encabezado}<section class="panel pref-panel" role="alert"><div class="cuerpo-panel"><p>${escapar(mensajeError(error))}</p><button type="button" class="boton-secundario" data-pref-reintentar>${escapar(t("preferencias_reintentar"))}</button></div></section>`;
    const v = datos.estado.valores;
    const estadoTexto = datos.estado.version === 0 ? t("preferencias_no_guardadas") : t("preferencias_guardadas", { version: datos.estado.version });
    const aviso = error ? `<p class="pref-aviso pref-aviso--error" role="alert">${escapar(mensajeError(error, true))} <button type="button" class="boton-secundario" data-pref-reintentar>${escapar(t("preferencias_reintentar"))}</button></p>` : "";
    const confirmado = recibo ? `<p class="pref-aviso pref-aviso--exito" role="status">${escapar(t("preferencias_exito", { recibo: recibo.recibo_ref }))} ${escapar(t("preferencias_idioma_guardado"))}</p>` : "";
    return `${encabezado}<p class="pref-estado" role="status">${escapar(estadoTexto)}</p>${aviso}${confirmado}<form id="formulario-preferencias" class="pref-formulario">${panel("preferencias_visual", "preferencias_visual_sub", campoSelect("idioma", datos.catalogo, v.idioma) + campoSelect("tamano_texto", datos.catalogo, v.tamano_texto) + campoBooleano("alto_contraste", v.alto_contraste) + campoSelect("tema", datos.catalogo, v.tema))}${panel("preferencias_navegacion", "preferencias_navegacion_sub", campoSelect("inicio", datos.catalogo, v.inicio) + campoSelect("filas", datos.catalogo, v.filas))}${panel("preferencias_avisos", "preferencias_avisos_sub", campoBooleano("aviso_correo_tareas", v.aviso_correo_tareas) + campoBooleano("aviso_correo_plazos", v.aviso_correo_plazos))}<div class="pref-acciones"><button class="boton-primario" type="submit"${estado !== "lista" ? " disabled" : ""}>${escapar(t(estado === "guardando" ? "preferencias_guardando" : "preferencias_guardar"))}</button></div></form>`;
  }
  function instalar(contenedor) {
    const clic = (evento) => {
      const ayudaBoton = evento.target.closest?.("[data-pref-ayuda]");
      if (ayudaBoton && contenedor.contains(ayudaBoton)) {
        const texto = contenedor.querySelector(`#pref-ayuda-${ayudaBoton.dataset.prefAyuda}`);
        if (texto) { texto.hidden = !texto.hidden; ayudaBoton.setAttribute("aria-expanded", String(!texto.hidden)); }
      }
      if (evento.target.closest?.("[data-pref-reintentar]")) void cargar();
    };
    const enviar = (evento) => {
      if (evento.target?.id !== "formulario-preferencias") return;
      evento.preventDefault(); void guardar(evento.target);
    };
    contenedor.addEventListener("click", clic); contenedor.addEventListener("submit", enviar);
    return () => { contenedor.removeEventListener("click", clic); contenedor.removeEventListener("submit", enviar); desmontarPeticion(); };
  }
  return Object.freeze({ cargar, renderizar, instalar, desmontarPeticion, leer: () => datos, leerCarga: () => estado });
}
