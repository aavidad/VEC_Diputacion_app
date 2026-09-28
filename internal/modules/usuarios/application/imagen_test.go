package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/usuarios/domain"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type proveedorImagenPrueba struct {
	llamadas   int
	denegar    bool
	materiales []ports.MaterialImagen
}

func (p *proveedorImagenPrueba) ProveerMaterialImagen(ctx context.Context, m ports.MaterialImagen) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.llamadas++
	p.materiales = append(p.materiales, m)
	if p.denegar {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrImagenProhibido
	}
	base := &proveedorPrueba{}
	return base.ProveerMaterialPreferencias(ctx, ports.MaterialPreferencias{PersonaRef: m.TitularPersonaRef, PerfilRef: m.PerfilRef, Accion: m.Accion})
}

type opImagenPrueba struct {
	huella string
	recibo ports.ReciboImagen
}
type registroImagenPrueba struct {
	catalogo            domain.CatalogoImagen
	estado              ports.EstadoImagen
	existe              bool
	ops                 map[string]opImagenPrueba
	guardarErr          error
	guardarReplay       bool
	lecturas, guardados int
}

func (r *registroImagenPrueba) CatalogoVigente(context.Context) (domain.CatalogoImagen, error) {
	return r.catalogo, nil
}
func (r *registroImagenPrueba) Leer(_ context.Context, _ ports.OrdenImagen, m ports.MaterialImagen, _ vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.EstadoImagen, bool, error) {
	r.lecturas++
	if r.existe && r.estado.PersonaRef != m.TitularPersonaRef {
		return ports.EstadoImagen{}, false, nil
	}
	return r.estado, r.existe, nil
}
func (r *registroImagenPrueba) Recuperar(_ context.Context, _ ports.OrdenImagen, m ports.MaterialImagen, _ vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboImagen, bool, error) {
	op, ok := r.ops[m.ClaveOperacion]
	if !ok {
		return ports.ReciboImagen{}, false, nil
	}
	if op.huella != m.HuellaPeticion {
		return ports.ReciboImagen{}, false, ports.ErrImagenConflicto
	}
	return op.recibo, true, nil
}
func (r *registroImagenPrueba) Guardar(_ context.Context, _ ports.OrdenImagen, m ports.MaterialImagen, _ vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboImagen, error) {
	r.guardados++
	if r.guardarErr != nil {
		return ports.ReciboImagen{}, r.guardarErr
	}
	actual := uint64(0)
	if r.existe {
		actual = r.estado.Version
	}
	if actual != m.VersionEsperada {
		return ports.ReciboImagen{}, ports.ErrImagenConflicto
	}
	if r.ops == nil {
		r.ops = map[string]opImagenPrueba{}
	}
	if old, ok := r.ops[m.ClaveOperacion]; ok {
		if old.huella != m.HuellaPeticion {
			return ports.ReciboImagen{}, ports.ErrImagenConflicto
		}
		return old.recibo, nil
	}
	r.estado = ports.EstadoImagen{PersonaRef: m.TitularPersonaRef, Version: actual + 1, CatalogoVersionRef: m.CatalogoVersionRef, Eleccion: m.Eleccion}
	r.existe = true
	recibo := ports.ReciboImagen{ReciboRef: "recibo_0123456789abcdef", PersonaRef: m.TitularPersonaRef, Version: actual + 1, CatalogoVersionRef: m.CatalogoVersionRef, Eleccion: m.Eleccion, FechaUTC: time.Now().UTC(), Replay: r.guardarReplay}
	r.ops[m.ClaveOperacion] = opImagenPrueba{m.HuellaPeticion, recibo}
	return recibo, nil
}

type transformadorImagenPrueba struct{ llamadas int }

func (p *transformadorImagenPrueba) Procesar(_ context.Context, b []byte, l ports.LimitesTransformacionImagen) (ports.ImagenProcesada, error) {
	p.llamadas++
	if l.MaxBytes != ports.TamanoMaximoOriginalImagen || l.LadoSalida != 256 {
		return ports.ImagenProcesada{}, errors.New("límites")
	}
	return ports.ImagenProcesada{Bytes: []byte("foto-recodificada"), TipoReal: "image/png", TipoSalida: "image/png", AnchoOriginal: 600, AltoOriginal: 800, Ancho: 256, Alto: 256, OrientacionAplicada: true, MetadatosEliminados: true}, nil
}

type custodiaImagenPrueba struct {
	reserva                        ports.ReservaImagen
	existe, confirmada, disponible bool
	reservas, recuperaciones       int
	errorConfirmar                 bool
}

func (c *custodiaImagenPrueba) Reservar(_ context.Context, _ ports.OrdenImagen, _ ports.MaterialImagen, r ports.ReservaImagen, p ports.ImagenProcesada) (ports.ReservaImagen, error) {
	c.reservas++
	if len(p.Bytes) == 0 {
		return ports.ReservaImagen{}, errors.New("sin bytes")
	}
	r.DocumentoRef = "doc_0123456789abcdef"
	c.reserva = r
	c.existe = true
	c.disponible = true
	return r, nil
}
func (c *custodiaImagenPrueba) RecuperarReserva(_ context.Context, _ ports.OrdenImagen, _ ports.MaterialImagen, persona, clave string) (ports.ReservaImagen, bool, error) {
	c.recuperaciones++
	if c.existe && c.reserva.PersonaRef == persona && c.reserva.ClaveOperacion == clave {
		return c.reserva, true, nil
	}
	return ports.ReservaImagen{}, false, nil
}
func (c *custodiaImagenPrueba) ConfirmarReserva(_ context.Context, _ ports.OrdenImagen, _ ports.MaterialImagen, _ ports.ReservaImagen) error {
	if c.errorConfirmar {
		return errors.New("documentos caído")
	}
	c.confirmada = true
	return nil
}
func (c *custodiaImagenPrueba) Disponible(_ context.Context, _ ports.OrdenImagen, _ ports.MaterialImagen, ref string) (bool, error) {
	return c.disponible && ref == c.reserva.DocumentoRef, nil
}

type nombreImagenPrueba struct {
	nombre string
	err    error
}

func (n nombreImagenPrueba) NombreVisible(_ context.Context, _ ports.OrdenImagen, _ ports.MaterialImagen, _ string) (string, error) {
	return n.nombre, n.err
}

func fixtureImagen(t *testing.T, audiencia ports.AudienciaImagen) (*ServicioImagen, ports.OrdenImagen, *proveedorImagenPrueba, *registroImagenPrueba, *custodiaImagenPrueba, *transformadorImagenPrueba, vecdomain.ContextoActor) {
	t.Helper()
	_, actor := ordenPrueba(t, &proveedorPrueba{})
	p := &proveedorImagenPrueba{}
	o, err := ports.NuevaOrdenImagen(actor, audiencia, p)
	if err != nil {
		t.Fatal(err)
	}
	r := &registroImagenPrueba{catalogo: domain.CatalogoBaseImagen()}
	c := &custodiaImagenPrueba{}
	tr := &transformadorImagenPrueba{}
	s, err := NuevoServicioImagen(r, tr, c, nombreImagenPrueba{err: errors.New("sin fuente nominal")}, func() time.Time { return time.Now().UTC() })
	if err != nil {
		t.Fatal(err)
	}
	return s, o, p, r, c, tr, actor
}

func TestImagenModosRetiradaCASYReplayAutorizado(t *testing.T) {
	ctx := context.Background()
	s, o, p, r, _, _, actor := fixtureImagen(t, ports.AudienciaImagenPersonal)
	cat := r.catalogo.VersionRef
	v, err := s.ConsultarPropia(ctx, o)
	if err != nil {
		t.Fatal(err)
	}
	if v.Estado.Version != 0 || v.Estado.Eleccion.Modo != domain.ModoIniciales || v.NombreAutorizado != "" || r.guardados != 0 {
		t.Fatalf("default %+v", v)
	}
	pet := ports.PeticionElegirImagen{VersionEsperada: 0, CatalogoVersionRef: cat, ClaveOperacion: "imagen_op_0123456789", Modo: domain.ModoIcono, Paleta: "verde", Icono: "hoja"}
	rec, err := s.Elegir(ctx, o, pet)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Version != 1 || rec.Eleccion.Modo != domain.ModoIcono || r.guardados != 1 {
		t.Fatalf("recibo %+v", rec)
	}
	_, err = s.Elegir(ctx, o, ports.PeticionElegirImagen{VersionEsperada: 0, CatalogoVersionRef: cat, ClaveOperacion: "imagen_op_otra_123456", Modo: domain.ModoIniciales, Paleta: "azul"})
	if !errors.Is(err, ports.ErrImagenConflicto) {
		t.Fatalf("CAS: %v", err)
	}
	replay, err := s.Elegir(ctx, o, pet)
	if err != nil || !replay.Replay || replay.ReciboRef != rec.ReciboRef || r.guardados != 2 {
		t.Fatalf("replay %+v %v", replay, err)
	}
	p.denegar = true
	_, err = s.Elegir(ctx, o, pet)
	if !errors.Is(err, ports.ErrImagenProhibido) {
		t.Fatalf("replay sin permiso: %v", err)
	}
	p.denegar = false
	quit, err := s.Retirar(ctx, o, 1, cat, "imagen_op_retirar_012345", "azul")
	if err != nil || quit.Version != 2 || quit.Eleccion.Modo != domain.ModoIniciales || r.estado.Eleccion.DocumentoRef != "" {
		t.Fatalf("retirada %+v %v", quit, err)
	}
	if p.materiales[0].ActorPersonaRef != actor.PersonaRef {
		t.Fatal("persona libre")
	}
	// Un catálogo posterior puede retirar una opción sin invalidar el estado
	// histórico que la persona ya había elegido.
	r.estado.Eleccion = domain.EleccionImagen{Modo: domain.ModoIcono, Paleta: "verde", Icono: "hoja"}
	r.catalogo.Iconos = r.catalogo.Iconos[:1]
	v, err = s.ConsultarPropia(ctx, o)
	if err != nil || v.Estado.Eleccion.Icono != "hoja" {
		t.Fatalf("opción histórica %+v %v", v, err)
	}
}

func TestLecturaAjenaSoloAudienciaInternaYPermisoNominal(t *testing.T) {
	ctx := context.Background()
	s, o, p, _, _, _, _ := fixtureImagen(t, ports.AudienciaImagenPersonal)
	otra := "per_123456789abcdefghijkl"
	_, err := s.ConsultarAjena(ctx, o, otra)
	if !errors.Is(err, ports.ErrImagenProhibido) || p.llamadas != 0 {
		t.Fatalf("exterior: %v", err)
	}
	interna, ordenInterna, pi, ri, _, _, _ := fixtureImagen(t, ports.AudienciaImagenInterna)
	ri.estado = ports.EstadoImagen{PersonaRef: otra, Version: 1, CatalogoVersionRef: ri.catalogo.VersionRef, Eleccion: domain.EleccionImagen{Modo: domain.ModoIniciales, Paleta: "azul"}}
	ri.existe = true
	pi.denegar = true
	_, err = interna.ConsultarAjena(ctx, ordenInterna, otra)
	if !errors.Is(err, ports.ErrImagenProhibido) || ri.lecturas != 0 {
		t.Fatalf("permiso nominal: %v", err)
	}
	pi.denegar = false
	vista, err := interna.ConsultarAjena(ctx, ordenInterna, otra)
	if err != nil || vista.Estado.PersonaRef != otra || vista.NombreAutorizado != "" || ri.lecturas != 1 {
		t.Fatalf("lectura interna %+v %v", vista, err)
	}
	m := pi.materiales[len(pi.materiales)-1]
	if m.Accion != ports.AccionImagenConsultarAjena || m.TitularPersonaRef != otra {
		t.Fatal("material ajeno no nominal")
	}
}

func TestFotoReservaReconciliableSinOriginal(t *testing.T) {
	ctx := context.Background()
	s, o, _, r, c, tr, _ := fixtureImagen(t, ports.AudienciaImagenPersonal)
	r.guardarErr = ports.ErrImagenNoDisponible
	pet := ports.PeticionSubirImagen{VersionEsperada: 0, CatalogoVersionRef: r.catalogo.VersionRef, ClaveOperacion: "imagen_foto_0123456789", Paleta: "azul", TipoDeclarado: "image/png", Original: []byte("bytes-originales")}
	_, err := s.Subir(ctx, o, pet)
	if !errors.Is(err, ports.ErrImagenNoDisponible) || !c.existe || c.confirmada || r.existe || tr.llamadas != 1 {
		t.Fatalf("reserva tras fallo: %v %+v", err, c)
	}
	r.guardarErr = nil
	rec, err := s.ReconciliarFoto(ctx, o, pet.ClaveOperacion)
	if err != nil || !c.confirmada || rec.Version != 1 || rec.Eleccion.DocumentoRef != c.reserva.DocumentoRef || tr.llamadas != 1 {
		t.Fatalf("reconciliar %+v %v", rec, err)
	}
	c.errorConfirmar = true
	_, err = s.ReconciliarFoto(ctx, o, pet.ClaveOperacion)
	if !errors.Is(err, ports.ErrImagenNoDisponible) {
		t.Fatalf("confirmación interrumpida: %v", err)
	}
	c.errorConfirmar = false
	replay, err := s.ReconciliarFoto(ctx, o, pet.ClaveOperacion)
	if err != nil || !replay.Replay || replay.ReciboRef != rec.ReciboRef || r.estado.Version != 1 {
		t.Fatalf("replay reconciliado %+v %v", replay, err)
	}
	visible, err := s.ConsultarPropia(ctx, o)
	if err != nil || visible.Estado.Eleccion.Modo != domain.ModoFoto || !visible.FotoDisponible {
		t.Fatalf("foto disponible %+v %v", visible, err)
	}
	// La ausencia del objeto custodiado pide iniciales para presentar, pero
	// conserva la elección real, la versión y la referencia en el estado.
	c.disponible = false
	vista, err := s.ConsultarPropia(ctx, o)
	if err != nil || vista.Estado.Eleccion.Modo != domain.ModoFoto || vista.FotoDisponible || vista.Estado.Eleccion.DocumentoRef != rec.Eleccion.DocumentoRef || vista.Estado.Version != 1 || vista.NombreAutorizado != "" {
		t.Fatalf("fallback %+v %v", vista, err)
	}
	retirada, err := s.Retirar(ctx, o, 1, r.catalogo.VersionRef, "imagen_retirar_foto_123456", "azul")
	if err != nil || retirada.Version != 2 || r.estado.Eleccion.DocumentoRef != "" {
		t.Fatalf("retirada de referencia %+v %v", retirada, err)
	}
}

func TestFotoConservaReplayQueDevuelveGuardarEnCarrera(t *testing.T) {
	ctx := context.Background()
	s, o, _, r, c, _, _ := fixtureImagen(t, ports.AudienciaImagenPersonal)
	r.guardarReplay = true
	p := ports.PeticionSubirImagen{VersionEsperada: 0, CatalogoVersionRef: r.catalogo.VersionRef, ClaveOperacion: "imagen_foto_carrera_12345", Paleta: "azul", TipoDeclarado: "image/png", Original: []byte("dato")}
	recibo, err := s.Subir(ctx, o, p)
	if err != nil || !recibo.Replay || !c.confirmada || r.guardados != 1 {
		t.Fatalf("replay concurrente %+v %v", recibo, err)
	}
}

func TestFotoRechazaTamanoYMaterialDistinto(t *testing.T) {
	ctx := context.Background()
	s, o, _, r, c, tr, _ := fixtureImagen(t, ports.AudienciaImagenPersonal)
	p := ports.PeticionSubirImagen{VersionEsperada: 0, CatalogoVersionRef: r.catalogo.VersionRef, ClaveOperacion: "imagen_foto_0123456789", Paleta: "azul", Original: []byte("dato"), TipoDeclarado: "image/jpeg"}
	_, err := s.Subir(ctx, o, p)
	if !errors.Is(err, ports.ErrImagenPeticionInvalida) || c.reservas != 0 {
		t.Fatalf("tipo real: %v", err)
	}
	p.TipoDeclarado = "image/png"
	p.Original = []byte(strings.Repeat("x", ports.TamanoMaximoOriginalImagen+1))
	_, err = s.Subir(ctx, o, p)
	if !errors.Is(err, ports.ErrImagenPeticionInvalida) || tr.llamadas != 1 {
		t.Fatalf("tamaño: %v", err)
	}
	p.Original = []byte("dato")
	_, err = s.Subir(ctx, o, p)
	if err != nil {
		t.Fatal(err)
	}
	p.Original = []byte("otro")
	_, err = s.Subir(ctx, o, p)
	if !errors.Is(err, ports.ErrImagenConflicto) {
		t.Fatalf("misma clave otro material: %v", err)
	}
}

func TestFotoDenegadaNoLeeReservaNiProcesa(t *testing.T) {
	ctx := context.Background()
	s, o, proveedor, r, c, tr, _ := fixtureImagen(t, ports.AudienciaImagenPersonal)
	proveedor.denegar = true
	p := ports.PeticionSubirImagen{VersionEsperada: 0, CatalogoVersionRef: r.catalogo.VersionRef, ClaveOperacion: "imagen_foto_denegada_123", Paleta: "azul", TipoDeclarado: "image/png", Original: []byte("dato")}
	_, err := s.Subir(ctx, o, p)
	if !errors.Is(err, ports.ErrImagenProhibido) || c.recuperaciones != 0 || c.reservas != 0 || tr.llamadas != 0 {
		t.Fatalf("subida sin permiso: %v", err)
	}
	_, err = s.ReconciliarFoto(ctx, o, p.ClaveOperacion)
	if !errors.Is(err, ports.ErrImagenProhibido) || c.recuperaciones != 0 {
		t.Fatalf("reconciliación sin permiso: %v", err)
	}
}
