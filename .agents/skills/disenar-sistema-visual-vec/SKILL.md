---
name: disenar-sistema-visual-vec
description: Diseña o unifica pantallas web de VEC con el sistema visual corporativo común, permitiendo variar el tema cromático sin cambiar la estructura, semántica ni comportamiento de los componentes.
---

# Diseñar el sistema visual VEC

Procedimiento para componer pantallas con una sola gramática visual y comparar temas. El
aspecto concreto lo trabaja `aspecto-vec`; aplicar las dos juntas. Las reglas están en
`docs/producto/DISENO_UI_UX.md`; si algo de aquí choca con el documento, manda el
documento.

## Dónde está cada regla

| Qué | Dónde |
| --- | --- |
| Orden común de una pantalla (armazón, título, pestañas, indicadores, área de trabajo, resumen, acciones) y valor, rótulo y estado separados | `docs/producto/DISENO_UI_UX.md` §3 |
| Componentes comunes y sus ficheros; no crear uno paralelo | `docs/producto/DISENO_UI_UX.md` §5 |
| Ayuda solo tras «?»; no afirmar firma, envío o guardado sin respuesta real | `docs/producto/DISENO_UI_UX.md` §2, principios 6 y 12 |
| Estados cargando, vacío, error, sin permiso, solo lectura; botón deshabilitado con el motivo en una frase | `docs/producto/DISENO_UI_UX.md` §6 (y §17, contradicción resuelta) |
| Móvil, tablas con scroll propio | `docs/producto/DISENO_UI_UX.md` §7 |
| Foco, etiquetas, contraste AA, orden semántico | `docs/producto/DISENO_UI_UX.md` §8 |
| Temas: qué cambia y qué no, alto contraste como capa aparte | `docs/producto/DISENO_UI_UX.md` §4.4 |
| Ninguna regla en HTTP, DOM ni navegador | `docs/producto/TRD.md` §2, regla 4 |

Regla que el documento no recoge y se mantiene aquí: la vista recibe modelos de
presentación, los clientes HTTP implementan puertos del frontend y el montaje inyecta
ambos. Un cambio de tema no altera DTO, caso de uso, autorización, endpoint, recibo ni
persistencia.

## Cómo componer una pantalla

1. Buscar la pantalla o una parecida (índice de código y ramas abiertas).
2. Montarla en el orden de §3, reutilizando primero las clases de
   `web/static/portal-empleado/portal-componentes.css`, `portal-patrones.css` y
   `portal-flujos.css`, y los tokens de `web/static/comun/tema-vec.css`.
3. Si falta algo, añadirlo en el fichero común, no en el módulo.
4. Pintar cada estado de §6 con datos reales o forzando el fallo.

## Temas

Comparar temas sobre el mismo mini-recorrido: navegación, KPI, panel, formulario, botón,
estado, tabla y mapa. Cambiar únicamente tokens de color y, si se justifica, radio y
sombra. No variar contenido ni distribución para favorecer una alternativa. Revisar al
menos Portal, Dietas, Cronos, Personal y Bolsa en escritorio y móvil.

## Comprobación

- Contraste de texto y controles en tema claro, oscuro y alto contraste.
- 1440, 1024 y 390 px y zoom 200 %, sin desbordamiento horizontal de la página.
- Foco por teclado en toda la pantalla.
- Estados carga, vacío y error vistos de verdad.
- La variante no añade cookies, almacenamiento web, fuentes ni llamadas externas.
- Bloques «Aspecto» y «Adaptación y accesibilidad» de `docs/producto/DISENO_UI_UX.md` §16.
