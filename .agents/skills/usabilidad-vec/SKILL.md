---
name: usabilidad-vec
description: Hace que las pantallas de VEC sean intuitivas para cualquier trabajador de RRHH sin formación previa (la plantilla rota) y que cumplan la normativa de accesibilidad y lenguaje claro de una administración pública. Usar SIEMPRE al crear, cambiar o revisar una pantalla, formulario, lista, cuadro de mando, mensaje o flujo del portal, y cuando alguien diga que algo «no se entiende», «no se encuentra», «tiene demasiados pasos» o «no es intuitivo». Complementa a aspecto-vec (cómo se ve) y disenar-sistema-visual-vec (estructura y temas): esta skill manda sobre qué se dice, en qué orden y con cuántos pasos.
---

# Usabilidad de VEC

Petición de RRHH (25/09/2026): «que sea una app intuitiva, ya que algunos de los que están en
este Departamento se irán pronto, y que sirviera para gestionarlo cualquier trabajador con
relativa facilidad». La prueba de fuego: **una persona que llega nueva al Departamento hace
su primera petición, localiza un expediente y sabe en qué fase está, sin manual ni ayuda
de un compañero.**

## 1. Obligaciones legales (no negociables)

- **Real Decreto 1112/2018** (accesibilidad de webs y apps del sector público, Directiva UE
  2016/2102): cumplir **UNE-EN 301 549**, que equivale a **WCAG 2.1 nivel AA**. Diseñar ya
  para **WCAG 2.2 AA** (añade: foco nunca tapado, objetivos táctiles ≥ 24×24 px, sin
  arrastrar como única forma, ayuda en el mismo sitio en todas las pantallas, no pedir otra
  vez datos ya dados, autenticación sin pruebas cognitivas).
- Aplica también a **documentos descargables** (PDF/DOCX accesibles: etiquetados, orden de
  lectura, idioma, títulos).
- Idioma: toda pantalla en español e inglés por claves i18n; el atributo `lang` cambia con
  el idioma.
- Lenguaje claro de la Administración (guías de la Comunidad de Madrid, Región de Murcia,
  Cataluña, IVAP y FEMP): el destinatario debe **encontrar, entender y usar** la información
  a la primera.

## 2. Primera pantalla y navegación

1. La portada de RRHH es un **cuadro de mandos**: qué hay pendiente hoy y en qué estado está
   todo (expedientes en trámite, bolsas de trabajo, ofertas al SAE). Cada tarjeta dice un
   número y lleva a la lista filtrada («Ver trámites»).
2. **Lo pendiente primero.** Arriba lo que requiere acción de quien mira (tareas, plazos que
   vencen, incidencias); después la información.
3. **Tres clics como máximo** desde la portada hasta cualquier acción frecuente (nueva
   petición, abrir un expediente, registrar una respuesta).
4. Menú con los **nombres del trabajo real** («Peticiones de personal temporal», «Bolsas de
   trabajo», «Ofertas al SAE»), nunca nombres técnicos, siglas internas ni códigos (B7, CT,
   V3, AD3…).
5. Migas de pan y título de página que digan **dónde estoy**; botón «Volver» que vuelve a
   donde estaba, con los filtros conservados.
6. El mismo elemento en el mismo sitio en todas las pantallas (buscar arriba, acción
   principal abajo a la derecha, «?» siempre en la cabecera).

## 3. Expedientes y estados (sistema complejo, estilo gestión de casos)

1. **La fase y el estado se ven siempre**, con palabra y color (nunca solo color): una línea
   de fases con la actual marcada, lo hecho y lo que falta.
2. En cada expediente, **«Siguiente paso»** visible: qué hay que hacer ahora, quién y hasta
   cuándo. Si no se puede hacer nada, decir por qué y quién tiene que actuar.
3. Documentos del expediente como **lista de comprobación**: cuáles faltan, cuáles están, en
   qué fase de firma (borrador, enviado a firma, firmado, rechazado).
4. Historial legible: «18/09/2026 10:32 — Carmen Molina registró la aceptación», no
   identificadores ni recibos técnicos (esos, detrás de «Ver detalle técnico»).
5. Listas con **búsqueda, filtros por fase/estado/centro/categoría y orden**; los filtros
   activos se ven y se quitan con un clic; el recuento cuadra con la tarjeta del cuadro.
6. Vías distintas se eligen al principio con dos botones grandes y claros (p. ej. «Por
   bolsa de trabajo» / «Por oferta al SAE»), y cada una muestra solo sus datos y documentos.

## 4. Formularios

1. **Una cosa por paso** cuando el trámite es largo o poco frecuente (pasos numerados con
   «Paso 2 de 4»); en pantallas de uso diario de RRHH se permite agrupar, pero en bloques
   con título.
2. Cada campo: etiqueta visible encima (nunca solo *placeholder*), texto de pista corto si
   hace falta, y **obligatorio/opcional** marcado. Pedir solo lo necesario.
3. Rellenar lo que el sistema ya sabe (centro de quien pide, fechas por defecto razonables,
   datos de la bolsa) y no volver a pedirlo (WCAG 2.2 «entrada redundante»).
4. Listas cortas como opciones visibles; largas con buscador. Fechas con formato visible
   (dd/mm/aaaa).
5. **Validar al salir del campo y al enviar**, con resumen de errores arriba enlazado a cada
   campo, y el campo marcado con el mensaje junto a él.
6. Antes de un acto que no se puede deshacer: **pantalla de revisión** con todo lo
   introducido y «Cambiar» en cada bloque; después, **confirmación** con qué ha pasado, su
   número/recibo y qué ocurrirá después.
7. Guardar borrador o avisar al salir si hay cambios sin guardar.

## 5. Mensajes y textos (lenguaje claro)

1. Frases cortas (≤ 20–25 palabras), voz activa, segunda persona («Revise la fecha»), verbo
   al principio en botones («Registrar respuesta», «Enviar a firma»); nada de «Aceptar»
   genérico.
2. Vocabulario del Departamento: nombramiento, petición, bolsa, llamamiento, renuncia, cese,
   Firmadoc, SAE. Sin jerga informática (API, V3, token, replay, 409, puerto, hash).
3. **Mensajes de error**: qué pasó + por qué + qué hacer ahora, en lenguaje llano. Nunca
   códigos HTTP ni técnicos en pantalla. Ej.: «No se ha podido cargar el cuadro. Compruebe
   la conexión y pulse Reintentar. Si sigue igual, avise a Informática.»
4. **Estados vacíos** que expliquen y ofrezcan la acción: «No hay expedientes en trámite.
   Cree una nueva petición.»
5. Ayuda **solo tras el botón «?»**, en el mismo sitio en todas las pantallas, con pasos
   concretos de esa pantalla; nunca párrafos de ayuda en la pantalla.
6. Números, fechas y horas formateados según el idioma; zona horaria de Madrid.

## 6. Accesibilidad práctica (comprobar en cada cambio)

- Todo usable con teclado, en orden lógico, con foco visible y no tapado.
- Contraste ≥ 4,5:1 texto, 3:1 componentes; nada comunicado solo por color.
- Encabezados jerárquicos, regiones (`main`, `nav`), tablas con cabeceras, etiquetas
  asociadas, `aria-live` para cambios de estado y errores.
- Zoom al 200 % y ancho de 320–390 px sin scroll horizontal de la página (las tablas anchas
  con scroll propio o como tarjetas).
- Objetivos táctiles ≥ 24×24 px; nada que dependa de pasar el ratón.

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
   externa y en ≤ 3 clics hasta la acción.
2. Revisar con la lista de §2–§7; cada fallo se corrige en el mismo corte o se anota con
   prioridad (bloquea / molesta / mejora).
3. Comprobar en Chrome a 1440 px y 390 px, con teclado y zoom 200 %, en español e inglés.
4. Guardar capturas antes/después en la PR y describir el cambio en lenguaje llano.
5. Pruebas automáticas cuando se pueda: textos sin clave i18n, etiquetas de campos,
   contraste de tokens, ausencia de códigos técnicos en mensajes visibles.

Fuentes: RD 1112/2018 (BOE-A-2018-12699), UNE-EN 301 549 / WCAG 2.1–2.2 (W3C), Nielsen
Norman Group «10 Usability Heuristics Applied to Complex Applications», GOV.UK Design
System («one thing per page», patrones de páginas de pregunta), guías de lenguaje claro de
la Comunidad de Madrid, Región de Murcia y FEMP.
