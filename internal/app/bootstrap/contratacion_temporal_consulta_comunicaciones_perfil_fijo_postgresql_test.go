package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// peticionComunicacionesPrueba compone lo que la frontera y el proveedor de
// la consulta dejan en el contexto antes de pedir la instantánea: capacidad
// GET de la ruta, consulta ligada y datos V3 de la solicitud.
func peticionComunicacionesPrueba(
	ctx context.Context, s *soporteAltaContratacionTemporalDesarrollo, fijo *perfilFijoCTDesarrollo, organizacion string,
) (context.Context, vecdomain.RecursoAutorizable, error) {
	c := ports.ConsultaComunicacionesExpediente{ExpedienteRef: "expediente:ct150:sintetico", Limite: 10}
	recurso, err := postgresct.RecursoConsultaComunicacionesExpediente(c, organizacion)
	if err != nil {
		return nil, recurso, err
	}
	datos := vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: fijo.contexto.Vinculo, ReferenciaMotivo: motivoConsultaComunicacionesExpedienteDesarrollo(),
		Accion: postgresct.AccionConsultaComunicacionesExpediente, Recurso: recurso, Finalidad: "gestionar_contratacion_temporal",
	}
	peticion := context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{},
		capacidadConsultaContratacionTemporalDesarrollo{sello: s.sello, ruta: httpinterno.RutaConsultaComunicacionesExpediente,
			metodo: http.MethodGet, principal: principalDeSoportePrueba(s)})
	peticion = context.WithValue(peticion, claveConsultaComunicacionesExpedienteDesarrollo{},
		solicitudLigadaComunicacionesExpedienteDesarrollo{c, organizacion})
	return context.WithValue(peticion, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos), recurso, nil
}

// F1-150: la consulta de comunicaciones del expediente chocaba con otra ruta
// dinámica (en la copia, la cancelación del expediente) que publicaba otro
// rol en la misma versión del perfil compartido; una de las dos respondía 503.
// Con su perfil fijo, la consulta consume siempre la misma asignación aunque
// las rutas dinámicas sigan publicando a la vez, y nunca publica.
func TestConsultaComunicacionesPerfilFijoNoChocaConRutaDinamicaPostgreSQL18(t *testing.T) {
	ctx, gobierno, admin := poolesPerfilDinamicoRRHHPostgreSQLPrueba(t)
	s := soporteRRHHPostgreSQLPrueba(t, ctx, gobierno)
	if _, err := asegurarPerfilDinamicoCTDesarrollo(ctx, gobierno, s); err != nil {
		t.Fatal(err)
	}
	dinamico := s.instantanea.AsignacionPerfil.PerfilActivoRef
	v, err := s.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	historica, err := nuevaInstantaneaConsultaComunicacionesExpedienteDesarrollo(v.PrincipalID, dinamico, s.reloj.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	// Antes del arreglo: la consulta en el perfil dinámico y la otra ruta
	// publican a la vez en la misma versión y una se queda sin permiso.
	comunicaciones, err := s.autoridadAsignaciones.PrepararInstantanea(ctx, historica)
	if err != nil {
		t.Fatal(err)
	}
	cancelacion, err := s.autoridadAsignaciones.PrepararInstantanea(ctx, rolDeExpedientePrueba(s, "expediente:ct150:cancelacion"))
	if err != nil {
		t.Fatal(err)
	}
	fallidas := 0
	for _, err := range publicarAlaVezPrueba(t, ctx, admin, s, comunicaciones, cancelacion) {
		if errors.Is(err, errPostgreSQLContratacionTemporalDesarrolloNoDisponible) {
			fallidas++
		} else if err != nil {
			t.Fatal(err)
		}
	}
	if fallidas != 1 {
		t.Fatalf("el choque del perfil dinámico no se reprodujo: %d fallidas", fallidas)
	}

	// Con el arreglo: perfil fijo propio, provisto una vez como al arrancar.
	fijo, err := nuevoPerfilFijoConsultaComunicacionesExpedienteDesarrollo(principalDeSoportePrueba(s), s.contexto, s.reloj.Ahora())
	if err != nil || s.registrarPerfilFijoCTDesarrollo(fijo) != nil {
		t.Fatal("perfil fijo de comunicaciones no compuesto", err)
	}
	if fijo.perfilRef() == dinamico || fijo.plantilla.VersionRol.Referencia() != historica.VersionRol.Referencia() {
		t.Fatal("el perfil fijo reutiliza el perfil dinámico o introduce otro rol")
	}
	if err := asegurarPerfilesFijosCTDesarrollo(ctx, gobierno, s, aprobacionProvisionPerfilesRRHHDesarrollo{}, fijo); err != nil {
		t.Fatalf("provisión del perfil fijo: %v", err)
	}
	historiaFijo := historiaPerfilPostgreSQLPrueba(t, ctx, admin, fijo.perfilRef())
	huellaFijo := huellaAsignacionVigentePrueba(t, ctx, admin, fijo.perfilRef())
	if historiaFijo.versiones != 1 || historiaFijo.acto != actoAsignacionPerfilFijoCTDesarrollo {
		t.Fatalf("provisión no única: %+v", historiaFijo)
	}
	peticion, recurso, err := peticionComunicacionesPrueba(ctx, s, fijo, organizacionAltaContratacionTemporalDesarrollo)
	if err != nil {
		t.Fatal(err)
	}
	leer := func() bool {
		i, ok := s.instantaneaParaContexto(peticion, httpinterno.RutaConsultaComunicacionesExpediente)
		return ok && i.AsignacionPerfil.PerfilActivoRef == fijo.perfilRef() && i.AsignacionPerfil.Cubre(recurso) &&
			i.VersionRol.Referencia() == historica.VersionRol.Referencia()
	}
	antesDinamico := historiaPerfilPostgreSQLPrueba(t, ctx, admin, dinamico)
	x, err := s.autoridadAsignaciones.PrepararInstantanea(ctx, rolDeExpedientePrueba(s, "expediente:ct150:x"))
	if err != nil {
		t.Fatal(err)
	}
	y, err := s.autoridadAsignaciones.PrepararInstantanea(ctx, rolDeExpedientePrueba(s, "expediente:ct150:y"))
	if err != nil {
		t.Fatal(err)
	}
	var lecturas, fallos atomic.Int64
	var grupo sync.WaitGroup
	parar := make(chan struct{})
	for range 8 {
		grupo.Go(func() {
			for {
				select {
				case <-parar:
					return
				default:
				}
				lecturas.Add(1)
				if !leer() {
					fallos.Add(1)
				}
			}
		})
	}
	// Con las dos publicaciones dinámicas en cola y al soltarlas, la consulta
	// sigue leyendo su permiso sin esperar al perfil dinámico.
	resultados := publicarAlaVezConEsperaPrueba(t, ctx, admin, s, func() {
		for range 5 {
			if !leer() {
				fallos.Add(1)
			}
		}
	}, x, y)
	close(parar)
	grupo.Wait()
	if fallos.Load() != 0 || lecturas.Load() == 0 {
		t.Fatalf("%d de %d lecturas de comunicaciones fallaron durante el choque", fallos.Load(), lecturas.Load())
	}
	if errores := len(resultados); errores != 2 || (resultados[0] == nil) == (resultados[1] == nil) {
		t.Fatalf("las rutas dinámicas no chocaron entre sí: %v", resultados)
	}
	if historiaPerfilPostgreSQLPrueba(t, ctx, admin, dinamico) == antesDinamico {
		t.Fatal("el perfil dinámico no avanzó: la prueba no ejercitó la carrera")
	}
	if historiaPerfilPostgreSQLPrueba(t, ctx, admin, fijo.perfilRef()) != historiaFijo ||
		huellaAsignacionVigentePrueba(t, ctx, admin, fijo.perfilRef()) != huellaFijo {
		t.Fatal("una ruta dinámica o la consulta escribieron en el perfil fijo")
	}

	// Denegación por defecto: otra organización o el perfil revocado no
	// obtienen permiso, y las rutas dinámicas no reviven la revocación.
	ajena, _, err := peticionComunicacionesPrueba(ctx, s, fijo, "organizacion:desarrollo:ajena")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.instantaneaParaContexto(ajena, httpinterno.RutaConsultaComunicacionesExpediente); ok {
		t.Fatal("se concedió la consulta de otra organización")
	}
	revocarPerfilFijoPrueba(t, ctx, gobierno, fijo, false)
	revocada := historiaPerfilPostgreSQLPrueba(t, ctx, admin, fijo.perfilRef())
	if err := peticionRRHHPrueba(ctx, s, rolDeExpedientePrueba(s, "expediente:ct150:tras-revocar")); err != nil {
		t.Fatalf("la ruta dinámica dejó de servirse: %v", err)
	}
	for range 2 {
		if leer() {
			t.Fatal("el perfil revocado siguió leyendo comunicaciones")
		}
	}
	if historiaPerfilPostgreSQLPrueba(t, ctx, admin, fijo.perfilRef()) != revocada {
		t.Fatal("la revocación del perfil fijo se reescribió")
	}
}
