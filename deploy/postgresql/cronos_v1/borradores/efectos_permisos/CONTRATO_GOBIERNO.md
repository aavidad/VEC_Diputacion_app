# C3: recepción y adopción de efectos de permisos

Estado: borrador de CRN13 (`cronos_v1 000013`, número ya reservado).
No habilita publicación, instalación, lectura real ni cálculo de saldos.
Base: `origin/main` en `960795f3090212257d8df92791bf740e3e663c8a`.

## Autoridad y dependencias

RRHH crea, revisa y publica mediante la autoridad común
`internal/vec/application/catalogos.go:ServicioCatalogos.Publicar` y
`internal/vec/ports/catalogos.go:RepositorioGobiernoCatalogos`.
La publicación exige aprobación, huella anterior, autorización vinculada al
actor actual, separación del publicador respecto del creador y último editor,
auditoría y outbox. La retirada también pertenece al gobierno común.
Cronos conserva una recepción verificable y la historia de adopciones.

El SQL disponible para gobierno está especializado en RPT. Faltan fachadas
genéricas durables que acrediten consulta y publicación de este catálogo.
AD117/AD126 no sirven como permiso ni consumidor de Cronos. CRN13 no crea
otro publicador, autoridad de aprobación o consumidor V3.

La dependencia de fuente CRN13 precede al lector CRN12, aunque su número sea
mayor. Su activación espera M3, la postimagen exacta, las fachadas comunes,
la autorización nominal acordada con D y el enclave permitido por D.
En bases separadas falta el puente durable de autorización y ejecución de D;
hasta entonces recepción/adopción permanecen bloqueadas. Una validación HTTP,
un TTL o una copia de decisión no resuelven esa consistencia.

Personal conserva la adscripción histórica a colectivo. Esta fuente aún está
pendiente: Cronos no deduce colectivo del permiso, certificado o cargo ni
crea su maestro. CRN12 deberá fijar su referencia, versión y vigencia por día.

## Contenido admitido

Se conserva el contrato de #269, candidato
`1230661b91684464dca31cf806f5f1a8cddcfb15`, en
`internal/modules/cronos/domain/efecto_permiso_saldo.go` y
`internal/modules/cronos/ports/ensayo_saldo_permisos.go`.
Cada regla tiene exactamente estos doce campos:

| Campo | Condición |
| --- | --- |
| `referencia` | Referencia opaca, única en la versión. |
| `permiso_ref` | Permiso C6 exacto. |
| `catalogo_version_ref` | Versión exacta del catálogo C6. |
| `colectivo_ref` | Referencia de la autoridad de Personal. |
| `efecto` | Únicamente `credito_jornada_completa`. |
| `fuente_ref` | Documento fuente identificable. |
| `fuente_url` | Procedencia HTTPS de hasta 512 bytes; nunca se descarga. |
| `vigente_desde` | Fecha civil ISO, inicio incluido. |
| `hasta_exclusivo` | Fecha civil ISO posterior al inicio, fin excluido. |
| `fuente_version_ref` | Versión exacta del documento fuente. |
| `fuente_publicada_en` | Fecha civil de publicación de esa fuente. |
| `fuente_consultada_en` | Fecha civil igual o posterior a su publicación. |

La colección es explícita, con un máximo de 100 reglas. Una versión admite
una sola regla por `(permiso_ref, catalogo_version_ref, colectivo_ref)`, incluso
si las vigencias difieren. Una modificación exige nueva versión común y nueva
adopción. Se validan todas las referencias y fechas con el contrato #269;
el SQL no introduce una gramática de referencias alternativa.

El envoltorio de recepción conserva bytes exactos y su SHA256, la referencia y
versión de publicación común, su huella canónica y su aprobación acreditada.
La autoridad común debe vincular expresamente esa publicación con el SHA256
del contenido recibido y la proyección de sus reglas. No se supone igualdad
entre SHA256 del archivo y huella canónica de publicación.

La evidencia documental externa debe acreditar, para cada regla, documento,
versión y SHA256 verificado de la fuente, junto con referencia de verificación
y su procedencia autorizada. Se conserva en un envoltorio inmutable separado
de los doce campos. Una referencia, fecha, URL o SHA declarado sin verificación
no acreditan la fuente ni su aprobación. No se almacenan documentos personales.

El JSON de demostración de #269 y su SHA256 acreditan integridad del ejemplo.
`catalogoefectos.Cargar` sigue exigiendo `demostracion=true`; no se amplía ni
se usa como lector de una publicación real. La futura recepción nominal necesita
su contrato y composición propios, reutilizando la validación de reglas.

## Proyección y adopción

`recepcion_catalogo_v1.sql.borrador` propone tres tablas, todas inmutables:

| Tabla | Hecho conservado |
| --- | --- |
| `efectos_catalogo_version` | Contenido exacto, SHA de bytes, publicación común y evidencia externa verificada. |
| `efectos_catalogo_regla` | Proyección tipada y verificable de los doce campos, ligada a la versión recibida. |
| `efectos_catalogo_adopcion` | Cadena de adopciones con versión/huellas previas y posteriores, actor, material, recibo, auditoría y outbox. |

La recepción comprueba la publicación común y su estado actual por un puerto
autorizado. Coteja bytes, huellas, límites y evidencia antes de proyectar.
Cada regla proyectada debe corresponder exactamente al contenido recibido;
no se aceptan reglas añadidas, omitidas ni sustituidas. La cardinalidad declarada
y el conjunto completo se cotejan dentro de la operación, incluida la lista vacía.

Adoptar exige concesión central positiva, exacta y vigente para actor, perfil,
acción, catálogo/versión, ámbito, finalidad, campos y obligaciones. El servidor
resuelve actor, autenticación y el perfil fijo de RRHH que entregue D; rechaza
cualquier perfil distinto y no admite un selector de perfil del cliente.
El identificador de ese perfil queda pendiente de D. D debe fijar
la acción, audiencia, consumidor y forma canónica de recurso/material antes
de implementar. Una aprobación común histórica no concede permiso para adoptar.

La comparación optimista (CAS) coteja secuencia, versión y las dos huellas de
la adopción anterior: SHA de bytes y huella común. La primera adopción declara
ausencia de anterior de forma explícita. Tras bloquear la cabeza de la cadena,
se vuelve a cotejar el estado; dos comandos sobre el mismo anterior permiten
un solo avance. No hay un valor de catálogo «actual» modificable a mano.

La clave se acota a actor y catálogo. Su material canónico incluye operación,
destino, versión/huellas esperadas, contenido/versión/huellas a adoptar, evidencia
de publicación y fuente, finalidad y motivo. Conserva bytes y SHA del material;
no incluye secretos ni credenciales efímeras como contenido de negocio.
Misma clave y material recuperan el recibo original solo tras autorización
actual para esa operación y el mismo perfil fijo de RRHH resuelto por la frontera.
Material distinto produce conflicto sin escrituras.
Un replay autorizado no crea otra adopción, recibo ni evento; su acceso requiere
la auditoría vigente acordada con D. Un permiso revocado deniega el replay.
Se busca la clave antes del CAS de primer uso: un recibo histórico se recupera
aunque existan adopciones posteriores o una retirada, sin activar de nuevo la
política. La autorización actual debe permitir esa recuperación exacta.

Con ubicación y contrato autorizados, una sola transacción confirma consumo V3,
contenido/proyección, adopción, recibo, auditoría común y outbox. Se revalidan
estado de publicación, autorización y caducidad tras esperas y antes de COMMIT.
El adaptador retiene el recibo hasta confirmar COMMIT; ante resultado incierto
recupera con la misma clave/material y autorización actual. Cualquier fallo de
integridad, consumo, CAS o auditoría revierte el efecto completo.

CRN12 fijará la adopción y la versión exacta de política en una lectura conjunta
con programación, trabajo, concesiones y colectivo histórico. La adopción no
transforma una solicitud en concesión ni recalcula historia con reglas nuevas.
Una retirada posterior común impide usos nuevos según el contrato que se acuerde;
no borra versiones, recibos ni cálculos históricos vinculados.

## Puerta para convertirlo en migración

Faltan fachadas comunes de publicación/consulta verificables, contrato V3 nominal
de D, puente durable entre bases, fuente histórica de Personal y postimagen
concreta tras M3. Deben acordarse las cotas del envoltorio de publicación,
evidencias y material; el contenido de reglas no supera 1 MiB como en #269.
Ningún tamaño sin límite queda admitido por este borrador.

La futura migración necesita roles mínimos, RLS, contexto local no ampliable,
guardas de solo adición para tablas y evidencia, restricciones y firmas exactas.
El propietario no tendrá LOGIN ni BYPASSRLS. Las fachadas fijarán `search_path`,
tiempos y límites; sus ACL denegarán PUBLIC y roles no autorizados.
Estas garantías están pendientes y no se atribuyen al esquema comentado.

Casos del ensayo futuro en el clon: publicación sin aprobación o retirada,
publicador creador/editor, huella común o de bytes cambiada, fuente sin verificar,
regla desconocida/duplicada, 101 reglas, proyección incompleta, vigencia límite,
colectivo ausente, perfil distinto del fijo RRHH o selector de perfil del cliente
en adopción y recuperación, permiso insuficiente/revocado, revocación/publicación concurrente,
replay autorizado/denegado, conflicto de clave, dos avances con el mismo CAS,
fallo de auditoría/outbox, serialización, COMMIT incierto y reinicio sin duplicados.
Se requieren dos revisiones sensibles independientes del hash final antes de
ensayo e instalación por Dirección. Este archivo no entra en listas SQL ni ORDEN.

Fuentes aplicables: `ESPECIFICACIONES_AGENTES.md` E03–E07/E10–E12;
`docs/estudio_requisitos/ficha_cronos_2026-09-23.md` C3/C6/C7;
`docs/estudio_requisitos/seguridad_y_despliegue_cronos.md` §§1/5/8–10/12–14;
`docs/portal_vec/catalogos_configurables.md` Gobierno;
`docs/portal_vec/seguridad_persistencia_postgresql.md` §§1/2/5/6;
`docs/estudio_requisitos/analisis_integral_rrhh.md` composición y aceptación.

Entrega limitada a contrato y esquema enteramente comentado. La comprobación
local verifica comentarios, alcance de archivos y `git diff --check`.
No hay SQL ejecutado, servicios, Go, navegador, instalación, PR ni publicación.
No cambia el cierre formal de C3 ni el de Cronos.
