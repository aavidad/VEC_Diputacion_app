# Mutantes O3c P6

Esta herramienta expande las 24 familias contractuales en 144 alternativas
atómicas distintas. Cada alternativa sustituye exactamente una ocurrencia de
la fuente sellada en `c0f2a9945ed2fc5648980ee48b91424a04977655`, vuelve a
formatear y parsear el resultado, exige `go vet` y un binario de pruebas
compilable, y solo la cuenta muerta por una prueba O3c causal o por el grupo
específico del analizador AST/tipos/DAG.

La base anterior es autoridad de hashes, no una dependencia del historial:
antes de mutar se comparan las fuentes de `HEAD` con sus SHA sellados y el
archive se crea desde ese `HEAD`. Por ello funciona en checkout `fetch-depth: 1`
sin requerir que exista el objeto padre. `HEAD` no se sella como autoridad: el
manifiesto usa `fuentes_objetivo_sha256`, digest estable de ruta y SHA de los
ocho targets, junto con los sellos explícitos de runner, AST y conductor.

El analizador se ejecuta desde el checkout de la herramienta contra el paquete
mutado con `-permitir-sha`; el archive efímero no contiene ni puede sustituir
al oráculo. Un error de parseo, tipado, infraestructura, timeout o un grupo AST
distinto del autorizado para la familia no cuenta como muerte. Cada proceso se
ejecuta en un PGID propio y debe desaparecer con `ESRCH`.

Ejecución completa desde un checkout limpio:

`go run` queda prohibido: crea un ejecutable temporal distinto y rompe el
sello que la fusión recalcula. Todos los comandos siguientes usan exactamente
el mismo binario canónico reproducible.

```text
go test ./tools/o3c_p6_mutantes
go build -trimpath -o /var/tmp/o3c-p6-mutantes-canonico ./tools/o3c_p6_mutantes
/var/tmp/o3c-p6-mutantes-canonico -repo . -out /var/tmp/o3c-p6-preflight \
  -desde 1 -hasta 144 -solo-compilar
/var/tmp/o3c-p6-mutantes-canonico -repo . -out /var/tmp/o3c-p6-lotes/lote-001-024 -desde 1 -hasta 24
# repetir secuencialmente 025-048, 049-072, 073-096, 097-120 y 121-144
/var/tmp/o3c-p6-mutantes-canonico -repo . -fusionar /var/tmp/o3c-p6-lotes \
  -out tools/o3c_p6_mutantes/evidencia
```

Los rangos `-desde/-hasta` sirven únicamente para diagnóstico y no pueden
sobrescribir la evidencia final. Para una ejecución durable por lotes, todos
se construyen con el mismo binario canónico `go build -trimpath` en
subdirectorios de una raíz y
se consolidan con `-fusionar RAIZ -out tools/o3c_p6_mutantes/evidencia`. La
fusión exige la misma base, toolchain, runner fuente/binario y cuatro huellas
AST, sus unidades auxiliares y el conductor; además acredita C001..C144 una vez,
24/24 familias y cero huecos. La salida durable incluye catálogo con
patrones hexadecimal, fuentes y SHAs, resultado causal por mutante, manifiesto
de toolchain/oráculos y `SHA256SUMS`; no incorpora rutas privadas ni el nombre
del temporal efímero.
