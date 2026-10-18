---
name: aspecto-vec
description: Hace que las pantallas web de VEC tengan aspecto profesional y pulido, como las maquetas de RRHH del cuadro de mando y del nuevo llamamiento. Usar siempre que se cree, modifique o revise cualquier pantalla, menú, lista, ficha, formulario o cuadro del portal, y ante quejas de que algo se ve «muy blanco», «plano», «anticuado», «poco pulido» o «poco profesional».
---

# Aspecto de VEC

Procedimiento para que una pantalla se vea como las maquetas de RRHH. Se aplica junto con
`disenar-sistema-visual-vec` (armazón y temas) y `usabilidad-vec` (qué se dice y en qué
orden). Las reglas de aspecto están en `docs/producto/DISENO_UI_UX.md`; si algo de aquí
choca con el documento, manda el documento.

## Referencias

El objetivo son las dos maquetas que RRHH hizo sobre SAVIA/GINPIX: **cuadro de mando** y
**nuevo llamamiento**. No están en Git: se guardan en local, fuera del repositorio
(`fotos/image006.png` y `fotos/image004.png` del checkout del operador). Si no las tienes,
el documento las describe con detalle suficiente. Ante la duda, gana lo que se parezca a
ellas.

Lo que se sustituye (portal interno amarillo, WCronos, Dietas, el GINPIX clásico) **no se
copia**. Solo se conserva lo que RRHH ya sabe usar: menú lateral, calendario anual con las
ausencias por colores y tablas densas. Se toma la composición y el acabado; nunca
logotipos, marcas ni textos de terceros.

## Qué hace pulida una pantalla

Una pantalla parece profesional cuando todo tiene un motivo: pocos colores, pocos tamaños,
alineación exacta, aire regular y ningún texto que no ayude a hacer la tarea. Parece
amateur con rótulos internos, avisos explicativos, pastillas por todas partes, bordes
dobles, mezcla de tamaños y cosas que no se alinean.

## Dónde está cada regla

| Qué | `docs/producto/DISENO_UI_UX.md` |
| --- | --- |
| Contenido: sin códigos internos, sin ayuda en pantalla, sin marcas de demo, rótulos cortos | §2 principios 4, 6, 7 y 11; §14 Textos |
| Menú lateral, cabecera de 72 px, lienzo y tarjetas | §3 Estructura de una pantalla |
| Tipografía (26, 18, 15, 14, 13 px; pesos 400 y 600) | §4.1 |
| Color y tokens `--portal-*`, colores de los indicadores, una pastilla por fila | §4.2 |
| Espaciado, radios y sombras | §4.3 |
| Indicadores, tarjeta de apartado, tablas, estados, cobertura, pasos, botones, agenda, actividad y accesos rápidos | §5 Componentes que ya existen |
| Asistente con columna de resumen y «Siguiente: … →» | §12.3 |

Detalles de las maquetas que el documento no recoge y se siguen aplicando:

- en el nuevo llamamiento, los contadores por estado van como tarjetas pequeñas con icono
  y color;
- el pie opcional de una tarjeta de apartado lleva el enlace centrado («Ver todas las
  bolsas →»).

## Cómo se trabaja

1. **Antes de tocar.**
   - Captura la pantalla con datos reales en 1440 y 390 px con
     `scripts/capturar_pantallas.py`. El destino y la credencial van por entorno, nunca en
     Git.
   - Ponla junto a la maqueta y anota qué reglas de la tabla anterior incumple.
2. **Primero lo común.** Tokens, menú, cabecera, tarjeta, tabla, KPI y estados, en
   `web/static/comun/tema-vec.css` y `portal-componentes.css` / `portal-patrones.css`; los
   módulos lo heredan. Después, lo propio de cada pantalla, sin duplicar CSS estructural.
   Si falta un token, se añade en `tema-vec.css` una vez (§4.2).
3. **Texto.** Todo rótulo pasa por i18n. Al quitar un texto explicativo, quita su clave si
   nadie más la usa.
4. **Después.** Vuelve a capturar con `--local web/static` y compara con la maqueta y con
   el «antes». Deben quedar:
   - cero errores JS y ningún `/api` ≥ 400 nuevo;
   - cero desborde horizontal;
   - foco visible, contraste AA y alto contraste sin roturas.
5. **Versiones.** Renueva el `?v=` de cada CSS o JS tocado, en su HTML y en los manifiestos.
6. **Entrega.** Repasa el bloque «Aspecto» de `docs/producto/DISENO_UI_UX.md` §16, lista
   las reglas aplicadas y la carpeta de capturas de antes y después.
