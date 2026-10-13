package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type registradorPerfilBolsaPrueba struct {
	llamadas int
	datos    vecports.DatosOrdenIntentoAuditoria
}

func (r *registradorPerfilBolsaPrueba) AppendIntentoAuditoria(ctx context.Context, orden vecports.OrdenIntentoAuditoria) (vecports.AcuseIntentoAuditoria, error) {
	datos, err := orden.Datos()
	if err != nil {
		return vecports.AcuseIntentoAuditoria{}, err
	}
	r.llamadas++
	r.datos = datos
	return vecports.AcuseIntentoAuditoria{AuditoriaRef: "auditoria_perfil", Secuencia: 1, HuellaSHA256: strings.Repeat("a", 64), CorrelacionRef: datos.Datos.CorrelacionRef, RegistradaEn: time.Now().UTC()}, nil
}

func TestPerfilBolsaFuenteAusenteAuditaErrorSinEntregarDatos(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	actor, _ := escenarioLecturaRespuestasLlamamiento(t, ahora)
	perfil := actor.Resultado.Contexto.PerfilActivoRef
	d := fronteraComunPrueba("consulta-perfil-bolsa", "GET", rutaBolsasRRHHDesarrollo, false)
	d.PerfilesActivosRef = []string{perfil}
	catalogo, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{d})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{metodo: "GET", ruta: rutaBolsasRRHHDesarrollo, superficie: superficieInternaSeguridadComunDesarrollo, catalogo: catalogo, descriptor: d})
	proveedor := proveedorContextoComunConsultasRRHHDesarrolloPrueba{resolver: func(context.Context) (contextoSeguridadComunDesarrollo, error) { return actor, nil }}
	seguridad, err := nuevaSeguridadComunDesarrollo(proveedor, &relojSesionConsultaPrueba{ahora: ahora})
	if err != nil {
		t.Fatal(err)
	}
	soporte := &soporteSesionBorradorBolsaDesarrollo{soporteCanal: &soporteAltaContratacionTemporalDesarrollo{contexto: ctports.ContextoAutorizacionAltaV3{Resultado: actor.Resultado, Vinculo: actor.Vinculo}}, unidadRef: "unidad:prueba", ambitoRef: "ambito:prueba"}
	registro := &registradorPerfilBolsaPrueba{}
	m := &manejadorParticipacionBolsaDesarrollo{preparador: &preparadorBorradorLlamamientoDesarrollo{sesion: seguridad, soporte: soporte}, registradorPerfil: registro, procesoPerfil: "vec-rrhh"}
	salida, err := m.ConsultarPerfilBolsaRRHH(ctx)
	if salida != nil || !errors.Is(err, vecports.ErrFuenteAutorizacionNoDisponible) || registro.llamadas != 1 || registro.datos.Datos.Resultado != vecdomain.ResultadoIntentoAuditoriaError || registro.datos.ResultadoContexto.Contexto.PersonaRef != actor.Resultado.Contexto.PersonaRef {
		t.Fatalf("fuente ausente: datos=%t error=%v asientos=%d", salida != nil, err, registro.llamadas)
	}
}
