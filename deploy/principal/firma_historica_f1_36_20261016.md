# Candidata F1-36: consulta histórica de firmas originales

Estado al 16 de octubre de 2026: guion preparado sobre una rama de trabajo.
No acredita ensayo SQL, instalación, arranque, recorrido en Chrome ni
publicación. Dirección fijará una candidata congelada y conservará los
resultados de cada paso antes de instalar exactamente ese corte. Solo se usan
datos sintéticos y conexiones privadas de la copia autorizada.

## Alcance y dependencias

La ficha de Contratación temporal conserva su lectura y recibo actuales.
La ampliación propone un selector exacto por firma original y un anexo privado
con otro sello. Permite consultar la fecha, el estado histórico y el recibo
de una firma registrada; no crea firmas ni prueba su validez actual. Exige
acceso vigente a la ficha y una asignación propia y vigente de cargo de firma
en la organización y unidad de aquella firma. No exige ser el firmante
original, ni que la unidad siga en el plan de firma actual. Sin unidad
histórica acreditada se deniega.

La candidata presupone las migraciones de firma ya instaladas en la copia,
incluidas AUT41, AD210 y CT224. Antes de aplicar nada, cotejar la historia de
la base y las preimágenes SQL. No reaplicar una migración ya registrada ni
continuar ante una diferencia. La fase inicial está enumerada en
`deploy/principal/lista_sql_codexh_firma_historica_f1_36_20261016.txt`:

1. `deploy/postgresql/autorizacion_atestada_v3/migraciones/000286_recuperacion_firmas_unidad_historica.up.sql` (AD286).
2. `deploy/postgresql/contratacion_temporal/migraciones/000228_recuperacion_firmas_corte_historico.up.sql` (CT228).

La ampliación preparada después sigue este orden, pendiente de ensayo y de
incorporación a la lista final de dirección:

3. `deploy/postgresql/autorizacion/migraciones/000102_ambito_propio_selectores_firma_ct.up.sql` (AUT102).
4. `deploy/postgresql/autorizacion/migraciones/000103_concesion_consulta_firma_historica.up.sql` (AUT103).
5. `deploy/postgresql/contratacion_temporal/migraciones/000229_selectores_recuperacion_detalle_rrhh.up.sql` (CT229).

AUT102 publica una fachada nominal para comprobar la asignación propia; AUT103
permite publicar una concesión exacta para el campo
`selectores_recuperacion_firmas` mediante el gobierno de cargos de firma ya
existente. CT229 propone el anexo inmutable unido uno a uno al acceso y al
recibo de la ficha. El recibo CT43 anterior no acredita este anexo. AUT93 y
CT219 continúan siendo los contratos de la firma original y su registro; esta
ampliación no los modifica. No se usan migraciones DOWN para activar o retirar
este corte.

## Actos y configuración de la copia

1. Preparar una copia actual de la principal con su configuración privada y
   registrar qué migraciones y actos de firma ya contiene. Ejecutar las SQL
   ausentes en el orden anterior con los roles y el procedimiento de migración
   existentes. Guardar salida, recibos y comprobaciones de preimagen. Los
   ficheros de `pruebas_sql/` se ensayan en la copia; no son migraciones.
2. Preparar la versión siguiente de los cuatro roles de **consulta** de cargos
   para Técnico, Jefatura, Diputación y Secretaría. En cada rol sustituir la
   operación `consultar_detalle` por `consultar_detalle_selectores`; conservar
   sus otras concesiones de cuadro y descarga, así como las personas,
   asignaciones y ámbitos propios vigentes. Usar el gobierno de AUT53, AUT87 y
   AUT91 con propuesta y cierre de dos administradores distintos. Los
   materiales y recibos se generan por copia; no trasladar los de otra base.
   Comprobar antes del acto que las vigencias ficticias de cargos, planes,
   asignaciones y admisiones cubren al menos 60 días. Un acto denegado no publica nada.
3. Usar el guion privado de actos que acompañe esta candidata para preparar,
   aprobar y comprobar esos cuatro roles. Sus argumentos y materiales se
   verificarán en la copia antes de ejecutarlo; este documento no presupone
   una interfaz de línea de órdenes que todavía no está cerrada. Conservar
   los recibos de ambas personas y la versión exacta publicada.
4. Mantener la configuración de firma original ya aprobada según
   `deploy/principal/candidata_hz39/firma/LEEME.md`. La ruta de recuperación
   existente se compone con `VEC_CT_FIRMAS_R5_V2_ENABLED=true` y su doble
   llave de desarrollo; comprobar en el corte final si F1-36 añade una
   configuración propia. Ensayar arranque sin la publicación de los roles
   ampliados y con ella: una función no disponible debe fallar por separado,
   sin impedir el arranque ni abrir datos. Toda modificación de fichero leído
   al arrancar exige reinicio vigilado y vía de vuelta atrás.

## Comprobación focal antes de integrar

Sobre la misma copia y el mismo árbol fuente, conservar los resultados de
las pruebas SQL de AD286, CT228, AUT102, AUT103 y CT229, y las dos revisiones
independientes de SQL e identidad. Revisar la frontera HTTP y ejecutar
Semgrep local sobre los archivos cambiados. La puerta de integración depende
de código Go y JS cerrado, sus pruebas focales y la CI. Este guion todavía no acredita ensayo SQL ni revisión final del conjunto.

Con una firma original y unidad histórica de datos sintéticos, abrir la ficha
con un cargo de consulta habilitado. La respuesta ampliada debe traer ambos
campos `selectores_recuperacion_firmas` y
`recibo_selectores_recuperacion_firmas`, incluso si la lista está vacía. Cada
selector debe referirse a ese expediente, su versión, documento y paso
originales. La lista vacía no es un comodín. Cotejar el sello del detalle y
el del anexo y el vínculo al mismo acceso en cada lectura. Tras reiniciar
aplicación y PostgreSQL, deben recuperarse los mismos selectores y recibos
originales de firma. Cada nueva lectura tiene su propio acceso y sus sellos;
los recibos anteriores se conservan.

En Chrome, pulsar «Consultar firma» junto a una firma. Deben verse su fecha,
estado histórico y recibo del registro original; el navegador no pide campos técnicos ni hace
consultas por todas las filas al abrir la ficha. Navegar a otro expediente
durante una consulta debe cancelarla y limpiar el panel. Probar también que
un fallo del catálogo de textos o del cliente afecta solo al panel y permite
seguir usando la ficha. Comprobar el recorrido con la configuración ampliada
y sin ella, y repetir la lectura tras reiniciar aplicación y base.

Negativos obligatorios en la copia: ficha sin concesión de consulta, campo
no concedido, asignación ajena o retirada, unidad distinta, firma sin unidad
histórica, selector alterado, falta de uno de los dos campos ampliados y
respuesta o sellos divergentes. En todos ellos se deniega o falla la consulta
afectada, sin ampliar permisos ni reconstruir selectores desde la pantalla.
Una denegación se conserva como tal; no se retiran guardas para completar el
recorrido.

La [Ley 39/2015, artículo 26.2](https://www.boe.es/eli/es/l/2015/10/01/39/con#a26)
y el [Real Decreto 4/2010, artículo 22.3 y 22.4](https://www.boe.es/buscar/act.php?id=BOE-A-2010-1331#a22)
orientan la conservación, integridad, trazabilidad y recuperación. No son
concesiones de acceso a expedientes. La aptitud de los documentos para uso
real, las políticas de conservación y los permisos definitivos requieren sus
aprobaciones correspondientes.
