# Comprobación de hechos para el ofrecimiento: primer corte

Tarea F1-09. Prepara el registro en Bolsa; la pantalla corresponde a F1-57
y la comprobación antes de reservar, a F1-10. No cambia situaciones ni permite
continuar una selección. El descuento de antecedentes sigue apagado (duda 165).

CONFIG NUEVA: ninguna. La aplicación conserva su arranque y sus rutas.

1. En una copia nueva, instalar una sola vez la lista completa de main tras HZ37
   más las SQL propias en el orden único de la candidata. No usar la lista
   parcial del primer corte para reponer una copia con historia.
2. Dirección incorpora el perfil `comprobacion_ofrecimiento_bolsa` al núcleo
   en su reconstrucción coordinada. Hasta entonces la acción deniega. La audiencia
   es `vec.bolsa.comprobacion_ofrecimiento.v1`; acción
   `bolsa.comprobacion_ofrecimiento.registrar`, recurso `participacion_bolsa`,
   finalidad `gestionar_contratacion_temporal`, garantía `alto`, campos `[]`
   y obligación `auditar`. El recurso y la huella deben coincidir con el efecto
   de la capacidad; la sesión debe pertenecer exclusivamente al ejecutor de Bolsa.
   El alta completa exige también admitir esa audiencia en la restricción
   `clave_capacidad_version_audiencia_consumo_check` y registrar en
   `configuracion_origen_consumos_v1` el LOGIN nominal de Bolsa, la audiencia,
   la operación y el canal `interna_corporativa`. Reutilizar el patrón de AD288,
   sin modificar su rama ni ampliar otras audiencias. Publicar el rol no
   sustituye estos requisitos. AD290 solo crea la fachada y no los instala;
   mientras falte alguno, el registro permanece cerrado. Las preimágenes y la
   instalación conjunta se fijan con dirección al final de la cadena vigente.
   AD291 reúne esta habilitación con F0-18/AD273 en una sola reconstrucción
   central. Está instalada en la copia conjunta; falta publicar el permiso.
3. Admitir la fuente `fuentes/bolsa_comprobacion_ofrecimiento_rrhh_v1.json`
   y publicar el rol por propuesta y cierre de dos administradores distintos,
   conservando las concesiones y los ámbitos existentes. El material dura 90 días.
   La consulta de motivos no concede esta escritura. El canon de fuente,
   serializado con los tipos Go de dominio, tiene huella
   `cfa1d4e8c3fdab65180c3f85d87c74f22c7844afc0754df023be0b4209eb6b68`.
   La lista vacía de campos se omite en el JSON por `omitempty`; la concesión
   sigue permitiendo cero campos. Esta fuente aún no está admitida.
4. Ejecutar `bolsa_llamamientos/pruebas_sql/b133_calculo_y_denegacion.sql`
   desde el directorio de SQL. Las pruebas revierten sus datos. El ensayo de
   escritura con autorización real, idempotencia y concurrencia sigue pendiente.

La escritura recibe una solicitud JSON y las diez piezas de autorización.
El atributo de recurso `material_sha256` es SHA-256 de la representación
PostgreSQL `solicitud::text`. La huella del efecto es la del contexto de recurso
V3: los ámbitos `ambito_ref` y `unidad_ref`, junto a ese atributo. Se usa el
mismo canon que `RecursoAutorizable` y B115; la huella del material no sustituye
la del contexto. La aplicación normaliza el JSON con PostgreSQL antes de
solicitar la autorización, sin consultar tablas de negocio.
La solicitud contiene `contexto`, `hechos`, `version_esperada` y
`clave_idempotencia`. Actor, perfil y fecha salen de la autorización consumida.
Cada hecho contiene `valor`, `documento_ref` opaca de Documentos y `huella`;
la escritura añade quién lo contrastó y cuándo. Esto registra la comprobación
documental de RRHH; no acredita custodia, antivirus ni firma del documento.
No se amplía el permiso del justificante de renuncia para leer otros documentos.

El contexto fija expediente, necesidad, unidad, ámbito, bolsa, participación,
prelación y versiones del expediente, bolsa, instantánea y política. CT coteja
su agregado actual bajo bloqueo; Bolsa relee la política y la primera persona
pendiente. Una discrepancia o una clave con otro material produce `VBC01`
(el adaptador posterior debe traducirlo a HTTP 409). Una rectificación usa otra
clave y la versión anterior; conserva todas las filas. Un reintento devuelve
el recibo original, también después de una rectificación.
La clave se acota a organización, expediente, necesidad y participación.
La recuperación revalida la autorización vigente, aunque para ello consuma
una decisión nueva; devuelve la fecha, el recibo y la auditoría del acto original.

Hechos: trabajo en la Diputación, tipo de relación, modalidad, causa administrativa
de esa relación (nunca el motivo de una baja), inicio y fin
previsto, fin inicialmente previsto del programa (con el inicio acredita su
duración inicial), inicio de la baja, vacante, prelación y antecedentes completos.
Cada uno lleva su evidencia. Los datos que falten dejan el caso pendiente.
Los límites se leen de la política publicada. La fuente es el Reglamento,
art. 9, apartado cuarto (numerado 3 en el BOP n.º 9 de 16/01/2026).
El recibo solo entrega referencias, versiones, resultado, motivo y fecha.
Las listas existentes recibirán esa proyección mínima en el siguiente corte;
hechos y evidencia quedan en el acto, sin lector nuevo.

Vuelta atrás: mantener desactivado el consumidor y la pantalla. No ejecutar
DOWN ni borrar actos. Este corte no instala configuraciones al arrancar.

Ensayo local del 18/10, candidata conjunta `169dca4ae`: main `9f2a072c7`,
primer corte `82a345058` y consumidor HTTP `6528fed78`. Se restauró HZ37 con
sus permisos y se instalaron las 73 SQL de main de la lista completa de dirección
(`f25173a2`), seguidas de CT233, AD290, B133 y B134. Antes de AD278 se
prepararon los dos LOGIN técnicos ficticios de origen CT y Bolsa con sus grupos
exactos, INHERIT y vigencia de 90 días. No se cambió ninguna guarda.

Las 77 migraciones terminaron y la comprobación de permisos pasó. El cálculo,
el vínculo entre material y contexto V3 y las denegaciones de B133 pasaron;
B134 registró el intento nominal y rechazó cruces de acción y clase. Sus datos
se revirtieron. La aplicación compiló sin red y devolvió 200 en salud y portal
al arrancar y después de reiniciar aplicación y PostgreSQL.
Chrome del sistema cargó el portal base; sin configuración, las consultas de
los módulos y las preferencias devuelven 503. Esto no acredita su uso funcional.
Siguen pendientes la configuración conjunta, la publicación del permiso y la
escritura autorizada, con recuperación, idempotencia y concurrencia. AD291
se instaló después en la copia conjunta `601686060`. Las dos
revisiones independientes de fuente son favorables; no sustituyen ese ensayo.

La candidata conjunta `601686060` instaló 98 SQL en el orden final en dos
copias nuevas. AD291, AD292 y CT238 pasaron sus focales con ROLLBACK; ambas
aplicaciones arrancaron y reiniciaron con la comprobación del guion. Las 97
SQL anteriores son idénticas a la candidata `1ed8b0e42`; CT238 es la única
SQL añadida y Go no cambió. No hubo registro F1-09 autorizado, recuperación
de recibo ni prueba de idempotencia o concurrencia reales. Ese recorrido
espera la publicación gobernada del permiso y la configuración conjunta.
