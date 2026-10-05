package application

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Correo y teléfono sintéticos: nunca deben aparecer en la orden AD169.
const correoContactoAuditadoPrueba = "antonio.reyes@example.org"

type operadorContactosAuditadoPrueba struct {
	pagina       ports.PaginaContactosParticipacion
	err          error
	cerrada      bool
	despues      func()
	registros    int
	evaluaciones int
}

func (o *operadorContactosAuditadoPrueba) RegistrarContactoParticipacion(context.Context, ports.SolicitudRegistrarContactoParticipacion) (ports.RegistroContactoParticipacion, error) {
	o.registros++
	return ports.RegistroContactoParticipacion{ReciboRef: "recibo:contacto:prueba"}, nil
}
func (o *operadorContactosAuditadoPrueba) ListarContactosParticipacion(context.Context, ports.ConsultaContactosParticipacion) (ports.PaginaContactosParticipacion, error) {
	return o.leer()
}
func (o *operadorContactosAuditadoPrueba) ListarContactosBolsa(context.Context, ports.ConsultaContactosBolsa) (ports.PaginaContactosParticipacion, error) {
	return o.leer()
}
func (o *operadorContactosAuditadoPrueba) leer() (ports.PaginaContactosParticipacion, error) {
	o.cerrada = true
	if o.despues != nil {
		o.despues()
	}
	return o.pagina, o.err
}

type operadorContactosConIntentosPrueba struct {
	*operadorContactosAuditadoPrueba
}

func (o operadorContactosConIntentosPrueba) EstadoIntentosTelefonicos(_ context.Context, ll string, _ []dominiobolsa.ContactoParticipacion, _ bool) (ports.EstadoIntentosContacto, error) {
	o.evaluaciones++
	return ports.EstadoIntentosContacto{LlamamientoRef: ll, Configurada: true}, nil
}

type registradorContactosAuditadoPrueba struct {
	t             *testing.T
	operador      *operadorContactosAuditadoPrueba
	ordenes       []vecports.DatosOrdenIntentoAuditoria
	err           error
	acuseInvalido bool
}

func (r *registradorContactosAuditadoPrueba) AppendIntentoAuditoria(ctx context.Context, o vecports.OrdenIntentoAuditoria) (vecports.AcuseIntentoAuditoria, error) {
	r.t.Helper()
	if !r.operador.cerrada || ctx.Err() != nil {
		r.t.Fatal("registro antes del cierre o cancelado por HTTP")
	}
	d, err := o.Datos()
	if err != nil {
		r.t.Fatal(err)
	}
	r.ordenes = append(r.ordenes, d)
	if r.err != nil || r.acuseInvalido {
		return vecports.AcuseIntentoAuditoria{}, r.err
	}
	return vecports.AcuseIntentoAuditoria{AuditoriaRef: "aud_contactos_sintetica", Secuencia: 1, HuellaSHA256: strings.Repeat("a", 64),
		CorrelacionRef: d.Datos.CorrelacionRef, RegistradaEn: time.Date(2026, 10, 5, 1, 0, 0, 0, time.UTC)}, nil
}

func contactoAuditadoPrueba(t *testing.T, bolsa, participacion string) dominiobolsa.ContactoParticipacion {
	t.Helper()
	c := dominiobolsa.ContactoParticipacion{ContactoRef: "contacto:" + strings.Repeat("c", 64), BolsaRef: bolsa, ParticipacionRef: participacion,
		Canal: dominiobolsa.CanalContactoTelefono, Instante: time.Date(2026, 10, 5, 8, 0, 0, 0, time.UTC),
		Actor: "per_0123456789abcdefghijkl", Resultado: dominiobolsa.ResultadoContactoContactado, Anotacion: "Acepta la oferta"}
	if c.Validar() != nil {
		t.Fatal("contacto sintético inválido")
	}
	return c
}

func consultaContactosAuditadaPrueba(t *testing.T) ports.ConsultaContactosParticipacion {
	q := solicitudSituacionPrueba(t, time.Date(2026, 10, 5, 1, 0, 0, 0, time.UTC))
	return ports.ConsultaContactosParticipacion{Vinculo: q.Vinculo, ResultadoContexto: q.ResultadoContexto, BolsaRef: q.BolsaRef,
		ParticipacionRef: q.ParticipacionRef, Limite: 20, Correlacion: q.Correlacion, MotivoAutorizacion: q.MotivoAutorizacion}
}

func consultaContactosBolsaAuditadaPrueba(t *testing.T) ports.ConsultaContactosBolsa {
	q := consultaContactosAuditadaPrueba(t)
	return ports.ConsultaContactosBolsa{Vinculo: q.Vinculo, ResultadoContexto: q.ResultadoContexto, BolsaRef: q.BolsaRef,
		Limite: 20, Correlacion: q.Correlacion, MotivoAutorizacion: q.MotivoAutorizacion}
}

func TestContactosAuditadosConservanPositivoSinRegistro(t *testing.T) {
	q := consultaContactosAuditadaPrueba(t)
	contacto := contactoAuditadoPrueba(t, q.BolsaRef, q.ParticipacionRef)
	for _, pagina := range []ports.PaginaContactosParticipacion{{}, {Contactos: []dominiobolsa.ContactoParticipacion{contacto}, CursorSiguiente: contacto.ContactoRef}} {
		o := &operadorContactosAuditadoPrueba{pagina: pagina}
		r := &registradorContactosAuditadoPrueba{t: t, operador: o}
		s, err := NuevosContactosRRHHAuditados(o, r, "vec-bolsa-prueba")
		if err != nil {
			t.Fatal(err)
		}
		porParticipacion, err := s.ListarContactosParticipacion(context.Background(), q)
		if err != nil || !reflect.DeepEqual(porParticipacion, pagina) {
			t.Fatal("positivo por participación alterado")
		}
		porBolsa, err := s.ListarContactosBolsa(context.Background(), consultaContactosBolsaAuditadaPrueba(t))
		if err != nil || !reflect.DeepEqual(porBolsa, pagina) || len(r.ordenes) != 0 {
			t.Fatal("positivo por bolsa alterado o registrado")
		}
	}
}

func TestContactosAuditadosRegistranDenegacionYError(t *testing.T) {
	for _, caso := range []struct {
		causa     error
		resultado core.ResultadoIntentoAuditoria
	}{
		{core.ErrAutorizacionDenegada, core.ResultadoIntentoAuditoriaDenegado},
		{ErrCambioSituacionParticipacionNoDisponible, core.ResultadoIntentoAuditoriaError},
		{ports.ErrContactoParticipacionNoDisponible, core.ResultadoIntentoAuditoriaError},
		{errors.Join(core.ErrAutorizacionDenegada, ports.ErrContactoParticipacionNoDisponible), core.ResultadoIntentoAuditoriaError},
		{ports.ErrContactoParticipacionNoEncontrado, core.ResultadoIntentoAuditoriaError},
	} {
		for _, porBolsa := range []bool{false, true} {
			q := consultaContactosAuditadaPrueba(t)
			o := &operadorContactosAuditadoPrueba{err: caso.causa}
			r := &registradorContactosAuditadoPrueba{t: t, operador: o}
			s, _ := NuevosContactosRRHHAuditados(o, r, "vec-bolsa-prueba")
			var pagina ports.PaginaContactosParticipacion
			var err error
			recurso := q.ParticipacionRef
			if porBolsa {
				pagina, err = s.ListarContactosBolsa(context.Background(), consultaContactosBolsaAuditadaPrueba(t))
				recurso = q.BolsaRef
			} else {
				pagina, err = s.ListarContactosParticipacion(context.Background(), q)
			}
			if pagina.Contactos != nil || !errors.Is(err, caso.causa) || len(r.ordenes) != 1 {
				t.Fatal("fallo sin auditoría o con datos")
			}
			d := r.ordenes[0]
			correlacion, _ := q.Correlacion.ValorCanonico()
			if d.Datos.Accion != ports.AccionConsultarContactoParticipacion || d.Datos.ModuloID != ports.ModuloSituacionParticipacion ||
				d.Datos.RecursoRef != recurso || d.Datos.FinalidadRef != ports.FinalidadConsultarContactoParticipacion ||
				d.Datos.Motivo != q.MotivoAutorizacion || d.Datos.Resultado != caso.resultado || d.Datos.CorrelacionRef != correlacion ||
				d.Datos.Canal != "interna_corporativa" || !reflect.DeepEqual(d.ResultadoContexto, q.ResultadoContexto) {
				t.Fatalf("sobre nominal sustituido: %+v", d.Datos)
			}
			if acuse, ok := AcuseConsultaContactosFallida(err); !ok || acuse.AuditoriaRef != "aud_contactos_sintetica" {
				t.Fatal("acuse confirmado perdido")
			}
		}
	}
}

func TestContactosAuditadosNoPublicanPaginaParcialOIncoherente(t *testing.T) {
	q := consultaContactosAuditadaPrueba(t)
	propio := contactoAuditadoPrueba(t, q.BolsaRef, q.ParticipacionRef)
	ajeno := contactoAuditadoPrueba(t, q.BolsaRef, "participacion:ajena")
	conCorreo := propio
	conCorreo.Anotacion = "Escribir a " + correoContactoAuditadoPrueba
	for _, caso := range []struct {
		pagina ports.PaginaContactosParticipacion
		err    error
	}{
		{ports.PaginaContactosParticipacion{Contactos: []dominiobolsa.ContactoParticipacion{propio}, CursorSiguiente: propio.ContactoRef}, core.ErrAutorizacionDenegada},
		{ports.PaginaContactosParticipacion{Contactos: []dominiobolsa.ContactoParticipacion{ajeno}, CursorSiguiente: ajeno.ContactoRef}, nil},
		{ports.PaginaContactosParticipacion{Contactos: []dominiobolsa.ContactoParticipacion{conCorreo}, CursorSiguiente: conCorreo.ContactoRef}, nil},
		{ports.PaginaContactosParticipacion{Contactos: []dominiobolsa.ContactoParticipacion{propio}}, nil},
		{ports.PaginaContactosParticipacion{CursorSiguiente: propio.ContactoRef}, nil},
	} {
		o := &operadorContactosAuditadoPrueba{pagina: caso.pagina, err: caso.err}
		r := &registradorContactosAuditadoPrueba{t: t, operador: o}
		s, _ := NuevosContactosRRHHAuditados(o, r, "vec-bolsa-prueba")
		pagina, err := s.ListarContactosParticipacion(context.Background(), q)
		if pagina.Contactos != nil || !errors.Is(err, ports.ErrContactoParticipacionNoDisponible) || len(r.ordenes) != 1 ||
			r.ordenes[0].Datos.Resultado != core.ResultadoIntentoAuditoriaError {
			t.Fatal("página no confiable publicada")
		}
		if strings.Contains(r.ordenes[0].Datos.RecursoRef, "@") || strings.Contains(err.Error(), correoContactoAuditadoPrueba) {
			t.Fatal("dato de contacto en la auditoría o en el error")
		}
	}
}

func TestContactosAuditadosCierranAnteRegistroFallidoOAcuseInvalido(t *testing.T) {
	q := consultaContactosAuditadaPrueba(t)
	for _, invalido := range []bool{false, true} {
		o := &operadorContactosAuditadoPrueba{err: core.ErrAutorizacionDenegada}
		r := &registradorContactosAuditadoPrueba{t: t, operador: o, acuseInvalido: invalido}
		if !invalido {
			r.err = errors.New("material privado del controlador")
		}
		s, _ := NuevosContactosRRHHAuditados(o, r, "vec-bolsa-prueba")
		pagina, err := s.ListarContactosParticipacion(context.Background(), q)
		if pagina.Contactos != nil || !errors.Is(err, ports.ErrContactoParticipacionNoDisponible) || !errors.Is(err, vecports.ErrIntentoAuditoriaNoDisponible) ||
			errors.Is(err, core.ErrAutorizacionDenegada) || strings.Contains(err.Error(), "privado") {
			t.Fatal("fallo del registrador con datos o causa cruda")
		}
		if _, ok := AcuseConsultaContactosFallida(err); ok {
			t.Fatal("acuse falso")
		}
	}
}

func TestContactosAuditadosConservanActorCapturadoTrasCancelacion(t *testing.T) {
	q := consultaContactosAuditadaPrueba(t)
	historico, err := q.ResultadoContexto.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	o := &operadorContactosAuditadoPrueba{despues: func() { cancelar(); q.ResultadoContexto.RepresentacionCanonica[0] = '!' }}
	r := &registradorContactosAuditadoPrueba{t: t, operador: o}
	s, _ := NuevosContactosRRHHAuditados(o, r, "vec-bolsa-prueba")
	pagina, err := s.ListarContactosParticipacion(ctx, q)
	if pagina.Contactos != nil || !errors.Is(err, context.Canceled) || len(r.ordenes) != 1 ||
		r.ordenes[0].Datos.Resultado != core.ResultadoIntentoAuditoriaError || !reflect.DeepEqual(r.ordenes[0].ResultadoContexto, historico) {
		t.Fatal("identidad histórica sustituida o cancelación no registrada")
	}
}

func TestContactosAuditadosNoInventanContextoYDeleganEscrituraEIntentos(t *testing.T) {
	o := &operadorContactosAuditadoPrueba{err: core.ErrAutorizacionDenegada}
	r := &registradorContactosAuditadoPrueba{t: t, operador: o}
	s, _ := NuevosContactosRRHHAuditados(o, r, "vec-bolsa-prueba")
	if _, err := s.ListarContactosParticipacion(context.Background(), ports.ConsultaContactosParticipacion{BolsaRef: "bolsa:b2", ParticipacionRef: "participacion:b2", Limite: 20}); err == nil || o.cerrada || len(r.ordenes) != 0 {
		t.Fatal("consulta o auditoría con actor inventado")
	}
	if _, err := s.ListarContactosBolsa(context.Background(), ports.ConsultaContactosBolsa{BolsaRef: "bolsa:b2", Limite: 20}); err == nil || o.cerrada || len(r.ordenes) != 0 {
		t.Fatal("consulta de bolsa con actor inventado")
	}
	if registro, err := s.RegistrarContactoParticipacion(context.Background(), ports.SolicitudRegistrarContactoParticipacion{}); err != nil || registro.ReciboRef != "recibo:contacto:prueba" || o.registros != 1 || len(r.ordenes) != 0 {
		t.Fatal("escritura alterada por el decorador")
	}
	sinEvaluador, err := s.EstadoIntentosTelefonicos(context.Background(), "llamamiento:1", nil, true)
	if err != nil || sinEvaluador.Configurada || sinEvaluador.LlamamientoRef != "llamamiento:1" {
		t.Fatal("estado de intentos inventado")
	}
	conIntentos, _ := NuevosContactosRRHHAuditados(operadorContactosConIntentosPrueba{o}, r, "vec-bolsa-prueba")
	if e, err := conIntentos.EstadoIntentosTelefonicos(context.Background(), "llamamiento:1", nil, true); err != nil || !e.Configurada || o.evaluaciones != 1 {
		t.Fatal("evaluador de intentos ocultado")
	}
	if _, err := NuevosContactosRRHHAuditados(o, (*registradorContactosAuditadoPrueba)(nil), "vec-bolsa-prueba"); err == nil {
		t.Fatal("registrador nil aceptado")
	}
	if _, err := NuevosContactosRRHHAuditados(nil, r, "vec-bolsa-prueba"); err == nil {
		t.Fatal("operador nil aceptado")
	}
}

func TestServicioContactosDistingueDenegacionExplicitaDeFalloV3(t *testing.T) {
	for _, explicita := range []bool{false, true} {
		causa := errors.New("emision material: fallo privado sintetico")
		if explicita {
			causa = errors.Join(causa, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3)
		}
		repo := &repositorioContactoPrueba{}
		servicio, err := NuevoServicioContactoParticipacion(contextoContactoPrueba{}, &autorizadorBorradorPrueba{t: t, err: causa}, repo)
		if err != nil {
			t.Fatal(err)
		}
		q := consultaContactosAuditadaPrueba(t)
		_, errParticipacion := servicio.ListarContactosParticipacion(context.Background(), q)
		_, errBolsa := servicio.ListarContactosBolsa(context.Background(), consultaContactosBolsaAuditadaPrueba(t))
		for _, err := range []error{errParticipacion, errBolsa} {
			if explicita != errors.Is(err, core.ErrAutorizacionDenegada) || errors.Is(err, ErrCambioSituacionParticipacionNoDisponible) == explicita ||
				strings.Contains(err.Error(), "privado") {
				t.Fatalf("denegación y fallo técnico confundidos: %v", err)
			}
		}
		if repo.lecturasParticipacion != 0 || repo.lecturasBolsa != 0 {
			t.Fatal("lectura sin autorización")
		}
	}
}
