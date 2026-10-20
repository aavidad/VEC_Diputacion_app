# Preferencias de modalidad de ofertas (F1-13)

Esta capacidad permite que la aspirante elija qué modalidades de **ofertas publicadas** quiere ver y sobre cuáles recibir avisos. La elección no cambia su inscripción, el orden de la bolsa, la selección ni los llamamientos directos. Tampoco cuenta como renuncia. Sin selección guardada (`null`) se muestran todas las modalidades; una selección guardada vacía (`[]`) oculta las ofertas abiertas sin respuesta que tienen una modalidad identificada. Conserva las ofertas con respuesta y el historial; «Ver todas» permite consultarlas temporalmente sin guardar otra elección.

## Entregas previstas

1. **Base de permisos y persistencia (PR1).** AUT104 adapta la lectura propia a la proyección de orden existente, de ocho campos, y a la nueva consulta de modalidades, de cinco. B138 guarda por separado las versiones y recibos de la elección y conserva la lectura anterior. La fuente propuesta para el catálogo es [`bolsa_preferencias_bolsas_propias_v2.json`](../../fuentes/bolsa_preferencias_bolsas_propias_v2.json). Esta base todavía no ofrece una función completa a la persona.
2. **Servicio y persistencia (PR2).** Conecta el catálogo vigente con la elección, su huella y el repositorio nominal. Conserva la selección vacía y la opción de todas, la versión y el recibo.
3. **API y composición (PR3).** El servidor debe conectar `GET/PUT /api/vec/bolsa/mi-bolsa/preferencias?tipo=modalidades` con la sesión, la concesión vigente y las funciones nominales. Debe devolver versión y recibo, y recuperar el mismo recibo al repetir una clave con la misma elección. El orden de bolsas conserva su consulta y guardado actuales.
4. **Vista y avisos (PR4).** La elección se aplica a la vista de ofertas publicadas y a los avisos solicitados por la persona. La comprobación debe demostrar que el orden de llamamiento y la inscripción no cambian. La pantalla para elegir modalidades, el aviso por selección vacía y «Ver todas» pertenecen al recorrido de F1-14.

PR1 puede entrar en main sola: instalada, no cambia lo que ve la aspirante mientras no se publique la versión del rol con los diez campos, y la API de modalidades llega en PR3. AUT104 y B138 se instalan juntas, en este orden y después de AD296, como fija la lista única. PR4 se comprueba después con la vista, los avisos y el recorrido en Chrome.

## Fuente y gobierno de la concesión

La fuente v2 conserva las dos acciones propias de preferencias y amplía los campos concedidos a la lectura. AUT104 admite que el GET anterior pida sus ocho campos y que el nuevo pida exactamente los cinco de modalidades; ninguno obtiene la proyección de la otra consulta por añadir campos en la petición. La fuente es vigente desde el **18 de octubre de 2026 a las 00:00 UTC** hasta antes del **1 de octubre de 2027 a las 00:00 UTC**, la misma fecha final que la v1 (ver la parte 3). Su huella de contenido canónico es `98c6206192248dd23a4bf58aa2753808fc6ecb6f302cbf02d70da4abde3410f4`; la huella SHA-256 del archivo es `8924fab53d1f9b0a835bbe0ae4cfac1cb43086be40f667fdc3daae8fdc78c5b0`.

El archivo en Git es una propuesta, no una concesión publicada. Para activarla, dirección debe comprobar la versión y el catálogo vigentes, admitir la fuente por el acto de gobierno del catálogo y publicar la nueva versión del rol por la propuesta y el cierre de dos administradores que ya usa el producto. La guía anterior de [preferencias entre bolsas](../../habilitacion_preferencias_bolsas_20261013.md) describe esas vías para la fuente v1; sus datos, versiones y acuses no se reutilizan como si acreditaran la v2. Antes de ejecutar un acto nuevo hay que comprobar que esas vías aceptan la nueva versión y sus dos campos adicionales. Si falta esa comprobación, la publicación del rol queda pendiente. La activación no se hace escribiendo filas privadas por SQL.

Las modalidades disponibles deben salir del catálogo gobernado que corresponde a la oferta y a su versión, no de nombres enviados por el navegador. B138 exige una instantánea de esas opciones al guardar y rechaza claves ausentes de ella. PR2 debe resolver ese catálogo; PR4 debe usarlo al filtrar vista y avisos. F1-14 presentará las opciones sin inventarlas cuando falte el catálogo.

## Estado y comprobaciones pendientes (19/10/2026)

Ensayo en copia local con la lista única completa (101 SQL; AUT104 y B138 al final, después de AD296 y AD291). Las dos se instalaron y sus pruebas focales pasaron. La versión 1 del rol candidato instalada se reconoce como base, así que se pueden seguir publicando versiones nuevas del rol. Con B138 instalada la aplicación arranca (copia con `--sin-inscripcion`) y Mi Bolsa responde con el usuario candidato, también después de reiniciar aplicación y PostgreSQL. El orden de bolsas sigue denegado a ese candidato porque su rol no tiene la concesión de preferencias, igual que en main.

El ensayo sacó dos fallos, ya corregidos en esta PR:

- AUT104 exigía un orden de concesiones que no coincide con la versión 1 publicada. Eso impedía cualquier versión nueva del rol. Ahora compara las siete concesiones de la versión 1 como conjunto exacto.
- B138 concede la escritura de modalidades al grupo del portal externo, y el arranque de Mi Bolsa exige la lista exacta de funciones de ese grupo. Sin ajustar ese cotejo, la aplicación no arrancaba. Ahora admite esa función solo con su firma exacta, sin acceso público y junto a las tres funciones de preferencias.

## Parte 2: servicio y persistencia (19/10/2026)

El servidor ya tiene el servicio de guardado y el de lectura de modalidades, con sus adaptadores de PostgreSQL. Ninguno está montado todavía en una ruta: eso es la parte 3.

- Las modalidades disponibles salen del catálogo de reglas de Contratación temporal (`c12.modalidad.*`), con su versión y su huella. Si el catálogo no responde o no tiene modalidades, el servicio falla y no inventa opciones. La lectura rellena `modalidades_disponibles` con ese catálogo; SQL sigue devolviendo la lista vacía.
- El guardado calcula la huella de la selección y del catálogo antes de pedir la decisión, y la liga al recurso. `null` (todas) y `[]` (ninguna) dan huellas distintas. Repetir la misma clave devuelve el mismo recibo aunque una modalidad se haya retirado del catálogo después.
- B143 cierra el primer punto pendiente de la parte 1: el guardado del orden de bolsas (B104 y B116) vuelve a calcular la huella de su contexto fijo. Una decisión emitida para modalidades ya no sirve para cambiar el orden de bolsas.
- AD300 cierra el segundo: la fila de auditoría de una lectura lleva `lectura_preferencias_modalidades_correcta` cuando se leen los cinco campos y `lectura_preferencias_bolsas_correcta` cuando se leen los ocho. La fachada rechaza cualquier otro par. Las dos migraciones van al final de la lista única, después de B138.

La vigencia se resolvió en la parte 3: la fuente v2 termina el 1 de octubre de 2027, igual que la v1.

## Parte 3: API y composición (19/10/2026)

La ruta `/api/vec/bolsa/mi-bolsa/preferencias` atiende las modalidades solo con la consulta exacta `?tipo=modalidades`. Sin consulta sigue sirviendo el orden de bolsas. Cualquier otra consulta recibe 400 y otro método, 405.

- **GET** devuelve `version`, `modalidades_guardadas` (`null` si nunca se guardó, o la lista), `modalidades_disponibles` (clave y fuente del catálogo), `recibo_ref` y `registrado_en`.
- **PUT** recibe `version`, `clave_idempotencia` y `modalidades` (`null` para todas, `[]` para ninguna). Responde 201 con recibo y versión nueva, o 200 con `reutilizado` si repite la clave con la misma elección.
- Los errores salen como códigos fijos: 403 `acceso_denegado`, 409 `preferencias_desactualizadas`, 422 `orden_invalido`, 503 `servicio_no_disponible` y 500 `error_interno`. El texto interno del error nunca llega a la respuesta; una prueba lo comprueba con errores unidos.
- El arranque comprueba que B138 existe con su firma exacta, que el usuario de conexión puede ejecutarla, que no tiene acceso público y que la restricción de auditoría ampliada por AD300 está validada y su definición coincide exactamente (por huella SHA-256) con la que instala AD300. Si una migración futura cambia esa restricción, hay que actualizar la huella en el arranque. Comprueba también el catálogo de modalidades. Si algo falla, registra solo la etapa y la consulta de modalidades responde 503. El orden de bolsas no se ve afectado.
- La huella de la selección acepta el formato de candidato que usa B138 (`can_` y de 22 a 128 caracteres). Antes solo admitía la referencia de constitución, más corta, y rechazaba a los candidatos externos.
- Vigencia: la fuente v2 vence ahora el 1 de octubre de 2027, como la v1. Alargarla no toca ninguna autoridad ajena: la v2 aún no está admitida en ningún catálogo. Las pruebas exigen que la v2 dure al menos 60 días. Que no supere el catálogo que la admita lo garantiza la validación del dominio al admitirla.

### Evidencia en copia local

Copia propia con la lista única (`--sin-inscripcion` bastó). El ensayo SQL con pruebas restauró HZ37, instaló 118 migraciones y pasó 9 pruebas antes y 9 después del reinicio. En la copia, con la aplicación arrancada:

- Sin la concesión publicada, la consulta de modalidades responde 403 y no 503, así que está montada. Una consulta inválida da 400 y un método no admitido, 405.
- Con la restricción de AD300 vigente, la comprobación del arranque da verdadero; con la restricción anterior, falso (probado dentro de una transacción revertida).
- Con un rol de semilla de copia, que no es un acto gobernado: PUT `[]` da 201 con recibo y versión 1. Repetirlo da 200 y `reutilizado`. Otra elección con la misma clave da 422, una versión antigua 409 y una clave desconocida 422. PUT `null` da 201 con otro recibo y versión 2. Tras reiniciar aplicación y PostgreSQL, las mismas repeticiones devuelven los mismos recibos y fechas. Hay dos filas (`{}` y `NULL`) y ningún duplicado.

### Lo que falta para activarla

- **GET con el rol de semilla.** La lectura rechaza los roles de la semilla de copia por la guarda contra emisores de desarrollo, y esa guarda no se rebaja. El GET solo se podrá acreditar con la versión del rol publicada por actos gobernados.
- **Admitir la fuente v2 y publicar la versión del rol.** La copia no tiene todavía la cadena de administración necesaria (F0-28). Cuando la tenga, los pasos son: admitir la fuente v2 en un catálogo nuevo con `vec-catalogo-acciones-admitir` (AUT58 y AUT61), cuya vigencia cubra la de la v2; proponer la versión del rol candidato con los diez campos; y cerrarla con un segundo administrador. Después, repetir en la copia el GET y el PUT con recibo y reinicio usando el certificado de la aspirante.
- **Asimetría heredada.** La escritura de preferencias (B104 y B138) acepta una decisión emitida para un rol de semilla y la lectura no. Ya ocurría antes de esta parte; se deja anotada para la revisión de seguridad.
- **Orden de las claves en B138.** Sus `ORDER BY` usan la colación de la base y Go ordena por bytes. Con las ocho claves actuales coinciden; con otras, el guardado fallaría cerrado. Una migración futura debe usar `COLLATE "C"`.
- **Parte 4 (B139).** Hecha; ver el apartado siguiente.

Sigue sin acreditar: la fuente v2 admitida, la versión del rol publicada, el GET/PUT de modalidades con recibo y reinicio, y el recorrido en Chrome. Cuando lleguen, el ensayo debe comprobar con datos ficticios que `null` muestra todas las modalidades, que `[]` oculta las ofertas abiertas sin respuesta y con modalidad identificada, que una clave desconocida se rechaza y que una repetición conserva versión, recibo y fecha tras reiniciar. Después hay que contrastar la vista y los avisos con las ofertas publicadas, sin alterar la inscripción, la posición en bolsa ni el llamamiento directo. Un fallo de configuración debe cerrar solo esta función y dejar disponible el resto de Mi Bolsa.

## Parte 4: vista de ofertas y avisos (20/10/2026)

**Vista de Mi Bolsa.** Si la lectura de modalidades está montada, `GET /api/vec/bolsa/mi-bolsa` quita las ofertas abiertas sin respuesta de la persona cuya modalidad no eligió. Las ofertas sin modalidad, las ya respondidas y las resueltas se ven siempre. Con `null` o sin elección guardada se ven todas; con `[]` solo desaparecen las abiertas con modalidad. La respuesta lleva `ofertas_segun_modalidades: true` cuando la lista ya aplica la elección. La elección solo se lee si hay alguna oferta que pueda quitarse. Si la lectura falla (rol sin la concesión, función ausente o error), se devuelven todas las ofertas sin esa marca y el registro técnico solo anota `denegada`, `no_disponible` o `error`. Ocultar ofertas por un fallo haría perder a la persona ofertas a las que puede responder. Participaciones, posición, situación y llamamientos no cambian.

**Avisos (B139).** `crear_aviso_oferta_b129()` congela los destinatarios al publicar. Ahora excluye a quien tiene una elección guardada que no incluye la modalidad de la oferta; lee la última versión de `preferencia_modalidades_version` por `vinculo_candidato`. Una oferta sin modalidad llega a todos los disponibles, como antes. La cola, la reclamación y el reintento usan la lista congelada y no vuelven a leer la elección; un cambio posterior solo cuenta para ofertas nuevas. B139 coteja la huella del cuerpo instalado (`464f8f0c…`), propietario, configuración y permisos, y comprueba la huella final (`56206ed1…`). No crea funciones ni toca el núcleo. Solo hay UP y se instala al final de la lista única.

| Migración | SHA-256 del fichero |
| --- | --- |
| B139 `bolsa_llamamientos/migraciones/000139_avisos_modalidades.up.sql` | `ef4c0b36253f63017fbb0591e7c29d9091d0a5ee48b0b16e18ebf3433bc31879` |

La prueba `bolsa_llamamientos/pruebas_sql/b139_avisos_modalidades.sql` necesita una copia con semilla (dos personas disponibles con vínculo en una bolsa vigente) y lo revierte todo. Comprueba el filtro, que la lista anterior no cambia al cambiar la elección, que una oferta sin modalidad llega a todos, que `null` vuelve a incluir y que la reclamación usa solo la lista congelada.

### Evidencia en copia local

`ensayar.py --pruebas`: 120 migraciones, 10 pruebas SQL antes y después del reinicio. Copia propia con portal externo: B139 instalada con su huella final.

En la bolsa de Administrativo (25 disponibles) dos personas de la semilla recibieron una elección como dato de copia, sin acto gobernado: A solo «vacante» y B solo «sustitucion». Se publicaron tres ofertas con las dos guardas de política V3 desactivadas solo durante la inserción:

- «vacante»: 24 avisos, A incluida y B no.
- «sustitucion»: 24 avisos, B incluida y A no.
- sin modalidad: 25 avisos, las dos incluidas.

Después A pasó a «ninguna» y B a «todas». Las tres listas anteriores no cambiaron. Otra oferta «vacante» excluyó a A e incluyó a B. Dos reclamaciones de diez y, tras reiniciar aplicación y PostgreSQL, otra de cuatro: 24 avisos reclamados de 24, 24 identificadores de mensaje distintos y un intento cada uno. El orden vigente de la bolsa (posición y situación de cada participación en el mismo instante) dio la misma huella antes, después y tras el reinicio. Mi Bolsa respondió 200 tras el reinicio.

La vista filtrada no se pudo ver en la copia: la aspirante del portal no tiene participaciones y su lectura de modalidades sigue denegada (403) hasta publicar la versión del rol por actos gobernados. Las pruebas de la aplicación cubren la vista.

Ensayo: unas funciones de ayuda de la demostración con `EXECUTE` para PUBLIC impidieron montar Bolsa y Contratación al reiniciar. La guarda de arranque hizo lo que debía; al retirar ese permiso la aplicación arrancó.

Queda: la pantalla para elegir y «Ver todas» (F1-14), publicar la versión del rol y acreditar la vista con la aspirante en Chrome. El portal interno no tiene todavía consulta de modalidades.
