package application

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
)

func TestDeclaracionYRecuperacionExigenAutoridadCadaVez(t *testing.T) {
	s, solicitud, auth, registro, audit := escenario(t)
	primero, err := s.Declarar(context.Background(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := s.Declarar(context.Background(), solicitud)
	if err != nil || !reflect.DeepEqual(primero, segundo) || auth.llamadas != 2 || registro.confirmaciones != 1 || registro.recuperaciones != 1 || audit.llamadas != 0 {
		t.Fatal("reintento cambió historia o evitó autoridad", err)
	}
	auth.err = vec.ErrAutorizacionDenegada
	if _, err := s.Declarar(context.Background(), solicitud); !errors.Is(err, vec.ErrAutorizacionDenegada) || registro.recuperaciones != 1 || audit.llamadas != 1 {
		t.Fatal("replay sin autoridad vigente", err)
	}
}

func TestVerificarNoAcreditaNiConsultaRegistro(t *testing.T) {
	s, solicitud, auth, registro, audit := escenario(t)
	if _, err := s.Verificar(context.Background(), solicitud); !errors.Is(err, ErrAcreditacionPendiente) || auth.llamadas != 1 || registro.lecturas != 0 || registro.confirmaciones != 0 || audit.llamadas != 1 {
		t.Fatal("verificación produjo efecto o perdió pendiente", err)
	}
}

func TestSolicitudCruzaVinculoAntesDeAutoridad(t *testing.T) {
	s, solicitud, auth, registro, audit := escenario(t)
	solicitud.Vinculo = vec.VinculoAutenticacionActorV2{}
	if _, err := s.Declarar(context.Background(), solicitud); !errors.Is(err, vec.ErrAutorizacionDenegada) || auth.llamadas != 0 || registro.lecturas != 0 || audit.llamadas != 0 {
		t.Fatal(err)
	}
}

func TestComandoInvalidoConContextoValidoAuditaSoloMetadatosSeguros(t *testing.T) {
	for _, caso := range []string{"version", "contenido", "motivo"} {
		t.Run(caso, func(t *testing.T) {
			s, solicitud, auth, registro, audit := escenario(t)
			switch caso {
			case "version":
				solicitud.Hecho.Version = 2
			case "contenido":
				solicitud.Hecho.Referencia = "dato privado inválido"
			case "motivo":
				solicitud.Motivo.EntradaClave = "motivo privado inválido"
			}
			if r, err := s.Declarar(context.Background(), solicitud); !errors.Is(err, ErrSolicitud) || r.Referencia != "" ||
				audit.llamadas != 1 || auth.llamadas != 0 || registro.lecturas != 0 || registro.confirmaciones != 0 {
				t.Fatal("rechazo de negocio sin auditoría o con efecto", err)
			}
			correlacion, _ := solicitud.Correlacion.ValorCanonico()
			if audit.ultima.ActorID != persona || audit.ultima.CorrelationRef != correlacion || audit.ultima.Action != accionDeclarar ||
				audit.ultima.Result != "no_confirmado" || audit.ultima.SubjectRef != "" || audit.ultima.RuleRef != "" ||
				audit.ultima.ObjectVersion != 0 || len(audit.ultima.Metadata) != 0 {
				t.Fatal("auditoría perdió contexto nominal o incorporó campos rechazados")
			}
		})
	}
}

func TestRecibosConfirmadosYRecuperadosNoCompartenMemoriaConRegistro(t *testing.T) {
	for _, operacion := range []string{"declarar", "rechazar"} {
		t.Run(operacion, func(t *testing.T) {
			s, solicitud, _, registro, _ := escenario(t)
			horas := 20
			solicitud.Hecho.Horas = &horas
			ejecutar := s.Declarar
			if operacion == "rechazar" {
				solicitud.Hecho.PersonaRef = "per_bbbbbbbbbbbbbbbbbbbbbb"
				actual := ports.RegistroActual{Hecho: copiarHecho(solicitud.Hecho), DeclaranteRef: solicitud.Hecho.PersonaRef}
				registro.actual = &actual
				solicitud.VersionEsperada, solicitud.Hecho.Version = 1, 2
				ejecutar = s.Rechazar
			}
			primero, err := ejecutar(context.Background(), solicitud)
			if err != nil {
				t.Fatal(err)
			}
			original := copiarRecibo(*registro.recibo)
			mutar := func(r *ports.Recibo) {
				r.Registro.Hecho.Evidencias[0].Version = 99
				*r.Registro.Hecho.Horas = 99
				if r.Registro.Hecho.Revision != nil {
					r.Registro.Hecho.Revision.ActorRef = "persona:ajena"
				}
			}
			mutar(&primero)
			if !reflect.DeepEqual(*registro.recibo, original) {
				t.Fatal("mutar resultado confirmado altera registro")
			}
			segundo, err := ejecutar(context.Background(), solicitud)
			if err != nil || !reflect.DeepEqual(segundo, original) {
				t.Fatal("recuperación perdió recibo original", err)
			}
			mutar(&segundo)
			if !reflect.DeepEqual(*registro.recibo, original) {
				t.Fatal("mutar resultado recuperado altera registro")
			}
			if tercero, err := ejecutar(context.Background(), solicitud); err != nil || !reflect.DeepEqual(tercero, original) || registro.confirmaciones != 1 || registro.recuperaciones != 2 {
				t.Fatal("mutación del resultado impide reintento o duplica efecto", err)
			}
		})
	}
}

func TestMaterialFueraDeAudienciaYCamposFallaCerrado(t *testing.T) {
	for _, caso := range []string{"audiencia", "campos", "caducada"} {
		t.Run(caso, func(t *testing.T) {
			s, solicitud, auth, registro, audit := escenario(t)
			auth.defecto = caso
			if _, err := s.Declarar(context.Background(), solicitud); err == nil || registro.lecturas != 0 || registro.confirmaciones != 0 || audit.llamadas != 1 {
				t.Fatal("material ajeno alcanzó registro", err)
			}
		})
	}
}

func TestResultadoInciertoYReciboAlteradoNoConfirman(t *testing.T) {
	for _, caso := range []string{"commit", "recibo", "audit"} {
		t.Run(caso, func(t *testing.T) {
			s, solicitud, auth, registro, audit := escenario(t)
			switch caso {
			case "commit":
				registro.err = ports.ErrRegistroNoDisponible
			case "recibo":
				registro.mutar = true
			case "audit":
				auth.err = vec.ErrAutorizacionDenegada
				audit.err = errors.New("test")
			}
			if r, err := s.Declarar(context.Background(), solicitud); err == nil || r.Referencia != "" {
				t.Fatal("resultado no confirmado presentado como éxito", err)
			}
		})
	}
}

func TestRectificacionNoConservaAcreditacionYRevisionSeparaPersonas(t *testing.T) {
	_, solicitud, _, _, _ := escenario(t)
	actual := ports.RegistroActual{Hecho: copiarHecho(solicitud.Hecho), DeclaranteRef: solicitud.Hecho.PersonaRef}
	actual.Hecho.Estado = domain.Acreditado
	actual.Hecho.Revision = &domain.Revision{Referencia: "revision:previa", ActorRef: "persona:revisor", MotivoRef: "motivo:previo", Fecha: instante.Format(time.RFC3339)}
	solicitud.VersionEsperada, solicitud.Hecho.Version = 1, 2
	solicitud.Hecho.Denominacion = "Curso rectificado"
	o, _ := ordenSolicitud(solicitud, accionRectificar)
	cambio, err := prepararCambio(o, &actual, instante)
	if err != nil || cambio.Nuevo.Hecho.Estado != domain.Pendiente || cambio.Nuevo.Hecho.Revision != nil || actual.Hecho.Estado != domain.Acreditado {
		t.Fatal("rectificación acreditada o historia alterada", err)
	}
	actual.Hecho.Estado, actual.Hecho.Revision = domain.Declarado, nil
	solicitud.Hecho = copiarHecho(actual.Hecho)
	solicitud.Hecho.Version = 2
	o, _ = ordenSolicitud(solicitud, accionRechazar)
	if _, err := prepararCambio(o, &actual, instante); !errors.Is(err, vec.ErrAutorizacionDenegada) {
		t.Fatal("autorrevisión aceptada", err)
	}
	o.ActorRef = "persona:revisor"
	cambio, err = prepararCambio(o, &actual, instante)
	if err != nil || cambio.Nuevo.Hecho.Estado != domain.Rechazado || cambio.Nuevo.DeclaranteRef != actual.DeclaranteRef {
		t.Fatal("rechazo pierde declarante", err)
	}
	o.Hecho.PersonaRef = "persona:ajena"
	if _, err := prepararCambio(o, &actual, instante); err == nil {
		t.Fatal("rectifica identidad")
	}
}

func TestHuellaLigaMotivoVersionFuenteCorteYContenido(t *testing.T) {
	_, solicitud, _, _, _ := escenario(t)
	original, _ := ordenSolicitud(solicitud, accionDeclarar)
	for _, mutar := range []func(*Solicitud){
		func(s *Solicitud) { s.Motivo.CatalogoVersion++ },
		func(s *Solicitud) { s.Hecho.Procedencia.Version = "2" },
		func(s *Solicitud) { s.FechaCorte = "2026-10-02" },
		func(s *Solicitud) { s.Hecho.Denominacion = "Contenido diferente" },
	} {
		copia := solicitud
		copia.Hecho = copiarHecho(copia.Hecho)
		mutar(&copia)
		modificado, _ := ordenSolicitud(copia, accionDeclarar)
		if modificado.HuellaComando == original.HuellaComando {
			t.Fatal("huella no liga significado")
		}
	}
}

func TestConstructorRechazaDependenciasAusentesOTipadasNulas(t *testing.T) {
	var auth *autorizadorPrueba
	if s, err := NuevoServicio(auth, nil, nil, nil); err == nil || s != nil {
		t.Fatal("constructor permisivo")
	}
}

func TestDenegacionConfirmadaSQLNoDuplicaAuditoria(t *testing.T) {
	for _, codigo := range []string{"conflicto_version", "clave_reutilizada", "denegada"} {
		t.Run(codigo, func(t *testing.T) {
			s, solicitud, _, registro, audit := escenario(t)
			registro.resultado = &ports.ResultadoOperacion{Codigo: codigo, AuditoriaRef: "auditoria:denegacion"}
			if r, err := s.Declarar(context.Background(), solicitud); !errors.Is(err, errorResultadoOperacion(codigo)) || r.Referencia != "" || audit.llamadas != 0 {
				t.Fatal("denegación confirmada pierde error o duplica auditoría", err)
			}
		})
	}
}

func TestSobreDenegadoConDatosONoConfirmadoFallaCerrado(t *testing.T) {
	for _, caso := range []string{"auditoria_ausente", "datos_en_denegacion", "codigo_ajeno"} {
		t.Run(caso, func(t *testing.T) {
			s, solicitud, _, registro, audit := escenario(t)
			resultado := ports.ResultadoOperacion{Codigo: "denegada", AuditoriaRef: "auditoria:denegacion"}
			switch caso {
			case "auditoria_ausente":
				resultado.AuditoriaRef = ""
			case "datos_en_denegacion":
				resultado.Anterior = &ports.RegistroActual{Hecho: solicitud.Hecho, DeclaranteRef: persona}
			case "codigo_ajeno":
				resultado.Codigo = "otro"
			}
			registro.resultado = &resultado
			if r, err := s.Declarar(context.Background(), solicitud); !errors.Is(err, ports.ErrRegistroNoDisponible) || r.Referencia != "" || audit.llamadas != 1 {
				t.Fatal("sobre inválido aceptado como denegación acreditada", err)
			}
		})
	}
}

type relojSecuencia struct{ llamadas int }

func (r *relojSecuencia) Ahora() time.Time {
	r.llamadas++
	if r.llamadas == 1 {
		return instante
	}
	return instante.Add(2 * time.Second)
}

func TestVentanaCompruebaInstantePosteriorAEmision(t *testing.T) {
	s, solicitud, _, registro, _ := escenario(t)
	reloj := &relojSecuencia{}
	s.reloj = reloj
	if _, err := s.Declarar(context.Background(), solicitud); err != nil || registro.confirmaciones != 1 || reloj.llamadas != 2 {
		t.Fatal("material recién emitido rechazado por instante previo", err)
	}
}
