# Decisión F0-H0a: aislar la autoprueba sintética de H0

Fecha: 1 de agosto de 2026.
Estado: **integrada en `eb21fdd`; CI técnica en curso y producción NO-GO**.

## Hallazgo

La primera ejecución real de A1 descubrió un defecto de composición en el
runner H0 integrado en `a0d63df`. El snapshot A1 se capturaba y copiaba
correctamente, pero el runner ejecutaba después, de forma incondicional, la
autoprueba sintética del propio arnés.

Esa autoprueba usa deliberadamente las rutas `M010` y `T010`, las sustituye por
dos componentes sintéticos y al terminar elimina los ficheros y sus
directorios. En H0 esos directorios no existían previamente; en A1 son
precisamente la clausura real que debe ejecutar el siguiente paso. Por ello el
wrapper A1 no podía copiarse y la etapa terminaba no cero, con rollback y cero
residuos.

El fallo no está en los validadores A1 ni autoriza a ocultarlo desde SQL. H0
probó correctamente su etapa sintética, pero no demostró que la misma
autoprueba dejara intacta una clausura real posterior.

## Corrección mínima

La autoprueba sintética pertenece exclusivamente a H0. Las etapas A1--T2 ya
ejecutan su propia clausura real mediante `ejecutar_etapa_dormida_f0`. El
runner debe invocar `probar_etapa_dormida_sintetica_f0` solo cuando
`etapa=H0`.

La corrección H0a se limita a una guardia explícita alrededor de esa llamada.
No mueve funciones, no cambia las rutas, no restaura ficheros desde nombres
temporales y no introduce una segunda copia del snapshot. Así evita ampliar la
frontera de confianza y conserva intacta la autoprueba nominal y de error que
H0 ya acredita.

## Write-set y orden

H0a puede modificar exclusivamente:

```text
deploy/postgresql/autorizacion_atestada_v3/
  probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh
```

Los dos auxiliares privados y el capturador permanecen byte a byte inmutables,
con las huellas cerradas en H0. El runner continúa con un máximo de 550 líneas
y conserva sus tres SHA-256 literales.

El DAG queda temporalmente:

```text
H0 -> H0a -> A1
```

H0a es una corrección del primer escritor H0 descubierta por su primer
consumidor; no es un nuevo componente funcional. La decisión posterior
[H0b](decision_f0_h0b_r0_sintetico_c2_2026-08-02.md) sustituye la reserva de
escritura: H0b puede modificar runner y arnés y debe partir de la huella H0a;
después de H0b, I0 es el único escritor posterior autorizado del runner.

## Matriz obligatoria

Antes de confirmar H0a se exige:

1. `bash -n`, ShellCheck, límites, `git diff --check` y Gitleaks;
2. huellas inmutables de auxiliar SQL, auxiliar operativo y capturador Go;
3. tres ejecuciones limpias `--etapa H0`, conservando su autoprueba sintética
   nominal y de error;
4. ejecución combinada con los dos componentes A1 candidatos, que debe usar
   sus rutas reales y terminar con rollback y línea base exacta;
5. un componente A1 de prueba que falle después de crear un objeto debe dejar
   resultado no cero, rollback total y cero residuos;
6. ausencia final de contenedores, etiquetas, temporales, sesiones, roles y
   objetos F0;
7. dos revisiones independientes sobre la huella congelada del runner.

El candidato A1 no se confirma ni se integra junto con H0a. Solo proporciona
la prueba de integración virtual; después de integrar H0a, A1 repite su matriz
completa desde la rama estable corregida.

## Alcance

H0a no añade SQL productivo, API, permiso, rol, tabla, autoridad ni dato. No
cambia las métricas. Obtuvo dos revisiones independientes `GO`,
`P0=P1=P2=0`, y queda documentada en la
[evidencia reproducible](revisiones/revision_f0_h0a_guardia_autoprueba_2026-08-01.md).
A1 está desbloqueada para repetir su matriz desde `eb21fdd`; producción
continúa en `NO-GO`.
