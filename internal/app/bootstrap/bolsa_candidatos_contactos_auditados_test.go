package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type operadorCandidatosContactosAuditadosPrueba struct {
	pagina   puertosbolsa.PaginaContactosParticipacion
	err      error
	consulta puertosbolsa.ConsultaContactosBolsa
	llamadas int
}

func (o *operadorCandidatosContactosAuditadosPrueba) RegistrarContactoParticipacion(context.Context, puertosbolsa.SolicitudRegistrarContactoParticipacion) (puertosbolsa.RegistroContactoParticipacion, error) {
	return puertosbolsa.RegistroContactoParticipacion{}, nil
}

func (o *operadorCandidatosContactosAuditadosPrueba) ListarContactosParticipacion(context.Context, puertosbolsa.ConsultaContactosParticipacion) (puertosbolsa.PaginaContactosParticipacion, error) {
	return puertosbolsa.PaginaContactosParticipacion{}, nil
}

func (o *operadorCandidatosContactosAuditadosPrueba) ListarContactosBolsa(_ context.Context, q puertosbolsa.ConsultaContactosBolsa) (puertosbolsa.PaginaContactosParticipacion, error) {
	o.llamadas++
	o.consulta = q
	return o.pagina, o.err
}

type registradorCandidatosContactosAuditadosPrueba struct {
	registradorPerfilBolsaPrueba
	err error
}

func (r *registradorCandidatosContactosAuditadosPrueba) AppendIntentoAuditoria(ctx context.Context, orden puertosvec.OrdenIntentoAuditoria) (puertosvec.AcuseIntentoAuditoria, error) {
	if r.err != nil {
		r.llamadas++
		return puertosvec.AcuseIntentoAuditoria{}, r.err
	}
	return r.registradorPerfilBolsaPrueba.AppendIntentoAuditoria(ctx, orden)
}

func escenarioCandidatosContactosAuditadosPrueba(t *testing.T, operador *operadorCandidatosContactosAuditadosPrueba, registro *registradorCandidatosContactosAuditadosPrueba) (*manejadorParticipacionBolsaDesarrollo, context.Context, contextoSeguridadComunDesarrollo, string) {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	actor, _ := escenarioLecturaRespuestasLlamamiento(t, ahora)
	bolsa := "bolsa:prueba"
	ruta := bolsahttp.RutaBolsasGestion + "/" + bolsa + "/candidatos"
	d := fronteraComunPrueba(claveFronteraConsultarContactosB5Bolsa, http.MethodGet, bolsahttp.RutaBolsasGestion, false)
	d.PerfilesActivosRef = []string{actor.Resultado.Contexto.PerfilActivoRef}
	d.PlantillaDetalle = []string{"*", "candidatos"}
	catalogo, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{d})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := puertosvec.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ctx = context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{metodo: http.MethodGet, ruta: ruta, superficie: superficieInternaSeguridadComunDesarrollo, catalogo: catalogo, descriptor: d})
	ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidadConsultaContratacionTemporalDesarrollo{ruta: ruta})
	seguridad, err := nuevaSeguridadComunDesarrollo(proveedorContextoComunConsultasRRHHDesarrolloPrueba{resolver: func(context.Context) (contextoSeguridadComunDesarrollo, error) { return actor, nil }}, &relojSesionConsultaPrueba{ahora: ahora})
	if err != nil {
		t.Fatal(err)
	}
	soporte := &soporteSesionBorradorBolsaDesarrollo{soporteCanal: &soporteAltaContratacionTemporalDesarrollo{contexto: ctports.ContextoAutorizacionAltaV3{Resultado: actor.Resultado, Vinculo: actor.Vinculo}}, unidadRef: "unidad:prueba", ambitoRef: "ambito:prueba"}
	auditados, err := aplicacionbolsa.NuevosContactosRRHHAuditados(operador, registro, "vec-bolsa-prueba")
	if err != nil {
		t.Fatal(err)
	}
	return &manejadorParticipacionBolsaDesarrollo{preparador: &preparadorBorradorLlamamientoDesarrollo{sesion: seguridad, soporte: soporte, generar: seguridadvec.GeneradorReferenciasCriptograficas{}}, contactosAuditados: auditados}, ctx, actor, bolsa
}

func TestCandidatosContactosUsaServicioAuditadoConContextoNominal(t *testing.T) {
	for _, caso := range []struct {
		nombre    string
		err       error
		resultado dominiovec.ResultadoIntentoAuditoria
	}{
		{nombre: "denegacion", err: dominiovec.ErrAutorizacionDenegada, resultado: dominiovec.ResultadoIntentoAuditoriaDenegado},
		{nombre: "error_operador", err: puertosbolsa.ErrContactoParticipacionNoDisponible, resultado: dominiovec.ResultadoIntentoAuditoriaError},
		{nombre: "positivo_vacio"},
		{nombre: "error_auditoria", err: dominiovec.ErrAutorizacionDenegada, resultado: dominiovec.ResultadoIntentoAuditoriaDenegado},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			operador := &operadorCandidatosContactosAuditadosPrueba{err: caso.err}
			registro := &registradorCandidatosContactosAuditadosPrueba{}
			if caso.nombre == "error_auditoria" {
				registro.err = errors.New("registro sintético caído")
			}
			m, ctx, actor, bolsa := escenarioCandidatosContactosAuditadosPrueba(t, operador, registro)
			pagina, err := m.ListarContactosBolsa(ctx, bolsa, "", 20)
			if operador.llamadas != 1 || !reflect.DeepEqual(pagina, puertosbolsa.PaginaContactosParticipacion{}) {
				t.Fatalf("consulta=%d página=%+v", operador.llamadas, pagina)
			}
			q := operador.consulta
			correlacion, errCorrelacion := q.Correlacion.ValorCanonico()
			if q.BolsaRef != bolsa || q.Limite != 20 || q.MotivoAutorizacion != motivoConsultarContactoParticipacionBolsaDesarrollo() || errCorrelacion != nil || !reflect.DeepEqual(q.ResultadoContexto, actor.Resultado) {
				t.Fatal("el preparador perdió el contexto nominal de la petición")
			}
			if caso.err == nil {
				if err != nil || registro.llamadas != 0 {
					t.Fatalf("lectura vacía: error=%v intentos=%d", err, registro.llamadas)
				}
				return
			}
			if registro.llamadas != 1 {
				t.Fatalf("intentos=%d, se esperaba uno", registro.llamadas)
			}
			if caso.nombre == "error_auditoria" {
				if !errors.Is(err, puertosbolsa.ErrContactoParticipacionNoDisponible) || !errors.Is(err, puertosvec.ErrIntentoAuditoriaNoDisponible) || errors.Is(err, caso.err) {
					t.Fatalf("fallo de auditoría no cierra la lectura: %v", err)
				}
				return
			}
			if !errors.Is(err, caso.err) {
				t.Fatalf("causa original perdida: %v", err)
			}
			d := registro.datos
			if d.Datos.Accion != puertosbolsa.AccionConsultarContactoParticipacion || d.Datos.RecursoRef != bolsa || d.Datos.Motivo != q.MotivoAutorizacion || d.Datos.CorrelacionRef != correlacion || d.Datos.Resultado != caso.resultado || !reflect.DeepEqual(d.ResultadoContexto, actor.Resultado) {
				t.Fatalf("intento sin actor, bolsa, motivo o correlación: %+v", d.Datos)
			}
		})
	}
}
