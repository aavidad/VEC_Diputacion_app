# Decisión F0-H0b: R0 sintético para C2

Fecha: 2 de agosto de 2026.

## Motivo

C2 debe instalarse antes de R0 y denegar en ejecución mientras los grupos y
el `LOGIN` técnico no existan. Su prueba positiva necesita identidades
sintéticas canónicas, pero el arnés H0 solo las preparaba desde C3 y usaba
valores predeterminados incompatibles con el contrato final.

H0b corrige únicamente el arnés. No amplía M080/T080, no crea roles
productivos y no cambia una métrica funcional.

## Write-set

```text
deploy/postgresql/autorizacion_atestada_v3/
  probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/
  arnes_fuente_corporativa_contexto_actor_v1.sh
```

El cambio de huella del auxiliar se actualiza en el runner dentro del mismo
commit. No se modifica otro componente, migración, prueba SQL o documento.

## Comportamiento exacto

El arnés ejecuta dos subensayos distintos y en este orden:

1. sin R0, instala la clausura M010…M080 —y M090 cuando la etapa focal sea
   C3—, comprueba que los objetos se crean y que invocar C2 deniega con
   `42501` antes de cambiar checkpoint, advisory o historia; después revierte
   y acredita línea base;
2. para la prueba focal completa, `etapa_necesita_r0_f0` incluye C2 y C3, crea
   fuera de la transacción dormida R0 y los `LOGIN` sintéticos canónicos y
   ejecuta sus casos positivos y negativos.

El primer subensayo demuestra que R0 no es precondición de instalación. El
segundo demuestra que la identidad técnica se reacredita en ejecución. Crear
el fixture antes del primer subensayo o reutilizarlo entre ambos es un fallo
del arnés.

Los tres grupos se crean como `NOLOGIN NOINHERIT`, `NOSUPERUSER`,
`NOCREATEDB`, `NOCREATEROLE`, `NOREPLICATION` y `NOBYPASSRLS`, con
`CONNECTION LIMIT -1`, contraseña, caducidad y configuración nulas. No tienen
descripción, etiqueta de seguridad, fila en `pg_db_role_setting`, membresía
superior ni uso como otorgante.

Cada `LOGIN` sintético es `LOGIN INHERIT`, sin atributos administrativos,
contraseña, caducidad y configuración nulas, sin descripción, etiqueta de
seguridad ni fila en `pg_db_role_setting`.

Cada arista usa explícitamente:

```text
ADMIN FALSE
INHERIT TRUE
SET FALSE
grantor = pg_database.datdba de la base efímera actual
```

Ese `datdba` debe seguir correspondiendo a un rol superusuario y ser distinto
del `LOGIN` y de los tres grupos. No se acepta como equivalente cualquier otro
superusuario.

Se conservan identidades separadas para publicador, revocador, despachador,
cruzado, adicional y sin rol. Las variantes no canónicas existen solo para
pruebas negativas; ninguna es un valor predeterminado de producción.

La preparación y retirada del segundo subensayo se ejecutan fuera de la
transacción dormida. El ensayo focal sigue usando una única transacción que
termina en `ROLLBACK`, y el arnés acredita después que roles, membresías,
ajustes por rol, objetos, sesiones y temporales vuelven a la línea base exacta.

## Cierre

H0b exige ShellCheck, autopruebas del analizador, snapshot y limpieza exactos,
límites, Gitleaks y revisión independiente. El runner conserva como máximo
550 líneas.

Antes de cerrar H0b se ejecutan dos integraciones virtuales de C2: una nominal
y otra con error posterior a crear R0. Ambas prueban instalación previa sin
roles, catálogo exacto de grupos, `LOGIN`, aristas y otorgante, C1 sin R0,
rollback y retorno byte a byte a la línea base. La clausura temporal usada para
estas integraciones no se versiona ni amplía el write-set de H0b. La etapa C2
completa se ejecutará después, cuando M080/T080 incorporen la reacreditación
R0.
