# Relevo de sesión: cierre de CT-000044

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

CT-000044 está integrado localmente y dispone de doble `GO` independiente:

- detalle y cuadro materializados una sola vez;
- capacidad VEC consumida antes de cualquier lectura de negocio;
- identidad corporativa revalidada;
- acción, recurso, finalidad, ámbito y contexto cotejados antes de leer;
- control causal independiente por familia;
- prueba durable CT43A y efectos del cursor en la misma transacción;
- cuatro SQLSTATE transitorios preservados;
- cuatro carreras causales reproducidas;
- token claro ausente de persistencia, registros y argumentos de procesos;
- paquete real `UP/DOWN`, ACL, huella, barreras y retirada segura.

Documento probatorio:

```text
docs/portal_vec/revisiones/o4_05_revision_motor_consultas_ct_000044_2026-07-29.md
```

Commits estables:

```text
2f7e9e2  detalle
bf21034  cuadro
ab6dce6  control causal
165e390  motor atómico
54a75ec  carreras
7412a01  paquete integral
8d2fe99  seguridad del ayudante y entorno
9a237bb  cursores fuera de argv
b442b02  comprobador de fugas cerrado
```

## Barreras

La línea base anterior es `23/7`. CT-000044 avanza de forma atómica a:

```text
control_migracion_cobertura_o4 = 24
control_migracion_consultas_rrhh = 8
```

CT-000045 deberá exigir exactamente `24/8`, avanzar sus propias barreras y
bloquear la reversión de CT-000044. No se reutiliza un número ni se modifica
una migración ya integrada.

## Métricas oficiales

| Ámbito | Estado |
| --- | --- |
| Contratación temporal | `22/46`, 48 % |
| O4-05 | `3/5` hitos |
| Bolsa productiva | `1/14`, 7 % |
| Producción | `NO-GO` |

CT-000044 cuenta porque cierra una pieza completa, instalable, reversible,
probada e integrada. Los componentes parciales previos no se contabilizaron.

## Siguiente corte: CT-000045

Objetivo: exponer el motor privado mediante dos fachadas nominales mínimas,
una para cuadro y otra para detalle.

Condiciones:

1. acción, finalidad, audiencia, módulo y tipo de recurso son constantes de
   cada fachada;
2. el llamador solo aporta el alcance y el canon funcional estrictamente
   tipados, más una capacidad VEC nueva;
3. cada fachada inicia o exige una transacción exterior
   `SERIALIZABLE READ WRITE`;
4. el rol runtime obtiene `EXECUTE` solo sobre las fachadas, nunca sobre el
   motor ni sus tipos privados;
5. ausente, ajeno, denegado y versión incorrecta son indistinguibles;
6. no se aceptan identidad, rol, organización ni finalidad desde JSON,
   cookies, cabeceras libres o navegador;
7. no se crea una segunda auditoría, identidad, autorización o sesión;
8. `DOWN` usa `RESTRICT`, exige barreras exactas y no elimina historia.

Antes de programar, el productor debe cerrar un diseño acotado de firmas,
roles, barreras, errores y pruebas. Un revisor distinto valida ese diseño.

## Después de CT-000045

Orden obligatorio:

```text
CT-000045 fachadas
→ adaptador PostgreSQL Go
→ composición raíz y propiedad de recursos
→ matriz TLS/mTLS viva
→ misma web definitiva sin adaptadores DEMO
→ E2E HTTP completo
→ conformidades RRHH, DPD y Sistemas
```

No abrir O5/O6, Bolsa, Dietas, Autofirma, Oracle, MCP ni reconstruir la web
antes de cerrar O4-05. Las dependencias comunes se implementan de forma
acotada y se vuelve inmediatamente a este camino crítico.

## Reglas para continuar

- castellano coherente e i18n;
- arquitectura hexagonal;
- denegación predeterminada y privilegio mínimo;
- cero cookies o almacenamiento web como autoridad;
- puertos y adaptadores intercambiables;
- trazabilidad completa y minimizada;
- PostgreSQL 18.4 real, no dobles como prueba de producción;
- ningún fichero de 800 líneas o más;
- productor distinto de revisor e integrador;
- commits pequeños; código, pruebas y documentación juntos;
- no publicar ni desplegar hasta tener revisión independiente y CI verde.

## Publicación pendiente de este corte

Antes de considerar publicado CT-000044:

1. confirmar la actualización documental;
2. ejecutar las puertas estáticas sobre la rama estable;
3. enviar la rama integradora;
4. comprobar GitHub Actions verde;
5. verificar que el árbol queda limpio y que ningún contenedor de prueba
   permanece activo.
