# Preferencias de modalidad de ofertas (F1-13)

Esta capacidad permite que la aspirante elija qué modalidades de **ofertas publicadas** quiere ver y sobre cuáles recibir avisos. La elección no cambia su inscripción, el orden de la bolsa, la selección ni los llamamientos directos. Tampoco cuenta como renuncia. Sin selección guardada (`null`) se muestran todas las modalidades; una selección guardada vacía (`[]`) oculta las ofertas abiertas sin respuesta que tienen una modalidad identificada. Conserva las ofertas con respuesta y el historial; «Ver todas» permite consultarlas temporalmente sin guardar otra elección.

## Entregas previstas

1. **Base de permisos y persistencia (PR1).** AUT104 adapta la lectura propia a la proyección de orden existente, de ocho campos, y a la nueva consulta de modalidades, de cinco. B138 guarda por separado las versiones y recibos de la elección y conserva la lectura anterior. La fuente propuesta para el catálogo es [`bolsa_preferencias_bolsas_propias_v2.json`](../../fuentes/bolsa_preferencias_bolsas_propias_v2.json). Esta base todavía no ofrece una función completa a la persona.
2. **Servicio y persistencia (PR2).** Conecta el catálogo vigente con la elección, su huella y el repositorio nominal. Conserva la selección vacía y la opción de todas, la versión y el recibo.
3. **API y composición (PR3).** El servidor debe conectar `GET/PUT /api/vec/bolsa/mi-bolsa/preferencias?tipo=modalidades` con la sesión, la concesión vigente y las funciones nominales. Debe devolver versión y recibo, y recuperar el mismo recibo al repetir una clave con la misma elección. El orden de bolsas conserva su consulta y guardado actuales.
4. **Vista y avisos (PR4).** La elección se aplica a la vista de ofertas publicadas y a los avisos solicitados por la persona. La comprobación debe demostrar que el orden de llamamiento y la inscripción no cambian. La pantalla para elegir modalidades, el aviso por selección vacía y «Ver todas» pertenecen al recorrido de F1-14.

PR1 puede entrar en main sola: instalada, no cambia lo que ve la aspirante mientras no se publique la versión del rol con los diez campos, y la API de modalidades llega en PR3. AUT104 y B138 se instalan juntas, en este orden y después de AD296, como fija la lista única. PR4 se comprueba después con la vista, los avisos y el recorrido en Chrome.

## Fuente y gobierno de la concesión

La fuente v2 conserva las dos acciones propias de preferencias y amplía los campos concedidos a la lectura. AUT104 admite que el GET anterior pida sus ocho campos y que el nuevo pida exactamente los cinco de modalidades; ninguno obtiene la proyección de la otra consulta por añadir campos en la petición. La fuente declara 90 días de vigencia: desde el **18 de octubre de 2026 a las 00:00 UTC** hasta antes del **16 de enero de 2027 a las 00:00 UTC**. Su huella de contenido canónico es `8241bbef088f2b88ddbb966a2ebfb4b435e116b3e81597ab8a0c299d1302a36f`; la huella SHA-256 del archivo es `a4daccf0f62ac5de9eb508b3e5ff818159353d5d9e0351b6dabfb13709dea991`.

El archivo en Git es una propuesta, no una concesión publicada. Para activarla, dirección debe comprobar la versión y el catálogo vigentes, admitir la fuente por el acto de gobierno del catálogo y publicar la nueva versión del rol por la propuesta y el cierre de dos administradores que ya usa el producto. La guía anterior de [preferencias entre bolsas](../../habilitacion_preferencias_bolsas_20261013.md) describe esas vías para la fuente v1; sus datos, versiones y acuses no se reutilizan como si acreditaran la v2. Antes de ejecutar un acto nuevo hay que comprobar que esas vías aceptan la nueva versión y sus dos campos adicionales. Si falta esa comprobación, la publicación del rol queda pendiente. La activación no se hace escribiendo filas privadas por SQL.

Las modalidades disponibles deben salir del catálogo gobernado que corresponde a la oferta y a su versión, no de nombres enviados por el navegador. B138 exige una instantánea de esas opciones al guardar y rechaza claves ausentes de ella. PR2 debe resolver ese catálogo; PR4 debe usarlo al filtrar vista y avisos. F1-14 presentará las opciones sin inventarlas cuando falte el catálogo.

## Estado y comprobaciones pendientes (19/10/2026)

Ensayo en copia local con la lista única completa (101 SQL; AUT104 y B138 al final, después de AD296 y AD291). Las dos se instalaron y sus pruebas focales pasaron. La versión 1 del rol candidato instalada se reconoce como base, así que se pueden seguir publicando versiones nuevas del rol. Con B138 instalada la aplicación arranca y Mi Bolsa responde con el usuario candidato, también después de reiniciar aplicación y PostgreSQL. El orden de bolsas sigue denegado a ese candidato porque su rol no tiene la concesión de preferencias, igual que en main.

El ensayo sacó dos fallos, ya corregidos en esta PR:

- AUT104 exigía un orden de concesiones que no coincide con la versión 1 publicada. Eso impedía cualquier versión nueva del rol. Ahora compara las siete concesiones de la versión 1 como conjunto exacto.
- B138 concede la escritura de modalidades al grupo del portal externo, y el arranque de Mi Bolsa exige la lista exacta de funciones de ese grupo. Sin ajustar ese cotejo, la aplicación no arrancaba. Ahora admite esa función solo con su firma exacta, sin acceso público y junto a las tres funciones de preferencias.

Antes de activar las modalidades (PR3) hay que resolver:

1. La decisión emitida para guardar modalidades comparte acción y audiencia con el guardado del orden de bolsas, y las funciones de ese guardado (B104 y B116) no recalculan la huella del contexto. Una migración nueva debe hacer que las dos la recalculen, o separar las audiencias.
2. La fila de auditoría de la lectura debe decir qué proyección se leyó (ocho o cinco campos), con un motivo propio o una columna.
3. PR2 debe sacar las modalidades disponibles del catálogo gobernado; SQL solo comprueba su forma. Las lecturas de cinco campos devuelven `modalidades_disponibles` vacío y PR2 o PR3 deben rellenarlo.
4. La fuente v2 vence el 16 de enero de 2027, antes que la v1. Si esa fecha fija la vigencia de la versión del rol, todas las acciones de Mi Bolsa vencerían ese día; hay que confirmarlo al publicar.

Sigue sin acreditar: la fuente v2 admitida, la versión del rol publicada, el GET/PUT de modalidades con recibo y reinicio, y el recorrido en Chrome. Cuando lleguen, el ensayo debe comprobar con datos ficticios que `null` muestra todas las modalidades, que `[]` oculta las ofertas abiertas sin respuesta y con modalidad identificada, que una clave desconocida se rechaza y que una repetición conserva versión, recibo y fecha tras reiniciar. Después hay que contrastar la vista y los avisos con las ofertas publicadas, sin alterar la inscripción, la posición en bolsa ni el llamamiento directo. Un fallo de configuración debe cerrar solo esta función y dejar disponible el resto de Mi Bolsa.
