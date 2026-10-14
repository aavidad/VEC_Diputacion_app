# Copia aislada para Cronos: inventario del 14/10/2026

La copia se restauró desde el volcado HZ37 con datos sintéticos confirmados. Usa PostgreSQL 18.6 en el contenedor `vec-codexc-cronos-pg-20261014`, red interna `vec-codexc-cronos-net-20261014`, límite de 2 GiB y puerto `127.0.0.1:55451`. La aplicación no se arrancó en este corte. Ninguna migración nueva se instaló.

## Artefactos de origen

| Fichero privado | SHA-256 |
| --- | --- |
| `principal-hz37-20261014.dump` | `7ae0613fa174aebc8aa36c079a6c4b268b771df41c9310129ee9dfae9650cc45` |
| `principal-globales-20261014.sql` | `95d943a3aab12dfe07895bce4b8b6928457cd6896b128537c53794a787d597ec` |
| `acl-tipos-20261014.sql` | `e058a00b9e0c8c9f06c2fc76d33882637ab7fb715618282305237c1ea959ea71` |
| `acl-base-20261014.sql` | `ab21b92e6f81a838c8c502d431de66454e8c47e7d7fca0becc01882d6d3584aa` |
| `material-cidonia-20261014.tgz` | `699f7c9ff2d800003ccd088167d835261da2abda981e59fdea0745f810cda8e9` |

El [guion de restauración](restaurar_copia.sh) exige dos variables con directorios absolutos privados: `VEC_CRONOS_BACKUP_DIR` para estos cinco ficheros y `VEC_CRONOS_STATE_DIR` para un destino nuevo. Comprueba las huellas antes de crear recursos, restaura roles, volcado y las dos ACL, y extrae el material solo en el destino privado. No modifica cidonia ni otros contenedores. El puerto y la red deben estar libres; el rango de la red se puede indicar con `VEC_CRONOS_SUBNET`. La red propia permite publicar el puerto únicamente en la interfaz local. Tras detener este contenedor desechable, el modo `reanudar` del mismo guion vuelve a iniciarlo con el directorio de datos conservado, sin restaurar ni repetir SQL; requiere la red propia y `VEC_CRONOS_STATE_DIR`.

## Resultado observado en la base

| Comprobación | Resultado |
| --- | --- |
| Restauración globales, volcado y ACL de tipos | Sin errores de `psql` ni `pg_restore` |
| ACL de base entregada por Dirección | 151 concesiones; `PUBLIC` sin `CONNECT` ni `TEMPORARY` después de aplicarla |
| Esquemas | `vec_cronos_v1`, `vec_personal`, `vec_calendarios`, `vec_autorizacion` y `vec_autorizacion_atestada_v3` presentes |
| Tablas/vistas de Cronos, Personal y Calendarios | 25, 56 y 2 respectivamente |
| Funciones de Cronos | 32 en `vec_cronos_v1` y 9 nominales de Cronos en autorización atestada V3 |
| CRN15, consulta interna de saldo | No instalado: SHA-256 de `prosrc` = `6da209aee3dc520303c9976aaefd2b8e22757d3605b35771c7ec224fb6ea5f77`, la preimagen declarada por CRN15 |
| Resolución y notificaciones | Las funciones de Cronos 000009/000010 y los consumidores AD57/AD58 no constan instalados |
| Personal26 para CRN11 | No instalados la tabla de recibo ni el lector histórico propio |

El catálogo de motivos usado por el fichero privado de Cronos tiene una única entrada positiva vigente. El fichero no contiene `auditoria_intentos.motivo_denegado`. El preflight de Cronos exige otra entrada publicada del mismo catálogo; aún no hay una referencia válida para configurarla. No se ha reutilizado el motivo positivo como denegación. Consta el rol sintético `cronos_empleado_sintetico` versión 1 con ocho concesiones y una asignación actual, pero la retirada del rol ya es efectiva. No se atribuye a esa asignación un permiso vigente.

## Material y conexiones

El paquete privado contiene 213 ficheros. La copia local conserva los originales extraídos y crea una segunda estructura de runtime propia; reorienta 37 conexiones de ficheros `.dsn` y 44 de JSON a su base local, con `sslmode=verify-full` y una CA que verifica el certificado activo del servidor. La negociación TLS desde `127.0.0.1:55451` verificó nombre y CA con TLS 1.3. El certificado activo admite `localhost` y `127.0.0.1`; 13 de los 15 certificados del paquete superan 60 días de vigencia. Los otros dos están en un directorio histórico de renovación y caducaron el 06/10/2026. No se usan para el servidor de esta copia.

En la instancia observada, Dirección autorizó cambiar solo las contraseñas de 14 LOGIN nominales CT/Bolsa cuyo texto no estaba en el respaldo. Las nuevas claves se guardan en privado y se referencian en el entorno local de arranque; nombres, membresías, atributos de rol, ACL y sujetos se comprobaron iguales antes y después. Los 14 LOGIN conectaron con `verify-full` y PostgreSQL confirmó TLS en 14/14 sondas. Una conexión adicional de auditoría de frontera ya tenía credencial propia en el material restaurado: se reutilizó sin rotarla y pasó también su sonda TLS. En total hay 15 conexiones nominales CT/Bolsa comprobadas. La rotación corresponde a esta copia y no forma parte del guion automático de restauración. El [preparador de LOGIN locales](preparar_login_ct.py) reproduce ese paso en una copia nueva autorizada: recibe `VEC_CRONOS_STATE_DIR` y `VEC_CRONOS_CONTAINER`, comprueba contenedor, red, puerto, roles y ACL, y añade las conexiones al entorno privado sin imprimir claves. Una copia nueva requiere autorización propia para rotar esas contraseñas o las conexiones proporcionadas por Dirección.

El guion produce un entorno privado con siete variables base de perfil, autenticación, material, TLS y escucha local en `127.0.0.1:18651`. En esta instancia se añadieron 15 conexiones nominales, para un total de 22 variables. La raíz y los directorios del material son privados (`0700`); sus claves, certificados, DSN y entorno tienen modo `0600`. Los binarios compilados por Dirección conservan sus permisos de ejecución y quedan fuera del paquete de restauración. Ninguna clave, certificado, DSN o fichero privado entra en Git.

## Aptitud y límites

La copia está apta para ensayar SQL de Cronos en una ventana exclusiva coordinada por Dirección. El arranque de la aplicación con y sin Cronos, los actos administrativos de catálogo/perfiles, el recorrido en Chrome y la recuperación tras reiniciar aplicación y base quedan pendientes. El estado instalado no acredita C5 ni C7 completos, ni validez normativa del catálogo provisional. La restauración y preparación de LOGIN se hicieron con comandos equivalentes de los guiones; pasaron `bash -n`, ShellCheck y Semgrep local con `p/python` y métricas apagadas sobre el preparador. No se ha repetido el proceso completo ejecutándolos sobre un segundo destino. La integración del preparador, que cambia contraseñas solo en una copia autorizada, requiere las revisiones independientes de identidad previstas por el proyecto.
