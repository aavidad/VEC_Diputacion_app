package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type relojMarcajePrueba struct{ instante time.Time }

func (r relojMarcajePrueba) AhoraUTC() time.Time { return r.instante }

type repositorioMarcajePrueba struct {
	llamadas int
}

func (r *repositorioMarcajePrueba) RegistrarOriginalAutorizado(_ context.Context, _ domain.MarcajeOriginal, _ domain.MaterialAutorizacionMarcajePropio, _ vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboMarcajePropio, error) {
	r.llamadas++
	return ports.ReciboMarcajePropio{}, errors.New("no debe persistir con V3 vacio")
}

type proveedorVacio struct{}

func (proveedorVacio) ProveerMaterialMarcajePropio(context.Context, domain.MaterialAutorizacionMarcajePropio) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, nil
}

func TestServicioMarcajesRechazaProveedorV3NoAtestadoAntesDePersistir(t *testing.T) {
	r := &repositorioMarcajePrueba{}
	s, _ := NuevoServicioMarcajes(r, relojMarcajePrueba{time.Now().UTC()})
	_, e := s.RegistrarMarcajePropio(context.Background(), contexto(t), ports.SolicitudMarcajePropio{Movimiento: domain.PunchEntry, ClaveOperacion: "op_00001"})
	if !errors.Is(e, ErrContextoMarcajeNoAcreditado) || r.llamadas != 0 {
		t.Fatal(e, r.llamadas)
	}
}
func TestServicioMarcajesFallaCerradoAntesDePersistir(t *testing.T) {
	r := &repositorioMarcajePrueba{}
	s, _ := NuevoServicioMarcajes(r, relojMarcajePrueba{time.Now().UTC()})
	c := contexto(t)
	c.OrdenConsumo = ports.OrdenConsumoAutorizacion{}
	_, e := s.RegistrarMarcajePropio(context.Background(), c, ports.SolicitudMarcajePropio{Movimiento: domain.PunchEntry, ClaveOperacion: "op_0001"})
	if !errors.Is(e, ErrContextoMarcajeNoAcreditado) || r.llamadas != 0 {
		t.Fatal(e)
	}
}
func contexto(t *testing.T) ports.ContextoMarcajePropio {
	a, e := domain.NuevaAcreditacionCanalMarcaje(domain.DatosAcreditacionCanalMarcaje{PoliticaVersionRef: "pol_v1", CanalRef: "canal_1", OrigenRef: "origen_1", CalidadRef: "nivel_1"})
	if e != nil {
		t.Fatal(e)
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_0123456789abcdefghijkl", Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	instantanea := vecdomain.InstantaneaContextoActor{VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 1, PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 1, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Minute), VigenteHasta: ahora.Add(time.Minute), Vinculos: []vecdomain.VinculoReferenciaContextoActor{{VinculoRef: "vin_0123456789abcdefghijkl", Version: 1, Tipo: vecdomain.TipoReferenciaContextoActorEmpleado, Referencia: "emp_0123456789abcdefghijkl", Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Minute), VigenteHasta: ahora.Add(time.Minute)}}}
	actor, e := vecdomain.NuevoContextoActor(cuenta, instantanea, ahora)
	if e != nil {
		t.Fatal(e)
	}
	o, e := ports.NuevaOrdenConsumoAutorizacion(actor, proveedorVacio{})
	if e != nil {
		t.Fatal(e)
	}
	return ports.ContextoMarcajePropio{CanalAcreditado: a, OrdenConsumo: o}
}

func TestRecursoMarcajeComprometeMaterialExactoSQL(t *testing.T) {
	c := contexto(t)
	a, err := c.OrdenConsumo.ContextoActor()
	if err != nil {
		t.Fatal(err)
	}
	refs, _ := a.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
	m := domain.MaterialAutorizacionMarcajePropio{ActorRef: a.PersonaRef, PerfilRef: a.PerfilActivoRef, EmpleadoRef: refs[0], ClaveOperacion: "op-cronos-0001", Movimiento: domain.PunchEntry, InstanteUTC: time.Date(2026, 9, 19, 12, 0, 0, 123456000, time.UTC), Canal: c.CanalAcreditado}
	r, err := RecursoMarcajePropio(m)
	if err != nil {
		t.Fatal(err)
	}
	material, _ := m.Canonico()
	h := sha256.Sum256(material)
	preimagen := `{"ambitos":{"empleado_ref":"` + refs[0] + `"},"atributos":{"material_sha256":"` + hex.EncodeToString(h[:]) + `"}}`
	esperado := sha256.Sum256([]byte(preimagen))
	real, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil || real != hex.EncodeToString(esperado[:]) {
		t.Fatal("recurso no coincide con preimagen SQL")
	}
	m.Movimiento = domain.PunchExit
	r2, _ := RecursoMarcajePropio(m)
	diferente, _ := r2.HuellaContextoAutorizacionSHA256()
	if real == diferente {
		t.Fatal("movimiento no ligado a concesión")
	}
}
