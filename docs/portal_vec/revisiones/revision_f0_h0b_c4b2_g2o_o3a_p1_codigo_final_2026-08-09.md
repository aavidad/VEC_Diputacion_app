# Revisión final F0-H0b/C4b-2/G2-O/O3a-P1

Fecha: 9 de agosto de 2026.

Estado: **GO técnico publicado; P0=0, P1=0 y P2=0**.

O3a-P1 adelanta la barrera física del ticket sin implementar O3a. No autoriza
`Start`, mapa FD, modo operativo, despliegue, datos reales ni producción. El
cierre material y su evidencia están publicados dentro de `ce0848e`; la CI
`31298943127` terminó `5/5` sobre ese SHA remoto exacto.

## Genealogía inmutable

| Hito | Commit | Padre | Estado |
| --- | --- | --- | --- |
| Contrato y acta documentales | `758e66f9a283337dd542de0f1c898c581e202f47` | base publicada anterior | Publicado; CI `31293958163` 5/5 |
| Material R-only | `a1aeab7ed2a884d0fb10c265bd11982d3519ce49` | `758e66f9a283337dd542de0f1c898c581e202f47` | Doble GO local |
| Evidencia durable | `f3a1e961849018e8cf8bb0aa38d3d43006b1fd44` | `a1aeab7ed2a884d0fb10c265bd11982d3519ce49` | Doble GO local |
| Cierre publicado | `ce0848ed4332c746d6a908673f3b3cad9cd90c1b` | `f3a1e961849018e8cf8bb0aa38d3d43006b1fd44` | Publicado; CI `31298943127` 5/5 |

La cadena exacta es:

```text
758e66f9a283337dd542de0f1c898c581e202f47
  -> a1aeab7ed2a884d0fb10c265bd11982d3519ce49
  -> f3a1e961849018e8cf8bb0aa38d3d43006b1fd44
  -> ce0848ed4332c746d6a908673f3b3cad9cd90c1b
```

La CI remota se atribuye al cierre `ce0848e`, que contiene sin reescritura el
material y la evidencia revisados.

## Alcance material y ledger

El commit material modifica exclusivamente:

`deploy/postgresql/autorizacion_atestada_v3/probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh`

Mueve sin alterar trece líneas del control de entorno para que, en modo hijo,
la lectura, cardinalidad, cierre y validación de FD 9 y el auto-`STOP` ocurran
antes de cualquier comando externo. Conserva el comportamiento directo y la
precedencia defensiva definida en el contrato.

| Unidad | Líneas | Bytes | Modo Git | SHA-256 |
| --- | ---: | ---: | ---: | --- |
| Runner R final | 702 | 46.123 | `100755` | `7ad65a66ece586710a4651e579385b7aba2ad5b84ef6baf02ba4c36659cd6487` |

La reconstrucción directa e inversa recupera byte a byte R final y su padre.
El resto de producción permanece invariante.

## Evidencia durable ligada al material

El commit `f3a1e96` añade solo seis ficheros regulares de evidencia, todos con
modo Git `100644` y por debajo del tope de 800 líneas:

| Artefacto | Líneas | SHA-256 |
| --- | ---: | --- |
| [Analizador](evidencias/f0_h0b_c4b2_g2o_o3a_p1_analizador_a1aeab7.go.txt) | 168 | `b8a5f5e1d5ecc38348f38a4e5ca1de34fcfe78c23fc09636a52734d7c5a5b51e` |
| [Mutador](evidencias/f0_h0b_c4b2_g2o_o3a_p1_mutador_a1aeab7.go.txt) | 374 | `7b1977a4f44ed9c4e76c09dc9ef94ee3ff9a3f16eb4c919f2dc5c707b3d6ebd2` |
| [Conductor](evidencias/f0_h0b_c4b2_g2o_o3a_p1_conductor_a1aeab7.go.txt) | 718 | `92b0dacc8eb57faf663bd1f7fdfcf372f4d7a8c95a88701e121a05787dd136e1` |
| [Lecturas acotadas del conductor](evidencias/f0_h0b_c4b2_g2o_o3a_p1_conductor_archivos_a1aeab7.go.txt) | 139 | `5583a99c95eeb20089626310695085bdb5a27859b70f16184a2cfa7306dec9f0` |
| [Matriz conductual](evidencias/f0_h0b_c4b2_g2o_o3a_p1_conductor_mutantes_a1aeab7.go.txt) | 352 | `9de476222f4a32dc3d2ab5c8f45fbf8e8a5946b91a0292914e287b73e02a28dd` |
| [Sonda pidfd y de grupo](evidencias/f0_h0b_c4b2_g2o_o3a_p1_conductor_sonda_a1aeab7.go.txt) | 243 | `9777697f064e6327c03ddfae9da246f2be1936c7c0be9d5dd21bdb9a67dd6e68` |

El mutador acredita la genealogía mediante `/usr/bin/git` y un entorno
cerrado, liga el runner físico al blob de Git y revalida HEAD, padre, estado y
fuentes después de generar. Su receta de build usa marcadores no sensibles,
un entorno `env -i`, directorios privados y un ejecutable Go 1.26.5
acreditado. La reproducción genera el conductor con SHA-256
`5d28818ec289294dc7bf60726cc94790a084396fb2fb01115a6f1c0d2005621e`.

## Matriz y puertas reproducidas

La salida conductual exacta fue:

```text
conductor=ok iteraciones=100 casos=600 mutantes=10/10+1 fd_exactos=1 hijos=0 temporales=0
```

Los diez mutantes conductuales M01--M08 y M10--M11 murieron por su causa
exacta. M09 murió por el oráculo estructural de cardinalidad. El analizador
aceptó R final con 702 líneas y SHA `7ad65a66…` y rechazó M01--M11 por los
once oráculos contractuales, sin usar la huella global como atajo.

Se reprodujeron además:

- `gofmt`, `go vet`, `bash -n`, ShellCheck y `git diff --check` verdes;
- dos builds Go 1.26.5 byte a byte reproducibles;
- H0 completo sobre PostgreSQL 18.4 verde;
- pruebas globales, carrera, `govulncheck` y puerta de calidad verdes;
- Gitleaks sin hallazgos;
- entorno Git hostil y Git falso rechazados sin salidas parciales;
- cierre TOCTOU de genealogía, blob y fuentes;
- inventarios finales de FD, hijos, contenedores y temporales sin residuos.

## Revisiones independientes

| Revisión | P0 | P1 | P2 | Veredicto |
| --- | ---: | ---: | ---: | --- |
| Funcional, estructura, proceso y mutantes | 0 | 0 | 0 | GO |
| Seguridad, hermeticidad, limpieza y secretos | 0 | 0 | 0 | GO |

Las revisiones detuvieron versiones previas del conductor y del mutador por
limpieza incompleta, plazos sin margen, lecturas no acotadas, FIFO bloqueante,
sonda de grupo insuficiente, Git no hermético y receta no reproducible. Ningún
dictamen previo se reutilizó. Ambos revisores reconstruyeron y ejecutaron la
versión final ligada a las seis huellas anteriores.

## Límites, métricas y continuación

O3a-P1 es una precondición defensiva. No cierra O3a, C4b-2, C4b, H0b, C2, F0
u O4-05. Las métricas permanecen:

- F0 `10/23`;
- O4-05 `3/5`;
- Contratación temporal `24/46`;
- Bolsa productiva `1/14`;
- producción `NO-GO`.

El siguiente y único trabajo de desarrollo es corregir y revisar el
**contrato O3a completo sobre `ce0848e`**. El código O3a, `Start` y mapa FD
continúan bloqueados hasta que ese contrato reciba doble `GO`, se publique y
su CI termine 5/5.
