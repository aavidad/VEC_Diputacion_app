# Ensayo de devengo de Dietas por terminal — 1 de octubre de 2026

RRHH puede ensayar una propuesta de cálculo nacional ordinario con fechas y horas
civiles, grupo, catálogo provisional y regla importados. El ensayo usa el motor
existente de Dietas. No crea una comisión, publica una tarifa ni guarda una
liquidación. La salida lleva `procedencia: propuesta_sin_publicar` y
`liquidable: false`.

Este corte aporta una herramienta de ensayo de D3. Los importes y criterios
siguen pendientes de confirmación por RRHH, según la ficha de Dietas. El ejemplo
contiene cantidades sintéticas y no incluye personas. No modifica D1, D6 ni los
permisos del portal.

## Recorrido reproducible

Desde la raíz del repositorio, con Go y la base de zonas horarias del sistema:

```sh
go run ./cmd/vec-dietas < cmd/vec-dietas/testdata/ensayo_nacional.json > /tmp/dietas-ensayo.json
```

El ejemplo obtiene una manutención orientativa de 1871 céntimos para el grupo 2,
sin alojamiento. Incluye la entrada completa para repetir el ensayo. Se puede
copiar el archivo JSON, modificar importes o porcentajes y ejecutar otra vez.
El catálogo y la configuración son datos: la herramienta no aporta una tabla
normativa ni importes por defecto.

El programa acepta un único objeto JSON por entrada estándar, hasta 1 MiB.
Devuelve un objeto JSON por salida estándar. El código de salida es 0 para un
ensayo calculado, 2 para una entrada rechazada y 1 para un error de escritura.
No admite argumentos de terminal. Los fallos devuelven solo `codigo`.

## Datos y comprobaciones

La entrada fija `esquema: vec_dietas_simulacion_v1` e incluye:

- `inicio` y `fin`, con `fecha` en formato `AAAA-MM-DD` y `hora` en `HH:MM`.
- `seleccion`, con grupo, versión de tarifa, referencia de regla y huella importada
  declarada esperada.
- `catalogo`, con una a tres tarifas de grupos distintos de la misma versión,
  país, rótulo provisional y cantidades expresadas en céntimos enteros. Debe estar
  incluido el grupo seleccionado. Cada fila se valida con el motor existente.
- `regla`, con el DTO `ReglaDevengoProvisional`, su configuración completa y su
  vigencia de inicio. El fin de vigencia es opcional y excluyente.

Se rechazan claves desconocidas, duplicadas o escritas con otra capitalización,
campos obligatorios ausentes, `null`, datos posteriores al objeto y UTF-8 inválido.
La selección debe coincidir con la regla en referencia, versión y huella declarada.
La regla y todas las tarifas deben cubrir el intervalo completo. Las fechas y
horas se resuelven mediante `ResolverInstanteCivil`; una hora inexistente o
ambigua no se corrige automáticamente. El motor exige fin posterior al inicio,
España y la variante nacional ordinaria provisional. Conserva sus límites,
porcentajes, redondeo y máximo de días.

## Huellas y alcance de la evidencia

`huellas.namespace` identifica `vec_dietas_simulacion_v1`. Se generan SHA-256 de
la entrada, resultado y configuración serializados mediante `encoding/json` sobre
los DTO tipados. No llevan espacios añadidos ni salto de línea. El orden de claves
del archivo original y sus espacios no afectan al cálculo de huellas; el orden
del catálogo sí forma parte de los datos. Las fechas y los valores se conservan.
Esta es la serialización definida para este esquema; no se declara compatible
con RFC 8785 ni con `jsonb::text`.

`regla_importada_declarada_sha256` conserva la huella que aporta la entrada. La
herramienta comprueba su formato y coincidencia con la selección. No verifica la
publicación, procedencia ni integridad del catálogo de PostgreSQL. Su huella de
configuración pertenece al ensayo local y no sustituye la publicada por PG000006.
Una configuración modificada sigue siendo una propuesta sin publicar, aunque
se conserve la huella declarada de origen; la huella local cambia y registra
esa diferencia. El resultado no constituye un recibo ni evidencia oficial.

## Comprobación focal

Ejecutado en la rama de preparación sobre la base `460e120c2`:

```sh
go test -p 32 ./internal/modules/dietas/application/simulaciondevengo ./cmd/vec-dietas
go test -race -p 32 ./internal/modules/dietas/application/simulaciondevengo ./cmd/vec-dietas
go vet ./internal/modules/dietas/application/simulaciondevengo ./cmd/vec-dietas
/home/alberto/go/bin/gosec -quiet ./internal/modules/dietas/application/simulaciondevengo ./cmd/vec-dietas
```

Las cuatro comprobaciones terminaron con código 0. También se ejecutó el comando
de ensayo completo: salida JSON con 1871 céntimos y las marcas provisionales
indicadas. Las pruebas verifican cambio
de tarifa y porcentajes, conservación de la instantánea anterior, redondeo,
límites horarios, vigencias, catálogo discordante, cambio de hora, entradas
inválidas, límite exacto de 1 MiB y fallos de lectura y escritura.

Semgrep local, con métricas y comprobación de versión desactivadas, analizó los
tres archivos Go de producción con dos reglas Go de la configuración facilitada
por dirección: cero hallazgos. Los archivos de prueba quedaron excluidos por
`.semgrepignore`. La revisión focal de seguridad leyó la frontera de entrada:
lectura limitada, esquema cerrado y datos sin ejecución, red ni persistencia.
No hubo auditoría completa ni validación de fuentes externas.

Quedan fuera del corte la publicación de reglas, el acceso autorizado a catálogos,
la comisión real, extranjero y supuestos especiales, la autorización, liquidación,
fiscalización y pago. No se ejecutaron SQL, pruebas globales ni navegador. La
integración y su revisión independiente corresponden a dirección.
