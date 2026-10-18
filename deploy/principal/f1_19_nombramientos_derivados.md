# Nombramientos por persona desde una oferta

Tarea F1-19. Primer corte en preparación: derivación y aceptación de plaza
con consumidor HTTP. El panel del padre pertenece al segundo corte. No está acreditada su instalación ni su
recorrido. La propuesta, resolución, incorporación, Personal y cese de los
hijos requieren el corte siguiente; sus rutas conservan las guardas actuales.

## Dependencias

Partir de una copia propia de PostgreSQL 18 con la lista completa
`lista_sql_main_tras_hz37.txt`, incluida F1-16. No reaplicar SQL instalada.
R publica el consumidor nominal AD292 y el alta del perfil/audiencia al final
de F0-18. B no modifica esa cadena. El permiso positivo del hijo es distinto
del acceso al padre: acción `contratacion_temporal.nombramiento.derivar`,
recurso `nombramiento_derivado_ct`, finalidad `gestionar_contratacion_temporal`,
campos vacíos, obligación `auditar`, garantía alta y certificado.

Después del consumidor, instalar B135 y CT234 una vez, en el orden de la lista
de este corte. Antes deben cotejarse los cuerpos instalados, propietarios,
configuraciones y permisos; el ensayo aún está pendiente. La instalación se
detiene si falta una dependencia. Nunca preparar firmas ni recibos fingidos.

## Activación

La composición usa el archivo privado optativo
`identidad/nombramientos-ct.json`. Declara la asignación y el rol publicados,
el motivo y las conexiones nominales separadas; no concede permisos. La
misma fuente de autoridad y el gobierno técnico existentes preparan la
audiencia `vec.ct.nombramiento-derivado.v1`. Sin archivo, perfil, consumidor o
SQL, solo la derivación queda cerrada y la aplicación sigue arrancando.
El esquema es `vec.ct.nombramientos.activacion.v1`. Contiene
`derivacion` con `perfil_ref`, `rol` (`rol_id`, `version`, `huella_sha256`)
y `motivo` (`catalogo_id`, `catalogo_version`, `catalogo_huella_sha256`,
`entrada_clave`), además de `fuente_autorizacion_dsn_file` y
`motivos_autorizacion_dsn_file`: nombres locales distintos bajo el árbol
privado seguro. Las preimágenes y el ensayo final siguen pendientes.

## Comprobar en la copia

1. Preparar la variante de dos personas del caso 4.0 y adjudicar ambas plazas
   mediante los actos de Bolsa, con aceptaciones y recibos reales de la copia.
2. Enviar desde la sesión certificada de RRHH un POST a
   `/api/vec/contratacion-temporal/nombramientos/derivaciones`. El JSON admite
   solo `padre_ref`, `version_padre`, `oferta_ref`, `numero_plaza`,
   `secuencia_plaza`, `participacion_ref`, `recibo_adjudicacion_ref` y
   `clave_idempotencia`. Obtener sus valores de los recibos y lecturas
   autorizadas de la copia. El servidor resuelve organización, hijo, unidad,
   identidad y snapshot; no se aportan desde el cliente. Cotejar hijo,
   aceptación y recibo con la plaza y persona exactas. El panel web del
   segundo corte usará esta misma operación.
3. Repetir la misma operación: mismo recibo. Otra clave para la misma plaza
   debe producir conflicto. No crear claves nuevas para resolver un resultado
   incierto; recuperar el intento y consultar los hijos del padre.
4. Comprobar denegaciones de otra unidad, perfil sin la acción, plaza sin
   aceptación, recibo de otra plaza y padre cancelado. No deben quedar efectos.
5. Reiniciar aplicación y PostgreSQL y recuperar ambos hijos y sus recibos.
   Repetir el arranque sin la configuración opcional y restaurarla después.
6. Completar en el corte siguiente la propuesta y el resto del recorrido de
   cada hijo; comprobar que el cese de uno no altera al otro y que el padre no
   se cierra mientras quede algún hijo abierto.

El cerrojo de la oferta se adquiere en la conexión antes de iniciar la
transacción serializable y se mantiene hasta terminarla. Su adquisición dentro
de una consulta serializable no bastaría para renovar la instantánea. Bolsa
revalida la adjudicación dentro de la misma transacción de la derivación.
Las lecturas de hijos reutilizan el cuadro autorizado y su prefijo de número;
la lista no hereda el permiso del padre. Estas lecturas y la comprobación
visual del panel se entregan en el segundo corte.

Las pruebas focales de dominio y del contrato no acreditan el recorrido de
Chrome, la instalación ni los actos de administración. Se conservarán las
huellas y la evidencia en el material privado de la copia, fuera de Git.
