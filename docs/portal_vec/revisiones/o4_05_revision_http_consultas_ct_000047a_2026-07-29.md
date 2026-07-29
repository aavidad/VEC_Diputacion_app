# Revisión HTTP de consultas RRHH CT-000047A

Fecha: 29 de julio de 2026.

## Resultado

**GO técnico independiente** para el conjunto completo CT-000047A.

| Elemento | Valor |
| --- | --- |
| Base del candidato | `d621522` |
| Commits integrados | `c430785`–`b00d2ec`, `cd82caa`, `fc039c2` |
| P0 | 0 |
| P1 | 0 |
| P2 | 0 |

Los dos primeros commits no eran integrables aisladamente. Una primera
revisión detectó tres P1: ruta opaca incompleta, prioridad incorrecta de
cancelación y validación insuficiente de la salida. Los cuatro correctores
posteriores separan y cierran esos defectos.

## Superficie acreditada

Se incorporan dos rutas `POST` exactas:

```text
/api/vec/contratacion-temporal/cuadro/consultas
/api/vec/contratacion-temporal/expedientes/consultas
```

Los manejadores:

- reciben solo intención de consulta;
- rechazan cookies, cabeceras de identidad libres y credenciales del
  navegador;
- exigen JSON cerrado y un máximo exacto de 4 KiB;
- rechazan campos desconocidos, duplicados, contenido sobrante, UTF-8 no
  válido y números fuera de contrato;
- usan DTO explícitos que no publican organización, actor, sesión, perfil,
  recibo durable ni material probatorio;
- responden con `no-store`, nunca con `Set-Cookie`;
- normalizan errores sin exponer causas privadas;
- comprueban el resultado neutral contra la solicitud antes de serializarlo.

La validación pública del cuadro conserva límite, filtros, cursor, orden y
ausencia de duplicados. La del detalle conserva resumen, solicitud, análisis,
cobertura, asignación, hitos y versión observada. Los validadores privados
continúan verificando capacidad, ámbito, recibo, huellas y tiempos.

## Evidencia reproducida

Productor, revisor y dirección ejecutaron:

```text
pruebas focales repetidas 20 veces
go test de HTTP y ports
go test -race de HTTP y ports
go vet de los paquetes afectados
git diff --check
gofmt
verificación de tamaños
Gitleaks sobre los seis commits
```

La revisión obtuvo 84,8 % de cobertura en HTTP y 67,4 % en `ports`. No se
detectaron fugas y ningún fichero supera 800 líneas.

La prueba global conserva un fallo preexistente de `bootstrap`: al ejecutar
desde un worktree situado dentro de `.worktrees`, su detector de pertenencia
considera esa ruta parte del repositorio. CT-000047A no modifica ese paquete.
El resto de paquetes y las puertas focales quedaron verdes.

## Cierre de los P2

Los dos endurecimientos se ejecutaron como minitareas independientes con
productor y revisor distintos:

1. `cd82caa` vuelve a comprobar cancelación después de decodificar y antes de
   invocar el caso de uso. Cubre cancelación y vencimiento para ambos
   manejadores, con cero llamadas al consultor;
2. `fc039c2` completa la tabla negativa directa de todos los componentes de
   `URL` para ambas rutas y conserva un control positivo por ruta.

Las revisiones independientes reprodujeron las pruebas focales veinte veces,
el detector de carreras, el paquete HTTP completo, `go vet`, formato,
revisión del diff y Gitleaks. Ambas emitieron `GO` con P0=P1=P2=0.

## Límites

Este corte no registra las rutas en la raíz ni acredita identidad corporativa,
PDP, emisión VEC-AD-3, PostgreSQL E2E, web conectada, TLS viva o producción.
OpenAPI y el catálogo i18n del cliente serán tareas separadas después de
congelar la composición real.
