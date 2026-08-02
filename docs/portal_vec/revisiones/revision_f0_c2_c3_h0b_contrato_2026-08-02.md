# Revisión F0-C2/C3/H0b: contrato de identidad técnica y publicación

Fecha: 2 de agosto de 2026.

## Resultado

El contrato documental obtuvo dos revisiones independientes finales `GO`,
ambas con `P0=P1=P2=0`. Este cierre autoriza implementar H0b; no acredita aún
el runner, C2, C3, R0 ni producción.

## Bloqueos detectados y corregidos

Las revisiones previas emitieron `NO-GO` porque el primer diseño preparaba R0
antes de instalar M080/M090 y, por tanto, no demostraba la instalación sin
roles. También detectaron:

- H0b ausente del DAG y reservas de escritura incompatibles con H0a/D2d;
- otorgante expresado como cualquier superusuario, no como `datdba` literal;
- fixture R0/`LOGIN` incompleto;
- propiedad ambigua de las membresías entre R0 y Sistemas;
- retirada inversa inconsistente;
- ACL C3 abierta a roles inexistentes y sin estado pre/post exacto del esquema;
- atribución incorrecta del subensayo sin R0 a T080/T090;
- unicidad global falsa de la dependencia normal del centinela.

El contrato final corrige cada punto y fija:

```text
F0 completo hasta C3
→ R0 crea los tres grupos
→ Sistemas provisiona LOGIN y membresía
→ M5 → M6 → M7
```

## Evidencia documental

H0b ejecuta primero la clausura M010…M080 —y M090 para C3— sin R0, comprueba
denegación `42501` sin efectos y revierte. Después crea un fixture sintético
canónico y ejecuta la clausura focal positiva y adversarial.

C2 reacredita en cada llamada, antes de checkpoint, advisory, replay o
historia:

- los tres grupos `NOLOGIN NOINHERIT` exactos;
- el `session_user` como `LOGIN INHERIT` mínimo y `role=none`;
- una única arista directa con opciones `FALSE/TRUE/FALSE`;
- `pg_auth_members.grantor = pg_database.datdba` de la base actual;
- ausencia de membresía adicional, transitiva, cruzada o despachadora.

C3 se instala con R0 ausente. Parte de una función C2 con ACL propietaria y
añade únicamente su ejecución al propietario ContextoActor. La ACL del esquema
conserva propietario V3 y propietario de Contratación y añade únicamente al
propietario ContextoActor, siempre con otorgante V3 y sin `grant option`.

La retirada completa queda fijada como:

```text
M7 → M6 → M5 → Sistemas desaprovisiona LOGIN/membresía → R0 → F0
```

## Comprobaciones

- dos revisores distintos del autor documental: `GO`, `P0=P1=P2=0`;
- semántica de `NOINHERIT` con arista `INHERIT TRUE, SET FALSE` reproducida
  por un revisor en PostgreSQL 18.4 real;
- llamada anidada conserva `session_user` y `role=none`;
- `SET ROLE` del grupo queda denegado con `42501`;
- `git diff --check` limpio;
- todos los documentos por debajo de 800 líneas;
- cero rutas privadas, credenciales o datos personales añadidos.

No se ejecutó todavía la etapa H0b porque el runner actual no implementa esta
decisión. Su implementación, prueba PostgreSQL 18.4 y revisión independiente
son el siguiente corte obligatorio.
