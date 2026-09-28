# RRHH 3.06/3.07: comprobación aislada de ofertas

Base del ensayo: `da48a409b7e75249fa1b2378d612a90998aa25bd`.
Solo se usan datos sintéticos. No se toca el runtime compartido ni la base
principal. La política de 48 horas es de **ejemplo** hasta que RRHH la ratifique.

```sh
./scripts/recorridos_rrhh_ofertas_web/ejecutar.sh pg18
./scripts/recorridos_rrhh_ofertas_web/ejecutar.sh focales
```

`pg18` reutiliza tres ensayos en PostgreSQL 18.4, cada uno con contenedor
`--rm` y datos en `/dev/shm`: B54 (48 horas naturales, cambio de horario de
Madrid, versión, replay, ACL y DOWN protegido), Bolsa 000029 (respuesta propia,
replay, denegaciones, concurrencia y mismo recibo tras reiniciar PostgreSQL) y
la revisión de Bolsa (propuesta por orden vigente, adjudicación confirmada y
oferta no cubierta con llamamiento directo). **El consumo AD3 está doblado** en
estos ensayos; no acreditan identidad criptográfica integral.

`focales` comprueba el cálculo en Go, los contratos HTTP con `httptest` y las
vistas RRHH y Mi Bolsa con Node. Los tests de HTTP y web no son un navegador
conectado a PostgreSQL. La resolución calcula una propuesta de adjudicación por
orden vigente o llamamiento directo cuando no hay disposición elegible; RRHH
confirma la resolución. La respuesta del candidato usa su permiso nominal.

Para acreditar el recorrido completo faltan un runtime **aislado** con dos
identidades mTLS sintéticas válidas (RRHH y candidato), concesiones V3 reales,
las migraciones de ofertas y un fixture con al menos dos candidatos ordenados y
una oferta cuyo plazo pueda vencer durante el ensayo. Hay que observar en
1440 y 390 px: publicación → respuesta propia dentro de plazo → propuesta por
orden → confirmación RRHH → GET con mismo recibo tras reiniciar aplicación y
PostgreSQL. En otra oferta sin respuestas elegibles hay que observar la
propuesta de llamamiento directo y su confirmación. La consulta de política
debe registrar auditoría de lectura. No usar como prueba un servidor compartido
ni adelantar el reloj de la principal.

En ausencia de ese entorno, los dos comandos anteriores son pruebas de
componentes, **no** un E2E navegador → API → autorización V3 → PostgreSQL.
