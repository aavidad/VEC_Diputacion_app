# CT193: ensayo firmado E3 bloqueado por la ligadura V3

Estado a 08/10/2026: **NO-GO dinámico y NO-EJECUTABLE bajo el perfil estricto
de `security-audit` mientras no exista una cuota integral de disco**. Este
artefacto conserva un ensayo preparado; no acredita la activación del POST v3 ni la instalación en la
base principal.

La fuente CT preparada es `2703bda17b3acc226fb4427e192c6c710c05597f`.
El único UP de CT193 tiene SHA256
`cbef35ad78d8b7551d639b8769f020ac9fd4a56c92a15292d0356d89ec127a2a`.
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

Tras acotar el disco y corregir V3, para repetirlo se usa una copia aislada de esta rama con la imagen local
PostgreSQL 18.4 ya instalada y el comando
`VEC_CT_E3_BD_DESECHABLE=SI bash deploy/postgresql/contratacion_temporal/probar_ct193_e3_postgresql18.sh`.
El runner exige el daemon por `/var/run/docker.sock` Unix local y rechaza
`DOCKER_HOST`, `DOCKER_CONTEXT`, TLS o configuración Docker heredada antes de
mutar recursos. Exige también `--pull=never`, red Docker deshabilitada,
bloqueo sin truncado en `/run/user/<uid>` de dueño propio y modo 0700,
socket PostgreSQL en ruta temporal privada, `umask 077`, un contenedor y volumen propios con etiqueta
`vec.prueba=ct193-e3-pg18`; limita PostgreSQL a 1536 MiB, dos CPU y 256
procesos. La compilación y el test Go usan `bwrap` sin red, fuente y módulos
de solo lectura, `GOCACHE=$HOME/.cache/go-build` en su HOME temporal, `-p 6`
y límites de tiempo, procesos, memoria y tamaño de fichero. El volumen de
datos Docker y la capa escribible/logs del contenedor no tienen una cuota total
de disco; comprobar espacio libre no cierra ese riesgo. Esta es una segunda
condición pendiente antes de otra ejecución por agentes. El runner sale **78**
antes de invocar Docker mientras la comprobación de cuota real no exista; hay
que implementar y revisar esa comprobación, no sustituirla por una variable de
confirmación. El sandbox Go ve un
`/etc` vacío, sin montar el del
host. El repositorio se extrae desde `git archive HEAD` a una carpeta privada;
Docker y Go solo ven archivos versionados, sin ignorados del worktree. El
runner reinicia solamente su contenedor durante el replay e intenta
eliminarlo junto con el volumen y temporales al salir, comprobando sus etiquetas
antes de borrar; si Docker falla durante la limpieza, informa el residuo y
requiere limpieza manual del recurso identificado. No usa la
principal, cidonia, secretos reales ni los datos de volumen CT192.

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

La repetición necesaria empieza después de una corrección focal del propietario
V, con sus dos revisiones sensibles. Debe construir **otra** base nueva, aplicar
CT193 una sola vez y pasar toda la suite sin divergencias, con digest opaco del
efecto E3 confirmado por V. Solo entonces puede considerarse el GO dinámico;
el consumidor visual y la autorización nominal requieren revisión propia.
