# Provisión interna H6: preparación local

La CLI acepta exclusivamente los bytes de la fuente sintética acreditada por Dirección el 01/10/2026 a las 05:00 CEST (SHA256 `14ee4c7b94e08d28ba299a7278e65261bec1d3344cb485b7ebef3ebd00cec056`). El fichero debe ser privado, regular y estar fuera de Git. El acuse de procedencia no aprueba perfiles.

```sh
go run ./cmd/vec-provisionar-identidades-internas --modo plan --fuente <ruta-privada>
go run ./cmd/vec-provisionar-identidades-internas --modo reconcile --fuente <ruta-privada> --plan-sha256 <huella-del-plan>
```

`plan` devuelve las cuatro funciones y sus bloqueos en JSON, sin nombres, sujetos ni referencias personales. `reconcile` coteja la huella del plan con los mismos bytes de fuente. Ambos son de solo lectura y devuelven `bloqueado`.

`apply` devuelve `autoridad_no_disponible` y no abre una conexión. Para habilitarlo hace falta un caso de uso gobernado que consuma, por cada actor, aprobación nominal separada de este acuse, preimagen autoritativa y CAS por huella dentro de la misma transacción que cuenta, contexto, asignación, auditoría e historia. Debe utilizar la custodia HMAC existente sin exponer la clave al proceso. La fuente actual deja sin sujeto y certificado a solicitante y ratificador, y no acredita la posesión de los certificados públicos de RRHH e Intervención. Tampoco prueba una asignación general H6 para ellos. La revisión de esas condiciones precede cualquier intento de escritura.

Este corte no instala SQL ni modifica la base H6 o cidonia. Si cambia la fuente, su nuevo contenido requiere acreditación y plan nuevos; no se reutiliza esta huella.
