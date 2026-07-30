# Coordinación CT-000047M1.3: resolución nominal de motivos

Fecha: 30 de julio de 2026.

## Condición de inicio

M1.2/`000009` está integrada en `0a564e2` con doble `GO` independiente.
M1.3 solo puede comenzar después de integrar
[M1.R](coordinacion_ct_000047m1r_rol_resolutor_motivos_rrhh_2026-07-30.md).
No habilita producción ni catálogos reales.

## Resultado único

Crear exclusivamente dos funciones nominales:

```text
resolver_motivo_cuadro_rrhh_v1(timestamptz)
resolver_motivo_detalle_rrhh_v1(timestamptz)
```

Cada una devuelve cero o una referencia completa:

```text
catalogo_id
catalogo_version
catalogo_huella_sha256
entrada_clave
```

No se admite clase, catálogo, entrada, organización, acción, finalidad ni
ningún otro selector. No se crea un kernel parametrizado por clase. M2
traducirá una resolución vacía o cualquier fallo al error opaco
`ErrMotivoConsultaRRHHNoDisponible`.

## Semántica

Cada función:

1. rechaza instante nulo, infinito, no canónico o futuro;
2. deriva el actor únicamente de `session_user`;
3. exige una identidad `LOGIN` segura y miembro directo exacto de
   `vec_autorizacion_motivos_rrhh_resolutor`;
4. adquiere el advisory compartido M1.R y después los bloqueos
   `000008 → 000009 → 000010`;
5. bloquea con `FOR SHARE` solo el checkpoint de su clase;
6. resuelve la publicación exacta apuntada por ese checkpoint;
7. exige que no exista retirada local;
8. cruza checkpoint, historia y evento de publicación;
9. comprueba catálogo y entrada V2 tanto en el instante solicitado como en el
   instante actual;
10. devuelve vacío ante ausencia, retirada o incoherencia.

La resolución no escribe auditoría ni una segunda fuente de verdad. El uso
posterior queda registrado por el recorrido VEC que emite y consume la
autorización.

## Seguridad y ACL

Las dos funciones son `SECURITY DEFINER`, con propietario
`vec_autorizacion_propietario`, `search_path=pg_catalog` y sin sobrecargas.

Solo `vec_autorizacion_motivos_rrhh_resolutor` obtiene `EXECUTE`. El evaluador
V2 histórico, `PUBLIC`, fuente, registro y proyector no reciben ejecución ni
lectura directa de tablas.

La instalación:

- exige PostgreSQL 18 y UTF-8;
- valida roles, propietario, tablas, RLS, políticas, restricciones, funciones
  y ACL fundamentales;
- no adopta ni repara homónimos;
- falla atómicamente si `000008` o `000009` están degradadas.

La retirada:

- bloquea en orden `000008 → 000009 → 000010`;
- revalida identidad, cuerpo, propietario, configuración y ACL;
- no usa `CASCADE`;
- falla ante una dependencia posterior;
- elimina solo las dos fachadas y no toca la evidencia anterior.

## Write-set exclusivo

```text
deploy/postgresql/autorizacion/migraciones/
  000010_resolucion_vinculaciones_motivo_consultas_rrhh.up.sql
  000010_resolucion_vinculaciones_motivo_consultas_rrhh.down.sql

deploy/postgresql/autorizacion/
  probar_resolucion_vinculaciones_motivo_consultas_rrhh_000010_pg18_4.sh
```

Un productor posee los tres ficheros. No modifica Go, web ni documentación
transversal. Dirección integra y actualiza el estado tras revisión.

## Matriz PostgreSQL 18.4

El runner debe cubrir:

- instalación limpia, reentrada rechazada y `up → down → up`;
- falta o degradación de fundamentos sin objetos parciales;
- homónimos, sobrecargas, propietario, `search_path`, ACL o RLS alterados;
- motivos distintos publicados para cuadro y detalle;
- ausencia, instante anterior o futuro, nulo, infinito o no canónico;
- retirada local, retirada V2 y entrada fuera de vigencia;
- ejecución positiva solo por resolutor RRHH;
- denegación a `PUBLIC`, fuente, registro, proyector y `LOGIN` no miembro;
- ausencia de lectura y DML directo para el resolutor RRHH;
- `search_path` hostil y objetos temporales homónimos;
- carrera causal: la retirada espera a una resolución que retiene `FOR SHARE`;
- dependencia SQL real que impide el `down` y conserva ambas fachadas.

## Puertas

```text
PostgreSQL 18.4: 3/3
bash -n
ShellCheck
git diff --check
máximo 800 líneas por fichero
Gitleaks
revisión independiente: P0=0 y P1=0
```

M2 queda desbloqueada únicamente después de integrar M1.3 con esas puertas
verdes.
