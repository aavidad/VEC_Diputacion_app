# O4-05: revisión final del motor de consultas CT-000044

Fecha: 29 de julio de 2026.

## Resultado

**GO técnico independiente** para CT-000044 en PostgreSQL 18.4.

Este cierre acredita el motor privado y atómico de cuadro y detalle RRHH. No
autoriza producción ni sustituye las fachadas CT-000045, el adaptador Go, la
composición raíz, la matriz TLS viva ni el E2E HTTP.

## Serie integrada

| Corte | Commit estable | Contenido |
| --- | --- | --- |
| CT-000044 detalle | `2f7e9e2` | Materialización de detalle con versión actual o exacta. |
| CT-000044 cuadro | `bf21034` | Colección única, orden total, corte y límite exactos. |
| CT-000044A | `ab6dce6` | Tipos privados, guardas y control causal por familia. |
| CT-000044B | `165e390` | Consumo VEC, ligadura prelectura, cierre CT43A y efectos atómicos. |
| CT-000044C | `54a75ec` | Cuatro carreras causales del motor completo. |
| Paquete | `7412a01` | `UP/DOWN`, ACL, catálogo, barreras `24/8` y runner integral. |
| Seguridad de pruebas | `8d2fe99`, `9a237bb`, `b442b02` | Cero cursores en `argv`, ayudantes sin ventana pública y comprobadores que fallan cerrados. |

## Contrato acreditado

El orden ejecutado dentro de una única transacción exterior
`SERIALIZABLE READ WRITE` es:

```text
validar forma
→ consumir una capacidad VEC-AD-3 nueva
→ revalidar identidad corporativa viva
→ cotejar acción, recurso, finalidad, ámbito y contexto antes de leer
→ bloquear la fila causal de la familia
→ releer familia, cursor, revocación y consumo
→ fijar el corte
→ materializar una sola colección
→ cerrar la prueba durable CT-000043A
→ persistir consumo, familia e hijo de cursor
→ COMMIT exterior único
```

Una capacidad emitida para una consulta, expediente u organización no puede
autorizar otra. El cruce se rechaza con `42501` después del consumo y antes de
resolver cursores, consultar la publicación o materializar datos. La
transacción revierte todos los efectos.

Los errores transitorios `40001`, `40P01`, `55P03` y `57014` se conservan sin
normalizarlos. Un resultado ambiguo no autoriza reintentar una capacidad ni un
cursor.

## Cursores y concurrencia

- el token claro solo vive como variable o salida transitoria;
- la persistencia contiene su SHA-256 y prueba canónica, nunca el token;
- familia, identidad, ámbito, filtros, límite, corte y TTL quedan ligados;
- una continuación consume el padre una sola vez;
- dos continuaciones concurrentes dejan un único ganador;
- una revocación confirmada antes del motor produce el `40001` causal
  esperado y cero efectos;
- si el motor confirma primero, la revocación espera su `COMMIT`;
- un rollback de revocación no genera un rechazo falso;
- dos familias distintas usan filas causales distintas y no comparten un
  cerrojo local.

Los runners tampoco incluyen el token en los argumentos de `docker`, `grep`,
`rg`, `awk` ni otro proceso externo. Los valores se transfieren mediante
entorno transitorio y `ENVIRON`; el comprobador de fugas considera inseguro
un secreto vacío o cualquier fallo distinto de «ausencia acreditada».

## Paquete y retirada

La migración real:

- exige PostgreSQL `18.4` y UTF-8;
- avanza las barreras `23/7 → 24/8` mediante comparación y sustitución;
- instala nueve componentes en una sola transacción;
- mantiene privadas once funciones, siete tipos base y una tabla causal con
  RLS forzada;
- verifica propietario, ACL, configuración, comentarios, restricciones,
  índices, disparadores, políticas, dependencias salientes y derivas;
- produce la misma huella tras tres altas frescas y dos retiradas;
- usa `DROP ... RESTRICT`, nunca `CASCADE`;
- bloquea la retirada si existe estado causal durable, una dependencia futura,
  una deriva semántica o una barrera CT-000045.

La huella solo incluye dependencias salientes propias de CT-000044. Las
dependencias entrantes futuras quedan protegidas por `RESTRICT` y por el
rollback transaccional; incorporarlas a la huella haría variar el manifiesto
sin cambiar la semántica del paquete.

## Pruebas reproducidas

Productor, integrador y revisores independientes ejecutaron:

- runner focal CT-000044A en PostgreSQL 18.4;
- runner focal CT-000044B en PostgreSQL 18.4;
- las cuatro carreras CT-000044C;
- runner integral CT-000044 con los nueve componentes omitidos y mutados;
- `UP → DOWN → UP`, reentradas y tres altas con huella estable;
- ACL privada, RLS, comentarios, columnas, índices, políticas, publicaciones
  y dependencias futuras hostiles;
- barrera CT-000045 y retirada con estado causal durable;
- ausencia de token claro en tablas, registros y argumentos de procesos;
- `bash -n`, ShellCheck, `git diff --check`, tamaños DEC-051 y Gitleaks focal.

Resultados independientes finales:

| Alcance | P0 | P1 | P2 | Resultado |
| --- | ---: | ---: | ---: | --- |
| Paquete exacto `7412a01` | 0 | 0 | 2 no bloqueantes | GO |
| Carreras y correctores acumulados hasta `b442b02` | 0 | 0 | 0 | GO |
| Integración completa en rama estable | 0 | 0 | 0 | GO técnico |

Todos los ficheros permanecen por debajo de 800 líneas.

## Mejoras no bloqueantes registradas

1. Extraer en un corte futuro una primitiva pura común para construir el canon
   de recurso/contexto que hoy está sellado por pruebas entre CT43 y CT44.
2. Complementar la detección estructural de CT-000043A con una huella exacta
   desde una migración posterior, sin modificar retrospectivamente CT43A.

No se abren ahora porque exigirían tocar primitivas ya cerradas y no corrigen
un hueco funcional o de seguridad vigente.

## Métrica y continuación

CT-000044 incrementa Contratación temporal a `22/46`, un `48 %` redondeado.
O4-05 conserva `3/5` hitos oficiales porque la vertical exterior todavía no
está completa. El siguiente corte obligatorio es CT-000045: fachadas
nominales privadas de cuadro y detalle, con privilegio mínimo y sin autoridad
alternativa.

Producción permanece en **NO-GO**.
