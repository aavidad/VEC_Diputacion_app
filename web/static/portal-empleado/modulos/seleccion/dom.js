export function nodo(documento, etiqueta, texto, clase) {
  const elemento = documento.createElement(etiqueta);
  if (texto !== undefined) elemento.textContent = texto;
  if (clase) elemento.className = clase;
  return elemento;
}
export function panel(documento, titulo) {
  const elemento = nodo(documento, 'section', undefined, 'panel');
  const cabecera = nodo(documento, 'div', undefined, 'cabecera-panel');
  cabecera.append(nodo(documento, 'h2', titulo));
  const cuerpo = nodo(documento, 'div', undefined, 'cuerpo-panel pila');
  elemento.append(cabecera, cuerpo);
  return { elemento, cuerpo };
}
export function tabla(documento, titulos, titulo) {
  const contenedor = nodo(documento, 'div', undefined, 'tabla-contenedor');
  contenedor.tabIndex = 0; contenedor.setAttribute('role', 'region'); contenedor.setAttribute('aria-label', titulo);
  const elemento = nodo(documento, 'table', undefined, 'tabla-datos');
  elemento.append(nodo(documento, 'caption', titulo));
  const cabecera = nodo(documento, 'thead'); const fila = nodo(documento, 'tr');
  for (const texto of titulos) { const th = nodo(documento, 'th', texto); th.scope = 'col'; fila.append(th); }
  const cuerpo = nodo(documento, 'tbody'); cabecera.append(fila); elemento.append(cabecera, cuerpo); contenedor.append(elemento);
  return { contenedor, cuerpo };
}
export function boton(documento, texto, clase = 'boton-secundario') {
  const b = nodo(documento, 'button', texto, clase); b.type = 'button'; return b;
}
