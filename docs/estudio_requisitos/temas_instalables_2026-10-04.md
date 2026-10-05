# Temas instalables de VEC

Inventario sobre `main@296f78373f4874b9a81def9df744df12c1096af1`, 4 de octubre de 2026. El encargo reactiva la instalación desde Administración, aplazada el 30 de septiembre. Este documento fija el primer formato; todavía no acredita un instalador ni cambios en la base principal.

## Qué necesita quien administra VEC

El administrador de aplicación debe poder instalar un paquete de color, comprobar cómo queda, activarlo o retirarlo. Cada cambio conserva versión, huella, quién lo hizo y su resultado. Añadir un color no debe exigir recompilar ni ejecutar una migración.

El tema común sigue controlando componentes, estructura, tipografía, tamaño del texto, estados y accesibilidad. Los paquetes solo aportan colores para las variantes clara y oscura. El alto contraste y los colores forzados del dispositivo tienen prioridad. Cambiar el tema no modifica documentos emitidos, permisos ni datos de RRHH.

## Qué existe

| Pieza | Ruta | Función y límite actual |
| --- | --- | --- |
| Tokens comunes | `web/static/comun/tema-vec.css` | Colores base, granate, modo oscuro, seis paletas y alto contraste. Los colores están en CSS. |
| Controladores de tema | `web/static/comun/tema-vec.js` | Vista previa y aplicación sin almacenamiento web. Catálogo fijo institucional/granate y lista fija de nueve opciones de color. |
| Preferencias del empleado | `web/static/portal-empleado/portal-preferencias-api.js` | Consulta y guardado de preferencias propias. La validación limita los temas admitidos. |
| Preferencias del área personal | `web/static/area-personal/preferencias.js` y `cliente-http.js` | Segundo consumidor; opciones y versiones de catálogo cerradas. |
| Dominio de Usuarios | `internal/modules/usuarios/domain/preferencias.go` | `CatalogoPreferencias` y `ValidarValores`. El catálogo solo puede reducir el vocabulario compilado. |
| Catálogo durable de preferencias | `deploy/postgresql/usuarios_vec/migraciones/000017_temas_preferencias_v2.up.sql` | Publica la versión 2 y amplía `valores_validos` a seis colores nuevos. No instala paquetes. No se reaplica ni revierte. |
| Nombres traducidos | `web/static/textos/{es,en}/preferencias.json` | Nombres de las opciones actuales. Un paquete nuevo necesitará su propio catálogo de nombres validado. |
| Ensayo de color | `web/static/comun/codexf-temas-contraste.test.mjs` | Comprueba pares de texto, navegación, botones, estados y foco. |
| Galería histórica | `web/static/presentacion/temas/`, citada por la skill visual | No existe en el commit inventariado. La referencia histórica no acredita un catálogo instalado. |

`aspecto-vec` aplica la maqueta de RRHH: fondo tintado, paneles, cabeceras, tablas y estados reconocibles. `disenar-sistema-visual-vec` conserva una gramática visual común. `usabilidad-vec` fija lenguaje claro, teclado y accesibilidad; Impeccable se aplica después de esas tres autoridades.

El [estudio del sistema visual](sistema_diseno_y_temas.md) contempla una extensión mayor. Este corte se limita a paquetes de color, conforme a la decisión del 30 de septiembre. La [fuente de la paleta corporativa](temas_colores_fuentes_2026-09-30.md) conserva el enlace al manual oficial de Diputación.

## Referencias públicas y decisión de encaje

| Aplicación | Qué aporta a la decisión |
| --- | --- |
| [Joomla: definición de plantilla](https://manual.joomla.org/docs/next/building-extensions/templates/template-details-file/) y [cambio de estilo](https://guide.joomla.org/user-manual/templates/template-tips-switching-templates) | Manifiesto y elección desde administración. Instalar y elegir el predeterminado son operaciones distintas. |
| [Drupal: instalar y elegir el tema predeterminado](https://www.drupal.org/docs/user_guide/en/extend-theme-install.html) | El catálogo instalado puede contener alternativas sin convertirlas todas en apariencia activa. |
| [Keycloak: temas](https://www.keycloak.org/ui-customization/themes) | Herencia del tema base, traducciones y recursos con huella. Su posibilidad de incluir plantillas y scripts queda fuera del formato de VEC. |
| [Power Apps: temas compartidos](https://learn.microsoft.com/power-apps/maker/canvas-apps/controls/modern-controls/modern-theming) | Datos de color aplicados a componentes comunes y prueba de accesibilidad de las personalizaciones. |

El encaje propuesto para VEC es un archivo JSON de datos, con una sola base visual y sin archivos ejecutables. Evita descomprimir rutas o instalar dependencias. La administración y las preferencias consumirán el mismo catálogo de paquetes aprobado.

Consenso de arquitectura del 4 de octubre: separar la gestión institucional en Administración y la preferencia personal en Usuarios. El primer consumidor será un validador local reutilizable, sin instalación ni activación. Esta decisión se ha contrastado de forma independiente sobre el commit del inventario.

## Formato mínimo del paquete

El archivo contiene estos campos, obligatorios y sin campos desconocidos:

| Campo | Contenido |
| --- | --- |
| `esquema` | Identificador de la versión del formato de paquete. |
| `tema_id` | Identificador estable, sin espacios ni rutas. No concede autoridad. |
| `version` | Número entero positivo. La misma pareja tema/versión siempre designa los mismos bytes normalizados. |
| `sistema_diseno_ref` | Referencia exacta al contrato visual compatible. |
| `politica_ref` | Versión de la política de validación usada. |
| `nombre_key` | Clave de texto del paquete. |
| `textos` | Catálogos por idioma, con el nombre del tema y únicamente las claves del paquete. |
| `variantes` | Paleta `clara` y paleta `oscura`, con todos los colores requeridos por el contrato. |

La política visual es un dato versionado de la plataforma. Define tokens de color permitidos, idiomas requeridos, límites y pares de contraste. El paquete no puede modificarla. El contrato puede fijar tokens protegidos, como la superficie del logotipo, y obliga a conservar el significado de los estados.

Cada color se expresa como RGB opaco `#rrggbb`. Quedan excluidos HTML, JavaScript, CSS libre, selectores, URLs, fuentes, imágenes, expresiones, referencias a variables y transparencia. Los nombres son texto plano y se muestran con el traductor común y escape de contenido. No se permite que un paquete sobrescriba las traducciones de otro módulo.

La huella se calcula sobre la representación canónica del paquete validado y se entrega aparte. No se incluye una huella autorreferente dentro de los bytes que se resumen. La representación canónica, la política aplicada y el informe calculado de contraste forman parte del material conservado. Una huella comprueba integridad; la aprobación procede de la autoridad administrativa.

## Validación

1. Limitar bytes antes de reservar memoria; rechazar profundidad, claves duplicadas, campos desconocidos, tipos incorrectos y contenido posterior al JSON.
2. Comprobar referencia de formato y compatibilidad exacta con la política visual de la plataforma. Los límites técnicos máximos siguen siendo invariantes del lector.
3. Exigir identificadores y claves de traducción válidos; nombres breves sin controles, marcado ni sustituciones interpretables. Validar los idiomas desde la política.
4. Exigir exactamente los tokens autorizados y las dos paletas completas. Resolver todo el color antes de generar CSS; no aceptar CSS como entrada.
5. Calcular contraste en las dos paletas: texto normal al menos 4,5:1; foco y controles al menos 3:1. Comprobar todos los pares usados por los componentes, incluidos lateral, enlaces, selección, botones y estados. El informe aportado por un paquete no sirve de validación.
6. Conservar bytes canónicos y huellas. La instalación vuelve a validar en el servidor; una validación previa en el navegador no autoriza ni sustituye esa comprobación.

Las comprobaciones de contraste son necesarias, pero no acreditan por sí solas accesibilidad completa. La activación exige revisión independiente con componentes reales: teclado, foco, español e inglés, escritorio, móvil, zoom, colores forzados y alto contraste.

## Propiedad y contratos

La autoridad visual es común a VEC. El contrato de paquetes se implementará en `internal/modules/administracion/domain/temas`; la lectura estricta de JSON, en un adaptador separado. Administración ofrecerá el recorrido de gestión en su superficie real, compuesta en `internal/app/administracion`, usando el perfil fijo de administrador de aplicación. Actualmente sus proveedores de lectura y selección no constituyen un instalador operativo; el panel del portal empleado tampoco lo sustituye. El montaje actual de perfiles y su lista positiva de activos no sirven `/admin/modulos/` ni `tema-vec.js`; la futura ruta se incorporará expresamente a la frontera con pruebas. Usuarios conserva las preferencias de cada persona y consume una instantánea aprobada por un puerto; no consulta tablas ajenas.

El catálogo público de color no contiene identidad, preferencias personales ni auditoría. Solo expone paquetes disponibles, versiones, huellas y nombres. Las lecturas de administración y de preferencias mantienen su autorización y auditoría nominal. Una lista de colores no permite consultar quién los usa.

La composición debe inyectar los adaptadores de catálogo, validación, autorización y auditoría comunes. No se crea otro login ni una concesión al recibir la petición. Las acciones exactas de instalar, aprobar, activar y retirar deben publicarse y provisionarse por el circuito central de perfiles fijos, huella y versión esperada antes de habilitar las rutas.

## Instalar, activar y retirar

Instalar conserva una versión inmutable validada. Instalar de nuevo los mismos bytes con la misma clave recupera el recibo; cambiar los bytes exige otra versión o produce conflicto. Instalar no activa el tema.

La activación recibe la revisión y huella esperadas del catálogo vigente, la referencia exacta al paquete y una clave de operación. En una transacción se revalidan la autorización positiva, la disponibilidad del paquete, la aprobación y la comparación con el estado esperado. Se escriben publicación, historia y auditoría común. Si otra sesión cambió el catálogo, la operación se rechaza y la pantalla ofrece consultar de nuevo. La respuesta conserva recibo, revisión resultante y fecha.

Los efectos permitidos se auditan en la misma transacción. Las denegaciones y los errores usan el contrato común de intentos nominales después de cerrar la operación original; ese registrador no sustituye la auditoría atómica de un éxito. Cada operación usa una nueva auditoría nominal. Un reintento recupera el efecto original y registra la consulta o el intento correspondiente según el contrato común. Un fallo de auditoría impide confirmar el cambio. Los errores de resultado indeterminado conservan la clave original para comprobar el resultado.

Retirar impide nuevas selecciones y conserva todas las versiones e historia. Si el tema estaba en uso, la resolución visual vuelve al predeterminado aprobado y muestra un aviso traducido; no reescribe masivamente preferencias históricas. Cambiar el predeterminado y retirar el anterior exige una publicación coherente con comparación de versión y huella. El tema base de recuperación permanece disponible.

La desinstalación elimina la disponibilidad para uso futuro. No borra paquetes, aprobaciones, recibos ni auditoría conservados como evidencia. Las preferencias antiguas siguen siendo válidas como historia; guardar una elección nueva exige un tema disponible en la publicación actual.

## Consumo web y recorrido administrativo

En Administración se muestran nombre, versión y estado, con acciones «Instalar tema», «Ver vista previa», «Activar» y «Retirar». La huella y la revisión quedan detrás de «Detalle técnico». La vista previa avisa que todavía no está activa y ofrece cancelar. La confirmación explica qué cambió y conserva el recibo.

El servidor genera la hoja de colores desde tokens validados, con URL propia identificada por huella. El cliente carga una publicación completa compatible antes de aplicarla. El CSS común conserva la estructura y las capas de accesibilidad. La superficie ADMIN exige `style-src 'self'`; el recorrido usa un enlace a CSS propio, sin estilos en atributos. El portal hoy admite estilos en línea, pero eso no es requisito ni permiso del nuevo formato. No se aceptan URLs suministradas por el paquete ni estilos libres. Las versiones `?v=` del grafo de código siguen siendo coherentes; los colores usan la huella de su contenido.

El tema y los modos clara/oscura/dispositivo son conceptos separados. Se necesita una transición explícita para las preferencias existentes: las nueve opciones actuales no se reinterpretan silenciosamente. El contrato futuro de preferencias separará `tema_ref` de `modo_color`, manteniendo lectura y recuperación de las versiones 1 y 2. La selección nueva cotejará la publicación vigente dentro de la transacción mediante una fachada propietaria; una consulta previa en Go no cierra la carrera contra una retirada. Alto contraste y tamaño de texto permanecen independientes. La lectura conserva la preferencia original y presenta la alternativa efectiva cuando un paquete se haya retirado.

## Cortes de implementación

| Orden | Resultado usable | Condición de entrega |
| --- | --- | --- |
| 1 | Inventario y diseño mínimo de este documento | Rutas cotejadas con el commit, comparadores públicos y consenso independiente. |
| 2 | Preparar y comprobar un paquete local por línea de comandos | `cmd/vec-temas-validar`, validador reutilizable, contrato de política, paquetes de ejemplo y salida con huella e informe. No instala ni activa. |
| 3 | Instalar y consultar paquetes desde Administración | Autoridad positiva fija, almacenamiento durable y auditoría común atómica; ensayo SQL y revisión exacta. |
| 4 | Aprobar, activar y retirar con versión esperada | Publicación CAS, recibos, recuperación y concurrencia; ninguna escritura parcial. |
| 5 | Elegir paquetes disponibles en Preferencias | Adaptadores Go/SQL/JS consumen catálogo; se conserva la historia antigua y la accesibilidad. |

Cada pieza abre su propia PR y se prueba en su alcance. Las rutas permanecen cerradas hasta tener autoridad y consumidor reales. Dirección revisa, integra y despliega.

## Verificaciones de cierre

Paquetes válidos claro/oscuro y terceros sin cambiar código; rechazo de color con poco contraste, token estructural, URL, clave duplicada y exceso de tamaño. Huella estable ante orden distinto de claves; cambio de contenido con la misma versión rechazado al instalar.

En persistencia: autorización exacta, CAS obsoleto, dos activaciones concurrentes, reintento tras resultado indeterminado, fallo de auditoría con rollback, recuperación tras reinicio y conservación de versiones anteriores. En web: foco y borradores conservados, respuestas tardías descartadas, retorno al predeterminado al retirar, alto contraste y colores forzados prioritarios.
