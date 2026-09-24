# D7: acceso técnico de Personal para Dietas

Paquete **aditivo e independiente de F4**. F4 mantiene exactamente sus ocho
LOGIN `vec_dietas_r1d_*_desarrollo`; D7 crea dos cuentas nominales fuera de
ese prefijo y después las activa. No instala migraciones, no modifica
asignaciones V3, no arranca el servidor y no prepara una novena cuenta F4.

| Variable privada del servidor | LOGIN | Única membresía heredada |
| --- | --- | --- |
| `VEC_DIETAS_PERSONAL_ASIGNACION_DATABASE_URL` | `vec_personal_d7_asignacion` | `vec_personal_d7_ejecutor` |
| `VEC_DIETAS_PERSONAL_AUDITORIA_FRONTERA_DATABASE_URL` | `vec_personal_d7_auditoria_frontera` | `vec_personal_registrador_frontera` |

Los DSN son privados, distintos, con TLS `verify-full` para el servidor. La
conexión administrativa de este paquete usa socket local y DBA; no se reutiliza
como DSN de la aplicación. Ninguna contraseña, DSN o fichero de servicio entra
en Git. No usar `SET ROLE` ni añadir miembros a las cuentas de aplicación.

## Puertas y secuencia

Requiere PostgreSQL 18, `postgres`, F4 `v3` activa y exactamente ocho LOGIN F4,
Personal 000012/000013 instaladas y revisadas, las cuatro fachadas de asignación
y la fachada de auditoría con ACL positiva de su grupo exacto. `vec_personal_d7_ejecutor`
es un grupo mínimo creado por 000012: las cuatro fachadas D7 son sus únicas
funciones permitidas. Este paquete añade `CONNECT` al grupo al preparar.
Inventaría las ACL y propiedades de ambos grupos y exige una lista positiva:
`CONNECT` en `postgres`, `USAGE` en `vec_personal` y solo las cinco fachadas
nominales. Rechaza `PUBLIC` en base y objetos VEC, concesiones CT, funciones
ajenas, ACL directa, rol propietario, membresía cruzada, firmas o funciones
`SECURITY DEFINER` ausentes, sesiones activas de las dos cuentas y repetición.
Antes de activar, drenar conexiones y cerrar externamente las nuevas; `ALTER
ROLE` no corta sesiones existentes. Mantener el selector Dietas apagado hasta
que Dirección acredite la postimagen completa de D7 y el recorrido real.

1. Configurar transporte privado: `PGSERVICE`, `PGSERVICEFILE` y `PGPASSFILE`
   fuera de Git, ficheros `0600`, servicio local por socket hacia `postgres`.
   Alternativamente, usar `VEC_D7_POSTGRES_CONTAINER` para un contenedor local
   accesible por el socket `/var/run/postgresql`. Crear `VEC_D7_EVIDENCIA_DIR`
   privado `0700` fuera de Git.
2. Ejecutar `bash deploy/principal/d7_personal_acceso/ejecutar.sh --inventario`.
   Revisar preimagen y migraciones; la numeración no se infiere del código.
3. Ejecutar `--preparar-rollback` y revisar que no deja cuentas. Con dos
   revisiones E10 del hash final y autorización operativa de Dirección:
   `VEC_D7_APLICAR=SI-D7-REVISADO bash deploy/principal/d7_personal_acceso/ejecutar.sh --preparar-commit`.
   Se crean dos roles `NOLOGIN` sin contraseña con membresías segregadas y se
   concede `CONNECT` solo al grupo D7 mínimo.
4. DBA establece una contraseña distinta y aleatoria para cada rol por su canal
   privado. No pasarla por argumentos, historial de shell, logs o este paquete.
   El paquete exige ambas contraseñas presentes antes de activar.
5. Ejecutar `--activar-rollback`; con el mismo criterio de revisión, ejecutar
   `VEC_D7_APLICAR=SI-D7-REVISADO bash deploy/principal/d7_personal_acceso/ejecutar.sh --activar-commit`.
   Cotejar el postinventario, conexiones TLS y las cinco firmas/ACL.

La fase de preparación no es idempotente. Si se pierde la respuesta, consultar
el inventario privado antes de decidir. Una cuenta parcialmente provisionada o
una F4 retirada requiere análisis DBA, no reintento automático ni `DOWN`.
La instalación en la base principal corresponde solo a Dirección.

## Ensayo aislado

`VEC_D7_TEST_BASE=<directorio_privado_0700> bash deploy/principal/d7_personal_acceso/probar_pg18.sh`

El ensayo usa PostgreSQL 18.4 desechable sin red, un fixture estructural
sintético y contraseñas aleatorias temporales. Prueba `ROLLBACK`/`COMMIT`, F4
`v3` y novena cuenta, función/ACL ausente, `PUBLIC`, concesión CT, función
Personal ajena, función sin `SECURITY DEFINER`, propiedad indebida, ausencia
de contraseña, membresía cruzada, ACL directa y repetición. Acredita el paquete y sus puertas;
la postimagen real de migraciones y el servidor se comprueban aparte.
