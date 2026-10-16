# F1-36A: base de consulta de firmas originales

Guion para una copia PostgreSQL 18 de la principal. El corte está en rama:
faltan ensayo SQL, revisiones, arranque y CI. Dirección integra e instala
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
historia. Portafirmas conserva su contrato previo.

El publicador `cmd/vec-publicar-cargos-firma` admite esa operación para los
cuatro roles de consulta. Preparar la versión siguiente según
`scripts/operacion/firma_ct_tres_pasos/LEEME.md` y AUT53: conservar cuadro y
descarga, sustituir solo `consultar_detalle` y comprobar ámbitos y vigencias.
En A se prepara el plan sin publicarlo: la activación conjunta de B+C añade
la fuente que admite el campo nuevo. La publicación exige aprobación externa
y configuración DBA; el ejecutable no tramita una aprobación HTTP.
Los actos nominales serán de C;
A conserva las asignaciones vigentes: no cambia las de consulta ni sus
admisiones en `identidad/firma-vec.json`.

## Ensayo y límites

En la copia, ejecutar pruebas SQL focales AD286, CT228 y AUT103 y la lectura
heredada de firmas y PDF. Comprobar acto exacto, corte, unidad propia o ajena,
asignación retirada, clave ausente y Portafirmas sin unidad. Revisar SQL e
identidad por dos personas independientes.
Arrancar antes y después de instalar los constructores, sin publicar ni
asignar los roles nuevos; comprobar ficha, lista y PDF. Reiniciar aplicación y PostgreSQL; recuperar los PDF y los recibos
originales de firma. Cada lectura conserva su nuevo apunte de acceso. Conservar
salidas, revisión, CI y evidencia del arranque antes de integrar.

B añadirá fuente y anexo; C, panel y activación. A no acredita
`docs/producto/FLUJO_APP.md` §4.0. Dirección decide integración y despliegue.
