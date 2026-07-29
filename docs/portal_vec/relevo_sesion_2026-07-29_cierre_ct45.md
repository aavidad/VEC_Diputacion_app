# Relevo de sesión: cierre de CT-000045

Fecha: 29 de julio de 2026.

## Autoridad de trabajo

Rama integradora:

```text
integracion/ct-o4-04e-20260726
```

Worktree:

```text
.worktrees/ct-stable-docs
```

El directorio raíz conserva una rama histórica. No se programa allí y no se
modifica el Word de RRHH que permanece sin seguimiento.

## Estado confirmado

CT-000045 está integrado en `02b02ed`, supera la matriz integral PostgreSQL
18.4 y dispone de `GO` independiente con P0=0 y P1=0.

El corte:

- instala dos fachadas nominales mínimas para cuadro y detalle;
- fija las constantes de seguridad dentro de PostgreSQL;
- admite solo los doce argumentos tipados del contrato;
- consume la capacidad VEC, registra el acceso y consulta el motor privado en
  la misma transacción `SERIALIZABLE READ WRITE`;
- concede al rol runtime `EXECUTE` solo sobre esas fachadas;
- minimiza la salida y rechaza resultados obligatorios nulos;
- normaliza los errores sin oráculos;
- avanza `24/8` a `25/9` y conserva una retirada segura a `24/8`.

Evidencia:

```text
docs/portal_vec/revisiones/o4_05_revision_fachadas_ct_000045_2026-07-29.md
```

## Métricas del corte

| Ámbito | Estado tras publicación verde |
| --- | --- |
| Contratación temporal | `23/46`, 50 % |
| O4-05 | `3/5` hitos |
| Bolsa productiva | `1/14`, 7 % |
| Producción | `NO-GO` |

O4-05 no aumenta todavía: las fachadas son una pieza necesaria de su tercer
hito funcional, no una ruta administrativa completa.

## Siguiente corte exacto

Implementar el adaptador Go/PostgreSQL mínimo de `SesionConsultaRRHH`.

Write-set previsto:

```text
internal/modules/contrataciontemporal/adapters/postgres/
```

Decisiones cerradas:

1. reutilizar `SesionConsultaRRHH`;
2. reutilizar las órdenes, el alcance, los recibos y los cánones existentes;
3. enlazar exactamente doce entradas, dieciocho salidas escalares de cuadro y
   quince de detalle;
4. usar un LOGIN nominal y un pool exclusivos para consulta RRHH;
5. abrir `SERIALIZABLE READ WRITE`;
6. no reutilizar el rol histórico de recuperación ni una transacción
   `READ ONLY`;
7. traducir errores sin filtrar existencia o detalles internos;
8. no modificar dominio, aplicación, puertos, HTTP, composición ni web en este
   corte.

Ficheros orientativos:

```text
consulta_rrhh_postgresql.go
consulta_rrhh_postgresql_sql.go
consulta_rrhh_postgresql_salida.go
consulta_rrhh_postgresql_canon.go
fabrica_pool_consultas_rrhh.go
```

## Camino crítico posterior

```text
adaptador PostgreSQL Go
→ composición raíz y propiedad de recursos
→ matriz TLS/mTLS viva
→ misma web definitiva sin adaptadores DEMO
→ E2E HTTP completo
→ conformidades RRHH, DPD y Sistemas
```

No se abre O5, O6 ni otro módulo hasta cerrar O4-05. No se construye otro
marco de pruebas: se reutilizan las puertas existentes y se separan las
pruebas focales de la preparación histórica cuando corresponda.

## Límites conocidos

- Los `go test ./...` globales fallan en pruebas heredadas de
  `internal/app/bootstrap` que confunden un worktree con un repositorio; el
  mismo fallo está reproducido sobre la base `c3e93ba` y no lo introduce
  CT-000045.
- El runner histórico conserva un P2 no bloqueante por transportar centinelas
  sintéticos con variables de entorno de `docker exec`.
- No hay autorización para datos reales, despliegue ni producción.

