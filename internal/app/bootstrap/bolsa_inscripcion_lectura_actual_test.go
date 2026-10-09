package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type fuenteLecturaActualInscripcionPrueba struct {
	i                 vecdomain.InstantaneaAutorizacion
	err               error
	llamadas          int
	principal, perfil string
}

func (f *fuenteLecturaActualInscripcionPrueba) ObtenerInstantaneaAutorizacion(_ context.Context, principal, perfil string) (vecdomain.InstantaneaAutorizacion, error) {
	f.llamadas++
	f.principal, f.perfil = principal, perfil
	return f.i, f.err
}

func descriptorLecturasInscripcionPrueba() map[string]DescriptorLecturaActualInscripcion {
	resultado := make(map[string]DescriptorLecturaActualInscripcion, 7)
	for _, accion := range []string{inscripcion.AccionListarAbiertas, inscripcion.AccionDetalleAbierta,
		inscripcion.AccionListarPropias, inscripcion.AccionDetallePropia, inscripcion.AccionListarRRHH,
		inscripcion.AccionDetalleRRHH, inscripcion.AccionMotivosRRHH} {
		resultado[accion] = DescriptorLecturaActualInscripcion{Accion: accion, ModuloID: "bolsa",
			TipoRecurso: "inscripcion", Finalidad: "consulta_inscripcion_propia",
			Campos: []string{"solicitud_ref"}, AmbitoPersonaClave: "persona_ref"}
	}
	return resultado
}

func TestDecisorLecturaActualInscripcionPostgreSQLExigePoolsYDescriptores(t *testing.T) {
	if _, err := NuevoDecisorLecturaActualInscripcionPostgreSQL(nil, nil, ConfiguracionDecisorLecturaActualInscripcion{}); !errors.Is(err, inscripcion.ErrNoDisponible) {
		t.Fatalf("pools ausentes: %v", err)
	}
	f := &fuenteLecturaActualInscripcionPrueba{}
	if _, err := nuevoDecisorLecturaActualInscripcionFuentes(f, f, ConfiguracionDecisorLecturaActualInscripcion{}); !errors.Is(err, inscripcion.ErrNoDisponible) {
		t.Fatalf("descriptores ausentes: %v", err)
	}
}

func TestDecisorLecturaActualInscripcionPermisoVigenteAntesDeBolsa(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	s, _ := contextoInscripcionCanalPrueba(t, ahora, true, false)
	a := acreditacionSesionInscripcionPrueba(t, s, "externa_personal", ahora)
	v, err := s.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	filtro := inscripcion.Filtro{Limite: 20}
	referencia, err := inscripcion.RecursoLectura(inscripcion.AccionListarPropias, v.PrincipalID, "es", filtro, "")
	if err != nil {
		t.Fatal(err)
	}
	identidad := &identidadCandidatoBolsaDesarrollo{personaRef: v.PrincipalID, perfilRef: v.PerfilActivoRef, candidatoRef: "candidato_prueba_inscripcion"}
	i, err := nuevaInstantaneaMiBolsaDesarrollo(identidad, ahora)
	if err != nil {
		t.Fatal(err)
	}
	i.VersionRol.PublicadaPor = "seguridad:prueba-publicada"
	i.ControlVigenciaVersionRol.ActualizadoPor = "seguridad:prueba-publicada"
	i.AsignacionPerfil.EmitidaPor = "identidad:prueba-publicada"
	i.VersionRol.Concesiones = []vecdomain.ConcesionRol{{Accion: inscripcion.AccionListarPropias,
		ModuloID: "bolsa", TipoRecurso: "inscripcion", Finalidades: []string{"consulta_inscripcion_propia"},
		CamposPermitidos: []string{"solicitud_ref"}, GarantiaMinima: vecdomain.AuthAssuranceHigh}}
	i.AsignacionPerfil.Ambitos = []vecdomain.AmbitoPerfil{{Clave: "persona_ref", Valores: []string{v.PrincipalID}}}
	if err := i.Validar(); err != nil {
		t.Fatal(err)
	}
	fuente := &fuenteLecturaActualInscripcionPrueba{i: i}
	otraFuente := &fuenteLecturaActualInscripcionPrueba{err: errors.New("fuente equivocada")}
	d, err := nuevoDecisorLecturaActualInscripcionFuentes(fuente, otraFuente,
		ConfiguracionDecisorLecturaActualInscripcion{Reloj: relojFijoAltaContratacionTemporalDesarrollo{ahora: ahora}, Descriptores: descriptorLecturasInscripcionPrueba()})
	if err != nil {
		t.Fatal(err)
	}
	decision, err := d.DecidirLecturaActual(context.Background(), s, a, inscripcion.AccionListarPropias, referencia, filtro)
	if err != nil || !decision.Concedida || decision.CertificadoHuellaSHA256 != a.CertificadoHuellaSHA256 ||
		decision.RevisionPermisos == 0 || !revisionHuellaLecturaInscripcionCoincide(decision.RevisionPermisos, decision.HuellaInstantaneaSHA256) ||
		decision.CorrelacionRef == "" || decision.RecursoRef != referencia ||
		decision.Filtro != filtro || !decision.ValidaHasta.After(ahora) || decision.ValidaHasta.After(ahora.Add(30*time.Second)) {
		t.Fatalf("lectura actual: %+v %v", decision, err)
	}
	if fuente.llamadas != 1 || otraFuente.llamadas != 0 || fuente.principal != v.PrincipalID || fuente.perfil != v.PerfilActivoRef {
		t.Fatalf("fuente seleccionada: externa=%d interna=%d", fuente.llamadas, otraFuente.llamadas)
	}
	contenidoCambiado := i
	contenidoCambiado.VersionRol.Nombre = "Publicacion de prueba con contenido distinto"
	if contenidoCambiado.Validar() != nil {
		t.Fatal("fixture de rol cambiado invalida")
	}
	fuente.i = contenidoCambiado
	otraDecision, err := d.DecidirLecturaActual(context.Background(), s, a, inscripcion.AccionListarPropias, referencia, filtro)
	if err != nil || otraDecision.HuellaInstantaneaSHA256 == decision.HuellaInstantaneaSHA256 ||
		otraDecision.RevisionPermisos == decision.RevisionPermisos {
		t.Fatalf("contenido cambiado con mismas refs/revisiones no cambio huella: %v", err)
	}
	fuente.i = i
	// La identidad inválida corta antes incluso de obtener la instantánea.
	for nombre, cambiar := range map[string]func(*AcreditacionSesionInscripcionBolsa){
		"certificado":     func(x *AcreditacionSesionInscripcionBolsa) { x.CertificadoHuellaSHA256 = "no_canonica" },
		"sesion revocada": func(x *AcreditacionSesionInscripcionBolsa) { x.ValidaHasta = ahora },
		"perfil":          func(x *AcreditacionSesionInscripcionBolsa) { x.PerfilRef = "prf_ajeno" },
		"canal":           func(x *AcreditacionSesionInscripcionBolsa) { x.Canal = "interna_corporativa" },
	} {
		t.Run(nombre, func(t *testing.T) {
			mutada := a
			cambiar(&mutada)
			fuente.llamadas = 0
			if _, err := d.DecidirLecturaActual(context.Background(), s, mutada, inscripcion.AccionListarPropias, referencia, filtro); err == nil || fuente.llamadas != 0 {
				t.Fatalf("identidad invalida alcanzó fuente: %d %v", fuente.llamadas, err)
			}
		})
	}
	for nombre, cambiar := range map[string]func(*vecdomain.InstantaneaAutorizacion){
		"permiso ausente": func(x *vecdomain.InstantaneaAutorizacion) {
			x.VersionRol.Concesiones = []vecdomain.ConcesionRol{{Accion: "otra.accion", ModuloID: "bolsa", TipoRecurso: "inscripcion", Finalidades: []string{"consulta_inscripcion_propia"}, CamposPermitidos: []string{"solicitud_ref"}, GarantiaMinima: vecdomain.AuthAssuranceHigh}}
		},
		"ambito ajeno": func(x *vecdomain.InstantaneaAutorizacion) {
			x.AsignacionPerfil.Ambitos[0].Valores = []string{"persona_ajena"}
		},
		"campos extra": func(x *vecdomain.InstantaneaAutorizacion) {
			x.VersionRol.Concesiones[0].CamposPermitidos = []string{"solicitud_ref", "dni"}
		},
		"perfil revocado": func(x *vecdomain.InstantaneaAutorizacion) {
			x.AsignacionPerfil.Estado = vecdomain.EstadoAsignacionPerfilRevocada
			x.AsignacionPerfil.RevocadaEn = ahora
			x.AsignacionPerfil.RevocadaPor = "revocador_prueba"
			x.AsignacionPerfil.RevocacionRef = "revocacion_prueba"
		},
		"perfil cambiado": func(x *vecdomain.InstantaneaAutorizacion) { x.AsignacionPerfil.PerfilActivoRef = "prf_ajeno" },
	} {
		t.Run(nombre, func(t *testing.T) {
			mutada := i
			mutada.VersionRol.Concesiones = append([]vecdomain.ConcesionRol(nil), i.VersionRol.Concesiones...)
			mutada.AsignacionPerfil.Ambitos = append([]vecdomain.AmbitoPerfil(nil), i.AsignacionPerfil.Ambitos...)
			cambiar(&mutada)
			fuente.i, fuente.llamadas = mutada, 0
			if _, err := d.DecidirLecturaActual(context.Background(), s, a, inscripcion.AccionListarPropias, referencia, filtro); !errors.Is(err, inscripcion.ErrAccesoDenegado) || fuente.llamadas != 1 {
				t.Fatalf("concesión inválida llegó a Bolsa: fuente=%d err=%v", fuente.llamadas, err)
			}
		})
	}
}
