# Dirección de diseño de las pantallas de RRHH — 28/09/2026

Por qué: RRHH dice que «la web no es intuitiva». Prueba de fuego (skill `usabilidad-vec`): una persona
nueva hace su primera petición, encuentra un expediente y sabe en qué fase está, sin ayuda.

Cómo se ha mirado: el portal real de `origin/main` (`4a532005`) montado en Chrome sin ventana con
datos sintéticos (API simulada que respeta los contratos del propio código) a 1440 y 390 px.
Capturas en `capturas_actuales/` (`*_1440.png` = lo que se ve al abrir; `*_completa.png` = página
entera). Criterio: `usabilidad-vec` > `aspecto-vec`/`disenar-sistema-visual-vec` > Impeccable
(`critique`, `layout`, `clarify`, `polish`). Complementa la auditoría del 28/09 (42 hallazgos); aquí
se mira la estructura de las pantallas, no cada texto.

> **Fallo urgente aparte del diseño.** En `origin/main` el portal no arranca en el navegador:
> `index.html` usa la clave `selector_idioma_etiqueta` (y `selector_idioma_es/en`), que no existe en
> ningún catálogo, y la traducción lanza «clave i18n del portal desconocida». La pantalla queda en
> blanco. Para capturar se quitaron esas tres marcas al servir la página. Hay que corregirlo ya.

## Nota de conjunto (heurísticos de Nielsen, 0–4)

Estado visible 2 · Lenguaje del trabajo 1 · Control y vuelta atrás 2 · Coherencia 1 · Prevención
de errores 2 · Reconocer mejor que recordar 1 · Eficiencia 2 · Diseño mínimo 1 · Errores
comprensibles 2 · Ayuda 2 → **16/40**. Carga mental alta: fallan 6 de 8 comprobaciones (foco único,
jerarquía, una cosa cada vez, pocas opciones, memoria de trabajo, mostrar poco a poco).

Lo que funciona y se conserva: la marca y el menú azul marino; las tarjetas blancas sobre gris; las
tablas con cabeceras bien marcadas; el formulario de nueva petición por bloques con revisión; la
línea de 8 fases de la ficha (buena idea, mal colocada); fechas en hora de Madrid.

## Los 10 problemas más graves

1. **La portada no es un cuadro de mandos.** Solo habla de peticiones: ni bolsas ni ofertas al SAE,
   y nada de «qué tengo que hacer hoy». Título repetido («Portal del Empleado» / «Inicio del
   portal») y «Ayuda» como si fuera una pestaña. → `capturas_actuales/01_portada_1440_completa.png`
2. **Los números no cuadran entre pantallas.** Portada: «En tramitación 5» y «1–6 de 8». Lista:
   «12 expedientes», «En curso 5», «Pendientes 3». Mismo concepto con nombres distintos (en
   tramitación / en curso / pendiente). Quien mira no sabe cuál creer.
   → `01_portada_1440_completa.png` y `02_lista_expedientes_1440_completa.png`
3. **Dos vocabularios de fases.** Lista y portada usan 11 nombres internos («Subsanación por la
   unidad», «Informe jurídico», «Asignación de unidad», «Llamamiento»); la ficha usa las 8 fases de
   RRHH. La misma ficha dice «Fase actual: Llamamiento» y en la línea «Obtención del candidato».
   → `02_lista_expedientes_1440_completa.png` y `03_ficha_expediente_1440_completa.png`
4. **La ficha no dice qué hay que hacer.** Arriba, 17 datos del mismo tamaño en cuatro columnas; no
   hay «siguiente paso», ni quién lo hace, ni plazo (el plazo solo aparece en la lista). Referencias
   internas a la vista: «unidad:rrhh-seleccion». → `03_ficha_expediente_1440.png`
5. **Registrar la respuesta de un candidato es un formulario técnico.** Pide «clave de operación»,
   «versión», «huella SHA-256», «referencia del correo», fecha en UTC y un `.eml` obligatorio; está
   partido en «1. Iniciar», «2. Comunicación», «Plazo de respuesta», «3. Respuesta», con «Pendiente
   de registrar» al lado de «Recibo verificado». Nadie nuevo lo completa sin ayuda.
   → `06_respuesta_candidato_1440.png` y `06_respuesta_candidato_1440_completa.png`
6. **Historial y documentos, escondidos o ausentes.** El historial está plegado y sin quién lo hizo;
   «Cambios de datos», también plegado; en la ficha no hay lista de documentos (faltan / están / en
   firma). → `03_ficha_expediente_1440_completa.png`
7. **La lista de expedientes no ayuda a priorizar.** Ordenada por número (el terminado sale el
   primero), la columna Plazo se corta a 1440 px, el buscador solo admite «el inicio del número»,
   los filtros exigen pulsar «Aplicar», y lo útil («Trabajo pendiente») está al final, debajo de la
   tabla, con 11 enlaces de «Distribución por fase». → `02_lista_expedientes_1440_completa.png`
8. **La navegación cambia según dónde estés.** Dentro de Peticiones desaparece «Bolsas» del menú y
   dentro de Bolsas desaparece «Peticiones»; hay menú lateral + pestañas del módulo + botones
   «Centros / Peticiones / Calendarios / Reglas» que abren otra pestaña del navegador; «Cuadro de
   mando» existe en tres sitios y el «?» está al pie del menú.
   → `02_lista_expedientes_1440_completa.png` y `05_bolsa_integrantes_1440_completa.png`
9. **Ruido del sistema que empuja lo importante hacia abajo.** Franjas «Expediente cargado.»,
   «Cuadro … actualizado.», una franja vacía con un punto en Nueva petición, «Servicio preparado
   para revisar y registrar la solicitud», «El servidor vuelve a comprobar su vigencia». Cabecera
   de dos líneas con «Idioma de la interfaz» partido. En móvil, 4 tarjetas y 3 filtros llenan la
   primera pantalla antes de ver un solo expediente.
   → `04_nueva_peticion_1440_completa.png` y `02_lista_expedientes_390.png`
10. **La bolsa se lee mal.** Tabla cortada a la derecha («Últi… llam»), abreviaturas («No disp.»,
    «Pend. incorp.», «Desde fecha»), «Pendiente incorporacion» sin tilde, hora «2:00» sin sentido
    en una fecha, siete contadores del mismo peso y «Registrar resultado» apagado sin decir por qué.
    → `05_bolsa_integrantes_1440_completa.png`

Además (comunes): los textos de formularios y listas del tema común son pequeños (etiquetas de
11 px en `.campo`, 12 px en `.lista-comprobacion` y `.fila-resumen`), por debajo de los 13–14 px que
fija `aspecto-vec`.

## Principios de la nueva dirección

1. **Lo pendiente primero.** Cada pantalla empieza por lo que requiere acción de quien mira.
2. **Cada ficha responde a tres preguntas, en este orden:** ¿qué toca ahora (qué, quién, hasta
   cuándo)? ¿en qué fase está? ¿qué falta (documentos)?
3. **Un solo vocabulario.** Las 8 fases de RRHH en todas partes, siempre como «Fase 5 de 8:
   Obtención del candidato». Un solo nombre por estado y los mismos números en portada y lista.
4. **Una sola navegación, estable.** Menú lateral fijo con los nombres del trabajo: Inicio ·
   Peticiones de personal temporal · Bolsas de trabajo · Ofertas al SAE. Migas y «Volver» con los
   filtros conservados. Nada se abre en otra pestaña. «?» siempre en la cabecera.
5. **Una acción principal por pantalla**, azul, arriba a la derecha (o al pie del paso). El resto,
   de contorno o como enlace.
6. **El sistema calla cuando todo va bien.** Sin franjas de «cargado» o «actualizado»; solo errores,
   resultados y límites reales. Claves, versiones, huellas y referencias las genera y guarda el
   sistema; si alguien las necesita, tras «Ver detalle técnico».
7. **Una cosa por paso en lo poco frecuente** (respuesta de candidato, alta rara), con «Paso 2 de
   3», revisión con «Cambiar» y confirmación que dice qué pasa después.
8. **Móvil de verdad:** las tablas pasan a fichas apiladas, los filtros secundarios se pliegan y la
   acción principal queda visible sin buscarla.
9. **Solo el tema común.** Tokens `--portal-*` y clases existentes; los patrones nuevos (siguiente
   paso, línea de fases, documentos, historial, filtros quitables, pasos) se añaden una vez al tema
   común, no en cada módulo. Propuesta concreta: `prototipos/css/prototipo.css`.

## Estructura propuesta de las seis pantallas

**1. Portada (cuadro de mandos)** — prototipo `prototipos/index.html`
- Arriba: saludo y fecha; a la derecha, la acción principal «Nueva petición de personal».
- Siempre visible: «Tiene N tareas pendientes», ordenadas por plazo (vencido en rojo, hoy en ámbar),
  cada una con verbo + expediente + plazo y su botón («Registrar respuesta», «Subsanar»).
- Después: 4 indicadores que llevan a la lista filtrada (peticiones en trámite, plazos de esta
  semana, personas disponibles en bolsas, ofertas al SAE abiertas).
- Debajo, en dos columnas: peticiones por fase (8 fases, mismo recuento que la lista) y bolsas +
  ofertas al SAE.
- Se quita: la rejilla de «módulos», el título duplicado y el botón «Ayuda» (va al «?»).

**2. Lista de expedientes** — prototipo `prototipos/expedientes.html`
- Arriba: «11 peticiones en trámite» (mismo número que la portada) y «Nueva petición de personal».
- Siempre visible: buscador único (número, centro o persona) que filtra al escribir; filtros de
  fase, centro y «Mostrar» (en trámite / me tocan a mí / esperan a otros / terminadas); etiquetas
  de filtros activos con «×» y «Quitar todos».
- Tabla de 5 columnas: Expediente · Centro y categoría · Fase («Fase 4 de 8») · Siguiente paso
  («Intervención: emitir informe») · Plazo (una sola pastilla, solo si vence o venció).
- Orden por defecto: plazo más próximo. En móvil, fichas apiladas y filtros tras «Más filtros».
- Se quita: «Distribución por fase» y «Trabajo pendiente» al pie (ya están en la portada), la
  columna Modalidad y los botones que abren otras pestañas (pasan al menú o a la ayuda).

**3. Ficha de expediente** — prototipo `prototipos/expediente.html`
- Arriba: número (título de la página), categoría y centro, «Fase 5 de 8: …» y un único estado.
- Siempre visible, justo debajo: bloque **Siguiente paso** con qué, quién («Sección de Selección
  (usted)»), antes de cuándo y el botón de la acción. Si no puede hacer nada: «Esperando a
  Intervención desde el 22/09».
- Luego: línea de 8 fases con palabra y símbolo (✓ Hecho / Ahora / Falta), nunca solo color.
- Dos columnas: **Documentos** como lista de comprobación (está / falta / en firma, con «Descargar»)
  e **Historial** legible («24/09/2026 11:02 — Carmen Molina Ortega envió el llamamiento…»); a la
  derecha, 8 datos clave de la petición y «Ver todos los datos».
- Se pliega: «Ver detalle técnico» (referencias, versión, flujo). Los formularios de cada fase no
  van en la ficha: se abren desde «Siguiente paso» en su propia pantalla.

**4. Nueva petición** — sin prototipo nuevo: la estructura actual es buena. Cambios:
- Quitar la franja vacía y «Servicio preparado…»; poner «Los campos con * son obligatorios» y
  «(opcional)» donde toque; rellenar el centro de quien pide.
- Al entrar, dos botones grandes: «La pide RRHH» / «Viene de un centro (ya ratificada)».
- Acción principal «Revisar petición» al pie a la derecha; revisión con «Cambiar» por bloque;
  confirmación con número, qué pasa ahora y «Abrir expediente» / «Crear otra petición».

**5. Bolsa con sus integrantes** — sin prototipo; misma gramática que la lista:
- Arriba: nombre de la bolsa, vigencia y «Nuevo llamamiento» como acción principal.
- Siempre visible: 4 contadores (disponibles, trabajando, pendientes de incorporarse, no
  disponibles); el resto de situaciones en el filtro. Sin abreviaturas.
- Tabla de 5 columnas: orden · persona · situación (punto + palabra) · último llamamiento ·
  acciones; en móvil, fichas. La hora solo si existe; las fechas, dd/mm/aaaa.
- «Registrar resultado» no vive aquí: lleva al expediente del llamamiento. Criterios de orden y
  resumen de la bolsa, plegados tras «Cómo se ordena esta bolsa».

**6. Registrar respuesta de candidato** — prototipo `prototipos/respuesta.html`
- Arriba: «Registrar la respuesta de Antonio Reyes Álvarez», con expediente, fecha del llamamiento
  y hasta cuándo puede responder. Indicador «Paso N de 3».
- Paso 1: dos opciones grandes, «Acepta» / «Renuncia», cada una diciendo qué pasará después; enlace
  para «no respondió en plazo».
- Paso 2: fecha y hora (Madrid, con ejemplo), cómo llegó (correo / teléfono / en persona) y el
  correo adjunto como **opcional**. Errores arriba enlazados y junto a cada campo.
- Paso 3: revisión con «Cambiar» y aviso de que no se puede deshacer; botón «Registrar respuesta».
- Confirmación: número de registro, fase a la que pasa y el siguiente botón («Preparar la
  resolución» o «Llamar a la siguiente persona»). Clave, versión y huella no se ven: las pone el
  sistema.

## Prototipos

`prototipos/` — cuatro páginas estáticas enlazadas entre sí (portada → lista → ficha → respuesta),
sin frameworks ni fuentes externas. Usan copias sin tocar de `styles.css`, `tema-vec.css`,
`portal.css`, `portal-componentes.css` y `portal-flujos.css` de `origin/main`, más
`css/prototipo.css` (solo tokens `--portal-*`), que es la propuesta de añadidos al tema común.
Datos sintéticos. Sin desbordamiento horizontal ni errores JS a 1440 ni a 390 px. Llevan regiones,
encabezados, etiquetas visibles, resumen de errores y foco visible por CSS. `respuesta.html?paso=2` y `?paso=3` abren los
pasos intermedios. Capturas: `prototipos/NN_*_1440.png`, `*_390.png` y `*_completa.png`.

Límites: no se recorrieron con teclado real, zoom 200 %, lector de pantalla ni alto contraste; la API se
simuló, así que no hay datos de tiempos. Las herramientas de captura (`herramientas/`) necesitan un
árbol de `origin/main` en `/dev/shm/vec_diseno_ro` y se dejan solo como referencia.

## Orden de trabajo sugerido

1. Arreglar el arranque del portal (clave i18n que falta).
2. Un solo catálogo de fases y de estados, y los mismos recuentos en portada y lista.
3. Bloque «Siguiente paso» + documentos + historial en la ficha (necesita que el servidor diga
   quién actúa y hasta cuándo).
4. Registrar respuesta en 3 pasos, con clave, versión y huella generadas por el sistema.
5. Portada como cuadro de mandos (añade bolsas y SAE) y menú lateral estable.
6. Pasar los patrones de `prototipo.css` a `portal-componentes.css` y subir tamaños de letra.
