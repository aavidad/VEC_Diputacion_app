package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	httpinterno "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type auditorDenegacionEntregaPreV3Prueba struct {
	ordenes []vecports.OrdenAuditoriaFronteraRutaExacta
	err     error
}

type lectorEstadoEntregaPerfilPrueba struct {
	publicada instantaneaPublicadaDesarrollo
	err       error
	lecturas  int
}

func (l *lectorEstadoEntregaPerfilPrueba) PrepararInstantanea(_ context.Context, i vecdomain.InstantaneaAutorizacion) (vecdomain.InstantaneaAutorizacion, error) {
	return i, nil
}
func (l *lectorEstadoEntregaPerfilPrueba) PublicarInstantanea(context.Context, vecdomain.InstantaneaAutorizacion) error {
	return errors.New("publicación por petición prohibida")
}
func (l *lectorEstadoEntregaPerfilPrueba) leerAsignacionPublicada(context.Context, string) (instantaneaPublicadaDesarrollo, bool, error) {
	l.lecturas++
	return l.publicada, true, l.err
}

func TestEntregaDistingueRevocacionDeFalloLectorEnGETyPOST(t *testing.T) {
	for _, metodo := range []string{"GET", "POST"} {
		t.Run(metodo, func(t *testing.T) {
			s, _ := escenarioPerfilesFijosPrueba(t)
			_, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
			principal.ID, principal.Attributes["certificate_sha256"] = s.principalID, s.certificadoSHA256
			fijo := s.perfilFijoParaRutaYMetodo(rutaEntregaPeticionCentro, metodo)
			if fijo == nil {
				t.Fatal("perfil ausente")
			}
			s.mu.Lock()
			fijo.contextoEsperadoRegistrado = fijo.contexto.Resultado
			fijo.sesionOperativa = proveedorSesionOperativaCTPrueba{contexto: fijo.contexto}
			s.mu.Unlock()
			revocada := clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(fijo.plantilla)
			revocada.AsignacionPerfil.Estado = vecdomain.EstadoAsignacionPerfilRevocada
			revocada.AsignacionPerfil.RevocadaPor = "seguridad:prueba"
			revocada.AsignacionPerfil.RevocadaEn = revocada.AsignacionPerfil.EmitidaEn.Add(time.Second)
			revocada.AsignacionPerfil.RevocacionRef = "revocacion:prueba:entrega"
			if revocada.Validar() != nil {
				t.Fatal("fixture de revocación inválida")
			}
			lector := &lectorEstadoEntregaPerfilPrueba{publicada: instantaneaPublicadaDesarrollo{
				instantanea: revocada, actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo}}
			s.autoridadAsignaciones = lector
			auditor := &auditorDenegacionEntregaPreV3Prueba{}
			proveedor := &proveedorEntregaPeticionDesarrollo{alta: &dependenciasAltaContratacionTemporalDesarrollo{soporte: s},
				reloj: relojContratacionTemporalDesarrollo{}, auditor: auditor}
			ahora := proveedor.reloj.Ahora()
			ctx := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{},
				capacidadConsultaContratacionTemporalDesarrollo{
					sello: s.sello, ruta: rutaEntregaPeticionCentro, metodo: metodo, principal: principal,
					certificadoVerificadoEn: ahora, certificadoValidoHasta: ahora.Add(time.Hour),
					contextoOperacion: &contextoOperacionCTDesarrollo{},
				})
			comprobar := func() error {
				if metodo == "POST" {
					return proveedor.ComprobarPerfilEntregaPeticionCentro(ctx)
				}
				v, err := fijo.contexto.Vinculo.Datos()
				if err != nil {
					return err
				}
				_, err = proveedor.AutorizarEntregaPeticionCentro(ctx,
					ports.MaterialEntregaPeticionCentro{Modo: "bandeja", ActorRef: v.PrincipalID, PerfilRef: v.PerfilActivoRef})
				return err
			}
			if err := comprobar(); !errors.Is(err, ports.ErrAutorizacionDenegada) ||
				!errors.Is(err, vecdomain.ErrAutorizacionDenegada) ||
				errors.Is(err, ports.ErrPeticionCentroNoDisponible) || len(auditor.ordenes) != 1 {
				t.Fatalf("revocación no dio 403 auditado: %v órdenes=%d", err, len(auditor.ordenes))
			}
			lector.err = errors.New("lector SQL caído")
			if err := comprobar(); !errors.Is(err, ports.ErrPeticionCentroNoDisponible) ||
				errors.Is(err, ports.ErrAutorizacionDenegada) || len(auditor.ordenes) != 1 {
				t.Fatalf("fallo lector no dio 503 sin auditar denegación: %v órdenes=%d", err, len(auditor.ordenes))
			}
			if lector.lecturas != 2 {
				t.Fatalf("lector de estado no reconsultado: %d", lector.lecturas)
			}
		})
	}
}

func entregaPreparadaPerfilFijoPrueba(actor, perfil string, ahora time.Time) ports.EntregaPeticionCentro {
	return ports.EntregaPeticionCentro{EstadoEntrega: "preparada",
		ClaveAlta:      "c60518b7-f8b4-4fe7-b17e-635c46ac2e11",
		AmbitoAltaHMAC: "hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:" + strings.Repeat("a", 64),
		ActorRef:       actor, PerfilRef: perfil,
		Peticion: domain.DatosPeticionCentro{Referencia: "peticion:centro:001", Version: 2,
			Estado: "ratificada", CreadaEn: ahora, RatificadaEn: ahora.Add(time.Second),
			MotivoRatificacion: "Revisión sintética",
			Configuracion: domain.ConfiguracionPeticionCentro{Referencia: "config:001", Version: 1,
				Solicitante: domain.ActorPeticionCentro{ActorRef: "actor:solicita", PerfilRef: "perfil:centro", CentroRef: "centro-520", PuestoRef: "puesto:001"},
				Ratificador: domain.ActorPeticionCentro{ActorRef: "actor:ratifica", PerfilRef: "perfil:centro", CentroRef: "centro-520", PuestoRef: "puesto:002"}},
			Solicitud: domain.SolicitudCentro{CentroRef: "centro-520", ContactoRef: "contacto:001",
				CategoriaRef: categoriaAltaContratacionTemporalDesarrollo, GrupoSubgrupo: "C2",
				MotivoClave: "sustitucion", Detalle: "Necesidad sintética",
				Periodo: domain.PeriodoPrevisto{Inicio: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
					Fin: time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)}}}}
}

func TestEntregaYAltaAnidadaUsanCentroOrganizacionYReservaExacta(t *testing.T) {
	s, autoridad := escenarioPerfilesFijosPrueba(t)
	_, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	principal.ID, principal.Attributes["certificate_sha256"] = s.principalID, s.certificadoSHA256
	fijo := s.perfilFijoParaRutaYMetodo(rutaEntregaPeticionCentro, "POST")
	lector := s.perfilFijoParaRutaYMetodo(rutaEntregaPeticionCentro, "GET")
	if fijo == nil || lector == nil {
		t.Fatal("perfiles GET/POST ausentes")
	}
	v, err := fijo.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	e := entregaPreparadaPerfilFijoPrueba(v.PrincipalID, v.PerfilActivoRef, ahora)
	if e.ValidarReserva() != nil {
		t.Fatal("reserva de prueba inválida")
	}
	autoridad.asignaciones = map[string]instantaneaPublicadaDesarrollo{
		fijo.perfilRef(): {instantanea: fijo.plantilla, actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo}}
	capacidad := capacidadConsultaContratacionTemporalDesarrollo{sello: s.sello, ruta: rutaEntregaPeticionCentro,
		metodo: "POST", principal: principal}
	ctx := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	m := ports.MaterialEntregaPeticionCentro{Modo: "preparar", ActorRef: v.PrincipalID,
		PerfilRef: v.PerfilActivoRef, PeticionRef: e.Peticion.Referencia, VersionEsperada: 2,
		CentroRef: e.Peticion.Solicitud.CentroRef, CategoriaRef: e.Peticion.Solicitud.CategoriaRef,
		ClaveAltaCandidata: e.ClaveAlta, AmbitoAltaHMAC: e.AmbitoAltaHMAC}
	rEntrega, err := postgresct.RecursoEntregaPeticionCentro(m)
	if err != nil {
		t.Fatal(err)
	}
	dEntrega := vecdomain.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: fijo.contexto.Vinculo,
		ReferenciaMotivo: motivoEntregaPeticionDesarrollo(), Accion: ports.AccionEntregarPeticionRRHH,
		Recurso: rEntrega, Finalidad: ports.FinalidadEntregaPeticionCentro}
	ctxEntrega := context.WithValue(ctx, claveMaterialEntregaPeticionDesarrollo{}, m)
	ctxEntrega = context.WithValue(ctxEntrega, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, dEntrega)
	if _, ok := s.instantaneaPerfilFijoParaContexto(ctxEntrega, rutaEntregaPeticionCentro, fijo); !ok {
		t.Fatal("entrega no consumió asignación publicada")
	}
	dAlta := vecdomain.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: fijo.contexto.Vinculo,
		Accion: ports.AccionCrearSolicitud, Finalidad: ports.FinalidadCrearSolicitud,
		Recurso: vecdomain.RecursoAutorizable{Referencia: "expediente:001", ModuloID: ports.ModuloContratacion,
			Tipo: ports.TipoRecursoExpediente,
			Ambitos: map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo,
				"centro_ref": "centro-520", "categoria_ref": categoriaAltaContratacionTemporalDesarrollo}}}
	ctxAlta := context.WithValue(ctx, claveAltaDePeticionDesarrollo{}, e)
	ctxAlta = context.WithValue(ctxAlta, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, dAlta)
	if _, ok := s.instantaneaPerfilFijoParaContexto(ctxAlta, rutaEntregaPeticionCentro, fijo); !ok {
		t.Fatal("alta anidada no consumió el mismo perfil y reserva")
	}
	solicitud := ports.SolicitudResolverFlujo{OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo,
		CentroRef: "centro-520", CategoriaRef: categoriaAltaContratacionTemporalDesarrollo,
		MotivoClave: motivoAltaContratacionTemporalDesarrollo, Instante: ahora}
	if _, err := s.ResolverFlujoAlta(ctxAlta, solicitud); err != nil {
		t.Fatalf("alta anidada con centro de organización denegada: %v", err)
	}
	solicitud.CentroRef = "centro:rpt:520"
	if _, err := s.ResolverFlujoAlta(ctxAlta, solicitud); !errors.Is(err, ports.ErrFlujoNoDisponible) {
		t.Fatalf("alias de alta directa saltó la reserva: %v", err)
	}
	solicitud.CentroRef = "centro-520"
	capacidad.ruta = httpinterno.RutaAltaSolicitudes
	ctxDirecto := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	if _, err := s.ResolverFlujoAlta(ctxDirecto, solicitud); !errors.Is(err, ports.ErrFlujoNoDisponible) {
		t.Fatalf("alta directa aceptó centro de petición: %v", err)
	}
	dAlta.Recurso.Ambitos["centro_ref"] = "centro:rpt:520"
	ctxAlias := context.WithValue(ctxAlta, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, dAlta)
	if _, ok := s.instantaneaPerfilFijoParaContexto(ctxAlias, rutaEntregaPeticionCentro, fijo); ok {
		t.Fatal("alta anidada aceptó alias distinto de la reserva")
	}
	dAlta.Recurso.Ambitos["centro_ref"] = "centro-520"
	dAlta.Recurso.Ambitos["categoria_ref"] = "categoria:ajena"
	ctxCategoria := context.WithValue(ctxAlta, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, dAlta)
	if _, ok := s.instantaneaPerfilFijoParaContexto(ctxCategoria, rutaEntregaPeticionCentro, fijo); ok {
		t.Fatal("alta anidada aceptó categoría ajena")
	}
	dAlta.Recurso.Ambitos["categoria_ref"] = categoriaAltaContratacionTemporalDesarrollo
	ctxSinReserva := context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, dAlta)
	if _, ok := s.instantaneaPerfilFijoParaContexto(ctxSinReserva, rutaEntregaPeticionCentro, fijo); ok {
		t.Fatal("alta anidada sin reserva confiable")
	}
	if _, ok := s.instantaneaPerfilFijoParaContexto(ctxAlta, rutaEntregaPeticionCentro, lector); ok {
		t.Fatal("lector GET autorizó alta anidada")
	}
}

func (a *auditorDenegacionEntregaPreV3Prueba) RegistrarAuditoriaFronteraRutaExacta(
	_ context.Context, orden vecports.OrdenAuditoriaFronteraRutaExacta,
) error {
	a.ordenes = append(a.ordenes, orden)
	return a.err
}

func TestDenegacionEntregaPreV3RegistraCausaFijaSinPeticion(t *testing.T) {
	auditor := &auditorDenegacionEntregaPreV3Prueba{}
	p := &proveedorEntregaPeticionDesarrollo{auditor: auditor}
	if err := p.registrarDenegacionPreV3(context.Background(), "actor:rrhh"); err != nil {
		t.Fatal(err)
	}
	if len(auditor.ordenes) != 1 {
		t.Fatal("la denegación previa a V3 no llegó a auditoría")
	}
	o := auditor.ordenes[0]
	if o.Validar() != nil || o.Motivo != vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado ||
		o.Superficie != vecports.SuperficieAuditoriaFronteraRutaExactaContratacionTemporal ||
		o.Ruta != rutaEntregaPeticionCentro || o.ActorRef != "actor:rrhh" {
		t.Fatalf("orden de auditoría no minimizada: %+v", o)
	}
}

func TestPerfilEntregaNoPublicadoAuditaAntesDeProyectarAmbitos(t *testing.T) {
	s, _ := escenarioPerfilesFijosPrueba(t)
	_, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	principal.ID, principal.Attributes["certificate_sha256"] = s.principalID, s.certificadoSHA256
	fijo := s.perfilFijoParaRutaYMetodo(rutaEntregaPeticionCentro, "POST")
	if fijo == nil {
		t.Fatal("sin perfil de entrega")
	}
	s.mu.Lock()
	fijo.contextoEsperadoRegistrado = fijo.contexto.Resultado
	fijo.sesionOperativa = proveedorSesionOperativaCTPrueba{contexto: fijo.contexto}
	s.mu.Unlock()
	auditor := &auditorDenegacionEntregaPreV3Prueba{}
	p := &proveedorEntregaPeticionDesarrollo{
		alta:  &dependenciasAltaContratacionTemporalDesarrollo{soporte: s},
		reloj: relojContratacionTemporalDesarrollo{}, auditor: auditor,
	}
	ahora := p.reloj.Ahora()
	ctx := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{},
		capacidadConsultaContratacionTemporalDesarrollo{
			sello: s.sello, ruta: rutaEntregaPeticionCentro, metodo: "POST", principal: principal,
			certificadoVerificadoEn: ahora, certificadoValidoHasta: ahora.Add(time.Hour),
			contextoOperacion: &contextoOperacionCTDesarrollo{},
		})
	if err := p.ComprobarPerfilEntregaPeticionCentro(ctx); !errors.Is(err, ports.ErrAutorizacionDenegada) {
		t.Fatalf("asignación ausente llegó a proyección: %v", err)
	}
	if len(auditor.ordenes) != 1 || auditor.ordenes[0].ActorRef == "" ||
		auditor.ordenes[0].Ruta != rutaEntregaPeticionCentro {
		t.Fatalf("denegación previa a V3 no auditada: %+v", auditor.ordenes)
	}
	auditor.err = errors.New("auditoría de prueba indisponible")
	if err := p.ComprobarPerfilEntregaPeticionCentro(ctx); !errors.Is(err, ports.ErrPeticionCentroNoDisponible) {
		t.Fatalf("auditoría indisponible no detuvo la proyección con 503: %v", err)
	}
}

func TestEntregaPeticionDesarrolloLigaIdentidadMaterialYAltaOriginal(t *testing.T) {
	p := vecdomain.Principal{ID: "desarrollo:rrhh", DisplayName: "RRHH sintético", Roles: []string{rolTecnicoRRHHContratacionTemporalDesarrollo}, AuthMethod: vecdomain.AuthMethodCertificate, AuthAssurance: vecdomain.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment, "certificate_sha256": strings.Repeat("a", 64)}}
	ahora := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	c, err := nuevoContextoSinteticoContratacionTemporalDesarrollo(p, ahora)
	if err != nil {
		t.Fatal(err)
	}
	v, err := c.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	m := ports.MaterialEntregaPeticionCentro{Modo: "bandeja", ActorRef: v.PrincipalID, PerfilRef: v.PerfilActivoRef}
	r, err := postgresct.RecursoEntregaPeticionCentro(m)
	if err != nil {
		t.Fatal(err)
	}
	d := vecdomain.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: c.Vinculo, ReferenciaMotivo: motivoEntregaPeticionDesarrollo(), Accion: ports.AccionConsultarPeticionesRRHH, Recurso: r, Finalidad: ports.FinalidadEntregaPeticionCentro}
	ctx := context.WithValue(context.Background(), claveMaterialEntregaPeticionDesarrollo{}, m)
	if !solicitudAutorizacionEntregaPeticionValida(ctx, d) {
		t.Fatal("consulta propia denegada")
	}
	d.Recurso.Atributos["material_sha256"] = strings.Repeat("b", 64)
	if solicitudAutorizacionEntregaPeticionValida(ctx, d) {
		t.Fatal("material alterado autorizado")
	}
	if solicitudAutorizacionEntregaPeticionValida(context.Background(), d) {
		t.Fatal("cliente sin material de servidor autorizado")
	}
	for _, rol := range []string{"solicitante_centro", "ratificador_centro", "intervencion"} {
		p.Roles = []string{rol}
		if principalContratacionTemporalDesarrolloValidoParaRuta(p, rutaEntregaPeticionCentro) {
			t.Fatal("rol ajeno obtiene recepción RRHH", rol)
		}
	}
	e := ports.EntregaPeticionCentro{EstadoEntrega: "preparada", ClaveAlta: "c60518b7-f8b4-4fe7-b17e-635c46ac2e11", AmbitoAltaHMAC: "hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:" + strings.Repeat("a", 64), ActorRef: v.PrincipalID, PerfilRef: v.PerfilActivoRef,
		Peticion: domain.DatosPeticionCentro{Referencia: "peticion:centro:001", Version: 2, Estado: "ratificada", CreadaEn: ahora, RatificadaEn: ahora.Add(time.Second), MotivoRatificacion: "Revisión sintética",
			Configuracion: domain.ConfiguracionPeticionCentro{Referencia: "config:001", Version: 1, Solicitante: domain.ActorPeticionCentro{ActorRef: "actor:solicita", PerfilRef: "perfil:centro", CentroRef: "centro-520", PuestoRef: "puesto:001"}, Ratificador: domain.ActorPeticionCentro{ActorRef: "actor:ratifica", PerfilRef: "perfil:centro", CentroRef: "centro-520", PuestoRef: "puesto:002"}},
			Solicitud:     domain.SolicitudCentro{CentroRef: "centro-520", ContactoRef: "contacto:001", CategoriaRef: categoriaAltaContratacionTemporalDesarrollo, GrupoSubgrupo: "C2", MotivoClave: "sustitucion", Detalle: "Necesidad sintética", Periodo: domain.PeriodoPrevisto{Inicio: time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC), Fin: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)}}}}
	if e.ValidarReserva() != nil {
		t.Fatal("fixture de reserva inválido")
	}
	d.Accion, d.Finalidad = ports.AccionCrearSolicitud, ports.FinalidadCrearSolicitud
	d.Recurso = vecdomain.RecursoAutorizable{Referencia: "expediente:001", ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoExpediente, Ambitos: map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo, "centro_ref": "centro-520", "categoria_ref": categoriaAltaContratacionTemporalDesarrollo}}
	ctx = context.WithValue(context.Background(), claveAltaDePeticionDesarrollo{}, e)
	if !solicitudAutorizacionAltaDePeticionValida(ctx, d) {
		t.Fatal("alta de original ratificado denegada")
	}
	d.Recurso.Ambitos["centro_ref"] = "centro:otro"
	if solicitudAutorizacionAltaDePeticionValida(ctx, d) {
		t.Fatal("alta de otro centro autorizada")
	}
	if solicitudAutorizacionAltaDePeticionValida(context.Background(), d) {
		t.Fatal("alta sin reserva confiable autorizada")
	}
}

type selladorEntregaPeticionPrueba struct{ clave, perfil string }

func (s *selladorEntregaPeticionPrueba) SellarAmbitoIdempotencia(_ context.Context, m ports.SolicitudSellarAmbitoIdempotencia) (ports.ColeccionSellosHMAC, error) {
	s.clave = m.ClaveIdempotencia
	s.perfil = m.PerfilRef
	return ports.NuevaColeccionSellosHMAC("hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:"+strings.Repeat("a", 64), nil)
}

type derivadorHuellaEntregaOriginalPrueba struct{ material ports.MaterialHuellaAlta }

func (d *derivadorHuellaEntregaOriginalPrueba) DerivarHuellaAlta(_ context.Context, m ports.MaterialHuellaAlta) (ports.ColeccionSellosHMAC, error) {
	d.material = m
	return ports.NuevaColeccionSellosHMAC("hmac-sha256:vec.contratacion-temporal.huella-peticion/v1:"+strings.Repeat("b", 64), nil)
}

func TestVerificadorOriginalNoConcedePerfilHistoricoYRechazaHuellaMOAD(t *testing.T) {
	s, _ := escenarioPerfilesFijosPrueba(t)
	_, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	principal.ID, principal.Attributes["certificate_sha256"] = s.principalID, s.certificadoSHA256
	fijo := s.perfilFijoParaRutaYMetodo(rutaEntregaPeticionCentro, "POST")
	if fijo == nil {
		t.Fatal("perfil fijo de entrega ausente")
	}
	s.mu.Lock()
	fijo.contextoEsperadoRegistrado = fijo.contexto.Resultado
	fijo.sesionOperativa = proveedorSesionOperativaCTPrueba{contexto: fijo.contexto}
	s.mu.Unlock()
	v, err := fijo.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	capacidad := capacidadConsultaContratacionTemporalDesarrollo{
		sello: s.sello, ruta: rutaEntregaPeticionCentro, metodo: "POST", principal: principal,
		certificadoVerificadoEn: ahora, certificadoValidoHasta: ahora.Add(time.Hour),
		contextoOperacion: &contextoOperacionCTDesarrollo{},
	}
	ctx := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	ambitos := &selladorEntregaPeticionPrueba{}
	huellas := &derivadorHuellaEntregaOriginalPrueba{}
	s.ambitos = ambitos
	p := &proveedorEntregaPeticionDesarrollo{alta: &dependenciasAltaContratacionTemporalDesarrollo{soporte: s, huellas: huellas}, reloj: relojContratacionTemporalDesarrollo{}}
	e := entregaPreparadaPerfilFijoPrueba(v.PrincipalID, "perfil:rrhh:historico", ahora)
	recibo := ports.ReciboAlta{ExpedienteRef: "expediente:ct:historico", NumeroVisible: "2026/CT-0001", Version: 1,
		ReciboRef: "recibo:ct:historico", AuditoriaRef: "auditoria:ct:historico", EventoRef: "evento:ct:historico", ConfirmadaEn: ahora}
	original := ports.OriginalAltaEntrega{
		Esquema:         "vec.contratacion-temporal.original-alta-entrega.v1",
		OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo,
		ActorRef:        e.ActorRef, PerfilRef: e.PerfilRef,
		Flujo:              domain.ReferenciaFlujo{DefinicionRef: "flujo:ct:historico", Version: 1, HuellaSHA256: strings.Repeat("c", 64)},
		AmbitoHMAC:         e.AmbitoAltaHMAC,
		HuellaPeticionHMAC: "hmac-sha256:vec.contratacion-temporal.huella-peticion/v1:" + strings.Repeat("b", 64),
		ReciboAlta:         recibo,
	}
	if err := p.VerificarOriginalAltaEntrega(ctx, e, original); err != nil ||
		ambitos.perfil != e.PerfilRef || huellas.material.PerfilRef != e.PerfilRef ||
		huellas.material.NumeroExpedienteMOAD != "" {
		t.Fatalf("original con perfil histórico: error=%v ámbito=%q huella=%+v", err, ambitos.perfil, huellas.material)
	}
	politica := domain.PoliticaFin{ReglaRef: "regla:ct:fin-historica", CatalogoVersion: 2,
		CatalogoHuellaSHA256: strings.Repeat("e", 64), FechaFin: "opcional", CausaFin: "fin_sustitucion"}
	e.Peticion.Solicitud.Periodo.Fin = time.Time{}
	e.Peticion.Solicitud.Periodo.CausaFin = "fin_sustitucion"
	original.PoliticaFin = &politica
	if err := p.VerificarOriginalAltaEntrega(ctx, e, original); err != nil ||
		huellas.material.Solicitud.Periodo.PoliticaFin != politica {
		t.Fatalf("política Fin original perdida antes de huella: %v", err)
	}
	original.HuellaPeticionHMAC = "hmac-sha256:vec.contratacion-temporal.huella-peticion/v1:" + strings.Repeat("d", 64)
	if err := p.VerificarOriginalAltaEntrega(ctx, e, original); !errors.Is(err, ports.ErrClaveIdempotenciaUsada) {
		t.Fatalf("alta MOAD sin número recuperada como anterior: %v", err)
	}
}

func TestEntregaPeticionDesarrolloRechazaSelloDeOtraClaveAntesDeAutorizar(t *testing.T) {
	p := vecdomain.Principal{ID: "desarrollo:rrhh", DisplayName: "RRHH sintético", Roles: []string{rolTecnicoRRHHContratacionTemporalDesarrollo}, AuthMethod: vecdomain.AuthMethodCertificate, AuthAssurance: vecdomain.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment, "certificate_sha256": strings.Repeat("a", 64)}}
	ahora := relojContratacionTemporalDesarrollo{}.Ahora()
	c, err := nuevoContextoSinteticoContratacionTemporalDesarrollo(p, ahora)
	if err != nil {
		t.Fatal(err)
	}
	v, err := c.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	sello := &selloConsultasContratacionTemporalDesarrollo{}
	hmac := &selladorEntregaPeticionPrueba{}
	soporte := &soporteAltaContratacionTemporalDesarrollo{sello: sello, principalID: p.ID, certificadoSHA256: p.Attributes["certificate_sha256"], contexto: c, ambitos: hmac}
	soporte.contextoEsperadoRegistrado = c.Resultado
	soporte.sesionOperativa = proveedorSesionOperativaCTPrueba{contexto: c}
	proveedor := &proveedorEntregaPeticionDesarrollo{alta: &dependenciasAltaContratacionTemporalDesarrollo{soporte: soporte},
		auditor: &auditorDenegacionEntregaPreV3Prueba{}}
	ctx := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidadConsultaContratacionTemporalDesarrollo{sello: sello, ruta: rutaEntregaPeticionCentro, metodo: "POST", principal: p, certificadoVerificadoEn: ahora, certificadoValidoHasta: ahora.Add(time.Hour), contextoOperacion: &contextoOperacionCTDesarrollo{}})
	clave, selloCorrecto, err := proveedor.NuevaClaveAltaDePeticion(ctx)
	if err != nil || !ports.ClaveIdempotenciaValida(clave) || hmac.clave != clave {
		t.Fatal("clave y sello no ligados", err)
	}
	m := ports.MaterialEntregaPeticionCentro{Modo: "preparar", ActorRef: v.PrincipalID, PerfilRef: v.PerfilActivoRef, PeticionRef: "peticion:001", CentroRef: centroAltaContratacionTemporalDesarrollo, CategoriaRef: categoriaAltaContratacionTemporalDesarrollo, VersionEsperada: 2, ClaveAltaCandidata: clave, AmbitoAltaHMAC: strings.Replace(selloCorrecto, strings.Repeat("a", 64), strings.Repeat("b", 64), 1)}
	if _, err := proveedor.AutorizarEntregaPeticionCentro(ctx, m); !errors.Is(err, ports.ErrAutorizacionDenegada) {
		t.Fatal("sello cruzado alcanzó autorización", err)
	}
}
