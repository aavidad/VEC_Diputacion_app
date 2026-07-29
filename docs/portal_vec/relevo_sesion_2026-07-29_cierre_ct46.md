# Relevo de sesión: cierre de CT-000046

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

CT-000046 está integrado en `1834143`, `063d5d4` y `6c57644`. Implementa el
adaptador PostgreSQL de `SesionConsultaRRHH` y tiene `GO` técnico independiente
con P0=0, P1=0 y P2=0.

El corte:

- analiza de forma estricta los cánones minimizados de cuadro y detalle;
- invoca las dos fachadas CT-000045 con valores, tipos y orden exactos;
- escanea completas las salidas de veintiuna y veinte columnas;
- reconstruye proyecciones y recibos V2 y los valida antes de confirmar;
- ejecuta una única transacción `SERIALIZABLE READ WRITE`, sin reintento;
- conserva cancelación y normaliza errores sin oráculos;
- crea un pool TLS exclusivo ligado a un LOGIN nominal de Sistemas;
- acredita con pruebas el orden nominal y fallos de escaneo, validación y
  confirmación ambigua sin reintento.

Evidencia:

```text
docs/portal_vec/revisiones/o4_05_revision_adaptador_consultas_rrhh_ct_000046_2026-07-29.md
```

## Métricas del corte

| Ámbito | Estado tras publicación verde |
| --- | --- |
| Contratación temporal | `24/46`, 52 % |
| O4-05 | `3/5` hitos |
| Bolsa productiva | `1/14`, 7 % |
| Producción | `NO-GO` |

O4-05 no aumenta todavía: el adaptador completa una dependencia del recorrido,
pero aún no existe una ruta administrativa compuesta y probada de extremo a
extremo.

## Siguiente corte exacto

Componer la raíz interna real de consultas RRHH.

Alcance:

1. construir el pool nominal y transferir su propiedad al ciclo de vida;
2. construir `SesionConsultaRRHHPostgreSQL`;
3. inyectarla en los casos de uso existentes de cuadro y detalle;
4. conectar exclusivamente autoridad corporativa y PDP reales;
5. registrar los manejadores ya cerrados sin rutas paralelas;
6. cerrar recursos en orden inverso y fallar cerrado ante cualquier dependencia
   ausente o incoherente.

No se cambia dominio, SQL, canon, presentación visual ni contrato HTTP salvo
el cableado mínimo indispensable. No se habilita un doble DEMO en la raíz
productiva y no se usan cookies ni almacenamiento del navegador como
autoridad.

## Camino crítico posterior

```text
composición raíz y propiedad de recursos
→ matriz TLS/mTLS viva
→ misma web definitiva sin adaptadores DEMO
→ E2E HTTP completo contra PostgreSQL 18
→ conformidades RRHH, DPD y Sistemas
```

No se abre O5, O6 ni otro módulo hasta cerrar O4-05.

## Límites conocidos

- Las pruebas globales ejecutadas desde algunos worktrees conservan el fallo
  heredado de detección de raíz en `internal/app/bootstrap`; el flujo CI en un
  checkout normal está verde y CT-000046 no modifica ese paquete.
- CT-000046 no ejecuta una segunda instalación PostgreSQL: reutiliza la
  evidencia real de CT-000045 y reserva la unión adaptador-fachada para el E2E
  de la composición.
- No hay autorización para datos reales, despliegue ni producción.
