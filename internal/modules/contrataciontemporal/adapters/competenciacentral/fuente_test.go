package competenciacentral

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type identidadPrueba struct {
	vinculo  VinculoCertificadoFirmante
	err      error
	llamadas int
}

func (i *identidadPrueba) ResolverFirmantePorCertificado(context.Context, string) (VinculoCertificadoFirmante, error) {
	i.llamadas++
	return i.vinculo, i.err
}

type asignacionPrueba struct {
	evidencia vecdomain.EvidenciaAsignacionCompetencialV1
	err       error
	llamadas  int
	solicitud vecdomain.SolicitudAsignacionCompetencialV1
}

func (a *asignacionPrueba) LeerAsignacionCompetencialV1(_ context.Context, s vecdomain.SolicitudAsignacionCompetencialV1) (vecdomain.EvidenciaAsignacionCompetencialV1, error) {
	a.llamadas++
	a.solicitud = s
	return a.evidencia, a.err
}

type solicitudPrueba struct {
	errorAutorizacion error
	autorizaciones    int
	solicitud         vecdomain.SolicitudAsignacionCompetencialV1
	relacion          RelacionRecursoCT
	err               error
	llamadas          int
}

func (s *solicitudPrueba) AutorizarLecturaCertificado(context.Context, ctports.SolicitudCompetenciaFirmante) error {
	s.autorizaciones++
	return s.errorAutorizacion
}
func (s *solicitudPrueba) ResolverSolicitudCentral(context.Context, ctports.SolicitudCompetenciaFirmante, VinculoCertificadoFirmante) (ResolucionSolicitudCentral, error) {
	s.llamadas++
	return ResolucionSolicitudCentral{Solicitud: s.solicitud, Relacion: s.relacion}, s.err
}

type relojPrueba struct{ ahora time.Time }

func (r *relojPrueba) Ahora() time.Time { return r.ahora }
func asignacionCompetencialPrueba(t *testing.T) (vecdomain.SolicitudAsignacionCompetencialV1, vecdomain.EvidenciaAsignacionCompetencialV1, time.Time) {
	t.Helper()
	ahora := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora,
		"per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl", vecdomain.AuthMethodCertificate, vecdomain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	s := vecdomain.SolicitudAsignacionCompetencialV1{
		Actor: resultado.Contexto, ResultadoContexto: resultado, Vinculo: vinculo,
		PersonaRef: "per_0123456789abcdefghijkm", PerfilFirmanteRef: "perfil:ct:direccion_rrhh",
		CertificadoHuellaSHA256: strings.Repeat("1", 64),
		Cargo:                   vecdomain.ReferenciaCargoCompetencialV1{Referencia: "cargo:secretaria", Version: 2, HuellaSHA256: strings.Repeat("2", 64)},
		Recurso: vecdomain.RecursoAutorizable{Referencia: "documento:resolucion", ModuloID: "contratacion_temporal", Tipo: "documento",
			Ambitos:   map[string]string{"organizacion_ref": "organizacion:diputacion", "unidad_ref": "unidad:secretaria"},
			Atributos: map[string]string{"esquema_recurso": "esquema:ct:firma:v1", "expediente_ref": "expediente:prueba", "documento_ref": "documento:resolucion", "version_vinculo": "4", "prueba_vinculo_sha256": strings.Repeat("9", 64)}},
		AccionLectura: "administracion.asignaciones_competenciales.consultar", FinalidadLectura: "comprobar_competencia",
		AccionCompetencial: "contratacion_temporal.documento.firmar", FinalidadCompetencial: "formalizar",
		CorrelacionRef: "correlacion_" + strings.Repeat("a", 32),
		Motivo: vecdomain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("3", 64), EntradaClave: "motivo_" + strings.Repeat("b", 32)},
	}
	rol := vecdomain.VersionRol{RolID: "ct_cargo_direccion_rrhh", Version: 1, Nombre: "Secretaria", Estado: vecdomain.EstadoVersionRolPublicada,
		PublicadaPor: "acto:publicacion:rol", PublicadaEn: ahora.Add(-time.Hour),
		Concesiones: []vecdomain.ConcesionRol{{Accion: s.AccionCompetencial, ModuloID: s.Recurso.ModuloID, TipoRecurso: s.Recurso.Tipo,
			Finalidades: []string{s.FinalidadCompetencial}, GarantiaMinima: vecdomain.AuthAssuranceHigh}}}
	e := vecdomain.EvidenciaAsignacionCompetencialV1{
		PersonaRef: s.PersonaRef, PerfilFirmanteRef: s.PerfilFirmanteRef, PerfilActivoFirmanteRef: "prf_0123456789abcdefghijkm",
		CertificadoHuellaSHA256: s.CertificadoHuellaSHA256, UnidadRef: "unidad:secretaria", Cargo: s.Cargo,
		CargoVigenteDesde: ahora.Add(-time.Hour), CargoVigenteHasta: ahora.Add(time.Hour),
		EnlaceOcupante: vecdomain.EnlaceOcupanteCompetencialV1{Referencia: "ocupacion:secretaria", Version: 4,
			HuellaSHA256: strings.Repeat("4", 64), PersonaRef: s.PersonaRef, CargoRef: s.Cargo.Referencia,
			VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)},
		AccionCompetencial: s.AccionCompetencial, FinalidadCompetencial: s.FinalidadCompetencial,
		RecursoRef: s.Recurso.Referencia, ModuloID: s.Recurso.ModuloID, TipoRecurso: s.Recurso.Tipo,
		Asignacion: vecdomain.AsignacionPerfil{AsignacionID: "firmante-secretaria", Version: 3,
			PerfilActivoRef: "prf_0123456789abcdefghijkm", PrincipalID: s.PersonaRef, VersionRolRef: rol.Referencia(),
			Estado: vecdomain.EstadoAsignacionPerfilActiva,
			Ambitos: []vecdomain.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{"organizacion:diputacion"}},
				{Clave: "unidad_ref", Valores: []string{"unidad:secretaria"}}},
			VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), EmitidaPor: "acto:asignacion", EmitidaEn: ahora.Add(-time.Hour)},
		VersionRol: rol,
		ControlVigencia: vecdomain.ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 2,
			Estado: vecdomain.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: "acto:control", ActualizadoEn: ahora.Add(-time.Hour)},
		ActoCompetenciaRef: "acto:competencia", ComprobadaEn: ahora,
	}
	e.RecursoHuellaSHA256, err = s.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	e.AsignacionHuellaSHA256, err = e.Asignacion.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	e.VersionRolHuellaSHA256, err = e.VersionRol.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	e.ControlVigenciaHuellaSHA256, err = e.ControlVigencia.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	return s, e, ahora
}

func datosPrueba(t *testing.T) (*Fuente, *identidadPrueba, *asignacionPrueba, *solicitudPrueba, ctports.SolicitudCompetenciaFirmante) {
	t.Helper()
	s, e, ahora := asignacionCompetencialPrueba(t)
	i := &identidadPrueba{vinculo: VinculoCertificadoFirmante{CertificadoHuella: s.CertificadoHuellaSHA256, PrincipalRef: s.PersonaRef, CuentaRef: "cta_0123456789abcdefghijkm", VinculoCredencialRef: "vcc_0123456789abcdefghijkm", Revision: 1, Huella: strings.Repeat("a", 64), Vigente: true}}
	a := &asignacionPrueba{evidencia: e}
	proveedor := &solicitudPrueba{solicitud: s, relacion: RelacionRecursoCT{OrganizacionRef: s.Recurso.Ambitos["organizacion_ref"], UnidadRef: s.Recurso.Ambitos["unidad_ref"], ExpedienteRef: s.Recurso.Atributos["expediente_ref"], DocumentoClave: "resolucion", DocumentoRef: s.Recurso.Referencia, VersionExpediente: 7, VersionOrigenVinculo: 4, PruebaSnapshotOrigenHuellaSHA256: s.Recurso.Atributos["prueba_vinculo_sha256"]}}
	q := ctports.SolicitudCompetenciaFirmante{OrganizacionRef: s.Recurso.Ambitos["organizacion_ref"], ExpedienteRef: s.Recurso.Atributos["expediente_ref"], Documento: "resolucion", CatalogoRef: "catalogo:ct:v2", CatalogoVersion: 2, CatalogoHuella: strings.Repeat("b", 64), PasoRef: "catalogo:ct:v2:resolucion.p1", PasoOrden: 1, PerfilFirmanteRef: s.PerfilFirmanteRef, FirmanteRef: "ref:" + s.CertificadoHuellaSHA256, CertificadoHuella: s.CertificadoHuellaSHA256}
	f, err := NuevaFuente(i, a, proveedor, &relojPrueba{ahora}, DescriptorCircuito{CatalogoRef: q.CatalogoRef, CatalogoVersion: q.CatalogoVersion, CatalogoHuella: q.CatalogoHuella, AccionLectura: s.AccionLectura, FinalidadLectura: s.FinalidadLectura, TipoRecurso: s.Recurso.Tipo, Recurso: DescriptorRecurso{Esquema: "esquema:ct:firma:v1", AmbitoOrganizacion: "organizacion_ref", AmbitoUnidad: "unidad_ref", AtributoEsquema: "esquema_recurso", AtributoExpediente: "expediente_ref", AtributoDocumento: "documento_ref", AtributoVersionVinculo: "version_vinculo", AtributoPruebaVinculo: "prueba_vinculo_sha256"}, Pasos: []DescriptorPaso{{Documento: q.Documento, PasoRef: q.PasoRef, Orden: 1, AccionCompetencial: s.AccionCompetencial, FinalidadCompetencial: s.FinalidadCompetencial, Perfiles: map[string]string{q.PerfilFirmanteRef: e.VersionRol.RolID}}}})
	if err != nil {
		t.Fatal(err)
	}
	return f, i, a, proveedor, q
}

func TestFuenteSeparaConsultanteDeFirmanteYConservaCargoPersonal(t *testing.T) {
	f, i, a, _, q := datosPrueba(t)
	x, err := f.AcreditarCompetenciaCentral(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	e := x.Proyeccion()
	if e.Solicitud != q || e.FirmantePrincipalRef != i.vinculo.PrincipalRef || e.PerfilFirmanteRef != q.PerfilFirmanteRef || e.RolIDFirmante != "ct_cargo_direccion_rrhh" ||
		e.CuentaFirmanteRef != i.vinculo.CuentaRef || a.solicitud.Actor.PersonaRef == e.FirmantePrincipalRef ||
		a.solicitud.Actor.PerfilActivoRef == e.PerfilActivoFirmanteRef || x.evidencia.Cargo.Version != 2 || x.evidencia.EnlaceOcupante.Version != 4 || a.llamadas != 1 ||
		len(a.solicitud.Recurso.Ambitos) != 2 || !x.evidencia.Asignacion.Cubre(a.solicitud.Recurso) {
		t.Fatal("autoridades o evidencia sustituidas")
	}
	alterado, err := clonarSolicitud(x.solicitud)
	if err != nil {
		t.Fatal(err)
	}
	alterado.Recurso.Atributos[f.descriptor.Recurso.AtributoDocumento] = "documento:otro"
	huella, err := alterado.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || huella == x.evidencia.RecursoHuellaSHA256 {
		t.Fatal("documento exacto no ligado a la huella del recurso")
	}
}

func TestFuenteNoConsultaAnteCatalogoPasoPerfilOCertificadoAlterados(t *testing.T) {
	casos := map[string]func(*ctports.SolicitudCompetenciaFirmante){
		"perfil":      func(q *ctports.SolicitudCompetenciaFirmante) { q.PerfilFirmanteRef = "perfil:ct:administrador" },
		"paso":        func(q *ctports.SolicitudCompetenciaFirmante) { q.PasoOrden++ },
		"catalogo":    func(q *ctports.SolicitudCompetenciaFirmante) { q.CatalogoVersion++ },
		"certificado": func(q *ctports.SolicitudCompetenciaFirmante) { q.CertificadoHuella = strings.Repeat("f", 64) },
		"cargo":       func(q *ctports.SolicitudCompetenciaFirmante) { q.CargoFirmante = "Dirección o Jefatura" },
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			f, i, a, p, q := datosPrueba(t)
			cambiar(&q)
			_, err := f.AcreditarCompetenciaCentral(context.Background(), q)
			if !errors.Is(err, ctports.ErrCompetenciaFirmanteNoAcreditada) || i.llamadas != 0 || a.llamadas != 0 || p.llamadas != 0 {
				t.Fatal("entrada no publicada consultó autoridad")
			}
		})
	}
}

func TestFuenteRechazaCrucesServidorYAutoridadCentral(t *testing.T) {
	casos := map[string]func(*identidadPrueba, *asignacionPrueba, *solicitudPrueba){
		"vinculo revocado": func(i *identidadPrueba, _ *asignacionPrueba, _ *solicitudPrueba) { i.vinculo.Vigente = false },
		"persona": func(_ *identidadPrueba, _ *asignacionPrueba, p *solicitudPrueba) {
			p.solicitud.PersonaRef = "per_0123456789abcdefghijkx"
		},
		"expediente": func(_ *identidadPrueba, _ *asignacionPrueba, p *solicitudPrueba) {
			p.solicitud.Recurso.Atributos["expediente_ref"] = "expediente:otro"
		},
		"perfil": func(_ *identidadPrueba, _ *asignacionPrueba, p *solicitudPrueba) {
			p.solicitud.PerfilFirmanteRef = "perfil:ct:otro"
		},
		"certificado": func(_ *identidadPrueba, _ *asignacionPrueba, p *solicitudPrueba) {
			p.solicitud.CertificadoHuellaSHA256 = strings.Repeat("e", 64)
		},
		"sesion consultante": func(_ *identidadPrueba, _ *asignacionPrueba, p *solicitudPrueba) {
			p.solicitud.Vinculo = vecdomain.VinculoAutenticacionActorV2{}
		},
		"persona asignacion": func(_ *identidadPrueba, a *asignacionPrueba, _ *solicitudPrueba) {
			a.evidencia.Asignacion.PrincipalID = "per_0123456789abcdefghijkx"
		},
		"ocupante": func(_ *identidadPrueba, a *asignacionPrueba, _ *solicitudPrueba) {
			a.evidencia.EnlaceOcupante.PersonaRef = "per_0123456789abcdefghijkx"
		},
		"caducidad": func(_ *identidadPrueba, a *asignacionPrueba, _ *solicitudPrueba) {
			a.evidencia.CargoVigenteHasta = a.evidencia.ComprobadaEn
		},
		"huella rol": func(_ *identidadPrueba, a *asignacionPrueba, _ *solicitudPrueba) {
			a.evidencia.VersionRolHuellaSHA256 = strings.Repeat("e", 64)
		},
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			f, i, a, p, q := datosPrueba(t)
			cambiar(i, a, p)
			_, err := f.AcreditarCompetenciaCentral(context.Background(), q)
			if !errors.Is(err, ctports.ErrCompetenciaFirmanteNoAcreditada) {
				t.Fatal("cruce acreditado", err)
			}
		})
	}
}

func TestFuenteOcultaErrorCentralYConservaCancelacion(t *testing.T) {
	f, _, a, _, q := datosPrueba(t)
	a.err = errors.New("dsn privado")
	if _, err := f.AcreditarCompetenciaCentral(context.Background(), q); err != ctports.ErrCompetenciaFirmanteNoDisponible {
		t.Fatal("error central expuesto")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := f.AcreditarCompetenciaCentral(ctx, q); err != context.Canceled {
		t.Fatal("cancelación ignorada")
	}
	if _, err := (&Fuente{}).AcreditarCompetenciaFirmante(context.Background(), q); err != ctports.ErrCompetenciaFirmanteNoDisponible {
		t.Fatal("fuente vacía")
	}
}

func TestFuenteNoAceptaAccionesOFinalidadesCruzadas(t *testing.T) {
	for _, campo := range []string{"accion lectura", "finalidad lectura", "accion firmante", "finalidad firmante", "tipo"} {
		t.Run(campo, func(t *testing.T) {
			f, _, a, p, q := datosPrueba(t)
			switch campo {
			case "accion lectura":
				p.solicitud.AccionLectura = p.solicitud.AccionCompetencial
			case "finalidad lectura":
				p.solicitud.FinalidadLectura = p.solicitud.FinalidadCompetencial
			case "accion firmante":
				p.solicitud.AccionCompetencial = p.solicitud.AccionLectura
			case "finalidad firmante":
				p.solicitud.FinalidadCompetencial = p.solicitud.FinalidadLectura
			case "tipo":
				p.solicitud.Recurso.Tipo = "recurso:otro"
			}
			if _, err := f.AcreditarCompetenciaCentral(context.Background(), q); err != ctports.ErrCompetenciaFirmanteNoAcreditada || a.llamadas != 0 {
				t.Fatal("competencia diferente consultada")
			}
		})
	}
}

func TestAcreditacionConservaCopiasYNoSalePorCanalOLog(t *testing.T) {
	f, _, a, p, q := datosPrueba(t)
	x, err := f.AcreditarCompetenciaCentral(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	p.solicitud.Recurso.Atributos["expediente_ref"] = "expediente:alterado"
	a.evidencia.Asignacion.Ambitos[0].Valores[0] = "organizacion:alterada"
	a.evidencia.VersionRol.Concesiones[0].Finalidades[0] = "finalidad:alterada"
	a.solicitud.Recurso.Ambitos["organizacion_ref"] = "organizacion:alterada"
	if x.evidencia.ValidarParaEn(x.solicitud, f.reloj.Ahora()) != nil {
		t.Fatal("mapas externos alteraron acreditación")
	}
	if _, err := json.Marshal(x); !errors.Is(err, vecdomain.ErrSerializacionAsignacionCompetencialV1Prohibida) {
		t.Fatal("evidencia completa serializada")
	}
	if fmt.Sprintf("%#v", x) != "[ACREDITACION-COMPETENCIA-INTERNA]" {
		t.Fatal("evidencia completa expuesta en log")
	}
}

func TestFuenteDenegacionNominalEvitaLeerBindingYCompetencia(t *testing.T) {
	f, i, a, p, q := datosPrueba(t)
	p.errorAutorizacion = ctports.ErrCompetenciaFirmanteNoAcreditada
	if _, err := f.AcreditarCompetenciaCentral(context.Background(), q); err != ctports.ErrCompetenciaFirmanteNoAcreditada || i.llamadas != 0 || a.llamadas != 0 || p.autorizaciones != 1 || p.llamadas != 0 {
		t.Fatal("lectura sin autorización nominal")
	}
}

func TestFuenteDeniegaCrucesDeRelacionYRecursoAntesDeConsultarAsignacion(t *testing.T) {
	casos := map[string]func(*solicitudPrueba){
		"unidad":           func(p *solicitudPrueba) { p.relacion.UnidadRef = "unidad:otra" },
		"organizacion":     func(p *solicitudPrueba) { p.relacion.OrganizacionRef = "organizacion:otra" },
		"expediente":       func(p *solicitudPrueba) { p.relacion.ExpedienteRef = "expediente:otro" },
		"documento":        func(p *solicitudPrueba) { p.relacion.DocumentoRef = "documento:otro" },
		"clave documento":  func(p *solicitudPrueba) { p.relacion.DocumentoClave = "informe" },
		"prueba":           func(p *solicitudPrueba) { p.relacion.PruebaSnapshotOrigenHuellaSHA256 = strings.Repeat("8", 64) },
		"version":          func(p *solicitudPrueba) { p.relacion.VersionOrigenVinculo++ },
		"esquema":          func(p *solicitudPrueba) { p.solicitud.Recurso.Atributos["esquema_recurso"] = "esquema:ajeno" },
		"ambito añadido":   func(p *solicitudPrueba) { p.solicitud.Recurso.Ambitos["expediente_ref"] = p.relacion.ExpedienteRef },
		"atributo añadido": func(p *solicitudPrueba) { p.solicitud.Recurso.Atributos["externo"] = "otro" },
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			f, _, asignacion, proveedor, q := datosPrueba(t)
			cambiar(proveedor)
			if _, err := f.AcreditarCompetenciaCentral(context.Background(), q); err != ctports.ErrCompetenciaFirmanteNoAcreditada || asignacion.llamadas != 0 {
				t.Fatal("recurso ajeno alcanzó el lector central", err)
			}
		})
	}
}

func TestFuenteExigeDescriptorRecursoPublicadoCompleto(t *testing.T) {
	f, i, a, p, _ := datosPrueba(t)
	d := f.descriptor
	d.Recurso = DescriptorRecurso{}
	if _, err := NuevaFuente(i, a, p, f.reloj, d); err != ctports.ErrCompetenciaFirmanteNoDisponible {
		t.Fatal("descriptor ausente aceptado")
	}
	d.Recurso = f.descriptor.Recurso
	d.Recurso.AtributoDocumento = d.Recurso.AtributoExpediente
	if _, err := NuevaFuente(i, a, p, f.reloj, d); err != ctports.ErrCompetenciaFirmanteNoDisponible {
		t.Fatal("claves solapadas aceptadas")
	}
}
