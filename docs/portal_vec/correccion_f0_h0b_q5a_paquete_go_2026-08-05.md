# Corrección F0-H0b Q5a: aislamiento del ejecutable Go probatorio

## Incidencia

La ejecución CI `30961258790` rechazó el corte `c16422b` en la puerta general
de calidad. `go test ./...` encontró dos funciones `main` en el paquete
`deploy/postgresql/autorizacion_atestada_v3/pruebas_sql`: el capturador H0 y el
supervisor M38. Las otras cuatro puertas —secretos, artefactos productivos,
PostgreSQL ContextoActor y PostgreSQL Bolsa pública— terminaron correctamente.

El defecto no alteró producción ni datos: ambos ejecutables son herramientas
probatorias locales, pero la colisión impedía compilar el árbol completo y
también hacía fallar el inventario arquitectónico basado en `go list`.

## Corrección mínima

El supervisor declara `//go:build ignore && linux && amd64`. El tag
convencional `ignore` lo excluye de las compilaciones ordinarias del paquete.
El runner conserva `go vet fichero.go` y `go build fichero.go`; al recibir una
lista explícita de ficheros, la herramienta Go forma un paquete sintético e
ignora sus build constraints. Por tanto se mantienen Linux/amd64, la
autoprueba ABI y el modelo de captura privada de quinta fuente.

No se modifica el capturador, D2c, D2d, H0b ni el adaptador M38. Tampoco se
añade una etiqueta activable a la CI ni una segunda ruta de compilación.

## Huellas posteriores

| Artefacto | SHA-256 |
| --- | --- |
| Fuente del supervisor | `a1bbb941baa91cbc6e32f565542cb68b1548a31214d17d4b8a8c9ee87f764f95` |
| Binario Go 1.26.5 reproducible | `4bd9e83093a345528b690197eec129fbe22cb029dc5f0ace40e82dd0333079a8` |

El binario se reprodujo dos veces desde temporales distintos, con entorno
aislado, `GOOS=linux`, `GOARCH=amd64`, `GOAMD64=v1`, `CGO_ENABLED=0`, red de
módulos desactivada y `-trimpath`; ambas compilaciones dieron la misma huella.

## Criterio de cierre

La corrección solo puede aceptarse después de acreditar:

- `go vet` y `go build` del fichero explícito;
- autoprueba ABI real y modo desconocido 64;
- compilación y `go vet` del paquete compartido sin colisión;
- `go test ./...`, `go test -race ./...` y `go vet ./...`;
- sintaxis Shell, ShellCheck, hashes, límites y `git diff --check`;
- revisión independiente del nuevo corte;
- cinco puertas CI verdes tras publicar la reparación.

Hasta entonces Q5a sigue integrada pero no publicada con una CI aceptable, y
C4b-2 operativo permanece cerrado a cambios de código.
