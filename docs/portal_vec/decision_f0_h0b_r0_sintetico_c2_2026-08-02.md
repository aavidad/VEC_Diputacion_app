# Decisión F0-H0b: R0 sintético para C2

Fecha: 2 de agosto de 2026.

Estado: **estructura aislada integrada en `ad8b170`; comportamiento funcional
pendiente del commit 2**.

## Motivo

C2 debe instalarse antes de R0 y denegar en ejecución mientras los grupos y
el `LOGIN` técnico no existan. Su prueba positiva necesita identidades
sintéticas canónicas, pero el arnés H0 solo las preparaba desde C3 y usaba
valores predeterminados incompatibles con el contrato final.

H0b corrige únicamente el arnés. No amplía M080/T080, no crea roles
productivos y no cambia una métrica funcional.

## Write-set

La [enmienda del auxiliar privado H0b](enmienda_f0_h0b_auxiliar_privado_r0_2026-08-02.md)
prevalece tras el doble `NO-GO` del primer candidato. El write-set corregido
es:

```text
deploy/postgresql/autorizacion_atestada_v3/
  probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/
  arnes_fuente_corporativa_contexto_actor_v1.sh
  arnes_r0_sintetico_h0b_fuente_corporativa_contexto_actor_v1.sh
```

Los cambios de huella de los auxiliares se actualizan en el runner dentro del
mismo commit. No se modifica otro componente, migración o prueba SQL.

## Estado de implementación

`2d27303` recibió doble `NO-GO`, `P0=0, P1=1, P2=0`, porque H0a solo
contrastaba por SHA-256 sus M010/T010 sintéticos después de copiarlos. La
corrección `ca46ba9` añadió validación SQL previa para M010, T010 nominal y
T010 de error, y obtuvo doble `GO`, `P0=P1=P2=0`, con H0 ×3, A1, C1 y cero
residuos sobre PostgreSQL 18.4.

La estructura quedó integrada por *squash* en `ad8b170`: H0a, snapshot único,
dos raíces y tercer auxiliar están cerrados. El flujo descrito a continuación
permanece dormido hasta el commit funcional 2. Véase la
[evidencia reproducible](revisiones/revision_f0_h0b_estructura_aislada_2026-08-02.md).

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

H0b conserva además la autoprueba H0a nominal/error en la raíz base. Las
integraciones virtuales usan la segunda raíz de la captura coherente fijada en
la enmienda, por lo que nunca sobrescriben ni retiran M010/T010 reales.

La integración estructural no incrementa métricas: F0 sigue en `10/23`,
O4-05 en `3/5`, Contratación en `24/46`, Bolsa productiva en `1/14` y
producción en `NO-GO`.
