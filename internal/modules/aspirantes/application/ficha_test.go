package application

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/aspirantes/canonico"
	"vec-diputacion-granada/internal/modules/aspirantes/domain"
	"vec-diputacion-granada/internal/modules/aspirantes/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Persona sintética del caso típico: Lucía Fernández Moreno, DNI sintético.
const dniPrueba = "12345678Z"

type proveedorPrueba struct {
	errorDetallado   error
	denegar          bool
	audienciaForzada string
	materiales       []ports.MaterialFicha
	serializados     [][]byte
}

func (p *proveedorPrueba) ProveerMaterialFicha(_ context.Context, vinculo vecdomain.VinculoAutenticacionActorV2, m ports.MaterialFicha) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacia := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	datos, err := vinculo.Datos()
	if err != nil || datos.Superficie != m.Superficie {
		return vacia, ports.ErrProhibido
	}
	p.materiales = append(p.materiales, m)
	if p.errorDetallado != nil {
		return vacia, p.errorDetallado
	}
	if p.denegar {
		return vacia, ports.ErrProhibido
	}
	b, err := canonico.SerializarMaterial(m)
	if err != nil {
		return vacia, err
	}
	p.serializados = append(p.serializados, b)
	recurso, err := canonico.Recurso(m)
	if err != nil {
		return vacia, err
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return vacia, err
	}
	audiencia, _ := ports.Audiencia(m.Accion)
	if p.audienciaForzada != "" {
		audiencia = p.audienciaForzada
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(fmt.Sprintf("dec_aspirantes_%d", len(p.materiales)), strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), m.Accion, m.PersonaRef, huella, audiencia, ahora, ahora.Add(3*time.Second))
	if err != nil {
		return vacia, err
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	return vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), raiz)
}

// protectorPrueba «cifra» de forma reversible y comprueba en cada descifrado
// que ficha, campo y versión coinciden con los del cifrado.
type protectorPrueba struct {
	llamadas int
	fallar   bool
}

func (p *protectorPrueba) sobre(ligado string, claro []byte) ports.SobreCifrado {
	p.llamadas++
	return ports.SobreCifrado{ClaveRef: "clave:prueba:v1", Nonce: bytes.Repeat([]byte{1}, 12), Cifrado: append([]byte(ligado+"|"), append(append([]byte{}, claro...), bytes.Repeat([]byte{0}, 16)...)...)}
}

func (p *protectorPrueba) abrir(ligado string, s ports.SobreCifrado) ([]byte, error) {
	if p.fallar {
		return nil, errors.New("clave revocada")
	}
	resto, ok := bytes.CutPrefix(s.Cifrado, []byte(ligado+"|"))
	if !ok || len(resto) < 16 {
		return nil, errors.New("ligadura")
	}
	return append([]byte{}, resto[:len(resto)-16]...), nil
}

func (p *protectorPrueba) CifrarValor(_ context.Context, asp string, campo domain.CampoFicha, version uint64, claro []byte) (ports.SobreCifrado, error) {
	return p.sobre(fmt.Sprintf("%s/%s/%d", asp, campo, version), claro), nil
}
func (p *protectorPrueba) DescifrarValor(_ context.Context, asp string, campo domain.CampoFicha, version uint64, s ports.SobreCifrado) ([]byte, error) {
	return p.abrir(fmt.Sprintf("%s/%s/%d", asp, campo, version), s)
}
func (p *protectorPrueba) CifrarDocumento(_ context.Context, asp, doc string, claro []byte) (ports.SobreCifrado, error) {
	return p.sobre(asp+"/"+doc, claro), nil
}
func (p *protectorPrueba) DescifrarDocumento(_ context.Context, asp, doc string, s ports.SobreCifrado) ([]byte, error) {
	return p.abrir(asp+"/"+doc, s)
}
func (p *protectorPrueba) IndiceDocumento(_ context.Context, d domain.DocumentoIdentidad) (ports.IndiceDocumento, error) {
	mac := hmac.New(sha256.New, []byte("clave-indice-prueba"))
	fmt.Fprintf(mac, "%s\x00%s\x00%s", d.Tipo, d.Pais, d.Numero)
	return ports.IndiceDocumento{ClaveRef: "clave:indice:v1", Valor: hex.EncodeToString(mac.Sum(nil))}, nil
}

type selladorPrueba struct{}

func (selladorPrueba) SellarHuella(_ context.Context, b []byte) (ports.HuellasSemanticas, error) {
	mac := hmac.New(sha256.New, []byte("clave-huella-prueba"))
	mac.Write(b)
	return ports.HuellasSemanticas{Activa: ports.HuellaSemantica{ClaveRef: "clave:huella:v1", Valor: hex.EncodeToString(mac.Sum(nil))}}, nil
}

type catalogoPrueba struct {
	e   ports.ExigenciasContacto
	err error
}

func (c catalogoPrueba) ExigenciasContactoFichaPropia(context.Context) (ports.ExigenciasContacto, error) {
	return c.e, c.err
}

// registroPrueba guarda en memoria lo que escribiría PostgreSQL.
type registroPrueba struct {
	ficha            *ports.FichaNueva
	version          uint64
	valores          map[domain.CampoFicha]ports.ValorCifrado
	errAlta          error
	errRect          error
	estado           *ports.EstadoParaCambio
	recibidos        []ports.ValorCifrado
	motivo           domain.MotivoCambio
	catalogo         string
	consultas        int
	documentoForzado *ports.DocumentoCifrado
}

func (r *registroPrueba) ConsultarPropia(_ context.Context, _ ports.OrdenFicha, m ports.MaterialFicha, _ vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.FichaCifrada, bool, error) {
	r.consultas++
	if r.ficha == nil || r.ficha.Documento.Indice != m.IndiceDocumento {
		return ports.FichaCifrada{}, false, nil
	}
	f := ports.FichaCifrada{AspiranteRef: r.ficha.AspiranteRef, Version: r.version, Documento: r.ficha.Documento, AccesoRef: "aspacc_" + strings.Repeat("a", 32)}
	if r.documentoForzado != nil {
		f.Documento = *r.documentoForzado
	}
	for _, c := range append(append([]domain.CampoFicha{}, domain.CamposIdentidad...), domain.CamposContacto...) {
		if v, ok := r.valores[c]; ok {
			f.Valores = append(f.Valores, v)
		}
	}
	return f, true, nil
}

func (r *registroPrueba) Alta(_ context.Context, _ ports.OrdenFicha, m ports.MaterialFicha, _ vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, f ports.FichaNueva) (ports.ReciboFicha, error) {
	if r.errAlta != nil {
		return ports.ReciboFicha{}, r.errAlta
	}
	if f.Documento.Indice != m.IndiceDocumento {
		return ports.ReciboFicha{}, errors.New("índice distinto")
	}
	r.ficha, r.version, r.valores = &f, 1, map[domain.CampoFicha]ports.ValorCifrado{}
	for _, v := range f.Valores {
		r.valores[v.Campo] = v
	}
	return ports.ReciboFicha{ReciboRef: "asprec_" + strings.Repeat("1", 32), Accion: ports.AccionAlta, Version: 1, FechaUTC: time.Now().UTC()}, nil
}

func (r *registroPrueba) Rectificar(ctx context.Context, _ ports.OrdenFicha, m ports.MaterialFicha, _ vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, motivo domain.MotivoCambio, catalogo string, c ports.CifradorCambios) (ports.ReciboFicha, error) {
	if r.errRect != nil {
		return ports.ReciboFicha{}, r.errRect
	}
	if r.ficha == nil {
		return ports.ReciboFicha{}, ports.ErrSinFicha
	}
	estado := ports.EstadoParaCambio{AspiranteRef: r.ficha.AspiranteRef, Version: r.version}
	if r.estado != nil {
		estado = *r.estado
	}
	for campo, v := range r.valores {
		if v.Sobre != nil {
			estado.Presentes = append(estado.Presentes, campo)
		}
	}
	valores, err := c.CifrarCambios(ctx, estado)
	if err != nil {
		return ports.ReciboFicha{}, err
	}
	r.recibidos, r.motivo, r.catalogo = valores, motivo, catalogo
	r.version++
	for _, v := range valores {
		r.valores[v.Campo] = v
	}
	return ports.ReciboFicha{ReciboRef: "asprec_" + strings.Repeat("2", 32), Accion: ports.AccionRectificar, Version: r.version, FechaUTC: time.Now().UTC()}, nil
}

type revalidadorPrueba struct {
	a vecdomain.AutenticacionRevalidadaV1
}

func (r revalidadorPrueba) RevalidarAutenticacionActorV1(context.Context, vecdomain.SolicitudRevalidacionAutenticacionActorV1) (vecdomain.AutenticacionRevalidadaV1, error) {
	return r.a, nil
}

type resolutorPrueba struct {
	r vecdomain.ResultadoContextoActorRegistradoV2
}

func (r resolutorPrueba) ResolverContextoActorRegistradoV2(context.Context, vecdomain.SolicitudContextoActor) (vecdomain.ResultadoContextoActorRegistradoV2, error) {
	return r.r, nil
}

type relojPrueba struct{ ahora time.Time }

func (r relojPrueba) Ahora() time.Time { return r.ahora }

func sesionPrueba(t *testing.T, superficie vecdomain.SuperficieAutenticacionActorV1) (vecdomain.ContextoActor, vecdomain.VinculoAutenticacionActorV2) {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	z := strings.Repeat("a", 24)
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	snap := vecdomain.InstantaneaContextoActor{VinculoRef: "vca_" + z, VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_" + strings.Repeat("l", 24), PersonaVersion: 1, PerfilActivoRef: "prf_" + strings.Repeat("p", 24), PerfilVersion: 1, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}
	actor, err := vecdomain.NuevoContextoActor(cuenta, snap, ahora)
	if err != nil {
		t.Fatal(err)
	}
	canon, _ := actor.RepresentacionCanonicaVinculadaV2()
	huella, _ := actor.HuellaSHA256VinculadaV2()
	ac := vecdomain.AcreditacionProcedenciaComponenteContextoActorV1{ProcedenciaRef: "prc_" + z, ProcedenciaVersion: 1, ProcedenciaHuellaSHA256: strings.Repeat("a", 64), ProcedenciaAutoridad: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1}
	man := vecdomain.ManifiestoProcedenciaContextoActorV1{Esquema: vecdomain.EsquemaManifiestoProcedenciaContextoActorV1, AutoridadEfectiva: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, Cuenta: vecdomain.ProcedenciaCuentaContextoActorV1{CuentaRef: cuenta.CuentaRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Persona: vecdomain.ProcedenciaPersonaContextoActorV1{PersonaRef: actor.PersonaRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Perfil: vecdomain.ProcedenciaPerfilContextoActorV1{PerfilRef: actor.PerfilActivoRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Contexto: vecdomain.ProcedenciaVinculoContextoActorV1{VinculoRef: snap.VinculoRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Vinculos: []vecdomain.ProcedenciaVinculoReferenciaContextoActorV1{}}
	bm, _ := man.RepresentacionCanonicaV1()
	hm, _ := vecdomain.HuellaSHA256ManifiestoProcedenciaContextoActorV1(bm)
	res := vecdomain.ResultadoContextoActorRegistradoV2{RegistroContextoRef: "rca_" + z, Contexto: actor, RepresentacionCanonica: canon, HuellaSHA256: huella, ManifiestoProcedenciaCanonico: bm, ManifiestoProcedenciaHuellaSHA256: hm, AutoridadEfectiva: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, ResueltoEnAutoritativo: ahora}
	if err := res.Validar(); err != nil {
		t.Fatal(err)
	}
	auth := vecdomain.AutenticacionRevalidadaV1{AutenticacionRef: "aut_" + z, AutenticacionHuellaSHA256: strings.Repeat("a", 64), AsercionRef: "ase_" + z, SesionRef: "ses_" + z, ControlSesionRef: "cse_" + z, ControlSesionRevision: 1, ControlSesionHuellaSHA256: strings.Repeat("b", 64), CuentaRef: cuenta.CuentaRef, CuentaOrdinariaRef: cuenta.CuentaRef, Superficie: superficie, MetodoObservado: vecdomain.AuthMethodCertificate, GarantiaObservada: vecdomain.AuthAssuranceHigh, PoliticaGarantiaRef: "pga_" + z, PoliticaGarantiaHuellaSHA256: strings.Repeat("c", 64), AutenticacionVerificadaEn: ahora.Add(-time.Minute), SesionEmitidaEn: ahora.Add(-time.Minute), SesionRevalidadaEn: ahora.Add(-time.Second), SesionValidaHasta: ahora.Add(time.Minute)}
	v, err := vecdomain.CrearVinculoAutenticacionActorV2(context.Background(), revalidadorPrueba{auth}, vecdomain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: auth.AutenticacionRef, SesionRef: auth.SesionRef}, resolutorPrueba{res}, vecdomain.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: actor.PerfilActivoRef}, relojPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	return actor, v
}

func identidadPrueba(t *testing.T) domain.IdentidadAcreditada {
	t.Helper()
	doc, err := domain.NuevoDocumentoIdentidad(domain.DocumentoDNI, "ES", dniPrueba)
	if err != nil {
		t.Fatal(err)
	}
	id, err := domain.NuevaIdentidadAcreditada("Lucía", "Fernández Moreno", doc)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

type entorno struct {
	servicio  *ServicioFichaPropia
	registro  *registroPrueba
	protector *protectorPrueba
	proveedor *proveedorPrueba
	orden     ports.OrdenFicha
}

var exigenciasBolsa = ports.ExigenciasContacto{CatalogoRef: "catalogo:campos:bolsa:v1", Campos: []domain.ExigenciaCampo{
	{Campo: domain.CampoTelefono, Obligatorio: true}, {Campo: domain.CampoMovil}, {Campo: domain.CampoDomicilio}, {Campo: domain.CampoCodigoPostal},
}}

func nuevoEntorno(t *testing.T, catalogo catalogoPrueba) *entorno {
	t.Helper()
	e := &entorno{registro: &registroPrueba{}, protector: &protectorPrueba{}, proveedor: &proveedorPrueba{}}
	s, err := NuevoServicioFichaPropia(Dependencias{Registro: e.registro, Protector: e.protector, Sellador: selladorPrueba{}, Catalogo: catalogo, Azar: rand.Reader, AhoraUTC: time.Now})
	if err != nil {
		t.Fatal(err)
	}
	actor, vinculo := sesionPrueba(t, vecdomain.SuperficieAutenticacionExternaPersonalV1)
	e.orden, err = NuevaOrdenFicha(actor, vinculo, vecdomain.SuperficieAutenticacionExternaPersonalV1, identidadPrueba(t), e.proveedor)
	if err != nil {
		t.Fatal(err)
	}
	e.servicio = s
	return e
}

func TestSuperficieInternaNoTieneFicha(t *testing.T) {
	actor, vinculo := sesionPrueba(t, vecdomain.SuperficieAutenticacionInternaCorporativaV1)
	if _, err := NuevaOrdenFicha(actor, vinculo, vecdomain.SuperficieAutenticacionExternaPersonalV1, identidadPrueba(t), &proveedorPrueba{}); !errors.Is(err, ports.ErrProhibido) {
		t.Fatalf("vínculo interno: %v", err)
	}
	if _, err := NuevaOrdenFicha(actor, vinculo, vecdomain.SuperficieAutenticacionInternaCorporativaV1, identidadPrueba(t), &proveedorPrueba{}); !errors.Is(err, ports.ErrProhibido) {
		t.Fatalf("ruta interna: %v", err)
	}
	externo, vinculoExterno := sesionPrueba(t, vecdomain.SuperficieAutenticacionExternaPersonalV1)
	if _, err := NuevaOrdenFicha(externo, vinculoExterno, vecdomain.SuperficieAutenticacionExternaPersonalV1, domain.IdentidadAcreditada{}, &proveedorPrueba{}); !errors.Is(err, ports.ErrNoAutenticado) {
		t.Fatalf("sin identidad: %v", err)
	}
	if _, err := NuevaOrdenFicha(externo, vinculoExterno, vecdomain.SuperficieAutenticacionExternaPersonalV1, identidadPrueba(t), nil); !errors.Is(err, ports.ErrNoAutenticado) {
		t.Fatalf("sin proveedor: %v", err)
	}
}

func TestConsultarSinFichaMuestraElCertificadoSinGuardarlo(t *testing.T) {
	e := nuevoEntorno(t, catalogoPrueba{e: exigenciasBolsa})
	v, err := e.servicio.Consultar(context.Background(), e.orden)
	if err != nil {
		t.Fatal(err)
	}
	if v.Estado != EstadoSinFicha || v.Identidad.Nombre != "Lucía" || v.Identidad.Documento != "***4567**" || v.Identidad.TipoDocumento != "dni" || v.Version != 0 || !v.CatalogoDisponible || len(v.Exigencias) != 4 {
		t.Fatalf("vista %+v", v)
	}
	if e.registro.ficha != nil || e.protector.llamadas != 0 {
		t.Fatal("consultar no escribe ni cifra")
	}
	m := e.proveedor.materiales[0]
	if m.Accion != ports.AccionConsultar || m.ClaveOperacion != "" || m.Superficie != vecdomain.SuperficieAutenticacionExternaPersonalV1 || !canonico.IndiceValido(m.IndiceDocumento) {
		t.Fatalf("material %+v", m)
	}
}

func altaLucia(t *testing.T, e *entorno) ports.ReciboFicha {
	t.Helper()
	r, err := e.servicio.Alta(context.Background(), e.orden, PeticionFicha{ClaveOperacion: "alta-lucia-0000000001", Campos: map[string]string{"telefono": "958 12 34 56", "domicilio": "Calle Recogidas, 12, 3.º B", "codigo_postal": "18002"}})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestAltaCifraTodoYConsultarLoDevuelve(t *testing.T) {
	e := nuevoEntorno(t, catalogoPrueba{e: exigenciasBolsa})
	r := altaLucia(t, e)
	if r.Version != 1 || r.Accion != ports.AccionAlta {
		t.Fatalf("recibo %+v", r)
	}
	f := e.registro.ficha
	if !domain.ReferenciaAspiranteValida(f.AspiranteRef) || !domain.ReferenciaDocumentoValida(f.Documento.DocumentoRef) || f.CatalogoRef != exigenciasBolsa.CatalogoRef || len(f.Valores) != 5 {
		t.Fatalf("ficha %+v", f)
	}
	for _, v := range f.Valores {
		if v.Version != 1 || v.Sobre == nil || v.Origen != v.Campo.OrigenEsperado() {
			t.Fatalf("valor %+v", v)
		}
	}
	// El material V3 nunca lleva datos en claro.
	for _, b := range e.proveedor.serializados {
		for _, claro := range []string{dniPrueba, "Lucía", "958123456", "Recogidas", "18002"} {
			if bytes.Contains(b, []byte(claro)) {
				t.Fatalf("material con %q: %s", claro, b)
			}
		}
	}
	m := e.proveedor.materiales[len(e.proveedor.materiales)-1]
	if m.Accion != ports.AccionAlta || m.VersionEsperada != 0 || m.ClaveOperacion != "alta-lucia-0000000001" || !canonico.HuellasValidas(m.HuellasPeticion) {
		t.Fatalf("material alta %+v", m)
	}
	v, err := e.servicio.Consultar(context.Background(), e.orden)
	if err != nil {
		t.Fatal(err)
	}
	if v.Estado != EstadoActiva || v.Version != 1 || v.Identidad.Apellidos != "Fernández Moreno" || v.Contacto["telefono"] != "958123456" || v.Contacto["codigo_postal"] != "18002" || len(v.Contacto) != 3 {
		t.Fatalf("vista %+v", v)
	}
}

func TestAltaRechazaLoQueNoPideElCatalogo(t *testing.T) {
	e := nuevoEntorno(t, catalogoPrueba{e: ports.ExigenciasContacto{CatalogoRef: "catalogo:campos:vacio:v1"}})
	if _, err := e.servicio.Alta(context.Background(), e.orden, PeticionFicha{ClaveOperacion: "alta-lucia-0000000002", Campos: map[string]string{"telefono": "958123456"}}); !errors.Is(err, ports.ErrInvalida) {
		t.Fatalf("campo no pedido: %v", err)
	}
	if len(e.proveedor.materiales) != 0 {
		t.Fatal("no se pide V3 para una petición inválida")
	}
	if _, err := e.servicio.Alta(context.Background(), e.orden, PeticionFicha{ClaveOperacion: "alta-lucia-0000000003"}); err != nil {
		t.Fatalf("alta solo con identidad: %v", err)
	}
}

func TestAltaValidaPeticion(t *testing.T) {
	e := nuevoEntorno(t, catalogoPrueba{e: exigenciasBolsa})
	casos := []PeticionFicha{
		{ClaveOperacion: "corta", Campos: map[string]string{"telefono": "958123456"}},
		{ClaveOperacion: "alta-lucia-0000000004", VersionEsperada: 1, Campos: map[string]string{"telefono": "958123456"}},
		{ClaveOperacion: "alta-lucia-0000000004", Motivo: "dato_nuevo", Campos: map[string]string{"telefono": "958123456"}},
		{ClaveOperacion: "alta-lucia-0000000004", Campos: map[string]string{"nombre": "Otra"}},
		{ClaveOperacion: "alta-lucia-0000000004", Campos: map[string]string{"discapacidad": "33"}},
		{ClaveOperacion: "alta-lucia-0000000004", Campos: map[string]string{}},
	}
	for i, p := range casos {
		if _, err := e.servicio.Alta(context.Background(), e.orden, p); !errors.Is(err, ports.ErrInvalida) {
			t.Fatalf("caso %d: %v", i, err)
		}
	}
}

func TestAltaErroresDelRegistroYCatalogo(t *testing.T) {
	e := nuevoEntorno(t, catalogoPrueba{err: errors.New("catálogo caído")})
	if _, err := e.servicio.Alta(context.Background(), e.orden, PeticionFicha{ClaveOperacion: "alta-lucia-0000000005"}); !errors.Is(err, ports.ErrNoDisponible) {
		t.Fatalf("catálogo caído: %v", err)
	}
	e = nuevoEntorno(t, catalogoPrueba{e: exigenciasBolsa})
	e.registro.errAlta = ports.ErrFichaExistente
	if _, err := e.servicio.Alta(context.Background(), e.orden, PeticionFicha{ClaveOperacion: "alta-lucia-0000000006", Campos: map[string]string{"telefono": "958123456"}}); !errors.Is(err, ports.ErrFichaExistente) {
		t.Fatalf("existente: %v", err)
	}
	e.registro.errAlta = errors.New("pq: detalle interno")
	_, err := e.servicio.Alta(context.Background(), e.orden, PeticionFicha{ClaveOperacion: "alta-lucia-0000000007", Campos: map[string]string{"telefono": "958123456"}})
	if !errors.Is(err, ports.ErrNoDisponible) || strings.Contains(err.Error(), "pq") {
		t.Fatalf("error interno: %v", err)
	}
}

func TestV3NoLigadaSeRechaza(t *testing.T) {
	e := nuevoEntorno(t, catalogoPrueba{e: exigenciasBolsa})
	e.proveedor.audienciaForzada = "vec_usuarios.correos.consultar.externa_personal.v1"
	if _, err := e.servicio.Consultar(context.Background(), e.orden); !errors.Is(err, ports.ErrNoDisponible) {
		t.Fatalf("audiencia ajena: %v", err)
	}
	e.proveedor.audienciaForzada, e.proveedor.denegar = "", true
	if _, err := e.servicio.Consultar(context.Background(), e.orden); !errors.Is(err, ports.ErrProhibido) {
		t.Fatalf("denegada: %v", err)
	}
	if e.registro.consultas != 0 {
		t.Fatal("el registro no se toca sin V3")
	}
}

func TestConsultarDetectaDocumentoAjenoYCatalogoCaido(t *testing.T) {
	e := nuevoEntorno(t, catalogoPrueba{e: exigenciasBolsa})
	altaLucia(t, e)
	otro := e.registro.ficha.Documento
	otro.Tipo = domain.DocumentoNIE
	e.registro.documentoForzado = &otro
	if _, err := e.servicio.Consultar(context.Background(), e.orden); !errors.Is(err, ports.ErrNoDisponible) {
		t.Fatalf("documento distinto: %v", err)
	}
	e.registro.documentoForzado = nil
	e.protector.fallar = true
	if _, err := e.servicio.Consultar(context.Background(), e.orden); !errors.Is(err, ports.ErrNoDisponible) {
		t.Fatalf("clave revocada: %v", err)
	}
	e2 := nuevoEntorno(t, catalogoPrueba{err: errors.New("caído")})
	v, err := e2.servicio.Consultar(context.Background(), e2.orden)
	if err != nil || v.CatalogoDisponible || len(v.Exigencias) != 0 {
		t.Fatalf("sin catálogo no se pide nada: %+v %v", v, err)
	}
}

func TestRectificarConMotivo(t *testing.T) {
	e := nuevoEntorno(t, catalogoPrueba{e: exigenciasBolsa})
	altaLucia(t, e)
	r, err := e.servicio.Rectificar(context.Background(), e.orden, PeticionFicha{ClaveOperacion: "rect-lucia-0000000001", VersionEsperada: 1, Motivo: "cambio_de_dato", Campos: map[string]string{"telefono": "612 345 678", "domicilio": ""}})
	if err != nil {
		t.Fatal(err)
	}
	if r.Version != 2 || e.registro.motivo != domain.MotivoCambioDeDato || e.registro.catalogo != exigenciasBolsa.CatalogoRef || len(e.registro.recibidos) != 2 {
		t.Fatalf("recibo %+v %+v", r, e.registro)
	}
	for _, v := range e.registro.recibidos {
		if v.Version != 2 || v.Origen != domain.OrigenTitular || (v.Campo == domain.CampoDomicilio) != (v.Sobre == nil) {
			t.Fatalf("valor %+v", v)
		}
	}
	v, _ := e.servicio.Consultar(context.Background(), e.orden)
	if v.Version != 2 || v.Contacto["telefono"] != "612345678" || v.Contacto["domicilio"] != "" {
		t.Fatalf("vista %+v", v)
	}
	if _, err := e.servicio.Rectificar(context.Background(), e.orden, PeticionFicha{ClaveOperacion: "rect-lucia-0000000002", VersionEsperada: 2, Motivo: "dato_nuevo", Campos: map[string]string{"telefono": "958123456"}}); !errors.Is(err, ports.ErrInvalida) {
		t.Fatalf("dato nuevo sobre valor existente: %v", err)
	}
	if _, err := e.servicio.Rectificar(context.Background(), e.orden, PeticionFicha{ClaveOperacion: "rect-lucia-0000000003", VersionEsperada: 2, Motivo: "dato_nuevo", Campos: map[string]string{"movil": "699 111 222"}}); err != nil {
		t.Fatalf("móvil nuevo: %v", err)
	}
}

func TestRectificarRechazosAntesDeV3(t *testing.T) {
	e := nuevoEntorno(t, catalogoPrueba{e: exigenciasBolsa})
	altaLucia(t, e)
	antes := len(e.proveedor.materiales)
	casos := []PeticionFicha{
		{ClaveOperacion: "rect-lucia-0000000004", VersionEsperada: 1, Motivo: "cambio_de_dato", Campos: map[string]string{"telefono": ""}},
		{ClaveOperacion: "rect-lucia-0000000004", VersionEsperada: 1, Motivo: "alta_titular", Campos: map[string]string{"movil": "612345678"}},
		{ClaveOperacion: "rect-lucia-0000000004", VersionEsperada: 0, Motivo: "dato_nuevo", Campos: map[string]string{"movil": "612345678"}},
		{ClaveOperacion: "rect-lucia-0000000004", VersionEsperada: 1, Motivo: "dato_nuevo"},
		{ClaveOperacion: "rect-lucia-0000000004", VersionEsperada: 1, Motivo: "cambio_de_dato", Campos: map[string]string{"apellidos": "Otro"}},
		{ClaveOperacion: "rect lucia", VersionEsperada: 1, Motivo: "dato_nuevo", Campos: map[string]string{"movil": "612345678"}},
	}
	for i, p := range casos {
		if _, err := e.servicio.Rectificar(context.Background(), e.orden, p); !errors.Is(err, ports.ErrInvalida) {
			t.Fatalf("caso %d: %v", i, err)
		}
	}
	if len(e.proveedor.materiales) != antes {
		t.Fatal("peticiones inválidas no piden V3")
	}
}

func TestRectificarConflictoYSinFicha(t *testing.T) {
	e := nuevoEntorno(t, catalogoPrueba{e: exigenciasBolsa})
	if _, err := e.servicio.Rectificar(context.Background(), e.orden, PeticionFicha{ClaveOperacion: "rect-lucia-0000000005", VersionEsperada: 1, Motivo: "dato_nuevo", Campos: map[string]string{"movil": "612345678"}}); !errors.Is(err, ports.ErrSinFicha) {
		t.Fatalf("sin ficha: %v", err)
	}
	altaLucia(t, e)
	e.registro.estado = &ports.EstadoParaCambio{AspiranteRef: e.registro.ficha.AspiranteRef, Version: 7}
	if _, err := e.servicio.Rectificar(context.Background(), e.orden, PeticionFicha{ClaveOperacion: "rect-lucia-0000000006", VersionEsperada: 1, Motivo: "dato_nuevo", Campos: map[string]string{"movil": "612345678"}}); !errors.Is(err, ports.ErrConflicto) {
		t.Fatalf("versión: %v", err)
	}
}

func TestHuellaSemanticaEstableYDistinta(t *testing.T) {
	e := nuevoEntorno(t, catalogoPrueba{e: exigenciasBolsa})
	altaLucia(t, e)
	peticion := PeticionFicha{ClaveOperacion: "rect-lucia-0000000007", VersionEsperada: 1, Motivo: "dato_nuevo", Campos: map[string]string{"movil": "612 345 678"}}
	e.registro.errRect = ports.ErrConflicto
	_, _ = e.servicio.Rectificar(context.Background(), e.orden, peticion)
	a := e.proveedor.materiales[len(e.proveedor.materiales)-1].HuellasPeticion
	peticion.Campos["movil"] = "612345678"
	_, _ = e.servicio.Rectificar(context.Background(), e.orden, peticion)
	b := e.proveedor.materiales[len(e.proveedor.materiales)-1].HuellasPeticion
	peticion.Campos["movil"] = "699111222"
	_, _ = e.servicio.Rectificar(context.Background(), e.orden, peticion)
	c := e.proveedor.materiales[len(e.proveedor.materiales)-1].HuellasPeticion
	if a.Activa != b.Activa || a.Activa == c.Activa {
		t.Fatal("la huella depende del valor normalizado, no de su escritura")
	}
}

func TestServicioExigeDependencias(t *testing.T) {
	if _, err := NuevoServicioFichaPropia(Dependencias{}); !errors.Is(err, ports.ErrNoDisponible) {
		t.Fatal(err)
	}
	var s *ServicioFichaPropia
	if _, err := s.Consultar(context.Background(), ports.OrdenFicha{}); !errors.Is(err, ports.ErrNoDisponible) {
		t.Fatal(err)
	}
}

func TestErrorDelProveedorNoFiltraDetalle(t *testing.T) {
	e := nuevoEntorno(t, catalogoPrueba{e: exigenciasBolsa})
	e.proveedor.errorDetallado = fmt.Errorf("perfil prf_secreto sin rol en tabla vec_autorizacion.asignacion: %w", ports.ErrProhibido)
	_, err := e.servicio.Consultar(context.Background(), e.orden)
	if err != ports.ErrProhibido {
		t.Fatalf("se esperaba el error nominal exacto: %v", err)
	}
	e.proveedor.errorDetallado = errors.New("dial tcp 10.0.0.5:5432: conexión rechazada")
	if _, err := e.servicio.Consultar(context.Background(), e.orden); err != ports.ErrNoDisponible {
		t.Fatalf("error interno: %v", err)
	}
}

func TestVistaRotulaCatalogoDeEjemploYCondicion(t *testing.T) {
	e := nuevoEntorno(t, catalogoPrueba{e: ports.ExigenciasContacto{CatalogoRef: "vec.aspirantes.datos_personales:1", Ejemplo: true,
		Campos: []domain.ExigenciaCampo{{Campo: domain.CampoDomicilio, Condicion: "si_elige_notificacion_papel"}}}})
	v, err := e.servicio.Consultar(context.Background(), e.orden)
	if err != nil || !v.CatalogoEjemplo || v.Exigencias[0].Condicion != "si_elige_notificacion_papel" {
		t.Fatalf("vista %+v %v", v.Exigencias, err)
	}
	if strings.Contains(fmt.Sprint(v), "Lucía") || strings.Contains(fmt.Sprint(PeticionFicha{Campos: map[string]string{"telefono": "958123456"}}), "958") {
		t.Fatal("vista o petición impresas en claro")
	}
}
