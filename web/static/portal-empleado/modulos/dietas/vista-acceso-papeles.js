import { crearTraductorD1Dietas } from "./i18n-d1.js?v=20260924-dietas-ayuda-sin-guia-v2";

const PAPELES = Object.freeze([
  Object.freeze({ codigo: "empleado", nombre: "d1_empleado", tarea: "d1_empleado_tarea" }),
  Object.freeze({ codigo: "administrativo_servicio", nombre: "d1_administrativo", tarea: "d1_administrativo_tarea" }),
  Object.freeze({ codigo: "responsable_centro", nombre: "d1_responsable", tarea: "d1_responsable_tarea" }),
  Object.freeze({ codigo: "rrhh", nombre: "d1_rrhh", tarea: "d1_rrhh_tarea" }),
  Object.freeze({ codigo: "intervencion", nombre: "d1_intervencion", tarea: "d1_intervencion_tarea" }),
]);

const nodo = (documento, etiqueta, texto = "") => {
  const elemento = documento.createElement(etiqueta);
  elemento.textContent = texto;
  return elemento;
};

/**
 * Encaja en un contenedor exclusivo junto al recorrido de Dietas.
 * `estadosVerificados` es una proyección de lectura de la composición autorizada,
 * nunca un selector de perfil ni una fuente de permisos. Sin ella, todo queda cerrado.
 */
export function montarVistaAccesoPapelesDietas(contenedor, {
  traducir = crearTraductorD1Dietas(),
  estadosVerificados,
  registrarDesmontar,
} = {}) {
  if (!contenedor?.ownerDocument?.createElement || typeof contenedor.replaceChildren !== "function" ||
      typeof traducir !== "function" ||
      (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function"))
    throw new TypeError("vista de acceso de Dietas no disponible");

  const documento = contenedor.ownerDocument;
  const t = crearTraductorD1Dietas(traducir);
  const raiz = nodo(documento, "section");
  raiz.className = "modulo-dietas dietas-acceso-papeles";
  raiz.dataset.dietasAccesoPapeles = "";
  raiz.setAttribute("aria-label", t("d1_titulo"));

  const introduccion = nodo(documento, "section");
  introduccion.className = "panel";
  const cabecera = nodo(documento, "div");
  cabecera.className = "cabecera-panel";
  const titulos = nodo(documento, "div");
  titulos.append(nodo(documento, "h2", t("d1_titulo")), nodo(documento, "p", t("d1_subtitulo")));
  const ayuda = nodo(documento, "details");
  ayuda.className = "dietas-recorridos-ayuda";
  const abrirAyuda = nodo(documento, "summary", "?");
  abrirAyuda.setAttribute("aria-label", t("d1_ayuda_etiqueta"));
  ayuda.append(abrirAyuda, nodo(documento, "p", t("d1_ayuda")));
  cabecera.append(titulos, ayuda);
  const cuerpo = nodo(documento, "div");
  cuerpo.className = "cuerpo-panel";
  const limite = nodo(documento, "p", t("d1_limite"));
  limite.setAttribute("role", "status");
  cuerpo.append(limite);
  introduccion.append(cabecera, cuerpo);

  const tarjetas = nodo(documento, "div");
  tarjetas.className = "rejilla-modulos";
  tarjetas.dataset.dietasPapeles = "";
  for (const papel of PAPELES) {
    const disponible = estadosVerificados && Object.hasOwn(estadosVerificados, papel.codigo) &&
      estadosVerificados[papel.codigo] === "disponible";
    const tarjeta = nodo(documento, "article");
    tarjeta.className = `tarjeta-modulo${disponible ? "" : " tarjeta-modulo-bloqueada"}`;
    tarjeta.dataset.dietasPapel = papel.codigo;
    const encabezado = nodo(documento, "h3", t(papel.nombre));
    const tarea = nodo(documento, "p", t(papel.tarea));
    const pie = nodo(documento, "div");
    pie.className = "pie-tarjeta";
    const estado = nodo(documento, "span", t(disponible ? "d1_estado_disponible" : "d1_estado_no_configurado"));
    estado.className = `estado-chip ${disponible ? "exito" : "neutro"}`;
    estado.dataset.dietasEstadoPapel = disponible ? "disponible" : "no_configurado";
    pie.append(estado);
    tarjeta.append(encabezado, tarea, pie);
    tarjetas.append(tarjeta);
  }
  raiz.append(introduccion, tarjetas);
  contenedor.replaceChildren(raiz);

  let activa = true;
  function desmontar() {
    if (!activa) return;
    activa = false;
    if (raiz.parentNode === contenedor) contenedor.removeChild(raiz);
  }
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar });
}
