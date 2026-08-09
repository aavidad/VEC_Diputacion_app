# Enmienda O3a V5: autoridad CI sin R

Fecha: 9 de agosto de 2026.

Estado: **CANDIDATA; no autoriza integración hasta doble revisión y CI 5/5**.

## Motivo

El contrato original asignaba a R la captura de cinco fuentes nuevas y la
ampliación de su manifiesto de nueve a catorce entradas. La instrucción vigente
«No uses R» obliga a mantener R byte a byte: 702 líneas y SHA-256
`7ad65a66ece586710a4651e579385b7aba2ad5b84ef6baf02ba4c36659cd6487`.
La proyección V20 ya dispone de un conductor durable versionado que posee un
ledger exacto de diez fuentes y ejecuta los catorce bloques C01..C21 en modo
normal y `-race` real. Esta enmienda cambia el propietario de esa verificación,
no reduce sus oráculos.

## Decisión

Para O3a V5, la autoridad de manifiesto, `gofmt`, `go vet`, builds normal/race,
casos C01..C21, residuos y huellas es
`tools/o3a_v5_conductor/conductor.sh` junto a `fuentes_v5.tsv` y sus siete
bloques. R continúa siendo autoridad histórica de los cortes anteriores y no
se edita ni se invoca para V5.

La job CI `puerta-calidad` conserva su nombre y añade, después de
`scripts/verificar_calidad.sh`, la invocación exacta:

```sh
cd tools/o3a_v5_conductor
./conductor.sh "$GITHUB_WORKSPACE" "$RUNNER_TEMP/evidencia-o3a-v5"
```

El destino nace vacío en el runner. El conductor exige Go 1.26.5, diez fuentes
con líneas/SHA exactos, 14/14 bloques GO, 74 registros, modos normal y carrera,
FD inicial/final iguales y residuos cero. Cualquier divergencia termina no
cero y hace fallar `puerta-calidad`. No usa Docker, PostgreSQL, red ni el modo
operativo de Orquesta.

Resumen, manifiesto, TSV y logs de los catorce bloques se vuelcan al log CI sin
ocultar el estado de salida. La espera test-only de terminalidad nominal usa
diez segundos; `duracionRetiradaO3aM38 = 3 s` productivo permanece intacto.
C21 cuenta con `F_GETFD` los descriptores que siguen vivos: solo descarta una
entrada obsoleta de `/proc/self/fd` cuando devuelve `EBADF` y falla ante cualquier
otro error o descriptor vivo adicional.

## Write-set material posterior

La materialización autorizable pasa a ser:

1. G1 y G6a--c/G7a--b: seis ficheros Go exactos del ledger V20;
2. `tools/o3a_v5_ast/**`: analizador, catálogos y evidencia V20 autorizada;
3. `tools/o3a_v5_conductor/**`: conductor, ledger y evidencia r3;
4. `.github/workflows/ci.yml`: una invocación fail-closed en la puerta existente;
5. contrato, acta y esta enmienda.

R, D, G2--G5, capturador, adaptador, SQL, migraciones y producción quedan
byte a byte. No se mezcla contenido de V19 ni de ledgers anteriores revocados,
ni del checkpoint portable.

## Evidencia requerida

- V20 técnico: conductor 14/14 y 74 casos, C21 estable normal/race, mutantes
  195/195 y SEC 4/4; todo ello sujeto al doble `GO` del corte vigente.
- Revisión funcional y de seguridad completa de esta enmienda y del delta CI.
- Reproducción local del comando CI sobre el commit candidato.
- Push sin force y CI posterior 5/5, donde `puerta-calidad` incluya el conductor.
- Revisión post-CI de SHA, jobs y ausencia de residuos.

Solo después se podrá materializar la cadena exacta en
`integracion/ct-o4-04e-20260726`, repetir las puertas y publicar esa rama. La
enmienda no abre O3b, producción, datos reales ni cambia métricas.

## Paradas duras

Se detiene si R cambia; si la job omite o permite saltar el conductor; si el
target no es el checkout exacto; si el destino de evidencia no está vacío; si
cambia una huella del ledger sin nueva revisión; si aparece `SKIP`/`NO-GO`; si
no hay build race real; si queda proceso/FD; si se inicia Orquesta; si cualquier
revisión, gate o job no termina verde.
