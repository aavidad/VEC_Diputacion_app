# Enmienda O3a V5 CND V2: falso verde de C18

Fecha: 14 de agosto de 2026.

Tarea: `O3A-V5-CND-ESTADOS-CAUSALES-V2-FALSO-VERDE-C18`.

Estado: candidato técnico local. Requiere revisión funcional y de seguridad
independientes; no acredita estabilidad, toolchain, O4, publicación ni CI.

## Base y criterio único

La base exacta es
`a997acbb9a51c02168f1648225c0e69c84e648d4`, con padre
`74f249587d5c0da41092a2c017ccd6cb54817248` y árbol
`38473c53cd1838113e549bf53233136a7e1da3cd`.

Dos revisiones independientes del corte base localizaron el mismo falso verde
en `probarVueltaTardiaExternaO3aM38`. La condición agrupaba la preparación y
la primera escritura de CONTROL, pero devolvía únicamente el `err` asignado
por la preparación. Si esta terminaba verde y la escritura fallaba, la función
podía devolver nil y C18 podía concluir 0 en lugar del estado causal 111.

El criterio único de V2 es asignar y propagar ese error de escritura. No se
cambia qué operación se ejecuta, su orden, la limpieza diferida, los estados,
los casos, los cien procesos C21, los plazos ni ningún código productivo.

## Write-set exacto

```text
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas_adversas.go
tools/o3a_v5_conductor/fuentes_v5.tsv
tools/o3b_p7_conductor/fuentes.tsv
tools/o3c_p6_conductor/fuentes.tsv
docs/portal_vec/enmienda_o3a_v5_cnd_estados_causales_v2_c18_2026-08-14.md
```

G7b es test-only por `//go:build ignore`. Los tres ledgers son las autoridades
vivas que consumen exactamente ese fuente; su cambio es mecánico. No se toca
G7a, producción, conductores, workflow, Go/Docker, oráculos, evidencia
histórica, credenciales, estado transversal ni métricas.

## Corrección cerrada

La secuencia queda explícita y fail-closed:

1. `prepararFixtureO3aM38` asigna su resultado a `err`; un fallo se devuelve;
2. solo si la preparación fue verde se llama a
   `escribirControlPruebaO3aM38`;
3. el resultado de esa escritura se asigna a `err`; un fallo se devuelve;
4. el `defer` conserva `errors.Join(err, limpiarFixtureO3aM38(f))`.

El despachador C18 ya convierte cualquier error de esta función en
`estadoVueltaTardiaExternoO3aM38`, valor 111. La corrección no acepta 111 como
éxito: el conductor continúa esperando exclusivamente estado 0 para C18.

Un barrido de los diez fuentes no encontró otra condición con una segunda
operación falible descartada y retorno del error anterior. El patrón gemelo de
`probarAliasRetiradaExternaO3aM38` ya quedó corregido en la base.

## Presupuesto y huellas

G7b queda en 744 líneas, por debajo de la parada local 750 y del tope duro 800,
con SHA-256
`5ec6be1ccb917ec2908bfa75d2d947ae7a107c39044239166cc10e1067d7677e`.
G7a permanece byte a byte en 742 líneas y SHA-256
`4631384a9ecd85b09386aed312f16a52c56234795929219d2770b25234f70ec7`.
Los tres ledgers vivos fijan ambas huellas.

## Puertas requeridas

1. `gofmt`, `go vet` y builds normal/race de los diez fuentes;
2. mutante efímero que fuerza solo la primera escritura de C18 a fallar y
   exige estado 111, sin alterar el candidato;
3. una corrida O3a V5 normal/race como `orquesta`, sin retry;
4. conductores O3b P7 y O3c P6 con Go 1.26.5 y `flock --close`;
5. calidad global proporcional, diff-check y Gitleaks;
6. doble revisión independiente del SHA exacto.

Una puerta roja se conserva y no se repite para buscar verde. No se autoriza
retry, `SKIP`, sleep, tolerancia, exclusión de FD ni reclasificación de un
estado negativo.

## Límites y relevo

V2 corrige el falso verde observado, pero una ejecución verde no elimina la
intermitencia C21 sellada sobre `74f2495`. Los estados causales siguen siendo
un corte diagnóstico: el primer estado negativo futuro asignará el propietario
de otro parche; no se aprobará por mayoría.

El P2 documental de vulnerabilidades tiene candidato disjunto propio y tampoco
queda acreditado por este corte. No se autoriza integración, push, despliegue,
producción, credenciales ni modificación de porcentajes.
