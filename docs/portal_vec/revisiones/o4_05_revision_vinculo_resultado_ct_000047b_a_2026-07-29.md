# Revisión del par nominal VEC CT-000047B-A

Fecha: 29 de julio de 2026.

## Resultado

**GO técnico independiente** para el corte mínimo CT-000047B-A.

| Elemento | Valor |
| --- | --- |
| Base | `d621522` |
| Commits integrados | `700d72a`, `f49afd0`, `f186ce9` |
| P0 | 0 |
| P1 | 0 |
| P2 | 0 |

Este corte no implementa la autoridad de canal, el PDP, los recursos
autorizables, la composición raíz ni una ruta productiva. El avance funcional
de O4-05 no cambia.

## Garantía añadida

`CrearVinculoAutenticacionActorV2ConResultado` realiza exactamente una
revalidación de autenticación y una resolución registrada del contexto. La
misma operación devuelve:

- el `VinculoAutenticacionActorV2`;
- el `ResultadoContextoActorRegistradoV2` exacto que originó el vínculo.

El constructor anterior delega en la nueva fábrica y conserva su contrato y
sus errores. No se vuelve a resolver el contexto para intentar reconstruir el
par.

La salida es una copia defensiva. No comparte memoria mutable con la autoridad
en:

- representación canónica;
- manifiesto de procedencia;
- roles;
- permisos;
- atributos;
- vínculos de la instantánea.

Se conserva además la diferencia semántica entre un contenedor nulo y un
contenedor vacío no nulo.

## Evidencia reproducida

El productor y un revisor distinto ejecutaron:

```text
pruebas focales, 50 repeticiones
pruebas focales con detector de carreras, 20 repeticiones
go test ./internal/vec/domain
go test -race ./internal/vec/domain
go test ./internal/vec/... ./internal/modules/contrataciontemporal/ports
go vet sobre los paquetes afectados y consumidores directos
git diff --check
verificación de tamaños
análisis de secretos del rango de commits
```

Dirección repitió sobre la rama integradora las pruebas de dominio y puertos,
la carrera focal, `go vet`, tamaños, diferencias y secretos. Todas quedaron
verdes.

## Límites y siguiente decisión

El par nominal no se expondrá al cliente ni se construirá desde HTTP, cookies,
cabeceras libres, variables DEMO o configuración de presentación. Antes de
crear el adaptador se cerrará el diseño de:

1. retención privada del par en `ContextoConsultaRRHH`;
2. fábricas cerradas de recursos y solicitudes para cuadro y detalle;
3. guardianes mínimos que permitan usar el resultado exacto sin publicar
   getters sensibles;
4. separación entre decisión PDP y emisión del material VEC-AD-3.

Cada punto será una tarea y un commit verificable; no se agruparán bajo una
implementación monolítica de «autoridad/PDP».
