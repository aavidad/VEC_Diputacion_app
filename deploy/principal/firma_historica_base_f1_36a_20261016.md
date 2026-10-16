# F1-36A: base de consulta de firmas originales

Guion para una copia PostgreSQL 18 de la principal. Las tres migraciones y sus pruebas focales se ensayaron en la copia H;
SQL y autorización tienen dos revisiones independientes. Faltan la
comprobación final tras reinicio y la CI de la cabeza que se entregue. Dirección integra e instala
solo una candidata congelada y ensayada completa.

## Preparación y orden

Comprobar historia, ACL y preimágenes de las tres SQL enumeradas en
`lista_sql_codexh_firma_historica_base_f1_36a_20261016.txt`. Aplicar solo
AD286 → CT228 → AUT103 ausentes, con sus roles propietarios. No usar DOWN ni
reaplicar historia; detenerse ante divergencia. Conexiones y actos son privados.

AD286 verifica el consumo V3 y la unidad histórica ante la asignación vigente.
CT228 recupera por clave exacta y corte; una clave ausente no activa búsqueda
alternativa. AUT103 prepara la concesión
`consultar_detalle_selectores` para la misma acción de consulta de expediente.
El consumidor V2 exige el resultado exacto; conserva PDF, bytes, recibos e
historia. Su ruta nominal reutiliza el manejador y la audiencia de recuperación
existentes, con el permiso propio `recuperar_r5`: consultar o firmar no lo
conceden. Portafirmas sin unidad histórica sigue denegado.

El publicador `cmd/vec-publicar-cargos-firma` admite esa operación para los
cuatro roles de consulta y `recuperar_r5` para los cuatro de firma. Preparar la versión siguiente según
`scripts/operacion/firma_ct_tres_pasos/LEEME.md` y AUT53: conservar cuadro y
descarga, sustituir solo `consultar_detalle` y comprobar ámbitos y vigencias.
En A, el plan de los cargos de consulta se prepara sin publicarlo: B+C añade
la fuente que admite el campo nuevo. La publicación exige aprobación externa
y configuración DBA; el ejecutable no tramita una aprobación HTTP.
Los actos nominales de los cargos de consulta serán de C.

La recuperación nominal exige publicar la sucesora del cargo de firma por
CAS y una asignación vigente de ese rol para cada firmante. En la copia H,
dirección publicó legítimamente los cuatro cargos V2 por el CLI con recibo
del plan `f000b9cd…`. Con autorización expresa para datos ficticios, cargó
cuatro versiones sucesoras de las asignaciones iniciales mediante SQL de
datos, conservando personas, perfiles, ámbitos e historia. Esa carga no es
un acto ADMIN ni acredita el lote ADMIN de organización legada; al regenerar
la copia se sustituye por organización canónica y actos normales.
`identidad/firma-vec.json` usa los mismos cuatro perfiles y cuatro políticas
privadas DEV de admisión con referencias, versiones y huellas nuevas, ligadas
a los cargos V2 y controles publicados; su retirada queda antes de 90 días.
El fichero se activa mediante reinicio vigilado.
Las asignaciones de consulta y sus admisiones siguen en v1 hasta B+C.

La primera recuperación de H devolvió `503` con `VA172`: faltaba únicamente
el origen técnico de consumo para su audiencia nueva. AD178/AD286 y el núcleo
instalado ya admiten el perfil, la acción, el canal y la audiencia exactos.
Después de cotejar la preimagen instalada y obtener dos revisiones, dirección
añade en esa copia **solo** esta fila de datos a
`vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1`:

```sql
INSERT INTO vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
 (login_nombre,audiencia_consumo,operacion,proceso,canal_permitido)
VALUES ('vec_ct_o207_runtime',
 'vec_contratacion_temporal.firmas_r5.recuperar.v2',
 'contratacion_temporal.documento.firmas_r5_v2.recuperar',
 'vec-server','interna_corporativa');
```

La ejecución usa el guion privado con guardas de propietario, núcleo exacto,
LOGIN existente no privilegiado y su único grupo ejecutor. No se ejecuta el
`INSERT` aislado, no se concede un permiso y no se sustituye el consumo V3.
`origen_consumos_ad172/ternas.tsv` documenta la terna; la huella del núcleo H
no se añade a la lista global de núcleos cotejados por una sola rama.

## Ensayo y límites

En la copia, ejecutar pruebas SQL focales AD286, CT228 y AUT103 y la lectura
heredada de firmas y PDF. Comprobar acto exacto, corte, unidad propia o ajena,
asignación retirada, clave ausente y Portafirmas sin unidad. Revisar SQL e
identidad por dos personas independientes.
Arrancar antes y después de instalar los constructores, sin publicar ni
asignar los roles nuevos de consulta; comprobar ficha, lista y PDF.
Comprobar los cuatro cargos publicados, sus asignaciones y las cuatro
políticas DEV privadas, luego ensayar la fila de origen con reversión y
aplicarla una sola vez. Comprobar recuperación propia y rechazo de otra
unidad, clave ausente y asignación retirada. Sin el permiso propio de
recuperación debe quedar denegada. Reiniciar aplicación y PostgreSQL;
recuperar los PDF y los recibos
originales de firma. Cada lectura conserva su nuevo apunte de acceso. Conservar
salidas, revisión, CI y evidencia del arranque antes de integrar.

En H, la consulta nominal devuelve los dos originales del informe y los tres
originales de la resolución. Un firmante vigente puede recuperar el original
de otro firmante de su unidad. Las huellas de los cánones devueltos coinciden
con sus bytes; no se firma de nuevo. Sin cargo devuelve 403 y una clave
inexistente devuelve 404, sin búsqueda alternativa.

El negativo temporal de otra unidad devolvió 401 al resolver la sesión, antes
de CT228. Se restauró RRHH mediante una nueva versión y la recuperación volvió
a 200. Este resultado acredita el cierre de sesión por ámbito; la guarda de
unidad de CT228 queda cubierta por su prueba SQL focal, no por ese HTTP.
Las seis guardas de asignación permanecieron activas y las otras tres
asignaciones no cambiaron. El arranque con el material privado de A tarda
2,27 segundos y la recuperación del Técnico responde 200.

La garantía publicada de consulta, firma y recuperación es SUSTANCIAL en estos
cuatro cargos DEV. No se eleva la garantía declarada ni se admite recuperación
fuera de esos cargos. H responde de la carga inicial de datos ficticios para
la organización legada; se retira al regenerar con organización canónica y
provisionar sus asignaciones por los actos ordinarios de administración.

B añadirá fuente y anexo; C, panel y activación. A no acredita
`docs/producto/FLUJO_APP.md` §4.0. Dirección decide integración y despliegue.
