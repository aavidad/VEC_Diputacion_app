# Decisión F0-H0b/C4b-2/G2-O/O3a-P0: margen del runner

Fecha: 9 de agosto de 2026.

Estado: **AUTORIZADO DOCUMENTALMENTE tras doble GO; permite publicar este
contrato y, solo después de CI 5/5, producir O3a-P0 con su write-set exacto**.

## Motivo y resultado único

O2b está publicado hasta `c1ca5aa64221ad6a1895b8be7563a0d43ff59c9e` y
su CI `31287803830` terminó con cinco de cinco puertas verdes. La siguiente
fase funcional es O3a, `Start` y mapa FD, pero el runner ocupa exactamente
800 líneas. Añadir rutas, huellas y fuentes Go exigiría minificar controles o
mezclar producción y pruebas.

O3a-P0 tiene una sola responsabilidad observable: **trasladar literalmente
ocho funciones Shell ya existentes desde el runner a su auxiliar D2d ya
capturado**, sin cambiar el orden de sus invocaciones, estados de salida,
salida estándar o de error, comandos, efectos, fuentes Go ni binario en los
modos gobernados. El runner debe pasar de 800 a 702 líneas y D2d de 164 a
264.

La ubicación, `BASH_SOURCE` y el instante de definición de esas funciones sí
cambian por naturaleza. Ningún cuerpo trasladado consulta `BASH_SOURCE`,
`caller`, `FUNCNAME`, trampas `DEBUG`/`ERR` ni se invoca antes de cargar D.
P0 prohíbe añadir esas observaciones y no promete identidad de una traza
interna de definiciones fuera de los modos contratados.

O3a-P0 no implementa `Start`, Bash, mapa FD, pidfd, señales, `/proc`, ticket,
plazos, `CONT`, `Wait`, terminal, modo operativo ni ninguna conducta O3a. Si
el resultado no es una reorganización byte a byte reversible, se detiene.

## Autoridad y base exactas

| Concepto | Autoridad |
| --- | --- |
| Base de código y documentación | `c1ca5aa64221ad6a1895b8be7563a0d43ff59c9e` |
| Material O2b | `d86aea8b4ed2b9fffbf74ef04cd90f397b017a55` |
| Evidencia O2b | `4b39265405957b47b909f0b9e1bc4960c38f4011` |
| Publicación y CI de O2b | `c1ca5aa`, CI `31287803830`, 5/5 |

La especificación conjunta G2-O de 5 de agosto permanece en `NO-GO`
documental y solo es antecedente. Esta decisión no la reactiva ni autoriza
O3a. El futuro código P0 deberá nacer del commit que publique esta decisión,
no directamente de una rama candidata anterior.

## Ledger anterior obligatorio

| Alias | Unidad | Líneas | SHA-256 |
| --- | --- | ---: | --- |
| R | Runner H0 | 800 | `8e15443b120dc68721aa4cc0959610ca393af44d842f0e07ed7e0b18873fc059` |
| D | D2d, operaciones runner | 164 | `039b75dd15a2888798c7f257c46fdbb97587cbdd4a6519e11cb043cce0e72e5e` |
| G1 | Supervisor privado | 692 | `6b7f93b8b43c1040cc4ae2b6322c4e99e914eee415475e3fd50bf294b5a17afb` |
| G4 | Control previo O2b | 404 | `2befe2a4c16fc7a57aacd421ea6c8419ab49160bb2ae0d0eb6f03786194aa744` |
| G5 | Pruebas O2b | 507 | `10ccaf8347bfcaa5f3990b75b4c9becd62cd39b60249b628af6c7a1fc6bc8867` |
| G2 | Lector y codec | 798 | `01acb818e9abefcbfe4c279bb0dd5e3317bf03f082f1ed3fba4f257c5642866b` |
| G3 | Sobre S0 | 431 | `d608868ecb2cb753876f488b522975e05af06c013c82222959be5d85100c3633` |
| B | Binario Go reproducible | — | `6153f03a93c0a2618fdaf922443004244aa3bec7cbe9074466b22935c693edd0` |

El capturador queda en 799 líneas y SHA
`4a967fd13bac213ea7ebf7316af98dcc9a9dfb39b9b3b28f68e0c91958878902`.
El adaptador M38 queda en 527 líneas y SHA
`98d22a302bfd8ad3964b9135ce78c655f7a31171088ad9c5c49c285f647a8cb7`.

## Write-set material cerrado

Solo pueden cambiar estas dos rutas:

1. `deploy/postgresql/autorizacion_atestada_v3/probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh`;
2. `deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/operaciones_runner_fuente_corporativa_contexto_actor_v1.sh`.

No se crea, renombra ni elimina ningún fichero. Todo Go, capturador, adaptador,
SQL, migración, documentación y resto del repositorio queda byte a byte.

En R solo se permiten:

- retirar los dos bloques literales enumerados más abajo;
- sustituir una vez la huella D anterior por la huella D posterior exacta.

En D solo se permite insertar los 98 renglones originales en el mismo orden
relativo, inmediatamente después de `declare -g temporales` y antes de
`derivar_repo_base_h0_f0`, y añadir exactamente estas dos directivas locales
intercaladas en las posiciones fijadas a continuación:

```bash
# shellcheck disable=SC2154  # Variable aportada por el runner acreditado.
# shellcheck disable=SC2034  # Variable consumida por el runner acreditado.
```

La primera queda inmediatamente antes de `archivo()`. La segunda queda
inmediatamente antes de la asignación final `wrapper_activo=''` de
`ejecutar_etapa_dormida_f0()`. No se desplaza ni modifica la directiva
preexistente que precede a `derivar_repo_base_h0_f0()`.

No se añade ninguna otra directiva, comentario, excepción ShellCheck, variable,
rama, retorno, redirección, comando o espacio dentro de los 98 renglones
originales. Las dos directivas nuevas no alteran ejecución y son necesarias:
sin ellas ShellCheck 0.11.0 reproduce `SC2154` para `contenedor` y `SC2034`
para `wrapper_activo` al analizar D como fichero separado.

## Bloques exactos que se trasladan

Desde R anterior se retiran, completos y sin alterar sus bytes:

1. líneas 319–390, desde `archivo() {` hasta el cierre de
   `ejecutar_etapa_dormida_f0()`;
2. líneas 549–574, función completa
   `probar_etapa_dormida_sintetica_f0()`.

Son exactamente ocho funciones y 98 líneas originales. Sus bloques, sin las
dos directivas locales nuevas, tienen estas huellas:

| Bloque | Líneas | SHA-256 |
| --- | ---: | --- |
| R 319–390 | 72 | `e811e624b7d0399368871de4ea64a05b1ca989eec5e8a7571755d984f583117a` |
| R 549–574 | 26 | `0a49085527dd7bd726759c10222345e8c7a30aca24652b4829ed014712c1a41a` |
| concatenación en ese orden | 98 | `7dd728116bf29f2cdba860a8ab7a74d0b406d99026dbe649297187f27176cafd` |

Los ocho nombres son:

```text
archivo
sql
valor
probar_sqlstate_real_f0
foto_catalogo
etapa_necesita_r0_f0
ejecutar_etapa_dormida_f0
probar_etapa_dormida_sintetica_f0
```

No se traslada `capturar_auxiliares_privados_f0`: esa función captura y carga
D2d, por lo que moverla a D crearía una dependencia circular y es parada
obligatoria.

Todas las funciones trasladadas se invocan después de que R haya capturado,
acreditado y cargado D bajo `VEC_F0_CARGA_PRIVADA=1`. D conserva sus dos
guardas: ejecución directa devuelve 64 y una carga no acreditada devuelve 64
sin definir la API privada.

El marcador de entorno no es autoridad por sí solo. La acreditación exige en
conjunto ruta privada, captura sin seguimiento de enlaces, modo/propietario,
huella D exacta y marcador de un solo uso creado y retirado por el runner.

El inventario de carga se prueba de forma exacta. Los once nombres previos de
D son:

```text
derivar_repo_base_h0_f0
huella_contenedor_f0
descubrir_contenedor_propio_f0
acreditar_hallazgo_contenedor_f0
acreditar_cidfile_f0
retirar_contenedor_propio_f0
acreditar_snapshot_contenedor_f0
rechazar_snapshot_adverso_f0
probar_snapshot_adverso_f0
comparar_huellas_f0
exigir_salida_f0
```

Inmediatamente antes de cargar D, esos once nombres y los ocho trasladados
deben estar ausentes en el candidato. Inmediatamente después, la diferencia
ordenada del inventario de funciones debe contener exactamente esos diecinueve
nombres, una sola vez y sin sustituir una definición previa. El inventario
final debe coincidir con el de la base después de cargar D. Una búsqueda
estática debe acreditar cero llamada o referencia ejecutable a los ocho
trasladados antes de esa carga; cualquier colisión, nombre adicional o uso
prematuro detiene P0.

## Ledger posterior contractual

| Unidad | Anterior | Delta | Posterior exigido |
| --- | ---: | ---: | ---: |
| R | 800 | -98 | **702** |
| D | 164 | +100 | **264** |
| G1 | 692 | 0 | 692 |
| G4 | 404 | 0 | 404 |
| G5 | 507 | 0 | 507 |
| G2 | 798 | 0 | 798 |
| G3 | 431 | 0 | 431 |
| Capturador | 799 | 0 | 799 |
| Adaptador M38 | 527 | 0 | 527 |

La proyección exacta exige D de 264 líneas y SHA
`681efbbd7f856eb539d1656cffed87c26f48609e65d6d6adf8265c350ae69442`,
y R de 702 líneas y SHA
`e617024a52c4a042971b026d0799816933b489ed4221e9b6147317936d18054c`.
B debe seguir siendo exactamente
`6153f03a93c0a2618fdaf922443004244aa3bec7cbe9074466b22935c693edd0`
en dos builds Go 1.26.5 aislados. Cualquier desviación de estas huellas exige
parar y revisar el contrato, no actualizar un literal.

El manifiesto conserva exactamente nueve entradas y el mismo orden. Solo
cambia la huella de D en la entrada cuarta; las cinco fuentes Go, sus huellas
y el binario permanecen invariantes.

## Prueba reversible obligatoria

Un revisor debe reconstruir fuera del repositorio el par anterior desde el
candidato:

1. localizar cada función por anclas únicas y acreditar ocho de ocho;
2. retirar y acreditar las dos directivas locales exactas;
3. extraer de D los dos bloques trasladados;
4. reinsertarlos en R en sus dos posiciones anteriores;
5. restaurar en R el literal SHA-256 anterior de D;
6. obtener exactamente R de 800 líneas/SHA `8e15443b…` y D de 164
   líneas/SHA `039b75dd…`.

La prueba inversa, aplicada al padre `c1ca5aa`, debe producir byte a byte el
candidato. Un mero `diff` semánticamente parecido no sirve. Cualquier cambio
fuera del traslado, las dos directivas locales exactas o el único literal de
huella es `NO-GO`.

## Puertas materiales

Antes del commit de código deben quedar verdes:

- `bash -n` de R y D;
- ShellCheck de R y todos los auxiliares capturados;
- D ejecutado directamente y cargado sin acreditación: estado 64, sin efectos;
- inventario exacto de diecinueve funciones privadas antes y después de
  cargar D y cero uso prematuro de las ocho trasladadas;
- manifiesto de nueve entradas y captura no-follow;
- `gofmt`, `go vet` y dos builds privados Go 1.26.5 idénticos;
- autoprueba Go completa y modos `--supervisar-m38`/desconocido en 64;
- H0 completo sobre PostgreSQL 18.4 por digest, incluido clasificador
  SQLSTATE, etapa sintética nominal/error, rollback y residuos cero;
- `scripts/verificar_calidad.sh`, `git diff --check`, tamaños y Gitleaks;
- contenedores, procesos y temporales propios residuales: cero.

No se exige la matriz operativa M38 de 39 procesos para acreditar un traslado
que no modifica ese camino; sí se exige que el adaptador, runner y fuentes que
la gobiernan permanezcan por huella salvo R y D. La futura O3a volverá a fijar
sus propias pruebas Linux y mutantes.

## Evolución de la revisión documental

La primera versión tenía 218 líneas y SHA
`5a2e6c33e416daa952c8e66c4c82076cd12dbf353d7873d3edac3c5f7638c9fc`.
Incluía deliberadamente una huella G3 incorrecta para comprobar la revisión,
pero no se consideró aprobable por ello. La revisión funcional acabó
clasificándola `NO-GO`, `P0=0`, `P1=2`, `P2=0`; la de seguridad, `NO-GO`,
`P0=0`, `P1=3`, `P2=2`.

Además del canario, se reprodujo que el traslado literal fallaba ShellCheck,
que la secuencia de aprobación era circular, que faltaba el oráculo exacto del
inventario y que la identidad observable se formulaba de manera demasiado
amplia. Esta versión corrige los cinco puntos. Los dictámenes de la primera
versión no se reutilizan: los nuevos bytes deben recibir una revisión completa
y dos `GO` nuevos.

## Evidencia y secuencia

1. la primera ronda ya emitió los dos `NO-GO` registrados arriba;
2. esta versión corregida recibe una revisión íntegra nueva;
3. se obtiene doble `GO` independiente sobre los mismos bytes corregidos;
4. se cambia solo entonces el estado, se crea el acta y se contrarrevisa el par;
5. se confirma y publica contrato/acta y se espera CI 5/5;
6. se aplica el traslado en un worktree y rama exclusivos;
7. se ejecutan las puertas y se confirma un commit material autónomo;
8. un revisor funcional y otro de seguridad reconstruyen y prueban el commit;
9. tras doble GO, se integra por avance rápido, se actualiza el relevo, se
   publica y se espera CI 5/5;
10. solo entonces se redacta el contrato funcional O3a.

El productor no revisa ni integra su propio commit. La evidencia no amplía el
write-set material; si necesita un artefacto durable, se confirma después en
un commit probatorio separado y ligado al SHA material.

## Paradas duras

Se detiene sin commit si ocurre cualquiera de estos casos:

- R no queda exactamente en 702 líneas o D en 264;
- se toca una tercera ruta material;
- cambia un byte dentro de una función trasladada;
- falta una de las dos directivas locales exactas o aparece otra supresión;
- cambia el orden de carga o de ejecución;
- aparece una observación de `BASH_SOURCE`, `caller`, `FUNCNAME` o trampas de
  depuración dentro de los bloques trasladados;
- cambia el inventario final de funciones o existe una llamada antes de cargar D;
- aparece una dependencia circular o una carga de D anterior a su captura;
- cambia una fuente Go, su huella o el binario reproducible;
- el manifiesto deja de tener nueve entradas en el orden previo;
- se minifica, combina o elimina un control para ganar espacio;
- D puede ejecutarse o cargarse sin la capacidad privada;
- H0, calidad, secretos, reconstrucción o limpieza no quedan verdes;
- se añade cualquier conducta de O3a o se habilita el modo operativo;
- se supera 800 líneas o se oculta una superación con líneas físicas opacas.

## Métricas y continuación

O3a-P0 no cierra O3a, C4b-2, C4b, H0b, C2, F0 ni O4-05. Las métricas
permanecen F0 `10/23`, O4-05 `3/5`, Contratación temporal `24/46`, Bolsa
productiva `1/14` y producción `NO-GO`.

Tras el cierre P0 se redactará O3a con una frontera nueva. `Start` y mapa FD
no se programan bajo este documento.
