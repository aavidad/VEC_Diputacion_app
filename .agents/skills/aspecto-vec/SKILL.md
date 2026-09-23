---
name: aspecto-vec
description: Da a las pantallas web de VEC un aspecto cuidado y legible (superficies, color, separación de apartados, tablas, fichas, indicadores y estados) según la maqueta de RRHH. Usar siempre que se cree, modifique o revise cualquier pantalla, lista, ficha o formulario del portal, y al recibir quejas de que algo se ve «muy blanco», «plano», «sin separar» o «poco currado».
---

# Aspecto de VEC

Autoridad del **aspecto**: cómo se ven superficies, apartados, listas y estados. La
estructura de la pantalla, los temas y la accesibilidad los fija
`disenar-sistema-visual-vec`; esta skill la completa, no la sustituye.

**Referencia visual obligatoria:** `fotos/image004.png`, la maqueta que RRHH hizo sobre
SAVIA/GINPIX. Ante la duda, gana lo que se parezca a esa imagen.

Queja de Alberto que origina esta skill (23/09/2026): «está todo muy blanco y no se acotan
bien los apartados; solo sale una línea muy fina». Todo lo de abajo existe para que eso no
vuelva a pasar.

## Las diez reglas

1. **Lienzo tintado, apartados blancos.** El fondo del área de contenido es
   `--portal-fondo` (gris azulado claro), nunca blanco. Cada apartado es un **panel blanco**
   (`--portal-superficie`) que se recorta sobre ese fondo. Si todo es blanco, está mal.
2. **Un apartado se delimita con caja, no con una raya.** Panel = borde de 1 px
   `--portal-borde` + radio `--portal-radio-lg` (8 px) + sombra `--portal-sombra-sm`.
   Separación entre paneles de 16 a 24 px. Prohibido separar apartados solo con una línea
   fina (`hr` o `border-top`) sobre fondo blanco.
3. **Cada panel tiene cabecera.** Banda superior con fondo levemente tintado
   (`--portal-superficie-alterna`), título de 16–17 px en seminegrita, subtítulo en
   `--portal-muted` y, a la derecha, contadores o acciones del apartado. Borde inferior que
   la separa del cuerpo.
4. **Tablas con ritmo.** Cabecera de columnas tintada, filas de 44–48 px, **cebra suave**
   (una fila de cada dos con `--portal-superficie-alterna`), hover visible y foco por
   teclado. Número o nombre principal en azul enlazable. Números alineados a la derecha.
5. **Cada ficha o expediente se distingue.** Un elemento desplegable (expediente, candidato,
   comisión) lleva **acento de color a la izquierda** de 3–4 px según su estado o fase, y al
   desplegarse su detalle queda **unido a su fila** (mismo acento, fondo tintado, borde) y
   separado de la siguiente. El detalle **no repite** las columnas: muestra lo que aporta.
6. **Indicadores como fichas con icono.** KPI = tarjeta con icono en círculo tintado del
   color semántico, rótulo pequeño en `--portal-muted` y valor grande en seminegrita del
   mismo color (disponibles en verde, excluidos en rojo, pendientes en ámbar…). Nunca un
   número suelto en texto corrido.
7. **Estados como pastillas.** Estado = pastilla con fondo `*-suave`, texto del color fuerte
   y punto de color delante. Verde éxito, ámbar aviso, rojo peligro, azul información,
   violeta en revisión. El color nunca es la única señal: el texto dice el estado.
8. **Recorridos con pasos visibles.** Formularios largos van por pasos numerados: círculo
   relleno azul en el paso actual, marca de hecho en los anteriores, conector entre pasos.
   Resumen lateral con los datos clave de la decisión, como en la maqueta.
9. **Botones con jerarquía.** Una acción principal por bloque: azul sólido, con flecha si
   avanza. Secundarias en contorno. Destructivas en rojo con confirmación. Acciones sin caso
   de uso: deshabilitadas y con el motivo.
10. **Ritmo y tipografía.** Escala de espaciado 4 · 8 · 12 · 16 · 24 px. Título de página
    22–24 px, título de apartado 16–17 px, texto 14 px, rótulos 12–13 px en `--portal-muted`.
    Nada de bloques de texto centrados ni de mayúsculas largas.

## Tokens

Usar siempre los tokens `--portal-*` de `web/static/portal-empleado/portal.css`. Si falta
uno para cumplir estas reglas (por ejemplo `--portal-superficie-alterna`,
`--portal-cabecera-panel` o un acento por fase), **se añade una sola vez en `portal.css`**
con su variante de alto contraste, y los módulos lo consumen. Ningún módulo escribe
colores en hexadecimal ni duplica CSS estructural.

## Antipatrones que hay que corregir al verlos

- Página entera blanca, sin recortar paneles.
- Apartados separados solo por una línea de 1 px.
- Listas donde todas las filas son idénticas y no se sabe dónde empieza cada elemento.
- Detalle desplegado que repite lo que ya dice la fila.
- KPI escritos como «390Disponibles» o sin color semántico.
- Formularios con etiqueta y campo sin estilo del tema (selects nativos sin enmarcar).
- Columnas enteras con «—» porque el dato no se lee: si existe, se muestra; si no, se
  dice por qué.

## Cómo comprobarlo antes de cerrar

1. Captura real a **1440×900** (y 390 px para móvil) de cada pantalla tocada, con datos.
2. Ponerla junto a `fotos/image004.png` y repasar las diez reglas una a una.
3. Barrido sin scroll de página en escritorio (R10), sin errores JS y con contraste AA.
4. En el cierre, citar qué reglas se aplicaron y adjuntar la ruta de las capturas.
