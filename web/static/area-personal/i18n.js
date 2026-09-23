const RUTA_CATALOGO_ES = "/area-personal/locales/es.json";
const respaldo = Object.freeze({
  "areaPersonal.rutas.inicio": "Inicio y plazos", "areaPersonal.rutas.convocatorias": "Convocatorias", "areaPersonal.rutas.convocatoria": "Detalle de convocatoria", "areaPersonal.rutas.perfil": "Perfil y contacto", "areaPersonal.rutas.meritos": "Méritos y documentos", "areaPersonal.rutas.solicitud": "Nueva solicitud", "areaPersonal.rutas.autobaremacion": "Autobaremación", "areaPersonal.rutas.seguimiento": "Mis expedientes", "areaPersonal.rutas.llamamientos": "Disponibilidad y llamamientos", "areaPersonal.rutas.subsanaciones": "Subsanaciones", "areaPersonal.rutas.alegaciones": "Alegaciones", "areaPersonal.rutas.mensajes": "Mensajes y noticias", "areaPersonal.rutas.certificados": "Certificados y descargas", "areaPersonal.rutas.ayuda": "Ayuda y accesibilidad",
  "areaPersonal.tabla.sinResultados": "Sin resultados", "areaPersonal.tabla.sinRegistros": "No hay registros para mostrar.",
  "areaPersonal.estado.error.titulo": "No se pudo cargar el área personal", "areaPersonal.estado.error.detalle": "Servicio no disponible.", "areaPersonal.estado.error.garantia": "No se muestran datos aparentes y no se ha realizado ninguna operación.", "areaPersonal.estado.error.reintentar": "Reintentar conexión segura",
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
  "areaPersonal.demo.nota": "Recorrido de demostración. Los títulos, CVE y fechas de publicación BOP son referencias públicas reales. La identidad, los expedientes, los plazos operativos, las puntuaciones y todas las acciones son sintéticos; solo viven en memoria y generan recibos DEMO sin validez administrativa."
});
let catalogo = respaldo;
export function traducir(clave, variables = {}) { const mensaje = catalogo[clave] ?? respaldo[clave] ?? clave; return mensaje.replace(/\{([a-z_]+)\}/giu, (_, nombre) => String(variables[nombre] ?? "")); }
export function rutaCatalogoAreaPersonal() { return RUTA_CATALOGO_ES; }
export function aplicarCatalogoAreaPersonal(documento, entradas) { if (!documento?.querySelectorAll || !entradas || typeof entradas !== "object" || Array.isArray(entradas)) return; documento.querySelectorAll("[data-i18n]").forEach((elemento) => { const clave = elemento.getAttribute("data-i18n"); if (typeof entradas[clave] === "string") elemento.textContent = entradas[clave]; }); }
export async function iniciarI18nAreaPersonal(documento = document, fetcher = fetch) { try { const respuesta = await fetcher(rutaCatalogoAreaPersonal(), { credentials: "omit" }); if (!respuesta?.ok) return "es"; const cargado = await respuesta.json(); if (!cargado || typeof cargado !== "object" || Array.isArray(cargado)) return "es"; catalogo = Object.freeze({ ...respaldo, ...cargado }); aplicarCatalogoAreaPersonal(documento, catalogo); } catch { /* El HTML y las rutas incluyen castellano de respaldo sin persistir preferencias. */ } return "es"; }
