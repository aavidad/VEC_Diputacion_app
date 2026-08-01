# Revisión F0-C1: acreditación privada del material de fuente

Fecha: 2 de agosto de 2026.

## Resultado

F0-C1 obtuvo dos revisiones independientes finales `GO`, ambas con
`P0=P1=P2=0`. El commit productor `c3f42fb` se integró en la rama estable
como `e441400`; ambos tienen el mismo `patch-id`
`c0f115001be82df1c4b309295ca0b555554e0220`.

El corte añade exclusivamente los componentes `070_acreditar_material_fuente.sql`.
La función de acreditación permanece privada y no publica un rol, una fachada
ni una migración `000007` instalable.

## Huellas finales

```text
M070: f36be1419a1a8d668f07b3a9c81da7ab82ad8ebce03880f2cc14e7fd6d936f09
      554 líneas
T070: 2742d24aa5680e47f5d42d80fb8c9e957bbeee76412299047ab488997f0a9907
      799 líneas
```

Las huellas del cuerpo almacenado en catálogo son:

```text
disparador de checkpoint:
0d2a6ec8b7288b61e3a85a7da4d3ad490920d11c8a80d052ceb93aa0879b13ca
acreditador privado:
f4da25b409d42f6ea50bb6c97358e224dc021b397462b22672dc810eb6c32f1f
```

## Correcciones exigidas por la revisión

El primer candidato recibió `NO-GO` porque una clave anterior todavía podía
emitir después de rotar el puntero efectivo de su audiencia. La corrección:

- liga la emisión a la última clave efectiva de la audiencia exacta;
- hace avanzar el checkpoint al insertar punteros de clave o configuración;
- mantiene audiencias independientes;
- fija la ABI nominal y la ACL propietaria de la función privada;
- amplía la matriz temporal, causal y de revocación.

Una revisión posterior detectó dos brechas P2: no se acreditaba la igualdad
temporal exacta ni el salto causal mayor que uno al cambiar configuración. La
prueba final rechaza el desfase cero, acredita `+1` bajo mínimo y `+2` para
una configuración creciente, reutilizando claves dentro de subtransacciones
para demostrar su reversión efectiva.

## Evidencia reproducida

Productor, dos revisores y dirección ejecutaron la etapa C1 sobre PostgreSQL
18.4 fijado por digest:

```text
postgres@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296
```

La matriz acredita:

- material, manifiesto, capacidad y consumo ligados a sus cánones y huellas;
- vigencias UTC finitas, orden temporal exacto y límite de desfase;
- clave efectiva por audiencia y configuración corporativa vigentes;
- revocaciones de fuente, clave, configuración y raíz;
- checkpoint causal y sus dos disparadores privados que invocan la función
  `SECURITY DEFINER`;
- ABI nominal, propietario, ACL, `search_path`, catálogo y dependencias;
- comparación de huellas en tiempo constante mediante A3;
- liberación de locks y reversión completa de los ensayos.

Cinco mutantes temporales independientes —puntero de configuración y
revocaciones de fuente, clave, configuración y raíz— fueron rechazados. La
matriz mata además mutantes de desigualdad temporal y checkpoint.

Las carreras multisesión cubren ambos órdenes:

- C1 primero: la rotación espera el `transactionid` y termina en `55P03`, sin
  interbloqueo ni residuo;
- C1 revertido: la rotación se confirma y una emisión posterior con la clave
  anterior se rechaza con `42501`;
- rotación o cambio de configuración primero: una instantánea obsoleta falla
  con `55P03` o `40001`, sin aceptar una decisión caducada.

La prueba oficial `--etapa C1` quedó verde después del `cherry-pick`, con
`ROLLBACK`, línea base restaurada y cero contenedores o temporales propios.
`git diff --check`, tamaños y Gitleaks quedaron verdes. La ejecución CI
`30721617186` terminó completamente verde sobre `e441400` en sus cinco
puertas.

## Límites y continuación

C1 no consume ni persiste por sí sola la capacidad, no hace replay exterior y
no habilita producción. C2 debe componer A4, B2 y C1 dentro de la misma
transacción `SERIALIZABLE READ WRITE`. Después continúan R0, M5–M7,
selección/recibo, fachada/reconciliación, PDP, composición raíz, TLS/mTLS y el
E2E. Las métricas funcionales no cambian.
