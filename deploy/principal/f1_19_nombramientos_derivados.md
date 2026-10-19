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
crear tablas temporales; ahora la tabla de cotejo la crea el migrador y solo
cede al propietario leerla y escribirla durante la transacción.

Las pruebas `ad292_…`, `b135_…` y `ct234_…` pasaron con `ROLLBACK`. Con el
certificado de RRHH, la ruta responde 503 con un cuerpo válido, 422 con un
cuerpo vacío y 405 a GET; sin certificado no se abre la conexión. Tras
reiniciar PostgreSQL y aplicación (3,6 s), las consultas de Bolsa y
Contratación temporal responden igual y no hay ningún hijo creado.

