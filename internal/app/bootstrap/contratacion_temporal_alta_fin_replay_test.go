package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestModalidadRetiradaSinConfirmacionSeDeniegaEnFlujoYMotivo(t *testing.T) {
	soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	soporte.opcionesCatalogo = &opcionesAnalisisCTDesarrollo{}
	ctx := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, httpinterno.RutaAltaSolicitudes)
	flujo := ports.SolicitudResolverFlujo{
		OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo,
		CentroRef:       centroAltaContratacionTemporalDesarrollo,
		CategoriaRef:    categoriaAltaContratacionTemporalDesarrollo,
		MotivoClave:     "modalidad.retirada.sintetica",
		Instante:        soporte.reloj.Ahora(),
	}
	if _, err := soporte.ResolverFlujoAlta(ctx, flujo); !errors.Is(err, ports.ErrFlujoNoDisponible) {
		t.Fatalf("flujo admitió modalidad retirada sin CT167: %v", err)
	}
	motivo := ports.SolicitudResolverMotivoAutorizacionAltaV3{
		OrganizacionRef: flujo.OrganizacionRef,
		Flujo:           soporte.flujo.Flujo,
		MotivoClave:     flujo.MotivoClave,
		Instante:        flujo.Instante,
	}
	if _, err := soporte.ResolverMotivoAutorizacionAltaV3(ctx, motivo); !errors.Is(err, ports.ErrMotivoAutorizacionNoDisponible) {
		t.Fatalf("motivo admitió modalidad retirada sin CT167: %v", err)
	}
}

func TestPeticionRatificadaPendienteConservaPoliticaTrasRetirarModalidad(t *testing.T) {
	soporte, _ := escenarioPerfilesFijosPrueba(t)
	soporte.flujo = ports.ConfiguracionAltaFlujo{
		Flujo: domain.ReferenciaFlujo{
			DefinicionRef: "flujo:ct:desarrollo", Version: 1,
			HuellaSHA256: huellaAltaContratacionTemporalDesarrollo("flujo"),
		},
		FaseInicial: "solicitud", UnidadInicialRef: "unidad:desarrollo:rrhh", AccionInicial: "alta",
	}
	_, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	principal.ID = soporte.principalID
	principal.Attributes["certificate_sha256"] = soporte.certificadoSHA256
	fijo := soporte.perfilFijoParaRutaYMetodo(rutaEntregaPeticionCentro, "POST")
	if fijo == nil {
		t.Fatal("perfil de entrega ausente")
	}
	fijo.contextoEsperadoRegistrado = fijo.contexto.Resultado
	fijo.sesionOperativa = proveedorSesionOperativaCTPrueba{contexto: fijo.contexto}
	vinculo, err := fijo.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	entrega := entregaPreparadaPerfilFijoPrueba(vinculo.PrincipalID, vinculo.PerfilActivoRef, ahora)
	entrega.Peticion.Solicitud.Periodo.Fin = time.Time{}
	entrega.Peticion.Solicitud.Periodo.CausaFin = "fin_sustitucion"
	entrega.Peticion.Solicitud.Periodo.PoliticaFin = domain.PoliticaFin{
		ReglaRef: "regla:modalidad-ratificada-sintetica", CatalogoVersion: 3,
		CatalogoHuellaSHA256: strings.Repeat("a", 64), FechaFin: "opcional", CausaFin: "fin_sustitucion",
	}
	if err := entrega.ValidarReserva(); err != nil {
		t.Fatalf("petición ratificada inválida: %v", err)
	}
	soporte.opcionesCatalogo = &opcionesAnalisisCTDesarrollo{}
	ctx := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, rutaEntregaPeticionCentro)
	capacidad := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	capacidad.metodo = "POST"
	ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	ctx = context.WithValue(ctx, claveAltaDePeticionDesarrollo{}, entrega)
	solicitud := ports.SolicitudResolverFlujo{
		OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo,
		CentroRef:       entrega.Peticion.Solicitud.CentroRef,
		CategoriaRef:    entrega.Peticion.Solicitud.CategoriaRef,
		MotivoClave:     entrega.Peticion.Solicitud.MotivoClave, Instante: ahora,
	}
	if _, err := soporte.ResolverFlujoAlta(ctx, solicitud); err != nil {
		t.Fatalf("petición ratificada pendiente perdió su flujo original: %v", err)
	}
	motivo := ports.SolicitudResolverMotivoAutorizacionAltaV3{
		OrganizacionRef: solicitud.OrganizacionRef, Flujo: soporte.flujo.Flujo,
		MotivoClave: solicitud.MotivoClave, Instante: ahora,
	}
	if _, err := soporte.ResolverMotivoAutorizacionAltaV3(ctx, motivo); err != nil {
		t.Fatalf("petición ratificada pendiente perdió su motivo original: %v", err)
	}
}
