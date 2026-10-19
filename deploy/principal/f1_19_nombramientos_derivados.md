# Nombramientos por persona desde una oferta

F1-19, primer corte: contrato y operación de derivación y aceptación de plaza.
El panel y la composición de autorización pertenecen al segundo corte.
La instalación quedó ensayada el 19/10 (apartado final). El recorrido
positivo sigue pendiente, igual que la continuación del hijo por propuesta,
resolución, Personal y cese.

## Activación

Usar PostgreSQL 18 propio con main completo, incluida F1-16. Tras AD291/AD292
de R, instalar B135→CT234 una vez según la lista de este corte.
Cotejar antes cuerpos instalados, huellas, propietario y permisos. El guion
privado reproduce las cinco definiciones explícitas desde el volcado cotejado.
No preparar firmas ni recibos fingidos ni modificar la cadena de autorización.

La composición responde 503 hasta conectar la autoridad propia del hijo en
el segundo corte, con `identidad/nombramientos-ct.json`. No hereda permisos
del padre. Un 503 o un doble de prueba no acreditan la creación del hijo.

## Comprobación conjunta pendiente

1. Preparar la variante de dos personas del caso 4.0 y adjudicar sus plazas mediante actos de Bolsa, con aceptaciones y recibos propios de la copia.
2. Desde la sesión certificada de RRHH, enviar `POST /api/vec/contratacion-temporal/nombramientos/derivaciones` con solo `padre_ref`, `version_padre`, `oferta_ref`, `numero_plaza`, `secuencia_plaza`, `participacion_ref`, `recibo_adjudicacion_ref` y `clave_idempotencia`. Obtenerlos de las lecturas y recibos autorizados. El servidor resuelve organización, hijo y unidad con la lectura autorizada; SQL extrae la instrucción solo tras consumir el permiso y liga al recibo su preimagen original. Cotejar la persona y la plaza exactas del hijo, su aceptación y su recibo.
3. Repetir con la misma clave: debe recuperar el recibo original. Otra clave para esa plaza debe dar conflicto. Ante resultado incierto, conservar la clave y recuperar el intento; no crear otra operación.
4. Denegar otra unidad, perfil sin permiso, plaza sin aceptación, recibo de otra plaza y padre cancelado, sin efectos parciales.
5. Reiniciar aplicación y PostgreSQL y recuperar hijos y recibos; arrancar también sin la configuración opcional y restaurarla. El permiso del hijo siempre se verifica de nuevo.
6. En el corte siguiente, comprobar el panel, las lecturas autorizadas por hijo y su circuito completo: el cese de una persona no altera a la otra y el padre no puede cerrarse con hijos abiertos.

El cerrojo de la oferta precede a la transacción serializable y dura hasta su
fin; Bolsa revalida la plaza dentro de esa transacción. Guardar en el material
privado las huellas y pruebas de instalación, actos, navegador y reinicio.

## Ensayo del 19/10

Copia local nueva desde la cabeza unida a main, con la lista única completa:
102 SQL instaladas una vez y sin errores, con AD291, AD292, B135 y CT234 al final.
CT234 fallaba antes porque el propietario de Contratación temporal no puede
crear tablas temporales; ahora la tabla de cotejo la crea la sesión que
instala, antes de cambiar de rol, y solo cede al propietario leerla y
escribirla durante la transacción. Si la base se instala con un rol sin TEMP,
CT234 se detiene sin efectos.

Las pruebas `ad292_…`, `b135_…` y `ct234_…` pasaron con `ROLLBACK`. Con el
certificado de RRHH, la ruta responde 503 con un cuerpo válido, 400 con un
cuerpo vacío, 422 con `{}` y 405 a GET; sin certificado no se abre la conexión. Tras
reiniciar PostgreSQL y aplicación (3,6 s), las consultas de Bolsa y
Contratación temporal responden igual y no hay ningún hijo creado.

## Segundo corte, parte 1: guardas del hijo (CT241)

El hijo copia del padre la asignación, la fiscalización y el circuito, y queda
con una persona. Antes de abrir la ruta se cierran dos cosas.

Selección. El hijo ya tiene su plaza adjudicada y no entra en otra selección
de Bolsa. Go lo rechaza en `SeleccionarSegunNumeroPersonasCT` (409, estado no
seleccionable). En SQL lo rechazan con 23505 la reserva singular O6, la reserva
de oferta (sobre la postimagen de CT240) y el vínculo de emisión de CT201;
las dos últimas antes de consumir la autorización.

Autorización por campos copiados. Con 42501 se rechazan:

- la consulta del estado de la oferta (CT225) y la preparación de su
  contexto (CT227), que tomaban la unidad gestora copiada;
- la consulta del circuito, que devolvía el circuito copiado del padre;
- con un disparador común: toda versión nueva del hijo salvo la v6 de su
  creación, su publicación en la proyección RRHH y su firma de documentos
  CT118.

La proyección RRHH alimenta la bandeja, el detalle, los cambios, el resumen
y las estadísticas, y filtra por la unidad y el centro copiados. Además, el
canon de la bandeja no admite el sufijo `-P`: un hijo publicado hacía fallar
la página entera para todos los que comparten unidad con el padre. Por eso,
mientras el hijo no tenga ámbito propio, CT234 no puede completar una
derivación: la publicación de la v6 se rechaza y la operación se deshace
entera. El corte siguiente debe dar al hijo su ámbito (rol
`ct_nombramiento_derivado`) y sustituir este cierre antes de abrir la ruta.

Sin cambios: las comunicaciones y las firmas R5 se autorizan con la
organización del propio hijo y no devuelven nada suyo. La firma R5 con plan
ya buscaba la asignación en la historia del hijo y la rechaza. Las demás
transiciones (propuesta, resolución, cancelación, Personal, cese) quedaban
cerradas por fase, versión o número visible; ahora las cierra además el
disparador común.

### Activación

1. Copia con main completo, incluidos CT234 y CT240. Instalar CT241 una vez,
   al final de `lista_sql_main_tras_hz37.txt`. Coteja la preimagen de seis
   funciones y fija su posimagen, propietario, permisos y configuración. La
   reserva O6 y la consulta del circuito pasan a `search_path` con
   `pg_temp` al final, como exige la puerta de calidad.
2. Ejecutar `pruebas_sql/ct241_guardas_nombramiento_derivado.sql` en la copia
   con datos: rechaza al hijo en cada vía y el expediente ordinario sigue
   hasta el paso siguiente. Termina en `ROLLBACK`.
3. Repetir la prueba tras reiniciar PostgreSQL y la aplicación.
4. No reaplicar ni revertir CT241 una vez instalada.
