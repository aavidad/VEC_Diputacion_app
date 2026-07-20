/**
 * Selector de identidades exclusivamente demostrativas.
 *
 * La elección viaja en la URL y provoca un arranque nuevo de la superficie
 * elegida. No concede permisos, no cambia una sesión real y no usa cookies ni
 * almacenamiento del navegador.
 */

const CATALOGO_PERFILES = Object.freeze([
  Object.freeze({
    clave: "usuario_externo",
    etiqueta: "Usuario externo",
    descripcion: "Área personal de aspirante: convocatorias, solicitudes y datos propios.",
    sigla: "UE",
    ruta: "/area-personal/?presentacion=rrhh&vista=inicio",
  }),
  Object.freeze({
    clave: "funcionario",
    etiqueta: "Funcionario",
    descripcion: "Autoservicio interno: Cronos y Dietas, sin gestión técnica de RRHH.",
    sigla: "FU",
    ruta: "/portal-empleado/?presentacion=rrhh&perfil=funcionario#portal",
  }),
  Object.freeze({
    clave: "tecnico",
    etiqueta: "Técnico de RRHH",
    descripcion: "Revisión de solicitudes, méritos, alegaciones y documentos autorizados.",
    sigla: "TR",
    ruta: "/portal-empleado/?presentacion=rrhh&perfil=tecnico#portal",
  }),
  Object.freeze({
    clave: "administrador",
    etiqueta: "Administrador",
    descripcion: "Configuración y gestión funcional completa de la demostración.",
    sigla: "AD",
    ruta: "/portal-empleado/?presentacion=rrhh&perfil=administrador#portal",
  }),
]);

const CLAVES_PERFIL = new Set(CATALOGO_PERFILES.map(({ clave }) => clave));
const RUTA_ESTILOS = "/presentacion/selector-perfiles.css?v=20260720-selector-perfiles-v1";

export function obtenerPerfilesPresentacion() {
  return CATALOGO_PERFILES;
}

export function obtenerRutaPerfilPresentacion(clave) {
  const perfil = CATALOGO_PERFILES.find((item) => item.clave === clave);
  if (!perfil) throw new TypeError("perfil de presentación no reconocido");
  return perfil.ruta;
}

function crearElemento(documento, etiqueta, clase = "", texto = "") {
  const elemento = documento.createElement(etiqueta);
  if (clase) elemento.className = clase;
  if (texto) elemento.textContent = texto;
  return elemento;
}

function asegurarEstilos(documento) {
  if (documento.querySelector(`link[href="${RUTA_ESTILOS}"]`)) return;
  const enlace = documento.createElement("link");
  enlace.rel = "stylesheet";
  enlace.href = RUTA_ESTILOS;
  enlace.dataset.estilosSelectorPerfiles = "true";
  documento.head.append(enlace);
}

function crearPanel(documento, idPanel, idTitulo, perfilActivo) {
  const panel = crearElemento(documento, "section", "selector-perfiles-presentacion");
  panel.id = idPanel;
  panel.hidden = true;
  panel.setAttribute("role", "dialog");
  panel.setAttribute("aria-modal", "false");
  panel.setAttribute("aria-labelledby", idTitulo);

  const cabecera = crearElemento(documento, "header", "selector-perfiles-cabecera");
  const bloqueTitulo = crearElemento(documento, "div");
  const sobrelinea = crearElemento(documento, "p", "selector-perfiles-sobrelinea", "Demostración");
  const titulo = crearElemento(documento, "h2", "", "Cambiar punto de vista");
  titulo.id = idTitulo;
  const aviso = crearElemento(documento, "p", "selector-perfiles-aviso", "La elección no cambia permisos reales ni persiste al cerrar la demostración.");
  bloqueTitulo.append(sobrelinea, titulo);
  cabecera.append(bloqueTitulo);

  const lista = crearElemento(documento, "ul", "selector-perfiles-lista");
  for (const perfil of CATALOGO_PERFILES) {
    const elemento = crearElemento(documento, "li");
    const enlace = crearElemento(documento, "a", "selector-perfil-enlace");
    enlace.href = perfil.ruta;
    enlace.dataset.perfilPresentacion = perfil.clave;
    const sigla = crearElemento(documento, "span", "selector-perfil-sigla", perfil.sigla);
    sigla.setAttribute("aria-hidden", "true");
    const contenido = crearElemento(documento, "span", "selector-perfil-contenido");
    const etiqueta = crearElemento(documento, "strong", "", perfil.etiqueta);
    const descripcion = crearElemento(documento, "small", "", perfil.descripcion);
    contenido.append(etiqueta, descripcion);
    enlace.append(sigla, contenido);
    if (perfil.clave === perfilActivo) {
      enlace.setAttribute("aria-current", "page");
      enlace.append(crearElemento(documento, "span", "selector-perfil-actual", "Perfil actual"));
    }
    elemento.append(enlace);
    lista.append(elemento);
  }
  panel.append(cabecera, aviso, lista);
  return panel;
}

/** Instala el selector en un bloque de sesión ya existente. */
export function instalarSelectorPerfilesPresentacion({
  disparador,
  perfilActivo,
  documento = globalThis.document,
} = {}) {
  if (!disparador || typeof disparador.addEventListener !== "function"
    || !documento || typeof documento.createElement !== "function"
    || !CLAVES_PERFIL.has(perfilActivo)) {
    throw new TypeError("configuración del selector de perfiles no válida");
  }

  const sufijo = perfilActivo.replaceAll("_", "-");
  asegurarEstilos(documento);
  const idDisparador = disparador.id || `selector-perfiles-disparador-${sufijo}`;
  const idPanel = `selector-perfiles-panel-${sufijo}`;
  const idTitulo = `selector-perfiles-titulo-${sufijo}`;
  disparador.id = idDisparador;
  if ("disabled" in disparador) disparador.disabled = false;
  if (disparador.tagName !== "BUTTON") {
    disparador.setAttribute("role", "button");
    disparador.setAttribute("tabindex", "0");
  }
  disparador.removeAttribute("data-accion");
  disparador.setAttribute("aria-haspopup", "dialog");
  disparador.setAttribute("aria-expanded", "false");
  disparador.setAttribute("aria-controls", idPanel);
  disparador.dataset.selectorPerfil = "true";
  const perfil = CATALOGO_PERFILES.find((item) => item.clave === perfilActivo);
  disparador.setAttribute("aria-label", `Cambiar perfil de demostración. Perfil actual: ${perfil.etiqueta}`);

  const panel = crearPanel(documento, idPanel, idTitulo, perfilActivo);
  disparador.insertAdjacentElement("afterend", panel);
  documento.body.dataset.selectorPerfilesPresentacion = "true";

  const enlaces = [...panel.querySelectorAll("a[data-perfil-presentacion]")];
  const cerrar = ({ restaurarFoco = false } = {}) => {
    panel.hidden = true;
    disparador.setAttribute("aria-expanded", "false");
    if (restaurarFoco) disparador.focus({ preventScroll: true });
  };
  const abrir = () => {
    panel.hidden = false;
    disparador.setAttribute("aria-expanded", "true");
    (panel.querySelector('[aria-current="page"]') || enlaces[0])?.focus({ preventScroll: true });
  };
  const alternar = () => { if (panel.hidden) abrir(); else cerrar({ restaurarFoco: true }); };

  const alPulsarDisparador = (evento) => {
    evento.preventDefault();
    evento.stopPropagation();
    alternar();
  };
  const alTecladoDisparador = (evento) => {
    if (!["ArrowDown", "Enter", " "].includes(evento.key)) return;
    evento.preventDefault();
    abrir();
  };
  const alTecladoPanel = (evento) => {
    if (evento.key === "Escape") {
      evento.preventDefault();
      cerrar({ restaurarFoco: true });
      return;
    }
    if (!["ArrowDown", "ArrowUp", "Home", "End"].includes(evento.key)) return;
    evento.preventDefault();
    const indice = Math.max(0, enlaces.indexOf(documento.activeElement));
    const destino = evento.key === "Home" ? 0 : evento.key === "End" ? enlaces.length - 1
      : evento.key === "ArrowDown" ? (indice + 1) % enlaces.length
        : (indice - 1 + enlaces.length) % enlaces.length;
    enlaces[destino]?.focus({ preventScroll: true });
  };
  const alPulsarDocumento = (evento) => {
    if (!panel.hidden && !panel.contains(evento.target) && !disparador.contains(evento.target)) cerrar();
  };

  disparador.addEventListener("click", alPulsarDisparador);
  disparador.addEventListener("keydown", alTecladoDisparador);
  panel.addEventListener("keydown", alTecladoPanel);
  documento.addEventListener("click", alPulsarDocumento);

  return Object.freeze({
    cerrar,
    destruir() {
      disparador.removeEventListener("click", alPulsarDisparador);
      disparador.removeEventListener("keydown", alTecladoDisparador);
      panel.removeEventListener("keydown", alTecladoPanel);
      documento.removeEventListener("click", alPulsarDocumento);
      delete disparador.dataset.selectorPerfil;
      if (disparador.tagName !== "BUTTON") {
        disparador.removeAttribute("role");
        disparador.removeAttribute("tabindex");
      }
      panel.remove();
    },
  });
}
