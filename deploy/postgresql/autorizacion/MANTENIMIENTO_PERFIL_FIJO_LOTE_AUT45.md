# Mantenimiento del perfil fijo para lote — AUT45

AUT45 prepara el mantenimiento privado de Rol5 a Rol6. Instalar su estructura
no publica Rol6 ni cambia asignaciones. La operación exige un LOGIN técnico
mínimo propio, configuración externa vigente y huellas del plan, catálogo y
preimagen. Usa la misma barrera de continuidad que AUT42 y conserva el código
histórico de AUT42/43 sin reaplicarlo.

La nueva concesión es `administracion.perfiles.aplicar_lote_ordinario`, módulo
`administracion`, tipo `persona`, finalidad `gestion_perfiles`, garantía `alto`,
campos `[]` y obligaciones `[auditar]`. El catálogo está en
`data/catalogos/administracion/acciones_lote_ordinario_v1.json`. AUT44/AD190
aportan el efecto y el consumo nominal del lote; AUT45 no crea otra fachada.

Gate privado para AD190:
`acreditar_perfil_aplicacion_lote_ordinario_v1(text,text,text,text,text,text,text,text,jsonb,jsonb)`
recibe versión, asignación, Persona, perfil, acción, módulo, tipo, finalidad,
campos y autenticación; devuelve boolean. Sólo admite Rol6, asignación actual
vigente y concesión/catálogo de lote exactos, desde el propietario AD.
AD190 lo coteja antes y después del consumo y liga la Persona única al canon.

Las lecturas de usuarios y denominación mantienen las ramas exactas 5 y 6;
el dispatcher histórico conserva 4/5 y añade sólo la herencia exacta de 6.
No admite versiones superiores por comparación numérica. Las concesiones
anteriores no cambian de contenido ni de obligaciones. Cada entrada de su
catálogo se publica con la siguiente revisión, ligada a la huella de Rol6.

El plan mantiene sus diez campos y tiene `version=2`. Clona el documento de
Rol5, cambia a versión 6, añade únicamente la concesión cerrada de lote y fija
los metadatos aprobados de publicación. Sus dos objetivos son las mismas APP
efectivas y distintas; las asignaciones pasan de revisión 2 a 3. Personas,
perfiles, ámbitos y Hasta permanecen idénticos. Desde se fija al mayor instante
entre el inicio original y la nueva emisión aprobada, usando la utilidad
prospectiva AUT46. CA1 y Sistemas no cambian.
Una asignación caducada o revocada no se rescata ni se prolonga.

Confirmación AD183 genérica y sellos AUT24 se escriben junto al efecto. Cada
llamada gestionada, incluido replay o rechazo, registra el intento AD183 en
la misma transacción. Efecto, append permitido y revalidación final después
de la barrera de cadena comparten subbloque. Si la ventana caduca esperando,
se revierten efecto y append permitido; sólo se añade la negativa gestionada
fuera del subbloque. No se devuelve el recibo provisional; si falla el append,
se aborta todo. Replay recupera el
recibo original sólo si el destino sigue vivo. No hay actor humano ficticio,
permisos por petición ni una tabla de auditoría aparte.

Estado: ensayado en el clon con dos revisiones independientes. Los cuatro
hashes de guarda coinciden con las capturas reales del escritor único, tanto
en el objetivo anterior como en el fixture nuevo. Los gates de versiones
anteriores conservan propietario, ACL y configuración; usuarios mantiene
`search_path=pg_catalog`, `TimeZone=UTC` y `row_security=on`. AUT45 se instaló
una sola vez en el fixture nuevo, después del corrector AUT46 y de sus
dependencias ausentes.
La CLI existente `vec-mantener-admin-fijo` conserva planv1/AUT42/acusev1 y
selecciona planv2/AUT45/acusev2 con un selector cerrado. No acepta otro número,
SQL libre ni otro acuse; se mantienen archivos privados, O_EXCL antes de DB,
COMMIT de todos los estados y cierre indeterminado sin reintento automático.
Usa el catálogo ES/EN existente, cuyo texto no presupone una versión concreta.

Compatibilidad Go preparada en fuente de usuarios y emisor: sólo 5 y 6, con
los mismos campos y obligación de auditar. Los vectores admiten ambas y
rechazan 4/7; el de fuente usa un proveedor que falla y no inventa una lectura
V3 favorable. CLI, fuente y emisor: normal, race y vet verdes, gofmt/diff check
verdes; vecsilencio sin nuevos errores. Semgrep local en once Go y dos SQL:
cero hallazgos. Gosec en los tres paquetes: un G101 heredado en `acreditarSQL`
de la fuente, consulta de atributos/ACL sin contraseña ni clave, sin avisos
propios. No se añaden supresiones ni cambia esa consulta.

El vector SQL estructural y el de fechas terminaron con código 0. La CLI real
publicó Rol6 y su replay, también tras reinicio, conservó el mismo recibo,
fecha y huella. Las asignaciones persistidas pasan el validador Go; CA,
identidad, ámbitos, Hasta y la asignación de Sistemas permanecen intactos.
La aprobación divergente dejó una denegación común durable. El detalle y los
límites están en `FECHAS_PROSPECTIVAS_MANTENIMIENTO_AUT46.md`.

No se acredita todavía lectura nominal HTTP, V2, navegador ni efecto del lote
AUT44/AD190. La espera temporal con varias sesiones no se ejecutó; no se
presenta el vector puro como sustituto de ese ensayo.
