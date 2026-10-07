# Copias de seguridad y restauración desde Administración

Plan para Dirección, 1 de octubre de 2026. Encargo de Alberto de las 17:30.
Base de inventario: `main@f49e01089fb51f58441141decd449df82cb6a420`.
Estado: plan incorporado a `main` mediante la PR #327. La ejecución desde
Administración sigue pendiente. Este documento no instala copias, no abre permisos
y no acredita una restauración operativa.

## Estado comprobado al retomar el 4 de octubre de 2026

Inventario sobre `main@92aedcc59`, antes de retomar los borradores. Los resultados
posteriores se comunican por SHA en el canal y en cada PR; una preparación revisada
no equivale a capacidad instalada.

| Parte del recorrido | Código disponible | Qué falta para usarlo desde Administración |
| --- | --- | --- |
| Manifiesto, versión e inventario | CS01/02 y sus CLI están en main. | Descriptor autenticado y observación de todo el conjunto instalado. |
| Captura y destino | CS03 cifrado y CS05 físico están en main; captura lógica #392 actualizada a `7d0159b97`. | Inventario completo, exclusión real de todos los escritores y configuración admitida de Sistemas. La captura lógica es parcial. |
| Verificación | CS06 físico/lógico y contraste #501 están en main, con multibase y objetos grandes. | Dos restauraciones del conjunto real y consulta nominal de sus datos mediante el binario archivado. No basta salud o sesión. |
| Registro y recuperación de fallos | CS07 está en main; abandono observado #405 actualizado a `be8419234`. | Autoridad y auditoría comunes. El diario offline con actor declarado no las sustituye. |
| API, calendario y doble revisión | API #406, política #399 y control #396 están en main. | Proveedores nominales y montaje en el listener ADMIN. `NuevoHandlerCopias` es una factoría, no una ruta publicada. |
| Pantalla | #404 actualizada a `417a374c3`, con los mismos archivos y revisión de usabilidad anterior. | Integración con la API nominal y recorrido real de dos personas. |
| Órdenes autorizadas | #593 recupera CS08 `c32984285`: CLI, contratos y adaptadores, sin SQL. La separación de claves #407 ya está en main. | Materializador y consumo V3, COMMIT con auditoría común y relectura del mismo compromiso; anclaje vigente fuera del conjunto restaurado. |
| Ejecutar y recuperar | CS11 recuperado por archivos propios en `trabajo/codexa-cs11-recuperacion-20261004@2e029b166`. | Conciliación de diario/testigo, consumidores multibase y autoridades comunes. Sigue en preparación, sin PR ni GO operativo. |

El WIP SQL de K se conserva: AD143, AdministraciónCopias1 y sus dependencias de
identidad/perfil. Antes de prepararlo como instalable hay que inventariar las
reservas y la instalación post-H9, acordar la cadena con K/L y medir las huellas
sobre esa fuente. No se reaplica SQL instalada ni se incorpora una preimagen antigua
por conservar su número. Los borradores originales y los archivos sin versionar de
`controlrestauracionpg` permanecen en su copia de origen.

La primera corrección nueva conserva los errores y la desconexión en la auditoría
HTTP. Su adaptador exige el intento exacto por correlación, contexto y vínculo
originales, canal ADMIN y acuse del registrador común. Acción, finalidad, recurso,
motivos y plazo vienen de configuración. Los rechazos anteriores a sesión utilizan
su frontera propia; no se fabrica identidad. El registro de un error no acredita
rollback ni repite el efecto. Las operaciones permitidas mantienen su auditoría
junto al efecto en la transacción del consumidor propietario.

Para cerrar el recorrido completo siguen pendientes la fuente nominal de K, los
catálogos/registrador de L, la autoridad durante mantenimiento, la configuración de
Sistemas y el montaje. Después corresponderá el ensayo post-H9 con dos personas,
copia previa verificada, sustitución, fallos intermedios y recuperación tras reinicio.
No se ha escrito en cidonia ni instalado SQL durante esta retoma.

## Resultado que debe poder usar un administrador

En el subdominio ADMIN, una persona administradora autorizada podrá ver qué copias
están completas, cuáles se han restaurado con éxito y cuáles sirven para el destino
elegido. Podrá pedir una copia, fijar su calendario y retención. Para restaurar,
otra persona administradora deberá revisar y aprobar el conjunto y el destino.
El sistema comprobará versiones antes de copiar y antes de restaurar; una
incompatibilidad bloqueará la operación sin intentar escribir en el destino.

Cada copia reunirá base física y lógica, configuración, material, documentos y demás
ficheros persistentes, binarios y activos de la versión instalada. Tendrá manifiesto
autenticado, cifrado, huellas y evidencia de dos restauraciones aisladas. Antes de
sustituir el estado actual se hará otra copia completa y verificada de ese estado.

## Inventario y trabajo que se reutiliza

La referencia del incidente es el canal de coordinación del 01/10: una CHECK cuyo
validador había cambiado impedía recuperar el volcado histórico; CT159 corrigió esa
frontera. El ensayo comunicado por D restauró un nuevo volcado sin quitar restricciones.
Más tarde, Dirección recuperó manualmente el estado anterior con una copia física fría
tras fallar el arranque del conjunto H6. Son hechos comunicados por sus responsables,
no ensayos realizados por K. Confirman la necesidad de verificar datos y arranque del
conjunto completo, además del código de salida de `pg_dump`.

En las referencias siguientes, `H6/` designa el kit local autorizado de hito 6. No se
publican sus configuraciones, volcados ni rutas operativas privadas. Sus scripts se
han leído, no ejecutado. `LEEME.md` remite al runbook R2; las vías anteriores quedan
como infraestructura histórica, no como órdenes de instalación.
Dirección declaró después R2 superado en su mensaje «18:10»: exige ensayo local de
todo el conjunto y un único runbook nuevo. R2 se cita aquí solo por sus mecanismos
de copia/recuperación; no es un procedimiento vigente que deba ejecutarse.

| Fuente inspeccionada | Qué aporta | Qué falta para este encargo |
| --- | --- | --- |
| `H6/h6_1b_verificar.py`, `Engine.start_snapshot`, `apply`, `restore_and_verify`, `acl_restore` | Snapshot común para dump lógico, esquema, DATACL y recuentos; globals privados, restore en clon aislado y recibo durable. | Está ligado a PG18/una base/sin tablespaces. Recuentos no prueban igualdad de filas; roles/globales no pertenecen al snapshot de base. No cifrado, periodicidad ni copia física general. |
| `H6/H6_DIRECTO_PRINCIPAL_R2_20261001.md`, §§2 y 6 | Copia fría de PGDATA/configuración PG/HBA, artefactos, configuración APP, material, importaciones y arranque; conserva postimagen para recuperar. | Procedimiento concreto de despliegue. La validación de archivos simulados del runbook no acredita recuperación general ni todos los metadatos/ACL. El incidente real comunicado por Dirección se distingue de esa prueba. |
| `H6/h6_p_principal_o.sh`, `copia_exacta`, `conciliar_postimagen`, `vuelta_atras`; `h6_comun.sh`, `restaurar_pgdata_frio` | Copia física y retorno históricos, contenido/metadatos y conciliación si la aplicación pudo recibir tráfico. | No ejecutarlos como procedimiento vigente. No cubren por sí solos todos los consumidores/almacenes ni efectos externos. |
| `deploy/principal/piden_rrhh_corte2_20260928/README.md`, §copias | Dumps/globales, restauración aislada con propietarios y error cerrado. | Runbook, no servicio de copias; `ensayar_clon.sh` recibe un clon ya preparado. |
| `deploy/principal/composicion_interna/precondicion_f1_identidad/README.md` y `precondicion.py` | Inventario de ACL explícitas, también de tipos de fila, y acta de restauración ligada a preimagen. | El Python valida el acta, no realiza la recuperación. No reutilizar excepciones históricas que omitían filas. |
| `deploy/principal/desplegar.sh` | Respalda binario/web en despliegue. | No respalda toda la base/configuración/material; no es copia previa completa. |
| `deploy/almacen_s3_ceph/README.md` | Perfil de almacén documental con versionado, Object Lock, SHA256 y cifrado por objeto. | Adaptador posible sujeto a Sistemas; no acredita backup integral ni KMS/recuperación completos. |
| `deploy/postgresql/confianza_atestacion_v2/README.md` y `autorizacion_atestada_v3/README.md` | Identifican la necesidad de anclaje externo ante restauraciones atrasadas. | La copia puede recuperar autoridad revocada; resolverlo antes de restaurar en operación. |

Se reutilizan secuencias de captura, aislamiento, contraste y recuperación. Los IDs,
huellas y aprobaciones de una instalación histórica no se transforman en constantes
del nuevo servicio. Ningún resultado del kit sustituye la verificación de cada copia.

Administración tiene esta cola propia, que conserva F/Dirección:

| Referencia exacta | Capacidad y límite |
| --- | --- |
| #232, `d49e6adfcac7a1a68752c239efc6c68a4679191d`; `internal/app/administracion/servidor.go`, `cmd/vec-admin` | Listener ADMIN separado, CA/CRL y verificación mTLS. Solo `GET /livez` (204); no cuenta PG ni API de copias. |
| `origin/trabajo/codexf-admin-http-20260930@bfca798085d984a7a259f4d2924463c2fbb3cae6` | Handler de perfiles con `ResolvedorSesion.ResolverSesionADMIN` y `AuditorFrontera.RegistrarDenegacionADMIN`, sin montaje. Reutilizar autoridad acordada con F, no copiar el árbol completo de su rama. |
| `origin/trabajo/codexf-admin-ui-20260930@700956101e539cb80fdabdfa4ef7cf2d9e6ce46c` | Cliente/vista `administracion-perfiles` y catálogos existentes, sin montaje; no hacer otro portal. |
| `origin/trabajo/codexf-admin-domain-20260930@93ec229f8eb934cb12947d288f3c64526ccc70de` y `codexf-admin-bootstrap-plan-20260930@f5125bc9dc213edf4da616996e23a03f9be01c04` | Contratos/plan de perfiles ya preparados. No son permisos de copia. |
| `internal/modules/administracion` y `web/static/portal-empleado/modulos/administracion/vista.js` en main | Manifiesto y presentación con acciones pendientes; no son la superficie privilegiada ADMIN. |
| `internal/vec/ports/incidencias_tecnicas.go` | Emisor de incidencias no bloqueante; no sustituye el registro obligatorio durable de copia/restauración. |

La dependencia vigente de ADMIN es paso 3 de M → AD137 → AUT24 → sesión, lecturas y
adaptadores → API/UI. Su rol `rol:administracion_perfiles:v2` está limitado a gestionar
perfiles y no autoriza copias. La matriz asigna copia/recuperación al operador de
plataforma: acordar sus concesiones fijas específicas con F/Dirección, sin ampliar el
rol de perfiles ni crear un administrador universal.

No hay autoridad global de versión instalada acreditada en este inventario. Las listas
`deploy/principal/*/migraciones.txt`, `02_migraciones.sh`, las sondas PG por capacidad y
`migration_state()` de `preparar_dietas_desarrollo.py` sirven como fuentes parciales.
CS02 debe separar paquete previsto de postimagen observada y dejar desconocido lo
que no pueda demostrar; ese hueco no se resuelve tomando el mayor número SQL.

Las fuentes vinculantes son ESPECIFICACIONES_AGENTES E03–E12, el contrato modular,
la matriz de roles, seguridad PostgreSQL y
[Administración de perfiles](../estudio_requisitos/administracion_perfiles_2026-09-30.md).
La [PR #232](https://github.com/aavidad/VEC_Diputacion_app/pull/232) prepara la frontera
ADMIN y planes; no acredita asignaciones administrativas ni un circuito web operativo.
No se reconstruirá esa cola ni se provisionarán permisos al pedir una copia.

## Contraste con otros productos

Consulta de documentación primaria realizada el 01/10. El encaje en VEC es una
propuesta propia, no una práctica atribuida a la Diputación.

| Producto y fuente | Comportamiento documentado | Aplicación propuesta a VEC |
| --- | --- | --- |
| [GitLab](https://docs.gitlab.com/administration/backup_restore/restore_gitlab/) | Exige la misma versión y edición; también necesita recuperar secretos. | Guardar y comprobar el conjunto instalado, y prever recuperación de claves fuera de la copia. |
| [Nextcloud](https://docs.nextcloud.com/server/latest/admin_manual/maintenance/backup.html) y [restauración](https://docs.nextcloud.com/server/latest/admin_manual/maintenance/restore.html) | Copia configuración, datos y base; emplea mantenimiento y comprobaciones posteriores. | Inventario completo de almacenes y captura bajo una ventana coherente. |
| [Gitea](https://docs.gitea.com/1.26/administration/backup-and-restore/) | Requiere detener la instancia para alinear base, ficheros y repositorios. | Primera versión con parada verificable de todos los escritores. |
| [pgBackRest](https://pgbackrest.org/user-guide.html) | Ofrece copias físicas, WAL, retención y cifrado del repositorio. | Posible adaptador posterior; por sí solo no copia el conjunto VEC ni gobierna dos aprobaciones. |

[PostgreSQL 18](https://www.postgresql.org/docs/18/backup-file.html) exige detener el
servidor o usar una instantánea coherente para copiar sus ficheros.
[pg_verifybackup](https://www.postgresql.org/docs/18/app-pgverifybackup.html) comprueba
una copia base, pero no sustituye una restauración real con verificaciones funcionales.
No se aplicará a una copia fría corriente como si esta tuviera un manifiesto de pg_basebackup.

## Diseño mínimo y fronteras hexagonales

La capacidad pertenece a `internal/modules/administracion`, con un subdominio de copias.
Administración gobierna petición, política, manifiesto, estados y recibos. Los adaptadores
de plataforma copian almacenes; no convierten Administración en lector funcional de
tablas de Contratación, Personal u otros módulos. Los inventarios de migraciones se
obtienen mediante contratos de infraestructura o proveedores del módulo propietario.
Esta ubicación propuesta debe acordarse con F antes de CS01; la composición y los
archivos ADMIN existentes siguen siendo suyos. Configuración nueva en `config/copias.go`,
sin variables leídas directamente desde dominio o adaptadores.

| Capa o puerto propuesto | Responsabilidad |
| --- | --- |
| `domain/copias` | Manifiesto tipado, compatibilidad, estados y reglas puras. Sin SQL, rutas privadas, comandos ni i18n literal. |
| `application/copias` | Preparar y ejecutar copia; verificar; proponer/aprobar/ejecutar recuperación; revalidar autoridad y registrar efectos. |
| `ports/OrigenCopia` | Inventario de versión instalada y almacenes, exclusión de escritores, captura coherente. |
| `ports/DestinoCopia` | Guardar/leer conjuntos cifrados por referencia opaca y política de destino versionada. Sin ruta enviada por el navegador. |
| `ports/VerificadorRestauracion` | Restaurar físico y lógico por separado en aislamiento y devolver evidencia de resultado. |
| `ports/RegistroOperacionesCopia` | Estado durable, CAS, claves idempotentes, aprobaciones y reconciliación. |
| `ports/ControlPlataforma` | Ventana, parada/drenaje, preparación y sustitución del conjunto; comandos cerrados en adaptador. |
| Puertos comunes existentes | Identidad, PDP/V3, auditoría, reloj, configuración y custodia de claves. No otra autoridad de permisos. |

Un ejecutor separado del proceso que se restaura conserva el registro de operación y
el catálogo fuera de los volúmenes restaurados. Recibe órdenes tipadas autorizadas;
no recibe shell, SQL libre, rutas absolutas o herramientas elegidas por el cliente.
El proceso ADMIN y el ejecutor usan cuentas técnicas separadas y privilegios mínimos.
El ejecutor no obtiene acceso funcional a documentos ni permiso universal.

La autorización central vigente, las aprobaciones y la auditoría deben sobrevivir al
rollback. Si la autoridad central debe detenerse, se necesita un protocolo existente o
acotado de autorización/revalidación durante mantenimiento, revisado antes de abrir
restauraciones. Un recibo antiguo o la asignación recuperada desde la copia no concede
permiso. Esta dependencia es bloqueante para restauración productiva, no para la CLI
de comprobación offline ni para ensayos sintéticos.

## Manifiesto de copia y descriptor de versión

Formato de datos versionado, canónico y validado estrictamente. El descriptor de
release se produce con el paquete instalado y se verifica contra sus bytes; no basta
el nombre del paquete, un commit escrito a mano ni el `main` actual. Si el binario no
incorpora el commit, su descriptor autenticado debe demostrar la correspondencia.

| Bloque | Datos mínimos |
| --- | --- |
| Identificación | Versión del formato, referencia de conjunto/operación, fechas UTC de inicio y fin, motivo y referencia opaca de solicitante; política vigente. |
| PostgreSQL | Versión exacta y herramientas, imagen/runtime por digest, arquitectura y parámetros de formato necesarios, extensiones/versiones, identificador de cluster y bases incluidas; tablespaces y almacenamiento externo declarados. |
| Aplicación | Release y commit de cada binario instalado, SHA256, plataforma, activos web/catálogos/material y configuración compatible; esquema esperado por módulo. |
| Migraciones | Conjunto instalado por módulo: cada identificador y huella de SQL/definición registrada; no solo el número más alto. Procedencia y casos históricos de huella pendientes se explicitan. |
| Componentes | Tipo, identificador lógico, tamaño, SHA256 de cada archivo y raíz del inventario; metadatos necesarios de permisos/propietario/ACL. Sin rutas privadas en la vista ADMIN. |
| Punto consistente | Modo de captura, exclusión de escritores, inventario de origen, WAL/punto de recuperación requerido y evidencia de parada limpia. |
| Cifrado y autenticidad | Formato y algoritmo admitidos por política, referencia/versión externa de clave; autenticación del manifiesto y payload; huellas del cifrado y del contenido dentro del manifiesto protegido. |
| Verificación | Referencias a evidencias físicas/lógicas, recuentos y contraste canónico de contenido por tabla, estado de secuencias/objetos grandes si existen, huellas de esquema/roles/ACL y de ficheros, versión del verificador, resultado y fecha. |

El manifiesto detallado, globals/roles, configuración y contenido van cifrados.
La lista ADMIN expone solo metadatos autorizados: fecha, tamaño, release, estado y
motivos de compatibilidad; no secretos, nombres de tablas/roles privados ni huellas de
valores secretos aislados. Se autentica también el índice mínimo externo para impedir
sustituir el conjunto o falsear su estado. SHA256 acredita integridad, no sustituye
autenticación criptográfica.

Las claves de recuperación no se guardan exclusivamente dentro de la copia que cifran.
Material secreto imprescindible se recupera por custodia externa o componente cifrado
con control reforzado. El plan no introduce claves, certificados ni valores reales en Git.

## Compatibilidad antes de cualquier intento

El verificador devuelve `compatible`, `incompatible` o `no_comprobable`, con razones
estructuradas, clave comparada, esperado, obtenido y acción requerida. Los dos últimos
estados bloquean. Valores sensibles se sustituyen por referencias/digests seguros;
un PARO nunca imprime contenido privado. No hay opción «forzar».

| Momento | Comparación y resultado exigido |
| --- | --- |
| Antes de copiar | Versión PG/herramientas soportada por política, release realmente instalada, binarios/material completos y migraciones reales compatibles con el esquema declarado. Destino/cifrado/custodia disponibles, ámbito completo y espacio suficiente. |
| Antes de probar | Autenticidad/huellas completas, formato conocido y runtime aislado admitido. Ninguna herramienta o ruta procede libremente de la copia. |
| Antes de restaurar | Conjunto autenticado y verificado; comparación con inventario actual del destino y con la combinación que quedará instalada. Capacidad de recuperar config, claves, extensiones y todos los almacenes. |
| Antes de sustituir | Revalidación de la misma política, conjunto, destino, preimagen, dos personas, permisos/caducidad y copia previa válida; CAS bajo exclusión de escritores. |

V1 admite restauración de un conjunto completo con su propio binario y activos. Exige
la misma versión exacta de PG/runtime admitida y que el binario archivado espere
exactamente el conjunto de migraciones y esquema de la copia. También debe ser
admitido por la política de seguridad vigente: una release revocada no se reactiva.
La versión actual del destino se identifica y se compara, pero puede diferir si será
sustituida explícitamente por ese conjunto completo. Nunca se mezcla una base antigua
con el binario nuevo por ser ambos «VEC».

No se restaurará solo la base contra otra release ni se aplicarán migraciones
automáticamente en V1. Si hace falta, se indicará preparar un procedimiento separado
de actualización/restauración, con secuencia aprobada y ensayo del conjunto final.
Igual número de migración con huella distinta, inventario incompleto o un módulo
ausente impiden afirmar compatibilidad. Se compara como conjunto canónico: el orden
de una lista no es un motivo de PARO.

La copia física es de cluster completo, no de una base individual. Si un cluster
comparte bases ajenas al ámbito autorizado, V1 bloquea su sustitución hasta disponer
de destino dedicado o procedimiento de Sistemas que preserve todas ellas. No se
silencian tablespaces, volúmenes externos ni relaciones con otros sistemas.

## Captura y validación de cada nueva copia

1. Autorizar la petición y reservar operación idempotente, destino y versión de política.
2. Comprobar versiones/capacidad y cerrar la admisión de cambios. Detener tareas,
   consumidores, despachos, migradores y edición de configuración; drenar y conciliar
   operaciones en curso que ligan base y ficheros. Mantenimiento HTTP por sí solo no basta.
3. Con escritores excluidos, inventariar datos y obtener copia lógica de cada base,
   globals/roles/membresías, propiedades/ACL de bases y tablespaces. Mantener constraints,
   triggers y funciones: restaurar retirándolos no verifica la copia.
4. Detener PG limpiamente y copiar cluster completo/WAL/tablespaces, configuración,
   material, ficheros y binarios del mismo punto. Sellar/cifrar el conjunto; publicar
   por referencia solo al terminar. Liberar captura sin declarar todavía válida la copia.
5. Recuperar físico en cluster desechable y lógico en otro vacío. Usar runtime admitido,
   red sin salida, volúmenes exclusivos y límites de recursos/tiempo. No montar principal
   ni reutilizar su config de conexiones, replicación o archivado.
6. Comparar ambos con el inventario capturado: recuentos por tabla y huellas de esquema,
   funciones, constraints, triggers, extensiones, propietarios, roles/membresías, ACL
   de base/esquemas/tablas/secuencias/funciones/tipos y privilegios por defecto.
   Contrastar obligatoriamente el contenido lógico de todas las tablas y objetos
   persistentes inventariados mediante huellas canónicas o comparación equivalente,
   incluyendo estado de secuencias y objetos grandes si existen. Igual recuento con
   una celda distinta produce NO VÁLIDA. Normalizar representación y orden de filas sin
   omitir semántica; comprobar también ficheros y enlaces documentales. Si un componente
   no puede contrastarse, queda no comprobable y la copia no válida, nunca se omite.
   No comparar bytes internos de PG ni identificadores internos que cambian legítimamente.
7. Arrancar el binario archivado contra el conjunto de ensayo con despachos bloqueados;
   comprobar salud de dependencias y consultas autorizadas mínimas. Éxito de PG sin
   arranque de VEC no basta. Registrar evidencia y destruir solo recursos del ensayo.

Estados de copia: solicitada, capturando, pendiente de verificación, válida, no válida.
La compatibilidad es otra dimensión calculada para destino/política/fecha concretos.
Un fallo de cualquiera de las restauraciones marca NO VÁLIDA, conserva el diagnóstico
seguro y genera aviso auditado al administrador. Nunca se ofrece como restaurable.

## Restauración con doble control y copia previa

La primera persona propone conjunto, destino, alcance completo, motivo y ventana.
La segunda, distinta por Persona canónica y con asignación nominal vigente, ve la
comparación de versiones, lo que se perderá desde la fecha de copia y el conjunto que
quedará instalado. Aprueba la huella de esa propuesta y su preimagen, con caducidad.
Dos cuentas o certificados de una misma persona no cumplen el doble control.

El ejecutor registra intención en control durable y auditoría segregada antes del
efecto. Bajo exclusión hace una copia completa, cifrada y ensayada del estado actual;
mantiene la exclusión desde su captura hasta la sustitución. Si la copia previa falla,
no hay sustitución. Revalida aprobación, destino y autoridad antes del primer efecto
destructivo y prepara el conjunto recuperado en ubicaciones separadas.

La sustitución cambia base, configuración, ficheros y release como una sola operación
de plataforma recuperable, con diario de pasos y puntos de conciliación. No se afirma
una transacción SQL que abarque volúmenes y servicios. Un fallo tras sustituir pero
antes del recibo queda pendiente de conciliación: al reiniciar se identifica el conjunto
real y se recupera el resultado, sin repetir destrucción a ciegas. Fallo de arranque
mantiene mantenimiento y permite volver al conjunto previo verificado según el plan aprobado.

Restaurar no rebobina correo entregado, firma, GINPIX ni otros efectos externos.
Despachos/outbox e integraciones siguen detenidos hasta conciliarlos y revalidar
identidad, revocaciones y perfiles contra la autoridad actual; no se resucitan permisos
ni se reenvía historia automáticamente. El catálogo y la auditoría de la recuperación
permanecen fuera del conjunto sustituido.

## Administración, programación y retención

El acceso usa listener/subdominio ADMIN, mTLS verificado y perfil nominal fijo con
concesiones exactas por consultar, copiar, configurar, proponer, aprobar y ejecutar.
La provisión es por plan aprobado, huella y CAS; ningún endpoint publica permisos por
petición. El navegador no recibe credenciales de PG ni de almacenamiento.

La pantalla tendrá tabla paginada, detalle de componentes/verificación, razones de
incompatibilidad y formularios de calendario/retención/restauración. Escribirá textos
en `web/static/textos/{es,en}/copias-seguridad.json` y usará los componentes ADMIN.
Se aplicarán las skills de interfaz, usabilidad, aspecto, sistema visual e Impeccable;
revisión de usabilidad independiente en escritorio, móvil, teclado y zoom antes de fusionar.
Los cambios de importadores/manifiestos/versiones se harán en el turno compartido K.

Calendario, zona horaria, destino, límites y retención son configuración versionada y
auditada; no se inventan días ni frecuencias. Cada ejecución programada revalida la
delegación nominal vigente; un permiso revocado detiene nuevos trabajos. No acumula
ejecuciones simultáneas ni duplica copias tras reinicio. Cambiar retención o borrar
requiere doble control según la política administrativa; nunca elimina la copia previa
en uso, conjuntos pendientes de conciliación, protegidos o dependencias necesarias.
Un fallo de almacenamiento/aviso no pasa silenciosamente a éxito.

## Minitareas, dependencias y cierre

Rutas propuestas, sujetas al GO de Dirección y a contrastar con el inventario final.
Una persona escribe cada archivo. No se empezará SQL sin reservar número; será borrador
hasta ensayo/revisión y no se reaplicarán migraciones instaladas.

| ID | Entrega utilizable y archivos propios propuestos | Dependencias | Cierre comprobable | Horas de equipo |
| --- | --- | --- | --- | --- |
| CS01 | Manifiesto y verificador puro; `administracion/domain/copias/**`, `cmd/vec-copias-comprobar/**`, catálogos nuevos. | GO del plan. | CLI offline compara dos manifiestos sintéticos, explica bloqueos y no abre red ni ejecuta restauración. | 8–14 |
| CS02 | Descriptor de release y lector de inventario instalado; adaptadores de copias y empaquetado específico. | CS01; contrato de versión/migraciones. | Detecta binario/material ausente, huella SQL distinta y módulo omitido antes de copiar. | 6–12 |
| CS03 | Destino cifrado/autenticado y recuperación de claves por puertos. | CS01; destino/custodia de Sistemas. | Alteración/truncado/clave incorrecta rechazados; sin secretos en salidas. | 8–14 |
| CS04 | Exclusión de escritores, captura lógica y globals; adaptadores plataforma. | CS02; inventario de servicios/almacenes. | Copia coherente sintética con roles y ACL; conflicto de captura no deja conjunto publicable. | 8–14 |
| CS05 | Captura física fría y componentes completos. | CS03/04; ventana y scope cluster. | PG detenido limpiamente; todos los almacenes y release incluidos, fuente preservada. | 8–14 |
| CS06 | Verificador físico y lógico aislado. | CS03/05. | Ambos restauran constraints intactas, comparan contenido/estructura/ACL/secuencias/objetos grandes, arrancan VEC y se limpian. CHECK regresiva o igual recuento con contenido distinto: NO VÁLIDA. | 10–18 |
| CS07 | Registro durable externo, auditoría y reconciliación del ejecutor. | CS01; autoridad común y diseño de mantenimiento. | Reinicio en cada transición recupera una sola operación/recibo; el rollback no borra control. | 10–18 |
| CS08 | Contratos autorizados de copia/configuración y programación. | CS06/07; perfil ADMIN/provisión vigente. | V3 exacto y revocación; calendario/retención versionados, ejecución única y aviso de fallo. | 8–14 |
| CS09 | Lista, detalle y lanzar copia desde ADMIN; `web/static/admin/copias/**`, catálogos. | CS08; API/portal ADMIN real. | Navegador → permiso → copia → verificación → recibo tras reinicio. | 8–14 |
| CS10 | Propuesta/aprobación de restauración y CAS. | CS07/08; dos administradores y autoridad fuera del rollback. | Misma persona, propuesta alterada, revocación, caducidad o preimagen distinta: cero sustituciones. | 8–14 |
| CS11 | Recuperación completa, copia previa y vuelta atrás. | CS06/10; servicios/destino Sistemas. | Copia previa verificada; sustitución ensayada, arranque y fallos intermedios conciliados; outbox sin reenvío. | 12–22 |
| CS12 | Calendario/retención/doble control en UI y restauración en UI. | CS09/11. | Recorrido de dos personas, denegaciones y explicación del efecto; sin scroll global PC ni secretos. | 8–14 |
| CS13 | Ensayo integral, manual Sistemas y entrega. | CS01–12; infraestructura aprobada. | Conjunto sintético restaurado físico/lógico, recuperación tras fallo/reinicio, revisiones y evidencias exactas. | 8–14 |

Estimación inicial: 110–196 horas de trabajo, incluida validación proporcional y una
corrección por pieza: unas 14–25 jornadas-persona de ocho horas. Con dos personas y
trabajos independientes, previsión orientativa de 10–18 jornadas de calendario técnico,
sujeta a disponibilidad y coordinación con ADMIN. No son horas de caída de servicio.
CS01 es el primer corte y puede entregarse sin servidores. Después pueden correr CS03
y CS02 en paralelo; CS07 puede avanzar con contratos acordados. Captura/verificación
preceden a ejecutar restauración; la UI comparte una secuencia única de versiones.

No se suman estas horas a la cola de ADMIN ya existente: su habilitación es dependencia
externa. La espera de destino/custodia/ventana y decisiones de Sistemas/RRHH no tiene
SLA aprobado; no se promete fecha hasta recibirlas. Medir volúmenes reales y tiempos de
dump/restore dará la duración de mantenimiento y verificará si esta V1 sirve a operación.

## Decisiones que necesita Dirección de Sistemas o RRHH

| Responsable | Decisión pendiente | Consecuencia |
| --- | --- | --- |
| Sistemas | Alcance de cluster/bases, tablespaces, almacenes/material y todas las instancias que escriben. | Sin inventario completo no se puede llamar completa a la copia ni sustituir un cluster compartido. |
| Sistemas | Ventana y objetivos de pérdida máxima/tiempo de recuperación; capacidad para dos ensayos y la copia previa. | V1 con parada total queda pendiente de esa aceptación; no inventar disponibilidad continua. |
| Sistemas/Seguridad | Destino admitido fuera del servidor, cifrado autenticado, custodios, rotación y recuperación de claves. | Bloquea copias operativas cifradas; CLI sintética independiente. |
| Sistemas/Seguridad | Control/auditoría externos, autoridad vigente durante mantenimiento y recuperación administrativa excepcional. | Bloquea restauración operativa; no habilitar autoalta ni confiar en permisos antiguos. |
| Sistemas/Seguridad | Releases/runtime admitidos, revocados y origen autenticado de descriptores. | Define catálogo de compatibilidad; no asumir que «misma major» basta. |
| RRHH/archivo y Sistemas | Frecuencia, retención, conjuntos protegidos y aprobación de borrado. | Se configura con historia; no deducir plazos de conservación legal. |
| Dirección/RRHH/Sistemas | Personas administradoras y responsables de aprobar pérdida de cambios/conciliar efectos externos. | Dos personas acreditadas antes de restaurar, con motivo y alcance aprobados. |

## Pruebas y límites de hoy

Hoy se valida solo el plan: fuentes/rutas contrastadas, consenso Astra por rondas,
repaso Humanizer y `git diff --check`. No hay código, SQL, permisos ni criptografía
nueva; no corresponde recompilar VEC ni repetir campañas de tests para Markdown.
La PR del plan y CS01 se abrirán/iniciarán únicamente con el GO de Claude pedido en
el canal. Al implementar piezas sensibles se requieren ensayo aislado y revisiones
independientes de la versión exacta; gosec/Semgrep local sobre lo cambiado, puertas
focales y global proporcional una sola vez por PR. Ningún ensayo tocará cidonia.

La norma de proporción aplica al resultado: comparar contenido, esquema, roles, ACL y
preservación de la fuente. No bloquear un clon por ctime, xattrs, inodes, orden de listas
o actividad de procesos de fondo. Si a los 30 minutos no hay avance, comunicar clave
fallida con valores seguros y una propuesta más simple. Recursos propios y temporales
se retiran al publicar; material privado ajeno se conserva.

Copias incrementales/PITR, captura sin parada y adaptador pgBackRest quedan para V2,
tras medir la V1. Su anotación en `pendientes_v2.md` se pedirá durante el turno compartido;
no se edita ese archivo fuera de turno ni se presenta V2 como requisito cerrado hoy.

## Consenso Astra

Ronda 1: acuerdo de diseño para parada total, físico frío y doble ensayo. Se exige
drenaje DB/ficheros, control/auditoría fuera de lo restaurado, identidad actual,
cifrado autenticado, dos personas y copia previa completa ensayada bajo exclusión.
La compatibilidad se compara con el conjunto que quedará instalado, permitiendo
recuperar también su binario; nunca mezclar release nueva con esquema antiguo.
Ronda 2: se corrigió la única objeción material, exigir contraste de contenido lógico
en todas las tablas y objetos persistentes, además de recuentos. Incluye secuencias y
objetos grandes; igual recuento con contenido distinto deja la copia NO VÁLIDA.
La horquilla fue consensuada como estimación condicionada, sin prometer una fecha.
Ronda 3: Astra dio GO documental al contenido corregido. Sin objeciones de arquitectura
pendientes; dictamen estático, sin ejecución. La candidata y su huella final se comunican
en CANAL_CLAUDE_CODEX.md. CS01 y la PR del plan siguen pendientes del GO de Claude.
