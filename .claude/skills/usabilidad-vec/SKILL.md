---
name: usabilidad-vec
description: Hace que las pantallas de VEC sean intuitivas para cualquier trabajador de RRHH sin formación previa (la plantilla rota) y que cumplan la normativa de accesibilidad y lenguaje claro de una administración pública. Usar SIEMPRE al crear, cambiar o revisar una pantalla, formulario, lista, cuadro de mando, mensaje o flujo del portal, y cuando alguien diga que algo «no se entiende», «no se encuentra», «tiene demasiados pasos» o «no es intuitivo». Complementa a aspecto-vec (cómo se ve) y disenar-sistema-visual-vec (estructura y temas): esta skill manda sobre qué se dice, en qué orden y con cuántos pasos.
---

# Usabilidad de VEC

Prueba de fuego (petición de RRHH del 25/09/2026): **una persona recién llegada al
Departamento hace su primera petición, localiza un expediente y sabe en qué fase está, sin
manual ni ayuda de un compañero.**

Las reglas están en `docs/producto/DISENO_UI_UX.md`. Esta skill dice cómo aplicarlas y qué
mirar. Si algo de aquí choca con el documento, manda el documento.

## 1. Obligaciones legales

Accesibilidad (RD 1112/2018, UNE-EN 301 549, WCAG 2.1 AA y diseño ya para 2.2 AA), también
en PDF y Word: `docs/producto/DISENO_UI_UX.md` §8 Accesibilidad. Idiomas y atributo `lang`:
§9 Idiomas. Lenguaje claro de la Administración: §14 Textos.

## 2. Primera pantalla y navegación

Reglas en `docs/producto/DISENO_UI_UX.md` §2 Principios (1 a 4: novato, lo pendiente
primero, tres clics, un solo vocabulario), §3 Estructura de una pantalla (migas, «Volver»
con filtros, cada elemento siempre en el mismo sitio) y §11.1 Portada de RRHH.

Qué mirar: ¿la portada dice qué hay pendiente hoy antes que la información? ¿Cada tarjeta
lleva a su lista filtrada con el mismo número? ¿Hay algún nombre técnico o sigla en el
menú? ¿Se llega a la acción frecuente en tres clics o menos?

## 3. Expedientes y estados

Reglas en `docs/producto/DISENO_UI_UX.md` §2 (principio 3: qué toca, fase, qué falta),
§11.2 Lista de expedientes, §11.3 Ficha de expediente, §12.1 Bandeja o lista y §12.2 Ficha.
Componentes ya hechos (siguiente paso, línea de fases, documentos, historial, filtros
quitables, opciones grandes): §5.

Qué mirar: fase y estado con palabra y no solo color; «Siguiente paso» con qué, quién y
hasta cuándo; historial con personas y fechas, sin identificadores; filtros activos
visibles y quitables; recuentos iguales a los del cuadro.

## 4. Formularios

Reglas en `docs/producto/DISENO_UI_UX.md` §12.3 Asistente o formulario (pasos, etiquetas,
obligatorio u opcional, no volver a pedir datos, validación, revisión, confirmación,
borrador), §12.6 Recibo y confirmación y los ejemplos de §11.4 y §11.6.

Qué mirar: rellenar con errores a propósito y comprobar el resumen arriba enlazado y el
mensaje junto al campo; salir con cambios sin guardar; repetir el envío y ver que sale el
mismo recibo.

## 5. Mensajes y textos (lenguaje claro)

Reglas en `docs/producto/DISENO_UI_UX.md` §14 Textos (frases cortas, voz activa, usted,
botones con verbo, vocabulario del Departamento, sin jerga), §2 principios 6, 7 y 9 (ayuda
solo tras «?», sin carteles sin sentido, errores que dicen qué hacer), §6 Estados de una
pantalla (vacío, error, sin permiso, con ejemplos de texto) y §9 Idiomas (fechas, números,
hora de Madrid).

Cómo se aplica: leer cada texto nuevo en voz alta como lo leería alguien de RRHH sin
contexto; buscar códigos HTTP, siglas y palabras técnicas; repasar con `humanizer`
(`.agents/skills/humanizer/VEC-USO.md`).

## 6. Accesibilidad práctica (comprobar en cada cambio)

Reglas en `docs/producto/DISENO_UI_UX.md` §7 Adaptación a la pantalla y §8 Accesibilidad.
Cómo comprobarlas:

- recorrer la pantalla solo con Tab, Mayús+Tab, Intro y Espacio: orden lógico y foco
  visible, nunca tapado por la cabecera;
- zoom al 200 % y anchos de 1440, 1024 y 390 px: sin scroll horizontal de la página;
- tema claro, oscuro y alto contraste: nada se pierde ni se comunica solo por color;
- inspeccionar encabezados, regiones, `caption`/`th scope`, etiquetas y `aria-live`;
- tocar en móvil los controles pequeños (24 × 24 px como mínimo).

## 7. Heurísticos de revisión (Nielsen, aplicados a gestión)

Antes de dar por buena una pantalla, recorrerla con estos diez: estado visible del sistema;
lenguaje del trabajo real; control y deshacer (cancelar, volver, corregir antes de
confirmar); coherencia; prevención de errores (validar antes, confirmar lo irreversible);
reconocer mejor que recordar (mostrar nombres, no códigos); eficiencia para quien repite
(atajos, acciones en lote, filtros guardados); diseño mínimo (lo raro, plegado); errores
que se entienden y se arreglan; ayuda contextual.

## 8. Cómo aplicarla

1. **Recorrido del novato**: para cada tarea clave (crear petición, encontrar un expediente
   y su fase, registrar respuesta de un candidato, ver una bolsa y sus integrantes, enviar a
   firma), anotar pasos, clics, dudas y textos que no se entienden. Objetivo: sin ayuda
   externa y en tres clics o menos hasta la acción.
2. Revisar con la lista de comprobación de `docs/producto/DISENO_UI_UX.md` §16; cada fallo
   se corrige en el mismo corte o se anota como «bloquea», «molesta» o «mejora».
3. Comprobar en Chrome (Playwright con el Chrome del sistema) a 1440 y 390 px, con teclado
   y zoom 200 %, en castellano e inglés.
4. Guardar capturas antes y después en la PR y describir el cambio en lenguaje llano.
5. Pruebas automáticas cuando se pueda: textos sin clave i18n, etiquetas de campos,
   contraste de tokens, ausencia de códigos técnicos en mensajes visibles.
6. Revisión independiente con el agente `revisor-usabilidad-vec` antes de fusionar
   (`docs/producto/DISENO_UI_UX.md` §15).

Fuentes de las reglas: RD 1112/2018 (BOE-A-2018-12699), UNE-EN 301 549 / WCAG 2.1–2.2,
Nielsen Norman Group «10 Usability Heuristics Applied to Complex Applications», GOV.UK
Design System («one thing per page») y guías de lenguaje claro de la Comunidad de Madrid,
Región de Murcia y FEMP.
