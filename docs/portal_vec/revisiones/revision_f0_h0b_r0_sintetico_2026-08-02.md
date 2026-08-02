# Revisión F0-H0b: R0 sintético para C2

Fecha: 2 de agosto de 2026.

## Primer candidato: NO-GO

El commit productor `99491d3` no se integró ni se publicó. Dos revisiones
independientes emitieron `NO-GO`. El recuento consolidado es
`P0=0, P1=2, P2=0`:

1. eliminó la autoprueba H0/H0a nominal y de error en vez de añadir los dos
   subensayos H0b;
2. trasladó Docker, sesiones, fixture R0 y decisiones de ejecución al auxiliar
   SQL, contra la frontera D2c.

La objeción inicial a versionar la plantilla virtual quedó retirada. Es
correcto versionar una plantilla determinista dentro del arnés; lo prohibido
es confirmar componentes ficticios bajo las rutas catalogadas M080/T080 o
presentarlos como implementación C2.

## Evidencia verde que no basta para integrar

El candidato superó sobre PostgreSQL 18.4 fijado por digest:

- H0 tres veces y C1 real;
- instalación sin R0 y rechazo `42501` exacto;
- diez roles, siete aristas y `grantor=datdba` exactos;
- integración virtual C2 nominal y variante con error posterior;
- rollback, catálogo, checkpoint, sesiones, roles y temporales sin residuos;
- `bash -n`, ShellCheck, `git diff --check`, SHA-256 y Gitleaks.

El write-set era exacto y los tamaños eran runner `532/550` y auxiliar SQL
`798/800`. Estas puertas funcionales no compensan una regresión H0a ni una
violación de la frontera de confianza.

## Hallazgo de implementabilidad

Devolver al runner toda la ejecución incompatible añade unas 237 líneas;
restaurar H0a añade unas 35. El auxiliar SQL solo libera unas 238 líneas y el
total combinado excede los máximos. Compactar el código consumiría la reserva
I0 y reduciría la revisabilidad.

Además, la restauración literal sobre el árbol candidato colisionaría con los
M010/T010 reales que H0b añadió al snapshot H0. El arreglo correcto necesita
un snapshot aislado y no una sobrescritura seguida de restauración.

La [enmienda H0b](../enmienda_f0_h0b_auxiliar_privado_r0_2026-08-02.md)
autoriza un tercer auxiliar privado y dos raíces de snapshot disjuntas. Debe
obtuvo después doble `GO` documental, `P0=P1=P2=0`. El candidato corregido
puede programarse y repetirá toda la matriz; ninguna evidencia del commit
rechazado se hereda como cierre.

## Estado

H0b continúa abierta y el contador F0 permanece en `10/23` (43 %).
Contratación permanece en `24/46` (52 %), O4-05 en `3/5`, Bolsa productiva en
`1/14` (7 %) y producción en `NO-GO`.
