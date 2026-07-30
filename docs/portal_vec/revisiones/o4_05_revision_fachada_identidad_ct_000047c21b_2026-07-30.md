# Revisión CT-000047C2.1b: fachada de Identidad para ContextoActor

Fecha: 30 de julio de 2026.

## Resultado

`GO` funcional y de seguridad, con P0=P1=P2=0.

Integración estable:

| Commit | Contenido |
| --- | --- |
| `ea91b30` | Migraciones ascendente y descendente y runner principal |
| `a2ccb42` | Readiness PostgreSQL definitivo |
| `a218d6e` | Estados, carreras, cardinalidad, ACL y safe-down |
| `4743b89` | Venenos de topología y fachada |
| `35c33bf` | Prueba no tautológica del contrato de retorno |
| `d768007` | Recuperación física con réplica hot standby |

La fachada recibe solo referencias de autenticación y sesión y devuelve
cuenta, método, garantía y caducidad observados. No selecciona perfil u
organización, no consulta el PDP y no habilita producción.

## Revisiones y correcciones

El cierre conserva los `NO-GO` en lugar de ocultarlos:

1. La primera revisión encontró cobertura insuficiente de réplica física,
   estados de cuenta, carreras, cardinalidad, ACL y safe-down.
2. CT89 detectó que `INHERIT FALSE` fallaba antes de atravesar C2.1b y que
   faltaban venenos vivos de lenguaje, seguridad y retorno.
3. CT94 demostró que el primer caso de retorno era tautológico: el `up`
   rechazaba cualquier función preexistente.
4. CT98 emitió el `GO` final después de derivar la función hostil desde la
   definición productiva y cambiar exclusivamente el nombre del cuarto
   parámetro `OUT`.

La prueba final compara en ambos sentidos propietario, lenguaje, clase,
volatilidad, paralelismo, `SECURITY`, strictness, argumentos, tipos,
configuración, cuerpo, ACL y dependencias. Solo difiere
`identidad_valida_hasta` frente a `identidad_caduca_en`; el `down` productivo
rechaza esa deriva con `55000` y después se restaura el OID original.

## PostgreSQL 18.4

La imagen está fijada por digest. Productores, revisores y dirección
acreditaron:

- resultado exacto de cuatro columnas;
- superficies, garantías, cuentas y referencias cruzadas;
- frontera de quince minutos, caducidad y revocación;
- cuenta inactiva sin resurrección de sesión;
- ambos órdenes de carrera entre revalidación y revocación;
- cardinalidad hostil y restauración íntegra;
- LOGIN, selector, membresías, atributos y ACL efectivas;
- `PUBLIC`, privilegios heredados y defaults hostiles;
- lenguaje, seguridad, paralelismo, configuración, cuerpo y retorno;
- `40001`, cancelación, timeout e interbloqueo no capturados;
- carreras C2.1a/C2.1b y preparación de C2.5 mediante locks observables;
- safe-down con consumidor directo, heredado o por `PUBLIC`;
- reinicio, reconexión y base intacta.

El runner de recuperación crea una primaria y una réplica física sin publicar
puertos. Prueba:

- `recovery=false` en primaria y `recovery=true` en hot standby;
- replay de WAL, pausa y retardo observable;
- fachada cerrada en réplica;
- contraste con la función base que falla `25006`;
- `SERIALIZABLE` cerrado por PostgreSQL con `0A000`;
- caída y reenganche de primaria;
- reinicio de réplica y WAL posterior;
- rechazo de escritura;
- limpieza completa ante éxito y `SIGTERM`.

## Puertas de integración

Dirección integró únicamente los árboles revisados sobre la rama estable y
comprobó sus hashes byte a byte. Después ejecutó:

```text
probar_revalidacion_contexto_corporativo_rrhh_000004_pg18_4.sh
probar_revalidacion_contexto_corporativo_rrhh_000004_recuperacion_pg18_4.sh
identidad_sesiones_v1/probar_integracion.sh
bash -n
ShellCheck
comprobar_tamano_ficheros.sh
git diff --check
Gitleaks sobre los seis commits
```

Todas las puertas terminaron verdes. Gitleaks recorrió seis commits y
123,90 KiB sin detectar secretos.

## Alcance restante

C2.1b es un cierre técnico interno y no aumenta las métricas funcionales.
Siguen pendientes organización y vínculo corporativos, publicación,
selección, recibo, acreditación, PDP, composición raíz, TLS/mTLS y el E2E
HTTP/web completo. Producción permanece en `NO-GO`.
