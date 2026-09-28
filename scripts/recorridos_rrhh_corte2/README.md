# Ensayo aislado del corte 2 RRHH

Fuente probada: `trabajo/piden-rrhh-corte2-20260928` en
`da48a409b7e75249fa1b2378d612a90998aa25bd` (28-09-2026). Solo se usaron
datos sintéticos, contenedores PostgreSQL 18 efímeros sin puertos publicados y
datos en `/dev/shm`. No se tocó una base ni un servicio compartido.

## Comprobaciones ejecutadas

| Capacidad | Comando o prueba | Resultado |
| --- | --- | --- |
| B46, recepción de reincorporación | `bash deploy/postgresql/bolsa_llamamientos/probar_reincorporacion_titular_pg18.sh` | OK: replay, cese tardío o incompatible, ACL, carrera con DOWN e historia protegida. |
| B47/B54, política de ofertas | `bash deploy/postgresql/bolsa_llamamientos/probar_politica_ofertas_48h_pg18.sh` | OK: política versionada, oferta ligada, replay, 48 horas y cambio horario, +90 rechazado, DOWN protegido. Este ensayo usa un doble AD3; no acredita PDP V3 completo. |
| CT136, auditoría de denegaciones | `bash deploy/postgresql/contratacion_temporal/probar_ct136_auditoria_frontera_auditoria_pg18.sh` | OK: LOGIN registrador nominal, dos rutas exactas, sin lectura directa, deriva de privilegios denegada, historia protegida. |
| CT130, perfil y revocación central | `TestCT130PreimagenCentralPostgreSQL` y `TestCT130PublicacionSinPreimagenPreparadaFallaCerrada` en PostgreSQL 18.4 | OK: cuatro casos (permitida, restringida, revocada, carrera), perfil separado, CAS y replay. Tras reiniciar PostgreSQL, dos asignaciones revocadas y ocho sesiones activas conservaron idéntica la huella de todas las tablas centrales. |
| API y composición focal | `go test` focal en bootstrap, aplicación CT, HTTP CT/Bolsa y Auditoría | OK en cinco paquetes. Incluye `GET /api/vec/auditoria/opciones` 200 por `httptest`, denegación de fuente o motivo ajenos y separación V3 de lectura/escritura CT130. |

La prueba CT130 usó una adaptación temporal del runner
`scripts/rrhh_ct130_revocacion/probar_pg18.sh` del candidato aislado
`f6c6a95a9`: se sustituyó únicamente la comprobación del hash de checkout
por `da48a409b` y se fijó `GOCACHE=/dev/shm/vec-go-cache-corte2`. Su SHA256
temporal fue `61f9f9286ec0b4445af744195e12f871628b40a752054862dbe2754417333480`.
El runner instala las migraciones reales de ContextoActor y Autorización,
crea un LOGIN de prueba sin superusuario ni `BYPASSRLS`, conecta por socket
local con SCRAM y reinicia el contenedor. La adaptación no modifica el
candidato ajeno ni el código fuente probado.

La invocación Go focal fue:

```sh
GOCACHE=/dev/shm/vec-go-cache-corte2 GOMAXPROCS=2 go test \
  ./internal/app/bootstrap \
  ./internal/modules/contrataciontemporal/application \
  ./internal/modules/contrataciontemporal/adapters/httpinterno \
  ./internal/modules/bolsa/adapters/httpinterno \
  ./internal/vec/auditoria \
  -run 'Test(Reincorporacion|CT130|PoliticaOfertas|Auditoria|RaizExactaAuditoria|ManejadorAuditoria|PreflightFronteraAuditoria|RegistradorFronteraAuditoria|CapacidadReincorporacion|CapacidadPoliticaOfertas)' -count=1
```

## Límite de la evidencia

No hay en este checkout una semilla institucional V3 y mTLS que conecte en
un mismo servidor las tablas CT, Bolsa y Auditoría con las rutas nuevas. El
ensayo HTTP es `httptest` con autoridad sintética; no hubo llamada HTTP a un
servidor en escucha, Chrome a 1440/390, ni reinicio de la aplicación. La
recuperación observada tras reinicio corresponde exclusivamente a la base
central de ContextoActor/Autorización CT130. Ninguna de estas pruebas acredita
el recorrido E2E de RRHH ni instalación en cidonia.
