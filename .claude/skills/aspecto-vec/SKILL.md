---
name: aspecto-vec
description: Hace que las pantallas web de VEC tengan aspecto profesional y pulido, como las maquetas de RRHH del cuadro de mando y del nuevo llamamiento. Usar siempre que se cree, modifique o revise cualquier pantalla, menú, lista, ficha, formulario o cuadro del portal, y ante quejas de que algo se ve «muy blanco», «plano», «anticuado», «poco pulido» o «poco profesional».
---

# Aspecto de VEC

Autoridad única del aspecto del portal. `disenar-sistema-visual-vec` fija la estructura de
la pantalla, los temas y la accesibilidad; esta skill fija cómo se ve. Se aplican juntas.

## Referencias

El objetivo son las dos maquetas que RRHH hizo sobre SAVIA/GINPIX: **cuadro de mando** y
**nuevo llamamiento**. No están en Git: se guardan en local, fuera del repositorio
(`fotos/image006.png` y `fotos/image004.png` del checkout del operador). Si no las tienes,
las reglas de abajo las describen con detalle suficiente. Ante la duda, gana lo que se
parezca a ellas.

Lo que se sustituye (portal interno amarillo, WCronos, Dietas, el GINPIX clásico) **no se
copia**. Solo se conserva lo que RRHH ya sabe usar: menú lateral, calendario anual con las
ausencias por colores y tablas densas. Se toma la composición y el acabado; nunca
logotipos, marcas ni textos de terceros.

## Qué hace pulida una pantalla

Una pantalla parece profesional cuando **todo tiene un motivo**: pocos colores, pocos
tamaños, alineación exacta, aire regular y ningún texto que no ayude a hacer la tarea.
Parece amateur con rótulos internos, avisos explicativos, pastillas por todas partes,
bordes dobles, mezcla de tamaños y cosas que no se alinean.

## Contenido (lo primero que se nota)

1. **Ningún código interno en pantalla.** Nada de «Cuadro B12», «Vista B5», «R7», «CT58»,
   «dudas 13–14», «fase inicial», «AD3», migraciones ni capacidades. Se ve «Bolsas de
   trabajo activas», no «Bolsas de trabajo activas (Cuadro B12)».
2. **Ningún texto de ayuda ni explicativo.** La ayuda vive tras el botón «?» de la
   cabecera. Fuera: párrafos bajo los títulos, franjas «Operaciones conectadas…», notas
   «Provisional…», leyendas de funcionamiento. Se queda solo lo que el usuario necesita para
   decidir: validaciones, errores, resultado y límites reales.
3. **Nada de demo.** Ningún «recorrido sintético», «datos de ejemplo» o «presentación». Los
   datos son sintéticos; la pantalla es de producción.
4. **Rótulos cortos y de negocio.** Sin mayúsculas largas ni textos centrados.

## Estructura (cuadro de mando)

5. **Menú lateral azul marino** (`--portal-azul-950`), ancho `--portal-lateral`.
   - Arriba, la marca: icono blanco en cuadrado azul redondeado, nombre del área en dos
     líneas de 18 px seminegrita y «Diputación de Granada» debajo en 13 px.
   - Cada entrada: número en círculo de color de 32 px (una tinta por módulo), texto blanco
     de 15 px en dos líneas como máximo, separador tenue entre entradas y 14 px de relleno.
   - La activa se rellena con `--portal-azul-700` a todo el ancho.
   - Abajo: «Ayuda ›» y «Cerrar sesión».
   - Sin pastillas «ACTIVO» o «DESDE CADA BOLSA» ni avisos dentro del menú.
6. **Cabecera blanca de 72 px.**
   - A la izquierda, el título de página en 26 px, con la miga pequeña encima si hay
     jerarquía («Inicio → Llamamientos → Nuevo llamamiento»).
   - A la derecha: campana con contador rojo, «?» de ayuda y el bloque de usuario (avatar
     circular con iniciales, nombre y perfil en dos líneas, flecha de menú).
   - A+ y contraste van dentro del menú de usuario, no como botones sueltos.
7. **Lienzo gris azulado** (`--portal-fondo`) con **tarjetas blancas**: radio 12 px, borde
   de 1 px `--portal-borde` y sombra muy suave, separadas 20–24 px. Nunca apartados
   separados solo por una raya.

## Componentes

8. **Indicadores (KPI).**
   - Fila de 4–5 tarjetas iguales.
   - Cada una: icono de 56 px en cuadrado redondeado con fondo suave de su color (azul
     bolsas, verde disponibles, naranja pendientes, violeta contratos, turquesa cobertura),
     valor de 28 px, rótulo de 14 px en `--portal-muted` y «Ver detalle →» si lleva a una
     lista.
   - Sin dato: «—», con el motivo en el título accesible.
9. **Tarjeta de apartado.** Título de 18 px a la izquierda; a la derecha un selector
   («Todas las bolsas ⌄») o una acción. Sin cabecera tintada ni doble borde. Pie opcional
   con «Ver todas las bolsas →» centrado.
10. **Tablas.**
    - Cabecera de 13 px en `--portal-muted` sin fondo fuerte.
    - Filas de 52–56 px separadas por una línea muy suave.
    - Nombre principal en azul enlazable y números alineados a la derecha.
    - Acciones como iconos (ojo, tres puntos) o un botón de contorno «Seleccionar ›».
    - Paginación abajo: «Mostrando 1 a 6 de 30» a la izquierda y las páginas a la derecha.
    - Casillas de selección a la izquierda cuando se puede actuar en lote.
11. **Estados.** Punto de color y texto («● Disponible») o pastilla suave pequeña. **Como
    mucho una pastilla por fila.** Los números nunca van en pastilla.
12. **Cobertura o progreso.** Barra fina verde con el porcentaje a la derecha.
13. **Recorridos por pasos** (nuevo llamamiento).
    - Pasos numerados en línea, unidos por un conector; el actual en círculo azul relleno y
      los pendientes en gris.
    - Contenido por bloques numerados («1. Seleccionar bolsa»).
    - Columna derecha con «Resumen», los filtros y la acción principal al pie: ancha, azul,
      «Siguiente: … →». «Cancelar» en contorno debajo.
    - Contadores por estado como tarjetas pequeñas con icono y color.
14. **Botones.** Principal azul sólido de 40 px con radio 8 px; secundario de contorno;
    peligro en rojo y con confirmación. Una principal por bloque. Sin degradados.
15. **Agenda y actividad.**
    - Agenda: fecha en un cuadro tintado (día grande, mes pequeño), dos líneas de texto y el
      estado a la derecha.
    - Actividad: icono circular de color, enlace y descripción, y «Por autor · fecha» en gris.
16. **Accesos rápidos.** Tarjetas bajas con icono y texto, en una fila al pie del cuadro.

## Ritmo

17. **Tipografía.** Una familia, la del tema. Tamaños: 26 título de página, 18 título de
    tarjeta, 15 menú, 14 texto y 13 rótulos. Pesos 400 y 600, nada más.
18. **Espaciado.** Escala 4 · 8 · 12 · 16 · 20 · 24 · 32; relleno de tarjeta de 20–24 px. Los
    bordes izquierdos de título, KPI y tarjetas coinciden.
19. **Color.**
    - Marino, un azul de acción y los semánticos (verde, ámbar, rojo, violeta), cada uno
      con su variante fuerte y suave.
    - Los módulos no escriben hexadecimales: usan solo los tokens `--portal-*` de
      `web/static/comun/tema-vec.css`.
    - Si falta un token, se añade allí una sola vez, con su variante de alto contraste.

## Cómo se trabaja

1. **Antes de tocar.**
   - Captura la pantalla con datos reales en 1440 y 390 px con
     `scripts/capturar_pantallas.py`. El destino y la credencial van por entorno, nunca en Git.
   - Ponla junto a la maqueta y anota qué reglas incumple.
2. **Primero lo común.** Tokens, menú, cabecera, tarjeta, tabla, KPI y estados, en
   `tema-vec.css` y `portal-componentes.css`; los módulos lo heredan. Después, lo propio de
   cada pantalla, sin duplicar CSS estructural.
3. **Texto.** Todo rótulo pasa por i18n. Al quitar un texto explicativo, quita su clave si
   nadie más la usa.
4. **Después.** Vuelve a capturar con `--local web/static` y compara con la maqueta y con el
   «antes». Deben quedar:
   - cero errores JS y ningún `/api` ≥ 400 nuevo;
   - cero desborde horizontal;
   - foco visible, contraste AA y alto contraste sin roturas.
5. **Versiones.** Renueva el `?v=` de cada CSS o JS tocado, en su HTML y en los manifiestos.
6. **Entrega.** Lista las reglas aplicadas y la carpeta de capturas de antes y después.
