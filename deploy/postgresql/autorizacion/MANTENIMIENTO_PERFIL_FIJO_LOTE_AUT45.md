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
perfiles, ámbitos y ventanas permanecen idénticos. CA1 y Sistemas no cambian.
Una asignación caducada o revocada no se rescata ni se prolonga.

Confirmación AD183 genérica y sellos AUT24 se escriben junto al efecto. Cada
llamada gestionada, incluido replay o rechazo, registra el intento AD183 en
la misma transacción; si falla el append, se aborta todo. Replay recupera el
recibo original sólo si el destino sigue vivo. No hay actor humano ficticio,
permisos por petición ni una tabla de auditoría aparte.

Estado: SQL en borrador. Las guardas iniciales proceden de las definiciones
causales AUT42/43; falta cotejar sus hashes con las capturas actuales del
escritor único del clon. No se ha instalado AUT45 ni ejecutado PostgreSQL.
Pendientes: vectores, CLI privada, compatibilidad Go exacta 5/6, dos revisiones
y ensayo con fixture vigente producido por el pipeline autorizado.
