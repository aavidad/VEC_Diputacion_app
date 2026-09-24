const RUTA_CATALOGO_ES = "/area-personal/locales/es.json";
const respaldo = Object.freeze({
  "areaPersonal.rutas.inicio": "Inicio y plazos", "areaPersonal.rutas.convocatorias": "Convocatorias", "areaPersonal.rutas.convocatoria": "Detalle de convocatoria", "areaPersonal.rutas.perfil": "Perfil y contacto", "areaPersonal.rutas.meritos": "Méritos y documentos", "areaPersonal.rutas.solicitud": "Nueva solicitud", "areaPersonal.rutas.autobaremacion": "Autobaremación", "areaPersonal.rutas.seguimiento": "Mis expedientes", "areaPersonal.rutas.llamamientos": "Disponibilidad y llamamientos", "areaPersonal.rutas.subsanaciones": "Subsanaciones", "areaPersonal.rutas.alegaciones": "Alegaciones", "areaPersonal.rutas.mensajes": "Mensajes y noticias", "areaPersonal.rutas.certificados": "Certificados y descargas", "areaPersonal.rutas.ayuda": "Ayuda y accesibilidad",
  "areaPersonal.tabla.sinResultados": "Sin resultados", "areaPersonal.tabla.sinRegistros": "No hay registros para mostrar.",
  "areaPersonal.estado.error.titulo": "No se pudo cargar el área personal", "areaPersonal.estado.error.detalle": "Servicio no disponible.", "areaPersonal.estado.error.garantia": "No se muestran datos aparentes y no se ha realizado ninguna operación.", "areaPersonal.estado.error.reintentar": "Reintentar conexión segura",
  "areaPersonal.estado.error.autenticacion.titulo": "Identifíquese para consultar su bolsa",
  "areaPersonal.estado.error.autenticacion.detalle": "Use su DNIe o certificado para continuar. Identificarse con certificado no firma documentos.",
  "areaPersonal.estado.error.acceso.titulo": "No tiene acceso a esta área personal",
  "areaPersonal.estado.error.acceso.detalle": "El servicio no ha autorizado esta consulta para su identidad.",
  "areaPersonal.estado.error.acceso.garantia": "No se muestran datos de otra persona.",
  "areaPersonal.estado.error.recurso.titulo": "No se pudo acceder a la información solicitada",
  "areaPersonal.estado.error.recurso.detalle": "No hay una consulta disponible para esta identidad y ámbito.",
  "areaPersonal.estado.error.servicio.detalle": "El servicio no está disponible temporalmente. Vuelva a intentarlo más tarde.",
  "areaPersonal.estado.error.red.detalle": "No se pudo establecer una conexión segura con el servicio.",
  "areaPersonal.estado.error.carga.garantia": "No se muestran datos ni se confirma el resultado de operaciones anteriores.",
  "areaPersonal.capacidad.noHabilitada": "{operacion} no está habilitada para la identidad y el expediente actuales.", "areaPersonal.capacidad.accionNoReconocida": "La acción no está reconocida por esta superficie.", "areaPersonal.capacidad.operacionNoDisponible": "Operación no disponible.", "areaPersonal.documento.noDisponible": "El documento solicitado no está disponible en el ámbito actual.",
  "areaPersonal.miBolsa.situacion.titulo": "Última situación registrada de mi participación",
  "areaPersonal.miBolsa.ordenInicial": "Mi número de orden inicial", "areaPersonal.miBolsa.vigenciaBolsa": "Vigencia de la bolsa",
  "areaPersonal.miBolsa.situacion.sinDato": "La situación actual de esta participación aún no está disponible en Mi bolsa.",
  "areaPersonal.miBolsa.situacion.estado": "Estado", "areaPersonal.miBolsa.situacion.desde": "Desde", "areaPersonal.miBolsa.situacion.hasta": "Hasta",
  "areaPersonal.miBolsa.situacion.sinFin": "Sin fecha de fin registrada", "areaPersonal.miBolsa.situacion.fechaDisponible": "Fecha de disponibilidad indicada",
  "areaPersonal.miBolsa.situacion.explicacion": "Qué significa",
  "areaPersonal.miBolsa.situacion.disponible": "Disponible", "areaPersonal.miBolsa.situacion.no_disponible": "No disponible",
  "areaPersonal.miBolsa.situacion.trabajando": "Trabajando", "areaPersonal.miBolsa.situacion.pendiente_incorporacion": "Pendiente de incorporación",
  "areaPersonal.miBolsa.situacion.renuncia": "Renuncia", "areaPersonal.miBolsa.situacion.excluido": "Excluido",
  "areaPersonal.miBolsa.situacion.disponible_desde": "Disponible desde fecha",
  "areaPersonal.miBolsa.situacion.explicacion.disponible": "Consta disponible en Bolsa; un llamamiento depende del orden y las reglas aplicables.",
  "areaPersonal.miBolsa.situacion.explicacion.no_disponible": "Consta temporalmente no disponible en Bolsa.",
  "areaPersonal.miBolsa.situacion.explicacion.trabajando": "Consta una situación de trabajo comunicada a Bolsa.",
  "areaPersonal.miBolsa.situacion.explicacion.pendiente_incorporacion": "Consta una incorporación pendiente; todavía no acredita una relación de servicio.",
  "areaPersonal.miBolsa.situacion.explicacion.renuncia": "Consta una renuncia registrada para esta participación.",
  "areaPersonal.miBolsa.situacion.explicacion.excluido": "Consta una exclusión registrada para esta participación.",
  "areaPersonal.miBolsa.situacion.explicacion.disponible_desde": "Consta una fecha indicada para recuperar disponibilidad; no acredita un llamamiento ni determina por sí sola los efectos legales.",
  "areaPersonal.miBolsa.disponibilidad.titulo": "Disponibilidad", "areaPersonal.miBolsa.disponibilidad.subtitulo": "Situación por participación",
  "areaPersonal.miBolsa.disponibilidad.detalle": "Consulte arriba la situación de cada participación. Las acciones de pausa o reactivación aún no están habilitadas en Mi bolsa.",
  "areaPersonal.miBolsa.llamamiento.titulo": "Último resultado de correo", "areaPersonal.miBolsa.llamamiento.subtitulo": "De mis llamamientos en Bolsa",
  "areaPersonal.miBolsa.llamamiento.bolsa": "Bolsa", "areaPersonal.miBolsa.llamamiento.categoria": "Categoría", "areaPersonal.miBolsa.llamamiento.fecha": "Emisión registrada", "areaPersonal.miBolsa.llamamiento.canal": "Canal", "areaPersonal.miBolsa.llamamiento.correo": "Correo",
  "areaPersonal.miBolsa.llamamiento.resultado": "Resultado", "areaPersonal.miBolsa.llamamiento.enviado": "Enviado", "areaPersonal.miBolsa.llamamiento.no_enviado": "No enviado",
  "areaPersonal.miBolsa.llamamiento.limite": "El resultado de envío no acredita recepción, respuesta ni plazo aprobado.",
  "areaPersonal.miBolsa.llamamiento.sinDato": "No consta un resultado de correo B7 para estas participaciones. Esta consulta no muestra otros llamamientos o contactos.",
  "areaPersonal.demo.nota": "Recorrido de demostración. Los títulos, CVE y fechas de publicación BOP son referencias públicas reales. La identidad, los expedientes, los plazos operativos, las puntuaciones y todas las acciones son sintéticos; solo viven en memoria y generan recibos DEMO sin validez administrativa."
});
let catalogo = respaldo;
export function traducir(clave, variables = {}) { const mensaje = catalogo[clave] ?? respaldo[clave] ?? clave; return mensaje.replace(/\{([a-z_]+)\}/giu, (_, nombre) => String(variables[nombre] ?? "")); }
const CLAVES_ERROR_CARGA = Object.freeze({
  autenticacion_requerida: Object.freeze({ titulo: "autenticacion.titulo", detalle: "autenticacion.detalle" }),
  acceso_denegado: Object.freeze({ titulo: "acceso.titulo", detalle: "acceso.detalle", garantia: "acceso.garantia", reintentar: false }),
  recurso_no_encontrado: Object.freeze({ titulo: "recurso.titulo", detalle: "recurso.detalle" }),
  servicio_no_disponible: Object.freeze({ detalle: "servicio.detalle" }),
});
export function textosErrorCargaAreaPersonal(error) {
  const base = "areaPersonal.estado.error.";
  const claves = Object.hasOwn(CLAVES_ERROR_CARGA, error?.codigo) ? CLAVES_ERROR_CARGA[error.codigo] : {};
  const detalle = error?.codigo === "servicio_no_disponible" && error?.cause ? "red.detalle" : claves.detalle ?? "detalle";
  return Object.freeze({
    titulo: traducir(`${base}${claves.titulo ?? "titulo"}`), detalle: traducir(`${base}${detalle}`),
    garantia: traducir(`${base}${claves.garantia ?? "carga.garantia"}`),
    reintentar: claves.reintentar === false ? "" : traducir(`${base}reintentar`),
  });
}
export function rutaCatalogoAreaPersonal() { return RUTA_CATALOGO_ES; }
export function aplicarCatalogoAreaPersonal(documento, entradas) { if (!documento?.querySelectorAll || !entradas || typeof entradas !== "object" || Array.isArray(entradas)) return; documento.querySelectorAll("[data-i18n]").forEach((elemento) => { const clave = elemento.getAttribute("data-i18n"); if (typeof entradas[clave] === "string") elemento.textContent = entradas[clave]; }); }
export async function iniciarI18nAreaPersonal(documento = document, fetcher = fetch) { try { const respuesta = await fetcher(rutaCatalogoAreaPersonal(), { credentials: "omit" }); if (!respuesta?.ok) return "es"; const cargado = await respuesta.json(); if (!cargado || typeof cargado !== "object" || Array.isArray(cargado)) return "es"; catalogo = Object.freeze({ ...respaldo, ...cargado }); aplicarCatalogoAreaPersonal(documento, catalogo); } catch { /* El HTML y las rutas incluyen castellano de respaldo sin persistir preferencias. */ } return "es"; }
