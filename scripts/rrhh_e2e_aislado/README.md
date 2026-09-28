# Arnés local aislado RRHH

`bash scripts/rrhh_e2e_aislado/run.sh --preflight` crea un PostgreSQL 18
desechable con publicación solo en `127.0.0.1`, instala la secuencia SQL
necesaria y verifica sus antecedentes. `--full` intenta además montar VEC con
LOGINs nominales, certificados mTLS sintéticos y Chrome a 1440/390 px.
`--smoke` comprueba solo el arranque mTLS/Chrome, sin acreditar negocio ni SQL.

El runner borra del proceso todas las variables `VEC_*` y `PG*` heredadas.
Datos, secretos, binario y caché viven en `/dev/shm/vec-e2e-*`; Docker usa
`--rm` contra el socket Unix local y monta únicamente `deploy/postgresql`.
La única salida conservada es `ultima_evidencia.json`, ignorada por Git, con
códigos de estado, métricas visuales y referencias de recibo sintéticas; no
guarda valores de cookies ni mensajes de consola. No usa SSH,
servicios compartidos ni la base principal. Nunca se debe pasar un DSN de
administrador al servidor. `rrhh_e2e_exportar_dsns` falla cerrado hasta que
exista un contrato de LOGINs nominales compatible con las cuatro autoridades
que el pool de gobierno CT debe asumir.

La consulta positiva de Auditoría requiere dos hechos CT y dos hechos Bolsa
del mismo actor en cada fuente, creados por los casos de uso y en la ventana
autorizada. La política B47 exige publicación nominal de 48 horas naturales
y condición de no cubierta; CT130 requiere su antecedente CT115/CT134. Un
PostgreSQL virgen no contiene esos hechos. El fixture los **comprueba**, no
los inserta ni fabrica recibos. Si falta alguno, el arnés se detiene con la
primera precondición exacta y no declara E2E.

En `--full`, la prueba existente
`TestAuditoriaConsultaRRHHHTTPPostgreSQL18` recorre las consultas HTTP
positivas y las denegaciones exactas de CT y Bolsa usando el servidor mTLS
del arnés. La sonda Chrome usa el certificado de RRHH, observa el portal y, si existe
`VEC_E2E_BOLSA_REF` dentro del proceso aislado, consulta la política de esa
bolsa. Verifica códigos HTTP, errores JS, cookies, almacenamiento y
desbordamiento global, estado 200 de la política y su regla de 48 horas/no
cubierta. El proxy de certificados cliente de Playwright valida la CA local
y Chrome exige rechazo sin certificado cliente. La recuperación de recibo tras reinicio y las
interacciones específicas en pantalla siguen pendientes de un recorrido
versionado cuando la historia sintética y las cuentas nominales estén
preparadas.
