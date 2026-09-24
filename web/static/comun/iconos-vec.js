/**
 * Catálogo cerrado de iconos de línea del portal VEC.
 *
 * Devuelve SVG en línea de 24×24 que heredan el color del texto (`currentColor`), sin
 * recursos externos. Son decorativos: llevan `aria-hidden` y el significado lo da siempre
 * el texto que los acompaña. Un nombre desconocido devuelve el icono genérico en lugar de
 * fallar, para que una vista nunca quede rota por un icono.
 */

const TRAZOS = Object.freeze({
  bolsas: '<rect x="3" y="7" width="18" height="13" rx="2"/><path d="M8 7V5a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2M3 12h18"/>',
  personas: '<circle cx="9" cy="8" r="3.5"/><path d="M2.5 20a6.5 6.5 0 0 1 13 0"/><path d="M16 4.6a3.5 3.5 0 0 1 0 6.8M18 14.2a6.5 6.5 0 0 1 3.5 5.8"/>',
  persona_ok: '<circle cx="10" cy="8" r="3.5"/><path d="M3.5 20a6.5 6.5 0 0 1 11-4.7"/><path d="m15.5 18 2 2 4-4.5"/>',
  persona_baja: '<circle cx="10" cy="8" r="3.5"/><path d="M3.5 20a6.5 6.5 0 0 1 11-4.7"/><path d="M16 18h6"/>',
  calendario: '<rect x="3" y="5" width="18" height="16" rx="2"/><path d="M3 10h18M8 3v4M16 3v4"/>',
  reloj: '<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>',
  documento: '<path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8z"/><path d="M14 3v5h5M9 13h6M9 17h6"/>',
  contrato: '<path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8z"/><path d="M14 3v5h5M9 13h4"/><path d="m9 18 1.5-1.5 1.5 1.5 3-3"/>',
  expediente: '<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>',
  pendiente: '<circle cx="12" cy="12" r="9"/><path d="M12 7v5h4"/>',
  alerta: '<path d="M10.3 4.3 2.6 18a2 2 0 0 0 1.7 3h15.4a2 2 0 0 0 1.7-3L13.7 4.3a2 2 0 0 0-3.4 0z"/><path d="M12 9v4M12 17h.01"/>',
  correcto: '<circle cx="12" cy="12" r="9"/><path d="m8 12 3 3 5-6"/>',
  en_curso: '<path d="M21 12a9 9 0 1 1-3-6.7"/><path d="M21 4v5h-5"/>',
  llamamiento: '<path d="M6 16V11a6 6 0 0 1 12 0v5l1.5 2h-15z"/><path d="M10 20a2 2 0 0 0 4 0"/>',
  correo: '<rect x="3" y="5" width="18" height="14" rx="2"/><path d="m3 7 9 6 9-6"/>',
  grafico: '<path d="M4 20V10M10 20V4M16 20v-7M22 20H2"/>',
  cobertura: '<path d="M3 17 9 11l4 4 8-8"/><path d="M15 7h6v6"/>',
  euro: '<path d="M17 6.5A7 7 0 1 0 17 17.5"/><path d="M4 10.5h9M4 13.5h9"/>',
  ruta: '<circle cx="6" cy="18" r="2.5"/><circle cx="18" cy="6" r="2.5"/><path d="M8.5 18H16a3 3 0 0 0 0-6H8a3 3 0 0 1 0-6h7.5"/>',
  reglas: '<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-2.9 1.2V21a2 2 0 0 1-4 0v-.1a1.7 1.7 0 0 0-2.9-1.2l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0-1.2-2.9H3a2 2 0 0 1 0-4h.1a1.7 1.7 0 0 0 1.2-2.9l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 2.9-1.2V3a2 2 0 0 1 4 0v.1a1.7 1.7 0 0 0 2.9 1.2l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0 1.2 2.9H21a2 2 0 0 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z"/>',
  auditoria: '<path d="M12 3 4 6v6c0 4.5 3.4 8.3 8 9 4.6-.7 8-4.5 8-9V6z"/><path d="m9 12 2 2 4-4"/>',
  buscar: '<circle cx="11" cy="11" r="7"/><path d="m20 20-4-4"/>',
  generico: '<rect x="4" y="4" width="16" height="16" rx="3"/><path d="M9 12h6"/>',
});

/** Nombres admitidos, para que las vistas y sus pruebas puedan comprobar el catálogo. */
export const NOMBRES_ICONO = Object.freeze(Object.keys(TRAZOS));

/** SVG decorativo del icono `nombre`; `clase` se añade al elemento raíz si es un nombre de clase simple. */
export function icono(nombre, clase = "") {
  const trazo = Object.hasOwn(TRAZOS, nombre) ? TRAZOS[nombre] : TRAZOS.generico;
  const claseSegura = /^[a-z][a-z0-9-]*$/.test(clase) ? ` class="${clase}"` : "";
  return `<svg${claseSegura} aria-hidden="true" focusable="false" viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">${trazo}</svg>`;
}
