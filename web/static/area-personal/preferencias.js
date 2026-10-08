import { escaparHTML } from "./vistas/comunes.js";
import { IDIOMAS_DISPONIBLES } from "../comun/idioma.js";
import { idiomaActivoAreaPersonal, textosPreferenciasAreaPersonal, traducir } from "./i18n.js";
import { crearClienteCorreos, crearSuperficieCorreos } from "../comun/correos-propios.js?v=20260929-correos-508b-v1";
import { crearAvatarCabecera, crearClienteImagen, crearSuperficieImagen, peticionesEnSerie } from "../comun/imagen-propia.js?v=20261007-p7-http-v1";

// i18n.js ya carga el idioma activo y prepara el índice antes de este módulo.
const P = (clave, variables = {}) => traducir(`areaPersonal.preferencias.${clave}`, variables);
const CAMPOS = Object.freeze([
  ["idioma", "idiomas"], ["tamano_texto", "tamanos_texto"],
  ["tema", "temas"], ["inicio", "inicios"], ["filas", "filas"],
]);
const OPCIONES = Object.freeze({
  idioma: ["navegador", ...IDIOMAS_DISPONIBLES.map(({ codigo }) => codigo)], tamano_texto: ["normal", "grande", "muy_grande"],
  tema: ["sistema", "claro", "oscuro", "diputacion_granada", "arena", "salvia", "lavanda", "azul_sereno", "noche_suave"],
  inicio: ["cuadro", "peticiones", "bolsas"],
});

export function errorPreferencias(error) {
  return P(`error.${["autenticacion", "denegado", "conflicto", "validacion", "servicio", "respuesta"].includes(error?.codigo)
    ? error.codigo : "servicio"}`);
}

function opcionesCampo(catalogo, campo, lista, elegido) {
  const elementos = lista === "filas"
    ? catalogo.filas.map((numero) => ({ codigo: numero, nombre: String(numero) }))
    : catalogo[lista].filter((opcion) => OPCIONES[campo].includes(opcion.codigo))
      .map((opcion) => ({ codigo: opcion.codigo, nombre: P(`opcion.${campo}.${opcion.codigo}`) }));
  return elementos.map(({ codigo, nombre }) => `<option value="${escaparHTML(codigo)}"${codigo === elegido ? " selected" : ""}>${escaparHTML(nombre)}</option>`).join("");
}

export function renderizarPreferencias(preferencias) {
  const { catalogo, estado, error, guardando, recibo, pendiente, avisoInicio, borrador } = preferencias || {};
  const mensaje = error ? `<p class="preferencias-estado preferencias-error" role="alert" tabindex="-1" data-pref-resultado>${escaparHTML(errorPreferencias(error))}</p>` : "";
  if (!catalogo || !estado) return `<section class="panel preferencias-panel"><header><div><h2 tabindex="-1">${escaparHTML(P("titulo"))}</h2></div></header><div class="panel-contenido">${mensaje}<p>${escaparHTML(P("sinDatos"))}</p><button type="button" class="boton-secundario" data-accion="recargar-preferencias">${escaparHTML(P("recargar"))}</button></div></section>`;
  const valores = borrador || estado.valores;
  const selects = CAMPOS.map(([campo, lista]) => `<div class="preferencias-campo"><label for="preferencia-${campo}">${escaparHTML(P(`campo.${campo}`))}</label><select id="preferencia-${campo}" name="${campo}" ${guardando ? "disabled" : ""}>${opcionesCampo(catalogo, campo, lista, valores[campo])}</select></div>`).join("");
  const booleanos = ["alto_contraste", "aviso_correo_tareas", "aviso_correo_plazos"].map((campo) => `<label class="preferencias-casilla"><input type="checkbox" name="${campo}" ${valores[campo] ? "checked" : ""} ${guardando ? "disabled" : ""}><span>${escaparHTML(P(`campo.${campo}`))}</span></label>`).join("");
  const confirmacion = recibo ? `<p class="preferencias-estado preferencias-exito" role="status" tabindex="-1" data-pref-resultado>${escaparHTML(P("guardado"))}</p>` : "";
  const incierto = pendiente ? `<p class="preferencias-estado preferencias-aviso" role="alert">${escaparHTML(P("incierto"))}</p><button type="button" class="boton-secundario" data-accion="reintentar-preferencias">${escaparHTML(P("reintentarExacto"))}</button>` : "";
  return `<section class="panel preferencias-panel"><header><div><h2 tabindex="-1">${escaparHTML(P("titulo"))}</h2></div><button type="button" class="boton-ayuda" data-accion="ayuda-preferencias" aria-expanded="false" aria-controls="ayuda-preferencias" aria-label="${escaparHTML(P("ayudaAbrir"))}">?</button></header><div class="panel-contenido"><div id="ayuda-preferencias" class="preferencias-ayuda" hidden><p>${escaparHTML(P("ayuda"))}</p></div>${avisoInicio ? `<p class="preferencias-estado preferencias-aviso">${escaparHTML(P("inicioAjeno"))}</p>` : ""}${mensaje}${confirmacion}${incierto}<form id="formulario-preferencias"><div class="preferencias-rejilla">${selects}<div class="preferencias-campo preferencias-opciones">${booleanos}</div></div><div class="fila-acciones"><button type="submit" class="boton-primario" ${guardando || pendiente ? "disabled" : ""}>${escaparHTML(guardando ? P("guardando") : P("guardar"))}</button><button type="button" class="boton-secundario" data-accion="recargar-preferencias" ${guardando ? "disabled" : ""}>${escaparHTML(P("recargar"))}</button></div></form></div></section>`;
}

export function valoresDelFormulario(formulario) {
  const datos = new FormData(formulario);
  return Object.freeze({
    idioma: String(datos.get("idioma") || ""),
    tamano_texto: String(datos.get("tamano_texto") || ""),
    alto_contraste: datos.has("alto_contraste"),
    tema: String(datos.get("tema") || ""),
    inicio: String(datos.get("inicio") || ""),
    filas: Number(datos.get("filas")),
    aviso_correo_tareas: datos.has("aviso_correo_tareas"),
    aviso_correo_plazos: datos.has("aviso_correo_plazos"),
  });
}

export function crearOperacionPreferencias(preferencias, valores, cryptoImpl = globalThis.crypto) {
  if (!preferencias?.estado || !preferencias.catalogo || typeof cryptoImpl?.randomUUID !== "function") {
    throw new Error("preferencias_operacion_invalida");
  }
  return Object.freeze({ version_esperada: preferencias.estado.version,
    catalogo_version_ref: preferencias.catalogo.version_ref,
    clave_operacion: `web-pref-${cryptoImpl.randomUUID()}`, valores });
}

/** Atajo de sesión de la cabecera: cambia texto o contraste sin guardar la preferencia. */
export function alternarVisualSesion(controlador, accion, documento = globalThis.document) {
  const texto = accion === "alternar-texto";
  let activo;
  if (controlador) {
    const actual = controlador.leerEstado().preferencias_servidor
      || { tema: "sistema", alto_contraste: false, tamano_texto: "normal" };
    const nuevos = texto
      ? { ...actual, tamano_texto: actual.tamano_texto === "normal" ? "grande" : "normal" }
      : { ...actual, alto_contraste: !actual.alto_contraste };
    controlador.aplicarPreferenciasServidor(nuevos);
    activo = texto ? nuevos.tamano_texto !== "normal" : nuevos.alto_contraste;
  } else {
    const destino = texto ? documento.documentElement : documento.body;
    const atributo = texto ? "textoGrande" : "contraste";
    activo = destino.dataset[atributo] !== "true";
    destino.dataset[atributo] = String(activo);
  }
  sincronizarAtajosVisuales(texto ? { tamano_texto: activo ? "grande" : "normal" } : { alto_contraste: activo }, documento);
  return activo;
}

/** Refleja en aria-pressed de los atajos el estado visual aplicado. */
export function sincronizarAtajosVisuales(valores, documento = globalThis.document) {
  if (!valores || !documento?.querySelectorAll) return;
  const marcar = (accion, activo) => documento.querySelectorAll(`[data-accion="${accion}"]`)
    .forEach((control) => control.setAttribute("aria-pressed", String(activo)));
  if (typeof valores.tamano_texto === "string") marcar("alternar-texto", valores.tamano_texto !== "normal");
  if (typeof valores.alto_contraste === "boolean") marcar("alternar-contraste", valores.alto_contraste);
}

const MARCO_AREA_PERSONAL = Object.freeze({ panel: "panel preferencias-panel", cabecera: "header", claseCabecera: "", cuerpo: "panel-contenido" });

function reflejarErrorImagen(documento, fallo) {
  const aviso = documento?.getElementById?.("aviso-imagen");
  if (!aviso) return;
  const estabaVisible = !aviso.hidden;
  aviso.hidden = !fallo;
  if (!fallo && estabaVisible) {
    const anuncio = documento.getElementById("anuncios");
    if (anuncio) anuncio.textContent = traducir("areaPersonal.imagen.cargada");
    if (documento.activeElement === documento.getElementById("reintentar-imagen")) {
      documento.getElementById("contenido-principal")?.focus?.({ preventScroll: true });
    }
  }
}

/** El fallo del avatar conserva iniciales y ofrece un reintento visible. */
export function reintentarImagenAreaPersonal(estado, documento = globalThis.document) {
  if (estado.lecturaImagenEnCurso) return estado.lecturaImagenEnCurso;
  const lectura = estado.clienteImagen.consultar();
  estado.lecturaImagenInicial = lectura;
  const resultado = lectura.then((vista) => {
    estado.avatar.fijarImagen(vista);
    reflejarErrorImagen(documento, false);
    return true;
  }, () => {
    reflejarErrorImagen(documento, true);
    return false;
  });
  estado.lecturaImagenEnCurso = resultado;
  void resultado.then(() => { if (estado.lecturaImagenEnCurso === resultado) estado.lecturaImagenEnCurso = null; });
  return resultado;
}

/** El avatar conserva su lectura sin cargar el catálogo de Preferencias. */
export function montarAvatarAreaPersonal(estado, fetchImpl = globalThis.fetch, documento = globalThis.document) {
  if (estado.avatar) return;
  const enSerie = peticionesEnSerie(fetchImpl);
  const avatar = crearAvatarCabecera(documento?.getElementById?.("avatar-sesion"));
  estado.avatar = avatar;
  estado.fetchUsuariosEnSerie = enSerie;
  estado.clienteImagen = crearClienteImagen({ ruta: "/api/vec/usuarios/area-personal/mi-imagen", fetchImpl: enSerie });
  void reintentarImagenAreaPersonal(estado, documento);
}

/** Imagen y Correos se montan cuando la persona abre Preferencias. */
export function montarUsuariosAreaPersonal(estado, fetchImpl = globalThis.fetch, contenedor = null, documento = globalThis.document) {
  if (estado.imagen && estado.correos && estado.idiomaUsuarios === idiomaActivoAreaPersonal()) return true;
  const textos = textosPreferenciasAreaPersonal();
  if (!textos) return false;
  estado.desmontarImagenUsuarios?.();
  estado.desmontarCorreosUsuarios?.();
  estado.desmontarImagenUsuarios = null;
  estado.desmontarCorreosUsuarios = null;
  estado.imagen = null;
  estado.correos = null;
  montarAvatarAreaPersonal(estado, fetchImpl, documento);
  try {
    const clienteImagen = Object.freeze({ ...estado.clienteImagen,
      consultar: (opciones) => {
        const inicial = estado.lecturaImagenInicial;
        estado.lecturaImagenInicial = null;
        return inicial ?? estado.clienteImagen.consultar(opciones);
      } });
    estado.imagen = crearSuperficieImagen({ cliente: clienteImagen,
      textos, marco: MARCO_AREA_PERSONAL, alCambiar: (vista) => estado.avatar.fijarImagen(vista), iniciales: () => estado.datos?.sesion?.iniciales ?? "" });
    estado.correos = crearSuperficieCorreos({ cliente: crearClienteCorreos({ ruta: "/api/vec/usuarios/area-personal/mis-correos", fetchImpl: estado.fetchUsuariosEnSerie }),
      textos, marco: MARCO_AREA_PERSONAL, cargaAlMostrar: true });
  } catch {
    estado.imagen = null;
    estado.correos = null;
    return false;
  }
  if (contenedor) {
    estado.desmontarImagenUsuarios = estado.imagen.instalar(contenedor);
    estado.desmontarCorreosUsuarios = estado.correos.instalar(contenedor);
  }
  estado.idiomaUsuarios = idiomaActivoAreaPersonal();
  void estado.imagen.cargar();
  return true;
}

/** Pone las iniciales de la sesión en el avatar sin pisar la imagen elegida. */
export function pintarInicialesSesion(estado, elemento, iniciales) {
  if (estado?.avatar) estado.avatar.fijarIniciales(iniciales);
  else if (elemento) elemento.textContent = iniciales;
}
