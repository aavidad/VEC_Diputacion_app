package bootstrap

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type consultorCuadroLectorPrueba struct{ llamadas int }

func (c *consultorCuadroLectorPrueba) Consultar(context.Context, ports.SolicitudCuadroRRHH) (ports.PaginaCuadroRRHH, error) {
	c.llamadas++
	return ports.PaginaCuadroRRHH{}, nil
}

type consultorDetalleLectorPrueba struct{ llamadas int }

func (c *consultorDetalleLectorPrueba) Consultar(context.Context, ports.SolicitudDetalleRRHH) (ports.DetalleExpedienteRRHH, error) {
	c.llamadas++
	return ports.DetalleExpedienteRRHH{}, nil
}

var _ httpinterno.ConsultorCuadroRRHH = (*consultorCuadroLectorPrueba)(nil)
var _ httpinterno.ConsultorDetalleRRHH = (*consultorDetalleLectorPrueba)(nil)

func TestMultiplexoresLectoresRRHHDesarrolloSeleccionanSoloCapacidadNominal(t *testing.T) {
	soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	lector := &soporteAltaContratacionTemporalDesarrollo{
		sello:             soporte.sello,
		principalID:       "lector:rrhh:prueba",
		certificadoSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	cuadro := &consultorCuadroLectorPrueba{}
	detalle := &consultorDetalleLectorPrueba{}
	mux := nuevoMultiplexoresLectoresRRHHDesarrollo(soporte.sello)
	if !mux.registrar(lector.principalID, lector, cuadro, detalle) {
		t.Fatal("no registró lector nominal")
	}
	principalLector := clonarPrincipalDesarrollo(principal)
	principalLector.ID = lector.principalID
	principalLector.Attributes["certificate_sha256"] = lector.certificadoSHA256
	cuadroSolicitud, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 1, "")
	if err != nil {
		t.Fatal(err)
	}
	detalleSolicitud, err := ports.NuevaSolicitudDetalleRRHH("expediente:rrhh:lector", 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		nombre string
		ruta   string
		actor  vecdomain.Principal
	}{
		{"cuadro", httpinterno.RutaConsultaCuadroRRHH, principalLector},
		{"detalle", httpinterno.RutaConsultaDetalleRRHH, principalLector},
	} {
		ctx := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidadConsultaContratacionTemporalDesarrollo{sello: soporte.sello, ruta: caso.ruta, principal: caso.actor})
		if caso.ruta == httpinterno.RutaConsultaCuadroRRHH {
			if _, err := mux.Consultar(ctx, cuadroSolicitud); err != nil {
				t.Fatalf("%s: %v", caso.nombre, err)
			}
		} else if _, err := mux.ConsultarDetalle(ctx, detalleSolicitud); err != nil {
			t.Fatalf("%s: %v", caso.nombre, err)
		}
	}
	if cuadro.llamadas != 1 || detalle.llamadas != 1 {
		t.Fatalf("delegación inesperada: %d/%d", cuadro.llamadas, detalle.llamadas)
	}
	ajeno := clonarPrincipalDesarrollo(principalLector)
	ajeno.ID = "lector:rrhh:ajeno"
	ctxAjeno := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidadConsultaContratacionTemporalDesarrollo{sello: soporte.sello, ruta: httpinterno.RutaConsultaCuadroRRHH, principal: ajeno})
	if _, err := mux.Consultar(ctxAjeno, cuadroSolicitud); !errors.Is(err, ports.ErrAutorizacionDenegada) || cuadro.llamadas != 1 {
		t.Fatalf("hubo fallback para lector ajeno: %v", err)
	}
	for _, caso := range []struct {
		nombre string
		ctx    context.Context
	}{
		{"sello_ajeno", context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidadConsultaContratacionTemporalDesarrollo{sello: &selloConsultasContratacionTemporalDesarrollo{}, ruta: httpinterno.RutaConsultaCuadroRRHH, principal: principalLector})},
		{"certificado_ajeno", context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidadConsultaContratacionTemporalDesarrollo{sello: soporte.sello, ruta: httpinterno.RutaConsultaCuadroRRHH, principal: func() vecdomain.Principal {
			p := clonarPrincipalDesarrollo(principalLector)
			p.Attributes["certificate_sha256"] = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
			return p
		}()})},
		{"ruta_ajena", context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidadConsultaContratacionTemporalDesarrollo{sello: soporte.sello, ruta: httpinterno.RutaConsultaDetalleRRHH, principal: principalLector})},
		{"nulo", nil},
	} {
		if _, err := mux.Consultar(caso.ctx, cuadroSolicitud); !errors.Is(err, ports.ErrAutorizacionDenegada) || cuadro.llamadas != 1 {
			t.Fatalf("%s fue aceptado: %v", caso.nombre, err)
		}
	}
	ctxCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := mux.Consultar(ctxCancelado, cuadroSolicitud); !errors.Is(err, ports.ErrAutorizacionDenegada) || cuadro.llamadas != 1 {
		t.Fatalf("cancelado fue aceptado: %v", err)
	}
}

func TestAsignacionConsultaLectorRRHHCompatibleRecuperaVersionHeredada(t *testing.T) {
	alta, _, _ := escenarioConsultasRRHHDesarrolloPrueba(t)
	soporte := alta.soporte
	plantilla := soporte.instantaneaDetalleRRHH
	perfil := plantilla.AsignacionPerfil
	perfil.AsignacionID, perfil.Version = "asg_lector_heredado_0123456789", 7
	huella, err := perfil.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	base := asignacionActualPostgreSQLContratacionTemporalDesarrollo{referencia: perfil.Referencia(), identificador: perfil.AsignacionID, version: int64(perfil.Version), perfilRef: perfil.PerfilActivoRef, principalID: perfil.PrincipalID, versionRolRef: plantilla.VersionRol.Referencia(), huella: huella}
	if !asignacionConsultaLectorRRHHCompatible(soporte, base) {
		t.Fatal("no recuperó detalle legítimo con ID/version heredados")
	}
	for _, caso := range []struct {
		nombre string
		mutar  func(*asignacionActualPostgreSQLContratacionTemporalDesarrollo)
	}{
		{"rol", func(a *asignacionActualPostgreSQLContratacionTemporalDesarrollo) { a.versionRolRef = "rol:ajeno" }},
		{"huella", func(a *asignacionActualPostgreSQLContratacionTemporalDesarrollo) {
			a.huella = "0000000000000000000000000000000000000000000000000000000000000000"
		}},
		{"referencia", func(a *asignacionActualPostgreSQLContratacionTemporalDesarrollo) { a.referencia = "asignacion:ajena" }},
	} {
		actual := base
		caso.mutar(&actual)
		if asignacionConsultaLectorRRHHCompatible(soporte, actual) {
			t.Fatalf("%s aceptado", caso.nombre)
		}
	}
}

func TestSemillaInicialLectorRRHHRechazaAparicionConcurrente(t *testing.T) {
	alta, _, _ := escenarioConsultasRRHHDesarrolloPrueba(t)
	semilla := alta.soporte.instantaneaCuadroRRHH
	if !semillaInicialLectorRRHHIntacta(semilla, semilla) {
		t.Fatal("semilla inicial rechazada")
	}
	aparecida := semilla
	aparecida.AsignacionPerfil.Version++
	if semillaInicialLectorRRHHIntacta(semilla, aparecida) {
		t.Fatal("aceptó versión posterior aparecida durante preparar")
	}
	aparecida = semilla
	aparecida.AsignacionPerfil.AsignacionID = "asg_aparecida_0123456789"
	if semillaInicialLectorRRHHIntacta(semilla, aparecida) {
		t.Fatal("aceptó identificador aparecido durante preparar")
	}
	aparecida = semilla
	aparecida.AsignacionPerfil.EmitidaPor = "identidad:ajena"
	if semillaInicialLectorRRHHIntacta(semilla, aparecida) {
		t.Fatal("aceptó huella divergente durante preparar")
	}
}
