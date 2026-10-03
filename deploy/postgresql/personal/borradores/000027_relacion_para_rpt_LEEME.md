# Relación de Personal para RPT

AD154 y Personal27 preparan la lectura nominal de una relación laboral para
RPT-005/006. Las migraciones UP/DOWN y el ensayo están en sus directorios
habituales. La lista `deploy/principal/lista_sql_codexb_rpt_20261003.txt`
contiene únicamente AD154 → Personal27 y requiere POST149 acreditada.
No se reaplican AD149 ni Personal26. No hay instalación en principal ni
montaje de esta capacidad acreditados por esta candidata.

El lector Go procede de #437, con contrato final
`9f0ae4d688a830e9e1be36291957c45205e112fb`. Personal aporta el hecho laboral;
Organización/RPT decide ocupación, reserva y vacantes. No se presta la
concesión, la ficha ni la evidencia B2 a la nueva lectura.

## Preimagen causal

La captura PG18.4 procede del clon principal, después del reanclaje CRN11
`b2a9bcb9278effa68ec1e89ad63e80624ea90291`. Conserva AD155 y las demás
extensiones instaladas. Personal17 aporta `relacion_servicio_historia`,
sus funciones de validación y su historia. Méritos/Baremo no son dependencias
funcionales del consumidor: su ausencia no exige instalarlos.

| Captura POST149 | SHA256 |
| --- | --- |
| Definición del núcleo | `d912064d905e4349ffb2e1e8f1aab1aebef71e1fd842d5603373c4e6236fcd15` |
| Cuerpo del núcleo | `1bb33d97bf8bbaa7f97dec4e1af1aa41577be38cea17f4b18375346cf7d62dcd` |
| CHECK de audiencias | `0ba3eabde2f45d27afd278cfc008ddc0c3a24cc6d0e5de790b5c6d65dec906a6` |

AD154 exige esas huellas, marcas únicas y los metadatos/ACL/dependencias
previos. Su inversa textual conserva el núcleo completo, sin ejecutar DOWN.
Añade exclusivamente la audiencia `vec_personal.relacion_rpt.v1` al CHECK
validado. Las dependencias de Personal se inspeccionan por `pg_catalog`;
Autorización no recibe acceso a la fuente laboral.

## Consulta y permisos

| Elemento | Contrato nominal |
| --- | --- |
| Consumidor AD154 | `consumir_relacion_para_rpt_v3_atestada`, diez argumentos V3 |
| Perfil técnico del núcleo | `relacion_para_rpt` |
| Acción / audiencia | `personal.relacion_rpt.consultar` / `vec_personal.relacion_rpt.v1` |
| Finalidad / tipo | `conciliar_relacion_laboral_para_rpt` / `relacion_para_rpt` |
| Recurso y efecto | La misma referencia `rel_…` |
| Ámbitos exactos | `empleado_ref`, `organismo_ref`, `relacion_ref` |
| Atributos exactos | `conocido_en`, `material_sha256`, `operacion`, `version_esperada`, `vigente_en` |
| Campos, en orden | `cobertura`, `corte`, `estado`, `periodo`, `procedencia`, `version` |
| Obligaciones | Lista vacía |

La consulta conserva actor, cuenta, perfil y contexto originales. Otro
empleado puede ser objetivo sólo con concesión positiva para esa relación,
organismo, versión y corte. La superficie exigida es `interna_corporativa`;
no se transforma un contexto externo ni se deduce permiso de la titularidad.
El material canónico conserva sus 17 campos y el actor se revalida con el
reloj actual, aunque se solicite un corte histórico.

Personal27 selecciona la última revisión conocida de la relación y después
coteja empleado, organismo y versión. No rescata una revisión anterior para
obtener coincidencia. Conserva estado y periodo de la fuente, con `hasta`
vacía para un intervalo abierto. La respuesta mantiene `no_acreditado` y
`no_acreditada`: no atribuye firma ni eficacia administrativa a B2.

Consumo V3, lectura, recibo y auditoría se confirman en una transacción
SERIALIZABLE. El consumo precede al bloqueo de la relación. El control de
generaciones conserva la fuente y detecta una instantánea obsoleta con
`40001`; cualquier fallo revierte los efectos de la lectura.

El LOGIN lector tiene un solo grupo directo, `vec_personal_ejecutor`, con
`INHERIT TRUE`, `SET FALSE` y `ADMIN FALSE`. El otro LOGIN/pool usa el grupo
común `vec_autorizacion_atestada_v3_registrador_intentos`, con proceso y canal
configurados por el servidor. No recibe acceso a la historia ni a los recibos.
Personal27 no crea una tabla, función o rol propios de intentos.

El lector usa `RegistradorIntentosAuditoria` de #502 y su preflight real. El
fallo se registra después del rollback, con el contexto y vínculo originales
acreditados, actor, perfil y correlación. CA26 e IS13 cotejan la historia
original: una revocación posterior impide la lectura, pero no borra la
identidad del intento anterior. Sin evidencia original no se inventa actor.
AD169, CA26 e IS13 son requisitos del runtime completo; AD154 no los llama
ni los incorpora a sus huellas del núcleo.

## Comprobación y límites

Dirección ejecuta el ensayo en el clon POST149 autorizado y fija la huella
del snapshot previo. La ejecución se coordina con Dirección sobre el hash
exacto y mantiene límites del entorno y módulos locales sin descargas. Usa
COSE EdDSA y HMAC reales con fuentes, identidades y PDP sintéticos explícitos.
No acredita IdP, fuente institucional ni política de RRHH.

El ensayo anterior, recogido a continuación, corresponde al destino propio
de intentos ya retirado de esta candidata. Conserva su acta como historia;
no acredita el registrador común en el nuevo hash.

Ese ensayo conservó funciones previas, CHECK, ACL, metadatos, dependencias y
filas anteriores de Personal17. La comparación inversa normaliza sólo el
literal de audiencia RPT y exige una aparición. Incluye estados, objetivos
y versiones, revocación, negativos de material/COSE, intentos segregados,
registrador caído y concurrencia `40001`. El runner
`b18c370faaa8a34d68bd370aaa324096ca8192ab` terminó `RPT27 ENSAYO-OK`
en el clon POST149: positivos y negativas SQL, preservación y servicio/adaptadores
Go originales con dos LOGIN/pools nominales pasaron. El recorrido Go comprobó
positivo, replay denegado, pools cruzados, registrador sin permiso, sesión
revocada y concurrencia con intento confirmado tras rollback. Se usó una sola
compilación del binario de ensayo. Se conservaron cinco recibos y catorce
intentos, incluidos tres intentos Go sin actor atribuido. Tras reiniciar
PostgreSQL, las huellas de recibos, intentos, consumos y auditoría fueron
idénticas; el recibo CRN11 previo permaneció intacto. El clon, el socket y
los temporales propios fueron retirados tras conservar acta y huellas. La comprobación de
persistencia fue una lectura de la historia conservada, sin recorrido HTTP.
Acta del ensayo SHA256
`2725042c21a747d9815ab310bc041a5ff284e8667e27b88abd9d49cbefaa4861`.

## Reensayo con auditoría común — 3 de octubre

La fuente unida `61363d082db7a635b7658a28f6002c006c87bfaf` conserva
el lector `a7b550b4ab3f9f4bcb822206b091a715676af265`. El SQL recibió dos GO
sobre `e603a0b9123cf385f975e99fd3465fb4e2b46b55` y se instaló una sola vez
en el clon propio POST149 con CA26, IS13 y AD169. La captura normalizada
previa y posterior conserva SHA256
`af2597093754c00801e64bc0dd8fcb8cbeb5b5c87f03589048fe906cde48441e`.
Se conservaron las ACL y los objetos anteriores.

El único binario compilado pasó las tres lecturas SQL de relaciones vigentes,
suspendidas y finalizadas. La primera denegación falló con `22023` al cotejar
su contexto original en CA26: la proyección de empleado de Personal16/CA7
usa `pep_`, mientras CA26 sólo reconstruye vínculos `vin_` de su tabla local.
Cambiar la referencia del fixture alteraría el contexto acreditado. La
corrección corresponde a la autoridad común y debe conservar la procedencia
y los bytes originales. Esta fase quedó pendiente de esa dependencia; el cierre conjunto posterior figura abajo.

El cierre de esa fase fue parcial. Quedaron tres recibos RPT, 6.243 consumos y 6.243
registros de auditoría confirmada, sin intentos RPT registrados. Tras reiniciar
PostgreSQL, esos contadores y las huellas de recibos, consumos, auditoría,
fuente Personal17, generaciones y vectores permanecieron iguales. Los tres
recibos tienen SHA256
`028cbd0333a149ce7bb9a1fc22d9188bb3ee2e4c561147bbb7bc0d637e308f6c`.
IS13 cotejó el vínculo original con resultado positivo en una transacción
revertida; el fallo quedó aislado en CA26.

Se conservaron 1.250 funciones, 9.694 restricciones y las 6.240 auditorías
anteriores. Las 18.599 filas originales de las 33 tablas de Contexto de actor
e Identidad conservaron sus huellas. El ensayo añadió sus fixtures y avanzó
el control de generación de punteros de Contexto de actor.

El contenedor, PGDATA expandido, socket, scratch, overlays, claves efímeras y
binario se retiraron después del cierre. El binario retirado medía 19.090.691
bytes y tenía SHA256
`b740f7d2275c29a966527ffba494c4b279bb17fcaf13903b698b8464633f0732`.
Se conservaron entonces el acta privada del ensayo y el checkpoint frío
`checkpoint-rpt27-post154-parcial.tgz`, de 118.255.519 bytes, permiso `0600`
y SHA256
`a7ac3321d9faa80938b0019742a13a65ef77e35497f807b376540cd325a0d667`.
El checkpoint contiene PGDATA sintético, journal, capturas y las fuentes
mínimas de continuación; no incluye binario, caché, overlay ni claves scratch.

El checkpoint se restauró en el clon coordinado por Dirección. CA27,
`2a28236983e5833c10f0fc62efcca4e38557152f`, recibió dos GO y se instaló una
sola vez. Su ensayo focal cotejó las tres proyecciones de empleado originales,
los vínculos anteriores `vin_`, los negativos de canon y procedencia y una
revocación posterior revertida. Los tres vectores y recibos RPT permanecen
conservados. Ese corte conservó el trabajo para las continuaciones posteriores.

El runner prepara `--continuar-postcheckpoint`: comprueba journal SQL,
captura y los contadores `3 / 6243 / 6243 / 0`; regenera el overlay y compila
una sola vez desde las fuentes acordadas, sin reinstalar SQL, roles, motivos
ni las tres lecturas positivas. Recupera únicamente la misma clave HMAC
sintética instalada en el clon a un archivo scratch `0600`, sin imprimirla.
Si falta, diverge o ha caducado, detiene la continuación; no crea otra versión
ni renueva su vigencia.

La continuación posterior pasó el replay SQL, diez negativas, el corte
histórico y MVCC `40001`, con once intentos durables y contadores
`4 / 6244 / 6244 / 11`. El primer caso Go no llegó a consultar: el constructor
rechazó la referencia técnica del fixture, que carecía de `:` o `_`. Se
corrige a `personal:relacion-rpt:entrada`; el producto conserva su validación.

El modo `--continuar-go` prepara sólo lo pendiente desde esos contadores.
Conserva el primer vector Go caducado sin efectos y usa nuevos casos con
sufijo `_v69`, preparados inmediatamente antes de ejecutarlos. Dirección
autorizó una versión sintética 69 de la misma clave, mediante los INSERT
append-only del propietario y del puntero que ya usaba el fixture. La 68 no
se modifica. Los 32 bytes nuevos permanecen en scratch `0600`; la duración
se configura en el ensayo, con ejemplo de 60 minutos, sin cambiar el reloj.
El selector exige una única versión vigente y exporta su versión real.
La recompilación del overlay se justifica por el literal corregido; no
repite SQL, bootstrap ni comprobaciones anteriores. Esta preparación exige
revisión del fragmento de gobierno y ejecución coordinada en el clon.

## Cierre conjunto y recuperación

ENSAYO-OK en el clon privado, por fases conservadas. La fuente del lector Go
es `5c5e305a397306612299a11741a26a0bdce1abbe`. El runner
`332f8eee392a33ebc73c65b6d9565eb8bde78ef5` corrigió el literal del fixture y
pasó positivo, replay, pools cruzados, registrador caído, revocación y fallo
del sink técnico. El último concurrente anterior quedó denegado por la
comparación textual de fechas equivalentes; su intento histórico se conserva.

Personal30 `86ee0557827da7d4d61c5e00e85af0083209afbd` se instaló una sola vez
tras dos GO y pasó sus ocho casos focales. Su cuerpo final conserva SHA256
`dd0b06eb3ecb2a8317054739ad01f6c363c725f477d7f98d345f01d83487a78f`.
La continuación final `77e07cbd98541c9633b2dccbd16885331d51545f` ejecutó sólo
el concurrente nuevo, con el mismo binario corregido y una barrera SQL sobre
el bloqueo real de la relación. PostgreSQL confirmó `40001` a las
`2026-10-03T16:10:26.792Z`; el intento de error se confirmó en otra transacción
a las `16:10:26.811Z`, con correlación
`2a1f7be92b8296e9a69e0247a65be726`. No hubo DTO, consumo o recibo adicional.
El JSONL y las métricas del emisor común verificaron esa misma correlación.
El fallo del sink ya comprobado conservó el COMMIT de negocio.

El gobierno de ensayo añadió las versiones sintéticas 69 y 70 de la clave
por el mecanismo append-only existente. La 70 tuvo una duración configurada
de 240 minutos; las anteriores permanecieron intactas. No se renovaron
materiales, capacidades ni sesiones anteriores. COSE Ed25519/HMAC, SQL,
servicio y pools fueron reales; fuente, identidad, IdP/PDP y garantías del
fixture fueron sintéticos y declarados.

Tras reiniciar PostgreSQL quedaron los mismos seis recibos, 6.246 consumos,
6.246 auditorías confirmadas y quince intentos. El snapshot antes/después
conservó SHA256
`a1fb605cb0cf6bad42ace3bedc3c152af2b7c8a748cbd545bf71d46a8f227235`;
los recibos, `66579c5cf367ddbd8033bd59024572bde97b09d556b835bfad00dc9c449ced0d`.
Los objetos, ACL, fuentes, vectores y bytes históricos quedaron conservados.
El binario reutilizado tenía SHA256
`15377c020be8eef54be86ecb461d60836257f2b5275de8f647542b03b539c3e5`.
Se retiraron después el clon, datos expandidos, socket, claves, binario,
overlay, controladores y temporales propios. También se retiró el checkpoint
parcial sustituido; quedan las actas y huellas del ensayo.

Orden causal: POST149 → AD154 → Personal27 → Personal30. El runtime de
intentos requiere la auditoría común y CA27
`7f34c77014113bbb54069312e84dd1a62d7c8b6f`, que coteja la proyección histórica
`pep_` sin cambiar su canon. Las listas seleccionan sólo SQL pendientes;
ninguna migración instalada se reaplica.

El montaje operativo permanece cerrado hasta que L complete la procedencia
de proceso y canal de los consumos permitidos. Es una dependencia del
registro común, no un fallo del corte RPT. Este ensayo no acredita HTTP,
mTLS, navegador, IdP institucional, instalación principal ni producción.

La fixture conserva `catalogo_snapshot` sin inventar un catálogo admitido.
Ambos LOGIN de prueba conservan CONNECT propio; retirar el grupo del
registrador quita la ejecución nominal sin confundirla con pérdida de conexión.
Las continuaciones del runner validan el journal y la preimagen conservados
y avanzan sólo las fases pendientes, sin reinstalar SQL ni repetir capacidades.

Los DOWN están preparados y no se ejecutan durante el ensayo. Personal27
rechaza retirada con recibos y conserva siempre la auditoría y los roles
comunes. AD154 rechaza retirada con claves de esa audiencia o dependencias Personal27.
No borrar historia ni usar DOWN sobre una instalación conservada.

La versión anterior de SQL recibió dos revisiones estáticas independientes sobre
`e2a626e91fc748c5dd00be992e42f4a10053a225`. El destino común recibió las revisiones y el ensayo por fases descritos arriba.
La ratificación documental y CI corresponden al hash final publicado. La provisión productiva y el montaje requieren emisor V3, contexto
y material nominales admitidos: un flag no concede acceso. Esta pieza no acredita ocupación,
vacante, grado, antigüedad calculada, certificado ni incorporación eficaz.
