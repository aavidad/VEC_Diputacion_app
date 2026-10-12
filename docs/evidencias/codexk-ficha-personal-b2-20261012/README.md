# Lista y ficha de Personal desde el portal

RRHH consulta empleados, abre su ficha y consulta plazas desde Registro de Personal en `vec-server`. El servicio de ficha y el repositorio PostgreSQL ya se construían para el plan CT→Personal; ahora los manejadores existentes se conectan también a la pantalla. La lista usa su acción propia y un perfil lector del mismo actor, elegido por el servidor. No comparte las acciones de alta del plan. La configuración anterior sigue admitida; para servir la lista hay que añadir la operación privada `personal_empleados`.

Reutiliza: `ServicioRegistroEmpleadoB2`, `ServicioEmpleadosRegistroB2`, sus manejadores HTTP, el pool `personal_consultas`, la autoridad nominal B2 y la selección de sesión por certificado/acción. No se crea un puente a otro proceso, una consulta, una fachada SQL de lectura ni otro formulario. M mantiene la interfaz y el enlace del recibo CT a la ficha, en #1038.

## Ensayo local

PostgreSQL 18.4, copia propia con datos inventados, contenedor sin red externa, `--rm --restart=no --memory 2g` y datos en disco. Chrome del sistema mediante playwright-cli, certificado RRHH de ensayo. El preparador existente generó material B2 de 27 operaciones y 19 audiencias. El LOGIN provisional de preparación se eliminó al acabar.

CT204 se instaló una sola vez sobre CT203, desde el cuerpo literal instalado. Preserva OID, ACL, propietario, RLS y configuración. Sólo añade tres rutas de negativa de Personal; la ficha guarda el patrón `{emp_ref}`, sin referencia del empleado en la auditoría de frontera. No se ejecutó DOWN ni se reaplicó una SQL con historia.

La primera lista devolvió 503 con causa interna VA172: faltaba su terna de origen. En el clon se añadió una fila técnica por SQL directo, igual al origen de la ficha. Tras ello devolvió 200 con auditoría. La terna final está en el paquete AD172 existente: su ensayo en ROLLBACK cotejó 27 entradas y conservó el inventario. Para desplegar se usa el paquete, no el INSERT privado del ensayo.

Se añadieron por SQL directo 1.000 relaciones ficticias, repartidas entre vigente, suspendida y finalizada. Son antecedentes de lectura para volumen y estados; no se presentan como altas efectuadas desde CT. La copia conserva cero recibos de alta B2 por este fixture. La hora de conocimiento se toma después de insertar: el trigger fija el instante real.

| Comprobación | HTTP | Resultado |
| --- | --- | --- |
| Lista desde pantalla | 200 | 25 filas, paginación, sin alerta de lectura |
| Clic «Ver ficha» | 200 | Ficha del empleado seleccionado, relación y auditoría |
| Lista con límite 1 | diez 200 | p50 55 ms; p95 56 ms |
| Lista con límite 100 | diez 200 | p50 108 ms; p95 111 ms; 100 filas |
| Ficha directa | 200 | 54 ms; una relación; auditoría propia |
| Vacantes | 200 | 33 plazas; auditoría propia |

Las medidas HTTP son locales con el volumen indicado; no son medidas de cidonia. Cada respuesta medida conserva `evidencia.auditoria_ref`. La pantalla de ficha se observó en escritorio y 390 px: sin desbordamiento global; localStorage y sessionStorage vacíos. La sesión conserva errores anteriores de preferencias; no se atribuye una comprobación global del portal.

Pruebas focales finales: `go test -p 6 ./internal/app/bootstrap ./internal/vec/adapters/httpapi ./cmd/vec-preparar-material-interno -run 'IncorporacionB2|CatalogoEmpleadoB2|RegistroPersonalB2|RutasExactasAuditoria|Incorporacion|ServidorB2' -count=1`, verdes. El montaje se prueba con el dispatcher real: las negativas no leen repositorio, dejan ruta canónica y no permiten actos. Formato y `git diff --check` verdes. Semgrep local con métricas desactivadas: un aviso idéntico al de main, fuera del cambio. Gosec: 20 avisos heredados, ninguno en líneas cambiadas. La puerta completa se ejecuta en la CI de la PR.

## Instalación y continuación

[Lista SQL y configuración](../../../deploy/principal/lista_sql_codexk_ficha_personal_b2_20261012.txt): CT204 después de CT203, sin repetir prerrequisitos instalados; material de la operación `personal_empleados` y terna AD172. El nuevo perfil se publica inicialmente sólo si se configura esa operación; no modifica el perfil del plan ni exige un selector en pantalla. Se conserva la provisión del catálogo documentada en #1036.

El siguiente paso de la misma cola es el alta desde la resolución de CT y el cese con vuelta a Bolsa. El caso de ensayo 94014 todavía necesita una aceptación de Bolsa legible bajo la clave disponible; esa dependencia está comunicada a I en el canal. Este corte prueba lecturas usables, no acredita un alta CT, un cese ni su recuperación tras reinicio.
