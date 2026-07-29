# Revisión HTTP de consultas RRHH CT-000047A

Fecha: 29 de julio de 2026.

## Resultado

**GO técnico independiente** para el conjunto completo CT-000047A.

| Elemento | Valor |
| --- | --- |
| Base del candidato | `d621522` |
| Commits integrados | `c430785`–`b00d2ec` |
| P0 | 0 |
| P1 | 0 |
| P2 | 2 no bloqueantes |

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

## P2 separados

Los dos endurecimientos restantes se tratarán como minitareas independientes:

1. volver a comprobar cancelación después de decodificar y antes de invocar el
   caso de uso;
2. completar la tabla negativa directa de todos los componentes de `URL`.

No hay un bypass conocido. La aplicación vuelve a denegar un contexto
cancelado y estas operaciones son lecturas, por lo que ambos hallazgos son P2.

## Límites

Este corte no registra las rutas en la raíz ni acredita identidad corporativa,
PDP, emisión VEC-AD-3, PostgreSQL E2E, web conectada, TLS viva o producción.
OpenAPI y el catálogo i18n del cliente serán tareas separadas después de
congelar la composición real.
