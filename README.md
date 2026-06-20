# VEC Diputacion Granada

Prototipo Go de VEC, Ventanilla Electronica del empleado publico para la
Diputacion de Granada. La aplicacion raiz es un shell modular con identidad,
menu, permisos, auditoria, eventos e i18n comunes. Personal/Nominas, Cronos,
Dietas y Bolsa son modulos independientes con `ModuleID`, permisos y menus
propios; VEC solo los agrega y permite relacionarlos por empleado, expediente,
justificante o auditoria.

## Estado honesto

Probado:

- Servidor HTTP con `/healthz`, `/`, `/api`, `/api/demo`, rutas de candidatos y
  shell VEC en `/api/vec`.
- UI estatica en `http://127.0.0.1:8080/` como tablero operativo VEC denso:
  modulos, expedientes, filtros, cola, detalle y flujo de acciones.
- Registro de modulos Personal/Nominas (`vec.module.personal`), Cronos
  (`vec.module.cronos`), Dietas (`vec.module.dietas`) y Bolsa
  (`vec.module.bolsa`) via manifiestos, menu segun permisos y acciones demo con
  recibo auditable.
- Workspace VEC en `/api/vec/workspace` con bandeja unificada: expediente de
  empleado, puestos, nomina, antiguedad, servicios prestados, certificados,
  horarios, fichajes, asuntos propios, vacaciones, prejubilacion 63/64, dietas,
  kilometraje provincial y expedientes de Bolsa.
- Endpoint de modulo Bolsa heredado en `GET /api/portal` y demo administrativa en
  `POST /api/demo`.
- Modulo Bolsa integrado en VEC con manifiesto operacional en
  `GET /api/modules/bolsa`, capacidades/admin status, documentos,
  alegaciones, avisos, auditoria y persistencia durable local opt-in.
- Casos de dominio, puertos, repositorios en memoria y handlers HTTP.
- Test obligatorio local: `go test ./...`.

Simulado:

- Autenticacion fake en memoria por defecto; `trusted_headers` disponible para
  simular SSO/Kerberos detras de proxy local.
- Persistencia en memoria por defecto; `local_durable` disponible para el modulo
  Bolsa en `var/bolsa`.
- Integraciones AAPP como puertos/stubs iniciales, sin clientes reales SCSP, SIR,
  Notific@, InSiDe ni AutofirmaV2 cableado en runtime.

Pendiente productivo:

- Autenticacion real por certificado/AutofirmaV2, Kerberos AD/SSO y autorizacion
  operativa.
- Adaptador PostgreSQL, migraciones, auditoria probatoria encadenada y backups.
- TLS/reverse proxy, observabilidad, limites de peticion y hardening.
- Desarrollo completo de cada modulo real: Personal/Nominas con maestro de
  empleados, puestos, situaciones, trienios, servicios prestados y certificados;
  Cronos con cuadrantes y normativa de jornada; Dietas con calculo oficial de
  kilometraje/dietas; Bolsa con baremo, listados, alegaciones, firma,
  notificaciones y persistencia duradera.

## Documentacion de implantacion

- [Contrato de modulos VEC](docs/portal_vec/contrato_modulos_vec.md): contrato
  para enchufar Bolsa, nominas, concursos u otros modulos.
- [Orden de desarrollo Orquesta](docs/portal_vec/desarrollo_vec_orquesta.md):
  alcance, gates y plan de tareas para evolucionar el shell.
- [Plan de implantacion con Orquesta](docs/portal_vec/plan_implantacion_orquesta.md):
  olas, runs, dependencias y agentes recomendados.
- [Estudio profesional de pantallas VEC](docs/portal_vec/estudio_pantallas_profesionales.md):
  flujos completos por modulo/menu, datos visibles, acciones, estados,
  integraciones, validaciones y criterio de terminado.
- [Modulo Personal/Nominas publico](docs/portal_vec/nominas_personal_publico.md):
  normativa, referencias profesionales, modelo funcional y gates de calidad.

## Requisitos

- Go 1.22 o superior.
- Docker y Docker Compose para arranque containerizado.
- `curl` para smoke checks.

## Configuracion

| Variable | Default | Uso |
| --- | --- | --- |
| `VEC_HTTP_ADDR` | `:8080` | Direccion de escucha HTTP canonica. |
| `BOLSA_HTTP_ADDR` | vacio | Alias legado, usado solo si `VEC_HTTP_ADDR` no existe. |
| `VEC_AUTH_MODE` | `fake` | `fake` o `trusted_headers` para identidad local tipo SSO. |
| `VEC_BOLSA_STORAGE_MODE` | `memory` | `memory`, `file` o `local_durable` para datos Bolsa. |
| `VEC_BOLSA_DATA_DIR` | `var/bolsa` | Directorio durable del modulo Bolsa. |
| `VEC_TRUSTED_PROXY_CIDRS` | `127.0.0.1/32,::1/128` | Proxies autorizados a enviar cabeceras trusted. |

La ruta raiz `GET /` sirve la UI estatica. El shell VEC vive en `/api/vec` y el
modulo Bolsa conserva por ahora endpoints heredados bajo `/api`.

## Arranque local con Go

```bash
go test ./...
VEC_HTTP_ADDR=:8080 \
  VEC_AUTH_MODE=trusted_headers \
  VEC_BOLSA_STORAGE_MODE=local_durable \
  VEC_BOLSA_DATA_DIR=var/bolsa \
  go run ./cmd/vec-server
```

Entrada legada compatible:

```bash
BOLSA_HTTP_ADDR=:8080 go run ./cmd/bolsa-server
```

## Arranque local con Docker Compose

```bash
docker compose --profile local up --build vec-api
```

El servicio publica `http://127.0.0.1:8080`.

Parar:

```bash
docker compose --profile local down
```

## Smoke checks

Salud:

```bash
curl -fsS http://127.0.0.1:8080/healthz
```

Shell VEC y modulos:

```bash
curl -fsS http://127.0.0.1:8080/api/vec/modules

curl -fsS http://127.0.0.1:8080/api/vec/workspace

curl -fsS http://127.0.0.1:8080/api/vec/menu \
  -H 'X-Auth-Mechanism: kerberos_ad' \
  -H 'X-Auth-Subject: staff' \
  -H 'X-Auth-Roles: tecnico_rrhh' \
  -H 'X-VEC-Auth-Mechanism: kerberos_ad' \
  -H 'X-VEC-Subject: staff' \
  -H 'X-VEC-Roles: validator_l1'
```

Smoke completo de shell VEC + modulo Bolsa:

```bash
VEC_SMOKE_MANAGED=1 scripts/smoke_local_productizable.sh
```

Las cabeceras `X-Auth-*` alimentan el shell VEC local y las `X-VEC-*` alimentan
el modulo Bolsa en modo `trusted_headers`. En produccion ambas familias deben
venir de un reverse proxy/SSO Kerberos autorizado, no del navegador.

Ejemplo de accion shell:

```bash
curl -fsS -X POST http://127.0.0.1:8080/api/vec/modules/bolsa/action \
  -H 'X-Auth-Mechanism: kerberos_ad' \
  -H 'X-Auth-Subject: staff' \
  -H 'X-Auth-Roles: tecnico_rrhh'
```

Crear candidato en el modulo Bolsa con identidad trusted local:

```bash
curl -fsS -X POST http://127.0.0.1:8080/api/candidates \
  -H 'Content-Type: application/json' \
  -H 'X-VEC-Auth-Mechanism: clave' \
  -H 'X-VEC-Subject: cand-1' \
  -H 'X-VEC-Roles: candidate' \
  -d '{"id":"cand-1","dni":"12345678A","nombre":"Ana Perez","email":"ana@example.test"}'
```

Demo administrativa del modulo Bolsa:

```bash
curl -fsS -X POST http://127.0.0.1:8080/api/demo \
  -H 'X-VEC-Auth-Mechanism: kerberos_ad' \
  -H 'X-VEC-Subject: staff' \
  -H 'X-VEC-Roles: validator_l1'
```

Portal operativo del modulo Bolsa:

```bash
curl -fsS http://127.0.0.1:8080/api/portal \
  -H 'X-VEC-Auth-Mechanism: kerberos_ad' \
  -H 'X-VEC-Subject: staff' \
  -H 'X-VEC-Roles: validator_l1'
```

Los endpoints actuales son de prototipo. No sustituyen registro electronico,
firma, notificacion fehaciente, archivo ENI ni persistencia duradera.

## Arquitectura

- `internal/vec/domain`: tipos del shell VEC sin HTTP ni persistencia concreta.
- `internal/vec/ports`: contratos de registro de modulos, auditoria, eventos e
  interoperabilidad AAPP.
- `internal/vec/application`: casos de uso del shell probados contra memoria.
- `internal/vec/adapters`: HTTP y memoria.
- `internal/modules/personal`: manifiesto de Personal/Nominas: expediente de
  empleado, puestos, situaciones administrativas, antiguedad, servicios
  prestados, certificados, nomina e incidencias retributivas.
- `internal/modules/cronos`: manifiesto de Cronos: fichajes, horarios,
  incidencias, permisos, vacaciones, reducciones 63/64, saldos y aprobaciones.
- `internal/modules/dietas`: manifiesto de Dietas: comisiones de servicio,
  kilometraje, mapa provincial, justificantes, aprobaciones y liquidaciones.
- `internal/modules/bolsa`: manifiesto del modulo Bolsa para registrarlo en VEC.
- `internal/candidate`: nucleo heredado de Bolsa, usado por el primer modulo.
- `cmd/vec-server` y `config`: composicion y configuracion canonica.
- `cmd/bolsa-server`: entrada legada compatible mientras se completa la migracion.

La i18n se centraliza en `internal/shared/i18n`; el prototipo usa catalogo
espanol con fallback si no hay ficheros externos.
