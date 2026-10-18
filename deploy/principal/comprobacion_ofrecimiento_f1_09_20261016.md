# Comprobación de hechos para el ofrecimiento: primer corte

Tarea F1-09. Prepara el registro en Bolsa; la pantalla corresponde a F1-57
y la comprobación antes de reservar, a F1-10. No cambia situaciones ni permite
continuar una selección. El descuento de antecedentes sigue apagado (duda 165).

CONFIG NUEVA: ninguna. La aplicación conserva su arranque y sus rutas.

1. Sobre una copia propia de la candidata con F1-08, instalar una sola vez
   `lista_sql_codexr_comprobacion_ofrecimiento_f1_09_20261016.txt`.
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
3. Admitir la fuente `fuentes/bolsa_comprobacion_ofrecimiento_rrhh_v1.json`
   y publicar el rol por propuesta y cierre de dos administradores distintos,
   conservando las concesiones y los ámbitos existentes. El material dura 90 días.
   La consulta de motivos no concede esta escritura.
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

Comprobación local inicial del 18/10: restauración de HZ37 con sus permisos, seguida
de B117, CT221 y B118 de main. CT233, AD290 y B133 se instalaron una vez;
el cálculo y las denegaciones de B133 terminaron en ROLLBACK. La preimagen de
CT233 se cotejó contra el cuerpo instalado por CT221 de main. La utilidad
temporal de las pruebas la crea el operador; el propietario de Bolsa conserva
su permiso de base sin TEMP. El arranque base devolvió 200 en salud y portal
antes y después de instalar, y tras reiniciar aplicación y PostgreSQL.
No acredita módulos activados ni registro autorizado: faltan el alta central
completa, la escritura positiva, su recuperación y la concurrencia.
La corrección posterior del vínculo entre el material y el contexto V3 exige
una copia nueva y revisión focal antes de actualizar esta evidencia.
