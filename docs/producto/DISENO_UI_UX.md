# Diseño de pantallas de VEC (UI/UX)

Fuentes absorbidas el 13/10/2026:
`docs/diseno/direccion-20260929/DIRECCION_DISENO.md` (retirado), el contrato
visual de `docs/portal_vec/expediente_contratacion_temporal_rrhh.md` (retirado),
la parte de pantallas de `docs/diseno/aspirantes_modulo_2026-09-29.md`, la
sección de i18n y presentación de `AGENTS.md` y E08 de
`ESPECIFICACIONES_AGENTES.md`.

Revisión documental del 20/10/2026 sobre `origin/main@3ef7dee0b`. Las reglas
de diseño siguen vigentes. Los inventarios de implantación de los apartados
10 y 17 conservan su corte del 13/10 y se identifican como históricos; no
son una nueva auditoría visual ni una lista actual de tareas. El estado de
cada tarea está en el [tablero](TABLERO_TAREAS.md).

Este documento reúne en un solo sitio las reglas de cómo se ven y se usan las pantallas,
los textos visibles, los PDF y los correos de VEC. Lo leen las personas de RRHH que
quieran saber cómo se decide el aspecto de la aplicación y quienes modifiquen el producto
antes de tocar `web/static/` o un texto que vaya a leer alguien.

Sustituye a la dirección de diseño del 29/09/2026, a la parte de pantallas del diseño del
módulo Aspirantes y a las reglas de presentación que estaban repartidas en `AGENTS.md` y
`ESPECIFICACIONES_AGENTES.md` (lista exacta en la sección 18). Las skills de diseño siguen
existiendo como procedimientos de trabajo; las reglas viven aquí.

Documentos hermanos de `docs/producto/`, que no se repiten aquí:

| Documento | Qué consultar allí |
| --- | --- |
| `PRD.md` | Qué es VEC, para quién y qué módulos van primero |
| `FLUJO_APP.md` | Quién hace cada paso y en qué orden (perfiles y recorridos) |
| `TRD.md` | Arquitectura, identidad por certificado, despliegue, rendimiento del servidor |
| `ESQUEMA_BACKEND.md` | Rutas de API, permisos, cifrado y datos que una pantalla puede pedir |
| `PLAN_IMPLEMENTACION.md` | Qué pantallas faltan y en qué orden se hacen |

## 1. Orden de autoridad y fuentes

Cuando dos reglas chocan:

1. Las reglas vigentes más recientes prevalecen sobre las anteriores. Si una regla de aquí
   contradice una orden posterior, se aplica la orden y se corrige este documento.
2. Dentro de este documento, lo que viene de la normativa y del lenguaje claro (secciones
   2, 9 y 13) manda sobre el aspecto (secciones 3 a 7), y el aspecto manda sobre los
   criterios generales de diseño de Impeccable. Es el orden de
   `.agents/skills/impeccable/VEC-PRIORIDAD.md`.
3. Si este documento y el código no coinciden, la diferencia se anota en la sección 17 y
   se corrige uno de los dos en su propio corte.

Procedimientos que se siguen al trabajar una pantalla (skills, en `.agents/skills/`):

| Skill | Para qué se usa |
| --- | --- |
| `usabilidad-vec` | Recorrido del novato y revisión con heurísticos |
| `aspecto-vec` | Comparar con las maquetas de RRHH, capturar antes y después |
| `disenar-sistema-visual-vec` | Componer la pantalla con el armazón común y comparar temas |
| `impeccable` | Auditar, criticar y pulir (`audit`, `critique`, `polish`, `clarify`, `harden`, `layout`). En VEC no se usan `overdrive` ni `bolder` ni fuentes externas |
| `humanizer` (con `VEC-USO.md`) | Repasar cualquier texto que lea una persona |
| `programar-interfaz-vec`, `probar-recorridos-vec` | Conectar la pantalla a la API real y recorrerla |

Material de referencia que se conserva:

| Material | Contenido |
| --- | --- |
| `docs/diseno/direccion-20260929/prototipos/` | Prototipos estáticos de portada, lista, ficha y respuesta de candidato, con capturas a 1440 y 390 px (`*_1440.png`, `*_390.png`) |
| `docs/diseno/direccion-20260929/capturas-antes/` | Cómo eran las seis pantallas el 28/09/2026 |
| Maquetas de RRHH del cuadro de mando y del nuevo llamamiento | Referencia de aspecto. No están en Git; se guardan fuera del repositorio (ver `aspecto-vec`) |
| `docs/estudio_requisitos/sistema_diseno_y_temas.md` | Estudio general del sistema visual y de los temas |
| `docs/estudio_requisitos/temas_instalables_2026-10-04.md` | Formato de los paquetes de tema de color |

Código que es autoridad visual:

| Fichero | Qué contiene |
| --- | --- |
| `web/static/comun/tema-vec.css` | Única autoridad de tokens (colores, radios, sombras, espacios), temas, modo oscuro y alto contraste |
| `web/static/portal-empleado/portal-componentes.css` | Panel, tarjeta de indicador, tablas, paginación, actividad |
| `web/static/portal-empleado/portal-patrones.css` | Patrones de trabajo: siguiente paso, línea de fases, documentos, historial, filtros, pasos, tablas que pasan a fichas |
| `web/static/portal-empleado/portal-flujos.css` | Formularios, pasos, diálogos, borradores, recibos |

## 2. Principios

1. **La persona nueva entiende la pantalla sin ayuda.** Petición de RRHH del 25/09/2026:
   «que sea una app intuitiva, ya que algunos de los que están en este Departamento se
   irán pronto». Prueba de fuego: alguien recién llegado hace su primera petición,
   localiza un expediente y sabe en qué fase está, sin manual ni compañero. Como máximo
   tres clics desde la portada hasta cualquier acción frecuente.
2. **Lo pendiente primero.** Cada pantalla empieza por lo que pide acción de quien mira
   (tareas, plazos que vencen, incidencias) y después la información.
3. **Cada ficha responde, en este orden:** qué toca ahora (qué, quién, hasta cuándo), en
   qué fase está y qué falta. Si no se puede hacer nada, dice por qué y quién debe actuar.
4. **Un solo vocabulario.** Los nombres del trabajo real («Peticiones de personal
   temporal», «Bolsas de trabajo», «Ofertas al SAE»), las ocho fases de RRHH en todas
   partes («Fase 5 de 8: Obtención del candidato»), un solo nombre por estado y los mismos
   recuentos en la portada y en la lista. Nunca códigos internos en pantalla (B12, CT58,
   AD3, V3, R7, números de duda, migraciones, capacidades).
5. **Una pantalla no es una fase.** Las pantallas sirven al trabajo; no se fuerza una
   pantalla por fase ni se altera la definición de las ocho fases sin validarlo con RRHH.
6. **Ayuda solo tras el botón «?».** El «?» está siempre en la cabecera, en el mismo
   sitio. En la pantalla no hay párrafos de ayuda, subtítulos explicativos, franjas de
   «cargado» o «actualizado», ni notas «Provisional…». La ayuda dice qué puede hacer la
   persona, qué significa lo que ve y qué hacer si algo falla; nunca cómo funciona la
   aplicación por dentro (servidor, autorización, huellas). Órdenes del 23/09 y 07/10/2026.
7. **Sin carteles sin sentido.** Nada de «pendiente de definir con RRHH», «elija una
   bolsa» cuando no hay bolsas o avisos de funcionamiento interno. El sistema calla cuando
   todo va bien; habla para errores, resultados y límites reales.
8. **Todo dato agrupado se puede pulsar.** Cifras, recuentos, estados, filas y barras de
   gráficos y tarjetas de inicio llevan con un clic a su listado filtrado. El recuento y
   la lista usan el mismo filtro del servidor, la dirección de la página conserva el
   filtro para poder volver, y la lista se pide paginada solo al pulsar (07/10/2026).
9. **Los errores dicen qué falta.** Qué ha pasado, por qué y qué hacer ahora, en lenguaje
   llano. Nunca un código HTTP, una ruta, una huella ni un identificador técnico.
10. **Una acción principal por bloque**, azul y en el mismo sitio. El resto, de contorno o
    como enlace. Antes de algo que no se puede deshacer, pantalla de revisión; después,
    confirmación con qué ha pasado, su número y qué ocurre ahora.
11. **Nada de demostración.** Todo es código de producción; solo los datos son sintéticos,
    con nombres verosímiles (Antonio Reyes Álvarez, no «Usuario de prueba»). En lo nuevo no
    aparecen «prueba», «demo», «ejemplo», «sintético», «sin eficacia», «sin validez», «no
    acredita» ni «borrador de desarrollo», ni en pantallas, ni en PDF, ni en correos
    (12/10/2026; el 13/10 se aclaró que no hace falta limpiar lo ya hecho). Se mantienen
    los términos legales propios («periodo de prueba») y los estados reales de un circuito.
12. **Cada cosa se llama por lo que es.** Un borrador no es un documento firmado;
    identificarse con certificado no es firmar; un aviso no es una entrega acreditada; una
    declaración no es una resolución; guardar la referencia de un correo no es custodiarlo.
    La pantalla nunca afirma firma, pago, envío, registro, cálculo o guardado sin respuesta
    real del servidor.
13. **Todo se hace desde la web.** Subir ficheros, generar PDF o Word, enviar a firma,
    descargar y ver recibos tienen pantalla. Una capacidad sin pantalla no está terminada
    (12/10/2026). Lo que está a medias se termina; no se quita del menú (13/10/2026).
14. **El ordenador es la vista prioritaria.** Las tablas, los formularios y el espacio se
    optimizan para escritorio; el móvil sigue siendo usable sin perjudicar al escritorio
    (19/09/2026).
15. **Los fallos pequeños de diseño son fallos.** Textos en dos idiomas cargados a la vez,
    errores tragados, pantallas en blanco, botones que sacan de la pantalla, carteles sin
    sentido: se arreglan antes de añadir funciones (07/10/2026).
16. **Instantánea.** La pantalla útil aparece en menos de un segundo; cada lectura del
    servidor, por debajo de 300 ms en el 95 % de los casos con volumen real. El portal
    pinta primero y carga cada módulo después sin bloquear a los demás. Los requisitos de
    servidor están en `TRD.md`.
17. **Lo que RRHH no ha decidido se configura.** Plazos, catálogos y opciones que dependen
    de RRHH se cambian desde la aplicación, con versión y registro; la pantalla no los
    trae fijos.

## 3. Estructura de una pantalla

Orden común, cuando sus partes existan:

1. armazón con menú lateral y cabecera;
2. título, miga de pan y estado del módulo;
3. navegación del recorrido o pestañas;
4. indicadores (`rejilla-kpi` y `tarjeta-kpi`);
5. área de trabajo (`panel`, `cabecera-panel`, `cuerpo-panel`);
6. resumen lateral, solo si ayuda a decidir;
7. acciones principales al final del bloque que modifican.

Valor, rótulo y estado se muestran como elementos separados; nunca se juntan en una frase.

- **Menú lateral** azul marino (`--portal-lateral-fondo`), de 280 px (`--portal-lateral`).
  Arriba la marca: icono blanco en cuadrado azul redondeado, nombre del área en dos líneas
  de 18 px y «Diputación de Granada» debajo en 13 px. Cada entrada con un número en círculo
  de 32 px de color (una tinta por módulo), texto blanco de 15 px en dos líneas como
  máximo, separador tenue y 14 px de relleno. La activa se rellena con
  `--portal-azul-700`. Abajo, «Ayuda» y «Cerrar sesión». Sin pastillas «ACTIVO» ni avisos
  dentro del menú. El menú no cambia según el módulo en el que se esté.
- **Cabecera blanca de 72 px**: título de página de 26 px con la miga pequeña encima
  («Inicio → Llamamientos → Nuevo llamamiento»); a la derecha, campana con contador rojo,
  «?» y bloque de usuario (avatar con iniciales, nombre y perfil en dos líneas). Tamaño de
  texto y alto contraste van dentro del menú de usuario.
- **Lienzo** gris azulado (`--portal-fondo`) con tarjetas blancas de radio 12 px, borde de
  1 px y sombra muy suave, separadas 20 a 24 px. Nunca apartados separados solo por una
  raya.
- Navegación estable: nada se abre en otra pestaña del navegador; «Volver» regresa a donde
  se estaba, con los filtros conservados; el mismo elemento siempre en el mismo sitio
  (buscar arriba, acción principal arriba a la derecha o al pie del paso, «?» en la
  cabecera).
- Ocultar una entrada del menú no es un permiso. El servidor decide qué puede hacer cada
  persona; el menú solo refleja esa decisión.

## 4. Sistema visual

### 4.1 Tipografía

| Uso | Tamaño | Peso |
| --- | --- | --- |
| Título de página | 26 px | 600 |
| Título de tarjeta o panel | 18 px | 600 |
| Menú lateral | 15 px | 400 |
| Texto normal y celdas | 14 px | 400 |
| Rótulos, cabeceras de tabla, metadatos | 13 px | 400 |
| Valor de indicador | 28 px | 600 |

Una sola familia, la del tema; pesos 400 y 600, nada más. Hoy `portal.css` declara
`Inter, system-ui, …` sin incluir el fichero de Inter, así que se ve la fuente del sistema.
No se añaden fuentes externas. La tipografía institucional está sin decidir
(`sistema_diseno_y_temas.md` §14).

Cada persona elige el tamaño del texto en Preferencias (normal, grande y muy grande: 100,
125 y 150 %). Por eso los tamaños se escriben en `rem`.

### 4.2 Color y tokens

Todos los colores salen de los tokens `--portal-*` de `web/static/comun/tema-vec.css`. Los
módulos no escriben hexadecimales ni crean otra paleta. Si falta un token, se añade allí
una vez, con su valor en modo oscuro y en alto contraste.

| Token | Valor claro | Para qué |
| --- | --- | --- |
| `--portal-azul-950` | `#062d56` | Menú lateral, marca |
| `--portal-azul-700` | `#075fca` | Acción principal, enlace, foco, entrada activa |
| `--portal-azul-100` | `#e9f2ff` | Fondos suaves de selección |
| `--portal-tinta` | `#10213b` | Texto principal |
| `--portal-muted` | `#586985` | Rótulos, metadatos, cabeceras de tabla |
| `--portal-borde` | `#dde5ef` | Bordes de tarjeta y separadores |
| `--portal-borde-control` | `#687588` | Bordes de campos (3:1 con el fondo) |
| `--portal-fondo` | `#f2f5f9` | Lienzo |
| `--portal-superficie` | `#ffffff` | Tarjetas y paneles |
| `--portal-superficie-alterna`, `--portal-cabecera-panel` | `#f7f9fc`, `#f5f8fc` | Filas alternas, cabeceras |
| `--portal-exito` / `-suave` | `#147a36` / `#eaf8ee` | Hecho, disponible |
| `--portal-aviso` / `-suave` | `#9a5c00` / `#fff5d9` | Plazo cercano, pendiente |
| `--portal-peligro` / `-suave` | `#ba2737` / `#fff0f1` | Vencido, error, acción destructiva |
| `--portal-violeta`, `--portal-cian`, `--portal-naranja`, `--portal-turquesa`, `--portal-oliva`, `--portal-morado` | varios | Tinta de cada módulo en el menú e iconos de indicador |

Colores de los indicadores: azul para bolsas, verde para disponibles, naranja para
pendientes, violeta para contratos y turquesa para cobertura.

Reglas de color:

- Nada se comunica solo con color. Cada estado lleva palabra o símbolo («● Disponible»,
  «✓ Hecho», «Ahora», «Falta»).
- Contraste mínimo 4,5:1 para texto y 3:1 para componentes y bordes de control. Lo
  comprueban `web/static/comun/codexf-accesibilidad-contraste.test.mjs` y
  `codexf-temas-contraste.test.mjs`.
- Como mucho una pastilla de estado por fila; los números nunca van en pastilla.

### 4.3 Espaciado, radios y sombras

- Escala de espacios: 4 · 8 · 12 · 16 · 20 · 24 · 32 px. Tokens existentes:
  `--portal-espacio-1` a `--portal-espacio-5` (4, 8, 12, 16 y 24 px); 20 y 32 no tienen
  token todavía.
- Relleno de tarjeta de 20 a 24 px. Los bordes izquierdos del título, los indicadores y
  las tarjetas coinciden.
- Radios: `--portal-radio` (8 px) para botones y campos, `--portal-radio-lg` (12 px) para
  tarjetas. Sombras: `--portal-sombra-sm` y `--portal-sombra`. Sin degradados.

### 4.4 Temas, modo oscuro y alto contraste

| Pieza | Qué hace hoy | Fichero |
| --- | --- | --- |
| Modo de color | Nueve opciones que cada persona elige en Preferencias: sistema, claro, oscuro, diputación de Granada, arena, salvia, lavanda, azul sereno y noche suave | `tema-vec.css`, `tema-vec.js` |
| Tema institucional | Dos temas compilados, institucional y granate | `TEMAS_VEC` en `tema-vec.js` |
| Alto contraste | Capa independiente que prevalece sobre cualquier tema | `body[data-contraste="true"]` |
| Tamaño de texto | Normal, grande y muy grande | `html[data-tamano-texto]` |
| Temas instalables | Formato de paquete de color y validador por línea de órdenes; todavía no se instalan ni activan desde Administración | `internal/modules/administracion/domain/temas/`, `cmd/vec-temas-validar/` |

Un tema solo cambia colores (variante clara y oscura). Nunca cambia estructura,
tipografía, contenido, permisos ni documentos ya emitidos. Al comparar temas se usa el
mismo recorrido (navegación, indicadores, panel, formulario, botón, estado, tabla) y se
revisan al menos Portal, Bolsa, Personal, Cronos y Dietas en escritorio y móvil. Las
preferencias las guarda el servidor; no se usa almacenamiento del navegador. Los documentos
generados tienen plantillas propias y no siguen el tema de la web.

Los temas son instalables y no quedan fijados en el código. Por ahora solo
hay temas de color elegibles en Preferencias. El instalador llegará con la administración:
paquete con manifiesto versionado, tokens claro y oscuro, nombres en cada idioma,
comprobación de contraste WCAG AA y auditoría; un tema en uso solo se desactiva.

Pendiente de los temas instalables: instalar y consultar desde Administración, y aprobar,
activar y retirar con versión esperada (`temas_instalables_2026-10-04.md`, cortes 3 y 4).

### 4.5 Iconos

`web/static/comun/iconos-vec.js` (`icono(nombre, clase)`) da los iconos en SVG en línea. Un
icono decorativo lleva `aria-hidden="true"`; un botón que solo tiene icono lleva
`aria-label` con clave de texto. Las acciones de fila pueden ser iconos (ver, más
opciones) o un botón de contorno («Seleccionar ›»). No se usan logotipos, marcas ni textos
de otras aplicaciones.

## 5. Componentes que ya existen

Antes de crear una clase, busca aquí. Un componente que repite la función de uno común no
se acepta.

| Componente | Clases | Fichero | Cuándo usarlo |
| --- | --- | --- | --- |
| Panel o tarjeta de apartado | `panel`, `cabecera-panel`, `cuerpo-panel`, `pie-panel` | `portal-componentes.css`, `portal-patrones.css` | Todo bloque. Título de 18 px a la izquierda y selector o acción a la derecha; sin cabecera tintada ni doble borde |
| Indicadores | `rejilla-kpi`, `tarjeta-kpi`, `icono-kpi`, `valor-kpi`, `etiqueta-kpi`, `enlace-kpi` | `portal-componentes.css` | Fila de 4 o 5 tarjetas iguales: icono de 56 px, valor de 28 px, rótulo de 14 px y «Ver detalle →». Sin dato, «—» con el motivo en el título accesible |
| Tabla de datos | `tabla-contenedor`, `tabla-datos`, `marco-tabla-paginado`, `paginacion-marco` | `portal-componentes.css` | Cabecera de 13 px sin fondo fuerte, filas de 52 a 56 px, nombre principal enlazable, números a la derecha, casillas a la izquierda si hay acciones en lote |
| Tabla que pasa a fichas en móvil | `tabla-apilable`, con `data-etiqueta` en cada celda | `portal-patrones.css` | Listados que deben leerse bien a 390 px |
| Cabecera de página | `cabeza-pagina` | `portal-patrones.css` | Título a la izquierda, acción principal a la derecha |
| Siguiente paso | `siguiente-paso` | `portal-patrones.css` | Primer bloque de una ficha de expediente |
| Línea de fases | `linea-fases` | `portal-patrones.css` | Fase actual, hechas y pendientes, con palabra y símbolo |
| Documentos | `lista-documentos` | `portal-patrones.css` | Lista de comprobación: está, falta, en firma, firmado |
| Historial legible | `historial-frases` | `portal-patrones.css` | «Fecha hora · persona hizo tal cosa» |
| Filtros quitables | `filtros-quitables`, `filtros-activos`, `chip-quitar` | `portal-patrones.css` | Filtros activos visibles que se quitan con un clic |
| Datos clave | `datos-clave` | `portal-patrones.css` | Resumen corto de la ficha, con «Ver todos los datos» |
| Detalle técnico plegado | `detalle-tecnico-plegado` | `portal-patrones.css` | Referencias, versión y recibos técnicos |
| Opciones grandes | `opciones-grandes`, `opcion-grande` | `portal-patrones.css` | Elegir vía al principio («Acepta» / «Renuncia») |
| Pasos | `pasos-numerados`; en formularios `pasos`, `paso`, `paso-numero` | `portal-patrones.css`, `portal-flujos.css` | Asistentes: pasos en línea unidos por un conector, el actual en círculo azul relleno y los pendientes en gris |
| Resumen de errores | `resumen-errores` | `portal-patrones.css` | Arriba del formulario, enlazado a cada campo |
| Revisión y confirmación | `revision-datos`, `confirmacion-tramite` | `portal-patrones.css` | Antes y después de un acto irreversible |
| Tareas pendientes | `tareas-pendientes` | `portal-patrones.css` | Lista de trabajo del cuadro de mandos |
| Campos | `campo`, `grupo-campo`, `control-formulario`, `ayuda-campo` | `portal-flujos.css` | Etiqueta visible encima, pista corta, error junto al campo |
| Diálogo | `dialogo-detalle`, `cabecera-dialogo`, `contenido-dialogo`, `acciones-dialogo` | `portal-flujos.css` | Detalle o confirmación sin cambiar de pantalla |
| Estado vacío | `vacio-controlado`, `vacio-coleccion` | `portal-flujos.css` | Lista sin resultados, con su acción |
| Recibo | `ct-recibo`, `recibo-borrador` y otros por módulo | varios | Resultado de una operación con número y fecha |
| Cobertura o progreso | `barra-cobertura` | `portal-componentes.css` | Barra fina verde con el porcentaje a la derecha |
| Agenda y actividad | `lista-proximos`, `elemento-proximo`, `lista-actividad`, `elemento-actividad` | `portal-componentes.css` | Agenda: fecha en cuadro tintado, dos líneas y estado. Actividad: icono circular, enlace y «Por autor · fecha» |
| Accesos rápidos | `accesos-rapidos`, `acceso-rapido` | `portal-componentes.css` | Tarjetas bajas con icono y texto al pie del cuadro |

Botones: principal azul sólido de 40 px de alto y radio 8 px; secundario de contorno;
peligro en rojo y siempre con confirmación. Una principal por bloque.

`pasos-numerados`, `tareas-pendientes` y `confirmacion-tramite` existen en CSS pero ninguna
pantalla los usa aún (búsqueda del 13/10/2026). Quien haga la pantalla que los necesita
los usa en vez de inventar otros.

## 6. Estados de una pantalla

Toda zona que pide datos al servidor distingue estos estados, con palabra y no solo color:

| Estado | Qué ve la persona |
| --- | --- |
| Cargando | El armazón y los títulos ya pintados; la zona con `aria-busy="true"` y un indicador discreto. Nunca la pantalla en blanco |
| Con datos | Los datos, sin franjas de «cargado» |
| Vacío | Qué significa y la acción que procede: «No hay expedientes en trámite. Cree una nueva petición.» |
| Error | Qué ha pasado y qué hacer, con «Reintentar». El resto de la pantalla sigue funcionando. Ejemplo: «No se ha podido cargar el cuadro. Compruebe la conexión y pulse Reintentar. Si sigue igual, avise a Informática.» |
| Sin permiso | «No tiene permiso para ver esto» y a quién pedirlo, sin datos parciales |
| Solo lectura | Los datos sin botones de acción, y el motivo si no es evidente |
| Resultado incierto | Un envío sin respuesta no se da por hecho ni por fallido: se bloquea el reenvío con datos distintos y se ofrece reintentar el mismo, que el servidor reconoce sin duplicar |

Carga progresiva: el portal pinta el armazón y cada módulo carga por su cuenta; si uno
falla, los demás siguen. Las listas largas se piden paginadas y el detalle solo al
abrirlo. Un botón que todavía no se puede usar queda deshabilitado con el motivo concreto
en una frase («Necesita una bolsa constituida»), nunca con un cartel genérico.

## 7. Adaptación a la pantalla

Anchos que se comprueban siempre: 1440, 1024 y 390 px, más zoom al 200 %. En ninguno puede
haber desbordamiento horizontal de la página.

- Las tablas anchas tienen scroll horizontal dentro de su contenedor (`tabla-contenedor`,
  con `overflow-x: auto`); nunca ensanchan la página. Si la tabla debe leerse en móvil,
  usa `tabla-apilable` y pasa a fichas por debajo de 760 px.
- Menú, indicadores y columnas se apilan. A 520 px los indicadores van de dos en dos.
- La acción principal queda visible sin buscarla; los filtros secundarios se pliegan tras
  «Más filtros».
- Cortes de ancho de los CSS comunes: 1100, 1024, 760/780, 520 y 320 px. Un módulo no
  inventa los suyos si le vale uno de estos. `html` tiene `min-width: 320px`.
- Objetivos táctiles de al menos 24 × 24 px.

## 8. Accesibilidad

Obligación legal: Real Decreto 1112/2018 (Directiva UE 2016/2102), que remite a UNE-EN
301 549, equivalente a WCAG 2.1 nivel AA. Se diseña ya para WCAG 2.2 AA: foco nunca
tapado, objetivos de 24 × 24 px, nada que solo se pueda hacer arrastrando, ayuda en el
mismo sitio en todas las pantallas, no pedir otra vez datos ya dados y acceso sin pruebas
cognitivas. La accesibilidad se piensa desde el diseño, no se añade al final.

| Aspecto | Regla | Qué hay en el código |
| --- | --- | --- |
| Teclado | Todo se usa con teclado, en orden lógico; nada depende de pasar el ratón | Enlace «Saltar al contenido» (`salto-contenido` en `portal.css`) |
| Foco | Siempre visible y nunca tapado por la cabecera | Contorno de 3 px `--portal-azul-700` en `:focus-visible`; `scroll-margin-top` en anclas |
| Contraste | 4,5:1 texto, 3:1 componentes, en cada tema y en alto contraste | Pruebas de contraste en `web/static/comun/` |
| Zoom y tamaño | Usable al 200 % y con texto grande, sin scroll horizontal | Tamaño de texto en Preferencias |
| Lector de pantalla | `main` y `nav`, encabezados en orden, tablas con `caption` y `th scope`, etiquetas asociadas, `aria-live` para cambios y errores | Región `#anuncios` con `aria-live="polite"` en `portal-empleado/index.html` |
| Movimiento | Se respeta `prefers-reduced-motion` | En los CSS comunes |
| Colores forzados | Se respeta `forced-colors` | En varios CSS de módulo |
| Idioma | El atributo `lang` cambia con el idioma | `web/static/comun/idioma.js` |
| Estados alternativos | Vacío, carga, error y sin permiso, también para lector de pantalla | Sección 6 |
| Documentos | PDF y Word accesibles: etiquetados, orden de lectura, idioma y título | Hueco, ver abajo |

PDF. El renderizador común (`internal/vec/adapters/documentos/pdf/renderizador.go`, con
la librería fpdf) fija idioma y título, pero no genera PDF etiquetado (estructura de
lectura para lectores de pantalla) y el título está escrito en castellano en el código.
Cumplir la norma en documentos exige otra vía de generación o un postproceso, sin decidir.
Mientras tanto, todo dato de un PDF debe estar también en la ficha de la pantalla.

## 9. Idiomas

Los textos y los idiomas se mantienen en catálogos para permitir su traducción sin cambiar el código.

- Todo texto visible, validación, estado, ayuda, documento y notificación sale de los
  catálogos de datos por idioma: `web/static/textos/<idioma>/<modulo>.json` (hoy `es` y
  `en`, 165 catálogos cada uno). La lista de idiomas, el de defecto y su localización
  están en `web/static/textos/idiomas.json` (`es-ES` y `en-GB`). Añadir un idioma es
  añadirlo al índice y traducir su carpeta.
- Se leen con `web/static/comun/textos.js`, que da `traducir`, `plural`, `numero` y
  `fecha`. Los plurales usan las categorías de `Intl.PluralRules` (`one`, `other`…), nunca
  condiciones en el código.
- Fechas, números, moneda y plurales se formatean con `Intl` según la localización del
  idioma activo; la moneda, en euros con esa localización. Fechas y horas se presentan en
  hora de Madrid (`Europe/Madrid`); los campos de fecha y hora se interpretan en Madrid y
  viajan en UTC (`web/static/portal-empleado/hora-madrid.js`). En castellano la fecha se
  ve dd/mm/aaaa y el formato se indica en el campo. La hora solo aparece si existe.
- El idioma se elige con el parámetro `lang` de la dirección o, si el índice lo permite,
  con la preferencia del navegador. No se guarda en el navegador. Solo se carga el
  catálogo del idioma activo.
- Los datos que llegan del servidor se escapan siempre al pintarlos. Los errores y estados
  del servidor viajan como código y la pantalla los traduce.
- Al quitar un texto se quita su clave si nadie más la usa. Las claves que faltan las
  detecta `web/static/comun/textos-catalogos.test.mjs`; las pruebas usan el traductor real.
- Vocabulario del código: castellano coherente, salvo términos técnicos universales y
  convenciones de Go. No se mezclan castellano e inglés en el mismo vocabulario.

## 10. Privacidad visible

La columna de estado siguiente es una fotografía histórica del 13/10.
Las reglas de minimización, acceso y ayuda siguen siendo requisitos; para
saber qué está integrado y qué falta, consulte el [tablero](TABLERO_TAREAS.md)
y las evidencias de la tarea. Este corte documental no vuelve a comprobar
el contacto completo ni los demás estados de cada pantalla.

| Dónde | Qué se ve | Estado (13/10/2026) |
| --- | --- | --- |
| Listados (bolsas, candidatos, inscripciones) | Contacto y documento de identidad enmascarados | Existe. La API de Bolsa devuelve siempre la forma enmascarada |
| Ficha del candidato para RRHH | Contacto completo, dejando constancia de quién lo consulta | La API existe (`?ver=completo` en `internal/modules/bolsa/adapters/httpinterno/datos_contacto_participacion.go`, con permiso, finalidad y motivo propios y asiento en la auditoría). Ninguna pantalla la usa todavía |
| Mi Bolsa (el propio aspirante) | Su contacto completo, para comprobar que es correcto | Decidido el 13/10/2026; ver `PLAN_IMPLEMENTACION.md` |
| Sin permiso para el dato completo | La forma enmascarada con un aviso claro de que no tiene permiso | Pendiente junto con lo anterior |

Enmascarado del documento de identidad (orientación de la AEPD de 2019, pendiente de
confirmar por la persona delegada de protección de datos): DNI `***4567**`, NIE
`****4567*`; pasaporte y otros, cuatro caracteres en las posiciones 4 a 7 si el número
tiene al menos 8, y todo oculto si es más corto.

Otras reglas:

- Datos de trabajadores y de aspirantes no se mezclan en la misma pantalla salvo en los
  traspasos previstos (de bolsa a interinidad, de proceso selectivo a plaza).
- Las listas muestran lo mínimo para identificar y decidir; el resto, en la ficha. Los
  identificadores técnicos no son la presentación principal.
- La pantalla no muestra datos de quien no tiene permiso, ni en mensajes de error, ni
  revela si existe una ficha ajena.
- Sin cookies, `localStorage`, `sessionStorage` ni IndexedDB como sesión o para guardar
  credenciales, y nunca un token en la dirección de la página.
- Las consultas quedan registradas. No hace falta decirlo en cada pantalla; lo explica la
  ayuda «?» y la política de privacidad.

## 11. Pantallas de RRHH: estructura aprobada

Estructura de la dirección de diseño aprobada el 29/09/2026. Prototipos en
`docs/diseno/direccion-20260929/prototipos/` (`index.html`, `expedientes.html`,
`expediente.html`, `respuesta.html`; `respuesta.html?paso=2` y `?paso=3` abren los pasos
intermedios). Qué parte está ya hecha en el portal lo dice `PLAN_IMPLEMENTACION.md`.

### 11.1 Portada de RRHH (cuadro de mandos)

- Arriba: saludo y fecha; a la derecha, «Nueva petición de personal».
- Siempre visible: «Tiene N tareas pendientes», por plazo (vencido en rojo, hoy en
  ámbar), cada una con verbo, expediente, plazo y su botón («Registrar respuesta»,
  «Subsanar»).
- Después: cuatro indicadores que llevan a la lista filtrada (peticiones en trámite,
  plazos de esta semana, personas disponibles en bolsas, ofertas al SAE abiertas).
- Debajo, en dos columnas: peticiones por fase (las ocho fases, con el mismo recuento que
  la lista) y bolsas con ofertas al SAE.
- Sin rejilla de «módulos», sin título duplicado y sin «Ayuda» como pestaña.

### 11.2 Lista de expedientes

- Arriba: «11 peticiones en trámite» (el mismo número que la portada) y «Nueva petición de
  personal».
- Buscador único (número, centro o persona) que filtra al escribir; filtros de fase,
  centro y «Mostrar» (en trámite, me tocan a mí, esperan a otros, terminadas); etiquetas
  de filtros activos con «×» y «Quitar todos».
- Cinco columnas: Expediente · Centro y categoría · Fase («Fase 4 de 8») · Siguiente paso
  («Intervención: emitir informe») · Plazo (una pastilla, solo si vence o venció).
- Orden por defecto: plazo más próximo. Paginación abajo, «Mostrando 1 a 20 de 132» a la
  izquierda y las páginas a la derecha.
- En móvil, fichas apiladas y filtros tras «Más filtros».
- Sin «Distribución por fase» ni «Trabajo pendiente» al pie (están en la portada), sin la
  columna Modalidad y sin botones que abran otras pestañas.

### 11.3 Ficha de expediente

- Arriba: número (título de la página), categoría y centro, «Fase 5 de 8: …» y un único
  estado.
- Justo debajo, «Siguiente paso»: qué, quién («Sección de Selección (usted)»), hasta
  cuándo y el botón. Si espera a otro: «Esperando a Intervención desde el 22/09».
- Línea de las ocho fases con palabra y símbolo (✓ Hecho, Ahora, Falta).
- Dos columnas: Documentos como lista de comprobación (está, falta, en firma, con
  «Descargar») e Historial en frases («24/09/2026 11:02 · Carmen Molina Ortega envió el
  llamamiento…»); a la derecha, ocho datos clave y «Ver todos los datos».
- Plegado tras «Ver detalle técnico»: referencias, versión, flujo.
- Los formularios de cada fase no van en la ficha: se abren desde «Siguiente paso» en su
  propia pantalla.

### 11.4 Nueva petición

- Al entrar, dos botones grandes: «La pide RRHH» y «Viene de un centro (ya ratificada)».
- «Los campos con * son obligatorios» y «(opcional)» donde toque; el centro de quien pide,
  ya rellenado.
- Acción principal «Revisar petición» al pie, a la derecha; revisión con «Cambiar» por
  bloque; confirmación con número, qué pasa ahora y «Abrir expediente» o «Crear otra
  petición».

### 11.5 Bolsa con sus integrantes

- Arriba: nombre de la bolsa, vigencia y «Nuevo llamamiento» como acción principal.
- Cuatro contadores pulsables: disponibles, trabajando, pendientes de incorporarse y no
  disponibles; el resto de situaciones en el filtro. Sin abreviaturas.
- Las situaciones usan el nombre singular del catálogo en la lista, la ficha y los filtros:
  «Disponible», «No disponible», «Trabajando» o «Excluido».
- Cinco columnas: orden · persona · situación (punto y palabra) · último llamamiento ·
  acciones; en móvil, fichas. Fechas dd/mm/aaaa y la hora solo si existe.
- «Registrar resultado» no vive aquí: lleva al expediente del llamamiento. Los criterios de
  orden, plegados tras «Cómo se ordena esta bolsa».

### 11.6 Registrar la respuesta de un candidato

- Arriba: «Registrar la respuesta de Antonio Reyes Álvarez», con expediente, fecha del
  llamamiento y hasta cuándo puede responder; «Paso N de 3».
- Paso 1: dos opciones grandes, «Acepta» y «Renuncia», cada una diciendo qué pasará
  después; enlace para «no respondió en plazo».
- Paso 2: fecha y hora (Madrid, con ejemplo), cómo llegó (correo, teléfono, en persona) y
  el correo adjunto como opcional. Errores arriba enlazados y junto a cada campo.
- Paso 3: revisión con «Cambiar» y aviso de que no se puede deshacer; botón «Registrar
  respuesta».
- Confirmación: número de registro, fase a la que pasa y el siguiente botón («Preparar la
  resolución» o «Llamar a la siguiente persona»). Clave de operación, versión y huella las
  pone el sistema y no se ven.

### 11.7 Estados y actos del expediente

Viene del contrato visual de la especificación del expediente de Contratación temporal
(julio de 2026):

- El estado de un expediente o de una tarea se dice con color, símbolo y palabra a la vez:
  gris para pendiente, azul para en curso, verde para completado, rojo para incidencia y
  ámbar para «esperando a otro departamento o sistema».
- Cada acto administrativo es un botón y una operación distintos: guardar borrador,
  validar, enviar, firmar, registrar y confirmar. Ninguno hace a la vez el trabajo de otro.
- La ficha reserva sitio para documentos, comunicaciones, historial y auditoría, aunque
  alguno esté vacío.

## 12. Patrones por tipo de pantalla

Sirven para cualquier módulo. Las pantallas de la sección 11 son su aplicación a RRHH.

### 12.1 Bandeja o lista

Recuento y acción principal arriba; buscador único que filtra al escribir; filtros
visibles y quitables, guardados en la dirección; pocas columnas (identificación, quién o
dónde, fase, siguiente paso, plazo); orden por lo que vence antes; paginación abajo; fichas
apiladas en móvil. Acciones en lote con casillas a la izquierda cuando tengan sentido.

### 12.2 Ficha

Identificador de negocio como título, fase y un estado; «Siguiente paso»; línea de fases;
documentos como lista de comprobación; historial en frases; datos clave con «Ver todos los
datos»; detalle técnico plegado. Los formularios se abren aparte.

### 12.3 Asistente o formulario

- Lo largo o poco frecuente, una cosa por paso, con «Paso 2 de 4». Lo diario de RRHH puede
  agrupar campos en bloques con título. Columna derecha opcional con «Resumen» y la acción
  principal al pie («Siguiente: … →», ancha y azul) y «Cancelar» de contorno debajo.
- Si hay vías distintas, se eligen al principio con dos botones grandes y cada vía muestra
  solo sus campos y documentos.
- Etiqueta visible encima de cada campo (nunca solo el texto de muestra), obligatorio u
  opcional marcado, pista corta si hace falta. Pedir solo lo necesario.
- Se rellena lo que el sistema ya sabe (centro, fechas razonables, datos de la bolsa) y no
  se vuelve a pedir.
- Listas cortas como opciones visibles; largas con buscador. Fechas con formato visible.
- Validación al salir del campo y al enviar, con resumen de errores arriba enlazado.
- Revisión con «Cambiar» antes de un acto irreversible; confirmación con número, qué pasa
  ahora y el siguiente botón.
- Claves de operación, versiones y huellas las pone el sistema.
- Aviso al salir con cambios sin guardar, o borrador guardado.

### 12.4 Firma

- La firma se hace con GrxFirma desde el navegador (protocolo `afirma://`, cliente en
  `web/static/portal-empleado/modulos/contratacion-temporal/firma-autofirma.js`) y la
  verifica el servidor con su validador. El estado del circuito está en
  `PLAN_IMPLEMENTACION.md`.
- Antes de firmar, la persona ve el documento (o lo descarga) y quién firma en cada paso.
- Durante la firma: aviso de que se abrirá GrxFirma en su equipo y estado visible mientras
  espera. Si GrxFirma no responde, el mensaje dice qué comprobar (que esté instalado y
  abierto) y ofrece reintentar.
- Después: el documento aparece como «Firmado» en la lista de documentos, con fecha,
  firmante y recibo. Se muestran los estados reales del circuito («Pendiente de firma de
  la Jefatura», «Pendiente de firma en Portafirmas»).
- Identificarse con certificado no es firmar: la pantalla nunca presenta el acceso como
  firma de un documento.

### 12.5 Descarga y generación de documentos

- Botón con verbo y formato («Descargar resolución (PDF)», «Generar informe (Word)»).
- Si el documento tarda, estado visible y botón bloqueado hasta terminar, para no pedirlo
  dos veces.
- Nombre de fichero legible (tipo y número de expediente), sin identificadores internos.
- Cada descarga de un documento con datos personales queda registrada.
- Documento accesible (sección 8) y sin marcas de prueba.

### 12.6 Recibo y confirmación

- Qué ha pasado en una frase completa («Se ha registrado la aceptación de Antonio Reyes
  Álvarez»), número de registro, fecha y hora de Madrid, y qué ocurre después.
- El botón del paso siguiente.
- Si el envío se repite con los mismos datos, se muestra el mismo recibo, no uno nuevo.
- Los identificadores técnicos, tras «Ver detalle técnico».

## 13. Pantallas del área personal (aspirantes y empleados)

### 13.1 Perfil y contacto

- «Sus datos de identidad»: nombre, apellidos y documento enmascarado. Una línea breve
  indica que salen del certificado y que se corrigen en origen; no se editan aquí.
- «Teléfono y domicilio»: solo los campos que pide el catálogo de datos personales, con
  marca de obligatorio u opcional. Si un campo es condicional, la pantalla explica la
  condición con el código que envía el servidor. Si no se pide ninguno, una frase lo dice.
- Formatos que se validan en el campo: teléfono de 9 cifras (o internacional con «+» y 8
  a 15 cifras), móvil que empieza por 6 o 7, domicilio de 5 a 200 caracteres y código
  postal español de 01000 a 52999.
- Al cambiar un dato que ya tenía valor se pregunta si ha cambiado o estaba mal (los
  motivos son «dato nuevo», «cambio de dato» y «corrección de error»). Un cambio por
  formulario.
- Sin ficha: se muestran los datos del certificado y el botón «Crear mi ficha». Si ya
  existe, la pantalla recarga la ficha en vez de dar un error.
- Si el catálogo de datos personales no responde, se puede consultar la ficha pero no se
  ofrecen campos de contacto; nunca se piden todos por defecto.

Diseño de servidor de esta pantalla (cifrado, índice ciego, permisos, API):
`docs/diseno/aspirantes_modulo_2026-09-29.md`, que lo conserva porque no está en
`ESQUEMA_BACKEND.md`.

### 13.2 Aviso de convocatorias con plazo abierto

En el inicio del área personal, cada convocatoria o puesto con plazo abierto se compara
con los requisitos de acceso de sus bases. Por requisito se muestra «Cumple», «No cumple»
o «Pendiente», con el motivo y de dónde sale el dato. Un requisito escrito solo como
texto, una equivalencia no aprobada o un dato que falta queda «Pendiente»; no se
interpreta con IA. El resultado global distingue cuatro casos:

- puede presentar la solicitud ya, con sus datos;
- puede prepararla, pero debe completar o acreditar los datos que se indican;
- hay un incumplimiento comprobado, que se muestra y se explica;
- el plazo está cerrado o ya presentó la solicitud.

Un «Pendiente» no impide presentar la solicitud si las bases permiten aportarlo durante
el trámite. Al pulsar el aviso se abre el detalle público de la convocatoria (categoría,
bases, requisitos, plazo, documentos y vía de solicitud) y desde allí se inicia la
solicitud ya rellenada, después de que la persona lo confirme. Vale para aspirantes
externos y para empleados. La regla completa está en `PRD.md` y en `FLUJO_APP.md`.

### 13.3 Titulación pendiente en las bases (pantalla de RRHH)

Al editar un requisito de titulación de una oferta de empleo público hay un control
«Permitir inscripción con titulación pendiente», desactivado por defecto. Al activarlo son
obligatorios el hito o la fecha límite de obtención y la evidencia admitida. En bolsas de
incorporación inmediata el control aparece deshabilitado. Tras publicar las bases no se
puede cambiar libremente: la pantalla lleva al circuito de nueva versión o rectificación.

## 14. Textos

- Frases de 20 a 25 palabras como máximo, voz activa, trato de usted («Revise la fecha»).
- Botones que empiezan por verbo («Registrar respuesta», «Enviar a firma»); nunca
  «Aceptar» o «Enviar» a secas.
- Vocabulario del Departamento: petición, bolsa, llamamiento, nombramiento, renuncia,
  cese, Firmadoc, SAE. Sin jerga informática (API, token, servidor, versión, huella, 409).
- Historial y estados con nombres de personas y fechas, no con identificadores.
- Sin abreviaturas en rótulos («No disponible», no «No disp.») y con tildes.
- Rótulos cortos, sin mayúsculas largas ni texto centrado.
- Fuentes de lenguaje claro: guías de la Comunidad de Madrid, Región de Murcia, Cataluña,
  IVAP y FEMP. El destinatario debe encontrar, entender y usar la información a la
  primera.
- Todo texto se repasa con `humanizer`, que no puede volverlo coloquial en textos
  administrativos ni tocar textos jurídicos fijados.

## 15. Cómo se trabaja una pantalla

1. **Antes.** Leer este documento y `FLUJO_APP.md` para el paso concreto. Buscar si ya
   existe la pantalla o una parecida (índice de código y ramas abiertas). Capturar la
   pantalla actual a 1440 y 390 px con `scripts/capturar_pantallas.py` (destino y
   credencial por variables de entorno, nunca en Git) y ponerla junto a la maqueta.
2. **Lo común primero.** Si falta un token, va a `tema-vec.css`; si falta un patrón, a
   `portal-componentes.css` o `portal-patrones.css`. Después, lo propio de la pantalla,
   sin repetir CSS estructural.
3. **Textos** en los catálogos `es` y `en` a la vez.
4. **Versiones.** Las referencias usan `?v=VEC_WEB_VERSION`: el servidor calcula
   y publica la versión común. Al tocar un CSS o JS no se editan fechas ni cadenas
   de versión en sus importadores o en las pruebas.
5. **Después.** Capturar otra vez (`--local web/static`) y comparar con la maqueta y con el
   «antes». Recorrer con teclado, zoom 200 %, alto contraste, en castellano e inglés, con
   Playwright y el Chrome del sistema. Para cada tarea clave, anotar pasos, clics, dudas y
   textos que no se entienden.
6. **Revisión independiente** de usabilidad (agente `revisor-usabilidad-vec`) antes de
   fusionar, con la lista de la sección 16. La PR lleva capturas de antes y después y la
   descripción del cambio en lenguaje llano.

## 16. Lista de comprobación antes de dar por buena una pantalla

Es la que aplica el revisor de usabilidad. Cada fallo se marca «bloquea», «molesta» o
«mejora». Un «bloquea» impide fusionar.

**Recorrido del novato**

- [ ] Una persona sin formación hace la tarea sin ayuda y en tres clics o menos desde la
      portada.
- [ ] El título y la miga dicen dónde se está; «Volver» regresa con los filtros.
- [ ] En una ficha se ven, en este orden, el siguiente paso (qué, quién, hasta cuándo), la
      fase y lo que falta.
- [ ] Lo pendiente aparece antes que la información.
- [ ] Heurísticos de Nielsen repasados: estado visible, lenguaje del trabajo, control y
      deshacer, coherencia, prevención de errores, reconocer mejor que recordar,
      eficiencia para quien repite, diseño mínimo, errores que se arreglan, ayuda.

**Contenido y textos**

- [ ] Ningún código interno, sigla de proyecto, número de duda, identificador técnico,
      código HTTP ni texto de funcionamiento interno a la vista.
- [ ] Ningún párrafo de ayuda en la pantalla; el «?» está en la cabecera y abre la ayuda
      de esta pantalla.
- [ ] Ninguna marca de prueba, ejemplo o demostración en lo nuevo (pantalla, PDF, correo).
- [ ] Ningún cartel sin sentido ni franja de «cargado» o «actualizado».
- [ ] Nada afirma firma, envío, entrega o registro sin respuesta real.
- [ ] Botones con verbo; una acción principal por bloque.
- [ ] Los errores dicen qué pasó y qué hacer; en formularios hay resumen arriba y mensaje
      junto al campo.
- [ ] Los estados vacíos explican y ofrecen la acción.
- [ ] Vocabulario, fases y recuentos iguales que en el resto de pantallas.

**Datos**

- [ ] Toda cifra, recuento, estado o barra agrupada lleva a su lista filtrada, con el
      mismo número que esa lista.
- [ ] Contacto y documento enmascarados en listados; completo solo en la ficha y con
      permiso.
- [ ] Fechas, horas, números, moneda y plurales según el idioma, en hora de Madrid.
- [ ] Se distinguen cargando, vacío, error, sin permiso y solo lectura; un fallo en un
      bloque no deja la pantalla en blanco.

**Aspecto**

- [ ] Solo tokens `--portal-*`; ningún hexadecimal nuevo en el CSS del módulo.
- [ ] Se reutilizan los componentes de la sección 5; no hay uno paralelo.
- [ ] Letra de la escala (26, 18, 15, 14 y 13 px; nada por debajo de 13 px en lo nuevo),
      pesos 400 y 600.
- [ ] Una pastilla por fila como máximo; números fuera de pastillas.
- [ ] Bordes alineados y espaciado de la escala.

**Adaptación y accesibilidad**

- [ ] Sin desbordamiento horizontal a 1440, 1024 y 390 px ni con zoom al 200 %; tablas
      anchas con scroll propio o como fichas.
- [ ] Todo se usa con teclado, en orden lógico, con foco visible y no tapado.
- [ ] Contraste AA en tema claro, oscuro y alto contraste; nada solo por color.
- [ ] Encabezados en orden, regiones, tablas con cabeceras, etiquetas asociadas y avisos
      por `aria-live`.
- [ ] Objetivos táctiles de 24 × 24 px como mínimo.
- [ ] El atributo `lang` cambia con el idioma; todos los textos tienen clave en `es` y `en`.

**Técnica de la pantalla**

- [ ] Cero errores de JavaScript en consola y ninguna llamada nueva a `/api` con respuesta
      400 o superior.
- [ ] Sin cookies ni almacenamiento del navegador como sesión o autoridad.
- [ ] Pantalla útil en menos de un segundo con volumen real; listas paginadas.
- [ ] HTML, importadores y caché usan la misma versión calculada de CSS y JS.
- [ ] Capturas de antes y después en la PR.

## 17. Huecos y diferencias entre reglas y código

Inventario histórico comprobado el 13/10/2026. Se conserva como diagnóstico
de aquella fecha; sus filas no afirman que cada hueco siga abierto al 20/10.
La cola vigente está en el [tablero](TABLERO_TAREAS.md). Cada corrección exige
su comprobación propia; este inventario no justifica rehacer pantallas que
funcionan.

| Hueco | Dónde | Regla |
| --- | --- | --- |
| Un segundo juego de variables de color (`--bg`, `--brand`, `--text`…) con unos 140 hexadecimales, cargado entre otras por `portal-empleado/index.html`, `bolsa/index.html` y `acceso/index.html` | `web/static/styles.css` | `tema-vec.css` es la única autoridad |
| Usa un token `--portal-error` que no existe; en alto contraste queda el rojo fijo de reserva | `web/static/portal-empleado/estado-entrega.css` | Usar `--portal-peligro` |
| Unas 33 reglas ponen letra por debajo de 12 px (hasta 0,6 rem) | `web/static/portal-empleado/*.css` | Mínimo 13 px |
| 20 y 32 px no tienen token de espacio | `tema-vec.css` | Escala de la sección 4.3 |
| Tamaño de texto y alto contraste son botones sueltos de la cabecera | `web/static/portal-empleado/index.html` | Dentro del menú de usuario |
| `font-family` nombra Inter sin incluir su fichero | `portal-empleado/portal.css` | Una sola familia, la del tema |
| Catálogo de temas fijo en el código (`TEMAS_VEC`, `MODOS_COLOR`) | `web/static/comun/tema-vec.js` | Temas instalables (cortes 3 y 4 pendientes) |
| PDF sin etiquetado y con título en castellano en el código | `internal/vec/adapters/documentos/pdf/renderizador.go` | Documentos accesibles, sin texto en el código |
| Localización fija `es-ES` al formatear fechas o importes | `web/static/comun/oportunidades/vista.js`, `web/static/portal-empleado/peticiones-centro/peticiones-centro.js` | Localización del idioma activo |
| La ficha de RRHH no muestra todavía el contacto completo | Pantallas de Bolsa | Sección 10 |
| Recibos con clases distintas por módulo (`ct-recibo`, `pc-recibo`, `rrhh-plantillas-recibo`…) | Varios | Un componente común de recibo |
| `pasos-numerados`, `tareas-pendientes` y `confirmacion-tramite` sin uso | `portal-patrones.css` | Usarlos en las pantallas de la sección 11 |

Contradicciones entre órdenes, ya resueltas aquí por la más reciente:

| Orden antigua | Orden posterior | Qué se aplica |
| --- | --- | --- |
| `AGENTS.md` (06/09/2026): toda sustitución explica en pantalla qué se hizo y qué falta para uso real; componente `estado-entrega` («nivel de entrega de una vista») | 12/10/2026: ningún rastro de prueba, aunque sea cierto | En lo nuevo no se muestran limitaciones de desarrollo; sí los estados reales del circuito |
| `ESPECIFICACIONES_AGENTES.md` E08 (19/09/2026): posponer retoques estéticos menores hasta la valoración de RRHH | 07/10/2026: los fallos pequeños de diseño se arreglan antes que funciones nuevas | Principio 15 |
| `disenar-sistema-visual-vec`: una acción sin caso de uso queda deshabilitada y explica la dependencia | 13/10/2026: terminar, no ocultar; sin carteles sin sentido | Botón deshabilitado con el motivo en una frase mientras se termina; nunca un cartel genérico |

Sin decidir, y dónde se decide:

- Tipografía institucional, paleta definitiva y uso del logotipo: `sistema_diseno_y_temas.md` §14.
- Vía para generar PDF etiquetados: se anotará en `PLAN_IMPLEMENTACION.md`.
- Formato definitivo del enmascarado: persona delegada de protección de datos.

### Historia: diagnóstico del 29/09/2026

La dirección de diseño se hizo mirando el portal de `origin/main` (`4a532005`) a 1440 y
390 px con datos sintéticos. Nota de conjunto con los heurísticos de Nielsen: 16 sobre 40.
Los diez problemas que la motivaron, por si alguno reaparece:

1. La portada no era un cuadro de mandos (solo peticiones; ni bolsas ni SAE ni tareas).
2. Los números no cuadraban entre portada y lista, con nombres distintos para lo mismo.
3. Dos vocabularios de fases (once nombres internos frente a las ocho fases de RRHH).
4. La ficha no decía qué había que hacer: diecisiete datos del mismo tamaño y referencias
   internas.
5. Registrar la respuesta de un candidato pedía clave, versión, huella y un `.eml`
   obligatorio.
6. Historial plegado y sin autor; sin lista de documentos.
7. La lista no ayudaba a priorizar (orden por número, columna cortada, filtros con
   «Aplicar»).
8. El menú cambiaba según el módulo y había botones que abrían otra pestaña.
9. Ruido del sistema («Expediente cargado.», «Servicio preparado…») que empujaba lo
   importante hacia abajo.
10. La bolsa se leía mal: tabla cortada, abreviaturas, horas sin sentido y un botón
    apagado sin decir por qué.

Se conservó: marca y menú azul marino, tarjetas blancas sobre gris, tablas con cabeceras
marcadas, el formulario de nueva petición por bloques con revisión, la línea de fases y
las fechas en hora de Madrid. Los prototipos no se recorrieron con teclado real, zoom,
lector de pantalla ni alto contraste, y usaron una API simulada.

## 18. Ficheros absorbidos

Lo vigente de estas fuentes está ahora en este documento; pueden retirarse o dejarse
remitiendo aquí.

| Fuente | Absorción | Qué queda fuera |
| --- | --- | --- |
| `docs/diseno/direccion-20260929/DIRECCION_DISENO.md` | Completa | Nada vigente. Las carpetas `prototipos/` y `capturas-antes/` se conservan como material; las herramientas de captura que citaba (`herramientas/`, `/dev/shm/vec_diseno_ro`) no están en el repositorio |
| `docs/diseno/aspirantes_modulo_2026-09-29.md` | Parcial: enmascarado, pantalla «Perfil y contacto», formatos que valida el formulario | Paquetes Go, identificadores, cifrado, permisos, SQL, API y seguridad: siguen en ese fichero |
| `AGENTS.md`, sección «i18n, idioma y presentación» | Completa (secciones 4, 8 y 9) | Nada |
| `ESPECIFICACIONES_AGENTES.md` E08 | Completa | Nada |
| `ESPECIFICACIONES_AGENTES.md` E02 | Parcial: lo que se ve en pantalla (principios 5 y 12, secciones 13.2 y 13.3) | La regla de negocio de requisitos, méritos y titulaciones, que va a `PRD.md` y `FLUJO_APP.md` |
| `ESPECIFICACIONES_AGENTES.md` E04, E05, E07 y E12 | Parcial: el menú no es permiso, sin almacenamiento del navegador ni tokens en la dirección, revisión visual e i18n en web | Autorización, fronteras, secretos y aceptación, que van a `TRD.md` y `ESQUEMA_BACKEND.md` |

## 19. Cómo mantener este documento

- Lo actualiza quien cambie un patrón visual, un token, un componente común o una regla de
  pantalla, en la misma PR (skill `documentos-producto-vec`). Basta una fila o una línea.
- Las reglas viven aquí; las skills describen cómo trabajar. Si una skill repite una
  regla, la skill remite a este documento y no la redacta de nuevo.
- La sección 17 se repasa al cerrar cada tanda de pantallas: se quitan los huecos
  resueltos y se añaden los nuevos, con su fichero.
- Las decisiones de diseño se añaden con su fecha de vigencia en la sección 2
  o en la que corresponda, y la orden antigua que contradigan pasa a la tabla de
  contradicciones de la sección 17.
- El texto se repasa con `humanizer` antes de fusionar. La revisión de integración comprueba que
  no contradiga a `PRD.md` ni a `FLUJO_APP.md`.
