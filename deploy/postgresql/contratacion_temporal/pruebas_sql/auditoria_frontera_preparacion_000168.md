# Auditoría de frontera de preparación

CT168 amplía el registrador común de CT108/CT162. No crea tablas, roles ni
permisos. Su única dependencia SQL es la postimagen real de CT162; no usa
objetos de CT165–167 ni cambia el núcleo AD3.

La lista acepta dos rutas POST de Selección (`preparacion-bases/guardar` y
`consultar`) y tres de Bolsa (`reglas-baremo/borradores/alta`,
`versiones/consultar` y `recibos/recuperar`). Cada familia tiene su superficie
propia. Selección registra `acceso_denegado`; Bolsa conserva los dos motivos
fijos `autenticacion_requerida` y `acceso_denegado`.

Estos rechazos preceden a la resolución del contexto personal. El actor queda
vacío. No se trasladan JSON, cabeceras, certificados, nombres, IP ni mensajes
de error a la auditoría. La composición debe validar antes el canal mTLS y
el catálogo sellado; la lista de auditoría no concede acceso a las rutas.

El montador de Selección inyecta su callback
`RegistrarRechazoFrontera(*http.Request) error` sobre
`ports.RegistradorAuditoriaFronteraRutaExacta`. Forma una orden con la
superficie de Selección, la ruta nominal, causa `acceso_denegado` y una
correlación confiable; utiliza `corr_no_disponible` si aún no existe. El
callback no lee el cuerpo ni crea identidad. Una auditoría confirmada permite
responder 403; su ausencia o fallo exige 503. Ambos casos terminan antes del
material V3 y del repositorio de negocio.

El export neutral `httpapi.RegistrarDenegacionFronteraPreparacion` recibe
contexto, el registrador existente, método, ruta y motivo. Genera la correlación
con el helper común y deriva la superficie de la lista cerrada. El binder de
bootstrap comprueba primero su capacidad sellada y la misma instancia del
catálogo; esas comprobaciones pertenecen a la raíz. El export registra la
observación técnica y no acredita identidad ni autorización.

El dispatcher común aplica la misma regla de fallo a las nuevas familias,
exige POST sin query y conserva las reglas previas de las otras rutas.
Las denegaciones posteriores de sesión o PDP mantienen sus autoridades;
este corte no observa ni recodifica respuestas de los manejadores.

Dirección ensaya el UP nuevo y la prueba SQL en una transacción del clon,
con dos revisiones sobre el mismo SHA antes de ejecutarlo. La prueba usa
RESTART transaccional y termina mediante ROLLBACK del conductor, conservando
la secuencia y la historia anteriores. DOWN rechaza su ejecución para
proteger los registros de solo adición; la recuperación de la aplicación usa
un artefacto compatible y conserva la auditoría.

El conductor `auditoria_frontera_preparacion_000168_pg18.py` emite el SQL para
`psql -X -v ON_ERROR_STOP=1` del clon cedido por Dirección. No abre conexiones.
Envuelve el UP y los 22 casos existentes en una transacción, sin ejecutar DOWN.
Compara la función completa, sus ACL y dependencias, la tabla, restricciones,
políticas, índices, disparadores, secuencia e historia antes y después del
ROLLBACK. La preimagen debe ser la postimagen de CT162. La lista
`deploy/principal/lista_sql_codexa_ct168.txt` contiene únicamente CT168; CT162
se prepara una sola vez desde el kit causal si aún falta en el clon.

El 3 de octubre de 2026 se reconcilió la candidata de la PR #442 una sola vez
con `origin/main` en `ed9c7de8680d6282ba3f32e7e01118001ef7c262`, sin conflictos
ni cambios en el SQL o en el exportador. El UP conserva SHA256
`68fd471d8b810205de7edf5566560550de6888874199dabb461db0d76dfc9db2`.
El propietario del clon ejecutó el conductor sobre la copia fría h7 con el
kit h7/h8 y el contexto causal de main (AD155, Bolsa77 e ImportaciónConvoca5),
con CT162 ya preparada y los roles nominales de Baremo presentes. Dio código 0:
`CT168-10POSITIVOS-12NEGATIVOS-ACL-HISTORIA-OK` y
`CT168-ROLLBACK-POSTCT162-FUNCION-ACL-HISTORIA-SECUENCIA-IDENTICA`.
No se instaló CT168 de forma duradera ni se ejecutó DOWN.

Semgrep, con métricas desactivadas y reglas locales, revisó los ocho archivos
Go afectados y el conductor Python: 45 reglas, nueve archivos, cero hallazgos
y cero errores. `gosec` se limitó a los tres paquetes afectados: 128 archivos,
37 avisos en archivos ajenos al cambio y cero errores de carga; no señaló
líneas de CT168. `gopls check` terminó sin diagnósticos en los ocho archivos
Go, tras ampliar el límite de memoria del contenedor de 4 a 8 GiB.
Las comprobaciones Go usaron la herramienta 1.26.6 y dependencias locales,
sin red, con el árbol de fuentes de solo lectura.
