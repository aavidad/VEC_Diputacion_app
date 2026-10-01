# Colores de los temas de VEC

## Fuente del tema corporativo

El [Manual de identidad corporativa de la Diputación de Granada, versión 1.0, página 28](https://www.dipgra.es/export/sites/diputaciongranada/comunicacion/comunicacion-y-prensa/.galleries/COMUNICACION-Documentos-Imagen-corporativa/Mini-Manual-identidad-corporativa-Diputacion-de-Granada.pdf) fija dos colores del logotipo: azul `#173A4E` (Pantone 7546C) y verde `#ACCB49` (Pantone 2299C). La [página institucional de imagen corporativa](https://www.dipgra.es/comunicacion/comunicacion-y-prensa/imagen-corporativa/) publica el manual.

El tema `diputacion_granada` usa el azul exacto en la navegación lateral y el verde en el fondo de interacción de esa navegación. Los tonos de texto y acciones son derivados más oscuros para mantener el contraste. No añade ni modifica un logotipo.

## Otras opciones

`arena`, `salvia`, `lavanda`, `azul_sereno` y `noche_suave` son paletas de trabajo de VEC. Son opciones de comodidad visual, no colores oficiales de la Diputación. Las seis variantes cambian los tokens cromáticos comunes; conservan estructura, estados y comportamiento. El alto contraste sigue siendo una capa independiente.

La prueba `web/static/comun/codexf-temas-contraste.test.mjs` calcula la relación de contraste WCAG en cada paleta. Comprueba texto normal (mínimo 4,5:1), bordes de controles y foco (mínimo 3:1), navegación, botones y estados. La prueba no sustituye un recorrido de la interfaz con Chrome, teclado y zoom.
