# CT193: ensayo firmado E3 en PostgreSQL 18 efímero

Estado de esta revisión: CT193 define explícitamente las cuatro funciones que
antes se reconstruían en bloques dinámicos. El UP tiene SHA256 `c71fab86fdd15b0c0d77a3feff7e36f25592302fd589a46aab37261090a82094`.
Las tres funciones privilegiadas fijan `search_path=pg_catalog,pg_temp`; las
guardas cotejan preimagen, postimagen, propietario, permisos y metadatos.
Esta versión requiere revisión independiente y un ensayo PostgreSQL 18 nuevo.
Los resultados anteriores de `c5bc06e8…715c` se conservan como historia y no
acreditan esta sustitución.

Evidencia anterior a esta revisión: el ensayo ordinario de la rama
`966a479051c9ea7730bd44f6a86db43b132394e8` terminó con código **0** en una
base PostgreSQL 18 nueva usando CT193 SHA256
`cbef35ad78d8b7551d639b8769f020ac9fd4a56c92a15292d0356d89ec127a2a`.
No acredita la nueva versión del UP, la activación del POST v3 ni su instalación en la
base principal.
La corrección V3 `2de6dabbf6cbd4950fd70a75015b67dbee99309b` está
incorporada en esta rama mediante cherry-pick; sus dos revisiones sensibles
corresponden a esa fuente exacta.

La fuente CT preparada es `2703bda17b3acc226fb4427e192c6c710c05597f`.
El UP de CT193 de esta rama tiene SHA256
`c71fab86fdd15b0c0d77a3feff7e36f25592302fd589a46aab37261090a82094`.
El runner nuevo es `probar_ct193_e3_postgresql18.sh`, su fixture es
`pruebas_sql/ct193_e3_preparar.sql` y el test Go es
`internal/modules/contrataciontemporal/adapters/postgres/confirmacion_alta_v3_postgresql18_test.go`.
La compilación del test sin `VEC_CT_E3_PG18=SI` lo omite; ese resultado no es
una prueba PostgreSQL.
El runner exige un worktree Git limpio y las huellas fijadas de CT48, CT165 y
CT193 antes de usar Docker. CT48 SHA256:
`f33cf450fb6189ab0b21712a90ff80f4df95f20fb3a142b13a54bc65c9b6c6cc`;
CT165 SHA256:
`7a7ac82c0137d77339996022e234c416843a2525cf426f306430c0c66a05bf6e`.

Para repetirlo se usa esta rama limpia con la imagen local
PostgreSQL 18.4 ya instalada y el comando
`VEC_CT_E3_BD_DESECHABLE=SI bash deploy/postgresql/contratacion_temporal/probar_ct193_e3_postgresql18.sh`.
El runner exige el daemon por `/var/run/docker.sock` Unix local y rechaza
`DOCKER_HOST`, `DOCKER_CONTEXT`, TLS o configuración Docker heredada antes de
mutar recursos. Exige también `--pull=never`, red Docker deshabilitada,
bloqueo sin truncado en `/run/user/<uid>` de dueño propio y modo 0700,
socket PostgreSQL en ruta temporal privada, `umask 077`, un contenedor y volumen propios con etiqueta
`vec.prueba=ct193-e3-pg18`; limita PostgreSQL a 2 GiB, dos CPU y 256
procesos, sin red, sin descarga, sin logs Docker y sin política de reinicio.
El runner compara el espacio libre real con 2 GiB para `/var/tmp`, 3 GiB
para el almacén Docker y 1 GiB para el GOCACHE ordinario. Suma los mínimos
cuando dos rutas comparten sistema de ficheros y sale **78** si no caben.
Imprime dispositivo, rutas, espacio disponible y mínimo exigido en cada
comparación. Un timeout envolvente termina el ensayo a los 29 minutos y deja
hasta un minuto para la limpieza, con límite total de 30 minutos.
La entrada del script siempre aplica ese timeout; ya no acepta una variable
heredada para omitirlo.
No configura una cuota total para el volumen Docker; el tiempo y los recursos
del ensayo están acotados y solo usa datos sintéticos propios. La compilación
y el test Go usan `bwrap` sin red, fuente y módulos de solo lectura,
`GOCACHE=$HOME/.cache/go-build` del operador, `-p 6` y límites de tiempo,
procesos, memoria y tamaño de fichero; el runner limita a 256 MiB cada fichero
que escriben sus procesos. El sandbox Go ve un `/etc` vacío, sin montar el del
host. El repositorio se extrae desde `git archive HEAD` a una carpeta privada;
Docker y Go solo ven archivos versionados, sin ignorados del worktree. El
runner detiene su contenedor efímero y levanta otro con el mismo volumen para
comprobar la recuperación; intenta
eliminarlo junto con el volumen y temporales al salir, comprobando sus etiquetas
antes de borrar; si Docker falla durante la limpieza, informa el residuo y
requiere limpieza manual del recurso identificado. No usa la
principal, cidonia, secretos reales ni los datos de volumen CT192.
Una prueba mínima aislada con la imagen fijada confirmó que `--rm` retira el
contenedor al detenerlo y que un segundo contenedor lee el marcador del
mismo volumen propio. La etiqueta de esa prueba quedó sin contenedores ni
volúmenes al terminar. Esta comprobación no ejecutó la suite CT193.

La imagen local fijada es
`postgres@sha256:3a82e1f56c8f0f5616a11103ac3d47e632c3938698946a7ad26da0df1334744a`
(ID `sha256:1bf3d6960db467e87a506daef30feb41fecc23b7c5f96b157e873059f2ffb50a`).
El runner R3B histórico exigía otro digest,
`sha256:882236b897e39051d2368c5ccc6cda944904723506b2dfc97f2a8f5bc9afa382`;
no estaba disponible localmente. La diferencia es de imagen de ensayo y no
autoriza a alterar una base con historia.

La preimagen se construye desde los UP reales de contexto, autorización y
AD3, CT1–11, CT13, CT46–48, CT12/14/15, AUT-CT2–6, CT17–44 con sus roles,
identidad, AD3-3–6 y CT67/102/106/159/165. El preflight exige CT48 y CT165
presentes y CT193 ausente. Después confirma E2 **antes** de instalar CT193;
aplica CT193 una sola vez; recupera E2 y prueba E3 con fin, E3 con
`causa_fin+politica_fin`, replay tras reinicio real, colisión y dos
confirmaciones concurrentes. Cada vector sintético se emite, aplica y consume
de inmediato porque la capacidad dura cinco segundos. Los bundles y la clave
HMAC sintética permanecen en temporales `0600` fuera de Git.
El modo de diagnóstico `VEC_CT_E3_SOLO_PREIMAGEN=SI` sale con código **77** y
mensaje NO-GO antes de compilar o instalar CT193; no acredita E3. Cada fase
exige el marcador PASS del test exacto, además del código de salida; INT y
TERM salen con códigos 130 y 143 tras el intento de limpieza.

Un ensayo en una base nueva completó esas fases. La colisión devolvió 409 en
la barrera de candidatura y 42501 en la llamada SQL directa sin alias de
huella acreditado; no añadió historia. Otro ensayo en otra base nueva detectó
antes del consumo AD3 SQLSTATE 22023, «ligadura VEC-AD-3 inválida». Su
preflight de 18 predicados identificó `decision_vigencia` divergente: la
decisión canoniza `valida_hasta` con seis decimales fijos en
`internal/vec/domain/autorizacion_decision_huella_v3.go`, mientras el emisor
de capacidad usa `time.RFC3339Nano` en
`internal/vec/adapters/seguridad/confianzaatestacion/capacidad_v3_emisor.go`
y elimina ceros finales. `000002_consumidor_capacidad_v3.up.sql` compara ambos
textos literalmente. Es una diferencia anterior al control de caducidad; no
se atribuye a TTL. No se sesgó la marca temporal para evitarla ni se modificó
el núcleo V o una migración AD3 instalada.

La repetición final construyó otra base nueva con CT193 ausente, aplicó su UP
una sola vez y obtuvo los marcadores PASS de E2 antes y después, E3 con fin,
E3 con `causa_fin`, replay tras reconstruir el contenedor sobre el mismo
volumen, colisión y concurrencia. Versiones, actuaciones, auditoría, outbox y
recibo válido pasaron de `0/0/0/0/0` a `1/1/1/1/1` en cada alta nueva; E2 tras
CT193, replay y colisión conservaron `1/1/1/1/1`. El runner exige el marcador
PASS exacto en cada fase y terminó en 24,1 segundos. La comprobación final por
etiqueta encontró cero contenedores y cero volúmenes residuales.

Un intento previo sobre la misma fuente falló durante la compilación con
`bwrap: Creating new namespace failed: Resource temporarily unavailable`:
el límite de 2048 procesos era menor que los 3487 hilos del UID compartido.
Se restauró el límite anterior de 8192 en
`966a479051c9ea7730bd44f6a86db43b132394e8`, manteniendo `-p 6`, la memoria
y el tiempo acotados. Ese intento limpió sus recursos y no instaló CT193.
El consumidor visual y la autorización nominal requieren revisión propia.


## Nuevo canon de la circular: ensayo del 08/10

La candidata `256612e2fec4a19c5267b793d7ac1b559751b846` recibió dos ratificaciones independientes. El UP CT193 con SHA256 `c5bc06e8c717c11d38f3d6371e435d4fc3cd850bcf14d938988a4f7a5e45715c` se aplicó una sola vez en otra base PostgreSQL 18 desechable. La ejecución terminó con código 0: E2 antes y después, E3 con fin, periodo abierto, recuperación tras reconstruir el contenedor sobre el mismo volumen, colisión y concurrencia. Cada fase exige el marcador PASS de su prueba; las altas nuevas dejaron una versión, actuación, auditoría, outbox y recibo, y los replays mantuvieron esos cinco contadores.

La salida original del runner se conservó fuera de Git, con modo 0600; este resultado no depende de la transcripción del ensayo anterior. Al terminar no quedaban contenedores ni volúmenes con la etiqueta del ensayo. No se instaló CT193 en una base con historia ni se acredita aquí el POST nominal o el recorrido Chrome. La dependencia de consulta RPT por código exacto sigue pendiente de su propietario.


El ensayo ampliado en `1dbc5bf2d01f77e80d9700688da6ee2d62ccdf40` comprueba el dato nuevo: el vector cerrado y los de colisión/concurrencia contienen catálogo2 y `numero_personas="2"` dentro de `necesidad.campos`. El test exige ese valor en el canon y en la instantánea, y coteja los bytes persistidos y el recibo después de reiniciar. Salida0 y todas las fasesPASS; el vector abierto conserva catálogo1. El UP sigue siendo `c5bc06e8…715c`, aplicado una vez en otra base nueva; no quedaron recursos propios. La salida original está custodiada fuera deGit con modo0600.
