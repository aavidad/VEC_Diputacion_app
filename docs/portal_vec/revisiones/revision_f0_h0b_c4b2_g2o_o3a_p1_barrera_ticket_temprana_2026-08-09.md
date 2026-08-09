# Acta de revisión F0-H0b/C4b-2/G2-O/O3a-P1

Fecha: 9 de agosto de 2026.

Resultado: **GO documental, P0=0, P1=0, P2=0**. Se autoriza únicamente
confirmar y publicar el par documental formado por la decisión O3a-P1 y esta
acta. La edición material R-only permanece prohibida hasta que el commit
documental publicado obtenga CI `5/5`.

No se autoriza O3a, `Start`, mapa FD, pidfd, escritura del ticket, `STOP`
productivo, `/proc`, `CONT` productivo, Docker, PostgreSQL real, datos reales,
despliegue ni uso en producción.

## Base exacta

La revisión parte de la rama `integracion/ct-o4-04e-20260726` en el commit
publicado `623a3739169421d723ea3458c8acb5c12b157a2b`, hijo del material O3a-P0
`ac988d615d021b14cbcfc2929f715ab4d9bb5567`. La CI `31291669273` terminó
`completed/success` con cinco de cinco trabajos verdes.

El runner base R tiene:

```text
702 líneas
46.123 bytes
modo Git 100755
SHA-256 e617024a52c4a042971b026d0799816933b489ed4221e9b6147317936d18054c
```

## Hallazgo de origen

La primera falsación del borrador O3a demostró que R ejecutaba
`/usr/bin/env -0 | /usr/bin/grep ...` antes de leer FD 9 y antes de su
auto-`STOP`. El Bash provisional podía crear dos descendientes antes de la
barrera que O3a pretendía usar para garantizar cero efectos.

Se rechazó compensarlo dentro de O3a porque habría trasladado señalización de
grupo y drenaje de adoptados a una fase que no los posee. O3a-P1 se separó como
minitarea reversible, R-only y sin incremento funcional.

## Primera ronda documental

La primera versión O3a-P1 revisada tenía 207 líneas y SHA-256
`49ad8fd350c37fd0ffd88c70f8229f394ffa46a043b99780cc8a8ff5ea732fea`.
Ambos revisores emitieron `NO-GO` porque la allowlist omitía el builtin `exit`;
también exigieron hacer explícita la precedencia ticket/entorno y concretar los
mutantes de patrón y posición.

Las correcciones:

- distinguieron comandos externos, builtins y palabras de control;
- permitieron `exit` solo en rechazo y las construcciones `for … in` e `if`;
- fijaron los cruces ticket inválido/entorno hostil y ticket válido/entorno
  hostil;
- concretaron M10 y añadieron M11;
- acotaron la invariancia del modo directo a estados, salidas y efectos externos
  funcionales, declarando la reordenación interna y `LC_ALL=C`.

La versión previa a la proyección quedó en 230 líneas, SHA-256
`be01915fbd658b6981673ceb16c1885a37bed82097fb7217deb39fcf90e27bf3`,
y recibió doble `GO`, P0=0, P1=0, P2=0. Ese doble `GO` autorizó solo la
proyección externa.

## Proyección y ledger

Tres reconstrucciones independientes coincidieron:

| Unidad | Líneas | Bytes | Modo Git | SHA-256 |
| --- | ---: | ---: | ---: | --- |
| R base | 702 | 46.123 | `100755` | `e617024a52c4a042971b026d0799816933b489ed4221e9b6147317936d18054c` |
| bloque R8--R20 | 13 | — | — | `831b07df84fd6fa05d3f060e1c275de730b66eff54ac68caf720cc932eb2d00e` |
| R proyectado | 702 | 46.123 | `100755` | `7ad65a66ece586710a4651e579385b7aba2ad5b84ef6baf02ba4c36659cd6487` |

La proyección mueve exactamente R8--R20 después del `fi` base R47. En el
candidato, el auto-`STOP` queda en R32, el bloque en R35--R47, la pipeline en
R36 y la selección `modo_m38 == hijo` en R48. La directa e inversa recuperan
ambos extremos byte a byte; `bash -n`, ShellCheck 0.11.0 y diff-check quedan
verdes.

La comprobación física reprodujo:

- modo directo con `LD_PRUEBA=1`: estado 65;
- FD 9 en EOF o ticket adverso: estado 64 antes de descendientes;
- segunda línea: estado 64;
- ticket válido: Bash detenido en `T`, cero hijos propios antes de `CONT`;
- ticket válido y `LD_PRUEBA=1`: `STOP`, después `CONT`, estado 65;
- R con `--supervisar-m38` o argumento desconocido: estado 1, stdout vacío y
  stderr exacto de uso de 66 bytes;
- el binario Go invariante con ambos argumentos: estado 64 y salidas vacías.

La proyección no ejecutó H0, calidad global ni las cien iteraciones: esas puertas
permanecen obligatorias para el futuro candidato material y no se presentan
como verdes anticipados.

## Mutación y evidencia

La decisión final fija M01--M11 con transformación, líneas, SHA-256 y rechazo
estructural exactos. Los once candidatos pasan `bash -n` y ShellCheck antes de
su oráculo. La preparación reprodujo los once rechazos sin usar la huella global
del candidato como atajo.

El analizador preparatorio tiene 131 líneas/SHA-256
`936a1a242ca2c9120f87c412a7660ab357acf3e1d9c04184fc6890856abd7c1a`;
el generador preparatorio, 65 líneas/SHA-256
`ce00e0284a9349ece653b195afff7c935f75cf39ee4441c9078c2ff0bfa94977`.
Ambos son plantillas externas, no evidencia durable. La evidencia futura debe
ligarse al commit material, eliminar rutas físicas de los diagnósticos y
separar muertes estructurales de conductuales.

M10 conserva obligatoriamente el oráculo conductual `LD_PRUEBA=1`; M09 puede
morir solo por cardinalidad estructural. Un parseo fallido, un build imposible,
una huella distinta o una salida por causa ajena no cuentan como mutante muerto.

## Doble revisión final y ligadura

La versión completa que recibió los dos `GO` finales tiene:

```text
290 líneas
SHA-256 cb94abbd1e468d3f251419d5973f7f740594c7183f82c4b67a0851b19fedfa5a
```

Revisión funcional: `GO`, P0=0, P1=0, P2=0. Reprodujo ledger, reversión,
`bash -n`, ShellCheck, modos R/Go y las once líneas/huellas mutantes.

Revisión de seguridad: `GO`, P0=0, P1=0, P2=0. Reprodujo transformación,
posiciones, modos, plantilla y once rechazos estructurales; validó
documentalmente la obligación de limpieza aislada y la ausencia de secretos,
datos personales y rutas privadas.

Después de ambos dictámenes se cambió exclusivamente el párrafo de estado de
la decisión, sin tocar alcance, ledger, transformación, mutantes, puertas,
paradas ni métricas. La decisión autorizada final tiene:

```text
290 líneas
SHA-256 86fb77404eec5da0fc95caad46f9710b50249cc081854acfcfe27c728bae2707
```

La relación `cb94abbd…` → `86fb7740…` debe reconstruirse sustituyendo solo el
párrafo de estado. Cualquier otra diferencia invalida esta acta.

## Autorización limitada y siguiente puerta

Se autoriza:

1. confirmar solo la decisión O3a-P1 y esta acta;
2. publicar ese commit documental en la rama integradora;
3. esperar y comprobar CI `5/5` sobre el SHA publicado;
4. solo entonces aplicar el traslado exacto R-only en un worktree exclusivo;
5. ejecutar todas las puertas, confirmar material autónomo y someterlo a dos
   revisiones independientes antes de integrar o publicar.

El material debe conservar 702 líneas, 46.123 bytes, modo `100755`, LF y SHA
`7ad65a66…`; todo lo demás permanece byte a byte. H0 PostgreSQL 18.4, calidad,
carrera, `govulncheck`, Gitleaks, cien iteraciones, mutantes y residuos cero son
puertas futuras, no resultados de esta acta.

## Métricas

O3a-P1 no suma capacidad. Permanecen F0 `10/23`, O4-05 `3/5`, Contratación
temporal `24/46`, Bolsa productiva `1/14` y producción `NO-GO`. El borrador O3a
sigue detenido y deberá rebasarse sobre el futuro material O3a-P1 antes de una
nueva revisión completa.
