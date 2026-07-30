# Revisión independiente CT88: validación documental

Fecha: 30 de julio de 2026.

## Resultado

`GO` independiente sobre el candidato
`e39043c1b954f322640bcf042e0c8e4721e1cb25`, integrado en la rama estable
como `f662302`.

Severidades:

- P0: 0;
- P1: 0;
- P2: 0.

El corte sustituye dos validaciones mediante expresiones regulares y
decodificación hexadecimal por escáneres ASCII acotados. No modifica API,
errores públicos, límites, estado, caché ni semántica funcional.

## Equivalencia comprobada

- SHA-256 conserva longitud exacta de 64 octetos y alfabeto
  `[0-9a-f]`.
- Se mutaron los 256 valores posibles en las 64 posiciones de cuatro bases.
- Se comprobaron longitudes de 0 a 66, mayúsculas, NUL, UTF-8 e entradas
  inválidas.
- Las referencias conservan exactamente
  `^[a-z][a-z0-9._:-]{0,255}$`.
- Se comprobaron las fronteras 0, 1, 2, 255, 256 y 257.

## Rendimiento reproducido por el revisor

| Prueba sin caché | Base | Candidato | Mejora |
| --- | ---: | ---: | ---: |
| Paquete `internal/vec/ports` | 39,27 s | 24,55 s | 37,5 % |
| Foco documental | 11,77 s | 7,10 s | 39,7 % |
| Foco documental con `-race` | 129,94 s | 61,05 s | 53,0 % |

## Puertas

El productor, el revisor y dirección acreditaron, según su alcance:

- pruebas normales de `canonico/documental`, `domain` y `ports`;
- detector de carreras de los paquetes modificados y del foco documental;
- `go vet` de los tres paquetes;
- `gofmt` y `git diff --check`;
- tamaños dentro de DEC-051;
- Gitleaks sin secretos.

Dirección repitió las puertas focales después de integrar. El cambio no cierra
una capacidad funcional, no altera porcentajes y no autoriza producción.

## Incidencia de integración y cierre CT104/CT108

La ejecución GitHub `30564710316` detectó después de integrar CT88 una
violación real de la barrera C4: el dominio había pasado a importar
`internal/vec/canonico/documental`, por lo que el proceso interno arrastraba
una dependencia no aprobada. Las pruebas Go eran verdes, pero el corte
publicado no lo era.

CT104 eliminó esa dependencia sin ampliar la lista permitida ni recuperar
expresiones regulares o decodificación hexadecimal. La primera revisión
independiente obtuvo `NO-GO` con P0=0, P1=0 y P2=1 porque el nuevo escáner
duplicaba la primitiva `esSHA256` ya existente en el dominio.

CT108 cerró el hallazgo: `esSHA256` conserva el recorrido ASCII indexado como
autoridad única y la validación documental delega en ella. La revisión final
CT109 obtuvo `GO` con P0=P1=P2=0. Dirección reprodujo sobre la rama estable:

- pruebas sin caché de `domain`, `canonico/documental` y `ports`;
- barrera C4 limitada a nueve paquetes y su autoprueba negativa;
- `go vet`, control de tamaños y `git diff --check`;
- Gitleaks sobre los dos commits, con 2,74 KiB analizados y cero fugas.

El paquete `internal/vec/ports` completó en 25,27 segundos, por lo que la
mejora de CT88 permanece. Los commits integrados son `7760de1` y `71ac897`.
