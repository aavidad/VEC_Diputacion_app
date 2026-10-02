package documentosjustificacion

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	docapp "vec-diputacion-granada/internal/vec/documentos/application"
	docdomain "vec-diputacion-granada/internal/vec/documentos/domain"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type relojPrueba struct{ ahora time.Time }

func (r *relojPrueba) Ahora() time.Time { return r.ahora }

type politicasPrueba struct {
	p vecports.PoliticaConservacionDocumental
}

func (p *politicasPrueba) BuscarPoliticasConservacionDocumental(context.Context, vecports.SolicitudPoliticaConservacionDocumental) ([]vecports.PoliticaConservacionDocumental, error) {
	return []vecports.PoliticaConservacionDocumental{p.p}, nil
}

// Ninguna autorización positiva ficticia. El servicio común real debe llegar
// al autorizador y detenerse allí sin metadatos ni bytes confirmados.
type autorizadorDenegado struct {
	preimagen             []byte
	documento, expediente string
	llamadas              int
}

func (a *autorizadorDenegado) AutorizarRegistroExterno(_ context.Context, b []byte, d, e string) (docports.AutorizacionV3, error) {
	a.llamadas++
	a.preimagen = append([]byte(nil), b...)
	a.documento = d
	a.expediente = e
	return docports.AutorizacionV3{}, docports.ErrAccesoDenegado
}

type repoPrueba struct {
	docports.Repositorio
	llamadas int
}

func (r *repoPrueba) ConfirmarReferenciaExterna(context.Context, docports.AltaExternaPersistente) (docdomain.Documento, error) {
	r.llamadas++
	return docdomain.Documento{}, errors.New("efecto inesperado")
}

type contextosPrueba struct {
	politica    vecports.SolicitudPoliticaConservacionDocumental
	autorizador docports.AutorizadorRegistroExterno
	err         error
}

func (c *contextosPrueba) PrepararContextoRegistro(context.Context, ports.OrdenJustificacion, domain.SolicitudJustificable, domain.PoliticaJustificacion) (vecports.SolicitudPoliticaConservacionDocumental, docports.AutorizadorRegistroExterno, error) {
	return c.politica, c.autorizador, c.err
}

type proveedorDenegado struct{}

func (*proveedorDenegado) ProveerMaterialJustificacion(context.Context, domain.MaterialJustificacion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrJustificacionNoDisponible
}
func ordenPrueba(t *testing.T, ahora time.Time) ports.OrdenJustificacion {
	t.Helper()
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_0123456789abcdefghijkl", Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	inst := vecdomain.InstantaneaContextoActor{VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 1, PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 1, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Minute), VigenteHasta: ahora.Add(time.Minute), Vinculos: []vecdomain.VinculoReferenciaContextoActor{{VinculoRef: "vin_0123456789abcdefghijkl", Version: 1, Tipo: vecdomain.TipoReferenciaContextoActorEmpleado, Referencia: "emp_0123456789abcdefghijkl", Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Minute), VigenteHasta: ahora.Add(time.Minute)}}}
	actor, e := vecdomain.NuevoContextoActor(cuenta, inst, ahora)
	if e != nil {
		t.Fatal(e)
	}
	o, e := ports.NuevaOrdenJustificacion(actor, &proveedorDenegado{})
	if e != nil {
		t.Fatal(e)
	}
	return o
}
func escenario(t *testing.T) (*Adapter, *repoPrueba, *contextosPrueba, *autorizadorDenegado, ports.OrdenJustificacion, domain.SolicitudJustificable, domain.PoliticaJustificacion, domain.DocumentoJustificacion) {
	t.Helper()
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	ahora := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	p := domain.PoliticaJustificacion{Referencia: "politica:justificacion:v1", Version: 1, SHA256: strings.Repeat("a", 64), CatalogoVersionRef: "catalogo:permiso:v1", PermisoRef: "permiso:neutral", TipoDocumentalRef: ref("3"), CustodioID: "custodia.interna", MotivosRef: []string{"motivo:conforme"}}
	s := domain.SolicitudJustificable{SolicitudRef: "permiso:cronos:solicitud:ensayo001", EmpleadoRef: "emp_0123456789abcdefghijkl", CatalogoVersionRef: p.CatalogoVersionRef, PermisoRef: p.PermisoRef, ExpedienteDocumentalRef: ref("4"), Version: 3, Estado: domain.EstadoPermisoConcedido, JustificanteExigido: true}
	politica, e := vecports.NuevaSolicitudPoliticaConservacionDocumental(ref("1"), ref("2"), p.TipoDocumentalRef, s.ExpedienteDocumentalRef, ref("5"), 1, bytes.Repeat([]byte{0x6a}, 32), ref("6"), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	if e != nil {
		t.Fatal(e)
	}
	conservacion, e := vecports.NuevaPoliticaConservacionDocumental(politica, time.Date(2035, 1, 1, 0, 0, 0, 0, time.UTC), vecports.ProteccionPoliticaConservacionDocumentalOrdinaria, "", vecports.EstadoPoliticaConservacionDocumentalAprobada, time.Time{})
	if e != nil {
		t.Fatal(e)
	}
	repo := &repoPrueba{}
	aut := &autorizadorDenegado{}
	c := &contextosPrueba{politica: politica, autorizador: aut}
	servicio := &docapp.Servicio{Repositorio: repo, Politicas: &politicasPrueba{conservacion}, Reloj: &relojPrueba{ahora}}
	a, e := Nuevo(servicio, c)
	if e != nil {
		t.Fatal(e)
	}
	d := domain.DocumentoJustificacion{ID: ref("8"), Version: 1, SHA256: strings.Repeat("e", 64), CustodioID: p.CustodioID, CustodiaRef: "original:ensayo001"}
	return a, repo, c, aut, ordenPrueba(t, ahora), s, p, d
}
func TestAdapterUsaServicioComunSinBytesYExigeAutorizacion(t *testing.T) {
	a, r, _, aut, o, s, p, d := escenario(t)
	if e := a.PrepararRegistro(context.Background(), o, s, p); e != nil {
		t.Fatal(e)
	}
	_, e := a.RegistrarJustificante(context.Background(), o, s, p, d, "ref:"+strings.Repeat("9", 64))
	if !errors.Is(e, docports.ErrAccesoDenegado) || aut.llamadas != 1 || r.llamadas != 0 {
		t.Fatal("no pasa por autorizador comun", e, aut.llamadas, r.llamadas)
	}
	if aut.documento != d.ID || aut.expediente != s.ExpedienteDocumentalRef || !bytes.Contains(aut.preimagen, []byte(`"modulo_id":"cronos"`)) || !bytes.Contains(aut.preimagen, []byte(d.CustodiaRef)) || !bytes.Contains(aut.preimagen, []byte(d.SHA256)) || bytes.Contains(aut.preimagen, []byte("contenido")) {
		t.Fatal("preimagen documental discordante")
	}
}
func TestAdapterGateCerradoTypedNilYReferencias(t *testing.T) {
	for _, caso := range []string{"contextos", "repo", "politicas", "reloj", "autorizador", "orden", "politica", "expediente", "custodio"} {
		t.Run(caso, func(t *testing.T) {
			a, r, c, aut, o, s, p, d := escenario(t)
			switch caso {
			case "contextos":
				var n *contextosPrueba
				a.contextos = n
			case "repo":
				var n *repoPrueba
				a.servicio.Repositorio = n
			case "politicas":
				var n *politicasPrueba
				a.servicio.Politicas = n
			case "reloj":
				var n *relojPrueba
				a.servicio.Reloj = n
			case "autorizador":
				var n *autorizadorDenegado
				c.autorizador = n
			case "orden":
				o = ports.OrdenJustificacion{}
			case "politica":
				c.politica = vecports.SolicitudPoliticaConservacionDocumental{}
			case "expediente":
				s.ExpedienteDocumentalRef = "ref:" + strings.Repeat("f", 64)
			case "custodio":
				d.CustodioID = "otro.interno"
			}
			if _, e := a.RegistrarJustificante(context.Background(), o, s, p, d, "ref:"+strings.Repeat("9", 64)); e == nil || aut.llamadas != 0 || r.llamadas != 0 {
				t.Fatal("efecto con gate cerrado", e)
			}
		})
	}
	var c *contextosPrueba
	if _, e := Nuevo(&docapp.Servicio{}, c); !errors.Is(e, ports.ErrJustificacionNoDisponible) {
		t.Fatal(e)
	}
	var a *Adapter
	if e := a.PrepararRegistro(context.Background(), ports.OrdenJustificacion{}, domain.SolicitudJustificable{}, domain.PoliticaJustificacion{}); !errors.Is(e, ports.ErrJustificacionNoDisponible) {
		t.Fatal(e)
	}
}

func TestRegistroDocumentalConservaConfirmacionComun(t *testing.T) {
	_, _, _, _, _, s, p, d := escenario(t)
	ahora := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	entrada := docports.AltaExterna{ID: d.ID, ModuloID: "cronos", ExpedienteRef: s.ExpedienteDocumentalRef, TipoRef: p.TipoDocumentalRef,
		Version: d.Version, Custodia: docdomain.ReferenciaCustodiaExterna{CustodioID: d.CustodioID, Referencia: d.CustodiaRef, HuellaSHA256: d.SHA256}}
	confirmado := docdomain.Documento{ID: d.ID, NumeroVEC: "VEC-2026-14", ModuloID: entrada.ModuloID,
		ExpedienteRef: entrada.ExpedienteRef, TipoRef: entrada.TipoRef, Version: d.Version,
		HuellaSHA256: d.SHA256, PoliticaRef: ref("5"), VersionPolitica: 3, HuellaPoliticaSHA256: strings.Repeat("6", 64),
		ConservacionHasta: ahora.AddDate(5, 0, 0), Proteccion: "conservacion", EstadoPolitica: docdomain.EstadoPoliticaAprobada,
		EstadoFirma: docdomain.EstadoFirmaPendienteProveedor, CreadoEn: ahora, Custodia: docdomain.CustodiaExterna,
		CustodiaExternaRef: entrada.Custodia}
	r, err := registroConfirmado(confirmado, entrada, d)
	if err != nil || r.Documento != d || r.NumeroVEC != confirmado.NumeroVEC || !r.CreadoEnUTC.Equal(ahora) || r.PoliticaRef != confirmado.PoliticaRef || r.PoliticaVersion != confirmado.VersionPolitica {
		t.Fatal("se perdieron datos confirmados por Documentos", err, r)
	}
	for nombre, cambiar := range map[string]func(*docdomain.Documento){
		"otro expediente": func(c *docdomain.Documento) { c.ExpedienteRef = ref("9") },
		"otra huella": func(c *docdomain.Documento) {
			c.HuellaSHA256 = strings.Repeat("7", 64)
			c.CustodiaExternaRef.HuellaSHA256 = c.HuellaSHA256
		},
		"sin numero": func(c *docdomain.Documento) { c.NumeroVEC = "" },
	} {
		t.Run(nombre, func(t *testing.T) {
			c := confirmado
			cambiar(&c)
			if _, err := registroConfirmado(c, entrada, d); !errors.Is(err, ports.ErrJustificacionNoDisponible) {
				t.Fatal("aceptó confirmación discordante", err)
			}
		})
	}
}
