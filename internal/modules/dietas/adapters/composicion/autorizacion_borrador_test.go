package composicion

import (
	"context"
	"errors"
	"testing"

	dietasapp "vec-diputacion-granada/internal/modules/dietas/application"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type emisorBorradorPrueba struct{ llamadas int }

func (e *emisorBorradorPrueba) EmitirMaterialAutorizacionAtestadaV3(context.Context, vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	e.llamadas++
	return vecdomain.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, errors.New("emisor no disponible")
}

func TestEmisorBorradorRechazaRecursoYOperacionAntesDelPDP(t *testing.T) {
	base, _ := identidadYMaterialR15(t)
	relacion := relacionR15(base.Contexto.Contexto.PersonaRef, "a")
	sello := selloRelacion(relacion, base.FechaReferencia)
	emisor := &emisorBorradorPrueba{}
	proveedor, err := NuevoEmisorAutorizacionBorrador(emisor, vecdomain.ReferenciaEntradaCatalogo{CatalogoID: "motivos-dietas", CatalogoVersion: 1, CatalogoHuellaSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", EntradaClave: "crear-borrador"})
	if err != nil {
		t.Fatal(err)
	}
	s := dietasports.SolicitudOperacionBorrador{Operacion: dietasports.OperacionCrearBorrador, Crear: dietasports.SolicitudCrearBorradorPropio{ClaveIdempotencia: "clave_0123456789abcdef", FechaInicio: "2026-09-21", FechaFin: "2026-09-21", Motivo: "Visita técnica", CodigosRuta: []string{}}}
	efecto, err := dietasapp.ConstruirEfectoAutorizacionBorrador(base.Contexto, traducirRelacion(relacion), sello, s)
	if err != nil {
		t.Fatal(err)
	}
	efecto.Recurso.Referencia = "dietas:borradores:ajenos"
	if _, err := proveedor.AutorizarBorradorPropio(context.Background(), base, s, sello, efecto); !errors.Is(err, dietasports.ErrAccesoBorradorDenegado) || emisor.llamadas != 0 {
		t.Fatalf("recurso ajeno: error=%v llamadas=%d", err, emisor.llamadas)
	}
	s.Operacion = "operacion_ajena"
	if _, err := proveedor.AutorizarBorradorPropio(context.Background(), base, s, sello, efecto); !errors.Is(err, dietasports.ErrAccesoBorradorDenegado) || emisor.llamadas != 0 {
		t.Fatalf("operación ajena: error=%v llamadas=%d", err, emisor.llamadas)
	}
}
